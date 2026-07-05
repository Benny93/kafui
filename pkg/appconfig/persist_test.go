package appconfig

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "config.yaml") // parent dir must be created

	cfg := Default()
	cfg.UI.Theme = "light"
	cfg.DynamicConfigEnabled = true

	require.NoError(t, Save(path, cfg))

	loaded, err := Load(path)
	require.NoError(t, err)
	assert.Equal(t, "light", loaded.UI.Theme)
	assert.True(t, loaded.DynamicConfigEnabled)
}

func TestSaveRejectsDirectory(t *testing.T) {
	dir := t.TempDir()
	err := Save(dir, Default())
	assert.Error(t, err)
}

func TestSaveRejectsUnwritable(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root ignores file permissions")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "ro.yaml")
	require.NoError(t, os.WriteFile(path, []byte("ui:\n  theme: dark\n"), 0o400))
	err := Save(path, Default())
	assert.Error(t, err)
}
