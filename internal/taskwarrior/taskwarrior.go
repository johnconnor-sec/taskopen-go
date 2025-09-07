// Package taskwarrior provides integration with Taskwarrior task management system.
package taskwarrior

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
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

// CurrentContext returns the currently active Taskwarrior context.
func (c *Client) CurrentContext(ctx context.Context) (string, error) {
	args := append(c.taskArgs, "context", "show")

	result, err := c.executor.Execute(ctx, c.taskBinary, args, nil)
	if err != nil {
		return "", errors.Wrap(err, errors.TaskwarriorQuery, "Failed to get current context")
	}

	if result.ExitCode != 0 {
		return "", errors.New(errors.TaskwarriorQuery, "Failed to get current context").
			WithDetails(fmt.Sprintf("Exit code: %d, stderr: %s", result.ExitCode, result.Stderr))
	}

	// Parse context from output
	contextRegex1 := regexp.MustCompile(`.*with filter '(.*)' is currently applied\.$`)
	contextRegex2 := regexp.MustCompile(`.*read filter: '(.*)'$`)

	for line := range strings.SplitSeq(result.Stdout, "\n") {
		if matches := contextRegex1.FindStringSubmatch(line); matches != nil {
			return fmt.Sprintf("\\(%s\\)", matches[1]), nil
		}
		if matches := contextRegex2.FindStringSubmatch(line); matches != nil {
			return fmt.Sprintf("\\(%s\\)", matches[1]), nil
		}
	}

	return "", nil
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

// ExportStream retrieves tasks using streaming JSON parser for large datasets.
func (c *Client) ExportStream(ctx context.Context, filters []string) (<-chan Task, <-chan error) {
	taskChan := make(chan Task, 100)
	errorChan := make(chan error, 1)

	go func() {
		defer close(taskChan)
		defer close(errorChan)

		args := append(c.taskArgs, filters...)
		args = append(args, "export")

		// Stream output line by line
		outputChan, execErrorChan := c.executor.ExecuteStream(ctx, c.taskBinary, args, nil)

		var jsonBuffer strings.Builder
		var bracketCount int
		var inString bool
		var escaped bool

		for {
			select {
			case line, ok := <-outputChan:
				if !ok {
					// Process any remaining JSON in buffer
					if jsonBuffer.Len() > 0 {
						tasks, err := c.parseTasksJSON(jsonBuffer.String())
						if err != nil {
							errorChan <- err
							return
						}

						for _, task := range tasks {
							select {
							case taskChan <- task:
							case <-ctx.Done():
								return
							}
						}
					}
					return
				}

				// Remove prefix if present (from streaming)
				if after, ok0 := strings.CutPrefix(line, "[stdout] "); ok0 {
					line = after
				}

				// Buffer the line for JSON parsing
				jsonBuffer.WriteString(line)
				jsonBuffer.WriteRune('\n')

				// Simple JSON bracket counting to detect complete JSON objects
				for _, char := range line {
					switch char {
					case '"':
						if !escaped {
							inString = !inString
						}
						escaped = false
					case '\\':
						escaped = !escaped && inString
					case '[':
						if !inString {
							bracketCount++
						}
						escaped = false
					case ']':
						if !inString {
							bracketCount--
							if bracketCount == 0 {
								// Complete JSON array, parse it
								tasks, err := c.parseTasksJSON(jsonBuffer.String())
								if err != nil {
									errorChan <- err
									return
								}

								// Send tasks to channel
								for _, task := range tasks {
									select {
									case taskChan <- task:
									case <-ctx.Done():
										return
									}
								}

								// Reset buffer
								jsonBuffer.Reset()
							}
						}
						escaped = false
					default:
						escaped = false
					}
				}

			case err := <-execErrorChan:
				if err != nil {
					errorChan <- errors.Wrap(err, errors.TaskwarriorQuery, "Failed to stream task export")
					return
				}

			case <-ctx.Done():
				errorChan <- ctx.Err()
				return
			}
		}
	}()

	return taskChan, errorChan
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

// GetTask retrieves a specific task by UUID.
func (c *Client) GetTask(ctx context.Context, uuid string) (*Task, error) {
	tasks, err := c.Export(ctx, []string{fmt.Sprintf("uuid:%s", uuid)})
	if err != nil {
		return nil, err
	}

	if len(tasks) == 0 {
		return nil, errors.New(errors.TaskwarriorQuery, "Task not found").
			WithDetails(fmt.Sprintf("UUID: %s", uuid)).
			WithSuggestion("Verify the task UUID is correct")
	}

	return &tasks[0], nil
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

// CheckTaskwarrior verifies that Taskwarrior is available and functional.
func CheckTaskwarrior(ctx context.Context, taskBinary string) error {
	client := NewClient(taskBinary, []string{}, 10*time.Second)

	version, err := client.Version(ctx)
	if err != nil {
		return err
	}

	if version == "" {
		return errors.New(errors.TaskwarriorNotFound, "Taskwarrior version is empty").
			WithSuggestion("Ensure Taskwarrior is properly installed")
	}

	// Validate version format (should be like "3.4.1" or "2.6.0")
	versionPattern := `^\d+\.\d+(\.\d+)?$`
	matched, _ := regexp.MatchString(versionPattern, version)
	if !matched {
		return errors.New(errors.TaskwarriorNotFound, "Invalid Taskwarrior version format").
			WithDetails(fmt.Sprintf("Got version: %s", version)).
			WithSuggestion("Ensure a proper Taskwarrior binary is configured")
	}

	return nil
}

// min returns the minimum of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
