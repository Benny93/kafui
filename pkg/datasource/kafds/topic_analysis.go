package kafds

import (
	"context"
	"sync"

	"github.com/Benny93/kafui/pkg/analysis"
	"github.com/Benny93/kafui/pkg/api"
)

// analysisRegistry is the process-wide topic-analysis registry. It is lazily
// built with a ConsumeFunc backed by DoConsume so the engine stays decoupled
// from the datasource wiring.
var (
	analysisRegistry     *analysis.Registry
	analysisRegistryOnce sync.Once
)

func getAnalysisRegistry() *analysis.Registry {
	analysisRegistryOnce.Do(func() {
		analysisRegistry = analysis.NewRegistry(func(ctx context.Context, topic string, flags api.ConsumeFlags, handle api.MessageHandlerFunc, onError func(err any)) error {
			DoConsume(ctx, topic, flags, handle, onError)
			return nil
		})
	})
	return analysisRegistry
}

// StartTopicAnalysis implements api.KafkaDataSource. It validates the topic
// exists (TopicNotFoundError) and captures the current message total for the
// progress percentage before starting the background scan.
func (kp KafkaDataSourceKaf) StartTopicAnalysis(ctx context.Context, topicName string) error {
	details, err := kp.GetTopicDetails(topicName)
	if err != nil {
		return err
	}
	return getAnalysisRegistry().Start(ctx, topicName, details.MessageCount())
}

// GetTopicAnalysis implements api.KafkaDataSource.
func (kp KafkaDataSourceKaf) GetTopicAnalysis(topicName string) (*api.TopicAnalysis, error) {
	return getAnalysisRegistry().Get(topicName)
}

// CancelTopicAnalysis implements api.KafkaDataSource.
func (kp KafkaDataSourceKaf) CancelTopicAnalysis(topicName string) error {
	return getAnalysisRegistry().Cancel(topicName)
}
