package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/deldrid1/beehiiv-cli/internal/client"
)

const (
	AuthorizeURL = "https://app.beehiiv.com/oauth/authorize"
	TokenURL     = "https://app.beehiiv.com/oauth/token"
	RevokeURL    = "https://app.beehiiv.com/oauth/revoke"
	TokenInfoURL = "https://app.beehiiv.com/oauth/token/info"
)

var (
	defaultOAuthScopes = []string{"identify:read", "publications:read"}
	allOAuthScopes     = []string{
		"identify:read",
		"automations:read", "automations:write",
		"custom_fields:read", "custom_fields:write",
		"subscriptions:read", "subscriptions:write",
		"polls:read", "polls:write",
		"posts:read", "posts:write",
		"publications:read", "publications:write",
		"referral_program:read", "referral_program:write",
		"segments:read", "segments:write",
		"tiers:read", "tiers:write",
		"webhooks:read", "webhooks:write",
	}
)

type OAuthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
	CreatedAt    int64  `json:"created_at"`
}

type OAuthTokenInfo struct {
	ResourceOwnerID  string    `json:"resource_owner_id"`
	Scope            ScopeList `json:"scope"`
	ExpiresInSeconds int       `json:"expires_in_seconds"`
	Application      struct {
		UID  string `json:"uid"`
		Name string `json:"name"`
	} `json:"application"`
	CreatedAt int64 `json:"created_at"`
}

// ScopeList accepts the OAuth 2.0 scope field as either a space-separated
// JSON string (RFC 6749) or a JSON array of strings. Beehiiv's token/info
// endpoint returns an array even though the authorization response returns
// a space-separated string.
type ScopeList []string

func (s *ScopeList) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || trimmed == "null" {
		*s = nil
		return nil
	}
	if trimmed[0] == '[' {
		var arr []string
		if err := json.Unmarshal(data, &arr); err != nil {
			return err
		}
		*s = arr
		return nil
	}
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	*s = strings.Fields(str)
	return nil
}

type OAuthError struct {
	ErrorCode        string `json:"error"`
	ErrorDescription string `json:"error_description"`
	ErrorURI         string `json:"error_uri"`
	State            string `json:"state"`
}

func (e *OAuthError) Error() string {
	if e.ErrorDescription != "" {
		return fmt.Sprintf("%s: %s", e.ErrorCode, e.ErrorDescription)
	}
	if e.ErrorCode != "" {
		return e.ErrorCode
	}
	return "oauth request failed"
}

type TokenExchangeRequest struct {
	ClientID     string
	ClientSecret string
	Code         string
	RedirectURI  string
	CodeVerifier string
}

type RefreshTokenRequest struct {
	ClientID     string
	ClientSecret string
	RefreshToken string
}

type RevokeTokenRequest struct {
	ClientID     string
	ClientSecret string
	Token        string
	TokenType    string
}

func BuildAuthorizeURL(clientID, redirectURI, state, codeChallenge string, scopes []string) (string, error) {
	u, err := url.Parse(AuthorizeURL)
	if err != nil {
		return "", err
	}
	query := u.Query()
	query.Set("client_id", clientID)
	query.Set("redirect_uri", redirectURI)
	query.Set("response_type", "code")
	if len(scopes) > 0 {
		query.Set("scope", strings.Join(scopes, " "))
	}
	if state != "" {
		query.Set("state", state)
	}
	if codeChallenge != "" {
		query.Set("code_challenge", codeChallenge)
		query.Set("code_challenge_method", "S256")
	}
	u.RawQuery = query.Encode()
	return u.String(), nil
}

func GeneratePKCEVerifier() (string, string, error) {
	verifier, err := randomBase64URL(64)
	if err != nil {
		return "", "", err
	}
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])
	return verifier, challenge, nil
}

func GenerateState() (string, error) {
	return randomBase64URL(24)
}

func NormalizeScopes(scopes []string) []string {
	if len(scopes) == 0 {
		return append([]string(nil), defaultOAuthScopes...)
	}

	expanded := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		scope = strings.TrimSpace(scope)
		switch scope {
		case "", "default":
			expanded = append(expanded, defaultOAuthScopes...)
		case "all":
			expanded = append(expanded, allOAuthScopes...)
		default:
			expanded = append(expanded, scope)
		}
	}

	seen := make(map[string]struct{}, len(expanded))
	out := make([]string, 0, len(expanded))
	for _, scope := range expanded {
		if _, ok := seen[scope]; ok {
			continue
		}
		seen[scope] = struct{}{}
		out = append(out, scope)
	}
	return out
}

func ExchangeAuthorizationCode(ctx context.Context, httpClient client.HTTPClient, request TokenExchangeRequest) (OAuthTokenResponse, error) {
	values := url.Values{
		"grant_type":   []string{"authorization_code"},
		"code":         []string{request.Code},
		"redirect_uri": []string{request.RedirectURI},
		"client_id":    []string{request.ClientID},
	}
	if request.ClientSecret != "" {
		values.Set("client_secret", request.ClientSecret)
	}
	if request.CodeVerifier != "" {
		values.Set("code_verifier", request.CodeVerifier)
	}
	return doOAuthTokenRequest(ctx, httpClient, TokenURL, values)
}

func RefreshAccessToken(ctx context.Context, httpClient client.HTTPClient, request RefreshTokenRequest) (OAuthTokenResponse, error) {
	values := url.Values{
		"grant_type":    []string{"refresh_token"},
		"refresh_token": []string{request.RefreshToken},
		"client_id":     []string{request.ClientID},
	}
	if request.ClientSecret != "" {
		values.Set("client_secret", request.ClientSecret)
	}
	return doOAuthTokenRequest(ctx, httpClient, TokenURL, values)
}

func RevokeToken(ctx context.Context, httpClient client.HTTPClient, request RevokeTokenRequest) error {
	values := url.Values{
		"token": []string{request.Token},
	}
	if request.TokenType != "" {
		values.Set("token_type_hint", request.TokenType)
	}
	if request.ClientID != "" {
		values.Set("client_id", request.ClientID)
	}
	if request.ClientSecret != "" {
		values.Set("client_secret", request.ClientSecret)
	}
	_, err := doOAuthRequest(ctx, httpClient, http.MethodPost, RevokeURL, values, "")
	return err
}

func GetTokenInfo(ctx context.Context, httpClient client.HTTPClient, accessToken string) (OAuthTokenInfo, error) {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, TokenInfoURL, nil)
	if err != nil {
		return OAuthTokenInfo{}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return OAuthTokenInfo{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return OAuthTokenInfo{}, err
	}
	if resp.StatusCode >= 400 {
		return OAuthTokenInfo{}, decodeOAuthError(body, resp.Status)
	}

	var info OAuthTokenInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return OAuthTokenInfo{}, err
	}
	return info, nil
}

func SecretFromTokenResponse(response OAuthTokenResponse) OAuthSecret {
	createdAt := time.Now().UTC()
	if response.CreatedAt > 0 {
		createdAt = time.Unix(response.CreatedAt, 0).UTC()
	}
	expiresAt := time.Time{}
	if response.ExpiresIn > 0 {
		expiresAt = createdAt.Add(time.Duration(response.ExpiresIn) * time.Second)
	}
	return OAuthSecret{
		AccessToken:  response.AccessToken,
		RefreshToken: response.RefreshToken,
		TokenType:    response.TokenType,
		Scope:        response.Scope,
		CreatedAt:    createdAt,
		ExpiresAt:    expiresAt,
	}
}

func doOAuthTokenRequest(ctx context.Context, httpClient client.HTTPClient, endpoint string, values url.Values) (OAuthTokenResponse, error) {
	body, err := doOAuthRequest(ctx, httpClient, http.MethodPost, endpoint, values, "application/x-www-form-urlencoded")
	if err != nil {
		return OAuthTokenResponse{}, err
	}

	var response OAuthTokenResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return OAuthTokenResponse{}, err
	}
	return response, nil
}

func doOAuthRequest(ctx context.Context, httpClient client.HTTPClient, method, endpoint string, values url.Values, contentType string) ([]byte, error) {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}
	if contentType == "" {
		contentType = "application/x-www-form-urlencoded"
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, decodeOAuthError(body, resp.Status)
	}
	return body, nil
}

func decodeOAuthError(body []byte, fallback string) error {
	var oauthErr OAuthError
	if err := json.Unmarshal(body, &oauthErr); err == nil && oauthErr.ErrorCode != "" {
		return &oauthErr
	}
	if len(body) == 0 {
		return fmt.Errorf("oauth request failed: %s", fallback)
	}
	return fmt.Errorf("oauth request failed: %s", strings.TrimSpace(string(body)))
}

func randomBase64URL(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
