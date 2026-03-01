package components

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/Benny93/kafui/pkg/ui/template/ui/providers"
)

func TestCalculateMaxItems(t *testing.T) {
	s := &sidebar{}

	tests := []struct {
		name            string
		availableHeight int
		numSections     int
		wantMinTotal    int // Minimum total items expected
		wantMaxTotal    int // Maximum total items expected
	}{
		{"Very little space", 5, 3, 6, 9},
		{"Small space", 10, 3, 6, 15},
		{"Normal space", 30, 3, 6, 30},
		{"Large space", 50, 3, 6, 30},
		{"Single section", 20, 1, 2, 10},
		{"Many sections", 40, 5, 10, 50},
		{"Zero sections", 20, 0, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limits := s.calculateMaxItems(tt.availableHeight, tt.numSections)

			if len(limits) != tt.numSections {
				t.Errorf("calculateMaxItems() returned %d limits, want %d", len(limits), tt.numSections)
			}

			// Check total items
			total := 0
			for _, limit := range limits {
				total += limit
			}

			if total < tt.wantMinTotal {
				t.Errorf("calculateMaxItems() total = %d, want >= %d", total, tt.wantMinTotal)
			}
			if tt.wantMaxTotal > 0 && total > tt.wantMaxTotal {
				t.Errorf("calculateMaxItems() total = %d, want <= %d", total, tt.wantMaxTotal)
			}

			// Check that earlier sections get priority (more or equal items)
			for i := 1; i < len(limits); i++ {
				if limits[i] > limits[i-1] {
					t.Errorf("calculateMaxItems() should give priority to earlier sections: limits[%d]=%d > limits[%d]=%d",
						i, limits[i], i-1, limits[i-1])
				}
			}
		})
	}
}

func TestCalculateMaxItems_MinimumItems(t *testing.T) {
	s := &sidebar{}
	limits := s.calculateMaxItems(5, 3)

	// Each section should have at least 2 items
	for i, limit := range limits {
		if limit < 2 {
			t.Errorf("calculateMaxItems() limits[%d] = %d, want >= 2", i, limit)
		}
	}
}

func TestCalculateMaxItems_PriorityDistribution(t *testing.T) {
	s := &sidebar{}
	limits := s.calculateMaxItems(30, 3)

	// First section should have at least as many items as second
	if limits[0] < limits[1] {
		t.Errorf("First section should have priority: limits[0]=%d < limits[1]=%d", limits[0], limits[1])
	}

	// Second section should have at least as many items as third
	if limits[1] < limits[2] {
		t.Errorf("Second section should have priority over third: limits[1]=%d < limits[2]=%d", limits[1], limits[2])
	}
}

func TestCalculateMaxItems_MaxItemsCap(t *testing.T) {
	s := &sidebar{}
	limits := s.calculateMaxItems(100, 2)

	// Each section should be capped at defaultMaxItems (10)
	const defaultMaxItems = 10
	for i, limit := range limits {
		if limit > defaultMaxItems {
			t.Errorf("calculateMaxItems() limits[%d] = %d, want <= %d", i, limit, defaultMaxItems)
		}
	}
}

// MockSidebarSection for testing
type MockSidebarSection struct {
	title string
	items []providers.SidebarItem
}

func (m *MockSidebarSection) GetTitle() string {
	return m.title
}

func (m *MockSidebarSection) RenderItems(maxItems, width int) []providers.SidebarItem {
	if maxItems > len(m.items) {
		return m.items
	}
	return m.items[:maxItems]
}

func (m *MockSidebarSection) HandleSectionUpdate(msg tea.Msg) tea.Cmd {
	return nil
}

func (m *MockSidebarSection) InitSection() tea.Cmd {
	return nil
}

func (m *MockSidebarSection) RefreshSection() tea.Cmd {
	return nil
}

func TestRenderSection_TextTruncation(t *testing.T) {
	s := &sidebar{
		width:  30,
		height: 20,
	}

	// Create a section with very long item text
	longText := "This is a very long item text that should definitely be truncated to fit within the sidebar width"
	section := &MockSidebarSection{
		title: "Test",
		items: []providers.SidebarItem{
			{Icon: "●", Text: longText, Value: "100MB", Status: "success"},
		},
	}

	result := s.renderSection(section, 1, 30)
	lines := splitLines(result)

	// Test that item text (line 1, after header) is truncated and contains ellipsis
	if len(lines) > 1 {
		itemLine := lines[1]
		// The item line should contain ellipsis since text is long
		if !strings.Contains(itemLine, "…") {
			t.Errorf("renderSection() should truncate long text with ellipsis. Line: %s", itemLine)
		}
		// The item line should be reasonable (not extremely long)
		if len(itemLine) > 35 { // Allow some tolerance for icon and value
			t.Errorf("renderSection() item line too long: %d. Line: %s", len(itemLine), itemLine)
		}
	}
}

func TestRenderSection_HeaderTruncation(t *testing.T) {
	s := &sidebar{
		width:  20,
		height: 20,
	}

	// Create a section with a very long title
	longTitle := "This is a very long section title that should be truncated"
	section := &MockSidebarSection{
		title: longTitle,
		items: []providers.SidebarItem{},
	}

	result := s.renderSection(section, 0, 20)
	lines := splitLines(result)

	// Test that header title is truncated (contains ellipsis)
	if len(lines) > 0 {
		headerLine := lines[0]
		// The header should contain ellipsis since title is long
		if !strings.Contains(headerLine, "…") {
			t.Errorf("renderSection() should truncate long title with ellipsis. Header: %s", headerLine)
		}
	}
}

func splitLines(s string) []string {
	result := []string{}
	current := ""
	for _, r := range s {
		if r == '\n' {
			result = append(result, current)
			current = ""
		} else {
			current += string(r)
		}
	}
	result = append(result, current)
	return result
}
