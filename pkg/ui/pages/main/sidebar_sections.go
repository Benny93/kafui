package mainpage

import (
	"strconv"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/template/ui/providers"
	tea "github.com/charmbracelet/bubbletea"
)

// SidebarSection defines the interface for sidebar sections
// This should implement providers.SidebarSection
type SidebarSection interface {
	// GetTitle returns the section title
	GetTitle() string
	
	// RenderItems returns the items to display in this section
	RenderItems(maxItems, width int) []providers.SidebarItem
	
	// HandleSectionUpdate allows the section to handle messages and return commands
	HandleSectionUpdate(msg tea.Msg) tea.Cmd
	
	// InitSection initializes the section
	InitSection() tea.Cmd
	
	// RefreshSection refreshes the section data
	RefreshSection() tea.Cmd
}

// ResourcesSection shows available resource types
type ResourcesSection struct {
	dataSource      api.KafkaDataSource
	currentResource ResourceType
}

func NewResourcesSection(dataSource api.KafkaDataSource) *ResourcesSection {
	return &ResourcesSection{
		dataSource:      dataSource,
		currentResource: TopicResourceType,
	}
}

func (r *ResourcesSection) GetTitle() string {
	return "Resources"
}

func (r *ResourcesSection) RenderItems(maxItems, width int) []providers.SidebarItem {
	resources := []struct {
		name         string
		resourceType ResourceType
		icon         string
	}{
		{"Topics", TopicResourceType, "📋"},
		{"Consumer Groups", ConsumerGroupResourceType, "👥"},
		{"Schemas", SchemaResourceType, "📄"},
		{"Contexts", ContextResourceType, "🔧"},
	}
	
	items := make([]providers.SidebarItem, 0, len(resources))
	for _, res := range resources {
		status := "muted"
		icon := "○"
		if res.resourceType == r.currentResource {
			status = "success"
			icon = "●"
		}
		
		items = append(items, providers.SidebarItem{
			Icon:   icon,
			Text:   res.name,
			Value:  "",
			Status: status,
		})
		
		if len(items) >= maxItems {
			break
		}
	}
	
	return items
}

func (r *ResourcesSection) HandleSectionUpdate(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case SwitchResourceMsg:
		r.currentResource = ResourceType(msg)
	case CurrentResourceListMsg:
		r.currentResource = msg.ResourceType
	}
	return nil
}

func (r *ResourcesSection) InitSection() tea.Cmd {
	return nil
}

func (r *ResourcesSection) RefreshSection() tea.Cmd {
	return nil
}

// ClusterInfoSection shows cluster information
type ClusterInfoSection struct {
	dataSource   api.KafkaDataSource
	lastUpdate   time.Time
	clusterInfo  map[string]interface{}
	loading      bool
}

func NewClusterInfoSection(dataSource api.KafkaDataSource) *ClusterInfoSection {
	return &ClusterInfoSection{
		dataSource:  dataSource,
		lastUpdate:  time.Now(),
		clusterInfo: make(map[string]interface{}),
	}
}

func (c *ClusterInfoSection) GetTitle() string {
	return "Cluster Info"
}

func (c *ClusterInfoSection) RenderItems(maxItems, width int) []providers.SidebarItem {
	if c.loading {
		return []providers.SidebarItem{
			{
				Icon:   "⏳",
				Text:   "Loading...",
				Value:  "",
				Status: "info",
			},
		}
	}
	
	items := []providers.SidebarItem{
		{
			Icon:   "🌐",
			Text:   "Context",
			Value:  c.dataSource.GetContext(),
			Status: "info",
		},
		{
			Icon:   "🔗",
			Text:   "Connected",
			Value:  "Yes",
			Status: "success",
		},
		{
			Icon:   "⏰",
			Text:   "Last Update",
			Value:  c.lastUpdate.Format("15:04:05"),
			Status: "muted",
		},
	}
	
	// Add cluster-specific info if available
	if brokers, ok := c.clusterInfo["brokers"].(int); ok {
		items = append(items, providers.SidebarItem{
			Icon:   "🖥️",
			Text:   "Brokers",
			Value:  strconv.Itoa(brokers),
			Status: "info",
		})
	}
	
	if topics, ok := c.clusterInfo["topics"].(int); ok {
		items = append(items, providers.SidebarItem{
			Icon:   "📋",
			Text:   "Topics",
			Value:  strconv.Itoa(topics),
			Status: "info",
		})
	}
	
	// Limit items to maxItems
	if len(items) > maxItems {
		items = items[:maxItems]
	}
	
	return items
}

func (c *ClusterInfoSection) HandleSectionUpdate(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case TimerTickMsg:
		c.lastUpdate = time.Time(msg)
		return c.RefreshSection()
	case ClusterInfoMsg:
		c.clusterInfo = msg.Info
		c.loading = false
	}
	return nil
}

func (c *ClusterInfoSection) InitSection() tea.Cmd {
	return c.RefreshSection()
}

func (c *ClusterInfoSection) RefreshSection() tea.Cmd {
	c.loading = true
	return func() tea.Msg {
		// Get cluster information
		info := make(map[string]interface{})
		
		// Get topics count
		if topics, err := c.dataSource.GetTopics(); err == nil {
			info["topics"] = len(topics)
		}
		
		// Get consumer groups count (if available)
		// This would depend on your API having this method
		// info["consumer_groups"] = len(consumerGroups)
		
		// Mock broker count for now
		info["brokers"] = 3
		
		return ClusterInfoMsg{Info: info}
	}
}

// ShortcutsSection shows keyboard shortcuts
type ShortcutsSection struct{}

func NewShortcutsSection() *ShortcutsSection {
	return &ShortcutsSection{}
}

func (s *ShortcutsSection) GetTitle() string {
	return "Shortcuts"
}

func (s *ShortcutsSection) RenderItems(maxItems, width int) []providers.SidebarItem {
	shortcuts := []providers.SidebarItem{
		{
			Icon:   "🔍",
			Text:   "Search",
			Value:  "/",
			Status: "info",
		},
		{
			Icon:   "🔄",
			Text:   "Switch Resource",
			Value:  ":",
			Status: "info",
		},
		{
			Icon:   "↵",
			Text:   "Select",
			Value:  "Enter",
			Status: "info",
		},
		{
			Icon:   "⎋",
			Text:   "Back/Cancel",
			Value:  "Esc",
			Status: "info",
		},
		{
			Icon:   "🚪",
			Text:   "Quit",
			Value:  "q",
			Status: "warning",
		},
	}
	
	// Limit items to maxItems
	if len(shortcuts) > maxItems {
		shortcuts = shortcuts[:maxItems]
	}
	
	return shortcuts
}

func (s *ShortcutsSection) HandleSectionUpdate(msg tea.Msg) tea.Cmd {
	return nil
}

func (s *ShortcutsSection) InitSection() tea.Cmd {
	return nil
}

func (s *ShortcutsSection) RefreshSection() tea.Cmd {
	return nil
}

// Custom message types for sidebar sections
type ClusterInfoMsg struct {
	Info map[string]interface{}
}