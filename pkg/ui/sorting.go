package ui

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/charmbracelet/bubbles/list"
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

// SortResourceListNaturally sorts a slice of list items using natural sorting
func SortResourceListNaturally(items []list.Item) {
	sort.Slice(items, func(i, j int) bool {
		var nameI, nameJ string

		if item, ok := items[i].(resourceListItem); ok {
			nameI = item.resourceItem.GetID()
		} else if item, ok := items[i].(topicItem); ok {
			nameI = item.name
		}

		if item, ok := items[j].(resourceListItem); ok {
			nameJ = item.resourceItem.GetID()
		} else if item, ok := items[j].(topicItem); ok {
			nameJ = item.name
		}

		return NaturalLess(nameI, nameJ)
	})
}

// HighlightedResourceListItem wraps a resource item with search highlighting info
type HighlightedResourceListItem struct {
	resourceItem ResourceItem
	searchQuery  string
}

func (h HighlightedResourceListItem) FilterValue() string {
	return h.resourceItem.GetID()
}

// HighlightedTopicItem wraps a topic item with search highlighting info
type HighlightedTopicItem struct {
	name        string
	topic       api.Topic
	searchQuery string
}

func (h HighlightedTopicItem) FilterValue() string {
	return h.name
}

// HighlightSearchMatches highlights search query matches in text using lipgloss
func HighlightSearchMatches(text, searchQuery string) string {
	if searchQuery == "" {
		return text
	}

	// Define highlight style
	highlightStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("11")). // Bright yellow background
		Foreground(lipgloss.Color("0"))   // Black text

	// Case-insensitive regex for the search query
	pattern := "(?i)" + regexp.QuoteMeta(searchQuery)
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
	if resourceItem, ok := item.(resourceListItem); ok {
		return HighlightedResourceListItem{
			resourceItem: resourceItem.resourceItem,
			searchQuery:  searchQuery,
		}
	} else if topicItem, ok := item.(topicItem); ok {
		return HighlightedTopicItem{
			name:        topicItem.name,
			topic:       topicItem.topic,
			searchQuery: searchQuery,
		}
	}
	return item
}
