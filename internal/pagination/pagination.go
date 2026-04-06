package pagination

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

type Summary struct {
	Mode          string  `json:"mode"`
	PagesFetched  int     `json:"pages_fetched"`
	ItemsReturned int     `json:"items_returned"`
	HasMore       bool    `json:"has_more"`
	NextCursor    *string `json:"next_cursor"`
}

type Page struct {
	Items        []json.RawMessage
	HasMore      bool
	NextCursor   *string
	Page         int
	TotalPages   int
	TotalResults int
}

func ExtractPage(body []byte) (Page, error) {
	var arrayBody []json.RawMessage
	if err := json.Unmarshal(body, &arrayBody); err == nil {
		return Page{
			Items: arrayBody,
		}, nil
	}

	var objectBody map[string]json.RawMessage
	if err := json.Unmarshal(body, &objectBody); err != nil {
		return Page{}, fmt.Errorf("response body is not valid JSON: %w", err)
	}

	page := Page{}

	if rawData, ok := objectBody["data"]; ok {
		if err := json.Unmarshal(rawData, &page.Items); err != nil {
			var single json.RawMessage
			if err := json.Unmarshal(rawData, &single); err == nil && len(single) > 0 {
				page.Items = []json.RawMessage{single}
			}
		}
	}

	if rawPagination, ok := objectBody["pagination"]; ok {
		var nested map[string]json.RawMessage
		if err := json.Unmarshal(rawPagination, &nested); err == nil {
			page.HasMore = parseBool(nested["has_more"])
			page.NextCursor = parseStringPointer(nested["next_cursor"])
			page.Page = parseInt(nested["page"])
			page.TotalPages = parseInt(nested["total_pages"])
			page.TotalResults = parseInt(nested["total_results"])
		}
	}

	if page.TotalPages == 0 {
		page.TotalPages = parseInt(objectBody["total_pages"])
	}
	if page.Page == 0 {
		page.Page = parseInt(objectBody["page"])
	}
	if page.TotalResults == 0 {
		page.TotalResults = parseInt(objectBody["total_results"])
	}
	if page.NextCursor == nil {
		page.NextCursor = parseStringPointer(objectBody["next_cursor"])
	}
	if !page.HasMore {
		page.HasMore = parseBool(objectBody["has_more"])
	}

	return page, nil
}

func CollectAll(
	ctx context.Context,
	mode string,
	baseQuery url.Values,
	fetch func(context.Context, url.Values) ([]byte, error),
) ([]json.RawMessage, Summary, error) {
	selectedMode := normalizeMode(mode, baseQuery)
	current := cloneQuery(baseQuery)
	if selectedMode == "cursor" {
		current.Del("page")
	}

	allItems := make([]json.RawMessage, 0)
	summary := Summary{Mode: selectedMode}

	for {
		body, err := fetch(ctx, current)
		if err != nil {
			return nil, summary, err
		}
		page, err := ExtractPage(body)
		if err != nil {
			return nil, summary, err
		}
		summary.PagesFetched++
		allItems = append(allItems, page.Items...)

		if selectedMode == "cursor" {
			if !page.HasMore || page.NextCursor == nil || *page.NextCursor == "" {
				summary.HasMore = page.HasMore
				summary.NextCursor = page.NextCursor
				break
			}
			current.Set("cursor", *page.NextCursor)
			continue
		}

		if selectedMode == "offset" {
			currentPage := page.Page
			if currentPage == 0 {
				currentPage = queryInt(current, "page", 1)
			}
			if page.TotalPages == 0 || currentPage >= page.TotalPages {
				break
			}
			current.Set("page", strconv.Itoa(currentPage+1))
			continue
		}

		break
	}

	summary.ItemsReturned = len(allItems)
	return allItems, summary, nil
}

func normalizeMode(mode string, query url.Values) string {
	mode = strings.ToLower(mode)
	if mode == "hybrid" {
		if query.Get("page") != "" {
			return "offset"
		}
		if query.Get("cursor") != "" {
			return "cursor"
		}
		return "cursor"
	}
	if mode == "" || mode == "none" {
		return "none"
	}
	return mode
}

func cloneQuery(query url.Values) url.Values {
	cloned := make(url.Values, len(query))
	for key, values := range query {
		copied := make([]string, len(values))
		copy(copied, values)
		cloned[key] = copied
	}
	return cloned
}

func queryInt(query url.Values, key string, fallback int) int {
	value := query.Get(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func parseBool(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return false
	}
	var value bool
	if err := json.Unmarshal(raw, &value); err == nil {
		return value
	}
	return false
}

func parseInt(raw json.RawMessage) int {
	if len(raw) == 0 {
		return 0
	}
	var value int
	if err := json.Unmarshal(raw, &value); err == nil {
		return value
	}
	return 0
}

func parseStringPointer(raw json.RawMessage) *string {
	if len(raw) == 0 {
		return nil
	}
	if string(raw) == "null" {
		return nil
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil
	}
	return &value
}
