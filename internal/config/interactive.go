// Package config - Interactive configuration generation
package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/johnconnor-sec/taskopen-go/internal/errors"
	"github.com/johnconnor-sec/taskopen-go/internal/types"
)

// isInteractiveEnvironment checks if we're in a proper interactive environment
func isInteractiveEnvironment() bool {
	// Check if stdin is a terminal (not a pipe or redirect)
	if fileInfo, err := os.Stdin.Stat(); err == nil {
		if (fileInfo.Mode() & os.ModeCharDevice) == 0 {
			fmt.Printf("[DEBUG] stdin is not a character device (mode: %v)\n", fileInfo.Mode())
			return false
		}
	}

	// Check environment variables that indicate non-interactive usage
	if os.Getenv("CI") != "" || os.Getenv("TERM") == "dumb" {
		fmt.Printf("[DEBUG] CI or dumb terminal detected\n")
		return false
	}

	// Check if we have a reasonable terminal
	term := os.Getenv("TERM")
	if term == "" {
		fmt.Printf("[DEBUG] No TERM environment variable\n")
		return false
	}

	fmt.Printf("[DEBUG] Interactive environment: TERM=%s\n", term)
	return true
}

// GenerateInteractive creates a configuration interactively with user input.
func GenerateInteractive(configPath string) error {
	// Environment-specific checks
	if !isInteractiveEnvironment() {
		return errors.New(errors.ConfigInvalid, "Non-interactive environment detected. Use config file instead.")
	}

	fmt.Println("🎯 Taskopen Configuration Setup")
	fmt.Println("===============================")
	fmt.Println()

	config := DefaultConfig()
	reader := bufio.NewReader(os.Stdin)

	// General configuration
	fmt.Println("📝 General Configuration")
	fmt.Println("------------------------")

	// Editor
	fmt.Printf("Editor command [%s]: ", config.General.Editor)
	if editor := readLine(reader); editor != "" {
		config.General.Editor = editor
	}

	// Taskwarrior binary
	fmt.Printf("Taskwarrior binary [%s]: ", config.General.TaskBin)
	if taskBin := readLine(reader); taskBin != "" {
		config.General.TaskBin = taskBin
	}

	// Task attributes
	fmt.Printf("Task attributes to display [%s]: ", config.General.TaskAttributes)
	if attrs := readLine(reader); attrs != "" {
		config.General.TaskAttributes = attrs
	}

	// Debug mode
	fmt.Print("Enable debug mode? [y/N]: ")
	if debug := readLine(reader); strings.ToLower(debug) == "y" {
		config.General.Debug = true
	}

	fmt.Println()

	// Actions configuration
	fmt.Println("⚡ Actions Configuration")
	fmt.Println("------------------------")
	fmt.Println("Default actions include:")
	for _, action := range config.Actions {
		fmt.Printf("  • %s: %s\n", action.Name, action.Command)
	}
	fmt.Println()

	fmt.Print("Add custom actions? [y/N]: ")
	if addActions := readLine(reader); strings.ToLower(addActions) == "y" {
		if err := configureActionsInteractively(config, reader); err != nil {
			return err
		}
	}

	fmt.Println()

	// CLI configuration
	fmt.Println("🖥️  CLI Configuration")
	fmt.Println("---------------------")

	fmt.Printf("Default subcommand [%s]: ", config.CLI.DefaultSubcommand)
	if defaultCmd := readLine(reader); defaultCmd != "" {
		config.CLI.DefaultSubcommand = defaultCmd
	}

	fmt.Println()

	// Summary and confirmation
	fmt.Println("📋 Configuration Summary")
	fmt.Println("------------------------")
	fmt.Printf("• Editor: %s\n", config.General.Editor)
	fmt.Printf("• Taskwarrior: %s\n", config.General.TaskBin)
	fmt.Printf("• Actions: %d defined\n", len(config.Actions))
	fmt.Printf("• Debug mode: %v\n", config.General.Debug)
	fmt.Printf("• Config path: %s\n", configPath)
	fmt.Println()

	fmt.Print("Save this configuration? [Y/n]: ")
	if confirm := readLine(reader); confirm != "" && strings.ToLower(confirm) != "y" {
		return errors.New(errors.ConfigInvalid, "Configuration not saved")
	}

	// Save configuration
	if err := Save(config, configPath); err != nil {
		return errors.Wrap(err, errors.ConfigInvalid, "Failed to save configuration")
	}

	fmt.Println()
	fmt.Printf("✅ Configuration saved successfully to: %s\n", configPath)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("• Run 'taskopen diagnostics' to verify setup")
	fmt.Println("• Run 'taskopen' to start using taskopen")
	fmt.Println("• Edit the config file to add more custom actions")

	return nil
}

// configureActionsInteractively allows user to add custom actions.
func configureActionsInteractively(config *Config, reader *bufio.Reader) error {
	fmt.Println()
	fmt.Println("Adding custom actions...")
	fmt.Println("Leave action name empty to finish.")
	fmt.Println()

	for {
		fmt.Print("Action name: ")
		name := readLine(reader)
		if name == "" {
			break
		}

		// Check for duplicate names
		if _, exists := config.GetAction(name); exists {
			fmt.Printf("⚠️  Action '%s' already exists. Choose a different name.\n", name)
			continue
		}

		action := types.Action{
			Name:       name,
			Target:     "annotations",
			LabelRegex: ".*",
			Modes:      []string{"batch", "any", "normal"},
		}

		fmt.Print("Target (annotations/description) [annotations]: ")
		if target := readLine(reader); target != "" {
			action.Target = target
		}

		fmt.Print("Regex pattern [.*]: ")
		if regex := readLine(reader); regex != "" {
			action.Regex = regex
		} else {
			action.Regex = ".*"
		}

		fmt.Print("Command to execute: ")
		command := readLine(reader)
		if command == "" {
			fmt.Println("Command is required. Skipping this action.")
			continue
		}
		action.Command = command

		fmt.Print("Modes (comma-separated) [batch,any,normal]: ")
		if modes := readLine(reader); modes != "" {
			action.Modes = strings.Split(modes, ",")
			for i := range action.Modes {
				action.Modes[i] = strings.TrimSpace(action.Modes[i])
			}
		}

		// Validate the action
		if err := action.Validate(); err != nil {
			fmt.Printf("⚠️  Invalid action: %v\n", err)
			fmt.Println("Skipping this action.")
			continue
		}

		config.Actions = append(config.Actions, action)
		fmt.Printf("✅ Added action '%s'\n", name)
		fmt.Println()
	}

	return nil
}

// readLine reads a line from the reader, trimming whitespace.
func readLine(reader *bufio.Reader) string {
	line, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("[DEBUG] Error reading line: %v\n", err)
		return ""
	}

	// Handle different line ending types (Unix \n, Windows \r\n, Mac \r)
	trimmed := strings.TrimSpace(line)

	// Validate UTF-8 encoding
	if !utf8.ValidString(trimmed) {
		fmt.Printf("[DEBUG] Invalid UTF-8 sequence detected\n")
		// Try to clean up invalid UTF-8
		trimmed = strings.ToValidUTF8(trimmed, "�")
	}

	// Debug character encoding issues
	fmt.Printf("[DEBUG] Read line: %q (len=%d bytes, %d runes, bytes=%v)\n",
		trimmed, len(trimmed), utf8.RuneCountInString(trimmed), []byte(trimmed))

	return trimmed
}

// ShowConfigExample displays an example configuration file.
func ShowConfigExample() {
	example := `# Example Taskopen Configuration (YAML)
general:
  editor: "vim"
  taskbin: "task"
  taskargs: []
  task_attributes: "priority,project,tags,description"
  no_annotation_hook: "annotate $ID"
  sort: "urgency-,annot"
  base_filter: "+PENDING"
  debug: false

actions:
  - name: "files"
    target: "annotations"
    regex: '^[\.\/~]+.*\.(.*)'
    labelregex: ".*"
    command: "xdg-open $FILE"
    modes: ["batch", "any", "normal"]
    
  - name: "notes"
    target: "annotations"
		regex: '.*\.([a-zA-Z0-9]+)$'
    labelregex: ".*"
	  # Replace this with the default path of where you store task notes
    command: 'editnote ~/Notes/tasknotes/$UUID$LAST_MATCH "$TASK_DESCRIPTION" $UUID'
    modes: ["batch", "any", "normal"]
    
  - name: "url"
    target: "annotations"
    regex: '((?:www|http).*)'
    labelregex: ".*"
    command: "xdg-open $LAST_MATCH"
    modes: ["batch", "any", "normal"]
    
  - name: "custom-editor"
    target: "description"
    regex: "EDIT"
    labelregex: ".*"
    command: "$EDITOR /tmp/task-$UUID.txt"
    modes: ["normal"]

cli:
  default_subcommand: "normal"
  aliases:
    batch: ""
    any: ""
    normal: ""
  groups: {}

config_version: "2.0"`

	fmt.Println("Example Configuration:")
	fmt.Println("=====================")
	fmt.Println(example)
}
