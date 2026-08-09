package consumergroup

import "github.com/Benny93/kafui/pkg/ui/keys"

// This page used to carry its own key map, binding `a` to auto-refresh (the
// actions-menu key), `t` to go-to-topic, `R` to offset reset and `d` to offset
// deletion while `d` deletes the group everywhere else. All four are
// actions-menu entries now.

// pageScope is the key scope this page resolves against.
func pageScope() keys.Scope { return keys.ScopeListContent }
