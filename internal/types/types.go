// Package types provides core data structures for taskopen with validation support.
package types

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// Action represents a taskopen action configuration.
// Ported from Nim Action type with validation tags.
type Action struct {
	Name          string   `json:"name" validate:"required,min=1" yaml:"name"`
	Target        string   `json:"target" validate:"required" yaml:"target"`
	Regex         string   `json:"regex" validate:"regex_pattern" yaml:"regex"`
	LabelRegex    string   `json:"labelregex" validate:"regex_pattern" yaml:"labelregex"`
	Command       string   `json:"command" validate:"required,min=1" yaml:"command"`
	Modes         []string `json:"modes" yaml:"modes"`
	FilterCommand string   `json:"filtercommand" yaml:"filtercommand"`
	InlineCommand string   `json:"inlinecommand" yaml:"inlinecommand"`
}

// Actionable represents an action that can be executed on a task.
// Ported from Nim Actionable type.
type Actionable struct {
	Text   string            `json:"text" validate:"required" yaml:"text"`
	Task   json.RawMessage   `json:"task" validate:"required" yaml:"task"`
	Entry  string            `json:"entry" validate:"required" yaml:"entry"`
	Action Action            `json:"action" validate:"required" yaml:"action"`
	Env    map[string]string `json:"env" yaml:"env"`
}

// ValidationError represents a validation error with context.
type ValidationError struct {
	Field   string `json:"field"`
	Value   string `json:"value"`
	Message string `json:"message"`
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation failed for field '%s' with value '%s': %s", e.Field, e.Value, e.Message)
}

// Validate performs validation on an Action struct.
func (a *Action) Validate() error {
	var errors []ValidationError

	// Required field validations
	if strings.TrimSpace(a.Name) == "" {
		errors = append(errors, ValidationError{
			Field:   "name",
			Value:   a.Name,
			Message: "action name is required and cannot be empty",
		})
	}

	if strings.TrimSpace(a.Target) == "" {
		errors = append(errors, ValidationError{
			Field:   "target",
			Value:   a.Target,
			Message: "action target is required and cannot be empty",
		})
	}

	if strings.TrimSpace(a.Command) == "" {
		errors = append(errors, ValidationError{
			Field:   "command",
			Value:   a.Command,
			Message: "action command is required and cannot be empty",
		})
	}

	// Regex pattern validations
	if a.Regex != "" {
		if _, err := regexp.Compile(a.Regex); err != nil {
			errors = append(errors, ValidationError{
				Field:   "regex",
				Value:   a.Regex,
				Message: fmt.Sprintf("invalid regex pattern: %v", err),
			})
		}
	}

	if a.LabelRegex != "" {
		if _, err := regexp.Compile(a.LabelRegex); err != nil {
			errors = append(errors, ValidationError{
				Field:   "labelregex",
				Value:   a.LabelRegex,
				Message: fmt.Sprintf("invalid label regex pattern: %v", err),
			})
		}
	}

	if len(errors) > 0 {
		return &ValidationErrors{Errors: errors}
	}

	return nil
}

// Validate performs validation on an Actionable struct.
func (a *Actionable) Validate() error {
	var errors []ValidationError

	if strings.TrimSpace(a.Text) == "" {
		errors = append(errors, ValidationError{
			Field:   "text",
			Value:   a.Text,
			Message: "actionable text is required and cannot be empty",
		})
	}

	if strings.TrimSpace(a.Entry) == "" {
		errors = append(errors, ValidationError{
			Field:   "entry",
			Value:   a.Entry,
			Message: "actionable entry is required and cannot be empty",
		})
	}

	if len(a.Task) == 0 {
		errors = append(errors, ValidationError{
			Field:   "task",
			Value:   string(a.Task),
			Message: "task data is required and cannot be empty",
		})
	}

	// Validate the embedded Action
	if err := a.Action.Validate(); err != nil {
		if validationErrs, ok := err.(*ValidationErrors); ok {
			errors = append(errors, validationErrs.Errors...)
		} else {
			errors = append(errors, ValidationError{
				Field:   "action",
				Value:   "",
				Message: fmt.Sprintf("action validation failed: %v", err),
			})
		}
	}

	if len(errors) > 0 {
		return &ValidationErrors{Errors: errors}
	}

	return nil
}

// ValidationErrors holds multiple validation errors.
type ValidationErrors struct {
	Errors []ValidationError `json:"errors"`
}

func (e *ValidationErrors) Error() string {
	var messages []string
	for _, err := range e.Errors {
		messages = append(messages, err.Error())
	}
	return fmt.Sprintf("validation failed:\n  - %s", strings.Join(messages, "\n  - "))
}
