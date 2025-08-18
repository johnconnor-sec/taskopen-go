// Package core provides the main integration layer combining configuration, execution, and taskwarrior.
package core

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/johnconnor-sec/taskopen-go/taskopen/internal/config"
	"github.com/johnconnor-sec/taskopen-go/taskopen/internal/errors"
	"github.com/johnconnor-sec/taskopen-go/taskopen/internal/exec"
	"github.com/johnconnor-sec/taskopen-go/taskopen/internal/taskwarrior"
	"github.com/johnconnor-sec/taskopen-go/taskopen/internal/types"
)

// TaskOpen provides the main application functionality.
type TaskOpen struct {
	config     *config.Config
	executor   *exec.Executor
	taskClient *taskwarrior.Client
}

// New creates a new TaskOpen instance.
func New(cfg *config.Config) (*TaskOpen, error) {
	if cfg == nil {
		return nil, errors.New(errors.ConfigInvalid, "Configuration is required")
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, errors.Wrap(err, errors.ConfigInvalid, "Invalid configuration")
	}

	// Create executor with reasonable defaults
	execOptions := exec.ExecutionOptions{
		Timeout:       30 * time.Second,
		CaptureOutput: true,
		Retry: exec.RetryOptions{
			MaxAttempts:       2,
			BaseDelay:         200 * time.Millisecond,
			MaxDelay:          2 * time.Second,
			BackoffMultiplier: 1.5,
		},
		Sandbox: exec.SandboxOptions{
			MaxMemoryMB: 256,
		},
	}

	executor := exec.New(execOptions)

	// Create taskwarrior client
	taskClient := taskwarrior.NewClient(
		cfg.General.TaskBin,
		cfg.General.TaskArgs,
		30*time.Second,
	)

	return &TaskOpen{
		config:     cfg,
		executor:   executor,
		taskClient: taskClient,
	}, nil
}

// VerifySetup checks that all required components are available and functional.
func (to *TaskOpen) VerifySetup(ctx context.Context) error {
	// Check taskwarrior availability
	if err := taskwarrior.CheckTaskwarrior(ctx, to.config.General.TaskBin); err != nil {
		return err
	}

	// Check editor availability if needed
	if to.config.General.Editor != "" {
		editorParts := strings.Fields(to.config.General.Editor)
		if len(editorParts) > 0 {
			_, err := to.executor.ExecuteFilter(ctx, "command", []string{"-v", editorParts[0]}, nil)
			if err != nil {
				return errors.New(errors.ActionExecution, "Editor not found").
					WithDetails(fmt.Sprintf("Editor: %s", to.config.General.Editor)).
					WithSuggestions([]string{
						"Install the editor or update the configuration",
						"Check that the editor is in your PATH",
						"Run 'taskopen config init' to reconfigure",
					})
			}
		}
	}

	return nil
}

// GetTasks retrieves tasks based on filters.
func (to *TaskOpen) GetTasks(ctx context.Context, filters []string) ([]taskwarrior.Task, error) {
	// Add base filter from configuration
	if to.config.General.BaseFilter != "" {
		filters = append([]string{to.config.General.BaseFilter}, filters...)
	}

	return to.taskClient.Query(ctx, filters)
}

// GetTasksStream retrieves tasks using streaming for large datasets.
func (to *TaskOpen) GetTasksStream(ctx context.Context, filters []string) (<-chan taskwarrior.Task, <-chan error) {
	// Add base filter from configuration
	if to.config.General.BaseFilter != "" {
		filters = append([]string{to.config.General.BaseFilter}, filters...)
	}

	return to.taskClient.ExportStream(ctx, filters)
}

// FindActionables identifies actionable items from tasks based on configured actions.
func (to *TaskOpen) FindActionables(ctx context.Context, tasks []taskwarrior.Task) ([]types.Actionable, error) {
	var actionables []types.Actionable

	for _, task := range tasks {
		// Check each configured action
		for _, action := range to.config.Actions {
			taskActionables, err := to.findActionablesForTask(task, action)
			if err != nil {
				return nil, errors.Wrap(err, errors.ActionInvalid, "Failed to process task").
					WithDetails(fmt.Sprintf("Task UUID: %s, Action: %s", task.UUID, action.Name))
			}

			actionables = append(actionables, taskActionables...)
		}
	}

	return actionables, nil
}

// findActionablesForTask finds actionables for a specific task and action.
func (to *TaskOpen) findActionablesForTask(task taskwarrior.Task, action types.Action) ([]types.Actionable, error) {
	var actionables []types.Actionable

	// Compile regex patterns
	regex, err := regexp.Compile(action.Regex)
	if err != nil {
		return nil, errors.New(errors.ValidationFailed, "Invalid action regex").
			WithDetails(fmt.Sprintf("Action: %s, Regex: %s", action.Name, action.Regex))
	}

	var labelRegex *regexp.Regexp
	if action.LabelRegex != "" && action.LabelRegex != ".*" {
		labelRegex, err = regexp.Compile(action.LabelRegex)
		if err != nil {
			return nil, errors.New(errors.ValidationFailed, "Invalid action label regex").
				WithDetails(fmt.Sprintf("Action: %s, LabelRegex: %s", action.Name, action.LabelRegex))
		}
	}

	// Check target - annotations or description
	switch strings.ToLower(action.Target) {
	case "annotations":
		for _, annotation := range task.Annotations {
			if regex.MatchString(annotation.Description) {
				// Check label regex if specified
				if labelRegex == nil || labelRegex.MatchString(annotation.Description) {
					actionable, err := to.createActionable(task, action, annotation.Description, annotation.Description)
					if err != nil {
						return nil, err
					}
					actionables = append(actionables, actionable)
				}
			}
		}

	case "description":
		if regex.MatchString(task.Description) {
			// Check label regex if specified
			if labelRegex == nil || labelRegex.MatchString(task.Description) {
				actionable, err := to.createActionable(task, action, task.Description, task.Description)
				if err != nil {
					return nil, err
				}
				actionables = append(actionables, actionable)
			}
		}

	default:
		return nil, errors.New(errors.ActionInvalid, "Invalid action target").
			WithDetails(fmt.Sprintf("Target: %s, valid values: annotations, description", action.Target))
	}

	return actionables, nil
}

// createActionable creates an actionable from a task, action, and matched text.
func (to *TaskOpen) createActionable(task taskwarrior.Task, action types.Action, text, entry string) (types.Actionable, error) {
	// Serialize task to JSON for the actionable
	taskJSON, err := json.Marshal(task)
	if err != nil {
		return types.Actionable{}, errors.Wrap(err, errors.InternalError, "Failed to serialize task")
	}

	// Build environment variables for command execution
	env := to.buildEnvironment(task, action, text)

	actionable := types.Actionable{
		Text:   text,
		Task:   json.RawMessage(taskJSON),
		Entry:  entry,
		Action: action,
		Env:    env,
	}

	return actionable, nil
}

// buildEnvironment creates environment variables for action execution.
func (to *TaskOpen) buildEnvironment(task taskwarrior.Task, action types.Action, matchedText string) map[string]string {
	env := make(map[string]string)

	// Task-specific variables
	env["TASK_ID"] = fmt.Sprintf("%d", task.ID)
	env["TASK_UUID"] = task.UUID
	env["TASK_DESCRIPTION"] = task.Description
	env["TASK_STATUS"] = task.Status
	env["TASK_PROJECT"] = task.Project
	env["TASK_PRIORITY"] = task.Priority

	if len(task.Tags) > 0 {
		env["TASK_TAGS"] = strings.Join(task.Tags, ",")
	}

	if task.Due != nil {
		env["TASK_DUE"] = task.Due.Format(time.RFC3339)
	}

	// Action-specific variables
	env["ACTION_NAME"] = action.Name
	env["ACTION_TARGET"] = action.Target
	env["MATCHED_TEXT"] = matchedText

	// Find regex matches
	if action.Regex != "" {
		regex, err := regexp.Compile(action.Regex)
		if err == nil {
			matches := regex.FindStringSubmatch(matchedText)
			if len(matches) > 1 {
				env["LAST_MATCH"] = matches[len(matches)-1]

				// Add numbered match groups
				for i, match := range matches[1:] {
					env[fmt.Sprintf("MATCH_%d", i+1)] = match
				}
			}
		}
	}

	// File-related variables (if applicable)
	if strings.Contains(matchedText, "/") || strings.Contains(matchedText, "\\") {
		env["FILE"] = matchedText
	}

	// System variables
	env["EDITOR"] = to.config.General.Editor
	env["TASKOPEN_CONFIG"] = to.config.ConfigPath

	return env
}

// ExecuteAction executes an actionable with the configured action.
func (to *TaskOpen) ExecuteAction(ctx context.Context, actionable types.Actionable) error {
	action := actionable.Action

	// Choose the appropriate execution method
	var err error

	switch {
	case action.FilterCommand != "":
		// Execute filter command first
		success, filterErr := to.executeFilterCommand(ctx, action.FilterCommand, actionable.Env)
		if filterErr != nil {
			return errors.Wrap(filterErr, errors.ActionExecution, "Filter command failed")
		}
		if !success {
			return errors.New(errors.ActionExecution, "Filter command returned false").
				WithDetails(fmt.Sprintf("Filter: %s", action.FilterCommand)).
				WithSuggestion("Check filter command logic and requirements")
		}

		// Execute main command
		err = to.executeMainCommand(ctx, action.Command, actionable.Env)

	case action.InlineCommand != "":
		// Execute inline command (non-interactive)
		err = to.executeInlineCommand(ctx, action.InlineCommand, actionable.Env)

	default:
		// Execute main command
		err = to.executeMainCommand(ctx, action.Command, actionable.Env)
	}

	if err != nil {
		return errors.ActionExecutionError(action.Name, err)
	}

	return nil
}

// executeFilterCommand executes a filter command and returns success status.
func (to *TaskOpen) executeFilterCommand(ctx context.Context, command string, env map[string]string) (bool, error) {
	// Expand environment variables in command
	expandedCommand := to.expandCommand(command, env)

	// Parse command and arguments
	parts := strings.Fields(expandedCommand)
	if len(parts) == 0 {
		return false, errors.New(errors.ActionInvalid, "Empty filter command")
	}

	result, err := to.executor.Execute(ctx, parts[0], parts[1:], &exec.ExecutionOptions{
		Environment: env,
		Timeout:     10 * time.Second, // Shorter timeout for filters
	})

	if err != nil {
		return false, err
	}

	return result.ExitCode == 0, nil
}

// executeMainCommand executes the main action command.
func (to *TaskOpen) executeMainCommand(ctx context.Context, command string, env map[string]string) error {
	// Expand environment variables in command
	expandedCommand := to.expandCommand(command, env)

	// Parse command and arguments
	parts := strings.Fields(expandedCommand)
	if len(parts) == 0 {
		return errors.New(errors.ActionInvalid, "Empty command")
	}

	_, err := to.executor.Execute(ctx, parts[0], parts[1:], &exec.ExecutionOptions{
		Environment: env,
		Timeout:     60 * time.Second, // Longer timeout for main commands
	})

	return err
}

// executeInlineCommand executes an inline command and captures output.
func (to *TaskOpen) executeInlineCommand(ctx context.Context, command string, env map[string]string) error {
	// Expand environment variables in command
	expandedCommand := to.expandCommand(command, env)

	// Parse command and arguments
	parts := strings.Fields(expandedCommand)
	if len(parts) == 0 {
		return errors.New(errors.ActionInvalid, "Empty inline command")
	}

	result, err := to.executor.Execute(ctx, parts[0], parts[1:], &exec.ExecutionOptions{
		Environment:   env,
		CaptureOutput: true,
		Timeout:       30 * time.Second,
	})

	if err != nil {
		return err
	}

	if result.ExitCode != 0 {
		return errors.New(errors.ActionExecution, "Inline command failed").
			WithDetails(fmt.Sprintf("Exit code: %d, stderr: %s", result.ExitCode, result.Stderr))
	}

	// Output could be used for further processing
	// For now, we just ensure the command succeeded
	return nil
}

// expandCommand expands environment variables in a command string.
func (to *TaskOpen) expandCommand(command string, env map[string]string) string {
	result := command

	// Replace environment variables
	for key, value := range env {
		placeholder := fmt.Sprintf("$%s", key)
		result = strings.ReplaceAll(result, placeholder, value)

		// Also support ${VAR} syntax
		placeholder = fmt.Sprintf("${%s}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}

	return result
}

// GetVersion returns the taskwarrior version.
func (to *TaskOpen) GetVersion(ctx context.Context) (string, error) {
	return to.taskClient.Version(ctx)
}

// GetCurrentContext returns the current taskwarrior context.
func (to *TaskOpen) GetCurrentContext(ctx context.Context) (string, error) {
	return to.taskClient.CurrentContext(ctx)
}
