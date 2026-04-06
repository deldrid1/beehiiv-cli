package output

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/deldrid1/beehiiv-cli/internal/config"
)

func Write(out io.Writer, value any, rawBody []byte, runtime config.Runtime) error {
	switch runtime.Output {
	case config.OutputRaw:
		if len(rawBody) > 0 {
			writeBytes(out, rawBody)
			return nil
		}
		return writeJSON(out, value, true)
	case config.OutputTable:
		return writeTable(out, value)
	default:
		return writeJSON(out, value, runtime.Compact)
	}
}

func writeJSON(out io.Writer, value any, compact bool) error {
	var (
		data []byte
		err  error
	)
	if compact {
		data, err = json.Marshal(value)
	} else {
		data, err = json.MarshalIndent(value, "", "  ")
	}
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(out, string(data))
	return err
}

func writeBytes(output io.Writer, data []byte) {
	io.WriteString(output, string(data))
	if len(data) == 0 || data[len(data)-1] != '\n' {
		io.WriteString(output, "\n")
	}
}

func writeTable(output io.Writer, value any) error {
	rendered, err := FormatTable(value)
	if err != nil {
		return err
	}
	io.WriteString(output, rendered)
	if !strings.HasSuffix(rendered, "\n") {
		io.WriteString(output, "\n")
	}
	return nil
}

func FormatTable(value any) (string, error) {
	switch typed := value.(type) {
	case map[string]any:
		return formatMapSections(typed), nil
	case []any:
		return formatArraySection("", typed), nil
	default:
		return renderScalarSection("", typed), nil
	}
}

func formatMapSections(values map[string]any) string {
	if len(values) == 0 {
		return "No rows."
	}

	if len(values) == 1 {
		if data, ok := values["data"]; ok {
			return formatSection("data", data)
		}
	}

	sections := make([]string, 0, len(values))
	seenComposite := false
	for _, key := range sortedAnyKeys(values) {
		switch values[key].(type) {
		case []any, map[string]any:
			seenComposite = true
			sections = append(sections, formatSection(key, values[key]))
		}
	}

	if !seenComposite {
		return renderMapTable("", values)
	}

	for _, key := range sortedAnyKeys(values) {
		switch values[key].(type) {
		case []any, map[string]any:
			continue
		default:
			sections = append(sections, renderScalarSection(key, values[key]))
		}
	}

	return strings.Join(filterEmpty(sections), "\n\n")
}

func formatSection(title string, value any) string {
	switch typed := value.(type) {
	case []any:
		return formatArraySection(title, typed)
	case map[string]any:
		return renderMapTable(title, typed)
	default:
		return renderScalarSection(title, typed)
	}
}

func formatArraySection(title string, values []any) string {
	rows, ok := normalizeRows(values)
	if !ok {
		return renderScalarSliceTable(title, values)
	}
	return renderRowsTable(title, rows)
}

func normalizeRows(values []any) ([]map[string]any, bool) {
	rows := make([]map[string]any, 0, len(values))
	for _, value := range values {
		row, ok := value.(map[string]any)
		if !ok {
			return nil, false
		}
		rows = append(rows, row)
	}
	return rows, true
}

func renderScalarSliceTable(title string, values []any) string {
	if len(values) == 0 {
		return renderNoRows(title)
	}

	rows := make([]map[string]any, 0, len(values))
	for _, value := range values {
		rows = append(rows, map[string]any{"value": value})
	}
	return renderRowsTable(title, rows)
}

func renderMapTable(title string, values map[string]any) string {
	keys := sortedAnyKeys(values)
	rows := make([][]string, 0, len(keys))
	for _, key := range keys {
		rows = append(rows, []string{key, formatCell(values[key])})
	}
	return renderTableGrid(title, []string{"field", "value"}, rows)
}

func renderScalarSection(title string, value any) string {
	return renderTableGrid(title, []string{"field", "value"}, [][]string{{firstNonEmpty(title, "value"), formatCell(value)}})
}

func renderRowsTable(title string, rows []map[string]any) string {
	if len(rows) == 0 {
		return renderNoRows(title)
	}

	headers := collectHeaders(rows)
	dataRows := make([][]string, 0, len(rows))
	for _, row := range rows {
		cells := make([]string, 0, len(headers))
		for _, header := range headers {
			cells = append(cells, formatCell(row[header]))
		}
		dataRows = append(dataRows, cells)
	}
	return renderTableGrid(title, headers, dataRows)
}

func renderNoRows(title string) string {
	if title == "" {
		return "No rows."
	}
	return strings.ToUpper(title) + "\nNo rows."
}

func collectHeaders(rows []map[string]any) []string {
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
	moveToFront(headers, "id")
	moveToFront(headers, "name")
	return headers
}

func moveToFront(values []string, target string) {
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

func renderTableGrid(title string, headers []string, rows [][]string) string {
	widths := make([]int, len(headers))
	for index, header := range headers {
		widths[index] = len(header)
	}
	for _, row := range rows {
		for index, cell := range row {
			if len(cell) > widths[index] {
				widths[index] = len(cell)
			}
		}
	}

	var builder strings.Builder
	if title != "" {
		builder.WriteString(strings.ToUpper(title))
		builder.WriteString("\n")
	}

	border := renderBorder(widths)
	builder.WriteString(border)
	builder.WriteByte('\n')
	builder.WriteString(renderRow(headers, widths))
	builder.WriteByte('\n')
	builder.WriteString(border)
	if len(rows) > 0 {
		builder.WriteByte('\n')
		for index, row := range rows {
			builder.WriteString(renderRow(row, widths))
			builder.WriteByte('\n')
			if index == len(rows)-1 {
				builder.WriteString(border)
			}
		}
		return builder.String()
	}
	builder.WriteByte('\n')
	builder.WriteString(border)
	return builder.String()
}

func renderBorder(widths []int) string {
	var builder strings.Builder
	builder.WriteByte('+')
	for _, width := range widths {
		builder.WriteString(strings.Repeat("-", width+2))
		builder.WriteByte('+')
	}
	return builder.String()
}

func renderRow(cells []string, widths []int) string {
	var builder strings.Builder
	builder.WriteByte('|')
	for index, cell := range cells {
		builder.WriteByte(' ')
		builder.WriteString(cell)
		builder.WriteString(strings.Repeat(" ", widths[index]-len(cell)))
		builder.WriteByte(' ')
		builder.WriteByte('|')
	}
	return builder.String()
}

func sortedAnyKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func filterEmpty(values []string) []string {
	filtered := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			filtered = append(filtered, value)
		}
	}
	return filtered
}

func formatCell(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return collapseWhitespace(typed)
	case json.Number:
		return typed.String()
	case bool:
		return fmt.Sprint(typed)
	case float64, float32, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprint(typed)
	default:
		data, err := json.Marshal(typed)
		if err != nil {
			return collapseWhitespace(fmt.Sprint(typed))
		}
		return collapseWhitespace(string(data))
	}
}

func collapseWhitespace(value string) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(value, "\n", " ")), " ")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
