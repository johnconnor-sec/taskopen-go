// Package ui provides advanced Terminal User Interface components
package ui

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/johnconnor-sec/taskopen-go/internal/output"
	"github.com/johnconnor-sec/taskopen-go/internal/search"
)

// TUIMode represents different TUI interaction modes
type TUIMode int

const (
	ModeInteractive TUIMode = iota
	ModePreview
	ModeHelp
	ModeSearch
)

// AdvancedMenu provides a rich terminal interface with real-time search and preview
type AdvancedMenu struct {
	config       MenuConfig
	items        []MenuItem
	filtered     []MenuItem
	selected     int
	query        string
	mode         TUIMode
	formatter    *output.Formatter
	fuzzy        *search.Fuzzy
	previewWidth int
	showPreview  bool
	multiSelect  bool
	selections   map[int]bool
	mu           sync.RWMutex
	termWidth    int
	termHeight   int
}

// NewAdvancedMenu creates a new advanced terminal UI menu
func NewAdvancedMenu(items []MenuItem, config MenuConfig) *AdvancedMenu {
	fuzzy := search.NewFuzzy().
		SetCaseSensitive(config.CaseSensitive).
		SetMinScore(config.MinScore).
		SetHighlightMatches(true)

	menu := &AdvancedMenu{
		config:       config,
		items:        items,
		filtered:     items,
		selected:     0,
		query:        "",
		mode:         ModeInteractive,
		formatter:    output.NewFormatter(os.Stdout),
		fuzzy:        fuzzy,
		previewWidth: 40,
		showPreview:  config.PreviewFunc != nil,
		multiSelect:  false,
		selections:   make(map[int]bool),
		termWidth:    getTerminalWidth(),
		termHeight:   getTerminalHeight(),
	}

	// Find first non-disabled item
	for i, item := range menu.filtered {
		if !item.Disabled {
			menu.selected = i
			break
		}
	}

	return menu
}

// SetMultiSelect enables or disables multi-selection mode
func (m *AdvancedMenu) SetMultiSelect(enabled bool) *AdvancedMenu {
	m.multiSelect = enabled
	return m
}

// Show displays the advanced menu and returns the selection
func (m *AdvancedMenu) Show() (interface{}, error) {
	// Setup terminal
	if err := m.setupTerminal(); err != nil {
		return m.fallbackToSimpleMenu()
	}
	defer m.restoreTerminal()

	// Main interaction loop
	for {
		m.render()

		key, err := m.readKey()
		if err != nil {
			return nil, err
		}

		result, done := m.handleKey(key)
		if done {
			if m.multiSelect {
				return m.getMultipleSelections(), nil
			}
			return result, nil
		}
	}
}

// setupTerminal configures terminal for raw input
func (m *AdvancedMenu) setupTerminal() error {
	// Enable raw mode (simplified implementation)
	// In production, we'd use a proper terminal library like tcell or termbox
	fmt.Print("\033[?25l")   // Hide cursor
	fmt.Print("\033[?1049h") // Switch to alternate screen
	return nil
}

// restoreTerminal restores terminal settings
func (m *AdvancedMenu) restoreTerminal() {
	fmt.Print("\033[?1049l") // Switch back to main screen
	fmt.Print("\033[?25h")   // Show cursor
}

// readKey reads a single key press (enhanced version)
func (m *AdvancedMenu) readKey() (KeyEvent, error) {
	// This is a more sophisticated version that handles ANSI escape sequences
	var buf [8]byte
	n, err := os.Stdin.Read(buf[:])
	if err != nil {
		return KeyEvent{}, err
	}

	if n == 0 {
		return KeyEvent{Code: KeyEscape}, nil
	}

	// Handle escape sequences
	if buf[0] == 27 { // ESC
		if n == 1 {
			return KeyEvent{Code: KeyEscape}, nil
		}

		if n >= 3 && buf[1] == '[' {
			switch buf[2] {
			case 'A':
				return KeyEvent{Code: KeyUp}, nil
			case 'B':
				return KeyEvent{Code: KeyDown}, nil
			case 'C':
				return KeyEvent{Code: KeyRight}, nil
			case 'D':
				return KeyEvent{Code: KeyLeft}, nil
			case 'H':
				return KeyEvent{Code: KeyHome}, nil
			case 'F':
				return KeyEvent{Code: KeyEnd}, nil
			case '5':
				if n >= 4 && buf[3] == '~' {
					return KeyEvent{Code: KeyPageUp}, nil
				}
			case '6':
				if n >= 4 && buf[3] == '~' {
					return KeyEvent{Code: KeyPageDown}, nil
				}
			}
		}
		return KeyEvent{Code: KeyEscape}, nil
	}

	// Handle regular keys
	switch buf[0] {
	case 13, 10: // Enter
		return KeyEvent{Code: KeyEnter}, nil
	case 127, 8: // Backspace/Delete
		return KeyEvent{Code: KeyBackspace}, nil
	case 9: // Tab
		return KeyEvent{Code: KeyTab}, nil
	case 3: // Ctrl+C
		return KeyEvent{Code: KeyEscape}, nil
	case 16: // Ctrl+P (Previous)
		return KeyEvent{Code: KeyUp}, nil
	case 14: // Ctrl+N (Next)
		return KeyEvent{Code: KeyDown}, nil
	case 6: // Ctrl+F (Forward/Preview)
		return KeyEvent{Code: KeyRight}, nil
	case 2: // Ctrl+B (Backward)
		return KeyEvent{Code: KeyLeft}, nil
	case 1: // Ctrl+A (Home)
		return KeyEvent{Code: KeyHome}, nil
	case 5: // Ctrl+E (End)
		return KeyEvent{Code: KeyEnd}, nil
	case 32: // Space
		return KeyEvent{Code: KeyChar, Char: ' '}, nil
	case 47: // / (search mode)
		return KeyEvent{Code: KeyChar, Char: '/'}, nil
	case 63: // ? (help mode)
		return KeyEvent{Code: KeyChar, Char: '?'}, nil
	default:
		if buf[0] >= 32 && buf[0] < 127 { // Printable ASCII
			return KeyEvent{Code: KeyChar, Char: rune(buf[0])}, nil
		}
	}

	return KeyEvent{Code: KeyChar, Char: rune(buf[0])}, nil
}

// handleKey processes key events and returns result/done status
func (m *AdvancedMenu) handleKey(key KeyEvent) (interface{}, bool) {
	switch m.mode {
	case ModeSearch:
		return m.handleSearchMode(key)
	case ModeHelp:
		return m.handleHelpMode(key)
	case ModePreview:
		return m.handlePreviewMode(key)
	default:
		return m.handleInteractiveMode(key)
	}
}

// handleInteractiveMode handles keys in normal interactive mode
func (m *AdvancedMenu) handleInteractiveMode(key KeyEvent) (interface{}, bool) {
	switch key.Code {
	case KeyEscape:
		return nil, true
	case KeyEnter:
		if len(m.filtered) > 0 && m.selected < len(m.filtered) {
			if m.multiSelect {
				m.toggleSelection()
				return m.getMultipleSelections(), len(m.selections) > 0
			}
			return &m.filtered[m.selected], true
		}
	case KeyUp:
		m.moveSelection(-1)
	case KeyDown:
		m.moveSelection(1)
	case KeyPageUp:
		m.moveSelection(-5)
	case KeyPageDown:
		m.moveSelection(5)
	case KeyHome:
		m.selected = 0
		m.ensureValidSelection()
	case KeyEnd:
		m.selected = len(m.filtered) - 1
		m.ensureValidSelection()
	case KeyRight:
		if m.showPreview {
			m.mode = ModePreview
		}
	case KeyBackspace:
		if len(m.query) > 0 {
			m.query = m.query[:len(m.query)-1]
			m.updateFilter()
		}
	case KeyTab:
		if m.multiSelect {
			m.toggleSelection()
		} else {
			m.showPreview = !m.showPreview
		}
	case KeyChar:
		switch key.Char {
		case '/':
			m.mode = ModeSearch
			m.query = ""
		case '?':
			m.mode = ModeHelp
		case 'q':
			return nil, true
		case 'j': // Vim-style navigation
			m.moveSelection(1)
		case 'k':
			m.moveSelection(-1)
		case 'g':
			m.selected = 0
			m.ensureValidSelection()
		case 'G':
			m.selected = len(m.filtered) - 1
			m.ensureValidSelection()
		case ' ':
			if m.multiSelect {
				m.toggleSelection()
			}
		case 'p':
			m.showPreview = !m.showPreview
		default:
			// Add to search query for quick filtering
			m.query += string(key.Char)
			m.updateFilter()
		}
	}
	return nil, false
}

// handleSearchMode handles keys in search mode
func (m *AdvancedMenu) handleSearchMode(key KeyEvent) (interface{}, bool) {
	switch key.Code {
	case KeyEscape:
		m.mode = ModeInteractive
		m.query = ""
		m.updateFilter()
	case KeyEnter:
		m.mode = ModeInteractive
	case KeyBackspace:
		if len(m.query) > 0 {
			m.query = m.query[:len(m.query)-1]
			m.updateFilter()
		}
	case KeyChar:
		m.query += string(key.Char)
		m.updateFilter()
	}
	return nil, false
}

// handleHelpMode handles keys in help mode
func (m *AdvancedMenu) handleHelpMode(key KeyEvent) (interface{}, bool) {
	m.mode = ModeInteractive
	return nil, false
}

// handlePreviewMode handles keys in preview mode
func (m *AdvancedMenu) handlePreviewMode(key KeyEvent) (interface{}, bool) {
	switch key.Code {
	case KeyEscape, KeyLeft:
		m.mode = ModeInteractive
	case KeyEnter:
		if len(m.filtered) > 0 && m.selected < len(m.filtered) {
			return &m.filtered[m.selected], true
		}
	case KeyUp:
		m.moveSelection(-1)
	case KeyDown:
		m.moveSelection(1)
	}
	return nil, false
}

// render displays the current menu state
func (m *AdvancedMenu) render() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Clear screen
	fmt.Print("\033[2J\033[H")

	if m.mode == ModeHelp {
		m.renderHelp()
		return
	}

	// Calculate layout
	contentWidth := m.termWidth
	if m.showPreview {
		contentWidth = m.termWidth - m.previewWidth - 3
	}

	// Render header
	m.renderHeader()

	// Render search bar
	if m.mode == ModeSearch || m.query != "" {
		m.renderSearchBar()
	}

	// Split screen for preview mode
	if m.showPreview && m.config.PreviewFunc != nil {
		m.renderSplitView(contentWidth)
	} else {
		m.renderFullView()
	}

	// Render footer
	m.renderFooter()
}

// renderHeader displays the menu title and status
func (m *AdvancedMenu) renderHeader() {
	title := m.config.Title
	if m.multiSelect && len(m.selections) > 0 {
		title += fmt.Sprintf(" (%d selected)", len(m.selections))
	}

	m.formatter.Header(title)

	// Show current mode
	modeText := ""
	switch m.mode {
	case ModeSearch:
		modeText = "SEARCH MODE"
	case ModePreview:
		modeText = "PREVIEW MODE"
	case ModeHelp:
		modeText = "HELP MODE"
	}

	if modeText != "" {
		fmt.Printf("Mode: %s\n\n", modeText)
	}
}

// renderSearchBar displays the search input
func (m *AdvancedMenu) renderSearchBar() {
	prompt := "Search: "
	query := m.query
	if m.mode == ModeSearch {
		query += "█" // Cursor
	}

	fmt.Printf("%s%s\n", prompt, query)

	if len(m.filtered) != len(m.items) {
		fmt.Printf("Filtered: %d/%d items\n", len(m.filtered), len(m.items))
	}
	fmt.Println()
}

// renderSplitView displays menu with preview panel
func (m *AdvancedMenu) renderSplitView(contentWidth int) {
	maxDisplay := min(m.termHeight-10, len(m.filtered))
	if maxDisplay <= 0 {
		return
	}

	for i := 0; i < maxDisplay; i++ {
		if i >= len(m.filtered) {
			break
		}

		item := m.filtered[i]
		line := m.formatMenuItem(item, i, contentWidth)
		fmt.Printf("%-*s │", contentWidth, line)

		// Add preview content on the same line for first item
		if i == 0 && len(m.filtered) > 0 && m.selected < len(m.filtered) {
			preview := m.config.PreviewFunc(m.filtered[m.selected])
			previewLines := strings.Split(preview, "\n")
			if len(previewLines) > 0 {
				fmt.Printf(" %s", truncateString(previewLines[0], m.previewWidth))
			}
		} else if i < len(strings.Split(m.getPreviewContent(), "\n")) {
			previewLines := strings.Split(m.getPreviewContent(), "\n")
			if i < len(previewLines) {
				fmt.Printf(" %s", truncateString(previewLines[i], m.previewWidth))
			}
		}
		fmt.Println()
	}
}

// renderFullView displays menu in full width
func (m *AdvancedMenu) renderFullView() {
	if len(m.filtered) == 0 {
		m.formatter.Warning("No items found")
		return
	}

	// Calculate visible range
	maxDisplay := min(m.config.MaxItems, m.termHeight-8)
	start := 0
	end := min(len(m.filtered), maxDisplay)

	// Center view around selected item
	if m.selected >= maxDisplay/2 {
		start = max(0, m.selected-maxDisplay/2)
		end = min(len(m.filtered), start+maxDisplay)
	}

	for i := start; i < end; i++ {
		item := m.filtered[i]
		line := m.formatMenuItem(item, i, m.termWidth-4)
		fmt.Println(line)

		// Show description for selected item
		if i == m.selected && m.config.ShowDescription && item.Description != "" {
			desc := "    " + item.Description
			fmt.Printf("    %s\n", desc) // Simplified - in real implementation we'd access colorize method
		}
	}
}

// formatMenuItem formats a single menu item
func (m *AdvancedMenu) formatMenuItem(item MenuItem, index int, width int) string {
	// Selection marker
	marker := "  "
	if index == m.selected {
		marker = "→ "
	}

	// Multi-select checkbox
	if m.multiSelect {
		checkbox := "☐ "
		if m.selections[index] {
			checkbox = "☑ "
		}
		marker = checkbox + marker
	}

	// Format text with highlighting
	text := item.Text
	if m.query != "" {
		if match, ok := m.fuzzy.Match(m.query, item.Text); ok {
			text = m.fuzzy.HighlightString(item.Text, match.Highlights, "⚡", "⚡")
		}
	}

	line := marker + text

	// Apply colors and styles
	if item.Disabled {
		line = "  " + line // Muted disabled items
	} else if index == m.selected {
		// Highlight selected item (would use formatter.colorize in real implementation)
		line = "► " + line
	}

	return truncateString(line, width)
}

// renderFooter displays help and status information
func (m *AdvancedMenu) renderFooter() {
	fmt.Println()

	// Show current position
	if len(m.filtered) > 0 {
		fmt.Printf("Item %d of %d", m.selected+1, len(m.filtered))
		if len(m.filtered) != len(m.items) {
			fmt.Printf(" (filtered from %d)", len(m.items))
		}
		fmt.Println()
	}

	// Show key bindings based on mode
	switch m.mode {
	case ModeSearch:
		fmt.Println("Search: Type to filter • Enter: Exit search • Esc: Cancel")
	case ModePreview:
		fmt.Println("Preview: ←/Esc: Back • ↑/↓: Navigate • Enter: Select")
	case ModeHelp:
		fmt.Println("Press any key to return...")
	default:
		bindings := []string{"↑/↓/j/k: Navigate", "Enter: Select", "Esc/q: Quit"}
		if m.multiSelect {
			bindings = append(bindings, "Space/Tab: Toggle")
		}
		if m.showPreview {
			bindings = append(bindings, "→: Preview mode")
		}
		bindings = append(bindings, "/: Search", "?: Help")

		fmt.Println(strings.Join(bindings, " • "))
	}
}

// renderHelp displays the help screen
func (m *AdvancedMenu) renderHelp() {
	m.formatter.Header("Taskopen Interactive Menu - Help")

	fmt.Println("Navigation:")
	m.formatter.List("↑/↓ or j/k - Move selection up/down")
	m.formatter.List("PageUp/PageDown - Move selection by 5 items")
	m.formatter.List("Home/g - Go to first item")
	m.formatter.List("End/G - Go to last item")
	fmt.Println()

	fmt.Println("Selection:")
	m.formatter.List("Enter - Select current item")
	if m.multiSelect {
		m.formatter.List("Space/Tab - Toggle selection")
	}
	m.formatter.List("Esc/q - Quit without selection")
	fmt.Println()

	fmt.Println("Search & Filter:")
	m.formatter.List("/ - Enter search mode")
	m.formatter.List("Type characters - Quick filter")
	m.formatter.List("Backspace - Remove last character")
	fmt.Println()

	if m.showPreview {
		fmt.Println("Preview:")
		m.formatter.List("→ - Enter preview mode")
		m.formatter.List("p - Toggle preview panel")
		fmt.Println()
	}

	fmt.Println("Advanced:")
	m.formatter.List("Ctrl+C - Emergency exit")
	m.formatter.List("? - Show this help")

	fmt.Println("\nPress any key to return...")
}

// Helper functions

func (m *AdvancedMenu) moveSelection(delta int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.filtered) == 0 {
		return
	}

	m.selected += delta
	m.ensureValidSelection()
}

func (m *AdvancedMenu) ensureValidSelection() {
	if m.selected < 0 {
		m.selected = 0
	}
	if m.selected >= len(m.filtered) {
		m.selected = len(m.filtered) - 1
	}

	// Skip disabled items
	if len(m.filtered) > 0 {
		startSelected := m.selected
		for m.selected < len(m.filtered) && m.filtered[m.selected].Disabled {
			m.selected++
		}
		if m.selected >= len(m.filtered) {
			m.selected = startSelected
			for m.selected >= 0 && m.filtered[m.selected].Disabled {
				m.selected--
			}
		}
		if m.selected < 0 {
			m.selected = 0
		}
	}
}

func (m *AdvancedMenu) updateFilter() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.query == "" {
		m.filtered = m.items
	} else {
		texts := make([]string, len(m.items))
		for i, item := range m.items {
			texts[i] = item.Text + " " + item.Description // Search both text and description
		}

		matches := m.fuzzy.Search(m.query, texts)

		m.filtered = make([]MenuItem, 0, len(matches))
		for _, match := range matches {
			for _, item := range m.items {
				searchText := item.Text + " " + item.Description
				if searchText == match.Text {
					m.filtered = append(m.filtered, item)
					break
				}
			}
		}
	}

	m.selected = 0
	m.ensureValidSelection()
}

func (m *AdvancedMenu) toggleSelection() {
	if m.selected < len(m.filtered) {
		m.selections[m.selected] = !m.selections[m.selected]
	}
}

func (m *AdvancedMenu) getMultipleSelections() []MenuItem {
	var selected []MenuItem
	for i := range m.filtered {
		if m.selections[i] {
			selected = append(selected, m.filtered[i])
		}
	}
	return selected
}

func (m *AdvancedMenu) getPreviewContent() string {
	if len(m.filtered) > 0 && m.selected < len(m.filtered) && m.config.PreviewFunc != nil {
		return m.config.PreviewFunc(m.filtered[m.selected])
	}
	return "No preview available"
}

func (m *AdvancedMenu) fallbackToSimpleMenu() (interface{}, error) {
	// Fallback to simple menu if terminal setup fails
	item, err := ShowSimpleMenu(m.items, m.config.Title)
	return item, err
}

// Utility functions

func getTerminalWidth() int {
	if width := os.Getenv("COLUMNS"); width != "" {
		if w, err := strconv.Atoi(width); err == nil && w > 0 {
			return w
		}
	}
	return 80
}

func getTerminalHeight() int {
	if height := os.Getenv("LINES"); height != "" {
		if h, err := strconv.Atoi(height); err == nil && h > 0 {
			return h
		}
	}
	return 24
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
