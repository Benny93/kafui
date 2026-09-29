package topic

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/shared"
	stylesPkg "github.com/Benny93/kafui/pkg/ui/styles"
	"github.com/Benny93/kafui/pkg/ui/template/ui/styles"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
)

// updateTableDimensions updates the table dimensions and column widths based on available space
func (m *Model) updateTableDimensions(width, height int) {
	// Only update if dimensions have changed
	if m.lastTableWidth == width && m.lastTableHeight == height {
		return
	}
	m.lastTableWidth = width
	m.lastTableHeight = height

	m.markRenderDirty()

	// height is the inner content area height (border + padding already excluded by
	// the content component before calling RenderContent).
	// We just need to reserve lines for the table's own chrome: header(1) + separator(1) + footer(1) = 3.
	reservedLines := 3
	if m.searchMode {
		reservedLines += 4 // Search bar lines (prompt + help + spacing)
	}

	tableHeight := height - reservedLines

	// Enforce minimum
	if tableHeight < 2 {
		tableHeight = 2
	}

	shared.Log.Info("table dimensions", "topic", m.topicName,
		"contentW", width, "contentH", height, "tableHeight", tableHeight)

	// The page size is not set here: it must match the rows renderTableCustom
	// draws, which RenderContent derives from the content height (syncPageSize).

	// Calculate column widths based on available width
	// Account for table border (2) and separators (4 for 5 columns) = 6 chars total overhead
	availableWidth := width - 6
	if availableWidth < 60 {
		availableWidth = 60 // Minimum width for all columns
	}

	// Define minimum column widths
	const (
		minOffsetWidth    = 8
		minPartitionWidth = 8
		minTimestampWidth = 15
		minKeyWidth       = 10
		minValueWidth     = 15
	)

	// Calculate total minimum width required
	minTotalWidth := minOffsetWidth + minPartitionWidth + minTimestampWidth + minKeyWidth + minValueWidth

	// Ensure available width is at least the minimum total
	if availableWidth < minTotalWidth {
		availableWidth = minTotalWidth
	}

	// Calculate remaining width after allocating minimums
	remainingWidth := availableWidth - minTotalWidth

	// Distribute remaining width proportionally (10:10:20:20:40 = 1:1:2:2:4)
	// Total ratio = 10
	offsetWidth := minOffsetWidth + remainingWidth*10/100
	partitionWidth := minPartitionWidth + remainingWidth*10/100
	timestampWidth := minTimestampWidth + remainingWidth*20/100
	keyWidth := minKeyWidth + remainingWidth*20/100
	// Value gets the remainder to ensure exact fit
	valueWidth := availableWidth - offsetWidth - partitionWidth - timestampWidth - keyWidth

	// Ensure value column gets at least its minimum
	if valueWidth < minValueWidth {
		valueWidth = minValueWidth
	}

	// Update column widths
	columns := []table.Column{
		table.NewColumn("offset", "Offset", offsetWidth),
		table.NewColumn("partition", "Partition", partitionWidth),
		table.NewColumn("timestamp", "Timestamp", timestampWidth),
		table.NewColumn("key", "Key", keyWidth),
		table.NewColumn("value", "Value", valueWidth),
	}
	m.messageTable = m.messageTable.WithColumns(columns)
	m.tableColumns = columns

	// Rebuild table rows for the new page size / column widths.
	m.updateMessageTable()
	m.rowStringsDirty = true // column widths changed — force row string rebuild
}

// updateMessageTable updates the table with paginated messages
// updateMessageTable updates pagination state and preserves/resets the
// highlighted row. Row content is rendered on demand by renderTableCustom —
// no table.Row objects are built here.
func (m *Model) updateMessageTable() {
	visibleCount := len(m.pagination.GetVisibleMessages(m.filteredMessages))

	if m.pendingReset || m.cursorRow >= visibleCount {
		m.cursorRow = 0
	}
	m.pendingReset = false
	m.rowStringsDirty = true // visible rows changed — rebuild on next render
}

// searchBarLines is what the open search bar adds above the table: its
// prompt, its help line and a blank line.
const searchBarLines = 3

// tableRowBudget is the number of message rows renderTableCustom draws in a
// content area height lines tall. The pagination page size is set from the
// same number (see syncPageSize), so every message on a page is drawn.
func (m *Model) tableRowBudget(height int) int {
	rows := height - tableFirstRowLine - 1 // title, rules, column headers; footer
	if m.searchMode {
		rows -= searchBarLines
	}
	return max(rows, 5)
}

// syncPageSize sets the page size to the rows the table draws at height and
// re-clamps the cursor when it changes.
func (m *Model) syncPageSize(height int) {
	rows := m.tableRowBudget(height)
	if m.pagination.PerPage == rows {
		return
	}
	m.pagination.SetPerPage(rows)
	m.messageTable = m.messageTable.WithPageSize(rows)
	m.updateMessageTable()
}

// renderTableCustom renders the message table as plain text with direct string
// building. Row strings are cached and only rebuilt when data/layout changes;
// on cursor-only moves the cached rows are reused with just the highlight
// reapplied — making scrolling essentially free.
func (m *Model) renderTableCustom(width, height int) string {
	messages := m.pagination.GetVisibleMessages(m.filteredMessages)
	if len(messages) == 0 {
		// Empty state distinguishes an empty topic from a filter with no matches (MSG-27).
		if len(m.messages) > 0 {
			return lipgloss.NewStyle().Foreground(stylesPkg.FgMuted).Padding(1).
				Render("No messages match the current filter.")
		}
		return lipgloss.NewStyle().Foreground(stylesPkg.FgMuted).Padding(1).
			Render("No messages found.")
	}

	// Apply display sort order (storage is always ascending; newest_first reverses it).
	sortedMessages := make([]api.Message, len(messages))
	copy(sortedMessages, messages)
	if m.pagination.SortOrder == "newest_first" {
		// Reverse rather than re-sort, so the displayed order is exactly the
		// mapping GetSelectedMessage uses (several partitions share offsets).
		for i, j := 0, len(sortedMessages)-1; i < j; i, j = i+1, j-1 {
			sortedMessages[i], sortedMessages[j] = sortedMessages[j], sortedMessages[i]
		}
	}
	messages = sortedMessages

	// Column width calculation: a row is a leading space, four separators and
	// the five columns, so the columns share width-5 and a row is exactly width.
	availableWidth := width - 5
	if availableWidth < 60 {
		availableWidth = 60
	}
	const (
		minOffsetWidth    = 10
		minPartitionWidth = 9 // the "Partition" header
		minTimeWidth      = 19
		minKeyWidth       = 18
		minValueWidth     = 15
	)
	minTotalWidth := minOffsetWidth + minPartitionWidth + minTimeWidth + minKeyWidth + minValueWidth
	if availableWidth < minTotalWidth {
		availableWidth = minTotalWidth
	}
	remainingWidth := availableWidth - minTotalWidth
	offsetWidth := minOffsetWidth + remainingWidth*10/100
	partitionWidth := minPartitionWidth
	timeWidth := minTimeWidth
	keyWidth := minKeyWidth + remainingWidth*30/100
	valueWidth := availableWidth - offsetWidth - partitionWidth - timeWidth - keyWidth
	if valueWidth < minValueWidth {
		valueWidth = minValueWidth
	}

	// Limit to available screen rows
	availableRows := m.tableRowBudget(height)
	if len(messages) > availableRows {
		messages = messages[:availableRows]
	}

	// Rebuild unstyled row strings only when content changed or width changed.
	if m.rowStringsDirty || m.rowStringCacheWidth != width || len(m.rowStringCache) != len(messages) {
		rowFmt := fmt.Sprintf(" %%-%ds %%-%ds %%-%ds %%-%ds %%-%ds", offsetWidth, partitionWidth, timeWidth, keyWidth, valueWidth)
		m.rowStringCache = make([]string, len(messages))
		for i, msg := range messages {
			ts := ""
			if !msg.Timestamp.IsZero() {
				ts = shared.FormatTimestamp(msg.Timestamp)
			}
			m.rowStringCache[i] = fmt.Sprintf(rowFmt,
				fmt.Sprintf("%d", msg.Offset),
				fmt.Sprintf("%d", msg.Partition),
				truncateString(ts, timeWidth),
				truncateString(m.displayKey(msg), keyWidth),
				truncateString(m.displayValue(msg), valueWidth),
			)
		}
		m.rowStringCacheWidth = width
		m.rowStringsDirty = false
	}

	var sb strings.Builder

	// Header
	header := fmt.Sprintf(" %s | Page %d/%d | %d msgs",
		m.topicName, m.pagination.Page+1, m.pagination.TotalPages, m.pagination.TotalMessages)
	sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(stylesPkg.Primary).Render(header))
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("─", width))
	sb.WriteString("\n")

	// Column headers
	colHeaderFmt := fmt.Sprintf(" %%-%ds %%-%ds %%-%ds %%-%ds %%-%ds", offsetWidth, partitionWidth, timeWidth, keyWidth, valueWidth)
	sb.WriteString(lipgloss.NewStyle().Bold(true).Render(
		fmt.Sprintf(colHeaderFmt, "Offset", "Partition", "Timestamp", "Key", "Value"),
	))
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("─", width))
	sb.WriteString("\n")

	// Rows — highlight only the cursor row, everything else is plain text.
	// With a row expanded, its panel takes rows from the budget, so a window of
	// rows around the cursor is drawn (the page itself keeps its size).
	shown, panel := len(m.rowStringCache), ""
	if m.expanded && m.cursorRow < len(messages) {
		m.loadExpanded(messages[m.cursorRow])
		if inner := m.panelInnerHeight(width, availableRows); inner > 0 {
			panel = m.renderExpandPanel(messages[m.cursorRow], width)
			shown = min(shown, availableRows-inner-panelChrome)
		}
	}
	m.rowWindow = windowStart(m.rowWindow, m.cursorRow, shown, len(m.rowStringCache))
	m.tableLayout = tableLayout{firstRow: tableFirstRowLine, start: m.rowWindow, shown: shown, panelAfter: -1}
	if panel != "" {
		m.tableLayout.panelAfter = m.cursorRow
		m.tableLayout.panelLines = lipgloss.Height(panel)
	}

	highlightStyle := lipgloss.NewStyle().Background(stylesPkg.Primary).Foreground(stylesPkg.BgBase)
	for i := m.rowWindow; i < m.rowWindow+shown; i++ {
		if i == m.cursorRow {
			sb.WriteString(highlightStyle.Render(m.rowStringCache[i]))
		} else {
			sb.WriteString(m.rowStringCache[i])
		}
		sb.WriteString("\n")
		if i == m.cursorRow && panel != "" {
			sb.WriteString(panel)
			sb.WriteString("\n")
		}
	}

	// Footer with browse statistics (MSG-27): message/byte counts, elapsed, filter errors.
	stats := fmt.Sprintf(" %d msgs • %s • %dms",
		m.browseStats.MessagesConsumed, shared.FormatBytes2dp(m.browseStats.BytesConsumed), m.browseStats.ElapsedMs)
	if m.smartFilterErrs > 0 {
		stats += fmt.Sprintf(" • %d filter errors", m.smartFilterErrs)
	}
	var footer string
	if m.pagination.TotalPages > 1 {
		footer = fmt.Sprintf(" [←/→] Page %d/%d |%s | [x] expand [S] seek [#] parts [P] produce",
			m.pagination.Page+1, m.pagination.TotalPages, stats)
	} else {
		footer = fmt.Sprintf("%s | [x] expand [S] seek [#] parts [P] produce [/] search", stats)
	}
	sb.WriteString(lipgloss.NewStyle().Foreground(stylesPkg.FgMuted).Render(footer))

	return sb.String()
}

// tableFirstRowLine is the line of the first data row: title, rule, column
// headers and rule come first.
const tableFirstRowLine = 4

// windowStart returns the first row to draw so that cursor stays within the
// shown rows, moving the previous window as little as possible.
func windowStart(prev, cursor, shown, total int) int {
	if shown <= 0 || shown >= total {
		return 0
	}
	start := prev
	if cursor < start {
		start = cursor
	}
	if cursor >= start+shown {
		start = cursor - shown + 1
	}
	return max(0, min(start, total-shown))
}

// getCachedWidth returns cached column width or calculates and caches it
func (m *Model) getCachedWidth(columnTitle string, columns []table.Column) int {
	// Find the width for this column
	var width int
	for _, col := range columns {
		if col.Title() == columnTitle {
			width = col.Width()
			break
		}
	}

	// Try read lock first for better performance
	m.widthCacheMu.RLock()
	if m.widthCache[columnTitle] != nil {
		if cached, ok := m.widthCache[columnTitle][width]; ok {
			m.widthCacheMu.RUnlock()
			return cached
		}
	}
	m.widthCacheMu.RUnlock()

	// Need to write - acquire write lock
	m.widthCacheMu.Lock()
	defer m.widthCacheMu.Unlock()

	// Initialize cache for this column if needed
	if m.widthCache[columnTitle] == nil {
		m.widthCache[columnTitle] = make(map[int]int)
	}

	// Calculate and cache
	maxLen := width - 2 // Account for padding
	m.widthCache[columnTitle][width] = maxLen
	return maxLen
}

// initWidthCache initializes the width cache
func (m *Model) initWidthCache() {
	if m.widthCache == nil {
		m.widthCache = make(map[string]map[int]int)
	}
}

// highlightMatchingText highlights matching parts of text with color
func highlightMatchingText(text, query string) string {
	if query == "" {
		return text
	}

	// Convert to lowercase for case-insensitive comparison
	lowerText := strings.ToLower(text)
	lowerQuery := strings.ToLower(query)

	// Find all occurrences of the query in the text
	var result strings.Builder
	start := 0

	for {
		index := strings.Index(lowerText[start:], lowerQuery)
		if index == -1 {
			// No more matches, add the rest of the text
			result.WriteString(text[start:])
			break
		}

		// Add text before the match
		actualIndex := start + index
		result.WriteString(text[start:actualIndex])

		// Add highlighted match (using Lip Gloss styling)
		match := text[actualIndex : actualIndex+len(query)]
		highlighted := lipgloss.NewStyle().Foreground(stylesPkg.Primary).Render(match)
		result.WriteString(highlighted)

		// Move start position
		start = actualIndex + len(query)
	}

	return result.String()
}

func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// sanitizeForDisplay removes control characters from raw Kafka message content.
// Binary payloads (Avro, Protobuf, etc.) can contain bytes like ESC (0x1b),
// VT (0x0b), cursor-up (0x1b 0x5b 0x41) and similar terminal control sequences
// that cause the TUI to drift or break when rendered. Tabs are replaced with a
// single space; all other C0/C1 control characters are dropped.
func sanitizeForDisplay(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r == '\t' {
			b.WriteByte(' ')
		} else if unicode.IsControl(r) {
			// Drop: C0 (0x00-0x1f) and C1 (0x7f-0x9f) control characters.
			// This includes \n, \r, ESC (0x1b), VT (0x0b), FF (0x0c), etc.
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// truncateString sanitizes and truncates a string to fit within maxLen visual
// characters.  It handles ANSI escape codes, multi-byte characters, and strips
// all terminal control bytes that could corrupt the table layout.
func truncateString(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	s = sanitizeForDisplay(s)
	return styles.TruncateWithEllipsis(s, maxLen)
}
