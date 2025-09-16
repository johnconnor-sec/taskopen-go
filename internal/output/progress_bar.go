package output

import (
	"fmt"
	"strings"
)

// MultiProgress manages multiple progress indicators
type MultiProgress struct {
	formatter *Formatter
	bars      map[string]*ProgressBar
	order     []string
}

// ProgressBar represents a single progress bar
type ProgressBar struct {
	id       string
	current  int
	total    int
	message  string
	color    Color
	finished bool
}

// NewMultiProgress creates a new multi-progress manager
func (f *Formatter) NewMultiProgress() *MultiProgress {
	return &MultiProgress{
		formatter: f,
		bars:      make(map[string]*ProgressBar),
		order:     make([]string, 0),
	}
}

// AddProgress adds a new progress bar
func (mp *MultiProgress) AddProgress(id, message string, total int) *ProgressBar {
	bar := &ProgressBar{
		id:      id,
		current: 0,
		total:   total,
		message: message,
		color:   mp.formatter.theme.Primary,
	}
	mp.bars[id] = bar
	mp.order = append(mp.order, id)
	return bar
}

// Update updates a progress bar
func (mp *MultiProgress) Update(id string, current int, message string) {
	if bar, exists := mp.bars[id]; exists {
		bar.current = current
		if message != "" {
			bar.message = message
		}
		if current >= bar.total {
			bar.finished = true
		}
	}
}

// Render renders all progress bars
func (mp *MultiProgress) Render() {
	if mp.formatter.level == LevelQuiet {
		return
	}

	// Clear previous output
	fmt.Fprint(mp.formatter.writer, "\033[2K\r")

	for i, id := range mp.order {
		if bar, exists := mp.bars[id]; exists {
			mp.renderSingleBar(bar)
			if i < len(mp.order)-1 {
				fmt.Fprintln(mp.formatter.writer)
			}
		}
	}
}

// renderSingleBar renders a single progress bar
func (mp *MultiProgress) renderSingleBar(bar *ProgressBar) {
	width := 30
	if mp.formatter.width > 80 {
		width = 40
	}

	percentage := float64(bar.current) / float64(bar.total)
	filled := int(percentage * float64(width))

	progressChars := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	coloredBar := mp.formatter.colorize(progressChars, bar.color, StyleNormal)

	status := "⠿"
	if bar.finished {
		status = "✓"
		status = mp.formatter.colorize(status, mp.formatter.theme.Success, StyleBold)
	}

	progress := fmt.Sprintf("%s [%s] %3.0f%% %s (%d/%d)",
		status, coloredBar, percentage*100, bar.message, bar.current, bar.total)

	fmt.Fprint(mp.formatter.writer, progress)
}

// Finish marks all progress bars as complete
func (mp *MultiProgress) Finish() {
	for _, bar := range mp.bars {
		bar.current = bar.total
		bar.finished = true
	}
	mp.Render()
	fmt.Fprintln(mp.formatter.writer)
}
