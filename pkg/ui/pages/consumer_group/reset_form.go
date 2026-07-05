package consumergroup

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/shared"
	stylesPkg "github.com/Benny93/kafui/pkg/ui/styles"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// resetTimestampLayout is the documented datetime input format (local timezone).
const resetTimestampLayout = "2006-01-02 15:04:05"

var resetModes = []api.OffsetResetMode{
	api.OffsetResetEarliest, api.OffsetResetLatest, api.OffsetResetTimestamp, api.OffsetResetExplicit,
}

// resetFocus enumerates the reset form's focus positions.
type resetFocus int

const (
	focusTopic resetFocus = iota
	focusMode
	focusPartitions
	focusConditional // timestamp input (Timestamp) or per-partition offsets (Explicit)
	focusSubmit
)

// resetForm is the bespoke offset-reset form. The shared form component
// (pkg/ui/components/form) is intentionally not reused here: it has no
// multi-select or per-mode conditional inputs, which this form requires.
// ponytail: a bespoke component is the lazy-correct fit rather than extending
// the generic form with multi-select semantics only this page needs.
type resetForm struct {
	groupID string
	styles  *stylesPkg.Styles

	topics       []string
	partsByTopic map[string][]int32

	topicIdx int
	modeIdx  int
	selected map[int32]bool // partition -> selected (for the current topic)

	tsInput      textinput.Model
	offsetInputs map[int32]textinput.Model

	focus      resetFocus
	partCursor int
	errMsg     string

	width, height int
}

// newResetForm builds the form from the group's detail. Returns nil when the
// group has no associated topics (reset is disabled in that case).
func newResetForm(groupID string, detail api.ConsumerGroupDetail, styles *stylesPkg.Styles) *resetForm {
	partsByTopic := map[string]map[int32]struct{}{}
	for _, po := range detail.TopicOffsets {
		if partsByTopic[po.Topic] == nil {
			partsByTopic[po.Topic] = map[int32]struct{}{}
		}
		partsByTopic[po.Topic][po.Partition] = struct{}{}
	}
	if len(partsByTopic) == 0 {
		return nil
	}
	topics := make([]string, 0, len(partsByTopic))
	byTopic := map[string][]int32{}
	for t, set := range partsByTopic {
		topics = append(topics, t)
		ps := make([]int32, 0, len(set))
		for p := range set {
			ps = append(ps, p)
		}
		sort.Slice(ps, func(i, j int) bool { return ps[i] < ps[j] })
		byTopic[t] = ps
	}
	sort.Strings(topics)

	ts := textinput.New()
	ts.Placeholder = resetTimestampLayout
	ts.Width = 30

	f := &resetForm{
		groupID:      groupID,
		styles:       styles,
		topics:       topics,
		partsByTopic: byTopic,
		selected:     map[int32]bool{},
		tsInput:      ts,
		offsetInputs: map[int32]textinput.Model{},
	}
	return f
}

func (f *resetForm) SetDimensions(w, h int) { f.width, f.height = w, h }

func (f *resetForm) currentTopic() string             { return f.topics[f.topicIdx] }
func (f *resetForm) currentMode() api.OffsetResetMode { return resetModes[f.modeIdx] }
func (f *resetForm) currentParts() []int32            { return f.partsByTopic[f.currentTopic()] }

// selectedPartitions returns the sorted selected partitions of the current topic.
func (f *resetForm) selectedPartitions() []int32 {
	var out []int32
	for _, p := range f.currentParts() {
		if f.selected[p] {
			out = append(out, p)
		}
	}
	return out
}

// clearTopicState resets selections and entered offsets when the topic changes.
func (f *resetForm) clearTopicState() {
	f.selected = map[int32]bool{}
	f.offsetInputs = map[int32]textinput.Model{}
	f.partCursor = 0
	f.errMsg = ""
}

func (f *resetForm) hasConditional() bool {
	m := f.currentMode()
	return m == api.OffsetResetTimestamp || m == api.OffsetResetExplicit
}

// Update handles a key message; returns a command and whether it was consumed.
func (f *resetForm) Update(msg tea.Msg) (tea.Cmd, bool) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil, false
	}
	switch key.String() {
	case "esc":
		return func() tea.Msg { return resetFormCancelMsg{} }, true
	case "tab", "down":
		if f.focus == focusPartitions && key.String() == "down" {
			return f.movePartCursor(1), true
		}
		f.moveFocus(1)
		return nil, true
	case "shift+tab", "up":
		if f.focus == focusPartitions && key.String() == "up" {
			return f.movePartCursor(-1), true
		}
		f.moveFocus(-1)
		return nil, true
	case "left":
		return f.adjust(-1), true
	case "right":
		return f.adjust(1), true
	case " ":
		if f.focus == focusPartitions {
			f.togglePart()
			return nil, true
		}
	case "a":
		if f.focus == focusPartitions {
			f.toggleAll()
			return nil, true
		}
	case "enter":
		if f.focus == focusSubmit {
			return f.submit(), true
		}
		f.moveFocus(1)
		return nil, true
	}
	// Typing into the active conditional input.
	if f.focus == focusConditional {
		return f.updateConditionalInput(msg), true
	}
	return nil, true
}

func (f *resetForm) moveFocus(delta int) {
	order := []resetFocus{focusTopic, focusMode, focusPartitions}
	if f.hasConditional() {
		order = append(order, focusConditional)
	}
	order = append(order, focusSubmit)
	// find current index
	cur := 0
	for i, fc := range order {
		if fc == f.focus {
			cur = i
			break
		}
	}
	n := len(order)
	f.focus = order[(cur+delta+n)%n]
	f.syncInputFocus()
}

func (f *resetForm) syncInputFocus() {
	if f.focus == focusConditional && f.currentMode() == api.OffsetResetTimestamp {
		f.tsInput.Focus()
	} else {
		f.tsInput.Blur()
	}
}

// adjust changes the topic or mode selector with left/right.
func (f *resetForm) adjust(delta int) tea.Cmd {
	switch f.focus {
	case focusTopic:
		n := len(f.topics)
		f.topicIdx = (f.topicIdx + delta + n) % n
		f.clearTopicState() // changing the topic clears selections + offsets
	case focusMode:
		n := len(resetModes)
		f.modeIdx = (f.modeIdx + delta + n) % n
		f.errMsg = ""
	}
	return nil
}

func (f *resetForm) movePartCursor(delta int) tea.Cmd {
	parts := f.currentParts()
	if len(parts) == 0 {
		return nil
	}
	f.partCursor = (f.partCursor + delta + len(parts)) % len(parts)
	return nil
}

func (f *resetForm) togglePart() {
	parts := f.currentParts()
	if f.partCursor < 0 || f.partCursor >= len(parts) {
		return
	}
	p := parts[f.partCursor]
	f.selected[p] = !f.selected[p]
}

func (f *resetForm) toggleAll() {
	parts := f.currentParts()
	// If all selected, clear; else select all.
	all := true
	for _, p := range parts {
		if !f.selected[p] {
			all = false
			break
		}
	}
	for _, p := range parts {
		f.selected[p] = !all
	}
}

func (f *resetForm) updateConditionalInput(msg tea.Msg) tea.Cmd {
	if f.currentMode() == api.OffsetResetTimestamp {
		var cmd tea.Cmd
		f.tsInput, cmd = f.tsInput.Update(msg)
		return cmd
	}
	// Explicit: edit the offset for the partition under the cursor.
	sel := f.selectedPartitions()
	if len(sel) == 0 {
		return nil
	}
	idx := f.partCursor
	if idx < 0 || idx >= len(sel) {
		idx = 0
	}
	p := sel[idx]
	in, ok := f.offsetInputs[p]
	if !ok {
		in = textinput.New()
		in.Width = 16
	}
	var cmd tea.Cmd
	in, cmd = in.Update(msg)
	f.offsetInputs[p] = in
	return cmd
}

// submit validates the form and, on success, emits resetFormSubmitMsg with the
// built request. Validation errors are recorded inline and block submission.
func (f *resetForm) submit() tea.Cmd {
	sel := f.selectedPartitions()
	if len(sel) == 0 {
		f.errMsg = "select at least one partition"
		return nil
	}
	req := api.OffsetResetRequest{
		GroupID:    f.groupID,
		Topic:      f.currentTopic(),
		Mode:       f.currentMode(),
		Partitions: sel,
	}
	switch f.currentMode() {
	case api.OffsetResetTimestamp:
		v := strings.TrimSpace(f.tsInput.Value())
		if v == "" {
			f.errMsg = "timestamp is required"
			return nil
		}
		t, err := time.ParseInLocation(resetTimestampLayout, v, time.Local)
		if err != nil {
			f.errMsg = "invalid timestamp (expected " + resetTimestampLayout + ")"
			return nil
		}
		req.Timestamp = &t
	case api.OffsetResetExplicit:
		offsets := map[int32]int64{}
		for _, p := range sel {
			raw := strings.TrimSpace(f.offsetInputs[p].Value())
			if raw == "" {
				f.errMsg = fmt.Sprintf("offset required for partition %d", p)
				return nil
			}
			n, err := strconv.ParseInt(raw, 10, 64)
			if err != nil || n < 0 {
				f.errMsg = fmt.Sprintf("partition %d: offset must be a non-negative number", p)
				return nil
			}
			offsets[p] = n
		}
		req.PartitionOffsets = offsets
	}
	f.errMsg = ""
	return func() tea.Msg { return resetFormSubmitMsg{req: req} }
}

// SetError records an inline error (e.g. a precondition failure from the reset).
func (f *resetForm) SetError(msg string) { f.errMsg = msg }

func (f *resetForm) View() string {
	label := lipgloss.NewStyle().Foreground(stylesPkg.FgBase).Bold(true)
	focused := lipgloss.NewStyle().Foreground(stylesPkg.Primary).Bold(true)
	muted := lipgloss.NewStyle().Foreground(stylesPkg.FgMuted)

	sel := func(fc resetFocus, s string) string {
		if f.focus == fc {
			return focused.Render("▶ " + s)
		}
		return label.Render("  " + s)
	}

	var b strings.Builder
	b.WriteString(sel(focusTopic, "Topic: < "+f.currentTopic()+" >"))
	b.WriteString("\n")
	b.WriteString(sel(focusMode, "Reset type: < "+string(f.currentMode())+" >"))
	b.WriteString("\n\n")

	b.WriteString(sel(focusPartitions, "Partitions (space: toggle, a: all):"))
	b.WriteString("\n")
	parts := f.currentParts()
	for i, p := range parts {
		box := "[ ]"
		if f.selected[p] {
			box = "[x]"
		}
		cursor := "  "
		if f.focus == focusPartitions && i == f.partCursor {
			cursor = "> "
		}
		b.WriteString(fmt.Sprintf("  %s%s %d\n", cursor, box, p))
	}
	b.WriteString("\n")

	if f.hasConditional() {
		if f.currentMode() == api.OffsetResetTimestamp {
			b.WriteString(sel(focusConditional, "Timestamp ("+resetTimestampLayout+"): "))
			b.WriteString(f.tsInput.View())
		} else {
			b.WriteString(sel(focusConditional, "Explicit offsets (per selected partition):"))
			b.WriteString("\n")
			for _, p := range f.selectedPartitions() {
				in := f.offsetInputs[p]
				b.WriteString(fmt.Sprintf("    p%d: %s\n", p, in.View()))
			}
		}
		b.WriteString("\n")
	}

	submitLabel := "Submit"
	if len(f.selectedPartitions()) == 0 {
		submitLabel = "Submit (select a partition)"
	}
	b.WriteString(sel(focusSubmit, submitLabel))
	b.WriteString("\n")
	if f.errMsg != "" {
		b.WriteString(lipgloss.NewStyle().Foreground(stylesPkg.Error).Render(f.errMsg))
		b.WriteString("\n")
	}
	b.WriteString(muted.Render("tab/↑↓: move • ←/→: change • esc: cancel"))
	return b.String()
}

// --- model wiring for the reset flow (CG-19/20) ---

// openResetForm opens the reset form, or notifies when there are no topics.
func (m *Model) openResetForm() tea.Cmd {
	f := newResetForm(m.groupID, m.detail, m.common.Styles)
	if f == nil {
		return core.NewNotification(core.StatusWarning, "Reset offsets", "group has no associated topics")
	}
	f.SetDimensions(m.dims.Width, m.dims.Height)
	m.resetForm = f
	return nil
}

// handleResetSubmit shows a confirmation modal summarizing the reset, then runs
// ResetConsumerGroupOffsets on confirm.
func (m *Model) handleResetSubmit(v resetFormSubmitMsg) tea.Cmd {
	ds := m.common.DataSource
	req := v.req
	summary := fmt.Sprintf("Reset %s offsets of %s-%v for group %q?",
		req.Mode, req.Topic, req.Partitions, req.GroupID)
	return func() tea.Msg {
		return core.ShowConfirmMsg{
			Title:        "Reset offsets",
			Message:      summary,
			Danger:       true,
			ConfirmLabel: "Reset",
			OnConfirm: func() tea.Msg {
				err := ds.ResetConsumerGroupOffsets(context.Background(), req)
				return offsetsResetMsg{groupID: req.GroupID, topic: req.Topic, err: err}
			},
		}
	}
}

// handleOffsetsReset surfaces precondition/validation errors inline in the form;
// on success it closes the form, refreshes the detail, and notifies.
func (m *Model) handleOffsetsReset(v offsetsResetMsg) tea.Cmd {
	if v.err != nil {
		if m.resetForm != nil {
			m.resetForm.SetError(resetErrorText(v.err))
		}
		return func() tea.Msg { return shared.NewUIError("reset-offsets", "Reset offsets failed", v.err) }
	}
	m.resetForm = nil
	return tea.Batch(
		core.NewNotification(core.StatusSuccess, "Offsets reset", v.topic),
		m.loadDetail(),
	)
}

// resetErrorText maps typed reset errors to concise inline messages.
func resetErrorText(err error) string {
	var notEmpty api.GroupNotEmptyError
	if asGroupNotEmpty(err, &notEmpty) {
		return fmt.Sprintf("group is not empty (state: %s)", notEmpty.State)
	}
	var invalid api.InvalidOffsetResetError
	if asInvalidReset(err, &invalid) {
		return invalid.Error()
	}
	var notFound api.GroupNotFoundError
	if asGroupNotFound(err, &notFound) {
		return "consumer group not found"
	}
	return err.Error()
}

func asGroupNotEmpty(err error, dst *api.GroupNotEmptyError) bool {
	for err != nil {
		if e, ok := err.(api.GroupNotEmptyError); ok {
			*dst = e
			return true
		}
		u, ok := err.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
	return false
}

func asInvalidReset(err error, dst *api.InvalidOffsetResetError) bool {
	for err != nil {
		if e, ok := err.(api.InvalidOffsetResetError); ok {
			*dst = e
			return true
		}
		u, ok := err.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
	return false
}
