// Package core - Built-in command implementations
package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/johnconnor-sec/taskopen-go/internal/errors"
	"github.com/johnconnor-sec/taskopen-go/internal/exec"
	"github.com/johnconnor-sec/taskopen-go/internal/output"
)

// BuiltinHandler handles built-in commands
type BuiltinHandler struct {
	executor  *exec.Executor
	formatter *output.Formatter
	logger    *output.Logger
}

// NewBuiltinHandler creates a new builtin command handler
func NewBuiltinHandler(executor *exec.Executor, formatter *output.Formatter, logger *output.Logger) *BuiltinHandler {
	return &BuiltinHandler{
		executor:  executor,
		formatter: formatter,
		logger:    logger,
	}
}

// IsBuiltinCommand checks if a command is a built-in command
func (bh *BuiltinHandler) IsBuiltinCommand(command string) bool {
	// Parse the command to check for built-in commands
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return false
	}

	// Check the command name (first field)
	commandName := fields[0]
	return commandName == "editnote"
}

// ExecuteBuiltinCommand executes a built-in command
func (bh *BuiltinHandler) ExecuteBuiltinCommand(ctx context.Context, command string, env map[string]string) error {
	args, err := bh.parseShellCommand(command)
	if err != nil {
		return errors.Wrap(err, errors.ActionExecution, "Failed to parse command")
	}

	if len(args) == 0 {
		return errors.New(errors.ActionExecution, "Empty command")
	}

	commandName := args[0]
	cmdArgs := args[1:]

	switch commandName {
	case "editnote":
		return bh.executeEditNote(ctx, cmdArgs, env)
	default:
		return errors.New(errors.ActionExecution, "Unknown built-in command: "+commandName)
	}
}

// executeEditNote implements the editnote functionality
// Usage: editnote <file-path> <description> <uuid>
func (bh *BuiltinHandler) executeEditNote(ctx context.Context, args []string, env map[string]string) error {
	if len(args) != 3 {
		return errors.New(errors.ActionExecution, "editnote requires exactly 3 arguments: <file-path> <description> <uuid>")
	}

	filePath := bh.expandEnvironmentVars(args[0], env)
	description := bh.expandEnvironmentVars(args[1], env)
	uuid := bh.expandEnvironmentVars(args[2], env)

	bh.logger.Debug("editnote", map[string]any{
		"file_path":   filePath,
		"description": description,
		"uuid":        uuid,
	})

	// Expand tilde in file path
	if strings.HasPrefix(filePath, "~") {
		home := os.Getenv("HOME")
		if home != "" {
			filePath = strings.Replace(filePath, "~", home, 1)
		}
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return errors.Wrap(err, errors.ActionExecution, "Failed to create note directory").
			WithDetails(fmt.Sprintf("Directory: %s", dir)).
			WithSuggestion("Check file permissions")
	}

	// Check if file exists, if not create it with header
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		header := fmt.Sprintf("* [ ] %s  #%s\n", description, uuid)
		if err := os.WriteFile(filePath, []byte(header), 0644); err != nil {
			return errors.Wrap(err, errors.ActionExecution, "Failed to create note file").
				WithDetails(fmt.Sprintf("File: %s", filePath)).
				WithSuggestion("Check file permissions and disk space")
		}
		bh.formatter.Success("Created new note: %s", filePath)
	} else if err != nil {
		return errors.Wrap(err, errors.ActionExecution, "Failed to check note file").
			WithDetails(fmt.Sprintf("File: %s", filePath))
	} else {
		bh.logger.Debug("Note file already exists", map[string]any{"file": filePath})
	}

	// Open the file with the configured editor
	editor := env["EDITOR"]
	if editor == "" {
		editor = "vim" // fallback to vim
	}

	bh.formatter.Info("Opening note with %s: %s", editor, filePath)

	result, err := bh.executor.Execute(ctx, editor, []string{filePath}, &exec.ExecutionOptions{
		Environment: env,
	})
	if err != nil {
		return errors.Wrap(err, errors.ActionExecution, "Failed to open editor").
			WithDetails(fmt.Sprintf("Editor: %s, File: %s", editor, filePath)).
			WithSuggestions([]string{
				"Check that the editor is installed and in PATH",
				"Verify the EDITOR environment variable is set correctly",
				"Try setting editor in taskopen configuration",
			})
	}

	if result.ExitCode != 0 {
		return errors.New(errors.ActionExecution, "Editor exited with non-zero code").
			WithDetails(fmt.Sprintf("Exit code: %d, Editor: %s", result.ExitCode, editor))
	}

	bh.formatter.Success("Note editing completed")
	return nil
}

// parseShellCommand parses a shell command string into arguments, handling quotes
func (bh *BuiltinHandler) parseShellCommand(command string) ([]string, error) {
	var args []string
	var current strings.Builder
	inQuotes := false
	quoteChar := byte(0)
	escaped := false

	for i := 0; i < len(command); i++ {
		char := command[i]

		if escaped {
			current.WriteByte(char)
			escaped = false
			continue
		}

		if char == '\\' {
			escaped = true
			continue
		}

		if !inQuotes {
			if char == '"' || char == '\'' {
				inQuotes = true
				quoteChar = char
				continue
			}
			if char == ' ' || char == '\t' {
				if current.Len() > 0 {
					args = append(args, current.String())
					current.Reset()
				}
				continue
			}
		} else {
			if char == quoteChar {
				inQuotes = false
				quoteChar = 0
				continue
			}
		}

		current.WriteByte(char)
	}

	if current.Len() > 0 {
		args = append(args, current.String())
	}

	if inQuotes {
		return nil, fmt.Errorf("unclosed quote in command: %s", command)
	}

	return args, nil
}

// expandEnvironmentVars expands environment variables in a string
func (bh *BuiltinHandler) expandEnvironmentVars(text string, env map[string]string) string {
	result := text
	for key, value := range env {
		result = strings.ReplaceAll(result, "$"+key, value)
		result = strings.ReplaceAll(result, "${"+key+"}", value)
	}
	return result
}
