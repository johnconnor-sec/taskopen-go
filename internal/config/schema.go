// Package config - JSON Schema generation for IDE support
package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// JSONSchema generates a JSON schema for the taskopen configuration.
func GenerateJSONSchema() ([]byte, error) {
	schema := map[string]interface{}{
		"$schema":              "http://json-schema.org/draft-07/schema#",
		"title":                "Taskopen Configuration",
		"description":          "Configuration schema for taskopen task annotation opener",
		"type":                 "object",
		"additionalProperties": false,

		"properties": map[string]interface{}{
			"config_version": map[string]interface{}{
				"type":        "string",
				"description": "Configuration format version",
				"default":     "2.0",
			},

			"general": map[string]interface{}{
				"type":                 "object",
				"description":          "General taskopen settings",
				"required":             []string{"editor", "taskbin"},
				"additionalProperties": false,

				"properties": map[string]interface{}{
					"editor": map[string]interface{}{
						"type":        "string",
						"description": "Editor command for editing notes and files",
						"default":     "vim",
						"examples":    []string{"vim", "nano", "code", "emacs"},
					},

					"taskbin": map[string]interface{}{
						"type":        "string",
						"description": "Path to taskwarrior binary",
						"default":     "task",
						"examples":    []string{"task", "/usr/bin/task", "/usr/local/bin/task"},
					},

					"taskargs": map[string]interface{}{
						"type":        "array",
						"description": "Additional arguments to pass to taskwarrior",
						"items": map[string]interface{}{
							"type": "string",
						},
						"default": []interface{}{},
					},

					"path_ext": map[string]interface{}{
						"type":        "string",
						"description": "Path extension for file operations",
						"default":     "",
					},

					"task_attributes": map[string]interface{}{
						"type":        "string",
						"description": "Task attributes to display in output",
						"default":     "priority,project,tags,description",
						"examples": []string{
							"priority,project,tags,description",
							"urgency,project,description",
							"id,description,due",
						},
					},

					"no_annotation_hook": map[string]interface{}{
						"type":        "string",
						"description": "Command to run for tasks without annotations",
						"default":     "addnote $ID",
					},

					"sort": map[string]interface{}{
						"type":        "string",
						"description": "Default task sort order",
						"default":     "urgency-,annot",
						"examples":    []string{"urgency-", "due+", "priority-,urgency-"},
					},

					"base_filter": map[string]interface{}{
						"type":        "string",
						"description": "Base filter for task queries",
						"default":     "+PENDING",
						"examples":    []string{"+PENDING", "+READY", "status:pending"},
					},

					"debug": map[string]interface{}{
						"type":        "boolean",
						"description": "Enable debug output",
						"default":     false,
					},
				},
			},

			"actions": map[string]interface{}{
				"type":        "array",
				"description": "Action definitions for opening task annotations",
				"minItems":    1,

				"items": map[string]interface{}{
					"type":                 "object",
					"description":          "A single action configuration",
					"required":             []string{"name", "target", "command"},
					"additionalProperties": false,

					"properties": map[string]interface{}{
						"name": map[string]interface{}{
							"type":        "string",
							"description": "Unique name for this action",
							"minLength":   1,
							"pattern":     "^[a-zA-Z][a-zA-Z0-9_-]*$",
						},

						"target": map[string]interface{}{
							"type":        "string",
							"description": "Where to look for matches",
							"enum":        []string{"annotations", "description"},
							"default":     "annotations",
						},

						"regex": map[string]interface{}{
							"type":        "string",
							"description": "Regular expression pattern to match",
							"default":     ".*",
						},

						"labelregex": map[string]interface{}{
							"type":        "string",
							"description": "Regular expression for annotation labels",
							"default":     ".*",
						},

						"command": map[string]interface{}{
							"type":        "string",
							"description": "Command to execute when action matches",
							"minLength":   1,
						},

						"modes": map[string]interface{}{
							"type":        "array",
							"description": "Modes in which this action is available",
							"items": map[string]interface{}{
								"type": "string",
								"enum": []string{"batch", "any", "normal"},
							},
							"default":     []interface{}{"batch", "any", "normal"},
							"minItems":    1,
							"uniqueItems": true,
						},

						"filtercommand": map[string]interface{}{
							"type":        "string",
							"description": "Command to filter matches before execution",
							"default":     "",
						},

						"inlinecommand": map[string]interface{}{
							"type":        "string",
							"description": "Command to execute inline with task display",
							"default":     "",
						},
					},
				},
			},

			"cli": map[string]interface{}{
				"type":                 "object",
				"description":          "CLI-specific configuration",
				"additionalProperties": false,

				"properties": map[string]interface{}{
					"default_subcommand": map[string]interface{}{
						"type":        "string",
						"description": "Default subcommand when none specified",
						"default":     "normal",
						"enum":        []string{"batch", "any", "normal"},
					},

					"aliases": map[string]interface{}{
						"type":        "object",
						"description": "Command aliases",
						"additionalProperties": map[string]interface{}{
							"type": "string",
						},
					},

					"groups": map[string]interface{}{
						"type":        "object",
						"description": "Action groups",
						"additionalProperties": map[string]interface{}{
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
func GetSchemaExamples() map[string]interface{} {
	return map[string]interface{}{
		"minimal": map[string]interface{}{
			"general": map[string]interface{}{
				"editor":  "vim",
				"taskbin": "task",
			},
			"actions": []map[string]interface{}{
				{
					"name":    "files",
					"target":  "annotations",
					"regex":   `^[\.\/~]+.*\.(.*)",`,
					"command": "xdg-open $FILE",
				},
			},
		},

		"comprehensive": map[string]interface{}{
			"config_version": "2.0",
			"general": map[string]interface{}{
				"editor":             "code",
				"taskbin":            "task",
				"taskargs":           []string{"rc.verbose=off"},
				"task_attributes":    "priority,project,tags,description,due",
				"no_annotation_hook": "addnote $ID",
				"sort":               "urgency-,annot",
				"base_filter":        "+PENDING",
				"debug":              false,
			},
			"actions": []map[string]interface{}{
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
			"cli": map[string]interface{}{
				"default_subcommand": "normal",
				"aliases": map[string]interface{}{
					"b": "batch",
					"n": "normal",
				},
				"groups": map[string]interface{}{
					"files": "files,notes",
				},
			},
		},
	}
}
