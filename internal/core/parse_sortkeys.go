package core

import "strings"

// SortKey represents a sort field and direction
type SortKey struct {
	Key  string
	Desc bool
}

// parseSortKeys parses the sort string into sort keys
func (tp *TaskProcessor) parseSortKeys(sortStr string) []SortKey {
	var keys []SortKey

	for field := range strings.SplitSeq(sortStr, ",") {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}

		desc := false
		if strings.HasSuffix(field, "-") {
			desc = true
			field = field[:len(field)-1]
		} else if strings.HasSuffix(field, "+") {
			field = field[:len(field)-1]
		}

		keys = append(keys, SortKey{Key: field, Desc: desc})
	}

	return keys
}
