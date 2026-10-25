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

func TestUpdateIssueGraphSection(t *testing.T) {
	issuegraphSection := agents.RenderSection(agents.ProfileFull)

	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name: "replace existing section",
			content: `# My Project

Some content

<!-- BEGIN ISSUEGRAPH INTEGRATION -->
Old content here
<!-- END ISSUEGRAPH INTEGRATION -->

More content after`,
			expected: `# My Project

Some content

` + issuegraphSection + `
More content after`,
		},
		{
			name:     "no markers returns content unchanged",
			content:  "# My Project\n\nSome content",
			expected: "# My Project\n\nSome content",
		},
		{
			name: "handle section at end of file",
			content: `# My Project

<!-- BEGIN ISSUEGRAPH INTEGRATION -->
Old content
<!-- END ISSUEGRAPH INTEGRATION -->`,
			expected: `# My Project

` + issuegraphSection,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := updateIssueGraphSection(tt.content)
			if got != tt.expected {
				t.Errorf("updateIssueGraphSection() mismatch\ngot:\n%s\nwant:\n%s", got, tt.expected)
			}
		})
	}
}

func TestRemoveIssueGraphSection(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name: "remove section in middle",
			content: `# My Project

<!-- BEGIN ISSUEGRAPH INTEGRATION -->
IssueGraph content
<!-- END ISSUEGRAPH INTEGRATION -->

More content`,
			expected: `# My Project


More content`,
		},
		{
			name: "remove section at end",
			content: `# My Project

Content

<!-- BEGIN ISSUEGRAPH INTEGRATION -->
IssueGraph content
<!-- END ISSUEGRAPH INTEGRATION -->`,
			expected: `# My Project

Content

`,
		},
		{
			name:     "no markers - return unchanged",
			content:  "# My Project\n\nNo issuegraph section",
			expected: "# My Project\n\nNo issuegraph section",
		},
		{
			name:     "only begin marker - return unchanged",
			content:  "# My Project\n<!-- BEGIN ISSUEGRAPH INTEGRATION -->\nContent",
			expected: "# My Project\n<!-- BEGIN ISSUEGRAPH INTEGRATION -->\nContent",
		},
		{
			name:     "preserve surrounding whitespace and unrelated content",
			content:  "Header\n\n" + agents.EmbeddedIssueGraphSection() + "\n\nFooter\n",
			expected: "Header\n\n\n\nFooter\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := removeIssueGraphSection(tt.content)
			if got != tt.expected {
				t.Errorf("removeIssueGraphSection() mismatch\ngot:\n%q\nwant:\n%q", got, tt.expected)
			}
		})
	}
}

func TestCreateNewAgentsFile(t *testing.T) {
	content := createNewAgentsFile()

	// Verify it contains required elements
	if !strings.Contains(content, "# Project Instructions for AI Agents") {
		t.Error("Missing header in new agents file")
	}

	if !containsIssueGraphMarker(content) {
		t.Error("Missing begin marker in new agents file")
	}

	if !strings.Contains(content, agentsEndMarker) {
		t.Error("Missing end marker in new agents file")
	}

	if !strings.Contains(content, "profile:full") {
		t.Error("Missing profile metadata in new agents file")
	}

	if !strings.Contains(content, "## Build & Test") {
		t.Error("Missing Build & Test section")
	}

	if !strings.Contains(content, "## Architecture Overview") {
		t.Error("Missing Architecture Overview section")
	}
}

func newFactoryTestEnv(t *testing.T) (factoryEnv, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	dir := t.TempDir()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	return factoryEnv{
		agentsPath: filepath.Join(dir, "AGENTS.md"),
		stdout:     stdout,
		stderr:     stderr,
	}, stdout, stderr
}

func stubFactoryEnvProvider(t *testing.T, env factoryEnv) {
	t.Helper()
	orig := factoryEnvProvider
	factoryEnvProvider = func() factoryEnv {
		return env
	}
	t.Cleanup(func() { factoryEnvProvider = orig })
}

func TestInstallFactoryCreatesNewFile(t *testing.T) {
	env, stdout, _ := newFactoryTestEnv(t)
	if err := installFactory(env); err != nil {
		t.Fatalf("installFactory returned error: %v", err)
	}
	data, err := os.ReadFile(env.agentsPath)
	if err != nil {
		t.Fatalf("failed to read AGENTS.md: %v", err)
	}
	content := string(data)
	if !containsIssueGraphMarker(content) || !strings.Contains(content, agentsEndMarker) {
		t.Fatal("missing factory markers in new file")
	}
	if !strings.Contains(stdout.String(), "Factory.ai (Droid) integration installed") {
		t.Error("expected success message in stdout")
	}
}

func TestInstallFactoryUpdatesExistingSection(t *testing.T) {
	env, _, _ := newFactoryTestEnv(t)
	initial := `# Header

<!-- BEGIN ISSUEGRAPH INTEGRATION -->
Old content
<!-- END ISSUEGRAPH INTEGRATION -->

# Footer`
	if err := os.WriteFile(env.agentsPath, []byte(initial), 0644); err != nil {
		t.Fatalf("failed to seed AGENTS.md: %v", err)
	}
	if err := installFactory(env); err != nil {
		t.Fatalf("installFactory returned error: %v", err)
	}
	data, err := os.ReadFile(env.agentsPath)
	if err != nil {
		t.Fatalf("failed to read AGENTS.md: %v", err)
	}
	content := string(data)
	if strings.Contains(content, "Old content") {
		t.Error("old issuegraph section should be replaced")
	}
	if !strings.Contains(content, "# Footer") {
		t.Error("content after issuegraph section should remain")
	}
}

func TestInstallFactoryReportsWriteError(t *testing.T) {
	env, _, stderr := newFactoryTestEnv(t)
	if err := os.Mkdir(env.agentsPath, 0o755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}
	if err := installFactory(env); err == nil {
		t.Fatal("expected error when agents path is directory")
	}
	if !strings.Contains(stderr.String(), "failed to read") {
		t.Error("expected error message in stderr")
	}
}

func TestCheckFactoryScenarios(t *testing.T) {
	stubDetectRenderOpts(t)

	t.Run("missing file", func(t *testing.T) {
		env, stdout, _ := newFactoryTestEnv(t)
		err := checkFactory(env)
		if !errors.Is(err, errAgentsFileMissing) {
			t.Fatalf("expected errAgentsFileMissing, got %v", err)
		}
		if !strings.Contains(stdout.String(), "Run: issuegraph setup factory") {
			t.Error("expected guidance message")
		}
	})

	t.Run("missing section", func(t *testing.T) {
		env, stdout, _ := newFactoryTestEnv(t)
		if err := os.WriteFile(env.agentsPath, []byte("# Project"), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}
		err := checkFactory(env)
		if !errors.Is(err, errIssueGraphSectionMissing) {
			t.Fatalf("expected errIssueGraphSectionMissing, got %v", err)
		}
		if !strings.Contains(stdout.String(), "no issuegraph section") {
			t.Error("expected warning output")
		}
	})

	t.Run("success", func(t *testing.T) {
		env, stdout, _ := newFactoryTestEnv(t)
		// Use current rendered section (not legacy EmbeddedIssueGraphSection) so check reports "current"
		issuegraphSection := agents.RenderSection(agents.ProfileFull)
		if err := os.WriteFile(env.agentsPath, []byte(issuegraphSection), 0644); err != nil {
			t.Fatalf("failed to seed file: %v", err)
		}
		if err := checkFactory(env); err != nil {
			t.Fatalf("checkFactory returned error: %v", err)
		}
		if !strings.Contains(stdout.String(), "integration installed") {
			t.Error("expected success output")
		}
	})

	t.Run("stale legacy section", func(t *testing.T) {
		env, stdout, _ := newFactoryTestEnv(t)
		issuegraphSection := agents.EmbeddedIssueGraphSection()
		if err := os.WriteFile(env.agentsPath, []byte(issuegraphSection), 0644); err != nil {
			t.Fatalf("failed to seed file: %v", err)
		}
		err := checkFactory(env)
		if !errors.Is(err, errIssueGraphSectionStale) {
			t.Fatalf("expected errIssueGraphSectionStale, got %v", err)
		}
		if !strings.Contains(stdout.String(), "stale") {
			t.Error("expected stale output")
		}
	})
}

func TestRemoveFactoryScenarios(t *testing.T) {
	t.Run("remove section and keep file", func(t *testing.T) {
		env, stdout, _ := newFactoryTestEnv(t)
		issuegraphSection := agents.EmbeddedIssueGraphSection()
		content := "# Top\n\n" + issuegraphSection + "\n\n# Bottom"
		if err := os.WriteFile(env.agentsPath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to seed AGENTS.md: %v", err)
		}
		if err := removeFactory(env); err != nil {
			t.Fatalf("removeFactory returned error: %v", err)
		}
		data, err := os.ReadFile(env.agentsPath)
		if err != nil {
			t.Fatalf("failed to read AGENTS.md: %v", err)
		}
		if strings.Contains(string(data), agentsBeginMarker) {
			t.Error("issuegraph section should be removed")
		}
		if !strings.Contains(stdout.String(), "Removed issuegraph section") {
			t.Error("expected removal message")
		}
	})

	t.Run("clear file when only issuegraph", func(t *testing.T) {
		env, stdout, _ := newFactoryTestEnv(t)
		issuegraphSection := agents.EmbeddedIssueGraphSection()
		if err := os.WriteFile(env.agentsPath, []byte(issuegraphSection), 0644); err != nil {
			t.Fatalf("failed to seed AGENTS.md: %v", err)
		}
		if err := removeFactory(env); err != nil {
			t.Fatalf("removeFactory returned error: %v", err)
		}
		data, err := os.ReadFile(env.agentsPath)
		if err != nil {
			t.Fatalf("failed to read AGENTS.md after remove: %v", err)
		}
		if strings.TrimSpace(string(data)) != "" {
			t.Fatal("AGENTS.md should remain present but empty when only issuegraph section existed")
		}
		if !strings.Contains(stdout.String(), "Removed issuegraph section") {
			t.Error("expected removal message")
		}
	})

	t.Run("missing file", func(t *testing.T) {
		env, stdout, _ := newFactoryTestEnv(t)
		if err := removeFactory(env); err != nil {
			t.Fatalf("removeFactory returned error: %v", err)
		}
		if !strings.Contains(stdout.String(), "No AGENTS.md file found") {
			t.Error("expected info message for missing file")
		}
	})
}

func TestWrapperReturnsErrorOnFailure(t *testing.T) {
	t.Run("InstallFactory", func(t *testing.T) {
		env := factoryEnv{agentsPath: filepath.Join(t.TempDir(), "dir"), stdout: &bytes.Buffer{}, stderr: &bytes.Buffer{}}
		if err := os.Mkdir(env.agentsPath, 0o755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
		stubFactoryEnvProvider(t, env)
		if err := InstallFactory(); err == nil {
			t.Fatal("InstallFactory should return error")
		}
	})

	t.Run("CheckFactory", func(t *testing.T) {
		env := factoryEnv{agentsPath: filepath.Join(t.TempDir(), "missing"), stdout: &bytes.Buffer{}, stderr: &bytes.Buffer{}}
		stubFactoryEnvProvider(t, env)
		if err := CheckFactory(); err == nil {
			t.Fatal("CheckFactory should return error")
		}
	})

	t.Run("RemoveFactory", func(t *testing.T) {
		env := factoryEnv{agentsPath: filepath.Join(t.TempDir(), "AGENTS.md"), stdout: &bytes.Buffer{}, stderr: &bytes.Buffer{}}
		issuegraphSection := agents.EmbeddedIssueGraphSection()
		if err := os.WriteFile(env.agentsPath, []byte(issuegraphSection), 0644); err != nil {
			t.Fatalf("failed to seed file: %v", err)
		}
		if err := os.Chmod(env.agentsPath, 0o000); err != nil {
			t.Fatalf("failed to chmod file: %v", err)
		}
		stubFactoryEnvProvider(t, env)
		if err := RemoveFactory(); err == nil {
			t.Fatal("RemoveFactory should return error")
		}
	})
}

func TestFactoryIssueGraphSectionContent(t *testing.T) {
	section := agents.EmbeddedIssueGraphSection()
	required := []string{"issuegraph create", "issuegraph update", "issuegraph close", "issuegraph ready", "discovered-from"}
	for _, token := range required {
		if !strings.Contains(section, token) {
			t.Errorf("EmbeddedIssueGraphSection() missing %q", token)
		}
	}
}

func TestFactoryMarkers(t *testing.T) {
	if !strings.Contains(agentsBeginMarker, "BEGIN") {
		t.Error("begin marker should mention BEGIN")
	}
	if !strings.Contains(agentsEndMarker, "END") {
		t.Error("end marker should mention END")
	}
}

func TestMarkersMatch(t *testing.T) {
	issuegraphSection := agents.EmbeddedIssueGraphSection()
	if !strings.HasPrefix(issuegraphSection, agentsBeginMarker) {
		t.Error("section should start with begin marker")
	}
	trimmed := strings.TrimSpace(issuegraphSection)
	if !strings.HasSuffix(trimmed, agentsEndMarker) {
		t.Error("section should end with end marker")
	}
}

func TestUpdateIssueGraphSectionPreservesWhitespace(t *testing.T) {
	issuegraphSection := agents.EmbeddedIssueGraphSection()
	content := "# Header\n\n" + issuegraphSection + "\n\n# Footer"
	updated := updateIssueGraphSection(content)
	if !strings.Contains(updated, "# Header") || !strings.Contains(updated, "# Footer") {
		t.Error("update should preserve surrounding content")
	}
}

func TestRoundTripAddRemoveLeavesNoIssueGraphContent(t *testing.T) {
	userContent := "# My Project\n\nSome user content here.\n"

	// Simulate issuegraph init adding the issuegraph section to an existing file
	withIssueGraph := userContent + "\n" + agents.EmbeddedIssueGraphSection()

	// Verify issuegraph content (including landing) is present
	if !strings.Contains(withIssueGraph, "BEGIN ISSUEGRAPH INTEGRATION") {
		t.Fatal("issuegraph markers should be present after adding section")
	}
	if !strings.Contains(withIssueGraph, "Session Completion") {
		t.Fatal("session-completion should be present inside issuegraph section")
	}

	// Remove the issuegraph section
	cleaned := removeIssueGraphSection(withIssueGraph)

	// Verify no issuegraph-added content remains
	if strings.Contains(cleaned, "BEGIN ISSUEGRAPH INTEGRATION") {
		t.Error("begin marker should not remain after removal")
	}
	if strings.Contains(cleaned, "END ISSUEGRAPH INTEGRATION") {
		t.Error("end marker should not remain after removal")
	}
	if strings.Contains(cleaned, "Session Completion") {
		t.Error("session-completion should not remain after removal (must be inside markers)")
	}
	if strings.Contains(cleaned, "issuegraph ready") {
		t.Error("issuegraph commands should not remain after removal")
	}

	// User content should be preserved
	if !strings.Contains(cleaned, "# My Project") {
		t.Error("user content should be preserved after removal")
	}
}
