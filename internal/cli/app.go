package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/deldrid1/beehiiv-cli/internal/auth"
	"github.com/deldrid1/beehiiv-cli/internal/client"
	"github.com/deldrid1/beehiiv-cli/internal/commandset"
	"github.com/deldrid1/beehiiv-cli/internal/config"
	"github.com/deldrid1/beehiiv-cli/internal/pagination"
)

type App struct {
	stdin      io.Reader
	stdout     io.Writer
	stderr     io.Writer
	env        map[string]string
	httpClient client.HTTPClient
}

type parseResult struct {
	flags       map[string][]string
	positionals []string
}

type flagSpec struct {
	needsValue bool
}

func NewApp(stdin io.Reader, stdout, stderr io.Writer, env map[string]string, httpClient client.HTTPClient) *App {
	if stdin == nil {
		stdin = os.Stdin
	}
	if stdout == nil {
		stdout = os.Stdout
	}
	if stderr == nil {
		stderr = os.Stderr
	}
	if env == nil {
		env = make(map[string]string)
		for _, entry := range os.Environ() {
			key, value, ok := strings.Cut(entry, "=")
			if ok {
				env[key] = value
			}
		}
	}
	return &App{
		stdin:      stdin,
		stdout:     stdout,
		stderr:     stderr,
		env:        env,
		httpClient: httpClient,
	}
}

func (a *App) Run(ctx context.Context, args []string) int {
	leading, remainder, err := parseLeadingGlobals(args)
	if err != nil {
		a.writeError(err)
		return 1
	}

	if len(remainder) == 0 || hasHelpFlag(remainder) || isHelpOnly(leading.flags) {
		a.printRootHelp()
		return 0
	}

	if remainder[0] == "login" {
		return a.runAuth(ctx, append([]string{"login"}, remainder[1:]...))
	}
	if remainder[0] == "auth" {
		return a.runAuth(ctx, remainder[1:])
	}

	group := remainder[0]
	groupExists, groupErr := commandset.GroupExists(group)
	if groupErr != nil {
		a.writeError(groupErr)
		return 1
	}
	if !groupExists {
		a.writeError(fmt.Errorf("unknown command group %q", group))
		a.printRootHelp()
		return 1
	}

	if len(remainder) == 1 {
		a.printGroupHelp(group)
		return 0
	}

	action := remainder[1]
	if action == "--help" || action == "-h" {
		a.printGroupHelp(group)
		return 0
	}

	operation, found, findErr := commandset.Find(group, action)
	if findErr != nil {
		a.writeError(findErr)
		return 1
	}
	if !found {
		a.writeError(fmt.Errorf("unknown command %q", strings.Join(remainder[:2], " ")))
		a.printGroupHelp(group)
		return 1
	}

	parsed, err := parseTokens(remainder[2:], mergeFlagSpecs(operation))
	if err != nil {
		a.writeError(err)
		return 1
	}
	if hasHelpFlagMap(parsed.flags) {
		a.printOperationHelp(operation)
		return 0
	}

	overrides, err := buildOverrides(leading, parsed)
	if err != nil {
		a.writeError(err)
		return 1
	}

	runtime, err := a.loadRuntime(ctx, overrides)
	if err != nil {
		a.writeError(err)
		return 1
	}

	if operation.RequiresPublicationID && runtime.PublicationID == "" {
		a.writeError(errors.New("publication id is required; use `auth login`, `--publication-id`, or BEEHIIV_PUBLICATION_ID"))
		return 1
	}

	pathValues, err := collectPathValues(operation, parsed.positionals)
	if err != nil {
		a.writeError(err)
		return 1
	}

	queryValues, err := buildQueryValues(operation, parsed.flags["query"])
	if err != nil {
		a.writeError(err)
		return 1
	}

	body, err := loadBody(operation, parsed.flags["body"], a.stdin)
	if err != nil {
		a.writeError(err)
		return 1
	}

	apiClient := client.New(runtime, a.httpClient, a.stderr)
	allRequested := hasBoolFlag(parsed.flags, "all")
	if allRequested && !operation.List {
		a.writeError(fmt.Errorf("`--all` is only valid for list commands"))
		return 1
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
			a.writeError(err)
			return 1
		}

		payload := map[string]any{
			"data":       items,
			"pagination": summary,
		}
		a.writeOutput(payload, nil, runtime)
		return 0
	}

	response, err := apiClient.Execute(ctx, operation, pathValues, queryValues, body)
	if err != nil {
		a.writeError(err)
		return 1
	}

	if len(response.Body) == 0 {
		a.writeOutput(map[string]any{"status": response.StatusCode}, nil, runtime)
		return 0
	}

	var decoded any
	if err := json.Unmarshal(response.Body, &decoded); err != nil {
		io.WriteString(a.stdout, string(response.Body))
		if !strings.HasSuffix(string(response.Body), "\n") {
			io.WriteString(a.stdout, "\n")
		}
		return 0
	}

	a.writeOutput(decoded, response.Body, runtime)
	return 0
}

func (a *App) loadRuntime(ctx context.Context, overrides config.Overrides) (config.Runtime, error) {
	manager := auth.NewManager(a.env, a.httpClient)
	return manager.ResolveRuntime(ctx, overrides)
}

func (a *App) runAuth(ctx context.Context, args []string) int {
	if len(args) == 0 || hasHelpFlag(args) {
		a.printAuthHelp()
		return 0
	}

	manager := auth.NewManager(a.env, a.httpClient)

	switch args[0] {
	case "login":
		parsed, err := parseTokens(args[1:], mergeFlagSpecs(commandset.Operation{}))
		if err != nil {
			a.writeError(err)
			return 1
		}
		if hasHelpFlagMap(parsed.flags) {
			a.printAuthLoginHelp()
			return 0
		}

		overrides, err := buildOverrides(parseResult{}, parsed)
		if err != nil {
			a.writeError(err)
			return 1
		}
		runtime, err := config.LoadRuntime(overrides, a.env)
		if err != nil {
			a.writeError(err)
			return 1
		}

		apiKey := firstFlag(parsed.flags, "api-key")
		if apiKey == "" {
			fmt.Fprintf(a.stderr, "Enter your Beehiiv API key. Create one as described here: https://developers.beehiiv.com/welcome/create-an-api-key\n")
			apiKey, err = promptValue(a.stdin, a.stderr, "API key")
			if err != nil {
				a.writeError(err)
				return 1
			}
		}

		publicationID := firstFlag(parsed.flags, "publication-id")
		if publicationID == "" {
			fmt.Fprintf(a.stderr, "Finding publication IDs via GET /publications so you do not have to copy one manually.\n")
			loginRuntime := runtime
			loginRuntime.APIKey = apiKey
			loginRuntime.PublicationID = ""
			apiClient := client.New(loginRuntime, a.httpClient, a.stderr)

			operation, found, err := commandset.Find("publications", "list")
			if err != nil {
				a.writeError(err)
				return 1
			}
			if !found {
				a.writeError(errors.New("publications list operation is unavailable"))
				return 1
			}

			response, execErr := apiClient.Execute(ctx, operation, map[string]string{}, url.Values{}, nil)
			if execErr != nil {
				a.writeError(execErr)
				return 1
			}

			var payload struct {
				Data []struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"data"`
			}
			if err := json.Unmarshal(response.Body, &payload); err != nil {
				a.writeError(err)
				return 1
			}
			if len(payload.Data) == 0 {
				a.writeError(errors.New("no publications were returned for this API key"))
				return 1
			}
			if len(payload.Data) == 1 {
				publicationID = payload.Data[0].ID
				fmt.Fprintf(a.stderr, "Using publication %s (%s)\n", payload.Data[0].ID, payload.Data[0].Name)
			} else {
				publicationID, err = promptPublicationChoice(a.stdin, a.stderr, payload.Data)
				if err != nil {
					a.writeError(err)
					return 1
				}
			}
		}

		if err := manager.SaveAPIKeySession(auth.APIKeyLoginOptions{
			SettingsPath:  runtime.ConfigPath,
			APIKey:        apiKey,
			PublicationID: publicationID,
			BaseURL:       runtime.BaseURL,
			RateLimitRPM:  runtime.RateLimitRPM,
		}); err != nil {
			a.writeError(err)
			return 1
		}

		a.writeOutput(map[string]any{
			"message":        "Beehiiv credentials saved in the OS keyring",
			"auth_mode":      config.AuthModeAPIKey,
			"publication_id": publicationID,
			"settings_path":  runtime.ConfigPath,
			"secret_backend": config.SecretBackendKeyring,
		}, nil, runtime)
		return 0
	case "status":
		parsed, err := parseTokens(args[1:], mergeFlagSpecs(commandset.Operation{}))
		if err != nil {
			a.writeError(err)
			return 1
		}
		overrides, err := buildOverrides(parseResult{}, parsed)
		if err != nil {
			a.writeError(err)
			return 1
		}
		runtime, err := config.LoadRuntime(overrides, a.env)
		if err != nil {
			a.writeError(err)
			return 1
		}
		status, err := manager.Status(overrides)
		if err != nil {
			a.writeError(err)
			return 1
		}
		a.writeOutput(map[string]any{
			"configured":         status.Configured,
			"auth_mode":          status.AuthMode,
			"secret_backend":     status.SecretBackend,
			"publication_id":     status.PublicationID,
			"base_url":           status.BaseURL,
			"rate_limit_rpm":     status.RateLimitRPM,
			"settings_path":      status.SettingsPath,
			"token_source":       status.TokenSource,
			"token_expires_at":   status.TokenExpiresAt,
			"token_scope":        status.TokenScope,
			"oauth_client_id":    status.OAuthClientID,
			"oauth_redirect_uri": status.OAuthRedirectURI,
			"oauth_scopes":       status.OAuthScopes,
			"resource_owner_id":  status.ResourceOwnerID,
			"application_uid":    status.ApplicationUID,
			"application_name":   status.ApplicationName,
			"client_has_secret":  status.ClientHasSecret,
		}, nil, runtime)
		return 0
	case "path":
		parsed, err := parseTokens(args[1:], mergeFlagSpecs(commandset.Operation{}))
		if err != nil {
			a.writeError(err)
			return 1
		}
		overrides, err := buildOverrides(parseResult{}, parsed)
		if err != nil {
			a.writeError(err)
			return 1
		}
		runtime, err := config.LoadRuntime(overrides, a.env)
		if err != nil {
			a.writeError(err)
			return 1
		}
		paths, err := manager.Paths(runtime.ConfigPath)
		if err != nil {
			a.writeError(err)
			return 1
		}
		a.writeOutput(map[string]any{"settings_path": paths.SettingsPath}, nil, runtime)
		return 0
	case "logout":
		parsed, err := parseTokens(args[1:], mergeFlagSpecs(commandset.Operation{}))
		if err != nil {
			a.writeError(err)
			return 1
		}
		overrides, err := buildOverrides(parseResult{}, parsed)
		if err != nil {
			a.writeError(err)
			return 1
		}
		runtime, err := config.LoadRuntime(overrides, a.env)
		if err != nil {
			a.writeError(err)
			return 1
		}
		if err := manager.Logout(ctx, runtime.ConfigPath, false); err != nil {
			a.writeError(err)
			return 1
		}
		a.writeOutput(map[string]any{
			"message":       "Beehiiv credentials cleared",
			"revoked":       false,
			"settings_path": runtime.ConfigPath,
		}, nil, runtime)
		return 0
	default:
		a.writeError(fmt.Errorf("unknown auth command %q", args[0]))
		a.printAuthHelp()
		return 1
	}
}

func parseLeadingGlobals(args []string) (parseResult, []string, error) {
	specs := globalFlagSpecs()
	flags := make(map[string][]string)
	index := 0
	for index < len(args) {
		token := args[index]
		if token == "--" {
			index++
			break
		}
		if !strings.HasPrefix(token, "-") {
			break
		}

		name, value, hasInline, ok := parseFlagToken(token)
		if !ok {
			break
		}
		spec, exists := specs[name]
		if !exists {
			break
		}
		if spec.needsValue {
			if !hasInline {
				if index+1 >= len(args) {
					return parseResult{}, nil, fmt.Errorf("flag --%s requires a value", name)
				}
				value = args[index+1]
				index++
			}
			flags[name] = append(flags[name], value)
		} else {
			flags[name] = append(flags[name], "true")
		}
		index++
	}
	return parseResult{flags: flags}, args[index:], nil
}

func parseTokens(args []string, specs map[string]flagSpec) (parseResult, error) {
	result := parseResult{flags: make(map[string][]string)}
	for index := 0; index < len(args); index++ {
		token := args[index]
		if token == "--" {
			result.positionals = append(result.positionals, args[index+1:]...)
			break
		}
		if !strings.HasPrefix(token, "-") {
			result.positionals = append(result.positionals, token)
			continue
		}

		name, value, hasInline, ok := parseFlagToken(token)
		if !ok {
			result.positionals = append(result.positionals, token)
			continue
		}
		spec, exists := specs[name]
		if !exists {
			return parseResult{}, fmt.Errorf("unknown flag --%s", name)
		}
		if spec.needsValue {
			if !hasInline {
				if index+1 >= len(args) {
					return parseResult{}, fmt.Errorf("flag --%s requires a value", name)
				}
				value = args[index+1]
				index++
			}
			result.flags[name] = append(result.flags[name], value)
			continue
		}
		result.flags[name] = append(result.flags[name], "true")
	}
	return result, nil
}

func parseFlagToken(token string) (name, value string, hasInline, ok bool) {
	switch token {
	case "-h":
		return "help", "", false, true
	case "--help":
		return "help", "", false, true
	}
	if !strings.HasPrefix(token, "--") {
		return "", "", false, false
	}
	trimmed := strings.TrimPrefix(token, "--")
	name, value, hasInline = strings.Cut(trimmed, "=")
	return name, value, hasInline, true
}

func mergeFlagSpecs(operation commandset.Operation) map[string]flagSpec {
	specs := globalFlagSpecs()
	specs["help"] = flagSpec{}
	specs["body"] = flagSpec{needsValue: true}
	specs["query"] = flagSpec{needsValue: true}
	specs["all"] = flagSpec{}
	return specs
}

func globalFlagSpecs() map[string]flagSpec {
	return map[string]flagSpec{
		"config":         {needsValue: true},
		"api-key":        {needsValue: true},
		"publication-id": {needsValue: true},
		"base-url":       {needsValue: true},
		"rate-limit-rpm": {needsValue: true},
		"timeout":        {needsValue: true},
		"output":         {needsValue: true},
		"table":          {},
		"raw":            {},
		"compact":        {},
		"debug":          {},
		"verbose":        {},
		"help":           {},
	}
}

func buildOverrides(parts ...parseResult) (config.Overrides, error) {
	combined := parseResult{flags: make(map[string][]string)}
	for _, part := range parts {
		for key, values := range part.flags {
			combined.flags[key] = append(combined.flags[key], values...)
		}
	}

	timeout := time.Duration(0)
	if raw := firstFlag(combined.flags, "timeout"); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil {
			return config.Overrides{}, fmt.Errorf("invalid timeout %q", raw)
		}
		timeout = parsed
	}

	rateLimit := 0
	if raw := firstFlag(combined.flags, "rate-limit-rpm"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return config.Overrides{}, fmt.Errorf("invalid rate limit %q", raw)
		}
		rateLimit = parsed
	}

	output := strings.ToLower(strings.TrimSpace(firstFlag(combined.flags, "output")))
	if output == "" {
		switch {
		case hasBoolFlag(combined.flags, "raw"):
			output = config.OutputRaw
		case hasBoolFlag(combined.flags, "table"):
			output = config.OutputTable
		default:
			output = config.OutputJSON
		}
	}

	return config.Overrides{
		ConfigPath:    firstFlag(combined.flags, "config"),
		APIKey:        firstFlag(combined.flags, "api-key"),
		PublicationID: firstFlag(combined.flags, "publication-id"),
		BaseURL:       firstFlag(combined.flags, "base-url"),
		RateLimitRPM:  rateLimit,
		Timeout:       timeout,
		Output:        output,
		Compact:       hasBoolFlag(combined.flags, "compact"),
		Debug:         hasBoolFlag(combined.flags, "debug"),
		Verbose:       hasBoolFlag(combined.flags, "verbose"),
	}, nil
}

func collectPathValues(operation commandset.Operation, positionals []string) (map[string]string, error) {
	if len(positionals) != len(operation.PathParams) {
		return nil, fmt.Errorf("expected %d path arguments (%s), got %d", len(operation.PathParams), strings.Join(operation.PathParams, ", "), len(positionals))
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
		parts := strings.Split(rawValue, ",")
		for _, part := range parts {
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

func promptValue(input io.Reader, output io.Writer, label string) (string, error) {
	fmt.Fprintf(output, "%s: ", label)
	reader := bufio.NewReader(input)
	value, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	return strings.TrimSpace(value), nil
}

func promptPublicationChoice(input io.Reader, output io.Writer, publications []struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}) (string, error) {
	fmt.Fprintf(output, "Select a publication ID:\n")
	for index, publication := range publications {
		fmt.Fprintf(output, "%d. %s (%s)\n", index+1, publication.ID, publication.Name)
	}
	reader := bufio.NewReader(input)
	for {
		fmt.Fprintf(output, "Publication number: ")
		raw, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return "", err
		}
		choice, parseErr := strconv.Atoi(strings.TrimSpace(raw))
		if parseErr == nil && choice >= 1 && choice <= len(publications) {
			return publications[choice-1].ID, nil
		}
		fmt.Fprintf(output, "Please enter a number between 1 and %d.\n", len(publications))
	}
}

func (a *App) printRootHelp() {
	groups, err := commandset.Groups()
	if err != nil {
		a.writeError(err)
		return
	}
	fmt.Fprintln(a.stdout, "beehiiv")
	fmt.Fprintln(a.stdout, "")
	fmt.Fprintln(a.stdout, "Self-documenting Beehiiv API CLI for macOS and Windows.")
	fmt.Fprintln(a.stdout, "")
	fmt.Fprintln(a.stdout, "Usage:")
	fmt.Fprintln(a.stdout, "  beehiiv [global-flags] <command-group> <action> [path-args] [command-flags]")
	fmt.Fprintln(a.stdout, "  beehiiv auth <login|status|path|logout>")
	fmt.Fprintln(a.stdout, "  beehiiv login")
	fmt.Fprintln(a.stdout, "")
	fmt.Fprintln(a.stdout, "Global flags:")
	fmt.Fprintln(a.stdout, "  --config <path>           Override the settings file location")
	fmt.Fprintln(a.stdout, "  --api-key <token>         Override the Beehiiv API key")
	fmt.Fprintln(a.stdout, "  --publication-id <id>     Override the Beehiiv publication ID")
	fmt.Fprintln(a.stdout, "  --base-url <url>          Override the API base URL")
	fmt.Fprintln(a.stdout, "  --rate-limit-rpm <int>    Override the internal rate limit")
	fmt.Fprintln(a.stdout, "  --timeout <duration>      Override the request timeout, e.g. 45s")
	fmt.Fprintln(a.stdout, "  --output <json|table|raw> Choose the output format")
	fmt.Fprintln(a.stdout, "  --table                   Shorthand for --output table")
	fmt.Fprintln(a.stdout, "  --raw                     Shorthand for --output raw")
	fmt.Fprintln(a.stdout, "  --compact                 Print compact JSON")
	fmt.Fprintln(a.stdout, "  --debug                   Print request URLs to stderr")
	fmt.Fprintln(a.stdout, "  --verbose                 Print request and response details to stderr")
	fmt.Fprintln(a.stdout, "  --help                    Show help")
	fmt.Fprintln(a.stdout, "")
	fmt.Fprintln(a.stdout, "Command groups:")
	for _, group := range groups {
		fmt.Fprintf(a.stdout, "  %s\n", group)
	}
	fmt.Fprintln(a.stdout, "")
	fmt.Fprintln(a.stdout, "Run `beehiiv <group>` or `beehiiv <group> <action> --help` for details.")
}

func (a *App) printGroupHelp(group string) {
	operations, err := commandset.OperationsForGroup(group)
	if err != nil {
		a.writeError(err)
		return
	}
	fmt.Fprintf(a.stdout, "beehiiv %s\n\n", group)
	fmt.Fprintln(a.stdout, "Actions:")
	for _, operation := range operations {
		fmt.Fprintf(a.stdout, "  %-18s %s\n", operation.Command[1], firstNonEmpty(operation.Summary, operation.Description))
	}
}

func (a *App) printOperationHelp(operation commandset.Operation) {
	fmt.Fprintf(a.stdout, "beehiiv %s %s\n\n", operation.Command[0], operation.Command[1])
	if operation.Summary != "" {
		fmt.Fprintf(a.stdout, "%s\n\n", operation.Summary)
	}
	fmt.Fprintf(a.stdout, "Method: %s\n", operation.Method)
	fmt.Fprintf(a.stdout, "Path:   %s\n\n", operation.Path)
	fmt.Fprintf(a.stdout, "Usage:\n  beehiiv %s %s", operation.Command[0], operation.Command[1])
	for _, pathParam := range operation.PathParams {
		fmt.Fprintf(a.stdout, " <%s>", pathParam)
	}
	fmt.Fprintln(a.stdout, " [--query key=value] [--body json|@file|-] [--all]")
	if operation.RequiresPublicationID {
		fmt.Fprintln(a.stdout, "")
		fmt.Fprintln(a.stdout, "This command requires a publication ID from auth/login, config, env, or --publication-id.")
	}
	if len(operation.PathParams) > 0 {
		fmt.Fprintln(a.stdout, "")
		fmt.Fprintln(a.stdout, "Path parameters:")
		for _, pathParam := range operation.PathParams {
			fmt.Fprintf(a.stdout, "  %s\n", pathParam)
		}
	}
	if len(operation.QueryParams) > 0 {
		fmt.Fprintln(a.stdout, "")
		fmt.Fprintln(a.stdout, "Query parameters:")
		for _, queryParam := range operation.QueryParams {
			fmt.Fprintf(a.stdout, "  %s", queryParam.Name)
			if queryParam.Multiple {
				fmt.Fprint(a.stdout, " (repeatable)")
			}
			fmt.Fprintln(a.stdout)
			if queryParam.Description != "" {
				fmt.Fprintf(a.stdout, "    %s\n", stripHTML(queryParam.Description))
			}
		}
	}
	if operation.Body {
		fmt.Fprintln(a.stdout, "")
		fmt.Fprintln(a.stdout, "Body:")
		fmt.Fprintln(a.stdout, "  Pass JSON with --body '{...}', --body @request.json, or --body - to read stdin.")
	}
	if operation.List {
		fmt.Fprintln(a.stdout, "")
		fmt.Fprintln(a.stdout, "Pagination:")
		fmt.Fprintf(a.stdout, "  Default: first page only. Use --all to exhaust all pages. Pagination mode: %s.\n", operation.Pagination)
	}
}

func (a *App) printAuthHelp() {
	fmt.Fprintln(a.stdout, "beehiiv auth")
	fmt.Fprintln(a.stdout, "")
	fmt.Fprintln(a.stdout, "Commands:")
	fmt.Fprintln(a.stdout, "  login     Save an API key and publication ID securely")
	fmt.Fprintln(a.stdout, "  status    Show masked auth status without printing live credentials")
	fmt.Fprintln(a.stdout, "  path      Print the settings file path")
	fmt.Fprintln(a.stdout, "  logout    Remove saved Beehiiv credentials")
}

func (a *App) printAuthLoginHelp() {
	fmt.Fprintln(a.stdout, "beehiiv auth login")
	fmt.Fprintln(a.stdout, "")
	fmt.Fprintln(a.stdout, "Stores Beehiiv credentials in the OS keyring and writes non-secret settings to config.json.")
	fmt.Fprintln(a.stdout, "Create an API key as described here: https://developers.beehiiv.com/welcome/create-an-api-key")
	fmt.Fprintln(a.stdout, "Publication IDs are the `pub_...` values returned by `beehiiv publications list`.")
}

func (a *App) writeJSON(value any, compact bool) {
	var data []byte
	var err error
	if compact {
		data, err = json.Marshal(value)
	} else {
		data, err = json.MarshalIndent(value, "", "  ")
	}
	if err != nil {
		a.writeError(err)
		return
	}
	fmt.Fprintln(a.stdout, string(data))
}

func (a *App) writeError(err error) {
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
		fmt.Fprintf(a.stderr, "%s\n", err.Error())
		return
	}
	fmt.Fprintln(a.stderr, string(data))
}

func hasHelpFlag(args []string) bool {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return true
		}
	}
	return false
}

func hasHelpFlagMap(flags map[string][]string) bool {
	return hasBoolFlag(flags, "help")
}

func hasBoolFlag(flags map[string][]string, name string) bool {
	return len(flags[name]) > 0
}

func firstFlag(flags map[string][]string, name string) string {
	if values := flags[name]; len(values) > 0 {
		return values[len(values)-1]
	}
	return ""
}

func isHelpOnly(flags map[string][]string) bool {
	return len(flags) == 1 && hasBoolFlag(flags, "help")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func stripHTML(value string) string {
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
