//go:build cgo && unix

package main

// Front-door lifecycle coverage for `issuegraph dolt status` and `issuegraph dolt start` on a
// MANAGED-LOCAL proxied workspace, plus the direct/embedded control that keeps
// the proxied guard from leaking into topologies issuegraph really does manage.
//
// These live in the managed-local lane (TestManagedLocalProxied* runs under
// proxied-local-smoke.yml) because they are the only lane with a real proxy
// supervising a real dolt child: the defects being covered are about which
// process record issuegraph reads and which process issuegraph spawns, and neither is
// observable against an external testcontainer.

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/borg0ai/dx-issuegraph/internal/doltserver"
	"github.com/borg0ai/dx-issuegraph/internal/storage/dbproxy/proxy"
	"github.com/borg0ai/dx-issuegraph/internal/storage/dbproxy/server"
)

// classicServerArtifacts are the files issuegraph writes when IT owns a dolt
// sql-server. On a proxied workspace their appearance is the signature of the
// second-writer hazard: issuegraph has spawned or adopted a server over the root the
// proxy serves, and now holds lifecycle state for a process it does not own.
var classicServerArtifacts = []string{
	doltserver.PIDFileName,
	doltserver.PortFileName,
	"dolt-server-config.yaml",
}

// stopStrayClassicServer tears down a classic sql-server if one was recorded
// for issuegraphDir. It exists so that a RED run of these tests — where `issuegraph dolt
// start` still spawns its own server over the proxied root — cannot leak that
// process past the test. Teardown goes through doltserver.Stop, which signals
// only the PID issuegraph itself recorded.
func stopStrayClassicServer(t *testing.T, issuegraphDir string) {
	t.Helper()
	serverDir := doltserver.ResolveServerDir(issuegraphDir)
	if _, err := os.Stat(filepath.Join(serverDir, doltserver.PIDFileName)); err != nil {
		return
	}
	t.Logf("stray classic dolt-server.pid found under %s; stopping it", serverDir)
	if err := doltserver.Stop(serverDir); err != nil {
		t.Logf("doltserver.Stop(%s): %v", serverDir, err)
	}
}

func assertNoClassicServerArtifacts(t *testing.T, issuegraphDir, when string) {
	t.Helper()
	for _, name := range classicServerArtifacts {
		path := filepath.Join(issuegraphDir, name)
		if _, err := os.Stat(path); err == nil {
			body, _ := os.ReadFile(path) // #nosec G304 - test-owned temp path
			t.Errorf("%s: %s exists on a proxied workspace (contents %q); issuegraph took ownership of a server over the proxied root",
				when, name, strings.TrimSpace(string(body)))
		}
	}
}

// TestManagedLocalProxiedDoltStatusReportsLiveTopology pins the truthfulness
// half of the fix: with a proxy and its dolt child both live, `issuegraph dolt status`
// must report them. Before the fix it read the classic pidfile, which a
// proxied workspace never writes, and answered "not running" — a confident
// lie, and the reason an operator would reach for `issuegraph dolt start` next.
func TestManagedLocalProxiedDoltStatusReportsLiveTopology(t *testing.T) {
	requireManagedLocalProxiedEnv(t)
	issuegraph := buildEmbeddedBD(t)
	p := issuegraphManagedLocalInit(t, issuegraph, "mlstat", 5*time.Minute)
	// Exercise the symlinked temporary paths used on macOS on every Unix host.
	linkedWorkspace := filepath.Join(t.TempDir(), "linked-workspace")
	if err := os.Symlink(p.dir, linkedWorkspace); err != nil {
		t.Fatalf("symlink workspace directory: %v", err)
	}
	p.dir = linkedWorkspace
	p.issuegraphDir = filepath.Join(p.dir, ".issuegraph")
	p.proxyRoot = filepath.Join(p.issuegraphDir, "dolt")
	issuegraphProxiedCreate(t, issuegraph, p.dir, "status sentinel")

	proxyPid := readManagedProxyPidFile(t, p)
	if proxyPid == nil || !processAlive(proxyPid.Pid) {
		t.Fatalf("expected a live proxy after a create; pidfile=%+v", proxyPid)
	}
	backendPid := readManagedBackendPidFile(t, p)
	if backendPid == nil || !processAlive(backendPid.Pid) {
		t.Fatalf("expected a live dolt backend after a create; pidfile=%+v", backendPid)
	}

	stdout, stderr, err := issuegraphProxiedRunBuffersWithEnv(t, issuegraph, p.dir, []string{"ISSUEGRAPH_JSON=1"}, "--json", "dolt", "status")
	if err != nil {
		t.Fatalf("issuegraph dolt status failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	var got struct {
		Mode           string `json:"mode"`
		Running        bool   `json:"running"`
		ProxyPID       int    `json:"proxy_pid"`
		ProxyPort      int    `json:"proxy_port"`
		BackendRunning bool   `json:"backend_running"`
		BackendPID     int    `json:"backend_pid"`
		BackendPort    int    `json:"backend_port"`
		Root           string `json:"root"`
	}
	if jsonErr := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &got); jsonErr != nil {
		t.Fatalf("decode dolt status JSON: %v\nstdout:\n%s", jsonErr, stdout)
	}
	if got.Mode != "proxied-server" {
		t.Errorf("mode=%q, want %q (status must name the topology it is describing)", got.Mode, "proxied-server")
	}
	if !got.Running {
		t.Errorf("running=false while proxy pid %d is alive; status is reporting the wrong process record", proxyPid.Pid)
	}
	if got.ProxyPID != proxyPid.Pid {
		t.Errorf("proxy_pid=%d, want %d (from %s)", got.ProxyPID, proxyPid.Pid, proxy.PIDFileName)
	}
	if got.ProxyPort != proxyPid.Port {
		t.Errorf("proxy_port=%d, want %d (from %s)", got.ProxyPort, proxyPid.Port, proxy.PIDFileName)
	}
	if !got.BackendRunning {
		t.Errorf("backend_running=false while dolt pid %d is alive", backendPid.Pid)
	}
	if got.BackendPID != backendPid.Pid {
		t.Errorf("backend_pid=%d, want %d (from %s)", got.BackendPID, backendPid.Pid, server.PIDFileName)
	}
	if got.BackendPort != backendPid.Port {
		t.Errorf("backend_port=%d, want %d (from %s)", got.BackendPort, backendPid.Port, server.PIDFileName)
	}
	gotRoot, err := filepath.EvalSymlinks(got.Root)
	if err != nil {
		t.Fatalf("resolve reported root %q: %v", got.Root, err)
	}
	wantRoot, err := filepath.EvalSymlinks(p.proxyRoot)
	if err != nil {
		t.Fatalf("resolve expected root %q: %v", p.proxyRoot, err)
	}
	if gotRoot != wantRoot {
		t.Errorf("root=%q, want %q", got.Root, p.proxyRoot)
	}

	textOut, textErr, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, "dolt", "status")
	if err != nil {
		t.Fatalf("issuegraph dolt status (text) failed: %v\nstdout:\n%s\nstderr:\n%s", err, textOut, textErr)
	}
	if strings.Contains(textOut, "Dolt server: not running") {
		t.Errorf("text status still claims the server is not running:\n%s", textOut)
	}
	for _, want := range []string{"proxied-server", "Backend"} {
		if !strings.Contains(textOut, want) {
			t.Errorf("text status missing %q:\n%s", want, textOut)
		}
	}

	// Reading status must not disturb what it is reporting on.
	if after := readManagedProxyPidFile(t, p); after == nil || after.Pid != proxyPid.Pid {
		t.Errorf("proxy pidfile changed across a status read: before=%+v after=%+v", proxyPid, after)
	}
	assertNoClassicServerArtifacts(t, p.issuegraphDir, "after issuegraph dolt status")
}

// TestManagedLocalProxiedDoltStartRefusesOverLiveProxy covers the hazard in
// its live-topology shape: with the proxy up, `issuegraph dolt start` used to adopt
// the proxy's own dolt child into issuegraph's classic server bookkeeping, leaving two
// managers for one process.
func TestManagedLocalProxiedDoltStartRefusesOverLiveProxy(t *testing.T) {
	requireManagedLocalProxiedEnv(t)
	issuegraph := buildEmbeddedBD(t)
	p := issuegraphManagedLocalInit(t, issuegraph, "mlstart", 5*time.Minute)
	t.Cleanup(func() { stopStrayClassicServer(t, p.issuegraphDir) })
	issuegraphProxiedCreate(t, issuegraph, p.dir, "start sentinel")

	proxyBefore := readManagedProxyPidFile(t, p)
	backendBefore := readManagedBackendPidFile(t, p)
	if proxyBefore == nil || backendBefore == nil {
		t.Fatalf("expected a live proxy and backend; proxy=%+v backend=%+v", proxyBefore, backendBefore)
	}

	stdout, stderr, err := issuegraphProxiedRunBuffersWithEnv(t, issuegraph, p.dir, []string{"ISSUEGRAPH_JSON=1"}, "--json", "dolt", "start")
	if err == nil {
		t.Errorf("issuegraph dolt start succeeded on a live proxied workspace:\nstdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
	assertProxyStartRefusal(t, stdout, stderr)
	assertNoClassicServerArtifacts(t, p.issuegraphDir, "after issuegraph dolt start")

	proxyAfter := readManagedProxyPidFile(t, p)
	backendAfter := readManagedBackendPidFile(t, p)
	if proxyAfter == nil || proxyAfter.Pid != proxyBefore.Pid {
		t.Errorf("proxy record changed across a refused start: before=%+v after=%+v", proxyBefore, proxyAfter)
	}
	if backendAfter == nil || backendAfter.Pid != backendBefore.Pid {
		t.Errorf("backend record changed across a refused start: before=%+v after=%+v", backendBefore, backendAfter)
	}
}

// TestManagedLocalProxiedDoltStartRefusesWithProxyDown covers the same hazard
// in its damaging shape. A proxied workspace whose proxy has idled out still
// has a proxied ROOT, and the classic start path resolves that root's port
// from its config.yaml: `issuegraph dolt start` used to launch a real, unmanaged dolt
// sql-server directly over the proxy's data directory. The next ordinary issuegraph
// command then failed, because the proxy came up and found a foreign server on
// its port. The workspace staying usable afterwards is the assertion that
// matters most here.
func TestManagedLocalProxiedDoltStartRefusesWithProxyDown(t *testing.T) {
	requireManagedLocalProxiedEnv(t)
	issuegraph := buildEmbeddedBD(t)
	p := issuegraphManagedLocalInit(t, issuegraph, "mldown", 5*time.Minute)
	t.Cleanup(func() { stopStrayClassicServer(t, p.issuegraphDir) })
	issuegraphProxiedCreate(t, issuegraph, p.dir, "down sentinel")

	if err := proxy.Shutdown(p.proxyRoot); err != nil {
		t.Fatalf("proxy.Shutdown(%s): %v", p.proxyRoot, err)
	}

	stdout, stderr, err := issuegraphProxiedRunBuffersWithEnv(t, issuegraph, p.dir, []string{"ISSUEGRAPH_JSON=1"}, "--json", "dolt", "start")
	if err == nil {
		t.Errorf("issuegraph dolt start succeeded against a quiesced proxied root:\nstdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
	assertProxyStartRefusal(t, stdout, stderr)
	assertNoClassicServerArtifacts(t, p.issuegraphDir, "after issuegraph dolt start with the proxy down")

	// The workspace must still work. Pre-fix this create failed with
	// "invalid connection": the proxy relaunched and adopted the foreign
	// server `issuegraph dolt start` had left on its port.
	out, createErr := issuegraphProxiedRun(t, issuegraph, p.dir, "create", "--json", "post-refusal write")
	if createErr != nil {
		t.Fatalf("workspace unusable after a refused dolt start: %v\n%s", createErr, out)
	}
}

func assertProxyStartRefusal(t *testing.T, stdout, stderr string) {
	t.Helper()
	var got struct {
		Code    string `json:"code"`
		Error   string `json:"error"`
		Mutates bool   `json:"mutates"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &got); err != nil {
		t.Errorf("dolt start refusal is not typed JSON (%v)\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
		return
	}
	if got.Code != proxyDoltStartConflictCode {
		t.Errorf("refusal code=%q, want %q", got.Code, proxyDoltStartConflictCode)
	}
	if got.Mutates {
		t.Errorf("refusal reported mutates=true; a refused start changes nothing")
	}
	if !strings.Contains(got.Error, "proxied-server mode") {
		t.Errorf("refusal message does not name the mode: %q", got.Error)
	}
}

// TestManagedLocalProxiedDoltLifecycleLeavesOtherTopologiesAlone is the
// non-regression control for the two tests above: the proxied guard keys on
// the workspace's own mode, so an embedded workspace and a bd-managed direct
// local server must behave exactly as they did before. It runs in this lane
// rather than a direct-server one because this is the lane that installs the
// dolt CLI the direct case needs.
func TestManagedLocalProxiedDoltLifecycleLeavesOtherTopologiesAlone(t *testing.T) {
	requireManagedLocalProxiedEnv(t)
	issuegraph := buildEmbeddedBD(t)

	t.Run("embedded", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph)

		statusOut, statusErr, err := runDirectBD(t, issuegraph, dir, false, "dolt", "status")
		if err != nil {
			t.Fatalf("issuegraph dolt status failed on embedded: %v\nstdout:\n%s\nstderr:\n%s", err, statusOut, statusErr)
		}
		if !strings.Contains(statusOut, "Dolt engine: embedded (in-process, no server)") {
			t.Errorf("embedded status changed:\n%s", statusOut)
		}

		startOut, startErr, err := runDirectBD(t, issuegraph, dir, false, "dolt", "start")
		if err == nil {
			t.Fatalf("issuegraph dolt start unexpectedly succeeded on embedded:\n%s", startOut)
		}
		if !strings.Contains(startErr, "'issuegraph dolt start' is not supported in embedded mode (no Dolt server)") {
			t.Errorf("embedded start refusal changed:\nstdout:\n%s\nstderr:\n%s", startOut, startErr)
		}
	})

	t.Run("direct-local-server", func(t *testing.T) {
		dir := t.TempDir()
		initGitRepoAt(t, dir)
		if out, errOut, err := runDirectBD(t, issuegraph, dir, true, "init", "--quiet", "--server", "--prefix", "dls"); err != nil {
			t.Fatalf("issuegraph init --server failed: %v\nstdout:\n%s\nstderr:\n%s", err, out, errOut)
		}
		issuegraphDir := filepath.Join(dir, ".issuegraph")
		t.Cleanup(func() { stopStrayClassicServer(t, issuegraphDir) })

		startOut, startErr, err := runDirectBD(t, issuegraph, dir, true, "dolt", "start")
		if err != nil {
			t.Fatalf("issuegraph dolt start failed on a direct local server: %v\nstdout:\n%s\nstderr:\n%s", err, startOut, startErr)
		}
		if !strings.Contains(startOut, "Dolt server started (PID ") {
			t.Fatalf("direct start output changed:\n%s", startOut)
		}

		statusOut, statusErr, err := runDirectBD(t, issuegraph, dir, true, "dolt", "status")
		if err != nil {
			t.Fatalf("issuegraph dolt status failed on a direct local server: %v\nstdout:\n%s\nstderr:\n%s", err, statusOut, statusErr)
		}
		if !strings.Contains(statusOut, "Dolt server: running") {
			t.Errorf("direct status did not report the server issuegraph just started:\n%s", statusOut)
		}

		stopOut, stopErr, err := runDirectBD(t, issuegraph, dir, true, "dolt", "stop")
		if err != nil {
			t.Fatalf("issuegraph dolt stop failed on a direct local server: %v\nstdout:\n%s\nstderr:\n%s", err, stopOut, stopErr)
		}
		if !strings.Contains(stopOut, "Dolt server stopped.") {
			t.Errorf("direct stop output changed:\n%s", stopOut)
		}
	})
}

// runDirectBD runs issuegraph against a NON-proxied workspace. autoStart controls
// ISSUEGRAPH_DOLT_AUTO_START: the direct-server case needs it on, because
// issuegraph dolt status routes a local server with auto-start disabled to the
// external SQL-probe path instead of the pidfile path under test.
func runDirectBD(t *testing.T, issuegraph, dir string, autoStart bool, args ...string) (string, string, error) {
	t.Helper()
	env := issuegraphEnv(dir)
	if autoStart {
		env = append(envWithout(env, "ISSUEGRAPH_DOLT_AUTO_START"), "ISSUEGRAPH_DOLT_AUTO_START=1")
	}
	cmd := exec.Command(issuegraph, args...)
	cmd.Dir = dir
	cmd.Env = env
	stdout, stderr, err := runCommandBuffers(t, cmd)
	return stdout.String(), stderr.String(), err
}
