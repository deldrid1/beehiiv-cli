package config

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultBaseURL      = "https://api.beehiiv.com/v2"
	DefaultRateLimitRPM = 150
	HardMaxRateLimitRPM = 180
	DefaultTimeout      = 30 * time.Second
	DefaultOutput       = OutputJSON

	EnvAuthMode          = "BEEHIIV_AUTH_MODE"
	EnvAPIKey            = "BEEHIIV_API_KEY"
	EnvBearerToken       = "BEEHIIV_BEARER_TOKEN"
	EnvPublicationID     = "BEEHIIV_PUBLICATION_ID"
	EnvBaseURL           = "BEEHIIV_BASE_URL"
	EnvRateLimitRPM      = "BEEHIIV_RATE_LIMIT_RPM"
	EnvOAuthClientID     = "BEEHIIV_OAUTH_CLIENT_ID"
	EnvOAuthClientSecret = "BEEHIIV_OAUTH_CLIENT_SECRET"
	EnvOAuthRedirectURI  = "BEEHIIV_OAUTH_REDIRECT_URI"
	EnvOAuthScopes       = "BEEHIIV_OAUTH_SCOPES"
	EnvLiveTests         = "BEEHIIV_LIVE_TESTS"
)

const (
	OutputJSON  = "json"
	OutputTable = "table"
	OutputRaw   = "raw"
)

type Runtime struct {
	ConfigPath    string
	APIKey        string
	PublicationID string
	BaseURL       string
	RateLimitRPM  int
	Timeout       time.Duration
	Output        string
	Compact       bool
	Debug         bool
	Verbose       bool
}

type Overrides struct {
	ConfigPath    string
	APIKey        string
	PublicationID string
	BaseURL       string
	RateLimitRPM  int
	Timeout       time.Duration
	Output        string
	Compact       bool
	Debug         bool
	Verbose       bool
}

func DefaultConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return DefaultConfigPathFor(runtimeGOOS(), home, os.Getenv("APPDATA"))
}

func LoadRuntime(overrides Overrides, env map[string]string) (Runtime, error) {
	configPath, err := ValidateConfigPath(overrides.ConfigPath)
	if err != nil {
		return Runtime{}, err
	}
	if configPath == "" {
		configPath, err = DefaultConfigPath()
		if err != nil {
			return Runtime{}, err
		}
	}

	settings, err := LoadSettings(configPath)
	if err != nil {
		return Runtime{}, err
	}

	apiKey := firstNonEmpty(
		strings.TrimSpace(overrides.APIKey),
		strings.TrimSpace(env[EnvAPIKey]),
		strings.TrimSpace(env[EnvBearerToken]),
	)
	publicationID := firstNonEmpty(
		strings.TrimSpace(overrides.PublicationID),
		strings.TrimSpace(env[EnvPublicationID]),
		strings.TrimSpace(settings.PublicationID),
	)
	baseURL := strings.TrimRight(firstNonEmpty(
		strings.TrimSpace(overrides.BaseURL),
		strings.TrimSpace(env[EnvBaseURL]),
		strings.TrimSpace(settings.BaseURL),
		DefaultBaseURL,
	), "/")

	rateLimit := DefaultRateLimitRPM
	for _, candidate := range []string{
		intToString(overrides.RateLimitRPM),
		env[EnvRateLimitRPM],
		intToString(settings.RateLimitRPM),
	} {
		if candidate == "" {
			continue
		}
		parsed, parseErr := strconv.Atoi(candidate)
		if parseErr != nil {
			return Runtime{}, fmt.Errorf("invalid %s value %q", EnvRateLimitRPM, candidate)
		}
		rateLimit = clampRateLimit(parsed)
		break
	}

	timeout := DefaultTimeout
	if overrides.Timeout > 0 {
		timeout = overrides.Timeout
	}

	output := strings.ToLower(strings.TrimSpace(overrides.Output))
	if output == "" {
		output = DefaultOutput
	}
	switch output {
	case OutputJSON, OutputTable, OutputRaw:
	default:
		return Runtime{}, fmt.Errorf("invalid output mode %q", output)
	}

	return Runtime{
		ConfigPath:    configPath,
		APIKey:        apiKey,
		PublicationID: publicationID,
		BaseURL:       baseURL,
		RateLimitRPM:  rateLimit,
		Timeout:       timeout,
		Output:        output,
		Compact:       overrides.Compact,
		Debug:         overrides.Debug,
		Verbose:       overrides.Verbose,
	}, nil
}

func MaskSecret(value string) string {
	if value == "" {
		return ""
	}
	if len(value) <= 8 {
		return strings.Repeat("*", len(value))
	}
	return value[:4] + strings.Repeat("*", len(value)-8) + value[len(value)-4:]
}

func clampRateLimit(value int) int {
	if value <= 0 {
		return DefaultRateLimitRPM
	}
	if value > HardMaxRateLimitRPM {
		return HardMaxRateLimitRPM
	}
	return value
}

func runtimeGOOS() string {
	if override := strings.ToLower(os.Getenv("BEEHIIV_CLI_TEST_GOOS_OVERRIDE")); override != "" {
		return override
	}
	return runtime.GOOS
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func intToString(value int) string {
	if value <= 0 {
		return ""
	}
	return strconv.Itoa(value)
}
