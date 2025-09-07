// Package security provides security-related utilities for taskopen.
package security

import (
	"os"
	"regexp"
	"strings"
)

// SensitivePattern defines patterns for sensitive environment variables
type SensitivePattern struct {
	Pattern     *regexp.Regexp
	Replacement string
	Description string
}

// VisibilityLevel controls how much information to show
type VisibilityLevel int

const (
	VisibilityHidden  VisibilityLevel = iota // Show nothing
	VisibilityMasked                         // Show masked values
	VisibilityLimited                        // Show first/last chars only
	VisibilityFull                           // Show full values (unsafe for sensitive data)
)

// EnvSanitizer provides secure environment variable handling
type EnvSanitizer struct {
	sensitivePatterns []SensitivePattern
	safeVars          map[string]bool
	visibilityLevel   VisibilityLevel
}

// NewEnvSanitizer creates a new environment variable sanitizer
func NewEnvSanitizer() *EnvSanitizer {
	sanitizer := &EnvSanitizer{
		sensitivePatterns: []SensitivePattern{
			// More specific patterns first
			{
				Pattern:     regexp.MustCompile(`(?i).*(aws|gcp|azure|cloud).*(key|token|secret|password|pass|pwd|auth|access).*`),
				Replacement: "[REDACTED-CLOUD]",
				Description: "Cloud provider credentials",
			},
			{
				Pattern:     regexp.MustCompile(`(?i).*(db|database|sql).*(pass|pwd|password|secret).*`),
				Replacement: "[REDACTED-DATABASE]",
				Description: "Database credentials",
			},
			// General pattern last (catch-all)
			{
				Pattern:     regexp.MustCompile(`(?i).*(key|token|secret|password|pass|pwd|auth|api).*`),
				Replacement: "[REDACTED-SECRET]",
				Description: "API keys, tokens, passwords, and secrets",
			},
		},
		safeVars: map[string]bool{
			// System variables that are generally safe to display
			"HOME":    true,
			"USER":    true,
			"TERM":    true,
			"SHELL":   true,
			"LANG":    true,
			"LC_ALL":  true,
			"TZ":      true,
			"COLUMNS": true,
			"LINES":   true,
			"DISPLAY": true,
			"EDITOR":  true,
			"VISUAL":  true,
			// CI/CD environment indicators (generally safe)
			"CI":          true,
			"CI_NO_COLOR": true,
			// Color and accessibility settings
			"NO_COLOR":       true,
			"FORCE_COLOR":    true,
			"CLICOLOR":       true,
			"CLICOLOR_FORCE": true,
			"NVDA":           true,
			"JAWS":           true,
			"ORCA":           true,
			// Taskopen-specific variables
			"TASKOPEN_ACCESSIBILITY": true,
			"TASKOPEN_OPEN_CMD":      true,
			"TASKOPENRC":             true,
			"XDG_CONFIG_HOME":        true,
		},
		visibilityLevel: VisibilityMasked, // Safe default
	}

	return sanitizer
}

// SetVisibilityLevel sets the visibility level for environment variables
func (es *EnvSanitizer) SetVisibilityLevel(level VisibilityLevel) {
	es.visibilityLevel = level
}

// GetVisibilityLevel returns the current visibility level
func (es *EnvSanitizer) GetVisibilityLevel() VisibilityLevel {
	return es.visibilityLevel
}

// IsSensitive checks if an environment variable name matches sensitive patterns
func (es *EnvSanitizer) IsSensitive(name string) bool {
	// Check if explicitly marked as safe
	if es.safeVars[name] {
		return false
	}

	// Check against sensitive patterns
	for _, pattern := range es.sensitivePatterns {
		if pattern.Pattern.MatchString(name) {
			return true
		}
	}

	return false
}

// SanitizeValue sanitizes an environment variable value based on its name and visibility level
func (es *EnvSanitizer) SanitizeValue(name, value string) string {
	if value == "" {
		return ""
	}

	// Always show safe variables at any visibility level
	if es.safeVars[name] {
		return value
	}

	// Handle sensitive variables based on visibility level
	if es.IsSensitive(name) {
		switch es.visibilityLevel {
		case VisibilityHidden:
			return "[HIDDEN]"
		case VisibilityMasked:
			return es.getMaskedReplacement(name)
		case VisibilityLimited:
			return es.getLimitedValue(value)
		case VisibilityFull:
			return value // User explicitly requested full visibility
		default:
			return "[MASKED]" // Safe default
		}
	}

	// For unknown variables (not in safe list, not matching sensitive patterns), be cautious
	switch es.visibilityLevel {
	case VisibilityHidden:
		return "[HIDDEN]"
	case VisibilityMasked:
		return "[MASKED]"
	case VisibilityLimited:
		return es.getLimitedValue(value)
	case VisibilityFull:
		return value
	default:
		return "[MASKED]" // Safe default
	}
}

// getMaskedReplacement gets the appropriate masked replacement for a sensitive variable
func (es *EnvSanitizer) getMaskedReplacement(name string) string {
	for _, pattern := range es.sensitivePatterns {
		if pattern.Pattern.MatchString(name) {
			return pattern.Replacement
		}
	}
	return "[REDACTED]"
}

// getLimitedValue shows only the first and last few characters of a value
func (es *EnvSanitizer) getLimitedValue(value string) string {
	if len(value) <= 8 {
		return strings.Repeat("*", len(value))
	}

	if len(value) <= 16 {
		return value[:2] + strings.Repeat("*", len(value)-4) + value[len(value)-2:]
	}

	return value[:3] + strings.Repeat("*", len(value)-6) + value[len(value)-3:]
}

// SafeGetenv safely retrieves an environment variable with appropriate sanitization
func (es *EnvSanitizer) SafeGetenv(name string) string {
	value := os.Getenv(name)
	return es.SanitizeValue(name, value)
}

// GetAllSanitized returns all environment variables with appropriate sanitization
func (es *EnvSanitizer) GetAllSanitized() map[string]string {
	result := make(map[string]string)

	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			name := parts[0]
			value := parts[1]
			result[name] = es.SanitizeValue(name, value)
		}
	}

	return result
}

// AddSensitivePattern adds a custom sensitive pattern
func (es *EnvSanitizer) AddSensitivePattern(pattern *regexp.Regexp, replacement, description string) {
	es.sensitivePatterns = append(es.sensitivePatterns, SensitivePattern{
		Pattern:     pattern,
		Replacement: replacement,
		Description: description,
	})
}

// AddSafeVar marks a variable as safe to display
func (es *EnvSanitizer) AddSafeVar(name string) {
	es.safeVars[name] = true
}

// RemoveSafeVar removes a variable from the safe list
func (es *EnvSanitizer) RemoveSafeVar(name string) {
	delete(es.safeVars, name)
}
