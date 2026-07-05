package router

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/core"
	appconfigpage "github.com/Benny93/kafui/pkg/ui/pages/appconfig_view"
	brokerpage "github.com/Benny93/kafui/pkg/ui/pages/broker"
	clusterformpage "github.com/Benny93/kafui/pkg/ui/pages/cluster_form"
	clusterspage "github.com/Benny93/kafui/pkg/ui/pages/clusters"
	connectorpage "github.com/Benny93/kafui/pkg/ui/pages/connector"
	consumergrouppage "github.com/Benny93/kafui/pkg/ui/pages/consumer_group"
	errorpage "github.com/Benny93/kafui/pkg/ui/pages/errorpage"
	ksqlpage "github.com/Benny93/kafui/pkg/ui/pages/ksql"
	metricspage "github.com/Benny93/kafui/pkg/ui/pages/metrics"
	mainpage "github.com/Benny93/kafui/pkg/ui/pages/main"
	messagedetailpage "github.com/Benny93/kafui/pkg/ui/pages/message_detail"
	resourcedetailpage "github.com/Benny93/kafui/pkg/ui/pages/resource_detail"
	schemadetailpage "github.com/Benny93/kafui/pkg/ui/pages/schema_detail"
	topicpage "github.com/Benny93/kafui/pkg/ui/pages/topic"
	"github.com/Benny93/kafui/pkg/ui/shared"
	tea "github.com/charmbracelet/bubbletea"
)

// Router manages page navigation and state
type Router struct {
	com         *core.Common
	pages       map[string]core.Page
	history     []string
	currentPage string
	width       int
	height      int

	// initialPageID/initialData deep-link the first render to a page other than
	// "main" (UI-9); "main" is seeded beneath it so esc/back behaves normally.
	initialPageID string
	initialData   interface{}
}

// SetInitialRoute deep-links the app to a page on startup (before Init).
func (r *Router) SetInitialRoute(pageID string, data interface{}) {
	r.initialPageID = pageID
	r.initialData = data
}

// NavigationData contains data passed during navigation
type NavigationData struct {
	TopicName    string
	Topic        api.Topic
	Message      api.Message
	ResourceItem shared.ResourceItem
	ResourceType string
	SchemaItem   *mainpage.SchemaResourceItem
	BrokerID     int32
	BrokerInfo   api.BrokerInfo
	HasBroker    bool
}

// NewRouter creates a new Router instance
func NewRouter(com *core.Common) *Router {
	return &Router{
		com:         com,
		pages:       make(map[string]core.Page),
		history:     make([]string, 0),
		currentPage: "main",
	}
}

// NavigateTo switches to a specific page with optional data
func (r *Router) NavigateTo(pageID string, data interface{}) tea.Cmd {
	r.pushHistory(pageID)
	return tea.Batch(r.navigateToWithoutHistory(pageID, data), r.updateBreadcrumbs())
}

// pushHistory records r.currentPage onto the history stack before navigating
// to pageID. When pageID is actually the page we just came from (a page
// returning to its caller via forward navigation, e.g. a PageChangeMsg,
// instead of Router.Back()), that top entry is popped instead of appending a
// duplicate — otherwise repeating such a round trip (e.g. open the Clusters
// dashboard, switch context back to "main", reopen) grows history and the
// breadcrumb bar without bound (BUG-8).
func (r *Router) pushHistory(pageID string) {
	if r.currentPage == "" || r.currentPage == pageID {
		return
	}
	if len(r.history) > 0 && r.history[len(r.history)-1] == pageID {
		r.history = r.history[:len(r.history)-1]
		return
	}
	r.history = append(r.history, r.currentPage)
}

// updateBreadcrumbs creates a command to update breadcrumbs for the current page
func (r *Router) updateBreadcrumbs() tea.Cmd {
	items := r.getBreadcrumbs()
	return func() tea.Msg {
		return core.BreadcrumbUpdateMsg{Items: items}
	}
}

// getBreadcrumbs returns a list of page titles representing the current navigation path
func (r *Router) getBreadcrumbs() []string {
	var breadcrumbs []string

	// Add historical pages
	for _, pageID := range r.history {
		if page, exists := r.pages[pageID]; exists {
			breadcrumbs = append(breadcrumbs, page.GetTitle())
		}
	}

	// Add current page
	if page, exists := r.pages[r.currentPage]; exists {
		breadcrumbs = append(breadcrumbs, page.GetTitle())
	}

	return breadcrumbs
}

// navigateToWithoutHistory switches to a specific page without adding to history
func (r *Router) navigateToWithoutHistory(pageID string, data interface{}) tea.Cmd {
	// Initialize page if needed
	if _, exists := r.pages[pageID]; !exists {
		page := r.createPage(pageID, data)
		if page != nil {
			r.pages[pageID] = page

			// Set dimensions if we have them
			if r.width > 0 && r.height > 0 {
				page.SetDimensions(r.width, r.height)
			}
		} else {
			// If page creation failed, don't navigate
			return nil
		}
	}

	// Blur current page
	if r.currentPage != "" {
		if currentPage, exists := r.pages[r.currentPage]; exists {
			currentPage.OnBlur()
		}
	}

	r.currentPage = pageID

	// Initialize and focus the new page
	if page, exists := r.pages[pageID]; exists {
		initCmd := page.Init()
		focusCmd := page.OnFocus()

		// Return both commands batched together
		if initCmd != nil && focusCmd != nil {
			return tea.Batch(initCmd, focusCmd)
		} else if initCmd != nil {
			return initCmd
		} else if focusCmd != nil {
			return focusCmd
		}
	}

	return nil
}

// Back navigates to the previous page
func (r *Router) Back() tea.Cmd {
	if len(r.history) > 0 {
		lastPage := r.history[len(r.history)-1]
		r.history = r.history[:len(r.history)-1]
		return tea.Batch(r.navigateToWithoutHistory(lastPage, nil), r.updateBreadcrumbs())
	}
	return nil
}

// GetCurrentPage returns the current page
func (r *Router) GetCurrentPage() core.Page {
	if page, exists := r.pages[r.currentPage]; exists {
		return page
	}
	return nil
}

// GetCurrentPageID returns the current page ID
func (r *Router) GetCurrentPageID() string {
	return r.currentPage
}

// GetHistory returns a copy of the navigation history
func (r *Router) GetHistory() []string {
	history := make([]string, len(r.history))
	copy(history, r.history)
	return history
}

// SetDimensions updates the dimensions for all pages
func (r *Router) SetDimensions(width, height int) {
	r.width = width
	r.height = height

	// Update dimensions for all pages
	for _, page := range r.pages {
		page.SetDimensions(width, height)
	}
}

// ClearHistory clears the navigation history
func (r *Router) ClearHistory() {
	r.history = r.history[:0]
}

// createPage creates a new page instance based on pageID and data
func (r *Router) createPage(pageID string, data interface{}) core.Page {
	// Support dynamic page IDs like "topic:my-topic"
	baseID := pageID
	if idx := strings.Index(pageID, ":"); idx != -1 {
		baseID = pageID[:idx]
	}

	switch baseID {
	case "main":
		return mainpage.NewModelWithCommon(r.com)

	case "topic":
		// Extract topic data
		if navData, ok := data.(*NavigationData); ok {
			return topicpage.NewTopicPageModelWithCommon(r.com, navData.TopicName, navData.Topic)
		}
		// Fallback with empty data
		return topicpage.NewTopicPageModelWithCommon(r.com, "unknown", api.Topic{})

	case "message_detail", "detail":
		// Extract message data - handle both "message_detail" and legacy "detail" page IDs
		if navData, ok := data.(*NavigationData); ok {
			// PageID format is "detail:<topicName>:<partition>:<offset>"; extract topic name if not set
			topicName := navData.TopicName
			if topicName == "" {
				parts := strings.SplitN(pageID, ":", 3)
				if len(parts) >= 2 {
					topicName = parts[1]
				}
			}
			return messagedetailpage.NewMessageDetailPageModelWithCommon(r.com, topicName, navData.Message)
		}
		// Fallback with empty data
		return messagedetailpage.NewMessageDetailPageModelWithCommon(r.com, "unknown", api.Message{})

	case "resource_detail":
		// Extract resource data
		if navData, ok := data.(*NavigationData); ok && navData.ResourceItem != nil {
			return resourcedetailpage.NewModelWithCommon(navData.ResourceItem, navData.ResourceType, r.com)
		}
		// Fallback with minimal resource
		minimalResource := &shared.MinimalResourceItem{ID: "unknown"}
		return resourcedetailpage.NewModelWithCommon(minimalResource, "unknown", r.com)

	case "schema_detail":
		if navData, ok := data.(*NavigationData); ok && navData.SchemaItem != nil {
			return schemadetailpage.NewSchemaDetailPageModel(r.com, navData.SchemaItem)
		}
		return mainpage.NewModelWithCommon(r.com)

	case "appconfig":
		return appconfigpage.NewModelWithCommon(r.com)

	case "clusters":
		return clusterspage.NewModelWithCommon(r.com)

	case "ksql":
		return ksqlpage.NewModelWithCommon(r.com)

	case "metrics":
		return metricspage.NewModelWithCommon(r.com)

	case "cluster_form":
		// cluster_form or cluster_form:<name> — empty name = add mode.
		name := ""
		if idx := strings.Index(pageID, ":"); idx != -1 {
			name = pageID[idx+1:]
		}
		return clusterformpage.NewModelWithCommon(r.com, name)

	case "ksql_query":
		if navData, ok := data.(*NavigationData); ok && navData.TopicName != "" {
			return ksqlpage.NewQueryModelWithSeed(r.com, navData.TopicName)
		}
		return ksqlpage.NewQueryModelWithCommon(r.com)

	case "connector":
		// connector:<connectCluster>:<name> — connect-cluster names contain no ':'.
		parts := strings.SplitN(pageID, ":", 3)
		if len(parts) == 3 {
			return connectorpage.NewModelWithCommon(r.com, parts[1], parts[2])
		}
		return mainpage.NewModelWithCommon(r.com)

	case "consumer_group":
		// consumer_group:<groupID> — parse the id from the page ID (robust even
		// if a group id contains ':', since we take everything after the first).
		if idx := strings.Index(pageID, ":"); idx != -1 {
			return consumergrouppage.NewModelWithCommon(r.com, pageID[idx+1:])
		}
		return mainpage.NewModelWithCommon(r.com)

	case "broker":
		// broker:<id> — extract the broker id (and optional info) from data or the ID.
		if navData, ok := data.(*NavigationData); ok && navData.HasBroker {
			return brokerpage.NewModelWithInfo(r.com, navData.BrokerID, navData.BrokerInfo)
		}
		// Fall back to parsing the id out of "broker:<id>".
		if idx := strings.Index(pageID, ":"); idx != -1 {
			if id, err := strconv.Atoi(pageID[idx+1:]); err == nil {
				return brokerpage.NewModelWithCommon(r.com, int32(id))
			}
		}
		return mainpage.NewModelWithCommon(r.com)

	default:
		// Unknown page ID → a not-found error page (UI-10) rather than silently
		// aborting navigation or masquerading as the main page.
		return errorpage.New(r.com, errorpage.NotFound, "Page not found", "No page is registered for \""+pageID+"\".")
	}
}

// Update handles router-level updates and delegates to current page
func (r *Router) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	currentPage := r.GetCurrentPage()
	if currentPage == nil {
		return r, nil
	}

	// Handle navigation messages
	switch msg := msg.(type) {
	case core.BackMsg:
		// Handle back navigation without adding to history
		return r, r.Back()

	case core.PageChangeMsg:
		// Create navigation data from the message
		var navData *NavigationData
		if msg.Data != nil {
			navData = &NavigationData{}

			// Extract data based on message data type
			switch data := msg.Data.(type) {
			case map[string]interface{}:
				if name, ok := data["name"].(string); ok {
					navData.TopicName = name
				}
				if topic, ok := data["topic"].(api.Topic); ok {
					navData.Topic = topic
				}
				if resourceItem, ok := data["resourceItem"].(shared.ResourceItem); ok {
					navData.ResourceItem = resourceItem
				}
				if resourceType, ok := data["resourceType"].(string); ok {
					navData.ResourceType = resourceType
				}
				if schemaItem, ok := data["schemaItem"].(*mainpage.SchemaResourceItem); ok {
					navData.SchemaItem = schemaItem
				}
				if brokerID, ok := data["brokerID"].(int32); ok {
					navData.BrokerID = brokerID
					navData.HasBroker = true
				}
				if brokerInfo, ok := data["brokerInfo"].(api.BrokerInfo); ok {
					navData.BrokerInfo = brokerInfo
				}
			case *NavigationData:
				navData = data
			case NavigationData:
				navData = &data
			case api.Message:
				navData.Message = data
			case shared.ResourceItem:
				navData.ResourceItem = data
			}
		}

		// Pass a genuinely nil interface{} when there's no data, rather than a
		// non-nil interface wrapping a nil *NavigationData — createPage's `data,
		// ok := data.(*NavigationData)` type assertions succeed on the latter
		// (the dynamic type matches) and then dereference the nil pointer.
		if navData == nil {
			return r, r.NavigateTo(msg.PageID, nil)
		}
		return r, r.NavigateTo(msg.PageID, navData)
	}

	// Check if current page wants to handle navigation
	newPage, navCmd := currentPage.HandleNavigation(msg)

	// If page returned a navigation command, execute it to get the actual message
	if navCmd != nil {
		if cmdMsg := navCmd(); cmdMsg != nil {
			// Process the returned message (likely a PageChangeMsg) recursively
			return r.Update(cmdMsg)
		}
	}

	// If page navigation occurred (page returned different page), update router state
	if newPage != currentPage {
		// Page navigation occurred, update router state
		if newPage != nil {
			// Determine the page ID from the new page
			newPageID := newPage.GetID()
			r.pushHistory(newPageID)

			// Blur current page
			if r.currentPage != "" {
				if currentPageObj, exists := r.pages[r.currentPage]; exists {
					currentPageObj.OnBlur()
				}
			}

			// Update router state
			r.currentPage = newPageID
			r.pages[newPageID] = newPage

			// Set dimensions if we have them
			if r.width > 0 && r.height > 0 {
				newPage.SetDimensions(r.width, r.height)
			}

			// Focus the new page
			focusCmd := newPage.OnFocus()
			return r, focusCmd
		}
	}

	// Delegate to current page
	updatedPage, cmd := currentPage.Update(msg)

	// Update the page in our map if it changed
	if updatedPage != nil {
		r.pages[r.currentPage] = updatedPage.(core.Page)
	}

	return r, cmd
}

// View renders the current page
func (r *Router) View() string {
	currentPage := r.GetCurrentPage()
	if currentPage == nil {
		return fmt.Sprintf("Page '%s' not found", r.currentPage)
	}
	return currentPage.View()
}

// Init initializes the router by navigating to the main page, then to the
// deep-link target (if any) so "main" sits beneath it in history.
func (r *Router) Init() tea.Cmd {
	mainCmd := r.NavigateTo("main", nil)
	if r.initialPageID == "" || r.initialPageID == "main" {
		return mainCmd
	}
	return tea.Batch(mainCmd, r.NavigateTo(r.initialPageID, r.initialData))
}
