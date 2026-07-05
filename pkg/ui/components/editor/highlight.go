package editor

import (
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss"

	"github.com/Benny93/kafui/pkg/ui/styles"
)

// Semantic styles for JSON syntax highlighting. Colors are taken from the
// shared palette by role so they stay theme-consistent.
var (
	jsonKeyStyle     = lipgloss.NewStyle().Foreground(styles.Primary)
	jsonStringStyle  = lipgloss.NewStyle().Foreground(styles.Success)
	jsonNumberStyle  = lipgloss.NewStyle().Foreground(styles.Warning)
	jsonKeywordStyle = lipgloss.NewStyle().Foreground(styles.Info)
	jsonPunctStyle   = lipgloss.NewStyle().Foreground(styles.FgMuted)
)

// HighlightJSON applies syntax highlighting to a single line of (already
// pretty-printed) JSON. It is line-based so it composes with soft-wrapping and
// line numbering. Non-JSON input is returned styled as plain text.
func HighlightJSON(line string) string {
	var b strings.Builder
	runes := []rune(line)
	for i := 0; i < len(runes); i++ {
		c := runes[i]
		switch {
		case c == '"':
			// Read the full string literal (handling escaped quotes).
			j := i + 1
			for j < len(runes) {
				if runes[j] == '\\' {
					j += 2
					continue
				}
				if runes[j] == '"' {
					break
				}
				j++
			}
			literal := string(runes[i:min(j+1, len(runes))])
			// A string is a key when the next non-space rune is a colon.
			if isKey(runes, j+1) {
				b.WriteString(jsonKeyStyle.Render(literal))
			} else {
				b.WriteString(jsonStringStyle.Render(literal))
			}
			i = j
		case c == '-' || unicode.IsDigit(c):
			j := i
			for j < len(runes) && (unicode.IsDigit(runes[j]) || strings.ContainsRune("-+.eE", runes[j])) {
				j++
			}
			b.WriteString(jsonNumberStyle.Render(string(runes[i:j])))
			i = j - 1
		case unicode.IsLetter(c):
			j := i
			for j < len(runes) && unicode.IsLetter(runes[j]) {
				j++
			}
			word := string(runes[i:j])
			switch word {
			case "true", "false", "null":
				b.WriteString(jsonKeywordStyle.Render(word))
			default:
				b.WriteString(word)
			}
			i = j - 1
		case strings.ContainsRune("{}[],:", c):
			b.WriteString(jsonPunctStyle.Render(string(c)))
		default:
			b.WriteRune(c)
		}
	}
	return b.String()
}

// isKey reports whether the next non-space rune starting at idx is a colon.
func isKey(runes []rune, idx int) bool {
	for idx < len(runes) {
		if runes[idx] == ' ' || runes[idx] == '\t' {
			idx++
			continue
		}
		return runes[idx] == ':'
	}
	return false
}
