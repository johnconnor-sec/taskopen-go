// Package config - INI to YAML migration functionality
package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/johnconnor-sec/taskopen-go/internal/errors"
	"github.com/johnconnor-sec/taskopen-go/internal/types"
)

// MigrateFromINI converts an INI configuration file to YAML format.
func MigrateFromINI(iniPath, yamlPath string) error {
	// Check if INI file exists
	if !fileExists(iniPath) {
		return errors.New(errors.ConfigNotFound, "INI configuration file not found").
			WithDetails(fmt.Sprintf("Path: %s", iniPath)).
			WithSuggestion("Verify the INI config file path")
	}

	// Parse INI configuration
	config, err := parseINIConfig(iniPath)
	if err != nil {
		return errors.Wrap(err, errors.ConfigInvalid, "Failed to parse INI configuration").
			WithSuggestions([]string{
				"Check INI file syntax",
				"Ensure file is readable",
			})
	}

	// Ensure YAML directory exists
	if err := os.MkdirAll(filepath.Dir(yamlPath), 0755); err != nil {
		return errors.Wrap(err, errors.PermissionDenied, "Cannot create YAML config directory").
			WithDetails(fmt.Sprintf("Directory: %s", filepath.Dir(yamlPath)))
	}

	// Save as YAML
	if err := Save(config, yamlPath); err != nil {
		return errors.Wrap(err, errors.ConfigInvalid, "Failed to save YAML configuration").
			WithDetails(fmt.Sprintf("Target path: %s", yamlPath))
	}

	fmt.Printf("✓ Successfully migrated configuration:\n")
	fmt.Printf("  From: %s\n", iniPath)
	fmt.Printf("  To:   %s\n", yamlPath)
	fmt.Printf("  Actions migrated: %d\n", len(config.Actions))

	return nil
}

// parseINIConfig parses an INI file and returns a Config struct.
func parseINIConfig(iniPath string) (*Config, error) {
	file, err := os.Open(iniPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	config := DefaultConfig()
	config.Actions = []types.Action{} // Start with empty actions, we'll add them from INI

	scanner := bufio.NewScanner(file)
	currentSection := ""
	actionRegex := regexp.MustCompile(`^(.+)\.(target|regex|labelregex|command|modes|filtercommand|inlinecommand)$`)
	aliasGroupRegex := regexp.MustCompile(`^(alias|group)\.([^\.]+)$`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		// Section headers
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentSection = strings.ToLower(strings.Trim(line, "[]"))
			continue
		}

		// Key-value pairs
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch currentSection {
		case "general":
			parseGeneralSection(config, key, value)
		case "actions":
			parseActionsSection(config, key, value, actionRegex)
		case "cli":
			parseCLISection(config, key, value, aliasGroupRegex)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Ensure we have at least the default actions if none were defined
	if len(config.Actions) == 0 {
		config.Actions = DefaultConfig().Actions
	}

	return config, nil
}

// parseGeneralSection parses the [General] section of INI config.
func parseGeneralSection(config *Config, key, value string) {
	switch strings.ToLower(key) {
	case "editor":
		config.General.Editor = value
	case "taskbin":
		config.General.TaskBin = value
	case "taskargs":
		config.General.TaskArgs = strings.Fields(value)
	case "path_ext":
		config.General.PathExt = value
	case "task_attributes":
		config.General.TaskAttributes = value
	case "no_annotation_hook":
		config.General.NoAnnotationHook = value
	case "--sort":
		config.General.Sort = value
	case "--active-tasks":
		config.General.BaseFilter = value
	case "--debug":
		config.General.Debug = strings.ToLower(value) == "on" || strings.ToLower(value) == "true"
	}
}

// parseActionsSection parses the [Actions] section of INI config.
func parseActionsSection(config *Config, key, value string, actionRegex *regexp.Regexp) {
	if matches := actionRegex.FindStringSubmatch(key); matches != nil {
		actionName := matches[1]
		field := matches[2]

		// Find or create action
		var action *types.Action
		for i := range config.Actions {
			if config.Actions[i].Name == actionName {
				action = &config.Actions[i]
				break
			}
		}

		if action == nil {
			// Create new action with defaults
			newAction := types.Action{
				Name:       actionName,
				Target:     "annotations",
				LabelRegex: ".*",
				Regex:      ".*",
				Modes:      []string{"batch", "any", "normal"},
			}
			config.Actions = append(config.Actions, newAction)
			action = &config.Actions[len(config.Actions)-1]
		}

		// Set the field value
		switch field {
		case "target":
			action.Target = value
		case "regex":
			action.Regex = value
		case "labelregex":
			action.LabelRegex = value
		case "command":
			action.Command = value
		case "modes":
			action.Modes = strings.Split(value, ",")
			for i := range action.Modes {
				action.Modes[i] = strings.TrimSpace(action.Modes[i])
			}
		case "filtercommand":
			action.FilterCommand = value
		case "inlinecommand":
			action.InlineCommand = value
		}
	}
}

// parseCLISection parses the [CLI] section of INI config.
func parseCLISection(config *Config, key, value string, aliasGroupRegex *regexp.Regexp) {
	if key == "default" {
		config.CLI.DefaultSubcommand = value
	} else if matches := aliasGroupRegex.FindStringSubmatch(key); matches != nil {
		category := matches[1]
		name := matches[2]

		switch category {
		case "alias":
			if config.CLI.Aliases == nil {
				config.CLI.Aliases = make(map[string]string)
			}
			config.CLI.Aliases[name] = value
		case "group":
			if config.CLI.Groups == nil {
				config.CLI.Groups = make(map[string]string)
			}
			config.CLI.Groups[name] = value
		}
	}
}
