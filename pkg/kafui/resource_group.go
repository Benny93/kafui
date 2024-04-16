package kafui

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/maruel/natural"
	"github.com/rivo/tview"
)

type ResourceGroup struct {
	onError               func(err error)
	FetchedConsumerGroups map[string]api.ConsumerGroup
}

func NewResouceConsumerGroup(onError func(err error)) *ResourceGroup {
	return &ResourceGroup{
		onError: onError,
	}
}

func (g *ResourceGroup) FetchGroupsRoutine(ctx context.Context, app *tview.Application, dataSource api.KafkaDataSource) {

	go func() {
		defer RecoverAndExit(app)
		for {

			groups := g.FetchConsumerGroups(dataSource)
			result := make(map[string]api.ConsumerGroup)
			for _, g := range groups {
				result[g.Name] = g
			}
			g.FetchedConsumerGroups = result

			// Check if the context has been canceled
			select {
			case <-ctx.Done():
				return
			default:
				time.Sleep(refreshInterval)
			}

		}
	}()
}

func (g *ResourceGroup) FetchConsumerGroups(dataSource api.KafkaDataSource) []api.ConsumerGroup {
	cgs, err := dataSource.GetConsumerGroups()
	if err != nil {
		//g.ShowNotification(fmt.Sprintf("Error fetching GetConsumerGroups:", err))
		g.onError(err)
		return []api.ConsumerGroup{}
	}
	return cgs
}
func (r ResourceGroup) StartFetchingData() {

}
func (r ResourceGroup) StopFetching() {

}
func (r ResourceGroup) UpdateTable(table *tview.Table, dataSource api.KafkaDataSource, search string) {

	r.ShowConsumerGroups(table, r.FetchedConsumerGroups, search)
	//m.ShowNotification("Fetched Consumer Groups ...")

}

func (r ResourceGroup) ShowConsumerGroups(table *tview.Table, cgs map[string]api.ConsumerGroup, search string) {
	table.Clear()
	// Define table headers
	table.SetCell(0, 0, tview.NewTableCell("Name").SetTextColor(tview.Styles.SecondaryTextColor))
	table.SetCell(0, 1, tview.NewTableCell("State").SetTextColor(tview.Styles.SecondaryTextColor))
	table.SetCell(0, 2, tview.NewTableCell("Consumers").SetTextColor(tview.Styles.SecondaryTextColor).SetExpansion(1))

	keys := make([]string, 0, len(cgs))
	for key := range cgs {
		if search == "" || strings.Contains(strings.ToLower(key), strings.ToLower(search)) {
			keys = append(keys, key)
		}
	}

	sort.Sort(natural.StringSlice(keys))

	for i, key := range keys {
		cg := cgs[key]
		// Add data to the table
		cell := tview.NewTableCell(cg.Name)
		table.SetCell(i+1, 0, cell)
		table.SetCell(i+1, 1, tview.NewTableCell(cg.State))
		table.SetCell(i+1, 2, tview.NewTableCell(strconv.Itoa(cg.Consumers)).SetExpansion(1))
	}
}

func (r ResourceGroup) GetName() string {
	return "ConsumerGroup"
}
