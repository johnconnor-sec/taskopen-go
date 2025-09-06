package main

import (
	"fmt"
	"os"
	"time"

	"github.com/johnconnor-sec/taskopen-go/internal/output"
)

func main() {
	fmt.Println("🚀 Taskopen Output System Demo - Sprint 4 Enhancements")

	// Create formatter with enhanced features
	formatter := output.NewFormatter(os.Stdout)

	// Demo 1: Enhanced Terminal Detection
	formatter.Header("Terminal Capabilities Detection")
	width := formatter.GetCurrentWidth()
	formatter.Info("Terminal width: %d columns", width)
	formatter.Info("Color support: %t", true) // Would be dynamic
	formatter.Info("Accessibility mode: Normal")
	fmt.Println()

	// Demo 2: Enhanced Color and Theming
	formatter.Header("Enhanced Color System")
	formatter.SetTheme(output.DarkTheme)
	formatter.Success("Dark theme activated")
	formatter.Warning("This is a warning message")
	formatter.Error("This is an error message")
	formatter.Debug("Debug information (level dependent)")
	fmt.Println()

	// Demo 3: Accessibility Features
	formatter.Header("Accessibility Features")
	formatter.SetAccessibilityMode(output.AccessibilityScreenReader)
	formatter.ScreenReaderText("info", "Screen reader optimized output")
	formatter.SetAccessibilityMode(output.AccessibilityHighContrast)
	formatter.ScreenReaderText("success", "High contrast mode active")
	formatter.SetAccessibilityMode(output.AccessibilityNormal)
	fmt.Println()

	// Demo 4: Output Templates
	formatter.Header("Customizable Output Templates")

	// Sample task data
	tasks := []map[string]interface{}{
		{
			"id":          1,
			"priority":    "H",
			"project":     "taskopen",
			"description": "Implement output system enhancements",
			"due":         "2025-09-05",
		},
		{
			"id":          2,
			"priority":    "M",
			"project":     "taskopen",
			"description": "Add accessibility features",
		},
		{
			"id":          3,
			"priority":    "L",
			"project":     "docs",
			"description": "Write user documentation",
		},
	}

	formatter.Subheader("Default Template")
	formatter.RenderTaskListWithTemplate(tasks, output.DefaultTaskTemplate)

	formatter.Subheader("Compact Template")
	formatter.RenderTaskListWithTemplate(tasks, output.CompactTaskTemplate)
	fmt.Println()

	// Demo 5: Enhanced Progress Indicators
	formatter.Header("Advanced Progress Indicators")

	// Multi-progress demo
	mp := formatter.NewMultiProgress()
	mp.AddProgress("parse", "Parsing configuration", 100)
	mp.AddProgress("scan", "Scanning tasks", 250)
	mp.AddProgress("process", "Processing results", 50)

	// Simulate progress
	for i := 0; i <= 100; i += 20 {
		mp.Update("parse", i, fmt.Sprintf("Parsing... %d%%", i))
		mp.Update("scan", i*2, fmt.Sprintf("Scanning... %d items", i*2))
		mp.Update("process", i/2, "Processing...")
		mp.Render()
		time.Sleep(200 * time.Millisecond)
	}

	mp.Finish()
	fmt.Println()

	// Demo 6: Status Line
	formatter.Header("Dynamic Status Line")
	statusLine := formatter.NewStatusLine()

	statusLine.Update("Initializing taskopen...")
	time.Sleep(500 * time.Millisecond)

	statusLine.Update("Loading configuration...")
	time.Sleep(500 * time.Millisecond)

	statusLine.Update("Connecting to taskwarrior...")
	time.Sleep(500 * time.Millisecond)

	statusLine.Success("Taskopen ready!")
	fmt.Println()

	// Demo 7: Enhanced Logger Integration
	formatter.Header("Enhanced Structured Logging")
	logger := output.NewLogger().SetFormatter(formatter)

	logger.Info("Application started")
	logger.WithField("component", "config").Debug("Configuration loaded")
	logger.WithFields(map[string]interface{}{
		"operation": "task_scan",
		"count":     15,
		"duration":  "1.2s",
	}).Info("Task scanning completed")
	logger.LogDuration("template_render", 250*time.Millisecond)
	fmt.Println()

	// Demo 8: System Diagnostics
	formatter.Header("System Diagnostics Display")
	diagnostics := []output.DiagnosticInfo{
		{
			Component: "Taskwarrior",
			Status:    "✓ Ready",
			Details: map[string]interface{}{
				"version": "2.6.0",
				"tasks":   "127",
			},
		},
		{
			Component: "Configuration",
			Status:    "✓ Functional",
			Details: map[string]interface{}{
				"file":    "~/.taskopen.yml",
				"actions": "12",
			},
		},
		{
			Component: "Terminal",
			Status:    "⚠ Warning",
			Details: map[string]interface{}{
				"width":  width,
				"colors": "supported",
			},
			Suggestions: []string{
				"Consider enabling high contrast mode for better visibility",
				"Terminal width is narrow - compact mode recommended",
			},
		},
	}

	formatter.RenderDiagnostics(diagnostics)

	// Summary
	formatter.Header("Sprint 4 Output System - Complete! ✅")
	formatter.List("Enhanced terminal detection and dynamic width")
	formatter.List("Robust color support with accessibility modes")
	formatter.List("Customizable output templates")
	formatter.List("Advanced progress indicators and status lines")
	formatter.List("Integrated structured logging")
	formatter.List("Comprehensive system diagnostics")
	formatter.List("Full backward compatibility maintained")

	formatter.Success("Ready for Sprint 5: Interactive Menu System!")
}
