package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const (
	DefaultConfigName    = "config.json"
	SettingsSchemaV1     = 1
	AuthModeAPIKey       = "api_key"
	AuthModeOAuth        = "oauth"
	SecretBackendKeyring = "keyring"
)

type Settings struct {
	SchemaVersion int           `json:"schema_version"`
	AuthMode      string        `json:"auth_mode,omitempty"`
	SecretBackend string        `json:"secret_backend,omitempty"`
	PublicationID string        `json:"publication_id,omitempty"`
	BaseURL       string        `json:"base_url,omitempty"`
	RateLimitRPM  int           `json:"rate_limit_rpm,omitempty"`
	OAuth         OAuthSettings `json:"oauth,omitempty"`
}

type OAuthSettings struct {
	ClientID        string   `json:"client_id,omitempty"`
	RedirectURI     string   `json:"redirect_uri,omitempty"`
	Scopes          []string `json:"scopes,omitempty"`
	ResourceOwnerID string   `json:"resource_owner_id,omitempty"`
	ApplicationUID  string   `json:"application_uid,omitempty"`
	ApplicationName string   `json:"application_name,omitempty"`
}

func DefaultConfigPathFor(goos, home, appData string) (string, error) {
	switch goos {
	case "darwin":
		if home == "" {
			return "", errors.New("home directory is required for macOS config path")
		}
		return filepath.Join(home, "Library", "Application Support", "beehiiv-cli", DefaultConfigName), nil
	case "windows":
		if appData == "" {
			return "", errors.New("APPDATA is required for Windows config path")
		}
		return filepath.Join(appData, "beehiiv-cli", DefaultConfigName), nil
	default:
		if home == "" {
			return "", errors.New("home directory is required for config path")
		}
		return filepath.Join(home, ".config", "beehiiv-cli", DefaultConfigName), nil
	}
}

func ValidateConfigPath(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", nil
	}
	if filepath.Base(trimmed) == ".env" {
		return "", errors.New("plaintext .env config files are no longer supported; use config.json or omit --config")
	}
	return trimmed, nil
}

func LoadSettings(path string) (Settings, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Settings{SchemaVersion: SettingsSchemaV1}, nil
		}
		return Settings{}, err
	}

	var settings Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return Settings{}, err
	}
	settings.Normalize()
	if settings.SchemaVersion == 0 {
		settings.SchemaVersion = SettingsSchemaV1
	}
	return settings, nil
}

func SaveSettings(path string, settings Settings) error {
	settings.Normalize()
	if settings.SchemaVersion == 0 {
		settings.SchemaVersion = SettingsSchemaV1
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func (s *Settings) Normalize() {
	s.AuthMode = strings.TrimSpace(s.AuthMode)
	s.SecretBackend = strings.TrimSpace(s.SecretBackend)
	s.PublicationID = strings.TrimSpace(s.PublicationID)
	s.BaseURL = strings.TrimRight(strings.TrimSpace(s.BaseURL), "/")
	s.OAuth.ClientID = strings.TrimSpace(s.OAuth.ClientID)
	s.OAuth.RedirectURI = strings.TrimSpace(s.OAuth.RedirectURI)
	s.OAuth.ResourceOwnerID = strings.TrimSpace(s.OAuth.ResourceOwnerID)
	s.OAuth.ApplicationUID = strings.TrimSpace(s.OAuth.ApplicationUID)
	s.OAuth.ApplicationName = strings.TrimSpace(s.OAuth.ApplicationName)

	scopes := make([]string, 0, len(s.OAuth.Scopes))
	seen := make(map[string]struct{}, len(s.OAuth.Scopes))
	for _, scope := range s.OAuth.Scopes {
		scope = strings.TrimSpace(scope)
		if scope == "" {
			continue
		}
		if _, ok := seen[scope]; ok {
			continue
		}
		seen[scope] = struct{}{}
		scopes = append(scopes, scope)
	}
	slices.Sort(scopes)
	s.OAuth.Scopes = scopes
}
