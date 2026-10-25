//go:build cgo

package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/borg0ai/dx-issuegraph/internal/storage/dbproxy/pidfile"
	"github.com/borg0ai/dx-issuegraph/internal/storage/dbproxy/proxy"
	"github.com/borg0ai/dx-issuegraph/internal/testutil"
	"github.com/borg0ai/dx-issuegraph/internal/types"
)

func requireProxiedServerEnv(t *testing.T) {
	t.Helper()
	if os.Getenv("ISSUEGRAPH_TEST_PROXIED_SERVER") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_PROXIED_SERVER=1 to run proxied-server integration tests")
	}
	testutil.RequireDoltBinary(t)
}

func issuegraphProxiedEnv(dir string) []string {
	var env []string
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "ISSUEGRAPH_") {
			continue
		}
		env = append(env, e)
	}
	return append(env,
		"HOME="+dir,
		"ISSUEGRAPH_DOLT_PROXIED_SERVER=1",
		"ISSUEGRAPH_NO_DAEMON=1",
		"ISSUEGRAPH_DISABLE_METRICS=1",
		"ISSUEGRAPH_DISABLE_EVENT_FLUSH=1",
	)
}

func issuegraphProxiedRun(t *testing.T, issuegraph, dir string, args ...string) ([]byte, error) {
	t.Helper()
	cmd := exec.Command(issuegraph, args...)
	cmd.Dir = dir
	cmd.Env = issuegraphProxiedEnv(dir)
	stdout, stderr, err := runCommandBuffers(t, cmd)
	if err != nil {
		return append(stdout.Bytes(), stderr.Bytes()...), err
	}
	return stdout.Bytes(), nil
}

func issuegraphProxiedCreate(t *testing.T, issuegraph, dir string, args ...string) *types.Issue {
	t.Helper()
	fullArgs := append([]string{"create", "--json"}, args...)
	out, err := issuegraphProxiedRun(t, issuegraph, dir, fullArgs...)
	if err != nil {
		t.Fatalf("issuegraph create %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}
	return parseIssueJSON(t, out)
}

func issuegraphProxiedCreateSilent(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"create", "--silent"}, args...)
	out, err := issuegraphProxiedRun(t, issuegraph, dir, fullArgs...)
	if err != nil {
		t.Fatalf("issuegraph create --silent %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

func issuegraphProxiedCreateFail(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"create"}, args...)
	out, err := issuegraphProxiedRun(t, issuegraph, dir, fullArgs...)
	if err == nil {
		t.Fatalf("issuegraph create %s should have failed; got:\n%s", strings.Join(args, " "), out)
	}
	return string(out)
}

func issuegraphProxiedList(t *testing.T, issuegraph string, p proxiedProject, args ...string) string {
	t.Helper()
	stdout, _ := issuegraphProxiedListCapture(t, issuegraph, p, args...)
	return stdout
}

func issuegraphProxiedListJSON(t *testing.T, issuegraph string, p proxiedProject, args ...string) []*types.IssueWithCounts {
	t.Helper()
	fullArgs := append([]string{"list", "--json"}, args...)
	stdout, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, fullArgs...)
	if err != nil {
		t.Fatalf("issuegraph list --json %s failed: %v\nstdout:\n%s\nstderr:\n%s",
			strings.Join(args, " "), err, stdout, stderr)
	}
	start := strings.Index(stdout, "[")
	if start < 0 {
		if strings.Contains(stdout, "null") || strings.TrimSpace(stdout) == "" {
			return nil
		}
		t.Fatalf("no JSON array found in output:\n%s", stdout)
	}
	var issues []*types.IssueWithCounts
	if err := json.Unmarshal([]byte(stdout[start:]), &issues); err != nil {
		t.Fatalf("failed to parse JSON list output: %v\nraw: %s", err, stdout[start:])
	}
	return issues
}

func issuegraphProxiedListCapture(t *testing.T, issuegraph string, p proxiedProject, args ...string) (string, string) {
	t.Helper()
	fullArgs := append([]string{"list"}, args...)
	stdout, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, fullArgs...)
	if err != nil {
		t.Fatalf("issuegraph list %s failed: %v\nstdout:\n%s\nstderr:\n%s",
			strings.Join(args, " "), err, stdout, stderr)
	}
	return stdout, stderr
}

func issuegraphProxiedListFail(t *testing.T, issuegraph string, p proxiedProject, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"list"}, args...)
	stdout, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, fullArgs...)
	if err == nil {
		t.Fatalf("expected issuegraph list %s to fail, but it succeeded:\nstdout:\n%s\nstderr:\n%s",
			strings.Join(args, " "), stdout, stderr)
	}
	return stdout + stderr
}

func issuegraphProxiedRunBuffers(t *testing.T, issuegraph, dir string, args ...string) (string, string, error) {
	t.Helper()
	return issuegraphProxiedRunBuffersWithEnv(t, issuegraph, dir, nil, args...)
}

// issuegraphProxiedRunBuffersWithEnv is issuegraphProxiedRunBuffers with extra environment
// variables appended after the standard proxied env (so they can override
// it, e.g. ISSUEGRAPH_MAX_ROWS for the proxied-server MaxRows-rejection tests).
func issuegraphProxiedRunBuffersWithEnv(t *testing.T, issuegraph, dir string, envExtras []string, args ...string) (string, string, error) {
	t.Helper()
	cmd := exec.Command(issuegraph, args...)
	cmd.Dir = dir
	cmd.Env = append(issuegraphProxiedEnv(dir), envExtras...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// issuegraphProxiedRunDeadline is issuegraphProxiedRunBuffers for a command that may never exit
// on its own — `issuegraph list --watch` polls until it is interrupted. It reports
// whether the deadline was what stopped it, so a case can tell "kept running"
// from "exited" instead of hanging the suite to find out.
func issuegraphProxiedRunDeadline(t *testing.T, issuegraph, dir string, timeout time.Duration, args ...string) (stdout, stderr string, err error, timedOut bool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, issuegraph, args...)
	cmd.Dir = dir
	cmd.Env = issuegraphProxiedEnv(dir)
	var out, errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	err = cmd.Run()
	return out.String(), errOut.String(), err, ctx.Err() != nil
}

// issuegraphProxiedRunUntilStderr starts a issuegraph command that is NOT expected to exit and
// stops it the moment marker appears on stderr. reached is false when the
// command exited, or the timeout elapsed, before it got there.
//
// The alternative is to run such a command under a fixed deadline and wait the
// whole thing out, which costs the deadline on every green run and fails on a
// machine slow enough to render late. This waits for the state instead, so the
// timeout is only a bound on a hang.
func issuegraphProxiedRunUntilStderr(t *testing.T, issuegraph, dir, marker string, timeout time.Duration, args ...string) (stdout, stderr string, reached bool) {
	t.Helper()
	w := &markerWriter{marker: marker, seen: make(chan struct{})}
	var out bytes.Buffer
	cmd := exec.Command(issuegraph, args...)
	cmd.Dir = dir
	cmd.Env = issuegraphProxiedEnv(dir)
	cmd.Stdout = &out
	cmd.Stderr = w
	if err := cmd.Start(); err != nil {
		t.Fatalf("start issuegraph %s: %v", strings.Join(args, " "), err)
	}
	exited := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(exited)
	}()
	select {
	case <-w.seen:
		reached = true
	case <-exited:
	case <-time.After(timeout):
	}
	_ = cmd.Process.Kill()
	<-exited
	return out.String(), w.String(), reached
}

// markerWriter buffers everything written to it and closes seen the first time
// marker shows up.
type markerWriter struct {
	marker string
	seen   chan struct{}

	mu   sync.Mutex
	buf  bytes.Buffer
	once sync.Once
}

func (w *markerWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	n, err := w.buf.Write(p)
	if strings.Contains(w.buf.String(), w.marker) {
		w.once.Do(func() { close(w.seen) })
	}
	return n, err
}

func (w *markerWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.String()
}

func issuegraphProxiedUpdate(t *testing.T, issuegraph, dir string, args ...string) []*types.Issue {
	t.Helper()
	fullArgs := append([]string{"update", "--json"}, args...)
	out, err := issuegraphProxiedRun(t, issuegraph, dir, fullArgs...)
	if err != nil {
		t.Fatalf("issuegraph update %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}
	s := string(out)
	start := strings.Index(s, "[")
	if start < 0 {
		t.Fatalf("no JSON array found in update output:\n%s", s)
	}
	var issues []*types.Issue
	if err := json.Unmarshal([]byte(s[start:]), &issues); err != nil {
		t.Fatalf("failed to parse update JSON: %v\nraw: %s", err, s[start:])
	}
	if len(issues) == 0 {
		t.Fatalf("update returned empty JSON array:\n%s", s)
	}
	return issues
}

func issuegraphProxiedUpdateOne(t *testing.T, issuegraph, dir string, args ...string) *types.Issue {
	t.Helper()
	issues := issuegraphProxiedUpdate(t, issuegraph, dir, args...)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue in update output, got %d", len(issues))
	}
	return issues[0]
}

func issuegraphProxiedUpdateRaw(t *testing.T, issuegraph, dir string, args ...string) (string, string, error) {
	t.Helper()
	fullArgs := append([]string{"update"}, args...)
	return issuegraphProxiedRunBuffers(t, issuegraph, dir, fullArgs...)
}

func issuegraphProxiedUpdateFail(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	stdout, stderr, err := issuegraphProxiedUpdateRaw(t, issuegraph, dir, args...)
	if err == nil {
		t.Fatalf("issuegraph update %s should have failed; got:\nstdout:\n%s\nstderr:\n%s",
			strings.Join(args, " "), stdout, stderr)
	}
	return stdout + stderr
}

func issuegraphProxiedShow(t *testing.T, issuegraph, dir, id string) *types.Issue {
	t.Helper()
	out, err := issuegraphProxiedRun(t, issuegraph, dir, "show", id, "--json")
	if err != nil {
		t.Fatalf("issuegraph show %s --json failed: %v\n%s", id, err, out)
	}
	return parseIssueJSON(t, out)
}

func issuegraphProxiedShowRaw(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"show"}, args...)
	out, err := issuegraphProxiedRun(t, issuegraph, dir, fullArgs...)
	if err != nil {
		t.Fatalf("issuegraph show %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

func issuegraphProxiedShowFail(t *testing.T, issuegraph, dir string, args ...string) (string, string) {
	t.Helper()
	fullArgs := append([]string{"show"}, args...)
	stdout, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, dir, fullArgs...)
	if err == nil {
		t.Fatalf("issuegraph show %s should have failed; got stdout:\n%s\nstderr:\n%s",
			strings.Join(args, " "), stdout, stderr)
	}
	return stdout, stderr
}

func issuegraphProxiedShowDetailsAll(t *testing.T, issuegraph, dir string, args ...string) []map[string]interface{} {
	t.Helper()
	fullArgs := append([]string{"show", "--json"}, args...)
	out, err := issuegraphProxiedRun(t, issuegraph, dir, fullArgs...)
	if err != nil {
		t.Fatalf("issuegraph show --json %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}
	s := strings.TrimSpace(string(out))
	start := strings.IndexAny(s, "[{")
	if start < 0 {
		t.Fatalf("no JSON in show output: %s", s)
	}
	s = s[start:]
	if strings.HasPrefix(s, "[") {
		var arr []map[string]interface{}
		if err := json.Unmarshal([]byte(s), &arr); err != nil {
			t.Fatalf("parse show JSON array: %v\n%s", err, s)
		}
		return arr
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		t.Fatalf("parse show JSON: %v\n%s", err, s)
	}
	return []map[string]interface{}{m}
}

func issuegraphProxiedShowDetailsFirst(t *testing.T, issuegraph, dir string, args ...string) map[string]interface{} {
	t.Helper()
	arr := issuegraphProxiedShowDetailsAll(t, issuegraph, dir, args...)
	if len(arr) == 0 {
		t.Fatalf("issuegraph show --json %s returned empty array", strings.Join(args, " "))
	}
	return arr[0]
}

type proxiedProject struct {
	dir           string
	issuegraphDir string
	proxyRoot     string
	database      string
	prefix        string
}

func issuegraphProxiedInitWithHooks(t *testing.T, issuegraph, prefix string, hooks map[string]string, extraInitArgs ...string) proxiedProject {
	t.Helper()
	p := issuegraphProxiedInitInternal(t, issuegraph, prefix, false, extraInitArgs...)
	hooksDir := filepath.Join(p.issuegraphDir, "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatalf("mkdir hooks dir: %v", err)
	}
	for name, body := range hooks {
		hookPath := filepath.Join(hooksDir, name)
		if err := os.WriteFile(hookPath, []byte(body), 0o755); err != nil {
			t.Fatalf("write hook %s: %v", name, err)
		}
	}
	return p
}

func issuegraphProxiedInit(t *testing.T, issuegraph, prefix string, extraInitArgs ...string) proxiedProject {
	t.Helper()
	return issuegraphProxiedInitInternal(t, issuegraph, prefix, true, extraInitArgs...)
}

func issuegraphProxiedInitInternal(t *testing.T, issuegraph, prefix string, skipHooks bool, extraInitArgs ...string) proxiedProject {
	t.Helper()

	dir := t.TempDir()
	initGitRepoAt(t, dir)
	issuegraphDir := filepath.Join(dir, ".issuegraph")
	proxyRoot := filepath.Join(issuegraphDir, "dolt")
	t.Cleanup(func() {
		if err := proxy.Shutdown(proxyRoot); err != nil {
			t.Logf("proxy.Shutdown(%s): %v", proxyRoot, err)
		}
	})
	shutdownProxyOnInterrupt(t, proxyRoot)

	baseArgs := []string{
		"init",
		"--proxied-server",
		"--quiet",
		"--prefix", prefix,
		"--non-interactive",
		"--skip-agents",
	}
	if skipHooks {
		baseArgs = append(baseArgs, "--skip-hooks")
	}
	args := append(baseArgs, extraInitArgs...)

	cmd := exec.Command(issuegraph, args...)
	cmd.Dir = dir
	cmd.Env = issuegraphProxiedEnv(dir)
	stdout, stderr, err := runCommandBuffers(t, cmd)
	if err != nil {
		t.Fatalf("issuegraph init --proxied-server failed: %v\nstdout:\n%s\nstderr:\n%s",
			err, stdout.String(), stderr.String())
	}

	database := sanitizePrefixForDB(prefix)

	return proxiedProject{
		dir:           dir,
		issuegraphDir: issuegraphDir,
		proxyRoot:     proxyRoot,
		database:      database,
		prefix:        prefix,
	}
}

func sanitizePrefixForDB(p string) string {
	p = strings.TrimLeft(p, ".")
	p = strings.TrimRight(p, "-")
	p = strings.ReplaceAll(p, ".", "_")
	if len(p) == 0 {
		return "issuegraph"
	}
	c := p[0]
	if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_') {
		p = "issuegraph_" + p
	}
	return p
}

func shutdownProxyOnInterrupt(t *testing.T, proxyRoot string) {
	t.Helper()
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan struct{})
	go func() {
		select {
		case <-ch:
			_ = proxy.Shutdown(proxyRoot)
			os.Exit(1)
		case <-done:
		}
	}()
	t.Cleanup(func() {
		signal.Stop(ch)
		close(done)
	})
}

func openProxiedDB(t *testing.T, p proxiedProject) *sql.DB {
	t.Helper()
	pf, err := pidfile.Read(p.proxyRoot, proxy.PIDFileName)
	if err != nil || pf == nil {
		t.Fatalf("read proxy pidfile %s: %v (pf=%v)", p.proxyRoot, err, pf)
	}

	dsn := fmt.Sprintf("root:@tcp(127.0.0.1:%d)/%s?multiStatements=true&parseTime=true",
		pf.Port, p.database)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("sql.Open mysql %s: %v", dsn, err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping proxied db: %v", err)
	}
	return db
}

func assertProxiedDepExists(t *testing.T, db *sql.DB, issueID, dependsOnID string) {
	t.Helper()
	var count int
	err := db.QueryRowContext(context.Background(),
		"SELECT COUNT(*) FROM dependencies WHERE issue_id = ? AND COALESCE(depends_on_issue_id, depends_on_wisp_id, depends_on_external) = ?",
		issueID, dependsOnID).Scan(&count)
	if err != nil {
		t.Fatalf("query dep %s -> %s: %v", issueID, dependsOnID, err)
	}
	if count != 1 {
		t.Fatalf("expected one dep row %s -> %s, got %d", issueID, dependsOnID, count)
	}
}

func assertProxiedDepExistsWithType(t *testing.T, db *sql.DB, issueID, dependsOnID, depType string) {
	t.Helper()
	var count int
	err := db.QueryRowContext(context.Background(),
		"SELECT COUNT(*) FROM dependencies WHERE issue_id = ? AND COALESCE(depends_on_issue_id, depends_on_wisp_id, depends_on_external) = ? AND type = ?",
		issueID, dependsOnID, depType).Scan(&count)
	if err != nil {
		t.Fatalf("query dep %s -> %s (%s): %v", issueID, dependsOnID, depType, err)
	}
	if count != 1 {
		t.Fatalf("expected one dep row %s -> %s of type %s, got %d", issueID, dependsOnID, depType, count)
	}
}

func getProxiedLabels(t *testing.T, db *sql.DB, issueID string) []string {
	t.Helper()
	rows, err := db.QueryContext(context.Background(),
		"SELECT label FROM labels WHERE issue_id = ?", issueID)
	if err != nil {
		t.Fatalf("query labels for %s: %v", issueID, err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var l string
		if err := rows.Scan(&l); err != nil {
			t.Fatalf("scan label: %v", err)
		}
		out = append(out, l)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("label rows iter: %v", err)
	}
	return out
}
