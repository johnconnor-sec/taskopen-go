// Package core implements the main taskopen business logic
package core

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/johnconnor-sec/taskopen-go/internal/config"
	"github.com/johnconnor-sec/taskopen-go/internal/errors"
	"github.com/johnconnor-sec/taskopen-go/internal/exec"
	"github.com/johnconnor-sec/taskopen-go/internal/output"
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
		if tp.config.General.NoAnnotationHook != "" && len(tasks) == 1 {
			tp.formatter.Warning("No actionable items found")
			taskEnv := tp.buildEnvironment(tasks[0])
			result, err := tp.executor.Execute(ctx, "sh", []string{"-c", tp.config.General.NoAnnotationHook}, &exec.ExecutionOptions{Environment: taskEnv})
			if err != nil {
				tp.logger.Error("Failed executing no_annotation_hook", map[string]any{"command": tp.config.General.NoAnnotationHook, "error": err.Error()})
				return errors.Wrap(err, errors.ActionExecution, "Failed to execute no_annotation_hook")
			}

			if result.ExitCode != 0 {
				tp.logger.Error("no_annotation_hook exited with non-zero code", map[string]any{"command": tp.config.General.NoAnnotationHook, "exit_code": result.ExitCode})
				return errors.New(errors.ActionExecution, fmt.Sprintf("no_annotation_hook failed with exit code %d", result.ExitCode))
			}
			return nil
		}
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

// executeFilter runs a filter command and returns whether it passed
func (tp *TaskProcessor) executeFilter(ctx context.Context, command string, env map[string]string) bool {
	expandedCommand := tp.expandEnvironmentVars(command, env)
	result, err := tp.executor.Execute(ctx, "sh", []string{"-c", expandedCommand},
		&exec.ExecutionOptions{Environment: env})
	return err == nil && result.ExitCode == 0
}
