package router

import (
	"fmt"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/core"
	messagedetailpage "github.com/Benny93/kafui/pkg/ui/pages/message_detail"
	mainpage "github.com/Benny93/kafui/pkg/ui/pages/main"
	resourcedetailpage "github.com/Benny93/kafui/pkg/ui/pages/resource_detail"
	topicpage "github.com/Benny93/kafui/pkg/ui/pages/topic"
	"github.com/Benny93/kafui/pkg/ui/shared"
	tea "github.com/charmbracelet/bubbletea"
)

// Router manages page navigation and state
type Router struct {
	dataSource  api.KafkaDataSource
	pages       map[string]core.Page
	history     []string
	currentPage string
	width       int
	height      int
}

// NavigationData contains data passed during navigation
type NavigationData struct {
	TopicName    string
	Topic        api.Topic
	Message      api.Message
	ResourceItem shared.ResourceItem
	ResourceType string
}

// NewRouter creates a new Router instance
func NewRouter(dataSource api.KafkaDataSource) *Router {
	return &Router{
		dataSource:  dataSource,
		pages:       make(map[string]core.Page),
		history:     make([]string, 0),
		currentPage: "main",
	}
}

// NavigateTo switches to a specific page with optional data
func (r *Router) NavigateTo(pageID string, data interface{}) tea.Cmd {
	// Add current page to history if it's different from the target
	if r.currentPage != "" && r.currentPage != pageID {
		r.history = append(r.history, r.currentPage)
	}
	
	return r.navigateToWithoutHistory(pageID, data)
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
		return r.navigateToWithoutHistory(lastPage, nil)
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
	switch pageID {
	case "main":
		return mainpage.NewModel(r.dataSource)
		
	case "topic":
		// Extract topic data
		if navData, ok := data.(*NavigationData); ok {
			return topicpage.NewTopicPageModel(r.dataSource, navData.TopicName, navData.Topic)
		}
		// Fallback with empty data
		return topicpage.NewTopicPageModel(r.dataSource, "unknown", api.Topic{})
		
	case "message_detail", "detail":
		// Extract message data - handle both "message_detail" and legacy "detail" page IDs
		if navData, ok := data.(*NavigationData); ok {
			return messagedetailpage.NewModel(r.dataSource, navData.TopicName, navData.Message)
		}
		// Fallback with empty data
		return messagedetailpage.NewModel(r.dataSource, "unknown", api.Message{})
		
	case "resource_detail":
		// Extract resource data
		if navData, ok := data.(*NavigationData); ok && navData.ResourceItem != nil {
			return resourcedetailpage.NewModel(navData.ResourceItem, navData.ResourceType)
		}
		// Fallback with minimal resource
		minimalResource := &minimalResourceItem{id: "unknown"}
		return resourcedetailpage.NewModel(minimalResource, "unknown")
		
	default:
		// Default to main page for unknown page IDs
		return mainpage.NewModel(r.dataSource)
	}
}

// minimalResourceItem is a minimal implementation of ResourceItem for fallback cases
type minimalResourceItem struct {
	id string
}

func (m *minimalResourceItem) GetID() string {
	return m.id
}

func (m *minimalResourceItem) GetValues() []string {
	return []string{m.id}
}

func (m *minimalResourceItem) GetDetails() map[string]string {
	return map[string]string{
		"Name": m.id,
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
		
		return r, r.NavigateTo(msg.PageID, navData)
	}
	
	// Check if current page wants to handle navigation
	shared.DebugLog("Router.Update: Checking if current page (%s) wants to handle navigation for message type: %T", r.currentPage, msg)
	newPage, navCmd := currentPage.HandleNavigation(msg)
	
	// If page returned a navigation command, execute it to get the actual message
	if navCmd != nil {
		shared.DebugLog("Router.Update: Page returned navigation command, executing it")
		if cmdMsg := navCmd(); cmdMsg != nil {
			shared.DebugLog("Router.Update: Navigation command returned message: %T", cmdMsg)
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
			
			// Add current page to history
			if r.currentPage != "" && r.currentPage != newPageID {
				r.history = append(r.history, r.currentPage)
			}
			
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

// Init initializes the router by navigating to the main page
func (r *Router) Init() tea.Cmd {
	return r.NavigateTo("main", nil)
}