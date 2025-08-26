package main

import (
	"fmt"
	"os"

	"github.com/johnconnor-sec/taskopen-go/internal/config"
)

func runConfigCommand(args []string) error {
	if len(args) == 0 {
		fmt.Println("Config commands:")
		fmt.Println("  init     - Create configuration interactively")
		fmt.Println("  migrate  - Migrate INI config to YAML")
		fmt.Println("  validate - Validate configuration file")
		fmt.Println("  example  - Show example configuration")
		fmt.Println("  schema   - Generate JSON schema")
		return nil
	}

	subcommand := args[0]

	switch subcommand {
	case "init":
		return runConfigInit()
	case "migrate":
		return runConfigMigrate(args[1:])
	case "validate":
		return runConfigValidate(args[1:])
	case "example":
		return runConfigExample()
	case "schema":
		return runConfigSchema(args[1:])
	default:
		return fmt.Errorf("unknown config subcommand: %s", subcommand)
	}
}

func runConfigInit() error {
	configPath, err := config.FindConfigPath()
	if err != nil {
		return err
	}

	return config.GenerateInteractive(configPath)
}

func runConfigMigrate(args []string) error {
	var iniPath, yamlPath string

	if len(args) >= 2 {
		iniPath = args[0]
		yamlPath = args[1]
	} else {
		// Auto-detect paths
		homeDir, _ := os.UserHomeDir()
		iniPath = homeDir + "/.taskopenrc"
		yamlPath, _ = config.FindConfigPath()
	}

	return config.MigrateFromINI(iniPath, yamlPath)
}

func runConfigValidate(args []string) error {
	var configPath string

	if len(args) > 0 {
		configPath = args[0]
	} else {
		var err error
		configPath, err = config.FindConfigPath()
		if err != nil {
			return err
		}
	}

	return config.ValidateFile(configPath)
}

func runConfigExample() error {
	config.ShowConfigExample()
	return nil
}

func runConfigSchema(args []string) error {
	var outputPath string

	if len(args) > 0 {
		outputPath = args[0]
	} else {
		outputPath = "taskopen-schema.json"
	}

	if err := config.SaveJSONSchema(outputPath); err != nil {
		return err
	}

	fmt.Printf("✓ JSON schema saved to: %s\n", outputPath)
	return nil
}
