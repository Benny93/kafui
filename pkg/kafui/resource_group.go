package kafui

import (
	"context"
	"time"

	"github.com/Benny93/kafui/pkg/api"
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
func (r ResourceGroup) GetName() string {
	return "ConsumerGroup"
}
