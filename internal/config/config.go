// Package config provides YAML-based configuration with schema validation and INI migration.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/johnconnor-sec/taskopen-go/internal/errors"
	"github.com/johnconnor-sec/taskopen-go/internal/types"
)

// Config represents the complete taskopen configuration.
type Config struct {
	// General configuration
	General GeneralConfig `yaml:"general" json:"general" validate:"required"`

	// Action definitions
	Actions []types.Action `yaml:"actions" json:"actions" validate:"required,min=1,dive"`

	// CLI configuration
	CLI CLIConfig `yaml:"cli" json:"cli"`

	// Internal metadata
	ConfigVersion string `yaml:"config_version,omitempty" json:"config_version,omitempty"`
	ConfigPath    string `yaml:"-" json:"-"`
}

// GeneralConfig contains general taskopen settings.
type GeneralConfig struct {
	// Editor command for editing notes/files
	Editor string `yaml:"editor" json:"editor" validate:"required" default:"vim"`

	// Taskwarrior binary path
	TaskBin string `yaml:"taskbin" json:"taskbin" validate:"required" default:"task"`

	// Additional arguments to pass to taskwarrior
	TaskArgs []string `yaml:"taskargs" json:"taskargs"`

	// Path extension for file operations
	PathExt string `yaml:"path_ext" json:"path_ext"`

	// Task attributes to display
	TaskAttributes string `yaml:"task_attributes" json:"task_attributes" default:"priority,project,tags,description"`

	// Hook for tasks without annotations
	NoAnnotationHook string `yaml:"no_annotation_hook" json:"no_annotation_hook" default:"annotate $ID"`

	// Default task sort order
	Sort string `yaml:"sort" json:"sort" default:"urgency-,annot"`

	// Base filter for tasks
	BaseFilter string `yaml:"base_filter" json:"base_filter" default:"+PENDING"`

	// Debug mode
	Debug bool `yaml:"debug" json:"debug"`
}

// CLIConfig contains CLI-specific configuration.
type CLIConfig struct {
	// Default subcommand when none specified
	DefaultSubcommand string `yaml:"default_subcommand" json:"default_subcommand" default:"normal"`

	// Command aliases
	Aliases map[string]string `yaml:"aliases" json:"aliases"`

	// Action groups
	Groups map[string]string `yaml:"groups" json:"groups"`
}

// DefaultConfig returns a configuration with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		ConfigVersion: "2.0",
		General: GeneralConfig{
			Editor:           getEnvDefault("EDITOR", "vim"),
			TaskBin:          "task",
			TaskArgs:         []string{},
			PathExt:          "",
			TaskAttributes:   "priority,project,tags,description",
			NoAnnotationHook: "annotate $ID",
			Sort:             "urgency-,annot",
			BaseFilter:       "+PENDING",
			Debug:            false,
		},
		Actions: []types.Action{
			{
				Name:       "files",
				Target:     "annotations",
				Regex:      `^[\.\/~]+.*\.(.*)`,
				LabelRegex: ".*",
				Command:    getOpenCommand() + " $FILE",
				Modes:      []string{"batch", "any", "normal"},
			},
			{
				Name:       "notes",
				Target:     "annotations",
				Regex:      `.*\.([a-zA-Z0-9]+)$`,
				LabelRegex: ".*",
				Command:    `editnote ~/Notes/tasknotes/$UUID$LAST_MATCH "$TASK_DESCRIPTION" $UUID`,
				Modes:      []string{"batch", "any", "normal"},
			},
			{
				Name:       "url",
				Target:     "annotations",
				Regex:      `((?:www|http).*)`,
				LabelRegex: ".*",
				Command:    getOpenCommand() + " $LAST_MATCH",
				Modes:      []string{"batch", "any", "normal"},
			},
		},
		CLI: CLIConfig{
			DefaultSubcommand: "normal",
			Aliases: map[string]string{
				"batch":       "",
				"any":         "",
				"normal":      "",
				"version":     "",
				"diagnostics": "",
			},
			Groups: make(map[string]string),
		},
	}
}

// getOpenCommand returns the appropriate open command for the platform.
func getOpenCommand() string {
	// Check environment variable first
	if cmd := os.Getenv("TASKOPEN_OPEN_CMD"); cmd != "" {
		return cmd
	}

	// Platform-specific defaults
	switch {
	case fileExists("/usr/bin/xdg-open"):
		return "xdg-open"
	case fileExists("/usr/bin/open"):
		return "open"
	case fileExists("/usr/bin/start"):
		return "start"
	default:
		return "xdg-open" // Fallback
	}
}

// getEnvDefault returns environment variable value or default if not set.
func getEnvDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// fileExists checks if a file exists.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// FindConfigPath locates the configuration file using standard locations.
func FindConfigPath() (string, error) {
	// Priority order for config file locations:
	// 1. $TASKOPENRC environment variable
	// 2. $XDG_CONFIG_HOME/taskopen/config.yml
	// 3. $HOME/.config/taskopen/config.yml
	// 4. $HOME/.taskopenrc (legacy INI format)

	if path := os.Getenv("TASKOPENRC"); path != "" {
		return path, nil
	}

	// XDG config directory
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", errors.Wrap(err, errors.ConfigNotFound, "Unable to determine home directory")
		}
		configDir = filepath.Join(homeDir, ".config")
	}

	// Check for YAML config
	yamlPath := filepath.Join(configDir, "taskopen", "config.yml")
	if fileExists(yamlPath) {
		return yamlPath, nil
	}

	// Check for legacy INI config
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", errors.Wrap(err, errors.ConfigNotFound, "Unable to determine home directory")
	}

	legacyPath := filepath.Join(homeDir, ".taskopenrc")
	if fileExists(legacyPath) {
		return legacyPath, nil
	}

	// Return preferred YAML path even if it doesn't exist
	return yamlPath, nil
}

// Validate performs comprehensive validation on the configuration.
func (c *Config) Validate() error {
	var validationErrors []types.ValidationError

	// Validate general settings
	if strings.TrimSpace(c.General.Editor) == "" {
		validationErrors = append(validationErrors, types.ValidationError{
			Field:   "general.editor",
			Value:   c.General.Editor,
			Message: "editor command is required",
		})
	}

	if strings.TrimSpace(c.General.TaskBin) == "" {
		validationErrors = append(validationErrors, types.ValidationError{
			Field:   "general.taskbin",
			Value:   c.General.TaskBin,
			Message: "taskwarrior binary path is required",
		})
	}

	// Validate actions
	if len(c.Actions) == 0 {
		validationErrors = append(validationErrors, types.ValidationError{
			Field:   "actions",
			Value:   fmt.Sprintf("%d actions", len(c.Actions)),
			Message: "at least one action must be defined",
		})
	}

	actionNames := make(map[string]bool)
	for i, action := range c.Actions {
		// Check for duplicate action names
		if actionNames[action.Name] {
			validationErrors = append(validationErrors, types.ValidationError{
				Field:   fmt.Sprintf("actions[%d].name", i),
				Value:   action.Name,
				Message: "duplicate action name",
			})
		}
		actionNames[action.Name] = true

		// Validate individual action
		if err := action.Validate(); err != nil {
			if actionValidationErrs, ok := err.(*types.ValidationErrors); ok {
				for _, actionErr := range actionValidationErrs.Errors {
					validationErrors = append(validationErrors, types.ValidationError{
						Field:   fmt.Sprintf("actions[%d].%s", i, actionErr.Field),
						Value:   actionErr.Value,
						Message: actionErr.Message,
					})
				}
			}
		}
	}

	// Validate CLI configuration
	if c.CLI.DefaultSubcommand == "" {
		validationErrors = append(validationErrors, types.ValidationError{
			Field:   "cli.default_subcommand",
			Value:   c.CLI.DefaultSubcommand,
			Message: "default subcommand is required",
		})
	}

	if len(validationErrors) > 0 {
		return &types.ValidationErrors{Errors: validationErrors}
	}

	return nil
}

// GetAction returns an action by name.
func (c *Config) GetAction(name string) (*types.Action, bool) {
	for i := range c.Actions {
		if c.Actions[i].Name == name {
			return &c.Actions[i], true
		}
	}
	return nil, false
}

// GetActionNames returns all action names.
func (c *Config) GetActionNames() []string {
	names := make([]string, len(c.Actions))
	for i, action := range c.Actions {
		names[i] = action.Name
	}
	return names
}

// AddAction adds a new action to the configuration.
func (c *Config) AddAction(action types.Action) error {
	// Check for duplicate name
	if _, exists := c.GetAction(action.Name); exists {
		return errors.New(errors.ConfigInvalid, "Action with this name already exists").
			WithDetails(fmt.Sprintf("Action name: %s", action.Name)).
			WithSuggestion("Use a different action name")
	}

	// Validate the action
	if err := action.Validate(); err != nil {
		return errors.Wrap(err, errors.ValidationFailed, "Invalid action configuration")
	}

	c.Actions = append(c.Actions, action)
	return nil
}
