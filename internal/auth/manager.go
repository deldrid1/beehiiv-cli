package auth

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/deldrid1/beehiiv-cli/internal/client"
	"github.com/deldrid1/beehiiv-cli/internal/config"
)

const refreshSkew = time.Minute

type Manager struct {
	env        map[string]string
	httpClient client.HTTPClient
	store      Store
}

type Status struct {
	Configured       bool     `json:"configured"`
	AuthMode         string   `json:"auth_mode,omitempty"`
	SecretBackend    string   `json:"secret_backend,omitempty"`
	PublicationID    string   `json:"publication_id,omitempty"`
	BaseURL          string   `json:"base_url,omitempty"`
	RateLimitRPM     int      `json:"rate_limit_rpm,omitempty"`
	SettingsPath     string   `json:"settings_path,omitempty"`
	TokenSource      string   `json:"token_source,omitempty"`
	TokenExpiresAt   string   `json:"token_expires_at,omitempty"`
	TokenScope       string   `json:"token_scope,omitempty"`
	OAuthClientID    string   `json:"oauth_client_id,omitempty"`
	OAuthRedirectURI string   `json:"oauth_redirect_uri,omitempty"`
	OAuthScopes      []string `json:"oauth_scopes,omitempty"`
	ResourceOwnerID  string   `json:"resource_owner_id,omitempty"`
	ApplicationUID   string   `json:"application_uid,omitempty"`
	ApplicationName  string   `json:"application_name,omitempty"`
	ClientHasSecret  bool     `json:"client_has_secret,omitempty"`
}

type APIKeyLoginOptions struct {
	SettingsPath  string
	APIKey        string
	PublicationID string
	BaseURL       string
	RateLimitRPM  int
}

type OAuthSessionOptions struct {
	SettingsPath    string
	ClientID        string
	ClientSecret    string
	RedirectURI     string
	PublicationID   string
	BaseURL         string
	RateLimitRPM    int
	RequestedScopes []string
	TokenResponse   OAuthTokenResponse
	TokenInfo       *OAuthTokenInfo
}

type Paths struct {
	SettingsPath string `json:"settings_path"`
}

func NewManager(env map[string]string, httpClient client.HTTPClient) *Manager {
	if env == nil {
		env = make(map[string]string)
		for _, entry := range os.Environ() {
			key, value, ok := strings.Cut(entry, "=")
			if ok {
				env[key] = value
			}
		}
	}
	return &Manager{
		env:        env,
		httpClient: httpClient,
		store:      NewKeyringStore(DefaultKeyringService, DefaultKeyringUser),
	}
}

func NewManagerWithStore(env map[string]string, httpClient client.HTTPClient, store Store) *Manager {
	manager := NewManager(env, httpClient)
	manager.store = store
	return manager
}

func (m *Manager) ResolveRuntime(ctx context.Context, overrides config.Overrides) (config.Runtime, error) {
	configPath, err := m.settingsPath(overrides.ConfigPath)
	if err != nil {
		return config.Runtime{}, err
	}

	overrides.ConfigPath = configPath
	runtime, err := config.LoadRuntime(overrides, m.env)
	if err != nil {
		return config.Runtime{}, err
	}
	if runtime.APIKey != "" {
		return runtime, nil
	}

	settings, err := config.LoadSettings(configPath)
	if err != nil {
		return config.Runtime{}, err
	}
	if settings.AuthMode == "" {
		return runtime, nil
	}

	secret, err := m.store.Load()
	if err != nil {
		if errors.Is(err, ErrSecretNotFound) {
			return runtime, nil
		}
		return config.Runtime{}, fmt.Errorf("load secure credentials from keyring: %w", err)
	}

	switch settings.AuthMode {
	case config.AuthModeAPIKey:
		runtime.APIKey = secret.APIKey
	case config.AuthModeOAuth:
		oauthSecret := secret.OAuth
		if needsRefresh(oauthSecret) {
			if oauthSecret.RefreshToken == "" {
				return config.Runtime{}, errors.New("stored OAuth session has expired and no refresh token is available; run `beehiiv auth oauth login` again")
			}

			response, err := RefreshAccessToken(ctx, m.httpClient, RefreshTokenRequest{
				ClientID:     settings.OAuth.ClientID,
				ClientSecret: secret.ClientSecret,
				RefreshToken: oauthSecret.RefreshToken,
			})
			if err != nil {
				return config.Runtime{}, fmt.Errorf("refresh Beehiiv OAuth access token: %w", err)
			}

			secret.OAuth = SecretFromTokenResponse(response)
			if secret.OAuth.RefreshToken == "" {
				secret.OAuth.RefreshToken = oauthSecret.RefreshToken
			}
			if err := m.store.Save(secret); err != nil {
				return config.Runtime{}, fmt.Errorf("persist refreshed OAuth token in keyring: %w", err)
			}
			oauthSecret = secret.OAuth
		}
		runtime.APIKey = oauthSecret.AccessToken
	default:
		return config.Runtime{}, fmt.Errorf("unsupported auth mode %q", settings.AuthMode)
	}

	return runtime, nil
}

func (m *Manager) Status(overrides config.Overrides) (Status, error) {
	configPath, err := m.settingsPath(overrides.ConfigPath)
	if err != nil {
		return Status{}, err
	}

	overrides.ConfigPath = configPath
	runtime, err := config.LoadRuntime(overrides, m.env)
	if err != nil {
		return Status{}, err
	}
	settings, err := config.LoadSettings(configPath)
	if err != nil {
		return Status{}, err
	}

	status := Status{
		SettingsPath:     configPath,
		AuthMode:         settings.AuthMode,
		SecretBackend:    settings.SecretBackend,
		PublicationID:    runtime.PublicationID,
		BaseURL:          runtime.BaseURL,
		RateLimitRPM:     runtime.RateLimitRPM,
		OAuthClientID:    settings.OAuth.ClientID,
		OAuthRedirectURI: settings.OAuth.RedirectURI,
		OAuthScopes:      append([]string(nil), settings.OAuth.Scopes...),
		ResourceOwnerID:  settings.OAuth.ResourceOwnerID,
		ApplicationUID:   settings.OAuth.ApplicationUID,
		ApplicationName:  settings.OAuth.ApplicationName,
	}

	if runtime.APIKey != "" {
		status.Configured = true
		status.TokenSource = "env_or_flag"
		return status, nil
	}

	secret, err := m.store.Load()
	if err != nil {
		if errors.Is(err, ErrSecretNotFound) {
			return status, nil
		}
		return Status{}, fmt.Errorf("load secure credentials from keyring: %w", err)
	}

	status.Configured = true
	status.TokenSource = "keyring"
	status.ClientHasSecret = secret.ClientSecret != ""
	switch settings.AuthMode {
	case config.AuthModeAPIKey:
		return status, nil
	case config.AuthModeOAuth:
		if !secret.OAuth.ExpiresAt.IsZero() {
			status.TokenExpiresAt = secret.OAuth.ExpiresAt.UTC().Format(time.RFC3339)
		}
		status.TokenScope = secret.OAuth.Scope
		return status, nil
	default:
		return status, nil
	}
}

func (m *Manager) SaveAPIKeySession(options APIKeyLoginOptions) error {
	configPath, err := m.settingsPath(options.SettingsPath)
	if err != nil {
		return err
	}

	settings, err := config.LoadSettings(configPath)
	if err != nil {
		return err
	}
	settings.AuthMode = config.AuthModeAPIKey
	settings.SecretBackend = config.SecretBackendKeyring
	settings.PublicationID = strings.TrimSpace(options.PublicationID)
	if trimmed := strings.TrimSpace(options.BaseURL); trimmed != "" && trimmed != config.DefaultBaseURL {
		settings.BaseURL = trimmed
	}
	if options.RateLimitRPM > 0 && options.RateLimitRPM != config.DefaultRateLimitRPM {
		settings.RateLimitRPM = options.RateLimitRPM
	}
	settings.OAuth = config.OAuthSettings{}
	if err := config.SaveSettings(configPath, settings); err != nil {
		return err
	}

	if err := m.store.Save(SecretRecord{APIKey: options.APIKey}); err != nil {
		return fmt.Errorf("save API key to keyring: %w", err)
	}
	return nil
}

func (m *Manager) SaveOAuthSession(options OAuthSessionOptions) error {
	configPath, err := m.settingsPath(options.SettingsPath)
	if err != nil {
		return err
	}

	settings, err := config.LoadSettings(configPath)
	if err != nil {
		return err
	}
	settings.AuthMode = config.AuthModeOAuth
	settings.SecretBackend = config.SecretBackendKeyring
	settings.PublicationID = strings.TrimSpace(options.PublicationID)
	if trimmed := strings.TrimSpace(options.BaseURL); trimmed != "" && trimmed != config.DefaultBaseURL {
		settings.BaseURL = trimmed
	}
	if options.RateLimitRPM > 0 && options.RateLimitRPM != config.DefaultRateLimitRPM {
		settings.RateLimitRPM = options.RateLimitRPM
	}
	settings.OAuth.ClientID = strings.TrimSpace(options.ClientID)
	settings.OAuth.RedirectURI = strings.TrimSpace(options.RedirectURI)
	settings.OAuth.Scopes = NormalizeScopes(options.RequestedScopes)
	if options.TokenInfo != nil {
		settings.OAuth.ResourceOwnerID = options.TokenInfo.ResourceOwnerID
		settings.OAuth.ApplicationUID = options.TokenInfo.Application.UID
		settings.OAuth.ApplicationName = options.TokenInfo.Application.Name
		if len(options.TokenInfo.Scope) > 0 {
			settings.OAuth.Scopes = append([]string(nil), options.TokenInfo.Scope...)
		}
	}
	if err := config.SaveSettings(configPath, settings); err != nil {
		return err
	}

	record := SecretRecord{
		ClientSecret: options.ClientSecret,
		OAuth:        SecretFromTokenResponse(options.TokenResponse),
	}
	if err := m.store.Save(record); err != nil {
		return fmt.Errorf("save OAuth tokens to keyring: %w", err)
	}
	return nil
}

func (m *Manager) Logout(ctx context.Context, configPath string, revoke bool) error {
	configPath, err := m.settingsPath(configPath)
	if err != nil {
		return err
	}

	settings, err := config.LoadSettings(configPath)
	if err != nil {
		return err
	}

	secret, loadErr := m.store.Load()
	if loadErr != nil && !errors.Is(loadErr, ErrSecretNotFound) {
		return fmt.Errorf("load secure credentials from keyring: %w", loadErr)
	}

	var revokeErr error
	if revoke && loadErr == nil && settings.AuthMode == config.AuthModeOAuth {
		token := secret.OAuth.RefreshToken
		tokenType := "refresh_token"
		if token == "" {
			token = secret.OAuth.AccessToken
			tokenType = "access_token"
		}
		if token != "" {
			revokeErr = RevokeToken(ctx, m.httpClient, RevokeTokenRequest{
				ClientID:     settings.OAuth.ClientID,
				ClientSecret: secret.ClientSecret,
				Token:        token,
				TokenType:    tokenType,
			})
		}
	}

	if err := m.store.Delete(); err != nil {
		return fmt.Errorf("delete keyring credentials: %w", err)
	}

	settings.AuthMode = ""
	settings.SecretBackend = ""
	settings.PublicationID = ""
	settings.OAuth = config.OAuthSettings{}
	if err := config.SaveSettings(configPath, settings); err != nil {
		return err
	}

	return revokeErr
}

func (m *Manager) Paths(configPath string) (Paths, error) {
	configPath, err := m.settingsPath(configPath)
	if err != nil {
		return Paths{}, err
	}
	return Paths{
		SettingsPath: configPath,
	}, nil
}

func (m *Manager) settingsPath(configPath string) (string, error) {
	trimmed, err := config.ValidateConfigPath(configPath)
	if err != nil {
		return "", err
	}
	if trimmed != "" {
		return trimmed, nil
	}
	return config.DefaultConfigPath()
}

func needsRefresh(secret OAuthSecret) bool {
	if secret.AccessToken == "" {
		return secret.RefreshToken != ""
	}
	if secret.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().UTC().Add(refreshSkew).After(secret.ExpiresAt)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func BuildLoopbackCallback(rawURL string) (host string, path string, err error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", "", err
	}
	host = parsed.Host
	path = parsed.EscapedPath()
	if path == "" {
		path = "/"
	}
	return host, path, nil
}
