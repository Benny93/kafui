package topic

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Benny93/kafui/pkg/authz"
	"github.com/Benny93/kafui/pkg/ui/components/menu"
	"github.com/Benny93/kafui/pkg/ui/keys"
)

// The topic screen used to claim twelve bare letters — C o s E t + F S # P Y L
// X — none of which appeared in its key map, so none appeared in the help
// overlay or the hint bar. Several also collided with shell chords. They live
// here now: named, permission-aware, and reachable by `a` or right-click.

// KeyScope implements core.KeyScoper.
func (m *Model) KeyScope() keys.Scope { return keys.ScopeTopic }

// Unwind implements core.Unwinder: esc closes whatever this screen has open,
// one level at a time, before the shell navigates back to the topic list.
func (m *Model) Unwind() (tea.Cmd, bool) {
	switch {
	case m.searchMode:
		m.searchMode = false
		m.searchInput.Blur()
		m.searchInput.SetValue("")
		m.smartFilter = nil
		m.FilterMessages()
		return nil, true
	case m.showGroups:
		m.showGroups = false
	case m.showOverview:
		m.showOverview = false
	case m.showSettings:
		m.showSettings = false
	case m.showAnalysis:
		m.showAnalysis = false
	case m.showSettingsEdit:
		m.showSettingsEdit = false
	case m.showMutationForm:
		m.showMutationForm = false
	case m.showSeek:
		m.showSeek = false
	case m.showPartitions:
		m.showPartitions = false
	case m.showProduce:
		m.showProduce = false
	case m.showProjections:
		m.showProjections = false
	case m.showSavedFilters:
		m.showSavedFilters = false
	case m.searchInput.Value() != "":
		m.searchInput.SetValue("")
		m.smartFilter = nil
		m.FilterMessages()
	default:
		return nil, false
	}
	m.markRenderDirty()
	return nil, true
}

// ContextActions implements core.ActionProvider.
func (m *Model) ContextActions() []menu.Entry {
	keyOf := func(a keys.Action) string { return keys.Default.KeyFor(a) }

	gated := func(e menu.Entry, action authz.Action, rt authz.ResourceType, name string) menu.Entry {
		if m.common != nil && !m.common.Can(action, rt, name) {
			e.Disabled = true
			e.Reason = "not permitted by the active profile or the cluster is read-only"
		}
		return e
	}

	topic := m.topicName
	k := m.keys

	return []menu.Entry{
		// Message-level.
		{Label: "Open message", Key: keyOf(keys.ActionActivate), Run: func() tea.Cmd { return k.handleSelect(m) }},
		{Label: "Copy message value", Key: keyOf(keys.ActionCopy), Run: func() tea.Cmd { return k.handleCopyValue(m) }},
		{Label: "Copy message key", Run: func() tea.Cmd { return k.handleCopyKey(m) }},
		{Label: "Pause / resume consumption", Key: keyOf(keys.ActionPause), Run: func() tea.Cmd { return k.handlePauseResume(m) }},
		{Label: "Change seek mode…", Run: func() tea.Cmd { return k.handleShowSeek(m) }},
		{Label: "Choose partitions and serde…", Run: func() tea.Cmd { return k.handleShowPartitions(m) }},
		{Label: "Saved filters…", Run: func() tea.Cmd { return k.handleShowSavedFilters(m) }},
		{Label: "Field projections…", Run: func() tea.Cmd { return k.handleShowProjections(m) }},
		{Label: "Cycle payload format", Key: keyOf(keys.ActionFormat), Run: func() tea.Cmd { return k.handleFormat(m) }},
		{Label: "Show / hide metadata", Key: keyOf(keys.ActionMetadata), Run: func() tea.Cmd { return k.handleMetadata(m) }},

		// Topic-level.
		{Label: "Topic overview", Run: func() tea.Cmd { return k.handleShowOverview(m) }},
		{Label: "Topic settings", Run: func() tea.Cmd { return k.handleShowSettings(m) }},
		{Label: "Topic statistics", Run: func() tea.Cmd { return k.handleShowAnalysis(m) }},
		{Label: "Consumer groups on this topic", Run: func() tea.Cmd { return k.handleShowGroups(m) }},

		// Mutating. All confirmed downstream; the guard is the backstop.
		gated(menu.Entry{Label: "Produce a message…", Key: keyOf(keys.ActionNew), Run: func() tea.Cmd { return k.handleShowProduce(m) }},
			authz.ActionProduceMessages, authz.ResourceTopic, topic),
		gated(menu.Entry{Label: "Reproduce selected message", Run: func() tea.Cmd { return k.handleReproduce(m) }},
			authz.ActionProduceMessages, authz.ResourceTopic, topic),
		gated(menu.Entry{Label: "Edit topic settings…", Run: func() tea.Cmd { return k.handleShowSettingsEdit(m) }},
			authz.ActionEdit, authz.ResourceTopic, topic),
		gated(menu.Entry{Label: "Add partitions…", Run: func() tea.Cmd { return k.handleIncreasePartitionsDialog(m) }},
			authz.ActionEdit, authz.ResourceTopic, topic),
		gated(menu.Entry{Label: "Change replication factor…", Run: func() tea.Cmd { return k.handleReplicationFactorDialog(m) }},
			authz.ActionEdit, authz.ResourceTopic, topic),
		gated(menu.Entry{Label: "Purge all messages", Destructive: true, Run: func() tea.Cmd { return k.handleClearAllMessages(m) }},
			authz.ActionDeleteMessages, authz.ResourceTopic, topic),
		gated(menu.Entry{Label: "Recreate topic", Destructive: true, Run: func() tea.Cmd { return k.handleRecreateTopic(m) }},
			authz.ActionDelete, authz.ResourceTopic, topic),
		gated(menu.Entry{Label: "Delete topic", Key: keyOf(keys.ActionDelete), Destructive: true,
			Run: func() tea.Cmd { return k.handleDeleteTopic(m) }}, authz.ActionDelete, authz.ResourceTopic, topic),

		{Label: "Refresh", Key: keyOf(keys.ActionRefresh), Run: func() tea.Cmd { return k.handleRefresh(m) }},
	}
}

// The router holds TopicPageModel, not the inner Model, so the wrapper forwards
// the controls-spec interfaces.

// ContextActions implements core.ActionProvider.
func (p *TopicPageModel) ContextActions() []menu.Entry {
	if p.topicModel == nil {
		return nil
	}
	return p.topicModel.ContextActions()
}

// Unwind implements core.Unwinder.
func (p *TopicPageModel) Unwind() (tea.Cmd, bool) {
	if p.topicModel == nil {
		return nil, false
	}
	return p.topicModel.Unwind()
}

// KeyScope implements core.KeyScoper.
func (p *TopicPageModel) KeyScope() keys.Scope { return keys.ScopeTopic }
