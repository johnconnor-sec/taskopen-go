package errors

import (
	"fmt"
	"strings"
	"testing"
)

func TestTaskopenError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *TaskopenError
		contains []string
	}{
		{
			name: "simple error",
			err: &TaskopenError{
				Type:    ConfigInvalid,
				Message: "Configuration is invalid",
			},
			contains: []string{"Configuration is invalid"},
		},
		{
			name: "error with details",
			err: &TaskopenError{
				Type:    ConfigInvalid,
				Message: "Configuration is invalid",
				Details: "Missing required field: actions",
			},
			contains: []string{"Configuration is invalid", "Details: Missing required field: actions"},
		},
		{
			name: "error with suggestions",
			err: &TaskopenError{
				Type:        ConfigInvalid,
				Message:     "Configuration is invalid",
				Suggestions: []string{"Check syntax", "Verify required fields"},
			},
			contains: []string{"Configuration is invalid", "Suggestions:", "Check syntax", "Verify required fields"},
		},
		{
			name: "comprehensive error",
			err: &TaskopenError{
				Type:        ActionExecution,
				Message:     "Failed to execute action",
				Details:     "Command not found: missing-command",
				Suggestions: []string{"Install the command", "Check PATH"},
			},
			contains: []string{
				"Failed to execute action",
				"Details: Command not found: missing-command",
				"Suggestions:",
				"Install the command",
				"Check PATH",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errorStr := tt.err.Error()
			for _, expected := range tt.contains {
				if !strings.Contains(errorStr, expected) {
					t.Errorf("Error string %q does not contain expected text %q", errorStr, expected)
				}
			}
		})
	}
}

func TestTaskopenError_Unwrap(t *testing.T) {
	originalErr := fmt.Errorf("original error")
	wrappedErr := &TaskopenError{
		Type:    InternalError,
		Message: "Wrapped error",
		Cause:   originalErr,
	}

	unwrapped := wrappedErr.Unwrap()
	if unwrapped != originalErr {
		t.Errorf("Unwrap() returned %v, want %v", unwrapped, originalErr)
	}
}

func TestNew(t *testing.T) {
	err := New(ConfigNotFound, "Config file missing")

	if err.Type != ConfigNotFound {
		t.Errorf("New() type = %v, want %v", err.Type, ConfigNotFound)
	}

	if err.Message != "Config file missing" {
		t.Errorf("New() message = %v, want %v", err.Message, "Config file missing")
	}

	if err.Cause != nil {
		t.Errorf("New() cause = %v, want nil", err.Cause)
	}
}

func TestWrap(t *testing.T) {
	originalErr := fmt.Errorf("original error")
	wrappedErr := Wrap(originalErr, ConfigInvalid, "Wrapped message")

	if wrappedErr.Type != ConfigInvalid {
		t.Errorf("Wrap() type = %v, want %v", wrappedErr.Type, ConfigInvalid)
	}

	if wrappedErr.Message != "Wrapped message" {
		t.Errorf("Wrap() message = %v, want %v", wrappedErr.Message, "Wrapped message")
	}

	if wrappedErr.Cause != originalErr {
		t.Errorf("Wrap() cause = %v, want %v", wrappedErr.Cause, originalErr)
	}
}

func TestTaskopenError_WithDetails(t *testing.T) {
	err := New(ConfigInvalid, "Invalid config")
	err = err.WithDetails("Missing actions section")

	if err.Details != "Missing actions section" {
		t.Errorf("WithDetails() details = %v, want %v", err.Details, "Missing actions section")
	}
}

func TestTaskopenError_WithSuggestion(t *testing.T) {
	err := New(ConfigInvalid, "Invalid config")
	err = err.WithSuggestion("Check syntax")

	if len(err.Suggestions) != 1 {
		t.Errorf("WithSuggestion() suggestions length = %v, want 1", len(err.Suggestions))
	}

	if err.Suggestions[0] != "Check syntax" {
		t.Errorf("WithSuggestion() suggestion = %v, want %v", err.Suggestions[0], "Check syntax")
	}
}

func TestTaskopenError_WithSuggestions(t *testing.T) {
	err := New(ConfigInvalid, "Invalid config")
	suggestions := []string{"Check syntax", "Verify fields", "Run init"}
	err = err.WithSuggestions(suggestions)

	if len(err.Suggestions) != 3 {
		t.Errorf("WithSuggestions() suggestions length = %v, want 3", len(err.Suggestions))
	}

	for i, expected := range suggestions {
		if err.Suggestions[i] != expected {
			t.Errorf("WithSuggestions() suggestion[%d] = %v, want %v", i, err.Suggestions[i], expected)
		}
	}
}

func TestConfigNotFoundError(t *testing.T) {
	path := "/home/user/.config/taskopen/config.yml"
	err := ConfigNotFoundError(path)

	if err.Type != ConfigNotFound {
		t.Errorf("ConfigNotFoundError() type = %v, want %v", err.Type, ConfigNotFound)
	}

	errorStr := err.Error()
	if !strings.Contains(errorStr, "Configuration file not found") {
		t.Errorf("ConfigNotFoundError() should contain main message")
	}

	if !strings.Contains(errorStr, path) {
		t.Errorf("ConfigNotFoundError() should contain path")
	}

	if !strings.Contains(errorStr, "taskopen config init") {
		t.Errorf("ConfigNotFoundError() should contain helpful suggestions")
	}
}

func TestTaskwarriorNotFoundError(t *testing.T) {
	err := TaskwarriorNotFoundError()

	if err.Type != TaskwarriorNotFound {
		t.Errorf("TaskwarriorNotFoundError() type = %v, want %v", err.Type, TaskwarriorNotFound)
	}

	errorStr := err.Error()
	if !strings.Contains(errorStr, "Taskwarrior not found") {
		t.Errorf("TaskwarriorNotFoundError() should contain main message")
	}

	if !strings.Contains(errorStr, "sudo apt-get install taskwarrior") {
		t.Errorf("TaskwarriorNotFoundError() should contain installation suggestions")
	}
}

func TestValidationError(t *testing.T) {
	err := ValidationError("name", "invalid-name!", "contains invalid characters")

	if err.Type != ValidationFailed {
		t.Errorf("ValidationError() type = %v, want %v", err.Type, ValidationFailed)
	}

	errorStr := err.Error()
	if !strings.Contains(errorStr, "name") {
		t.Errorf("ValidationError() should contain field name")
	}

	if !strings.Contains(errorStr, "invalid-name!") {
		t.Errorf("ValidationError() should contain value")
	}

	if !strings.Contains(errorStr, "contains invalid characters") {
		t.Errorf("ValidationError() should contain reason")
	}
}

func TestActionNotFoundError(t *testing.T) {
	err := ActionNotFoundError("missing-action")

	if err.Type != ActionNotFound {
		t.Errorf("ActionNotFoundError() type = %v, want %v", err.Type, ActionNotFound)
	}

	errorStr := err.Error()
	if !strings.Contains(errorStr, "No matching actions found") {
		t.Errorf("ActionNotFoundError() should contain main message")
	}

	if !strings.Contains(errorStr, "missing-action") {
		t.Errorf("ActionNotFoundError() should contain query")
	}
}

func TestActionExecutionError(t *testing.T) {
	originalErr := fmt.Errorf("command not found")
	err := ActionExecutionError("test-action", originalErr)

	if err.Type != ActionExecution {
		t.Errorf("ActionExecutionError() type = %v, want %v", err.Type, ActionExecution)
	}

	if err.Cause != originalErr {
		t.Errorf("ActionExecutionError() should wrap original error")
	}

	errorStr := err.Error()
	if !strings.Contains(errorStr, "test-action") {
		t.Errorf("ActionExecutionError() should contain action name")
	}
}

func TestPermissionDeniedError(t *testing.T) {
	err := PermissionDeniedError("/protected/file", "read")

	if err.Type != PermissionDenied {
		t.Errorf("PermissionDeniedError() type = %v, want %v", err.Type, PermissionDenied)
	}

	errorStr := err.Error()
	if !strings.Contains(errorStr, "/protected/file") {
		t.Errorf("PermissionDeniedError() should contain path")
	}

	if !strings.Contains(errorStr, "read") {
		t.Errorf("PermissionDeniedError() should contain operation")
	}
}

func TestIsType(t *testing.T) {
	err := New(ConfigInvalid, "Test error")

	if !IsType(err, ConfigInvalid) {
		t.Errorf("IsType() should return true for matching error type")
	}

	if IsType(err, TaskwarriorNotFound) {
		t.Errorf("IsType() should return false for non-matching error type")
	}

	genericErr := fmt.Errorf("generic error")
	if IsType(genericErr, ConfigInvalid) {
		t.Errorf("IsType() should return false for non-TaskopenError")
	}
}

func TestGetType(t *testing.T) {
	err := New(ConfigInvalid, "Test error")

	if GetType(err) != ConfigInvalid {
		t.Errorf("GetType() should return correct type for TaskopenError")
	}

	genericErr := fmt.Errorf("generic error")
	if GetType(genericErr) != InternalError {
		t.Errorf("GetType() should return InternalError for non-TaskopenError")
	}
}
