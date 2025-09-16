// Package ui provides secure TUI components with tcell integration
package ui

import (
	"fmt"
	"strings"
	"sync"

	"github.com/gdamore/tcell/v2"
	"github.com/johnconnor-sec/taskopen-go/internal/output"
	"github.com/johnconnor-sec/taskopen-go/internal/search"
	"github.com/johnconnor-sec/taskopen-go/internal/security"
)

// TUIMode represents different TUI interaction modes
type TUIMode int

const (
	ModeInteractive TUIMode = iota
	ModePreview
	ModeHelp
	ModeSearch
)

// SearchableMenuItem wraps MenuItem to implement search.Searchable
type SearchableMenuItem struct {
	MenuItem
}

func (s SearchableMenuItem) SearchText() string {
	return s.Text + " " + s.Description
}

func (s SearchableMenuItem) DisplayText() string {
	return s.Text
}

// SecureTUI provides a secure, full-featured TUI with tcell
type SecureTUI struct {
	screen       tcell.Screen
	items        []MenuItem
	filtered     []MenuItem
	selected     int
	searchQuery  string
	mode         TUIMode
	config       MenuConfig
	searchEngine *search.Fuzzy
	formatter    *output.Formatter
	sanitizer    *security.EnvSanitizer

	// Preview state
	showPreview  bool
	previewWidth int

	// State management
	running bool
	mutex   sync.RWMutex

	// Dimensions
	width  int
	height int

	// Colors (with accessibility support)
	selectedStyle tcell.Style
	normalStyle   tcell.Style
	searchStyle   tcell.Style
	previewStyle  tcell.Style
	borderStyle   tcell.Style

	// Security settings
	hideEnvVars     bool
	visibilityLevel security.VisibilityLevel
}

// SecureTUIConfig configures the secure TUI
type SecureTUIConfig struct {
	ShowPreview       bool
	PreviewWidth      int
	HideEnvVars       bool
	VisibilityLevel   security.VisibilityLevel
	AccessibilityMode output.AccessibilityMode
}

// NewSecureTUI creates a new secure TUI with tcell
func NewSecureTUI(items []MenuItem, config MenuConfig, tuiConfig SecureTUIConfig) (*SecureTUI, error) {
	screen, err := tcell.NewScreen()
	if err != nil {
		return nil, fmt.Errorf("failed to create screen: %w", err)
	}

	if err := screen.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize screen: %w", err)
	}

	// Get screen dimensions
	width, height := screen.Size()

	tui := &SecureTUI{
		screen:          screen,
		items:           items,
		filtered:        items,
		selected:        0,
		mode:            ModeInteractive,
		config:          config,
		searchEngine:    search.NewFuzzy(),
		formatter:       output.NewFormatter(nil), // We'll handle output ourselves
		sanitizer:       security.NewEnvSanitizer(),
		showPreview:     tuiConfig.ShowPreview,
		previewWidth:    tuiConfig.PreviewWidth,
		hideEnvVars:     tuiConfig.HideEnvVars,
		visibilityLevel: tuiConfig.VisibilityLevel,
		running:         false,
		width:           width,
		height:          height,
	}

	// Configure sanitizer
	tui.sanitizer.SetVisibilityLevel(tuiConfig.VisibilityLevel)

	// Set up colors based on accessibility mode
	tui.setupStyles(tuiConfig.AccessibilityMode)

	// searchEngine will be used dynamically - no pre-setting needed

	return tui, nil
}

// setupStyles configures TUI colors based on accessibility needs
func (t *SecureTUI) setupStyles(accessMode output.AccessibilityMode) {
	switch accessMode {
	case output.AccessibilityHighContrast:
		t.selectedStyle = tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack)
		t.normalStyle = tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorWhite)
		t.searchStyle = tcell.StyleDefault.Background(tcell.ColorYellow).Foreground(tcell.ColorBlack)
		t.previewStyle = tcell.StyleDefault.Background(tcell.ColorBlue).Foreground(tcell.ColorWhite)
		t.borderStyle = tcell.StyleDefault.Foreground(tcell.ColorWhite)
	case output.AccessibilityMinimal:
		// Minimal colors - just basic contrast
		t.selectedStyle = tcell.StyleDefault.Background(tcell.ColorGray).Foreground(tcell.ColorBlack)
		t.normalStyle = tcell.StyleDefault
		t.searchStyle = tcell.StyleDefault.Background(tcell.ColorDarkGray)
		t.previewStyle = tcell.StyleDefault
		t.borderStyle = tcell.StyleDefault
	default:
		// Standard colors
		t.selectedStyle = tcell.StyleDefault.Background(tcell.ColorBlue).Foreground(tcell.ColorWhite)
		t.normalStyle = tcell.StyleDefault
		t.searchStyle = tcell.StyleDefault.Background(tcell.ColorGreen).Foreground(tcell.ColorBlack)
		t.previewStyle = tcell.StyleDefault.Foreground(tcell.ColorGray)
		t.borderStyle = tcell.StyleDefault.Foreground(tcell.ColorDarkGray)
	}
}

// Show displays the TUI and handles user interaction
func (t *SecureTUI) Show() (*MenuItem, error) {
	t.running = true
	defer t.Close()

	// Initial draw
	t.draw()

	// Main event loop
	for t.running {
		ev := t.screen.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventResize:
			t.handleResize()
		case *tcell.EventKey:
			if !t.handleKeyEvent(ev) {
				// Selection made or cancelled
				if t.selected >= 0 && t.selected < len(t.filtered) {
					selected := t.filtered[t.selected]
					return &selected, nil
				}
				return nil, nil
			}
		}
		t.draw()
	}

	return nil, nil
}

// handleKeyEvent processes keyboard input securely
func (t *SecureTUI) handleKeyEvent(ev *tcell.EventKey) bool {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	switch t.mode {
	case ModeSearch:
		return t.handleSearchMode(ev)
	case ModePreview:
		return t.handlePreviewMode(ev)
	case ModeHelp:
		return t.handleHelpMode(ev)
	default:
		return t.handleInteractiveMode(ev)
	}
}

// handleInteractiveMode processes navigation and selection
func (t *SecureTUI) handleInteractiveMode(ev *tcell.EventKey) bool {
	switch ev.Key() {
	case tcell.KeyEscape, tcell.KeyCtrlC:
		t.running = false
		return false
	case tcell.KeyEnter:
		// Selection made
		t.running = false
		return false
	case tcell.KeyUp, tcell.KeyCtrlP:
		if t.selected > 0 {
			t.selected--
		} else {
			t.selected = len(t.filtered) - 1
		}
	case tcell.KeyDown, tcell.KeyCtrlN:
		if t.selected < len(t.filtered)-1 {
			t.selected++
		} else {
			t.selected = 0
		}
	case tcell.KeyHome, tcell.KeyCtrlA:
		t.selected = 0
	case tcell.KeyEnd, tcell.KeyCtrlE:
		if len(t.filtered) > 0 {
			t.selected = len(t.filtered) - 1
		}
	case tcell.KeyTab:
		if t.showPreview {
			t.mode = ModePreview
		}
	case tcell.KeyF1:
		t.mode = ModeHelp
	case tcell.KeyCtrlF, tcell.KeyRune:
		if ev.Key() == tcell.KeyCtrlF || (ev.Rune() >= 32 && ev.Rune() < 127) {
			t.mode = ModeSearch
			if ev.Rune() >= 32 && ev.Rune() < 127 {
				t.searchQuery = string(ev.Rune())
				t.performSearch()
			}
		}
	}

	return true
}

// handleSearchMode processes search input with real-time filtering
func (t *SecureTUI) handleSearchMode(ev *tcell.EventKey) bool {
	switch ev.Key() {
	case tcell.KeyEscape:
		t.mode = ModeInteractive
		t.searchQuery = ""
		t.filtered = t.items
		t.selected = 0
	case tcell.KeyEnter:
		t.mode = ModeInteractive
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if len(t.searchQuery) > 0 {
			t.searchQuery = t.searchQuery[:len(t.searchQuery)-1]
			t.performSearch()
		}
	case tcell.KeyCtrlU:
		t.searchQuery = ""
		t.performSearch()
	case tcell.KeyUp:
		if t.selected > 0 {
			t.selected--
		}
	case tcell.KeyDown:
		if t.selected < len(t.filtered)-1 {
			t.selected++
		}
	case tcell.KeyRune:
		if ev.Rune() >= 32 && ev.Rune() < 127 {
			t.searchQuery += string(ev.Rune())
			t.performSearch()
		}
	}

	return true
}

// handlePreviewMode processes preview navigation
func (t *SecureTUI) handlePreviewMode(ev *tcell.EventKey) bool {
	switch ev.Key() {
	case tcell.KeyEscape, tcell.KeyTab:
		t.mode = ModeInteractive
	case tcell.KeyUp:
		if t.selected > 0 {
			t.selected--
		}
	case tcell.KeyDown:
		if t.selected < len(t.filtered)-1 {
			t.selected++
		}
	case tcell.KeyEnter:
		t.running = false
		return false
	}

	return true
}

// handleHelpMode processes help screen
func (t *SecureTUI) handleHelpMode(ev *tcell.EventKey) bool {
	switch ev.Key() {
	case tcell.KeyEscape, tcell.KeyF1, tcell.KeyEnter:
		t.mode = ModeInteractive
	}

	return true
}

// performSearch executes fuzzy search with real-time results
func (t *SecureTUI) performSearch() {
	if t.searchQuery == "" {
		t.filtered = t.items
	} else {
		// Convert items to searchable
		searchableItems := make([]search.Searchable, len(t.items))
		for i, item := range t.items {
			searchableItems[i] = SearchableMenuItem{item}
		}

		// Perform search
		results := t.searchEngine.SearchItems(t.searchQuery, searchableItems)
		t.filtered = make([]MenuItem, len(results))
		for i, result := range results {
			// Extract original MenuItem from SearchableMenuItem
			if searchable, ok := result.Item.(SearchableMenuItem); ok {
				t.filtered[i] = searchable.MenuItem
			}
		}
	}

	// Reset selection to first item
	t.selected = 0
}

// handleResize handles terminal resize events
func (t *SecureTUI) handleResize() {
	t.screen.Sync()
	t.width, t.height = t.screen.Size()
}

// draw renders the complete TUI interface
func (t *SecureTUI) draw() {
	t.screen.Clear()

	switch t.mode {
	case ModeHelp:
		t.drawHelp()
	case ModePreview:
		t.drawWithPreview()
	default:
		t.drawMain()
	}

	t.screen.Show()
}

// drawMain renders the main interface
func (t *SecureTUI) drawMain() {
	// Draw title bar
	title := "Taskopen - Secure Interactive Menu"
	t.drawText(0, 0, t.width, title, t.borderStyle.Bold(true))

	// Draw search bar
	searchText := fmt.Sprintf("Search: %s", t.searchQuery)
	if t.mode == ModeSearch {
		t.drawText(0, 1, t.width, searchText, t.searchStyle)
	} else {
		t.drawText(0, 1, t.width, searchText, t.normalStyle)
	}

	// Draw separator
	t.drawHorizontalLine(2, t.borderStyle)

	// Draw items
	startY := 3
	visibleItems := t.height - startY - 3 // Leave space for status bar

	for i := 0; i < visibleItems && i < len(t.filtered); i++ {
		itemIndex := i
		if itemIndex >= len(t.filtered) {
			break
		}

		item := t.filtered[itemIndex]
		style := t.normalStyle
		if itemIndex == t.selected {
			style = t.selectedStyle
		}

		// Format item text securely
		text := t.formatItemSecurely(item, itemIndex)
		t.drawText(0, startY+i, t.width, text, style)
	}

	// Draw status bar
	t.drawStatusBar()
}

// formatItemSecurely formats menu items with environment variable sanitization
func (t *SecureTUI) formatItemSecurely(item MenuItem, index int) string {
	keyText := fmt.Sprintf("%d", index+1)
	text := fmt.Sprintf("%-2s %s", keyText, item.Text)

	if item.Description != "" {
		// Sanitize description that might contain environment variables
		safeDesc := t.sanitizeText(item.Description)
		text += fmt.Sprintf(" - %s", safeDesc)
	}

	return text
}

// sanitizeText sanitizes text that might contain sensitive information
func (t *SecureTUI) sanitizeText(text string) string {
	if !t.hideEnvVars {
		return text
	}

	// Look for environment variable patterns like $VAR or ${VAR}
	patterns := []struct {
		pattern     string
		replacement string
	}{
		{"$HOME", t.sanitizer.SafeGetenv("HOME")},
		{"$USER", t.sanitizer.SafeGetenv("USER")},
		{"$EDITOR", t.sanitizer.SafeGetenv("EDITOR")},
		// Add more as needed
	}

	result := text
	for _, p := range patterns {
		if strings.Contains(result, p.pattern) {
			result = strings.ReplaceAll(result, p.pattern, p.replacement)
		}
	}

	return result
}

// drawWithPreview renders the interface with preview panel
func (t *SecureTUI) drawWithPreview() {
	splitPoint := t.width - t.previewWidth

	// Draw main panel
	title := "Taskopen - Interactive Menu (with Preview)"
	t.drawText(0, 0, splitPoint, title, t.borderStyle.Bold(true))

	// Draw items in left panel
	startY := 2
	for i, item := range t.filtered {
		if i >= t.height-4 {
			break
		}

		style := t.normalStyle
		if i == t.selected {
			style = t.selectedStyle
		}

		text := t.formatItemSecurely(item, i)
		if len(text) > splitPoint-2 {
			text = text[:splitPoint-2]
		}
		t.drawText(0, startY+i, splitPoint, text, style)
	}

	// Draw vertical separator
	t.drawVerticalLine(splitPoint, t.borderStyle)

	// Draw preview panel
	if t.selected < len(t.filtered) {
		t.drawSecurePreview(splitPoint+1, 1, t.previewWidth-1, t.filtered[t.selected])
	}
}

// drawSecurePreview renders a secure preview of the selected item
func (t *SecureTUI) drawSecurePreview(x, y, width int, item MenuItem) {
	// Title
	t.drawText(x, y, width, "Preview", t.previewStyle.Bold(true))
	y++

	// Item details (sanitized)
	lines := []string{
		fmt.Sprintf("ID: %s", item.ID),
		fmt.Sprintf("Text: %s", item.Text),
	}

	if item.Description != "" {
		safeDesc := t.sanitizeText(item.Description)
		lines = append(lines, fmt.Sprintf("Desc: %s", safeDesc))
	}

	// Command preview (if available and safe)
	if data, ok := item.Data.(map[string]any); ok {
		if cmd, exists := data["command"]; exists {
			cmdStr := fmt.Sprintf("%v", cmd)
			// Sanitize command for display
			safeCmdStr := t.sanitizeText(cmdStr)
			lines = append(lines, "", "Command:", safeCmdStr)
		}

		// Environment variables (securely displayed)
		if !t.hideEnvVars {
			if env, exists := data["environment"]; exists {
				if envMap, ok := env.(map[string]string); ok {
					lines = append(lines, "", "Environment:")
					for k, v := range envMap {
						safeValue := t.sanitizer.SanitizeValue(k, v)
						lines = append(lines, fmt.Sprintf("  %s = %s", k, safeValue))
					}
				}
			}
		}
	}

	// Draw preview lines
	for i, line := range lines {
		if y+i >= t.height-2 {
			break
		}
		if len(line) > width {
			line = line[:width]
		}
		t.drawText(x, y+i, width, line, t.previewStyle)
	}
}

// drawHelp renders the help screen
func (t *SecureTUI) drawHelp() {
	helpText := []string{
		"Taskopen Secure TUI - Help",
		"",
		"Navigation:",
		"  ↑/↓ or j/k      Navigate items",
		"  Enter           Select item",
		"  Escape/Ctrl+C   Exit",
		"  Home/End        First/Last item",
		"",
		"Search:",
		"  Ctrl+F or type  Start search",
		"  Backspace       Delete search char",
		"  Ctrl+U          Clear search",
		"  Escape          Exit search",
		"",
		"Preview:",
		"  Tab             Toggle preview mode",
		"",
		"Other:",
		"  F1              Show/hide this help",
		"",
		"Security Features:",
		"  • Environment variables are sanitized",
		"  • Sensitive data is automatically masked",
		"  • Preview content is security-filtered",
		"",
		"Press any key to return...",
	}

	startY := (t.height - len(helpText)) / 2
	for i, line := range helpText {
		if startY+i >= 0 && startY+i < t.height {
			t.drawText(0, startY+i, t.width, line, t.normalStyle)
		}
	}
}

// drawStatusBar renders the status bar at the bottom
func (t *SecureTUI) drawStatusBar() {
	y := t.height - 1

	// Item count
	status := fmt.Sprintf("Item %d of %d", t.selected+1, len(t.filtered))
	if len(t.filtered) != len(t.items) {
		status += fmt.Sprintf(" (filtered from %d)", len(t.items))
	}

	// Mode indicator
	modeText := "Interactive"
	switch t.mode {
	case ModeSearch:
		modeText = "Search"
	case ModePreview:
		modeText = "Preview"
	case ModeHelp:
		modeText = "Help"
	}

	// Security status
	securityText := ""
	if t.hideEnvVars {
		securityText = " [ENV:HIDDEN]"
	} else {
		securityText = fmt.Sprintf(" [ENV:%s]", t.getVisibilityText())
	}

	fullStatus := fmt.Sprintf("%s | Mode: %s%s | F1:Help", status, modeText, securityText)
	t.drawText(0, y, t.width, fullStatus, t.borderStyle)
}

// getVisibilityText returns a human-readable visibility level
func (t *SecureTUI) getVisibilityText() string {
	switch t.visibilityLevel {
	case security.VisibilityHidden:
		return "HIDDEN"
	case security.VisibilityMasked:
		return "MASKED"
	case security.VisibilityLimited:
		return "LIMITED"
	case security.VisibilityFull:
		return "FULL"
	default:
		return "UNKNOWN"
	}
}

// Helper drawing functions

// drawText draws text at the specified position with word wrapping
func (t *SecureTUI) drawText(x, y, maxWidth int, text string, style tcell.Style) {
	for i, r := range text {
		if x+i >= maxWidth || x+i >= t.width {
			break
		}
		t.screen.SetContent(x+i, y, r, nil, style)
	}
}

// drawHorizontalLine draws a horizontal line
func (t *SecureTUI) drawHorizontalLine(y int, style tcell.Style) {
	for x := 0; x < t.width; x++ {
		t.screen.SetContent(x, y, '─', nil, style)
	}
}

// drawVerticalLine draws a vertical line
func (t *SecureTUI) drawVerticalLine(x int, style tcell.Style) {
	for y := 0; y < t.height; y++ {
		t.screen.SetContent(x, y, '│', nil, style)
	}
}

// Close properly shuts down the TUI
func (t *SecureTUI) Close() {
	if t.screen != nil {
		t.screen.Fini()
	}
}

// DefaultSecureTUIConfig returns sensible defaults for secure TUI
func DefaultSecureTUIConfig() SecureTUIConfig {
	return SecureTUIConfig{
		ShowPreview:       true,
		PreviewWidth:      40,
		HideEnvVars:       true, // Default to secure
		VisibilityLevel:   security.VisibilityMasked,
		AccessibilityMode: output.AccessibilityNormal,
	}
}
