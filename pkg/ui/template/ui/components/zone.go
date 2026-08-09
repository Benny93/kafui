package components

import (
	"sync"

	zone "github.com/lrstanley/bubblezone"
)

// bubblezone panics if a View marks a zone before a global manager exists. The
// application creates one at start-up, but a component's View is also called
// from unit tests and from render paths that run before the program starts.
// mark makes those safe instead of requiring every caller to know about the
// global manager.
var zoneReady sync.Once

func mark(id, s string) (out string) {
	defer func() {
		if recover() != nil {
			// No manager: render the content unmarked. The element simply is
			// not clickable, which is the correct degradation outside a program.
			out = s
		}
	}()
	zoneReady.Do(func() {})
	return zone.Mark(id, s)
}
