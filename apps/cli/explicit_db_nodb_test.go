//go:build cgo

package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/borg0ai/dx-issuegraph/internal/configfile"
)

// buildIssueGraphUnderTest builds the issuegraph binary once per test process and returns the path.
// Previously each caller built a fresh binary in t.TempDir(), which on slow runners
// (macOS arm64) took 30-240s each and blew the 10m package timeout when many
// buildIssueGraphUnderTest-using tests ran together.
var (
	buildIssueGraphOnce sync.Once
	buildIssueGraphPath string
	buildIssueGraphErr  error
	buildIssueGraphDir  string
)

func buildIssueGraphUnderTest(t *testing.T) string {
	t.Helper()
	buildIssueGraphOnce.Do(func() {
		prebuilt, err := findPrebuiltIssueGraphBinary()
		if err != nil {
			buildIssueGraphErr = err
			return
		}
		if prebuilt != "" {
			buildIssueGraphPath = prebuilt
			return
		}
		dir, err := testTempDir("issuegraph-testbin-*")
		if err != nil {
			buildIssueGraphErr = err
			return
		}
		buildIssueGraphDir = dir
		binName := "issuegraph"
		if runtime.GOOS == "windows" {
			binName = "issuegraph.exe"
		}
		buildIssueGraphPath = filepath.Join(dir, binName)
		buildCmd := exec.Command("go", "build", "-tags", "gms_pure_go", "-o", buildIssueGraphPath, ".")
		if out, err := buildCmd.CombinedOutput(); err != nil {
			buildIssueGraphErr = &buildIssueGraphError{err: err, output: out}
			return
		}
	})
	if buildIssueGraphErr != nil {
		t.Fatalf("go build failed: %v", buildIssueGraphErr)
	}
	return buildIssueGraphPath
}

type buildIssueGraphError struct {
	err    error
	output []byte
}

func (e *buildIssueGraphError) Error() string {
	return e.err.Error() + "\n" + string(e.output)
}

func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	cmd := exec.Command("git", "init", "-q")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init %s: %v\n%s", dir, err, out)
	}
	writeFile(t, filepath.Join(dir, ".gitignore"), []byte(".issuegraph/.env\n"))
	commitCmd := exec.Command("git", "add", ".")
	commitCmd.Dir = dir
	_, _ = commitCmd.CombinedOutput()
	commitCmd = exec.Command("git", "commit", "-q", "-m", "init")
	commitCmd.Dir = dir
	commitCmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	_, _ = commitCmd.CombinedOutput()
}

func writeFile(t *testing.T, path string, content []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir parent for %s: %v", path, err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeServerRepo(t *testing.T, repoDir, database, host, syncRemote string, port int) string {
	return writeServerRepoWithDataDir(t, repoDir, database, host, syncRemote, port, "")
}

func writeServerRepoWithDataDir(t *testing.T, repoDir, database, host, syncRemote string, port int, doltDataDir string) string {
	t.Helper()
	initGitRepo(t, repoDir)
	issuegraphDir := filepath.Join(repoDir, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0o755); err != nil {
		t.Fatalf("mkdir issuegraph dir: %v", err)
	}
	doltDir := filepath.Join(issuegraphDir, "dolt")
	if doltDataDir != "" {
		doltDir = filepath.Join(issuegraphDir, doltDataDir)
	}
	if err := os.MkdirAll(doltDir, 0o755); err != nil {
		t.Fatalf("mkdir dolt dir: %v", err)
	}
	cfg := &configfile.Config{
		Backend:        configfile.BackendDolt,
		DoltMode:       configfile.DoltModeServer,
		DoltServerHost: host,
		DoltDatabase:   database,
	}
	if doltDataDir != "" {
		cfg.DoltDataDir = doltDataDir
	}
	if err := cfg.Save(issuegraphDir); err != nil {
		t.Fatalf("save metadata: %v", err)
	}
	// A server-mode workspace with a local Dolt root is current only when it
	// carries a valid post-1.0 witness; otherwise the upgrade guard correctly
	// treats the shape as an ambiguous historical workspace.
	if err := writeLocalVersion(filepath.Join(issuegraphDir, localVersionFile), Version); err != nil {
		t.Fatalf("write local version: %v", err)
	}
	writeFile(t, filepath.Join(issuegraphDir, "dolt-server.port"), []byte(strconv.Itoa(port)))
	writeFile(t, filepath.Join(issuegraphDir, "config.yaml"), []byte("sync:\n  git-remote: "+syncRemote+"\n"))
	writeFile(t, filepath.Join(issuegraphDir, ".env"), []byte("ISSUEGRAPH_DOLT_SERVER_HOST="+host+"\n"))
	return issuegraphDir
}

func writeProjectConfig(t *testing.T, issuegraphDir string, syncRemote string, port int, shared bool) {
	t.Helper()
	sharedText := "false"
	if shared {
		sharedText = "true"
	}
	writeFile(t, filepath.Join(issuegraphDir, "config.yaml"), []byte(
		"sync:\n  git-remote: "+syncRemote+"\n"+
			"dolt:\n  port: "+strconv.Itoa(port)+"\n  shared-server: "+sharedText+"\n",
	))
}

func writeIssuePrefixConfig(t *testing.T, issuegraphDir, prefix string) {
	t.Helper()
	writeFile(t, filepath.Join(issuegraphDir, "config.yaml"), []byte("issue-prefix: "+prefix+"\n"))
}

// evalPath resolves symlinks in a path for consistent comparison.
// On macOS, t.TempDir() returns /var/folders/... but binaries resolve
// it to /private/var/folders/..., causing string comparison failures.
func evalPath(t *testing.T, path string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatalf("EvalSymlinks(%s): %v", path, err)
	}
	return resolved
}

func decodeJSONOutput(t *testing.T, out []byte, target any) {
	t.Helper()
	trimmed := strings.TrimSpace(string(out))
	idx := strings.Index(trimmed, "{")
	if idx == -1 {
		t.Fatalf("no JSON object found in output:\n%s", out)
	}
	if err := json.Unmarshal([]byte(trimmed[idx:]), target); err != nil {
		t.Fatalf("unmarshal json: %v\n%s", err, out)
	}
}

func runIssueGraphCommand(t *testing.T, binPath, dir string, extraEnv []string, args ...string) []byte {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binPath, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"HOME="+t.TempDir(),
		"XDG_CONFIG_HOME="+t.TempDir(),
		"ISSUEGRAPH_TEST_IGNORE_REPO_CONFIG=1",
		"ISSUEGRAPH_DIR=",
		"ISSUEGRAPH_DB=",
		"ISSUEGRAPH_DOLT_SERVER_DATABASE=",
		"ISSUEGRAPH_DOLT_SERVER_HOST=",
		"ISSUEGRAPH_DOLT_SERVER_PORT=",
		"ISSUEGRAPH_DOLT_PORT=",
	)
	cmd.Env = append(cmd.Env, extraEnv...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			t.Fatalf("%v timed out after %s\n%s", args, 10*time.Second, out)
		}
		t.Fatalf("%v failed: %v\n%s", args, err, out)
	}
	return out
}

type whereExpectation struct {
	issuegraphDir string
	database      string
	prefix        string
	omitPrefix    bool
}

func assertWhereOutput(t *testing.T, out []byte, want whereExpectation) {
	t.Helper()

	var got map[string]any
	decodeJSONOutput(t, out, &got)

	if evalPath(t, got["path"].(string)) != evalPath(t, want.issuegraphDir) {
		t.Fatalf("path = %v, want %s", got["path"], want.issuegraphDir)
	}
	if evalPath(t, got["database_path"].(string)) != evalPath(t, want.database) {
		t.Fatalf("database_path = %v, want %s", got["database_path"], want.database)
	}
	if want.omitPrefix {
		if _, ok := got["prefix"]; ok {
			t.Fatalf("prefix = %v, want omitted when only server metadata is available", got["prefix"])
		}
		return
	}
	if got["prefix"] != want.prefix {
		t.Fatalf("prefix = %v, want %s", got["prefix"], want.prefix)
	}
}

func TestContextUsesExplicitDBFlagForNoDBCommand(t *testing.T) {
	binPath := buildIssueGraphUnderTest(t)
	root := t.TempDir()
	repoA := filepath.Join(root, "repo-a")
	repoB := filepath.Join(root, "repo-b")
	writeServerRepo(t, repoA, "repo_a_db", "10.0.0.1", "origin-a", 3311)
	issuegraphDirB := writeServerRepo(t, repoB, "repo_b_db", "10.0.0.2", "origin-b", 3312)

	out := runIssueGraphCommand(t, binPath, repoA, nil, "--db", filepath.Join(issuegraphDirB, "dolt"), "context", "--json")

	var got map[string]any
	decodeJSONOutput(t, out, &got)
	if evalPath(t, got["issuegraph_dir"].(string)) != evalPath(t, issuegraphDirB) {
		t.Fatalf("issuegraph_dir = %v, want %s", got["issuegraph_dir"], issuegraphDirB)
	}
	if got["database"] != "repo_b_db" {
		t.Fatalf("database = %v, want repo_b_db", got["database"])
	}
	if got["server_host"] != "10.0.0.2" {
		t.Fatalf("server_host = %v, want 10.0.0.2", got["server_host"])
	}
	if got["sync_git_remote"] != "origin-b" {
		t.Fatalf("sync_git_remote = %v, want origin-b", got["sync_git_remote"])
	}
}

func TestWhereUsesExplicitDBFlagForNoDBCommand(t *testing.T) {
	binPath := buildIssueGraphUnderTest(t)
	root := t.TempDir()
	repoA := filepath.Join(root, "repo-a")
	repoB := filepath.Join(root, "repo-b")
	issuegraphDirA := writeServerRepo(t, repoA, "repo_a_db", "10.0.0.1", "origin-a", 3311)
	issuegraphDirB := writeServerRepo(t, repoB, "repo_b_db", "10.0.0.2", "origin-b", 3312)
	writeIssuePrefixConfig(t, issuegraphDirA, "repo-a")
	writeIssuePrefixConfig(t, issuegraphDirB, "repo-b")

	out := runIssueGraphCommand(t, binPath, repoA, nil, "--db", filepath.Join(issuegraphDirB, "dolt"), "where", "--json")
	assertWhereOutput(t, out, whereExpectation{
		issuegraphDir: issuegraphDirB,
		database:      filepath.Join(issuegraphDirB, "dolt"),
		prefix:        "repo-b",
	})
}

func TestDoltShowUsesExplicitDBFlagForNoDBCommand(t *testing.T) {
	binPath := buildIssueGraphUnderTest(t)
	root := t.TempDir()
	repoA := filepath.Join(root, "repo-a")
	repoB := filepath.Join(root, "repo-b")
	writeServerRepo(t, repoA, "repo_a_db", "10.0.0.1", "origin-a", 3311)
	issuegraphDirB := writeServerRepo(t, repoB, "repo_b_db", "10.0.0.2", "origin-b", 3312)

	out := runIssueGraphCommand(t, binPath, repoA, nil, "--db", filepath.Join(issuegraphDirB, "dolt"), "dolt", "show", "--json")

	var got map[string]any
	decodeJSONOutput(t, out, &got)
	if got["database"] != "repo_b_db" {
		t.Fatalf("database = %v, want repo_b_db", got["database"])
	}
	if got["host"] != "10.0.0.2" {
		t.Fatalf("host = %v, want 10.0.0.2", got["host"])
	}
	if got["port"] != float64(3312) {
		t.Fatalf("port = %v, want 3312", got["port"])
	}
}

func TestContextUsesISSUEGRAPHDBForNoDBCommand(t *testing.T) {
	binPath := buildIssueGraphUnderTest(t)
	root := t.TempDir()
	repoA := filepath.Join(root, "repo-a")
	repoB := filepath.Join(root, "repo-b")
	writeServerRepo(t, repoA, "repo_a_db", "10.0.0.1", "origin-a", 3311)
	issuegraphDirB := writeServerRepo(t, repoB, "repo_b_db", "10.0.0.2", "origin-b", 3312)

	out := runIssueGraphCommand(t, binPath, repoA, []string{"ISSUEGRAPH_DB=" + filepath.Join(issuegraphDirB, "dolt")}, "context", "--json")

	var got map[string]any
	decodeJSONOutput(t, out, &got)
	if evalPath(t, got["issuegraph_dir"].(string)) != evalPath(t, issuegraphDirB) {
		t.Fatalf("issuegraph_dir = %v, want %s", got["issuegraph_dir"], issuegraphDirB)
	}
	if got["database"] != "repo_b_db" {
		t.Fatalf("database = %v, want repo_b_db", got["database"])
	}
}

func TestWhereUsesISSUEGRAPHDBForNoDBCommand(t *testing.T) {
	binPath := buildIssueGraphUnderTest(t)
	root := t.TempDir()
	repoA := filepath.Join(root, "repo-a")
	repoB := filepath.Join(root, "repo-b")
	writeServerRepo(t, repoA, "repo_a_db", "10.0.0.1", "origin-a", 3311)
	issuegraphDirB := writeServerRepo(t, repoB, "repo_b_db", "10.0.0.2", "origin-b", 3312)
	writeIssuePrefixConfig(t, issuegraphDirB, "repo-b")

	out := runIssueGraphCommand(t, binPath, repoA, []string{"ISSUEGRAPH_DB=" + filepath.Join(issuegraphDirB, "dolt")}, "where", "--json")
	assertWhereOutput(t, out, whereExpectation{
		issuegraphDir: issuegraphDirB,
		database:      filepath.Join(issuegraphDirB, "dolt"),
		prefix:        "repo-b",
	})
}

func TestContextUsesISSUEGRAPHDBDirectoryForNoDBCommand(t *testing.T) {
	binPath := buildIssueGraphUnderTest(t)
	root := t.TempDir()
	repoA := filepath.Join(root, "repo-a")
	repoB := filepath.Join(root, "repo-b")
	writeServerRepo(t, repoA, "repo_a_db", "10.0.0.1", "origin-a", 3311)
	issuegraphDirB := writeServerRepo(t, repoB, "repo_b_db", "10.0.0.2", "origin-b", 3312)

	out := runIssueGraphCommand(t, binPath, repoA, []string{"ISSUEGRAPH_DB=" + issuegraphDirB}, "context", "--json")

	var got map[string]any
	decodeJSONOutput(t, out, &got)
	if evalPath(t, got["issuegraph_dir"].(string)) != evalPath(t, issuegraphDirB) {
		t.Fatalf("issuegraph_dir = %v, want %s", got["issuegraph_dir"], issuegraphDirB)
	}
	if got["database"] != "repo_b_db" {
		t.Fatalf("database = %v, want repo_b_db", got["database"])
	}
}

func TestContextUsesExplicitDBFlagForExternalDoltDataDir(t *testing.T) {
	binPath := buildIssueGraphUnderTest(t)
	root := t.TempDir()
	repoA := filepath.Join(root, "repo-a")
	repoB := filepath.Join(root, "repo-b")
	writeServerRepo(t, repoA, "repo_a_db", "10.0.0.1", "origin-a", 3311)
	issuegraphDirB := writeServerRepoWithDataDir(t, repoB, "repo_b_db", "10.0.0.2", "origin-b", 3312, "../external-dolt")

	out := runIssueGraphCommand(t, binPath, repoA, nil, "--db", filepath.Join(issuegraphDirB, "../external-dolt"), "context", "--json")

	var got map[string]any
	decodeJSONOutput(t, out, &got)
	if evalPath(t, got["issuegraph_dir"].(string)) != evalPath(t, issuegraphDirB) {
		t.Fatalf("issuegraph_dir = %v, want %s", got["issuegraph_dir"], issuegraphDirB)
	}
	if got["database"] != "repo_b_db" {
		t.Fatalf("database = %v, want repo_b_db", got["database"])
	}
	if got["server_host"] != "10.0.0.2" {
		t.Fatalf("server_host = %v, want 10.0.0.2", got["server_host"])
	}
}

func TestWhereUsesExplicitDBFlagForExternalDoltDataDir(t *testing.T) {
	binPath := buildIssueGraphUnderTest(t)
	root := t.TempDir()
	repoA := filepath.Join(root, "repo-a")
	repoB := filepath.Join(root, "repo-b")
	writeServerRepo(t, repoA, "repo_a_db", "10.0.0.1", "origin-a", 3311)
	issuegraphDirB := writeServerRepoWithDataDir(t, repoB, "repo_b_db", "10.0.0.2", "origin-b", 3312, "../external-dolt")
	writeIssuePrefixConfig(t, issuegraphDirB, "repo-b")

	out := runIssueGraphCommand(t, binPath, repoA, nil, "--db", filepath.Join(issuegraphDirB, "../external-dolt"), "where", "--json")
	assertWhereOutput(t, out, whereExpectation{
		issuegraphDir: issuegraphDirB,
		database:      filepath.Join(issuegraphDirB, "../external-dolt"),
		prefix:        "repo-b",
	})
}

func TestContextExplicitDBFlagOverridesISSUEGRAPHDBForNoDBCommand(t *testing.T) {
	binPath := buildIssueGraphUnderTest(t)
	root := t.TempDir()
	repoA := filepath.Join(root, "repo-a")
	repoB := filepath.Join(root, "repo-b")
	repoC := filepath.Join(root, "repo-c")
	writeServerRepo(t, repoA, "repo_a_db", "10.0.0.1", "origin-a", 3311)
	issuegraphDirB := writeServerRepo(t, repoB, "repo_b_db", "10.0.0.2", "origin-b", 3312)
	issuegraphDirC := writeServerRepo(t, repoC, "repo_c_db", "10.0.0.3", "origin-c", 3313)

	out := runIssueGraphCommand(t, binPath, repoA, []string{"ISSUEGRAPH_DB=" + filepath.Join(issuegraphDirC, "dolt")}, "--db", filepath.Join(issuegraphDirB, "dolt"), "context", "--json")

	var got map[string]any
	decodeJSONOutput(t, out, &got)
	if evalPath(t, got["issuegraph_dir"].(string)) != evalPath(t, issuegraphDirB) {
		t.Fatalf("issuegraph_dir = %v, want %s", got["issuegraph_dir"], issuegraphDirB)
	}
	if got["database"] != "repo_b_db" {
		t.Fatalf("database = %v, want repo_b_db", got["database"])
	}
}

func TestWhereExplicitDBFlagOverridesISSUEGRAPHDBForNoDBCommand(t *testing.T) {
	binPath := buildIssueGraphUnderTest(t)
	root := t.TempDir()
	repoA := filepath.Join(root, "repo-a")
	repoB := filepath.Join(root, "repo-b")
	repoC := filepath.Join(root, "repo-c")
	writeServerRepo(t, repoA, "repo_a_db", "10.0.0.1", "origin-a", 3311)
	issuegraphDirB := writeServerRepo(t, repoB, "repo_b_db", "10.0.0.2", "origin-b", 3312)
	issuegraphDirC := writeServerRepo(t, repoC, "repo_c_db", "10.0.0.3", "origin-c", 3313)
	writeIssuePrefixConfig(t, issuegraphDirB, "repo-b")
	writeIssuePrefixConfig(t, issuegraphDirC, "repo-c")

	out := runIssueGraphCommand(t, binPath, repoA, []string{"ISSUEGRAPH_DB=" + filepath.Join(issuegraphDirC, "dolt")}, "--db", filepath.Join(issuegraphDirB, "dolt"), "where", "--json")
	assertWhereOutput(t, out, whereExpectation{
		issuegraphDir: issuegraphDirB,
		database:      filepath.Join(issuegraphDirB, "dolt"),
		prefix:        "repo-b",
	})
}

func TestWhereUsesExplicitDBFlagForMetadataOnlyServerRepo(t *testing.T) {
	binPath := buildIssueGraphUnderTest(t)
	root := t.TempDir()
	repoA := filepath.Join(root, "repo-a")
	repoB := filepath.Join(root, "repo-b")
	writeServerRepo(t, repoA, "repo_a_db", "10.0.0.1", "origin-a", 3311)
	issuegraphDirB := writeServerRepo(t, repoB, "repo_b_db", "10.0.0.2", "origin-b", 3312)

	out := runIssueGraphCommand(t, binPath, repoA, nil, "--db", filepath.Join(issuegraphDirB, "dolt"), "where", "--json")
	assertWhereOutput(t, out, whereExpectation{
		issuegraphDir: issuegraphDirB,
		database:      filepath.Join(issuegraphDirB, "dolt"),
		omitPrefix:    true,
	})
}

func TestWhereISSUEGRAPHDBOverridesIssueGraphDBForNoDBCommand(t *testing.T) {
	binPath := buildIssueGraphUnderTest(t)
	root := t.TempDir()
	repoA := filepath.Join(root, "repo-a")
	repoB := filepath.Join(root, "repo-b")
	repoC := filepath.Join(root, "repo-c")
	writeServerRepo(t, repoA, "repo_a_db", "10.0.0.1", "origin-a", 3311)
	issuegraphDirB := writeServerRepo(t, repoB, "repo_b_db", "10.0.0.2", "origin-b", 3312)
	issuegraphDirC := writeServerRepo(t, repoC, "repo_c_db", "10.0.0.3", "origin-c", 3313)
	writeIssuePrefixConfig(t, issuegraphDirB, "repo-b")
	writeIssuePrefixConfig(t, issuegraphDirC, "repo-c")

	out := runIssueGraphCommand(t, binPath, repoA, []string{
		"ISSUEGRAPH_DB=" + filepath.Join(issuegraphDirB, "dolt"),
		"ISSUEGRAPH_DB=" + filepath.Join(issuegraphDirC, "dolt"),
	}, "where", "--json")
	assertWhereOutput(t, out, whereExpectation{
		issuegraphDir: issuegraphDirB,
		database:      filepath.Join(issuegraphDirB, "dolt"),
		prefix:        "repo-b",
	})
}

func TestContextISSUEGRAPHDBOverridesIssueGraphDBForNoDBCommand(t *testing.T) {
	binPath := buildIssueGraphUnderTest(t)
	root := t.TempDir()
	repoA := filepath.Join(root, "repo-a")
	repoB := filepath.Join(root, "repo-b")
	repoC := filepath.Join(root, "repo-c")
	writeServerRepo(t, repoA, "repo_a_db", "10.0.0.1", "origin-a", 3311)
	issuegraphDirB := writeServerRepo(t, repoB, "repo_b_db", "10.0.0.2", "origin-b", 3312)
	issuegraphDirC := writeServerRepo(t, repoC, "repo_c_db", "10.0.0.3", "origin-c", 3313)

	out := runIssueGraphCommand(t, binPath, repoA, []string{"ISSUEGRAPH_DB=" + filepath.Join(issuegraphDirB, "dolt"), "ISSUEGRAPH_DB=" + filepath.Join(issuegraphDirC, "dolt")}, "context", "--json")

	var got map[string]any
	decodeJSONOutput(t, out, &got)
	if evalPath(t, got["issuegraph_dir"].(string)) != evalPath(t, issuegraphDirB) {
		t.Fatalf("issuegraph_dir = %v, want %s", got["issuegraph_dir"], issuegraphDirB)
	}
	if got["database"] != "repo_b_db" {
		t.Fatalf("database = %v, want repo_b_db", got["database"])
	}
}

func TestContextPreservesSourceDatabaseAcrossRedirectForNoDBCommand(t *testing.T) {
	binPath := buildIssueGraphUnderTest(t)
	root := t.TempDir()
	repoA := filepath.Join(root, "repo-a")
	repoB := filepath.Join(root, "repo-b")
	writeServerRepo(t, repoA, "repo_a_db", "10.0.0.1", "origin-a", 3311)
	issuegraphDirB := writeServerRepo(t, repoB, "repo_b_db", "10.0.0.2", "origin-b", 3312)
	sharedIssueGraphDir := filepath.Join(repoB, "shared-issuegraph")
	if err := os.RemoveAll(issuegraphDirB); err != nil {
		t.Fatalf("remove source issuegraph dir: %v", err)
	}
	if err := os.MkdirAll(issuegraphDirB, 0o755); err != nil {
		t.Fatalf("mkdir source issuegraph dir: %v", err)
	}
	writeFile(t, filepath.Join(issuegraphDirB, "redirect"), []byte("../shared-issuegraph\n"))
	if err := (&configfile.Config{
		Backend:        configfile.BackendDolt,
		DoltMode:       configfile.DoltModeServer,
		DoltServerHost: "10.0.0.2",
		DoltDatabase:   "repo_b_db",
	}).Save(issuegraphDirB); err != nil {
		t.Fatalf("save source metadata: %v", err)
	}
	writeFile(t, filepath.Join(issuegraphDirB, "dolt-server.port"), []byte("3312"))
	if err := os.MkdirAll(filepath.Join(sharedIssueGraphDir, "dolt"), 0o755); err != nil {
		t.Fatalf("mkdir shared dolt dir: %v", err)
	}
	if err := (&configfile.Config{
		Backend:        configfile.BackendDolt,
		DoltMode:       configfile.DoltModeServer,
		DoltServerHost: "10.0.0.9",
		DoltDatabase:   "shared_db",
	}).Save(sharedIssueGraphDir); err != nil {
		t.Fatalf("save shared metadata: %v", err)
	}
	writeFile(t, filepath.Join(sharedIssueGraphDir, "dolt-server.port"), []byte("3399"))
	writeFile(t, filepath.Join(sharedIssueGraphDir, "config.yaml"), []byte("sync:\n  git-remote: shared-origin\n"))
	writeFile(t, filepath.Join(sharedIssueGraphDir, ".env"), []byte("ISSUEGRAPH_DOLT_SERVER_HOST=10.0.0.9\n"))

	out := runIssueGraphCommand(t, binPath, repoA, nil, "--db", filepath.Join(issuegraphDirB, "dolt"), "context", "--json")

	var got map[string]any
	decodeJSONOutput(t, out, &got)
	if got["database"] != "repo_b_db" {
		t.Fatalf("database = %v, want repo_b_db", got["database"])
	}
}

func TestDoltShowUsesSelectedRepoConfigForNoDBCommand(t *testing.T) {
	binPath := buildIssueGraphUnderTest(t)
	root := t.TempDir()
	repoA := filepath.Join(root, "repo-a")
	repoB := filepath.Join(root, "repo-b")
	issuegraphDirA := writeServerRepo(t, repoA, "repo_a_db", "10.0.0.1", "origin-a", 3311)
	issuegraphDirB := writeServerRepo(t, repoB, "repo_b_db", "10.0.0.2", "origin-b", 3312)
	if err := os.Remove(filepath.Join(issuegraphDirA, "dolt-server.port")); err != nil {
		t.Fatalf("remove port file A: %v", err)
	}
	if err := os.Remove(filepath.Join(issuegraphDirB, "dolt-server.port")); err != nil {
		t.Fatalf("remove port file B: %v", err)
	}
	writeProjectConfig(t, issuegraphDirA, "origin-a", 4401, true)
	writeProjectConfig(t, issuegraphDirB, "origin-b", 4402, false)

	out := runIssueGraphCommand(t, binPath, repoA, nil, "--db", filepath.Join(issuegraphDirB, "dolt"), "dolt", "show", "--json")

	var got map[string]any
	decodeJSONOutput(t, out, &got)
	if got["host"] != "10.0.0.2" {
		t.Fatalf("host = %v, want 10.0.0.2", got["host"])
	}
	if got["port"] != float64(4402) {
		t.Fatalf("port = %v, want 4402", got["port"])
	}
	if got["shared_server"] != false {
		t.Fatalf("shared_server = %v, want false", got["shared_server"])
	}
	if got["database"] != "repo_b_db" {
		t.Fatalf("database = %v, want repo_b_db", got["database"])
	}
}

func TestContextPreservesShellEnvPrecedenceForNoDBCommand(t *testing.T) {
	binPath := buildIssueGraphUnderTest(t)
	root := t.TempDir()
	repoA := filepath.Join(root, "repo-a")
	repoB := filepath.Join(root, "repo-b")
	writeServerRepo(t, repoA, "repo_a_db", "10.0.0.1", "origin-a", 3311)
	issuegraphDirB := writeServerRepo(t, repoB, "repo_b_db", "10.0.0.2", "origin-b", 3312)

	out := runIssueGraphCommand(t, binPath, repoA, []string{"ISSUEGRAPH_DOLT_SERVER_HOST=9.9.9.9"}, "--db", filepath.Join(issuegraphDirB, "dolt"), "context", "--json")

	var got map[string]any
	decodeJSONOutput(t, out, &got)
	if got["server_host"] != "9.9.9.9" {
		t.Fatalf("server_host = %v, want 9.9.9.9", got["server_host"])
	}
	if got["database"] != "repo_b_db" {
		t.Fatalf("database = %v, want repo_b_db", got["database"])
	}
}

func TestDoltShowUsesISSUEGRAPHDBForNoDBCommand(t *testing.T) {
	binPath := buildIssueGraphUnderTest(t)
	root := t.TempDir()
	repoA := filepath.Join(root, "repo-a")
	repoB := filepath.Join(root, "repo-b")
	writeServerRepo(t, repoA, "repo_a_db", "10.0.0.1", "origin-a", 3311)
	issuegraphDirB := writeServerRepo(t, repoB, "repo_b_db", "10.0.0.2", "origin-b", 3312)

	out := runIssueGraphCommand(t, binPath, repoA, []string{"ISSUEGRAPH_DB=" + filepath.Join(issuegraphDirB, "dolt")}, "dolt", "show", "--json")

	var got map[string]any
	decodeJSONOutput(t, out, &got)
	if got["database"] != "repo_b_db" {
		t.Fatalf("database = %v, want repo_b_db", got["database"])
	}
	if got["host"] != "10.0.0.2" {
		t.Fatalf("host = %v, want 10.0.0.2", got["host"])
	}
}
