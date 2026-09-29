package kafds

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/IBM/sarama"
)

// offsetResetter resolves and commits target offsets for a group reset. It is a
// small seam so tests can fake the broker interaction (Sarama has no admin-side
// offset-commit API; the real implementation joins the group via an
// OffsetManager).
type offsetResetter interface {
	GetOffset(topic string, partition int32, time int64) (int64, error)
	Partitions(topic string) ([]int32, error)
	// Commit sets the group's committed offset for each partition of the topic.
	Commit(groupID, topic string, offsets map[int32]int64) error
	Close() error
}

// newOffsetResetter builds a real resetter backed by a Sarama client and an
// OffsetManager joined as the group. Replaceable in tests.
var newOffsetResetter = func(groupID string) (offsetResetter, error) {
	cfg, err := getConfig()
	if err != nil {
		return nil, err
	}
	configureOffsetReset(cfg)
	client, err := getClientFromConfig(cfg)
	if err != nil {
		return nil, err
	}
	return newSaramaOffsetResetter(groupID, client)
}

// configureOffsetReset makes the OffsetManager commit only when asked (no
// background ticker racing Commit) and report commit failures on each
// partition's Errors channel instead of just logging them.
func configureOffsetReset(cfg *sarama.Config) {
	cfg.Consumer.Offsets.AutoCommit.Enable = false
	cfg.Consumer.Return.Errors = true
}

// newSaramaOffsetResetter takes ownership of client, which must have been
// built from a config passed through configureOffsetReset.
func newSaramaOffsetResetter(groupID string, client sarama.Client) (*saramaOffsetResetter, error) {
	om, err := sarama.NewOffsetManagerFromClient(groupID, client)
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("creating offset manager for group %q: %w", groupID, err)
	}
	cfg := client.Config()
	return &saramaOffsetResetter{
		client:  client,
		om:      om,
		retries: cfg.Consumer.Offsets.Retry.Max,
		backoff: cfg.Metadata.Retry.Backoff,
	}, nil
}

type saramaOffsetResetter struct {
	client  sarama.Client
	om      sarama.OffsetManager
	retries int           // extra commit attempts for partitions not yet acknowledged
	backoff time.Duration // pause between commit attempts
}

func (r *saramaOffsetResetter) GetOffset(topic string, partition int32, t int64) (int64, error) {
	return r.client.GetOffset(topic, partition, t)
}

func (r *saramaOffsetResetter) Partitions(topic string) ([]int32, error) {
	return r.client.Partitions(topic)
}

func (r *saramaOffsetResetter) Commit(groupID, topic string, offsets map[int32]int64) error {
	pending := make(map[int32]sarama.PartitionOffsetManager, len(offsets))
	for p, off := range offsets {
		pom, err := r.om.ManagePartition(topic, p)
		if err != nil {
			// Poms already managed are force-released by om.Close in Close.
			return fmt.Errorf("managing partition %s/%d: %w", topic, p, err)
		}
		// MarkOffset only moves the offset forward and ResetOffset only moves
		// it back (or keeps it), so apply both to land on off whatever the
		// currently committed offset is, including "none committed" (-1).
		pom.MarkOffset(off, "")
		pom.ResetOffset(off, "")
		// Mark it closed now: the OffsetManager releases a closed pom (closing
		// its Errors channel) only once its offset is committed.
		pom.AsyncClose()
		pending[p] = pom
	}

	lastErr := make(map[int32]error)
	for attempt := 0; attempt <= r.retries && len(pending) > 0; attempt++ {
		if attempt > 0 {
			time.Sleep(r.backoff)
		}
		r.om.Commit()
		for p, pom := range pending {
			released, err := drainPOMErrors(pom)
			if err != nil {
				lastErr[p] = err
			}
			if released {
				delete(pending, p)
				delete(lastErr, p)
			}
		}
	}
	if len(pending) == 0 {
		return nil
	}

	errs := make([]error, 0, len(pending))
	for p := range pending {
		err := lastErr[p]
		if err == nil {
			err = errors.New("commit not acknowledged by the group coordinator")
		}
		errs = append(errs, fmt.Errorf("partition %s/%d: %w", topic, p, err))
	}
	return errors.Join(errs...)
}

// drainPOMErrors reads the errors already queued on pom without blocking. It
// reports whether the pom was released (its channel closed), which the
// OffsetManager only does once the pom's offset has been committed.
func drainPOMErrors(pom sarama.PartitionOffsetManager) (released bool, last error) {
	for {
		select {
		case e, ok := <-pom.Errors():
			if !ok {
				return true, last
			}
			last = e
		default:
			return false, last
		}
	}
}

func (r *saramaOffsetResetter) Close() error {
	_ = r.om.Close()
	return r.client.Close()
}

// ResetConsumerGroupOffsets implements api.KafkaDataSource (CG-7, CG-8).
func (kp KafkaDataSourceKaf) ResetConsumerGroupOffsets(ctx context.Context, req api.OffsetResetRequest) error {
	if err := validateOffsetResetRequest(req); err != nil {
		return err
	}

	// Precondition: the group must exist and be inactive (Empty or Dead).
	admin, err := getClusterAdmin()
	if err != nil {
		return err
	}
	// The admin is datasource-owned and cached; don't close it.
	names, err := admin.ListConsumerGroups()
	if err != nil {
		return fmt.Errorf("listing consumer groups: %w", err)
	}
	if _, ok := names[req.GroupID]; !ok {
		return api.GroupNotFoundError{GroupID: req.GroupID}
	}
	descs, err := admin.DescribeConsumerGroups([]string{req.GroupID})
	if err != nil {
		return fmt.Errorf("describing consumer group %q: %w", req.GroupID, err)
	}
	desc := findGroupDesc(descs, req.GroupID)
	if desc == nil {
		return api.GroupNotFoundError{GroupID: req.GroupID}
	}
	state := normalizeGroupState(desc.State)
	if state != api.GroupStateEmpty && state != api.GroupStateDead {
		return api.GroupNotEmptyError{GroupID: req.GroupID, State: state}
	}

	resetter, err := newOffsetResetter(req.GroupID)
	if err != nil {
		return err
	}
	defer resetter.Close()

	// Resolve target partitions.
	partitions := req.Partitions
	if len(partitions) == 0 {
		partitions, err = resetter.Partitions(req.Topic)
		if err != nil {
			return fmt.Errorf("listing partitions for topic %q: %w", req.Topic, err)
		}
	}

	offsets := make(map[int32]int64, len(partitions))
	for _, p := range partitions {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		target, err := resolveResetOffset(req, resetter, p)
		if err != nil {
			return err
		}
		offsets[p] = target
	}

	invalidateGroupCache(req.GroupID)
	if err := resetter.Commit(req.GroupID, req.Topic, offsets); err != nil {
		return fmt.Errorf("committing reset offsets for group %q: %w", req.GroupID, err)
	}
	return nil
}

// validateOffsetResetRequest performs shared validation before any broker call.
func validateOffsetResetRequest(req api.OffsetResetRequest) error {
	switch req.Mode {
	case api.OffsetResetEarliest, api.OffsetResetLatest:
	case api.OffsetResetTimestamp:
		if req.Timestamp == nil {
			return api.InvalidOffsetResetError{Reason: "timestamp mode requires a timestamp"}
		}
	case api.OffsetResetExplicit:
		if len(req.PartitionOffsets) == 0 {
			return api.InvalidOffsetResetError{Reason: "explicit mode requires per-partition offsets"}
		}
	default:
		return api.InvalidOffsetResetError{Reason: fmt.Sprintf("unrecognized reset mode %q", req.Mode)}
	}
	return nil
}

// resolveResetOffset computes the target offset for a single partition based on
// the reset mode (CG-7 earliest/latest, CG-8 timestamp/explicit with clamping).
func resolveResetOffset(req api.OffsetResetRequest, r offsetResetter, partition int32) (int64, error) {
	switch req.Mode {
	case api.OffsetResetEarliest:
		return r.GetOffset(req.Topic, partition, sarama.OffsetOldest)
	case api.OffsetResetLatest:
		return r.GetOffset(req.Topic, partition, sarama.OffsetNewest)
	case api.OffsetResetTimestamp:
		off, err := r.GetOffset(req.Topic, partition, req.Timestamp.UnixMilli())
		if err != nil {
			return 0, err
		}
		if off < 0 {
			// No record at/after the timestamp — fall back to the end offset.
			return r.GetOffset(req.Topic, partition, sarama.OffsetNewest)
		}
		return off, nil
	case api.OffsetResetExplicit:
		requested := req.PartitionOffsets[partition] // missing => 0
		return clampOffset(req.Topic, partition, requested, r)
	default:
		return 0, api.InvalidOffsetResetError{Reason: fmt.Sprintf("unrecognized reset mode %q", req.Mode)}
	}
}

// clampOffset clamps a requested offset into [oldest, newest].
func clampOffset(topic string, partition int32, requested int64, r offsetResetter) (int64, error) {
	oldest, err := r.GetOffset(topic, partition, sarama.OffsetOldest)
	if err != nil {
		return 0, err
	}
	newest, err := r.GetOffset(topic, partition, sarama.OffsetNewest)
	if err != nil {
		return 0, err
	}
	if requested < oldest {
		return oldest, nil
	}
	if requested > newest {
		return newest, nil
	}
	return requested, nil
}
