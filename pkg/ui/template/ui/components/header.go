package components

import (
	"fmt"
	"strings"
	"time"
	"ui_example/ui/providers"
	"ui_example/ui/styles"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Header interface {
	Component
	Sizeable
}

type header struct {
	width, height int
	provider      providers.HeaderDataProvider
}

func NewHeader() Header {
	return &header{
		provider: providers.NewDefaultHeaderDataProvider(),
	}
}

func NewHeaderWithProvider(provider providers.HeaderDataProvider) Header {
	return &header{
		provider: provider,
	}
}

func (h *header) Init() tea.Cmd {
	if h.provider != nil {
		return h.provider.InitHeader()
	}
	return nil
}

func (h *header) Update(msg tea.Msg) (Component, tea.Cmd) {
	var cmd tea.Cmd
	
	// Let the provider handle the message
	if h.provider != nil {
		cmd = h.provider.HandleHeaderUpdate(msg)
	}
	
	return h, cmd
}

func (h *header) View() string {
	if h.width == 0 {
		return ""
	}

	const (
		gap          = " "
		diag         = "╱"
		minDiags     = 3
		leftPadding  = 1
		rightPadding = 1
	)

	t := styles.CurrentTheme()
	var b strings.Builder

	// Brand section
	var brandName, appName string
	var status map[string]interface{}
	
	if h.provider != nil {
		brandName = h.provider.GetBrandName()
		appName = h.provider.GetAppName()
		status = h.provider.GetStatusData()
	} else {
		brandName = "Example™"
		appName = "CRUSH UI"
		status = make(map[string]interface{})
	}
	
	b.WriteString(t.S().Base.Foreground(t.Secondary).Render(brandName))
	b.WriteString(gap)
	b.WriteString(styles.ApplyBoldForegroundGrad(appName, t.Secondary, t.Primary))
	b.WriteString(gap)
	
	// Add debug info to status if needed
	debugInfo := styles.DebugInfo("Header", h.width, h.height)
	if debugInfo != "" {
		status["debug"] = debugInfo
	}
	
	availDetailWidth := h.width - leftPadding - rightPadding - lipgloss.Width(b.String()) - minDiags
	details := h.renderDetails(status, availDetailWidth)

	// Calculate remaining width for diagonal fill
	remainingWidth := h.width -
		lipgloss.Width(b.String()) -
		lipgloss.Width(details) -
		leftPadding -
		rightPadding

	if remainingWidth > 0 {
		b.WriteString(t.S().Base.Foreground(t.Primary).Render(
			strings.Repeat(diag, max(minDiags, remainingWidth)),
		))
		b.WriteString(gap)
	}

	b.WriteString(details)

	return t.S().Base.Padding(0, rightPadding, 0, leftPadding).Render(b.String())
}


func (h *header) renderDetails(status map[string]interface{}, availWidth int) string {
	t := styles.CurrentTheme()
	var parts []string

	// Add status indicators
	if connections, ok := status["connections"].(int); ok {
		parts = append(parts, t.S().Success.Render(fmt.Sprintf("●%d", connections)))
	}

	if memory, ok := status["memory"].(string); ok {
		parts = append(parts, t.S().Muted.Render(memory))
	}

	if timeStr, ok := status["time"].(string); ok {
		parts = append(parts, t.S().Subtle.Render(timeStr))
	}

	dot := t.S().Subtle.Render(" • ")
	metadata := strings.Join(parts, dot)
	if len(parts) > 0 {
		metadata = dot + metadata
	}

	// Truncate if necessary
	if lipgloss.Width(metadata) > availWidth {
		metadata = metadata[:max(0, availWidth-3)] + "..."
	}

	return metadata
}

func (h *header) SetSize(width, height int) tea.Cmd {
	h.width = width
	h.height = height
	return nil
}

func (h *header) GetSize() (int, int) {
	return h.width, h.height
}

type tickMsg time.Time

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}