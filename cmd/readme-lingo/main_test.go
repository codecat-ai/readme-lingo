package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/codecat-ai/readme-lingo/pkg/lingo"
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

func TestTranslateCheckWithGitHubAnnotationsReportsMissingAndStaleFiles(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "README.md")
	if err := os.WriteFile(source, []byte("# Demo\nChanged\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	stale := lingo.AddMetadata("# Old\n", lingo.Metadata{
		SourcePath:  source,
		Target:      "zh",
		Model:       "test-model",
		SourceHash:  lingo.SourceDigest([]byte("# Demo\n")),
		GeneratedAt: time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC),
	})
	stalePath := filepath.Join(dir, "README-zh.md")
	if err := os.WriteFile(stalePath, []byte(stale), 0o644); err != nil {
		t.Fatalf("write stale translation: %v", err)
	}
	missingPath := filepath.Join(dir, "README-ja.md")

	output, err := captureStdout(t, func() error {
		return run([]string{
			"translate",
			"--source", source,
			"--targets", "zh,ja",
			"--output-dir", dir,
			"--glossary", filepath.Join(dir, "missing-glossary.md"),
			"--check",
			"--github-annotations",
		})
	})
	if err == nil {
		t.Fatal("expected stale/missing check error")
	}

	wantHuman := []string{
		"missing: " + missingPath,
		"stale: " + stalePath,
	}
	for _, want := range wantHuman {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing human line %q:\n%s", want, output)
		}
	}
	wantAnnotations := []string{
		"::error file=" + source + ",title=missing translation::Missing translation output " + missingPath,
		"::error file=" + source + ",title=stale translation::Stale translation output " + stalePath,
	}
	for _, want := range wantAnnotations {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing annotation %q:\n%s", want, output)
		}
	}
}

func TestTranslateCheckWithGitHubAnnotationsEscapesWorkflowCommandData(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "README%demo\r\nmain.md")
	if err := os.WriteFile(source, []byte("# Demo\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	outputPath := filepath.Join(dir, "README%demo\r\nzh.md")

	output, err := captureStdout(t, func() error {
		return run([]string{
			"translate",
			"--source", source,
			"--target", "zh",
			"--output", outputPath,
			"--check",
			"--github-annotations",
		})
	})
	if err == nil {
		t.Fatal("expected missing translation error")
	}

	escapedSource := escapeForTestGitHubAnnotation(source)
	escapedOutput := escapeForTestGitHubAnnotation(outputPath)
	want := "::error file=" + escapedSource + ",title=missing translation::Missing translation output " + escapedOutput
	if !strings.Contains(output, want) {
		t.Fatalf("output missing escaped annotation %q:\n%s", want, output)
	}
}

func escapeForTestGitHubAnnotation(value string) string {
	value = strings.ReplaceAll(value, "%", "%25")
	value = strings.ReplaceAll(value, "\r", "%0D")
	value = strings.ReplaceAll(value, "\n", "%0A")
	return value
}

func TestTranslateGitHubAnnotationsFlagHasNoEffectWithoutCheck(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "README.md")
	if err := os.WriteFile(source, []byte("# Demo\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	output, err := captureStdout(t, func() error {
		return run([]string{
			"translate",
			"--source", source,
			"--target", "zh",
			"--output", filepath.Join(dir, "README-zh.md"),
			"--dry-run",
			"--github-annotations",
		})
	})
	if err != nil {
		t.Fatalf("dry-run with annotations flag: %v", err)
	}
	if strings.Contains(output, "::error") {
		t.Fatalf("annotation flag should not affect non-check output:\n%s", output)
	}
}

func TestWorkflowRequiresTargetsFlag(t *testing.T) {
	err := run([]string{"workflow"})
	if err == nil {
		t.Fatal("expected missing targets error")
	}
	if !strings.Contains(err.Error(), "--targets") {
		t.Fatalf("error should mention --targets, got %q", err)
	}
}

func captureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	original := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = original
	}()

	runErr := fn()
	if err := w.Close(); err != nil {
		t.Fatalf("close stdout pipe: %v", err)
	}
	data, readErr := io.ReadAll(r)
	if readErr != nil {
		t.Fatalf("read stdout pipe: %v", readErr)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("close stdout reader: %v", err)
	}
	return string(data), runErr
}
