package agents

import (
	"strings"
	"testing"
)

func TestEmbeddedDefault(t *testing.T) {
	content := EmbeddedDefault()

	if content == "" {
		t.Fatal("EmbeddedDefault() returned empty string")
	}

	required := []string{
		"# Agent Instructions",
		"## Quick Reference",
		"issuegraph prime",
		"BEGIN ISSUEGRAPH INTEGRATION",
		"END ISSUEGRAPH INTEGRATION",
		"## Session Completion",
		"git push",
	}
	for _, want := range required {
		if !strings.Contains(content, want) {
			t.Errorf("EmbeddedDefault() missing %q", want)
		}
	}
}

// TestEmbeddedDefaultArchitectureSummary guards GH#3683: agents reading the
// generated AGENTS.md need an architecture statement at the top so they don't
// build wrong mental models (treating JSONL as source of truth, manually
// running issuegraph import, etc.) and have to discover the four deeper architecture
// docs the hard way. The summary must appear before the Quick Reference and
// link to the canonical sync-concepts entry-point.
func TestEmbeddedDefaultArchitectureSummary(t *testing.T) {
	content := EmbeddedDefault()

	required := []string{
		"Architecture in one line",
		"refs/dolt/data",
		"passive export",
		"docs/core-concepts/sync-concepts.md",
	}
	for _, want := range required {
		if !strings.Contains(content, want) {
			t.Errorf("EmbeddedDefault() missing architecture-summary fragment %q", want)
		}
	}

	archIdx := strings.Index(content, "Architecture in one line")
	quickRefIdx := strings.Index(content, "## Quick Reference")
	if archIdx == -1 || quickRefIdx == -1 {
		t.Fatal("missing required anchors")
	}
	if archIdx > quickRefIdx {
		t.Error("architecture summary should appear before Quick Reference (so agents see it first)")
	}
}

func TestEmbeddedIssueGraphSection(t *testing.T) {
	section := EmbeddedIssueGraphSection()

	if section == "" {
		t.Fatal("EmbeddedIssueGraphSection() returned empty string")
	}

	if !strings.HasPrefix(section, "<!-- BEGIN ISSUEGRAPH INTEGRATION -->") {
		t.Error("issuegraph section should start with begin marker")
	}

	trimmed := strings.TrimSpace(section)
	if !strings.HasSuffix(trimmed, "<!-- END ISSUEGRAPH INTEGRATION -->") {
		t.Error("issuegraph section should end with end marker")
	}

	required := []string{
		"issuegraph create",
		"issuegraph update",
		"issuegraph close",
		"issuegraph ready",
		"discovered-from",
	}
	for _, want := range required {
		if !strings.Contains(section, want) {
			t.Errorf("EmbeddedIssueGraphSection() missing %q", want)
		}
	}
}

func TestIssueGraphSectionContainsLanding(t *testing.T) {
	section := EmbeddedIssueGraphSection()
	if !strings.Contains(section, "Session Completion") {
		t.Error("issuegraph section should contain session completion content within markers")
	}
}

func TestDefaultContainsBothSections(t *testing.T) {
	content := EmbeddedDefault()

	issuegraphIdx := strings.Index(content, "BEGIN ISSUEGRAPH INTEGRATION")
	completionIdx := strings.Index(content, "Session Completion")

	if issuegraphIdx == -1 {
		t.Fatal("missing issuegraph integration section")
	}
	if completionIdx == -1 {
		t.Fatal("missing session completion section")
	}
	if issuegraphIdx > completionIdx {
		t.Error("issuegraph section should come before session completion section")
	}
}

func TestEmbeddedDefaultManagedMarkerIsCurrent(t *testing.T) {
	content := EmbeddedDefault()

	idx := strings.Index(content, "<!-- BEGIN ISSUEGRAPH INTEGRATION")
	if idx == -1 {
		t.Fatal("missing managed section marker")
	}
	line := content[idx:]
	if nl := strings.Index(line, "\n"); nl != -1 {
		line = line[:nl]
	}

	meta := ParseMarker(line)
	if meta == nil {
		t.Fatalf("failed to parse managed marker %q", line)
	}
	if meta.Version != MarkerVersion {
		t.Errorf("marker version = %d, want %d", meta.Version, MarkerVersion)
	}
	if meta.Profile != ProfileFull {
		t.Errorf("marker profile = %q, want %q", meta.Profile, ProfileFull)
	}
	if meta.Hash != CurrentHash(ProfileFull) {
		t.Errorf("marker hash = %q, want %q", meta.Hash, CurrentHash(ProfileFull))
	}
}
