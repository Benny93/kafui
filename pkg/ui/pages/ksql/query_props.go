package ksql

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
)

// propRow is one streams-property key/value pair in the properties editor.
type propRow struct {
	keyIn textinput.Model
	valIn textinput.Model
}

func newPropRow() propRow {
	k := textinput.New()
	k.Placeholder = "property"
	v := textinput.New()
	v.Placeholder = "value"
	return propRow{keyIn: k, valIn: v}
}

// addProp appends a property row, refused while any existing row has an empty
// key (KS-13).
func (m *QueryModel) addProp() {
	for _, r := range m.props {
		if strings.TrimSpace(r.keyIn.Value()) == "" {
			return
		}
	}
	m.props = append(m.props, newPropRow())
	// Focus the new row's key input.
	m.focusIdx = 1 + 2*(len(m.props)-1)
	m.syncFocus()
}

// delProp removes the property row the focus is on. Deleting the only remaining
// row resets it to empty instead of removing it (KS-13).
func (m *QueryModel) delProp() {
	if len(m.props) == 0 || m.focusIdx == 0 {
		return
	}
	row := (m.focusIdx - 1) / 2
	if row >= len(m.props) {
		return
	}
	if len(m.props) == 1 {
		m.props[0] = newPropRow()
		m.focusIdx = 1
		m.syncFocus()
		return
	}
	m.props = append(m.props[:row], m.props[row+1:]...)
	if m.focusIdx > 2*len(m.props) {
		m.focusIdx = 0
	}
	m.syncFocus()
}

// buildProps returns the streams-properties map, dropping rows with empty keys.
// Returns nil when no non-empty rows remain (request carries no map at all).
func (m *QueryModel) buildProps() map[string]string {
	out := map[string]string{}
	for _, r := range m.props {
		k := strings.TrimSpace(r.keyIn.Value())
		if k == "" {
			continue
		}
		out[k] = r.valIn.Value()
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// --- execution / streaming (KS-14/KS-15) ---

func (m *QueryModel) renderProps() string {
	var b strings.Builder
	b.WriteString(m.common.Styles.Header.Render("Properties"))
	b.WriteString("\n")
	if len(m.props) == 0 {
		b.WriteString(m.common.Styles.Muted.Render("(none — add a streams property from the actions menu)"))
		return b.String()
	}
	for _, r := range m.props {
		b.WriteString(r.keyIn.View() + " = " + r.valIn.View() + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}
