//go:build debug

package debug

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/Benny93/kafui/pkg/ui/keys"
	"github.com/Benny93/kafui/pkg/ui/styles"
)

// maxKeys is how many recent presses the strip shows. Enough to read a two-key
// sequence like `a` then a mnemonic, without covering the hint bar.
const maxKeys = 6

// Keycast records recent key presses and what the binding registry made of
// them. It exists for the demo recordings and for debugging the controls: a key
// that resolves to nothing looks identical to a key that resolved to an action
// which then did nothing, and this tells the two apart.
//
// Debug builds only. The production build gets the no-op stub.
type Keycast struct {
	entries []castEntry
}

type castEntry struct {
	key string
	// action is what the registry resolved the key to, or "" when unbound.
	action string
}

// Record adds a press. action is the resolved action id, or "" when the key is
// not bound in the current scope.
func (k *Keycast) Record(key, action string) {
	// Modifier-only and mouse-motion noise would drown the useful presses.
	if key == "" {
		return
	}
	k.entries = append(k.entries, castEntry{key: key, action: action})
	if len(k.entries) > maxKeys {
		k.entries = k.entries[len(k.entries)-maxKeys:]
	}
}

var (
	castFrame = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(styles.FgSubtle).
			Padding(0, 1)
	castKey = lipgloss.NewStyle().
		Foreground(styles.BgBase).Background(styles.Primary).Bold(true).Padding(0, 1)
	castAction  = lipgloss.NewStyle().Foreground(styles.FgMuted)
	castUnbound = lipgloss.NewStyle().Foreground(styles.Error).Italic(true)
)

// View renders the strip, newest press last. An empty string means nothing has
// been pressed yet.
func (k *Keycast) View() string {
	if len(k.entries) == 0 {
		return ""
	}
	parts := make([]string, 0, len(k.entries))
	for _, e := range k.entries {
		label := castKey.Render(keys.Display(e.key))
		if e.action == "" {
			label += castUnbound.Render(" unbound")
		} else {
			label += castAction.Render(" " + e.action)
		}
		parts = append(parts, label)
	}
	return castFrame.Render(strings.Join(parts, "  "))
}

// KeycastEnabled reports whether this build renders the strip.
const KeycastEnabled = true
