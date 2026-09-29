package topic

import (
	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/core"
	templateui "github.com/Benny93/kafui/pkg/ui/template/ui"
	"github.com/Benny93/kafui/pkg/ui/template/ui/providers"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// View handles rendering for the topic page (minimal implementation for compatibility)
type View struct {
	dimensions core.Dimensions
}

// NewView creates a new View instance
func NewView() *View {
	return &View{}
}

// Render renders the topic page view (minimal implementation)
func (v *View) Render(model *Model) string {
	// For the template-based topic page, this is not used
	// The rendering is handled by the template providers
	return "Topic page (template-based rendering)"
}

// SetDimensions updates the view dimensions
func (v *View) SetDimensions(width, height int) {
	v.dimensions = core.Dimensions{Width: width, Height: height}
}

// TopicPageModel wraps the ReusableApp with topic-specific providers
type TopicPageModel struct {
	// Shared context
	common *core.Common

	// Original topic model for business logic
	topicModel *Model

	// Template system
	reusableApp     *templateui.ReusableApp
	contentProvider *TopicContentProvider

	// Full-page view cache. The template system (JoinHorizontal/Vertical, border
	// rendering) is expensive. We rebuild it only when renderVersion advances.
	viewCache        string
	viewCacheVersion uint64
}

// GetCommon returns the shared context
func (t *TopicPageModel) GetCommon() *core.Common {
	return t.common
}

// NewTopicPageModel creates a new topic page model using the template system
// Deprecated: Use NewTopicPageModelWithCommon for new code
func NewTopicPageModel(dataSource api.KafkaDataSource, topicName string, topicDetails api.Topic) *TopicPageModel {
	// Create Common context with data source
	common := core.NewCommon(dataSource)
	return NewTopicPageModelWithCommon(common, topicName, topicDetails)
}

// NewTopicPageModelWithCommon creates a new topic page model using the Common context pattern
func NewTopicPageModelWithCommon(common *core.Common, topicName string, topicDetails api.Topic) *TopicPageModel {
	// Create the original topic model for business logic
	topicModel := NewModel(common.DataSource, topicName, topicDetails)
	// Set common context for layout system access
	topicModel.common = common
	// Build the display-time masker from per-cluster masking rules (MSG-28).
	topicModel.masker = buildMaskerFromConfig(common)
	// Apply per-cluster serde config: rebuild the registry with configured
	// serdes and pre-select any topic-bound serde (MSG-17).
	topicModel.applySerdeConfig(common)

	// Create topic-specific providers
	contentProvider := NewTopicContentProvider(topicModel)
	headerProvider := NewTopicHeaderDataProvider(topicModel)

	// Create sidebar sections
	sidebarSections := []providers.SidebarSection{
		NewTopicInfoSection(topicModel),
		NewMessageInfoSection(topicModel),
		NewConsumptionControlSection(topicModel),
		NewTopicShortcutsSection(topicModel),
	}

	// Create app configuration using template providers
	config := &providers.AppConfig{
		ContentProvider:             contentProvider,
		HeaderDataProvider:          headerProvider,
		SidebarSections:             sidebarSections,
		ShowSidebarByDefault:        true,
		CompactModeWidthBreakpoint:  120,
		CompactModeHeightBreakpoint: 30,
	}

	// Create the reusable app with our topic providers
	reusableApp := templateui.NewReusableApp(config)

	return &TopicPageModel{
		common:          common,
		topicModel:      topicModel,
		reusableApp:     reusableApp,
		contentProvider: contentProvider,
	}
}

// Init implements the Page interface
func (t *TopicPageModel) Init() tea.Cmd {
	// Initialize the reusable app (which will initialize providers)
	return t.reusableApp.Init()
}

// Update implements the Page interface
func (t *TopicPageModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Update the reusable app (which delegates to content provider)
	updatedApp, cmd := t.reusableApp.Update(msg)
	if updatedReusableApp, ok := updatedApp.(*templateui.ReusableApp); ok {
		t.reusableApp = updatedReusableApp
	}
	return t, cmd
}

// View implements the Page interface — caches the full rendered page and only
// rebuilds when the model's renderVersion has advanced.
func (t *TopicPageModel) View() string {
	v := t.topicModel.renderVersion
	if t.viewCache != "" && t.viewCacheVersion == v {
		return t.viewCache
	}
	t.viewCache = t.reusableApp.View()
	t.viewCacheVersion = v
	return t.viewCache
}

// SetDimensions implements the Page interface
func (t *TopicPageModel) SetDimensions(width, height int) {
	// Only update the reusable app. The topic model's dimensions will be
	// updated by the ContentProvider with the correct inner content dimensions.
	t.reusableApp.Update(tea.WindowSizeMsg{Width: width, Height: height})
}

// GetID implements the Page interface
func (t *TopicPageModel) GetID() string {
	if t.topicModel != nil && t.topicModel.topicName != "" {
		return "topic:" + t.topicModel.topicName
	}
	return "topic"
}

// GetTitle implements the Page interface
func (t *TopicPageModel) GetTitle() string {
	return t.topicModel.GetTitle()
}

// GetHelp implements the Page interface
func (t *TopicPageModel) GetHelp() []key.Binding {
	return t.topicModel.GetHelp()
}

// HandleNavigation implements the Page interface
func (t *TopicPageModel) HandleNavigation(msg tea.Msg) (core.Page, tea.Cmd) {
	// Handle topic-specific navigation
	switch msg := msg.(type) {
	case NavigateToMessageDetailMsg:
		// This would typically create and return a message detail page
		// For now, just return self
		_ = msg
		return t, nil
	}
	return t, nil
}

// OnFocus implements the Page interface
func (t *TopicPageModel) OnFocus() tea.Cmd {
	return t.topicModel.OnFocus()
}

// OnBlur implements the Page interface
func (t *TopicPageModel) OnBlur() tea.Cmd {
	return t.topicModel.OnBlur()
}

// IsInputMode implements core.InputModeReporter.
func (t *TopicPageModel) IsInputMode() bool {
	return t.topicModel != nil && t.topicModel.IsInputMode()
}

// Dispose implements core.Disposer: the router evicted the page, so every
// consumption, live stream and listener it started is cancelled.
func (t *TopicPageModel) Dispose() {
	if t.topicModel != nil {
		t.topicModel.Dispose()
	}
}

// GetTopicName returns the current topic name
func (t *TopicPageModel) GetTopicName() string {
	return t.topicModel.GetTopicName()
}

// GetSelectedMessage returns the currently selected message
func (t *TopicPageModel) GetSelectedMessage() *api.Message {
	return t.topicModel.GetSelectedMessage()
}

// TopicModel returns the internal topic model for testing purposes
func (t *TopicPageModel) TopicModel() *Model {
	return t.topicModel
}
