package components

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	modalStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Background(lipgloss.Color("235")).
			Padding(1, 2).
			Width(50).
			Align(lipgloss.Center)

	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true).
			MarginBottom(1)

	buttonStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Background(lipgloss.Color("235")).
			Padding(0, 2).
			Margin(0, 1)

	activeButtonStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("255")).
				Background(lipgloss.Color("205")).
				Padding(0, 2).
				Margin(0, 1)

	inputStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)

	modalErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)
)

// ModalType represents different types of modals
type ModalType int

const (
	AlertModal ModalType = iota
	ConfirmModal
	InputModal
	ErrorModal
)

// ModalSize represents different sizes for modals
type ModalSize int

const (
	Small ModalSize = iota
	Medium
	Large
)

// Modal represents a modal dialog
type Modal struct {
	Type        ModalType
	Title       string
	Message     string
	Buttons     []string
	ActiveBtn   int
	InputValue  string
	ShowInput   bool
	visible     bool
	Size        ModalSize
	OnConfirm   func() tea.Msg
	OnCancel    func() tea.Msg
	OnInput     func(string) tea.Msg
	Width       int
	Height      int
}

// ModalOption is a function that configures a Modal
type ModalOption func(*Modal)

// WithTitle sets the modal title
func WithTitle(title string) ModalOption {
	return func(m *Modal) {
		m.Title = title
	}
}

// WithMessage sets the modal message
func WithMessage(message string) ModalOption {
	return func(m *Modal) {
		m.Message = message
	}
}

// WithButtons sets the modal buttons
func WithButtons(buttons []string) ModalOption {
	return func(m *Modal) {
		m.Buttons = buttons
	}
}

// WithType sets the modal type
func WithType(modalType ModalType) ModalOption {
	return func(m *Modal) {
		m.Type = modalType
	}
}

// WithSize sets the modal size
func WithSize(size ModalSize) ModalOption {
	return func(m *Modal) {
		m.Size = size
	}
}

// WithOnConfirm sets the confirm callback
func WithOnConfirm(fn func() tea.Msg) ModalOption {
	return func(m *Modal) {
		m.OnConfirm = fn
	}
}

// WithOnCancel sets the cancel callback
func WithOnCancel(fn func() tea.Msg) ModalOption {
	return func(m *Modal) {
		m.OnCancel = fn
	}
}

// WithOnInput sets the input callback
func WithOnInput(fn func(string) tea.Msg) ModalOption {
	return func(m *Modal) {
		m.OnInput = fn
	}
}

// WithInputValue sets the initial input value
func WithInputValue(value string) ModalOption {
	return func(m *Modal) {
		m.InputValue = value
	}
}

// NewModal creates a new modal with the given options
func NewModal(options ...ModalOption) *Modal {
	m := &Modal{
		Type:      AlertModal,
		Title:     "Modal",
		Message:   "",
		Buttons:   []string{"OK"},
		ActiveBtn: 0,
		ShowInput: false,
		visible:   false,
		Size:      Medium,
		Width:     50,
		Height:    10,
	}

	// Apply options
	for _, opt := range options {
		opt(m)
	}

	// Set size based on modal type
	switch m.Type {
	case AlertModal:
		if m.Size == Medium {
			m.Width = 40
			m.Height = 8
		}
	case ConfirmModal:
		if m.Size == Medium {
			m.Width = 50
			m.Height = 10
		}
	case InputModal:
		m.ShowInput = true
		if m.Size == Medium {
			m.Width = 50
			m.Height = 12
		}
	case ErrorModal:
		if m.Size == Medium {
			m.Width = 50
			m.Height = 10
		}
	}

	// Set default buttons based on modal type
	if len(m.Buttons) == 0 {
		switch m.Type {
		case ConfirmModal:
			m.Buttons = []string{"Cancel", "Confirm"}
		case InputModal:
			m.Buttons = []string{"Cancel", "Submit"}
		case ErrorModal:
			m.Buttons = []string{"OK"}
		default:
			m.Buttons = []string{"OK"}
		}
	}

	// Set modal style based on type
	switch m.Type {
	case ErrorModal:
		modalStyle = modalStyle.BorderForeground(lipgloss.Color("196"))
	default:
		modalStyle = modalStyle.BorderForeground(lipgloss.Color("62"))
	}

	return m
}

// Show displays the modal
func (m *Modal) Show() {
	m.visible = true
}

// Hide hides the modal
func (m *Modal) Hide() {
	m.visible = false
}

// IsVisible returns whether the modal is visible
func (m *Modal) IsVisible() bool {
	return m.visible
}

// SetSize sets the modal size
func (m *Modal) SetSize(width, height int) {
	m.Width = width
	m.Height = height
	modalStyle = modalStyle.Width(width).Height(height)
}

// Update handles messages for the modal
func (m *Modal) Update(msg tea.Msg) (*Modal, tea.Cmd) {
	if !m.visible {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left", "h":
			if len(m.Buttons) > 1 {
				m.ActiveBtn = (m.ActiveBtn - 1 + len(m.Buttons)) % len(m.Buttons)
			}
		case "right", "l":
			if len(m.Buttons) > 1 {
				m.ActiveBtn = (m.ActiveBtn + 1) % len(m.Buttons)
			}
		case "enter":
			return m.handleButtonPress()
		case "esc":
			return m.handleEscape()
		case "backspace":
			if m.ShowInput {
				if len(m.InputValue) > 0 {
					m.InputValue = m.InputValue[:len(m.InputValue)-1]
				}
			}
		default:
			if m.ShowInput {
				// Handle text input
				if msg.Type == tea.KeyRunes {
					m.InputValue += msg.String()
				}
			}
		}
	}

	return m, nil
}

// handleButtonPress handles button press events
func (m *Modal) handleButtonPress() (*Modal, tea.Cmd) {
	if len(m.Buttons) == 0 {
		return m, nil
	}

	button := m.Buttons[m.ActiveBtn]
	
	switch button {
	case "OK":
		m.visible = false
		if m.OnConfirm != nil {
			return m, m.OnConfirm
		}
	case "Confirm":
		m.visible = false
		if m.OnConfirm != nil {
			return m, m.OnConfirm
		}
	case "Submit":
		m.visible = false
		if m.OnInput != nil {
			return m, func() tea.Msg { return m.OnInput(m.InputValue) }
		}
	case "Cancel":
		m.visible = false
		if m.OnCancel != nil {
			return m, m.OnCancel
		}
	default:
		// Custom button handling
		m.visible = false
		if m.OnConfirm != nil {
			return m, m.OnConfirm
		}
	}

	return m, nil
}

// handleEscape handles escape key events
func (m *Modal) handleEscape() (*Modal, tea.Cmd) {
	m.visible = false
	if m.OnCancel != nil {
		return m, m.OnCancel
	}
	return m, nil
}

// View renders the modal
func (m *Modal) View() string {
	if !m.visible {
		return ""
	}

	// Build modal content
	var content []string

	// Add title
	if m.Title != "" {
		content = append(content, titleStyle.Render(m.Title))
	}

	// Add message
	if m.Message != "" {
		content = append(content, m.Message)
	}

	// Add input field if needed
	if m.ShowInput {
		input := inputStyle.Render(m.InputValue)
		content = append(content, input)
	}

	// Add buttons
	if len(m.Buttons) > 0 {
		var buttons []string
		for i, btn := range m.Buttons {
			if i == m.ActiveBtn {
				buttons = append(buttons, activeButtonStyle.Render(btn))
			} else {
				buttons = append(buttons, buttonStyle.Render(btn))
			}
		}
		content = append(content, lipgloss.JoinHorizontal(lipgloss.Center, buttons...))
	}

	// Join content
	modalContent := lipgloss.JoinVertical(lipgloss.Center, content...)

	// Apply modal style
	styledModal := modalStyle.Render(modalContent)

	// Center the modal on screen (this would typically be handled by the parent view)
	return styledModal
}

// ModalManager manages multiple modals in a stack
type ModalManager struct {
	Modals []*Modal
}

// NewModalManager creates a new modal manager
func NewModalManager() *ModalManager {
	return &ModalManager{
		Modals: make([]*Modal, 0),
	}
}

// Push adds a modal to the stack
func (mm *ModalManager) Push(modal *Modal) {
	mm.Modals = append(mm.Modals, modal)
}

// Pop removes the top modal from the stack
func (mm *ModalManager) Pop() *Modal {
	if len(mm.Modals) == 0 {
		return nil
	}
	
	modal := mm.Modals[len(mm.Modals)-1]
	mm.Modals = mm.Modals[:len(mm.Modals)-1]
	return modal
}

// Top returns the top modal without removing it
func (mm *ModalManager) Top() *Modal {
	if len(mm.Modals) == 0 {
		return nil
	}
	return mm.Modals[len(mm.Modals)-1]
}

// IsEmpty returns whether the modal stack is empty
func (mm *ModalManager) IsEmpty() bool {
	return len(mm.Modals) == 0
}

// Show displays the top modal
func (mm *ModalManager) Show() {
	if top := mm.Top(); top != nil {
		top.Show()
	}
}

// Hide hides the top modal
func (mm *ModalManager) Hide() {
	if top := mm.Top(); top != nil {
		top.Hide()
	}
}

// Update handles messages for the modal manager
func (mm *ModalManager) Update(msg tea.Msg) (*ModalManager, tea.Cmd) {
	if mm.IsEmpty() {
		return mm, nil
	}

	top := mm.Top()
	if top == nil || !top.IsVisible() {
		return mm, nil
	}

	updatedModal, cmd := top.Update(msg)
	if updatedModal != top {
		// Replace the top modal with the updated one
		mm.Modals[len(mm.Modals)-1] = updatedModal
	}
	
	return mm, cmd
}

// View renders the top modal
func (mm *ModalManager) View() string {
	if mm.IsEmpty() {
		return ""
	}

	top := mm.Top()
	if top == nil || !top.IsVisible() {
		return ""
	}

	return top.View()
}

// Convenience functions for creating common modals

// NewAlertModal creates a new alert modal
func NewAlertModal(title, message string, onConfirm func() tea.Msg) *Modal {
	return NewModal(
		WithTitle(title),
		WithMessage(message),
		WithType(AlertModal),
		WithButtons([]string{"OK"}),
		WithOnConfirm(onConfirm),
	)
}

// NewConfirmModal creates a new confirmation modal
func NewConfirmModal(title, message string, onConfirm, onCancel func() tea.Msg) *Modal {
	return NewModal(
		WithTitle(title),
		WithMessage(message),
		WithType(ConfirmModal),
		WithButtons([]string{"Cancel", "Confirm"}),
		WithOnConfirm(onConfirm),
		WithOnCancel(onCancel),
	)
}

// NewInputModal creates a new input modal
func NewInputModal(title, message, initialValue string, onInput func(string) tea.Msg) *Modal {
	return NewModal(
		WithTitle(title),
		WithMessage(message),
		WithType(InputModal),
		WithButtons([]string{"Cancel", "Submit"}),
		WithInputValue(initialValue),
		WithOnInput(onInput),
	)
}

// NewErrorModal creates a new error modal
func NewErrorModal(title, message string, onConfirm func() tea.Msg) *Modal {
	return NewModal(
		WithTitle(title),
		WithMessage(modalErrorStyle.Render(message)),
		WithType(ErrorModal),
		WithButtons([]string{"OK"}),
		WithOnConfirm(onConfirm),
	)
}