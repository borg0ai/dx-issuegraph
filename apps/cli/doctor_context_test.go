package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/borg0ai/dx-issuegraph/internal/config"
	"github.com/borg0ai/dx-issuegraph/internal/configfile"
	"github.com/borg0ai/dx-issuegraph/internal/issuegraph"
	"github.com/borg0ai/dx-issuegraph/internal/utils"
)

func savePersistentPreRunState(t *testing.T) {
	t.Helper()

	// Track env vars that PersistentPreRun may modify via prepareSelectedCommandContext
	// or applyChangeDirSelection. Without this, a call to PersistentPreRun leaves
	// ISSUEGRAPH_DIR set to the project's .issuegraph dir, corrupting subsequent tests.
	for _, key := range []string{"ISSUEGRAPH_DIR", "ISSUEGRAPH_DB", "ISSUEGRAPH_DB"} {
		t.Setenv(key, os.Getenv(key))
	}

	oldServerMode := serverMode
	oldCmdCtx := cmdCtx
	oldDBPath := dbPath
	oldActor := actor
	oldJSONOutput := jsonOutput
	oldReadonlyMode := readonlyMode
	oldDoltAutoCommit := doltAutoCommit
	flagState := snapshotRootFlagState()
	t.Cleanup(func() {
		serverMode = oldServerMode
		cmdCtx = oldCmdCtx
		dbPath = oldDBPath
		actor = oldActor
		jsonOutput = oldJSONOutput
		readonlyMode = oldReadonlyMode
		doltAutoCommit = oldDoltAutoCommit
		restoreRootFlagState(t, flagState)
	})

	serverMode = false
	cmdCtx = nil
	dbPath = ""
	actor = ""
	jsonOutput = false
	readonlyMode = false
	doltAutoCommit = ""
}

func writeMetadataConfig(t *testing.T, issuegraphDir string, doltMode string, database string) {
	t.Helper()

	if err := (&configfile.Config{
		Backend:      configfile.BackendDolt,
		DoltMode:     doltMode,
		DoltDatabase: database,
	}).Save(issuegraphDir); err != nil {
		t.Fatalf("save metadata: %v", err)
	}
}

func TestDoctorPersistentPreRunLoadsServerModeForNoDBCommand(t *testing.T) {
	repoDir := t.TempDir()
	issuegraphDir := filepath.Join(repoDir, ".issuegraph")
	writeTestConfigYAML(t, issuegraphDir, "")
	writeMetadataConfig(t, issuegraphDir, configfile.DoltModeServer, "doctor_ctx_test")

	t.Chdir(repoDir)
	t.Setenv("ISSUEGRAPH_DIR", issuegraphDir)
	t.Setenv("ISSUEGRAPH_DOLT_SHARED_SERVER", "")
	t.Setenv("ISSUEGRAPH_DOLT_SERVER_DATABASE", "")
	t.Setenv("ISSUEGRAPH_DOLT_SERVER_PORT", "")

	config.ResetForTesting()
	t.Cleanup(config.ResetForTesting)
	savePersistentPreRunState(t)

	if rootCmd.PersistentPreRunE == nil {
		t.Fatal("rootCmd.PersistentPreRunE must be set")
	}
	if err := rootCmd.PersistentPreRunE(doctorCmd, nil); err != nil {
		t.Fatalf("PersistentPreRunE: %v", err)
	}

	if !serverMode {
		t.Fatal("doctor should load server mode before the no-store early return")
	}
}

func TestDoctorPersistentPreRunUsesExplicitDBTarget(t *testing.T) {
	callerRepo := filepath.Join(t.TempDir(), "caller")
	callerIssueGraphDir := filepath.Join(callerRepo, ".issuegraph")
	writeTestConfigYAML(t, callerIssueGraphDir, "")
	writeMetadataConfig(t, callerIssueGraphDir, configfile.DoltModeEmbedded, "caller_ctx_test")

	targetRepo := filepath.Join(t.TempDir(), "target")
	targetIssueGraphDir := filepath.Join(targetRepo, ".issuegraph")
	writeTestConfigYAML(t, targetIssueGraphDir, "")
	writeMetadataConfig(t, targetIssueGraphDir, configfile.DoltModeServer, "target_ctx_test")

	t.Chdir(callerRepo)
	t.Setenv("ISSUEGRAPH_DIR", callerIssueGraphDir)
	t.Setenv("ISSUEGRAPH_DOLT_SHARED_SERVER", "")
	t.Setenv("ISSUEGRAPH_DOLT_SERVER_DATABASE", "")
	t.Setenv("ISSUEGRAPH_DOLT_SERVER_PORT", "")

	config.ResetForTesting()
	t.Cleanup(config.ResetForTesting)
	savePersistentPreRunState(t)

	targetDBPath := filepath.Join(targetIssueGraphDir, "dolt")
	dbPath = targetDBPath
	if flag := rootCmd.PersistentFlags().Lookup("db"); flag != nil {
		flag.Changed = true
	}

	if err := rootCmd.PersistentPreRunE(doctorCmd, nil); err != nil {
		t.Fatalf("PersistentPreRunE: %v", err)
	}

	if got := os.Getenv("ISSUEGRAPH_DIR"); got != targetIssueGraphDir {
		t.Fatalf("ISSUEGRAPH_DIR = %q, want %q", got, targetIssueGraphDir)
	}
	if !serverMode {
		t.Fatal("doctor should use the explicit target repo's server mode")
	}
}

func TestBootstrapPersistentPreRunUsesExplicitDBTarget(t *testing.T) {
	callerRepo := filepath.Join(t.TempDir(), "caller")
	callerIssueGraphDir := filepath.Join(callerRepo, ".issuegraph")
	writeTestConfigYAML(t, callerIssueGraphDir, "")
	writeMetadataConfig(t, callerIssueGraphDir, configfile.DoltModeEmbedded, "caller_bootstrap_test")

	targetRepo := filepath.Join(t.TempDir(), "target")
	targetIssueGraphDir := filepath.Join(targetRepo, ".issuegraph")
	writeTestConfigYAML(t, targetIssueGraphDir, "")
	writeMetadataConfig(t, targetIssueGraphDir, configfile.DoltModeServer, "target_bootstrap_test")

	t.Chdir(callerRepo)
	t.Setenv("ISSUEGRAPH_DIR", callerIssueGraphDir)
	t.Setenv("ISSUEGRAPH_DOLT_SHARED_SERVER", "")
	t.Setenv("ISSUEGRAPH_DOLT_SERVER_DATABASE", "")
	t.Setenv("ISSUEGRAPH_DOLT_SERVER_PORT", "")

	config.ResetForTesting()
	t.Cleanup(config.ResetForTesting)
	savePersistentPreRunState(t)

	targetDBPath := filepath.Join(targetIssueGraphDir, "dolt")
	dbPath = targetDBPath
	if flag := rootCmd.PersistentFlags().Lookup("db"); flag != nil {
		flag.Changed = true
	}

	if err := rootCmd.PersistentPreRunE(bootstrapCmd, nil); err != nil {
		t.Fatalf("PersistentPreRunE: %v", err)
	}

	if got := os.Getenv("ISSUEGRAPH_DIR"); got != targetIssueGraphDir {
		t.Fatalf("ISSUEGRAPH_DIR = %q, want %q", got, targetIssueGraphDir)
	}
}

func TestLoadSelectionEnvironmentUsesAmbientEnvFileForISSUEGRAPHDB(t *testing.T) {
	callerRepo := filepath.Join(t.TempDir(), "caller")
	callerIssueGraphDir := filepath.Join(callerRepo, ".issuegraph")
	writeTestConfigYAML(t, callerIssueGraphDir, "")

	targetRepo := filepath.Join(t.TempDir(), "target")
	targetIssueGraphDir := filepath.Join(targetRepo, ".issuegraph")
	writeTestConfigYAML(t, targetIssueGraphDir, "")
	targetDBPath := filepath.Join(targetIssueGraphDir, "dolt")
	if err := os.MkdirAll(targetDBPath, 0o700); err != nil {
		t.Fatalf("mkdir target db dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(callerIssueGraphDir, ".env"), []byte("ISSUEGRAPH_DB="+targetDBPath+"\n"), 0o600); err != nil {
		t.Fatalf("write caller .env: %v", err)
	}

	t.Chdir(callerRepo)
	t.Setenv("ISSUEGRAPH_DIR", "")
	t.Setenv("ISSUEGRAPH_DB", "")
	t.Setenv("ISSUEGRAPH_DB", "")

	loadSelectionEnvironment()

	if got := os.Getenv("ISSUEGRAPH_DB"); utils.CanonicalizePath(got) != utils.CanonicalizePath(targetDBPath) {
		t.Fatalf("ISSUEGRAPH_DB = %q, want %q", got, targetDBPath)
	}
	if got := issuegraph.FindDatabasePath(); utils.CanonicalizePath(got) != utils.CanonicalizePath(targetDBPath) {
		t.Fatalf("FindDatabasePath() = %q, want %q", got, targetDBPath)
	}
}
func TestSelectedDoltIssueGraphDirUsesReboundISSUEGRAPHDir(t *testing.T) {
	callerRepo := filepath.Join(t.TempDir(), "caller")
	callerIssueGraphDir := filepath.Join(callerRepo, ".issuegraph")
	writeTestConfigYAML(t, callerIssueGraphDir, "")

	targetRepo := filepath.Join(t.TempDir(), "target")
	targetIssueGraphDir := filepath.Join(targetRepo, ".issuegraph")
	writeTestConfigYAML(t, targetIssueGraphDir, "")
	targetDBPath := filepath.Join(targetIssueGraphDir, "dolt")
	if err := os.MkdirAll(targetDBPath, 0o700); err != nil {
		t.Fatalf("mkdir target db dir: %v", err)
	}

	t.Chdir(callerRepo)
	t.Setenv("ISSUEGRAPH_DIR", targetIssueGraphDir)
	t.Setenv("ISSUEGRAPH_DB", filepath.Join(callerIssueGraphDir, "dolt"))

	if got := selectedDoltIssueGraphDir(); utils.CanonicalizePath(got) != utils.CanonicalizePath(targetIssueGraphDir) {
		t.Fatalf("selectedDoltIssueGraphDir() = %q, want %q", got, targetIssueGraphDir)
	}
}
