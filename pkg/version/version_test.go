package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShortCommit(t *testing.T) {
	orig := Commit
	defer func() { Commit = orig }()

	Commit = "abcdef1234567890"
	assert.Equal(t, "abcdef1", ShortCommit())

	Commit = "short"
	assert.Equal(t, "short", ShortCommit())
}

func TestGetIncludesFeatures(t *testing.T) {
	info := Get("dynamic-config", "mock")
	assert.Contains(t, info.Features, "dynamic-config")
	assert.Contains(t, info.Features, "mock")
	assert.Contains(t, info.String(), "kafui")
	assert.NotEmpty(t, info.GoVersion)
	assert.Contains(t, info.Platform, "/")
}
