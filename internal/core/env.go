package core

import (
	"fmt"
	"maps"
	"os"
	"strings"
)

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

// expandEnvironmentVars expands environment variables in a command string
func (tp *TaskProcessor) expandEnvironmentVars(command string, env map[string]string) string {
	result := command
	for key, value := range env {
		result = strings.ReplaceAll(result, "$"+key, value)
		result = strings.ReplaceAll(result, "${"+key+"}", value)
	}
	return result
}
