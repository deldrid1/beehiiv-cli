package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/deldrid1/beehiiv-cli/internal/auth"
	"github.com/deldrid1/beehiiv-cli/internal/client"
	"github.com/deldrid1/beehiiv-cli/internal/commandset"
	clioutput "github.com/deldrid1/beehiiv-cli/internal/output"
	"github.com/deldrid1/beehiiv-cli/internal/pagination"
)

// executeOperation is the single execution path for all auto-generated API
// commands.  It resolves auth, builds the request from Cobra flags, executes
// the HTTP call (with optional pagination), and writes formatted output.
func executeOperation(ctx context.Context, cmd *cobra.Command, args []string, operation commandset.Operation, options Options) error {
	overrides, err := commandOverrides(cmd)
	if err != nil {
		return err
	}

	manager := auth.NewManager(options.Env, options.HTTPClient)
	runtime, err := manager.ResolveRuntime(ctx, overrides)
	if err != nil {
		return err
	}

	if operation.RequiresPublicationID && runtime.PublicationID == "" {
		return errors.New("publication id is required; use `beehiiv auth login`, `--publication-id`, or set BEEHIIV_PUBLICATION_ID")
	}

	pathValues, err := collectPathValues(operation, args)
	if err != nil {
		return err
	}

	queryRaw, _ := cmd.Flags().GetStringArray("query")
	queryValues, err := buildQueryValues(operation, queryRaw)
	if err != nil {
		return err
	}

	var bodyValues []string
	if operation.Body {
		if bodyFlag := changedStringValue(cmd.Flags(), "body"); bodyFlag != "" {
			bodyValues = []string{bodyFlag}
		}
	}
	body, err := loadBody(operation, bodyValues, cmd.InOrStdin())
	if err != nil {
		return err
	}

	apiClient := client.New(runtime, options.HTTPClient, cmd.ErrOrStderr())

	allRequested := changedBoolValue(cmd.Flags(), "all")
	if allRequested && !operation.List {
		return errors.New("`--all` is only valid for list commands")
	}

	if allRequested {
		items, summary, err := pagination.CollectAll(ctx, operation.Pagination, queryValues, func(callCtx context.Context, nextQuery url.Values) ([]byte, error) {
			response, execErr := apiClient.Execute(callCtx, operation, pathValues, nextQuery, body)
			if execErr != nil {
				return nil, execErr
			}
			return response.Body, nil
		})
		if err != nil {
			return writeAPIError(cmd.ErrOrStderr(), err)
		}

		payload := map[string]any{
			"data":       items,
			"pagination": summary,
		}
		return clioutput.Write(cmd.OutOrStdout(), payload, nil, runtime)
	}

	response, err := apiClient.Execute(ctx, operation, pathValues, queryValues, body)
	if err != nil {
		return writeAPIError(cmd.ErrOrStderr(), err)
	}

	if len(response.Body) == 0 {
		return clioutput.Write(cmd.OutOrStdout(), map[string]any{"status": response.StatusCode}, nil, runtime)
	}

	var decoded any
	if err := json.Unmarshal(response.Body, &decoded); err != nil {
		_, _ = io.WriteString(cmd.OutOrStdout(), string(response.Body))
		if !strings.HasSuffix(string(response.Body), "\n") {
			_, _ = io.WriteString(cmd.OutOrStdout(), "\n")
		}
		return nil
	}

	return clioutput.Write(cmd.OutOrStdout(), decoded, response.Body, runtime)
}

// ---------------------------------------------------------------------------
// Helpers ported from internal/cli/app.go
// ---------------------------------------------------------------------------

func collectPathValues(operation commandset.Operation, positionals []string) (map[string]string, error) {
	if len(positionals) != len(operation.PathParams) {
		return nil, fmt.Errorf("expected %d path arguments (%s), got %d",
			len(operation.PathParams), strings.Join(operation.PathParams, ", "), len(positionals))
	}
	values := make(map[string]string, len(operation.PathParams))
	for index, name := range operation.PathParams {
		values[name] = positionals[index]
	}
	return values, nil
}

func buildQueryValues(operation commandset.Operation, rawQueries []string) (url.Values, error) {
	values := make(url.Values)
	for _, rawQuery := range rawQueries {
		name, rawValue, ok := strings.Cut(rawQuery, "=")
		if !ok {
			return nil, fmt.Errorf("queries must be formatted as key=value: %q", rawQuery)
		}
		name = normalizeQueryName(operation, name)
		for _, part := range strings.Split(rawValue, ",") {
			values.Add(name, part)
		}
	}
	return values, nil
}

func normalizeQueryName(operation commandset.Operation, name string) string {
	name = strings.TrimSpace(name)
	for _, parameter := range operation.QueryParams {
		if parameter.Name == name {
			return name
		}
		if strings.TrimSuffix(parameter.Name, "[]") == strings.TrimSuffix(name, "[]") {
			return parameter.Name
		}
	}
	return name
}

func loadBody(operation commandset.Operation, values []string, stdin io.Reader) ([]byte, error) {
	if len(values) == 0 {
		return nil, nil
	}
	if !operation.Body {
		return nil, fmt.Errorf("%s does not accept a request body", strings.Join(operation.Command, " "))
	}
	raw := values[len(values)-1]
	var data []byte
	switch {
	case raw == "-":
		var err error
		data, err = io.ReadAll(stdin)
		if err != nil {
			return nil, err
		}
	case strings.HasPrefix(raw, "@"):
		fileData, err := os.ReadFile(strings.TrimPrefix(raw, "@"))
		if err != nil {
			return nil, err
		}
		data = fileData
	default:
		data = []byte(raw)
	}
	if len(data) > 0 && !json.Valid(data) {
		return nil, errors.New("request body must be valid JSON")
	}
	return data, nil
}

// writeAPIError formats an error as structured JSON on stderr, matching the
// legacy output format.  It always returns an exitError so callers can
// propagate the non-zero exit code.
func writeAPIError(stderr io.Writer, err error) error {
	payload := map[string]any{
		"error": map[string]any{
			"message": err.Error(),
		},
	}
	if apiErr, ok := err.(*client.Error); ok {
		errorBody := map[string]any{
			"operation": apiErr.Operation,
			"status":    apiErr.Status,
			"message":   apiErr.Message,
		}
		if len(apiErr.Body) > 0 {
			var decoded any
			if json.Unmarshal(apiErr.Body, &decoded) == nil {
				errorBody["body"] = decoded
			}
		}
		payload["error"] = errorBody
	}
	data, marshalErr := json.MarshalIndent(payload, "", "  ")
	if marshalErr != nil {
		fmt.Fprintf(stderr, "%s\n", err.Error())
		return exitError{code: 1}
	}
	fmt.Fprintln(stderr, string(data))
	return exitError{code: 1}
}
