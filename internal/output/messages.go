package output

import "fmt"

// Success prints a success message
func (f *Formatter) Success(format string, args ...any) {
	if f.level == LevelQuiet {
		return
	}
	message := fmt.Sprintf(format, args...)
	styled := f.colorize("✓ "+message, f.theme.Success, StyleBold)
	fmt.Fprintln(f.writer, styled)
}

// Error prints an error message
func (f *Formatter) Error(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	styled := f.colorize("✗ "+message, f.theme.Error, StyleBold)
	fmt.Fprintln(f.writer, styled)
}

// Warning prints a warning message
func (f *Formatter) Warning(format string, args ...any) {
	if f.level == LevelQuiet {
		return
	}
	message := fmt.Sprintf(format, args...)
	styled := f.colorize("⚠ "+message, f.theme.Warning, StyleBold)
	fmt.Fprintln(f.writer, styled)
}

// Info prints an info message
func (f *Formatter) Info(format string, args ...any) {
	if f.level == LevelQuiet {
		return
	}
	message := fmt.Sprintf(format, args...)
	styled := f.colorize("ℹ "+message, f.theme.Info, StyleNormal)
	fmt.Fprintln(f.writer, styled)
}

// Debug prints a debug message
func (f *Formatter) Debug(format string, args ...any) {
	if f.level < LevelDebug {
		return
	}
	message := fmt.Sprintf(format, args...)
	styled := f.colorize("🐛 "+message, f.theme.Muted, StyleDim)
	fmt.Fprintln(f.writer, styled)
}
