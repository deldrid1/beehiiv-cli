package workflows_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/deldrid1/beehiiv-cli/internal/cmd"
	"github.com/deldrid1/beehiiv-cli/internal/cmd/workflows"
)

func TestLookupReturnsCuratedSpecsForPrimaryWorkflowGroups(t *testing.T) {
	t.Parallel()

	for _, group := range []string{"publications", "subscriptions", "posts", "webhooks"} {
		group := group
		t.Run(group, func(t *testing.T) {
			spec, ok := workflows.Lookup(group)
			if !ok {
				t.Fatalf("Lookup(%q) = not found", group)
			}
			if spec.Short == "" {
				t.Fatalf("Lookup(%q) returned empty Short", group)
			}
			if spec.Example == "" {
				t.Fatalf("Lookup(%q) returned empty Example", group)
			}
			if len(spec.Actions) == 0 {
				t.Fatalf("Lookup(%q) returned no action specs", group)
			}
		})
	}
}

func TestWorkflowGroupAliasesAndExamplesAppearInHelp(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := cmd.ExecuteContext(context.Background(), []string{"subs", "--help"}, cmd.Options{
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if exitCode != 0 {
		t.Fatalf("ExecuteContext exit code = %d, stderr = %s", exitCode, stderr.String())
	}
	if !strings.Contains(stdout.String(), "highest-traffic workflow groups") {
		t.Fatalf("stdout missing curated group long text: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "beehiiv subscriptions find person@example.com") {
		t.Fatalf("stdout missing curated examples: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Aliases:") || !strings.Contains(stdout.String(), "subs") {
		t.Fatalf("stdout missing aliases section: %s", stdout.String())
	}
}

func TestWorkflowActionAliasesResolveToExistingCommands(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		args    []string
		snippet string
	}{
		{
			name:    "subscriptions_show",
			args:    []string{"subscriptions", "show", "--help"},
			snippet: "API path: /publications/{publicationId}/subscriptions/{subscriptionId}",
		},
		{
			name:    "subscriptions_find",
			args:    []string{"subscriptions", "find", "--help"},
			snippet: "API path: /publications/{publicationId}/subscriptions/by_email/{email}",
		},
		{
			name:    "posts_stats",
			args:    []string{"posts", "stats", "--help"},
			snippet: "API path: /publications/{publicationId}/posts/aggregate_stats",
		},
		{
			name:    "webhooks_ping",
			args:    []string{"hooks", "ping", "--help"},
			snippet: "API path: /publications/{publicationId}/webhooks/{endpointId}/tests",
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			exitCode := cmd.ExecuteContext(context.Background(), testCase.args, cmd.Options{
				Stdout: &stdout,
				Stderr: &stderr,
			})
			if exitCode != 0 {
				t.Fatalf("ExecuteContext exit code = %d, stderr = %s", exitCode, stderr.String())
			}
			if !strings.Contains(stdout.String(), testCase.snippet) {
				t.Fatalf("stdout missing API path snippet %q: %s", testCase.snippet, stdout.String())
			}
		})
	}
}
