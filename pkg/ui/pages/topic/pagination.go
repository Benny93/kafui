package topic

import (
	"fmt"

	"github.com/Benny93/kafui/pkg/api"
)

// Pagination constants
const (
	// DefaultPerPage is the default number of messages per page
	DefaultPerPage = 20
	// MaxMessageBuffer limits total messages kept in memory
	MaxMessageBuffer = 1000
)

// PaginationModel handles pagination for topic messages
type PaginationModel struct {
	// Page is the current page number (0-indexed)
	Page int
	// PerPage is the number of messages per page
	PerPage int
	// TotalPages is calculated from total messages
	TotalPages int
	// TotalMessages is the total number of messages
	TotalMessages int
	// SortOrder determines message order ("newest_first" or "oldest_first")
	SortOrder string
}

// NewPaginationModel creates a new pagination model with defaults
func NewPaginationModel() *PaginationModel {
	return &PaginationModel{
		Page:        0,
		PerPage:     DefaultPerPage,
		TotalPages:  1,
		SortOrder:   "newest_first",
	}
}

// SetTotalMessages updates the total message count and recalculates pages
func (p *PaginationModel) SetTotalMessages(total int) {
	p.TotalMessages = total
	if total < 1 {
		p.TotalPages = 0
		p.Page = 0
		return
	}
	p.TotalPages = total / p.PerPage
	if total%p.PerPage > 0 {
		p.TotalPages++
	}
	// Ensure current page is still valid
	if p.Page >= p.TotalPages && p.TotalPages > 0 {
		p.Page = p.TotalPages - 1
	}
}

// SetPerPage updates messages per page and recalculates
func (p *PaginationModel) SetPerPage(perPage int) {
	if perPage < 1 {
		perPage = DefaultPerPage
	}
	p.PerPage = perPage
	p.SetTotalMessages(p.TotalMessages)
}

// GetPageBounds returns the start and end indices for the current page
// For newest_first: start is closer to the end of the slice
func (p *PaginationModel) GetPageBounds() (start int, end int) {
	if p.TotalMessages == 0 {
		return 0, 0
	}

	if p.SortOrder == "newest_first" {
		// Newest first: page 0 shows the last PerPage items
		end = p.TotalMessages - (p.Page * p.PerPage)
		start = end - p.PerPage
		if start < 0 {
			start = 0
		}
	} else {
		// Oldest first: page 0 shows the first PerPage items
		start = p.Page * p.PerPage
		end = start + p.PerPage
		if end > p.TotalMessages {
			end = p.TotalMessages
		}
	}

	return start, end
}

// GetVisibleMessages returns the messages for the current page
func (p *PaginationModel) GetVisibleMessages(messages []api.Message) []api.Message {
	if len(messages) == 0 {
		return []api.Message{}
	}

	start, end := p.GetPageBounds()
	if start >= len(messages) || end <= start {
		return []api.Message{}
	}

	return messages[start:end]
}

// NextPage moves to the next page if available
func (p *PaginationModel) NextPage() bool {
	if p.Page < p.TotalPages-1 {
		p.Page++
		return true
	}
	return false
}

// PrevPage moves to the previous page if available
func (p *PaginationModel) PrevPage() bool {
	if p.Page > 0 {
		p.Page--
		return true
	}
	return false
}

// FirstPage moves to the first page
func (p *PaginationModel) FirstPage() {
	p.Page = 0
}

// LastPage moves to the last page
func (p *PaginationModel) LastPage() {
	if p.TotalPages > 0 {
		p.Page = p.TotalPages - 1
	}
}

// OnFirstPage returns true if on the first page
func (p *PaginationModel) OnFirstPage() bool {
	return p.Page == 0
}

// OnLastPage returns true if on the last page
func (p *PaginationModel) OnLastPage() bool {
	return p.Page == p.TotalPages-1
}

// ItemsOnPage returns the number of items on the current page
func (p *PaginationModel) ItemsOnPage() int {
	if p.TotalMessages == 0 {
		return 0
	}
	start, end := p.GetPageBounds()
	return end - start
}

// PageStatus returns a human-readable page status string
func (p *PaginationModel) PageStatus() string {
	if p.TotalMessages == 0 {
		return "No messages"
	}

	start, end := p.GetPageBounds()
	if p.SortOrder == "newest_first" {
		// For newest first, show actual offset range
		return fmt.Sprintf("Messages %d-%d of %d (Page %d/%d)",
			p.TotalMessages-end+1, p.TotalMessages-start, p.TotalMessages,
			p.Page+1, p.TotalPages)
	}
	return fmt.Sprintf("Messages %d-%d of %d (Page %d/%d)",
		start+1, end, p.TotalMessages, p.Page+1, p.TotalPages)
}

// ToggleSortOrder switches between newest_first and oldest_first
func (p *PaginationModel) ToggleSortOrder() {
	if p.SortOrder == "newest_first" {
		p.SortOrder = "oldest_first"
	} else {
		p.SortOrder = "newest_first"
	}
	// Reset to first page when changing sort order
	p.Page = 0
}
