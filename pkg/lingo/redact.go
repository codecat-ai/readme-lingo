package lingo

import "strings"

func RedactDiagnostics(message string, env map[string]string) string {
	redacted := message
	for name, value := range env {
		if value == "" {
			continue
		}
		redacted = strings.ReplaceAll(redacted, name+"="+value, name+"=[REDACTED]")
		redacted = strings.ReplaceAll(redacted, value, "[REDACTED]")
	}
	return redacted
}
