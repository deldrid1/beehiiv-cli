package cmd

import (
	"github.com/spf13/cobra"

	"github.com/deldrid1/beehiiv-cli/internal/commandset"
)

func registerWorkflowHelpers(groupCommand *cobra.Command, group string, options Options) {
	switch group {
	case "automations":
		registerAutomationHelpers(groupCommand, options)
	case "polls":
		registerPollHelpers(groupCommand, options)
	case "segments":
		registerSegmentHelpers(groupCommand, options)
	case "workspaces":
		registerWorkspaceHelpers(groupCommand, options)
	}
}

func registerAutomationHelpers(groupCommand *cobra.Command, options Options) {
	mustAddOperationAlias(groupCommand, options, "automation-emails", "list", operationAliasSpec{
		Use:   "emails <automationId>",
		Short: "List the emails inside an automation",
		Long: "List the emails belonging to a specific automation, including the engagement " +
			"statistics Beehiiv returns for each automation email.\n\nAPI path: /publications/{publicationId}/automations/{automationId}/emails",
		Example: "beehiiv automations emails aut_123\n" +
			"beehiiv automations emails aut_123 --all",
	})

	mustAddOperationAlias(groupCommand, options, "automation-journeys", "list", operationAliasSpec{
		Use:     "journeys <automationId>",
		Aliases: []string{"runs"},
		Short:   "List journeys that occurred inside an automation",
		Long: "List the journeys that have occurred inside a specific automation.\n\n" +
			"API path: /publications/{publicationId}/automations/{automationId}/journeys",
		Example: "beehiiv automations journeys aut_123\n" +
			"beehiiv automations journeys aut_123 --all",
	})

	mustAddOperationAlias(groupCommand, options, "automation-journeys", "get", operationAliasSpec{
		Use:   "journey <automationId> <automationJourneyId>",
		Short: "Show a single automation journey",
		Long: "Show a single automation journey by automation ID and journey ID.\n\n" +
			"API path: /publications/{publicationId}/automations/{automationId}/journeys/{automationJourneyId}",
		Example: "beehiiv automations journey aut_123 journey_123",
	})

	mustAddOperationAlias(groupCommand, options, "automation-journeys", "create", operationAliasSpec{
		Use:   "enroll <automationId>",
		Short: "Enroll an existing subscriber into an automation",
		Long: "Enroll an existing subscriber into an automation when that automation has an " +
			"active Add by API trigger.\n\nAPI path: /publications/{publicationId}/automations/{automationId}/journeys",
		Example: "beehiiv automations enroll aut_123 --body '{\"email\":\"person@example.com\"}'\n" +
			"beehiiv automations enroll aut_123 --body '{\"subscription_id\":\"sub_123\"}'",
	})
}

func registerPollHelpers(groupCommand *cobra.Command, options Options) {
	mustAddOperationAlias(groupCommand, options, "poll-responses", "list", operationAliasSpec{
		Use:   "responses <pollId>",
		Short: "List the responses for a poll",
		Long: "List individual subscriber responses for a specific poll.\n\n" +
			"API path: /publications/{publicationId}/polls/{pollId}/responses",
		Example: "beehiiv polls responses poll_123\n" +
			"beehiiv polls responses poll_123 --query expand=post",
	})
}

func registerSegmentHelpers(groupCommand *cobra.Command, options Options) {
	mustAddOperationAlias(groupCommand, options, "segment-members", "list", operationAliasSpec{
		Use:   "members <segmentId>",
		Short: "List the full subscriber records for a segment",
		Long: "List full subscriber records for a specific segment, including optional expansion data.\n\n" +
			"API path: /publications/{publicationId}/segments/{segmentId}/members",
		Example: "beehiiv segments members segment_123\n" +
			"beehiiv segments members segment_123 --query expand=stats,custom_fields",
	})

	mustAddOperationAlias(groupCommand, options, "segment-results", "list", operationAliasSpec{
		Use:   "results <segmentId>",
		Short: "List the lightweight result set for a segment",
		Long: "List the lightweight result set for a specific segment when you only need IDs or a smaller response.\n\n" +
			"API path: /publications/{publicationId}/segments/{segmentId}/results",
		Example: "beehiiv segments results segment_123",
	})
}

func registerWorkspaceHelpers(groupCommand *cobra.Command, options Options) {
	mustAddOperationAlias(groupCommand, options, "workspaces", "publications-by-subscription-email", operationAliasSpec{
		Use:   "publications <email>",
		Short: "Find publications across the workspace by subscriber email",
		Long: "Find all publications in the current workspace that have a subscription for the given email address.\n\n" +
			"API path: /workspaces/publications/by_subscription_email/{email}",
		Example: "beehiiv workspaces publications person@example.com\n" +
			"beehiiv workspaces publications person@example.com --query expand=publication,subscription",
	})
}

type operationAliasSpec struct {
	Use     string
	Aliases []string
	Short   string
	Long    string
	Example string
}

func mustAddOperationAlias(parent *cobra.Command, options Options, targetGroup, targetAction string, spec operationAliasSpec) {
	operation, ok, err := commandset.Find(targetGroup, targetAction)
	if err != nil || !ok {
		return
	}

	command := &cobra.Command{
		Use:     spec.Use,
		Aliases: append([]string(nil), spec.Aliases...),
		Short:   spec.Short,
		Long:    spec.Long,
		Example: spec.Example,
		Args:    exactPathArgs(operation.PathParams),
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeOperation(cmd.Context(), cmd, args, operation, options)
		},
	}
	registerOperationFlags(command, operation)
	parent.AddCommand(command)
}
