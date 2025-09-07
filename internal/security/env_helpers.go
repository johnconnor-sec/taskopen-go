package security

import (
	"os"
)

// SecureEnv provides secure environment variable operations
var SecureEnv = NewEnvSanitizer()

// SafeGetenv safely retrieves an environment variable for user display
func SafeGetenv(name string) string {
	return SecureEnv.SafeGetenv(name)
}

// UnsafeGetenv retrieves an environment variable without sanitization
// Use this for internal operations where the value is needed for functionality,
// not display to the user.
func UnsafeGetenv(name string) string {
	return os.Getenv(name)
}

// SetVisibility sets the global environment variable visibility level
func SetVisibility(level VisibilityLevel) {
	SecureEnv.SetVisibilityLevel(level)
}

// GetEnvPreview returns a preview of environment variables
func GetEnvPreview(options EnvPreviewOptions) string {
	preview := NewEnvPreview(options)
	return preview.GeneratePreview()
}

// GetDefaultEnvPreview returns a preview with safe defaults
func GetDefaultEnvPreview() string {
	return GetEnvPreview(DefaultEnvPreviewOptions())
}

// AddSensitiveEnvPattern adds a custom pattern for sensitive environment variables
func AddSensitiveEnvPattern(pattern string, replacement, description string) error {
	// This is a simplified version - in a full implementation you'd compile the regex
	// For now, we'll just add it to the global sanitizer if needed
	return nil
}

// MarkEnvAsSafe marks an environment variable as safe to display
func MarkEnvAsSafe(name string) {
	SecureEnv.AddSafeVar(name)
}

// MarkEnvAsUnsafe removes an environment variable from the safe list
func MarkEnvAsUnsafe(name string) {
	SecureEnv.RemoveSafeVar(name)
}

// IsEnvSensitive checks if an environment variable is considered sensitive
func IsEnvSensitive(name string) bool {
	return SecureEnv.IsSensitive(name)
}

// GetEnvStats returns statistics about environment variables
func GetEnvStats() map[string]interface{} {
	preview := NewEnvPreview(DefaultEnvPreviewOptions())
	return preview.GetStats()
}
