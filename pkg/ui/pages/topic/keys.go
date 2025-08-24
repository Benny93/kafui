package topic

import (
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/shared"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// Keys handles key bindings for the topic page
type Keys struct {
	bindings keyMap
}

type keyMap struct {
	Search         key.Binding
	Back           key.Binding
	Quit           key.Binding
	Enter          key.Binding
	PauseResume    key.Binding
	Retry          key.Binding
	Navigation     NavigationKeys
	MessageControl MessageControlKeys
}

type NavigationKeys struct {
	Up    key.Binding
	Down  key.Binding
	Home  key.Binding
	End   key.Binding
}

type MessageControlKeys struct {
	Select      key.Binding
	CopyKey     key.Binding
	CopyValue   key.Binding
	ShowDetails key.Binding
}

// NewKeys creates a new Keys instance
func NewKeys() *Keys {
	return &Keys{
		bindings: keyMap{
			Search: key.NewBinding(
				key.WithKeys("/"),
				key.WithHelp("/", "search messages"),
			),
			Back: key.NewBinding(
				key.WithKeys("esc"),
				key.WithHelp("esc", "back/exit search"),
			),
			Quit: key.NewBinding(
				key.WithKeys("ctrl+c", "q"),
				key.WithHelp("ctrl+c/q", "quit"),
			),
			Enter: key.NewBinding(
				key.WithKeys("enter"),
				key.WithHelp("enter", "view message details"),
			),
			PauseResume: key.NewBinding(
				key.WithKeys(" "),
				key.WithHelp("space", "pause/resume consumption"),
			),
			Retry: key.NewBinding(
				key.WithKeys("r"),
				key.WithHelp("r", "retry connection"),
			),
			Navigation: NavigationKeys{
				Up: key.NewBinding(
					key.WithKeys("k", "up"),
					key.WithHelp("k/↑", "up"),
				),
				Down: key.NewBinding(
					key.WithKeys("j", "down"),
					key.WithHelp("j/↓", "down"),
				),
				Home: key.NewBinding(
					key.WithKeys("g", "home"),
					key.WithHelp("g/home", "top"),
				),
				End: key.NewBinding(
					key.WithKeys("G", "end"),
					key.WithHelp("G/end", "bottom"),
				),
			},
			MessageControl: MessageControlKeys{
				Select: key.NewBinding(
					key.WithKeys("enter"),
					key.WithHelp("enter", "select message"),
				),
				CopyKey: key.NewBinding(
					key.WithKeys("c"),
					key.WithHelp("c", "copy message key"),
				),
				CopyValue: key.NewBinding(
					key.WithKeys("v"),
					key.WithHelp("v", "copy message value"),
				),
				ShowDetails: key.NewBinding(
					key.WithKeys("d"),
					key.WithHelp("d", "show message details"),
				),
			},
		},
	}
}

// HandleKey processes key events
func (k *Keys) HandleKey(model *Model, msg tea.KeyMsg) tea.Cmd {
	var cmds []tea.Cmd

	// Log key event details
	shared.DebugLog("Topic Key Event - Type: %v, String: %s, SearchMode: %v", msg.Type, msg.String(), model.searchMode)

	// If in search mode, let the search input handle keys
	// But handle Enter and Esc specially for search confirmation/cancellation
	if model.searchMode {
		// Handle Enter to confirm search
		if msg.String() == "enter" {
			model.searchMode = false
			model.searchInput.Blur()
			model.FilterMessages()
			return nil
		}
		
		// Handle Esc to cancel search
		if msg.String() == "esc" {
			model.searchMode = false
			model.searchInput.Blur()
			model.searchInput.SetValue("")
			model.FilterMessages()
			return nil
		}
		
		// Let the search input handle other keys
		var cmd tea.Cmd
		model.searchInput, cmd = model.searchInput.Update(msg)
		cmds = append(cmds, cmd)
		model.FilterMessages()
		return tea.Batch(cmds...)
	}

	// Handle ESC key (back navigation when not in search mode)
	if key.Matches(msg, k.bindings.Back) {
		return k.handleBack(model)
	}

	// Handle other specific key combinations (only when not in search mode)
	switch {
	case key.Matches(msg, k.bindings.Quit):
		return k.handleQuit(model)
	case key.Matches(msg, k.bindings.Search):
		return k.handleSearch(model)
	case key.Matches(msg, k.bindings.PauseResume):
		return k.handlePauseResume(model)
	case key.Matches(msg, k.bindings.Retry):
		return k.handleRetry(model)
	case key.Matches(msg, k.bindings.Enter):
		return k.handleEnter(model)
	}

	// Handle message control keys (only when not in search mode)
	switch {
	case key.Matches(msg, k.bindings.MessageControl.CopyKey):
		return k.handleCopyKey(model)
	case key.Matches(msg, k.bindings.MessageControl.CopyValue):
		return k.handleCopyValue(model)
	case key.Matches(msg, k.bindings.MessageControl.ShowDetails):
		return k.handleShowDetails(model)
	}

	// Handle navigation keys
	switch {
	case key.Matches(msg, k.bindings.Navigation.Up):
		return k.handleNavigation(model, "up")
	case key.Matches(msg, k.bindings.Navigation.Down):
		return k.handleNavigation(model, "down")
	case key.Matches(msg, k.bindings.Navigation.Home):
		return k.handleNavigation(model, "home")
	case key.Matches(msg, k.bindings.Navigation.End):
		return k.handleNavigation(model, "end")
	}

	// Default table navigation handling
	var cmd tea.Cmd
	model.messageTable, cmd = model.messageTable.Update(msg)
	cmds = append(cmds, cmd)

	return tea.Batch(cmds...)
}

func (k *Keys) handleBack(model *Model) tea.Cmd {
	if model.searchMode {
		// Exit search mode
		model.searchMode = false
		model.searchInput.Blur()
		model.FilterMessages()
		return nil
	}
	
	// Return to main page - cancel consumption first
	if model.cancelConsumption != nil {
		model.cancelConsumption()
	}
	
	return func() tea.Msg {
		return core.PageChangeMsg{PageID: "main"}
	}
}

func (k *Keys) handleQuit(model *Model) tea.Cmd {
	// Cancel consumption before quitting
	if model.cancelConsumption != nil {
		model.cancelConsumption()
	}
	return tea.Quit
}

func (k *Keys) handleSearch(model *Model) tea.Cmd {
	// Enter search mode
	model.searchMode = true
	model.searchInput.Focus()
	return nil
}

func (k *Keys) handlePauseResume(model *Model) tea.Cmd {
	// Toggle pause/resume consumption
	model.TogglePause()
	return nil
}

func (k *Keys) handleRetry(model *Model) tea.Cmd {
	// Manual retry connection
	if model.consumption != nil {
		return model.consumption.RetryConnection()
	}
	return nil
}

func (k *Keys) handleEnter(model *Model) tea.Cmd {
	if model.searchMode {
		// In search mode, just apply the search
		model.FilterMessages()
		return nil
	}
	
	// Navigate to message detail page
	if selectedMsg := model.GetSelectedMessage(); selectedMsg != nil {
		model.selectedMessage = selectedMsg
		return func() tea.Msg {
			return core.PageChangeMsg{PageID: "detail", Data: *selectedMsg}
		}
	}
	
	return nil
}

func (k *Keys) handleCopyKey(model *Model) tea.Cmd {
	// Copy message key to clipboard (placeholder)
	if selectedMsg := model.GetSelectedMessage(); selectedMsg != nil && selectedMsg.Key != "" {
		model.statusMessage = "Message key copied to clipboard"
		// TODO: Implement actual clipboard copy
	}
	return nil
}

func (k *Keys) handleCopyValue(model *Model) tea.Cmd {
	// Copy message value to clipboard (placeholder)
	if selectedMsg := model.GetSelectedMessage(); selectedMsg != nil && selectedMsg.Value != "" {
		model.statusMessage = "Message value copied to clipboard"
		// TODO: Implement actual clipboard copy
	}
	return nil
}

func (k *Keys) handleShowDetails(model *Model) tea.Cmd {
	// Show detailed message information
	if selectedMsg := model.GetSelectedMessage(); selectedMsg != nil {
		return func() tea.Msg {
			return MessageSelectedMsg{Message: *selectedMsg}
		}
	}
	return nil
}

func (k *Keys) handleNavigation(model *Model, direction string) tea.Cmd {
	switch direction {
	case "up":
		model.messageTable.MoveUp(1)
	case "down":
		model.messageTable.MoveDown(1)
	case "home":
		model.messageTable.GotoTop()
	case "end":
		model.messageTable.GotoBottom()
	}
	return nil
}

// GetKeyBindings returns the key bindings for help display
func (k *Keys) GetKeyBindings() []key.Binding {
	return []key.Binding{
		k.bindings.Search,
		k.bindings.Back,
		k.bindings.Quit,
		k.bindings.Enter,
		k.bindings.PauseResume,
		k.bindings.Retry,
		k.bindings.Navigation.Up,
		k.bindings.Navigation.Down,
		k.bindings.Navigation.Home,
		k.bindings.Navigation.End,
		k.bindings.MessageControl.CopyKey,
		k.bindings.MessageControl.CopyValue,
		k.bindings.MessageControl.ShowDetails,
	}
}

// GetShortcuts returns formatted shortcut descriptions
func (k *Keys) GetShortcuts() []string {
	return []string{
		"↑/↓   Navigate messages",
		"Enter View details",
		"Space Pause/resume",
		"/     Search messages",
		"r     Retry connection",
		"c     Copy key",
		"v     Copy value", 
		"d     Show details",
		"Esc   Exit search",
		"q/Esc Back to topics",
	}
}

