package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestFormatter_Basic(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter(&buf)
	f.SetColorOutput(false) // Disable colors for predictable output
	f.SetLevel(LevelDebug)  // Enable debug messages

	f.Success("Operation completed")
	f.Error("Something went wrong")
	f.Warning("Be careful")
	f.Info("Here's some info")
	f.Debug("Debug message")

	output := buf.String()

	if !strings.Contains(output, "✓ Operation completed") {
		t.Error("Success message not found")
	}
	if !strings.Contains(output, "✗ Something went wrong") {
		t.Error("Error message not found")
	}
	if !strings.Contains(output, "⚠ Be careful") {
		t.Error("Warning message not found")
	}
	if !strings.Contains(output, "ℹ Here's some info") {
		t.Error("Info message not found")
	}
	if !strings.Contains(output, "🐛 Debug message") {
		t.Error("Debug message not found")
	}
}

func TestFormatter_OutputLevels(t *testing.T) {
	tests := []struct {
		level       OutputLevel
		expected    []string
		notExpected []string
	}{
		{
			level:       LevelQuiet,
			expected:    []string{"✗ Error"}, // Errors always shown
			notExpected: []string{"✓ Success", "⚠ Warning", "ℹ Info", "🐛 Debug"},
		},
		{
			level:       LevelNormal,
			expected:    []string{"✓ Success", "✗ Error", "⚠ Warning", "ℹ Info"},
			notExpected: []string{"🐛 Debug"},
		},
		{
			level:       LevelVerbose,
			expected:    []string{"✓ Success", "✗ Error", "⚠ Warning", "ℹ Info"},
			notExpected: []string{"🐛 Debug"},
		},
		{
			level:       LevelDebug,
			expected:    []string{"✓ Success", "✗ Error", "⚠ Warning", "ℹ Info", "🐛 Debug"},
			notExpected: []string{},
		},
	}

	for _, tt := range tests {
		var buf bytes.Buffer
		f := NewFormatter(&buf)
		f.SetColorOutput(false)
		f.SetLevel(tt.level)

		f.Success("Success")
		f.Error("Error")
		f.Warning("Warning")
		f.Info("Info")
		f.Debug("Debug")

		output := buf.String()

		for _, expected := range tt.expected {
			if !strings.Contains(output, expected) {
				t.Errorf("Level %d: Expected '%s' not found in output", tt.level, expected)
			}
		}

		for _, notExpected := range tt.notExpected {
			if strings.Contains(output, notExpected) {
				t.Errorf("Level %d: Unexpected '%s' found in output", tt.level, notExpected)
			}
		}
	}
}

func TestFormatter_Colors(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter(&buf)
	f.SetColorOutput(true)

	f.Success("Colored success")
	output := buf.String()

	// Should contain ANSI escape codes when colors are enabled
	if !strings.Contains(output, "\033[") {
		t.Error("Expected ANSI color codes when color output is enabled")
	}

	// Test disabling colors
	buf.Reset()
	f.SetColorOutput(false)
	f.Success("Non-colored success")
	output = buf.String()

	// Should not contain ANSI escape codes when colors are disabled
	if strings.Contains(output, "\033[") {
		t.Error("Unexpected ANSI color codes when color output is disabled")
	}
}

func TestFormatter_Header(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter(&buf)
	f.SetColorOutput(false)

	f.Header("Test Header")
	output := buf.String()

	if !strings.Contains(output, "Test Header") {
		t.Error("Header text not found")
	}
	if !strings.Contains(output, "═") {
		t.Error("Header border not found")
	}
}

func TestFormatter_Subheader(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter(&buf)
	f.SetColorOutput(false)

	f.Subheader("Test Subheader")
	output := buf.String()

	if !strings.Contains(output, "Test Subheader") {
		t.Error("Subheader text not found")
	}
	if !strings.Contains(output, "─") {
		t.Error("Subheader underline not found")
	}
}

func TestFormatter_List(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter(&buf)
	f.SetColorOutput(false)

	f.List("Item 1")
	f.List("Item 2")
	output := buf.String()

	if !strings.Contains(output, "• Item 1") {
		t.Error("List item 1 not found")
	}
	if !strings.Contains(output, "• Item 2") {
		t.Error("List item 2 not found")
	}
}

func TestTable_Basic(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter(&buf)
	f.SetColorOutput(false)

	table := f.Table()
	table.Headers("Name", "Age", "City")
	table.Row("Alice", "30", "New York")
	table.Row("Bob", "25", "San Francisco")
	table.Print()

	output := buf.String()

	if !strings.Contains(output, "Name") || !strings.Contains(output, "Age") || !strings.Contains(output, "City") {
		t.Error("Table headers not found")
	}
	if !strings.Contains(output, "Alice") || !strings.Contains(output, "30") || !strings.Contains(output, "New York") {
		t.Error("First table row not found")
	}
	if !strings.Contains(output, "Bob") || !strings.Contains(output, "25") || !strings.Contains(output, "San Francisco") {
		t.Error("Second table row not found")
	}
}

func TestFormatter_Progress(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter(&buf)
	f.SetColorOutput(false)

	f.Progress(50, 100, "Processing...")
	output := buf.String()

	if !strings.Contains(output, "50%") {
		t.Error("Progress percentage not found")
	}
	if !strings.Contains(output, "Processing...") {
		t.Error("Progress message not found")
	}
	if !strings.Contains(output, "█") && !strings.Contains(output, "░") {
		t.Error("Progress bar characters not found")
	}
}

func TestSpinner_Basic(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter(&buf)
	f.SetColorOutput(false)

	spinner := f.NewSpinner("Loading...")

	if spinner.message != "Loading..." {
		t.Error("Spinner message not set correctly")
	}
	if len(spinner.frames) == 0 {
		t.Error("Spinner frames not initialized")
	}
}

func TestFormatter_Colorize(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter(&buf)

	// Test with colors enabled
	f.SetColorOutput(true)
	colored := f.colorize("test", ColorRed, StyleBold)
	if !strings.Contains(colored, "\033[") {
		t.Error("Expected ANSI codes in colorized text")
	}

	// Test with colors disabled
	f.SetColorOutput(false)
	plain := f.colorize("test", ColorRed, StyleBold)
	if strings.Contains(plain, "\033[") {
		t.Error("Unexpected ANSI codes when colors disabled")
	}
	if plain != "test" {
		t.Error("Expected plain text when colors disabled")
	}
}

func TestThemes(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter(&buf)

	// Test default theme
	if f.theme.Primary != ColorBlue {
		t.Error("Default theme primary color incorrect")
	}

	// Test setting dark theme
	f.SetTheme(DarkTheme)
	if f.theme.Primary != ColorBrightBlue {
		t.Error("Dark theme primary color not set")
	}
}

func TestUtilityFunctions(t *testing.T) {
	// Test min function
	if min(5, 3) != 3 {
		t.Error("min function incorrect")
	}
	if min(2, 8) != 2 {
		t.Error("min function incorrect")
	}

	// Test getTerminalWidth
	width := getTerminalWidth()
	if width <= 0 {
		t.Error("Terminal width should be positive")
	}
}
