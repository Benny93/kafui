package shared

import (
	"bytes"
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func i64(v int64) *int64 { return &v }

func TestWriteConsumerGroupCSV(t *testing.T) {
	groups := []api.ConsumerGroup{
		{Name: "beta", State: api.GroupStateStable, MemberCount: 3, TopicCount: 2, Lag: i64(42), CoordinatorID: 1},
		// Undefined lag → empty cell; unknown coordinator (<0) → empty.
		{Name: "alpha", State: api.GroupStateEmpty, MemberCount: 0, TopicCount: 1, Lag: nil, CoordinatorID: -1},
	}
	var buf bytes.Buffer
	require.NoError(t, WriteConsumerGroupCSV(&buf, groups))

	golden := "Group ID,Members,Topics,Lag,Coordinator,State\n" +
		"alpha,0,1,,,Empty\n" + // sorted by name; nil lag & unknown coord → empty
		"beta,3,2,42,1,Stable\n"
	assert.Equal(t, golden, buf.String())
}
