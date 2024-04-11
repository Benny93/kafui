package kafui

import (
	"context"
	"time"

	"github.com/Benny93/kafui/pkg/api"
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

func (r ResouceTopic) GetName() string {
	return "Topic"
}
