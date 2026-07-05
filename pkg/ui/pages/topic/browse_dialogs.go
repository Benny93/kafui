package topic

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/masking"
	"github.com/Benny93/kafui/pkg/serde"
	formpkg "github.com/Benny93/kafui/pkg/ui/components/form"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/shared"
	stylesPkg "github.com/Benny93/kafui/pkg/ui/styles"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// seekPageSize is the number of messages a seek/partition fetch requests.
const seekPageSize = 100

// seekModeOptions lists the seek modes offered in the dialog, in cycle order.
var seekModeOptions = []string{
	string(api.SeekNewest),
	string(api.SeekOldest),
	string(api.SeekLive),
	string(api.SeekFromOffset),
	string(api.SeekToOffset),
	string(api.SeekFromTimestamp),
	string(api.SeekToTimestamp),
}

// serdeOptions returns the serde choices for the selector: "auto" plus the
// datasource's registry names (MSG-18/22).
func (m *Model) serdeOptions() []string {
	opts := []string{serde.Auto}
	if m.dataSource != nil {
		opts = append(opts, m.dataSource.ListSerdes()...)
	}
	return opts
}

// applySerdeConfig rebuilds the serde registry with the active cluster's
// configured serdes and pre-selects any topic-bound key/value serde (MSG-17).
func (m *Model) applySerdeConfig(common *core.Common) {
	if common == nil || common.AppConfig == nil || common.DataSource == nil {
		return
	}
	// Avoid touching the datasource when no cluster config exists at all.
	if len(common.AppConfig.Clusters) == 0 {
		return
	}
	ext, ok := common.AppConfig.Clusters[common.DataSource.GetContext()]
	if !ok {
		return
	}
	if reg, err := serde.BuildRegistry(nil, ext.Serdes); err == nil {
		m.serdeReg = reg
	}
	if name := serde.SelectSerde(ext.Serdes, m.topicName, true); name != "" {
		m.keySerde = name
	}
	if name := serde.SelectSerde(ext.Serdes, m.topicName, false); name != "" {
		m.valueSerde = name
	}
}

// parseSeekTime parses an absolute RFC3339 timestamp or a relative duration such
// as "-1h" / "-30m" (relative to now).
func parseSeekTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("timestamp is required")
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	if d, err := time.ParseDuration(s); err == nil {
		return time.Now().Add(d), nil
	}
	return time.Time{}, fmt.Errorf("invalid timestamp %q (use RFC3339 like 2006-01-02T15:04:05Z or a relative offset like -1h)", s)
}

// buildSeekFlags builds ConsumeFlags for the chosen seek mode and value.
// value carries the offset (integer) or timestamp (RFC3339/relative) for the
// modes that need one. It returns a validated set of flags (MSG-21).
func buildSeekFlags(mode, value string, count int, partitions []int32) (api.ConsumeFlags, error) {
	f := api.ConsumeFlags{
		LimitMessages: int64(count),
		Partitions:    partitions,
	}
	switch api.SeekMode(mode) {
	case api.SeekNewest, "":
		f.Seek = api.SeekNewest
		f.Follow = false
		f.Tail = int32(count)
		f.OffsetFlag = "latest"
	case api.SeekOldest:
		f.Seek = api.SeekOldest
		f.Follow = false
		f.OffsetFlag = "oldest"
	case api.SeekLive:
		f.Seek = api.SeekLive
		f.Follow = true
		f.OffsetFlag = "latest"
		f.LimitMessages = 0 // tailing applies no page limit
	case api.SeekFromOffset, api.SeekToOffset:
		n, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		if err != nil {
			return f, fmt.Errorf("offset must be an integer: %q", value)
		}
		f.Seek = api.SeekMode(mode)
		f.SeekOffset = &n
		f.OffsetFlag = strconv.FormatInt(n, 10) // legacy numeric offset path
	case api.SeekFromTimestamp, api.SeekToTimestamp:
		t, err := parseSeekTime(value)
		if err != nil {
			return f, err
		}
		f.Seek = api.SeekMode(mode)
		f.SeekTimestamp = &t
		f.OffsetFlag = "latest"
	default:
		return f, fmt.Errorf("unknown seek mode %q", mode)
	}
	if err := f.Validate(); err != nil {
		return f, err
	}
	return f, nil
}

// buildPartitionFilter parses a comma-separated partition list into a validated
// []int32. Empty input (or "all") returns nil meaning "all partitions" (MSG-22).
func buildPartitionFilter(input string, numPartitions int32) ([]int32, error) {
	input = strings.TrimSpace(input)
	if input == "" || strings.EqualFold(input, "all") {
		return nil, nil
	}
	var out []int32
	for _, tok := range strings.Split(input, ",") {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			continue
		}
		n, err := strconv.Atoi(tok)
		if err != nil {
			return nil, fmt.Errorf("invalid partition %q", tok)
		}
		if n < 0 || (numPartitions > 0 && int32(n) >= numPartitions) {
			return nil, fmt.Errorf("partition %d out of range (topic has %d partitions)", n, numPartitions)
		}
		out = append(out, int32(n))
	}
	return out, nil
}

// buildMaskerFromConfig builds a display-time masker from the active cluster's
// masking rules, or nil when none are configured / on parse error (MSG-28).
func buildMaskerFromConfig(common *core.Common) *masking.Masker {
	if common == nil || common.AppConfig == nil || common.DataSource == nil {
		return nil
	}
	// Avoid touching the datasource when no cluster masking is configured at all.
	if len(common.AppConfig.Clusters) == 0 {
		return nil
	}
	ext, ok := common.AppConfig.Clusters[common.DataSource.GetContext()]
	if !ok || len(ext.Masking) == 0 {
		return nil
	}
	rules, err := masking.ParseRules(ext.Masking)
	if err != nil {
		shared.Log.Warn("invalid masking rules", "err", err)
		return nil
	}
	m, err := masking.New(rules)
	if err != nil {
		shared.Log.Warn("invalid masking rules", "err", err)
		return nil
	}
	return m
}

// --- MSG-21: seek dialog ---

func (k *Keys) handleShowSeek(model *Model) tea.Cmd {
	model.seekForm = formpkg.New([]formpkg.Field{
		{Name: "mode", Label: "Seek mode", Type: formpkg.Select, Options: seekModeOptions, Default: string(model.consumeFlags.Seek)},
		{Name: "value", Label: "Offset or timestamp (RFC3339 / -1h)", Type: formpkg.Text},
	})
	model.showSeek = true
	if model.dimensions.Width > 0 {
		model.seekForm.SetDimensions(model.dimensions.Width-4, model.dimensions.Height-6)
	}
	cmd := model.seekForm.Focus()
	model.markRenderDirty()
	return cmd
}

func (k *Keys) handleSeekFormKey(model *Model, msg tea.KeyMsg) tea.Cmd {
	if model.seekForm == nil {
		model.showSeek = false
		return nil
	}
	cmd, _ := model.seekForm.Update(msg)
	model.markRenderDirty()
	return cmd
}

func (h *Handlers) handleSeekFormSubmit(model *Model, values map[string]string) (tea.Model, tea.Cmd) {
	model.showSeek = false
	model.seekForm = nil
	model.markRenderDirty()

	flags, err := buildSeekFlags(values["mode"], values["value"], seekPageSize, model.consumeFlags.Partitions)
	if err != nil {
		return model, core.NotifyError("Invalid seek", err)
	}
	model.statusMessage = fmt.Sprintf("Seek: %s", flags.Seek)
	return model, model.startForFlags(flags)
}

func (m *Model) renderSeekOverlay(width int) string {
	return renderFormOverlay("Seek — "+m.topicName,
		"pick a mode; from/to-offset need an integer, from/to-timestamp an RFC3339 or relative time",
		m.seekForm)
}

// --- MSG-22: partition filter + serde selector ---

func (k *Keys) handleShowPartitions(model *Model) tea.Cmd {
	cur := ""
	if len(model.consumeFlags.Partitions) > 0 {
		parts := make([]string, len(model.consumeFlags.Partitions))
		for i, p := range model.consumeFlags.Partitions {
			parts[i] = strconv.Itoa(int(p))
		}
		cur = strings.Join(parts, ",")
	}
	opts := model.serdeOptions()
	model.partitionForm = formpkg.New([]formpkg.Field{
		{Name: "partitions", Label: fmt.Sprintf("Partitions (comma-separated, empty=all of %d)", model.topicDetails.NumPartitions), Type: formpkg.Text, Default: cur},
		{Name: "keySerde", Label: "Key serde", Type: formpkg.Select, Options: opts, Default: model.keySerde},
		{Name: "valueSerde", Label: "Value serde", Type: formpkg.Select, Options: opts, Default: model.valueSerde},
	})
	model.showPartitions = true
	if model.dimensions.Width > 0 {
		model.partitionForm.SetDimensions(model.dimensions.Width-4, model.dimensions.Height-6)
	}
	cmd := model.partitionForm.Focus()
	model.markRenderDirty()
	return cmd
}

func (k *Keys) handlePartitionFormKey(model *Model, msg tea.KeyMsg) tea.Cmd {
	if model.partitionForm == nil {
		model.showPartitions = false
		return nil
	}
	cmd, _ := model.partitionForm.Update(msg)
	model.markRenderDirty()
	return cmd
}

func (h *Handlers) handlePartitionFormSubmit(model *Model, values map[string]string) (tea.Model, tea.Cmd) {
	model.showPartitions = false
	model.partitionForm = nil
	model.markRenderDirty()

	parts, err := buildPartitionFilter(values["partitions"], model.topicDetails.NumPartitions)
	if err != nil {
		return model, core.NotifyError("Invalid partitions", err)
	}
	model.keySerde = values["keySerde"]
	model.valueSerde = values["valueSerde"]

	flags := model.consumeFlags
	flags.Partitions = parts
	// Serde selection re-renders existing rows; the partition change refetches.
	model.rowStringsDirty = true
	model.statusMessage = fmt.Sprintf("Partitions: %s • serde k=%s v=%s", partitionLabel(parts), model.keySerde, model.valueSerde)
	return model, model.startForFlags(flags)
}

func partitionLabel(parts []int32) string {
	if len(parts) == 0 {
		return "all"
	}
	s := make([]string, len(parts))
	for i, p := range parts {
		s[i] = strconv.Itoa(int(p))
	}
	return strings.Join(s, ",")
}

func (m *Model) renderPartitionsOverlay(width int) string {
	return renderFormOverlay("Partitions & serde — "+m.topicName,
		"empty partitions = all; serde selects how key/value cells are decoded", m.partitionForm)
}

// applySerde applies the chosen serde to a displayed field. "auto" leaves the
// value as already decoded by the datasource; an explicit serde name decodes
// the raw bytes through the registry (falling back on failure) (MSG-15/16/22).
func (m *Model) applySerde(text string, raw []byte, pref string) string {
	if pref == "" || pref == serde.Auto || m.serdeReg == nil {
		return text
	}
	data := raw
	if len(data) == 0 {
		data = []byte(text)
	}
	out, _, _ := serde.Decode(m.serdeReg, pref, data)
	return out
}

// displayKey returns the fully display-processed key cell: serde selection,
// then masking, then projection (MSG-22/26/28).
func (m *Model) displayKey(msg api.Message) string {
	s := m.applySerde(msg.Key, msg.RawKey, m.keySerde)
	if m.masker != nil {
		s = m.masker.Apply(s, masking.Key)
	}
	return projectCell(s, m.keyProjection)
}

// displayValue returns the fully display-processed value cell.
func (m *Model) displayValue(msg api.Message) string {
	s := m.applySerde(msg.Value, msg.RawValue, m.valueSerde)
	if m.masker != nil {
		s = m.masker.Apply(s, masking.Value)
	}
	return projectCell(s, m.valueProjection)
}

// renderFormOverlay renders a titled form overlay with a consistent footer.
func renderFormOverlay(title, hint string, form *formpkg.Form) string {
	muted := lipgloss.NewStyle().Foreground(stylesPkg.FgMuted)
	header := lipgloss.NewStyle().Foreground(stylesPkg.Primary).Bold(true)
	var b strings.Builder
	b.WriteString(header.Render(title))
	b.WriteString("\n")
	b.WriteString(muted.Render("  " + hint))
	b.WriteString("\n\n")
	if form != nil {
		b.WriteString(form.View())
	}
	b.WriteString("\n")
	b.WriteString(muted.Render("tab: next • ←/→: change • enter: submit • esc: cancel"))
	return b.String()
}
