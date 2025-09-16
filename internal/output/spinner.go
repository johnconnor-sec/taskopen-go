package output

import (
	"fmt"
	"time"
)

// Spinner shows a spinning progress indicator
type Spinner struct {
	formatter *Formatter
	message   string
	frames    []string
	current   int
	done      chan bool
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

// Update changes the spinner message
func (s *Spinner) Update(message string) {
	s.message = message
}
