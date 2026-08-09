package router

import (
	"testing"

	"github.com/Benny93/kafui/pkg/datasource/mock"
	"github.com/Benny93/kafui/pkg/ui/core"
)

// The router holds a page WRAPPER for several screens, while the actions and
// palette live on the inner model. When a wrapper forgets to forward them the
// menu silently falls back to its empty placeholder — which is how the schema
// screen shipped with a dead `a` key until a re-recorded demo caught it.
func TestEveryPageOffersAnActionsMenu(t *testing.T) {
	ds := &mock.KafkaDataSourceMock{}
	ds.Init("")
	common := core.NewCommon(ds)
	r := NewRouter(common)
	r.SetDimensions(120, 40)

	// Pages reachable without navigation data. Dynamic pages (topic:<name>,
	// broker:<id>) are covered by their own package tests.
	for _, id := range []string{"main", "clusters", "metrics", "appconfig", "ksql"} {
		if cmd := r.NavigateTo(id, nil); cmd == nil {
			continue
		}
		page := r.GetCurrentPage()
		if page == nil {
			t.Errorf("%s: router produced no page", id)
			continue
		}
		provider, ok := page.(core.ActionProvider)
		if !ok {
			t.Errorf("%s: page type %T does not implement core.ActionProvider — `a` opens an empty menu", id, page)
			continue
		}
		if len(provider.ContextActions()) == 0 {
			t.Errorf("%s: actions menu is empty", id)
		}
	}
}
