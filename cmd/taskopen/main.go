// Taskopen - A powerful task annotation opener for Taskwarrior
package main

import (
	"context"
	"fmt"
	"os"
	"runtime"

	"github.com/johnconnor-sec/taskopen-go/internal/config"
	"github.com/johnconnor-sec/taskopen-go/internal/core"
	"github.com/johnconnor-sec/taskopen-go/internal/errors"
	"github.com/johnconnor-sec/taskopen-go/internal/output"
)

// Build information - set by linker flags
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	if err := run(); err != nil {
		handleError(err)
		os.Exit(1)
	}
}

func run() error {
	args := os.Args[1:]

	// Handle version flag
	if len(args) > 0 && (args[0] == "--version" || args[0] == "-v" || args[0] == "version") {
		printVersion()
		return nil
	}

	// Handle diagnostics command
	if len(args) > 0 && args[0] == "diagnostics" {
		return runDiagnostics()
	}

	// Handle config commands
	if len(args) > 0 && args[0] == "config" {
		return runConfigCommand(args[1:])
	}

	// Main taskopen functionality - run the core application
	return runTaskOpen(args)
}

func printVersion() {
	formatter := output.NewFormatter(os.Stdout)

	formatter.Header(fmt.Sprintf("Taskopen %s", version))

	// Create a table for version info
	table := formatter.Table()
	table.Headers("Component", "Version")
	table.Row("Taskopen", version)
	table.Row("Git commit", commit)
	table.Row("Build date", date)
	table.Row("Go version", runtime.Version())
	table.Row("Platform", fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH))
	table.Print()
}

func runTaskOpen(args []string) error {
	// Load configuration
	configPath, err := config.FindConfigPath()
	if err != nil {
		return fmt.Errorf("configuration not found: %w", err)
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create task processor
	processor := core.NewTaskProcessor(cfg)

	// Process tasks with the provided arguments as filters
	return processor.ProcessTasks(context.Background(), args, true, true)
}

func handleError(err error) {
	formatter := output.NewFormatter(os.Stderr)

	// Use our structured error handling with beautiful output
	if taskopenErr, ok := err.(*errors.TaskopenError); ok {
		formatter.Error("%s", taskopenErr.Error())
	} else {
		formatter.Error("Unexpected error: %v", err)
	}
}
