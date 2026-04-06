package testsupport

import (
	"os"
	"testing"

	"github.com/deldrid1/beehiiv-cli/internal/config"
)

type LiveConfig struct {
	APIKey        string
	PublicationID string
}

func RequireLiveConfig(t *testing.T) LiveConfig {
	t.Helper()

	if os.Getenv(config.EnvLiveTests) != "1" {
		t.Skip("live Beehiiv tests are disabled")
	}

	apiKey := os.Getenv(config.EnvAPIKey)
	if apiKey == "" {
		apiKey = os.Getenv(config.EnvBearerToken)
	}
	if apiKey == "" {
		t.Fatal("missing Beehiiv API key for live tests; set BEEHIIV_API_KEY or BEEHIIV_BEARER_TOKEN")
	}

	publicationID := os.Getenv(config.EnvPublicationID)
	if publicationID == "" {
		t.Fatal("missing BEEHIIV_PUBLICATION_ID for live tests")
	}

	return LiveConfig{
		APIKey:        apiKey,
		PublicationID: publicationID,
	}
}
