package components

import (
	"sort"
	"strings"
)

// FuzzyMatch represents a fuzzy match result
type FuzzyMatch struct {
	Text  string
	Score int
	Index int
}

// FuzzyMatcher provides fuzzy matching functionality
type FuzzyMatcher struct {
	caseSensitive bool
}

// NewFuzzyMatcher creates a new fuzzy matcher
func NewFuzzyMatcher(caseSensitive bool) *FuzzyMatcher {
	return &FuzzyMatcher{
		caseSensitive: caseSensitive,
	}
}

// Match performs fuzzy matching on a list of candidates
func (fm *FuzzyMatcher) Match(query string, candidates []string, maxResults int) []FuzzyMatch {
	if query == "" {
		// Return all candidates when no query
		results := make([]FuzzyMatch, 0, len(candidates))
		for i, candidate := range candidates {
			results = append(results, FuzzyMatch{
				Text:  candidate,
				Score: 0,
				Index: i,
			})
		}
		return results
	}

	var matches []FuzzyMatch
	
	// Normalize query for case-insensitive matching
	searchQuery := query
	if !fm.caseSensitive {
		searchQuery = strings.ToLower(query)
	}

	for i, candidate := range candidates {
		score := fm.calculateScore(searchQuery, candidate)
		if score > 0 {
			matches = append(matches, FuzzyMatch{
				Text:  candidate,
				Score: score,
				Index: i,
			})
		}
	}

	// Sort by score (higher is better)
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Score == matches[j].Score {
			// If scores are equal, prefer shorter strings
			return len(matches[i].Text) < len(matches[j].Text)
		}
		return matches[i].Score > matches[j].Score
	})

	// Limit results
	if maxResults > 0 && len(matches) > maxResults {
		matches = matches[:maxResults]
	}

	return matches
}

// calculateScore calculates the fuzzy match score for a candidate
func (fm *FuzzyMatcher) calculateScore(query, candidate string) int {
	if query == "" {
		return 1
	}

	// Normalize candidate for case-insensitive matching
	searchCandidate := candidate
	if !fm.caseSensitive {
		searchCandidate = strings.ToLower(candidate)
	}

	// Exact match gets highest score
	if query == searchCandidate {
		return 10000
	}

	// Prefix match gets high score, prefer longer matches for same prefix
	if strings.HasPrefix(searchCandidate, query) {
		// Boost score for longer matches when query is short
		lengthBonus := 0
		if len(query) <= 3 {
			lengthBonus = len(candidate) * 2 // Prefer longer matches for short queries
		}
		return 9000 + lengthBonus - (len(candidate) - len(query))
	}

	// Word boundary match (starts with query after space, dash, underscore)
	if fm.hasWordBoundaryMatch(query, searchCandidate) {
		return 8000 - len(candidate)
	}

	// Contains match gets medium score
	if strings.Contains(searchCandidate, query) {
		// Bonus for matches closer to the beginning
		index := strings.Index(searchCandidate, query)
		positionBonus := 100 - index
		if positionBonus < 0 {
			positionBonus = 0
		}
		return 7000 + positionBonus - len(candidate)
	}

	// Subsequence match (characters appear in order but not necessarily consecutive)
	if score := fm.subsequenceScore(query, searchCandidate); score > 0 {
		return 5000 + score
	}

	// Character frequency match (all characters present)
	if fm.hasAllCharacters(query, searchCandidate) {
		return 2000
	}

	return 0
}

// hasWordBoundaryMatch checks if query matches at word boundaries
func (fm *FuzzyMatcher) hasWordBoundaryMatch(query, candidate string) bool {
	separators := []string{" ", "-", "_", "."}
	
	for _, sep := range separators {
		parts := strings.Split(candidate, sep)
		for _, part := range parts {
			if strings.HasPrefix(part, query) {
				return true
			}
		}
	}
	
	return false
}

// subsequenceScore calculates score for subsequence matching
func (fm *FuzzyMatcher) subsequenceScore(query, candidate string) int {
	if len(query) == 0 {
		return 0
	}

	queryRunes := []rune(query)
	candidateRunes := []rune(candidate)
	
	queryIndex := 0
	score := 0
	consecutiveMatches := 0
	
	for i, candidateRune := range candidateRunes {
		if queryIndex < len(queryRunes) && candidateRune == queryRunes[queryIndex] {
			queryIndex++
			consecutiveMatches++
			
			// Bonus for consecutive matches
			if consecutiveMatches > 1 {
				score += consecutiveMatches * 2
			} else {
				score += 1
			}
			
			// Bonus for matches at the beginning
			if i == queryIndex-1 {
				score += 5
			}
		} else {
			consecutiveMatches = 0
		}
	}
	
	// Only return score if all query characters were matched
	if queryIndex == len(queryRunes) {
		return score
	}
	
	return 0
}

// hasAllCharacters checks if all characters in query exist in candidate
func (fm *FuzzyMatcher) hasAllCharacters(query, candidate string) bool {
	queryChars := make(map[rune]int)
	candidateChars := make(map[rune]int)
	
	// Count characters in query
	for _, char := range query {
		queryChars[char]++
	}
	
	// Count characters in candidate
	for _, char := range candidate {
		candidateChars[char]++
	}
	
	// Check if all query characters are present in sufficient quantity
	for char, count := range queryChars {
		if candidateChars[char] < count {
			return false
		}
	}
	
	return true
}

// GetBestMatch returns the best fuzzy match from candidates
func (fm *FuzzyMatcher) GetBestMatch(query string, candidates []string) string {
	matches := fm.Match(query, candidates, 1)
	if len(matches) > 0 {
		return matches[0].Text
	}
	return ""
}

// GetMatchedStrings returns just the matched strings (without scores)
func (fm *FuzzyMatcher) GetMatchedStrings(query string, candidates []string, maxResults int) []string {
	matches := fm.Match(query, candidates, maxResults)
	results := make([]string, len(matches))
	for i, match := range matches {
		results[i] = match.Text
	}
	return results
}