// Package core implements the main taskopen business logic
package core

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/johnconnor-sec/taskopen-go/internal/config"
	"github.com/johnconnor-sec/taskopen-go/internal/errors"
	"github.com/johnconnor-sec/taskopen-go/internal/exec"
	"github.com/johnconnor-sec/taskopen-go/internal/output"
	"github.com/johnconnor-sec/taskopen-go/internal/security"
	"github.com/johnconnor-sec/taskopen-go/internal/types"
	"github.com/johnconnor-sec/taskopen-go/internal/ui"
)

// TaskProcessor handles the main taskopen workflow
type TaskProcessor struct {
	config         *config.Config
	executor       *exec.Executor
	formatter      *output.Formatter
	logger         *output.Logger
	builtinHandler *BuiltinHandler
}

// NewTaskProcessor creates a new task processor
func NewTaskProcessor(cfg *config.Config) *TaskProcessor {
	executor := exec.New(exec.ExecutionOptions{Timeout: 30 * time.Second})
	formatter := output.NewFormatter(os.Stdout)
	logger := output.NewLogger()

	return &TaskProcessor{
		config:         cfg,
		executor:       executor,
		formatter:      formatter,
		logger:         logger,
		builtinHandler: NewBuiltinHandler(executor, formatter, logger),
	}
}

// Actionable represents a task annotation/attribute with matching actions
type Actionable struct {
	Text        string            `json:"text"`
	TaskID      string            `json:"task_id"`
	Task        map[string]any    `json:"task"`
	Entry       string            `json:"entry"`
	Action      types.Action      `json:"action"`
	Environment map[string]string `json:"environment"`
}

// ProcessTasks is the main taskopen workflow
func (tp *TaskProcessor) ProcessTasks(ctx context.Context, filters []string, single bool, interactive bool) error {
	// Skip context for now and use provided filters directly
	allFilters := filters
	if len(allFilters) == 0 && tp.config.General.BaseFilter != "" {
		// Add base filter when no filters provided
		allFilters = append(allFilters, strings.Fields(tp.config.General.BaseFilter)...)
	}

	// Get tasks from taskwarrior
	tasks, err := tp.getTasksFromTaskwarrior(ctx, allFilters)
	if err != nil {
		return errors.Wrap(err, errors.ActionExecution, "Failed to get tasks from taskwarrior")
	}

	if len(tasks) == 0 {
		tp.formatter.Warning("No tasks match the specified filter")
		return nil
	}

	tp.logger.Debug("Retrieved tasks", map[string]any{"count": len(tasks)})

	// Find actionable items
	actionables, err := tp.findActionableItems(ctx, tasks, single)
	if err != nil {
		return err
	}

	if len(actionables) == 0 {
		tp.formatter.Warning("No actionable items found")
		return nil
	}

	// Sort actionables
	tp.sortActionables(actionables)

	// Execute actions
	if interactive && len(actionables) > 1 {
		return tp.interactiveSelection(ctx, actionables)
	} else if len(actionables) == 1 {
		return tp.executeActionable(ctx, actionables[0])
	} else {
		return tp.listActionables(actionables)
	}
}

// getCurrentContext gets the current taskwarrior context
// func (tp *TaskProcessor) getCurrentContext(ctx context.Context) (string, error) {
// 	if tp == nil || tp.config == nil || tp.executor == nil {
// 		return "", fmt.Errorf("task processor not properly initialized")
// 	}
//
// 	args := append(tp.config.General.TaskArgs, "context", "show")
// 	result, err := tp.executor.Execute(ctx, tp.config.General.TaskBin, args,
// 		&exec.ExecutionOptions{CaptureOutput: true, Timeout: 5 * time.Second})
// 	if err != nil {
// 		return "", fmt.Errorf("failed to execute taskwarrior: %w", err)
// 	}
//
// 	if result == nil {
// 		return "", fmt.Errorf("no result from taskwarrior execution")
// 	}
//
// 	if result.ExitCode != 0 {
// 		return "", fmt.Errorf("taskwarrior context command failed with exit code %d", result.ExitCode)
// 	}
//
// 	return strings.TrimSpace(result.Stdout), nil
// }

// getTasksFromTaskwarrior retrieves tasks as JSON from taskwarrior
func (tp *TaskProcessor) getTasksFromTaskwarrior(ctx context.Context, filters []string) ([]map[string]any, error) {
	// Build taskwarrior export command: task [general_args] [filters] export
	args := append([]string{}, tp.config.General.TaskArgs...) // Copy slice
	args = append(args, filters...)
	args = append(args, "export")

	result, err := tp.executor.Execute(ctx, tp.config.General.TaskBin, args,
		&exec.ExecutionOptions{
			CaptureOutput: true,
			Timeout:       10 * time.Second,
		})
	if err != nil {
		return nil, fmt.Errorf("taskwarrior execution failed: %w", err)
	}

	if result == nil {
		return nil, fmt.Errorf("no result from taskwarrior execution")
	}

	if result.ExitCode != 0 {
		tp.logger.Error("Taskwarrior export failed", map[string]any{
			"exit_code": result.ExitCode,
			"stderr":    result.Stderr,
			"stdout":    result.Stdout,
		})
		return nil, fmt.Errorf("taskwarrior export failed with exit code %d: %s", result.ExitCode, result.Stderr)
	}

	if result.Stdout == "" {
		// No tasks found - return empty slice instead of error
		return []map[string]any{}, nil
	}

	// Parse JSON output
	var tasks []map[string]any
	if err := json.Unmarshal([]byte(result.Stdout), &tasks); err != nil {
		tp.logger.Error("Failed to parse taskwarrior JSON", map[string]any{
			"error":  err.Error(),
			"stdout": result.Stdout,
		})
		return nil, fmt.Errorf("failed to parse taskwarrior JSON: %w", err)
	}

	tp.logger.Info("Retrieved tasks from taskwarrior", map[string]any{
		"count": len(tasks),
	})

	return tasks, nil
}

// findActionableItems finds all actionable items across tasks
func (tp *TaskProcessor) findActionableItems(ctx context.Context, tasks []map[string]any, single bool) ([]*Actionable, error) {
	var actionables []*Actionable

	// Build action map by target
	actionMap := make(map[string][]types.Action)
	for _, action := range tp.config.Actions {
		if actionMap[action.Target] == nil {
			actionMap[action.Target] = make([]types.Action, 0)
		}
		actionMap[action.Target] = append(actionMap[action.Target], action)
	}

	// Process each task
	for _, task := range tasks {
		baseEnv := tp.buildEnvironment(task)

		// Process each task attribute
		for attr, value := range task {
			actions, hasActions := actionMap[attr]
			if !hasActions {
				continue
			}

			if attr == "annotations" {
				// Handle annotations array
				if annotations, ok := value.([]any); ok {
					for _, annInterface := range annotations {
						if ann, ok := annInterface.(map[string]any); ok {
							if desc, hasDesc := ann["description"].(string); hasDesc {
								entry := ""
								if entryVal, hasEntry := ann["entry"]; hasEntry {
									entry = fmt.Sprintf("%v", entryVal)
								}

								matches := tp.matchActionsLabel(ctx, baseEnv, desc, actions, single)
								for _, match := range matches {
									match.Entry = entry
									actionables = append(actionables, match)
								}
							}
						}
					}
				}
			} else {
				// Handle regular attributes
				text := fmt.Sprintf("%v", value)
				entry := ""
				if entryVal, hasEntry := task["entry"]; hasEntry {
					entry = fmt.Sprintf("%v", entryVal)
				}

				matches := tp.matchActionsPure(ctx, baseEnv, text, actions, single)
				for _, match := range matches {
					match.Entry = entry
					actionables = append(actionables, match)
				}
			}
		}
	}

	return actionables, nil
}

// buildEnvironment creates environment variables for a task
func (tp *TaskProcessor) buildEnvironment(task map[string]any) map[string]string {
	env := make(map[string]string)

	// Copy system environment
	for _, envVar := range os.Environ() {
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) == 2 {
			env[parts[0]] = parts[1]
		}
	}

	// Add PATH extension
	if tp.config.General.PathExt != "" {
		env["PATH"] = tp.config.General.PathExt + ":" + env["PATH"]
	}

	// Add editor
	if tp.config.General.Editor != "" {
		env["EDITOR"] = tp.config.General.Editor
	}

	// Add task-specific variables
	if uuid, ok := task["uuid"].(string); ok {
		env["UUID"] = uuid
	}

	if id, ok := task["id"]; ok {
		env["ID"] = fmt.Sprintf("%v", id)
	} else {
		env["ID"] = ""
	}

	// Add task attributes
	for attr := range strings.SplitSeq(tp.config.General.TaskAttributes, ",") {
		attr = strings.TrimSpace(attr)
		if value, exists := task[attr]; exists {
			env["TASK_"+strings.ToUpper(attr)] = fmt.Sprintf("%v", value)
		}
	}

	return env
}

// matchActionsLabel matches actions against annotation text with label support
func (tp *TaskProcessor) matchActionsLabel(ctx context.Context, baseEnv map[string]string, text string, actions []types.Action, single bool) []*Actionable {
	var matches []*Actionable

	// Split annotation into label and file part
	splitRegex := regexp.MustCompile(`^((\S+):\s+)?(.*)$`)
	splitMatches := splitRegex.FindStringSubmatch(text)
	if len(splitMatches) != 4 {
		tp.logger.Error(
			"Malformed annotation",
			map[string]any{"text": text},
		)
		return matches
	}

	label := splitMatches[2]
	file := splitMatches[3]

	for _, action := range actions {
		env := tp.copyEnvironment(baseEnv)

		// Check label regex
		if action.LabelRegex != "" {
			labelRegex, err := regexp.Compile(action.LabelRegex)
			if err != nil {
				tp.logger.Error("Invalid label regex", map[string]any{"regex": action.LabelRegex, "error": err.Error()})
				continue
			}
			if !labelRegex.MatchString(label) {
				continue
			}
		}

		// Check file regex
		fileRegex, err := regexp.Compile(action.Regex)
		if err != nil {
			tp.logger.Error("Invalid file regex", map[string]any{"regex": action.Regex, "error": err.Error()})
			continue
		}

		fileMatches := fileRegex.FindStringSubmatch(file)
		if len(fileMatches) == 0 {
			continue
		}

		// Set environment variables
		env["LAST_MATCH"] = ""
		if len(fileMatches) > 0 {
			env["LAST_MATCH"] = fileMatches[0]
		}
		env["LABEL"] = label
		env["FILE"] = tp.expandPath(file)
		env["ANNOTATION"] = text

		// Apply filter command if specified
		if action.FilterCommand != "" {
			if !tp.executeFilter(ctx, action.FilterCommand, env) {
				tp.logger.Info("Filter command filtered out action", map[string]any{
					"action": action.Name,
					"text":   text,
				})
				continue
			}
		}

		// Create actionable
		taskID := env["UUID"]
		if taskID == "" {
			taskID = env["ID"]
		}

		actionable := &Actionable{
			Text:        text,
			TaskID:      taskID,
			Action:      action,
			Environment: env,
		}

		matches = append(matches, actionable)

		if single {
			break
		}
	}

	return matches
}

// matchActionsPure matches actions against plain text (non-annotation attributes)
func (tp *TaskProcessor) matchActionsPure(ctx context.Context, baseEnv map[string]string, text string, actions []types.Action, single bool) []*Actionable {
	var matches []*Actionable

	for _, action := range actions {
		env := tp.copyEnvironment(baseEnv)

		// Check file regex
		fileRegex, err := regexp.Compile(action.Regex)
		if err != nil {
			tp.logger.Error("Invalid regex", map[string]any{"regex": action.Regex, "error": err.Error()})
			continue
		}

		fileMatches := fileRegex.FindStringSubmatch(text)
		if len(fileMatches) == 0 {
			continue
		}

		// Set environment variables
		env["LAST_MATCH"] = ""
		if len(fileMatches) > 0 {
			env["LAST_MATCH"] = fileMatches[0]
		}
		env["FILE"] = text
		env["ANNOTATION"] = text

		// Warn about unused labelregex
		if action.LabelRegex != "" {
			tp.logger.Warn("labelregex is ignored for actions not targeting annotations", map[string]any{
				"action": action.Name,
			})
		}

		// Apply filter command if specified
		if action.FilterCommand != "" {
			if !tp.executeFilter(ctx, action.FilterCommand, env) {
				tp.logger.Info("Filter command filtered out action", map[string]any{
					"action": action.Name,
					"text":   text,
				})
				continue
			}
		}

		// Create actionable
		taskID := env["UUID"]
		if taskID == "" {
			taskID = env["ID"]
		}

		actionable := &Actionable{
			Text:        text,
			TaskID:      taskID,
			Action:      action,
			Environment: env,
		}

		matches = append(matches, actionable)

		if single {
			break
		}
	}

	return matches
}

// copyEnvironment creates a copy of environment map
func (tp *TaskProcessor) copyEnvironment(baseEnv map[string]string) map[string]string {
	env := make(map[string]string)
	maps.Copy(env, baseEnv)
	return env
}

// expandPath expands tilde in file paths
func (tp *TaskProcessor) expandPath(path string) string {
	if strings.HasPrefix(path, "~") {
		home := os.Getenv("HOME")
		if home != "" {
			return strings.Replace(path, "~", home, 1)
		}
	}
	return path
}

// executeFilter runs a filter command and returns whether it passed
func (tp *TaskProcessor) executeFilter(ctx context.Context, command string, env map[string]string) bool {
	// Expand environment variables in command
	expandedCommand := tp.expandEnvironmentVars(command, env)

	result, err := tp.executor.Execute(ctx, "sh", []string{"-c", expandedCommand},
		&exec.ExecutionOptions{Environment: env})

	return err == nil && result.ExitCode == 0
}

// expandEnvironmentVars expands environment variables in a command string
func (tp *TaskProcessor) expandEnvironmentVars(command string, env map[string]string) string {
	result := command
	for key, value := range env {
		result = strings.ReplaceAll(result, "$"+key, value)
		result = strings.ReplaceAll(result, "${"+key+"}", value)
	}
	return result
}

// sortActionables sorts actionable items according to configuration
func (tp *TaskProcessor) sortActionables(actionables []*Actionable) {
	sortKeys := tp.parseSortKeys(tp.config.General.Sort)

	sort.Slice(actionables, func(i, j int) bool {
		a, b := actionables[i], actionables[j]

		for _, sortKey := range sortKeys {
			var result int

			switch sortKey.Key {
			case "annot":
				result = strings.Compare(a.Text, b.Text)
			case "entry":
				result = strings.Compare(a.Entry, b.Entry)
			case "id":
				aID := tp.getTaskInt(a.Task, "id")
				bID := tp.getTaskInt(b.Task, "id")
				result = aID - bID
			case "urgency":
				aUrgency := tp.getTaskFloat(a.Task, "urgency")
				bUrgency := tp.getTaskFloat(b.Task, "urgency")
				if aUrgency < bUrgency {
					result = -1
				} else if aUrgency > bUrgency {
					result = 1
				}
			default:
				aVal := tp.getTaskString(a.Task, sortKey.Key)
				bVal := tp.getTaskString(b.Task, sortKey.Key)
				result = strings.Compare(aVal, bVal)
			}

			if result != 0 {
				if sortKey.Desc {
					return result > 0
				}
				return result < 0
			}
		}
		return false
	})
}

// SortKey represents a sort field and direction
type SortKey struct {
	Key  string
	Desc bool
}

// parseSortKeys parses the sort string into sort keys
func (tp *TaskProcessor) parseSortKeys(sortStr string) []SortKey {
	var keys []SortKey

	for field := range strings.SplitSeq(sortStr, ",") {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}

		desc := false
		if strings.HasSuffix(field, "-") {
			desc = true
			field = field[:len(field)-1]
		} else if strings.HasSuffix(field, "+") {
			field = field[:len(field)-1]
		}

		keys = append(keys, SortKey{Key: field, Desc: desc})
	}

	return keys
}

// Helper functions for type conversion
func (tp *TaskProcessor) getTaskString(task map[string]any, key string) string {
	if value, exists := task[key]; exists {
		return fmt.Sprintf("%v", value)
	}
	return ""
}

func (tp *TaskProcessor) getTaskInt(task map[string]any, key string) int {
	if value, exists := task[key]; exists {
		if intVal, ok := value.(float64); ok {
			return int(intVal)
		}
		if strVal, ok := value.(string); ok {
			if intVal, err := strconv.Atoi(strVal); err == nil {
				return intVal
			}
		}
	}
	return 0
}

func (tp *TaskProcessor) getTaskFloat(task map[string]any, key string) float64 {
	if value, exists := task[key]; exists {
		if floatVal, ok := value.(float64); ok {
			return floatVal
		}
		if strVal, ok := value.(string); ok {
			if floatVal, err := strconv.ParseFloat(strVal, 64); err == nil {
				return floatVal
			}
		}
	}
	return 0.0
}

// interactiveSelection handles interactive selection of actionables using the secure TUI
func (tp *TaskProcessor) interactiveSelection(ctx context.Context, actionables []*Actionable) error {
	// Convert actionables to menu items
	menuItems := tp.convertActionablesToMenuItems(actionables)

	// Configure the menu
	config := tp.createMenuConfig()

	// Configure secure TUI settings
	tuiConfig := ui.SecureTUIConfig{
		ShowPreview:       true,
		PreviewWidth:      40,
		HideEnvVars:       true, // IMPORTANT: Hide environment variables by default
		VisibilityLevel:   security.VisibilityMasked,
		AccessibilityMode: output.AccessibilityNormal,
	}

	// Create the secure TUI
	tui, err := ui.NewSecureTUI(menuItems, config, tuiConfig)
	if err != nil {
		return fmt.Errorf("failed to create secure TUI: %w", err)
	}
	defer tui.Close()

	// Show the TUI and get user selection
	selected, err := tui.Show()
	if err != nil {
		return fmt.Errorf("TUI interaction failed: %w", err)
	}

	// Handle cancellation
	if selected == nil {
		tp.formatter.Info("Action cancelled by user")
		return nil
	}

	// Find the corresponding actionable
	selectedIndex := -1
	for i, item := range menuItems {
		if item.ID == selected.ID {
			selectedIndex = i
			break
		}
	}

	if selectedIndex == -1 {
		return fmt.Errorf("selected item not found")
	}

	// Execute the selected actionable
	tp.formatter.Success("Executing: %s", selected.Text)
	return tp.executeActionable(ctx, actionables[selectedIndex])
}

// executeActionable executes a single actionable item
func (tp *TaskProcessor) executeActionable(ctx context.Context, actionable *Actionable) error {
	tp.logger.Info("Executing action", map[string]any{
		"action": actionable.Action.Name,
		"text":   actionable.Text,
	})

	// Expand environment variables in command
	command := tp.expandEnvironmentVars(actionable.Action.Command, actionable.Environment)

	tp.formatter.Info("Executing: %s", command)

	// Check if this is a built-in command
	if tp.builtinHandler.IsBuiltinCommand(command) {
		return tp.builtinHandler.ExecuteBuiltinCommand(ctx, command, actionable.Environment)
	}

	// Execute as external command - use direct execution for better interactive support
	var result *exec.ExecutionResult
	var err error

	// Check if we need shell or can use direct execution
	if tp.executor.NeedsShell(command) {
		tp.logger.Debug("Using shell execution for command with shell features", map[string]any{
			"command": command,
		})
		// Use shell for complex commands
		result, err = tp.executor.Execute(ctx, "sh", []string{"-c", command},
			&exec.ExecutionOptions{Environment: actionable.Environment})
	} else {
		tp.logger.Debug("Using direct execution for interactive compatibility", map[string]any{
			"command": command,
		})
		// Use direct execution for simple commands (better for interactive programs)
		result, err = tp.executor.ExecuteDirect(ctx, command,
			&exec.ExecutionOptions{Environment: actionable.Environment})
	}

	if err != nil {
		return errors.Wrap(err, errors.ActionExecution, "Failed to execute action")
	}

	if result.ExitCode != 0 {
		tp.formatter.Error("Command failed with exit code %d", result.ExitCode)
		if result.Stderr != "" {
			tp.formatter.Error("Error output: %s", result.Stderr)
		}
		return errors.New(errors.ActionExecution, "Command execution failed")
	}

	if result.Stdout != "" {
		fmt.Println(result.Stdout)
	}

	tp.formatter.Success("Action completed successfully")
	return nil
}

// listActionables lists all actionable items for user selection
func (tp *TaskProcessor) listActionables(actionables []*Actionable) error {
	tp.formatter.Subheader("Available Actions")

	for i, actionable := range actionables {
		tp.formatter.List("%d. %s: %s", i+1, actionable.Action.Name, actionable.Text)
		tp.formatter.Info("   Command: %s", actionable.Action.Command)
	}

	fmt.Println()
	tp.formatter.Info("Use the interactive menu to select an action")

	return nil
}

// convertActionablesToMenuItems converts actionable items to interactive menu items
func (tp *TaskProcessor) convertActionablesToMenuItems(actionables []*Actionable) []ui.MenuItem {
	items := make([]ui.MenuItem, len(actionables))

	for i, actionable := range actionables {
		// Create a rich description with task and action details
		description := fmt.Sprintf("Action: %s | Command: %s",
			actionable.Action.Name,
			actionable.Action.Command)

		// Add task context if available
		if actionable.Task != nil {
			if desc, ok := actionable.Task["description"].(string); ok && desc != "" {
				description = fmt.Sprintf("Task: %s | %s", desc, description)
			}
		}

		items[i] = ui.MenuItem{
			ID:          fmt.Sprintf("actionable-%d", i),
			Text:        actionable.Text,
			Description: description,
			Data: map[string]any{
				"actionable": actionable,
				"command":    actionable.Action.Command,
				"action":     actionable.Action.Name,
				"index":      i,
			},
			Action: func() error {
				// This will be handled by the main selection logic
				return nil
			},
		}
	}

	return items
}

// createMenuConfig creates a menu configuration optimized for actionable selection
func (tp *TaskProcessor) createMenuConfig() ui.MenuConfig {
	config := ui.DefaultMenuConfig()

	// Customize for taskopen actionables
	config.Title = "🎯 Taskopen Actions"
	config.ShowDescription = true
	config.AllowSearch = true
	config.VimMode = true // Enable vim navigation by default
	config.MaxItems = 15  // Show more items for better overview

	// Add preview function for commands
	config.PreviewFunc = tp.createActionablePreview()

	// Customize help
	config.CustomHelp = "Select an action to execute on the matched annotation"

	return config
}

// createActionablePreview creates a preview function for actionable commands
func (tp *TaskProcessor) createActionablePreview() func(ui.MenuItem) string {
	// Create sanitizer once and reuse it
	sanitizer := security.NewEnvSanitizer()
	sanitizer.SetVisibilityLevel(security.VisibilityMasked)

	return func(item ui.MenuItem) string {
		data, ok := item.Data.(map[string]any)
		if !ok {
			return "No preview available"
		}

		actionable, ok := data["actionable"].(*Actionable)
		if !ok {
			return "Invalid actionable data"
		}

		var preview strings.Builder

		// Command preview (sanitized to hide sensitive info)
		command := actionable.Action.Command
		// Expand environment variables for display but sanitize them
		expandedCommand := tp.expandEnvironmentVars(command, actionable.Environment)
		preview.WriteString(fmt.Sprintf("📋 Command: %s\n", expandedCommand))

		// Risk assessment
		risk := tp.assessCommandRisk(actionable.Action.Command)
		preview.WriteString(fmt.Sprintf("⚠️  Risk Level: %s\n", risk))

		// Task information
		if actionable.Task != nil {
			if desc, ok := actionable.Task["description"].(string); ok && desc != "" {
				preview.WriteString(fmt.Sprintf("📝 Task: %s\n", desc))
			}
			if project, ok := actionable.Task["project"].(string); ok && project != "" {
				preview.WriteString(fmt.Sprintf("📁 Project: %s\n", project))
			}
			if priority, ok := actionable.Task["priority"].(string); ok && priority != "" {
				preview.WriteString(fmt.Sprintf("⭐ Priority: %s\n", priority))
			}
		}

		// Environment variables (securely sanitized)
		if len(actionable.Environment) > 0 {
			preview.WriteString("\n🔧 Task Variables (Sanitized):\n")

			// Show only important task-related vars, sanitized
			importantVars := []string{"UUID", "ID", "FILE", "ANNOTATION", "LABEL", "LAST_MATCH"}
			for _, varName := range importantVars {
				if value, exists := actionable.Environment[varName]; exists {
					sanitizedValue := sanitizer.SanitizeValue(varName, value)
					preview.WriteString(fmt.Sprintf("   %s=%s\n", varName, sanitizedValue))
				}
			}

			// Show count of hidden environment variables
			hiddenCount := len(actionable.Environment) - len(importantVars)
			if hiddenCount > 0 {
				preview.WriteString(fmt.Sprintf("\n   🔒 %d environment variables hidden for security\n", hiddenCount))
			}
		}

		return preview.String()
	}
}

// assessCommandRisk provides basic risk assessment for commands
func (tp *TaskProcessor) assessCommandRisk(command string) string {
	cmd := strings.ToLower(command)

	// Critical risk patterns
	critical := []string{"rm -rf", "format", "dd if="}
	for _, pattern := range critical {
		if strings.Contains(cmd, pattern) {
			return "CRITICAL"
		}
	}

	// High risk patterns
	high := []string{"sudo rm", "rm /", "shutdown", "reboot"}
	for _, pattern := range high {
		if strings.Contains(cmd, pattern) {
			return "HIGH"
		}
	}

	// Medium risk patterns
	medium := []string{"sudo", "rm ", "mv ", "chmod", "chown"}
	for _, pattern := range medium {
		if strings.Contains(cmd, pattern) {
			return "MEDIUM"
		}
	}

	return "SAFE"
}

// getCommandWarnings returns safety warnings for commands
// func (tp *TaskProcessor) getCommandWarnings(command string) []string {
// 	var warnings []string
// 	cmd := strings.ToLower(command)
//
// 	if strings.Contains(cmd, "sudo") {
// 		warnings = append(warnings, "Requires elevated privileges")
// 	}
// 	if strings.Contains(cmd, "rm ") {
// 		warnings = append(warnings, "Will delete files or directories")
// 	}
// 	if strings.Contains(cmd, "curl") || strings.Contains(cmd, "wget") {
// 		warnings = append(warnings, "Will access network resources")
// 	}
// 	if strings.Contains(cmd, "chmod") || strings.Contains(cmd, "chown") {
// 		warnings = append(warnings, "Will modify file permissions")
// 	}
//
// 	return warnings
// }
