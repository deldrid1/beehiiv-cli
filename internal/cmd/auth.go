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
		newAuthOAuthCommand(options),
		newAuthStatusCommand(options),
		newAuthPathCommand(options),
		newAuthLogoutCommand(options),
	)

	return authCommand
}

func newAuthLoginCommand(options Options) *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Save an API key and publication ID securely",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			manager := auth.NewManager(options.Env, options.HTTPClient)
			overrides, err := commandOverrides(cmd)
			if err != nil {
				return err
			}

			runtimeConfig, err := config.LoadRuntime(overrides, options.Env)
			if err != nil {
				return err
			}

			apiKey := firstNonEmpty(overrides.APIKey, options.Env[config.EnvAPIKey], options.Env[config.EnvBearerToken])
			if apiKey == "" {
				cmd.PrintErrln("Enter your Beehiiv API key. Create one as described here: https://developers.beehiiv.com/welcome/create-an-api-key")
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
		},
	}
}

func newAuthOAuthCommand(options Options) *cobra.Command {
	oauthCommand := &cobra.Command{
		Use:   "oauth",
		Short: "OAuth authentication flows",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	loginCommand := &cobra.Command{
		Use:   "login",
		Short: "Run the Beehiiv OAuth authorization-code flow with PKCE",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			manager := auth.NewManager(options.Env, options.HTTPClient)
			overrides, err := commandOverrides(cmd)
			if err != nil {
				return err
			}
			runtimeConfig, err := config.LoadRuntime(overrides, options.Env)
			if err != nil {
				return err
			}

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

			state, err := auth.GenerateState()
			if err != nil {
				return err
			}
			verifier, challenge, err := auth.GeneratePKCEVerifier()
			if err != nil {
				return err
			}
			authorizeURL, err := auth.BuildAuthorizeURL(clientID, redirectURI, state, challenge, scopes)
			if err != nil {
				return err
			}

			cmd.PrintErrf("Open this URL to authorize beehiiv-cli:\n%s\n\n", authorizeURL)

			var callbackURL string
			if !manual {
				callbackURL, err = waitForOAuthCallback(cmd.Context(), cmd.InOrStdin(), redirectURI, state, authorizeURL, !noBrowser, cmd.ErrOrStderr())
				if err != nil {
					return err
				}
			} else {
				if !noBrowser {
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

			tokenResponse, err := auth.ExchangeAuthorizationCode(cmd.Context(), options.HTTPClient, auth.TokenExchangeRequest{
				ClientID:     clientID,
				ClientSecret: clientSecret,
				Code:         code,
				RedirectURI:  redirectURI,
				CodeVerifier: verifier,
			})
			if err != nil {
				return err
			}

			tokenInfo, tokenInfoErr := auth.GetTokenInfo(cmd.Context(), options.HTTPClient, tokenResponse.AccessToken)

			publicationID := overrides.PublicationID
			if publicationID == "" && hasScope(scopes, "publications:read") {
				publicationID, err = selectPublicationID(cmd.Context(), options.HTTPClient, cmd.InOrStdin(), cmd.ErrOrStderr(), runtimeConfig, tokenResponse.AccessToken)
				if err != nil {
					return err
				}
			}

			saveOptions := auth.OAuthSessionOptions{
				SettingsPath:    runtimeConfig.ConfigPath,
				ClientID:        clientID,
				ClientSecret:    clientSecret,
				RedirectURI:     redirectURI,
				PublicationID:   publicationID,
				BaseURL:         runtimeConfig.BaseURL,
				RateLimitRPM:    runtimeConfig.RateLimitRPM,
				RequestedScopes: scopes,
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
				"message":         "Beehiiv OAuth session saved in the OS keyring",
				"auth_mode":       status.AuthMode,
				"publication_id":  status.PublicationID,
				"settings_path":   status.SettingsPath,
				"secret_backend":  status.SecretBackend,
				"oauth_client_id": status.OAuthClientID,
				"oauth_scopes":    status.OAuthScopes,
			}
			if status.TokenExpiresAt != "" {
				payload["token_expires_at"] = status.TokenExpiresAt
			}
			if tokenInfoErr != nil {
				payload["token_info_warning"] = tokenInfoErr.Error()
			}
			return writeCommandOutput(cmd, options.Env, payload)
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

func newAuthStatusCommand(options Options) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show masked auth status without printing live credentials",
		Args:  cobra.NoArgs,
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
		Use:   "path",
		Short: "Print the settings file path",
		Args:  cobra.NoArgs,
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
		Args:  cobra.NoArgs,
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
	return &cobra.Command{
		Use:     "login",
		Short:   "Alias for auth login",
		Args:    cobra.NoArgs,
		GroupID: commandGroupAuth,
		RunE: func(cmd *cobra.Command, args []string) error {
			return newAuthLoginCommand(options).RunE(cmd, args)
		},
	}
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
		return "", fmt.Errorf("listen for oauth callback on %s: %w; choose a different configured --redirect-uri if this port is already in use", host, err)
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
