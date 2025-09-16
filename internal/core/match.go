package core

import (
	"context"
	"regexp"

	"github.com/johnconnor-sec/taskopen-go/internal/types"
)

// matchActionsLabel matches actions against annotation text with label support
func (tp *TaskProcessor) matchActionsLabel(ctx context.Context, baseEnv map[string]string, text string, actions []types.Action, single bool) []*Actionable {
	var matches []*Actionable

	// Split annotation into label and file part
	splitRegex := regexp.MustCompile(`^((\S+):\s+)?(.*)$`)
	splitMatches := splitRegex.FindStringSubmatch(text)
	if len(splitMatches) != 4 {
		tp.logger.Error(
			"Malformed annotation",
			map[string]any{"text": text},
		)
		return matches
	}

	label := splitMatches[2]
	file := splitMatches[3]

	for _, action := range actions {
		env := tp.copyEnvironment(baseEnv)

		// Check label regex
		if action.LabelRegex != "" {
			labelRegex, err := regexp.Compile(action.LabelRegex)
			if err != nil {
				tp.logger.Error("Invalid label regex", map[string]any{"regex": action.LabelRegex, "error": err.Error()})
				continue
			}
			if !labelRegex.MatchString(label) {
				continue
			}
		}

		// Check file regex
		fileRegex, err := regexp.Compile(action.Regex)
		if err != nil {
			tp.logger.Error("Invalid file regex", map[string]any{"regex": action.Regex, "error": err.Error()})
			continue
		}

		fileMatches := fileRegex.FindStringSubmatch(file)
		if len(fileMatches) == 0 {
			continue
		}

		// Set environment variables
		env["LAST_MATCH"] = ""
		if len(fileMatches) > 0 {
			env["LAST_MATCH"] = fileMatches[0]
		}
		env["LABEL"] = label
		env["FILE"] = tp.expandPath(file)
		env["ANNOTATION"] = text

		// Apply filter command if specified
		if action.FilterCommand != "" {
			if !tp.executeFilter(ctx, action.FilterCommand, env) {
				tp.logger.Info("Filter command filtered out action", map[string]any{
					"action": action.Name,
					"text":   text,
				})
				continue
			}
		}

		// Create actionable
		taskID := env["UUID"]
		if taskID == "" {
			taskID = env["ID"]
		}

		actionable := &Actionable{
			Text:        text,
			TaskID:      taskID,
			Action:      action,
			Environment: env,
		}

		matches = append(matches, actionable)

		if single {
			break
		}
	}

	return matches
}

// matchActionsPure matches actions against plain text (non-annotation attributes)
func (tp *TaskProcessor) matchActionsPure(ctx context.Context, baseEnv map[string]string, text string, actions []types.Action, single bool) []*Actionable {
	var matches []*Actionable

	for _, action := range actions {
		env := tp.copyEnvironment(baseEnv)

		// Check file regex
		fileRegex, err := regexp.Compile(action.Regex)
		if err != nil {
			tp.logger.Error("Invalid regex", map[string]any{"regex": action.Regex, "error": err.Error()})
			continue
		}

		fileMatches := fileRegex.FindStringSubmatch(text)
		if len(fileMatches) == 0 {
			continue
		}

		// Set environment variables
		env["LAST_MATCH"] = ""
		if len(fileMatches) > 0 {
			env["LAST_MATCH"] = fileMatches[0]
		}
		env["FILE"] = text
		env["ANNOTATION"] = text

		// Warn about unused labelregex
		if action.LabelRegex != "" {
			tp.logger.Warn("labelregex is ignored for actions not targeting annotations", map[string]any{
				"action": action.Name,
			})
		}

		// Apply filter command if specified
		if action.FilterCommand != "" {
			if !tp.executeFilter(ctx, action.FilterCommand, env) {
				tp.logger.Info("Filter command filtered out action", map[string]any{
					"action": action.Name,
					"text":   text,
				})
				continue
			}
		}

		// Create actionable
		taskID := env["UUID"]
		if taskID == "" {
			taskID = env["ID"]
		}

		actionable := &Actionable{
			Text:        text,
			TaskID:      taskID,
			Action:      action,
			Environment: env,
		}

		matches = append(matches, actionable)

		if single {
			break
		}
	}

	return matches
}
