package workflows_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/deldrid1/beehiiv-cli/internal/cmd"
	"github.com/deldrid1/beehiiv-cli/internal/cmd/workflows"
	"github.com/deldrid1/beehiiv-cli/internal/commandset"
)

func TestLookupReturnsCuratedSpecsForAllCommandGroups(t *testing.T) {
	t.Parallel()

	groups, err := commandset.Groups()
	if err != nil {
		t.Fatalf("commandset.Groups() error = %v", err)
	}

	for _, group := range groups {
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

func TestAutomationWorkflowExamplesAppearInHelp(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := cmd.ExecuteContext(context.Background(), []string{"automations", "--help"}, cmd.Options{
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if exitCode != 0 {
		t.Fatalf("ExecuteContext exit code = %d, stderr = %s", exitCode, stderr.String())
	}
	if !strings.Contains(stdout.String(), "jump into the emails and journeys") {
		t.Fatalf("stdout missing automation workflow overview: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "beehiiv automations show aut_123 --query expand=stats") {
		t.Fatalf("stdout missing automation expand example: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "beehiiv automations emails aut_123") {
		t.Fatalf("stdout missing automation helper example: %s", stdout.String())
	}
}

func TestAdditionalCoreWorkflowExamplesAppearInHelp(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		args    []string
		snippet string
	}{
		{
			args:    []string{"segments", "--help"},
			snippet: "beehiiv segments members segment_123",
		},
		{
			args:    []string{"polls", "--help"},
			snippet: "beehiiv polls responses poll_123",
		},
		{
			args:    []string{"workspaces", "--help"},
			snippet: "beehiiv workspaces publications person@example.com",
		},
		{
			args:    []string{"custom-fields", "--help"},
			snippet: "beehiiv fields add --body @custom-field.json",
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(strings.Join(testCase.args, "_"), func(t *testing.T) {
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
				t.Fatalf("stdout missing snippet %q: %s", testCase.snippet, stdout.String())
			}
		})
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
		{
			name:    "automations_show",
			args:    []string{"automations", "show", "--help"},
			snippet: "API path: /publications/{publicationId}/automations/{automationId}",
		},
		{
			name:    "automation_journeys_show",
			args:    []string{"automation-journeys", "show", "--help"},
			snippet: "API path: /publications/{publicationId}/automations/{automationId}/journeys/{automationJourneyId}",
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

func TestAutomationHelperCommandsAppearUnderAutomations(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := cmd.ExecuteContext(context.Background(), []string{"automations", "--help"}, cmd.Options{
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if exitCode != 0 {
		t.Fatalf("ExecuteContext exit code = %d, stderr = %s", exitCode, stderr.String())
	}

	for _, expected := range []string{"emails", "journeys", "journey", "enroll"} {
		if !strings.Contains(stdout.String(), expected) {
			t.Fatalf("stdout missing helper command %q: %s", expected, stdout.String())
		}
	}
}

func TestAutomationHelperCommandsResolveToExpectedOperations(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		args    []string
		snippet string
	}{
		{
			name:    "automations_emails",
			args:    []string{"automations", "emails", "--help"},
			snippet: "API path: /publications/{publicationId}/automations/{automationId}/emails",
		},
		{
			name:    "automations_journeys",
			args:    []string{"automations", "journeys", "--help"},
			snippet: "API path: /publications/{publicationId}/automations/{automationId}/journeys",
		},
		{
			name:    "automations_journey",
			args:    []string{"automations", "journey", "--help"},
			snippet: "API path: /publications/{publicationId}/automations/{automationId}/journeys/{automationJourneyId}",
		},
		{
			name:    "automations_enroll",
			args:    []string{"automations", "enroll", "--help"},
			snippet: "API path: /publications/{publicationId}/automations/{automationId}/journeys",
		},
		{
			name:    "segments_members",
			args:    []string{"segments", "members", "--help"},
			snippet: "API path: /publications/{publicationId}/segments/{segmentId}/members",
		},
		{
			name:    "segments_results",
			args:    []string{"segments", "results", "--help"},
			snippet: "API path: /publications/{publicationId}/segments/{segmentId}/results",
		},
		{
			name:    "polls_responses",
			args:    []string{"polls", "responses", "--help"},
			snippet: "API path: /publications/{publicationId}/polls/{pollId}/responses",
		},
		{
			name:    "workspaces_publications",
			args:    []string{"workspaces", "publications", "--help"},
			snippet: "API path: /workspaces/publications/by_subscription_email/{email}",
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

func TestAdditionalActionAliasesResolveToExistingCommands(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		args    []string
		snippet string
	}{
		{
			name:    "subscriptions_jwt",
			args:    []string{"subscriptions", "jwt", "--help"},
			snippet: "API path: /publications/{publicationId}/subscriptions/{subscriptionId}/jwt_token",
		},
		{
			name:    "subscriptions_subscriber",
			args:    []string{"subscriptions", "subscriber", "--help"},
			snippet: "API path: /publications/{publicationId}/subscriptions/by_subscriber_id/{subscriberId}",
		},
		{
			name:    "custom_fields_show",
			args:    []string{"fields", "show", "--help"},
			snippet: "API path: /publications/{publicationId}/custom_fields/{id}",
		},
		{
			name:    "tiers_show",
			args:    []string{"tier", "show", "--help"},
			snippet: "API path: /publications/{publicationId}/tiers/{tierId}",
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
