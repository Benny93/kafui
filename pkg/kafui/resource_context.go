package kafui

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/maruel/natural"
	"github.com/rivo/tview"
)

type ResourceContext struct {
	onError         func(err error)
	FetchedContexts map[string]string
	Name            string
	dataSource      api.KafkaDataSource
	cancelFetch     func()
	recoverFunc     func()
}

func NewResourceContext(dataSource api.KafkaDataSource, onError func(err error), recoverFunc func()) *ResourceContext {
	return &ResourceContext{
		onError:     onError,
		Name:        "Context",
		dataSource:  dataSource,
		recoverFunc: recoverFunc,
	}
}

func (c *ResourceContext) FetchContextRoutine(ctx context.Context, dataSource api.KafkaDataSource) {

	go func() {
		defer c.recoverFunc()
		for {

			f := c.FetchContexts(dataSource) //TODO create struct for context holding more information
			result := make(map[string]string)
			for _, str := range f {
				result[str] = str
			}
			c.FetchedContexts = result

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

func (rc *ResourceContext) FetchContexts(dataSource api.KafkaDataSource) []string {
	contexts, err := dataSource.GetContexts()
	if err != nil {
		//rc.ShowNotification(fmt.Sprintf("Error fetching contexts:", err))
		rc.onError(err)
		return []string{}
	}
	return contexts
}
func (r *ResourceContext) StartFetchingData() {
	ctx, cancel := context.WithCancel(context.Background())
	//defer cancel()
	r.cancelFetch = cancel
	r.FetchContextRoutine(ctx, r.dataSource)

}
func (r *ResourceContext) StopFetching() {
	if r.cancelFetch != nil {
		r.cancelFetch()
	}
}
func (r *ResourceContext) UpdateTable(table *tview.Table, dataSource api.KafkaDataSource, search string) {

	r.ShowContextsInTable(table, r.FetchedContexts, search)
	//m.ShowNotification("Fetched Contexts ...")
	//r.UpdateMidFlexTitle(m.SearchBar.CurrentResource.GetName(), table.GetRowCount())

}

func (r ResourceContext) ShowContextsInTable(table *tview.Table, contexts map[string]string, search string) {
	table.Clear()

	table.SetCell(0, 0, tview.NewTableCell("Context").SetTextColor(tview.Styles.SecondaryTextColor).SetExpansion(1))

	keys := make([]string, 0, len(contexts))
	for key := range contexts {
		if search == "" || strings.Contains(strings.ToLower(key), strings.ToLower(search)) {
			keys = append(keys, key)
		}
	}

	sort.Sort(natural.StringSlice(keys))

	for i, key := range keys {
		context := contexts[key]
		cell := tview.NewTableCell(context)
		cell.SetExpansion(1)
		table.SetCell(i+1, 0, cell)
	}
	//table.SetTitle(m.SearchBar.CurrentResource.GetName())
}

func (r ResourceContext) GetName() string {
	return r.Name
}
