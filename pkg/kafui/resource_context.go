package kafui

import (
	"context"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/rivo/tview"
)

type ResourceContext struct {
	onError         func(err error)
	FetchedContexts map[string]string
	Name            string
}

func NewResourceContext(onError func(err error)) *ResourceContext {
	return &ResourceContext{
		onError: onError,
		Name:    "Context",
	}
}

func (c *ResourceContext) FetchContextRoutine(ctx context.Context, app *tview.Application, dataSource api.KafkaDataSource) {

	go func() {
		defer RecoverAndExit(app)
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
func (r ResourceContext) StartFetchingData() {

}
func (r ResourceContext) StopFetching() {

}

func (r ResourceContext) GetName() string {
	return r.Name
}
