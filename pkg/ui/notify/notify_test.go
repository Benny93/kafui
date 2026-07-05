package notify

import (
	"errors"
	"testing"
	"time"

	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/shared"
	"github.com/Benny93/kafui/pkg/ui/styles"
	"github.com/stretchr/testify/assert"
)

func newMgr(now *time.Time) *Manager {
	m := New(styles.DefaultStyles())
	m.nowFn = func() time.Time { return *now }
	return m
}

func TestDedupConsecutive(t *testing.T) {
	now := time.Unix(0, 0)
	m := newMgr(&now)
	m.Push(core.NotificationMsg{Severity: core.StatusInfo, Message: "same"})
	m.Push(core.NotificationMsg{Severity: core.StatusInfo, Message: "same"})
	assert.Len(t, m.items, 1, "identical consecutive messages dedup")

	m.Push(core.NotificationMsg{Severity: core.StatusInfo, Message: "different"})
	assert.Len(t, m.items, 2)
}

func TestAutoExpiry(t *testing.T) {
	now := time.Unix(0, 0)
	m := newMgr(&now)
	m.Push(core.NotificationMsg{Severity: core.StatusInfo, Message: "hi"})
	assert.False(t, m.Empty())

	// Advance past the default TTL and prune.
	now = now.Add(defaultTTL + time.Second)
	m.prune()
	assert.True(t, m.Empty(), "message should expire after TTL")
}

func TestStickyDoesNotExpire(t *testing.T) {
	now := time.Unix(0, 0)
	m := newMgr(&now)
	m.Push(core.NotificationMsg{Severity: core.StatusError, Message: "boom", Sticky: true})
	now = now.Add(time.Hour)
	m.prune()
	assert.False(t, m.Empty())
}

func TestUIErrorAdaptation(t *testing.T) {
	now := time.Unix(0, 0)
	m := newMgr(&now)
	uiErr := shared.NewUIError(shared.ErrorTypeConnection, "cannot connect", errors.New("refused"))
	_, consumed := m.HandleMsg(uiErr)
	assert.True(t, consumed)
	assert.Len(t, m.items, 1)
	assert.Equal(t, core.StatusError, m.items[0].sev)
}

func TestStatusMsgAdaptation(t *testing.T) {
	now := time.Unix(0, 0)
	m := newMgr(&now)
	_, consumed := m.HandleMsg(core.StatusMsg{Message: "done", Type: core.StatusSuccess})
	assert.True(t, consumed)
	assert.Equal(t, core.StatusSuccess, m.items[0].sev)
}

func TestSeverityStyleDistinct(t *testing.T) {
	m := New(styles.DefaultStyles())
	// Just assert the render is non-empty and includes the message for each severity.
	for _, sev := range []core.StatusType{core.StatusInfo, core.StatusError, core.StatusWarning, core.StatusSuccess} {
		m.Dismiss()
		m.Push(core.NotificationMsg{Severity: sev, Message: "x"})
		assert.Contains(t, m.View(40), "x")
	}
}
