package output

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"
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

// Success prints a success message
func (f *Formatter) Success(format string, args ...interface{}) {
	if f.level == LevelQuiet {
		return
	}
	message := fmt.Sprintf(format, args...)
	styled := f.colorize("✓ "+message, f.theme.Success, StyleBold)
	fmt.Fprintln(f.writer, styled)
}

// Error prints an error message
func (f *Formatter) Error(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	styled := f.colorize("✗ "+message, f.theme.Error, StyleBold)
	fmt.Fprintln(f.writer, styled)
}

// Warning prints a warning message
func (f *Formatter) Warning(format string, args ...interface{}) {
	if f.level == LevelQuiet {
		return
	}
	message := fmt.Sprintf(format, args...)
	styled := f.colorize("⚠ "+message, f.theme.Warning, StyleBold)
	fmt.Fprintln(f.writer, styled)
}

// Info prints an info message
func (f *Formatter) Info(format string, args ...interface{}) {
	if f.level == LevelQuiet {
		return
	}
	message := fmt.Sprintf(format, args...)
	styled := f.colorize("ℹ "+message, f.theme.Info, StyleNormal)
	fmt.Fprintln(f.writer, styled)
}

// Debug prints a debug message
func (f *Formatter) Debug(format string, args ...interface{}) {
	if f.level < LevelDebug {
		return
	}
	message := fmt.Sprintf(format, args...)
	styled := f.colorize("🐛 "+message, f.theme.Muted, StyleDim)
	fmt.Fprintln(f.writer, styled)
}

// Header prints a prominent header
func (f *Formatter) Header(text string) {
	if f.level == LevelQuiet {
		return
	}

	border := strings.Repeat("═", min(len(text)+4, f.width))

	fmt.Fprintln(f.writer, f.colorize(border, f.theme.Border, StyleBold))
	fmt.Fprintln(f.writer, f.colorize(fmt.Sprintf("  %s  ", text), f.theme.Primary, StyleBold))
	fmt.Fprintln(f.writer, f.colorize(border, f.theme.Border, StyleBold))
}

// Subheader prints a section header
func (f *Formatter) Subheader(text string) {
	if f.level == LevelQuiet {
		return
	}

	styled := f.colorize(text, f.theme.Secondary, StyleBold)
	fmt.Fprintln(f.writer, styled)

	underline := strings.Repeat("─", min(len(text), f.width))
	fmt.Fprintln(f.writer, f.colorize(underline, f.theme.Border, StyleNormal))
}

// List prints a bulleted list item
func (f *Formatter) List(format string, args ...interface{}) {
	if f.level == LevelQuiet {
		return
	}
	message := fmt.Sprintf(format, args...)
	styled := f.colorize("• "+message, f.theme.Primary, StyleNormal)
	fmt.Fprintln(f.writer, styled)
}

// Table starts a new table for columnized output
func (f *Formatter) Table() *Table {
	return &Table{
		formatter: f,
		headers:   make([]string, 0),
		rows:      make([][]string, 0),
	}
}

// Table represents a columnized table
type Table struct {
	formatter *Formatter
	headers   []string
	rows      [][]string
}

// Headers sets the table headers
func (t *Table) Headers(headers ...string) *Table {
	t.headers = headers
	return t
}

// Row adds a row to the table
func (t *Table) Row(cells ...string) *Table {
	t.rows = append(t.rows, cells)
	return t
}

// Print renders the table
func (t *Table) Print() {
	if t.formatter.level == LevelQuiet {
		return
	}

	if len(t.headers) > 0 {
		// Print headers
		headerRow := make([]string, len(t.headers))
		for i, header := range t.headers {
			headerRow[i] = t.formatter.colorize(header, t.formatter.theme.Primary, StyleBold)
		}
		fmt.Fprintln(t.formatter.tabwriter, strings.Join(headerRow, "\t"))

		// Print separator
		separators := make([]string, len(t.headers))
		for i, header := range t.headers {
			separators[i] = strings.Repeat("─", len(header))
		}
		fmt.Fprintln(t.formatter.tabwriter, strings.Join(separators, "\t"))
	}

	// Print rows
	for _, row := range t.rows {
		fmt.Fprintln(t.formatter.tabwriter, strings.Join(row, "\t"))
	}

	t.formatter.tabwriter.Flush()
}

// Progress creates a progress indicator
func (f *Formatter) Progress(current, total int, message string) {
	if f.level == LevelQuiet {
		return
	}

	width := 40
	if f.width > 80 {
		width = 50
	}

	percentage := float64(current) / float64(total)
	filled := int(percentage * float64(width))

	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	coloredBar := f.colorize(bar, f.theme.Primary, StyleNormal)

	progress := fmt.Sprintf("\r[%s] %3.0f%% %s", coloredBar, percentage*100, message)
	fmt.Fprint(f.writer, progress)

	if current == total {
		fmt.Fprintln(f.writer)
	}
}

// Spinner shows a spinning progress indicator
type Spinner struct {
	formatter *Formatter
	message   string
	frames    []string
	current   int
	done      chan bool
}

// NewSpinner creates a new spinner
func (f *Formatter) NewSpinner(message string) *Spinner {
	return &Spinner{
		formatter: f,
		message:   message,
		frames:    []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		done:      make(chan bool),
	}
}

// Start begins the spinner animation
func (s *Spinner) Start() {
	if s.formatter.level == LevelQuiet {
		return
	}

	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-s.done:
				return
			case <-ticker.C:
				frame := s.formatter.colorize(s.frames[s.current], s.formatter.theme.Primary, StyleNormal)
				fmt.Fprintf(s.formatter.writer, "\r%s %s", frame, s.message)
				s.current = (s.current + 1) % len(s.frames)
			}
		}
	}()
}

// Stop ends the spinner animation
func (s *Spinner) Stop() {
	s.done <- true
	fmt.Fprint(s.formatter.writer, "\r")
}

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

// Utility functions

func isColorSupported() bool {
	// Check for explicit accessibility settings first
	if accessibility := os.Getenv("TASKOPEN_ACCESSIBILITY"); accessibility != "" {
		switch accessibility {
		case "screen-reader", "minimal":
			return false
		case "high-contrast":
			return true
		}
	}

	// Respect NO_COLOR standard
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	// Force color if requested
	if os.Getenv("FORCE_COLOR") != "" {
		return true
	}

	term := os.Getenv("TERM")
	if term == "" || term == "dumb" {
		return false
	}

	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	// Check for common accessibility tools
	if os.Getenv("NVDA") != "" || os.Getenv("JAWS") != "" || os.Getenv("ORCA") != "" {
		return false
	}

	return true
}

func getTerminalWidth() int {
	if width := os.Getenv("COLUMNS"); width != "" {
		if w, err := strconv.Atoi(width); err == nil && w > 0 {
			return w
		}
	}
	return 80
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// DiagnosticInfo represents system diagnostic information
type DiagnosticInfo struct {
	Component   string
	Status      string
	Details     map[string]interface{}
	Suggestions []string
}

// RenderDiagnostics outputs comprehensive system diagnostics
func (f *Formatter) RenderDiagnostics(diagnostics []DiagnosticInfo) {
	f.Header("System Diagnostics")

	// System overview table
	table := f.Table().Headers("Component", "Status", "Details")

	for _, diag := range diagnostics {
		status := diag.Status
		if f.colorOutput {
			switch diag.Status {
			case "✓ Ready", "✓ Functional":
				status = f.colorize(diag.Status, f.theme.Success, StyleBold)
			case "⚠ Warning":
				status = f.colorize(diag.Status, f.theme.Warning, StyleBold)
			case "✗ Failed", "✗ Error":
				status = f.colorize(diag.Status, f.theme.Error, StyleBold)
			default:
				status = f.colorize(diag.Status, f.theme.Info, StyleNormal)
			}
		}

		// Format details
		details := ""
		if len(diag.Details) > 0 {
			var parts []string
			for k, v := range diag.Details {
				parts = append(parts, fmt.Sprintf("%s: %v", k, v))
			}
			details = strings.Join(parts, ", ")
		}

		table.Row(diag.Component, status, details)
	}

	table.Print()

	// Show suggestions for failed components
	for _, diag := range diagnostics {
		if strings.Contains(diag.Status, "Failed") || strings.Contains(diag.Status, "Warning") {
			if len(diag.Suggestions) > 0 {
				fmt.Fprintln(f.writer)
				f.ScreenReaderText("warning", fmt.Sprintf("%s Issues", diag.Component))
				for _, suggestion := range diag.Suggestions {
					f.List("%s", suggestion)
				}
			}
		}
	}
}

// RenderTaskList outputs a formatted task list with enhanced readability
func (f *Formatter) RenderTaskList(tasks []map[string]interface{}) {
	if len(tasks) == 0 {
		f.ScreenReaderText("info", "No tasks match the current filter")
		return
	}

	f.Subheader(fmt.Sprintf("Found %d tasks", len(tasks)))

	table := f.Table().Headers("ID", "Priority", "Project", "Description")

	for _, task := range tasks {
		id := fmt.Sprintf("%v", task["id"])
		priority := fmt.Sprintf("%v", task["priority"])
		project := fmt.Sprintf("%v", task["project"])
		description := fmt.Sprintf("%v", task["description"])

		// Colorize priority
		if f.colorOutput && priority != "" {
			switch priority {
			case "H":
				priority = f.colorize("HIGH", f.theme.Error, StyleBold)
			case "M":
				priority = f.colorize("MED", f.theme.Warning, StyleBold)
			case "L":
				priority = f.colorize("LOW", f.theme.Info, StyleNormal)
			}
		}

		table.Row(id, priority, project, description)
	}

	table.Print()
}
