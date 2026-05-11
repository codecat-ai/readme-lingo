package lingo

import (
	"strings"
	"testing"
)

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

func TestRenderWorkflowUsesScheduleAndGitHubBranches(t *testing.T) {
	got, err := RenderWorkflow(WorkflowOptions{
		Targets:  []string{"zh", "ja"},
		Schedule: "30 2 * * 1",
		Branches: []string{"main", "release"},
	})
	if err != nil {
		t.Fatalf("RenderWorkflow returned error: %v", err)
	}

	want := `name: readme-lingo stale translations

on:
  pull_request:
    branches:
      - main
      - release
  schedule:
    # Scheduled workflows run on the default branch; this job guard limits scheduled checks.
    - cron: "30 2 * * 1"

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
	if got != want {
		t.Fatalf("workflow mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestRenderWorkflowIgnoresEmptyBranches(t *testing.T) {
	got, err := RenderWorkflow(WorkflowOptions{
		Targets:  []string{"zh"},
		Branches: []string{" ", ""},
	})
	if err != nil {
		t.Fatalf("RenderWorkflow returned error: %v", err)
	}

	if strings.Contains(got, "branches:") || strings.Contains(got, "github.event_name") {
		t.Fatalf("empty branches should not add filters:\n%s", got)
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

func TestRenderWorkflowUsesScheduleAndGitLabBranches(t *testing.T) {
	got, err := RenderWorkflow(WorkflowOptions{
		Platform: "gitlab",
		Targets:  []string{"zh", "ja"},
		Schedule: "30 2 * * 1",
		Branches: []string{"main", "release"},
	})
	if err != nil {
		t.Fatalf("RenderWorkflow returned error: %v", err)
	}

	want := `readme_lingo:
  image: golang:1.24.3
  rules:
    - if: '$CI_PIPELINE_SOURCE == "merge_request_event" && ($CI_MERGE_REQUEST_TARGET_BRANCH_NAME == "main" || $CI_MERGE_REQUEST_TARGET_BRANCH_NAME == "release")'
    # Configure the GitLab pipeline schedule cron as: 30 2 * * 1
    - if: '$CI_PIPELINE_SOURCE == "schedule" && ($CI_COMMIT_BRANCH == "main" || $CI_COMMIT_BRANCH == "release")'
  script:
    - go install github.com/codecat-ai/readme-lingo/cmd/readme-lingo@latest
    - readme-lingo translate --source README.md --targets zh,ja --output-dir . --check
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
