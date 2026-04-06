package config

import (
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultConfigPathFor(t *testing.T) {
	t.Parallel()

	macPath, err := DefaultConfigPathFor("darwin", "/Users/tester", "")
	if err != nil {
		t.Fatalf("DefaultConfigPathFor macOS returned error: %v", err)
	}
	if want := filepath.Join("/Users/tester", "Library", "Application Support", "beehiiv-cli", "config.json"); macPath != want {
		t.Fatalf("DefaultConfigPathFor macOS = %q, want %q", macPath, want)
	}

	windowsPath, err := DefaultConfigPathFor("windows", "", `C:\Users\tester\AppData\Roaming`)
	if err != nil {
		t.Fatalf("DefaultConfigPathFor Windows returned error: %v", err)
	}
	if want := filepath.Join(`C:\Users\tester\AppData\Roaming`, "beehiiv-cli", "config.json"); windowsPath != want {
		t.Fatalf("DefaultConfigPathFor Windows = %q, want %q", windowsPath, want)
	}
}

func TestLoadRuntimeUsesOverrideEnvAndSettingsPrecedence(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	if err := SaveSettings(configPath, Settings{
		SchemaVersion: SettingsSchemaV1,
		PublicationID: "file-pub",
		RateLimitRPM:  90,
	}); err != nil {
		t.Fatalf("SaveSettings returned error: %v", err)
	}

	runtime, err := LoadRuntime(Overrides{
		ConfigPath:    configPath,
		PublicationID: "override-pub",
		Timeout:       45 * time.Second,
	}, map[string]string{
		EnvBearerToken: "env-bearer",
		EnvBaseURL:     "https://custom.example/v2",
	})
	if err != nil {
		t.Fatalf("LoadRuntime returned error: %v", err)
	}

	if runtime.APIKey != "env-bearer" {
		t.Fatalf("LoadRuntime APIKey = %q, want env-bearer", runtime.APIKey)
	}
	if runtime.PublicationID != "override-pub" {
		t.Fatalf("LoadRuntime PublicationID = %q, want override-pub", runtime.PublicationID)
	}
	if runtime.BaseURL != "https://custom.example/v2" {
		t.Fatalf("LoadRuntime BaseURL = %q, want https://custom.example/v2", runtime.BaseURL)
	}
	if runtime.RateLimitRPM != 90 {
		t.Fatalf("LoadRuntime RateLimitRPM = %d, want 90", runtime.RateLimitRPM)
	}
	if runtime.Timeout != 45*time.Second {
		t.Fatalf("LoadRuntime Timeout = %s, want 45s", runtime.Timeout)
	}
}

func TestValidateConfigPathRejectsLegacyEnvPath(t *testing.T) {
	t.Parallel()

	input := filepath.Join("/tmp", ".env")
	got, err := ValidateConfigPath(input)
	if err == nil {
		t.Fatal("ValidateConfigPath should reject legacy .env paths")
	}
	if got != "" {
		t.Fatalf("ValidateConfigPath(%q) = %q, want empty string", input, got)
	}
}
