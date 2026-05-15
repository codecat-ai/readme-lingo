package main

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/codecat-ai/readme-lingo/pkg/lingo"
)

type testJSONPlan struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Output string `json:"output"`
}

type testDryRunJSONOutput struct {
	Plans []testJSONPlan `json:"plans"`
}

type testCheckJSONOutput struct {
	Plans   []testJSONPlan `json:"plans"`
	Missing []testJSONPlan `json:"missing"`
	Stale   []testJSONPlan `json:"stale"`
}

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

func TestTranslateDryRunJSONPrintsOnlyPlanJSON(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "README.md")
	if err := os.WriteFile(source, []byte("# Demo\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	output, err := captureStdout(t, func() error {
		return run([]string{
			"translate",
			"--source", source,
			"--targets", "zh,ja",
			"--output-dir", dir,
			"--dry-run",
			"--json",
		})
	})
	if err != nil {
		t.Fatalf("dry-run json: %v", err)
	}
	if strings.Contains(output, "plan:") {
		t.Fatalf("json stdout contains text plan prefix:\n%s", output)
	}

	var got testDryRunJSONOutput
	if err := json.Unmarshal([]byte(output), &got); err != nil {
		t.Fatalf("stdout is not valid json: %v\n%s", err, output)
	}
	want := []testJSONPlan{
		{Source: source, Target: "zh", Output: filepath.Join(dir, "README-zh.md")},
		{Source: source, Target: "ja", Output: filepath.Join(dir, "README-ja.md")},
	}
	if len(got.Plans) != len(want) {
		t.Fatalf("plans length = %d, want %d: %+v", len(got.Plans), len(want), got.Plans)
	}
	for i := range want {
		if got.Plans[i] != want[i] {
			t.Fatalf("plan[%d] = %+v, want %+v", i, got.Plans[i], want[i])
		}
	}
}

func TestTranslateDryRunAcceptsChunkFlagsWithoutNetwork(t *testing.T) {
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
		"--chunk-headings",
		"--max-chars", "10",
		"--dry-run",
	})
	if err != nil {
		t.Fatalf("dry-run should accept chunk flags: %v", err)
	}
}

func TestTranslateCheckJSONReportsMissingAndStaleWithoutStdoutText(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "README.md")
	sourceMarkdown := []byte("# Demo\nChanged\n")
	if err := os.WriteFile(source, sourceMarkdown, 0o644); err != nil {
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

	stdout, stderr, err := captureOutput(t, func() error {
		return run([]string{
			"translate",
			"--source", source,
			"--targets", "zh,ja",
			"--output-dir", dir,
			"--check",
			"--json",
			"--github-annotations",
		})
	})
	if err == nil {
		t.Fatal("expected stale/missing check error")
	}
	for _, forbidden := range []string{"missing:", "stale:", "::error"} {
		if strings.Contains(stdout, forbidden) {
			t.Fatalf("json stdout contains %q:\n%s", forbidden, stdout)
		}
	}
	if !strings.Contains(stderr, "::error file="+source+",title=missing translation::Missing translation output "+missingPath) {
		t.Fatalf("stderr missing GitHub annotation:\n%s", stderr)
	}

	var got testCheckJSONOutput
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid json: %v\n%s", err, stdout)
	}
	if len(got.Plans) != 2 {
		t.Fatalf("plans = %+v, want two plans", got.Plans)
	}
	if len(got.Missing) != 1 || got.Missing[0] != (testJSONPlan{Source: source, Target: "ja", Output: missingPath}) {
		t.Fatalf("missing = %+v, want ja missing", got.Missing)
	}
	if len(got.Stale) != 1 || got.Stale[0] != (testJSONPlan{Source: source, Target: "zh", Output: stalePath}) {
		t.Fatalf("stale = %+v, want zh stale", got.Stale)
	}
}

func TestTranslateCheckJSONReportsSynchronizedOutputs(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "README.md")
	sourceMarkdown := []byte("# Demo\n")
	if err := os.WriteFile(source, sourceMarkdown, 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	outputPath := filepath.Join(dir, "README-zh.md")
	translated := lingo.AddMetadata("# Demo translated\n", lingo.Metadata{
		SourcePath:  source,
		Target:      "zh",
		Model:       "test-model",
		SourceHash:  lingo.SourceDigest(sourceMarkdown),
		GeneratedAt: time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC),
	})
	if err := os.WriteFile(outputPath, []byte(translated), 0o644); err != nil {
		t.Fatalf("write synchronized translation: %v", err)
	}

	output, err := captureStdout(t, func() error {
		return run([]string{
			"translate",
			"--source", source,
			"--target", "zh",
			"--output", outputPath,
			"--check",
			"--json",
		})
	})
	if err != nil {
		t.Fatalf("check json synchronized: %v", err)
	}
	if strings.Contains(output, "ok:") {
		t.Fatalf("json stdout contains text ok prefix:\n%s", output)
	}

	var got testCheckJSONOutput
	if err := json.Unmarshal([]byte(output), &got); err != nil {
		t.Fatalf("stdout is not valid json: %v\n%s", err, output)
	}
	if len(got.Plans) != 1 || got.Plans[0] != (testJSONPlan{Source: source, Target: "zh", Output: outputPath}) {
		t.Fatalf("plans = %+v, want synchronized plan", got.Plans)
	}
	if len(got.Missing) != 0 || len(got.Stale) != 0 {
		t.Fatalf("missing/stale = %+v/%+v, want empty", got.Missing, got.Stale)
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

func TestTranslateCheckAcceptsChunkFlagsWithoutNetwork(t *testing.T) {
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
		"--chunk-headings",
		"--max-chars", "10",
		"--check",
	})
	if err == nil {
		t.Fatal("expected missing translation error")
	}
	if strings.Contains(err.Error(), "API key") {
		t.Fatalf("check should not require API key; got error %q", err)
	}
}

func TestTranslateRejectsNonPositiveMaxChars(t *testing.T) {
	err := run([]string{
		"translate",
		"--source", "README.md",
		"--target", "zh",
		"--chunk-headings",
		"--max-chars", "0",
	})
	if err == nil {
		t.Fatal("expected max chars error")
	}
	if !strings.Contains(err.Error(), "max chars must be positive") {
		t.Fatalf("error = %q, want clear max chars message", err)
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

func TestTranslateDryRunAcceptsOutputPattern(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "docs", "README.md")
	if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
		t.Fatalf("make docs dir: %v", err)
	}
	if err := os.WriteFile(source, []byte("# Demo\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	outputDir := filepath.Join(dir, "docs", "i18n")

	output, err := captureStdout(t, func() error {
		return run([]string{
			"translate",
			"--source", source,
			"--targets", "zh,ja",
			"--output-dir", outputDir,
			"--output-pattern", "{sourceBase}.{target}{sourceExt}",
			"--dry-run",
		})
	})
	if err != nil {
		t.Fatalf("dry-run with output pattern: %v", err)
	}

	for _, want := range []string{
		filepath.Join(outputDir, "README.zh.md"),
		filepath.Join(outputDir, "README.ja.md"),
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing planned path %q:\n%s", want, output)
		}
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

func TestWorkflowSupportsGitLabPlatform(t *testing.T) {
	output, err := captureStdout(t, func() error {
		return run([]string{
			"workflow",
			"--platform", "gitlab",
			"--targets", "zh,ja",
			"--name", "Docs translation check",
			"--go-version", "1.24.3",
		})
	})
	if err != nil {
		t.Fatalf("workflow returned error: %v", err)
	}

	want := `Docs translation check:
  image: golang:1.24.3
  rules:
    - if: '$CI_PIPELINE_SOURCE == "merge_request_event"'
    - if: '$CI_PIPELINE_SOURCE == "schedule"'
  script:
    - go install github.com/codecat-ai/readme-lingo/cmd/readme-lingo@latest
    - readme-lingo translate --source README.md --targets zh,ja --output-dir . --check
`
	if output != want {
		t.Fatalf("workflow mismatch\nwant:\n%s\ngot:\n%s", want, output)
	}
}

func TestWorkflowSupportsScheduleAndBranchesFlags(t *testing.T) {
	output, err := captureStdout(t, func() error {
		return run([]string{
			"workflow",
			"--targets", "zh,ja",
			"--schedule", "15 4 * * 2",
			"--branches", "main, release, ,",
		})
	})
	if err != nil {
		t.Fatalf("workflow returned error: %v", err)
	}

	want := `name: readme-lingo stale translations

on:
  pull_request:
    branches:
      - main
      - release
  schedule:
    # Scheduled workflows run on the default branch; this job guard limits scheduled checks.
    - cron: "15 4 * * 2"

jobs:
  readme-lingo:
    if: github.event_name != 'schedule' || github.ref_name == 'main' || github.ref_name == 'release'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.24.3"
      - name: Install readme-lingo
        run: go install github.com/codecat-ai/readme-lingo/cmd/readme-lingo@latest
      - name: Check README translations
        run: readme-lingo translate --source README.md --targets zh,ja --output-dir . --check --github-annotations
`
	if output != want {
		t.Fatalf("workflow mismatch\nwant:\n%s\ngot:\n%s", want, output)
	}
}

func TestWorkflowRejectsInvalidPlatform(t *testing.T) {
	err := run([]string{
		"workflow",
		"--platform", "circle",
		"--targets", "zh",
	})
	if err == nil {
		t.Fatal("expected invalid platform error")
	}
	if err.Error() != `invalid workflow platform "circle": use github or gitlab` {
		t.Fatalf("error = %q", err)
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

func captureOutput(t *testing.T, fn func() error) (string, string, error) {
	t.Helper()
	originalStdout := os.Stdout
	originalStderr := os.Stderr
	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stderr pipe: %v", err)
	}
	os.Stdout = stdoutW
	os.Stderr = stderrW
	defer func() {
		os.Stdout = originalStdout
		os.Stderr = originalStderr
	}()

	runErr := fn()
	if err := stdoutW.Close(); err != nil {
		t.Fatalf("close stdout pipe: %v", err)
	}
	if err := stderrW.Close(); err != nil {
		t.Fatalf("close stderr pipe: %v", err)
	}
	stdoutData, readErr := io.ReadAll(stdoutR)
	if readErr != nil {
		t.Fatalf("read stdout pipe: %v", readErr)
	}
	stderrData, readErr := io.ReadAll(stderrR)
	if readErr != nil {
		t.Fatalf("read stderr pipe: %v", readErr)
	}
	if err := stdoutR.Close(); err != nil {
		t.Fatalf("close stdout reader: %v", err)
	}
	if err := stderrR.Close(); err != nil {
		t.Fatalf("close stderr reader: %v", err)
	}
	return string(stdoutData), string(stderrData), runErr
}
