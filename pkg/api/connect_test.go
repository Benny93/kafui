package api

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaskConnectorConfig(t *testing.T) {
	tests := []struct {
		name   string
		in     map[string]string
		masked []string // keys expected to be masked
		kept   []string // keys expected to keep their value
	}{
		{
			name: "secret-like keys masked",
			in: map[string]string{
				"database.password":     "pw",
				"connection.password":   "pw2",
				"aws.secret.access.key": "sk",
				"my.token":              "tk",
				"api.credential":        "cr",
				"sasl.jaas.config":      "cfg",
				"ssl.keystore.location": "/x",
			},
			masked: []string{"database.password", "connection.password", "aws.secret.access.key", "my.token", "api.credential", "sasl.jaas.config", "ssl.keystore.location"},
		},
		{
			name:   "mixed case keys masked",
			in:     map[string]string{"Database.PASSWORD": "pw", "My.Secret": "s"},
			masked: []string{"Database.PASSWORD", "My.Secret"},
		},
		{
			name:   "non-secret untouched",
			in:     map[string]string{"connector.class": "X", "tasks.max": "3", "topics": "t"},
			kept:   []string{"connector.class", "tasks.max", "topics"},
		},
		{
			name:   "empty value untouched",
			in:     map[string]string{"database.password": ""},
			kept:   []string{"database.password"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out := MaskConnectorConfig(tc.in)
			for _, k := range tc.masked {
				assert.Equal(t, ConnectorSecretPlaceholder, out[k], "key %q should be masked", k)
			}
			for _, k := range tc.kept {
				assert.Equal(t, tc.in[k], out[k], "key %q should keep value", k)
			}
		})
	}
}

func TestMaskConnectorConfig_Nil(t *testing.T) {
	assert.Nil(t, MaskConnectorConfig(nil))
}

func TestConnectErrors_MessagesAndUnwrap(t *testing.T) {
	cause := errors.New("root")

	notFoundCluster := ConnectClusterNotFoundError{Connect: "c", Cluster: "k", Cause: cause}
	assert.Contains(t, notFoundCluster.Error(), "c")
	assert.Contains(t, notFoundCluster.Error(), "k")
	assert.ErrorIs(t, notFoundCluster, cause)

	notFound := ConnectorNotFoundError{Connector: "n", Connect: "c", Cause: cause}
	assert.Contains(t, notFound.Error(), "n")
	assert.ErrorIs(t, notFound, cause)

	exists := ConnectorAlreadyExistsError{Connector: "n", Connect: "c", Cause: cause}
	assert.Contains(t, exists.Error(), "already exists")
	assert.ErrorIs(t, exists, cause)

	notStopped := ConnectorNotStoppedError{Connector: "n", Connect: "c", State: "RUNNING", Cause: cause}
	assert.Contains(t, notStopped.Error(), "RUNNING")
	assert.ErrorIs(t, notStopped, cause)

	// errors.As through a wrapped chain
	wrapped := errors.Join(errors.New("outer"), ConnectorNotStoppedError{State: "PAUSED"})
	var target ConnectorNotStoppedError
	assert.True(t, errors.As(wrapped, &target))
	assert.Equal(t, "PAUSED", target.State)
}
