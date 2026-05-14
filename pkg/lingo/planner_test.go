package lingo

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestPlanOutputsSingleTargetWithExplicitOutput(t *testing.T) {
	plans, err := PlanOutputs("README.md", []string{"zh"}, "docs/README.zh.md", "")
	if err != nil {
		t.Fatalf("PlanOutputs returned error: %v", err)
	}
	if len(plans) != 1 {
		t.Fatalf("expected 1 plan, got %d", len(plans))
	}
	if plans[0].Target != "zh" {
		t.Fatalf("target = %q, want zh", plans[0].Target)
	}
	if plans[0].OutputPath != filepath.Clean("docs/README.zh.md") {
		t.Fatalf("output = %q", plans[0].OutputPath)
	}
}

func TestPlanOutputsMultipleTargetsUseOutputDirAndDefaultNames(t *testing.T) {
	plans, err := PlanOutputs("README.md", []string{"zh", "ja", "fr"}, "", "translations")
	if err != nil {
		t.Fatalf("PlanOutputs returned error: %v", err)
	}
	want := []string{
		filepath.Clean("translations/README-zh.md"),
		filepath.Clean("translations/README-ja.md"),
		filepath.Clean("translations/README-fr.md"),
	}
	for i := range want {
		if plans[i].OutputPath != want[i] {
			t.Fatalf("plan %d output = %q, want %q", i, plans[i].OutputPath, want[i])
		}
	}
}

func TestPlanOutputsWithPatternUsesSourcePlaceholders(t *testing.T) {
	plans, err := PlanOutputsWithPattern("docs/guide.en.md", []string{"zh", "ja"}, "", "docs/i18n", "{sourceBase}.{target}{sourceExt}")
	if err != nil {
		t.Fatalf("PlanOutputsWithPattern returned error: %v", err)
	}
	want := []string{
		filepath.Clean("docs/i18n/guide.en.zh.md"),
		filepath.Clean("docs/i18n/guide.en.ja.md"),
	}
	for i := range want {
		if plans[i].OutputPath != want[i] {
			t.Fatalf("plan %d output = %q, want %q", i, plans[i].OutputPath, want[i])
		}
	}
}

func TestPlanOutputsWithPatternRejectsInvalidPatterns(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		want    string
	}{
		{name: "empty", pattern: "", want: "empty"},
		{name: "missing target", pattern: "{sourceBase}{sourceExt}", want: "{target}"},
		{name: "unsafe generated name", pattern: "{target}/README.md", want: "target \"zh\""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := PlanOutputsWithPattern("README.md", []string{"zh"}, "", "translations", tt.pattern)
			if err == nil {
				t.Fatal("expected pattern error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %q, want it to contain %q", err, tt.want)
			}
		})
	}
}

func TestPlanOutputsRejectsSingleOutputForMultipleTargets(t *testing.T) {
	_, err := PlanOutputs("README.md", []string{"zh", "ja"}, "README-translated.md", "")
	if err == nil {
		t.Fatal("expected error for --output with multiple targets")
	}
}

func TestSplitTargetsAcceptsTagsAndLanguageNames(t *testing.T) {
	got, err := SplitTargets("zh-Hans, Japanese, Brazilian Portuguese")
	if err != nil {
		t.Fatalf("SplitTargets returned error: %v", err)
	}
	want := []string{"zh-Hans", "Japanese", "Brazilian Portuguese"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("target %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestPlanMarkdownChunksSplitsOnHeadingBoundaries(t *testing.T) {
	source := strings.Join([]string{
		"---",
		"title: Demo",
		"---",
		"",
		"Intro paragraph.",
		"",
		"# Install",
		"Install instructions.",
		"",
		"## Usage",
		"Usage instructions.",
		"",
		"## API",
		"API details.",
		"",
	}, "\n")

	chunks, err := PlanMarkdownChunks(source, 40)
	if err != nil {
		t.Fatalf("PlanMarkdownChunks returned error: %v", err)
	}
	if len(chunks) != 3 {
		t.Fatalf("chunks = %d, want 3: %#v", len(chunks), chunks)
	}
	want := []string{
		"---\ntitle: Demo\n---\n\nIntro paragraph.\n\n# Install\nInstall instructions.",
		"## Usage\nUsage instructions.",
		"## API\nAPI details.",
	}
	for i := range want {
		if chunks[i].Markdown != want[i] {
			t.Fatalf("chunk %d = %q, want %q", i, chunks[i].Markdown, want[i])
		}
	}
}

func TestPlanMarkdownChunksDoesNotSplitHeadingInsideFencedCode(t *testing.T) {
	source := strings.Join([]string{
		"# Demo",
		"```md",
		"# Not a heading boundary",
		"```",
		"",
		"## Next",
		"Text.",
	}, "\n")

	chunks, err := PlanMarkdownChunks(source, 25)
	if err != nil {
		t.Fatalf("PlanMarkdownChunks returned error: %v", err)
	}
	if len(chunks) != 2 {
		t.Fatalf("chunks = %d, want 2: %#v", len(chunks), chunks)
	}
	if strings.Contains(chunks[1].Markdown, "Not a heading boundary") {
		t.Fatalf("fenced heading was split as a boundary: %#v", chunks)
	}
}

func TestPlanMarkdownChunksKeepsOversizedSectionTogether(t *testing.T) {
	longSection := "# Huge\n" + strings.Repeat("long prose ", 12)
	source := longSection + "\n\n## Small\ntext\n"

	chunks, err := PlanMarkdownChunks(source, 40)
	if err != nil {
		t.Fatalf("PlanMarkdownChunks returned error: %v", err)
	}
	if len(chunks) != 2 {
		t.Fatalf("chunks = %d, want 2: %#v", len(chunks), chunks)
	}
	if chunks[0].Markdown != strings.TrimRight(longSection, "\n") {
		t.Fatalf("oversized section was changed: %q", chunks[0].Markdown)
	}
}

func TestPlanMarkdownChunksRejectsNonPositiveMaxChars(t *testing.T) {
	_, err := PlanMarkdownChunks("# Demo\n", 0)
	if err == nil {
		t.Fatal("expected max chars error")
	}
	if !strings.Contains(err.Error(), "max chars must be positive") {
		t.Fatalf("error = %q, want clear max chars message", err)
	}
}
