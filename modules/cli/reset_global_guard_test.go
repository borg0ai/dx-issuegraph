package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// FindIssueGraphDir walks up from the working directory with no upper boundary, so a
// repository under $HOME that has no .issuegraph of its own resolves to ~/.issuegraph.
// Reset then removes it. These pin that it refuses instead — and that the
// refusal is narrow enough not to break the case where the home directory
// genuinely is the repository.
func TestRefuseGlobalIssueGraphDir(t *testing.T) {
	home := filepath.Join("/", "home", "someone")
	globalIssueGraph := filepath.Join(home, ".issuegraph")
	project := filepath.Join(home, "dev", "project")

	tests := []struct {
		name          string
		issuegraphDir string
		repoRoot      string
		home          string
		wantError     bool
	}{
		{
			// The incident shape: a reset inside a project whose walk climbed
			// out of the repository and landed on the user's global directory.
			name:          "project reset that resolved to the global issuegraph dir",
			issuegraphDir: globalIssueGraph,
			repoRoot:      project,
			home:          home,
			wantError:     true,
		},
		{
			// Not an over-reach: here ~/.issuegraph IS the repository's issuegraph dir,
			// and removing it is exactly what was asked for.
			name:          "home directory is itself the repository",
			issuegraphDir: globalIssueGraph,
			repoRoot:      home,
			home:          home,
			wantError:     false,
		},
		{
			name:          "ordinary repo-local issuegraph dir",
			issuegraphDir: filepath.Join(project, ".issuegraph"),
			repoRoot:      project,
			home:          home,
			wantError:     false,
		},
		{
			// A directory that merely lives under $HOME is not the global one.
			name:          "issuegraph dir under home but not the global path",
			issuegraphDir: filepath.Join(home, ".issuegraph-backup"),
			repoRoot:      project,
			home:          home,
			wantError:     false,
		},
		{
			name:          "no home directory: nothing global to protect",
			issuegraphDir: globalIssueGraph,
			repoRoot:      project,
			home:          "",
			wantError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := refuseGlobalIssueGraphDir(tt.issuegraphDir, tt.repoRoot, tt.home)
			if tt.wantError && err == nil {
				t.Fatalf("refuseGlobalIssueGraphDir() = nil, want a refusal")
			}
			if !tt.wantError && err != nil {
				t.Fatalf("refuseGlobalIssueGraphDir() = %v, want nil", err)
			}
			if err == nil {
				return
			}
			// The message has to name both paths: the whole failure mode is a
			// user who cannot tell which directory the command was about.
			for _, want := range []string{tt.repoRoot, tt.issuegraphDir} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("refusal message does not name %q:\n%s", want, err)
				}
			}
		})
	}
}

// Trailing separators and unclean paths are a normal way for these to arrive,
// and a guard that a separator defeats is not a guard.
func TestRefuseGlobalIssueGraphDir_NormalizesPaths(t *testing.T) {
	home := filepath.Join("/", "home", "someone")
	issuegraphDir := filepath.Join(home, "dev", "..", ".issuegraph") + string(filepath.Separator)

	if err := refuseGlobalIssueGraphDir(issuegraphDir, filepath.Join(home, "dev", "project"), home); err == nil {
		t.Fatalf("refuseGlobalIssueGraphDir(%q) = nil, want a refusal", issuegraphDir)
	}
}
