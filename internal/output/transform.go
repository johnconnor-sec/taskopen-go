package output

import (
	"strconv"
	"strings"
)

// applyFieldTransform applies various transformations to field values
func (f *Formatter) applyFieldTransform(value, transform string) string {
	switch transform {
	case "upper":
		return strings.ToUpper(value)
	case "lower":
		return strings.ToLower(value)
	case "title":
		return strings.Title(value)
	case "priority_color":
		return f.transformPriorityDisplay(value)
	case "relative_time":
		return f.transformRelativeTime(value)
	default:
		if strings.HasPrefix(transform, "truncate:") {
			if parts := strings.Split(transform, ":"); len(parts) == 2 {
				if maxLen, err := strconv.Atoi(parts[1]); err == nil && len(value) > maxLen {
					return value[:maxLen-3] + "..."
				}
			}
		}
	}
	return value
}

// transformPriorityDisplay converts priority codes to readable format
func (f *Formatter) transformPriorityDisplay(priority string) string {
	switch priority {
	case "H":
		return "HIGH"
	case "M":
		return "MED"
	case "L":
		return "LOW"
	default:
		return priority
	}
}

// transformRelativeTime converts timestamps to relative time
func (f *Formatter) transformRelativeTime(timeStr string) string {
	// Simple relative time - could be enhanced with proper parsing
	if timeStr == "" {
		return ""
	}
	// For now, just return the original - this would need proper time parsing
	return timeStr
}
