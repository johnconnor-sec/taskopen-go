package core

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/johnconnor-sec/taskopen-go/internal/exec"
)

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
