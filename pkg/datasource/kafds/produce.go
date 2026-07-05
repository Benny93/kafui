package kafds

import (
	"context"
	"fmt"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/IBM/sarama"
)

// newSyncProducer builds a SyncProducer from a client. It is a package-level
// seam so tests can inject a fake producer.
var newSyncProducer = func(client sarama.Client) (sarama.SyncProducer, error) {
	return sarama.NewSyncProducerFromClient(client)
}

// ProduceMessage implements api.KafkaDataSource (MSG-30).
func (kp KafkaDataSourceKaf) ProduceMessage(ctx context.Context, topic string, rec api.ProduceRecord) error {
	cfg, err := getConfig()
	if err != nil {
		return err
	}
	cfg.Producer.Return.Successes = true
	// Honour an explicit partition when one is requested.
	if rec.Partition != nil {
		cfg.Producer.Partitioner = sarama.NewManualPartitioner
	}
	client, err := getClientFromConfig(cfg)
	if err != nil {
		return api.NewConnectionErrorWithCause("unable to create client for produce", err)
	}
	defer client.Close()
	return doProduce(ctx, client, topic, rec)
}

// doProduce validates the request against topic metadata and sends the record.
// It is separated from ProduceMessage so it can be unit-tested with a fake
// client and producer.
func doProduce(ctx context.Context, client sarama.Client, topic string, rec api.ProduceRecord) error {
	parts, err := client.Partitions(topic)
	if err != nil {
		return api.TopicNotFoundError{TopicName: topic, Cause: err}
	}
	if len(parts) == 0 {
		return api.TopicNotFoundError{TopicName: topic}
	}
	if rec.Partition != nil {
		p := *rec.Partition
		if p < 0 || int(p) >= len(parts) {
			return api.NewPartitionError(
				fmt.Sprintf("partition %d out of range (topic has %d partitions)", p, len(parts)),
				topic, p)
		}
	}

	producer, err := newSyncProducer(client)
	if err != nil {
		return api.ProduceError{Topic: topic, Reason: "cannot create producer", Cause: err}
	}
	defer producer.Close()

	pm := &sarama.ProducerMessage{Topic: topic}
	if rec.Key != nil { // nil key => null record key
		pm.Key = sarama.ByteEncoder(rec.Key)
	}
	if rec.Value != nil { // nil value => null record value (e.g. tombstone)
		pm.Value = sarama.ByteEncoder(rec.Value)
	}
	if rec.Partition != nil {
		pm.Partition = *rec.Partition
	}
	for _, h := range rec.Headers {
		pm.Headers = append(pm.Headers, sarama.RecordHeader{
			Key:   []byte(h.Key),
			Value: []byte(h.Value),
		})
	}

	if _, _, err := producer.SendMessage(pm); err != nil {
		return api.ProduceError{Topic: topic, Reason: "send failed", Cause: err}
	}
	return nil
}
