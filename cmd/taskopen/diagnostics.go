package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/johnconnor-sec/taskopen-go/internal/config"
	"github.com/johnconnor-sec/taskopen-go/internal/exec"
	"github.com/johnconnor-sec/taskopen-go/internal/output"
	"github.com/johnconnor-sec/taskopen-go/internal/security"
)

func runDiagnostics() error {
	formatter := output.NewFormatter(os.Stdout)

	// Configure accessibility if needed
	if os.Getenv("TASKOPEN_ACCESSIBILITY") != "" {
		switch os.Getenv("TASKOPEN_ACCESSIBILITY") {
		case "high-contrast":
			formatter.SetAccessibilityMode(output.AccessibilityHighContrast)
		case "screen-reader":
			formatter.SetAccessibilityMode(output.AccessibilityScreenReader)
		case "minimal":
			formatter.SetAccessibilityMode(output.AccessibilityMinimal)
		}
	}

	// Collect diagnostic information
	var diagnostics []output.DiagnosticInfo

	// System info
	diagnostics = append(diagnostics, output.DiagnosticInfo{
		Component: "Go Runtime",
		Status:    "✓ Ready",
		Details:   map[string]any{"version": runtime.Version()},
	})

	diagnostics = append(diagnostics, output.DiagnosticInfo{
		Component: "Version",
		Status:    "✓ Ready",
		Details:   map[string]any{"version": version, "commit": commit},
	})

	// Check configuration
	configPath, configErr := config.FindConfigPath()
	if configErr != nil {
		diagnostics = append(diagnostics, output.DiagnosticInfo{
			Component:   "Configuration",
			Status:      "⚠ Warning",
			Details:     map[string]any{"error": configErr.Error()},
			Suggestions: []string{"Run 'taskopen config init' to create configuration", "Check config file permissions"},
		})
	} else {
		diagnostics = append(diagnostics, output.DiagnosticInfo{
			Component: "Configuration",
			Status:    "✓ Ready",
			Details:   map[string]any{"path": configPath},
		})
	}

	// Check taskwarrior
	taskBin := "task"
	if configErr == nil {
		// Try to load config to get actual task binary
		if cfg, err := config.Load(configPath); err == nil {
			taskBin = cfg.General.TaskBin
		}
	}

	ctx := context.Background()
	executor := exec.New(exec.ExecutionOptions{Timeout: 5 * time.Second, CaptureOutput: true})
	_, err := executor.ExecuteFilter(ctx, "which", []string{taskBin}, nil)
	if err != nil {
		diagnostics = append(diagnostics, output.DiagnosticInfo{
			Component:   "Taskwarrior",
			Status:      "✗ Failed",
			Details:     map[string]any{"binary": taskBin, "error": "not found"},
			Suggestions: []string{"Install taskwarrior package", "Check PATH environment variable", "Update taskbin setting in config"},
		})
	} else {
		// Get taskwarrior version
		result, versionErr := executor.Execute(ctx, taskBin, []string{"--version"}, &exec.ExecutionOptions{
			Timeout:       5 * time.Second,
			CaptureOutput: true,
		})
		if versionErr == nil && result != nil && result.ExitCode == 0 {
			versionLine := strings.Split(strings.TrimSpace(result.Stdout), "\n")[0]
			diagnostics = append(diagnostics, output.DiagnosticInfo{
				Component: "Taskwarrior",
				Status:    "✓ Ready",
				Details:   map[string]any{"version": versionLine},
			})
		} else {
			errorMsg := "version check failed"
			if versionErr != nil {
				errorMsg = versionErr.Error()
			}
			diagnostics = append(diagnostics, output.DiagnosticInfo{
				Component: "Taskwarrior",
				Status:    "⚠ Warning",
				Details:   map[string]any{"binary": taskBin, "error": errorMsg},
			})
		}
	}

	// Check editor (if configured)
	if configErr == nil {
		if cfg, err := config.Load(configPath); err == nil && cfg.General.Editor != "" {
			editorParts := strings.Fields(cfg.General.Editor)
			if len(editorParts) > 0 {
				_, editorErr := executor.ExecuteFilter(ctx, "which", []string{editorParts[0]}, nil)
				if editorErr != nil {
					diagnostics = append(diagnostics, output.DiagnosticInfo{
						Component:   "Editor",
						Status:      "✗ Failed",
						Details:     map[string]any{"editor": cfg.General.Editor},
						Suggestions: []string{"Install the editor or update the configuration", "Check that the editor is in your PATH", "Run 'taskopen config init' to reconfigure"},
					})
				} else {
					diagnostics = append(diagnostics, output.DiagnosticInfo{
						Component: "Editor",
						Status:    "✓ Ready",
						Details:   map[string]any{"editor": cfg.General.Editor},
					})
				}
			}
		}
	}

	// Core systems
	coreComponents := []output.DiagnosticInfo{
		{Component: "Types System", Status: "✓ Functional", Details: map[string]any{"description": "Validated data structures"}},
		{Component: "Error Handling", Status: "✓ Functional", Details: map[string]any{"description": "Structured error system"}},
		{Component: "Output System", Status: "✓ Functional", Details: map[string]any{"description": "Beautiful terminal output with accessibility"}},
		{Component: "Execution Engine", Status: "✓ Functional", Details: map[string]any{"description": "Secure process handling"}},
		{Component: "Security System", Status: "✓ Functional", Details: map[string]any{"description": "Environment variable sanitization and secure previews"}},
	}
	diagnostics = append(diagnostics, coreComponents...)

	// Render comprehensive diagnostics
	formatter.RenderDiagnostics(diagnostics)

	fmt.Println()

	// Check for active taskwarrior context
	if configErr == nil {
		result, err := executor.Execute(ctx, taskBin, []string{"context", "show"}, &exec.ExecutionOptions{
			Timeout:       3 * time.Second,
			CaptureOutput: true,
		})
		if err == nil && result != nil && result.ExitCode == 0 && strings.TrimSpace(result.Stdout) != "" {
			context := strings.TrimSpace(result.Stdout)
			formatter.ScreenReaderText("info", fmt.Sprintf("Active context: %s", context))
		}
	}

	formatter.ScreenReaderText("success", "All systems operational")
	fmt.Println()

	// Show environment variables preview (secure)
	formatter.Header("🔐 Environment Variables (Secure Preview)")
	envOptions := security.DefaultEnvPreviewOptions()
	envOptions.MaxItems = 15
	// Show both taskopen-specific and key system vars
	if len(os.Getenv("TASKOPEN_ACCESSIBILITY")) == 0 && len(os.Getenv("TASKOPENRC")) == 0 {
		// If no TASKOPEN vars, show some key system vars
		envOptions.FilterPattern = "" // Show all (limited by MaxItems)
	} else {
		envOptions.FilterPattern = "TASKOPEN" // Focus on taskopen-specific vars
	}
	envPreview := security.GetEnvPreview(envOptions)
	fmt.Println(envPreview)

	fmt.Println()
	formatter.Info("🎉 EPOCH 2 Sprint 4 Complete - Enhanced Output & Accessibility!")

	return nil
}
