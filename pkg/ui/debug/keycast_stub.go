//go:build !debug

package debug

// Keycast is the production no-op. The strip is a debugging and demo-recording
// aid, so it costs nothing in a release build.
type Keycast struct{}

// Record does nothing in a production build.
func (k *Keycast) Record(key, action string) {}

// View returns nothing in a production build.
func (k *Keycast) View() string { return "" }

// KeycastEnabled reports whether this build renders the strip.
const KeycastEnabled = false
