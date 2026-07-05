package connector

import "github.com/Benny93/kafui/pkg/api"

// tab identifies the active detail tab.
type tab int

const (
	tabOverview tab = iota
	tabTasks
	tabConfig
	tabTopics
)

func (t tab) String() string {
	switch t {
	case tabTasks:
		return "Tasks"
	case tabConfig:
		return "Config"
	case tabTopics:
		return "Topics"
	default:
		return "Overview"
	}
}

// tabTitles is the ordered tab bar.
var tabTitles = []tab{tabOverview, tabTasks, tabConfig, tabTopics}

// Async load / action result messages. Each carries the connect+name it was
// requested for so stale responses (after navigation) can be discarded.
type (
	detailsLoadedMsg struct {
		connect string
		name    string
		details api.ConnectorDetails
		found   bool
		err     error
	}

	// lifecycleResultMsg reports the outcome of a confirmed lifecycle action
	// (pause/resume/stop/restart/delete/reset-offsets).
	lifecycleResultMsg struct {
		action  string
		deleted bool // delete succeeded → navigate back
		err     error
	}

	// taskRestartResultMsg reports the outcome of a task-restart batch. failures
	// lists per-task error strings; total is the number attempted.
	taskRestartResultMsg struct {
		total    int
		failures []string
		err      error
	}

	// configUpdatedMsg reports the outcome of a confirmed config update.
	configUpdatedMsg struct {
		err error
	}
)
