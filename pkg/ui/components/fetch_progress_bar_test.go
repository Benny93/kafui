package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFetchProgressBar_IgnoresSupersededChannel(t *testing.T) {
	old := NewProgressChannel(40)
	cur := NewProgressChannel(10)
	bar := NewFetchProgressBar()
	bar.StartListening(old, 40)
	bar.StartListening(cur, 10)

	old <- ProgressMsg{Current: 40, Total: 40, Done: true}
	bar, cmd := bar.Update(ListenForProgress(old)())
	assert.Nil(t, cmd)
	assert.True(t, bar.IsActive(), "a superseded Done must not end the current operation")

	cur <- ProgressMsg{Current: 3, Total: 10}
	bar, cmd = bar.Update(ListenForProgress(cur)())
	assert.NotNil(t, cmd, "progress from the tracked channel re-arms the listener")
	assert.Equal(t, 3, bar.Current())

	// A message built directly, not read through ListenForProgress, is applied.
	bar, _ = bar.Update(ProgressMsg{Done: true})
	assert.False(t, bar.IsActive())
}
