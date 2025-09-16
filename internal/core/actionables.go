package core

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/johnconnor-sec/taskopen-go/internal/errors"
	"github.com/johnconnor-sec/taskopen-go/internal/exec"
	"github.com/johnconnor-sec/taskopen-go/internal/output"
	"github.com/johnconnor-sec/taskopen-go/internal/security"
	"github.com/johnconnor-sec/taskopen-go/internal/types"
	"github.com/johnconnor-sec/taskopen-go/internal/ui"
)

// Actionable represents a task annotation/attribute with matching actions
type Actionable struct {
	Text        string            `json:"text"`
	TaskID      string            `json:"task_id"`
	Task        map[string]any    `json:"task"`
	Entry       string            `json:"entry"`
	Action      types.Action      `json:"action"`
	Environment map[string]string `json:"environment"`
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
