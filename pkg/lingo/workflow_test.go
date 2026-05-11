package lingo

import "testing"

func TestRenderWorkflowUsesDefaultsAndTargets(t *testing.T) {
	got, err := RenderWorkflow(WorkflowOptions{
		Targets: []string{"zh", "ja"},
	})
	if err != nil {
		t.Fatalf("RenderWorkflow returned error: %v", err)
	}

	want := `name: readme-lingo stale translations

on:
  pull_request:
  schedule:
    - cron: "0 0 * * 1"

jobs:
  readme-lingo:
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
	if got != want {
		t.Fatalf("workflow mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestRenderWorkflowUsesCustomOptions(t *testing.T) {
	got, err := RenderWorkflow(WorkflowOptions{
		Name:      "Docs translation check",
		Source:    "docs/README.md",
		Targets:   []string{"fr", "de"},
		OutputDir: "docs/translations",
		GoVersion: "1.22.x",
	})
	if err != nil {
		t.Fatalf("RenderWorkflow returned error: %v", err)
	}

	want := `name: Docs translation check

on:
  pull_request:
  schedule:
    - cron: "0 0 * * 1"

jobs:
  readme-lingo:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.22.x"
      - name: Install readme-lingo
        run: go install github.com/codecat-ai/readme-lingo/cmd/readme-lingo@latest
      - name: Check README translations
        run: readme-lingo translate --source docs/README.md --targets fr,de --output-dir docs/translations --check --github-annotations
`
	if got != want {
		t.Fatalf("workflow mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestRenderWorkflowUsesGitLabPlatform(t *testing.T) {
	got, err := RenderWorkflow(WorkflowOptions{
		Platform:  "gitlab",
		Source:    "docs/README guide.md",
		Targets:   []string{"zh", "ja"},
		OutputDir: "docs/translations",
		GoVersion: "1.24.3",
	})
	if err != nil {
		t.Fatalf("RenderWorkflow returned error: %v", err)
	}

	want := `readme_lingo:
  image: golang:1.24.3
  rules:
    - if: '$CI_PIPELINE_SOURCE == "merge_request_event"'
    - if: '$CI_PIPELINE_SOURCE == "schedule"'
  script:
    - go install github.com/codecat-ai/readme-lingo/cmd/readme-lingo@latest
    - readme-lingo translate --source 'docs/README guide.md' --targets zh,ja --output-dir docs/translations --check
`
	if got != want {
		t.Fatalf("workflow mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestRenderWorkflowRejectsInvalidPlatform(t *testing.T) {
	_, err := RenderWorkflow(WorkflowOptions{
		Platform: "circle",
		Targets:  []string{"zh"},
	})
	if err == nil {
		t.Fatal("expected invalid platform error")
	}
	if err.Error() != `invalid workflow platform "circle": use github or gitlab` {
		t.Fatalf("error = %q", err)
	}
}
