package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHelpListsCoreCommands(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := ExecuteContext(context.Background(), []string{"--help"}, Options{
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if exitCode != 0 {
		t.Fatalf("ExecuteContext exit code = %d, stderr = %s", exitCode, stderr.String())
	}
	if !strings.Contains(stdout.String(), "auth") {
		t.Fatalf("help output missing auth command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "reports") {
		t.Fatalf("help output missing reports command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "publications") {
		t.Fatalf("help output missing publications group: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "version") {
		t.Fatalf("help output missing version command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "--output") {
		t.Fatalf("help output missing global output flag: %s", stdout.String())
	}
}

func TestVersionCommandPrintsBuildSummary(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := ExecuteContext(context.Background(), []string{"version"}, Options{
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if exitCode != 0 {
		t.Fatalf("ExecuteContext exit code = %d, stderr = %s", exitCode, stderr.String())
	}
	if !strings.Contains(stdout.String(), "beehiiv version") {
		t.Fatalf("version output missing summary: %s", stdout.String())
	}
}

func TestAuthStatusShowsConfiguredSessionWithoutSecrets(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := ExecuteContext(context.Background(), []string{"auth", "status"}, Options{
		Stdout: &stdout,
		Stderr: &stderr,
		Env: map[string]string{
			"BEEHIIV_API_KEY":        "test-token",
			"BEEHIIV_PUBLICATION_ID": "pub_test",
		},
	})
	if exitCode != 0 {
		t.Fatalf("ExecuteContext exit code = %d, stderr = %s", exitCode, stderr.String())
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

func TestGeneratedActionHelpUsesCobraCommandHelp(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := ExecuteContext(context.Background(), []string{"publications", "list", "--help"}, Options{
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if exitCode != 0 {
		t.Fatalf("ExecuteContext exit code = %d, stderr = %s", exitCode, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Retrieve all publications") && !strings.Contains(stdout.String(), "List publications") {
		t.Fatalf("stdout missing action help summary: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "API path: /publications") {
		t.Fatalf("stdout missing generated API path help: %s", stdout.String())
	}
}

func TestGlobalFlagsPropagateThroughCobra(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := ExecuteContext(context.Background(), []string{"--output", "table", "auth", "status"}, Options{
		Stdout: &stdout,
		Stderr: &stderr,
		Env: map[string]string{
			"BEEHIIV_API_KEY":        "test-token",
			"BEEHIIV_PUBLICATION_ID": "pub_test",
		},
	})
	if exitCode != 0 {
		t.Fatalf("ExecuteContext exit code = %d, stderr = %s", exitCode, stderr.String())
	}
	if !strings.Contains(stdout.String(), "| field") {
		t.Fatalf("stdout missing table header: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "publication_id") {
		t.Fatalf("stdout missing publication_id row: %s", stdout.String())
	}
	if strings.Contains(stdout.String(), "test-token") {
		t.Fatalf("stdout unexpectedly leaked token: %s", stdout.String())
	}
}

// jsonResponseWithHeaders is like jsonResponse but allows setting custom headers.
func jsonResponseWithHeaders(body string, headers http.Header) *http.Response {
	if headers == nil {
		headers = http.Header{}
	}
	if headers.Get("Content-Type") == "" {
		headers.Set("Content-Type", "application/json")
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Header:     headers,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

// ---------------------------------------------------------------------------
// End-to-end tests exercising executeOperation via the Cobra path
// ---------------------------------------------------------------------------

func TestSubscriptionsListAllAggregatesPagesAndSerializesArrayQueries(t *testing.T) {
	t.Parallel()

	requests := 0
	httpClient := stubHTTPClient(func(r *http.Request) (*http.Response, error) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("Authorization header = %q, want Bearer test-token", got)
		}
		if r.URL.Path != "/publications/pub_test/subscriptions" {
			t.Fatalf("request path = %q", r.URL.Path)
		}
		if got := r.URL.Query()["expand[]"]; len(got) != 2 || got[0] != "stats" || got[1] != "custom_fields" {
			t.Fatalf("expand[] query = %#v", got)
		}
		if got := r.URL.Query().Get("status"); got != "active" {
			t.Fatalf("status query = %q, want active", got)
		}
		requests++
		if requests == 1 {
			return jsonResponse(`{"data":[{"id":"sub_1"},{"id":"sub_2"}],"pagination":{"has_more":true,"next_cursor":"next"}}`), nil
		}
		if got := r.URL.Query().Get("cursor"); got != "next" {
			t.Fatalf("cursor query = %q, want next", got)
		}
		return jsonResponse(`{"data":[{"id":"sub_3"}],"pagination":{"has_more":false,"next_cursor":null}}`), nil
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := ExecuteContext(context.Background(), []string{
		"subscriptions", "list",
		"--api-key", "test-token",
		"--publication-id", "pub_test",
		"--base-url", "https://example.test",
		"--all",
		"--query", "expand=stats,custom_fields",
		"--query", "status=active",
	}, Options{Stdout: &stdout, Stderr: &stderr, HTTPClient: httpClient})
	if exitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %s", exitCode, stderr.String())
	}

	var payload struct {
		Data       []map[string]string `json:"data"`
		Pagination struct {
			Mode          string `json:"mode"`
			PagesFetched  int    `json:"pages_fetched"`
			ItemsReturned int    `json:"items_returned"`
		} `json:"pagination"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout.String())
	}
	if len(payload.Data) != 3 {
		t.Fatalf("aggregated data length = %d, want 3", len(payload.Data))
	}
	if payload.Pagination.Mode != "cursor" {
		t.Fatalf("pagination mode = %q, want cursor", payload.Pagination.Mode)
	}
	if payload.Pagination.PagesFetched != 2 {
		t.Fatalf("pages_fetched = %d, want 2", payload.Pagination.PagesFetched)
	}
}

func TestCreateCustomFieldReadsBodyFromFile(t *testing.T) {
	t.Parallel()

	httpClient := stubHTTPClient(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost {
			t.Fatalf("request method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/publications/pub_test/custom_fields" {
			t.Fatalf("request path = %q", r.URL.Path)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("ReadAll returned error: %v", err)
		}
		if string(body) != `{"kind":"string","display":"Codex Test"}` {
			t.Fatalf("request body = %s", string(body))
		}
		return jsonResponse(`{"data":{"id":"field_1","kind":"string","display":"Codex Test"}}`), nil
	})

	tempDir := t.TempDir()
	bodyPath := filepath.Join(tempDir, "body.json")
	if err := os.WriteFile(bodyPath, []byte(`{"kind":"string","display":"Codex Test"}`), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := ExecuteContext(context.Background(), []string{
		"custom-fields", "create",
		"--api-key", "test-token",
		"--publication-id", "pub_test",
		"--base-url", "https://example.test",
		"--body", "@" + bodyPath,
	}, Options{Stdout: &stdout, Stderr: &stderr, HTTPClient: httpClient})
	if exitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %s", exitCode, stderr.String())
	}
}

func TestOutputTableRendersTabularView(t *testing.T) {
	t.Parallel()

	httpClient := stubHTTPClient(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(`{"data":[{"id":"sub_1","email":"one@example.com"},{"id":"sub_2","email":"two@example.com"}]}`), nil
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := ExecuteContext(context.Background(), []string{
		"subscriptions", "list",
		"--api-key", "test-token",
		"--publication-id", "pub_test",
		"--base-url", "https://example.test",
		"--output", "table",
	}, Options{Stdout: &stdout, Stderr: &stderr, HTTPClient: httpClient})
	if exitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %s", exitCode, stderr.String())
	}
	if !strings.Contains(stdout.String(), "| id") || !strings.Contains(stdout.String(), "one@example.com") {
		t.Fatalf("stdout missing rendered table: %s", stdout.String())
	}
}

func TestVerboseAndRawOutputAidTroubleshooting(t *testing.T) {
	t.Parallel()

	httpClient := stubHTTPClient(func(r *http.Request) (*http.Response, error) {
		headers := http.Header{}
		headers.Set("Content-Type", "application/json")
		headers.Set("X-Test-Header", "present")
		return jsonResponseWithHeaders(`{"data":[{"id":"sub_1"}]}`, headers), nil
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := ExecuteContext(context.Background(), []string{
		"subscriptions", "list",
		"--api-key", "test-token",
		"--publication-id", "pub_test",
		"--base-url", "https://example.test",
		"--verbose",
		"--raw",
	}, Options{Stdout: &stdout, Stderr: &stderr, HTTPClient: httpClient})
	if exitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %s", exitCode, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != `{"data":[{"id":"sub_1"}]}` {
		t.Fatalf("stdout = %q, want raw response body", stdout.String())
	}
	if !strings.Contains(stderr.String(), "> GET ") {
		t.Fatalf("stderr missing request line: %s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "Authorization: Bearer [REDACTED]") {
		t.Fatalf("stderr missing redacted authorization header: %s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "< 200 OK") || !strings.Contains(stderr.String(), "X-Test-Header: present") {
		t.Fatalf("stderr missing response trace: %s", stderr.String())
	}
}

func TestConnectCommandAppearsInHelp(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	exitCode := ExecuteContext(context.Background(), []string{"--help"}, Options{Stdout: &stdout})
	if exitCode != 0 {
		t.Fatalf("exit code = %d", exitCode)
	}
	if !strings.Contains(stdout.String(), "connect") {
		t.Fatalf("help missing connect command: %s", stdout.String())
	}
}

func TestLegacyAuthCurrentAliasIsRemoved(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := ExecuteContext(context.Background(), []string{"auth", "current"}, Options{
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if exitCode == 0 {
		t.Fatalf("ExecuteContext exit code = %d, want non-zero", exitCode)
	}
	if !strings.Contains(stderr.String(), "unknown command") {
		t.Fatalf("stderr missing unknown command error: %s", stderr.String())
	}
}
