// Demo showcases the advanced interactive UI capabilities of taskopen
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/johnconnor-sec/taskopen-go/internal/output"
	"github.com/johnconnor-sec/taskopen-go/internal/search"
	"github.com/johnconnor-sec/taskopen-go/internal/ui"
)

func main() {
	fmt.Println("🚀 Taskopen EPOCH 2 Sprint 5 - Interactive Menu System Demo")
	fmt.Println("This demo showcases the world-class TUI we just built!")
	fmt.Println()

	// Create formatter for beautiful output
	formatter := output.NewFormatter(os.Stdout)

	// Demo the fuzzy search engine
	demoFuzzySearch(formatter)

	// Demo the interactive menu system
	demoInteractiveMenu(formatter)

	// Demo the preview system
	demoPreviewSystem(formatter)

	// Demo accessibility features
	demoAccessibilityFeatures(formatter)

	// Summary
	formatter.Header("🎉 Sprint 5 Complete - Interactive Excellence!")
	formatter.Success("✅ Advanced TUI with real terminal handling")
	formatter.Success("✅ Blazing-fast fuzzy search with smart ranking")
	formatter.Success("✅ Safe command preview with risk assessment")
	formatter.Success("✅ Vim-style navigation and keyboard shortcuts")
	formatter.Success("✅ Multi-selection for batch operations")
	formatter.Success("✅ Contextual help system")
	formatter.Success("✅ Full accessibility support")

	fmt.Println()
	formatter.Info("Ready for EPOCH 2 Sprint 6: Action Engine & Plugins!")
}

func demoFuzzySearch(formatter *output.Formatter) {
	formatter.Subheader("Demo 1: Fuzzy Search Engine")

	fuzzy := search.NewFuzzy()

	// Sample actions
	actions := []string{
		"edit file with vim",
		"open browser firefox",
		"view log files",
		"edit configuration",
		"browse directory",
		"view task details",
		"edit task annotations",
	}

	// Test various search queries
	queries := []string{"edit", "view", "firefox", "task"}

	for _, query := range queries {
		matches := fuzzy.Search(query, actions)

		formatter.Info("Search: '%s'", query)
		for i, match := range matches {
			if i >= 3 { // Show top 3 results
				break
			}

			highlighted := fuzzy.HighlightString(match.Text, match.Highlights, "[", "]")
			fmt.Printf("  %.2f - %s\n", match.Score, highlighted)
		}
		fmt.Println()
	}
}

func demoInteractiveMenu(formatter *output.Formatter) {
	formatter.Subheader("Demo 2: Interactive Menu System")

	// Create sample menu items
	items := []ui.MenuItem{
		{
			ID:          "edit",
			Text:        "Edit file",
			Description: "Open file in default editor",
			Action:      func() error { fmt.Println("Would edit file"); return nil },
		},
		{
			ID:          "browse",
			Text:        "Open browser",
			Description: "Launch web browser",
			Action:      func() error { fmt.Println("Would open browser"); return nil },
		},
		{
			ID:          "view",
			Text:        "View logs",
			Description: "Display log files",
			Action:      func() error { fmt.Println("Would view logs"); return nil },
		},
		{
			ID:          "config",
			Text:        "Edit configuration",
			Description: "Modify taskopen settings",
			Disabled:    true, // Example of disabled item
		},
	}

	// Show simple menu (non-interactive for demo)
	formatter.Info("Available actions:")

	for i, item := range items {
		status := "✓"
		if item.Disabled {
			status = "✗"
		}

		fmt.Printf("  %s %d. %s - %s\n", status, i+1, item.Text, item.Description)
	}

	fmt.Println()
	formatter.Info("In interactive mode, you would:")
	formatter.List("Use ↑/↓ to navigate")
	formatter.List("Type to search/filter items")
	formatter.List("Press Enter to select")
	formatter.List("Press Esc to cancel")
}

func demoPreviewSystem(formatter *output.Formatter) {
	formatter.Header("Safe Command Preview Demo")

	// Demo the preview system
	preview := ui.NewSimplePreview()

	// Sample commands with different risk levels
	commands := []struct {
		cmd  string
		desc string
		vars map[string]string
	}{
		{
			cmd:  "xdg-open ~/Documents/notes.txt",
			desc: "Open notes file",
			vars: map[string]string{"FILE": "~/Documents/notes.txt"},
		},
		{
			cmd:  "firefox https://github.com/user/repo",
			desc: "Open project repository",
			vars: map[string]string{"URL": "https://github.com/user/repo"},
		},
		{
			cmd:  "sudo systemctl restart nginx",
			desc: "Restart web server",
			vars: map[string]string{"SERVICE": "nginx"},
		},
	}

	for i, cmd := range commands {
		formatter.Subheader(fmt.Sprintf("Preview %d: %s", i+1, cmd.desc))

		previewInfo := preview.PreviewCommand(cmd.cmd, cmd.desc, cmd.vars)

		// Show key preview information
		fmt.Printf("Command: %s\n", previewInfo.Command)
		fmt.Printf("Risk: %s\n", previewInfo.RiskLevel)

		if len(previewInfo.Safety) > 0 {
			fmt.Println("Safety checks:")
			for _, check := range previewInfo.Safety {
				if strings.Contains(check, "No obvious risks") {
					formatter.Success("  ✓ %s", check)
				} else {
					formatter.Warning("  ⚠ %s", check)
				}
			}
		}

		if previewInfo.FileInfo != nil && previewInfo.FileInfo.Exists {
			formatter.Info("  📄 File: %s (%d bytes)",
				previewInfo.FileInfo.Path, previewInfo.FileInfo.Size)
		}

		fmt.Println()
	}
}

func demoAccessibilityFeatures(formatter *output.Formatter) {
	formatter.Header("Accessibility Features Demo")

	formatter.Success("Screen Reader Support:")
	formatter.List("Semantic role announcements (Success, Error, Warning, etc.)")
	formatter.List("Structured navigation with clear headings")
	formatter.List("Alternative text for visual elements")
	formatter.List("Keyboard-only navigation support")

	formatter.Success("Visual Accessibility:")
	formatter.List("High contrast color themes")
	formatter.List("Customizable color schemes")
	formatter.List("NO_COLOR environment variable support")
	formatter.List("TASKOPEN_ACCESSIBILITY=screen-reader mode")

	formatter.Success("Motor Accessibility:")
	formatter.List("Multiple navigation options (arrows, vim keys, etc.)")
	formatter.List("Configurable key bindings")
	formatter.List("No time-pressure interactions")
	formatter.List("Large click targets in TUI")

	// Demo different accessibility modes
	fmt.Println()
	formatter.Info("Demo: Different accessibility modes")

	// Normal mode
	formatter.Success("Normal mode: ✓ Full colors and formatting")

	// High contrast mode simulation
	fmt.Println("\033[1;97mHigh contrast mode: Enhanced visibility\033[0m")

	// Screen reader mode simulation
	fmt.Println("Screen reader mode: Success: Clean text output")

	fmt.Println()
}
