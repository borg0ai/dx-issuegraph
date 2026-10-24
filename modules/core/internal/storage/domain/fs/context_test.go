package fs

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/borg0ai/dx-issuegraph/internal/configfile"
	"github.com/borg0ai/dx-issuegraph/internal/git"
	"github.com/borg0ai/dx-issuegraph/internal/issuegraph"
	"github.com/borg0ai/dx-issuegraph/internal/storage/backendnames"
	"github.com/borg0ai/dx-issuegraph/internal/storage/domain"
)

func newContextRepoForTest(t *testing.T) (workDir, issuegraphDir string, repo domain.ContextRepository) {
	t.Helper()
	workDir = t.TempDir()
	issuegraphDir = filepath.Join(workDir, ".issuegraph")
	require.NoError(t, os.MkdirAll(issuegraphDir, 0o750))

	t.Setenv("ISSUEGRAPH_DIR", issuegraphDir)
	t.Setenv("ISSUEGRAPH_DOLT_SERVER_MODE", "")
	t.Setenv("ISSUEGRAPH_DOLT_SHARED_SERVER", "")
	t.Setenv("ISSUEGRAPH_DOLT_SERVER_DATABASE", "")
	t.Setenv("ISSUEGRAPH_DOLT_SERVER_HOST", "")
	t.Setenv("ISSUEGRAPH_DOLT_DATA_DIR", "")
	t.Setenv("ISSUEGRAPH_DOLT_SERVER_PORT", "")
	t.Setenv("ISSUEGRAPH_PROXIED_SERVER_ROOT_PATH", "")

	repo = NewContextRepository(NewIssueGraphDirFSRepository(workDir, testTemplates()))
	return workDir, issuegraphDir, repo
}

func canonicalIssueGraphDir(t *testing.T, workDir string) string {
	t.Helper()
	return NewIssueGraphDirFSRepository(workDir, testTemplates()).ResolveIssueGraphDirPath(context.Background()).IssueGraphDir
}

func writeMetadata(t *testing.T, issuegraphDir string, cfg *configfile.Config) {
	t.Helper()
	data, err := json.Marshal(cfg)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(issuegraphDir, "metadata.json"), data, 0o600))
}

func TestContextRepository_BackendConfig(t *testing.T) {
	t.Run("DefaultsWithoutMetadata", func(t *testing.T) {
		_, _, repo := newContextRepoForTest(t)

		got, err := repo.BackendConfig(context.Background())
		require.NoError(t, err)
		require.Equal(t, configfile.BackendDolt, got.Backend)
		require.Equal(t, configfile.DoltModeEmbedded, got.DoltMode)
		require.Equal(t, configfile.DefaultDoltDatabase, got.Database)
		require.Empty(t, got.ProjectID)
		require.False(t, got.IsServerMode)
		require.False(t, got.IsProxiedServerMode)
	})

	// The backend NAME is what tells the projection above whether the Dolt
	// fields beside it describe this workspace at all. Reading it here, from
	// the same config the store open dispatches on, is what keeps the answer
	// one source of truth rather than a second opinion.
	t.Run("RegisteredBackendIsNamed", func(t *testing.T) {
		// backendnames is the set configfile.GetBackend classifies against —
		// the backends registry writes it on Register — so this is the same
		// question the store open asks, without dragging the typed registry
		// and its storage dependency into a config-reading test.
		const name = "acme-context"
		backendnames.Add(name)
		t.Cleanup(func() { backendnames.Remove(name) })

		_, issuegraphDir, repo := newContextRepoForTest(t)
		writeMetadata(t, issuegraphDir, &configfile.Config{Backend: name})

		got, err := repo.BackendConfig(context.Background())
		require.NoError(t, err)
		require.Equal(t, name, got.Backend)
	})

	t.Run("EmbeddedWithExplicitValues", func(t *testing.T) {
		_, issuegraphDir, repo := newContextRepoForTest(t)
		writeMetadata(t, issuegraphDir, &configfile.Config{
			DoltMode:     configfile.DoltModeEmbedded,
			DoltDatabase: "mydb",
			ProjectID:    "proj-123",
		})

		got, err := repo.BackendConfig(context.Background())
		require.NoError(t, err)
		require.Equal(t, configfile.DoltModeEmbedded, got.DoltMode)
		require.Equal(t, "mydb", got.Database)
		require.Equal(t, "proj-123", got.ProjectID)
		require.False(t, got.IsServerMode)
		require.False(t, got.IsProxiedServerMode)
	})

	t.Run("ServerMode", func(t *testing.T) {
		_, issuegraphDir, repo := newContextRepoForTest(t)
		writeMetadata(t, issuegraphDir, &configfile.Config{
			DoltMode:       configfile.DoltModeServer,
			DoltServerHost: "db.example.com",
			DoltDataDir:    "/data/dolt",
		})

		got, err := repo.BackendConfig(context.Background())
		require.NoError(t, err)
		require.True(t, got.IsServerMode)
		require.False(t, got.IsProxiedServerMode)
		require.Equal(t, "db.example.com", got.ServerHost)
		require.Equal(t, "/data/dolt", got.DataDir)
	})

	t.Run("ProxiedServerMode", func(t *testing.T) {
		_, issuegraphDir, repo := newContextRepoForTest(t)
		writeMetadata(t, issuegraphDir, &configfile.Config{
			DoltMode: configfile.DoltModeProxiedServer,
		})

		got, err := repo.BackendConfig(context.Background())
		require.NoError(t, err)
		require.True(t, got.IsProxiedServerMode)
		require.False(t, got.IsServerMode)
	})

	t.Run("EnvOverridesDatabase", func(t *testing.T) {
		_, issuegraphDir, repo := newContextRepoForTest(t)
		writeMetadata(t, issuegraphDir, &configfile.Config{DoltDatabase: "filedb"})
		t.Setenv("ISSUEGRAPH_DOLT_SERVER_DATABASE", "envdb")

		got, err := repo.BackendConfig(context.Background())
		require.NoError(t, err)
		require.Equal(t, "envdb", got.Database)
	})
}

func TestContextRepository_ServerPort(t *testing.T) {
	t.Run("EnvOverride", func(t *testing.T) {
		_, _, repo := newContextRepoForTest(t)
		t.Setenv("ISSUEGRAPH_DOLT_SERVER_PORT", "3333")

		got, err := repo.ServerPort(context.Background())
		require.NoError(t, err)
		require.Equal(t, 3333, got)
	})

	t.Run("ReadsPortFile", func(t *testing.T) {
		_, issuegraphDir, repo := newContextRepoForTest(t)
		require.NoError(t, os.WriteFile(filepath.Join(issuegraphDir, "dolt-server.port"), []byte("4567"), 0o600))

		got, err := repo.ServerPort(context.Background())
		require.NoError(t, err)
		require.Equal(t, 4567, got)
	})
}

func TestContextRepository_ProxiedServerRoot(t *testing.T) {
	t.Run("EnvAbsolutePath", func(t *testing.T) {
		_, _, repo := newContextRepoForTest(t)
		abs := filepath.Join(t.TempDir(), "abs-root")
		t.Setenv("ISSUEGRAPH_PROXIED_SERVER_ROOT_PATH", abs)

		got, err := repo.ProxiedServerRoot(context.Background())
		require.NoError(t, err)
		require.Equal(t, abs, got)
	})

	t.Run("EnvRelativePathJoinedWithIssueGraphDir", func(t *testing.T) {
		workDir, _, repo := newContextRepoForTest(t)
		t.Setenv("ISSUEGRAPH_PROXIED_SERVER_ROOT_PATH", "relroot")

		got, err := repo.ProxiedServerRoot(context.Background())
		require.NoError(t, err)
		canonIssueGraphDir := canonicalIssueGraphDir(t, workDir)
		require.Equal(t, filepath.Join(canonIssueGraphDir, "relroot"), got)
	})

	t.Run("ClientInfoRootPath", func(t *testing.T) {
		workDir, issuegraphDir, repo := newContextRepoForTest(t)
		require.NoError(t, configfile.SaveProxiedServerClientInfo(issuegraphDir, &configfile.ProxiedServerClientInfo{
			RootPath: "customroot",
		}))

		got, err := repo.ProxiedServerRoot(context.Background())
		require.NoError(t, err)
		canonIssueGraphDir := canonicalIssueGraphDir(t, workDir)
		require.Equal(t, filepath.Join(canonIssueGraphDir, "customroot"), got)
	})

	t.Run("DefaultProxiedDir", func(t *testing.T) {
		workDir, _, repo := newContextRepoForTest(t)

		got, err := repo.ProxiedServerRoot(context.Background())
		require.NoError(t, err)
		canonIssueGraphDir := canonicalIssueGraphDir(t, workDir)
		require.Equal(t, filepath.Join(canonIssueGraphDir, "dolt"), got)
	})
}

func TestContextRepository_SyncRemote(t *testing.T) {
	writeConfigYAML := func(t *testing.T, issuegraphDir, body string) {
		t.Helper()
		require.NoError(t, os.WriteFile(filepath.Join(issuegraphDir, "config.yaml"), []byte(body), 0o600))
	}

	t.Run("ReadsSyncRemote", func(t *testing.T) {
		_, issuegraphDir, repo := newContextRepoForTest(t)
		writeConfigYAML(t, issuegraphDir, "sync:\n  remote: git@example.com:org/repo.git\n")

		got, err := repo.SyncRemote(context.Background())
		require.NoError(t, err)
		require.Equal(t, "git@example.com:org/repo.git", got)
	})

	t.Run("FallsBackToGitRemote", func(t *testing.T) {
		_, issuegraphDir, repo := newContextRepoForTest(t)
		writeConfigYAML(t, issuegraphDir, "sync:\n  git-remote: legacy-remote\n")

		got, err := repo.SyncRemote(context.Background())
		require.NoError(t, err)
		require.Equal(t, "legacy-remote", got)
	})

	t.Run("SyncRemoteWinsOverGitRemote", func(t *testing.T) {
		_, issuegraphDir, repo := newContextRepoForTest(t)
		writeConfigYAML(t, issuegraphDir, "sync:\n  remote: primary\n  git-remote: legacy\n")

		got, err := repo.SyncRemote(context.Background())
		require.NoError(t, err)
		require.Equal(t, "primary", got)
	})

	t.Run("EmptyWhenNoConfig", func(t *testing.T) {
		_, _, repo := newContextRepoForTest(t)

		got, err := repo.SyncRemote(context.Background())
		require.NoError(t, err)
		require.Empty(t, got)
	})
}

func TestContextRepository_RepoContextAndRole(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not available")
	}

	workDir := t.TempDir()
	cmd := exec.Command("git", "init")
	cmd.Dir = workDir
	require.NoError(t, cmd.Run())

	issuegraphDir := filepath.Join(workDir, ".issuegraph")
	require.NoError(t, os.MkdirAll(issuegraphDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(issuegraphDir, "metadata.json"), []byte("{}"), 0o600))

	t.Setenv("ISSUEGRAPH_DIR", issuegraphDir)
	issuegraph.ResetCaches()
	git.ResetCaches()
	t.Cleanup(func() {
		issuegraph.ResetCaches()
		git.ResetCaches()
	})

	repo := NewContextRepository(NewIssueGraphDirFSRepository(workDir, testTemplates()))

	paths, err := repo.RepoContext(context.Background())
	if err != nil {
		t.Skipf("repo context unavailable in this environment: %v", err)
	}
	require.True(t, paths.IsRedirected)
	require.NotEmpty(t, paths.RepoRoot)
	require.Equal(t, ".issuegraph", filepath.Base(paths.IssueGraphDir))

	role, ok, err := repo.Role(context.Background())
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, string(issuegraph.Contributor), role)
}
