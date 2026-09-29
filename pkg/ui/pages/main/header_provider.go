package mainpage

import (
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/core"
	tea "github.com/charmbracelet/bubbletea"
)

// KafuiHeaderDataProvider provides header data for Kafui
// Implements providers.HeaderDataProvider interface
type KafuiHeaderDataProvider struct {
	dataSource api.KafkaDataSource
	common     *core.Common
	lastUpdate time.Time
	// tickGen identifies the live 5s tick chain. Each InitHeader starts a new
	// generation, and ticks from an older chain are dropped instead of
	// re-arming, so at most one chain runs (PERF-5).
	tickGen uint64
}

// headerTickMsg is the header's private timer tick, tagged with the chain
// generation that armed it. A current one is re-broadcast as TimerTickMsg.
type headerTickMsg struct {
	gen uint64
	at  time.Time
}

// tick arms the next 5s tick of the current chain.
func (k *KafuiHeaderDataProvider) tick() tea.Cmd {
	gen := k.tickGen
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return headerTickMsg{gen: gen, at: t}
	})
}

func NewKafuiHeaderDataProvider(dataSource api.KafkaDataSource) *KafuiHeaderDataProvider {
	return &KafuiHeaderDataProvider{
		dataSource: dataSource,
		lastUpdate: time.Now(),
	}
}

func (k *KafuiHeaderDataProvider) GetBrandName() string {
	return "Kafui™"
}

func (k *KafuiHeaderDataProvider) GetAppName() string {
	return "Kafka TUI"
}

func (k *KafuiHeaderDataProvider) GetStatusData() map[string]interface{} {
	context := k.dataSource.GetContext()
	data := map[string]interface{}{
		"time":    k.lastUpdate.Format("15:04:05"),
		"status":  "connected",
		"context": context,
		"cluster": "kafka-cluster",
	}
	// Identity, active profile and read-only badge (AA-11). Identity + profile
	// are only shown when authorization is enabled: with authz off (the default
	// single-user mode) the identity is just the OS username, which carries no
	// authorization meaning and is PII we shouldn't surface in the header.
	if k.common != nil {
		if k.common.AuthzEnabled() {
			if k.common.Identity != "" {
				data["identity"] = k.common.Identity
			}
			data["profile"] = k.common.ActiveProfileName()
		}
		data["readonly"] = k.common.IsReadOnly()
	}
	return data
}

func (k *KafuiHeaderDataProvider) HandleHeaderUpdate(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case headerTickMsg:
		if msg.gen != k.tickGen {
			return nil // superseded chain: let it die
		}
		at := msg.at
		return tea.Batch(func() tea.Msg { return TimerTickMsg(at) }, k.tick())
	case TimerTickMsg:
		k.lastUpdate = time.Time(msg)
	}
	return nil
}

func (k *KafuiHeaderDataProvider) InitHeader() tea.Cmd {
	k.tickGen++
	return k.tick()
}

// NewKafuiHeaderDataProviderWithCommon creates a header provider using Common context
func NewKafuiHeaderDataProviderWithCommon(common *core.Common) *KafuiHeaderDataProvider {
	p := NewKafuiHeaderDataProvider(common.DataSource)
	p.common = common
	return p
}
