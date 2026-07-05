package shared_test

import (
	"testing"

	"github.com/Benny93/kafui/pkg/ui/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrefsRoundTrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	// Missing file yields zero-value prefs.
	assert.False(t, shared.LoadPrefs().HideInternalTopics)

	require.NoError(t, shared.SavePrefs(shared.Prefs{HideInternalTopics: true}))
	got := shared.LoadPrefs()
	assert.True(t, got.HideInternalTopics)

	require.NoError(t, shared.SavePrefs(shared.Prefs{HideInternalTopics: false}))
	assert.False(t, shared.LoadPrefs().HideInternalTopics)
}
