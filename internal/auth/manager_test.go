package auth

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/deldrid1/beehiiv-cli/internal/config"
)

type fakeKeyringBackend struct {
	values map[string]string
}

func (f *fakeKeyringBackend) Set(service, user, password string) error {
	if f.values == nil {
		f.values = make(map[string]string)
	}
	f.values[service+":"+user] = password
	return nil
}

func (f *fakeKeyringBackend) Get(service, user string) (string, error) {
	value, ok := f.values[service+":"+user]
	if !ok {
		return "", ErrSecretNotFound
	}
	return value, nil
}

func (f *fakeKeyringBackend) Delete(service, user string) error {
	delete(f.values, service+":"+user)
	return nil
}

type stubHTTPClient func(*http.Request) (*http.Response, error)

func (f stubHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestSaveAPIKeySessionStoresSecretInKeyringAndSettings(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	store := NewKeyringStoreWithBackend(DefaultKeyringService, DefaultKeyringUser, &fakeKeyringBackend{})
	manager := NewManagerWithStore(map[string]string{}, nil, store)

	err := manager.SaveAPIKeySession(APIKeyLoginOptions{
		SettingsPath:  configPath,
		APIKey:        "test-token",
		PublicationID: "pub_test",
		BaseURL:       "https://custom.example/v2",
		RateLimitRPM:  120,
	})
	if err != nil {
		t.Fatalf("SaveAPIKeySession returned error: %v", err)
	}

	settings, err := config.LoadSettings(configPath)
	if err != nil {
		t.Fatalf("LoadSettings returned error: %v", err)
	}
	if settings.AuthMode != config.AuthModeAPIKey {
		t.Fatalf("settings auth mode = %q", settings.AuthMode)
	}
	if settings.PublicationID != "pub_test" {
		t.Fatalf("settings publication id = %q", settings.PublicationID)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if strings.Contains(string(data), "test-token") {
		t.Fatalf("settings file unexpectedly contains plaintext token: %s", string(data))
	}

	secret, err := store.Load()
	if err != nil {
		t.Fatalf("store.Load returned error: %v", err)
	}
	if secret.APIKey != "test-token" {
		t.Fatalf("secret API key = %q", secret.APIKey)
	}
}

func TestResolveRuntimeUsesStoredAPIKeyWhenEnvAbsent(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	store := NewKeyringStoreWithBackend(DefaultKeyringService, DefaultKeyringUser, &fakeKeyringBackend{})
	manager := NewManagerWithStore(map[string]string{}, nil, store)

	if err := config.SaveSettings(configPath, config.Settings{
		SchemaVersion: config.SettingsSchemaV1,
		AuthMode:      config.AuthModeAPIKey,
		SecretBackend: config.SecretBackendKeyring,
		PublicationID: "pub_test",
	}); err != nil {
		t.Fatalf("SaveSettings returned error: %v", err)
	}
	if err := store.Save(SecretRecord{APIKey: "stored-token"}); err != nil {
		t.Fatalf("store.Save returned error: %v", err)
	}

	runtimeConfig, err := manager.ResolveRuntime(context.Background(), config.Overrides{ConfigPath: configPath})
	if err != nil {
		t.Fatalf("ResolveRuntime returned error: %v", err)
	}
	if runtimeConfig.APIKey != "stored-token" {
		t.Fatalf("runtime API key = %q", runtimeConfig.APIKey)
	}
	if runtimeConfig.PublicationID != "pub_test" {
		t.Fatalf("runtime publication id = %q", runtimeConfig.PublicationID)
	}
}

func TestResolveRuntimeRefreshesExpiredOAuthTokens(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	store := NewKeyringStoreWithBackend(DefaultKeyringService, DefaultKeyringUser, &fakeKeyringBackend{})
	httpClient := stubHTTPClient(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != TokenURL {
			t.Fatalf("unexpected URL: %s", req.URL.String())
		}
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("ReadAll returned error: %v", err)
		}
		values, err := url.ParseQuery(string(body))
		if err != nil {
			t.Fatalf("ParseQuery returned error: %v", err)
		}
		if values.Get("grant_type") != "refresh_token" {
			t.Fatalf("grant_type = %q", values.Get("grant_type"))
		}
		if values.Get("refresh_token") != "refresh-old" {
			t.Fatalf("refresh_token = %q", values.Get("refresh_token"))
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"access_token":"fresh-token","token_type":"Bearer","expires_in":3600,"refresh_token":"refresh-new","created_at":1712100000}`)),
			Header:     make(http.Header),
		}, nil
	})
	manager := NewManagerWithStore(map[string]string{}, httpClient, store)

	if err := config.SaveSettings(configPath, config.Settings{
		SchemaVersion: config.SettingsSchemaV1,
		AuthMode:      config.AuthModeOAuth,
		SecretBackend: config.SecretBackendKeyring,
		PublicationID: "pub_oauth",
		OAuth: config.OAuthSettings{
			ClientID:    "client_test",
			RedirectURI: "http://localhost:3008/callback",
			Scopes:      []string{"identify:read", "publications:read"},
		},
	}); err != nil {
		t.Fatalf("SaveSettings returned error: %v", err)
	}
	if err := store.Save(SecretRecord{
		ClientSecret: "",
		OAuth: OAuthSecret{
			AccessToken:  "expired-token",
			RefreshToken: "refresh-old",
			TokenType:    "Bearer",
			ExpiresAt:    time.Now().UTC().Add(-time.Minute),
		},
	}); err != nil {
		t.Fatalf("store.Save returned error: %v", err)
	}

	runtimeConfig, err := manager.ResolveRuntime(context.Background(), config.Overrides{ConfigPath: configPath})
	if err != nil {
		t.Fatalf("ResolveRuntime returned error: %v", err)
	}
	if runtimeConfig.APIKey != "fresh-token" {
		t.Fatalf("runtime API key = %q", runtimeConfig.APIKey)
	}

	secret, err := store.Load()
	if err != nil {
		t.Fatalf("store.Load returned error: %v", err)
	}
	if secret.OAuth.RefreshToken != "refresh-new" {
		t.Fatalf("stored refresh token = %q", secret.OAuth.RefreshToken)
	}
}

func TestPathsReturnsOnlySettingsPath(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	store := NewKeyringStoreWithBackend(DefaultKeyringService, DefaultKeyringUser, &fakeKeyringBackend{})
	manager := NewManagerWithStore(map[string]string{}, nil, store)

	paths, err := manager.Paths(configPath)
	if err != nil {
		t.Fatalf("Paths returned error: %v", err)
	}
	if paths.SettingsPath != configPath {
		t.Fatalf("settings path = %q, want %q", paths.SettingsPath, configPath)
	}
}
