package ui

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/johnconnor-sec/taskopen-go/internal/output"
	"github.com/johnconnor-sec/taskopen-go/internal/search"
)

// KeyCode represents keyboard input codes
type KeyCode int

const (
	KeyEnter KeyCode = iota
	KeyEscape
	KeyUp
	KeyDown
	KeyLeft
	KeyRight
	KeyTab
	KeyBackspace
	KeyDelete
	KeyHome
	KeyEnd
	KeyPageUp
	KeyPageDown
	KeyChar // For regular character input
	// Vim-style navigation
	KeyVimUp     // k
	KeyVimDown   // j
	KeyVimLeft   // h
	KeyVimRight  // l
	KeyVimFirst  // gg (go to first)
	KeyVimLast   // G (go to last)
	KeyVimSearch // / (start search)
	KeyVimQuit   // q (quit)
	KeyVimHelp   // ? (help)
	KeyVimSelect // space (toggle selection)
	KeyVimEnter  // enter/return
	// Accessibility shortcuts
	KeyAccessibilityMode // F12
	KeySpeak             // Ctrl+S (speak current item)
	KeyDescribe          // Ctrl+D (describe current item)
)

// KeyEvent represents a keyboard event
type KeyEvent struct {
	Code KeyCode
	Char rune
	Alt  bool
	Ctrl bool
}

// MenuLayout defines different menu layout styles
type MenuLayout int

const (
	LayoutDefault MenuLayout = iota // Traditional vertical list
	LayoutCompact                   // Minimal spacing
	LayoutTable                     // Tabular format with columns
	LayoutCards                     // Card-style with borders
	LayoutTree                      // Hierarchical tree view
)

// MenuTheme defines color and styling themes
type MenuTheme int

const (
	ThemeDefault      MenuTheme = iota // Default terminal colors
	ThemeDark                          // Dark theme optimized
	ThemeLight                         // Light theme optimized
	ThemeHighContrast                  // Accessibility high contrast
	ThemeVim                           // Vim-inspired colors
	ThemeModern                        // Modern UI colors
)

// MenuItem represents a menu item
type MenuItem struct {
	ID          string
	Text        string
	Description string
	Action      func() error
	Disabled    bool
	Data        any
}

// MenuConfig configures menu behavior and appearance
type MenuConfig struct {
	Title           string
	Prompt          string
	ShowDescription bool
	AllowSearch     bool
	MaxItems        int
	MinScore        float64
	CaseSensitive   bool
	PreviewFunc     func(item MenuItem) string
	// Keyboard navigation
	VimMode           bool // Enable vim-style key bindings
	AccessibilityMode bool // Enable accessibility features
	// Multi-selection
	AllowMultiSelect bool
	// Help system
	ShowHelp     bool
	CustomHelp   string
	HelpCallback func() string
	// Layout and theming
	Layout     MenuLayout
	ThemeStyle MenuTheme
}

// DefaultMenuConfig returns sensible defaults
func DefaultMenuConfig() MenuConfig {
	return MenuConfig{
		Title:             "Select an option",
		Prompt:            "> ",
		ShowDescription:   true,
		AllowSearch:       true,
		MaxItems:          10,
		MinScore:          0.1,
		CaseSensitive:     false,
		VimMode:           true,  // Enable vim bindings by default
		AccessibilityMode: false, // Detect automatically or user preference
		AllowMultiSelect:  false,
		ShowHelp:          true,
		Layout:            LayoutDefault, // Standard vertical list
		ThemeStyle:        ThemeDefault,  // Use terminal default colors
	}
}

// Menu represents an interactive menu
type Menu struct {
	config          MenuConfig
	items           []MenuItem
	filtered        []MenuItem
	selected        int
	query           string
	formatter       *output.Formatter
	fuzzy           *search.Fuzzy
	input           io.Reader
	output          io.Writer
	mu              sync.RWMutex // Protect concurrent access to menu state
	searchTicker    *time.Ticker // For debounced search updates
	searchChan      chan string  // Channel for search query updates
	selectedItems   map[int]bool // For multi-selection (filtered item indices)
	lastSpokenIndex int          // For accessibility - track last spoken item
	helpVisible     bool         // Whether help is currently visible
}

// NewMenu creates a new interactive menu
func NewMenu(items []MenuItem, config MenuConfig) *Menu {
	fuzzy := search.NewFuzzy().
		SetCaseSensitive(config.CaseSensitive).
		SetMinScore(config.MinScore)

	formatter := output.NewFormatter(os.Stdout)

	// Apply theme to formatter
	menu := &Menu{
		config:          config,
		items:           items,
		filtered:        items,
		selected:        0,
		query:           "",
		formatter:       formatter,
		fuzzy:           fuzzy,
		input:           os.Stdin,
		output:          os.Stdout,
		searchChan:      make(chan string, 1), // Buffered channel for debounced updates
		selectedItems:   make(map[int]bool),   // Initialize multi-selection map
		lastSpokenIndex: -1,                   // No item spoken yet
		helpVisible:     false,                // Help starts hidden
	}

	// Apply theme settings
	menu.applyTheme()

	// Start background search processor for real-time updates
	go menu.processSearchUpdates()

	return menu
}

// processSearchUpdates handles debounced search updates in the background
func (m *Menu) processSearchUpdates() {
	debounceTimer := time.NewTimer(0)
	debounceTimer.Stop() // Stop initial timer

	for {
		select {
		case query := <-m.searchChan:
			// Reset the debounce timer
			debounceTimer.Stop()
			debounceTimer.Reset(50 * time.Millisecond) // 50ms debounce

			// Wait for debounce period
			go func(q string) {
				<-debounceTimer.C
				m.query = q
				m.updateFilter()
			}(query)
		}
	}
}

// handleKeyEvent processes keyboard events with vim-style navigation support
func (m *Menu) handleKeyEvent(key KeyEvent) bool {
	switch key.Code {
	// Standard navigation
	case KeyUp:
		m.moveSelection(-1)
		m.speakCurrentItemIfAccessible()
		return true
	case KeyDown:
		m.moveSelection(1)
		m.speakCurrentItemIfAccessible()
		return true
	case KeyPageUp:
		m.moveSelection(-m.config.MaxItems)
		m.speakCurrentItemIfAccessible()
		return true
	case KeyPageDown:
		m.moveSelection(m.config.MaxItems)
		m.speakCurrentItemIfAccessible()
		return true
	case KeyHome:
		m.jumpToFirst()
		m.speakCurrentItemIfAccessible()
		return true
	case KeyEnd:
		m.jumpToLast()
		m.speakCurrentItemIfAccessible()
		return true

	// Vim-style navigation (if enabled)
	case KeyVimUp, KeyChar:
		if m.config.VimMode && (key.Code == KeyVimUp || key.Char == 'k') {
			m.moveSelection(-1)
			m.speakCurrentItemIfAccessible()
			return true
		}
		return m.handleCharacterInput(key)
	case KeyVimDown:
		if m.config.VimMode {
			m.moveSelection(1)
			m.speakCurrentItemIfAccessible()
			return true
		}
		return false
	case KeyVimFirst:
		if m.config.VimMode {
			m.jumpToFirst()
			m.speakCurrentItemIfAccessible()
			return true
		}
		return false
	case KeyVimLast:
		if m.config.VimMode {
			m.jumpToLast()
			m.speakCurrentItemIfAccessible()
			return true
		}
		return false

	// Multi-selection
	case KeyVimSelect:
		if m.config.AllowMultiSelect {
			m.toggleSelection()
			return true
		}
		return false

	// Help system
	case KeyVimHelp:
		m.toggleHelp()
		return true

	// Search and text input
	case KeyVimSearch:
		if m.config.AllowSearch {
			// Enter search mode (clear query and wait for input)
			m.query = ""
			m.updateFilter()
			return true
		}
		return false
	case KeyBackspace:
		if len(m.query) > 0 {
			m.query = m.query[:len(m.query)-1]
			m.sendSearchQuery(m.query)
			return true
		}
		return false

	// Accessibility
	case KeySpeak:
		if m.config.AccessibilityMode {
			m.speakCurrentItem()
			return true
		}
		return false
	case KeyDescribe:
		if m.config.AccessibilityMode {
			m.describeCurrentItem()
			return true
		}
		return false
	case KeyAccessibilityMode:
		m.toggleAccessibilityMode()
		return true

	default:
		return m.handleCharacterInput(key)
	}
}

// handleCharacterInput processes character input for search
func (m *Menu) handleCharacterInput(key KeyEvent) bool {
	if key.Code == KeyChar && m.config.AllowSearch {
		// Handle vim navigation first if enabled
		if m.config.VimMode {
			switch key.Char {
			case 'j':
				m.moveSelection(1)
				m.speakCurrentItemIfAccessible()
				return true
			case 'k':
				m.moveSelection(-1)
				m.speakCurrentItemIfAccessible()
				return true
			case 'h':
				// Left - could be used for collapsing in tree menus
				return true
			case 'l':
				// Right - could be used for expanding in tree menus
				return true
			case 'g':
				// Handle 'gg' for first (would need state tracking)
				m.jumpToFirst()
				m.speakCurrentItemIfAccessible()
				return true
			case 'G':
				m.jumpToLast()
				m.speakCurrentItemIfAccessible()
				return true
			case ' ':
				if m.config.AllowMultiSelect {
					m.toggleSelection()
					return true
				}
				fallthrough // If no multi-select, treat as search character
			case '/':
				// Start search mode
				if m.config.AllowSearch {
					m.query = ""
					return true
				}
				return false
			case 'q':
				// This will be handled as KeyVimQuit in main loop
				return false
			case '?':
				m.toggleHelp()
				return true
			default:
				// Regular character for search
				m.query += string(key.Char)
				m.sendSearchQuery(m.query)
				return true
			}
		} else {
			// Non-vim mode - all characters go to search
			m.query += string(key.Char)
			m.sendSearchQuery(m.query)
			return true
		}
	}
	return false
}

// sendSearchQuery sends query for debounced processing
func (m *Menu) sendSearchQuery(query string) {
	select {
	case m.searchChan <- query:
	default:
		// Channel full, skip to prevent blocking
	}
}

// jumpToFirst moves selection to first item
func (m *Menu) jumpToFirst() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.filtered) > 0 {
		m.selected = 0
		// Find first non-disabled item
		for i, item := range m.filtered {
			if !item.Disabled {
				m.selected = i
				break
			}
		}
	}
}

// jumpToLast moves selection to last item
func (m *Menu) jumpToLast() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.filtered) > 0 {
		m.selected = len(m.filtered) - 1
		// Find last non-disabled item
		for i := len(m.filtered) - 1; i >= 0; i-- {
			if !m.filtered[i].Disabled {
				m.selected = i
				break
			}
		}
	}
}

// toggleSelection toggles selection state for current item
func (m *Menu) toggleSelection() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.filtered) > 0 && m.selected < len(m.filtered) {
		if m.selectedItems[m.selected] {
			delete(m.selectedItems, m.selected)
		} else {
			m.selectedItems[m.selected] = true
		}
	}
}

// toggleHelp toggles help visibility
func (m *Menu) toggleHelp() {
	m.helpVisible = !m.helpVisible
}

// toggleAccessibilityMode toggles accessibility features
func (m *Menu) toggleAccessibilityMode() {
	m.config.AccessibilityMode = !m.config.AccessibilityMode
	if m.config.AccessibilityMode {
		m.formatter.SetAccessibilityMode(output.AccessibilityScreenReader)
		m.speakCurrentItem()
	} else {
		m.formatter.SetAccessibilityMode(output.AccessibilityNormal)
	}
}

// speakCurrentItemIfAccessible speaks current item if accessibility is enabled
func (m *Menu) speakCurrentItemIfAccessible() {
	if m.config.AccessibilityMode {
		m.speakCurrentItem()
	}
}

// speakCurrentItem announces the current item for screen readers
func (m *Menu) speakCurrentItem() {
	if len(m.filtered) > 0 && m.selected < len(m.filtered) {
		item := m.filtered[m.selected]
		text := fmt.Sprintf("Item %d of %d: %s", m.selected+1, len(m.filtered), item.Text)
		if item.Description != "" {
			text += ". " + item.Description
		}
		if m.config.AllowMultiSelect && m.selectedItems[m.selected] {
			text += ". Selected"
		}
		m.formatter.ScreenReaderText("navigation", text)
		m.lastSpokenIndex = m.selected
	}
}

// describeCurrentItem provides detailed description of current item
func (m *Menu) describeCurrentItem() {
	if len(m.filtered) > 0 && m.selected < len(m.filtered) {
		item := m.filtered[m.selected]
		text := fmt.Sprintf("Detailed description for %s: %s", item.Text, item.Description)

		// Add additional context if available
		if item.Data != nil {
			text += fmt.Sprintf(". Additional data available: %T", item.Data)
		}

		m.formatter.ScreenReaderText("description", text)
	}
}

// renderHelp displays comprehensive help information
func (m *Menu) renderHelp() {
	m.formatter.Subheader("📖 Help")

	fmt.Fprintln(m.output, "Navigation:")
	if m.config.VimMode {
		fmt.Fprintln(m.output, "  j/k or ↓/↑     Move up/down")
		fmt.Fprintln(m.output, "  h/l or ←/→     Move left/right (future use)")
		fmt.Fprintln(m.output, "  gg/G           Jump to first/last item")
		fmt.Fprintln(m.output, "  q              Quit/cancel")
		fmt.Fprintln(m.output, "  /              Start search")
		fmt.Fprintln(m.output, "  ?              Toggle this help")
	} else {
		fmt.Fprintln(m.output, "  ↑/↓            Move up/down")
		fmt.Fprintln(m.output, "  PageUp/PageDown Large jumps")
		fmt.Fprintln(m.output, "  Home/End       First/last item")
	}

	fmt.Fprintln(m.output, "\nSelection:")
	fmt.Fprintln(m.output, "  Enter          Select current item")
	if m.config.AllowMultiSelect {
		fmt.Fprintln(m.output, "  Space          Toggle selection")
	}
	fmt.Fprintln(m.output, "  Esc            Cancel/exit")

	if m.config.AllowSearch {
		fmt.Fprintln(m.output, "\nSearch:")
		fmt.Fprintln(m.output, "  Type           Filter items")
		fmt.Fprintln(m.output, "  Backspace      Delete character")
	}

	if m.config.AccessibilityMode {
		fmt.Fprintln(m.output, "\nAccessibility:")
		fmt.Fprintln(m.output, "  Ctrl+S         Speak current item")
		fmt.Fprintln(m.output, "  Ctrl+D         Describe current item")
		fmt.Fprintln(m.output, "  F12            Toggle accessibility mode")
	}

	if m.config.CustomHelp != "" {
		fmt.Fprintln(m.output, "\nCustom:")
		fmt.Fprintln(m.output, "  "+m.config.CustomHelp)
	}

	if m.config.HelpCallback != nil {
		fmt.Fprintln(m.output, "\n"+m.config.HelpCallback())
	}

	fmt.Fprintln(m.output, "\nPress any key to close help...")
}

// renderStatusLine displays a compact status/help line
func (m *Menu) renderStatusLine() {
	var parts []string

	// Navigation help
	if m.config.VimMode {
		parts = append(parts, "j/k Navigate")
	} else {
		parts = append(parts, "↑/↓ Navigate")
	}

	// Selection
	parts = append(parts, "Enter Select")
	if m.config.AllowMultiSelect {
		parts = append(parts, "Space Toggle")
		selectedCount := len(m.selectedItems)
		if selectedCount > 0 {
			parts = append(parts, fmt.Sprintf("(%d selected)", selectedCount))
		}
	}

	// Exit
	if m.config.VimMode {
		parts = append(parts, "q Quit")
	} else {
		parts = append(parts, "Esc Cancel")
	}

	// Search
	if m.config.AllowSearch {
		parts = append(parts, "/ Search")
	}

	// Help
	parts = append(parts, "? Help")

	// Accessibility indicator
	if m.config.AccessibilityMode {
		parts = append(parts, "🔊 A11y")
	}

	status := strings.Join(parts, " • ")
	fmt.Fprintln(m.output, status)
}

// applyTheme applies the selected theme to the formatter
func (m *Menu) applyTheme() {
	switch m.config.ThemeStyle {
	case ThemeDefault:
		m.formatter.SetTheme(output.DefaultTheme)
	case ThemeDark:
		m.formatter.SetTheme(output.DarkTheme)
	case ThemeLight:
		// Light theme - use default theme
		m.formatter.SetTheme(output.DefaultTheme)
	case ThemeHighContrast:
		m.formatter.SetAccessibilityMode(output.AccessibilityHighContrast)
		m.formatter.SetTheme(output.HighContrastTheme)
	case ThemeVim:
		// Vim-inspired theme - use dark theme
		m.formatter.SetTheme(output.DarkTheme)
	case ThemeModern:
		// Modern theme - use dark theme as base
		m.formatter.SetTheme(output.DarkTheme)
	}
}

// renderItemsWithLayout renders menu items based on the configured layout
func (m *Menu) renderItemsWithLayout() {
	if len(m.filtered) == 0 {
		m.formatter.Warning("No items found")
		return
	}

	displayCount := m.config.MaxItems
	if len(m.filtered) < displayCount {
		displayCount = len(m.filtered)
	}

	switch m.config.Layout {
	case LayoutDefault:
		m.renderDefaultLayout(displayCount)
	case LayoutCompact:
		m.renderCompactLayout(displayCount)
	case LayoutTable:
		m.renderTableLayout(displayCount)
	case LayoutCards:
		m.renderCardsLayout(displayCount)
	case LayoutTree:
		m.renderTreeLayout(displayCount)
	default:
		m.renderDefaultLayout(displayCount)
	}
}

// renderDefaultLayout renders the traditional vertical list
func (m *Menu) renderDefaultLayout(displayCount int) {
	for i := range displayCount {
		item := m.filtered[i]
		m.renderMenuItem(item, i)

		// Show description if enabled and item is selected
		if m.config.ShowDescription && i == m.selected && item.Description != "" {
			desc := "    " + item.Description
			m.formatter.Info("%s", desc)
		}
	}
}

// renderCompactLayout renders items with minimal spacing
func (m *Menu) renderCompactLayout(displayCount int) {
	for i := range displayCount {
		item := m.filtered[i]
		// Compact: no descriptions, minimal markers
		marker := ""
		if i == m.selected {
			marker = "▶ "
		}

		prefix := ""
		if m.config.AllowMultiSelect {
			if m.selectedItems[i] {
				prefix = "[✓]"
			} else {
				prefix = "[ ]"
			}
		}

		line := fmt.Sprintf("%s%s%s", marker, prefix, item.Text)

		if item.Disabled {
			m.formatter.Warning("%s", line)
		} else if i == m.selected {
			m.formatter.Success("%s", line)
		} else {
			fmt.Fprintln(m.output, line)
		}
	}
}

// renderTableLayout renders items in a tabular format
func (m *Menu) renderTableLayout(displayCount int) {
	// Create table
	table := m.formatter.Table()
	headers := []string{"#", "Item", "Description"}
	if m.config.AllowMultiSelect {
		headers = []string{"✓", "#", "Item", "Description"}
	}
	table.Headers(headers...)

	for i := range displayCount {
		item := m.filtered[i]

		var row []string
		if m.config.AllowMultiSelect {
			check := ""
			if m.selectedItems[i] {
				check = "✓"
			}
			row = []string{check, fmt.Sprintf("%d", i+1), item.Text, item.Description}
		} else {
			row = []string{fmt.Sprintf("%d", i+1), item.Text, item.Description}
		}

		// Highlight selected row
		if i == m.selected {
			for j := range row {
				row[j] = "→ " + row[j]
			}
		}

		table.Row(row...)
	}

	table.Print()
}

// renderCardsLayout renders items as cards with borders
func (m *Menu) renderCardsLayout(displayCount int) {
	for i := range displayCount {
		item := m.filtered[i]

		// Card border
		border := "┌─────────────────────────────────────────┐"
		if i == m.selected {
			border = "┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓"
		}

		fmt.Fprintln(m.output, border)

		// Content
		title := item.Text
		if m.config.AllowMultiSelect && m.selectedItems[i] {
			title = "✓ " + title
		}

		if i == m.selected {
			m.formatter.Success("┃ %-37s ┃", title)
		} else {
			fmt.Fprintf(m.output, "│ %-37s │\n", title)
		}

		if item.Description != "" {
			desc := item.Description
			if len(desc) > 35 {
				desc = desc[:32] + "..."
			}
			fmt.Fprintf(m.output, "│ %-37s │\n", desc)
		}

		// Bottom border
		if i == m.selected {
			fmt.Fprintln(m.output, "┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛")
		} else {
			fmt.Fprintln(m.output, "└─────────────────────────────────────────┘")
		}
	}
}

// renderTreeLayout renders items in a tree/hierarchical format
func (m *Menu) renderTreeLayout(displayCount int) {
	for i := range displayCount {
		item := m.filtered[i]

		// Tree structure
		var prefix string
		if i == displayCount-1 {
			prefix = "└── "
		} else {
			prefix = "├── "
		}

		// Selection indicator
		if i == m.selected {
			prefix = "┣━━ "
		}

		// Multi-select
		if m.config.AllowMultiSelect && m.selectedItems[i] {
			prefix += "[✓] "
		}

		line := prefix + item.Text

		if item.Disabled {
			m.formatter.Warning("%s", line)
		} else if i == m.selected {
			m.formatter.Success("%s", line)
		} else {
			fmt.Fprintln(m.output, line)
		}

		// Show description indented
		if item.Description != "" && (m.config.ShowDescription || i == m.selected) {
			descPrefix := "│   "
			if i == displayCount-1 {
				descPrefix = "    "
			}
			m.formatter.Info("%s%s", descPrefix, item.Description)
		}
	}
}

// renderMenuItem renders a single menu item (used by default layout)
func (m *Menu) renderMenuItem(item MenuItem, index int) {
	// Format item with selection and multi-select indicators
	var marker, prefix string

	// Multi-selection indicator
	if m.config.AllowMultiSelect {
		if m.selectedItems[index] {
			prefix = "[✓] "
		} else {
			prefix = "[ ] "
		}
	}

	// Current item indicator
	if index == m.selected {
		if m.config.VimMode {
			marker = "❯ "
		} else {
			marker = "→ "
		}
	} else {
		marker = "  "
	}

	line := fmt.Sprintf("%s%s%s", marker, prefix, item.Text)

	if item.Disabled {
		// Muted formatting for disabled items
		m.formatter.Warning("%s", line)
	} else if index == m.selected {
		// Highlight selected item
		m.formatter.Success("%s", line)
	} else {
		// Regular item
		fmt.Fprintln(m.output, line)
	}
}

// SetInput sets the input reader for testing
func (m *Menu) SetInput(r io.Reader) {
	m.input = r
}

// SetOutput sets the output writer for testing
func (m *Menu) SetOutput(w io.Writer) {
	m.output = w
	m.formatter = output.NewFormatter(w)
}

// Show displays the menu and handles user interaction
func (m *Menu) Show() (*MenuItem, error) {
	// Set up terminal for raw input (simplified for demo)
	// In a real implementation, we'd use a proper terminal library

	for {
		m.render()

		// Get user input (simplified - in reality we'd handle raw terminal input)
		key, err := m.getKey()
		if err != nil {
			return nil, err
		}

		handled := m.handleKeyEvent(key)
		if !handled {
			// Key wasn't handled, might be an exit condition
			continue
		}

		// Check if we should return a result
		if key.Code == KeyEnter || key.Code == KeyVimEnter {
			if len(m.filtered) > 0 && m.selected < len(m.filtered) {
				if m.config.AllowMultiSelect && len(m.selectedItems) > 0 {
					// Return multiple selected items
					var selected []MenuItem
					for i := range m.selectedItems {
						if i < len(m.filtered) {
							selected = append(selected, m.filtered[i])
						}
					}
					// For now, return first selected item
					if len(selected) > 0 {
						return &selected[0], nil
					}
				} else {
					// Single selection
					item := &m.filtered[m.selected]
					return item, nil
				}
			}
		}

		if key.Code == KeyEscape || key.Code == KeyVimQuit {
			return nil, nil // User cancelled
		}
	}
}

// render displays the current menu state
func (m *Menu) render() {
	// Clear screen (simplified)
	fmt.Fprint(m.output, "\033[2J\033[H")

	// Show title
	m.formatter.Header(m.config.Title)

	// Show search query if applicable
	if m.config.AllowSearch {
		if m.query != "" {
			m.formatter.Info("Search: %s", m.query)
		} else {
			m.formatter.Info("Type to search...")
		}
		fmt.Fprintln(m.output)
	}

	// Render items using the configured layout
	m.renderItemsWithLayout()

	// Show preview if available
	if m.config.PreviewFunc != nil && len(m.filtered) > 0 && m.selected < len(m.filtered) {
		fmt.Fprintln(m.output)
		m.formatter.Subheader("Preview")
		preview := m.config.PreviewFunc(m.filtered[m.selected])
		fmt.Fprintln(m.output, preview)
	}

	// Show help section
	fmt.Fprintln(m.output)
	if m.helpVisible {
		m.renderHelp()
	} else {
		m.renderStatusLine()
	}
}

// moveSelection moves the selection up or down
func (m *Menu) moveSelection(delta int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.filtered) == 0 {
		return
	}

	m.selected += delta

	if m.selected < 0 {
		m.selected = len(m.filtered) - 1
	}
	if m.selected >= len(m.filtered) {
		m.selected = 0
	}

	// Skip disabled items
	startSelected := m.selected
	for m.filtered[m.selected].Disabled {
		m.selected += delta
		if m.selected < 0 {
			m.selected = len(m.filtered) - 1
		}
		if m.selected >= len(m.filtered) {
			m.selected = 0
		}

		// Prevent infinite loop if all items are disabled
		if m.selected == startSelected {
			break
		}
	}
}

// updateFilter updates the filtered items based on search query with performance optimizations
func (m *Menu) updateFilter() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.query == "" {
		m.filtered = m.items
	} else {
		// Use SearchWithLimit for better performance with large lists
		limit := m.config.MaxItems * 2 // Buffer for better matches
		if limit <= 0 || limit > len(m.items) {
			limit = len(m.items)
		}

		// Convert items to searchable format
		texts := make([]string, len(m.items))
		for i, item := range m.items {
			texts[i] = item.Text
		}

		// Perform optimized fuzzy search with limit
		matches := m.fuzzy.SearchWithLimit(m.query, texts, limit)

		// Convert matches back to menu items with indexed lookup
		itemMap := make(map[string]MenuItem, len(m.items))
		for _, item := range m.items {
			itemMap[item.Text] = item
		}

		m.filtered = make([]MenuItem, 0, len(matches))
		for _, match := range matches {
			if item, exists := itemMap[match.Text]; exists {
				m.filtered = append(m.filtered, item)
			}
		}
	}

	// Reset selection to first non-disabled item
	m.selected = 0
	if len(m.filtered) > 0 {
		for i, item := range m.filtered {
			if !item.Disabled {
				m.selected = i
				break
			}
		}
	}
}

// getKey reads a key from input (simplified implementation)
func (m *Menu) getKey() (KeyEvent, error) {
	// This is a simplified implementation for demonstration
	// A real implementation would use proper terminal handling

	var input string
	fmt.Fprint(m.output, "\n> ")

	// Use bufio.Scanner for better input handling with timeout
	reader := bufio.NewScanner(m.input)

	// Create a channel to signal when input is ready
	inputChan := make(chan bool, 1)
	var scanResult bool

	go func() {
		scanResult = reader.Scan()
		inputChan <- true
	}()

	// Wait for input with a reasonable timeout (30 seconds)
	select {
	case <-inputChan:
		if !scanResult {
			err := reader.Err()
			if err != nil {
				fmt.Printf("[DEBUG] Scanner error: %v\n", err)
				return KeyEvent{}, err
			}
			return KeyEvent{}, fmt.Errorf("no input received")
		}
	case <-time.After(30 * time.Second):
		fmt.Printf("[DEBUG] Input timeout after 30 seconds\n")
		return KeyEvent{Code: KeyEscape}, nil // Treat timeout as escape
	}

	input = strings.TrimSpace(reader.Text())
	fmt.Printf("[DEBUG] Raw input: %q (len=%d, bytes=%v)\n", input, len(input), []byte(input))

	// Map simple commands and vim-style navigation
	switch input {
	// Exit commands
	case "q", "quit", "exit":
		fmt.Printf("[DEBUG] Exit command detected\n")
		return KeyEvent{Code: KeyVimQuit}, nil

	// Selection commands
	case "enter", "select", "":
		fmt.Printf("[DEBUG] Enter/select command detected\n")
		return KeyEvent{Code: KeyEnter}, nil

	// Navigation commands
	case "up", "u", "k":
		fmt.Printf("[DEBUG] Up command detected\n")
		if input == "k" {
			return KeyEvent{Code: KeyVimUp}, nil
		}
		return KeyEvent{Code: KeyUp}, nil
	case "down", "d", "j":
		fmt.Printf("[DEBUG] Down command detected\n")
		if input == "j" {
			return KeyEvent{Code: KeyVimDown}, nil
		}
		return KeyEvent{Code: KeyDown}, nil

	// Vim navigation
	case "h":
		return KeyEvent{Code: KeyVimLeft}, nil
	case "l":
		return KeyEvent{Code: KeyVimRight}, nil
	case "gg":
		return KeyEvent{Code: KeyVimFirst}, nil
	case "G":
		return KeyEvent{Code: KeyVimLast}, nil

	// Special functions
	case "clear", "backspace":
		fmt.Printf("[DEBUG] Clear/backspace command detected\n")
		return KeyEvent{Code: KeyBackspace}, nil
	case "/", "search":
		return KeyEvent{Code: KeyVimSearch}, nil
	case "?", "help":
		return KeyEvent{Code: KeyVimHelp}, nil
	case " ", "space":
		return KeyEvent{Code: KeyVimSelect}, nil

	// Home/End
	case "home", "first":
		return KeyEvent{Code: KeyHome}, nil
	case "end", "last":
		return KeyEvent{Code: KeyEnd}, nil

	// Page navigation
	case "pgup", "pageup":
		return KeyEvent{Code: KeyPageUp}, nil
	case "pgdn", "pagedown":
		return KeyEvent{Code: KeyPageDown}, nil

	// Accessibility
	case "speak":
		return KeyEvent{Code: KeySpeak}, nil
	case "describe":
		return KeyEvent{Code: KeyDescribe}, nil
	case "accessible", "accessibility":
		return KeyEvent{Code: KeyAccessibilityMode}, nil

	default:
		// Treat as search input
		if len(input) > 0 {
			char := rune(input[0])
			fmt.Printf("[DEBUG] Character input: %c (code=%d)\n", char, int32(char))
			return KeyEvent{Code: KeyChar, Char: char}, nil
		}
		fmt.Printf("[DEBUG] Empty input, treating as enter\n")
		return KeyEvent{Code: KeyEnter}, nil
	}
}

// MultiSelect allows multiple item selection
type MultiSelect struct {
	menu     *Menu
	selected map[int]bool
}

// Show displays the multi-select menu
func (ms *MultiSelect) Show() ([]MenuItem, error) {
	// Implementation would be similar to Menu.Show() but with space to toggle selection
	// For now, return single selection
	item, err := ms.menu.Show()
	if err != nil || item == nil {
		return nil, err
	}
	return []MenuItem{*item}, nil
}
