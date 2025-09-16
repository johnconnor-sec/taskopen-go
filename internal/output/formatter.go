/*
* Split this file up
* Too big
 */
package output

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

// Color represents ANSI color codes
type Color int

const (
	ColorReset Color = iota
	ColorRed
	ColorGreen
	ColorYellow
	ColorBlue
	ColorMagenta
	ColorCyan
	ColorWhite
	ColorBrightRed
	ColorBrightGreen
	ColorBrightYellow
	ColorBrightBlue
	ColorBrightMagenta
	ColorBrightCyan
	ColorBrightWhite
)

// Style represents text formatting
type Style int

const (
	StyleNormal Style = iota
	StyleBold
	StyleDim
	StyleItalic
	StyleUnderline
)

// OutputLevel represents the verbosity level
type OutputLevel int

const (
	LevelQuiet OutputLevel = iota
	LevelNormal
	LevelVerbose
	LevelDebug
)

// Theme defines the color scheme for different elements
type Theme struct {
	Primary    Color
	Secondary  Color
	Success    Color
	Warning    Color
	Error      Color
	Info       Color
	Muted      Color
	Highlight  Color
	Border     Color
	Background Color
}

// DefaultTheme provides a sensible default color scheme
var DefaultTheme = Theme{
	Primary:    ColorBlue,
	Secondary:  ColorCyan,
	Success:    ColorGreen,
	Warning:    ColorYellow,
	Error:      ColorRed,
	Info:       ColorBlue,
	Muted:      ColorWhite,
	Highlight:  ColorBrightYellow,
	Border:     ColorMagenta,
	Background: ColorReset,
}

// DarkTheme for terminals with dark backgrounds
var DarkTheme = Theme{
	Primary:    ColorBrightBlue,
	Secondary:  ColorBrightCyan,
	Success:    ColorBrightGreen,
	Warning:    ColorBrightYellow,
	Error:      ColorBrightRed,
	Info:       ColorBrightBlue,
	Muted:      ColorWhite,
	Highlight:  ColorBrightYellow,
	Border:     ColorBrightMagenta,
	Background: ColorReset,
}

// Formatter handles styled output formatting
type Formatter struct {
	writer      io.Writer
	theme       Theme
	level       OutputLevel
	colorOutput bool
	width       int
	tabwriter   *tabwriter.Writer
}

// NewFormatter creates a new formatter with the given configuration
func NewFormatter(w io.Writer) *Formatter {
	f := &Formatter{
		writer:      w,
		theme:       DefaultTheme,
		level:       LevelNormal,
		colorOutput: isColorSupported(),
		width:       getTerminalWidth(),
	}
	f.tabwriter = tabwriter.NewWriter(f.writer, 0, 4, 2, ' ', 0)
	return f
}

// SetTheme changes the color theme
func (f *Formatter) SetTheme(theme Theme) {
	f.theme = theme
}

// SetLevel changes the output verbosity level
func (f *Formatter) SetLevel(level OutputLevel) {
	f.level = level
}

// SetColorOutput enables or disables color output
func (f *Formatter) SetColorOutput(enabled bool) {
	f.colorOutput = enabled
}

// colorize applies color and style to text if color output is enabled
func (f *Formatter) colorize(text string, color Color, style Style) string {
	if !f.colorOutput {
		return text
	}

	var codes []string

	// Add style codes
	switch style {
	case StyleBold:
		codes = append(codes, "1")
	case StyleDim:
		codes = append(codes, "2")
	case StyleItalic:
		codes = append(codes, "3")
	case StyleUnderline:
		codes = append(codes, "4")
	}

	// Add color codes
	switch color {
	case ColorRed:
		codes = append(codes, "31")
	case ColorGreen:
		codes = append(codes, "32")
	case ColorYellow:
		codes = append(codes, "33")
	case ColorBlue:
		codes = append(codes, "34")
	case ColorMagenta:
		codes = append(codes, "35")
	case ColorCyan:
		codes = append(codes, "36")
	case ColorWhite:
		codes = append(codes, "37")
	case ColorBrightRed:
		codes = append(codes, "91")
	case ColorBrightGreen:
		codes = append(codes, "92")
	case ColorBrightYellow:
		codes = append(codes, "93")
	case ColorBrightBlue:
		codes = append(codes, "94")
	case ColorBrightMagenta:
		codes = append(codes, "95")
	case ColorBrightCyan:
		codes = append(codes, "96")
	case ColorBrightWhite:
		codes = append(codes, "97")
	default:
		return text
	}

	if len(codes) == 0 {
		return text
	}

	return fmt.Sprintf("\033[%sm%s\033[0m", strings.Join(codes, ";"), text)
}

// GetCurrentWidth dynamically gets the current terminal width
func (f *Formatter) GetCurrentWidth() int {
	currentWidth := getTerminalWidth()
	f.width = currentWidth
	return currentWidth
}

// formatTaskField formats a single task field according to template rules
func (f *Formatter) formatTaskField(task map[string]any, field string, template OutputTemplate) string {
	rawValue := task[field]
	if rawValue == nil {
		return ""
	}

	value := fmt.Sprintf("%v", rawValue)
	if value == "<nil>" || value == "" {
		return ""
	}

	// Apply transformations
	if style, ok := template.Styles[field]; ok {
		value = f.applyFieldTransform(value, style.Transform)
		if f.colorOutput {
			value = f.colorize(value, style.Color, style.Style)
		}
	}

	return value
}
