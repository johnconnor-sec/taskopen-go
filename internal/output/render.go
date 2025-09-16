package output

import (
	"encoding/json"
	"fmt"
	"strings"
)

// RenderTaskList outputs a formatted task list with enhanced readability
func (f *Formatter) RenderTaskList(tasks []map[string]any) {
	f.RenderTaskListWithTemplate(tasks, DefaultTaskTemplate)
}

// RenderTaskListWithTemplate renders tasks using a specific template
func (f *Formatter) RenderTaskListWithTemplate(tasks []map[string]any, template OutputTemplate) {
	if len(tasks) == 0 {
		f.ScreenReaderText("info", "No tasks match the current filter")
		return
	}

	f.Subheader(fmt.Sprintf("Found %d tasks", len(tasks)))

	switch template.Format {
	case "table":
		f.renderTaskTable(tasks, template)
	case "list":
		f.renderTaskList(tasks, template)
	case "json":
		f.renderTaskJSON(tasks)
	default:
		f.renderTaskTable(tasks, template)
	}
}

// renderTaskTable renders tasks in table format
func (f *Formatter) renderTaskTable(tasks []map[string]any, template OutputTemplate) {
	headers := make([]string, len(template.Fields))
	for i, field := range template.Fields {
		headers[i] = strings.Title(field)
	}

	table := f.Table().Headers(headers...)

	for _, task := range tasks {
		row := make([]string, len(template.Fields))
		for i, field := range template.Fields {
			value := f.formatTaskField(task, field, template)
			row[i] = value
		}
		table.Row(row...)
	}

	table.Print()
}

// renderTaskList renders tasks in list format
func (f *Formatter) renderTaskList(tasks []map[string]any, template OutputTemplate) {
	for i, task := range tasks {
		if i > 0 {
			fmt.Fprintln(f.writer)
		}

		var parts []string
		for _, field := range template.Fields {
			value := f.formatTaskField(task, field, template)
			if value != "" {
				parts = append(parts, value)
			}
		}

		f.List("%s", strings.Join(parts, " - "))
	}
}

// renderTaskJSON renders tasks as JSON
func (f *Formatter) renderTaskJSON(tasks []map[string]any) {
	for _, task := range tasks {
		if data, err := json.Marshal(task); err == nil {
			fmt.Fprintln(f.writer, string(data))
		}
	}
}
