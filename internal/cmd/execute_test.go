package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/deldrid1/beehiiv-cli/internal/commandset"
)

func TestCollectPathValuesMapPositionalArgs(t *testing.T) {
	t.Parallel()

	operation := commandset.Operation{PathParams: []string{"automationId", "journeyId"}}
	values, err := collectPathValues(operation, []string{"aut_123", "journey_456"})
	if err != nil {
		t.Fatalf("collectPathValues returned error: %v", err)
	}
	if values["automationId"] != "aut_123" || values["journeyId"] != "journey_456" {
		t.Fatalf("values = %#v", values)
	}
}

func TestCollectPathValuesRejectsWrongArgCount(t *testing.T) {
	t.Parallel()

	operation := commandset.Operation{PathParams: []string{"subscriptionId"}}
	_, err := collectPathValues(operation, nil)
	if err == nil {
		t.Fatal("expected error for missing positional")
	}
}

func TestBuildQueryValuesHandlesCommaExpansion(t *testing.T) {
	t.Parallel()

	operation := commandset.Operation{
		QueryParams: []commandset.Parameter{
			{Name: "expand[]", Multiple: true},
			{Name: "status"},
		},
	}
	values, err := buildQueryValues(operation, []string{"expand=stats,custom_fields", "status=active"})
	if err != nil {
		t.Fatalf("buildQueryValues returned error: %v", err)
	}
	expand := values["expand[]"]
	if len(expand) != 2 || expand[0] != "stats" || expand[1] != "custom_fields" {
		t.Fatalf("expand[] = %#v, want [stats custom_fields]", expand)
	}
	if values.Get("status") != "active" {
		t.Fatalf("status = %q", values.Get("status"))
	}
}

func TestBuildQueryValuesRejectsInvalidFormat(t *testing.T) {
	t.Parallel()

	_, err := buildQueryValues(commandset.Operation{}, []string{"no-equals-sign"})
	if err == nil {
		t.Fatal("expected error for missing '='")
	}
}

func TestNormalizeQueryNameHandlesArrayBrackets(t *testing.T) {
	t.Parallel()

	operation := commandset.Operation{
		QueryParams: []commandset.Parameter{
			{Name: "expand[]"},
		},
	}
	if got := normalizeQueryName(operation, "expand"); got != "expand[]" {
		t.Fatalf("normalizeQueryName(expand) = %q, want expand[]", got)
	}
	if got := normalizeQueryName(operation, "expand[]"); got != "expand[]" {
		t.Fatalf("normalizeQueryName(expand[]) = %q, want expand[]", got)
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

func TestLoadBodyRejectsInvalidJSON(t *testing.T) {
	t.Parallel()

	operation := commandset.Operation{Body: true, Command: []string{"test", "create"}}
	_, err := loadBody(operation, []string{`not-json`}, strings.NewReader(""))
	if err == nil {
		t.Fatal("expected error for invalid JSON body")
	}
}

func TestLoadBodyRejectsBodyOnNonBodyOperation(t *testing.T) {
	t.Parallel()

	operation := commandset.Operation{Body: false, Command: []string{"test", "list"}}
	_, err := loadBody(operation, []string{`{"x":1}`}, strings.NewReader(""))
	if err == nil {
		t.Fatal("expected error for body on non-body operation")
	}
}
