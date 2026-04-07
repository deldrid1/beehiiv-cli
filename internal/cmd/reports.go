package cmd

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/deldrid1/beehiiv-cli/internal/client"
	"github.com/deldrid1/beehiiv-cli/internal/commandset"
	"github.com/deldrid1/beehiiv-cli/internal/config"
	clioutput "github.com/deldrid1/beehiiv-cli/internal/output"
	"github.com/deldrid1/beehiiv-cli/internal/pagination"
)

type reportExecutor struct {
	apiClient *client.Client
}

type reportSection struct {
	Title string
	Value any
}

type chartPoint struct {
	Label string  `json:"label"`
	Value float64 `json:"value"`
}

func newReportsCommand(options Options) *cobra.Command {
	command := &cobra.Command{
		Use:     "reports",
		Aliases: []string{"report", "insights"},
		Short:   "Guided stats, charts, and CSV exports for a publication",
		Long: "Generate friendly publication summaries, engagement charts, and spreadsheet-ready " +
			"CSV exports without stitching raw Beehiiv endpoints together by hand.",
		Example: strings.TrimSpace(`
beehiiv reports summary
beehiiv reports chart --metric unique_opens --days 14
beehiiv reports export subscriptions --file subscriptions.csv
`),
		GroupID: commandGroupWorkflow,
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	command.AddCommand(
		newReportsSummaryCommand(options),
		newReportsChartCommand(options),
		newReportsExportCommand(options),
	)

	return command
}

func newReportsSummaryCommand(options Options) *cobra.Command {
	command := &cobra.Command{
		Use:   "summary",
		Short: "Create a readable publication summary",
		Long: "Build a business-friendly summary using publication stats, post rollups, recent posts, " +
			"and recent engagement totals.",
		Example: strings.TrimSpace(`
beehiiv reports summary
beehiiv reports summary --days 30 --recent-posts 10
beehiiv reports summary --output json
`),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			executor, runtimeConfig, err := newReportExecutor(cmd, options, config.OutputTable)
			if err != nil {
				return err
			}

			days, err := cmd.Flags().GetInt("days")
			if err != nil {
				return err
			}
			recentPosts, err := cmd.Flags().GetInt("recent-posts")
			if err != nil {
				return err
			}
			if days < 1 || days > 31 {
				return errors.New("--days must be between 1 and 31")
			}
			if recentPosts < 1 || recentPosts > 25 {
				return errors.New("--recent-posts must be between 1 and 25")
			}

			publicationPayload, err := executor.executeJSON(cmd.Context(), "publications", "get", nil, url.Values{
				"expand": {"stats"},
			})
			if err != nil {
				return err
			}
			postStatsPayload, err := executor.executeJSON(cmd.Context(), "posts", "aggregate-stats", nil, nil)
			if err != nil {
				return err
			}
			postsPayload, err := executor.executeJSON(cmd.Context(), "posts", "list", nil, url.Values{
				"expand":    {"stats"},
				"limit":     {strconv.Itoa(recentPosts)},
				"status":    {"confirmed"},
				"order_by":  {"publish_date"},
				"direction": {"desc"},
			})
			if err != nil {
				return err
			}
			startDate := time.Now().UTC().AddDate(0, 0, -(days - 1)).Format("2006-01-02")
			engagementPayload, err := executor.executeJSON(cmd.Context(), "engagements", "list", nil, url.Values{
				"start_date":     {startDate},
				"number_of_days": {strconv.Itoa(days)},
				"granularity":    {"day"},
				"direction":      {"asc"},
			})
			if err != nil {
				return err
			}

			publication := buildPublicationSummary(dataMap(publicationPayload))
			postRollup := buildPostRollup(dataMap(postStatsPayload))
			recentPostRows := buildRecentPostSummary(dataList(postsPayload))
			engagementRows := dataList(engagementPayload)
			engagementSummary := buildEngagementSummary(engagementRows, days)

			payload := map[string]any{
				"publication":        publication,
				"post_rollup":        postRollup,
				"engagement_summary": engagementSummary,
				"recent_posts":       recentPostRows,
			}

			if usesStructuredOutput(runtimeConfig) {
				return clioutput.Write(cmd.OutOrStdout(), normalizeOutputValue(payload), nil, runtimeConfig)
			}

			return writeReportSections(cmd.OutOrStdout(), []reportSection{
				{Title: "publication", Value: publication},
				{Title: "post_rollup", Value: postRollup},
				{Title: "engagement_summary", Value: engagementSummary},
				{Title: "recent_posts", Value: recentPostRows},
			})
		},
	}
	command.Flags().Int("days", 7, "How many recent days of engagement activity to summarize")
	command.Flags().Int("recent-posts", 5, "How many recent posts to include in the summary")
	return command
}

func newReportsChartCommand(options Options) *cobra.Command {
	command := &cobra.Command{
		Use:   "chart",
		Short: "Render an ASCII engagement chart",
		Long: "Plot recent Beehiiv engagement metrics in the terminal. By default this prints a plain " +
			"text chart; pass --output json for machine-readable chart data.",
		Example: strings.TrimSpace(`
beehiiv reports chart
beehiiv reports chart --metric unique_clicks --days 14
beehiiv reports chart --metric total_verified_clicks --granularity week
`),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			executor, runtimeConfig, err := newReportExecutor(cmd, options, config.OutputTable)
			if err != nil {
				return err
			}

			days, err := cmd.Flags().GetInt("days")
			if err != nil {
				return err
			}
			granularity, err := cmd.Flags().GetString("granularity")
			if err != nil {
				return err
			}
			emailType, err := cmd.Flags().GetString("email-type")
			if err != nil {
				return err
			}
			metric, err := cmd.Flags().GetString("metric")
			if err != nil {
				return err
			}
			width, err := cmd.Flags().GetInt("width")
			if err != nil {
				return err
			}
			if days < 1 || days > 31 {
				return errors.New("--days must be between 1 and 31")
			}
			if width < 10 || width > 80 {
				return errors.New("--width must be between 10 and 80")
			}

			metric = normalizeEngagementMetric(metric)
			if !isSupportedEngagementMetric(metric) {
				return fmt.Errorf("unsupported --metric %q", metric)
			}

			startDate := time.Now().UTC().AddDate(0, 0, -(days - 1)).Format("2006-01-02")
			engagementPayload, err := executor.executeJSON(cmd.Context(), "engagements", "list", nil, url.Values{
				"start_date":     {startDate},
				"number_of_days": {strconv.Itoa(days)},
				"granularity":    {granularity},
				"email_type":     {emailType},
				"direction":      {"asc"},
			})
			if err != nil {
				return err
			}

			rows := dataList(engagementPayload)
			points := make([]chartPoint, 0, len(rows))
			for _, row := range rows {
				points = append(points, chartPoint{
					Label: stringValue(row["date"]),
					Value: numericValue(row[metric]),
				})
			}

			title := fmt.Sprintf("Beehiiv engagement chart (%s)", metric)
			chart := renderASCIIChart(title, points, width)
			payload := map[string]any{
				"title":       title,
				"metric":      metric,
				"granularity": granularity,
				"email_type":  emailType,
				"days":        days,
				"chart":       chart,
				"data":        points,
			}

			if usesStructuredOutput(runtimeConfig) {
				return clioutput.Write(cmd.OutOrStdout(), normalizeOutputValue(payload), nil, runtimeConfig)
			}

			_, err = io.WriteString(cmd.OutOrStdout(), chart)
			if err == nil && !strings.HasSuffix(chart, "\n") {
				_, err = io.WriteString(cmd.OutOrStdout(), "\n")
			}
			return err
		},
	}
	command.Flags().Int("days", 7, "How many recent days to include")
	command.Flags().String("granularity", "day", "Chart granularity: day, week, or month")
	command.Flags().String("email-type", "all", "Filter engagement metrics by email type: all, post, or message")
	command.Flags().String("metric", "total_opens", "Metric to chart: total_opens, unique_opens, total_clicks, unique_clicks, total_verified_clicks, or unique_verified_clicks")
	command.Flags().Int("width", 36, "Maximum width of the ASCII bar area")
	return command
}

func newReportsExportCommand(options Options) *cobra.Command {
	command := &cobra.Command{
		Use:   "export",
		Short: "Export common Beehiiv datasets to CSV",
		Long:  "Export Beehiiv data to CSV files that open cleanly in Excel, Numbers, and Google Sheets.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	command.AddCommand(
		newReportsExportSubscriptionsCommand(options),
		newReportsExportPostsCommand(options),
		newReportsExportEngagementsCommand(options),
	)
	return command
}

func newReportsExportSubscriptionsCommand(options Options) *cobra.Command {
	command := &cobra.Command{
		Use:   "subscriptions",
		Short: "Export subscriptions to CSV",
		Example: strings.TrimSpace(`
beehiiv reports export subscriptions --file subscriptions.csv
beehiiv reports export subscriptions --status active
`),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			executor, _, err := newReportExecutor(cmd, options, config.OutputTable)
			if err != nil {
				return err
			}

			status, err := cmd.Flags().GetString("status")
			if err != nil {
				return err
			}
			filePath, err := cmd.Flags().GetString("file")
			if err != nil {
				return err
			}

			query := url.Values{
				"expand[]": {"stats", "custom_fields"},
				"limit":    {"100"},
			}
			if status != "" && status != "all" {
				query.Set("status", status)
			}

			rows, count, err := executor.collectCSVRows(cmd.Context(), "subscriptions", "list", query)
			if err != nil {
				return err
			}
			return writeCSVRows(cmd, rows, filePath, count)
		},
	}
	command.Flags().String("status", "all", "Optional subscription status filter")
	command.Flags().String("file", "", "Write the CSV to this file instead of stdout")
	return command
}

func newReportsExportPostsCommand(options Options) *cobra.Command {
	command := &cobra.Command{
		Use:   "posts",
		Short: "Export posts to CSV",
		Example: strings.TrimSpace(`
beehiiv reports export posts --file posts.csv
beehiiv reports export posts --status confirmed
`),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			executor, _, err := newReportExecutor(cmd, options, config.OutputTable)
			if err != nil {
				return err
			}

			status, err := cmd.Flags().GetString("status")
			if err != nil {
				return err
			}
			filePath, err := cmd.Flags().GetString("file")
			if err != nil {
				return err
			}

			query := url.Values{
				"expand":    {"stats"},
				"limit":     {"100"},
				"order_by":  {"publish_date"},
				"direction": {"desc"},
			}
			if status != "" && status != "all" {
				query.Set("status", status)
			}

			rows, count, err := executor.collectCSVRows(cmd.Context(), "posts", "list", query)
			if err != nil {
				return err
			}
			return writeCSVRows(cmd, rows, filePath, count)
		},
	}
	command.Flags().String("status", "all", "Optional post status filter")
	command.Flags().String("file", "", "Write the CSV to this file instead of stdout")
	return command
}

func newReportsExportEngagementsCommand(options Options) *cobra.Command {
	command := &cobra.Command{
		Use:   "engagements",
		Short: "Export engagement metrics to CSV",
		Example: strings.TrimSpace(`
beehiiv reports export engagements --file engagements.csv
beehiiv reports export engagements --days 30 --granularity week
`),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			executor, _, err := newReportExecutor(cmd, options, config.OutputTable)
			if err != nil {
				return err
			}

			days, err := cmd.Flags().GetInt("days")
			if err != nil {
				return err
			}
			granularity, err := cmd.Flags().GetString("granularity")
			if err != nil {
				return err
			}
			emailType, err := cmd.Flags().GetString("email-type")
			if err != nil {
				return err
			}
			filePath, err := cmd.Flags().GetString("file")
			if err != nil {
				return err
			}
			if days < 1 || days > 31 {
				return errors.New("--days must be between 1 and 31")
			}

			startDate := time.Now().UTC().AddDate(0, 0, -(days - 1)).Format("2006-01-02")
			payload, err := executor.executeJSON(cmd.Context(), "engagements", "list", nil, url.Values{
				"start_date":     {startDate},
				"number_of_days": {strconv.Itoa(days)},
				"granularity":    {granularity},
				"email_type":     {emailType},
				"direction":      {"asc"},
			})
			if err != nil {
				return err
			}

			rows := dataList(payload)
			flatRows := flattenCSVRows(rows)
			return writeCSVRows(cmd, flatRows, filePath, len(flatRows))
		},
	}
	command.Flags().Int("days", 30, "How many recent days of engagement data to export")
	command.Flags().String("granularity", "day", "Engagement granularity: day, week, or month")
	command.Flags().String("email-type", "all", "Filter engagement metrics by email type: all, post, or message")
	command.Flags().String("file", "", "Write the CSV to this file instead of stdout")
	return command
}

func newReportExecutor(cmd *cobra.Command, options Options, defaultOutput string) (*reportExecutor, config.Runtime, error) {
	runtimeConfig, err := reportRuntime(cmd, options.Env, defaultOutput)
	if err != nil {
		return nil, config.Runtime{}, err
	}
	if strings.TrimSpace(runtimeConfig.PublicationID) == "" {
		return nil, config.Runtime{}, errors.New("publication id is required; use `auth login`, `--publication-id`, or BEEHIIV_PUBLICATION_ID")
	}

	return &reportExecutor{
		apiClient: client.New(runtimeConfig, options.HTTPClient, cmd.ErrOrStderr()),
	}, runtimeConfig, nil
}

func reportRuntime(cmd *cobra.Command, env map[string]string, defaultOutput string) (config.Runtime, error) {
	overrides, err := commandOverrides(cmd)
	if err != nil {
		return config.Runtime{}, err
	}
	runtimeConfig, err := config.LoadRuntime(overrides, env)
	if err != nil {
		return config.Runtime{}, err
	}
	if !hasExplicitOutputChoice(cmd) && defaultOutput != "" {
		runtimeConfig.Output = defaultOutput
	}
	return runtimeConfig, nil
}

func hasExplicitOutputChoice(cmd *cobra.Command) bool {
	for _, name := range []string{"output", "table", "raw"} {
		if flag := cmd.Flags().Lookup(name); flag != nil && flag.Changed {
			return true
		}
	}
	return false
}

func usesStructuredOutput(runtimeConfig config.Runtime) bool {
	return runtimeConfig.Output == config.OutputJSON || runtimeConfig.Output == config.OutputRaw
}

func (e *reportExecutor) executeJSON(ctx context.Context, group, action string, pathValues map[string]string, query url.Values) (any, error) {
	operation, err := lookupOperation(group, action)
	if err != nil {
		return nil, err
	}

	response, err := e.apiClient.Execute(ctx, operation, pathValues, query, nil)
	if err != nil {
		return nil, err
	}

	var payload any
	if err := json.Unmarshal(response.Body, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func (e *reportExecutor) collectCSVRows(ctx context.Context, group, action string, query url.Values) ([]map[string]string, int, error) {
	operation, err := lookupOperation(group, action)
	if err != nil {
		return nil, 0, err
	}
	if !operation.List {
		return nil, 0, fmt.Errorf("%s %s is not a list operation", group, action)
	}

	records, _, err := pagination.CollectAll(ctx, operation.Pagination, query, func(callCtx context.Context, nextQuery url.Values) ([]byte, error) {
		response, execErr := e.apiClient.Execute(callCtx, operation, nil, nextQuery, nil)
		if execErr != nil {
			return nil, execErr
		}
		return response.Body, nil
	})
	if err != nil {
		return nil, 0, err
	}

	rows := make([]map[string]any, 0, len(records))
	for _, raw := range records {
		var decoded any
		if err := json.Unmarshal(raw, &decoded); err != nil {
			return nil, 0, err
		}
		row := mapValue(decoded)
		if len(row) == 0 {
			row = map[string]any{"value": decoded}
		}
		rows = append(rows, row)
	}

	return flattenCSVRows(rows), len(rows), nil
}

func lookupOperation(group, action string) (commandset.Operation, error) {
	operation, found, err := commandset.Find(group, action)
	if err != nil {
		return commandset.Operation{}, err
	}
	if !found {
		return commandset.Operation{}, fmt.Errorf("operation %s %s not found", group, action)
	}
	return operation, nil
}

func writeReportSections(output io.Writer, sections []reportSection) error {
	renderedSections := make([]string, 0, len(sections))
	for _, section := range sections {
		if isEmptySection(section.Value) {
			continue
		}
		rendered, err := clioutput.FormatTable(map[string]any{section.Title: normalizeSectionValue(section.Value)})
		if err != nil {
			return err
		}
		renderedSections = append(renderedSections, rendered)
	}
	if len(renderedSections) == 0 {
		_, err := io.WriteString(output, "No report data.\n")
		return err
	}
	_, err := io.WriteString(output, strings.Join(renderedSections, "\n\n")+"\n")
	return err
}

func normalizeSectionValue(value any) any {
	switch typed := value.(type) {
	case []map[string]any:
		normalized := make([]any, 0, len(typed))
		for _, row := range typed {
			normalized = append(normalized, row)
		}
		return normalized
	default:
		return value
	}
}

func isEmptySection(value any) bool {
	switch typed := value.(type) {
	case nil:
		return true
	case []map[string]any:
		return len(typed) == 0
	case []any:
		return len(typed) == 0
	case map[string]any:
		return len(typed) == 0
	default:
		return false
	}
}

func buildPublicationSummary(publication map[string]any) map[string]any {
	return compactMap(map[string]any{
		"id":                           publication["id"],
		"name":                         publication["name"],
		"organization_name":            publication["organization_name"],
		"referral_program_enabled":     publication["referral_program_enabled"],
		"created":                      publication["created"],
		"active_subscriptions":         nestedValue(publication, "stats", "active_subscriptions"),
		"active_free_subscriptions":    nestedValue(publication, "stats", "active_free_subscriptions"),
		"active_premium_subscriptions": nestedValue(publication, "stats", "active_premium_subscriptions"),
		"average_open_rate":            nestedValue(publication, "stats", "average_open_rate"),
		"average_click_rate":           nestedValue(publication, "stats", "average_click_rate"),
		"total_sent":                   nestedValue(publication, "stats", "total_sent"),
		"total_unique_opened":          nestedValue(publication, "stats", "total_unique_opened"),
		"total_clicked":                nestedValue(publication, "stats", "total_clicked"),
	})
}

func buildPostRollup(data map[string]any) map[string]any {
	return compactMap(map[string]any{
		"email_recipients":             nestedValue(data, "stats", "email", "recipients"),
		"email_delivered":              nestedValue(data, "stats", "email", "delivered"),
		"email_opens":                  nestedValue(data, "stats", "email", "opens"),
		"email_unique_opens":           nestedValue(data, "stats", "email", "unique_opens"),
		"email_open_rate":              nestedValue(data, "stats", "email", "open_rate"),
		"email_clicks":                 nestedValue(data, "stats", "email", "clicks"),
		"email_unique_clicks":          nestedValue(data, "stats", "email", "unique_clicks"),
		"email_verified_clicks":        nestedValue(data, "stats", "email", "verified_clicks"),
		"email_unique_verified_clicks": nestedValue(data, "stats", "email", "unique_verified_clicks"),
		"email_click_rate":             nestedValue(data, "stats", "email", "click_rate"),
		"email_unsubscribes":           nestedValue(data, "stats", "email", "unsubscribes"),
		"email_spam_reports":           nestedValue(data, "stats", "email", "spam_reports"),
		"web_views":                    nestedValue(data, "stats", "web", "views"),
		"web_clicks":                   nestedValue(data, "stats", "web", "clicks"),
	})
}

func buildRecentPostSummary(rows []map[string]any) []map[string]any {
	summary := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		summary = append(summary, compactMap(map[string]any{
			"id":                 row["id"],
			"title":              row["title"],
			"status":             row["status"],
			"publish_date":       firstNonNil(row["publish_date"], row["displayed_date"], row["created"]),
			"audience":           row["audience"],
			"platform":           row["platform"],
			"email_open_rate":    nestedValue(row, "stats", "email", "open_rate"),
			"email_click_rate":   nestedValue(row, "stats", "email", "click_rate"),
			"email_unique_opens": nestedValue(row, "stats", "email", "unique_opens"),
			"web_views":          nestedValue(row, "stats", "web", "views"),
			"web_clicks":         nestedValue(row, "stats", "web", "clicks"),
		}))
	}
	return summary
}

func buildEngagementSummary(rows []map[string]any, days int) map[string]any {
	var (
		totalOpens           float64
		uniqueOpens          float64
		totalClicks          float64
		uniqueClicks         float64
		totalVerifiedClicks  float64
		uniqueVerifiedClicks float64
	)

	startDate := ""
	endDate := ""
	for index, row := range rows {
		if index == 0 {
			startDate = stringValue(row["date"])
		}
		endDate = stringValue(row["date"])
		totalOpens += numericValue(row["total_opens"])
		uniqueOpens += numericValue(row["unique_opens"])
		totalClicks += numericValue(row["total_clicks"])
		uniqueClicks += numericValue(row["unique_clicks"])
		totalVerifiedClicks += numericValue(row["total_verified_clicks"])
		uniqueVerifiedClicks += numericValue(row["unique_verified_clicks"])
	}

	return compactMap(map[string]any{
		"days_requested":            days,
		"period_start":              startDate,
		"period_end":                endDate,
		"rows_returned":             len(rows),
		"total_opens":               roundNumber(totalOpens),
		"unique_opens":              roundNumber(uniqueOpens),
		"total_clicks":              roundNumber(totalClicks),
		"unique_clicks":             roundNumber(uniqueClicks),
		"total_verified_clicks":     roundNumber(totalVerifiedClicks),
		"unique_verified_clicks":    roundNumber(uniqueVerifiedClicks),
		"avg_daily_unique_opens":    averageValue(uniqueOpens, len(rows)),
		"avg_daily_unique_clicks":   averageValue(uniqueClicks, len(rows)),
		"avg_daily_verified_clicks": averageValue(uniqueVerifiedClicks, len(rows)),
	})
}

func roundNumber(value float64) any {
	if math.Mod(value, 1) == 0 {
		return int64(value)
	}
	return math.Round(value*100) / 100
}

func averageValue(total float64, count int) any {
	if count == 0 {
		return 0
	}
	return math.Round((total/float64(count))*100) / 100
}

func normalizeEngagementMetric(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, "-", "_")
	switch value {
	case "opens":
		return "total_opens"
	case "clicks":
		return "total_clicks"
	case "verified_clicks":
		return "total_verified_clicks"
	default:
		return value
	}
}

func isSupportedEngagementMetric(value string) bool {
	switch value {
	case "total_opens", "unique_opens", "total_clicks", "unique_clicks", "total_verified_clicks", "unique_verified_clicks":
		return true
	default:
		return false
	}
}

func renderASCIIChart(title string, points []chartPoint, width int) string {
	if len(points) == 0 {
		return title + "\nNo data."
	}

	maxValue := 0.0
	for _, point := range points {
		if point.Value > maxValue {
			maxValue = point.Value
		}
	}
	if maxValue <= 0 {
		maxValue = 1
	}

	var builder strings.Builder
	builder.WriteString(title)
	builder.WriteByte('\n')
	for _, point := range points {
		barWidth := 0
		if point.Value > 0 {
			barWidth = int(math.Round((point.Value / maxValue) * float64(width)))
			if barWidth == 0 {
				barWidth = 1
			}
		}
		builder.WriteString(fmt.Sprintf("%-10s | %-*s %v\n", point.Label, width, strings.Repeat("#", barWidth), roundNumber(point.Value)))
	}
	return strings.TrimRight(builder.String(), "\n")
}

func writeCSVRows(cmd *cobra.Command, rows []map[string]string, filePath string, count int) error {
	headers := collectCSVHeaders(rows)

	var (
		writer io.Writer = cmd.OutOrStdout()
		file   *os.File
		err    error
	)
	if strings.TrimSpace(filePath) != "" {
		if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
			return err
		}
		file, err = os.Create(filePath)
		if err != nil {
			return err
		}
		defer file.Close()
		writer = file
	}

	csvWriter := csv.NewWriter(writer)
	if err := csvWriter.Write(headers); err != nil {
		return err
	}
	for _, row := range rows {
		record := make([]string, 0, len(headers))
		for _, header := range headers {
			record = append(record, row[header])
		}
		if err := csvWriter.Write(record); err != nil {
			return err
		}
	}
	csvWriter.Flush()
	if err := csvWriter.Error(); err != nil {
		return err
	}

	if strings.TrimSpace(filePath) != "" {
		_, err = fmt.Fprintf(cmd.ErrOrStderr(), "Wrote %d rows to %s\n", count, filePath)
		return err
	}
	return nil
}

func collectCSVHeaders(rows []map[string]string) []string {
	seen := make(map[string]struct{})
	headers := make([]string, 0)
	for _, row := range rows {
		for key := range row {
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			headers = append(headers, key)
		}
	}
	sort.Strings(headers)
	for _, preferred := range []string{"id", "email", "title", "date", "status", "name"} {
		moveStringToFront(headers, preferred)
	}
	return headers
}

func moveStringToFront(values []string, target string) {
	index := -1
	for i, value := range values {
		if value == target {
			index = i
			break
		}
	}
	if index <= 0 {
		return
	}
	copy(values[1:index+1], values[0:index])
	values[0] = target
}

func flattenCSVRows(rows []map[string]any) []map[string]string {
	flattened := make([]map[string]string, 0, len(rows))
	for _, row := range rows {
		flat := make(map[string]string)
		flattenInto(flat, "", row)
		flattened = append(flattened, flat)
	}
	return flattened
}

func flattenInto(output map[string]string, prefix string, value any) {
	switch typed := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			nextPrefix := key
			if prefix != "" {
				nextPrefix = prefix + "." + key
			}
			flattenInto(output, nextPrefix, typed[key])
		}
	case []any:
		if prefix == "" {
			prefix = "value"
		}
		if len(typed) == 0 {
			output[prefix] = ""
			return
		}
		if entries, ok := flattenNamedValues(prefix, typed); ok {
			for key, value := range entries {
				output[key] = value
			}
			return
		}
		if allScalar(typed) {
			parts := make([]string, 0, len(typed))
			for _, item := range typed {
				parts = append(parts, scalarString(item))
			}
			output[prefix] = strings.Join(parts, "; ")
			return
		}
		data, _ := json.Marshal(typed)
		output[prefix] = string(data)
	default:
		if prefix == "" {
			prefix = "value"
		}
		output[prefix] = scalarString(typed)
	}
}

func flattenNamedValues(prefix string, values []any) (map[string]string, bool) {
	entries := make(map[string]string)
	for _, value := range values {
		row := mapValue(value)
		if len(row) == 0 {
			return nil, false
		}
		name := stringValue(row["name"])
		if name == "" {
			return nil, false
		}
		valueField, ok := row["value"]
		if !ok {
			return nil, false
		}
		entries[prefix+"."+slugifyHeader(name)] = scalarString(valueField)
	}
	return entries, len(entries) > 0
}

func allScalar(values []any) bool {
	for _, value := range values {
		switch value.(type) {
		case map[string]any, []any:
			return false
		}
	}
	return true
}

func slugifyHeader(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, " ", "_")
	value = strings.ReplaceAll(value, "-", "_")
	var builder strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			builder.WriteRune(r)
		}
	}
	if builder.Len() == 0 {
		return "value"
	}
	return builder.String()
}

func dataMap(payload any) map[string]any {
	root := mapValue(payload)
	return mapValue(root["data"])
}

func dataList(payload any) []map[string]any {
	root := mapValue(payload)
	list, _ := root["data"].([]any)
	rows := make([]map[string]any, 0, len(list))
	for _, item := range list {
		row := mapValue(item)
		if len(row) == 0 {
			row = map[string]any{"value": item}
		}
		rows = append(rows, row)
	}
	return rows
}

func mapValue(value any) map[string]any {
	row, _ := value.(map[string]any)
	return row
}

func nestedValue(value map[string]any, path ...string) any {
	current := any(value)
	for _, segment := range path {
		row, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current, ok = row[segment]
		if !ok {
			return nil
		}
	}
	return current
}

func compactMap(values map[string]any) map[string]any {
	compacted := make(map[string]any, len(values))
	for key, value := range values {
		if isZeroValue(value) {
			continue
		}
		compacted[key] = value
	}
	return compacted
}

func isZeroValue(value any) bool {
	switch typed := value.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(typed) == ""
	case []any:
		return len(typed) == 0
	case []map[string]any:
		return len(typed) == 0
	case map[string]any:
		return len(typed) == 0
	default:
		return false
	}
}

func firstNonNil(values ...any) any {
	for _, value := range values {
		if !isZeroValue(value) {
			return value
		}
	}
	return nil
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	case nil:
		return ""
	default:
		return fmt.Sprint(value)
	}
}

func numericValue(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case int32:
		return float64(typed)
	case json.Number:
		parsed, _ := typed.Float64()
		return parsed
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if err == nil {
			return parsed
		}
	}
	return 0
}

func scalarString(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(typed), 'f', -1, 64)
	case bool:
		return strconv.FormatBool(typed)
	default:
		return fmt.Sprint(value)
	}
}
