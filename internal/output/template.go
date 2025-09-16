package output

// OutputTemplate defines a customizable output template
type OutputTemplate struct {
	Name        string
	Description string
	Format      string              // "table", "list", "json", "custom"
	Fields      []string            // Fields to display
	Filters     map[string]string   // Field filters/transforms
	Styles      map[string]StyleDef // Custom styling per field
}

// StyleDef defines styling for a field
type StyleDef struct {
	Color     Color
	Style     Style
	Transform string // "upper", "lower", "title", "truncate:N"
}

// DefaultTaskTemplate provides a sensible default for task display
var DefaultTaskTemplate = OutputTemplate{
	Name:        "default",
	Description: "Standard task list view",
	Format:      "table",
	Fields:      []string{"id", "priority", "project", "description", "due"},
	Styles: map[string]StyleDef{
		"priority": {Color: ColorYellow, Style: StyleBold, Transform: "priority_color"},
		"due":      {Color: ColorRed, Style: StyleNormal, Transform: "relative_time"},
		"project":  {Color: ColorCyan, Style: StyleNormal},
	},
}

// CompactTaskTemplate for limited space
var CompactTaskTemplate = OutputTemplate{
	Name:        "compact",
	Description: "Compact task view for small terminals",
	Format:      "list",
	Fields:      []string{"id", "description"},
	Styles: map[string]StyleDef{
		"id": {Color: ColorBlue, Style: StyleBold},
	},
}

// SetTemplate allows switching output templates
func (f *Formatter) SetTemplate(template OutputTemplate) {
	// Future enhancement: store current template on formatter
}

// ListTemplates shows available output templates
func (f *Formatter) ListTemplates() []OutputTemplate {
	return []OutputTemplate{
		DefaultTaskTemplate,
		CompactTaskTemplate,
		{
			Name:        "json",
			Description: "JSON output for scripting",
			Format:      "json",
			Fields:      []string{}, // All fields
		},
		{
			Name:        "minimal",
			Description: "Minimal output for accessibility",
			Format:      "list",
			Fields:      []string{"id", "description"},
		},
	}
}
