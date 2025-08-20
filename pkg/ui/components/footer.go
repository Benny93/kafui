package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
)

// FooterConfig holds configuration for the footer
type FooterConfig struct {
	Width         int
	SearchMode    bool
	SelectedItem  string
	TotalItems    int
	StatusMessage string
	LastUpdate    time.Time
	Spinner       spinner.Model
}

// Footer represents a reusable footer component
type Footer struct {
	config FooterConfig
}

// NewFooter creates a new footer component
func NewFooter(config FooterConfig) *Footer {
	return &Footer{config: config}
}

// RenderSearchModeFooter renders footer for search mode
func (f *Footer) RenderSearchModeFooter() string {
	return "Type to search  Enter: confirm  Esc: cancel"
}

// RenderNormalFooter renders footer for normal mode
func (f *Footer) RenderNormalFooter() string {
	// Left side: Selection information
	selected := "None"
	if f.config.SelectedItem != "" {
		selected = f.config.SelectedItem
	}
	leftInfo := fmt.Sprintf("Selected: %s  •  %d items total", selected, f.config.TotalItems)
	
	// Right side: Status information
	spinnerView := ""
	if f.config.Spinner.View() != "" {
		spinnerView = f.config.Spinner.View() + " "
	}
	
	rightInfo := fmt.Sprintf("%s%s  •  Last update: %s",
		spinnerView,
		f.config.StatusMessage,
		f.config.LastUpdate.Format("15:04:05"),
	)
	
	// Calculate available width for each side
	totalWidth := f.config.Width - 4 // Account for padding
	leftWidth := len(leftInfo)
	rightWidth := len(rightInfo)
	
	// If both fit, use them with proper spacing
	if leftWidth+rightWidth+3 <= totalWidth {
		spacer := strings.Repeat(" ", totalWidth-leftWidth-rightWidth)
		return leftInfo + spacer + rightInfo
	}
	
	// If they don't fit, truncate the left side
	maxLeftWidth := totalWidth - rightWidth - 3
	if maxLeftWidth > 20 {
		if len(leftInfo) > maxLeftWidth {
			leftInfo = leftInfo[:maxLeftWidth-3] + "..."
		}
		spacer := strings.Repeat(" ", totalWidth-len(leftInfo)-rightWidth)
		return leftInfo + spacer + rightInfo
	}
	
	// Fallback: just show the right info if space is very limited
	return rightInfo
}

// Render renders the footer based on current mode
func (f *Footer) Render() string {
	if f.config.SearchMode {
		return f.RenderSearchModeFooter()
	}
	return f.RenderNormalFooter()
}

// UpdateConfig updates the footer configuration
func (f *Footer) UpdateConfig(config FooterConfig) {
	f.config = config
}

// GetConfig returns the current footer configuration
func (f *Footer) GetConfig() FooterConfig {
	return f.config
}

// SetSearchMode updates the search mode
func (f *Footer) SetSearchMode(searchMode bool) {
	f.config.SearchMode = searchMode
}

// SetSelectedItem updates the selected item
func (f *Footer) SetSelectedItem(item string) {
	f.config.SelectedItem = item
}

// SetTotalItems updates the total items count
func (f *Footer) SetTotalItems(count int) {
	f.config.TotalItems = count
}

// SetStatusMessage updates the status message
func (f *Footer) SetStatusMessage(message string) {
	f.config.StatusMessage = message
}

// SetLastUpdate updates the last update time
func (f *Footer) SetLastUpdate(lastUpdate time.Time) {
	f.config.LastUpdate = lastUpdate
}

// SetSpinner updates the spinner
func (f *Footer) SetSpinner(spinner spinner.Model) {
	f.config.Spinner = spinner
}