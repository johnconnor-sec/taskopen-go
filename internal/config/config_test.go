package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/johnconnor-sec/taskopen-go/taskopen/internal/types"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config == nil {
		t.Fatal("DefaultConfig() returned nil")
	}

	// Check required fields
	if config.General.Editor == "" {
		t.Error("Default config should have editor set")
	}

	if config.General.TaskBin == "" {
		t.Error("Default config should have taskbin set")
	}

	if len(config.Actions) == 0 {
		t.Error("Default config should have actions")
	}

	// Validate the default config
	if err := config.Validate(); err != nil {
		t.Errorf("Default config should be valid: %v", err)
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		wantError bool
		errorText string
	}{
		{
			name:      "valid config",
			config:    DefaultConfig(),
			wantError: false,
		},
		{
			name: "missing editor",
			config: &Config{
				General: GeneralConfig{
					TaskBin: "task",
				},
				Actions: []types.Action{{
					Name:    "test",
					Target:  "annotations",
					Command: "echo test",
				}},
				CLI: CLIConfig{DefaultSubcommand: "normal"},
			},
			wantError: true,
			errorText: "editor command is required",
		},
		{
			name: "no actions",
			config: &Config{
				General: GeneralConfig{
					Editor:  "vim",
					TaskBin: "task",
				},
				Actions: []types.Action{},
				CLI:     CLIConfig{DefaultSubcommand: "normal"},
			},
			wantError: true,
			errorText: "at least one action must be defined",
		},
		{
			name: "duplicate action names",
			config: &Config{
				General: GeneralConfig{
					Editor:  "vim",
					TaskBin: "task",
				},
				Actions: []types.Action{
					{Name: "test", Target: "annotations", Command: "echo 1"},
					{Name: "test", Target: "annotations", Command: "echo 2"},
				},
				CLI: CLIConfig{DefaultSubcommand: "normal"},
			},
			wantError: true,
			errorText: "duplicate action name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantError {
				t.Errorf("Config.Validate() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if tt.wantError && tt.errorText != "" {
				if err == nil || !containsString(err.Error(), tt.errorText) {
					t.Errorf("Config.Validate() error = %v, want error containing %q", err, tt.errorText)
				}
			}
		})
	}
}

func TestGetAction(t *testing.T) {
	config := DefaultConfig()

	// Test getting existing action
	action, found := config.GetAction("files")
	if !found {
		t.Error("GetAction should find 'files' action in default config")
	}
	if action == nil {
		t.Error("GetAction should return non-nil action")
	}
	if action.Name != "files" {
		t.Errorf("GetAction returned wrong action: got %s, want files", action.Name)
	}

	// Test getting non-existent action
	_, found = config.GetAction("nonexistent")
	if found {
		t.Error("GetAction should not find non-existent action")
	}
}

func TestAddAction(t *testing.T) {
	config := DefaultConfig()
	originalCount := len(config.Actions)

	newAction := types.Action{
		Name:    "test-action",
		Target:  "annotations",
		Command: "echo test",
		Regex:   ".*",
	}

	// Test adding valid action
	if err := config.AddAction(newAction); err != nil {
		t.Errorf("AddAction should not error for valid action: %v", err)
	}

	if len(config.Actions) != originalCount+1 {
		t.Errorf("AddAction should increase action count: got %d, want %d", len(config.Actions), originalCount+1)
	}

	// Test adding duplicate action
	if err := config.AddAction(newAction); err == nil {
		t.Error("AddAction should error for duplicate action name")
	}
}

func TestGetActionNames(t *testing.T) {
	config := DefaultConfig()
	names := config.GetActionNames()

	if len(names) != len(config.Actions) {
		t.Errorf("GetActionNames length mismatch: got %d, want %d", len(names), len(config.Actions))
	}

	// Check that all default action names are present
	expectedActions := []string{"files", "notes", "url"}
	for _, expected := range expectedActions {
		found := false
		for _, name := range names {
			if name == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("GetActionNames missing expected action: %s", expected)
		}
	}
}

func TestFindConfigPath(t *testing.T) {
	// Test with clean environment
	originalTASKOPENRC := os.Getenv("TASKOPENRC")
	originalXDGCONFIGHOME := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		os.Setenv("TASKOPENRC", originalTASKOPENRC)
		os.Setenv("XDG_CONFIG_HOME", originalXDGCONFIGHOME)
	}()

	// Clear environment variables
	os.Unsetenv("TASKOPENRC")
	os.Unsetenv("XDG_CONFIG_HOME")

	path, err := FindConfigPath()
	if err != nil {
		t.Errorf("FindConfigPath should not error: %v", err)
	}

	if path == "" {
		t.Error("FindConfigPath should return a path")
	}

	// Should return YAML config path when no config exists
	if !containsString(path, "config.yml") {
		t.Errorf("FindConfigPath should prefer YAML format: %s", path)
	}
}

func TestFileExists(t *testing.T) {
	// Create a temporary file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	if fileExists(testFile) {
		t.Error("fileExists should return false for non-existent file")
	}

	// Create the file
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	if !fileExists(testFile) {
		t.Error("fileExists should return true for existing file")
	}
}

// Helper function from types_test.go
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			func() bool {
				for i := 0; i <= len(s)-len(substr); i++ {
					if s[i:i+len(substr)] == substr {
						return true
					}
				}
				return false
			}())))
}
