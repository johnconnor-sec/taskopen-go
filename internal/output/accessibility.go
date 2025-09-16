package output

import "fmt"

// AccessibilityMode represents different accessibility configurations
type AccessibilityMode int

const (
	AccessibilityNormal AccessibilityMode = iota
	AccessibilityHighContrast
	AccessibilityScreenReader
	AccessibilityMinimal
)

// SetAccessibilityMode configures the formatter for accessibility needs
func (f *Formatter) SetAccessibilityMode(mode AccessibilityMode) {
	switch mode {
	case AccessibilityHighContrast:
		f.theme = HighContrastTheme
		f.colorOutput = true
	case AccessibilityScreenReader:
		f.colorOutput = false
		f.theme = DefaultTheme
	case AccessibilityMinimal:
		f.colorOutput = false
		f.theme = DefaultTheme
	default:
		// Normal mode - keep current settings
	}
}

// HighContrastTheme for better visibility
var HighContrastTheme = Theme{
	Primary:    ColorBrightWhite,
	Secondary:  ColorBrightCyan,
	Success:    ColorBrightGreen,
	Warning:    ColorBrightYellow,
	Error:      ColorBrightRed,
	Info:       ColorBrightBlue,
	Muted:      ColorWhite,
	Highlight:  ColorBrightYellow,
	Border:     ColorBrightWhite,
	Background: ColorReset,
}

// ScreenReaderText outputs text optimized for screen readers
func (f *Formatter) ScreenReaderText(semanticRole, content string) {
	if f.colorOutput {
		// Standard output with visual indicators
		switch semanticRole {
		case "success":
			fmt.Fprintln(f.writer, f.colorize(content, f.theme.Success, StyleNormal))
		case "error":
			fmt.Fprintln(f.writer, f.colorize(content, f.theme.Error, StyleNormal))
		case "warning":
			fmt.Fprintln(f.writer, f.colorize(content, f.theme.Warning, StyleNormal))
		case "info":
			fmt.Fprintln(f.writer, f.colorize(content, f.theme.Info, StyleNormal))
		default:
			fmt.Fprintln(f.writer, content)
		}
	} else {
		// Screen reader optimized output
		switch semanticRole {
		case "success":
			fmt.Fprintf(f.writer, "SUCCESS: %s\n", content)
		case "error":
			fmt.Fprintf(f.writer, "ERROR: %s\n", content)
		case "warning":
			fmt.Fprintf(f.writer, "WARNING: %s\n", content)
		case "info":
			fmt.Fprintf(f.writer, "INFO: %s\n", content)
		default:
			fmt.Fprintln(f.writer, content)
		}
	}
}
