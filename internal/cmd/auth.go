package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/deldrid1/beehiiv-cli/internal/auth"
	"github.com/deldrid1/beehiiv-cli/internal/client"
	"github.com/deldrid1/beehiiv-cli/internal/commandset"
	"github.com/deldrid1/beehiiv-cli/internal/config"
	clioutput "github.com/deldrid1/beehiiv-cli/internal/output"
)

const defaultOAuthRedirectURI = "http://localhost:3008/callback"

// Pre-configured OAuth credentials for the DailyDrop beehiiv integration.
// The callback URL is handled by a Lambda relay at api.dailydrop.com that
// 302-redirects the browser back to the CLI's loopback server, so the
// standard local-callback flow works end-to-end without manual copy-paste.
const (
	appOAuthClientID     = "5rNfowFO3sSGzqnN9fwGF6HJpSpWyyyu377RYVuf1Y8"
	appOAuthClientSecret = "kM1zBHK-WMZ3arusicCdXrZzIkKg8VpGGG5R8SnzNPY"
	appOAuthRedirectURI  = "https://api.dailydrop.com/beehiiv/callback"
	appOAuthLoopbackURI  = defaultOAuthRedirectURI // local server the relay redirects back to
)

func newAuthCommand(options Options) *cobra.Command {
	authCommand := &cobra.Command{
		Use:     "auth",
		Short:   "Authentication and config commands",
		Args:    cobra.NoArgs,
		GroupID: commandGroupAuth,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	authCommand.AddCommand(
		newAuthLoginCommand(options),
		newAuthStatusCommand(options),
		newAuthLogoutCommand(options),
		newAuthPathCommand(options),
		newAuthOAuthCommand(options),
		newAuthConnectCommand(options),
	)

	return authCommand
}

// buildLoginCommand is the canonical sign-in command definition used by both
// newAuthLoginCommand and newLoginCommand. It defaults to OAuth using the
// embedded DailyDrop app credentials. Passing --api-key opts into API key auth.
func buildLoginCommand(options Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Sign in to Beehiiv",
		Long: strings.TrimSpace(`
Sign in to Beehiiv using OAuth. Your browser opens the Beehiiv authorization
page and your credentials are saved securely in the OS keyring automatically.
No API key or client ID required.

For CI/CD or programmatic use, pass --api-key to authenticate with an API
key instead of OAuth.`),
		Example: strings.TrimSpace(`
beehiiv login
beehiiv login --no-browser
beehiiv login --api-key YOUR_API_KEY
beehiiv login --api-key YOUR_API_KEY --publication-id pub_123`),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			overrides, err := commandOverrides(cmd)
			if err != nil {
				return err
			}

			// Only use API key mode when --api-key was explicitly passed on the
			// command line (overrides.APIKey is set only when the flag Changed).
			if overrides.APIKey != "" {
				return runAPIKeyLoginFlow(cmd, options, overrides.APIKey)
			}

			// Precedence: flags > env vars > embedded DailyDrop defaults.
			clientID, _ := cmd.Flags().GetString("client-id")
			clientSecret, _ := cmd.Flags().GetString("client-secret")
			redirectURI, _ := cmd.Flags().GetString("redirect-uri")
			scopes, _ := cmd.Flags().GetStringArray("scope")

			clientID = firstNonEmpty(strings.TrimSpace(clientID), options.Env[config.EnvOAuthClientID], appOAuthClientID)
			clientSecret = firstNonEmpty(strings.TrimSpace(clientSecret), options.Env[config.EnvOAuthClientSecret], appOAuthClientSecret)
			redirectURI = firstNonEmpty(strings.TrimSpace(redirectURI), options.Env[config.EnvOAuthRedirectURI], appOAuthRedirectURI)
			scopes = mergeScopes(scopes, options.Env[config.EnvOAuthScopes])
			scopes = auth.NormalizeScopes(scopes)

			// If redirect URI is external (relay) the CLI still listens on
			// localhost for the relay's 302.  If it IS localhost, listen there.
			listenURI := ""
			if !isLocalhostURI(redirectURI) {
				listenURI = appOAuthLoopbackURI
			}

			noBrowser, _ := cmd.Flags().GetBool("no-browser")
			manual, _ := cmd.Flags().GetBool("manual")
			return runOAuthLoginFlow(cmd.Context(), cmd, options, oauthLoginParams{
				ClientID:     clientID,
				ClientSecret: clientSecret,
				RedirectURI:  redirectURI,
				ListenURI:    listenURI,
				Scopes:       scopes,
				NoBrowser:    noBrowser,
				Manual:       manual,
			})
		},
	}
	cmd.Flags().String("client-id", "", "Override the OAuth client ID")
	cmd.Flags().String("client-secret", "", "Override the OAuth client secret")
	cmd.Flags().String("redirect-uri", "", "Override the OAuth redirect URI")
	cmd.Flags().StringArray("scope", nil, "Requested OAuth scope; repeat or use 'all'")
	cmd.Flags().Bool("no-browser", false, "Print the authorization URL without opening a browser")
	cmd.Flags().Bool("manual", false, "Skip the local callback listener and paste the callback URL manually")
	return cmd
}

func newAuthLoginCommand(options Options) *cobra.Command {
	return buildLoginCommand(options)
}

// runAPIKeyLoginFlow saves an API key session. If apiKey is empty it prompts
// the user interactively.
func runAPIKeyLoginFlow(cmd *cobra.Command, options Options, apiKey string) error {
	manager := auth.NewManager(options.Env, options.HTTPClient)
	overrides, err := commandOverrides(cmd)
	if err != nil {
		return err
	}

	runtimeConfig, err := config.LoadRuntime(overrides, options.Env)
	if err != nil {
		return err
	}

	if apiKey == "" {
		cmd.PrintErrln("Enter your Beehiiv API key. Create one at: https://developers.beehiiv.com/welcome/create-an-api-key")
		apiKey, err = promptValue(cmd.InOrStdin(), cmd.ErrOrStderr(), "API key")
		if err != nil {
			return err
		}
	}

	publicationID := overrides.PublicationID
	if publicationID == "" {
		publicationID, err = selectPublicationID(cmd.Context(), options.HTTPClient, cmd.InOrStdin(), cmd.ErrOrStderr(), runtimeConfig, apiKey)
		if err != nil {
			return err
		}
	}

	if err := manager.SaveAPIKeySession(auth.APIKeyLoginOptions{
		SettingsPath:  runtimeConfig.ConfigPath,
		APIKey:        apiKey,
		PublicationID: publicationID,
		BaseURL:       runtimeConfig.BaseURL,
		RateLimitRPM:  runtimeConfig.RateLimitRPM,
	}); err != nil {
		return err
	}

	status, err := manager.Status(overrides)
	if err != nil {
		return err
	}
	return writeCommandOutput(cmd, options.Env, map[string]any{
		"message":        "Beehiiv credentials saved in the OS keyring",
		"auth_mode":      status.AuthMode,
		"publication_id": status.PublicationID,
		"settings_path":  status.SettingsPath,
		"secret_backend": status.SecretBackend,
	})
}

func newAuthOAuthCommand(options Options) *cobra.Command {
	oauthCommand := &cobra.Command{
		Use:   "oauth",
		Short: "OAuth authentication for custom Beehiiv OAuth apps",
		Long: "Advanced OAuth authentication using your own Beehiiv OAuth app.\n\n" +
			"If you just want to sign in, run `beehiiv login` instead — no client ID needed.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	loginCommand := &cobra.Command{
		Use:   "login",
		Short: "Sign in with a custom Beehiiv OAuth app",
		Long: "Authorize beehiiv-cli via the OAuth 2.0 authorization-code flow with PKCE\n" +
			"using a client ID from your own Beehiiv OAuth app.\n\n" +
			"For the default sign-in experience (no client ID required), use `beehiiv login`.",
		Example: strings.TrimSpace(`
beehiiv auth oauth login --client-id <id>
beehiiv auth oauth login --client-id <id> --scope all
beehiiv auth oauth login --client-id <id> --manual --no-browser`),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientID, _ := cmd.Flags().GetString("client-id")
			clientSecret, _ := cmd.Flags().GetString("client-secret")
			redirectURI, _ := cmd.Flags().GetString("redirect-uri")
			manual, _ := cmd.Flags().GetBool("manual")
			noBrowser, _ := cmd.Flags().GetBool("no-browser")
			scopes, _ := cmd.Flags().GetStringArray("scope")

			clientID = firstNonEmpty(strings.TrimSpace(clientID), strings.TrimSpace(options.Env[config.EnvOAuthClientID]))
			clientSecret = firstNonEmpty(strings.TrimSpace(clientSecret), strings.TrimSpace(options.Env[config.EnvOAuthClientSecret]))
			redirectURI = firstNonEmpty(strings.TrimSpace(redirectURI), strings.TrimSpace(options.Env[config.EnvOAuthRedirectURI]), defaultOAuthRedirectURI)
			scopes = mergeScopes(scopes, options.Env[config.EnvOAuthScopes])
			scopes = auth.NormalizeScopes(scopes)

			if clientID == "" {
				return errors.New("oauth client id is required; pass --client-id or set BEEHIIV_OAUTH_CLIENT_ID")
			}

			return runOAuthLoginFlow(cmd.Context(), cmd, options, oauthLoginParams{
				ClientID:     clientID,
				ClientSecret: clientSecret,
				RedirectURI:  redirectURI,
				Scopes:       scopes,
				NoBrowser:    noBrowser,
				Manual:       manual,
			})
		},
	}

	loginCommand.Flags().String("client-id", "", "Beehiiv OAuth client ID")
	loginCommand.Flags().String("client-secret", "", "Beehiiv OAuth client secret for confidential clients")
	loginCommand.Flags().String("redirect-uri", defaultOAuthRedirectURI, "Registered OAuth redirect URI")
	loginCommand.Flags().StringArray("scope", []string{"default"}, "Requested OAuth scope; repeat or use 'all'")
	loginCommand.Flags().Bool("manual", false, "Skip the local callback listener and paste the callback URL manually")
	loginCommand.Flags().Bool("no-browser", false, "Print the authorization URL without opening a browser")

	oauthCommand.AddCommand(loginCommand)
	return oauthCommand
}

// newAuthConnectCommand is an alias for the login command kept for backward
// compatibility. Prefer `beehiiv login` for new usage.
func newAuthConnectCommand(options Options) *cobra.Command {
	cmd := buildLoginCommand(options)
	cmd.Use = "connect"
	cmd.Short = "Sign in to Beehiiv (alias for login)"
	cmd.Example = strings.TrimSpace(`
beehiiv connect
beehiiv auth connect
beehiiv connect --no-browser
beehiiv connect --api-key YOUR_API_KEY`)
	return cmd
}

// ---------------------------------------------------------------------------
// Shared OAuth login flow
// ---------------------------------------------------------------------------

// oauthLoginParams carries the inputs for runOAuthLoginFlow.
type oauthLoginParams struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string   // URI registered with beehiiv (sent in the authorize URL)
	ListenURI    string   // Loopback URI to listen on; if empty, uses RedirectURI
	Scopes       []string
	NoBrowser    bool
	Manual       bool
}

// runOAuthLoginFlow executes the full OAuth authorization-code + PKCE flow:
// generate PKCE, open browser, wait for callback, exchange code, save session.
func runOAuthLoginFlow(ctx context.Context, cmd *cobra.Command, options Options, params oauthLoginParams) error {
	manager := auth.NewManager(options.Env, options.HTTPClient)
	overrides, err := commandOverrides(cmd)
	if err != nil {
		return err
	}
	runtimeConfig, err := config.LoadRuntime(overrides, options.Env)
	if err != nil {
		return err
	}

	state, err := auth.GenerateState()
	if err != nil {
		return err
	}
	verifier, challenge, err := auth.GeneratePKCEVerifier()
	if err != nil {
		return err
	}
	authorizeURL, err := auth.BuildAuthorizeURL(params.ClientID, params.RedirectURI, state, challenge, params.Scopes)
	if err != nil {
		return err
	}

	// The URI the loopback server actually listens on.  For the relay flow
	// this differs from RedirectURI (e.g. listen on localhost while beehiiv
	// redirects to api.dailydrop.com).
	listenURI := params.ListenURI
	if listenURI == "" {
		listenURI = params.RedirectURI
	}
	isRelay := listenURI != params.RedirectURI

	if isRelay {
		cmd.PrintErrf("Opening your browser to authorize beehiiv-cli...\n\nAuthorization URL:\n%s\n\n", authorizeURL)
	} else {
		cmd.PrintErrf("Open this URL to authorize beehiiv-cli:\n%s\n\n", authorizeURL)
	}

	manual := params.Manual
	// If the listen URI is not localhost the CLI cannot bind a listener.
	if !manual && !isLocalhostURI(listenURI) {
		fmt.Fprintln(cmd.ErrOrStderr(), "Note: redirect URI is not localhost — switching to manual mode.")
		fmt.Fprintln(cmd.ErrOrStderr(), "After authorizing, paste the full callback URL from your browser.")
		manual = true
	}

	var callbackURL string
	if !manual {
		callbackURL, err = waitForOAuthCallback(ctx, cmd.InOrStdin(), listenURI, state, authorizeURL, !params.NoBrowser, cmd.ErrOrStderr())
		if err != nil {
			var portErr *errPortBusy
			if errors.As(err, &portErr) {
				if isRelay {
					return fmt.Errorf("%w\n\nAnother process is using port 3008. Free that port and run `beehiiv connect` again.\n(The authorization URL was never opened, so no credentials were exposed.)", err)
				}
				return fmt.Errorf("%w\n\nFree port %s or pass a different --redirect-uri", err, portErr.host)
			}
			return err
		}
	} else {
		if !params.NoBrowser {
			_ = openBrowser(authorizeURL)
		}
		callbackURL, err = promptValue(cmd.InOrStdin(), cmd.ErrOrStderr(), "Paste the full callback URL")
		if err != nil {
			return err
		}
	}

	code, returnedState, err := parseCallbackURL(callbackURL)
	if err != nil {
		return err
	}
	if returnedState != state {
		return fmt.Errorf("oauth state mismatch; expected %q", state)
	}

	tokenResponse, err := auth.ExchangeAuthorizationCode(ctx, options.HTTPClient, auth.TokenExchangeRequest{
		ClientID:     params.ClientID,
		ClientSecret: params.ClientSecret,
		Code:         code,
		RedirectURI:  params.RedirectURI,
		CodeVerifier: verifier,
	})
	if err != nil {
		return err
	}

	tokenInfo, tokenInfoErr := auth.GetTokenInfo(ctx, options.HTTPClient, tokenResponse.AccessToken)

	publicationID := overrides.PublicationID
	if publicationID == "" && hasScope(params.Scopes, "publications:read") {
		publicationID, err = selectPublicationID(ctx, options.HTTPClient, cmd.InOrStdin(), cmd.ErrOrStderr(), runtimeConfig, tokenResponse.AccessToken)
		if err != nil {
			return err
		}
	}

	saveOptions := auth.OAuthSessionOptions{
		SettingsPath:    runtimeConfig.ConfigPath,
		ClientID:        params.ClientID,
		ClientSecret:    params.ClientSecret,
		RedirectURI:     params.RedirectURI,
		PublicationID:   publicationID,
		BaseURL:         runtimeConfig.BaseURL,
		RateLimitRPM:    runtimeConfig.RateLimitRPM,
		RequestedScopes: params.Scopes,
		TokenResponse:   tokenResponse,
	}
	if tokenInfoErr == nil {
		saveOptions.TokenInfo = &tokenInfo
	}
	if err := manager.SaveOAuthSession(saveOptions); err != nil {
		return err
	}

	status, err := manager.Status(overrides)
	if err != nil {
		return err
	}

	payload := map[string]any{
		"message":        "Beehiiv OAuth session saved in the OS keyring",
		"auth_mode":      status.AuthMode,
		"publication_id": status.PublicationID,
		"settings_path":  status.SettingsPath,
		"secret_backend": status.SecretBackend,
		"oauth_scopes":   status.OAuthScopes,
	}
	if status.OAuthClientID != "" {
		payload["oauth_client_id"] = status.OAuthClientID
	}
	if status.TokenExpiresAt != "" {
		payload["token_expires_at"] = status.TokenExpiresAt
	}
	if tokenInfoErr != nil {
		payload["token_info_warning"] = tokenInfoErr.Error()
	}
	return writeCommandOutput(cmd, options.Env, payload)
}

// errPortBusy is returned by waitForOAuthCallback when the loopback port is
// already in use.  Callers can detect it with errors.As to emit context-
// specific advice (e.g. which port to free or which flag to change).
type errPortBusy struct {
	host  string
	cause error
}

func (e *errPortBusy) Error() string {
	return fmt.Sprintf("listen for OAuth callback on %s: %s", e.host, e.cause)
}

func (e *errPortBusy) Unwrap() error { return e.cause }

// isLocalhostURI reports whether rawURL is a loopback address (localhost,
// 127.0.0.1, or ::1) so the CLI knows it can start a local listener for it.
func isLocalhostURI(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	h := parsed.Hostname()
	return h == "localhost" || h == "127.0.0.1" || h == "::1"
}

func newAuthStatusCommand(options Options) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show masked auth status without printing live credentials",
		Long: "Display the current authentication state: auth mode, publication ID,\n" +
			"settings path, and token metadata — without revealing secrets.",
		Example: strings.TrimSpace(`
beehiiv auth status
beehiiv auth status --output table`),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			manager := auth.NewManager(options.Env, options.HTTPClient)
			overrides, err := commandOverrides(cmd)
			if err != nil {
				return err
			}
			status, err := manager.Status(overrides)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, options.Env, status)
		},
	}
}

func newAuthPathCommand(options Options) *cobra.Command {
	return &cobra.Command{
		Use:     "path",
		Short:   "Print the settings file path",
		Example: "beehiiv auth path",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			manager := auth.NewManager(options.Env, options.HTTPClient)
			overrides, err := commandOverrides(cmd)
			if err != nil {
				return err
			}
			paths, err := manager.Paths(overrides.ConfigPath)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, options.Env, paths)
		},
	}
}

func newAuthLogoutCommand(options Options) *cobra.Command {
	command := &cobra.Command{
		Use:   "logout",
		Short: "Remove saved Beehiiv credentials and optionally revoke OAuth tokens",
		Long: "Delete all stored credentials from the OS keyring and reset the config\n" +
			"file.  For OAuth sessions, also revokes the token server-side by default.",
		Example: strings.TrimSpace(`
beehiiv auth logout
beehiiv auth logout --revoke=false`),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			manager := auth.NewManager(options.Env, options.HTTPClient)
			overrides, err := commandOverrides(cmd)
			if err != nil {
				return err
			}
			revoke, err := cmd.Flags().GetBool("revoke")
			if err != nil {
				return err
			}
			revokeErr := manager.Logout(cmd.Context(), overrides.ConfigPath, revoke)
			payload := map[string]any{
				"message": "Beehiiv credentials cleared",
				"revoked": revokeErr == nil && revoke,
			}
			if revokeErr != nil {
				payload["revoke_warning"] = revokeErr.Error()
			}
			return writeCommandOutput(cmd, options.Env, payload)
		},
	}
	command.Flags().Bool("revoke", true, "Revoke the saved OAuth token before deleting the local session")
	return command
}

func newLoginCommand(options Options) *cobra.Command {
	cmd := buildLoginCommand(options)
	cmd.GroupID = commandGroupAuth
	return cmd
}

func writeCommandOutput(cmd *cobra.Command, env map[string]string, value any) error {
	overrides, err := commandOverrides(cmd)
	if err != nil {
		return err
	}
	runtimeConfig, err := config.LoadRuntime(overrides, env)
	if err != nil {
		return err
	}
	return clioutput.Write(cmd.OutOrStdout(), normalizeOutputValue(value), nil, runtimeConfig)
}

func normalizeOutputValue(value any) any {
	switch value.(type) {
	case map[string]any, []any:
		return value
	default:
		data, err := json.Marshal(value)
		if err != nil {
			return value
		}
		var normalized any
		if err := json.Unmarshal(data, &normalized); err != nil {
			return value
		}
		return normalized
	}
}

func selectPublicationID(ctx context.Context, httpClient client.HTTPClient, stdin io.Reader, stderr io.Writer, runtimeConfig config.Runtime, token string) (string, error) {
	loginRuntime := runtimeConfig
	loginRuntime.APIKey = token
	loginRuntime.PublicationID = ""
	apiClient := client.New(loginRuntime, httpClient, stderr)

	operation, found, err := commandset.Find("publications", "list")
	if err != nil {
		return "", err
	}
	if !found {
		return "", errors.New("publications list operation is unavailable")
	}

	response, err := apiClient.Execute(ctx, operation, map[string]string{}, url.Values{}, nil)
	if err != nil {
		return "", err
	}

	var payload struct {
		Data []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body, &payload); err != nil {
		return "", err
	}
	if len(payload.Data) == 0 {
		return "", errors.New("no publications were returned for this account")
	}
	if len(payload.Data) == 1 {
		fmt.Fprintf(stderr, "Using publication %s (%s)\n", payload.Data[0].ID, payload.Data[0].Name)
		return payload.Data[0].ID, nil
	}
	return promptPublicationChoice(stdin, stderr, payload.Data)
}

func promptValue(input io.Reader, output io.Writer, label string) (string, error) {
	if label != "" {
		fmt.Fprintf(output, "%s: ", label)
	}
	reader := bufio.NewReader(input)
	value, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	return strings.TrimSpace(value), nil
}

func promptPublicationChoice(input io.Reader, output io.Writer, publications []struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}) (string, error) {
	fmt.Fprintln(output, "Select a publication ID:")
	for index, publication := range publications {
		fmt.Fprintf(output, "%d. %s (%s)\n", index+1, publication.ID, publication.Name)
	}
	for {
		value, err := promptValue(input, output, "Publication number")
		if err != nil {
			return "", err
		}
		choice, parseErr := strconv.Atoi(strings.TrimSpace(value))
		if parseErr == nil && choice >= 1 && choice <= len(publications) {
			return publications[choice-1].ID, nil
		}
		fmt.Fprintf(output, "Please enter a number between 1 and %d.\n", len(publications))
	}
}

func mergeScopes(scopes []string, envValue string) []string {
	if trimmed := strings.TrimSpace(envValue); trimmed != "" {
		for _, part := range strings.FieldsFunc(trimmed, func(r rune) bool {
			return r == ',' || r == ' '
		}) {
			scopes = append(scopes, part)
		}
	}
	return scopes
}

func hasScope(scopes []string, target string) bool {
	for _, scope := range scopes {
		if scope == target || scope == "all" {
			return true
		}
	}
	return false
}

func waitForOAuthCallback(ctx context.Context, stdin io.Reader, redirectURI, state, authorizeURL string, open bool, stderr io.Writer) (string, error) {
	host, path, err := auth.BuildLoopbackCallback(redirectURI)
	if err != nil {
		return "", err
	}

	listener, err := net.Listen("tcp", host)
	if err != nil {
		return "", &errPortBusy{host: host, cause: err}
	}
	defer listener.Close()

	callbackCh := make(chan string, 1)
	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != path {
				http.NotFound(w, r)
				return
			}
			if returnedState := r.URL.Query().Get("state"); returnedState != state {
				http.Error(w, "state mismatch", http.StatusBadRequest)
				return
			}
			fmt.Fprintln(w, "beehiiv-cli authentication complete. You can close this tab.")
			callbackCh <- r.URL.String()
		}),
	}

	go func() {
		_ = server.Serve(listener)
	}()
	defer server.Shutdown(context.Background())

	if open {
		if err := openBrowser(authorizeURL); err != nil {
			fmt.Fprintf(stderr, "Could not open a browser automatically: %v\n", err)
		}
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	select {
	case <-timeoutCtx.Done():
		fmt.Fprintln(stderr, "Timed out waiting for the browser callback. Paste the full callback URL from your browser address bar.")
		return promptValue(stdin, stderr, "Callback URL")
	case rawURL := <-callbackCh:
		return rawURL, nil
	}
}

func parseCallbackURL(raw string) (string, string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", "", err
	}
	code := parsed.Query().Get("code")
	if code == "" {
		return "", "", errors.New("callback URL is missing the authorization code")
	}
	return code, parsed.Query().Get("state"), nil
}

func openBrowser(rawURL string) error {
	var command *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		command = exec.Command("open", rawURL)
	case "windows":
		command = exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL)
	default:
		command = exec.Command("xdg-open", rawURL)
	}
	return command.Start()
}
