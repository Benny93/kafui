package mainpage

import (
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/stretchr/testify/require"
)

// brokerCountStubDS returns a single-entry bootstrap list from GetClusterDetails
// (as a real ~/.kaf/config typically does) but a full broker list from
// GetBrokers (as cluster metadata does), so a test asserting on the wrong
// source would see 1 instead of 3.
type brokerCountStubDS struct {
	api.KafkaDataSource
}

func (brokerCountStubDS) GetClusterDetails(name string) (api.ClusterInfo, error) {
	return api.ClusterInfo{Name: name, Brokers: []string{"kafka-bootstrap.example:443"}}, nil
}

func (brokerCountStubDS) GetBrokers() ([]api.BrokerInfo, error) {
	return []api.BrokerInfo{{ID: 0}, {ID: 1}, {ID: 2}}, nil
}

func (brokerCountStubDS) GetContext() string { return "vehub-dev-aks" }

// TestSidebarBrokerCountUsesRealBrokerList guards against bug #5 regressing:
// the sidebar must show the cluster's actual broker count (from metadata),
// not the length of the configured bootstrap address list.
func TestSidebarBrokerCountUsesRealBrokerList(t *testing.T) {
	section := NewClusterInfoSection(brokerCountStubDS{})
	cmd := section.RefreshSection()
	require.NotNil(t, cmd)

	msg, ok := cmd().(ClusterInfoMsg)
	require.True(t, ok)
	require.Equal(t, 3, msg.Info["brokers"], "expected the real broker count (3), not the bootstrap list length (1)")
}
