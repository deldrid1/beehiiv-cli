package cli

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

	"github.com/deldrid1/beehiiv-cli/internal/commandset"
)

type stubHTTPClient func(*http.Request) (*http.Response, error)

func (f stubHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return f(req)
}

func jsonResponse(body string, headers http.Header) *http.Response {
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

func TestRootHelpWhenNoCommandIsProvided(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := NewApp(strings.NewReader(""), &stdout, &stderr, map[string]string{}, nil)
	exitCode := app.Run(context.Background(), nil)
	if exitCode != 0 {
		t.Fatalf("Run exit code = %d, want 0", exitCode)
	}
	if !strings.Contains(stdout.String(), "beehiiv auth") {
		t.Fatalf("root help missing auth command: %s", stdout.String())
	}
}

func TestParseTokensSupportsInterleavedFlagsAndPositionals(t *testing.T) {
	t.Parallel()

	parsed, err := parseTokens([]string{"sub_123", "--query", "expand=stats,custom_fields", "--all"}, mergeFlagSpecs(commandset.Operation{}))
	if err != nil {
		t.Fatalf("parseTokens returned error: %v", err)
	}
	if len(parsed.positionals) != 1 || parsed.positionals[0] != "sub_123" {
		t.Fatalf("parseTokens positionals = %#v, want [sub_123]", parsed.positionals)
	}
	if !hasBoolFlag(parsed.flags, "all") {
		t.Fatal("parseTokens did not capture --all")
	}
}

func TestLoadBodySupportsInlineFileAndStdin(t *testing.T) {
	t.Parallel()

	operation := commandset.Operation{Body: true, Command: []string{"custom-fields", "create"}}
	inline, err := loadBody(operation, []string{`{"kind":"string"}`}, strings.NewReader(""))
	if err != nil {
		t.Fatalf("loadBody inline returned error: %v", err)
	}
	if string(inline) != `{"kind":"string"}` {
		t.Fatalf("loadBody inline = %s", string(inline))
	}

	tempDir := t.TempDir()
	bodyPath := filepath.Join(tempDir, "body.json")
	if err := os.WriteFile(bodyPath, []byte(`{"display":"Field"}`), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	fileBody, err := loadBody(operation, []string{"@" + bodyPath}, strings.NewReader(""))
	if err != nil {
		t.Fatalf("loadBody file returned error: %v", err)
	}
	if string(fileBody) != `{"display":"Field"}` {
		t.Fatalf("loadBody file = %s", string(fileBody))
	}

	stdinBody, err := loadBody(operation, []string{"-"}, strings.NewReader(`{"stdin":true}`))
	if err != nil {
		t.Fatalf("loadBody stdin returned error: %v", err)
	}
	if string(stdinBody) != `{"stdin":true}` {
		t.Fatalf("loadBody stdin = %s", string(stdinBody))
	}
}

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
			return jsonResponse(`{"data":[{"id":"sub_1"},{"id":"sub_2"}],"pagination":{"has_more":true,"next_cursor":"next"}}`, nil), nil
		}
		if got := r.URL.Query().Get("cursor"); got != "next" {
			t.Fatalf("cursor query = %q, want next", got)
		}
		return jsonResponse(`{"data":[{"id":"sub_3"}],"pagination":{"has_more":false,"next_cursor":null}}`, nil), nil
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := NewApp(strings.NewReader(""), &stdout, &stderr, map[string]string{}, httpClient)
	exitCode := app.Run(context.Background(), []string{
		"subscriptions", "list",
		"--api-key", "test-token",
		"--publication-id", "pub_test",
		"--base-url", "https://example.test",
		"--all",
		"--query", "expand=stats,custom_fields",
		"--query", "status=active",
	})
	if exitCode != 0 {
		t.Fatalf("Run exit code = %d, stderr = %s", exitCode, stderr.String())
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
		return jsonResponse(`{"data":{"id":"field_1","kind":"string","display":"Codex Test"}}`, nil), nil
	})

	tempDir := t.TempDir()
	bodyPath := filepath.Join(tempDir, "body.json")
	if err := os.WriteFile(bodyPath, []byte(`{"kind":"string","display":"Codex Test"}`), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := NewApp(strings.NewReader(""), &stdout, &stderr, map[string]string{}, httpClient)
	exitCode := app.Run(context.Background(), []string{
		"custom-fields", "create",
		"--api-key", "test-token",
		"--publication-id", "pub_test",
		"--base-url", "https://example.test",
		"--body", "@" + bodyPath,
	})
	if exitCode != 0 {
		t.Fatalf("Run exit code = %d, stderr = %s", exitCode, stderr.String())
	}
}

func TestAuthStatusDoesNotPrintCredentials(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	app := NewApp(strings.NewReader(""), &stdout, &stderr, map[string]string{
		"BEEHIIV_API_KEY":        "test-token",
		"BEEHIIV_PUBLICATION_ID": "pub_test",
	}, nil)
	exitCode := app.Run(context.Background(), []string{"auth", "status", "--config", configPath})
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

func TestOutputTableRendersTabularView(t *testing.T) {
	t.Parallel()

	httpClient := stubHTTPClient(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(`{"data":[{"id":"sub_1","email":"one@example.com"},{"id":"sub_2","email":"two@example.com"}]}`, nil), nil
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := NewApp(strings.NewReader(""), &stdout, &stderr, map[string]string{}, httpClient)
	exitCode := app.Run(context.Background(), []string{
		"subscriptions", "list",
		"--api-key", "test-token",
		"--publication-id", "pub_test",
		"--base-url", "https://example.test",
		"--output", "table",
	})
	if exitCode != 0 {
		t.Fatalf("Run exit code = %d, stderr = %s", exitCode, stderr.String())
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
		return jsonResponse(`{"data":[{"id":"sub_1"}]}`, headers), nil
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := NewApp(strings.NewReader(""), &stdout, &stderr, map[string]string{}, httpClient)
	exitCode := app.Run(context.Background(), []string{
		"subscriptions", "list",
		"--api-key", "test-token",
		"--publication-id", "pub_test",
		"--base-url", "https://example.test",
		"--verbose",
		"--raw",
	})
	if exitCode != 0 {
		t.Fatalf("Run exit code = %d, stderr = %s", exitCode, stderr.String())
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
