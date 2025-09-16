package output

import (
	"bytes"
	"os"
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

func TestEnhancedTerminalWidth(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter(&buf)

	// Test dynamic width updating
	currentWidth := f.GetCurrentWidth()
	if currentWidth <= 0 {
		t.Error("Dynamic width should be positive")
	}
}

func TestEnhancedColorSupport(t *testing.T) {
	// Test NO_COLOR environment variable
	oldNoColor := os.Getenv("NO_COLOR")
	os.Setenv("NO_COLOR", "1")

	if isColorSupported() {
		t.Error("Color should be disabled when NO_COLOR is set")
	}

	// Restore environment
	if oldNoColor == "" {
		os.Unsetenv("NO_COLOR")
	} else {
		os.Setenv("NO_COLOR", oldNoColor)
	}

	// Test FORCE_COLOR
	oldForceColor := os.Getenv("FORCE_COLOR")
	os.Setenv("FORCE_COLOR", "1")

	if !isColorSupported() {
		t.Error("Color should be enabled when FORCE_COLOR is set")
	}

	// Restore environment
	if oldForceColor == "" {
		os.Unsetenv("FORCE_COLOR")
	} else {
		os.Setenv("FORCE_COLOR", oldForceColor)
	}
}

func TestOutputTemplates(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter(&buf)
	f.SetColorOutput(false)

	// Test task rendering with templates
	tasks := []map[string]interface{}{
		{
			"id":          1,
			"priority":    "H",
			"project":     "test",
			"description": "Test task",
			"due":         "2025-09-05",
		},
		{
			"id":          2,
			"priority":    "L",
			"project":     "demo",
			"description": "Demo task",
		},
	}

	// Test default template
	f.RenderTaskList(tasks)
	output := buf.String()

	if !strings.Contains(output, "Test task") {
		t.Error("Task description not found in default template output")
	}
	if !strings.Contains(output, "Found 2 tasks") {
		t.Error("Task count not found")
	}

	// Test compact template
	buf.Reset()
	f.RenderTaskListWithTemplate(tasks, CompactTaskTemplate)
	compactOutput := buf.String()

	if !strings.Contains(compactOutput, "Test task") {
		t.Error("Task description not found in compact template output")
	}
}

func TestMultiProgress(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter(&buf)
	f.SetColorOutput(false)

	mp := f.NewMultiProgress()
	bar1 := mp.AddProgress("task1", "Processing files", 100)
	_ = mp.AddProgress("task2", "Uploading data", 50) // bar2 for testing setup

	// Test initial state
	if bar1.current != 0 || bar1.total != 100 {
		t.Error("Progress bar initial state incorrect")
	}

	// Test updates
	mp.Update("task1", 25, "Processing files (25%)")
	mp.Update("task2", 10, "")

	if mp.bars["task1"].current != 25 {
		t.Error("Progress bar update failed")
	}

	// Test completion
	mp.Update("task1", 100, "Complete")
	if !mp.bars["task1"].finished {
		t.Error("Progress bar should be marked as finished")
	}
}

func TestStatusLine(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter(&buf)
	f.SetColorOutput(false)

	sl := f.NewStatusLine()

	// Test status updates
	sl.Update("Processing %d items", 10)
	if !sl.active {
		t.Error("Status line should be active after update")
	}

	// Test success completion
	sl.Success("Processing complete")
	if sl.active {
		t.Error("Status line should be inactive after success")
	}

	// Test error completion
	sl.Update("Processing items...")
	sl.Error("Processing failed")
	if sl.active {
		t.Error("Status line should be inactive after error")
	}
}

func TestAccessibilityModes(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter(&buf)

	// Test accessibility mode switching
	f.SetAccessibilityMode(AccessibilityScreenReader)
	if f.colorOutput {
		t.Error("Color output should be disabled in screen reader mode")
	}

	f.SetAccessibilityMode(AccessibilityHighContrast)
	if !f.colorOutput {
		t.Error("Color output should be enabled in high contrast mode")
	}
	if f.theme.Primary != ColorBrightWhite {
		t.Error("High contrast theme not applied")
	}

	// Test screen reader output
	f.SetAccessibilityMode(AccessibilityScreenReader)
	f.ScreenReaderText("error", "Test error message")
	output := buf.String()

	if !strings.Contains(output, "ERROR: Test error message") {
		t.Error("Screen reader format not applied")
	}
}

func TestFieldTransforms(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter(&buf)

	// Test priority transform
	result := f.transformPriorityDisplay("H")
	if result != "HIGH" {
		t.Error("Priority transform failed")
	}

	// Test case transforms
	result = f.applyFieldTransform("hello", "upper")
	if result != "HELLO" {
		t.Error("Upper transform failed")
	}

	result = f.applyFieldTransform("WORLD", "lower")
	if result != "world" {
		t.Error("Lower transform failed")
	}

	// Test truncate transform
	result = f.applyFieldTransform("very long text", "truncate:8")
	if len(result) > 8 {
		t.Error("Truncate transform failed")
	}
}

func TestDiagnosticsRendering(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter(&buf)
	f.SetColorOutput(false)

	diagnostics := []DiagnosticInfo{
		{
			Component: "Taskwarrior",
			Status:    "✓ Ready",
			Details:   map[string]interface{}{"version": "2.6.0"},
		},
		{
			Component:   "Config",
			Status:      "⚠ Warning",
			Details:     map[string]interface{}{"file": "/home/user/.taskrc"},
			Suggestions: []string{"Check configuration syntax"},
		},
	}

	f.RenderDiagnostics(diagnostics)
	output := buf.String()

	if !strings.Contains(output, "System Diagnostics") {
		t.Error("Diagnostics header not found")
	}
	if !strings.Contains(output, "Taskwarrior") {
		t.Error("Component name not found")
	}
	if !strings.Contains(output, "Check configuration syntax") {
		t.Error("Suggestion not found")
	}
}
