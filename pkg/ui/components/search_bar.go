package components

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	searchBarStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)

	searchIconStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			PaddingRight(1)

	searchInputStyle = lipgloss.NewStyle()

	resultCountStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")).
				PaddingLeft(1)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))
)

// SearchMode represents different search modes
type SearchMode int

const (
	SimpleSearch SearchMode = iota
	AdvancedSearch
	RegexSearch
)

// SearchBarModel represents the search bar component
type SearchBarModel struct {
	textInput     textinput.Model
	searchHistory []string
	historyIndex  int
	searchMode    SearchMode
	placeholder   string
	resultCount   int
	errorMessage  string
	width         int
	focused       bool
	onSearch      func(query string) tea.Msg
	onClear       func() tea.Msg
}

// SearchBarOption is a function that configures a SearchBarModel
type SearchBarOption func(*SearchBarModel)

// WithPlaceholder sets the placeholder text
func WithPlaceholder(placeholder string) SearchBarOption {
	return func(sb *SearchBarModel) {
		sb.placeholder = placeholder
	}
}

// WithSearchMode sets the search mode
func WithSearchMode(mode SearchMode) SearchBarOption {
	return func(sb *SearchBarModel) {
		sb.searchMode = mode
	}
}

// WithOnSearch sets the callback function for search
func WithOnSearch(fn func(query string) tea.Msg) SearchBarOption {
	return func(sb *SearchBarModel) {
		sb.onSearch = fn
	}
}

// WithOnClear sets the callback function for clear
func WithOnClear(fn func() tea.Msg) SearchBarOption {
	return func(sb *SearchBarModel) {
		sb.onClear = fn
	}
}

// NewSearchBar creates a new search bar component
func NewSearchBar(options ...SearchBarOption) SearchBarModel {
	ti := textinput.New()
	ti.Placeholder = "Search..."
	ti.CharLimit = 156
	ti.Width = 30

	sb := SearchBarModel{
		textInput:     ti,
		searchHistory: []string{},
		historyIndex:  -1,
		searchMode:    SimpleSearch,
		placeholder:   "Search...",
		resultCount:   0,
		width:         0,
		focused:       false,
	}

	// Apply options
	for _, opt := range options {
		opt(&sb)
	}

	// Set placeholder if provided
	if sb.placeholder != "" {
		sb.textInput.Placeholder = sb.placeholder
	}

	return sb
}

// Init initializes the search bar
func (sb SearchBarModel) Init() tea.Cmd {
	return nil
}

// Focus focuses the search bar
func (sb *SearchBarModel) Focus() tea.Cmd {
	sb.focused = true
	return sb.textInput.Focus()
}

// Blur removes focus from the search bar
func (sb *SearchBarModel) Blur() {
	sb.focused = false
	sb.textInput.Blur()
}

// Focused returns whether the search bar is focused
func (sb SearchBarModel) Focused() bool {
	return sb.focused
}

// SetValue sets the value of the search bar
func (sb *SearchBarModel) SetValue(value string) {
	sb.textInput.SetValue(value)
}

// Value returns the current value of the search bar
func (sb SearchBarModel) Value() string {
	return sb.textInput.Value()
}

// SetResultCount sets the result count
func (sb *SearchBarModel) SetResultCount(count int) {
	sb.resultCount = count
}

// SetError sets an error message
func (sb *SearchBarModel) SetError(err string) {
	sb.errorMessage = err
}

// ClearError clears the error message
func (sb *SearchBarModel) ClearError() {
	sb.errorMessage = ""
}

// SetWidth sets the width of the search bar
func (sb *SearchBarModel) SetWidth(width int) {
	sb.width = width
	sb.textInput.Width = width - 10 // Account for padding and icon
}

// Update handles messages for the search bar
func (sb SearchBarModel) Update(msg tea.Msg) (SearchBarModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if sb.textInput.Value() != "" {
				// Add to history
				sb.searchHistory = append(sb.searchHistory, sb.textInput.Value())
				sb.historyIndex = -1 // Reset history index

				// Trigger search callback if provided
				if sb.onSearch != nil {
					return sb, func() tea.Msg { return sb.onSearch(sb.textInput.Value()) }
				}
			}
			return sb, nil

		case "esc":
			// Clear search or exit
			sb.textInput.SetValue("")
			sb.resultCount = 0
			sb.ClearError()

			// Trigger clear callback if provided
			if sb.onClear != nil {
				return sb, func() tea.Msg { return sb.onClear() }
			}
			return sb, nil

		case "up":
			// Navigate search history
			if len(sb.searchHistory) > 0 {
				if sb.historyIndex == -1 {
					sb.historyIndex = len(sb.searchHistory) - 1
				} else if sb.historyIndex > 0 {
					sb.historyIndex--
				}
				sb.textInput.SetValue(sb.searchHistory[sb.historyIndex])
			}
			return sb, nil

		case "down":
			// Navigate search history
			if len(sb.searchHistory) > 0 {
				if sb.historyIndex == -1 {
					// Do nothing
				} else if sb.historyIndex < len(sb.searchHistory)-1 {
					sb.historyIndex++
					sb.textInput.SetValue(sb.searchHistory[sb.historyIndex])
				} else {
					sb.historyIndex = -1
					sb.textInput.SetValue("")
				}
			}
			return sb, nil

		case "tab":
			// TODO: Implement completion
			return sb, nil
		}
	}

	// Handle text input updates
	var cmd tea.Cmd
	sb.textInput, cmd = sb.textInput.Update(msg)
	cmds = append(cmds, cmd)

	return sb, tea.Batch(cmds...)
}

// View renders the search bar
func (sb SearchBarModel) View() string {
	// Build the search bar components
	searchIcon := searchIconStyle.Render("🔍")
	input := searchInputStyle.Render(sb.textInput.View())

	// Result count display
	var resultCount string
	if sb.resultCount > 0 {
		resultCount = resultCountStyle.Render(fmt.Sprintf("%d results", sb.resultCount))
	}

	// Error display
	var errorText string
	if sb.errorMessage != "" {
		errorText = errorStyle.Render(sb.errorMessage)
	}

	// Combine components
	components := []string{searchIcon, input}
	if resultCount != "" {
		components = append(components, resultCount)
	}
	if errorText != "" {
		components = append(components, errorText)
	}

	searchBar := lipgloss.JoinHorizontal(lipgloss.Left, components...)

	return searchBarStyle.Render(searchBar)
}

// SearchMsg is a message type for search events
type SearchMsg struct {
	Query string
	Mode  SearchMode
}

// ClearMsg is a message type for clear events
type ClearMsg struct{}