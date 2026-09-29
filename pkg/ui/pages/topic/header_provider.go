package topic

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// TopicHeaderDataProvider provides header data for the topic page
type TopicHeaderDataProvider struct {
	model *Model
}

func NewTopicHeaderDataProvider(model *Model) *TopicHeaderDataProvider {
	return &TopicHeaderDataProvider{
		model: model,
	}
}

func (t *TopicHeaderDataProvider) GetBrandName() string {
	return "Kafui™"
}

func (t *TopicHeaderDataProvider) GetAppName() string {
	return fmt.Sprintf("Topic: %s", t.model.topicName)
}

func (t *TopicHeaderDataProvider) GetStatusData() map[string]interface{} {
	return map[string]interface{}{
		"time":      t.model.lastUpdate.Format("15:04:05"),
		"status":    t.model.connectionStatus,
		"topic":     t.model.topicName,
		"messages":  len(t.model.messages),
		"consuming": t.model.consuming,
		"paused":    t.model.paused,
		"mode":      t.model.consumeMode.String(),
	}
}

func (t *TopicHeaderDataProvider) HandleHeaderUpdate(msg tea.Msg) tea.Cmd {
	// Handle timer ticks for header updates — only when actively consuming.
	// Stopping the tick when idle prevents timer proliferation: if this
	// handler and the model handler both re-schedule on the same message, the
	// number of pending timers doubles every cycle.
	switch msg := msg.(type) {
	case TimerTickMsg:
		t.model.lastUpdate = time.Time(msg)
		if t.model.consuming {
			return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
				return TimerTickMsg(t)
			})
		}
		// Not consuming — let the timer stop.
		return nil
	}
	return nil
}

func (t *TopicHeaderDataProvider) InitHeader() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return TimerTickMsg(t)
	})
}
