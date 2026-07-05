package shared

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// SavedFilter is a named smart-filter expression persisted across runs (MSG-25).
type SavedFilter struct {
	Name string `json:"name"`
	Expr string `json:"expr"`
}

// TopicProjection holds the key/value JSON-path projections for a topic's
// message table columns (MSG-26). Empty path means "no projection".
type TopicProjection struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Prefs holds small, user-local UI preferences persisted across runs.
// Add fields as new persisted toggles are introduced.
type Prefs struct {
	HideInternalTopics bool `json:"hideInternalTopics"`
	// SavedFilters are named smart-filter expressions (MSG-25).
	SavedFilters []SavedFilter `json:"savedFilters,omitempty"`
	// Projections maps a topic name to its column projections (MSG-26).
	Projections map[string]TopicProjection `json:"projections,omitempty"`
}

// prefsPath returns the location of the prefs file:
// $XDG_CONFIG_HOME/kafui/prefs.json, falling back to $HOME/.config/kafui/prefs.json.
func prefsPath() string {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "kafui", "prefs.json")
}

// LoadPrefs reads the prefs file. A missing/unreadable file yields zero-value
// Prefs (never an error) so callers can use it unconditionally on startup.
func LoadPrefs() Prefs {
	var p Prefs
	path := prefsPath()
	if path == "" {
		return p
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return p
	}
	_ = json.Unmarshal(data, &p)
	return p
}

// SavePrefs writes prefs to the prefs file, creating the directory as needed.
func SavePrefs(p Prefs) error {
	path := prefsPath()
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
