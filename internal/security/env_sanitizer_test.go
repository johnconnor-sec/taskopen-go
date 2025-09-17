package security

import (
	"os"
	"regexp"
	"testing"
)

func TestEnvSanitizer_IsSensitive(t *testing.T) {
	sanitizer := NewEnvSanitizer()

	tests := []struct {
		name     string
		envVar   string
		expected bool
	}{
		// Safe variables
		{"HOME is safe", "HOME", false},
		{"TERM is safe", "TERM", false},
		{"TASKOPEN_ACCESSIBILITY is safe", "TASKOPEN_ACCESSIBILITY", false},

		// Sensitive variables
		{"API_KEY is sensitive", "API_KEY", true},
		{"SECRET_TOKEN is sensitive", "SECRET_TOKEN", true},
		{"DB_PASSWORD is sensitive", "DB_PASSWORD", true},
		{"AWS_SECRET_ACCESS_KEY is sensitive", "AWS_SECRET_ACCESS_KEY", true},
		{"GITHUB_TOKEN is sensitive", "GITHUB_TOKEN", true},
		{"DATABASE_PASSWORD is sensitive", "DATABASE_PASSWORD", true},

		// Edge cases
		{"KEYRING is sensitive", "KEYRING", true},
		{"TOKEN_CACHE is sensitive", "TOKEN_CACHE", true},
		{"MY_SECRET is sensitive", "MY_SECRET", true},
		{"APP_AUTH_KEY is sensitive", "APP_AUTH_KEY", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.IsSensitive(tt.envVar)
			if result != tt.expected {
				t.Errorf("IsSensitive(%s) = %v, want %v", tt.envVar, result, tt.expected)
			}
		})
	}
}

func TestEnvSanitizer_SanitizeValue(t *testing.T) {
	sanitizer := NewEnvSanitizer()

	tests := []struct {
		name          string
		envVar        string
		value         string
		visibility    VisibilityLevel
		expectedValue string
	}{
		// Safe variables at any visibility
		{"HOME always visible", "HOME", "/home/user", VisibilityMasked, "/home/user"},
		{"TERM always visible", "TERM", "xterm", VisibilityHidden, "xterm"},

		// Sensitive variables - Hidden
		{"API_KEY hidden", "API_KEY", "secret123", VisibilityHidden, "[HIDDEN]"},
		{"DB_PASSWORD hidden", "DATABASE_PASSWORD", "mypass", VisibilityHidden, "[HIDDEN]"},

		// Sensitive variables - Masked
		{"API_KEY masked", "API_KEY", "secret123", VisibilityMasked, "[REDACTED-SECRET]"},
		{"AWS_KEY masked", "AWS_ACCESS_KEY", "AKIAIOSFODNN7EXAMPLE", VisibilityMasked, "[REDACTED-CLOUD]"},
		{"DB_PASSWORD masked", "DB_PASSWORD", "mypass", VisibilityMasked, "[REDACTED-DATABASE]"},

		// Sensitive variables - Limited
		{"Short value limited", "API_KEY", "short", VisibilityLimited, "*****"},
		{"Medium value limited", "API_KEY", "mediumvalue", VisibilityLimited, "me*******ue"},
		{"Long value limited", "API_KEY", "verylongpasswordvalue", VisibilityLimited, "ver***************lue"},

		// Sensitive variables - Full (dangerous but requested)
		{"API_KEY full visibility", "API_KEY", "secret123", VisibilityFull, "secret123"},

		// Unknown variables (cautious approach)
		{"Unknown var hidden", "UNKNOWN_VAR", "value", VisibilityHidden, "[HIDDEN]"},
		{"Unknown var masked", "UNKNOWN_VAR", "value", VisibilityMasked, "[MASKED]"},
		{"Unknown var limited", "UNKNOWN_VAR", "somevalue", VisibilityLimited, "so*****ue"},
		{"Unknown var full", "UNKNOWN_VAR", "value", VisibilityFull, "value"},

		// Empty values
		{"Empty value", "API_KEY", "", VisibilityMasked, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sanitizer.SetVisibilityLevel(tt.visibility)
			result := sanitizer.SanitizeValue(tt.envVar, tt.value)
			if result != tt.expectedValue {
				t.Errorf("SanitizeValue(%s, %s) with visibility %d = %s, want %s",
					tt.envVar, tt.value, tt.visibility, result, tt.expectedValue)
			}
		})
	}
}

func TestEnvSanitizer_GetLimitedValue(t *testing.T) {
	sanitizer := NewEnvSanitizer()

	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{"Very short", "abc", "***"},
		{"Short", "abcdef", "******"},
		{"Medium", "abcdefghij", "ab******ij"},
		{"Long", "abcdefghijklmnopqr", "abc************pqr"},
		{"Very long", "abcdefghijklmnopqrstuvwxyz", "abc********************xyz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.getLimitedValue(tt.value)
			if result != tt.expected {
				t.Errorf("getLimitedValue(%s) = %s, want %s", tt.value, result, tt.expected)
			}
		})
	}
}

func TestEnvSanitizer_SafeGetenv(t *testing.T) {
	sanitizer := NewEnvSanitizer()
	sanitizer.SetVisibilityLevel(VisibilityMasked)

	// Set up test environment variables
	originalValue := os.Getenv("TEST_SAFE_VAR")
	originalSensitive := os.Getenv("TEST_SENSITIVE_KEY")

	defer func() {
		if originalValue != "" {
			os.Setenv("TEST_SAFE_VAR", originalValue)
		} else {
			os.Unsetenv("TEST_SAFE_VAR")
		}
		if originalSensitive != "" {
			os.Setenv("TEST_SENSITIVE_KEY", originalSensitive)
		} else {
			os.Unsetenv("TEST_SENSITIVE_KEY")
		}
	}()

	// Add test variables as safe
	sanitizer.AddSafeVar("TEST_SAFE_VAR")

	os.Setenv("TEST_SAFE_VAR", "safe_value")
	os.Setenv("TEST_SENSITIVE_KEY", "secret_value")

	// Test safe variable
	safeResult := sanitizer.SafeGetenv("TEST_SAFE_VAR")
	if safeResult != "safe_value" {
		t.Errorf("SafeGetenv(TEST_SAFE_VAR) = %s, want %s", safeResult, "safe_value")
	}

	// Test sensitive variable
	sensitiveResult := sanitizer.SafeGetenv("TEST_SENSITIVE_KEY")
	if sensitiveResult != "[REDACTED-SECRET]" {
		t.Errorf("SafeGetenv(TEST_SENSITIVE_KEY) = %s, want %s", sensitiveResult, "[REDACTED-SECRET]")
	}
}

func TestEnvSanitizer_AddSensitivePattern(t *testing.T) {
	sanitizer := NewEnvSanitizer()

	// Add a custom pattern
	customPattern := regexp.MustCompile(`(?i).*custom.*`)
	sanitizer.AddSensitivePattern(customPattern, "[CUSTOM-REDACTED]", "Custom sensitive pattern")

	// Test that the custom pattern works
	if !sanitizer.IsSensitive("MY_CUSTOM_VAR") {
		t.Error("Custom pattern should make MY_CUSTOM_VAR sensitive")
	}

	sanitizer.SetVisibilityLevel(VisibilityMasked)
	result := sanitizer.SanitizeValue("MY_CUSTOM_VAR", "value")
	if result != "[CUSTOM-REDACTED]" {
		t.Errorf("Custom pattern should return [CUSTOM-REDACTED], got %s", result)
	}
}

func TestEnvSanitizer_VisibilityLevels(t *testing.T) {
	sanitizer := NewEnvSanitizer()

	levels := []VisibilityLevel{VisibilityHidden, VisibilityMasked, VisibilityLimited, VisibilityFull}

	for _, level := range levels {
		sanitizer.SetVisibilityLevel(level)
		if sanitizer.GetVisibilityLevel() != level {
			t.Errorf("SetVisibilityLevel/GetVisibilityLevel mismatch: set %d, got %d",
				level, sanitizer.GetVisibilityLevel())
		}
	}
}

func TestEnvSanitizer_SafeVarManagement(t *testing.T) {
	sanitizer := NewEnvSanitizer()

	// Test adding a safe variable
	sanitizer.AddSafeVar("NEW_SAFE_VAR")
	if sanitizer.IsSensitive("NEW_SAFE_VAR") {
		t.Error("NEW_SAFE_VAR should not be sensitive after AddSafeVar")
	}

	// Test removing a safe variable
	sanitizer.RemoveSafeVar("HOME") // HOME is safe by default
	sanitizer.SetVisibilityLevel(VisibilityMasked)
	result := sanitizer.SanitizeValue("HOME", "/home/user")
	if result == "/home/user" {
		t.Error("HOME should be masked after RemoveSafeVar")
	}
}

func TestEnvSanitizer_GetAllSanitized(t *testing.T) {
	sanitizer := NewEnvSanitizer()
	sanitizer.SetVisibilityLevel(VisibilityMasked)

	// Set up test environment
	os.Setenv("TEST_SAFE_123", "safe_value")
	os.Setenv("TEST_SECRET_123", "secret_value")
	sanitizer.AddSafeVar("TEST_SAFE_123")

	defer func() {
		os.Unsetenv("TEST_SAFE_123")
		os.Unsetenv("TEST_SECRET_123")
	}()

	all := sanitizer.GetAllSanitized()

	// Check that our test variables are handled correctly
	if safeVal, exists := all["TEST_SAFE_123"]; !exists || safeVal != "safe_value" {
		t.Errorf("TEST_SAFE_123 should be 'safe_value', got %s (exists: %v)", safeVal, exists)
	}

	if secretVal, exists := all["TEST_SECRET_123"]; !exists || secretVal == "secret_value" {
		t.Errorf("TEST_SECRET_123 should be sanitized, got %s (exists: %v)", secretVal, exists)
	}
}

// Benchmark tests
func BenchmarkIsSensitive(b *testing.B) {
	sanitizer := NewEnvSanitizer()
	testVars := []string{"HOME", "API_KEY", "DATABASE_PASSWORD", "TERM", "AWS_SECRET_KEY"}

	for b.Loop() {
		for _, varName := range testVars {
			sanitizer.IsSensitive(varName)
		}
	}
}

func BenchmarkSanitizeValue(b *testing.B) {
	sanitizer := NewEnvSanitizer()
	sanitizer.SetVisibilityLevel(VisibilityMasked)

	for b.Loop() {
		sanitizer.SanitizeValue("API_KEY", "some_secret_value_here")
		sanitizer.SanitizeValue("HOME", "/home/user")
	}
}
