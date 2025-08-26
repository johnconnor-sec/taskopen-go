// Package ui provides simplified preview functionality
package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/johnconnor-sec/taskopen-go/internal/exec"
	"github.com/johnconnor-sec/taskopen-go/internal/output"
)

// SimplePreview provides basic command previews
type SimplePreview struct {
	formatter *output.Formatter
	executor  *exec.Executor
}

// NewSimplePreview creates a simple preview engine
func NewSimplePreview() *SimplePreview {
	return &SimplePreview{
		formatter: output.NewFormatter(os.Stdout),
		executor:  exec.New(exec.ExecutionOptions{Timeout: 5 * time.Second, CaptureOutput: true}),
	}
}

// PreviewInfo contains basic preview information
type PreviewInfo struct {
	Command     string
	Description string
	RiskLevel   string
	Variables   map[string]string
	FileInfo    *FileInfo
	Safety      []string
}

// FileInfo contains basic file information
type FileInfo struct {
	Path    string
	Exists  bool
	Size    int64
	Content string
}

// PreviewCommand creates a preview for a command string
func (sp *SimplePreview) PreviewCommand(command, description string, variables map[string]string) *PreviewInfo {
	preview := &PreviewInfo{
		Command:     command,
		Description: description,
		Variables:   variables,
		RiskLevel:   sp.assessRisk(command),
		Safety:      sp.checkSafety(command),
	}

	// Check for file operations
	if filePath := sp.extractFilePath(command); filePath != "" {
		preview.FileInfo = sp.getFileInfo(filePath)
	}

	return preview
}

// assessRisk provides basic risk assessment
func (sp *SimplePreview) assessRisk(command string) string {
	dangerous := []string{"rm ", "del ", "format", "shutdown", "sudo rm"}
	medium := []string{"sudo ", "chmod", "mv ", "git push"}

	cmd := strings.ToLower(command)
	for _, pattern := range dangerous {
		if strings.Contains(cmd, pattern) {
			return "DANGEROUS"
		}
	}

	for _, pattern := range medium {
		if strings.Contains(cmd, pattern) {
			return "MEDIUM"
		}
	}

	return "SAFE"
}

// checkSafety performs basic safety checks
func (sp *SimplePreview) checkSafety(command string) []string {
	var warnings []string

	if strings.Contains(command, "sudo") {
		warnings = append(warnings, "Command requires elevated privileges")
	}

	if strings.Contains(command, "rm ") {
		warnings = append(warnings, "Command will delete files")
	}

	if strings.Contains(command, "curl") || strings.Contains(command, "wget") {
		warnings = append(warnings, "Command accesses network")
	}

	if len(warnings) == 0 {
		warnings = append(warnings, "No obvious risks detected")
	}

	return warnings
}

// extractFilePath attempts to extract file path from command
func (sp *SimplePreview) extractFilePath(command string) string {
	// Look for file-like patterns
	filePattern := regexp.MustCompile(`[~./][^\s]*\.[a-zA-Z0-9]+`)
	matches := filePattern.FindStringSubmatch(command)
	if len(matches) > 0 {
		return matches[0]
	}
	return ""
}

// getFileInfo retrieves basic file information
func (sp *SimplePreview) getFileInfo(filePath string) *FileInfo {
	if strings.HasPrefix(filePath, "~/") {
		filePath = filepath.Join(os.Getenv("HOME"), filePath[2:])
	}

	info := &FileInfo{Path: filePath}

	if stat, err := os.Stat(filePath); err == nil {
		info.Exists = true
		info.Size = stat.Size()

		// Read small files
		if info.Size < 1024 && !stat.IsDir() {
			if content, err := os.ReadFile(filePath); err == nil {
				info.Content = string(content)
			}
		}
	}

	return info
}

// RenderPreview displays preview information
func (sp *SimplePreview) RenderPreview(preview *PreviewInfo) {
	sp.formatter.Subheader("Command Preview")

	if preview.Description != "" {
		sp.formatter.Info("Description: %s", preview.Description)
	}

	// Risk level with colors
	switch preview.RiskLevel {
	case "SAFE":
		sp.formatter.Success("Risk Level: %s", preview.RiskLevel)
	case "MEDIUM":
		sp.formatter.Warning("Risk Level: %s", preview.RiskLevel)
	case "DANGEROUS":
		sp.formatter.Error("Risk Level: %s", preview.RiskLevel)
	}

	// Command
	fmt.Printf("\nCommand: %s\n", preview.Command)

	// Variables
	if len(preview.Variables) > 0 {
		fmt.Println("\nVariables:")
		for k, v := range preview.Variables {
			fmt.Printf("  $%s = %s\n", k, v)
		}
	}

	// File info
	if preview.FileInfo != nil {
		fmt.Println("\nFile Information:")
		fmt.Printf("  Path: %s\n", preview.FileInfo.Path)
		if preview.FileInfo.Exists {
			fmt.Printf("  Size: %d bytes\n", preview.FileInfo.Size)
			if preview.FileInfo.Content != "" {
				fmt.Printf("  Content:\n%s\n", preview.FileInfo.Content)
			}
		} else {
			fmt.Println("  Status: File not found")
		}
	}

	// Safety warnings
	if len(preview.Safety) > 0 {
		fmt.Println("\nSafety Checks:")
		for _, warning := range preview.Safety {
			if strings.Contains(warning, "No obvious risks") {
				sp.formatter.Success("✓ %s", warning)
			} else {
				sp.formatter.Warning("⚠ %s", warning)
			}
		}
	}
}

// CreatePreviewFunction returns a function suitable for MenuConfig.PreviewFunc
func CreatePreviewFunction() func(MenuItem) string {
	preview := NewSimplePreview()

	return func(item MenuItem) string {
		// Extract command-like information from menu item
		command := item.Text
		if item.Description != "" {
			command = item.Description
		}

		// Basic preview generation
		variables := map[string]string{
			"ITEM": item.Text,
			"ID":   item.ID,
		}

		previewInfo := preview.PreviewCommand(command, item.Description, variables)

		// Build preview string
		var result strings.Builder
		result.WriteString(fmt.Sprintf("Command: %s\n", previewInfo.Command))
		result.WriteString(fmt.Sprintf("Risk: %s\n", previewInfo.RiskLevel))

		if len(previewInfo.Safety) > 0 && !strings.Contains(previewInfo.Safety[0], "No obvious risks") {
			result.WriteString("Warnings:\n")
			for _, warning := range previewInfo.Safety {
				result.WriteString(fmt.Sprintf("  • %s\n", warning))
			}
		}

		if previewInfo.FileInfo != nil && previewInfo.FileInfo.Exists {
			result.WriteString(fmt.Sprintf("File: %s (%d bytes)\n",
				previewInfo.FileInfo.Path, previewInfo.FileInfo.Size))
		}

		return result.String()
	}
}
