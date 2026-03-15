package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/datasource/kafds"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
)

// DebugModel isolates the table behavior for debugging
type DebugModel struct {
	dataSource    api.KafkaDataSource
	topicName     string
	messages      []api.Message
	table         table.Model
	spinner       spinner.Model
	loading       bool
	err           error
	tableHeight   int
	perPage       int
	currentPage   int
	totalPages    int
	width         int
	height        int
	topics        []string
	showTopics    bool
}

const perPageDefault = 20

func NewDebugModel(topicName string) *DebugModel {
	// Initialize data source with config file
	ds := kafds.NewKafkaDataSourceKaf()
	ds.Init("../../example-config.yaml")

	// Define table columns
	columns := []table.Column{
		table.NewColumn("offset", "Offset", 10),
		table.NewColumn("partition", "Partition", 10),
		table.NewColumn("timestamp", "Timestamp", 20),
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

	return &DebugModel{
		dataSource:  ds,
		topicName:   topicName,
		table:       t,
		spinner:     spinner.New(),
		loading:     true,
		perPage:     perPageDefault,
		currentPage: 0,
		showTopics:  topicName == "",
	}
}

func (m *DebugModel) Init() tea.Cmd {
	if m.showTopics {
		return fetchTopics()
	}
	return fetchMessages(m.topicName)
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
		case "n", "right":
			// Next page
			if m.currentPage < m.totalPages-1 {
				m.currentPage++
				m.updateTableRows()
			}
		case "p", "left":
			// Previous page
			if m.currentPage > 0 {
				m.currentPage--
				m.updateTableRows()
			}
		case "r":
			// Refresh
			m.loading = true
			if m.showTopics {
				return m, fetchTopics()
			}
			return m, fetchMessages(m.topicName)
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
		case "1", "2", "3", "4", "5":
			// Quick topic selection
			if m.showTopics && int(msg.String()[0]-'1') < len(m.topics) {
				idx := int(msg.String()[0] - '1')
				if idx < len(m.topics) {
					m.topicName = m.topics[idx]
					m.showTopics = false
					m.loading = true
					return m, fetchMessages(m.topicName)
				}
			}
		}

	case topicsLoadedMsg:
		m.topics = msg.topics
		m.loading = false

	case messagesLoadedMsg:
		m.messages = msg.messages
		m.sortMessages()
		m.recalculatePages()
		m.currentPage = 0 // Start at first page (newest)
		m.updateTableRows()
		m.loading = false

	case errMsg:
		m.err = msg
		m.loading = false
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m *DebugModel) View() string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).Render("=== TABLE ISOLATION DEBUG ===\n"))

	if m.showTopics {
		sb.WriteString("\nSelect a topic (1-5, or q to quit):\n\n")
		for i, topic := range m.topics {
			sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, topic))
		}
		sb.WriteString("\nPress r to refresh topic list\n")
		return sb.String()
	}

	sb.WriteString(fmt.Sprintf("Topic: %s | Messages: %d | Page: %d/%d | PerPage: %d\n",
		m.topicName, len(m.messages), m.currentPage+1, m.totalPages, m.perPage))
	sb.WriteString(fmt.Sprintf("Terminal: %dx%d | TableHeight: %d\n", m.width, m.height, m.tableHeight))
	sb.WriteString("Controls: n/p=next/prev, r=refresh, +/-=pagesize, q=quit\n")
	sb.WriteString("\n")

	if m.loading {
		sb.WriteString(fmt.Sprintf("%s Loading...\n", m.spinner.View()))
		return sb.String()
	}

	if m.err != nil {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(fmt.Sprintf("Error: %v\n", m.err)))
		return sb.String()
	}

	if len(m.messages) == 0 {
		sb.WriteString("No messages found\n")
		return sb.String()
	}

	// Show debug info about current page bounds
	start, end := m.getPageBounds()
	sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(
		fmt.Sprintf("Showing messages %d-%d (slice indices) of %d total\n", start, end-1, len(m.messages))))
	
	// Show message offsets for debugging
	var offsets []string
	for _, msg := range m.messages {
		offsets = append(offsets, fmt.Sprintf("%d", msg.Offset))
	}
	sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(
		fmt.Sprintf("All offsets: %s\n", strings.Join(offsets, ", "))))
	
	// Show current page offsets
	var pageOffsets []string
	start, end = m.getPageBounds()
	for i := start; i < end && i < len(m.messages); i++ {
		pageOffsets = append(pageOffsets, fmt.Sprintf("%d", m.messages[i].Offset))
	}
	sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Render(
		fmt.Sprintf("Page offsets (before sort): %s\n", strings.Join(pageOffsets, ", "))))
	sb.WriteString("\n")

	// Render the table
	tableView := m.table.View()
	sb.WriteString(tableView)
	sb.WriteString("\n")

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
		minTimestampWidth = 15
		minKeyWidth       = 10
		minValueWidth     = 15
	)

	minTotalWidth := minOffsetWidth + minPartitionWidth + minTimestampWidth + minKeyWidth + minValueWidth
	if availableWidth < minTotalWidth {
		availableWidth = minTotalWidth
	}

	remainingWidth := availableWidth - minTotalWidth

	offsetWidth := minOffsetWidth + remainingWidth*10/100
	partitionWidth := minPartitionWidth + remainingWidth*10/100
	timestampWidth := minTimestampWidth + remainingWidth*20/100
	keyWidth := minKeyWidth + remainingWidth*20/100
	valueWidth := availableWidth - offsetWidth - partitionWidth - timestampWidth - keyWidth

	columns := []table.Column{
		table.NewColumn("offset", "Offset", offsetWidth),
		table.NewColumn("partition", "Partition", partitionWidth),
		table.NewColumn("timestamp", "Timestamp", timestampWidth),
		table.NewColumn("key", "Key", keyWidth),
		table.NewColumn("value", "Value", valueWidth),
	}
	m.table = m.table.WithColumns(columns)
}

func (m *DebugModel) sortMessages() {
	sort.Slice(m.messages, func(i, j int) bool {
		if m.messages[i].Offset != m.messages[j].Offset {
			return m.messages[i].Offset < m.messages[j].Offset
		}
		return m.messages[i].Partition < m.messages[j].Partition
	})
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
	// Newest first: page 0 shows the last PerPage items
	end := len(m.messages) - (m.currentPage * m.perPage)
	start := end - m.perPage
	if start < 0 {
		start = 0
	}
	return start, end
}

func (m *DebugModel) updateTableRows() {
	if len(m.messages) == 0 {
		return
	}

	start, end := m.getPageBounds()
	pageMessages := m.messages[start:end]

	// Sort for display (newest first)
	sortedMessages := make([]api.Message, len(pageMessages))
	copy(sortedMessages, pageMessages)
	sort.Slice(sortedMessages, func(i, j int) bool {
		return sortedMessages[i].Offset > sortedMessages[j].Offset
	})

	rows := make([]table.Row, len(sortedMessages))
	for i, msg := range sortedMessages {
		rows[i] = table.NewRow(table.RowData{
			"offset":    fmt.Sprintf("%d", msg.Offset),
			"partition": fmt.Sprintf("%d", msg.Partition),
			"timestamp": "N/A",
			"key":       truncateString(msg.Key, 18),
			"value":     truncateString(msg.Value, 38),
		})
	}

	m.table = m.table.WithRows(rows).WithHighlightedRow(0)
}

func truncateString(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	if len(s) > maxLen {
		return s[:maxLen-2] + ".."
	}
	return s
}

type messagesLoadedMsg struct {
	messages []api.Message
}

type topicsLoadedMsg struct {
	topics []string
}

type errMsg struct{ err error }

func (e errMsg) Error() string { return e.err.Error() }

func fetchTopics() tea.Cmd {
	return func() tea.Msg {
		ds := kafds.NewKafkaDataSourceKaf()
		ds.Init("../../example-config.yaml")

		topicsMap, err := ds.GetTopics()
		if err != nil {
			return errMsg{fmt.Errorf("failed to get topics: %w", err)}
		}

		topics := make([]string, 0, len(topicsMap))
		for name := range topicsMap {
			topics = append(topics, name)
		}
		sort.Strings(topics)

		return topicsLoadedMsg{topics: topics}
	}
}

func fetchMessages(topicName string) tea.Cmd {
	return func() tea.Msg {
		ds := kafds.NewKafkaDataSourceKaf()
		ds.Init("../../example-config.yaml")

		topics, err := ds.GetTopics()
		if err != nil {
			return errMsg{fmt.Errorf("failed to get topics: %w", err)}
		}

		topic, exists := topics[topicName]
		if !exists {
			return errMsg{fmt.Errorf("topic '%s' not found. Available: %v", topicName, getTopicNames(topics))}
		}

		// Fetch latest 60 messages (3 pages)
		fetchCount := 60
		if topic.MessageCount > 0 && int64(fetchCount) > topic.MessageCount {
			fetchCount = int(topic.MessageCount)
		}

		messages := make([]api.Message, 0, fetchCount)

		flags := api.ConsumeFlags{
			Follow:     false,
			Tail:       int32(fetchCount),
			OffsetFlag: "oldest",
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		done := make(chan struct{})
		go func() {
			defer close(done)
			_ = ds.ConsumeTopic(ctx, topicName, flags, func(msg api.Message) {
				messages = append(messages, msg)
			}, func(e any) {
				log.Printf("consume error: %v", e)
			})
		}()

		<-done

		return messagesLoadedMsg{messages: messages}
	}
}

func getTopicNames(topics map[string]api.Topic) []string {
	names := make([]string, 0, len(topics))
	for name := range topics {
		names = append(names, name)
	}
	return names
}

func main() {
	topicName := ""
	if len(os.Args) > 1 {
		topicName = os.Args[1]
	}

	if topicName != "" {
		fmt.Printf("Starting table isolation debug for topic: %s\n", topicName)
	} else {
		fmt.Println("Starting table isolation debug - select a topic")
	}
	fmt.Println("Press Ctrl+C to quit")

	p := tea.NewProgram(NewDebugModel(topicName))
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
