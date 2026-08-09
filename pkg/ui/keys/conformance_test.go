package keys_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Conformance rule 1 from the controls spec: no file outside the binding
// registry compares a raw key string, and no page keeps a parallel key map.
// This is the test that stops the old shape growing back.
func TestNoPageComparesRawKeyStrings(t *testing.T) {
	// Contexts where a literal key is legitimate: text-entry and form widgets
	// read raw runes, and the registry itself declares the literals.
	allowedFiles := map[string]bool{
		"registry.go": true, "help.go": true,
	}
	// Keys a text field must handle itself to edit text.
	editing := regexp.MustCompile(`^"(enter|esc|tab|shift\+tab|backspace|up|down|left|right|ctrl\+c|ctrl\+u|ctrl\+w|ctrl\+a|f2|f5| )"$`)
	caseLiteral := regexp.MustCompile(`case ("(?:[^"]+)"(?:, "(?:[^"]+)")*):`)
	// `msg.String() == "ctrl+s"` is the same defect wearing a different shape,
	// and it is how a forbidden flow-control key survived three sweeps.
	eqLiteral := regexp.MustCompile(`String\(\)\s*==\s*("[^"]+")`)

	root := filepath.Join("..", "..", "..", "pkg", "ui")
	var offenders []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		if allowedFiles[filepath.Base(path)] {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		// Forms and editors legitimately switch on editing keys.
		if strings.Contains(path, "components/form") || strings.Contains(path, "components/editor") ||
			strings.Contains(path, "dialog") {
			return nil
		}
		matches := caseLiteral.FindAllStringSubmatch(string(src), -1)
		matches = append(matches, eqLiteral.FindAllStringSubmatch(string(src), -1)...)
		for _, m := range matches {
			for _, lit := range strings.Split(m[1], ", ") {
				if editing.MatchString(lit) {
					continue
				}
				// Non-key literals (state names, config keys) are strings too;
				// only single characters and chords look like keys.
				v := strings.Trim(lit, `"`)
				isKeyish := len(v) == 1 || strings.HasPrefix(v, "ctrl+") || strings.HasPrefix(v, "shift+")
				if isKeyish {
					offenders = append(offenders, path+": case "+lit)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, o := range offenders {
		t.Errorf("raw key literal outside the registry: %s", o)
	}
}

// The `case "x"` sweep above misses the other shape a parallel key map takes:
// a page declaring its own key.NewBinding set and matching with key.Matches.
// That is how the clusters, broker, connector, metrics and ksqlDB pages each
// kept a second registry that could disagree with the global one.
func TestNoPageDeclaresItsOwnBindings(t *testing.T) {
	newBinding := regexp.MustCompile(`key\.NewBinding\(`)

	root := filepath.Join("..", "..", "..", "pkg", "ui")
	var offenders []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		// The registry is the one place allowed to construct bindings. The
		// dialog and the reusable datatable are the two exceptions: they are
		// widgets whose bindings ARE the overlay/list vocabulary, and they are
		// listed here explicitly so a new exception has to be argued for.
		p := filepath.ToSlash(path)
		if strings.Contains(p, "pkg/ui/keys/") ||
			strings.HasSuffix(p, "ui/dialog/confirm.go") ||
			strings.HasSuffix(p, "components/datatable/datatable.go") {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if newBinding.Match(src) {
			offenders = append(offenders, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, o := range offenders {
		t.Errorf("key bindings declared outside the registry: %s", o)
	}
}

// Rendered hint text is the third way a removed key survives: not in a `case`
// and not in a key.Binding, but in a string the user reads. The ksqlDB editor
// advertised "ctrl+x: execute" for a release after ctrl+x had gone.
func TestNoRenderedTextNamesAKey(t *testing.T) {
	// A chord in a user-visible string is always suspect; the registry is the
	// only thing allowed to know what a key is called.
	chord := regexp.MustCompile(`"[^"]*\bctrl\+[a-z]\b[^"]*"`)
	// "x: label" or "• x: label" — the shape of a hand-written hint strip.
	hint := regexp.MustCompile(`"[^"]*(?:^|• )[a-zA-Z]{1,5}: [a-z][^"]*"`)

	root := filepath.Join("..", "..", "..", "pkg", "ui")
	var offenders []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		p := filepath.ToSlash(path)
		if strings.Contains(p, "pkg/ui/keys/") ||
			// reusable_app maps a registry key name back to a KeyMsg so that
			// clicking a hint and pressing its key take the same path; that
			// table has to spell the keys out.
			strings.HasSuffix(p, "template/ui/reusable_app.go") {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, line := range strings.Split(string(src), "\n") {
			// A `case` is handled by the raw-key test above, not here.
			if strings.HasPrefix(strings.TrimSpace(line), "case ") {
				continue
			}
			// Log messages and notifications are prose, not hint strips.
			if strings.Contains(line, "slog.") || strings.Contains(line, "log.") ||
				strings.Contains(line, "Notification") || strings.Contains(line, "Errorf") {
				continue
			}
			if chord.MatchString(line) || hint.MatchString(line) {
				offenders = append(offenders, p+": "+strings.TrimSpace(line))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, o := range offenders {
		t.Errorf("hand-written key name in rendered text (use inlineHint / keys.Default.KeyFor): %s", o)
	}
}
