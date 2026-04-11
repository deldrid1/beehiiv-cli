package cmd

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/deldrid1/beehiiv-cli/internal/commandset"
	"github.com/deldrid1/beehiiv-cli/internal/config"
)

const (
	commandGroupCore     = "core"
	commandGroupAuth     = "auth"
	commandGroupWorkflow = "workflow"
	commandGroupAPI      = "api"
)

func registerGlobalFlags(root *cobra.Command) {
	flags := root.PersistentFlags()
	flags.String("config", "", "Override the settings file location")
	flags.String("api-key", "", "Override the Beehiiv API key")
	flags.String("publication-id", "", "Override the Beehiiv publication ID")
	flags.String("base-url", "", "Override the API base URL")
	flags.Int("rate-limit-rpm", 0, "Override the internal rate limit")
	flags.String("timeout", "", "Override the request timeout, e.g. 45s")
	flags.String("output", "", "Choose the output format (json, table, raw)")
	flags.Bool("table", false, "Shorthand for --output table")
	flags.Bool("raw", false, "Shorthand for --output raw")
	flags.Bool("compact", false, "Print compact JSON")
	flags.Bool("debug", false, "Print request URLs to stderr")
	flags.Bool("verbose", false, "Print request and response details to stderr")

	_ = root.MarkPersistentFlagFilename("config")
	_ = root.RegisterFlagCompletionFunc("output", func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		return []string{"json", "table", "raw"}, cobra.ShellCompDirectiveNoFileComp
	})
}

func registerOperationFlags(command *cobra.Command, operation commandset.Operation) {
	flags := command.Flags()
	flags.StringArray("query", nil, "Add a repeatable query parameter in key=value form")
	if operation.Body {
		flags.String("body", "", "Provide a request body as inline JSON, @path, or - for stdin")
	}
	if operation.List {
		flags.Bool("all", false, "Fetch every page and aggregate the results")
	}
}


func commandOverrides(command *cobra.Command) (config.Overrides, error) {
	flags := command.Flags()

	timeout := ""
	if flag := flags.Lookup("timeout"); flag != nil {
		timeout = flag.Value.String()
	}

	rateLimitRPM := 0
	var err error
	if flag := flags.Lookup("rate-limit-rpm"); flag != nil && flag.Changed {
		rateLimitRPM, err = flags.GetInt("rate-limit-rpm")
		if err != nil {
			return config.Overrides{}, err
		}
	}

	output := ""
	if flag := flags.Lookup("output"); flag != nil {
		output = flag.Value.String()
	}
	if output == "" {
		switch {
		case changedBoolValue(flags, "raw"):
			output = config.OutputRaw
		case changedBoolValue(flags, "table"):
			output = config.OutputTable
		}
	}

	return config.Overrides{
		ConfigPath:    changedStringValue(flags, "config"),
		APIKey:        changedStringValue(flags, "api-key"),
		PublicationID: changedStringValue(flags, "publication-id"),
		BaseURL:       changedStringValue(flags, "base-url"),
		RateLimitRPM:  rateLimitRPM,
		Timeout:       parseDuration(timeout),
		Output:        output,
		Compact:       changedBoolValue(flags, "compact"),
		Debug:         changedBoolValue(flags, "debug"),
		Verbose:       changedBoolValue(flags, "verbose"),
	}, nil
}

func changedStringValue(flags *pflag.FlagSet, name string) string {
	flag := flags.Lookup(name)
	if flag == nil || !flag.Changed {
		return ""
	}
	return flag.Value.String()
}

func changedBoolValue(flags *pflag.FlagSet, name string) bool {
	flag := flags.Lookup(name)
	return flag != nil && flag.Changed
}

func parseDuration(value string) time.Duration {
	if value == "" {
		return 0
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0
	}
	return parsed
}

