package mainpage

import (
	"testing"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// slowTopicDetailsDS implements KafkaDataSource by embedding a nil interface
// and overriding only GetTopicDetails/GetTopicSizes, each sleeping delay to
// simulate a remote cluster where every call opens its own connection.
type slowTopicDetailsDS struct {
	api.KafkaDataSource
	delay time.Duration
}

func (s slowTopicDetailsDS) GetTopicDetails(topicName string) (api.TopicDetails, error) {
	time.Sleep(s.delay)
	return api.TopicDetails{}, nil
}

func (s slowTopicDetailsDS) GetTopicSizes(topicNames []string) (map[string]int64, error) {
	return map[string]int64{}, nil
}

// TestLoadTopicDetailsExtFetchesPageConcurrently guards against bug #4
// regressing: fetching a page of N topics must take roughly one call's
// latency, not N times that — otherwise a remote cluster where each
// GetTopicDetails call is slow makes the OSR/Size columns look permanently
// stuck behind a "…" placeholder (BUG-4).
func TestLoadTopicDetailsExtFetchesPageConcurrently(t *testing.T) {
	const n = 10
	const delay = 100 * time.Millisecond
	ds := slowTopicDetailsDS{delay: delay}

	provider := NewKafuiContentProvider(ds)
	items := make([]interface{}, 0, n)
	for i := 0; i < n; i++ {
		items = append(items, shared.ResourceListItem{ResourceItem: &TopicResourceItem{
			id: string(rune('a' + i)), outOfSync: -1, size: -1,
		}})
	}
	provider.allItems = items
	provider.pagination.SetTotalItems(len(items))

	cmd := provider.loadTopicDetailsExt()
	require.NotNil(t, cmd)

	start := time.Now()
	msg := cmd()
	elapsed := time.Since(start)

	out, ok := msg.(TopicDetailsExtLoadedMsg)
	require.True(t, ok)
	assert.Len(t, out, n)
	assert.Less(t, elapsed, time.Duration(n)*delay/2,
		"fetching %d topics took %s — looks sequential, not concurrent", n, elapsed)
}
