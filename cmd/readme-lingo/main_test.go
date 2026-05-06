package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTranslateDryRunAcceptsGlossaryWithoutReadingIt(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "README.md")
	if err := os.WriteFile(source, []byte("# Demo\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	err := run([]string{
		"translate",
		"--source", source,
		"--target", "zh",
		"--output", filepath.Join(dir, "README-zh.md"),
		"--glossary", filepath.Join(dir, "missing-glossary.md"),
		"--dry-run",
	})
	if err != nil {
		t.Fatalf("dry-run should accept unreadable glossary: %v", err)
	}
}

func TestTranslateCheckAcceptsGlossaryWithoutReadingIt(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "README.md")
	if err := os.WriteFile(source, []byte("# Demo\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	glossary := filepath.Join(dir, "missing-glossary.md")
	err := run([]string{
		"translate",
		"--source", source,
		"--target", "zh",
		"--output", filepath.Join(dir, "README-zh.md"),
		"--glossary", glossary,
		"--check",
	})
	if err == nil {
		t.Fatal("expected missing translation error")
	}
	if strings.Contains(err.Error(), glossary) {
		t.Fatalf("check should not read glossary; got error %q", err)
	}
}
