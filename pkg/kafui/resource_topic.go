package kafui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/maruel/natural"
	"github.com/rivo/tview"
)

type ResouceTopic struct {
	LastFetchedTopics map[string]api.Topic
	dataSource        api.KafkaDataSource
	onError           func(err error)
	cancelFetch       func()
	recoverFunc       func()
}

func NewResouceTopic(dataSource api.KafkaDataSource, onError func(err error), recoverFunc func()) *ResouceTopic {
	return &ResouceTopic{
		dataSource:  dataSource,
		onError:     onError,
		recoverFunc: recoverFunc,
	}
}

func (r *ResouceTopic) UpdateTableDataRoutine(ctx context.Context, dataSource api.KafkaDataSource) {
	go func() {
		defer r.recoverFunc()
		for {

			r.LastFetchedTopics = r.FetchTopics(r.dataSource)

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

func (r *ResouceTopic) FetchTopics(dataSource api.KafkaDataSource) map[string]api.Topic {
	//time.Sleep(20 * time.Second) //TODO: remove
	topics, err := dataSource.GetTopics()
	if err != nil {
		//m.ShowNotification(fmt.Sprintf("Error reading topics:", err))
		r.onError(err)
		return make(map[string]api.Topic)
	}
	r.LastFetchedTopics = topics
	//m.ShowNotification("Fetched topics...")
	return topics
}

func (r ResouceTopic) StartFetchingData() {
	ctx, cancel := context.WithCancel(context.Background())
	//defer cancel()
	r.cancelFetch = cancel
	r.UpdateTableDataRoutine(ctx, r.dataSource)

}
func (r ResouceTopic) StopFetching() {
	r.cancelFetch()
}

func (r ResouceTopic) UpdateTable(table *tview.Table, dataSource api.KafkaDataSource, search string) {

	r.ShowTopicsInTable(table, r.LastFetchedTopics, search)
	//r.ShowNotification(fmt.Sprintf("Fetched Topics ... %d", len(r.LastFetchedTopics)))

}

func (r ResouceTopic) ShowTopicsInTable(table *tview.Table, topics map[string]api.Topic, search string) {
	table.Clear()
	table.SetCell(0, 0, tview.NewTableCell("Topic").SetTextColor(tview.Styles.SecondaryTextColor))
	//table.SetCell(0, 1, tview.NewTableCell("Num Messages").SetTextColor(tview.Styles.SecondaryTextColor))
	table.SetCell(0, 1, tview.NewTableCell("Num Partitions").SetTextColor(tview.Styles.SecondaryTextColor))
	table.SetCell(0, 2, tview.NewTableCell("Replication Factor").SetTextColor(tview.Styles.SecondaryTextColor).SetExpansion(1))

	keys := make([]string, 0, len(topics))
	for key := range topics {
		if search == "" || strings.Contains(strings.ToLower(key), strings.ToLower(search)) {
			keys = append(keys, key)
		}
	}

	sort.Sort(natural.StringSlice(keys))

	for i, key := range keys {
		value := topics[key]

		cell := tview.NewTableCell(key)
		cell.SetExpansion(1)
		table.SetCell(i+1, 0, cell)
		//table.SetCell(i+1, 1, tview.NewTableCell(fmt.Sprint(value.MessageCount)))
		table.SetCell(i+1, 1, tview.NewTableCell(fmt.Sprint(value.NumPartitions)))
		table.SetCell(i+1, 2, tview.NewTableCell(fmt.Sprint(value.ReplicationFactor)))

	}
	//table.SetTitle(m.SearchBar.CurrentResource.GetName())
}

func (r ResouceTopic) GetName() string {
	return "Topic"
}
