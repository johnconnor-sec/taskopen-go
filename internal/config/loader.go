// Package config - YAML configuration loading and parsing
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/johnconnor-sec/taskopen-go/taskopen/internal/errors"
)

// Load reads and parses the configuration from the specified path.
func Load(configPath string) (*Config, error) {
	// Check if config file exists
	if !fileExists(configPath) {
		return nil, errors.ConfigNotFoundError(configPath)
	}

	// Read the config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, errors.Wrap(err, errors.ConfigNotFound, "Failed to read configuration file").
			WithDetails(fmt.Sprintf("Path: %s", configPath)).
			WithSuggestion("Check file permissions and path")
	}

	// Parse YAML
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, errors.Wrap(err, errors.ConfigInvalid, "Invalid YAML configuration").
			WithDetails(fmt.Sprintf("Parse error: %v", err)).
			WithSuggestions([]string{
				"Check YAML syntax",
				"Validate indentation",
				"Ensure proper field names",
				"Run 'taskopen config validate' for detailed validation",
			})
	}

	// Set the config path for reference
	config.ConfigPath = configPath

	// Validate the configuration
	if err := config.Validate(); err != nil {
		return nil, errors.Wrap(err, errors.ConfigInvalid, "Configuration validation failed").
			WithSuggestions([]string{
				"Check required fields",
				"Verify action definitions",
				"Run 'taskopen config init' to create a new config",
			})
	}

	return &config, nil
}

// LoadOrCreate attempts to load configuration, creating default if not found.
func LoadOrCreate(configPath string) (*Config, error) {
	// Try to load existing configuration
	config, err := Load(configPath)
	if err != nil {
		// If config not found, create default
		if errors.IsType(err, errors.ConfigNotFound) {
			config = DefaultConfig()
			config.ConfigPath = configPath

			// Ask user if they want to create the config
			if err := createConfigInteractively(configPath, config); err != nil {
				return nil, err
			}

			return config, nil
		}
		return nil, err
	}

	return config, nil
}

// Save writes the configuration to the specified path.
func Save(config *Config, configPath string) error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return errors.Wrap(err, errors.PermissionDenied, "Cannot create config directory").
			WithDetails(fmt.Sprintf("Path: %s", filepath.Dir(configPath)))
	}

	// Marshal to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return errors.Wrap(err, errors.InternalError, "Failed to serialize configuration")
	}

	// Write to file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return errors.Wrap(err, errors.PermissionDenied, "Cannot write configuration file").
			WithDetails(fmt.Sprintf("Path: %s", configPath))
	}

	return nil
}

// createConfigInteractively prompts user to create configuration.
func createConfigInteractively(configPath string, config *Config) error {
	fmt.Printf("Configuration file '%s' does not exist.\n", configPath)
	fmt.Print("Create default configuration? [Y/n]: ")

	var answer string
	fmt.Scanln(&answer)

	// Default to yes if no answer provided
	if answer == "" || answer == "y" || answer == "Y" {
		if err := Save(config, configPath); err != nil {
			return errors.Wrap(err, errors.ConfigNotFound, "Failed to create default configuration")
		}
		fmt.Printf("✓ Created default configuration at: %s\n", configPath)
		return nil
	}

	return errors.New(errors.ConfigNotFound, "Configuration file required").
		WithSuggestion("Run 'taskopen config init' to create configuration interactively")
}

// Validate validates a configuration file without loading it.
func ValidateFile(configPath string) error {
	config, err := Load(configPath)
	if err != nil {
		return err
	}

	fmt.Printf("✓ Configuration is valid: %s\n", configPath)
	fmt.Printf("  - %d actions defined\n", len(config.Actions))
	fmt.Printf("  - Editor: %s\n", config.General.Editor)
	fmt.Printf("  - Task binary: %s\n", config.General.TaskBin)

	return nil
}
