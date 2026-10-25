package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/borg0ai/dx-issuegraph/internal/config"
	"github.com/borg0ai/dx-issuegraph/internal/configfile"
)

func TestInitCommandRegistersRemoteFlag(t *testing.T) {
	if initCmd.Flags().Lookup("remote") == nil {
		t.Fatal("init command does not register --remote")
	}
}

func TestInitExplicitRemoteDrivesCloneAndPersistence(t *testing.T) {
	const remote = "git+ssh://git@example.com/right/repo.git"
	resolveConfiguredRemote := func() string {
		return "git+ssh://git@example.com/wrong/repo.git"
	}

	syncURL, source := resolveInitConfiguredSyncRemote(remote, true, resolveConfiguredRemote)
	if syncURL != remote {
		t.Fatalf("syncURL = %q, want explicit remote %q", syncURL, remote)
	}
	if source != initSyncRemoteExplicit {
		t.Fatalf("source = %v, want explicit", source)
	}
	syncFromRemote := syncURL != ""
	syncURLFromConfig := syncURL != "" && source != initSyncRemoteNone
	if !syncFromRemote {
		t.Fatal("explicit remote did not select clone-from-remote path")
	}
	if !syncURLFromConfig {
		t.Fatal("explicit remote was not treated as user-configured sync URL")
	}

	issuegraphDir := filepath.Join(t.TempDir(), ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(issuegraphDir, "config.yaml"), []byte("# IssueGraph Config\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := persistInitSyncRemote(issuegraphDir, remote, syncURL, syncFromRemote, syncURLFromConfig, false); err != nil {
		t.Fatalf("persistInitSyncRemote failed: %v", err)
	}
	configBytes, err := os.ReadFile(filepath.Join(issuegraphDir, "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(configBytes), remote) {
		t.Fatalf("config.yaml does not contain explicit remote %q:\n%s", remote, configBytes)
	}
}

func TestInitServerExternalRemoteUsesServerCloneMode(t *testing.T) {
	if got := initRemoteCloneMode(true, true); got != remoteCloneExternalServer {
		t.Fatalf("initRemoteCloneMode(server=true, external=true) = %v, want external server clone mode", got)
	}
}

func TestInitDoltServerTLSFromEnv(t *testing.T) {
	tests := []struct {
		name string
		env  string
		want bool
	}{
		{name: "unset", want: false},
		{name: "zero", env: "0", want: false},
		{name: "false", env: "false", want: false},
		{name: "one", env: "1", want: true},
		{name: "true", env: "true", want: true},
		{name: "mixed case true", env: "TrUe", want: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.env == "" {
				t.Setenv("ISSUEGRAPH_DOLT_SERVER_TLS", "")
			} else {
				t.Setenv("ISSUEGRAPH_DOLT_SERVER_TLS", tc.env)
			}
			if got := initDoltServerTLSFromEnv(); got != tc.want {
				t.Fatalf("initDoltServerTLSFromEnv() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestInitExplicitEmptyRemoteSkipsAmbientConfig(t *testing.T) {
	called := false
	resolveConfiguredRemote := func() string {
		called = true
		return "git+ssh://git@example.com/ambient/repo.git"
	}

	syncURL, source := resolveInitConfiguredSyncRemote("", true, resolveConfiguredRemote)
	if called {
		t.Fatal("explicit empty --remote consulted ambient sync config")
	}
	if syncURL != "" {
		t.Fatalf("syncURL = %q, want empty explicit remote", syncURL)
	}
	if source != initSyncRemoteExplicit {
		t.Fatalf("source = %v, want explicit", source)
	}
	if syncURL != "" {
		t.Fatal("explicit empty --remote would trigger early remote safety")
	}
}

func TestPersistInitSyncRemoteExplicitRemoteWritesTargetDir(t *testing.T) {
	tmpDir := t.TempDir()
	callerDir := filepath.Join(tmpDir, "caller")
	targetIssueGraphDir := filepath.Join(tmpDir, "target", ".issuegraph")
	callerIssueGraphDir := filepath.Join(callerDir, ".issuegraph")
	if err := os.MkdirAll(callerIssueGraphDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(targetIssueGraphDir, 0o750); err != nil {
		t.Fatal(err)
	}

	callerConfig := filepath.Join(callerIssueGraphDir, "config.yaml")
	const callerRemote = "git+ssh://git@example.com/wrong/repo.git"
	seedSyncRemote(t, callerIssueGraphDir, callerRemote)
	targetConfig := filepath.Join(targetIssueGraphDir, "config.yaml")
	if err := os.WriteFile(targetConfig, []byte("# IssueGraph Config\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWd) }()
	if err := os.Chdir(callerDir); err != nil {
		t.Fatal(err)
	}

	const remote = "git+ssh://git@example.com/right/repo.git"
	if err := persistInitSyncRemote(targetIssueGraphDir, remote, remote, false, true, false); err != nil {
		t.Fatalf("persistInitSyncRemote failed: %v", err)
	}

	targetBytes, err := os.ReadFile(targetConfig)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(targetBytes), remote) {
		t.Fatalf("target config.yaml does not contain explicit remote %q:\n%s", remote, targetBytes)
	}

	callerBytes, err := os.ReadFile(callerConfig)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(callerBytes), remote) {
		t.Fatalf("caller config.yaml was modified instead of target:\n%s", callerBytes)
	}
	// Stronger than the absence above: the caller still resolves to what it had.
	// A write that landed somewhere unreadable would satisfy "does not contain
	// the new remote" while having quietly destroyed the old one.
	if got := config.GetStringFromDir(callerIssueGraphDir, "sync.remote"); got != callerRemote {
		t.Fatalf("caller sync.remote = %q, want it untouched at %q:\n%s", got, callerRemote, callerBytes)
	}
}

// `issuegraph init --remote` runs createConfigYaml and then immediately writes
// sync.remote into what it just created. That template is comments and nothing
// else, which is the one document shape yaml.v3 parses to no nodes at all — so a
// writer that round-trips a node tree has nothing to write back but the key it
// added, and the operator's first commit after init carries a config.yaml with
// its entire documentation deleted. It is git-tracked, so they see it.
//
// This runs the real pair in the real order rather than asserting on a replica
// of the template, so it stays true if either side moves.
func TestInitPersistingSyncRemoteKeepsTheConfigTemplate(t *testing.T) {
	issuegraphDir := filepath.Join(t.TempDir(), ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := createConfigYaml(issuegraphDir, false, ""); err != nil {
		t.Fatalf("createConfigYaml: %v", err)
	}
	template, err := os.ReadFile(filepath.Join(issuegraphDir, "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	const remote = "git+ssh://git@example.com/right/repo.git"
	if err := persistInitSyncRemote(issuegraphDir, remote, remote, false, true, false); err != nil {
		t.Fatalf("persistInitSyncRemote: %v", err)
	}

	body, err := os.ReadFile(filepath.Join(issuegraphDir, "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(template), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if !strings.Contains(string(body), line) {
			t.Errorf("init dropped a line of its own config template: %q\ngot:\n%s", line, body)
		}
	}
	if got := config.GetStringFromDir(issuegraphDir, "sync.remote"); got != remote {
		t.Errorf("sync.remote = %q, want %q\n%s", got, remote, body)
	}
}

func TestInitTimeCloneConfigExternalDefaultsAreSelfContained(t *testing.T) {
	t.Setenv("ISSUEGRAPH_DOLT_SERVER_HOST", "")
	t.Setenv("ISSUEGRAPH_DOLT_SERVER_PORT", "")
	t.Setenv("ISSUEGRAPH_DOLT_PORT", "")
	t.Setenv("ISSUEGRAPH_DOLT_SERVER_USER", "")
	t.Setenv("ISSUEGRAPH_DOLT_SERVER_DATABASE", "")

	cfg := initTimeCloneConfig(true, "", 3312, "", "", "issuegraph_proj")

	if cfg.GetDoltMode() != configfile.DoltModeServer {
		t.Fatalf("dolt mode = %q, want server", cfg.GetDoltMode())
	}
	if cfg.GetDoltServerHost() != configfile.DefaultDoltServerHost {
		t.Fatalf("host = %q, want default %q", cfg.GetDoltServerHost(), configfile.DefaultDoltServerHost)
	}
	if cfg.GetDoltServerUser() != configfile.DefaultDoltServerUser {
		t.Fatalf("user = %q, want default %q", cfg.GetDoltServerUser(), configfile.DefaultDoltServerUser)
	}
	if cfg.GetDoltServerPort() != 3312 {
		t.Fatalf("port = %d, want 3312", cfg.GetDoltServerPort())
	}
	if cfg.GetDoltDatabase() != "issuegraph_proj" {
		t.Fatalf("database = %q, want issuegraph_proj", cfg.GetDoltDatabase())
	}
}
