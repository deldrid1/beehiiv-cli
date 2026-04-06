package runtime

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestExecutorDelegatesToLegacyCLI(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	executor := NewExecutor(strings.NewReader(""), &stdout, &stderr, map[string]string{
		"BEEHIIV_API_KEY":        "test-token",
		"BEEHIIV_PUBLICATION_ID": "pub_test",
	}, nil)

	exitCode := executor.Run(context.Background(), []string{"auth", "status"})
	if exitCode != 0 {
		t.Fatalf("Run exit code = %d, stderr = %s", exitCode, stderr.String())
	}
	if strings.Contains(stdout.String(), "test-token") {
		t.Fatalf("stdout unexpectedly leaked token: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), `"configured": true`) {
		t.Fatalf("stdout missing configured state: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), `"publication_id": "pub_test"`) {
		t.Fatalf("stdout missing publication_id: %s", stdout.String())
	}
}
