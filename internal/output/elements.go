package output

import (
	"fmt"
	"strings"
)

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
func (f *Formatter) List(format string, args ...any) {
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

// NewSpinner creates a new spinner
func (f *Formatter) NewSpinner(message string) *Spinner {
	return &Spinner{
		formatter: f,
		message:   message,
		frames:    []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		done:      make(chan bool),
	}
}
