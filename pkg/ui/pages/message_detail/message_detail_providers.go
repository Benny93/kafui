package messagedetail

import (
	"encoding/json"
	"strings"

	"github.com/evertras/bubble-table/table"

	"github.com/Benny93/kafui/pkg/ui/components/editor"
	"github.com/Benny93/kafui/pkg/ui/components/tabstrip"
	"github.com/Benny93/kafui/pkg/ui/keys"
	tea "github.com/charmbracelet/bubbletea"
)

// MessageDetailContentProvider implements the ContentProvider interface for message detail view
type MessageDetailContentProvider struct {
	model         *Model
	tabs          []string
	activeTab     int
	keyEditor     *editor.Viewer
	valueEditor   *editor.Viewer
	headersTable  table.Model
	metadataTable table.Model
	// tabStrip is created once so its click zones stay stable across renders.
	tabStrip      *tabstrip.Model
	focusedEditor int // 0 = key, 1 = value (only for Content tab)
	// Last content pushed into each viewer. RenderContent runs every frame, so
	// re-setting identical content would reset the viewer's search and scroll
	// position on every redraw.
	keyContent   string
	valueContent string
	width        int
	height       int
}

// NewMessageDetailContentProvider creates a new content provider for message detail
func NewMessageDetailContentProvider(model *Model) *MessageDetailContentProvider {
	provider := &MessageDetailContentProvider{
		model:         model,
		tabs:          []string{"Content", "Headers", "Metadata"},
		activeTab:     0,
		focusedEditor: 1, // Start with value editor focused
	}
	provider.tabStrip = tabstrip.New("md", provider.tabs)

	// Initialize editors
	provider.keyEditor = editor.NewViewer("")
	provider.valueEditor = editor.NewViewer("")
	provider.headersTable = createHeadersTable()
	provider.metadataTable = createMetadataTable()

	// Set initial content
	provider.updateEditorContent()

	return provider
}

// setViewerContent loads content into a viewer when it differs from what the
// viewer already shows, turning JSON highlighting on only when the content
// actually parses as a JSON object or array.
func setViewerContent(v *editor.Viewer, content string, last *string) {
	if v != nil && content == *last {
		return
	}
	*last = content
	v.SetContent(content)
	v.SetHighlight(looksLikeJSON(content))
}

// looksLikeJSON reports whether content is a JSON object or array. It checks
// the delimiters first so the (expensive) parse is skipped for plain payloads.
func looksLikeJSON(content string) bool {
	trimmed := strings.TrimSpace(content)
	if len(trimmed) < 2 {
		return false
	}
	if (trimmed[0] != '{' || trimmed[len(trimmed)-1] != '}') &&
		(trimmed[0] != '[' || trimmed[len(trimmed)-1] != ']') {
		return false
	}
	return json.Valid([]byte(trimmed))
}

// activeViewer returns the viewer the Content tab currently has focused.
func (m *MessageDetailContentProvider) activeViewer() *editor.Viewer {
	if m.focusedEditor == 0 {
		return m.keyEditor
	}
	return m.valueEditor
}

// updateEditorContent updates the content of all editors
func (m *MessageDetailContentProvider) updateEditorContent() {
	if m.model == nil {
		return
	}

	// Update key and value viewers. JSON highlighting is enabled per pane so
	// a plain-text key next to a JSON value still renders correctly.
	setViewerContent(m.keyEditor, m.model.GetFormattedKey(), &m.keyContent)
	setViewerContent(m.valueEditor, m.model.GetFormattedValue(), &m.valueContent)

	// Update headers table
	m.headersTable = m.headersTable.WithRows(buildHeadersRows(m.model.message.Headers))

	// Update metadata table
	m.metadataTable = m.metadataTable.WithRows(buildMetadataRows(m.model.GetMessageInfo()))
}

// HandleContentUpdate handles content updates
func (m *MessageDetailContentProvider) HandleContentUpdate(msg tea.Msg) tea.Cmd {
	if m.model == nil {
		return nil
	}

	switch msg := msg.(type) {
	case SchemaLoadedMsg:
		m.model.applySchemaLoaded(msg)
		return nil

	case tea.MouseMsg:
		switch msg.Button {
		case tea.MouseButtonWheelUp, tea.MouseButtonWheelDown:
			// Pass scroll events directly to the active editor so the user can
			// scroll through long content with the mouse wheel.
			return m.updateActiveEditor(msg)
		case tea.MouseButtonLeft, tea.MouseButtonNone:
			// Clicking a tab activates it; hovering highlights it.
			if i, ok := m.tabStrip.HandleMouse(msg); ok {
				m.activeTab = i
				return nil
			}
		}

	case tea.KeyMsg:
		// While the viewer's `/` search prompt is open every keystroke belongs
		// to it — otherwise typing "cef" would copy, export and reformat.
		if m.activeTab == 0 && m.activeViewer().Searching() {
			return m.updateActiveEditor(msg)
		}
		action, bound := keys.Default.Resolve(keys.ScopeListContent, msg.String())
		if !bound {
			return m.updateActiveEditor(msg)
		}
		switch action {
		case keys.ActionSelectTab:
			// 1..3 select the tab directly. `shift+tab` used to move to the
			// NEXT tab here, the exact inverse of what it means everywhere else.
			if n := int(msg.String()[0] - '1'); n >= 0 && n < len(m.tabs) {
				m.activeTab = n
			}
			return nil

		case keys.ActionFocusNext:
			// Tab cycles the panes of the Content tab, and the tab strip
			// elsewhere — focus movement, never a data-mode change.
			if m.activeTab == 0 {
				m.focusedEditor = 1 - m.focusedEditor
				m.model.SwitchFocus()
				m.sizeEditors()
			}
			return nil
		case keys.ActionFocusPrev:
			if m.activeTab == 0 {
				m.focusedEditor = 1 - m.focusedEditor
				m.model.SwitchFocus()
				m.sizeEditors()
			}
			return nil

		case keys.ActionFormat:
			if m.activeTab == 0 {
				m.model.ToggleDisplayFormat()
			}
			return nil

		case keys.ActionCopy:
			switch m.activeTab {
			case 0:
				m.model.CopyContentWithFeedback()
			case 1:
				m.copyHeadersAsCSV()
			case 2:
				m.copyMetadataAsCSV()
			}
			return nil

		case keys.ActionRefresh:
			return m.model.ReloadSchemaInfoAsync()

		case keys.ActionUp, keys.ActionDown, keys.ActionPageBack, keys.ActionPageForward,
			keys.ActionFirst, keys.ActionLast, keys.ActionSearch, keys.ActionNextMatch,
			keys.ActionPrevMatch, keys.ActionWrap:
			// Scrolling, search and wrap belong to the focused viewer.
			return m.updateActiveEditor(msg)
		}
	}

	// Update the active editor with the message
	return m.updateActiveEditor(msg)
}

// updateActiveEditor updates the currently active editor with the given message
func (m *MessageDetailContentProvider) updateActiveEditor(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd

	switch m.activeTab {
	case 0: // Content tab
		_, cmd = m.activeViewer().Update(msg)
	case 1: // Headers tab
		m.headersTable, cmd = m.headersTable.Update(msg)
	case 2: // Metadata tab
		m.metadataTable, cmd = m.metadataTable.Update(msg)
	}

	return cmd
}

// InitContent initializes the content provider. The schema load is started by
// OnFocus, which the router runs on every activation including the first.
func (m *MessageDetailContentProvider) InitContent() tea.Cmd {
	return nil
}

func (m *MessageDetailContentProvider) IsInputMode() bool {
	return m.activeTab == 0 && m.activeViewer().Searching()
}

// GetContentSize returns the estimated content size for scrollbar calculation
func (m *MessageDetailContentProvider) GetContentSize(width int) int {
	// Estimate based on message content lines
	if m.model == nil {
		return 10
	}
	// Count lines in key and value content
	keyLines := len(strings.Split(m.model.GetFormattedKey(), "\n"))
	valueLines := len(strings.Split(m.model.GetFormattedValue(), "\n"))
	// Add metadata and header lines
	return keyLines + valueLines + 15
}
