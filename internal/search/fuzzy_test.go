package search

import (
	"fmt"
	"strings"
	"testing"
)

func TestFuzzy_BasicMatch(t *testing.T) {
	fuzzy := NewFuzzy()

	// Test exact match
	match, ok := fuzzy.Match("hello", "hello")
	if !ok {
		t.Error("Expected exact match to succeed")
	}
	if match.Score != 1.0 {
		t.Errorf("Expected exact match score of 1.0, got %f", match.Score)
	}

	// Test substring match
	match, ok = fuzzy.Match("ell", "hello")
	if !ok {
		t.Error("Expected substring match to succeed")
	}
	if match.Score <= 0 {
		t.Errorf("Expected positive score, got %f", match.Score)
	}

	// Test no match
	_, ok = fuzzy.Match("xyz", "hello")
	if ok {
		t.Error("Expected no match for non-matching strings")
	}
}

func TestFuzzy_CaseInsensitive(t *testing.T) {
	fuzzy := NewFuzzy().SetCaseSensitive(false)

	_, ok := fuzzy.Match("HELLO", "hello")
	if !ok {
		t.Error("Expected case-insensitive match to succeed")
	}

	_, ok = fuzzy.Match("HeLLo", "hello")
	if !ok {
		t.Error("Expected case-insensitive match to succeed")
	}

	// Test case-sensitive
	fuzzy.SetCaseSensitive(true)
	_, ok = fuzzy.Match("HELLO", "hello")
	if ok {
		t.Error("Expected case-sensitive match to fail")
	}
}

func TestFuzzy_FuzzyMatching(t *testing.T) {
	fuzzy := NewFuzzy()

	// Test fuzzy match with gaps
	match, ok := fuzzy.Match("hlo", "hello")
	if !ok {
		t.Error("Expected fuzzy match to succeed")
	}
	if len(match.Positions) != 3 {
		t.Errorf("Expected 3 match positions, got %d", len(match.Positions))
	}

	// Test fuzzy match with more complex pattern
	match, ok = fuzzy.Match("tsk", "taskwarrior")
	if !ok {
		t.Error("Expected fuzzy match to succeed")
	}
	if match.Score <= 0 {
		t.Errorf("Expected positive score, got %f", match.Score)
	}
}

func TestFuzzy_Search(t *testing.T) {
	fuzzy := NewFuzzy()
	texts := []string{
		"hello world",
		"hello there",
		"hi everyone",
		"goodbye world",
		"testing hello",
	}

	matches := fuzzy.Search("hello", texts)

	if len(matches) < 3 {
		t.Errorf("Expected at least 3 matches, got %d", len(matches))
	}

	// Results should be sorted by score
	for i := 1; i < len(matches); i++ {
		if matches[i-1].Score < matches[i].Score {
			t.Error("Results should be sorted by score (highest first)")
		}
	}
}

func TestFuzzy_SearchMap(t *testing.T) {
	fuzzy := NewFuzzy()
	items := map[string]interface{}{
		"edit file":    "vim",
		"open browser": "firefox",
		"file manager": "nautilus",
		"text editor":  "gedit",
		"image editor": "gimp",
	}

	matches := fuzzy.SearchMap("edit", items)

	if len(matches) < 2 {
		t.Errorf("Expected at least 2 matches, got %d", len(matches))
	}

	// Check that matches contain the expected items
	found := false
	for _, match := range matches {
		if match.Key == "edit file" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'edit file' to be in matches")
	}
}

func TestFuzzy_Highlights(t *testing.T) {
	fuzzy := NewFuzzy().SetHighlightMatches(true)

	match, ok := fuzzy.Match("hlo", "hello")
	if !ok {
		t.Error("Expected match to succeed")
	}

	if len(match.Highlights) == 0 {
		t.Error("Expected highlight ranges to be generated")
	}

	// Test highlight string generation
	highlighted := fuzzy.HighlightString("hello", match.Highlights, "[", "]")
	if highlighted == "hello" {
		t.Error("Expected highlighted string to be different from original")
	}

	// Should contain highlight markers
	if !strings.Contains(highlighted, "[") || !strings.Contains(highlighted, "]") {
		t.Errorf("Expected highlighted string to contain markers, got: %s", highlighted)
	}
}

func TestFuzzy_MinScore(t *testing.T) {
	fuzzy := NewFuzzy().SetMinScore(0.9)

	// Low quality match should be filtered out
	_, ok := fuzzy.Match("xyz", "hello world test")
	if ok {
		t.Error("Expected low-quality match to be filtered out")
	}

	// High quality match should pass
	_, ok = fuzzy.Match("hello", "hello world")
	if !ok {
		t.Error("Expected high-quality match to succeed")
	}
}

func TestFuzzy_SmartMatch(t *testing.T) {
	fuzzy := NewFuzzy()

	// Test exact substring match (should have high score)
	match, ok := fuzzy.SmartMatch("world", "hello world")
	if !ok {
		t.Error("Expected smart match to succeed")
	}
	if match.Score < 0.8 {
		t.Errorf("Expected high score for exact substring match, got %f", match.Score)
	}

	// Test word boundary match
	match, ok = fuzzy.SmartMatch("hw", "hello world")
	if !ok {
		t.Error("Expected word boundary match to succeed")
	}
}

func TestFuzzy_EmptyQuery(t *testing.T) {
	fuzzy := NewFuzzy()

	match, ok := fuzzy.Match("", "hello")
	if !ok {
		t.Error("Expected empty query to match everything")
	}
	if match.Score != 1.0 {
		t.Errorf("Expected empty query to have score 1.0, got %f", match.Score)
	}
	if len(match.Positions) != 0 {
		t.Errorf("Expected empty query to have no positions, got %d", len(match.Positions))
	}
}

func TestFuzzy_EmptyText(t *testing.T) {
	fuzzy := NewFuzzy()

	_, ok := fuzzy.Match("hello", "")
	if ok {
		t.Error("Expected match against empty text to fail")
	}

	// Empty query against empty text should match
	match, ok := fuzzy.Match("", "")
	if !ok {
		t.Error("Expected empty query against empty text to match")
	}
	if match.Score != 1.0 {
		t.Errorf("Expected score 1.0, got %f", match.Score)
	}
}

// TestSearchable implements the Searchable interface for testing
type TestSearchable struct {
	searchText  string
	displayText string
}

func (t TestSearchable) SearchText() string  { return t.searchText }
func (t TestSearchable) DisplayText() string { return t.displayText }

func TestFuzzy_SearchItems(t *testing.T) {
	fuzzy := NewFuzzy()
	items := []Searchable{
		TestSearchable{"edit file", "Edit File (vim)"},
		TestSearchable{"open browser", "Open Browser (firefox)"},
		TestSearchable{"file manager", "File Manager (nautilus)"},
		TestSearchable{"text editor", "Text Editor (gedit)"},
	}

	matches := fuzzy.SearchItems("edit", items)

	if len(matches) < 2 {
		t.Errorf("Expected at least 2 matches, got %d", len(matches))
	}

	// Check that results are properly sorted
	for i := 1; i < len(matches); i++ {
		if matches[i-1].Match.Score < matches[i].Match.Score {
			t.Error("Results should be sorted by score (highest first)")
		}
	}
}

func TestFuzzy_WordBoundaryMatch(t *testing.T) {
	fuzzy := NewFuzzy()

	// Test word boundary matching
	match, ok := fuzzy.wordBoundaryMatch("edit file", "edit my file today")
	if !ok {
		t.Error("Expected word boundary match to succeed")
	}
	if match.Score <= 0.5 {
		t.Errorf("Expected high score for word boundary match, got %f", match.Score)
	}
}

func TestFuzzy_SplitIntoWords(t *testing.T) {
	fuzzy := NewFuzzy()

	words := fuzzy.splitIntoWords("hello world test")
	if len(words) != 3 {
		t.Errorf("Expected 3 words, got %d", len(words))
	}

	if words[0].Text != "hello" || words[0].Start != 0 || words[0].End != 5 {
		t.Errorf("Unexpected first word: %+v", words[0])
	}

	if words[1].Text != "world" || words[1].Start != 6 || words[1].End != 11 {
		t.Errorf("Unexpected second word: %+v", words[1])
	}
}

func TestFuzzy_NormalizeSpaces(t *testing.T) {
	fuzzy := NewFuzzy().SetNormalizeSpaces(true)

	normalized := fuzzy.normalize("  hello   world\t\ttest  ")
	expected := "hello world test"
	if normalized != expected {
		t.Errorf("Expected '%s', got '%s'", expected, normalized)
	}
}

func TestFuzzy_ScoreCalculation(t *testing.T) {
	fuzzy := NewFuzzy()

	// Test that exact matches get high scores
	match1, ok1 := fuzzy.Match("abc", "abc")
	if !ok1 {
		t.Error("Expected exact match to succeed")
	}
	if match1.Score < 0.8 {
		t.Errorf("Expected exact match to have high score, got %f", match1.Score)
	}

	// Test that matches produce reasonable scores
	match2, ok2 := fuzzy.Match("a", "abc")
	if !ok2 {
		t.Error("Expected single character match to succeed")
	}
	if match2.Score <= 0 || match2.Score > 1 {
		t.Errorf("Expected reasonable score between 0 and 1, got %f", match2.Score)
	}
}

// Performance benchmarks for fuzzy search optimization
func BenchmarkFuzzy_SingleMatch(b *testing.B) {
	fuzzy := NewFuzzy()
	query := "test"
	text := "this is a test string for benchmarking"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fuzzy.Match(query, text)
	}
}

func BenchmarkFuzzy_SearchSmall(b *testing.B) {
	fuzzy := NewFuzzy()
	query := "edit"
	texts := []string{
		"edit file", "open editor", "file manager", "text editor",
		"image editor", "video editor", "config editor", "edit settings",
		"quick edit", "batch edit", "edit mode", "editor preferences",
		"code editor", "markdown editor", "html editor", "css editor",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fuzzy.Search(query, texts)
	}
}

func BenchmarkFuzzy_SearchMedium(b *testing.B) {
	fuzzy := NewFuzzy()
	query := "task"

	// Generate 100 items
	texts := make([]string, 100)
	for i := 0; i < 100; i++ {
		texts[i] = generateTaskText(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fuzzy.Search(query, texts)
	}
}

func BenchmarkFuzzy_SearchLarge(b *testing.B) {
	fuzzy := NewFuzzy()
	query := "config"

	// Generate 1000 items
	texts := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		texts[i] = generateTaskText(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fuzzy.Search(query, texts)
	}
}

func BenchmarkFuzzy_SearchVeryLarge(b *testing.B) {
	fuzzy := NewFuzzy()
	query := "urgent"

	// Generate 5000 items (stress test)
	texts := make([]string, 5000)
	for i := 0; i < 5000; i++ {
		texts[i] = generateTaskText(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fuzzy.Search(query, texts)
	}
}

func BenchmarkFuzzy_SmartMatch(b *testing.B) {
	fuzzy := NewFuzzy()
	query := "edit file"
	text := "quickly edit the configuration file today"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fuzzy.SmartMatch(query, text)
	}
}

func BenchmarkFuzzy_SearchItems(b *testing.B) {
	fuzzy := NewFuzzy()
	query := "task"

	items := make([]Searchable, 1000)
	for i := 0; i < 1000; i++ {
		items[i] = TestSearchable{
			searchText:  generateTaskText(i),
			displayText: "Task #" + string(rune('0'+(i%10))),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fuzzy.SearchItems(query, items)
	}
}

// Helper function to generate varied test data
func generateTaskText(i int) string {
	templates := []string{
		"urgent task #%d needs completion",
		"configure project settings for item %d",
		"edit file %d in the project directory",
		"review and update documentation %d",
		"test functionality in module %d",
		"deploy changes to environment %d",
		"analyze performance metrics %d",
		"optimize query execution %d",
		"implement feature request %d",
		"fix bug report number %d",
		"schedule meeting for task %d",
		"coordinate team effort %d",
		"research new technology %d",
		"write unit tests for %d",
		"update dependencies %d",
		"monitor system health %d",
		"backup important data %d",
		"clean up temporary files %d",
		"organize project structure %d",
		"document api changes %d",
	}

	template := templates[i%len(templates)]
	return fmt.Sprintf(template, i)
}

func BenchmarkFuzzy_RealWorldScenario(b *testing.B) {
	fuzzy := NewFuzzy().SetMinScore(0.1)

	// Simulate real taskwarrior data
	realWorldTexts := []string{
		"Buy groceries at the store", "Call mom about dinner plans",
		"Fix the leaking faucet", "Schedule dentist appointment",
		"Review quarterly budget", "Update project timeline",
		"Prepare presentation slides", "Book flight tickets",
		"Clean garage this weekend", "Pay monthly bills online",
		"Exercise at the gym", "Read chapter 5 of book",
		"Water the plants", "Backup computer files",
		"Order new office supplies", "Plan vacation itinerary",
		"Research new programming language", "Write blog post",
		"Organize photo collection", "Learn guitar chords",
		"Study for certification exam", "Volunteer at local shelter",
		"Repair bicycle tire", "Download software updates",
		"Visit art museum exhibition", "Plant spring vegetables",
		"Configure home network", "Practice foreign language",
		"Meditate for 20 minutes", "Attend networking event",
	}

	queries := []string{"call", "fix", "update", "plan", "study", "config"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query := queries[i%len(queries)]
		fuzzy.Search(query, realWorldTexts)
	}
}

func BenchmarkFuzzy_SearchWithLimit(b *testing.B) {
	fuzzy := NewFuzzy()
	query := "task"

	// Generate 1000 items
	texts := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		texts[i] = generateTaskText(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fuzzy.SearchWithLimit(query, texts, 10) // Only return top 10 results
	}
}

func BenchmarkFuzzy_SearchAsync(b *testing.B) {
	fuzzy := NewFuzzy()
	query := "config"

	texts := make([]string, 500)
	for i := 0; i < 500; i++ {
		texts[i] = generateTaskText(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		results := make(chan Match, 100)
		done := make(chan struct{})

		go fuzzy.SearchAsync(query, texts, results, done)

		// Consume results
		for {
			select {
			case <-results:
				// Process result
			case <-done:
				goto next
			}
		}
	next:
	}
}

func BenchmarkFuzzy_NormalizationCached(b *testing.B) {
	fuzzy := NewFuzzy()
	texts := []string{
		"This is a test string",
		"Another Test String",
		"this is a test string", // Same as first when normalized
		"ANOTHER TEST STRING",   // Same as second when normalized
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		text := texts[i%len(texts)]
		fuzzy.normalize(text)
	}
}

func TestFuzzy_SearchWithLimit(t *testing.T) {
	fuzzy := NewFuzzy()
	texts := []string{
		"hello world", "hello there", "hi everyone",
		"goodbye world", "testing hello", "another hello",
		"more hello items", "final hello test",
	}

	matches := fuzzy.SearchWithLimit("hello", texts, 3)

	if len(matches) > 3 {
		t.Errorf("Expected at most 3 matches, got %d", len(matches))
	}

	// Results should still be sorted by score
	for i := 1; i < len(matches); i++ {
		if matches[i-1].Score < matches[i].Score {
			t.Error("Results should be sorted by score (highest first)")
		}
	}
}

func TestFuzzy_SearchAsync(t *testing.T) {
	fuzzy := NewFuzzy()
	texts := []string{
		"hello world", "hello there", "hi everyone",
		"goodbye world", "testing hello",
	}

	results := make(chan Match, 10)
	done := make(chan struct{})

	go fuzzy.SearchAsync("hello", texts, results, done)

	var matches []Match
	collecting := true
	for collecting {
		select {
		case match, ok := <-results:
			if !ok {
				collecting = false
			} else {
				matches = append(matches, match)
			}
		case <-done:
			// Drain any remaining results
			for {
				select {
				case match, ok := <-results:
					if !ok {
						collecting = false
						goto finished
					}
					matches = append(matches, match)
				default:
					collecting = false
					goto finished
				}
			}
		}
	}

finished:
	if len(matches) < 3 {
		t.Errorf("Expected at least 3 matches, got %d", len(matches))
	}
}
