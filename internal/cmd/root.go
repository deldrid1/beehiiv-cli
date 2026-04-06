package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/deldrid1/beehiiv-cli/internal/buildinfo"
	"github.com/deldrid1/beehiiv-cli/internal/client"
	"github.com/deldrid1/beehiiv-cli/internal/cmd/workflows"
	"github.com/deldrid1/beehiiv-cli/internal/commandset"
	cliruntime "github.com/deldrid1/beehiiv-cli/internal/runtime"
)

type Options struct {
	Stdin      io.Reader
	Stdout     io.Writer
	Stderr     io.Writer
	Env        map[string]string
	HTTPClient client.HTTPClient
}

type exitError struct {
	code int
}

func (e exitError) Error() string {
	return fmt.Sprintf("command exited with status %d", e.code)
}

func ExecuteContext(ctx context.Context, args []string, options Options) int {
	root := NewRoot(options)
	root.SetArgs(args)
	root.SetContext(ctx)

	if err := root.ExecuteContext(ctx); err != nil {
		var exitErr exitError
		if errors.As(err, &exitErr) {
			return exitErr.code
		}
		fmt.Fprintln(defaultWriter(options.Stderr, os.Stderr), err.Error())
		return 1
	}
	return 0
}

func NewRoot(options Options) *cobra.Command {
	executor := cliruntime.NewExecutor(options.Stdin, options.Stdout, options.Stderr, options.Env, options.HTTPClient)

	root := &cobra.Command{
		Use:              "beehiiv",
		Short:            "Cross-platform Beehiiv API CLI",
		Long:             "Cross-platform Beehiiv API CLI with standard Cobra help, version, and completion surfaces.",
		SilenceUsage:     true,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	root.SetIn(defaultReader(options.Stdin, os.Stdin))
	root.SetOut(defaultWriter(options.Stdout, os.Stdout))
	root.SetErr(defaultWriter(options.Stderr, os.Stderr))
	root.Version = buildinfo.Summary("beehiiv")
	root.SetVersionTemplate("{{.Version}}\n")
	root.AddGroup(
		&cobra.Group{ID: commandGroupCore, Title: "Core Commands"},
		&cobra.Group{ID: commandGroupAuth, Title: "Auth Commands"},
		&cobra.Group{ID: commandGroupAPI, Title: "API Command Groups"},
	)
	registerGlobalFlags(root)

	root.AddCommand(newVersionCommand(defaultWriter(options.Stdout, os.Stdout)))
	root.AddCommand(newCompletionCommand())
	root.AddCommand(newAuthCommand(options))
	root.AddCommand(newLoginCommand(options))

	registerOperationGroups(root, executor)

	return root
}

func newVersionCommand(stdout io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:     "version",
		Short:   "Print version information",
		Args:    cobra.NoArgs,
		GroupID: commandGroupCore,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprintln(defaultWriter(stdout, os.Stdout), buildinfo.Summary("beehiiv"))
			return err
		},
	}
}

func registerOperationGroups(root *cobra.Command, executor *cliruntime.Executor) {
	groups, err := commandset.Groups()
	if err != nil {
		return
	}

	for _, group := range groups {
		group := group
		groupSpec, hasGroupSpec := workflows.Lookup(group)

		groupShort := fmt.Sprintf("%s commands", humanize(group))
		groupLong := ""
		groupAliases := []string(nil)
		groupExample := ""
		if hasGroupSpec {
			groupShort = firstNonEmpty(groupSpec.Short, groupShort)
			groupLong = groupSpec.Long
			groupAliases = append([]string(nil), groupSpec.Aliases...)
			groupExample = groupSpec.Example
		}

		groupCommand := &cobra.Command{
			Use:     group,
			Aliases: groupAliases,
			Short:   groupShort,
			Long:    groupLong,
			Example: groupExample,
			GroupID: commandGroupAPI,
			RunE: func(cmd *cobra.Command, args []string) error {
				return cmd.Help()
			},
		}

		operations, err := commandset.OperationsForGroup(group)
		if err != nil {
			continue
		}
		for _, operation := range operations {
			operation := operation
			actionSpec, hasActionSpec := workflows.ActionFor(group, operation.Command[1])
			actionShort := firstNonEmpty(operation.Summary, cleanDescription(operation.Description), fmt.Sprintf("%s %s", humanize(group), operation.Command[1]))
			actionLong := buildLongDescription(operation)
			actionAliases := []string(nil)
			actionExample := ""
			if hasActionSpec {
				actionShort = firstNonEmpty(actionSpec.Short, actionShort)
				if actionSpec.Long != "" {
					actionLong = actionSpec.Long + "\n\nAPI path: " + operation.Path
				}
				actionAliases = append([]string(nil), actionSpec.Aliases...)
				actionExample = actionSpec.Example
			}

			actionCommand := &cobra.Command{
				Use:     buildUse(operation.Command[1], operation.PathParams),
				Aliases: actionAliases,
				Short:   actionShort,
				Long:    actionLong,
				Example: actionExample,
				Args:    exactPathArgs(operation.PathParams),
				RunE: func(cmd *cobra.Command, args []string) error {
					legacyArgs, err := appendGlobalFlags(nil, cmd)
					if err != nil {
						return err
					}
					legacyArgs = append(legacyArgs, group, operation.Command[1])
					legacyArgs = append(legacyArgs, args...)
					legacyArgs, err = appendOperationFlags(legacyArgs, cmd, operation)
					if err != nil {
						return err
					}
					return runLegacy(cmd.Context(), executor, legacyArgs)
				},
			}
			registerOperationFlags(actionCommand, operation)
			groupCommand.AddCommand(actionCommand)
		}

		registerWorkflowHelpers(groupCommand, group, executor)

		root.AddCommand(groupCommand)
	}
}

func buildUse(action string, pathParams []string) string {
	var builder strings.Builder
	builder.WriteString(action)
	for _, pathParam := range pathParams {
		builder.WriteString(" <")
		builder.WriteString(pathParam)
		builder.WriteString(">")
	}
	return builder.String()
}

func buildLongDescription(operation commandset.Operation) string {
	var parts []string
	if operation.Summary != "" {
		parts = append(parts, operation.Summary)
	}
	if description := cleanDescription(operation.Description); description != "" && description != operation.Summary {
		parts = append(parts, description)
	}
	if operation.Path != "" {
		parts = append(parts, fmt.Sprintf("API path: %s", operation.Path))
	}
	return strings.Join(parts, "\n\n")
}

func cleanDescription(value string) string {
	replacer := strings.NewReplacer(
		"<br>", " ",
		"<br/>", " ",
		"<br />", " ",
		"<Info>", "",
		"</Info>", "",
		"<Warning>", "",
		"</Warning>", "",
		"<Note>", "",
		"</Note>", "",
	)
	return strings.Join(strings.Fields(replacer.Replace(value)), " ")
}

func humanize(value string) string {
	return strings.ReplaceAll(value, "-", " ")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func exactPathArgs(pathParams []string) cobra.PositionalArgs {
	if len(pathParams) == 0 {
		return cobra.NoArgs
	}
	return func(cmd *cobra.Command, args []string) error {
		if len(args) != len(pathParams) {
			return fmt.Errorf("expected %d path arguments (%s), got %d", len(pathParams), strings.Join(pathParams, ", "), len(args))
		}
		return nil
	}
}

func runLegacy(ctx context.Context, executor *cliruntime.Executor, args []string) error {
	exitCode := executor.Run(ctx, args)
	if exitCode != 0 {
		return exitError{code: exitCode}
	}
	return nil
}

func defaultWriter(writer io.Writer, fallback *os.File) io.Writer {
	if writer != nil {
		return writer
	}
	return fallback
}

func defaultReader(reader io.Reader, fallback *os.File) io.Reader {
	if reader != nil {
		return reader
	}
	return fallback
}
