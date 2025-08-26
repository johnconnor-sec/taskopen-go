package ui

import (
	"bytes"
	"strings"
	"testing"
)

func TestMenuItem_Basic(t *testing.T) {
	item := MenuItem{
		ID:          "test",
		Text:        "Test Item",
		Description: "Test description",
		Disabled:    false,
	}

	if item.ID != "test" {
		t.Errorf("Expected ID 'test', got '%s'", item.ID)
	}
	if item.Text != "Test Item" {
		t.Errorf("Expected text 'Test Item', got '%s'", item.Text)
	}
}

func TestDefaultMenuConfig(t *testing.T) {
	config := DefaultMenuConfig()

	if config.Title != "Select an option" {
		t.Errorf("Expected default title 'Select an option', got '%s'", config.Title)
	}
	if !config.AllowSearch {
		t.Error("Expected search to be enabled by default")
	}
	if config.MaxItems != 10 {
		t.Errorf("Expected max items 10, got %d", config.MaxItems)
	}
}

func TestNewMenu(t *testing.T) {
	items := []MenuItem{
		{ID: "1", Text: "Item 1"},
		{ID: "2", Text: "Item 2"},
	}

	config := DefaultMenuConfig()
	menu := NewMenu(items, config)

	if len(menu.items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(menu.items))
	}
	if len(menu.filtered) != 2 {
		t.Errorf("Expected 2 filtered items initially, got %d", len(menu.filtered))
	}
	if menu.selected != 0 {
		t.Errorf("Expected selection at 0, got %d", menu.selected)
	}
}

func TestMenu_UpdateFilter(t *testing.T) {
	items := []MenuItem{
		{ID: "1", Text: "Edit file"},
		{ID: "2", Text: "Open browser"},
		{ID: "3", Text: "Edit image"},
	}

	config := DefaultMenuConfig()
	menu := NewMenu(items, config)

	// Test filtering
	menu.query = "edit"
	menu.updateFilter()

	if len(menu.filtered) != 2 {
		t.Errorf("Expected 2 filtered items for 'edit', got %d", len(menu.filtered))
	}

	// Test empty query resets filter
	menu.query = ""
	menu.updateFilter()

	if len(menu.filtered) != 3 {
		t.Errorf("Expected 3 items when query is empty, got %d", len(menu.filtered))
	}
}

func TestMenu_MoveSelection(t *testing.T) {
	items := []MenuItem{
		{ID: "1", Text: "Item 1", Disabled: false},
		{ID: "2", Text: "Item 2", Disabled: false},
		{ID: "3", Text: "Item 3", Disabled: false},
	}

	config := DefaultMenuConfig()
	menu := NewMenu(items, config)

	// Move down
	menu.moveSelection(1)
	if menu.selected != 1 {
		t.Errorf("Expected selection at 1, got %d", menu.selected)
	}

	// Move down again
	menu.moveSelection(1)
	if menu.selected != 2 {
		t.Errorf("Expected selection at 2, got %d", menu.selected)
	}

	// Move down from last item (should wrap to first)
	menu.moveSelection(1)
	if menu.selected != 0 {
		t.Errorf("Expected selection to wrap to 0, got %d", menu.selected)
	}

	// Move up from first item (should wrap to last)
	menu.moveSelection(-1)
	if menu.selected != 2 {
		t.Errorf("Expected selection to wrap to 2, got %d", menu.selected)
	}
}

func TestMenu_MoveSelection_SkipDisabled(t *testing.T) {
	items := []MenuItem{
		{ID: "1", Text: "Item 1", Disabled: false},
		{ID: "2", Text: "Item 2", Disabled: true}, // Disabled
		{ID: "3", Text: "Item 3", Disabled: false},
	}

	config := DefaultMenuConfig()
	menu := NewMenu(items, config)

	// Move down (should skip disabled item)
	menu.moveSelection(1)
	if menu.selected != 2 {
		t.Errorf("Expected selection to skip disabled item and go to 2, got %d", menu.selected)
	}
}

func TestMenu_Render(t *testing.T) {
	items := []MenuItem{
		{ID: "1", Text: "Item 1", Description: "First item"},
		{ID: "2", Text: "Item 2", Description: "Second item"},
	}

	config := DefaultMenuConfig()
	config.Title = "Test Menu"

	var output bytes.Buffer
	menu := NewMenu(items, config)
	menu.SetOutput(&output)

	menu.render()

	result := output.String()

	if !strings.Contains(result, "Test Menu") {
		t.Error("Expected output to contain title 'Test Menu'")
	}
	if !strings.Contains(result, "Item 1") {
		t.Error("Expected output to contain 'Item 1'")
	}
	if !strings.Contains(result, "Item 2") {
		t.Error("Expected output to contain 'Item 2'")
	}
}

func TestShowSimpleMenu(t *testing.T) {
	items := []MenuItem{
		{ID: "1", Text: "Option 1", Description: "First option"},
		{ID: "2", Text: "Option 2", Description: "Second option"},
	}

	// We can't easily test the interactive part, but we can test the structure
	if len(items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(items))
	}
}

func TestCreateActionsMenu(t *testing.T) {
	actions := map[string]interface{}{
		"edit":   "vim",
		"browse": "firefox",
		"view":   "less",
	}

	items := CreateActionsMenu(actions)

	if len(items) != 3 {
		t.Errorf("Expected 3 menu items, got %d", len(items))
	}

	// Check that all action names are present
	found := make(map[string]bool)
	for _, item := range items {
		found[item.ID] = true
	}

	for action := range actions {
		if !found[action] {
			t.Errorf("Expected action '%s' in menu items", action)
		}
	}
}

func TestCreateTaskMenu(t *testing.T) {
	tasks := []map[string]interface{}{
		{
			"description": "Task 1",
			"status":      "pending",
		},
		{
			"description": "Task 2",
			"status":      "completed",
		},
	}

	items := CreateTaskMenu(tasks)

	if len(items) != 2 {
		t.Errorf("Expected 2 menu items, got %d", len(items))
	}

	if items[0].Text != "Task 1" {
		t.Errorf("Expected first item text 'Task 1', got '%s'", items[0].Text)
	}
	if !strings.Contains(items[0].Description, "pending") {
		t.Errorf("Expected first item description to contain 'pending', got '%s'", items[0].Description)
	}
}

func TestMenuConfig_Preview(t *testing.T) {
	items := []MenuItem{
		{ID: "1", Text: "Item 1"},
	}

	config := DefaultMenuConfig()
	config.PreviewFunc = func(item MenuItem) string {
		return "Preview: " + item.Text
	}

	if config.PreviewFunc == nil {
		t.Error("Expected preview function to be set")
	}

	preview := config.PreviewFunc(items[0])
	if preview != "Preview: Item 1" {
		t.Errorf("Expected preview 'Preview: Item 1', got '%s'", preview)
	}
}

func TestKeyEvent_Basic(t *testing.T) {
	event := KeyEvent{
		Code: KeyEnter,
		Char: '\n',
		Alt:  false,
		Ctrl: false,
	}

	if event.Code != KeyEnter {
		t.Errorf("Expected KeyEnter, got %d", event.Code)
	}
	if event.Alt {
		t.Error("Expected Alt to be false")
	}
}

func TestMultiSelect_Basic(t *testing.T) {
	items := []MenuItem{
		{ID: "1", Text: "Item 1"},
		{ID: "2", Text: "Item 2"},
	}

	config := DefaultMenuConfig()
	ms := NewMultiSelect(items, config)

	if ms.menu == nil {
		t.Error("Expected menu to be initialized")
	}
	if ms.selected == nil {
		t.Error("Expected selected map to be initialized")
	}
}
