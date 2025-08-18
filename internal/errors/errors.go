// Package errors provides structured error handling with user-friendly messages.
package errors

import (
	"fmt"
	"strings"
)

// ErrorType represents different categories of errors for better user experience.
type ErrorType string

const (
	// Configuration errors
	ConfigNotFound  ErrorType = "config_not_found"
	ConfigInvalid   ErrorType = "config_invalid"
	ConfigMigration ErrorType = "config_migration"

	// Taskwarrior errors
	TaskwarriorNotFound ErrorType = "taskwarrior_not_found"
	TaskwarriorQuery    ErrorType = "taskwarrior_query"
	TaskwarriorTimeout  ErrorType = "taskwarrior_timeout"

	// Action errors
	ActionNotFound  ErrorType = "action_not_found"
	ActionInvalid   ErrorType = "action_invalid"
	ActionExecution ErrorType = "action_execution"

	// System errors
	PermissionDenied ErrorType = "permission_denied"
	FileNotFound     ErrorType = "file_not_found"
	NetworkError     ErrorType = "network_error"

	// Validation errors
	ValidationFailed ErrorType = "validation_failed"

	// Internal errors
	InternalError ErrorType = "internal_error"
)

// TaskopenError represents a structured error with user-friendly messaging.
type TaskopenError struct {
	Type        ErrorType `json:"type"`
	Message     string    `json:"message"`
	Details     string    `json:"details,omitempty"`
	Suggestions []string  `json:"suggestions,omitempty"`
	Cause       error     `json:"-"`
}

func (e *TaskopenError) Error() string {
	var parts []string

	// Main error message
	parts = append(parts, e.Message)

	// Additional details if available
	if e.Details != "" {
		parts = append(parts, fmt.Sprintf("Details: %s", e.Details))
	}

	// Helpful suggestions
	if len(e.Suggestions) > 0 {
		parts = append(parts, fmt.Sprintf("Suggestions:\n  • %s", strings.Join(e.Suggestions, "\n  • ")))
	}

	return strings.Join(parts, "\n\n")
}

func (e *TaskopenError) Unwrap() error {
	return e.Cause
}

// New creates a new TaskopenError with the given type and message.
func New(errorType ErrorType, message string) *TaskopenError {
	return &TaskopenError{
		Type:    errorType,
		Message: message,
	}
}

// Wrap creates a new TaskopenError that wraps an existing error.
func Wrap(err error, errorType ErrorType, message string) *TaskopenError {
	return &TaskopenError{
		Type:    errorType,
		Message: message,
		Cause:   err,
	}
}

// WithDetails adds detailed information to an error.
func (e *TaskopenError) WithDetails(details string) *TaskopenError {
	e.Details = details
	return e
}

// WithSuggestion adds a helpful suggestion to an error.
func (e *TaskopenError) WithSuggestion(suggestion string) *TaskopenError {
	e.Suggestions = append(e.Suggestions, suggestion)
	return e
}

// WithSuggestions adds multiple helpful suggestions to an error.
func (e *TaskopenError) WithSuggestions(suggestions []string) *TaskopenError {
	e.Suggestions = append(e.Suggestions, suggestions...)
	return e
}

// Common error constructors for frequently encountered issues

// ConfigNotFoundError creates an error for missing configuration.
func ConfigNotFoundError(path string) *TaskopenError {
	return New(ConfigNotFound, "Configuration file not found").
		WithDetails(fmt.Sprintf("Looking for config at: %s", path)).
		WithSuggestions([]string{
			"Run 'taskopen config init' to create a new configuration",
			"Check if the config file exists and is readable",
			"Ensure the config directory has proper permissions",
		})
}

// TaskwarriorNotFoundError creates an error for missing taskwarrior.
func TaskwarriorNotFoundError() *TaskopenError {
	return New(TaskwarriorNotFound, "Taskwarrior not found in PATH").
		WithDetails("The 'task' command is required but not available").
		WithSuggestions([]string{
			"Install taskwarrior: sudo apt-get install taskwarrior (Ubuntu/Debian)",
			"Install taskwarrior: brew install task (macOS)",
			"Ensure taskwarrior is in your PATH",
			"Run 'taskopen diagnostics' to verify installation",
		})
}

// ValidationError creates an error for validation failures.
func ValidationError(field string, value string, reason string) *TaskopenError {
	return New(ValidationFailed, fmt.Sprintf("Validation failed for '%s'", field)).
		WithDetails(fmt.Sprintf("Value '%s' is invalid: %s", value, reason))
}

// ActionNotFoundError creates an error for missing actions.
func ActionNotFoundError(query string) *TaskopenError {
	return New(ActionNotFound, "No matching actions found").
		WithDetails(fmt.Sprintf("No actions matched the query: %s", query)).
		WithSuggestions([]string{
			"Check your action configuration in the config file",
			"Verify that action patterns match your task annotations",
			"Run 'taskopen config init' to create default actions",
			"Use 'taskopen diagnostics' to check action configuration",
		})
}

// ActionExecutionError creates an error for action execution failures.
func ActionExecutionError(actionName string, err error) *TaskopenError {
	return Wrap(err, ActionExecution, fmt.Sprintf("Failed to execute action '%s'", actionName)).
		WithSuggestions([]string{
			"Check that the action command exists and is executable",
			"Verify that required environment variables are set",
			"Ensure the action has proper permissions",
			"Check system logs for additional error details",
		})
}

// PermissionDeniedError creates an error for permission issues.
func PermissionDeniedError(path string, operation string) *TaskopenError {
	return New(PermissionDenied, fmt.Sprintf("Permission denied: cannot %s %s", operation, path)).
		WithSuggestions([]string{
			"Check file/directory permissions",
			"Ensure you have the required access rights",
			"Try running with appropriate privileges if necessary",
		})
}

// IsType checks if an error is of a specific TaskopenError type.
func IsType(err error, errorType ErrorType) bool {
	if taskopenErr, ok := err.(*TaskopenError); ok {
		return taskopenErr.Type == errorType
	}
	return false
}

// GetType returns the ErrorType of a TaskopenError, or InternalError for other errors.
func GetType(err error) ErrorType {
	if taskopenErr, ok := err.(*TaskopenError); ok {
		return taskopenErr.Type
	}
	return InternalError
}
