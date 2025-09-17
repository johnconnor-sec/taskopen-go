// Package taskwarrior provides integration with Taskwarrior task management system.
package taskwarrior

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/johnconnor-sec/taskopen-go/internal/errors"
	"github.com/johnconnor-sec/taskopen-go/internal/exec"
)

// DefaultArgs are the default arguments passed to taskwarrior commands.
var DefaultArgs = []string{
	"rc.verbose=blank,label,edit",
	"rc.json.array=on",
	"rc.gc=off",
}

// Task represents a Taskwarrior task.
type Task struct {
	ID          int          `json:"id,omitempty"`
	UUID        string       `json:"uuid"`
	Description string       `json:"description"`
	Status      string       `json:"status"`
	Project     string       `json:"project,omitempty"`
	Priority    string       `json:"priority,omitempty"`
	Tags        []string     `json:"tags,omitempty"`
	Annotations []Annotation `json:"annotations,omitempty"`
	Urgency     float64      `json:"urgency,omitempty"`
	Due         *time.Time   `json:"due,omitempty"`
	Created     *time.Time   `json:"entry,omitempty"`
	Modified    *time.Time   `json:"modified,omitempty"`

	// Raw JSON for additional fields
	Raw json.RawMessage `json:"-"`
}

// Annotation represents a task annotation.
type Annotation struct {
	Entry       time.Time `json:"entry"`
	Description string    `json:"description"`
}

// Context represents a Taskwarrior context.
type Context struct {
	Name   string `json:"name"`
	Filter string `json:"filter"`
}

// Client provides access to Taskwarrior functionality.
type Client struct {
	executor   *exec.Executor
	taskBinary string
	taskArgs   []string
	timeout    time.Duration
}

// NewClient creates a new Taskwarrior client.
func NewClient(taskBinary string, taskArgs []string, timeout time.Duration) *Client {
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	// Configure executor with appropriate options
	execOptions := exec.ExecutionOptions{
		Timeout:       timeout,
		CaptureOutput: true,
		Retry: exec.RetryOptions{
			MaxAttempts:       3,
			BaseDelay:         100 * time.Millisecond,
			MaxDelay:          2 * time.Second,
			BackoffMultiplier: 2.0,
			RetryOnExitCodes:  []int{}, // Don't retry on specific exit codes for now
		},
		Sandbox: exec.SandboxOptions{
			// Taskwarrior is generally safe, but we can add restrictions if needed
			MaxMemoryMB: 512, // Reasonable limit
		},
	}

	return &Client{
		executor:   exec.New(execOptions),
		taskBinary: taskBinary,
		taskArgs:   append(DefaultArgs, taskArgs...),
		timeout:    timeout,
	}
}

// Version returns the Taskwarrior version.
func (c *Client) Version(ctx context.Context) (string, error) {
	args := append(c.taskArgs, "_version")

	result, err := c.executor.Execute(ctx, c.taskBinary, args, nil)
	if err != nil {
		return "", errors.TaskwarriorNotFoundError()
	}

	if result.ExitCode != 0 {
		return "", errors.New(errors.TaskwarriorQuery, "Failed to get Taskwarrior version").
			WithDetails(fmt.Sprintf("Exit code: %d, stderr: %s", result.ExitCode, result.Stderr)).
			WithSuggestions([]string{
				"Ensure Taskwarrior is installed and accessible",
				"Check PATH environment variable",
				"Run 'task --version' manually to verify installation",
			})
	}

	return strings.TrimSpace(result.Stdout), nil
}

// Export retrieves tasks in JSON format using streaming for large datasets.
func (c *Client) Export(ctx context.Context, filters []string) ([]Task, error) {
	args := append(c.taskArgs, filters...)
	args = append(args, "export")

	result, err := c.executor.Execute(ctx, c.taskBinary, args, nil)
	if err != nil {
		return nil, errors.Wrap(err, errors.TaskwarriorQuery, "Failed to export tasks")
	}

	if result.ExitCode != 0 {
		return nil, errors.New(errors.TaskwarriorQuery, "Taskwarrior export failed").
			WithDetails(fmt.Sprintf("Exit code: %d, stderr: %s", result.ExitCode, result.Stderr)).
			WithSuggestions([]string{
				"Check task filter syntax",
				"Ensure no tasks are locked",
				"Verify Taskwarrior configuration",
			})
	}

	return c.parseTasksJSON(result.Stdout)
}

// Query executes a Taskwarrior query and returns matching tasks.
func (c *Client) Query(ctx context.Context, filters []string) ([]Task, error) {
	// Add status:pending by default if no status filter provided
	hasStatusFilter := false
	for _, filter := range filters {
		if strings.Contains(filter, "status:") || strings.Contains(filter, "+PENDING") || strings.Contains(filter, "+COMPLETED") {
			hasStatusFilter = true
			break
		}
	}

	if !hasStatusFilter {
		filters = append(filters, "+PENDING")
	}

	return c.Export(ctx, filters)
}

// parseTasksJSON parses JSON output from Taskwarrior into Task structs.
func (c *Client) parseTasksJSON(jsonData string) ([]Task, error) {
	jsonData = strings.TrimSpace(jsonData)
	if jsonData == "" {
		return []Task{}, nil
	}

	// Parse as array of raw JSON messages first to preserve all fields
	var rawTasks []json.RawMessage
	if err := json.Unmarshal([]byte(jsonData), &rawTasks); err != nil {
		// Try parsing as single task (not array)
		var rawTask json.RawMessage
		if err2 := json.Unmarshal([]byte(jsonData), &rawTask); err2 != nil {
			return nil, errors.Wrap(err, errors.TaskwarriorQuery, "Failed to parse task JSON").
				WithDetails(fmt.Sprintf("JSON: %s", jsonData[:min(200, len(jsonData))]))
		}
		rawTasks = []json.RawMessage{rawTask}
	}

	tasks := make([]Task, len(rawTasks))
	for i, rawTask := range rawTasks {
		var task Task
		if err := json.Unmarshal(rawTask, &task); err != nil {
			return nil, errors.Wrap(err, errors.TaskwarriorQuery, "Failed to parse task").
				WithDetails(fmt.Sprintf("Task %d in JSON array", i+1))
		}

		// Store raw JSON for access to additional fields
		task.Raw = rawTask
		tasks[i] = task
	}

	return tasks, nil
}

// min returns the minimum of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
