package cmd

import (
	"bytes"
	"context"
	"encoding/csv"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type stubHTTPClient func(*http.Request) (*http.Response, error)

func (f stubHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return f(req)
}

func jsonResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestReportsSummaryRendersFriendlySectionsByDefault(t *testing.T) {
	t.Parallel()

	httpClient := stubHTTPClient(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/publications/pub_test":
			if got := r.URL.Query()["expand"]; len(got) != 1 || got[0] != "stats" {
				t.Fatalf("publication expand query = %#v", got)
			}
			return jsonResponse(`{"data":{"id":"pub_test","name":"Morning Dispatch","organization_name":"Every Daily","referral_program_enabled":true,"stats":{"active_subscriptions":1200,"average_open_rate":0.62,"average_click_rate":0.14}}}`), nil
		case "/publications/pub_test/posts/aggregate_stats":
			return jsonResponse(`{"data":{"stats":{"email":{"recipients":5000,"delivered":4900,"unique_opens":2800,"open_rate":57.1,"unique_clicks":700,"click_rate":14.2},"web":{"views":3200,"clicks":410}}}}`), nil
		case "/publications/pub_test/posts":
			if got := r.URL.Query()["expand"]; len(got) != 1 || got[0] != "stats" {
				t.Fatalf("posts expand query = %#v", got)
			}
			return jsonResponse(`{"data":[{"id":"post_1","title":"Launch Week","status":"confirmed","publish_date":"2026-04-01","stats":{"email":{"open_rate":61.2,"click_rate":16.8,"unique_opens":1200},"web":{"views":900,"clicks":112}}},{"id":"post_2","title":"April Roadmap","status":"confirmed","publish_date":"2026-03-28","stats":{"email":{"open_rate":55.0,"click_rate":10.4,"unique_opens":980},"web":{"views":750,"clicks":98}}}]}`), nil
		case "/publications/pub_test/engagements":
			if got := r.URL.Query().Get("number_of_days"); got != "7" {
				t.Fatalf("number_of_days query = %q, want 7", got)
			}
			return jsonResponse(`{"data":[{"date":"2026-03-31","total_opens":400,"unique_opens":320,"total_clicks":88,"unique_clicks":70,"total_verified_clicks":84,"unique_verified_clicks":66},{"date":"2026-04-01","total_opens":430,"unique_opens":340,"total_clicks":92,"unique_clicks":73,"total_verified_clicks":90,"unique_verified_clicks":70}]}`), nil
		default:
			t.Fatalf("unexpected request path %q", r.URL.Path)
			return nil, nil
		}
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := ExecuteContext(context.Background(), []string{
		"reports", "summary",
		"--api-key", "test-token",
		"--publication-id", "pub_test",
		"--base-url", "https://example.test",
	}, Options{
		Stdout:     &stdout,
		Stderr:     &stderr,
		HTTPClient: httpClient,
	})
	if exitCode != 0 {
		t.Fatalf("ExecuteContext exit code = %d, stderr = %s", exitCode, stderr.String())
	}

	output := stdout.String()
	for _, snippet := range []string{
		"PUBLICATION",
		"POST_ROLLUP",
		"ENGAGEMENT_SUMMARY",
		"RECENT_POSTS",
		"Morning Dispatch",
		"Launch Week",
		"active_subscriptions",
		"avg_daily_unique_opens",
	} {
		if !strings.Contains(output, snippet) {
			t.Fatalf("summary output missing %q: %s", snippet, output)
		}
	}
}

func TestReportsChartPrintsASCIIChartByDefault(t *testing.T) {
	t.Parallel()

	httpClient := stubHTTPClient(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/publications/pub_test/engagements" {
			t.Fatalf("unexpected request path %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("email_type"); got != "all" {
			t.Fatalf("email_type query = %q, want all", got)
		}
		return jsonResponse(`{"data":[{"date":"2026-04-01","unique_opens":120},{"date":"2026-04-02","unique_opens":180},{"date":"2026-04-03","unique_opens":90}]}`), nil
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := ExecuteContext(context.Background(), []string{
		"reports", "chart",
		"--metric", "unique_opens",
		"--api-key", "test-token",
		"--publication-id", "pub_test",
		"--base-url", "https://example.test",
	}, Options{
		Stdout:     &stdout,
		Stderr:     &stderr,
		HTTPClient: httpClient,
	})
	if exitCode != 0 {
		t.Fatalf("ExecuteContext exit code = %d, stderr = %s", exitCode, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "Beehiiv engagement chart (unique_opens)") {
		t.Fatalf("chart output missing title: %s", output)
	}
	if !strings.Contains(output, "2026-04-02") || !strings.Contains(output, "#") {
		t.Fatalf("chart output missing rendered bars: %s", output)
	}
}

func TestReportsExportSubscriptionsWritesCSVFile(t *testing.T) {
	t.Parallel()

	requests := 0
	httpClient := stubHTTPClient(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/publications/pub_test/subscriptions" {
			t.Fatalf("unexpected request path %q", r.URL.Path)
		}
		if got := r.URL.Query()["expand[]"]; len(got) != 2 {
			t.Fatalf("expand[] query = %#v", got)
		}
		requests++
		switch requests {
		case 1:
			if got := r.URL.Query().Get("cursor"); got != "" {
				t.Fatalf("cursor query on first request = %q, want empty", got)
			}
			return jsonResponse(`{"data":[{"id":"sub_1","email":"one@example.com","status":"active","custom_fields":[{"name":"Favorite Color","value":"blue"}],"stats":{"open_rate":61.2}},{"id":"sub_2","email":"two@example.com","status":"inactive","custom_fields":[{"name":"Favorite Color","value":"green"}],"stats":{"open_rate":45.0}}],"has_more":true,"next_cursor":"next"}`), nil
		case 2:
			if got := r.URL.Query().Get("cursor"); got != "next" {
				t.Fatalf("cursor query on second request = %q, want next", got)
			}
			return jsonResponse(`{"data":[],"has_more":false,"next_cursor":null}`), nil
		default:
			t.Fatalf("unexpected request count %d", requests)
			return nil, nil
		}
	})

	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "subscriptions.csv")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := ExecuteContext(context.Background(), []string{
		"reports", "export", "subscriptions",
		"--file", outputPath,
		"--api-key", "test-token",
		"--publication-id", "pub_test",
		"--base-url", "https://example.test",
	}, Options{
		Stdout:     &stdout,
		Stderr:     &stderr,
		HTTPClient: httpClient,
	})
	if exitCode != 0 {
		t.Fatalf("ExecuteContext exit code = %d, stderr = %s", exitCode, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout should be empty when --file is used: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Wrote 2 rows") {
		t.Fatalf("stderr missing export status message: %s", stderr.String())
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	reader := csv.NewReader(strings.NewReader(string(data)))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll returned error: %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("csv row count = %d, want 3", len(records))
	}
	headers := strings.Join(records[0], ",")
	for _, expected := range []string{"email", "id", "custom_fields.favorite_color", "stats.open_rate"} {
		if !strings.Contains(headers, expected) {
			t.Fatalf("csv headers missing %q: %s", expected, headers)
		}
	}
	body := strings.Join(records[1], ",") + "\n" + strings.Join(records[2], ",")
	for _, expected := range []string{"one@example.com", "two@example.com", "blue", "green"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("csv body missing %q: %s", expected, body)
		}
	}
}
