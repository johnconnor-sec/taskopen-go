package output

import (
	"fmt"
	"strings"
)

// DiagnosticInfo represents system diagnostic information
type DiagnosticInfo struct {
	Component   string
	Status      string
	Details     map[string]any
	Suggestions []string
}

// RenderDiagnostics outputs comprehensive system diagnostics
func (f *Formatter) RenderDiagnostics(diagnostics []DiagnosticInfo) {
	f.Header("System Diagnostics")

	// System overview table
	table := f.Table().Headers("Component", "Status", "Details")

	for _, diag := range diagnostics {
		status := diag.Status
		if f.colorOutput {
			switch diag.Status {
			case "✓ Ready", "✓ Functional":
				status = f.colorize(diag.Status, f.theme.Success, StyleBold)
			case "⚠ Warning":
				status = f.colorize(diag.Status, f.theme.Warning, StyleBold)
			case "✗ Failed", "✗ Error":
				status = f.colorize(diag.Status, f.theme.Error, StyleBold)
			default:
				status = f.colorize(diag.Status, f.theme.Info, StyleNormal)
			}
		}

		// Format details
		details := ""
		if len(diag.Details) > 0 {
			var parts []string
			for k, v := range diag.Details {
				parts = append(parts, fmt.Sprintf("%s: %v", k, v))
			}
			details = strings.Join(parts, ", ")
		}

		table.Row(diag.Component, status, details)
	}

	table.Print()

	// Show suggestions for failed components
	for _, diag := range diagnostics {
		if strings.Contains(diag.Status, "Failed") || strings.Contains(diag.Status, "Warning") {
			if len(diag.Suggestions) > 0 {
				fmt.Fprintln(f.writer)
				f.ScreenReaderText("warning", fmt.Sprintf("%s Issues", diag.Component))
				for _, suggestion := range diag.Suggestions {
					f.List("%s", suggestion)
				}
			}
		}
	}
}
