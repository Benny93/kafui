package topic

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	formpkg "github.com/Benny93/kafui/pkg/ui/components/form"
	"github.com/Benny93/kafui/pkg/ui/shared"
	tea "github.com/charmbracelet/bubbletea"
)

// projectField extracts the value at the dotted JSON path from payload and
// returns it as a display string (MSG-26). It returns ok=false when payload is
// not JSON, the path is empty, or the path does not resolve to a scalar.
// Array indices are supported as numeric path segments (e.g. "items.0.id").
func projectField(payload, path string) (string, bool) {
	path = strings.TrimSpace(path)
	if path == "" || payload == "" {
		return "", false
	}
	var root interface{}
	if err := json.Unmarshal([]byte(payload), &root); err != nil {
		return "", false
	}
	cur := root
	for _, seg := range strings.Split(path, ".") {
		switch node := cur.(type) {
		case map[string]interface{}:
			v, ok := node[seg]
			if !ok {
				return "", false
			}
			cur = v
		case []interface{}:
			idx, err := strconv.Atoi(seg)
			if err != nil || idx < 0 || idx >= len(node) {
				return "", false
			}
			cur = node[idx]
		default:
			return "", false
		}
	}
	switch v := cur.(type) {
	case string:
		return v, true
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), true
	case bool:
		return strconv.FormatBool(v), true
	case nil:
		return "null", true
	default:
		// Object/array leaf — render compact JSON.
		b, err := json.Marshal(v)
		if err != nil {
			return "", false
		}
		return string(b), true
	}
}

// projectCell renders a table cell for a key/value column. When a projection
// path is configured and resolves, it shows "path=extracted"; otherwise it
// falls back to the plain (already display-processed) content.
func projectCell(content, path string) string {
	if path == "" {
		return content
	}
	if v, ok := projectField(content, path); ok {
		return path + "=" + v
	}
	return content
}

// --- MSG-26: projection dialog (reuses the partition form overlay slot) ---

func (k *Keys) handleShowProjections(model *Model) tea.Cmd {
	model.produceForm = nil // ensure no other form claims the slot
	model.seekForm = formpkg.New([]formpkg.Field{
		{Name: "keyPath", Label: "Key column JSON path (empty = off)", Type: formpkg.Text, Default: model.keyProjection},
		{Name: "valuePath", Label: "Value column JSON path (empty = off)", Type: formpkg.Text, Default: model.valueProjection},
	})
	model.showProjections = true
	if model.dimensions.Width > 0 {
		model.seekForm.SetDimensions(model.dimensions.Width-4, model.dimensions.Height-6)
	}
	cmd := model.seekForm.Focus()
	model.markRenderDirty()
	return cmd
}

func (k *Keys) handleProjectionsKey(model *Model, msg tea.KeyMsg) tea.Cmd {
	if model.seekForm == nil {
		model.showProjections = false
		return nil
	}
	cmd, _ := model.seekForm.Update(msg)
	model.markRenderDirty()
	return cmd
}

func (h *Handlers) handleProjectionsSubmit(model *Model, values map[string]string) (tea.Model, tea.Cmd) {
	model.showProjections = false
	model.seekForm = nil
	model.keyProjection = strings.TrimSpace(values["keyPath"])
	model.valueProjection = strings.TrimSpace(values["valuePath"])
	model.rowStringsDirty = true // projections changed — rebuild row cache (MSG-26)
	model.markRenderDirty()

	// Persist per-topic projections.
	p := shared.LoadPrefs()
	if p.Projections == nil {
		p.Projections = map[string]shared.TopicProjection{}
	}
	if model.keyProjection == "" && model.valueProjection == "" {
		delete(p.Projections, model.topicName)
	} else {
		p.Projections[model.topicName] = shared.TopicProjection{Key: model.keyProjection, Value: model.valueProjection}
	}
	_ = shared.SavePrefs(p)
	model.statusMessage = fmt.Sprintf("Projections: key=%q value=%q", model.keyProjection, model.valueProjection)
	return model, nil
}

func (m *Model) renderProjectionsOverlay(width int) string {
	return renderFormOverlay("Field projections — "+m.topicName,
		"dotted JSON path per column (e.g. user.id or items.0.name); empty disables", m.seekForm)
}
