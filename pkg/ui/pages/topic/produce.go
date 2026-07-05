package topic

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/authz"
	formpkg "github.com/Benny93/kafui/pkg/ui/components/form"
	"github.com/Benny93/kafui/pkg/ui/core"
	tea "github.com/charmbracelet/bubbletea"
)

// canProduce reports whether producing to the current topic is permitted, and
// returns a status-bar notification command when it is not (AA-10 convention:
// mutating UI actions check Common before acting; the guard is the backstop).
func (k *Keys) canProduce(model *Model) tea.Cmd {
	if model.common == nil {
		return nil
	}
	if model.common.IsReadOnly() {
		return core.NewNotification(core.StatusWarning, "Read-only", "This cluster is configured read-only")
	}
	if !model.common.Can(authz.ActionProduceMessages, authz.ResourceTopic, model.topicName) {
		return core.NewNotification(core.StatusWarning, "Access denied", "producing to "+model.topicName+" is not permitted")
	}
	return nil
}

// parseHeaderField parses a "k1=v1,k2=v2" header string into MessageHeaders.
func parseHeaderField(s string) []api.MessageHeader {
	var out []api.MessageHeader
	for _, pair := range strings.Split(s, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		kv := strings.SplitN(pair, "=", 2)
		h := api.MessageHeader{Key: strings.TrimSpace(kv[0])}
		if len(kv) == 2 {
			h.Value = kv[1]
		}
		out = append(out, h)
	}
	return out
}

// formatHeaderField renders headers back into the "k=v,k=v" form for editing.
func formatHeaderField(headers []api.MessageHeader) string {
	parts := make([]string, 0, len(headers))
	for _, h := range headers {
		parts = append(parts, h.Key+"="+h.Value)
	}
	return strings.Join(parts, ",")
}

// buildProduceRecord assembles a ProduceRecord from form values (MSG-31). A
// blank key/value field yields a null record part (nil, not empty). The
// partition is "auto" or a valid partition index.
func buildProduceRecord(values map[string]string, numPartitions int32) (api.ProduceRecord, error) {
	rec := api.ProduceRecord{}
	if k := values["key"]; k != "" {
		rec.Key = []byte(k)
	}
	if v := values["value"]; v != "" {
		rec.Value = []byte(v)
	}
	rec.Headers = parseHeaderField(values["headers"])

	p := strings.TrimSpace(values["partition"])
	if p != "" && !strings.EqualFold(p, "auto") {
		n, err := strconv.Atoi(p)
		if err != nil {
			return rec, fmt.Errorf("partition must be 'auto' or an integer: %q", p)
		}
		if n < 0 || (numPartitions > 0 && int32(n) >= numPartitions) {
			return rec, fmt.Errorf("partition %d out of range (topic has %d partitions)", n, numPartitions)
		}
		pv := int32(n)
		rec.Partition = &pv
	}
	return rec, nil
}

// produceFields builds the produce form fields, pre-filled from prefill (nil for
// a blank form). Used by both produce (MSG-31) and reproduce (MSG-32).
func produceFields(prefill *api.Message) []formpkg.Field {
	var key, value, headers string
	if prefill != nil {
		key = prefill.Key
		value = prefill.Value
		headers = formatHeaderField(prefill.Headers)
	}
	return []formpkg.Field{
		{Name: "key", Label: "Key (blank = null)", Type: formpkg.Text, Default: key},
		{Name: "value", Label: "Value (blank = null)", Type: formpkg.Text, Default: value},
		{Name: "headers", Label: "Headers (k=v,k=v)", Type: formpkg.Text, Default: headers},
		{Name: "partition", Label: "Partition (auto or index)", Type: formpkg.Text, Default: "auto"},
		{Name: "keep", Label: "Keep contents after send", Type: formpkg.Bool, Default: "false"},
	}
}

func (k *Keys) openProduceForm(model *Model, prefill *api.Message) tea.Cmd {
	model.produceForm = formpkg.New(produceFields(prefill))
	model.showProduce = true
	if model.dimensions.Width > 0 {
		model.produceForm.SetDimensions(model.dimensions.Width-4, model.dimensions.Height-6)
	}
	cmd := model.produceForm.Focus()
	model.markRenderDirty()
	return cmd
}

// handleShowProduce opens a blank produce form (MSG-31).
func (k *Keys) handleShowProduce(model *Model) tea.Cmd {
	if cmd := k.canProduce(model); cmd != nil {
		return cmd
	}
	return k.openProduceForm(model, nil)
}

// handleReproduce opens the produce form pre-filled from the selected message (MSG-32).
func (k *Keys) handleReproduce(model *Model) tea.Cmd {
	if cmd := k.canProduce(model); cmd != nil {
		return cmd
	}
	sel := model.GetSelectedMessage()
	if sel == nil {
		return core.NewNotification(core.StatusWarning, "No message", "Select a message to reproduce")
	}
	return k.openProduceForm(model, sel)
}

func (k *Keys) handleProduceFormKey(model *Model, msg tea.KeyMsg) tea.Cmd {
	if model.produceForm == nil {
		model.showProduce = false
		return nil
	}
	cmd, _ := model.produceForm.Update(msg)
	model.markRenderDirty()
	return cmd
}

func (h *Handlers) handleProduceFormSubmit(model *Model, values map[string]string) (tea.Model, tea.Cmd) {
	rec, err := buildProduceRecord(values, model.topicDetails.NumPartitions)
	if err != nil {
		return model, core.NotifyError("Invalid message", err)
	}
	keep := values["keep"] == "true"

	// Close (or reopen blank-preserving) the form.
	model.showProduce = false
	model.produceForm = nil
	if keep {
		model.produceForm = formpkg.New(produceFields(&api.Message{
			Key: values["key"], Value: values["value"], Headers: parseHeaderField(values["headers"]),
		}))
		model.showProduce = true
		if model.dimensions.Width > 0 {
			model.produceForm.SetDimensions(model.dimensions.Width-4, model.dimensions.Height-6)
		}
	}
	model.markRenderDirty()

	ds := model.dataSource
	topic := model.topicName
	return model, func() tea.Msg {
		if err := ds.ProduceMessage(context.Background(), topic, rec); err != nil {
			return core.NotificationMsg{Severity: core.StatusError, Title: "Produce failed", Message: err.Error()}
		}
		return core.NotificationMsg{Severity: core.StatusSuccess, Title: "Message produced", Message: "to " + topic}
	}
}

func (m *Model) renderProduceOverlay(width int) string {
	return renderFormOverlay("Produce to "+m.topicName,
		"blank key/value = null; headers as k=v,k=v; partition 'auto' or index", m.produceForm)
}
