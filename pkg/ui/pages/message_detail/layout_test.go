package messagedetail

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/datasource/mock"
	"github.com/Benny93/kafui/pkg/ui/components/editor"
)

// RenderContent must fit the size the template gives it on every tab, or the
// template's frame is pushed off screen.
func TestRenderContentFitsGivenSize(t *testing.T) {
	ds := &mock.KafkaDataSourceMock{}
	ds.Init("")
	value := "{" + strings.Repeat(`"field": "some value that is fairly long",`, 60) + `"end": 1}`
	msg := api.Message{Key: "order-1", Value: value, Headers: []api.MessageHeader{{Key: "h", Value: "v"}}}

	for _, size := range [][2]int{{60, 16}, {100, 30}, {150, 40}} {
		for tab := 0; tab < 3; tab++ {
			for _, search := range []string{"none", "key", "value"} {
				t.Run(fmt.Sprintf("%dx%d/tab%d/search=%s", size[0], size[1], tab, search), func(t *testing.T) {
					p := NewMessageDetailContentProvider(NewModel(ds, "orders", msg))
					p.activeTab = tab
					if v := map[string]*editor.Viewer{"key": p.keyEditor, "value": p.valueEditor}[search]; v != nil {
						v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
						require.True(t, v.Searching())
					}
					out := p.RenderContent(size[0], size[1])
					assert.LessOrEqual(t, lipgloss.Height(out), size[1], "height")
					assert.LessOrEqual(t, lipgloss.Width(out), size[0], "width")
				})
			}
		}
	}
}

// The key pane is sized to a short key instead of taking a full-height column.
func TestShortKeyGetsCompactPane(t *testing.T) {
	ds := &mock.KafkaDataSourceMock{}
	ds.Init("")
	p := NewMessageDetailContentProvider(NewModel(ds, "orders", api.Message{Key: "k1", Value: "{\"a\": 1}"}))
	p.RenderContent(100, 30)
	assert.Equal(t, 1, lipgloss.Height(p.keyEditor.View()))
	assert.Greater(t, lipgloss.Height(p.valueEditor.View()), 15)
}

// Esc while the search prompt is open closes the prompt; the page must not
// also navigate back (the shell owns esc outside text entry).
func TestEscDoesNotNavigateFromPage(t *testing.T) {
	ds := &mock.KafkaDataSourceMock{}
	ds.Init("")
	page := NewMessageDetailPageModel(ds, "orders", api.Message{Key: "k", Value: "v"})
	_, cmd := page.HandleNavigation(tea.KeyMsg{Type: tea.KeyEsc})
	assert.Nil(t, cmd)
}
