package mainpage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResourcePaginationModel_SetTotalItems(t *testing.T) {
	tests := []struct {
		name           string
		total          int
		perPage        int
		expectedPages  int
		expectedPage   int
	}{
		{"zero items", 0, 50, 0, 0},
		{"exactly one page", 50, 50, 1, 0},
		{"partial last page", 75, 50, 2, 0},
		{"multiple full pages", 200, 50, 4, 0},
		{"single item", 1, 50, 1, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &ResourcePaginationModel{PerPage: tt.perPage}
			p.SetTotalItems(tt.total)
			assert.Equal(t, tt.expectedPages, p.TotalPages)
			assert.Equal(t, tt.expectedPage, p.Page)
		})
	}
}

func TestResourcePaginationModel_SetTotalItems_ClampsPage(t *testing.T) {
	p := NewResourcePaginationModel()
	p.SetTotalItems(200) // 4 pages
	p.LastPage()         // page 3
	p.SetTotalItems(50)  // now only 1 page → page should clamp to 0
	assert.Equal(t, 0, p.Page)
}

func TestResourcePaginationModel_GetPageBounds(t *testing.T) {
	tests := []struct {
		name          string
		total         int
		page          int
		perPage       int
		expectedStart int
		expectedEnd   int
	}{
		{"first page full", 200, 0, 50, 0, 50},
		{"second page full", 200, 1, 50, 50, 100},
		{"last page partial", 75, 1, 50, 50, 75},
		{"single page", 30, 0, 50, 0, 30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &ResourcePaginationModel{Page: tt.page, PerPage: tt.perPage, TotalItems: tt.total}
			start, end := p.GetPageBounds()
			assert.Equal(t, tt.expectedStart, start)
			assert.Equal(t, tt.expectedEnd, end)
		})
	}
}

func TestResourcePaginationModel_GetCurrentPageItems(t *testing.T) {
	items := make([]interface{}, 120)
	for i := range items {
		items[i] = i
	}

	p := &ResourcePaginationModel{Page: 0, PerPage: 50, TotalItems: 120, TotalPages: 3}

	// Page 0: items 0-49
	page0 := p.GetCurrentPageItems(items)
	assert.Len(t, page0, 50)
	assert.Equal(t, 0, page0[0])
	assert.Equal(t, 49, page0[49])

	// Page 2 (last, partial): items 100-119
	p.Page = 2
	page2 := p.GetCurrentPageItems(items)
	assert.Len(t, page2, 20)
	assert.Equal(t, 100, page2[0])
	assert.Equal(t, 119, page2[19])
}

func TestResourcePaginationModel_NextPrevPage(t *testing.T) {
	p := NewResourcePaginationModel()
	p.SetTotalItems(150) // 3 pages

	assert.True(t, p.OnFirstPage())
	assert.False(t, p.OnLastPage())

	changed := p.NextPage()
	assert.True(t, changed)
	assert.Equal(t, 1, p.Page)

	p.NextPage()
	assert.True(t, p.OnLastPage())
	assert.Equal(t, 2, p.Page)

	changed = p.NextPage() // already on last page
	assert.False(t, changed)
	assert.Equal(t, 2, p.Page)

	p.PrevPage()
	assert.Equal(t, 1, p.Page)

	p.FirstPage()
	assert.Equal(t, 0, p.Page)

	p.LastPage()
	assert.Equal(t, 2, p.Page)
}

func TestResourcePaginationModel_GlobalIndex(t *testing.T) {
	p := &ResourcePaginationModel{Page: 2, PerPage: 50}
	assert.Equal(t, 105, p.GlobalIndex(5))
	assert.Equal(t, 100, p.GlobalIndex(0))
}

func TestResourcePaginationModel_PageStatus(t *testing.T) {
	p := NewResourcePaginationModel()
	assert.Equal(t, "No items", p.PageStatus())

	p.SetTotalItems(120)
	assert.Equal(t, "1-50 of 120  (Page 1/3)", p.PageStatus())

	p.NextPage()
	assert.Equal(t, "51-100 of 120  (Page 2/3)", p.PageStatus())

	p.NextPage()
	assert.Equal(t, "101-120 of 120  (Page 3/3)", p.PageStatus())
}

func TestTruncateMiddle(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{"short string unchanged", "short", 10, "short"},
		{"exact length unchanged", "hello", 5, "hello"},
		// maxLen < 5: no truncation applied
		{"too short maxLen unchanged", "hello world", 4, "hello world"},
		{"long topic name", "com.company.payments.transaction.created.v1", 20, "com.co…on.created.v1"},
		{"similar prefix shows suffix", "com.company.payments.transaction.created.v1", 30, "com.compa…ansaction.created.v1"},
		{"unicode safe", "αβγδεζηθικ", 6, "α…ηθικ"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateMiddle(tt.input, tt.maxLen)
			runeLen := len([]rune(result))
			if tt.maxLen >= 5 && len([]rune(tt.input)) > tt.maxLen {
				// truncated: result rune count == maxLen (ellipsis counts as 1)
				assert.Equal(t, tt.maxLen, runeLen, "truncated result rune length should equal maxLen")
				assert.Contains(t, result, "…", "truncated result should contain ellipsis")
			}
			assert.Equal(t, tt.expected, result)
		})
	}
}
