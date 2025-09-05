// Package config - JSON Schema generation for IDE support
package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// JSONSchema generates a JSON schema for the taskopen configuration.
func GenerateJSONSchema() ([]byte, error) {
	schema := map[string]any{
		"$schema":              "http://json-schema.org/draft-07/schema#",
		"title":                "Taskopen Configuration",
		"description":          "Configuration schema for taskopen task annotation opener",
		"type":                 "object",
		"additionalProperties": false,

		"properties": map[string]any{
			"config_version": map[string]any{
				"type":        "string",
				"description": "Configuration format version",
				"default":     "2.0",
			},

			"general": map[string]any{
				"type":                 "object",
				"description":          "General taskopen settings",
				"required":             []string{"editor", "taskbin"},
				"additionalProperties": false,

				"properties": map[string]any{
					"editor": map[string]any{
						"type":        "string",
						"description": "Editor command for editing notes and files",
						"default":     "vim",
						"examples":    []string{"vim", "nano", "code", "emacs"},
					},

					"taskbin": map[string]any{
						"type":        "string",
						"description": "Path to taskwarrior binary",
						"default":     "task",
						"examples":    []string{"task", "/usr/bin/task", "/usr/local/bin/task"},
					},

					"taskargs": map[string]any{
						"type":        "array",
						"description": "Additional arguments to pass to taskwarrior",
						"items": map[string]any{
							"type": "string",
						},
						"default": []any{},
					},

					"path_ext": map[string]any{
						"type":        "string",
						"description": "Path extension for file operations",
						"default":     "",
					},

					"task_attributes": map[string]any{
						"type":        "string",
						"description": "Task attributes to display in output",
						"default":     "priority,project,tags,description",
						"examples": []string{
							"priority,project,tags,description",
							"urgency,project,description",
							"id,description,due",
						},
					},

					"no_annotation_hook": map[string]any{
						"type":        "string",
						"description": "Command to run for tasks without annotations",
						"default":     "annotate $ID",
					},

					"sort": map[string]any{
						"type":        "string",
						"description": "Default task sort order",
						"default":     "urgency-,annot",
						"examples":    []string{"urgency-", "due+", "priority-,urgency-"},
					},

					"base_filter": map[string]any{
						"type":        "string",
						"description": "Base filter for task queries",
						"default":     "+PENDING",
						"examples":    []string{"+PENDING", "+READY", "status:pending"},
					},

					"debug": map[string]any{
						"type":        "boolean",
						"description": "Enable debug output",
						"default":     false,
					},
				},
			},

			"actions": map[string]any{
				"type":        "array",
				"description": "Action definitions for opening task annotations",
				"minItems":    1,

				"items": map[string]any{
					"type":                 "object",
					"description":          "A single action configuration",
					"required":             []string{"name", "target", "command"},
					"additionalProperties": false,

					"properties": map[string]any{
						"name": map[string]any{
							"type":        "string",
							"description": "Unique name for this action",
							"minLength":   1,
							"pattern":     "^[a-zA-Z][a-zA-Z0-9_-]*$",
						},

						"target": map[string]any{
							"type":        "string",
							"description": "Where to look for matches",
							"enum":        []string{"annotations", "description"},
							"default":     "annotations",
						},

						"regex": map[string]any{
							"type":        "string",
							"description": "Regular expression pattern to match",
							"default":     ".*",
						},

						"labelregex": map[string]any{
							"type":        "string",
							"description": "Regular expression for annotation labels",
							"default":     ".*",
						},

						"command": map[string]any{
							"type":        "string",
							"description": "Command to execute when action matches",
							"minLength":   1,
						},

						"modes": map[string]any{
							"type":        "array",
							"description": "Modes in which this action is available",
							"items": map[string]any{
								"type": "string",
								"enum": []string{"batch", "any", "normal"},
							},
							"default":     []any{"batch", "any", "normal"},
							"minItems":    1,
							"uniqueItems": true,
						},

						"filtercommand": map[string]any{
							"type":        "string",
							"description": "Command to filter matches before execution",
							"default":     "",
						},

						"inlinecommand": map[string]any{
							"type":        "string",
							"description": "Command to execute inline with task display",
							"default":     "",
						},
					},
				},
			},

			"cli": map[string]any{
				"type":                 "object",
				"description":          "CLI-specific configuration",
				"additionalProperties": false,

				"properties": map[string]any{
					"default_subcommand": map[string]any{
						"type":        "string",
						"description": "Default subcommand when none specified",
						"default":     "normal",
						"enum":        []string{"batch", "any", "normal"},
					},

					"aliases": map[string]any{
						"type":        "object",
						"description": "Command aliases",
						"additionalProperties": map[string]any{
							"type": "string",
						},
					},

					"groups": map[string]any{
						"type":        "object",
						"description": "Action groups",
						"additionalProperties": map[string]any{
							"type": "string",
						},
					},
				},
			},
		},

		"required": []string{"general", "actions"},
	}

	return json.MarshalIndent(schema, "", "  ")
}

// SaveJSONSchema saves the JSON schema to a file.
func SaveJSONSchema(filePath string) error {
	schema, err := GenerateJSONSchema()
	if err != nil {
		return fmt.Errorf("failed to generate JSON schema: %w", err)
	}

	if err := os.WriteFile(filePath, schema, 0644); err != nil {
		return fmt.Errorf("failed to write JSON schema: %w", err)
	}

	return nil
}

// GetSchemaExamples returns example configurations for documentation.
func GetSchemaExamples() map[string]any {
	return map[string]any{
		"minimal": map[string]any{
			"general": map[string]any{
				"editor":  "vim",
				"taskbin": "task",
			},
			"actions": []map[string]any{
				{
					"name":    "files",
					"target":  "annotations",
					"regex":   `^[\.\/~]+.*\.(.*)",`,
					"command": "xdg-open $FILE",
				},
			},
		},

		"comprehensive": map[string]any{
			"config_version": "2.0",
			"general": map[string]any{
				"editor":             "code",
				"taskbin":            "task",
				"taskargs":           []string{"rc.verbose=off"},
				"task_attributes":    "priority,project,tags,description,due",
				"no_annotation_hook": "annotate $ID",
				"sort":               "urgency-,annot",
				"base_filter":        "+PENDING",
				"debug":              false,
			},
			"actions": []map[string]any{
				{
					"name":       "files",
					"target":     "annotations",
					"regex":      `^[\.\/~]+.*\.(.*)",`,
					"labelregex": ".*",
					"command":    "xdg-open $FILE",
					"modes":      []string{"batch", "any", "normal"},
				},
				{
					"name":          "edit-task",
					"target":        "description",
					"regex":         "EDIT",
					"command":       "$EDITOR /tmp/task-$UUID.txt",
					"modes":         []string{"normal"},
					"filtercommand": "test -w /tmp",
				},
			},
			"cli": map[string]any{
				"default_subcommand": "normal",
				"aliases": map[string]any{
					"b": "batch",
					"n": "normal",
				},
				"groups": map[string]any{
					"files": "files,notes",
				},
			},
		},
	}
}
