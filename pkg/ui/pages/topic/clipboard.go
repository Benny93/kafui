package topic

import (
	"fmt"
	"strings"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/masking"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/shared"
	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
)

// writeClipboard is the clipboard sink; tests replace it.
var writeClipboard = clipboard.WriteAll

func (k *Keys) handleCopyKey(model *Model) tea.Cmd {
	return model.copySelected("key")
}

func (k *Keys) handleCopyValue(model *Model) tea.Cmd {
	if model.expanded {
		return model.copyExpanded()
	}
	return model.copySelected("value")
}

// copyExpanded copies exactly what the expanded panel shows for the selected
// message: the displayed value, pretty-printed when it is JSON.
func (m *Model) copyExpanded() tea.Cmd {
	msg := m.GetSelectedMessage()
	if msg == nil {
		return nil
	}
	text, _ := expandedBody(m.displayValue(*msg))
	if err := writeClipboard(text); err != nil {
		shared.Log.Error("clipboard copy failed", "field", "expanded", "err", err)
		return core.NotifyError("Copy failed", err)
	}
	return core.NewNotification(core.StatusSuccess, "Copied to clipboard",
		fmt.Sprintf("p%d @ %d · %d lines", msg.Partition, msg.Offset, strings.Count(text, "\n")+1))
}

// copySelected copies the selected message's key or value to the clipboard,
// decoded by the selected serde and masked like the table shows it (the
// field projection is not applied: the copy is the whole payload).
func (m *Model) copySelected(field string) tea.Cmd {
	msg := m.GetSelectedMessage()
	if msg == nil {
		m.statusMessage = "No message selected"
		return nil
	}
	text := m.copyText(*msg, field == "key")
	if text == "" {
		m.statusMessage = fmt.Sprintf("Message %s is empty — nothing copied", field)
		return nil
	}
	if err := writeClipboard(text); err != nil {
		shared.Log.Error("clipboard copy failed", "field", field, "err", err)
		m.statusMessage = fmt.Sprintf("Copy failed: %v", err)
		return core.NotifyError("Copy failed", err)
	}
	m.statusMessage = fmt.Sprintf("Message %s copied to clipboard", field)
	return core.NewNotification(core.StatusSuccess, "Copied to clipboard", "Message "+field)
}

// copyText is the key or value text a copy puts on the clipboard.
func (m *Model) copyText(msg api.Message, key bool) string {
	if key {
		s := m.applySerde(msg.Key, msg.RawKey, m.keySerde)
		if m.masker != nil {
			s = m.masker.Apply(s, masking.Key)
		}
		return s
	}
	s := m.applySerde(msg.Value, msg.RawValue, m.valueSerde)
	if m.masker != nil {
		s = m.masker.Apply(s, masking.Value)
	}
	return s
}
