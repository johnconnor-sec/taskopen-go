package core

import "strings"

// assessCommandRisk provides basic risk assessment for commands
func (tp *TaskProcessor) assessCommandRisk(command string) string {
	cmd := strings.ToLower(command)

	// Critical risk patterns
	critical := []string{"rm -rf", "format", "dd if="}
	for _, pattern := range critical {
		if strings.Contains(cmd, pattern) {
			return "CRITICAL: Don't do this"
		}
	}

	// High risk patterns
	high := []string{"sudo rm", "rm /", "shutdown", "reboot"}
	for _, pattern := range high {
		if strings.Contains(cmd, pattern) {
			return "HIGH: Are you sure?"
		}
	}

	// Medium risk patterns
	medium := []string{"sudo", "rm ", "mv ", "chmod", "chown"}
	for _, pattern := range medium {
		if strings.Contains(cmd, pattern) {
			return "MEDIUM: Are you sure you want to run this?"
		}
	}

	return "SAFE"
}
