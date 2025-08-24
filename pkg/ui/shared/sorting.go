package shared

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
)

// NaturalSort sorts strings in natural order (numbers in logical order)
type NaturalSort []string

func (s NaturalSort) Len() int      { return len(s) }
func (s NaturalSort) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s NaturalSort) Less(i, j int) bool {
	return NaturalLess(s[i], s[j])
}

// NaturalLess compares two strings naturally (numeric parts as numbers)
func NaturalLess(a, b string) bool {
	ai, bi := 0, 0

	for ai < len(a) && bi < len(b) {
		ac, bc := rune(a[ai]), rune(b[bi])

		// If both are digits, compare numerically
		if unicode.IsDigit(ac) && unicode.IsDigit(bc) {
			// Extract the full number from both strings
			aNum := ""
			for ai < len(a) && unicode.IsDigit(rune(a[ai])) {
				aNum += string(a[ai])
				ai++
			}

			bNum := ""
			for bi < len(b) && unicode.IsDigit(rune(b[bi])) {
				bNum += string(b[bi])
				bi++
			}

			// Convert to numbers and compare
			aVal, errA := strconv.Atoi(aNum)
			bVal, errB := strconv.Atoi(bNum)

			if errA == nil && errB == nil {
				if aVal != bVal {
					return aVal < bVal
				}
			} else {
				// If conversion fails, compare as strings
				if aNum != bNum {
					return aNum < bNum
				}
			}
		} else {
			// Compare characters normally
			if ac != bc {
				return ac < bc
			}
			ai++
			bi++
		}
	}

	// If one string is a prefix of another, the shorter one comes first
	return len(a) < len(b)
}

// SortOrder represents different sorting orders
type SortOrder int

const (
	SortAscending SortOrder = iota
	SortDescending
)

// SortOptions contains options for sorting operations
type SortOptions struct {
	Order      SortOrder
	CaseSensitive bool
	Natural    bool
	ColumnIndex int
}

// DefaultSortOptions returns default sorting options
func DefaultSortOptions() SortOptions {
	return SortOptions{
		Order:         SortAscending,
		CaseSensitive: false,
		Natural:       true,
		ColumnIndex:   0,
	}
}

// SortResourceListNaturally sorts a slice of list items using natural sorting
func SortResourceListNaturally(items []list.Item) {
	sort.Slice(items, func(i, j int) bool {
		nameI := getItemName(items[i])
		nameJ := getItemName(items[j])
		return NaturalLess(nameI, nameJ)
	})
}

// SortResourceListWithOptions sorts a slice of list items with the given options
func SortResourceListWithOptions(items []list.Item, options SortOptions) {
	sort.Slice(items, func(i, j int) bool {
		nameI := getItemName(items[i])
		nameJ := getItemName(items[j])
		
		var result bool
		if options.Natural {
			result = NaturalLess(nameI, nameJ)
		} else {
			if options.CaseSensitive {
				result = nameI < nameJ
			} else {
				result = strings.ToLower(nameI) < strings.ToLower(nameJ)
			}
		}
		
		if options.Order == SortDescending {
			result = !result
		}
		
		return result
	})
}

// getItemName extracts the name from different types of list items
func getItemName(item list.Item) string {
	switch v := item.(type) {
	case ResourceListItem:
		return v.ResourceItem.GetID()
	case TopicListItem:
		return v.Name
	case ConsumerGroupListItem:
		return v.GroupID
	case MessageListItem:
		return v.GetID()
	case FilterableItem:
		return v.FilterValue()
	default:
		// Fallback to string representation
		return ""
	}
}

// SortTableRowsNaturally sorts a slice of table rows using natural sorting
func SortTableRowsNaturally(rows []table.Row) {
	SortTableRowsWithOptions(rows, DefaultSortOptions())
}

// SortTableRowsWithOptions sorts a slice of table rows with the given options
func SortTableRowsWithOptions(rows []table.Row, options SortOptions) {
	sort.Slice(rows, func(i, j int) bool {
		// Ensure both rows have enough columns
		if len(rows[i]) <= options.ColumnIndex || len(rows[j]) <= options.ColumnIndex {
			return false
		}
		
		valueI := rows[i][options.ColumnIndex]
		valueJ := rows[j][options.ColumnIndex]
		
		var result bool
		if options.Natural {
			result = NaturalLess(valueI, valueJ)
		} else {
			if options.CaseSensitive {
				result = valueI < valueJ
			} else {
				result = strings.ToLower(valueI) < strings.ToLower(valueJ)
			}
		}
		
		if options.Order == SortDescending {
			result = !result
		}
		
		return result
	})
}

// SortStringSlice sorts a string slice with the given options
func SortStringSlice(items []string, options SortOptions) {
	sort.Slice(items, func(i, j int) bool {
		var result bool
		if options.Natural {
			result = NaturalLess(items[i], items[j])
		} else {
			if options.CaseSensitive {
				result = items[i] < items[j]
			} else {
				result = strings.ToLower(items[i]) < strings.ToLower(items[j])
			}
		}
		
		if options.Order == SortDescending {
			result = !result
		}
		
		return result
	})
}

// Highlighting functionality

// HighlightedResourceListItem wraps a resource item with search highlighting info
type HighlightedResourceListItem struct {
	ResourceItem ResourceItem
	SearchQuery  string
}

func (h HighlightedResourceListItem) FilterValue() string {
	return h.ResourceItem.GetID()
}

func (h HighlightedResourceListItem) GetID() string {
	return h.ResourceItem.GetID()
}

// HighlightedTopicItem wraps a topic item with search highlighting info
type HighlightedTopicItem struct {
	Name        string
	Topic       api.Topic
	SearchQuery string
}

func (h HighlightedTopicItem) FilterValue() string {
	return h.Name
}

func (h HighlightedTopicItem) GetID() string {
	return h.Name
}

// HighlightConfig contains options for text highlighting
type HighlightConfig struct {
	BackgroundColor string
	ForegroundColor string
	Bold            bool
	CaseSensitive   bool
	UseRegex        bool
}

// DefaultHighlightConfig returns default highlighting configuration
func DefaultHighlightConfig() HighlightConfig {
	return HighlightConfig{
		BackgroundColor: "",   // No background
		ForegroundColor: "205", // Pink/magenta for highlighting
		Bold:            true, // Bold text for emphasis
		CaseSensitive:   false,
		UseRegex:        false,
	}
}

// SearchHighlightConfig returns highlighting configuration for search matches
// Only changes font color, no background highlighting
func SearchHighlightConfig() HighlightConfig {
	return HighlightConfig{
		BackgroundColor: "",   // No background
		ForegroundColor: "205", // Pink/magenta for highlighting
		Bold:            true,  // Bold text for emphasis
		CaseSensitive:   false,
		UseRegex:        false,
	}
}

// HighlightSearchMatches highlights search query matches in text using lipgloss
func HighlightSearchMatches(text, searchQuery string) string {
	return HighlightSearchMatchesWithConfig(text, searchQuery, SearchHighlightConfig())
}

// HighlightSearchMatchesWithConfig highlights search matches with custom configuration
func HighlightSearchMatchesWithConfig(text, searchQuery string, config HighlightConfig) string {
	if searchQuery == "" {
		return text
	}

	// Define highlight style
	highlightStyle := lipgloss.NewStyle()
	
	// Only set background if it's not empty
	if config.BackgroundColor != "" {
		highlightStyle = highlightStyle.Background(lipgloss.Color(config.BackgroundColor))
	}
	
	// Set foreground color
	highlightStyle = highlightStyle.Foreground(lipgloss.Color(config.ForegroundColor))
	
	if config.Bold {
		highlightStyle = highlightStyle.Bold(true)
	}

	var pattern string
	if config.UseRegex {
		pattern = searchQuery
	} else {
		pattern = regexp.QuoteMeta(searchQuery)
	}
	
	if !config.CaseSensitive {
		pattern = "(?i)" + pattern
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return text // Return original text if regex is invalid
	}

	// Find all matches and their positions
	matches := re.FindAllStringIndex(text, -1)
	if len(matches) == 0 {
		return text
	}

	// Build the highlighted string
	var result strings.Builder
	lastEnd := 0

	for _, match := range matches {
		start, end := match[0], match[1]

		// Add text before the match
		result.WriteString(text[lastEnd:start])

		// Add highlighted match
		matchText := text[start:end]
		result.WriteString(highlightStyle.Render(matchText))

		lastEnd = end
	}

	// Add remaining text after the last match
	result.WriteString(text[lastEnd:])

	return result.String()
}

// CreateHighlightedItem creates a highlighted version of a list item
func CreateHighlightedItem(item list.Item, searchQuery string) list.Item {
	switch v := item.(type) {
	case ResourceListItem:
		return HighlightedResourceListItem{
			ResourceItem: v.ResourceItem,
			SearchQuery:  searchQuery,
		}
	case TopicListItem:
		return HighlightedTopicItem{
			Name:        v.Name,
			Topic:       v.Topic,
			SearchQuery: searchQuery,
		}
	default:
		return item
	}
}

// RemoveHighlighting removes highlighting from text
func RemoveHighlighting(text string) string {
	// This is a simple implementation that removes ANSI escape sequences
	// In a real implementation, you might want a more sophisticated approach
	ansiEscape := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return ansiEscape.ReplaceAllString(text, "")
}

// FilterItems filters a slice of items based on a search query
func FilterItems(items []list.Item, query string, caseSensitive bool) []list.Item {
	if query == "" {
		return items
	}
	
	var filtered []list.Item
	queryToMatch := query
	if !caseSensitive {
		queryToMatch = strings.ToLower(query)
	}
	
	for _, item := range items {
		name := getItemName(item)
		nameToMatch := name
		if !caseSensitive {
			nameToMatch = strings.ToLower(name)
		}
		
		if strings.Contains(nameToMatch, queryToMatch) {
			filtered = append(filtered, item)
		}
	}
	
	return filtered
}

// FuzzyMatch performs fuzzy matching between a query and text
func FuzzyMatch(query, text string, caseSensitive bool) bool {
	if query == "" {
		return true
	}
	
	if !caseSensitive {
		query = strings.ToLower(query)
		text = strings.ToLower(text)
	}
	
	queryIndex := 0
	for _, char := range text {
		if queryIndex < len(query) && rune(query[queryIndex]) == char {
			queryIndex++
		}
	}
	
	return queryIndex == len(query)
}

// FuzzyFilterItems filters items using fuzzy matching
func FuzzyFilterItems(items []list.Item, query string, caseSensitive bool) []list.Item {
	if query == "" {
		return items
	}
	
	var filtered []list.Item
	for _, item := range items {
		name := getItemName(item)
		if FuzzyMatch(query, name, caseSensitive) {
			filtered = append(filtered, item)
		}
	}
	
	return filtered
}