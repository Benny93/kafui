package shared

import (
	"encoding/csv"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/Benny93/kafui/pkg/api"
)

// CSVFormat configures message CSV output.
type CSVFormat struct {
	Separator      rune   // field separator, default ','
	Quote          rune   // quote char, default '"'
	QuoteAll       bool   // quote every field when true
	LineTerminator string // default "\n"
}

// DefaultCSVFormat returns the default RFC-4180-ish format (comma, double-quote, \n).
func DefaultCSVFormat() CSVFormat {
	return CSVFormat{
		Separator:      ',',
		Quote:          '"',
		QuoteAll:       false,
		LineTerminator: "\n",
	}
}

// MessageCSVHeader is the column header row.
var MessageCSVHeader = []string{"Partition", "Offset", "Timestamp", "Key", "Value", "Headers"}

// WriteMessagesCSV writes msgs as CSV to w using the given format.
// Header row first, then one row per message. Headers column joins each
// "name=value" header with commas (a single field). Timestamp is RFC3339 (empty if zero).
func WriteMessagesCSV(w io.Writer, msgs []api.Message, f CSVFormat) error {
	if f.Separator == 0 {
		f.Separator = ','
	}
	if f.Quote == 0 {
		f.Quote = '"'
	}
	if f.LineTerminator == "" {
		f.LineTerminator = "\n"
	}

	rows := make([][]string, 0, len(msgs)+1)
	rows = append(rows, MessageCSVHeader)
	for _, m := range msgs {
		rows = append(rows, messageRow(m))
	}

	// Fast path: standard double quotes, no forced quoting, and a line
	// terminator encoding/csv can emit ("\n" or "\r\n").
	if !f.QuoteAll && f.Quote == '"' && (f.LineTerminator == "\n" || f.LineTerminator == "\r\n") {
		cw := csv.NewWriter(w)
		cw.Comma = f.Separator
		cw.UseCRLF = f.LineTerminator == "\r\n"
		for _, row := range rows {
			if err := cw.Write(row); err != nil {
				return err
			}
		}
		cw.Flush()
		return cw.Error()
	}

	// Manual path: honor QuoteAll, a custom quote char, or a custom line
	// terminator that encoding/csv cannot produce.
	return writeManualCSV(w, rows, f)
}

// messageRow builds the CSV cells for a single message.
func messageRow(m api.Message) []string {
	ts := ""
	if !m.Timestamp.IsZero() {
		ts = m.Timestamp.Format(time.RFC3339)
	}
	headers := make([]string, 0, len(m.Headers))
	for _, h := range m.Headers {
		headers = append(headers, h.Key+"="+h.Value)
	}
	return []string{
		strconv.FormatInt(int64(m.Partition), 10),
		strconv.FormatInt(m.Offset, 10),
		ts,
		m.Key,
		m.Value,
		strings.Join(headers, ","),
	}
}

// writeManualCSV writes rows honoring a custom quote char, QuoteAll, and an
// arbitrary line terminator. Quotes are escaped by doubling.
func writeManualCSV(w io.Writer, rows [][]string, f CSVFormat) error {
	quote := string(f.Quote)
	sep := string(f.Separator)
	var sb strings.Builder
	for _, row := range rows {
		for i, cell := range row {
			if i > 0 {
				sb.WriteString(sep)
			}
			sb.WriteString(encodeField(cell, f, quote, sep))
		}
		sb.WriteString(f.LineTerminator)
	}
	_, err := io.WriteString(w, sb.String())
	return err
}

// encodeField quotes and escapes a single field per the format rules.
func encodeField(cell string, f CSVFormat, quote, sep string) string {
	needsQuote := f.QuoteAll ||
		strings.Contains(cell, quote) ||
		strings.Contains(cell, sep) ||
		strings.ContainsAny(cell, "\r\n")
	if !needsQuote {
		return cell
	}
	escaped := strings.ReplaceAll(cell, quote, quote+quote)
	return quote + escaped + quote
}
