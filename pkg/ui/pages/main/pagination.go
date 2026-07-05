package mainpage

import "fmt"

// DefaultPageSize is the number of resource items shown per page.
const DefaultPageSize = 50

// ResourcePaginationModel handles pagination for the main resource list.
type ResourcePaginationModel struct {
	// Page is the current page number (0-indexed).
	Page int
	// PerPage is the number of items per logical page.
	PerPage int
	// TotalItems is the total number of items across all pages.
	TotalItems int
	// TotalPages is derived from TotalItems / PerPage.
	TotalPages int
}

// NewResourcePaginationModel creates a pagination model with defaults.
func NewResourcePaginationModel() *ResourcePaginationModel {
	return &ResourcePaginationModel{
		Page:    0,
		PerPage: DefaultPageSize,
	}
}

// SetTotalItems updates the total item count and recalculates page count.
func (p *ResourcePaginationModel) SetTotalItems(total int) {
	p.TotalItems = total
	if total < 1 {
		p.TotalPages = 0
		p.Page = 0
		return
	}
	p.TotalPages = total / p.PerPage
	if total%p.PerPage > 0 {
		p.TotalPages++
	}
	// Keep current page in bounds.
	if p.Page >= p.TotalPages {
		p.Page = p.TotalPages - 1
	}
}

// GetPageBounds returns the [start, end) indices for the current page.
func (p *ResourcePaginationModel) GetPageBounds() (start, end int) {
	start = p.Page * p.PerPage
	end = start + p.PerPage
	if end > p.TotalItems {
		end = p.TotalItems
	}
	return start, end
}

// GetCurrentPageItems returns the slice of items for the current page.
func (p *ResourcePaginationModel) GetCurrentPageItems(items []interface{}) []interface{} {
	if len(items) == 0 {
		return []interface{}{}
	}
	start, end := p.GetPageBounds()
	if start >= len(items) {
		return []interface{}{}
	}
	if end > len(items) {
		end = len(items)
	}
	return items[start:end]
}

// GlobalIndex converts a page-local row index to a global item index.
func (p *ResourcePaginationModel) GlobalIndex(localIndex int) int {
	return p.Page*p.PerPage + localIndex
}

// NextPage advances to the next page. Returns true if the page changed.
func (p *ResourcePaginationModel) NextPage() bool {
	if p.Page < p.TotalPages-1 {
		p.Page++
		return true
	}
	return false
}

// PrevPage moves to the previous page. Returns true if the page changed.
func (p *ResourcePaginationModel) PrevPage() bool {
	if p.Page > 0 {
		p.Page--
		return true
	}
	return false
}

// FirstPage jumps to the first page.
func (p *ResourcePaginationModel) FirstPage() {
	p.Page = 0
}

// LastPage jumps to the last page.
func (p *ResourcePaginationModel) LastPage() {
	if p.TotalPages > 0 {
		p.Page = p.TotalPages - 1
	}
}

// OnFirstPage returns true when on the first page.
func (p *ResourcePaginationModel) OnFirstPage() bool {
	return p.Page == 0
}

// OnLastPage returns true when on the last page.
func (p *ResourcePaginationModel) OnLastPage() bool {
	return p.TotalPages == 0 || p.Page == p.TotalPages-1
}

// PageStatus returns a human-readable pagination string, e.g. "1-50 of 200 (Page 1/4)".
func (p *ResourcePaginationModel) PageStatus() string {
	if p.TotalItems == 0 {
		return "No items"
	}
	start, end := p.GetPageBounds()
	return fmt.Sprintf("%d-%d of %d  (Page %d/%d)",
		start+1, end, p.TotalItems, p.Page+1, p.TotalPages)
}
