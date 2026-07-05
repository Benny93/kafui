package appconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadKsqlEndpoint_FullSection(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.yaml")
	yaml := "clusters:\n" +
		"  kafka-dev:\n" +
		"    ksql:\n" +
		"      url: http://ksql:8088\n" +
		"      username: alice\n" +
		"      password: s3cr3t\n" +
		"      tlsCaPath: /etc/ca.pem\n" +
		"      maxResponseBytes: 1048576\n"
	require.NoError(t, os.WriteFile(p, []byte(yaml), 0o600))

	cfg, err := Load(p)
	require.NoError(t, err)
	ep := cfg.Clusters["kafka-dev"].Ksql
	require.NotNil(t, ep)
	assert.Equal(t, "http://ksql:8088", ep.URL)
	assert.Equal(t, "alice", ep.Username)
	assert.Equal(t, "s3cr3t", ep.Password)
	assert.Equal(t, "/etc/ca.pem", ep.TLSCAPath)
	assert.Equal(t, int64(1048576), ep.MaxResponseBytes)
}

func TestLoadKsqlEndpoint_Absent(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(p, []byte("clusters:\n  kafka-dev:\n    readOnly: true\n"), 0o600))
	cfg, err := Load(p)
	require.NoError(t, err)
	assert.Nil(t, cfg.Clusters["kafka-dev"].Ksql)
}

func TestKsqlEndpoint_StringRedactsPassword(t *testing.T) {
	ep := KsqlEndpoint{URL: "http://ksql:8088", Username: "alice", Password: "s3cr3t"}
	s := ep.String()
	assert.NotContains(t, s, "s3cr3t")
	assert.Contains(t, s, "********")
	assert.Contains(t, s, "http://ksql:8088")

	// Empty password is not masked (no secret to hide).
	assert.NotContains(t, KsqlEndpoint{URL: "u"}.String(), "********")
	_ = strings.TrimSpace(s)
}
