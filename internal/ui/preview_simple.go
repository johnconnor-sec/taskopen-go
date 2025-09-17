// Package ui provides advanced preview functionality for interactive menus
package ui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/johnconnor-sec/taskopen-go/internal/exec"
	"github.com/johnconnor-sec/taskopen-go/internal/output"
	"github.com/johnconnor-sec/taskopen-go/internal/security"
	"github.com/johnconnor-sec/taskopen-go/internal/taskwarrior"
)

// SimplePreview provides basic command previews
type SimplePreview struct {
	formatter *output.Formatter
	executor  *exec.Executor
}

// AdvancedPreview provides comprehensive command previews with dry-run capabilities
type AdvancedPreview struct {
	formatter   *output.Formatter
	executor    *exec.Executor
	taskwarrior *taskwarrior.Client
	cache       map[string]*PreviewInfo // Cache for performance
}

// PreviewMode defines different preview modes
type PreviewMode int

const (
	PreviewBasic PreviewMode = iota
	PreviewDetailed
	PreviewDryRun
	PreviewInteractive
)

// PreviewOptions configures the preview behavior
type PreviewOptions struct {
	Mode        PreviewMode
	ShowRisks   bool
	ShowOutput  bool
	ShowContext bool
	Timeout     time.Duration
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

// extractFilePath for AdvancedPreview (same implementation)
func (ap *AdvancedPreview) extractFilePath(command string) string {
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
		sanitizer := security.NewEnvSanitizer()
		for k, v := range preview.Variables {
			// Sanitize variable value for display
			safeValue := sanitizer.SanitizeValue(k, v)
			fmt.Printf("  $%s = %s\n", k, safeValue)
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

// DryRunPreview performs a safe dry-run of the command to show its effects
func (ap *AdvancedPreview) DryRunPreview(command string, variables map[string]string) (*PreviewInfo, error) {
	// Check cache first for performance
	cacheKey := fmt.Sprintf("%s:%v", command, variables)
	if cached, exists := ap.cache[cacheKey]; exists {
		return cached, nil
	}

	preview := &PreviewInfo{
		Command:   command,
		Variables: variables,
		RiskLevel: ap.assessRiskAdvanced(command),
		Safety:    ap.checkSafetyAdvanced(command),
	}

	// Try to create a safe dry-run version of the command
	dryRunCmd := ap.createDryRunCommand(command)
	if dryRunCmd != "" {
		// Execute dry-run to show potential effects
		if output, err := ap.executeSafely(dryRunCmd, variables); err == nil {
			preview.Description = fmt.Sprintf("Dry-run output:\n%s", output)
		}
	}

	// Cache the result
	ap.cache[cacheKey] = preview
	return preview, nil
}

// createDryRunCommand attempts to create a safe dry-run version of a command
func (ap *AdvancedPreview) createDryRunCommand(command string) string {
	cmd := strings.TrimSpace(strings.ToLower(command))

	// Map dangerous commands to safe alternatives
	dryRunMappings := map[string]string{
		"rm ":      "ls -la ", // Show files that would be deleted
		"del ":     "dir ",    // Windows equivalent
		"git rm":   "git ls-files ",
		"git push": "git log --oneline -5", // Show commits that would be pushed
		"mv ":      "ls -la ",              // Show files that would be moved
		"cp ":      "ls -la ",              // Show files that would be copied
	}

	for dangerous, safe := range dryRunMappings {
		if strings.HasPrefix(cmd, dangerous) {
			return strings.Replace(command, dangerous, safe, 1)
		}
	}

	// For edit commands, just check if file exists
	if strings.Contains(cmd, "vim") || strings.Contains(cmd, "nano") || strings.Contains(cmd, "emacs") {
		filePath := ap.extractFilePath(command)
		if filePath != "" {
			return fmt.Sprintf("ls -la %s", filePath)
		}
	}

	// For web commands, just parse the URL
	if strings.Contains(cmd, "curl") || strings.Contains(cmd, "wget") {
		return "echo '[DRY RUN] Would access network'"
	}

	return "" // No safe dry-run available
}

// executeSafely runs a command safely with timeout and output capture
func (ap *AdvancedPreview) executeSafely(command string, variables map[string]string) (string, error) {
	// Replace variables in command
	expandedCmd := command
	for k, v := range variables {
		expandedCmd = strings.ReplaceAll(expandedCmd, fmt.Sprintf("$%s", k), v)
		expandedCmd = strings.ReplaceAll(expandedCmd, fmt.Sprintf("${%s}", k), v)
	}

	// Parse command into parts
	parts := strings.Fields(expandedCmd)
	if len(parts) == 0 {
		return "", fmt.Errorf("empty command")
	}

	// Use context with short timeout for safety
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Execute with output capture
	result, err := ap.executor.Execute(ctx, parts[0], parts[1:], &exec.ExecutionOptions{
		CaptureOutput: true,
		Timeout:       3 * time.Second,
	})
	if err != nil {
		return "", err
	}

	return result.Stdout, nil
}

// assessRiskAdvanced provides enhanced risk assessment
func (ap *AdvancedPreview) assessRiskAdvanced(command string) string {
	cmd := strings.ToLower(command)

	// Critical risk patterns
	critical := []string{
		"rm -rf", "rm -r /", "del /s", "format c:", "dd if=",
		"chmod 777", "chown -R", "sudo rm -rf",
	}

	// High risk patterns
	high := []string{
		"sudo ", "> /dev", "shutdown", "reboot", "systemctl stop",
		"service stop", "kill -9", "pkill", "killall",
	}

	// Medium risk patterns
	medium := []string{
		"chmod", "chown", "mv ", "git push", "git reset --hard",
		"npm publish", "docker run", "ssh ", "scp ",
	}

	for _, pattern := range critical {
		if strings.Contains(cmd, pattern) {
			return "CRITICAL"
		}
	}

	for _, pattern := range high {
		if strings.Contains(cmd, pattern) {
			return "HIGH"
		}
	}

	for _, pattern := range medium {
		if strings.Contains(cmd, pattern) {
			return "MEDIUM"
		}
	}

	return "SAFE"
}

// checkSafetyAdvanced performs comprehensive safety checks
func (ap *AdvancedPreview) checkSafetyAdvanced(command string) []string {
	var warnings []string
	cmd := strings.ToLower(command)

	// System modification warnings
	if strings.Contains(cmd, "sudo") {
		warnings = append(warnings, "Requires elevated privileges - may modify system")
	}

	if strings.Contains(cmd, "rm ") || strings.Contains(cmd, "del ") {
		warnings = append(warnings, "Will permanently delete files/directories")
	}

	if strings.Contains(cmd, "chmod") || strings.Contains(cmd, "chown") {
		warnings = append(warnings, "Will modify file permissions or ownership")
	}

	// Network access warnings
	if strings.Contains(cmd, "curl") || strings.Contains(cmd, "wget") || strings.Contains(cmd, "ssh") {
		warnings = append(warnings, "Will access network resources")
	}

	// Service management warnings
	if strings.Contains(cmd, "systemctl") || strings.Contains(cmd, "service") {
		warnings = append(warnings, "Will modify system services")
	}

	// Process management warnings
	if strings.Contains(cmd, "kill") || strings.Contains(cmd, "pkill") {
		warnings = append(warnings, "Will terminate running processes")
	}

	// Package management warnings
	if strings.Contains(cmd, "apt ") || strings.Contains(cmd, "yum ") || strings.Contains(cmd, "npm install") {
		warnings = append(warnings, "Will install/modify system packages")
	}

	// Git warnings
	if strings.Contains(cmd, "git push") {
		warnings = append(warnings, "Will publish changes to remote repository")
	}

	if strings.Contains(cmd, "git reset --hard") {
		warnings = append(warnings, "Will permanently discard local changes")
	}

	// Docker warnings
	if strings.Contains(cmd, "docker run") {
		warnings = append(warnings, "Will run containerized application")
	}

	if len(warnings) == 0 {
		warnings = append(warnings, "No significant risks detected - command appears safe")
	}

	return warnings
}

// RenderAdvancedPreview displays comprehensive preview information
func (ap *AdvancedPreview) RenderAdvancedPreview(preview *PreviewInfo, options PreviewOptions) {
	ap.formatter.Subheader("🔍 Advanced Command Preview")

	// Risk assessment with enhanced colors and icons
	switch preview.RiskLevel {
	case "SAFE":
		ap.formatter.Success("✅ Risk Level: %s", preview.RiskLevel)
	case "MEDIUM":
		ap.formatter.Warning("⚠️  Risk Level: %s", preview.RiskLevel)
	case "HIGH":
		ap.formatter.Error("🔥 Risk Level: %s", preview.RiskLevel)
	case "CRITICAL":
		ap.formatter.Error("💀 Risk Level: %s", preview.RiskLevel)
	}

	// Command details
	fmt.Printf("\n📋 Command: %s\n", preview.Command)

	// Show context if requested (with security sanitization)
	if options.ShowContext && len(preview.Variables) > 0 {
		fmt.Println("\n🔧 Environment Variables:")
		sanitizer := security.NewEnvSanitizer()
		for k, v := range preview.Variables {
			// Sanitize variable value for display
			safeValue := sanitizer.SanitizeValue(k, v)
			fmt.Printf("   $%s = %s\n", k, safeValue)
		}
	}

	// Enhanced safety information
	if options.ShowRisks && len(preview.Safety) > 0 {
		fmt.Println("\n🛡️  Safety Analysis:")
		for _, warning := range preview.Safety {
			if strings.Contains(warning, "No significant risks") || strings.Contains(warning, "appears safe") {
				ap.formatter.Success("   ✅ %s", warning)
			} else {
				ap.formatter.Warning("   ⚠️  %s", warning)
			}
		}
	}

	// Show dry-run output if available
	if options.ShowOutput && preview.Description != "" && strings.Contains(preview.Description, "Dry-run") {
		fmt.Println("\n🧪 Dry-run Preview:")
		ap.formatter.Info("%s", strings.TrimPrefix(preview.Description, "Dry-run output:\n"))
	}

	// Enhanced file information
	if preview.FileInfo != nil {
		fmt.Println("\n📁 File Information:")
		fmt.Printf("   Path: %s\n", preview.FileInfo.Path)
		if preview.FileInfo.Exists {
			ap.formatter.Success("   Status: File exists")
			fmt.Printf("   Size: %d bytes\n", preview.FileInfo.Size)

			if options.ShowOutput && preview.FileInfo.Content != "" && len(preview.FileInfo.Content) < 200 {
				fmt.Println("   Preview:")
				lines := strings.Split(preview.FileInfo.Content, "\n")
				for i, line := range lines {
					if i >= 5 { // Limit preview to 5 lines
						fmt.Printf("   ... (%d more lines)\n", len(lines)-5)
						break
					}
					fmt.Printf("   │ %s\n", line)
				}
			}
		} else {
			ap.formatter.Warning("   Status: File not found")
		}
	}

	fmt.Println()
}
