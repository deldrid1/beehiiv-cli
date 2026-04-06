package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/deldrid1/beehiiv-cli/internal/commandset"
	cliruntime "github.com/deldrid1/beehiiv-cli/internal/runtime"
)

func registerWorkflowHelpers(groupCommand *cobra.Command, group string, executor *cliruntime.Executor) {
	switch group {
	case "automations":
		registerAutomationHelpers(groupCommand, executor)
	}
}

func registerAutomationHelpers(groupCommand *cobra.Command, executor *cliruntime.Executor) {
	mustAddOperationAlias(groupCommand, executor, "automation-emails", "list", operationAliasSpec{
		Use:   "emails <automationId>",
		Short: "List the emails inside an automation",
		Long: "List the emails belonging to a specific automation, including the engagement " +
			"statistics Beehiiv returns for each automation email.\n\nAPI path: /publications/{publicationId}/automations/{automationId}/emails",
		Example: "beehiiv automations emails aut_123\n" +
			"beehiiv automations emails aut_123 --all",
	})

	mustAddOperationAlias(groupCommand, executor, "automation-journeys", "list", operationAliasSpec{
		Use:     "journeys <automationId>",
		Aliases: []string{"runs"},
		Short:   "List journeys that occurred inside an automation",
		Long: "List the journeys that have occurred inside a specific automation.\n\n" +
			"API path: /publications/{publicationId}/automations/{automationId}/journeys",
		Example: "beehiiv automations journeys aut_123\n" +
			"beehiiv automations journeys aut_123 --all",
	})

	mustAddOperationAlias(groupCommand, executor, "automation-journeys", "get", operationAliasSpec{
		Use:   "journey <automationId> <automationJourneyId>",
		Short: "Show a single automation journey",
		Long: "Show a single automation journey by automation ID and journey ID.\n\n" +
			"API path: /publications/{publicationId}/automations/{automationId}/journeys/{automationJourneyId}",
		Example: "beehiiv automations journey aut_123 journey_123",
	})

	mustAddOperationAlias(groupCommand, executor, "automation-journeys", "create", operationAliasSpec{
		Use:   "enroll <automationId>",
		Short: "Enroll an existing subscriber into an automation",
		Long: "Enroll an existing subscriber into an automation when that automation has an " +
			"active Add by API trigger.\n\nAPI path: /publications/{publicationId}/automations/{automationId}/journeys",
		Example: "beehiiv automations enroll aut_123 --body '{\"email\":\"person@example.com\"}'\n" +
			"beehiiv automations enroll aut_123 --body '{\"subscription_id\":\"sub_123\"}'",
	})
}

type operationAliasSpec struct {
	Use     string
	Aliases []string
	Short   string
	Long    string
	Example string
}

func mustAddOperationAlias(parent *cobra.Command, executor *cliruntime.Executor, targetGroup, targetAction string, spec operationAliasSpec) {
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
			return runOperationAlias(cmd.Context(), executor, targetGroup, targetAction, operation, cmd, args)
		},
	}
	registerOperationFlags(command, operation)
	parent.AddCommand(command)
}

func runOperationAlias(ctx context.Context, executor *cliruntime.Executor, targetGroup, targetAction string, operation commandset.Operation, command *cobra.Command, pathArgs []string) error {
	legacyArgs, err := appendGlobalFlags(nil, command)
	if err != nil {
		return err
	}
	legacyArgs = append(legacyArgs, targetGroup, targetAction)
	legacyArgs = append(legacyArgs, pathArgs...)
	legacyArgs, err = appendOperationFlags(legacyArgs, command, operation)
	if err != nil {
		return err
	}

	exitCode := executor.Run(ctx, legacyArgs)
	if exitCode != 0 {
		return exitError{code: exitCode}
	}
	return nil
}
