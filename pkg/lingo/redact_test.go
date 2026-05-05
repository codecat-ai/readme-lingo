package lingo

import (
	"strings"
	"testing"
)

func TestRedactDiagnosticsHidesSensitiveValues(t *testing.T) {
	value := "placeholder-" + strings.ReplaceAll(t.Name(), "/", "-")
	got := RedactDiagnostics("using README_LINGO_API_KEY="+value+" for request", map[string]string{
		"README_LINGO_API_KEY": value,
	})
	if strings.Contains(got, value) {
		t.Fatalf("secret leaked in diagnostics: %q", got)
	}
	if !strings.Contains(got, "README_LINGO_API_KEY=[REDACTED]") {
		t.Fatalf("redacted env marker missing: %q", got)
	}
}
