package ksql

import "github.com/Benny93/kafui/pkg/ui/keys"

// The ksqlDB pages used to carry their own key maps — a second registry that
// could, and did, disagree with the global one (ctrl+d for "delete property"
// while ctrl+d also meant delete and toggled the debug overlay; ctrl+r for
// "clear results" while ctrl+r refreshed the sidebar). Both pages now resolve
// through the single registry.

// overviewScope is the key scope of the ksqlDB overview page.
func overviewScope() keys.Scope { return keys.ScopeList }

// queryScope is the key scope of the query editor. It is a text-entry context:
// every printable key is typed, and only F5 (run), tab, esc and ctrl+c act.
func queryScope() keys.Scope { return keys.ScopeTextEntry }
