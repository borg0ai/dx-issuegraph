package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/borg0ai/dx-issuegraph/internal/configfile"
	"github.com/borg0ai/dx-issuegraph/internal/molecules"
)

// TestRegisteredBackendResolvesHookAndMoleculeDirsUnderIssueGraph guards the root
// command's directory derivation for a registered WorkspaceIsIssueGraphDir backend.
//
// For such a backend the command sets dbPath to the .issuegraph directory itself
// (main.go: `dbPath = issuegraph` when registeredBackendWorkspaceIsIssueGraphDir(cfg) is
// true), not to a .issuegraph/<db-file>. The hook runner and molecule loader must
// therefore resolve their directory from the issuegraphDir the command already
// computed with resolveCommandIssueGraphDir(dbPath), never from filepath.Dir(dbPath):
// for this case filepath.Dir(dbPath) is the repository root, so hooks and
// molecule templates would load from <repo>/hooks and <repo>/molecules.jsonl
// instead of .issuegraph/hooks and .issuegraph/molecules.jsonl.
//
// Driving the full PersistentPreRunE end-to-end would require a live
// storage.DoltStorage from the fake backend, which is impractical — the
// interface is large and has no test double (see
// internal/storage/hook_decorator_test.go). This test instead reproduces the
// exact directory-derivation chain the command uses and asserts the .issuegraph
// hooks directory and molecule catalog are selected over repo-root decoys.
func TestRegisteredBackendResolvesHookAndMoleculeDirsUnderIssueGraph(t *testing.T) {
	const name = "contract-workspace-paths"
	registerContractBackend(t, name)

	repo := t.TempDir()
	issuegraphDir := filepath.Join(repo, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0o755); err != nil {
		t.Fatalf("mkdir .issuegraph: %v", err)
	}
	if err := (&configfile.Config{Backend: name}).Save(issuegraphDir); err != nil {
		t.Fatalf("save metadata.json: %v", err)
	}

	// The workspace's real hooks directory and molecule catalog live under .issuegraph.
	wantHooksDir := filepath.Join(issuegraphDir, "hooks")
	if err := os.MkdirAll(wantHooksDir, 0o755); err != nil {
		t.Fatalf("mkdir .issuegraph/hooks: %v", err)
	}
	wantMoleculeFile := filepath.Join(issuegraphDir, molecules.MoleculeFileName)
	if err := os.WriteFile(wantMoleculeFile, []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write .issuegraph/molecules.jsonl: %v", err)
	}

	// Decoys at the repository root: the pre-fix filepath.Dir(dbPath) code would
	// have selected these instead of the .issuegraph copies.
	decoyHooksDir := filepath.Join(repo, "hooks")
	if err := os.MkdirAll(decoyHooksDir, 0o755); err != nil {
		t.Fatalf("mkdir decoy hooks: %v", err)
	}
	decoyMoleculeFile := filepath.Join(repo, molecules.MoleculeFileName)
	if err := os.WriteFile(decoyMoleculeFile, []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write decoy molecules.jsonl: %v", err)
	}

	// This workspace is exactly the case that makes the command use the .issuegraph
	// directory as dbPath.
	cfg, err := configfile.Load(issuegraphDir)
	if err != nil {
		t.Fatalf("load metadata.json: %v", err)
	}
	if !registeredBackendWorkspaceIsIssueGraphDir(cfg) {
		t.Fatal("registered WorkspaceIsIssueGraphDir backend not recognized; dbPath would not be the .issuegraph directory")
	}

	// Mirror the command: dbPath is the .issuegraph directory and the hook/molecule
	// setup resolves the issuegraph dir with resolveCommandIssueGraphDir(dbPath).
	dbPath := issuegraphDir
	gotIssueGraphDir := resolveCommandIssueGraphDir(dbPath)

	requireSameFile(t, "resolved issuegraph dir", gotIssueGraphDir, issuegraphDir)
	requireSameFile(t, "hook runner dir", filepath.Join(gotIssueGraphDir, "hooks"), wantHooksDir)
	requireSameFile(t, "molecule catalog", filepath.Join(gotIssueGraphDir, molecules.MoleculeFileName), wantMoleculeFile)

	// Regression guard: the pre-fix filepath.Dir(dbPath) resolves to the repo
	// root, so hooks and molecules would have loaded from the decoys, not .issuegraph.
	preFix := filepath.Dir(dbPath)
	if sameExistingFile(preFix, issuegraphDir) {
		t.Fatalf("filepath.Dir(dbPath)=%q unexpectedly equals .issuegraph %q; regression guard is vacuous", preFix, issuegraphDir)
	}
	requireSameFile(t, "pre-fix hook dir resolves to repo-root decoy", filepath.Join(preFix, "hooks"), decoyHooksDir)
}

func requireSameFile(t *testing.T, label, got, want string) {
	t.Helper()
	gotInfo, err := os.Stat(got)
	if err != nil {
		t.Fatalf("%s: stat %q: %v", label, got, err)
	}
	wantInfo, err := os.Stat(want)
	if err != nil {
		t.Fatalf("%s: stat %q: %v", label, want, err)
	}
	if !os.SameFile(gotInfo, wantInfo) {
		t.Fatalf("%s: got %q, want %q (different paths)", label, got, want)
	}
}

func sameExistingFile(a, b string) bool {
	aInfo, err := os.Stat(a)
	if err != nil {
		return false
	}
	bInfo, err := os.Stat(b)
	if err != nil {
		return false
	}
	return os.SameFile(aInfo, bInfo)
}
