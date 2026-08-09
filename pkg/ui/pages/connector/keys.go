package connector

import "github.com/Benny93/kafui/pkg/ui/keys"

// This page used to carry its own key map — a second registry that could, and
// did, disagree with the global one. The hint bar it fed still advertised keys
// the page had stopped handling. Both now come from the single registry.

// pageScope is the key scope this page resolves against.
func pageScope() keys.Scope { return keys.ScopeConnector }
