package types

import (
	"encoding/json"
	"testing"
)

func TestActionValidation(t *testing.T) {
	tests := []struct {
		name      string
		action    Action
		wantError bool
		errorText string
	}{
		{
			name: "valid action",
			action: Action{
				Name:    "test-action",
				Target:  "annotation",
				Regex:   ".*",
				Command: "echo test",
				Modes:   []string{"batch"},
			},
			wantError: false,
		},
		{
			name: "missing name",
			action: Action{
				Target:  "annotation",
				Command: "echo test",
			},
			wantError: true,
			errorText: "action name is required",
		},
		{
			name: "missing target",
			action: Action{
				Name:    "test",
				Command: "echo test",
			},
			wantError: true,
			errorText: "action target is required",
		},
		{
			name: "missing command",
			action: Action{
				Name:   "test",
				Target: "annotation",
			},
			wantError: true,
			errorText: "action command is required",
		},
		{
			name: "invalid regex",
			action: Action{
				Name:    "test",
				Target:  "annotation",
				Regex:   "[invalid",
				Command: "echo test",
			},
			wantError: true,
			errorText: "invalid regex pattern",
		},
		{
			name: "invalid label regex",
			action: Action{
				Name:       "test",
				Target:     "annotation",
				LabelRegex: "[invalid",
				Command:    "echo test",
			},
			wantError: true,
			errorText: "invalid label regex pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.action.Validate()
			if (err != nil) != tt.wantError {
				t.Errorf("Action.Validate() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if tt.wantError && tt.errorText != "" {
				if err == nil || !containsString(err.Error(), tt.errorText) {
					t.Errorf("Action.Validate() error = %v, want error containing %q", err, tt.errorText)
				}
			}
		})
	}
}

func TestActionableValidation(t *testing.T) {
	validAction := Action{
		Name:    "test-action",
		Target:  "annotation",
		Command: "echo test",
	}

	taskData, _ := json.Marshal(map[string]any{
		"uuid":        "test-uuid",
		"description": "test task",
	})

	tests := []struct {
		name       string
		actionable Actionable
		wantError  bool
		errorText  string
	}{
		{
			name: "valid actionable",
			actionable: Actionable{
				Text:   "test text",
				Task:   taskData,
				Entry:  "test-entry",
				Action: validAction,
				Env:    map[string]string{"TEST": "value"},
			},
			wantError: false,
		},
		{
			name: "missing text",
			actionable: Actionable{
				Task:   taskData,
				Entry:  "test-entry",
				Action: validAction,
			},
			wantError: true,
			errorText: "actionable text is required",
		},
		{
			name: "missing task",
			actionable: Actionable{
				Text:   "test text",
				Entry:  "test-entry",
				Action: validAction,
			},
			wantError: true,
			errorText: "task data is required",
		},
		{
			name: "missing entry",
			actionable: Actionable{
				Text:   "test text",
				Task:   taskData,
				Action: validAction,
			},
			wantError: true,
			errorText: "actionable entry is required",
		},
		{
			name: "invalid action",
			actionable: Actionable{
				Text:  "test text",
				Task:  taskData,
				Entry: "test-entry",
				Action: Action{
					Name:   "test",
					Target: "annotation",
					// Missing required command
				},
			},
			wantError: true,
			errorText: "action command is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.actionable.Validate()
			if (err != nil) != tt.wantError {
				t.Errorf("Actionable.Validate() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if tt.wantError && tt.errorText != "" {
				if err == nil || !containsString(err.Error(), tt.errorText) {
					t.Errorf("Actionable.Validate() error = %v, want error containing %q", err, tt.errorText)
				}
			}
		})
	}
}

func TestJSONSerialization(t *testing.T) {
	action := Action{
		Name:          "test-action",
		Target:        "annotation",
		Regex:         ".*",
		LabelRegex:    "label:.*",
		Command:       "echo test",
		Modes:         []string{"batch", "interactive"},
		FilterCommand: "filter-cmd",
		InlineCommand: "inline-cmd",
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(action)
	if err != nil {
		t.Errorf("Failed to marshal Action to JSON: %v", err)
		return
	}

	// Test JSON unmarshaling
	var unmarshaled Action
	if err := json.Unmarshal(jsonData, &unmarshaled); err != nil {
		t.Errorf("Failed to unmarshal Action from JSON: %v", err)
		return
	}

	// Verify fields are preserved
	if unmarshaled.Name != action.Name {
		t.Errorf("Name mismatch after JSON round-trip: got %s, want %s", unmarshaled.Name, action.Name)
	}
	if unmarshaled.Target != action.Target {
		t.Errorf("Target mismatch after JSON round-trip: got %s, want %s", unmarshaled.Target, action.Target)
	}
	if len(unmarshaled.Modes) != len(action.Modes) {
		t.Errorf("Modes length mismatch after JSON round-trip: got %d, want %d", len(unmarshaled.Modes), len(action.Modes))
	}
}

// Helper function to check if a string contains a substring
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
