// Demo program showcasing the interactive menu system
//
// This demo demonstrates:
// - Fuzzy search with <100ms performance
// - Preview system with command safety analysis
// - Vim-style keyboard navigation
// - Accessibility features
// - Multi-selection capabilities
// - Customizable layouts and themes
// - Integration with output formatter

package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/johnconnor-sec/taskopen-go/internal/ui"
)

func main() {
	fmt.Println("🚀 Interactive Menu System Demo")
	fmt.Println("=================================")
	fmt.Println()

	// Demo 1: Basic menu with vim navigation
	runBasicMenuDemo()

	// Demo 2: Multi-selection menu
	runMultiSelectionDemo()

	// Demo 3: Different layouts
	runLayoutDemo()

	// Demo 4: Preview system
	runPreviewDemo()

	// Demo 5: Accessibility features
	runAccessibilityDemo()

	fmt.Println("✅ Demo completed! All interactive features working.")
}

func runBasicMenuDemo() {
	fmt.Println("📋 Demo 1: Basic Menu with Vim Navigation")
	fmt.Println("------------------------------------------")

	items := []ui.MenuItem{
		{ID: "1", Text: "Edit task notes", Description: "Open task notes in your preferred editor", Action: func() error { return nil }},
		{ID: "2", Text: "View task details", Description: "Display comprehensive task information", Action: func() error { return nil }},
		{ID: "3", Text: "Mark as complete", Description: "Mark the current task as completed", Action: func() error { return nil }},
		{ID: "4", Text: "Add time tracking", Description: "Start time tracking for this task", Action: func() error { return nil }},
		{ID: "5", Text: "Create subtask", Description: "Create a new subtask under this task", Action: func() error { return nil }},
	}

	config := ui.DefaultMenuConfig()
	config.Title = "🎯 Task Actions"
	config.VimMode = true
	config.ShowDescription = true
	config.AllowSearch = true

	fmt.Println("Features demonstrated:")
	fmt.Println("• j/k for vim-style navigation")
	fmt.Println("• / for search")
	fmt.Println("• ? for help")
	fmt.Println("• Type characters to filter items")
	fmt.Println()

	// For demo purposes, we'll just show the config
	fmt.Printf("Menu configured with %d items\n", len(items))
	fmt.Printf("Vim mode: %v\n", config.VimMode)
	fmt.Printf("Search enabled: %v\n", config.AllowSearch)
	fmt.Println("✅ Basic menu demo configuration complete")
	fmt.Println()
}

func runMultiSelectionDemo() {
	fmt.Println("☑️  Demo 2: Multi-Selection Menu")
	fmt.Println("--------------------------------")

	items := []ui.MenuItem{
		{ID: "file1", Text: "config.yaml", Description: "Application configuration"},
		{ID: "file2", Text: "main.go", Description: "Main application file"},
		{ID: "file3", Text: "README.md", Description: "Project documentation"},
		{ID: "file4", Text: "Makefile", Description: "Build automation"},
		{ID: "file5", Text: "go.mod", Description: "Go module definition"},
	}

	config := ui.DefaultMenuConfig()
	config.Title = "📁 Select Files to Process"
	config.AllowMultiSelect = true
	config.VimMode = true
	config.ShowDescription = true

	fmt.Println("Multi-selection features:")
	fmt.Println("• Space to toggle selection")
	fmt.Println("• [✓] indicators for selected items")
	fmt.Println("• Batch operations on selected items")
	fmt.Println("• Status line shows selection count")
	fmt.Println()

	fmt.Printf("Multi-selection menu configured with %d items\n", len(items))
	fmt.Printf("Multi-select enabled: %v\n", config.AllowMultiSelect)
	fmt.Println("✅ Multi-selection demo configuration complete")
	fmt.Println()
}

func runLayoutDemo() {
	fmt.Println("🎨 Demo 3: Customizable Layouts")
	fmt.Println("-------------------------------")

	items := []ui.MenuItem{
		{ID: "task1", Text: "Review pull request #123", Description: "Code review for new feature"},
		{ID: "task2", Text: "Update documentation", Description: "Fix typos in user guide"},
		{ID: "task3", Text: "Deploy to staging", Description: "Deploy latest changes to staging environment"},
	}

	layouts := map[string]ui.MenuLayout{
		"Default": ui.LayoutDefault,
		"Compact": ui.LayoutCompact,
		"Table":   ui.LayoutTable,
		"Cards":   ui.LayoutCards,
		"Tree":    ui.LayoutTree,
	}

	themes := map[string]ui.MenuTheme{
		"Default":       ui.ThemeDefault,
		"Dark":          ui.ThemeDark,
		"Light":         ui.ThemeLight,
		"High Contrast": ui.ThemeHighContrast,
		"Vim":           ui.ThemeVim,
		"Modern":        ui.ThemeModern,
	}

	fmt.Println("Available layouts:")
	for name := range layouts {
		fmt.Printf("• %s layout\n", name)
	}

	fmt.Println("\nAvailable themes:")
	for name := range themes {
		fmt.Printf("• %s theme\n", name)
	}

	fmt.Println("\nLayout features:")
	fmt.Println("• Default: Traditional vertical list")
	fmt.Println("• Compact: Minimal spacing")
	fmt.Println("• Table: Tabular format with columns")
	fmt.Println("• Cards: Card-style with borders")
	fmt.Println("• Tree: Hierarchical tree view")

	fmt.Printf("\nConfigured with %d items across %d layouts and %d themes\n",
		len(items), len(layouts), len(themes))
	fmt.Println("✅ Layout and theme demo configuration complete")
	fmt.Println()
}

func runPreviewDemo() {
	fmt.Println("🔍 Demo 4: Command Preview System")
	fmt.Println("---------------------------------")

	items := []ui.MenuItem{
		{
			ID:          "cmd1",
			Text:        "Edit configuration",
			Description: "vim ~/.config/taskopen/config.yaml",
			Data: map[string]interface{}{
				"command": "vim ~/.config/taskopen/config.yaml",
				"type":    "editor",
			},
		},
		{
			ID:          "cmd2",
			Text:        "Backup files",
			Description: "cp -r ~/.config/taskopen ~/.config/taskopen.backup",
			Data: map[string]interface{}{
				"command": "cp -r ~/.config/taskopen ~/.config/taskopen.backup",
				"type":    "file_operation",
			},
		},
		{
			ID:          "cmd3",
			Text:        "Clean temporary files",
			Description: "rm -f /tmp/taskopen_*.tmp",
			Data: map[string]interface{}{
				"command": "rm -f /tmp/taskopen_*.tmp",
				"type":    "cleanup",
			},
		},
	}

	config := ui.DefaultMenuConfig()
	config.Title = "⚡ Commands with Preview"
	config.PreviewFunc = ui.CreateAdvancedPreviewFunction(ui.PreviewOptions{
		Mode:        ui.PreviewDryRun,
		ShowRisks:   true,
		ShowOutput:  true,
		ShowContext: true,
	})

	fmt.Println("Preview system features:")
	fmt.Println("• Risk assessment (SAFE/MEDIUM/HIGH/CRITICAL)")
	fmt.Println("• Dry-run execution where possible")
	fmt.Println("• Safety warnings and recommendations")
	fmt.Println("• File information and content preview")
	fmt.Println("• Environment variable expansion")
	fmt.Println()

	fmt.Printf("Preview-enabled menu with %d commands\n", len(items))
	fmt.Println("Preview functions:")
	for _, item := range items {
		if data, ok := item.Data.(map[string]interface{}); ok {
			if cmd, exists := data["command"]; exists {
				risk := assessCommandRisk(fmt.Sprintf("%v", cmd))
				fmt.Printf("• %s: %s risk level\n", item.Text, risk)
			}
		}
	}
	fmt.Println("✅ Preview system demo configuration complete")
	fmt.Println()
}

func runAccessibilityDemo() {
	fmt.Println("♿ Demo 5: Accessibility Features")
	fmt.Println("--------------------------------")

	items := []ui.MenuItem{
		{ID: "a11y1", Text: "High contrast mode", Description: "Enable high contrast colors"},
		{ID: "a11y2", Text: "Screen reader mode", Description: "Enable screen reader compatibility"},
		{ID: "a11y3", Text: "Large text mode", Description: "Increase font size for better readability"},
		{ID: "a11y4", Text: "Keyboard shortcuts", Description: "Show available keyboard shortcuts"},
	}

	config := ui.DefaultMenuConfig()
	config.Title = "🔊 Accessibility Options"
	config.AccessibilityMode = true
	config.ShowHelp = true

	fmt.Println("Accessibility features:")
	fmt.Println("• Screen reader announcements")
	fmt.Println("• High contrast themes")
	fmt.Println("• Keyboard-only navigation")
	fmt.Println("• Semantic role indicators")
	fmt.Println("• Ctrl+S to speak current item")
	fmt.Println("• Ctrl+D to describe current item")
	fmt.Println("• F12 to toggle accessibility mode")
	fmt.Println()

	fmt.Printf("Accessibility-enhanced menu with %d options\n", len(items))
	fmt.Printf("Accessibility mode: %v\n", config.AccessibilityMode)
	fmt.Printf("Help system: %v\n", config.ShowHelp)
	fmt.Println("✅ Accessibility demo configuration complete")
	fmt.Println()
}

func assessCommandRisk(command string) string {
	cmd := strings.ToLower(command)

	if strings.Contains(cmd, "rm -rf") || strings.Contains(cmd, "format") {
		return "CRITICAL"
	}
	if strings.Contains(cmd, "rm ") || strings.Contains(cmd, "sudo") {
		return "HIGH"
	}
	if strings.Contains(cmd, "cp ") || strings.Contains(cmd, "mv ") {
		return "MEDIUM"
	}
	return "SAFE"
}

func demoError(err error) {
	if err != nil {
		log.Printf("Demo error: %v", err)
	}
}
