// Package search provides intelligent fuzzy search with context-aware ranking
package search

import (
	"regexp"
	"sort"
	"strings"
	"unicode"
)

// SmartFuzzy provides context-aware fuzzy search optimized for taskopen
type SmartFuzzy struct {
	*Fuzzy
	contextBoosts map[string]float64
	acronymBoost  float64
	pathBoost     float64
	urlBoost      float64
	recencyBoost  float64
	frequencyData map[string]int
}

// SetContextBoosts sets boost values for different contexts
func (sf *SmartFuzzy) SetContextBoosts(boosts map[string]float64) *SmartFuzzy {
	sf.contextBoosts = boosts
	return sf
}

// SetFrequencyData provides usage frequency data for items
func (sf *SmartFuzzy) SetFrequencyData(data map[string]int) *SmartFuzzy {
	sf.frequencyData = data
	return sf
}

// SmartSearch performs intelligent fuzzy search with context awareness
func (sf *SmartFuzzy) SmartSearch(query string, items []SmartItem) []SmartMatch {
	var matches []SmartMatch

	for _, item := range items {
		if match, ok := sf.smartMatch(query, item); ok {
			matches = append(matches, *match)
		}
	}

	// Sort by enhanced score
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].EnhancedScore == matches[j].EnhancedScore {
			// Secondary sort by original score, then length
			if matches[i].Match.Score == matches[j].Match.Score {
				return len(matches[i].Item.GetText()) < len(matches[j].Item.GetText())
			}
			return matches[i].Match.Score > matches[j].Match.Score
		}
		return matches[i].EnhancedScore > matches[j].EnhancedScore
	})

	return matches
}

// SmartItem represents an item that can be intelligently searched
type SmartItem interface {
	GetText() string
	GetContext() string
	GetType() ItemType
	GetMetadata() map[string]interface{}
}

// ItemType represents different types of searchable items
type ItemType int

const (
	TypeAction ItemType = iota
	TypeAnnotation
	TypeTask
	TypeFile
	TypeURL
	TypeNote
)

// String returns the string representation of ItemType
func (it ItemType) String() string {
	switch it {
	case TypeAction:
		return "action"
	case TypeAnnotation:
		return "annotation"
	case TypeTask:
		return "task"
	case TypeFile:
		return "file"
	case TypeURL:
		return "url"
	case TypeNote:
		return "note"
	default:
		return "unknown"
	}
}

// SmartMatch represents an enhanced fuzzy match with context scoring
type SmartMatch struct {
	Item          SmartItem
	Match         Match
	EnhancedScore float64
	ContextScore  float64
	BoostFactors  map[string]float64
}

// TaskAnnotation implements SmartItem for taskwarrior annotations
type TaskAnnotation struct {
	Text        string
	Description string
	TaskID      string
	Project     string
	Tags        []string
	Priority    string
	Context     string
}

func (ta *TaskAnnotation) GetText() string    { return ta.Text }
func (ta *TaskAnnotation) GetContext() string { return ta.Context }
func (ta *TaskAnnotation) GetType() ItemType  { return TypeAnnotation }
func (ta *TaskAnnotation) GetMetadata() map[string]interface{} {
	return map[string]interface{}{
		"task_id":     ta.TaskID,
		"project":     ta.Project,
		"tags":        ta.Tags,
		"priority":    ta.Priority,
		"description": ta.Description,
	}
}

// ActionItem implements SmartItem for taskopen actions
type ActionItem struct {
	Name        string
	Command     string
	Target      string
	Regex       string
	Description string
	Context     string
	UsageCount  int
	LastUsed    int64
}

func (ai *ActionItem) GetText() string    { return ai.Name }
func (ai *ActionItem) GetContext() string { return ai.Context }
func (ai *ActionItem) GetType() ItemType  { return TypeAction }
func (ai *ActionItem) GetMetadata() map[string]interface{} {
	return map[string]interface{}{
		"command":     ai.Command,
		"target":      ai.Target,
		"regex":       ai.Regex,
		"description": ai.Description,
		"usage_count": ai.UsageCount,
		"last_used":   ai.LastUsed,
	}
}

// smartMatch performs intelligent matching with context awareness
func (sf *SmartFuzzy) smartMatch(query string, item SmartItem) (*SmartMatch, bool) {
	// Get base fuzzy match
	baseMatch, ok := sf.SmartMatch(query, item.GetText())
	if !ok {
		return nil, false
	}

	// Calculate enhanced score with context
	boostFactors := sf.calculateBoostFactors(query, item)
	enhancedScore := sf.calculateEnhancedScore(baseMatch.Score, boostFactors)

	return &SmartMatch{
		Item:          item,
		Match:         *baseMatch,
		EnhancedScore: enhancedScore,
		ContextScore:  boostFactors["context"],
		BoostFactors:  boostFactors,
	}, true
}

// calculateBoostFactors determines various boost factors for an item
func (sf *SmartFuzzy) calculateBoostFactors(query string, item SmartItem) map[string]float64 {
	factors := make(map[string]float64)

	// Context boost
	if boost, exists := sf.contextBoosts[item.GetContext()]; exists {
		factors["context"] = boost
	}

	// Type-specific boosts
	factors["type"] = sf.getTypeBoost(item.GetType(), query)

	// Acronym boost - check if query matches acronym of text
	if sf.isAcronymMatch(query, item.GetText()) {
		factors["acronym"] = sf.acronymBoost
	}

	// Path/file boost for file-like patterns
	if sf.isPathLike(item.GetText()) {
		factors["path"] = sf.pathBoost
	}

	// URL boost for URL patterns
	if sf.isURLLike(item.GetText()) {
		factors["url"] = sf.urlBoost
	}

	// Frequency boost based on usage
	if count, exists := sf.frequencyData[item.GetText()]; exists && count > 0 {
		factors["frequency"] = float64(count) / 100.0 // Normalize to reasonable range
	}

	// Exact match boost
	if strings.EqualFold(query, item.GetText()) {
		factors["exact"] = 0.5
	}

	// Prefix match boost
	if strings.HasPrefix(strings.ToLower(item.GetText()), strings.ToLower(query)) {
		factors["prefix"] = 0.3
	}

	// Word boundary boost
	if sf.matchesWordBoundary(query, item.GetText()) {
		factors["word_boundary"] = 0.2
	}

	return factors
}

// calculateEnhancedScore combines base score with boost factors
func (sf *SmartFuzzy) calculateEnhancedScore(baseScore float64, factors map[string]float64) float64 {
	enhanced := baseScore

	// Apply multiplicative boosts
	for _, boost := range factors {
		enhanced += boost
	}

	// Ensure score doesn't exceed 1.0
	if enhanced > 1.0 {
		enhanced = 1.0
	}

	return enhanced
}

// getTypeBoost returns a boost based on item type and query characteristics
func (sf *SmartFuzzy) getTypeBoost(itemType ItemType, query string) float64 {
	switch itemType {
	case TypeAction:
		// Boost actions for single-word queries
		if !strings.Contains(query, " ") {
			return 0.15
		}
	case TypeFile:
		// Boost files for queries that look like file extensions
		if strings.HasPrefix(query, ".") || strings.Contains(query, "/") {
			return 0.2
		}
	case TypeURL:
		// Boost URLs for queries with URL-like patterns
		if strings.Contains(query, "http") || strings.Contains(query, "www") || strings.Contains(query, ".com") {
			return 0.25
		}
	case TypeAnnotation:
		// Boost annotations for longer queries
		if len(query) > 5 {
			return 0.1
		}
	}
	return 0.0
}

// isAcronymMatch checks if query matches the acronym of text
func (sf *SmartFuzzy) isAcronymMatch(query, text string) bool {
	if len(query) == 0 {
		return false
	}

	// Extract potential acronym from text
	words := strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})

	if len(words) < 2 {
		return false
	}

	var acronym strings.Builder
	for _, word := range words {
		if len(word) > 0 {
			acronym.WriteRune(unicode.ToLower(rune(word[0])))
		}
	}

	return strings.EqualFold(query, acronym.String())
}

// isPathLike checks if text looks like a file path
func (sf *SmartFuzzy) isPathLike(text string) bool {
	// Simple heuristics for path detection
	return strings.Contains(text, "/") ||
		strings.Contains(text, "\\") ||
		strings.Contains(text, ".") && len(strings.Split(text, ".")) > 1
}

// isURLLike checks if text looks like a URL
func (sf *SmartFuzzy) isURLLike(text string) bool {
	urlPatterns := []string{
		"http://", "https://", "ftp://", "www.",
		".com", ".org", ".net", ".edu", ".gov",
	}

	lowerText := strings.ToLower(text)
	for _, pattern := range urlPatterns {
		if strings.Contains(lowerText, pattern) {
			return true
		}
	}
	return false
}

// matchesWordBoundary checks if query matches at word boundaries
func (sf *SmartFuzzy) matchesWordBoundary(query, text string) bool {
	if len(query) == 0 {
		return false
	}

	// Create regex pattern for word boundary matching
	pattern := `\b` + regexp.QuoteMeta(strings.ToLower(query))
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false
	}

	return re.MatchString(strings.ToLower(text))
}

// SearchTaskAnnotations performs optimized search for taskwarrior annotations
func (sf *SmartFuzzy) SearchTaskAnnotations(query string, annotations []TaskAnnotation) []SmartMatch {
	items := make([]SmartItem, len(annotations))
	for i, annotation := range annotations {
		items[i] = &annotation
	}
	return sf.SmartSearch(query, items)
}

// SearchActions performs optimized search for taskopen actions
func (sf *SmartFuzzy) SearchActions(query string, actions []ActionItem) []SmartMatch {
	items := make([]SmartItem, len(actions))
	for i, action := range actions {
		items[i] = &action
	}
	return sf.SmartSearch(query, items)
}

// MultiFieldSearch searches across multiple fields of an item
func (sf *SmartFuzzy) MultiFieldSearch(query string, items []MultiFieldItem) []SmartMatch {
	var matches []SmartMatch

	for _, item := range items {
		bestScore := 0.0
		var bestMatch *Match

		// Search across all fields
		for fieldName, fieldValue := range item.GetSearchFields() {
			if match, ok := sf.SmartMatch(query, fieldValue); ok {
				// Weight fields differently
				weightedScore := match.Score * sf.getFieldWeight(fieldName)
				if weightedScore > bestScore {
					bestScore = weightedScore
					bestMatch = match
				}
			}
		}

		if bestMatch != nil {
			// Calculate boost factors
			boostFactors := sf.calculateBoostFactors(query, item)
			enhancedScore := sf.calculateEnhancedScore(bestScore, boostFactors)

			matches = append(matches, SmartMatch{
				Item:          item,
				Match:         *bestMatch,
				EnhancedScore: enhancedScore,
				ContextScore:  boostFactors["context"],
				BoostFactors:  boostFactors,
			})
		}
	}

	// Sort matches
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].EnhancedScore > matches[j].EnhancedScore
	})

	return matches
}

// MultiFieldItem represents an item with multiple searchable fields
type MultiFieldItem interface {
	SmartItem
	GetSearchFields() map[string]string
}

// getFieldWeight returns importance weight for different fields
func (sf *SmartFuzzy) getFieldWeight(fieldName string) float64 {
	weights := map[string]float64{
		"name":        1.0,
		"title":       1.0,
		"description": 0.8,
		"tags":        0.6,
		"project":     0.7,
		"annotation":  0.9,
		"command":     0.5,
		"path":        0.4,
		"content":     0.3,
	}

	if weight, exists := weights[fieldName]; exists {
		return weight
	}
	return 0.5 // Default weight
}

// ContextualSearch performs search with current taskwarrior context awareness
type ContextualSearch struct {
	*SmartFuzzy
	currentProject string
	currentTags    []string
	currentContext string
}

// Search performs contextual search with current taskwarrior state
func (cs *ContextualSearch) Search(query string, items []SmartItem) []SmartMatch {
	// Boost items that match current context
	contextBoosts := make(map[string]float64)

	// Boost current project
	if cs.currentProject != "" {
		contextBoosts[cs.currentProject] = 0.3
	}

	// Boost current context
	if cs.currentContext != "" {
		contextBoosts[cs.currentContext] = 0.2
	}

	cs.SetContextBoosts(contextBoosts)
	return cs.SmartSearch(query, items)
}

// PredictiveSearch suggests completions as user types
type PredictiveSearch struct {
	*SmartFuzzy
	index       map[string][]SmartItem
	minQueryLen int
}

// BuildIndex creates search index for fast predictions
func (ps *PredictiveSearch) BuildIndex(items []SmartItem) {
	ps.index = make(map[string][]SmartItem)

	for _, item := range items {
		text := strings.ToLower(item.GetText())

		// Index all prefixes
		for i := ps.minQueryLen; i <= len(text); i++ {
			prefix := text[:i]
			ps.index[prefix] = append(ps.index[prefix], item)
		}

		// Index words
		words := strings.Fields(text)
		for _, word := range words {
			if len(word) >= ps.minQueryLen {
				ps.index[word] = append(ps.index[word], item)
			}
		}
	}
}

// Predict returns likely matches for partial queries
func (ps *PredictiveSearch) Predict(query string, maxResults int) []SmartMatch {
	if len(query) < ps.minQueryLen {
		return nil
	}

	lowerQuery := strings.ToLower(query)
	candidateItems := make(map[SmartItem]bool)

	// Find items from index
	if items, exists := ps.index[lowerQuery]; exists {
		for _, item := range items {
			candidateItems[item] = true
		}
	}

	// Convert to slice for searching
	items := make([]SmartItem, 0, len(candidateItems))
	for item := range candidateItems {
		items = append(items, item)
	}

	// Perform smart search on candidates
	matches := ps.SmartSearch(query, items)

	// Limit results
	if len(matches) > maxResults {
		matches = matches[:maxResults]
	}

	return matches
}
