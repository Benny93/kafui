package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
)

// DebugModel tests table pagination with mock data
type DebugModel struct {
	messages    []Message
	table       table.Model
	spinner     spinner.Model
	loading     bool
	tableHeight int
	perPage     int
	currentPage int
	totalPages  int
	width       int
	height      int
	sortOrder   string // "newest_first" or "oldest_first"
}

type Message struct {
	Offset    int64
	Partition int32
	Key       string
	Value     string
}

const perPageDefault = 20

func generateMockMessages(count int) []Message {
	msgs := make([]Message, count)
	for i := 0; i < count; i++ {
		msgs[i] = Message{
			Offset:    int64(i),
			Partition: int32(i % 3),
			Key:       fmt.Sprintf("key-%d", i),
			Value:     fmt.Sprintf("value-%d with some longer text to test truncation", i),
		}
	}
	return msgs
}

func NewDebugModel() *DebugModel {
	// Define table columns
	columns := []table.Column{
		table.NewColumn("offset", "Offset", 10),
		table.NewColumn("partition", "Partition", 10),
		table.NewColumn("key", "Key", 20),
		table.NewColumn("value", "Value", 40),
	}

	// Initialize table
	t := table.New(columns).
		WithPageSize(perPageDefault).
		WithHighlightedRow(0).
		WithBaseStyle(lipgloss.NewStyle().BorderForeground(lipgloss.Color("240"))).
		HeaderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Bold(true)).
		HighlightStyle(lipgloss.NewStyle().Background(lipgloss.Color("205")).Foreground(lipgloss.Color("0")).Bold(true)).
		Focused(true).
		SortByDesc("offset")

	// Generate 300 mock messages (simulating overflow scenario)
	messages := generateMockMessages(300)

	m := &DebugModel{
		messages:    messages,
		table:       t,
		spinner:     spinner.New(),
		loading:     false,
		perPage:     perPageDefault,
		currentPage: 0,
		sortOrder:   "newest_first",
	}

	// Sort messages ascending (oldest to newest) - this is how kafui stores them
	m.sortMessages()
	m.recalculatePages()
	m.updateTableRows()

	return m
}

func (m *DebugModel) Init() tea.Cmd {
	return nil
}

func (m *DebugModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateTableDimensions(msg.Width, msg.Height)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "n", "right", "l":
			// Next page
			if m.currentPage < m.totalPages-1 {
				m.currentPage++
				m.updateTableRows()
			}
		case "p", "left", "h":
			// Previous page
			if m.currentPage > 0 {
				m.currentPage--
				m.updateTableRows()
			}
		case "g":
			// First page
			m.currentPage = 0
			m.updateTableRows()
		case "G":
			// Last page
			m.currentPage = m.totalPages - 1
			m.updateTableRows()
		case "+":
			// Increase page size
			m.perPage += 5
			m.recalculatePages()
			m.updateTableDimensions(m.width, m.height)
		case "-":
			// Decrease page size
			if m.perPage > 5 {
				m.perPage -= 5
				m.recalculatePages()
				m.updateTableDimensions(m.width, m.height)
			}
		case "s":
			// Toggle sort order
			if m.sortOrder == "newest_first" {
				m.sortOrder = "oldest_first"
			} else {
				m.sortOrder = "newest_first"
			}
			m.currentPage = 0
			m.updateTableRows()
		case "r":
			// Refresh (regenerate messages)
			m.messages = generateMockMessages(300)
			m.sortMessages()
			m.recalculatePages()
			m.currentPage = 0
			m.updateTableRows()
		case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9":
			// Direct page jump (for testing)
			if len(msg.String()) == 1 {
				pageNum := int(msg.String()[0] - '0')
				if pageNum == 0 {
					pageNum = 10
				}
				if pageNum-1 < m.totalPages {
					m.currentPage = pageNum - 1
					m.updateTableRows()
				}
			}
		}
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m *DebugModel) View() string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).Render("=== TABLE PAGINATION DEBUG (MOCK DATA) ===\n"))
	sb.WriteString(fmt.Sprintf("Messages: %d | Page: %d/%d | PerPage: %d | Sort: %s\n",
		len(m.messages), m.currentPage+1, m.totalPages, m.perPage, m.sortOrder))
	sb.WriteString(fmt.Sprintf("Terminal: %dx%d | TableHeight: %d\n", m.width, m.height, m.tableHeight))
	sb.WriteString("Controls: n/g=next, p/h=prev, g=first, G=last, s=sort, +/-=pagesize, r=refresh, q=quit\n")
	sb.WriteString("\n")

	if m.loading {
		sb.WriteString(fmt.Sprintf("%s Loading...\n", m.spinner.View()))
		return sb.String()
	}

	if len(m.messages) == 0 {
		sb.WriteString("No messages found\n")
		return sb.String()
	}

	// Show debug info about current page bounds
	start, end := m.getPageBounds()
	sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(
		fmt.Sprintf("Page slice indices: [%d:%d] of %d total\n", start, end, len(m.messages))))

	// Show current page offsets
	var pageOffsets []string
	for i := start; i < end && i < len(m.messages); i++ {
		pageOffsets = append(pageOffsets, fmt.Sprintf("%d", m.messages[i].Offset))
	}
	sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Render(
		fmt.Sprintf("Page offsets (before display sort): %s\n", strings.Join(pageOffsets, ", "))))

	// Show expected display order
	if m.sortOrder == "newest_first" {
		// Reverse for display
		reversed := make([]string, len(pageOffsets))
		for i, v := range pageOffsets {
			reversed[len(pageOffsets)-1-i] = v
		}
		pageOffsets = reversed
	}
	sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Render(
		fmt.Sprintf("Expected display order: %s\n", strings.Join(pageOffsets, ", "))))
	sb.WriteString("\n")

	// Render the table
	tableView := m.table.View()
	sb.WriteString(tableView)
	sb.WriteString("\n")

	// Show row count
	sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(
		fmt.Sprintf("Table has %d rows rendered (from page slice)\n", end-start)))

	return sb.String()
}

func (m *DebugModel) updateTableDimensions(width, height int) {
	// Calculate available height for table rows
	// Overhead: header lines (~8) + table border/padding
	overhead := 8
	tableHeight := height - overhead
	if tableHeight < 2 {
		tableHeight = 2
	}
	m.tableHeight = tableHeight

	// Update table page size
	m.table = m.table.WithPageSize(tableHeight)

	// Calculate column widths
	availableWidth := width - 6 // Account for borders
	if availableWidth < 60 {
		availableWidth = 60
	}

	const (
		minOffsetWidth    = 8
		minPartitionWidth = 8
		minKeyWidth       = 10
		minValueWidth     = 15
	)

	minTotalWidth := minOffsetWidth + minPartitionWidth + minKeyWidth + minValueWidth
	if availableWidth < minTotalWidth {
		availableWidth = minTotalWidth
	}

	remainingWidth := availableWidth - minTotalWidth

	offsetWidth := minOffsetWidth + remainingWidth*10/100
	partitionWidth := minPartitionWidth + remainingWidth*10/100
	keyWidth := minKeyWidth + remainingWidth*20/100
	valueWidth := availableWidth - offsetWidth - partitionWidth - keyWidth

	columns := []table.Column{
		table.NewColumn("offset", "Offset", offsetWidth),
		table.NewColumn("partition", "Partition", partitionWidth),
		table.NewColumn("key", "Key", keyWidth),
		table.NewColumn("value", "Value", valueWidth),
	}
	m.table = m.table.WithColumns(columns)

	// Update rows with new column widths
	m.updateTableRows()
}

func (m *DebugModel) sortMessages() {
	// Sort ascending (oldest to newest) - this is how kafui stores them internally
	for i := 0; i < len(m.messages)-1; i++ {
		for j := i + 1; j < len(m.messages); j++ {
			if m.messages[i].Offset > m.messages[j].Offset {
				m.messages[i], m.messages[j] = m.messages[j], m.messages[i]
			}
		}
	}
}

func (m *DebugModel) recalculatePages() {
	if len(m.messages) == 0 {
		m.totalPages = 0
		return
	}
	m.totalPages = len(m.messages) / m.perPage
	if len(m.messages)%m.perPage > 0 {
		m.totalPages++
	}
	if m.currentPage >= m.totalPages && m.totalPages > 0 {
		m.currentPage = m.totalPages - 1
	}
}

func (m *DebugModel) getPageBounds() (int, int) {
	if len(m.messages) == 0 {
		return 0, 0
	}

	if m.sortOrder == "newest_first" {
		// Newest first: page 0 shows the last PerPage items
		end := len(m.messages) - (m.currentPage * m.perPage)
		start := end - m.perPage
		if start < 0 {
			start = 0
		}
		return start, end
	}

	// Oldest first: page 0 shows the first PerPage items
	start := m.currentPage * m.perPage
	end := start + m.perPage
	if end > len(m.messages) {
		end = len(m.messages)
	}
	return start, end
}

func (m *DebugModel) updateTableRows() {
	if len(m.messages) == 0 {
		return
	}

	start, end := m.getPageBounds()
	pageMessages := m.messages[start:end]

	// Sort for display based on sort order
	sortedMessages := make([]Message, len(pageMessages))
	copy(sortedMessages, pageMessages)

	if m.sortOrder == "newest_first" {
		// Sort descending for display (newest first)
		for i := 0; i < len(sortedMessages)-1; i++ {
			for j := i + 1; j < len(sortedMessages); j++ {
				if sortedMessages[i].Offset < sortedMessages[j].Offset {
					sortedMessages[i], sortedMessages[j] = sortedMessages[j], sortedMessages[i]
				}
			}
		}
	} else {
		// Sort ascending for display (oldest first)
		for i := 0; i < len(sortedMessages)-1; i++ {
			for j := i + 1; j < len(sortedMessages); j++ {
				if sortedMessages[i].Offset > sortedMessages[j].Offset {
					sortedMessages[i], sortedMessages[j] = sortedMessages[j], sortedMessages[i]
				}
			}
		}
	}

	rows := make([]table.Row, len(sortedMessages))
	for i, msg := range sortedMessages {
		rows[i] = table.NewRow(table.RowData{
			"offset":    fmt.Sprintf("%d", msg.Offset),
			"partition": fmt.Sprintf("%d", msg.Partition),
			"key":       msg.Key,
			"value":     msg.Value,
		})
	}

	// STATELESS: Only pass current page's rows, reset highlight to 0
	m.table = m.table.WithRows(rows).WithHighlightedRow(0)
}

func main() {
	fmt.Println("Starting table pagination debug with 300 mock messages")
	fmt.Println("This tests the pagination logic in isolation")
	fmt.Println("Press Ctrl+C to quit")

	p := tea.NewProgram(NewDebugModel())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
