package output

import (
	"fmt"
	"strings"
)

// Table represents a columnized table
type Table struct {
	formatter *Formatter
	headers   []string
	rows      [][]string
}

// Headers sets the table headers
func (t *Table) Headers(headers ...string) *Table {
	t.headers = headers
	return t
}

// Row adds a row to the table
func (t *Table) Row(cells ...string) *Table {
	t.rows = append(t.rows, cells)
	return t
}

// Print renders the table
func (t *Table) Print() {
	if t.formatter.level == LevelQuiet {
		return
	}

	if len(t.headers) > 0 {
		// Print headers
		headerRow := make([]string, len(t.headers))
		for i, header := range t.headers {
			headerRow[i] = t.formatter.colorize(header, t.formatter.theme.Primary, StyleBold)
		}
		fmt.Fprintln(t.formatter.tabwriter, strings.Join(headerRow, "\t"))

		// Print separator
		separators := make([]string, len(t.headers))
		for i, header := range t.headers {
			separators[i] = strings.Repeat("─", len(header))
		}
		fmt.Fprintln(t.formatter.tabwriter, strings.Join(separators, "\t"))
	}

	// Print rows
	for _, row := range t.rows {
		fmt.Fprintln(t.formatter.tabwriter, strings.Join(row, "\t"))
	}

	t.formatter.tabwriter.Flush()
}
