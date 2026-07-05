package broker

import "github.com/Benny93/kafui/pkg/api"

// tab identifies the active detail tab.
type tab int

const (
	tabLogDirs tab = iota
	tabConfigs
	tabMetrics
)

func (t tab) String() string {
	switch t {
	case tabConfigs:
		return "Configs"
	case tabMetrics:
		return "Metrics"
	default:
		return "Log Dirs"
	}
}

// tabTitles is the ordered tab bar.
var tabTitles = []tab{tabLogDirs, tabConfigs, tabMetrics}

// Async load result messages. Each carries the broker ID it was requested for so
// stale responses (after navigation) can be discarded.
type (
	brokerInfoLoadedMsg struct {
		brokerID int32
		info     api.BrokerInfo
		found    bool
		err      error
	}

	brokerStatsLoadedMsg struct {
		brokerID int32
		stats    api.BrokerStats
		ok       bool
	}

	logDirsLoadedMsg struct {
		brokerID int32
		dirs     []api.BrokerLogDir
		err      error
	}

	configsLoadedMsg struct {
		brokerID int32
		entries  []api.BrokerConfigEntry
		err      error
	}

	metricsLoadedMsg struct {
		brokerID int32
		data     string
		err      error
	}

	// configAlteredMsg is dispatched after a confirmed config change attempt.
	configAlteredMsg struct {
		brokerID int32
		key      string
		value    string
		err      error
	}

	// replicaMovedMsg is dispatched after a confirmed replica log-dir move.
	replicaMovedMsg struct {
		brokerID  int32
		topic     string
		partition int32
		logDir    string
		err       error
	}
)
