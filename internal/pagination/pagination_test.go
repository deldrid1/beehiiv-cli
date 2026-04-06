package pagination

import (
	"context"
	"encoding/json"
	"net/url"
	"testing"
)

func TestExtractPageWithNestedPagination(t *testing.T) {
	t.Parallel()

	page, err := ExtractPage([]byte(`{"data":[{"id":"1"}],"pagination":{"has_more":true,"next_cursor":"next","page":1,"total_pages":3,"total_results":5}}`))
	if err != nil {
		t.Fatalf("ExtractPage returned error: %v", err)
	}

	if len(page.Items) != 1 {
		t.Fatalf("ExtractPage items = %d, want 1", len(page.Items))
	}
	if !page.HasMore {
		t.Fatal("ExtractPage HasMore = false, want true")
	}
	if page.NextCursor == nil || *page.NextCursor != "next" {
		t.Fatalf("ExtractPage NextCursor = %v, want next", page.NextCursor)
	}
}

func TestCollectAllCursor(t *testing.T) {
	t.Parallel()

	requests := 0
	items, summary, err := CollectAll(context.Background(), "hybrid", url.Values{}, func(_ context.Context, query url.Values) ([]byte, error) {
		requests++
		switch requests {
		case 1:
			if got := query.Get("cursor"); got != "" {
				t.Fatalf("first cursor query = %q, want empty", got)
			}
			return []byte(`{"data":[{"id":"1"},{"id":"2"}],"pagination":{"has_more":true,"next_cursor":"next"}}`), nil
		case 2:
			if got := query.Get("cursor"); got != "next" {
				t.Fatalf("second cursor query = %q, want next", got)
			}
			return []byte(`{"data":[{"id":"3"}],"pagination":{"has_more":false,"next_cursor":null}}`), nil
		default:
			t.Fatalf("unexpected request %d", requests)
			return nil, nil
		}
	})
	if err != nil {
		t.Fatalf("CollectAll returned error: %v", err)
	}

	if len(items) != 3 {
		t.Fatalf("CollectAll items = %d, want 3", len(items))
	}
	if summary.Mode != "cursor" {
		t.Fatalf("CollectAll mode = %q, want cursor", summary.Mode)
	}
	if summary.PagesFetched != 2 {
		t.Fatalf("CollectAll pages = %d, want 2", summary.PagesFetched)
	}
}

func TestCollectAllOffset(t *testing.T) {
	t.Parallel()

	requests := 0
	items, summary, err := CollectAll(context.Background(), "offset", url.Values{"page": {"1"}}, func(_ context.Context, query url.Values) ([]byte, error) {
		requests++
		switch requests {
		case 1:
			if got := query.Get("page"); got != "1" {
				t.Fatalf("first page query = %q, want 1", got)
			}
			return []byte(`{"data":[{"id":"1"}],"page":1,"total_pages":2,"total_results":2}`), nil
		case 2:
			if got := query.Get("page"); got != "2" {
				t.Fatalf("second page query = %q, want 2", got)
			}
			return []byte(`{"data":[{"id":"2"}],"page":2,"total_pages":2,"total_results":2}`), nil
		default:
			t.Fatalf("unexpected request %d", requests)
			return nil, nil
		}
	})
	if err != nil {
		t.Fatalf("CollectAll returned error: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("CollectAll items = %d, want 2", len(items))
	}
	if summary.Mode != "offset" {
		t.Fatalf("CollectAll mode = %q, want offset", summary.Mode)
	}
}

func TestExtractPageTopLevelArray(t *testing.T) {
	t.Parallel()

	page, err := ExtractPage([]byte(`[{"id":"1"},{"id":"2"}]`))
	if err != nil {
		t.Fatalf("ExtractPage returned error: %v", err)
	}

	if len(page.Items) != 2 {
		t.Fatalf("ExtractPage items = %d, want 2", len(page.Items))
	}

	var decoded map[string]string
	if err := json.Unmarshal(page.Items[0], &decoded); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}
	if decoded["id"] != "1" {
		t.Fatalf("decoded first item id = %q, want 1", decoded["id"])
	}
}
