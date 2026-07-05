package serde

import "sort"

// Registry holds the serdes available for lookup and auto-detection. A subset
// of registered serdes participate in auto-detection (in registration order);
// the rest are selectable only by explicit name (e.g. numeric/hex serdes that
// would falsely match arbitrary bytes). Registry is not safe for concurrent
// registration, but lookups after construction are read-only and safe.
type Registry struct {
	byName map[string]Serde
	auto   []Serde // auto-detection order
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{byName: make(map[string]Serde)}
}

// Register adds a serde selectable by name only. It rejects duplicate names.
func (r *Registry) Register(s Serde) error {
	if _, ok := r.byName[s.Name()]; ok {
		return DuplicateSerdeError{Name: s.Name()}
	}
	r.byName[s.Name()] = s
	return nil
}

// RegisterAuto adds a serde that participates in auto-detection (appended to
// the detection order) in addition to being selectable by name.
func (r *Registry) RegisterAuto(s Serde) error {
	if err := r.Register(s); err != nil {
		return err
	}
	r.auto = append(r.auto, s)
	return nil
}

// Get returns the serde registered under name.
func (r *Registry) Get(name string) (Serde, bool) {
	s, ok := r.byName[name]
	return s, ok
}

// Names returns all registered serde names, sorted, for the UI selector.
func (r *Registry) Names() []string {
	out := make([]string, 0, len(r.byName))
	for name := range r.byName {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// AutoDetect returns the first auto-detection serde that claims the bytes, or
// nil when none match.
func (r *Registry) AutoDetect(data []byte) Serde {
	for _, s := range r.auto {
		if s.CanDeserialize(data) {
			return s
		}
	}
	return nil
}
