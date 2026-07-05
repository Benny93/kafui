package mainpage

import (
	"fmt"
	"strconv"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/template/ui/providers"
	tea "github.com/charmbracelet/bubbletea"
)

// BrokerSummarySection renders the brokers summary panel (BR-11) in the sidebar.
// It only shows content while the brokers resource is active and after the
// BrokerStatsLoadedMsg has delivered a summary.
type BrokerSummarySection struct {
	active     bool
	hasSummary bool
	summary    api.BrokerSummary
}

// NewBrokerSummarySection creates the (initially inactive) summary section.
func NewBrokerSummarySection() *BrokerSummarySection {
	return &BrokerSummarySection{}
}

func (s *BrokerSummarySection) GetTitle() string { return "Broker Summary" }

func (s *BrokerSummarySection) RenderItems(maxItems, width int) []providers.SidebarItem {
	if !s.active {
		return nil
	}
	if !s.hasSummary {
		return []providers.SidebarItem{{Icon: "⏳", Text: "Loading...", Status: "info"}}
	}
	items := brokerSummaryItems(s.summary)
	if len(items) > maxItems {
		items = items[:maxItems]
	}
	return items
}

func (s *BrokerSummarySection) HandleSectionUpdate(msg tea.Msg) tea.Cmd {
	switch m := msg.(type) {
	case SwitchResourceMsg:
		s.active = ResourceType(m) == BrokerResourceType
	case CurrentResourceListMsg:
		s.active = m.ResourceType == BrokerResourceType
	case BrokerStatsLoadedMsg:
		s.summary = m.Summary
		s.hasSummary = true
	}
	return nil
}

func (s *BrokerSummarySection) InitSection() tea.Cmd    { return nil }
func (s *BrokerSummarySection) RefreshSection() tea.Cmd { return nil }

// brokerSummaryItems is the pure view-model for the broker summary panel: it maps
// a BrokerSummary to labelled sidebar items with the status/severity styling
// described by BR-11. Kept side-effect-free so it can be table-tested.
func brokerSummaryItems(sum api.BrokerSummary) []providers.SidebarItem {
	items := []providers.SidebarItem{
		{Icon: "🖥️", Text: "Brokers", Value: strconv.Itoa(sum.BrokerCount), Status: "info"},
	}

	// Active controller (or no-controller warning).
	if sum.ControllerID != nil {
		items = append(items, providers.SidebarItem{
			Icon: "★", Text: "Controller", Value: "#" + strconv.FormatInt(int64(*sum.ControllerID), 10), Status: "success",
		})
	} else {
		items = append(items, providers.SidebarItem{
			Icon: "⚠", Text: "Controller", Value: "No Active Controller", Status: "error",
		})
	}

	version := sum.ClusterVersion
	if version == "" {
		version = "Unknown"
	}
	items = append(items, providers.SidebarItem{Icon: "🏷", Text: "Version", Value: version, Status: "muted"})

	controllerType := sum.ControllerType
	if controllerType == "" {
		controllerType = "Unknown"
	}
	items = append(items, providers.SidebarItem{Icon: "⚙", Text: "Type", Value: controllerType, Status: "info"})

	// Partitions block.
	offline := sum.TotalPartitions - sum.OnlinePartitions
	onlineStatus := "success"
	if offline > 0 {
		onlineStatus = "error"
	}
	items = append(items, providers.SidebarItem{
		Icon: "◧", Text: "Online", Value: fmt.Sprintf("%d of %d", sum.OnlinePartitions, sum.TotalPartitions), Status: onlineStatus,
	})

	underStatus := "success"
	if sum.UnderReplicated > 0 {
		underStatus = "error"
	}
	items = append(items, providers.SidebarItem{
		Icon: "◔", Text: "Under-repl", Value: strconv.Itoa(sum.UnderReplicated), Status: underStatus,
	})

	inSyncStatus := "success"
	if sum.InSyncReplicas < sum.TotalReplicas {
		inSyncStatus = "error"
	}
	items = append(items, providers.SidebarItem{
		Icon: "◉", Text: "In-Sync", Value: fmt.Sprintf("%d of %d", sum.InSyncReplicas, sum.TotalReplicas), Status: inSyncStatus,
	})

	outStatus := "muted"
	if sum.OutOfSync > 0 {
		outStatus = "error"
	}
	items = append(items, providers.SidebarItem{
		Icon: "○", Text: "Out-of-Sync", Value: strconv.Itoa(sum.OutOfSync), Status: outStatus,
	})

	return items
}
