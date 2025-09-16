package output

import "fmt"

// StatusLine provides a dynamic status line for operations
type StatusLine struct {
	formatter *Formatter
	active    bool
}

// NewStatusLine creates a new status line
func (f *Formatter) NewStatusLine() *StatusLine {
	return &StatusLine{
		formatter: f,
		active:    false,
	}
}

// Update updates the status line
func (sl *StatusLine) Update(status string, args ...any) {
	if sl.formatter.level == LevelQuiet {
		return
	}

	message := fmt.Sprintf(status, args...)

	// Clear line and write new status
	fmt.Fprintf(sl.formatter.writer, "\r\033[K%s", message)
	sl.active = true
}

// Success shows a success message and ends the status line
func (sl *StatusLine) Success(message string, args ...any) {
	if sl.active {
		fmt.Fprint(sl.formatter.writer, "\r\033[K")
	}
	sl.formatter.Success(message, args...)
	sl.active = false
}

// Error shows an error message and ends the status line
func (sl *StatusLine) Error(message string, args ...any) {
	if sl.active {
		fmt.Fprint(sl.formatter.writer, "\r\033[K")
	}
	sl.formatter.Error(message, args...)
	sl.active = false
}

// Clear clears the status line
func (sl *StatusLine) Clear() {
	if sl.active {
		fmt.Fprint(sl.formatter.writer, "\r\033[K")
		sl.active = false
	}
}
