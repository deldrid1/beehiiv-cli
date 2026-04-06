package cmd

import "github.com/spf13/cobra"

func newCompletionCommand() *cobra.Command {
	completion := &cobra.Command{
		Use:     "completion",
		Short:   "Generate the autocompletion script for the specified shell",
		Long:    "Generate the autocompletion script for beehiiv for the specified shell.\nSee each sub-command's help for details on how to use the generated script.",
		GroupID: commandGroupCore,
	}

	completion.AddCommand(
		&cobra.Command{
			Use:   "bash",
			Short: "Generate the autocompletion script for bash",
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				return cmd.Root().GenBashCompletionV2(cmd.OutOrStdout(), true)
			},
		},
		&cobra.Command{
			Use:   "fish",
			Short: "Generate the autocompletion script for fish",
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				return cmd.Root().GenFishCompletion(cmd.OutOrStdout(), true)
			},
		},
		&cobra.Command{
			Use:   "powershell",
			Short: "Generate the autocompletion script for powershell",
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				return cmd.Root().GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
			},
		},
		&cobra.Command{
			Use:   "zsh",
			Short: "Generate the autocompletion script for zsh",
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				return cmd.Root().GenZshCompletion(cmd.OutOrStdout())
			},
		},
	)

	return completion
}
