package search

import (
	"sort"
	"strings"
	"sync"
	"unicode"
)

// Performance optimization: cache normalized strings
var normalizeCache sync.Map // map[string]string

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Match represents a fuzzy search match with score and positions
type Match struct {
	Text       string  // The original text that was matched
	Score      float64 // Match score (higher is better)
	Positions  []int   // Character positions that matched the query
	Highlights []Range // Ranges to highlight in the text
}

// Range represents a range of characters for highlighting
type Range struct {
	Start int
	End   int
}

// Fuzzy performs fuzzy string matching
type Fuzzy struct {
	caseSensitive    bool
	normalizeSpaces  bool
	highlightMatches bool
	minScore         float64
}

// NewFuzzy creates a new fuzzy matcher with default settings
func NewFuzzy() *Fuzzy {
	return &Fuzzy{
		caseSensitive:    false,
		normalizeSpaces:  true,
		highlightMatches: true,
		minScore:         0.1,
	}
}

// SetCaseSensitive enables or disables case-sensitive matching
func (f *Fuzzy) SetCaseSensitive(enabled bool) *Fuzzy {
	f.caseSensitive = enabled
	return f
}

// SetNormalizeSpaces enables or disables space normalization
func (f *Fuzzy) SetNormalizeSpaces(enabled bool) *Fuzzy {
	f.normalizeSpaces = enabled
	return f
}

// SetHighlightMatches enables or disables match highlighting
func (f *Fuzzy) SetHighlightMatches(enabled bool) *Fuzzy {
	f.highlightMatches = enabled
	return f
}

// SetMinScore sets the minimum score threshold for matches
func (f *Fuzzy) SetMinScore(score float64) *Fuzzy {
	f.minScore = score
	return f
}

// Match performs fuzzy matching of query against text
func (f *Fuzzy) Match(query, text string) (*Match, bool) {
	if query == "" {
		return &Match{
			Text:      text,
			Score:     1.0,
			Positions: []int{},
		}, true
	}

	normalizedQuery := f.normalize(query)
	normalizedText := f.normalize(text)

	positions, score := f.calculateMatch(normalizedQuery, normalizedText)
	if score < f.minScore {
		return nil, false
	}

	match := &Match{
		Text:      text,
		Score:     score,
		Positions: positions,
	}

	if f.highlightMatches {
		match.Highlights = f.calculateHighlights(positions, len(text))
	}

	return match, true
}

// Search performs fuzzy search across multiple strings
func (f *Fuzzy) Search(query string, texts []string) []Match {
	if len(texts) == 0 {
		return []Match{}
	}

	// Pre-allocate slice with estimated capacity for better memory performance
	matches := make([]Match, 0, min(len(texts)/4, 50))

	for _, text := range texts {
		if match, ok := f.Match(query, text); ok {
			matches = append(matches, *match)
		}
	}

	// Sort by score (highest first)
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Score == matches[j].Score {
			// If scores are equal, prefer shorter strings
			return len(matches[i].Text) < len(matches[j].Text)
		}
		return matches[i].Score > matches[j].Score
	})

	return matches
}

// SearchWithLimit performs fuzzy search with a result limit for better performance
func (f *Fuzzy) SearchWithLimit(query string, texts []string, limit int) []Match {
	if len(texts) == 0 || limit <= 0 {
		return []Match{}
	}

	matches := make([]Match, 0, min(limit*2, 100)) // Buffer for better matches

	for _, text := range texts {
		if match, ok := f.Match(query, text); ok {
			matches = append(matches, *match)
		}
	}

	// Sort by score (highest first)
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Score == matches[j].Score {
			return len(matches[i].Text) < len(matches[j].Text)
		}
		return matches[i].Score > matches[j].Score
	})

	// Return limited results
	if len(matches) > limit {
		return matches[:limit]
	}
	return matches
}

// SearchAsync performs fuzzy search with channel-based results for real-time UX
func (f *Fuzzy) SearchAsync(query string, texts []string, results chan<- Match, done chan<- struct{}) {
	defer close(done)
	defer close(results)

	if len(texts) == 0 {
		return
	}

	for _, text := range texts {
		if match, ok := f.Match(query, text); ok {
			select {
			case results <- *match:
			default:
				// Channel full, skip to prevent blocking
			}
		}
	}
}

// SearchMap performs fuzzy search across a map of string keys
func (f *Fuzzy) SearchMap(query string, items map[string]any) []MapMatch {
	var matches []MapMatch

	for key, value := range items {
		if match, ok := f.Match(query, key); ok {
			matches = append(matches, MapMatch{
				Key:   key,
				Value: value,
				Match: *match,
			})
		}
	}

	// Sort by score (highest first)
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Match.Score == matches[j].Match.Score {
			return len(matches[i].Key) < len(matches[j].Key)
		}
		return matches[i].Match.Score > matches[j].Match.Score
	})

	return matches
}

// MapMatch represents a fuzzy search match in a map
type MapMatch struct {
	Key   string
	Value any
	Match Match
}

// normalize normalizes text for matching with caching for performance
func (f *Fuzzy) normalize(text string) string {
	// Create cache key based on text and normalization settings
	cacheKey := text
	if f.caseSensitive {
		cacheKey += ":cs"
	}
	if f.normalizeSpaces {
		cacheKey += ":ns"
	}

	// Check cache first
	if cached, ok := normalizeCache.Load(cacheKey); ok {
		return cached.(string)
	}

	// Perform normalization
	result := text
	if !f.caseSensitive {
		result = strings.ToLower(result)
	}

	if f.normalizeSpaces {
		result = strings.TrimSpace(result)
		result = strings.ReplaceAll(result, "\t", " ")
		// More efficient space collapse
		result = strings.Join(strings.Fields(result), " ")
	}

	// Cache the result
	normalizeCache.Store(cacheKey, result)
	return result
}

// calculateMatch calculates fuzzy match positions and score
func (f *Fuzzy) calculateMatch(query, text string) ([]int, float64) {
	if len(query) == 0 {
		return []int{}, 1.0
	}

	if len(text) == 0 {
		return []int{}, 0.0
	}

	// Simple fuzzy matching algorithm
	var positions []int
	queryIndex := 0
	consecutiveMatches := 0
	bestConsecutive := 0

	for textIndex, char := range text {
		if queryIndex < len(query) && rune(query[queryIndex]) == char {
			positions = append(positions, textIndex)
			queryIndex++
			consecutiveMatches++
			if consecutiveMatches > bestConsecutive {
				bestConsecutive = consecutiveMatches
			}
		} else {
			consecutiveMatches = 0
		}
	}

	// If we didn't match all query characters, it's not a match
	if queryIndex < len(query) {
		return []int{}, 0.0
	}

	// Calculate score based on various factors
	score := f.calculateScore(query, text, positions, bestConsecutive)

	return positions, score
}

// calculateScore calculates the match score
func (f *Fuzzy) calculateScore(query, text string, positions []int, bestConsecutive int) float64 {
	if len(positions) == 0 {
		return 0.0
	}

	// Base score: ratio of matched characters to query length
	baseScore := float64(len(positions)) / float64(len(query))

	// Bonus for consecutive matches
	consecutiveBonus := float64(bestConsecutive) / float64(len(query)) * 0.5

	// Bonus for matches at the beginning of the string
	startBonus := 0.0
	if len(positions) > 0 && positions[0] == 0 {
		startBonus = 0.2
	}

	// Bonus for shorter strings (all else being equal)
	lengthBonus := 0.0
	if len(text) > 0 {
		lengthBonus = float64(len(query)) / float64(len(text)) * 0.3
	}

	// Penalty for gaps between matches
	gapPenalty := 0.0
	if len(positions) > 1 {
		totalGaps := positions[len(positions)-1] - positions[0] + 1 - len(positions)
		gapPenalty = float64(totalGaps) / float64(len(text)) * 0.3
	}

	// Calculate final score
	score := baseScore + consecutiveBonus + startBonus + lengthBonus - gapPenalty

	// Ensure score is between 0 and 1
	if score > 1.0 {
		score = 1.0
	}
	if score < 0.0 {
		score = 0.0
	}

	return score
}

// calculateHighlights calculates highlight ranges from match positions
func (f *Fuzzy) calculateHighlights(positions []int, textLength int) []Range {
	if len(positions) == 0 {
		return []Range{}
	}

	var highlights []Range
	start := positions[0]
	end := positions[0] + 1

	for i := 1; i < len(positions); i++ {
		if positions[i] == end {
			// Consecutive positions, extend current range
			end++
		} else {
			// Gap found, close current range and start new one
			highlights = append(highlights, Range{Start: start, End: end})
			start = positions[i]
			end = positions[i] + 1
		}
	}

	// Close the final range
	highlights = append(highlights, Range{Start: start, End: end})

	return highlights
}

// HighlightString applies highlighting to a string based on match positions
func (f *Fuzzy) HighlightString(text string, highlights []Range, startTag, endTag string) string {
	if len(highlights) == 0 {
		return text
	}

	var result strings.Builder
	lastEnd := 0

	for _, highlight := range highlights {
		// Add text before highlight
		if highlight.Start > lastEnd {
			result.WriteString(text[lastEnd:highlight.Start])
		}

		// Add highlighted text
		result.WriteString(startTag)
		result.WriteString(text[highlight.Start:highlight.End])
		result.WriteString(endTag)

		lastEnd = highlight.End
	}

	// Add remaining text
	if lastEnd < len(text) {
		result.WriteString(text[lastEnd:])
	}

	return result.String()
}

// Searchable interface for types that can be searched
type Searchable interface {
	SearchText() string
	DisplayText() string
}

// SearchItems performs fuzzy search on items implementing Searchable
func (f *Fuzzy) SearchItems(query string, items []Searchable) []SearchableMatch {
	var matches []SearchableMatch

	for _, item := range items {
		if match, ok := f.Match(query, item.SearchText()); ok {
			matches = append(matches, SearchableMatch{
				Item:  item,
				Match: *match,
			})
		}
	}

	// Sort by score
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Match.Score == matches[j].Match.Score {
			return len(matches[i].Item.SearchText()) < len(matches[j].Item.SearchText())
		}
		return matches[i].Match.Score > matches[j].Match.Score
	})

	return matches
}

// SearchableMatch represents a match on a Searchable item
type SearchableMatch struct {
	Item  Searchable
	Match Match
}

// SmartMatch performs intelligent fuzzy matching with word boundary awareness
func (f *Fuzzy) SmartMatch(query, text string) (*Match, bool) {
	// First try exact substring match
	if match, ok := f.exactSubstringMatch(query, text); ok {
		return match, true
	}

	// Try word boundary matching
	if match, ok := f.wordBoundaryMatch(query, text); ok {
		return match, true
	}

	// Fall back to regular fuzzy match
	return f.Match(query, text)
}

// exactSubstringMatch checks for exact substring matches
func (f *Fuzzy) exactSubstringMatch(query, text string) (*Match, bool) {
	normalizedQuery := f.normalize(query)
	normalizedText := f.normalize(text)

	index := strings.Index(normalizedText, normalizedQuery)
	if index == -1 {
		return nil, false
	}

	// Create positions for the substring match
	positions := make([]int, len(normalizedQuery))
	for i := range positions {
		positions[i] = index + i
	}

	// High score for exact matches
	score := 0.9
	if index == 0 {
		score = 1.0 // Perfect score for matches at the beginning
	}

	match := &Match{
		Text:      text,
		Score:     score,
		Positions: positions,
	}

	if f.highlightMatches {
		match.Highlights = []Range{{Start: index, End: index + len(normalizedQuery)}}
	}

	return match, true
}

// wordBoundaryMatch performs matching with word boundary awareness
func (f *Fuzzy) wordBoundaryMatch(query, text string) (*Match, bool) {
	normalizedQuery := f.normalize(query)
	normalizedText := f.normalize(text)

	words := f.splitIntoWords(normalizedText)
	queryWords := f.splitIntoWords(normalizedQuery)

	var allPositions []int
	totalScore := 0.0
	matchedWords := 0

	for _, queryWord := range queryWords {
		bestWordScore := 0.0
		var bestPositions []int

		for _, word := range words {
			if positions, score := f.calculateMatch(queryWord.Text, word.Text); score > bestWordScore {
				bestWordScore = score
				// Adjust positions to global text positions
				adjustedPositions := make([]int, len(positions))
				for i, pos := range positions {
					adjustedPositions[i] = word.Start + pos
				}
				bestPositions = adjustedPositions
			}
		}

		if bestWordScore > 0 {
			allPositions = append(allPositions, bestPositions...)
			totalScore += bestWordScore
			matchedWords++
		}
	}

	if matchedWords == 0 {
		return nil, false
	}

	// Average score across matched words
	avgScore := totalScore / float64(len(queryWords))

	// Bonus for matching all query words
	if matchedWords == len(queryWords) {
		avgScore *= 1.2
	}

	if avgScore > 1.0 {
		avgScore = 1.0
	}

	// Sort positions
	sort.Ints(allPositions)

	match := &Match{
		Text:      text,
		Score:     avgScore,
		Positions: allPositions,
	}

	if f.highlightMatches {
		match.Highlights = f.calculateHighlights(allPositions, len(text))
	}

	return match, avgScore >= f.minScore
}

// Word represents a word with its position in the original text
type Word struct {
	Text  string
	Start int
	End   int
}

// splitIntoWords splits text into words with their positions
func (f *Fuzzy) splitIntoWords(text string) []Word {
	var words []Word
	var currentWord strings.Builder
	wordStart := 0
	inWord := false

	for i, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if !inWord {
				wordStart = i
				inWord = true
			}
			currentWord.WriteRune(r)
		} else {
			if inWord {
				words = append(words, Word{
					Text:  currentWord.String(),
					Start: wordStart,
					End:   i,
				})
				currentWord.Reset()
				inWord = false
			}
		}
	}

	// Handle final word
	if inWord {
		words = append(words, Word{
			Text:  currentWord.String(),
			Start: wordStart,
			End:   len(text),
		})
	}

	return words
}
