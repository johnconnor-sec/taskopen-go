package security

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

// EnvPreviewOptions controls how environment variables are previewed
type EnvPreviewOptions struct {
	VisibilityLevel VisibilityLevel
	ShowOnlySafe    bool
	ShowOnlyUnsafe  bool
	FilterPattern   string
	MaxItems        int
	ShowCounts      bool
}

// DefaultEnvPreviewOptions returns sensible defaults for environment preview
func DefaultEnvPreviewOptions() EnvPreviewOptions {
	return EnvPreviewOptions{
		VisibilityLevel: VisibilityMasked,
		ShowOnlySafe:    false,
		ShowOnlyUnsafe:  false,
		FilterPattern:   "",
		MaxItems:        50, // Reasonable default to prevent overwhelming output
		ShowCounts:      true,
	}
}

// EnvPreview generates a safe preview of environment variables
type EnvPreview struct {
	sanitizer *EnvSanitizer
	options   EnvPreviewOptions
}

// NewEnvPreview creates a new environment preview generator
func NewEnvPreview(options EnvPreviewOptions) *EnvPreview {
	return &EnvPreview{
		sanitizer: NewEnvSanitizer(),
		options:   options,
	}
}

// EnvVarInfo holds information about a single environment variable
type EnvVarInfo struct {
	Name        string
	Value       string
	IsSensitive bool
	IsFiltered  bool
}

// GeneratePreview creates a formatted preview of environment variables
func (ep *EnvPreview) GeneratePreview() string {
	ep.sanitizer.SetVisibilityLevel(ep.options.VisibilityLevel)

	// Get all environment variables
	envVars := ep.getFilteredEnvVars()

	if len(envVars) == 0 {
		return "No environment variables match the current filter criteria."
	}

	var output strings.Builder

	// Add header with summary
	if ep.options.ShowCounts {
		safeCount, sensitiveCount := ep.countVarTypes(envVars)
		output.WriteString(fmt.Sprintf("Environment Variables (showing %d", len(envVars)))
		if ep.options.MaxItems > 0 && len(envVars) >= ep.options.MaxItems {
			totalCount := len(os.Environ())
			output.WriteString(fmt.Sprintf(" of %d", totalCount))
		}
		output.WriteString(fmt.Sprintf(", %d safe, %d sensitive)\n", safeCount, sensitiveCount))
		output.WriteString(fmt.Sprintf("Visibility: %s\n\n", ep.getVisibilityDescription()))
	}

	// Sort variables by name for consistent output
	sort.Slice(envVars, func(i, j int) bool {
		return envVars[i].Name < envVars[j].Name
	})

	// Generate the variable list
	for i, envVar := range envVars {
		if ep.options.MaxItems > 0 && i >= ep.options.MaxItems {
			remaining := len(envVars) - ep.options.MaxItems
			output.WriteString(fmt.Sprintf("... and %d more variables\n", remaining))
			break
		}

		ep.formatEnvVar(&output, envVar)
	}

	// Add footer with helpful information
	if ep.options.VisibilityLevel == VisibilityMasked || ep.options.VisibilityLevel == VisibilityHidden {
		output.WriteString("\n")
		output.WriteString("💡 Tip: Use higher visibility levels to see more details (caution: may expose secrets)\n")
	}

	return output.String()
}

// getFilteredEnvVars returns environment variables filtered according to options
func (ep *EnvPreview) getFilteredEnvVars() []EnvVarInfo {
	allEnv := ep.sanitizer.GetAllSanitized()
	var result []EnvVarInfo

	for name, value := range allEnv {
		isSensitive := ep.sanitizer.IsSensitive(name)

		// Apply safe/unsafe filtering
		if ep.options.ShowOnlySafe && isSensitive {
			continue
		}
		if ep.options.ShowOnlyUnsafe && !isSensitive {
			continue
		}

		// Apply pattern filtering
		isFiltered := false
		if ep.options.FilterPattern != "" {
			matched := strings.Contains(strings.ToLower(name), strings.ToLower(ep.options.FilterPattern))
			if !matched {
				isFiltered = true
				continue
			}
		}

		result = append(result, EnvVarInfo{
			Name:        name,
			Value:       value,
			IsSensitive: isSensitive,
			IsFiltered:  isFiltered,
		})
	}

	return result
}

// countVarTypes counts safe and sensitive variables
func (ep *EnvPreview) countVarTypes(envVars []EnvVarInfo) (safe, sensitive int) {
	for _, envVar := range envVars {
		if envVar.IsSensitive {
			sensitive++
		} else {
			safe++
		}
	}
	return safe, sensitive
}

// formatEnvVar formats a single environment variable for display
func (ep *EnvPreview) formatEnvVar(output *strings.Builder, envVar EnvVarInfo) {
	// Add security indicator
	var indicator string
	if envVar.IsSensitive {
		indicator = "🔒"
	} else {
		indicator = "✅"
	}

	// Format the line
	maxNameWidth := 30 // Reasonable column width
	nameDisplay := envVar.Name
	if len(nameDisplay) > maxNameWidth {
		nameDisplay = nameDisplay[:maxNameWidth-3] + "..."
	}

	output.WriteString(fmt.Sprintf("%s %-*s = %s\n", indicator, maxNameWidth, nameDisplay, envVar.Value))
}

// getVisibilityDescription returns a human-readable description of the current visibility level
func (ep *EnvPreview) getVisibilityDescription() string {
	switch ep.options.VisibilityLevel {
	case VisibilityHidden:
		return "Hidden (safest - no sensitive data shown)"
	case VisibilityMasked:
		return "Masked (safe - sensitive data redacted)"
	case VisibilityLimited:
		return "Limited (caution - partial sensitive data shown)"
	case VisibilityFull:
		return "Full (danger - all sensitive data visible)"
	default:
		return "Unknown"
	}
}

// SetVisibilityLevel updates the visibility level
func (ep *EnvPreview) SetVisibilityLevel(level VisibilityLevel) {
	ep.options.VisibilityLevel = level
}

// SetFilter sets the pattern filter
func (ep *EnvPreview) SetFilter(pattern string) {
	ep.options.FilterPattern = pattern
}

// SetMaxItems sets the maximum number of items to display
func (ep *EnvPreview) SetMaxItems(max int) {
	ep.options.MaxItems = max
}

// ShowOnlySafe configures to show only safe variables
func (ep *EnvPreview) ShowOnlySafe(onlySafe bool) {
	ep.options.ShowOnlySafe = onlySafe
	if onlySafe {
		ep.options.ShowOnlyUnsafe = false
	}
}

// ShowOnlyUnsafe configures to show only sensitive variables
func (ep *EnvPreview) ShowOnlyUnsafe(onlyUnsafe bool) {
	ep.options.ShowOnlyUnsafe = onlyUnsafe
	if onlyUnsafe {
		ep.options.ShowOnlySafe = false
	}
}

// GetSanitizer returns the underlying sanitizer for advanced configuration
func (ep *EnvPreview) GetSanitizer() *EnvSanitizer {
	return ep.sanitizer
}

// GetStats returns statistics about environment variables
func (ep *EnvPreview) GetStats() map[string]interface{} {
	allEnv := ep.sanitizer.GetAllSanitized()
	safeCount := 0
	sensitiveCount := 0

	for name := range allEnv {
		if ep.sanitizer.IsSensitive(name) {
			sensitiveCount++
		} else {
			safeCount++
		}
	}

	return map[string]interface{}{
		"total":      len(allEnv),
		"safe":       safeCount,
		"sensitive":  sensitiveCount,
		"visibility": ep.getVisibilityDescription(),
	}
}
