package core

import (
	"context"
	"testing"
	"time"

	"github.com/johnconnor-sec/taskopen-go/taskopen/internal/config"
	"github.com/johnconnor-sec/taskopen-go/taskopen/internal/taskwarrior"
	"github.com/johnconnor-sec/taskopen-go/taskopen/internal/types"
)

func TestNewTaskOpen(t *testing.T) {
	cfg := config.DefaultConfig()

	taskOpen, err := New(cfg)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	if taskOpen == nil {
		t.Fatal("New() returned nil")
	}

	if taskOpen.config != cfg {
		t.Error("Config not properly set")
	}
}

func TestNewTaskOpenWithNilConfig(t *testing.T) {
	_, err := New(nil)
	if err == nil {
		t.Error("New() should error with nil config")
	}
}

func TestBuildEnvironment(t *testing.T) {
	cfg := config.DefaultConfig()
	taskOpen, err := New(cfg)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	// Create test task
	task := taskwarrior.Task{
		ID:          123,
		UUID:        "test-uuid-123",
		Description: "Test task description",
		Status:      "pending",
		Project:     "testproject",
		Priority:    "H",
		Tags:        []string{"tag1", "tag2"},
	}

	// Create test action
	action := types.Action{
		Name:   "test-action",
		Target: "annotations",
		Regex:  `https?://(.+)`,
	}

	matchedText := "https://example.com/path"

	env := taskOpen.buildEnvironment(task, action, matchedText)

	// Check required environment variables
	expectedVars := map[string]string{
		"TASK_ID":          "123",
		"TASK_UUID":        "test-uuid-123",
		"TASK_DESCRIPTION": "Test task description",
		"TASK_STATUS":      "pending",
		"TASK_PROJECT":     "testproject",
		"TASK_PRIORITY":    "H",
		"TASK_TAGS":        "tag1,tag2",
		"ACTION_NAME":      "test-action",
		"ACTION_TARGET":    "annotations",
		"MATCHED_TEXT":     "https://example.com/path",
		"LAST_MATCH":       "example.com/path",
		"MATCH_1":          "example.com/path",
		"EDITOR":           cfg.General.Editor,
	}

	for key, expected := range expectedVars {
		if actual, exists := env[key]; !exists {
			t.Errorf("Environment variable %s not set", key)
		} else if actual != expected {
			t.Errorf("Environment variable %s: got %s, want %s", key, actual, expected)
		}
	}
}

func TestExpandCommand(t *testing.T) {
	cfg := config.DefaultConfig()
	taskOpen, err := New(cfg)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	env := map[string]string{
		"EDITOR":     "vim",
		"FILE":       "/path/to/file.txt",
		"TASK_UUID":  "test-uuid",
		"LAST_MATCH": "example.com",
	}

	tests := []struct {
		name     string
		command  string
		expected string
	}{
		{
			name:     "simple variable",
			command:  "$EDITOR $FILE",
			expected: "vim /path/to/file.txt",
		},
		{
			name:     "braced variables",
			command:  "${EDITOR} ${FILE}",
			expected: "vim /path/to/file.txt",
		},
		{
			name:     "mixed variables",
			command:  "$EDITOR ${FILE} --uuid=$TASK_UUID",
			expected: "vim /path/to/file.txt --uuid=test-uuid",
		},
		{
			name:     "url opening",
			command:  "xdg-open https://$LAST_MATCH",
			expected: "xdg-open https://example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := taskOpen.expandCommand(tt.command, env)
			if result != tt.expected {
				t.Errorf("expandCommand() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestCreateActionable(t *testing.T) {
	cfg := config.DefaultConfig()
	taskOpen, err := New(cfg)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	// Create test task
	task := taskwarrior.Task{
		UUID:        "test-uuid",
		Description: "Test task",
		Status:      "pending",
	}

	// Create test action
	action := types.Action{
		Name:    "test-action",
		Target:  "annotations",
		Command: "echo test",
		Regex:   ".*",
	}

	actionable, err := taskOpen.createActionable(task, action, "test text", "test entry")
	if err != nil {
		t.Errorf("createActionable() error: %v", err)
	}

	if actionable.Text != "test text" {
		t.Errorf("Actionable text: got %s, want %s", actionable.Text, "test text")
	}

	if actionable.Entry != "test entry" {
		t.Errorf("Actionable entry: got %s, want %s", actionable.Entry, "test entry")
	}

	if actionable.Action.Name != "test-action" {
		t.Errorf("Actionable action name: got %s, want %s", actionable.Action.Name, "test-action")
	}

	if len(actionable.Env) == 0 {
		t.Error("Actionable environment should not be empty")
	}
}

func TestFindActionablesForTask(t *testing.T) {
	cfg := config.DefaultConfig()
	taskOpen, err := New(cfg)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	// Create test task with annotations
	task := taskwarrior.Task{
		UUID:        "test-uuid",
		Description: "Test task",
		Status:      "pending",
		Annotations: []taskwarrior.Annotation{
			{
				Entry:       time.Now(),
				Description: "https://example.com/link",
			},
			{
				Entry:       time.Now(),
				Description: "Note: This is just a note",
			},
		},
	}

	// Create action that matches URLs
	action := types.Action{
		Name:    "url",
		Target:  "annotations",
		Regex:   `https?://.*`,
		Command: "xdg-open $MATCHED_TEXT",
	}

	actionables, err := taskOpen.findActionablesForTask(task, action)
	if err != nil {
		t.Errorf("findActionablesForTask() error: %v", err)
	}

	// Should find one actionable (the URL)
	if len(actionables) != 1 {
		t.Errorf("Expected 1 actionable, got %d", len(actionables))
	}

	if len(actionables) > 0 {
		if actionables[0].Text != "https://example.com/link" {
			t.Errorf("Wrong actionable text: got %s", actionables[0].Text)
		}
	}
}

func TestFindActionablesForTaskDescription(t *testing.T) {
	cfg := config.DefaultConfig()
	taskOpen, err := New(cfg)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	// Create test task with description containing actionable text
	task := taskwarrior.Task{
		UUID:        "test-uuid",
		Description: "EDIT this task later",
		Status:      "pending",
	}

	// Create action that matches EDIT in description
	action := types.Action{
		Name:    "edit",
		Target:  "description",
		Regex:   `EDIT`,
		Command: "$EDITOR /tmp/task-$TASK_UUID.txt",
	}

	actionables, err := taskOpen.findActionablesForTask(task, action)
	if err != nil {
		t.Errorf("findActionablesForTask() error: %v", err)
	}

	// Should find one actionable
	if len(actionables) != 1 {
		t.Errorf("Expected 1 actionable, got %d", len(actionables))
	}

	if len(actionables) > 0 {
		if actionables[0].Text != "EDIT this task later" {
			t.Errorf("Wrong actionable text: got %s", actionables[0].Text)
		}
	}
}

func TestFindActionablesInvalidRegex(t *testing.T) {
	cfg := config.DefaultConfig()
	taskOpen, err := New(cfg)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	task := taskwarrior.Task{
		UUID:        "test-uuid",
		Description: "Test task",
		Status:      "pending",
	}

	// Create action with invalid regex
	action := types.Action{
		Name:    "invalid",
		Target:  "annotations",
		Regex:   "[invalid",
		Command: "echo test",
	}

	_, err = taskOpen.findActionablesForTask(task, action)
	if err == nil {
		t.Error("Expected error for invalid regex")
	}
}

func TestFindActionablesInvalidTarget(t *testing.T) {
	cfg := config.DefaultConfig()
	taskOpen, err := New(cfg)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	task := taskwarrior.Task{
		UUID:        "test-uuid",
		Description: "Test task",
		Status:      "pending",
	}

	// Create action with invalid target
	action := types.Action{
		Name:    "invalid",
		Target:  "invalid_target",
		Regex:   ".*",
		Command: "echo test",
	}

	_, err = taskOpen.findActionablesForTask(task, action)
	if err == nil {
		t.Error("Expected error for invalid target")
	}
}

// Mock tests for verification without requiring actual taskwarrior installation

func TestVerifySetupMockSuccess(t *testing.T) {
	cfg := config.DefaultConfig()

	// Use /bin/echo as a safe mock for taskwarrior binary
	cfg.General.TaskBin = "/bin/echo"
	cfg.General.Editor = "/bin/echo"

	taskOpen, err := New(cfg)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	ctx := context.Background()

	// This will fail because echo doesn't return taskwarrior version format,
	// but it tests the execution path
	err = taskOpen.VerifySetup(ctx)
	// We expect this to fail with a taskwarrior-specific error, not a "command not found" error
	if err == nil {
		t.Error("VerifySetup should fail with mock binary")
	}
}

// Helper function to check if string contains substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())
}
