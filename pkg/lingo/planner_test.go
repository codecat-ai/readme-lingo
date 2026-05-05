package lingo

import (
	"path/filepath"
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
