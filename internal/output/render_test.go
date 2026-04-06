package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/deldrid1/beehiiv-cli/internal/config"
)

func TestWriteTableOutputRendersRows(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	err := Write(&out, map[string]any{
		"data": []any{
			map[string]any{"id": "sub_1", "email": "one@example.com"},
			map[string]any{"id": "sub_2", "email": "two@example.com"},
		},
	}, nil, config.Runtime{Output: config.OutputTable})
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}
	if !strings.Contains(out.String(), "| id") {
		t.Fatalf("table output missing header: %s", out.String())
	}
	if !strings.Contains(out.String(), "one@example.com") {
		t.Fatalf("table output missing row: %s", out.String())
	}
}

func TestWriteRawOutputUsesResponseBody(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	err := Write(&out, map[string]any{"ignored": true}, []byte(`{"data":[{"id":"sub_1"}]}`), config.Runtime{Output: config.OutputRaw})
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}
	if got := strings.TrimSpace(out.String()); got != `{"data":[{"id":"sub_1"}]}` {
		t.Fatalf("raw output = %q", got)
	}
}

func TestWriteJSONOutputHonorsCompactFlag(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	err := Write(&out, map[string]any{"status": 200}, nil, config.Runtime{Output: config.OutputJSON, Compact: true})
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}
	if got := strings.TrimSpace(out.String()); got != `{"status":200}` {
		t.Fatalf("compact output = %q", got)
	}
}
