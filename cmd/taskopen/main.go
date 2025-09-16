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

func printUsage() {
	formatter := output.NewFormatter(os.Stdout)

	formatter.Header("Taskopen - Interactive Task Annotation Opener")

	fmt.Println("Usage:")
	fmt.Println("  taskopen [OPTIONS] [FILTERS...]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -i, --interactive      Enable interactive menu (default)")
	fmt.Println("  --no-interactive       Disable interactive menu, use first match")
	fmt.Println("  --batch                Same as --no-interactive")
	fmt.Println("  -s, --single           Process single task only (default)")
	fmt.Println("  -m, --multiple         Process multiple tasks")
	fmt.Println("  -v, --version          Show version information")
	fmt.Println("  -h, --help             Show this help message")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  taskopen config init   Initialize configuration")
	fmt.Println("  taskopen diagnostics   Run system diagnostics")
	fmt.Println("  taskopen version       Show version information")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  taskopen                           # Interactive menu for all tasks")
	fmt.Println("  taskopen project:work              # Interactive menu for work project")
	fmt.Println("  taskopen --no-interactive urgent   # Execute first urgent task action")
	fmt.Println("  taskopen -m project:home           # Process multiple home project tasks")
	fmt.Println()
	fmt.Println("Interactive Menu Controls:")
	fmt.Println("  j/k or ↑/↓    Navigate up/down")
	fmt.Println("  Enter         Select action")
	fmt.Println("  /             Search actions")
	fmt.Println("  ?             Show help")
	fmt.Println("  q or Esc      Cancel/quit")
	fmt.Println("  Space         Multi-select (when available)")
}

func runTaskOpen(args []string) error {
	// Parse command-line flags
	interactive := true // Default to interactive mode
	single := true      // Default to single mode
	var filters []string

	// Simple flag parsing
	for i, arg := range args {
		switch arg {
		case "--interactive", "-i":
			interactive = true
		case "--no-interactive", "--batch":
			interactive = false
		case "--single", "-s":
			single = true
		case "--multiple", "-m":
			single = false
		case "--help", "-h":
			printUsage()
			return nil
		default:
			// All remaining arguments are filters
			filters = args[i:]
			break
		}
	}

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
	return processor.ProcessTasks(context.Background(), filters, single, interactive)
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
