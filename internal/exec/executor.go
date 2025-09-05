// Package exec provides secure process execution with context cancellation and sandboxing.
package exec

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/johnconnor-sec/taskopen-go/internal/errors"
)

// ExecutionOptions configures process execution behavior.
type ExecutionOptions struct {
	// Timeout for process execution
	Timeout time.Duration

	// Environment variables (if nil, inherits current environment)
	Environment map[string]string

	// Working directory (if empty, uses current directory)
	WorkingDir string

	// Whether to capture stdout
	CaptureOutput bool

	// Whether to stream output line by line
	StreamOutput bool

	// Whether this is an interactive command (no timeout)
	Interactive bool

	// Security sandbox options
	Sandbox SandboxOptions

	// Retry configuration
	Retry RetryOptions
}

// SandboxOptions configures process sandboxing for security.
type SandboxOptions struct {
	// Disable network access (not implemented on all platforms)
	DisableNetwork bool

	// Restrict filesystem access to specific directories
	AllowedPaths []string

	// Maximum memory usage (not implemented on all platforms)
	MaxMemoryMB int64

	// Drop privileges (Unix only)
	DropPrivileges bool
}

// RetryOptions configures retry behavior for process execution.
type RetryOptions struct {
	// Number of retry attempts
	MaxAttempts int

	// Base delay between retries
	BaseDelay time.Duration

	// Maximum delay between retries
	MaxDelay time.Duration

	// Exponential backoff multiplier
	BackoffMultiplier float64

	// Retry on these exit codes
	RetryOnExitCodes []int
}

// ExecutionResult holds the result of process execution.
type ExecutionResult struct {
	// Exit code of the process
	ExitCode int

	// Standard output (if captured)
	Stdout string

	// Standard error (if captured)
	Stderr string

	// Execution duration
	Duration time.Duration

	// Whether process was killed due to timeout
	TimedOut bool

	// Number of retry attempts made
	RetryAttempts int
}

// Executor provides secure process execution capabilities.
type Executor struct {
	// Default options for all executions
	defaultOptions ExecutionOptions
}

// New creates a new Executor with default options.
func New(options ExecutionOptions) *Executor {
	// Set reasonable defaults
	if options.Timeout == 0 {
		options.Timeout = 30 * time.Second
	}

	if options.Retry.MaxAttempts == 0 {
		options.Retry.MaxAttempts = 1
	}

	if options.Retry.BaseDelay == 0 {
		options.Retry.BaseDelay = 100 * time.Millisecond
	}

	if options.Retry.MaxDelay == 0 {
		options.Retry.MaxDelay = 5 * time.Second
	}

	if options.Retry.BackoffMultiplier == 0 {
		options.Retry.BackoffMultiplier = 2.0
	}

	return &Executor{
		defaultOptions: options,
	}
}

// Execute runs a command with the given options.
func (e *Executor) Execute(ctx context.Context, command string, args []string, options *ExecutionOptions) (*ExecutionResult, error) {
	// Merge provided options with defaults
	finalOptions := e.defaultOptions
	if options != nil {
		// Override defaults with provided options
		if options.Timeout != 0 {
			finalOptions.Timeout = options.Timeout
		}
		if options.Environment != nil {
			finalOptions.Environment = options.Environment
		}
		if options.WorkingDir != "" {
			finalOptions.WorkingDir = options.WorkingDir
		}
		if options.CaptureOutput {
			finalOptions.CaptureOutput = options.CaptureOutput
		}
		if options.StreamOutput {
			finalOptions.StreamOutput = options.StreamOutput
		}
		// Merge sandbox options
		if options.Sandbox.DisableNetwork {
			finalOptions.Sandbox.DisableNetwork = options.Sandbox.DisableNetwork
		}
		if len(options.Sandbox.AllowedPaths) > 0 {
			finalOptions.Sandbox.AllowedPaths = options.Sandbox.AllowedPaths
		}
		if options.Sandbox.MaxMemoryMB > 0 {
			finalOptions.Sandbox.MaxMemoryMB = options.Sandbox.MaxMemoryMB
		}
		if options.Sandbox.DropPrivileges {
			finalOptions.Sandbox.DropPrivileges = options.Sandbox.DropPrivileges
		}
		// Merge retry options
		if options.Retry.MaxAttempts > 0 {
			finalOptions.Retry.MaxAttempts = options.Retry.MaxAttempts
		}
		if options.Retry.BaseDelay > 0 {
			finalOptions.Retry.BaseDelay = options.Retry.BaseDelay
		}
		if options.Retry.MaxDelay > 0 {
			finalOptions.Retry.MaxDelay = options.Retry.MaxDelay
		}
		if options.Retry.BackoffMultiplier > 0 {
			finalOptions.Retry.BackoffMultiplier = options.Retry.BackoffMultiplier
		}
		if len(options.Retry.RetryOnExitCodes) > 0 {
			finalOptions.Retry.RetryOnExitCodes = options.Retry.RetryOnExitCodes
		}
	}

	return e.executeWithRetry(ctx, command, args, finalOptions)
}

// ExecuteFilter runs a command and returns true if exit code is 0.
func (e *Executor) ExecuteFilter(ctx context.Context, command string, args []string, options *ExecutionOptions) (bool, error) {
	result, err := e.Execute(ctx, command, args, options)
	if err != nil {
		return false, err
	}

	return result.ExitCode == 0, nil
}

// ExecuteStream runs a command and streams output line by line.
func (e *Executor) ExecuteStream(ctx context.Context, command string, args []string, options *ExecutionOptions) (<-chan string, <-chan error) {
	outputChan := make(chan string, 100)
	errorChan := make(chan error, 1)

	go func() {
		defer close(outputChan)
		defer close(errorChan)

		if options == nil {
			opts := e.defaultOptions
			options = &opts
		}

		// Set streaming mode
		options.StreamOutput = true
		options.CaptureOutput = false

		if err := e.executeStream(ctx, command, args, *options, outputChan); err != nil {
			errorChan <- err
		}
	}()

	return outputChan, errorChan
}

// IsInteractiveEditor determines if a command is an interactive editor
func (e *Executor) IsInteractiveEditor(command string) bool {
	// Extract the first word (executable name) from the command
	parts := strings.Fields(strings.TrimSpace(command))
	if len(parts) == 0 {
		return false
	}

	// Remove path components to get just the executable name (cross-platform)
	executable := strings.ToLower(filepath.Base(parts[0]))

	// Remove common executable extensions (like .exe) for better cross-platform support
	executable = strings.TrimSuffix(executable, ".exe")

	// List of known interactive editors (base names without extensions)
	interactiveEditorsMap := map[string]bool{
		"vim": true, "nvim": true, "vi": true, "nano": true, "emacs": true, "pico": true, "joe": true, "micro": true,
		"code": true, "subl": true, "atom": true, "gedit": true, "kate": true, "mousepad": true, "leafpad": true,
		"ne": true, "mg": true, "zile": true, "jed": true, "mcedit": true, "tilde": true, "kakoune": true, "kak": true,
		"helix": true, "hx": true, "ed": true,
		// Hidden executable variants (common on Unix)
		".vim": true, ".nvim": true, ".nano": true, ".emacs": true,
	}

	return interactiveEditorsMap[executable]
}

// NeedsShell determines if a command requires shell execution
func (e *Executor) NeedsShell(command string) bool {
	// Check for shell features that require shell execution
	shellFeatures := []string{
		"|", "&", ";", "&&", "||", // Pipes and operators
		">", ">>", "<", // Redirections
		"$(", "`", // Command substitution
		"*", "?", "[", // Glob patterns
	}

	for _, feature := range shellFeatures {
		if strings.Contains(command, feature) {
			return true
		}
	}

	// Check for variable assignments (VAR=value command)
	if strings.Contains(command, "=") && !strings.HasPrefix(strings.TrimSpace(command), "=") {
		parts := strings.Fields(strings.TrimSpace(command))
		if len(parts) > 1 {
			// Check if first part looks like a variable assignment
			for _, part := range parts {
				if strings.Contains(part, "=") && !strings.HasPrefix(part, "-") {
					// This looks like VAR=value, need shell
					return true
				}
			}
		}
	}

	return false
}

// ExecuteDirect executes a command directly without shell wrapper for better interactive support
func (e *Executor) ExecuteDirect(ctx context.Context, command string, options *ExecutionOptions) (*ExecutionResult, error) {
	// Parse command into executable and arguments
	cmdParts := strings.Fields(strings.TrimSpace(command))
	if len(cmdParts) == 0 {
		return nil, errors.New(errors.ActionExecution, "Empty command")
	}

	executable := cmdParts[0]
	args := cmdParts[1:]

	// Use direct execution
	finalOptions := e.defaultOptions
	if options != nil {
		finalOptions = e.mergeOptions(finalOptions, *options)
	}

	return e.executeDirectSingle(ctx, executable, args, finalOptions)
}

// executeWithRetry handles retry logic for command execution.
func (e *Executor) executeWithRetry(ctx context.Context, command string, args []string, options ExecutionOptions) (*ExecutionResult, error) {
	var lastResult *ExecutionResult
	var lastError error

	delay := options.Retry.BaseDelay

	for attempt := 0; attempt < options.Retry.MaxAttempts; attempt++ {
		// Add retry delay (except for first attempt)
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
				// Continue with retry
			}

			// Exponential backoff
			delay = min(time.Duration(float64(delay)*options.Retry.BackoffMultiplier), options.Retry.MaxDelay)
		}

		result, err := e.executeSingle(ctx, command, args, options)
		if result != nil {
			result.RetryAttempts = attempt
		}

		// If no error and successful exit code, return success
		if err == nil && result.ExitCode == 0 {
			return result, nil
		}

		lastResult = result
		lastError = err

		// Check if we should retry based on exit code
		if result != nil && len(options.Retry.RetryOnExitCodes) > 0 {
			shouldRetry := slices.Contains(options.Retry.RetryOnExitCodes, result.ExitCode)
			if !shouldRetry {
				break
			}
		}
	}

	// Return the last result and error
	if lastError != nil {
		return lastResult, errors.Wrap(lastError, errors.ActionExecution, "Command execution failed after retries").
			WithDetails(fmt.Sprintf("Command: %s %s", command, strings.Join(args, " "))).
			WithSuggestion("Check command availability and arguments")
	}

	return lastResult, nil
}

// executeSingle executes a command once without retry logic.
func (e *Executor) executeSingle(ctx context.Context, command string, args []string, options ExecutionOptions) (*ExecutionResult, error) {
	startTime := time.Now()

	// Create context with timeout
	execCtx, cancel := context.WithTimeout(ctx, options.Timeout)
	defer cancel()

	// Create command
	cmd := exec.CommandContext(execCtx, command, args...)

	// Set working directory
	if options.WorkingDir != "" {
		cmd.Dir = options.WorkingDir
	}

	// Set environment
	if options.Environment != nil {
		env := make([]string, 0, len(options.Environment))
		for key, value := range options.Environment {
			env = append(env, fmt.Sprintf("%s=%s", key, value))
		}
		cmd.Env = env
	}

	// Apply security sandbox
	if err := e.applySandbox(cmd, options.Sandbox); err != nil {
		return nil, errors.Wrap(err, errors.ActionExecution, "Failed to apply security sandbox")
	}

	result := &ExecutionResult{}

	// Set up output handling
	if options.CaptureOutput {
		var stdout, stderr strings.Builder
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		// Execute command
		err := cmd.Run()

		result.Stdout = stdout.String()
		result.Stderr = stderr.String()
		result.Duration = time.Since(startTime)

		return e.handleCommandResult(cmd, err, result, execCtx)
	} else {
		// Inherit parent streams
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin

		// Execute command
		err := cmd.Run()
		result.Duration = time.Since(startTime)

		return e.handleCommandResult(cmd, err, result, execCtx)
	}
}

// executeStream executes a command and streams output.
func (e *Executor) executeStream(ctx context.Context, command string, args []string, options ExecutionOptions, outputChan chan<- string) error {
	// Create context with timeout
	execCtx, cancel := context.WithTimeout(ctx, options.Timeout)
	defer cancel()

	// Create command
	cmd := exec.CommandContext(execCtx, command, args...)

	// Set working directory and environment
	if options.WorkingDir != "" {
		cmd.Dir = options.WorkingDir
	}

	if options.Environment != nil {
		env := make([]string, 0, len(options.Environment))
		for key, value := range options.Environment {
			env = append(env, fmt.Sprintf("%s=%s", key, value))
		}
		cmd.Env = env
	}

	// Apply security sandbox
	if err := e.applySandbox(cmd, options.Sandbox); err != nil {
		return errors.Wrap(err, errors.ActionExecution, "Failed to apply security sandbox")
	}

	// Set up output pipes
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return errors.Wrap(err, errors.ActionExecution, "Failed to create stdout pipe")
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return errors.Wrap(err, errors.ActionExecution, "Failed to create stderr pipe")
	}

	// Start command
	if err := cmd.Start(); err != nil {
		return errors.Wrap(err, errors.ActionExecution, "Failed to start command")
	}

	// Stream output
	go e.streamReader(stdout, "stdout", outputChan)
	go e.streamReader(stderr, "stderr", outputChan)

	// Wait for command to complete
	return cmd.Wait()
}

// streamReader reads from a pipe and sends lines to output channel.
func (e *Executor) streamReader(reader io.Reader, prefix string, outputChan chan<- string) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		if prefix != "" {
			line = fmt.Sprintf("[%s] %s", prefix, line)
		}

		select {
		case outputChan <- line:
		default:
			// Channel is full, drop the line to prevent blocking
		}
	}
}

// handleCommandResult processes the result of command execution.
func (e *Executor) handleCommandResult(cmd *exec.Cmd, err error, result *ExecutionResult, ctx context.Context) (*ExecutionResult, error) {
	// Check if context was cancelled (timeout or cancellation)
	if ctx.Err() != nil {
		result.TimedOut = ctx.Err() == context.DeadlineExceeded
		if result.TimedOut {
			return result, errors.New(errors.ActionExecution, "Command execution timed out").
				WithDetails(fmt.Sprintf("Timeout: %v", e.defaultOptions.Timeout))
		}
		return result, errors.Wrap(ctx.Err(), errors.ActionExecution, "Command execution cancelled")
	}

	// Get exit code
	if exitError, ok := err.(*exec.ExitError); ok {
		if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
			result.ExitCode = status.ExitStatus()
		} else {
			result.ExitCode = 1
		}
	} else if err != nil {
		return result, errors.Wrap(err, errors.ActionExecution, "Failed to execute command")
	} else {
		result.ExitCode = 0
	}

	return result, nil
}

// applySandbox applies security sandbox restrictions to the command.
func (e *Executor) applySandbox(cmd *exec.Cmd, sandbox SandboxOptions) error {
	// Platform-specific sandboxing implementation would go here
	// For now, we implement basic restrictions that work on most Unix systems

	if sandbox.DropPrivileges {
		// This would need platform-specific implementation
		// On Unix, we could use setuid/setgid syscalls
		// For now, just document the intent
	}

	// Memory limits, network restrictions, etc. would require
	// platform-specific implementations using cgroups, namespaces, etc.

	return nil
}

// mergeOptions merges execution options
func (e *Executor) mergeOptions(base ExecutionOptions, override ExecutionOptions) ExecutionOptions {
	result := base

	if override.Timeout != 0 {
		result.Timeout = override.Timeout
	}
	if override.Environment != nil {
		result.Environment = override.Environment
	}
	if override.WorkingDir != "" {
		result.WorkingDir = override.WorkingDir
	}
	if override.CaptureOutput {
		result.CaptureOutput = override.CaptureOutput
	}
	if override.StreamOutput {
		result.StreamOutput = override.StreamOutput
	}
	if override.Interactive {
		result.Interactive = override.Interactive
	}

	return result
}

// executeDirectSingle executes a command directly without shell wrapper
func (e *Executor) executeDirectSingle(ctx context.Context, command string, args []string, options ExecutionOptions) (*ExecutionResult, error) {
	startTime := time.Now()

	// Create context - for interactive commands, don't use timeout
	var execCtx context.Context
	var cancel context.CancelFunc

	if options.Interactive {
		// For interactive commands, use the parent context without timeout
		execCtx = ctx
		cancel = func() {} // No-op cancel function
	} else {
		// For non-interactive commands, use timeout
		execCtx, cancel = context.WithTimeout(ctx, options.Timeout)
	}
	defer cancel()

	// Create command - execute directly without shell
	cmd := exec.CommandContext(execCtx, command, args...)

	// Set working directory
	if options.WorkingDir != "" {
		cmd.Dir = options.WorkingDir
	}

	// Set environment
	if options.Environment != nil {
		env := make([]string, 0, len(options.Environment))
		for key, value := range options.Environment {
			env = append(env, fmt.Sprintf("%s=%s", key, value))
		}
		cmd.Env = env
	}

	// Apply security sandbox
	if err := e.applySandbox(cmd, options.Sandbox); err != nil {
		return nil, errors.Wrap(err, errors.ActionExecution, "Failed to apply security sandbox")
	}

	result := &ExecutionResult{}

	// Set up output handling - for interactive commands, inherit parent streams
	if options.CaptureOutput {
		var stdout, stderr strings.Builder
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		// Execute command
		err := cmd.Run()

		result.Stdout = stdout.String()
		result.Stderr = stderr.String()
		result.Duration = time.Since(startTime)

		return e.handleCommandResult(cmd, err, result, execCtx)
	} else {
		// Inherit parent streams for interactive programs
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin

		// For interactive commands, set up proper signal handling
		if options.Interactive {
			return e.runInteractiveCommand(cmd, result, execCtx, startTime)
		} else {
			// Execute non-interactive command normally
			err := cmd.Run()
			result.Duration = time.Since(startTime)
			return e.handleCommandResult(cmd, err, result, execCtx)
		}
	}
}

// runInteractiveCommand runs an interactive command with proper signal handling and terminal state management
func (e *Executor) runInteractiveCommand(cmd *exec.Cmd, result *ExecutionResult, ctx context.Context, startTime time.Time) (*ExecutionResult, error) {
	// Start the command
	err := cmd.Start()
	if err != nil {
		return result, errors.Wrap(err, errors.ActionExecution, "Failed to start interactive command")
	}

	// Wait for the command to complete
	err = cmd.Wait()
	result.Duration = time.Since(startTime)

	return e.handleCommandResult(cmd, err, result, ctx)
}
