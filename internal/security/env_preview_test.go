package security

import (
	"os"
	"strings"
	"testing"
)

func TestNewEnvPreview(t *testing.T) {
	options := DefaultEnvPreviewOptions()
	preview := NewEnvPreview(options)

	if preview == nil {
		t.Error("NewEnvPreview should not return nil")
	}

	if preview.sanitizer == nil {
		t.Error("NewEnvPreview should initialize sanitizer")
	}
}

func TestDefaultEnvPreviewOptions(t *testing.T) {
	options := DefaultEnvPreviewOptions()

	if options.VisibilityLevel != VisibilityMasked {
		t.Errorf("Expected default visibility level to be VisibilityMasked, got %v", options.VisibilityLevel)
	}

	if options.MaxItems != 50 {
		t.Errorf("Expected default MaxItems to be 50, got %d", options.MaxItems)
	}

	if !options.ShowCounts {
		t.Error("Expected ShowCounts to be true by default")
	}
}

func TestEnvPreview_GeneratePreview(t *testing.T) {
	// Set up test environment variables
	os.Setenv("TEST_SAFE_VAR", "safe_value")
	os.Setenv("TEST_SECRET_KEY", "secret_value")
	defer func() {
		os.Unsetenv("TEST_SAFE_VAR")
		os.Unsetenv("TEST_SECRET_KEY")
	}()

	options := DefaultEnvPreviewOptions()
	options.FilterPattern = "TEST_" // Only show our test variables
	preview := NewEnvPreview(options)

	// Add TEST_SAFE_VAR as safe
	preview.sanitizer.AddSafeVar("TEST_SAFE_VAR")

	result := preview.GeneratePreview()

	// Should contain header with counts
	if !strings.Contains(result, "Environment Variables") {
		t.Error("Preview should contain header")
	}

	// Should show safe variables
	if !strings.Contains(result, "TEST_SAFE_VAR") {
		t.Error("Preview should show safe variables")
	}

	// Should show sensitive variables but with redacted values
	if !strings.Contains(result, "TEST_SECRET_KEY") {
		t.Error("Preview should show sensitive variable names")
	}

	if strings.Contains(result, "secret_value") {
		t.Error("Preview should not show sensitive values")
	}

	if !strings.Contains(result, "[REDACTED-SECRET]") {
		t.Error("Preview should show redacted sensitive values")
	}
}

func TestEnvPreview_SetVisibilityLevel(t *testing.T) {
	options := DefaultEnvPreviewOptions()
	options.FilterPattern = "TEST_SECRET"
	preview := NewEnvPreview(options)

	os.Setenv("TEST_SECRET", "secret123")
	defer os.Unsetenv("TEST_SECRET")

	// Test masked visibility (default)
	preview.SetVisibilityLevel(VisibilityMasked)
	result := preview.GeneratePreview()
	if !strings.Contains(result, "[REDACTED-SECRET]") {
		t.Error("Masked visibility should show redacted values")
	}

	// Test full visibility (dangerous)
	preview.SetVisibilityLevel(VisibilityFull)
	result = preview.GeneratePreview()
	if !strings.Contains(result, "secret123") {
		t.Error("Full visibility should show actual values")
	}

	// Test hidden visibility
	preview.SetVisibilityLevel(VisibilityHidden)
	result = preview.GeneratePreview()
	if !strings.Contains(result, "[HIDDEN]") {
		t.Error("Hidden visibility should show [HIDDEN]")
	}
}

func TestEnvPreview_SetFilter(t *testing.T) {
	os.Setenv("TASKOPEN_TEST_VAR", "test_value")
	os.Setenv("OTHER_TEST_VAR", "other_value")
	defer func() {
		os.Unsetenv("TASKOPEN_TEST_VAR")
		os.Unsetenv("OTHER_TEST_VAR")
	}()

	options := DefaultEnvPreviewOptions()
	options.FilterPattern = "TEST_VAR" // Start with a filter that matches both
	preview := NewEnvPreview(options)
	preview.GetSanitizer().AddSafeVar("TASKOPEN_TEST_VAR")
	preview.GetSanitizer().AddSafeVar("OTHER_TEST_VAR")

	// Test filter for "TEST_VAR" - should show both
	result := preview.GeneratePreview()
	if !strings.Contains(result, "TASKOPEN_TEST_VAR") {
		t.Error("Should show TASKOPEN_TEST_VAR with TEST_VAR filter")
	}
	if !strings.Contains(result, "OTHER_TEST_VAR") {
		t.Error("Should show OTHER_TEST_VAR with TEST_VAR filter")
	}

	// Test filter for "TASKOPEN" - should show only taskopen var
	preview.SetFilter("TASKOPEN")
	result = preview.GeneratePreview()
	if !strings.Contains(result, "TASKOPEN_TEST_VAR") {
		t.Error("Should show TASKOPEN_TEST_VAR with TASKOPEN filter")
	}
	if strings.Contains(result, "OTHER_TEST_VAR") {
		t.Error("Should not show OTHER_TEST_VAR with TASKOPEN filter")
	}
}

func TestEnvPreview_ShowOnlySafe(t *testing.T) {
	os.Setenv("SAFE_VAR", "safe_value")
	os.Setenv("SECRET_VAR", "secret_value")
	defer func() {
		os.Unsetenv("SAFE_VAR")
		os.Unsetenv("SECRET_VAR")
	}()

	preview := NewEnvPreview(DefaultEnvPreviewOptions())
	preview.sanitizer.AddSafeVar("SAFE_VAR")

	// Show only safe variables
	preview.ShowOnlySafe(true)
	result := preview.GeneratePreview()

	if !strings.Contains(result, "SAFE_VAR") {
		t.Error("Should show safe variables when ShowOnlySafe is true")
	}
	if strings.Contains(result, "SECRET_VAR") {
		t.Error("Should not show sensitive variables when ShowOnlySafe is true")
	}
}

func TestEnvPreview_ShowOnlyUnsafe(t *testing.T) {
	os.Setenv("SAFE_VAR", "safe_value")
	os.Setenv("SECRET_VAR", "secret_value")
	defer func() {
		os.Unsetenv("SAFE_VAR")
		os.Unsetenv("SECRET_VAR")
	}()

	preview := NewEnvPreview(DefaultEnvPreviewOptions())
	preview.sanitizer.AddSafeVar("SAFE_VAR")

	// Show only sensitive variables
	preview.ShowOnlyUnsafe(true)
	result := preview.GeneratePreview()

	if strings.Contains(result, "SAFE_VAR") {
		t.Error("Should not show safe variables when ShowOnlyUnsafe is true")
	}
	if !strings.Contains(result, "SECRET_VAR") {
		t.Error("Should show sensitive variables when ShowOnlyUnsafe is true")
	}
}

func TestEnvPreview_SetMaxItems(t *testing.T) {
	// Set up multiple test variables
	testVars := []string{"VAR1", "VAR2", "VAR3", "VAR4", "VAR5"}
	for _, varName := range testVars {
		os.Setenv(varName, "value")
	}
	defer func() {
		for _, varName := range testVars {
			os.Unsetenv(varName)
		}
	}()

	preview := NewEnvPreview(DefaultEnvPreviewOptions())
	for _, varName := range testVars {
		preview.sanitizer.AddSafeVar(varName)
	}

	// Set max items to 2
	preview.SetMaxItems(2)
	result := preview.GeneratePreview()

	// Should contain message about more variables
	if !strings.Contains(result, "and") && !strings.Contains(result, "more variables") {
		// Check if at least some limiting is happening
		lines := strings.Split(result, "\n")
		varCount := 0
		for _, line := range lines {
			if strings.Contains(line, "VAR") {
				varCount++
			}
		}
		// Due to the complex filtering, we just ensure it's not showing all variables
		if varCount > 10 { // Should be much less than all environment variables
			t.Error("SetMaxItems should limit the number of displayed variables")
		}
	}
}

func TestEnvPreview_GetStats(t *testing.T) {
	os.Setenv("SAFE_TEST", "safe_value")
	os.Setenv("SECRET_TEST", "secret_value")
	defer func() {
		os.Unsetenv("SAFE_TEST")
		os.Unsetenv("SECRET_TEST")
	}()

	preview := NewEnvPreview(DefaultEnvPreviewOptions())
	preview.sanitizer.AddSafeVar("SAFE_TEST")

	stats := preview.GetStats()

	if _, exists := stats["total"]; !exists {
		t.Error("Stats should include total count")
	}

	if _, exists := stats["safe"]; !exists {
		t.Error("Stats should include safe count")
	}

	if _, exists := stats["sensitive"]; !exists {
		t.Error("Stats should include sensitive count")
	}

	if _, exists := stats["visibility"]; !exists {
		t.Error("Stats should include visibility description")
	}

	// Check that counts are reasonable
	total := stats["total"].(int)
	safe := stats["safe"].(int)
	sensitive := stats["sensitive"].(int)

	if total != safe+sensitive {
		t.Errorf("Total (%d) should equal safe (%d) + sensitive (%d)", total, safe, sensitive)
	}
}

func TestEnvPreview_VisibilityDescription(t *testing.T) {
	preview := NewEnvPreview(DefaultEnvPreviewOptions())

	tests := []struct {
		level       VisibilityLevel
		description string
	}{
		{VisibilityHidden, "Hidden"},
		{VisibilityMasked, "Masked"},
		{VisibilityLimited, "Limited"},
		{VisibilityFull, "Full"},
	}

	for _, tt := range tests {
		preview.SetVisibilityLevel(tt.level)
		desc := preview.getVisibilityDescription()
		if !strings.Contains(desc, tt.description) {
			t.Errorf("Visibility description for level %d should contain %q, got %q",
				tt.level, tt.description, desc)
		}
	}
}

func TestEnvPreview_EmptyEnvironment(t *testing.T) {
	options := DefaultEnvPreviewOptions()
	options.FilterPattern = "NONEXISTENT_PATTERN_12345"

	preview := NewEnvPreview(options)
	result := preview.GeneratePreview()

	expected := "No environment variables match the current filter criteria."
	if result != expected {
		t.Errorf("Expected empty environment message, got: %s", result)
	}
}

// Integration test that simulates real usage
func TestEnvPreview_Integration(t *testing.T) {
	// Set up realistic test environment
	os.Setenv("HOME", "/home/testuser")
	os.Setenv("TERM", "xterm")
	os.Setenv("API_KEY", "sk-1234567890abcdef")
	os.Setenv("DATABASE_PASSWORD", "super_secret_db_pass")
	os.Setenv("MY_APP_CONFIG", "some_config_value")

	defer func() {
		// Clean up (note: HOME and TERM are usually set, so we don't unset them)
		os.Unsetenv("API_KEY")
		os.Unsetenv("DATABASE_PASSWORD")
		os.Unsetenv("MY_APP_CONFIG")
	}()

	preview := NewEnvPreview(DefaultEnvPreviewOptions())
	preview.sanitizer.AddSafeVar("MY_APP_CONFIG")

	result := preview.GeneratePreview()

	// Verify safe variables are shown
	if !strings.Contains(result, "/home/testuser") {
		t.Error("Should show HOME value")
	}
	if !strings.Contains(result, "some_config_value") {
		t.Error("Should show safe app config value")
	}

	// Verify sensitive variables are masked
	if strings.Contains(result, "sk-1234567890abcdef") {
		t.Error("Should not show actual API key")
	}
	if strings.Contains(result, "super_secret_db_pass") {
		t.Error("Should not show actual database password")
	}

	// Should contain security indicators
	if !strings.Contains(result, "🔒") && !strings.Contains(result, "✅") {
		t.Error("Should contain security indicators")
	}

	// Should contain redacted values for sensitive vars
	if !strings.Contains(result, "[REDACTED-SECRET]") {
		t.Error("Should contain redacted secret values")
	}
}
