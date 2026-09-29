package mainpage

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/shared"
	stylesPkg "github.com/Benny93/kafui/pkg/ui/styles"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
)

// sortBrokerItems sorts a slice of broker list items in place by the given
// column key ("id", "host", "port", "disk", "isr", "skew"). Items whose skew is
// absent (nil) always sort last, regardless of direction. Non-broker items are
// left in place (they sort as equal).
func sortBrokerItems(items []interface{}, col string, desc bool) {
	less := func(a, b *BrokerResourceItem) bool {
		switch col {
		case "host":
			return a.info.Host < b.info.Host
		case "port":
			return a.info.Port < b.info.Port
		case "disk":
			return a.stats.SegmentSize < b.stats.SegmentSize
		case "isr":
			return a.stats.InSyncReplicaCount < b.stats.InSyncReplicaCount
		case "skew":
			as, bs := a.stats.ReplicaSkew, b.stats.ReplicaSkew
			// nil always last
			if as == nil || bs == nil {
				if as == nil && bs == nil {
					return false
				}
				// The absent one is "greater"; when descending we still want it
				// last, so bias it independent of direction (handled by caller flip).
				return as != nil
			}
			return *as < *bs
		default: // "id"
			return a.info.ID < b.info.ID
		}
	}

	sort.SliceStable(items, func(i, j int) bool {
		ai, aok := brokerItemFrom(items[i])
		bj, bok := brokerItemFrom(items[j])
		if !aok || !bok {
			return false
		}
		// Absent skews are pinned last for both directions.
		if col == "skew" {
			an, bn := ai.stats.ReplicaSkew == nil, bj.stats.ReplicaSkew == nil
			if an != bn {
				return bn // non-nil (bn==true means j is nil) sorts before nil
			}
			if an && bn {
				return false
			}
		}
		if desc {
			return less(bj, ai)
		}
		return less(ai, bj)
	})
}

// brokerItemFrom unwraps a broker resource item from the list-item wrappers.
func brokerItemFrom(item interface{}) (*BrokerResourceItem, bool) {
	switch v := item.(type) {
	case shared.ResourceListItem:
		bri, ok := v.ResourceItem.(*BrokerResourceItem)
		return bri, ok
	case shared.HighlightedResourceListItem:
		bri, ok := v.ResourceItem.(*BrokerResourceItem)
		return bri, ok
	case *BrokerResourceItem:
		return v, true
	default:
		return nil, false
	}
}

// brokerRowData builds a bubble-table row for a broker, styling the ISR cell in
// the alert style when under-replicated and the skew cell by severity.
func brokerRowData(b *BrokerResourceItem, placeholder string) table.RowData {
	port := strconv.FormatInt(int64(b.info.Port), 10)
	if !b.hasStats {
		return table.RowData{
			colBrokerID:   b.idCell(),
			colBrokerHost: b.info.Host,
			colBrokerPort: port,
			colBrokerDisk: placeholder,
			colBrokerISR:  placeholder,
			colBrokerSkew: placeholder,
		}
	}

	disk := shared.FormatDiskUsage(b.stats.SegmentSize, b.stats.SegmentCount)

	isrText, alert := shared.FormatISR(b.stats.InSyncReplicaCount, b.stats.ReplicaCount)
	isrCell := isrText
	if alert && isrText != "" {
		isrCell = lipgloss.NewStyle().Foreground(stylesPkg.Error).Render(isrText)
	}

	skewText := shared.FormatSkew(b.stats.ReplicaSkew)
	skewCell := skewText
	switch shared.SkewSeverity(b.stats.ReplicaSkew) {
	case shared.SkewError:
		skewCell = lipgloss.NewStyle().Foreground(stylesPkg.Error).Render(skewText)
	case shared.SkewWarning:
		skewCell = lipgloss.NewStyle().Foreground(stylesPkg.Warning).Render(skewText)
	}

	return table.RowData{
		colBrokerID:   b.idCell(),
		colBrokerHost: b.info.Host,
		colBrokerPort: port,
		colBrokerDisk: disk,
		colBrokerISR:  isrCell,
		colBrokerSkew: skewCell,
	}
}

// isBrokerResource reports whether the brokers resource is currently active.
func (k *KafuiContentProvider) isBrokerResource() bool {
	return k.currentResource != nil && k.currentResource.GetType() == BrokerResourceType
}

// loadBrokerStats fetches per-broker statistics + summary asynchronously and
// returns a BrokerStatsLoadedMsg (second phase of the two-phase broker load).
func (k *KafuiContentProvider) loadBrokerStats() tea.Cmd {
	ds := k.dataSource
	return func() tea.Msg {
		stats, summary, err := ds.GetBrokerStats()
		if err != nil {
			shared.Log.Error("loadBrokerStats: GetBrokerStats failed", "err", err)
			return BrokerStatsLoadedMsg{Stats: map[int32]api.BrokerStats{}}
		}
		return BrokerStatsLoadedMsg{Stats: stats, Summary: summary}
	}
}

// applyBrokerStats merges enriched stats onto the broker items and rebuilds rows,
// preserving the active sort order.
func (k *KafuiContentProvider) applyBrokerStats(msg BrokerStatsLoadedMsg) {
	for _, item := range k.allItems {
		if bri, ok := brokerItemFrom(item); ok {
			if s, found := msg.Stats[bri.info.ID]; found {
				bri.SetStats(s)
			}
		}
	}
	k.applyBrokerSort()
	if k.isFiltered {
		k.reapplyFilter()
	} else {
		k.updateTableForCurrentPage()
	}
}

// brokerSortColumns maps the visible column order to a sort key.
var brokerSortColumns = []string{"id", "host", "port", "disk", "isr", "skew"}

// cycleBrokerSortColumn advances the sort column (wrapping) and re-sorts.
func (k *KafuiContentProvider) cycleBrokerSortColumn() {
	k.brokerSortCol = (k.brokerSortCol + 1) % len(brokerSortColumns)
	k.applyBrokerSortAndRefresh()
}

// toggleBrokerSortDir flips the sort direction and re-sorts.
func (k *KafuiContentProvider) toggleBrokerSortDir() {
	if k.brokerSortCol < 0 {
		k.brokerSortCol = 0
	}
	k.brokerSortDesc = !k.brokerSortDesc
	k.applyBrokerSortAndRefresh()
}

func (k *KafuiContentProvider) applyBrokerSortAndRefresh() {
	k.applyBrokerSort()
	k.pagination.Page = 0
	k.pagination.SetTotalItems(len(k.allItems))
	k.updateTableForCurrentPageAndReset()
}

// applyBrokerSort sorts allItems by the active broker sort column/direction.
// Absent (nil) skews always sort last, regardless of direction.
func (k *KafuiContentProvider) applyBrokerSort() {
	if k.brokerSortCol < 0 || k.brokerSortCol >= len(brokerSortColumns) || !k.isBrokerResource() {
		return
	}
	sortBrokerItems(k.allItems, brokerSortColumns[k.brokerSortCol], k.brokerSortDesc)
}

// exportBrokersCSV writes the current broker list (with stats) to a timestamped
// CSV file in the working directory and reports the path via a notification.
func (k *KafuiContentProvider) exportBrokersCSV() tea.Cmd {
	stats := map[int32]api.BrokerStats{}
	brokers := make([]api.BrokerInfo, 0, len(k.allItems))
	for _, item := range k.allItems {
		if bri, ok := brokerItemFrom(item); ok {
			brokers = append(brokers, bri.info)
			if bri.hasStats {
				stats[bri.info.ID] = bri.stats
			}
		}
	}
	if len(brokers) == 0 {
		return nil
	}
	filename := fmt.Sprintf("brokers-%s.csv", time.Now().Format("20060102-150405"))
	f, err := os.Create(filename)
	if err != nil {
		return core.NotifyError("CSV export failed", err)
	}
	defer f.Close()
	if err := shared.WriteBrokerCSV(f, brokers, stats); err != nil {
		return core.NotifyError("CSV export failed", err)
	}
	abs, _ := filepath.Abs(filename)
	return core.NewNotification(core.StatusInfo, "Brokers exported", abs)
}
