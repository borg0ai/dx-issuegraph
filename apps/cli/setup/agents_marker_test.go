package setup

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/borg0ai/dx-issuegraph/internal/templates/agents"
)

func TestContainsIssueGraphMarkerLegacy(t *testing.T) {
	content := "# Header\n\n<!-- BEGIN ISSUEGRAPH INTEGRATION -->\nContent\n<!-- END ISSUEGRAPH INTEGRATION -->\n"
	if !containsIssueGraphMarker(content) {
		t.Error("should detect legacy marker")
	}
}

func TestContainsIssueGraphMarkerNew(t *testing.T) {
	content := "# Header\n\n<!-- BEGIN ISSUEGRAPH INTEGRATION profile:full hash:a1b2c3d4 -->\nContent\n<!-- END ISSUEGRAPH INTEGRATION -->\n"
	if !containsIssueGraphMarker(content) {
		t.Error("should detect new-format marker")
	}
}

func TestContainsIssueGraphMarkerAbsent(t *testing.T) {
	content := "# Header\n\nNo issuegraph here\n"
	if containsIssueGraphMarker(content) {
		t.Error("should not detect marker when absent")
	}
}

func TestUpdateIssueGraphSectionLegacyToNew(t *testing.T) {
	// Legacy marker should be replaced with new-format marker
	legacy := `# Header

<!-- BEGIN ISSUEGRAPH INTEGRATION -->
Old content here
<!-- END ISSUEGRAPH INTEGRATION -->

# Footer`

	updated := updateIssueGraphSection(legacy)

	// Should have new-format marker with profile and hash
	if !strings.Contains(updated, "profile:full") {
		t.Error("updated section should use new-format marker with profile:full")
	}
	if !strings.Contains(updated, "hash:") {
		t.Error("updated section should use new-format marker with hash")
	}

	// Should preserve surrounding content
	if !strings.Contains(updated, "# Header") {
		t.Error("should preserve header")
	}
	if !strings.Contains(updated, "# Footer") {
		t.Error("should preserve footer")
	}

	// Old content should be replaced
	if strings.Contains(updated, "Old content here") {
		t.Error("old content should be replaced")
	}
}

func TestUpdateIssueGraphSectionNewFormatUpdate(t *testing.T) {
	// New-format marker should also be replaceable
	content := `# Header

<!-- BEGIN ISSUEGRAPH INTEGRATION profile:full hash:oldoldhash -->
Stale content
<!-- END ISSUEGRAPH INTEGRATION -->

# Footer`

	updated := updateIssueGraphSection(content)

	if strings.Contains(updated, "oldoldhash") {
		t.Error("old hash should be replaced")
	}
	if strings.Contains(updated, "Stale content") {
		t.Error("stale content should be replaced")
	}
	if !strings.Contains(updated, "# Header") || !strings.Contains(updated, "# Footer") {
		t.Error("surrounding content should be preserved")
	}
}

func TestRemoveIssueGraphSectionLegacy(t *testing.T) {
	content := "Header\n<!-- BEGIN ISSUEGRAPH INTEGRATION -->\nContent\n<!-- END ISSUEGRAPH INTEGRATION -->\nFooter"
	result := removeIssueGraphSection(content)
	if strings.Contains(result, "BEGIN ISSUEGRAPH") {
		t.Error("markers should be removed")
	}
	if !strings.Contains(result, "Header") || !strings.Contains(result, "Footer") {
		t.Error("surrounding content should be preserved")
	}
}

func TestRemoveIssueGraphSectionNewFormat(t *testing.T) {
	content := "Header\n<!-- BEGIN ISSUEGRAPH INTEGRATION profile:full hash:a1b2c3d4 -->\nContent\n<!-- END ISSUEGRAPH INTEGRATION -->\nFooter"
	result := removeIssueGraphSection(content)
	if strings.Contains(result, "BEGIN ISSUEGRAPH") {
		t.Error("markers should be removed")
	}
	if !strings.Contains(result, "Header") || !strings.Contains(result, "Footer") {
		t.Error("surrounding content should be preserved")
	}
}

func TestUpdateIssueGraphSectionWithProfile(t *testing.T) {
	// Test with explicit profile parameter
	content := `# Header

<!-- BEGIN ISSUEGRAPH INTEGRATION -->
Old content
<!-- END ISSUEGRAPH INTEGRATION -->

# Footer`

	updated := updateIssueGraphSectionWithProfile(content, agents.ProfileMinimal)
	if !strings.Contains(updated, "profile:minimal") {
		t.Error("should use minimal profile")
	}
	if !strings.Contains(updated, "# Header") || !strings.Contains(updated, "# Footer") {
		t.Error("should preserve surrounding content")
	}
}

func TestInstallAgentsWithProfileCreatesNew(t *testing.T) {
	env, _, _ := newFactoryTestEnv(t)
	integration := agentsIntegration{
		name:         "TestAgent",
		setupCommand: "issuegraph setup testagent",
		profile:      agents.ProfileMinimal,
	}
	if err := installAgents(env, integration); err != nil {
		t.Fatalf("installAgents returned error: %v", err)
	}
	data, err := readFileBytes(env.agentsPath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "profile:minimal") {
		t.Error("new file should use minimal profile from integration")
	}
}

func TestInstallAgentsDefaultsToFullProfile(t *testing.T) {
	env, _, _ := newFactoryTestEnv(t)
	integration := agentsIntegration{
		name:         "TestAgent",
		setupCommand: "issuegraph setup testagent",
		// no profile set — should default to full
	}
	if err := installAgents(env, integration); err != nil {
		t.Fatalf("installAgents returned error: %v", err)
	}
	data, err := readFileBytes(env.agentsPath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "profile:full") {
		t.Error("default profile should be full")
	}
}

// readFileBytes is a test helper to read file content
func readFileBytes(path string) ([]byte, error) {
	return readFileBytesImpl(path)
}

func TestExistingIssueGraphProfileLegacy(t *testing.T) {
	content := "# Header\n<!-- BEGIN ISSUEGRAPH INTEGRATION -->\nContent\n<!-- END ISSUEGRAPH INTEGRATION -->\n"
	got := existingIssueGraphProfile(content)
	if got != agents.ProfileFull {
		t.Errorf("legacy marker should return ProfileFull, got %q", got)
	}
}

func TestExistingIssueGraphProfileFull(t *testing.T) {
	content := "<!-- BEGIN ISSUEGRAPH INTEGRATION profile:full hash:abcd1234 -->\nContent\n<!-- END ISSUEGRAPH INTEGRATION -->\n"
	got := existingIssueGraphProfile(content)
	if got != agents.ProfileFull {
		t.Errorf("expected ProfileFull, got %q", got)
	}
}

func TestExistingIssueGraphProfileMinimal(t *testing.T) {
	content := "<!-- BEGIN ISSUEGRAPH INTEGRATION profile:minimal hash:deadbeef -->\nContent\n<!-- END ISSUEGRAPH INTEGRATION -->\n"
	got := existingIssueGraphProfile(content)
	if got != agents.ProfileMinimal {
		t.Errorf("expected ProfileMinimal, got %q", got)
	}
}

func TestExistingIssueGraphProfileNoMarker(t *testing.T) {
	content := "# Just a file\nNo markers\n"
	got := existingIssueGraphProfile(content)
	if got != agents.ProfileFull {
		t.Errorf("no marker should default to ProfileFull, got %q", got)
	}
}

func TestCheckAgentsDetectsStale(t *testing.T) {
	env, stdout, _ := newFactoryTestEnv(t)
	// Write a section with a bogus hash so it's stale
	content := "<!-- BEGIN ISSUEGRAPH INTEGRATION profile:full hash:00000000 -->\nOld content\n<!-- END ISSUEGRAPH INTEGRATION -->\n"
	if err := os.WriteFile(env.agentsPath, []byte(content), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	integration := agentsIntegration{
		name:         "TestAgent",
		setupCommand: "issuegraph setup testagent",
		profile:      agents.ProfileFull,
	}
	err := checkAgents(env, integration)
	if !errors.Is(err, errIssueGraphSectionStale) {
		t.Fatalf("expected errIssueGraphSectionStale, got %v", err)
	}
	if !strings.Contains(stdout.String(), "stale") {
		t.Error("expected stale message in stdout")
	}
}

func TestCheckAgentsCurrent(t *testing.T) {
	stubDetectRenderOpts(t)

	env, stdout, _ := newFactoryTestEnv(t)
	section := agents.RenderSection(agents.ProfileFull)
	if err := os.WriteFile(env.agentsPath, []byte(section), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	integration := agentsIntegration{
		name:         "TestAgent",
		setupCommand: "issuegraph setup testagent",
		profile:      agents.ProfileFull,
	}
	if err := checkAgents(env, integration); err != nil {
		t.Fatalf("expected nil error for current section, got %v", err)
	}
	if !strings.Contains(stdout.String(), "current") {
		t.Error("expected (current) in output")
	}
}

func TestCheckAgentsMinimalAcceptsFullProfile(t *testing.T) {
	stubDetectRenderOpts(t)

	env, _, _ := newFactoryTestEnv(t)
	section := agents.RenderSection(agents.ProfileFull)
	if err := os.WriteFile(env.agentsPath, []byte(section), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	integration := agentsIntegration{
		name:         "ClaudeCode",
		setupCommand: "issuegraph setup claude",
		profile:      agents.ProfileMinimal,
	}
	if err := checkAgents(env, integration); err != nil {
		t.Fatalf("expected full profile to be accepted for minimal integration, got %v", err)
	}
}

func TestCheckAgentsMissingUsesTargetFileName(t *testing.T) {
	stdout := &bytes.Buffer{}
	env := agentsEnv{
		agentsPath: filepath.Join(t.TempDir(), "CLAUDE.md"),
		stdout:     stdout,
		stderr:     &bytes.Buffer{},
	}
	integration := agentsIntegration{name: "ClaudeCode", setupCommand: "issuegraph setup claude", profile: agents.ProfileMinimal}
	err := checkAgents(env, integration)
	if !errors.Is(err, errAgentsFileMissing) {
		t.Fatalf("expected errAgentsFileMissing, got %v", err)
	}
	if !strings.Contains(stdout.String(), "CLAUDE.md not found") {
		t.Fatalf("expected target filename in output, got: %s", stdout.String())
	}
}

func TestInstallAgentsPreservesFullProfile(t *testing.T) {
	// Simulate: file already has full profile, requesting minimal install
	env, stdout, _ := newFactoryTestEnv(t)
	fullSection := agents.RenderSection(agents.ProfileFull)
	if err := os.WriteFile(env.agentsPath, []byte(fullSection), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	integration := agentsIntegration{
		name:         "MinimalAgent",
		setupCommand: "issuegraph setup minimalagent",
		profile:      agents.ProfileMinimal,
	}
	if err := installAgents(env, integration); err != nil {
		t.Fatalf("installAgents: %v", err)
	}
	data, err := readFileBytes(env.agentsPath)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	content := string(data)
	// Should preserve full profile, not downgrade to minimal
	if !strings.Contains(content, "profile:full") {
		t.Error("should preserve full profile when minimal requested on file with full")
	}
	if !strings.Contains(stdout.String(), "preserving") {
		t.Error("expected informational message about preserving full profile")
	}
}

func TestInstallAgentsSymlinkSafety(t *testing.T) {
	dir := t.TempDir()
	realFile := filepath.Join(dir, "AGENTS.md")
	linkPath := filepath.Join(dir, "CLAUDE.md")

	// Seed the target with content that should remain unchanged.
	original := "# Existing target content\n"
	if err := os.WriteFile(realFile, []byte(original), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Create symlink
	if err := os.Symlink(realFile, linkPath); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	env := agentsEnv{
		agentsPath: linkPath, // install targets the symlink
		stdout:     stdout,
		stderr:     stderr,
	}
	integration := agentsIntegration{
		name:         "ClaudeCode",
		setupCommand: "issuegraph setup claude",
		profile:      agents.ProfileMinimal,
	}
	if err := installAgents(env, integration); err != nil {
		t.Fatalf("installAgents via symlink: %v", err)
	}

	// Symlink should still be a symlink.
	if info, err := os.Lstat(linkPath); err != nil {
		t.Fatalf("lstat symlink: %v", err)
	} else if info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("expected CLAUDE.md to remain a symlink")
	}

	// Read the real file — should be untouched (no managed section injection).
	data, err := os.ReadFile(realFile)
	if err != nil {
		t.Fatalf("read real file: %v", err)
	}
	if string(data) != original {
		t.Errorf("symlink target changed:\n got: %q\nwant: %q", string(data), original)
	}
	if strings.Contains(string(data), "BEGIN ISSUEGRAPH INTEGRATION") {
		t.Error("symlink target should not receive managed section")
	}
	if stdout.String() != "Installing ClaudeCode integration...\n" {
		t.Errorf("expected stdout to contain only install start, got: %q", stdout.String())
	}
	stderrText := stderr.String()
	if !strings.Contains(stderrText, "CLAUDE.md is a symlink to "+realFile) {
		t.Errorf("expected warning to include symlink target, got: %s", stderrText)
	}
	if !strings.Contains(stderrText, "re-run 'issuegraph setup claude'") {
		t.Errorf("expected warning to include corrective command, got: %s", stderrText)
	}
}

func TestInstallAgentsInspectError(t *testing.T) {
	dir := t.TempDir()
	notDir := filepath.Join(dir, "not-dir")
	if err := os.WriteFile(notDir, []byte("not a directory"), 0644); err != nil {
		t.Fatalf("write sentinel file: %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	env := agentsEnv{
		agentsPath: filepath.Join(notDir, "AGENTS.md"),
		stdout:     stdout,
		stderr:     stderr,
	}
	integration := agentsIntegration{
		name:         "TestAgent",
		setupCommand: "issuegraph setup testagent",
		profile:      agents.ProfileMinimal,
	}

	err := installAgents(env, integration)
	if err == nil {
		t.Fatal("expected inspect error")
	}
	if os.IsNotExist(err) {
		t.Fatalf("expected non-not-exist error, got: %v", err)
	}
	if !strings.Contains(stderr.String(), "Error: failed to inspect") {
		t.Errorf("expected inspect error on stderr, got: %s", stderr.String())
	}
}

func TestLegacyToNewMigrationViaInstall(t *testing.T) {
	env, _, _ := newFactoryTestEnv(t)
	// Seed with legacy markers
	legacy := "# Header\n\n<!-- BEGIN ISSUEGRAPH INTEGRATION -->\nOld content\n<!-- END ISSUEGRAPH INTEGRATION -->\n\n# Footer"
	if err := os.WriteFile(env.agentsPath, []byte(legacy), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	integration := agentsIntegration{
		name:         "Factory.ai",
		setupCommand: "issuegraph setup factory",
		profile:      agents.ProfileFull,
	}
	if err := installAgents(env, integration); err != nil {
		t.Fatalf("installAgents: %v", err)
	}
	data, err := readFileBytes(env.agentsPath)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	content := string(data)
	// Should now have versioned markers
	if !strings.Contains(content, "profile:full") {
		t.Error("legacy markers should be upgraded to versioned format")
	}
	if !strings.Contains(content, "hash:") {
		t.Error("upgraded section should contain hash")
	}
	if strings.Contains(content, "Old content") {
		t.Error("old legacy content should be replaced")
	}
	if !strings.Contains(content, "# Header") || !strings.Contains(content, "# Footer") {
		t.Error("surrounding content should be preserved")
	}
}
