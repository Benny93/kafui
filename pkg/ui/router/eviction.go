package router

import "github.com/Benny93/kafui/pkg/ui/core"

// The router caches pages so that Back returns to them intact. It keeps only
// the pages Back can still reach: "main", the page showing, and the pages in
// history. Everything else is evicted, so every message ever opened no longer
// stays in memory, and a page reopened later starts fresh (ARCH-3, CONC-5).

// prune evicts every page that is neither "main", the current page, nor in the
// history.
func (r *Router) prune() {
	keep := make(map[string]bool, len(r.history)+2)
	keep["main"] = true
	keep[r.currentPage] = true
	for _, id := range r.history {
		keep[id] = true
	}
	for id := range r.pages {
		if !keep[id] {
			r.evict(id)
		}
	}
}

// syncContext evicts every cached page but "main" and the current one when the
// active cluster changed since the last navigation. Page IDs name entities
// (topic:<name>, broker:<id>) that mean something else on another cluster, and
// a cached page keeps the old cluster's data and masking rules. Detecting the
// change here covers every switch path without each one having to announce it.
func (r *Router) syncContext() {
	if r.com == nil || r.com.DataSource == nil {
		return
	}
	ctx := r.com.DataSource.GetContext()
	if !r.contextKnown {
		r.context, r.contextKnown = ctx, true
		return
	}
	if ctx == r.context {
		return
	}
	r.context = ctx

	hadMain := false
	for _, id := range r.history {
		if id == "main" {
			hadMain = true
			break
		}
	}
	r.history = r.history[:0]
	if hadMain && r.currentPage != "main" {
		r.history = append(r.history, "main")
	}
	for id := range r.pages {
		if id != "main" && id != r.currentPage {
			r.evict(id)
		}
	}
}

// evict drops a cached page, releasing what it holds, along with any results
// still waiting for it.
func (r *Router) evict(pageID string) {
	if d, ok := r.pages[pageID].(core.Disposer); ok {
		d.Dispose()
	}
	delete(r.pages, pageID)
	delete(r.gens, pageID)
	delete(r.pending, pageID)
}
