package mainpage

import (
	"strconv"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/template/ui/providers"
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
)

// sidebarZoneID returns a stable, unique zone ID for a resource type sidebar item.
func sidebarZoneID(rt ResourceType) string {
	return "sidebar-resource-" + rt.String()
}

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
	common          *core.Common // optional; enables capability gating
	currentResource ResourceType
}

func NewResourcesSection(dataSource api.KafkaDataSource) *ResourcesSection {
	return &ResourcesSection{
		dataSource:      dataSource,
		currentResource: TopicResourceType,
	}
}

// NewResourcesSectionWithCommon creates a ResourcesSection using Common context
func NewResourcesSectionWithCommon(common *core.Common) *ResourcesSection {
	r := NewResourcesSection(common.DataSource)
	r.common = common
	return r
}

// enabled reports whether a resource type should be shown for the active cluster.
// Optional integrations are gated on capabilities; core resources always show.
func (r *ResourcesSection) enabled(rt ResourceType) bool {
	if r.common == nil {
		return true
	}
	switch rt {
	case SchemaResourceType:
		return r.common.HasCapability(api.CapSchemaRegistry)
	case ACLResourceType:
		return r.common.HasCapability(api.CapACLView)
	case ConnectClusterResourceType, ConnectorResourceType:
		return r.common.HasCapability(api.CapKafkaConnect)
	default:
		return true
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
		{"ACLs", ACLResourceType, "🔒"},
		{"Brokers", BrokerResourceType, "🖥️"},
		{"Quotas", QuotaResourceType, "📊"},
		{"Connect Clusters", ConnectClusterResourceType, "🔌"},
		{"Connectors", ConnectorResourceType, "🔗"},
	}

	items := make([]providers.SidebarItem, 0, len(resources))
	for _, res := range resources {
		if !r.enabled(res.resourceType) {
			continue
		}
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
			ZoneID: sidebarZoneID(res.resourceType),
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
	case tea.MouseMsg:
		// Check if any resource sidebar item was clicked.
		for _, rt := range []ResourceType{TopicResourceType, ConsumerGroupResourceType, SchemaResourceType, ContextResourceType, ACLResourceType, BrokerResourceType, QuotaResourceType, ConnectClusterResourceType, ConnectorResourceType} {
			if r.enabled(rt) && zone.Get(sidebarZoneID(rt)).InBounds(msg) {
				return func() tea.Msg { return SwitchResourceMsg(rt) }
			}
		}
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
	dataSource  api.KafkaDataSource
	common      *core.Common // optional; enables health/read-only indicators
	lastUpdate  time.Time
	clusterInfo map[string]interface{}
	loading     bool
}

func NewClusterInfoSection(dataSource api.KafkaDataSource) *ClusterInfoSection {
	return &ClusterInfoSection{
		dataSource: dataSource,
		lastUpdate: time.Now(),
		// clusterInfo starts nil; RenderItems shows "Loading..." only until the
		// first fetch completes.
	}
}

// NewClusterInfoSectionWithCommon creates a ClusterInfoSection using Common context
func NewClusterInfoSectionWithCommon(common *core.Common) *ClusterInfoSection {
	c := NewClusterInfoSection(common.DataSource)
	c.common = common
	return c
}

// activeOverview returns the collector's cached overview for the active cluster,
// or (zero, false) if unavailable.
func (c *ClusterInfoSection) activeOverview() (api.ClusterOverview, bool) {
	if c.common == nil || c.common.Collector == nil {
		return api.ClusterOverview{}, false
	}
	active := c.dataSource.GetContext()
	for _, ov := range c.common.Collector.ListClusters() {
		if ov.Name == active {
			return ov, true
		}
	}
	return api.ClusterOverview{}, false
}

func (c *ClusterInfoSection) GetTitle() string {
	if c.dataSource != nil {
		if ctx := c.dataSource.GetContext(); ctx != "" {
			return ctx
		}
	}
	return "kafui"
}

func (c *ClusterInfoSection) RenderItems(maxItems, width int) []providers.SidebarItem {
	// Show "Loading..." only on the initial fetch, not on background refreshes.
	if c.loading && c.clusterInfo == nil {
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

	// Health + read-only indicators from the background collector.
	if ov, ok := c.activeOverview(); ok {
		glyph, status := "●", "success"
		switch ov.Status {
		case api.ClusterOffline:
			glyph, status = "○", "error"
		case api.ClusterInitializing:
			glyph, status = "◍", "info"
		}
		items = append(items, providers.SidebarItem{
			Icon:   glyph,
			Text:   "Status",
			Value:  string(ov.Status),
			Status: status,
		})
		if ov.ReadOnly {
			items = append(items, providers.SidebarItem{
				Icon:   "🔒",
				Text:   "Mode",
				Value:  "read-only",
				Status: "warning",
			})
		}
		if ov.Status == api.ClusterOffline && ov.LastError != "" {
			items = append(items, providers.SidebarItem{
				Icon:   "⚠",
				Text:   "Error",
				Value:  core.TruncateString(ov.LastError, 24),
				Status: "error",
			})
		}
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
	if c.clusterInfo == nil {
		c.loading = true
	}
	ds := c.dataSource
	return func() tea.Msg {
		info := make(map[string]interface{})
		// GetBrokers() enumerates brokers from cluster metadata (matching the
		// Clusters dashboard), not the configured bootstrap address list, which
		// is typically a single entry regardless of the cluster's real size (BUG-5).
		if brokers, err := ds.GetBrokers(); err == nil {
			info["brokers"] = len(brokers)
		}
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
		{Icon: "↑↓", Text: "Navigate", Value: "↑ / ↓", Status: "info"},
		{Icon: "↵", Text: "Select", Value: "Enter", Status: "info"},
		{Icon: "◁▷", Text: "Prev/Next page", Value: "h / l", Status: "info"},
		{Icon: "⇤⇥", Text: "First/Last page", Value: "g / G", Status: "info"},
		{Icon: "🔍", Text: "Search", Value: "/", Status: "info"},
		{Icon: "🔄", Text: "Switch resource", Value: ":", Status: "info"},
		{Icon: "⎋", Text: "Back/Cancel", Value: "Esc", Status: "info"},
		{Icon: "🚪", Text: "Quit", Value: "q", Status: "warning"},
	}

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
