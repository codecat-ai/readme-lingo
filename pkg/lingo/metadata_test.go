package lingo

import (
	"strings"
	"testing"
	"time"
)

func TestMetadataRoundTripAndSynchronization(t *testing.T) {
	source := []byte("# Demo\n\nHello world.\n")
	digest := SourceDigest(source)
	generated := time.Date(2026, 5, 5, 10, 30, 0, 0, time.UTC)

	withMarker := AddMetadata("translated body", Metadata{
		SourcePath:  "README.md",
		Target:      "zh",
		Model:       "google/gemma-4-26b-a4b-it:free",
		SourceHash:  digest,
		GeneratedAt: generated,
	})

	meta, ok := ParseMetadata([]byte(withMarker))
	if !ok {
		t.Fatal("expected metadata marker")
	}
	if meta.SourceHash != digest || meta.SourcePath != "README.md" || meta.Target != "zh" {
		t.Fatalf("unexpected metadata: %+v", meta)
	}
	if !IsSynchronized(source, []byte(withMarker)) {
		t.Fatal("expected translated file to be synchronized")
	}
	if IsSynchronized([]byte("# Demo\n\nChanged.\n"), []byte(withMarker)) {
		t.Fatal("expected changed source to be stale")
	}
}

func TestInsertSwitcherReplacesExistingTopSwitcher(t *testing.T) {
	body := "<!-- readme-lingo-switcher:start -->\nold\n<!-- readme-lingo-switcher:end -->\n\n# Title\n"
	switcher, err := BuildSwitcher("en:README.md,zh:README-zh.md")
	if err != nil {
		t.Fatalf("BuildSwitcher returned error: %v", err)
	}
	got := InsertSwitcher(body, switcher)
	if strings.Contains(got, "old") {
		t.Fatalf("old switcher was not replaced: %s", got)
	}
	if !strings.HasPrefix(got, "<!-- readme-lingo-switcher:start -->") {
		t.Fatalf("switcher not inserted at top: %s", got)
	}
	if !strings.Contains(got, "[en](README.md)") || !strings.Contains(got, "[zh](README-zh.md)") {
		t.Fatalf("switcher links missing: %s", got)
	}
	if !strings.Contains(got, "\n\n# Title\n") {
		t.Fatalf("switcher did not preserve spacing before body: %s", got)
	}
}
