package messagedetail

import (
	"sort"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/evertras/bubble-table/table"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/shared"
	stylesPkg "github.com/Benny93/kafui/pkg/ui/styles"
	"github.com/charmbracelet/lipgloss"
)

const (
	colMdName  = "md_name"
	colMdValue = "md_value"
	colHdrKey  = "hdr_key"
	colHdrVal  = "hdr_val"
)

// createHeadersTable returns a freshly initialised bubble-table for the
// Headers tab. Column widths are adjusted dynamically in sizeEditors().
func createHeadersTable() table.Model {
	return table.New([]table.Column{
		table.NewColumn(colHdrKey, "Key", 30),
		table.NewColumn(colHdrVal, "Value", 50),
	}).
		WithPageSize(30).
		Focused(true).
		WithBaseStyle(
			lipgloss.NewStyle().BorderForeground(stylesPkg.FgSubtle),
		).
		HeaderStyle(
			lipgloss.NewStyle().Foreground(stylesPkg.FgMuted).Bold(true),
		).
		HighlightStyle(
			lipgloss.NewStyle().
				Background(stylesPkg.Primary).
				Foreground(stylesPkg.BgBase).
				Bold(true),
		)
}

// buildHeadersRows converts a message's headers slice to a slice of table rows.
func buildHeadersRows(headers []api.MessageHeader) []table.Row {
	rows := make([]table.Row, 0, len(headers))
	for _, h := range headers {
		rows = append(rows, table.NewRow(table.RowData{
			colHdrKey: h.Key,
			colHdrVal: h.Value,
		}))
	}
	return rows
}

// copyHeadersAsCSV serialises the headers table as RFC-4180 CSV and writes to clipboard.
func (m *MessageDetailContentProvider) copyHeadersAsCSV() {
	var buf strings.Builder
	buf.WriteString("Key,Value\n")
	for _, h := range m.model.message.Headers {
		k, v := h.Key, h.Value
		if strings.ContainsAny(k, ",\"\n") {
			k = `"` + strings.ReplaceAll(k, `"`, `""`) + `"`
		}
		if strings.ContainsAny(v, ",\"\n") {
			v = `"` + strings.ReplaceAll(v, `"`, `""`) + `"`
		}
		buf.WriteString(k + "," + v + "\n")
	}
	if err := clipboard.WriteAll(buf.String()); err != nil {
		m.model.statusMsg = "Failed to copy: " + err.Error()
		m.model.statusTime = time.Now()
		return
	}
	m.model.statusMsg = "Headers copied as CSV"
	m.model.statusTime = time.Now()
}

// createMetadataTable returns a freshly initialised bubble-table for the
// Metadata tab. Column widths are adjusted dynamically in sizeEditors().
func createMetadataTable() table.Model {
	return table.New([]table.Column{
		table.NewColumn(colMdName, "Name", 20),
		table.NewColumn(colMdValue, "Value", 40),
	}).
		WithPageSize(30).
		Focused(true).
		WithBaseStyle(
			lipgloss.NewStyle().BorderForeground(stylesPkg.FgSubtle),
		).
		HeaderStyle(
			lipgloss.NewStyle().Foreground(stylesPkg.FgMuted).Bold(true),
		).
		HighlightStyle(
			lipgloss.NewStyle().
				Background(stylesPkg.Primary).
				Foreground(stylesPkg.BgBase).
				Bold(true),
		)
}

// buildMetadataRows converts the GetMessageInfo map to a sorted slice of
// table rows so the display order is stable regardless of map iteration.
func buildMetadataRows(info map[string]string) []table.Row {
	keys := make([]string, 0, len(info))
	for k := range info {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	rows := make([]table.Row, 0, len(info))
	for _, k := range keys {
		rows = append(rows, table.NewRow(table.RowData{
			colMdName:  k,
			colMdValue: info[k],
		}))
	}
	return rows
}

// copyMetadataAsCSV serialises the current metadata rows as RFC-4180 CSV
// (header line + one row per field, sorted by name) and writes to clipboard.
func (m *MessageDetailContentProvider) copyMetadataAsCSV() {
	info := m.model.GetMessageInfo()
	keys := make([]string, 0, len(info))
	for k := range info {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var buf strings.Builder
	buf.WriteString("Name,Value\n")
	for _, k := range keys {
		v := info[k]
		// Wrap fields that contain commas, quotes, or newlines in double-quotes.
		if strings.ContainsAny(v, ",\"\n") {
			v = `"` + strings.ReplaceAll(v, `"`, `""`) + `"`
		}
		buf.WriteString(k + "," + v + "\n")
	}

	if err := clipboard.WriteAll(buf.String()); err != nil {
		m.model.statusMsg = "Failed to copy: " + err.Error()
		m.model.statusTime = time.Now()
		return
	}
	m.model.statusMsg = "Metadata copied as CSV"
	m.model.statusTime = time.Now()
}

// exportMessageToFile writes the message to a JSON file next to the working
// directory and reports the path in the status line (MSG-29).
func (m *MessageDetailContentProvider) exportMessageToFile() {
	path := shared.DefaultExportPath(m.model.topicName, m.model.message)
	if err := shared.ExportMessageJSON(path, m.model.topicName, m.model.message); err != nil {
		m.model.statusMsg = "Export failed: " + err.Error()
	} else {
		m.model.statusMsg = "Saved message to " + path
	}
	m.model.statusTime = time.Now()
}
