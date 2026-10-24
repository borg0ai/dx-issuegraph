//go:build cgo && unix

package main

// Real front-door outage/reconnect acceptance matrix. Each case seeds a
// sentinel through issuegraph, takes only the upstream out of service, and verifies a
// bounded failure without mutation before restoring the same endpoint. Proxy
// ownership is exercised separately by killing the proxy and letting issuegraph adopt
// a replacement on the next invocation.

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/borg0ai/dx-issuegraph/internal/configfile"
	"github.com/borg0ai/dx-issuegraph/internal/storage/dbproxy/proxy"
	"github.com/borg0ai/dx-issuegraph/internal/testutil"
	"github.com/borg0ai/dx-issuegraph/internal/types"
	"github.com/stretchr/testify/require"
)

func TestProxiedOutageReconnectAcceptanceMatrix(t *testing.T) {
	requireProxiedServerEnv(t)
	issuegraph := buildEmbeddedBD(t)
	for _, topology := range []struct {
		name   string
		socket bool
	}{
		{name: "external-tcp"},
		{name: "external-unix-socket", socket: true},
	} {
		t.Run(topology.name, func(t *testing.T) {
			upstream := testutil.StartIsolatedDoltContainerHandle(t)
			if _, err := exec.LookPath("socat"); err != nil {
				t.Skip("socat is required")
			}
			gatePort, err := proxy.PickFreePort()
			require.NoError(t, err)
			var endpoint string
			if topology.socket {
				endpoint = filepath.Join(t.TempDir(), "dolt.sock")
			} else {
				endpoint = strconv.Itoa(gatePort)
			}
			bridge := startOutageBridge(t, endpoint, upstream.Port, topology.socket)

			var p proxiedProject
			if topology.socket {
				p = issuegraphProxiedInit(t, issuegraph, "outage_socket", "--proxied-server-external-socket-path", endpoint)
			} else {
				p = issuegraphProxiedInit(t, issuegraph, "outage_tcp", "--proxied-server-external-host", "127.0.0.1", "--proxied-server-external-port", endpoint)
			}
			sentinel := issuegraphProxiedCreate(t, issuegraph, p.dir, "outage sentinel")
			before := issuegraphProxiedShow(t, issuegraph, p.dir, sentinel.ID)
			if _, err := os.Stat(filepath.Join(p.proxyRoot, "proxy-child.pid")); err == nil {
				t.Fatal("external topology spawned a local Dolt sidecar")
			}

			// Stop only the upstream. The proxy remains alive and must surface a
			// bounded transport failure; no local fallback is permitted.
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			require.NoError(t, upstream.Stop(ctx))
			cancel()
			_, stderr, err, timedOut := runProxiedDeadline(t, issuegraph, p.dir, 3*time.Second, "show", sentinel.ID, "--json")
			if timedOut {
				t.Fatal("upstream outage command exceeded 3s bound")
			}
			if err == nil {
				t.Fatal("show unexpectedly succeeded while upstream was stopped")
			}
			if !actionableTransportError(stderr) {
				t.Fatalf("outage error is not actionable transport refusal: %q", stderr)
			}
			// Restore and verify the exact sentinel survives the outage.
			ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
			require.NoError(t, upstream.Start(ctx))
			newPort, err := upstream.CurrentPort(ctx)
			cancel()
			require.NoError(t, err)
			require.NoError(t, bridge.Process.Kill())
			bridge = startOutageBridge(t, endpoint, newPort, topology.socket)
			var after *types.Issue
			require.Eventually(t, func() bool {
				stdout, _, runErr := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, "show", sentinel.ID, "--json")
				if runErr != nil {
					return false
				}
				after = parseIssueJSON(t, []byte(stdout))
				return true
			}, 10*time.Second, 100*time.Millisecond, "issuegraph reconnect after upstream restart")
			if after.ID != before.ID || after.Title != before.Title {
				t.Fatalf("sentinel changed after reconnect: before=%q/%q after=%q/%q", before.ID, before.Title, after.ID, after.Title)
			}
			if _, err := os.Stat(filepath.Join(p.proxyRoot, "proxy-child.pid")); err == nil {
				t.Fatal("external topology created a local Dolt sidecar during reconnect")
			}

			// Killing the proxy itself must cause a fresh proxy to be adopted on
			// the next command, still reaching the same upstream data.
			if err := proxy.Shutdown(p.proxyRoot); err != nil {
				t.Fatalf("shutdown proxy: %v", err)
			}
			afterProxy := issuegraphProxiedShow(t, issuegraph, p.dir, sentinel.ID)
			if afterProxy.ID != sentinel.ID || afterProxy.Title != sentinel.Title {
				t.Fatalf("sentinel lost after proxy restart: %+v", afterProxy)
			}

			// Endpoint identity changes must refuse stale-proxy adoption.
			info, err := configfile.LoadProxiedServerClientInfo(p.issuegraphDir)
			if err != nil || info == nil || info.External == nil {
				t.Fatalf("load endpoint identity: info=%+v err=%v", info, err)
			}
			original := *info.External
			if topology.socket {
				info.External.Socket = filepath.Join(t.TempDir(), "stale.sock")
			} else {
				info.External.Port++
			}
			if err := configfile.SaveProxiedServerClientInfo(p.issuegraphDir, info); err != nil {
				t.Fatalf("rewrite endpoint identity: %v", err)
			}
			stdout, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, "show", sentinel.ID, "--json")
			if err == nil {
				t.Fatalf("stale endpoint unexpectedly succeeded:\nstdout=%s\nstderr=%s", stdout, stderr)
			}
			if !strings.Contains(strings.ToLower(stdout+stderr), "endpoint") {
				t.Fatalf("stale endpoint refusal lacks identity context: %s%s", stdout, stderr)
			}
			info.External = &original
			require.NoError(t, configfile.SaveProxiedServerClientInfo(p.issuegraphDir, info))
		})
	}
}

func TestManagedLocalProxiedOutageReconnectContract(t *testing.T) {
	requireManagedLocalProxiedEnv(t)
	issuegraph := buildEmbeddedBD(t)
	p := issuegraphManagedLocalInit(t, issuegraph, "outage_managed", 5*time.Minute)
	sentinel := issuegraphProxiedCreate(t, issuegraph, p.dir, "managed outage sentinel")
	proxyBefore := readManagedProxyPidFile(t, p)
	if proxyBefore == nil || !processAlive(proxyBefore.Pid) {
		t.Fatalf("managed-local proxy is not running: %+v", proxyBefore)
	}
	backend := readManagedBackendPidFile(t, p)
	if backend == nil || !processAlive(backend.Pid) {
		t.Fatal("managed-local backend is not running")
	}
	proc, err := os.FindProcess(backend.Pid)
	require.NoError(t, err)
	require.NoError(t, proc.Kill())
	require.Eventually(t, func() bool { return !processAlive(proxyBefore.Pid) }, 3*time.Second, 50*time.Millisecond, "managed proxy did not retire after child death")
	// 1.3.0 contract: child death terminates the owned proxy; the next issuegraph
	// invocation starts a fresh proxy/backend rather than in-place restarting.
	got := issuegraphProxiedShow(t, issuegraph, p.dir, sentinel.ID)
	if got.ID != sentinel.ID || got.Title != "managed outage sentinel" {
		t.Fatalf("managed-local sentinel lost after command-level restart: %+v", got)
	}
	proxyAfter := readManagedProxyPidFile(t, p)
	backendAfter := readManagedBackendPidFile(t, p)
	if proxyBefore == nil || processAlive(proxyBefore.Pid) || processAlive(backend.Pid) {
		t.Fatalf("managed child death left old processes alive: proxy=%v backend=%v", processAlive(proxyBefore.Pid), processAlive(backend.Pid))
	}
	if backendAfter == nil || !processAlive(backendAfter.Pid) {
		t.Fatalf("managed restart backend not alive: %+v", backendAfter)
	}
	if proxyBefore == nil || proxyAfter == nil || proxyAfter.Pid == proxyBefore.Pid {
		t.Fatalf("managed child death did not force a fresh proxy: before=%+v after=%+v", proxyBefore, proxyAfter)
	}
}

func runProxiedDeadline(t *testing.T, issuegraph, dir string, timeout time.Duration, args ...string) (stdout, stderr string, err error, timedOut bool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, issuegraph, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Dir, cmd.Env = dir, issuegraphProxiedEnv(dir)
	var out, errOut strings.Builder
	cmd.Stdout, cmd.Stderr = &out, &errOut
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			if cmd.Process != nil {
				_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			}
		case <-done:
		}
	}()
	err = cmd.Run()
	close(done)
	return out.String(), errOut.String(), err, ctx.Err() != nil
}

func actionableTransportError(s string) bool {
	s = strings.ToLower(s)
	for _, token := range []string{"connection refused", "connect:", "endpoint", "dial", "upstream", "unexpected eof", "invalid connection"} {
		if strings.Contains(s, token) {
			return true
		}
	}
	return false
}

func waitForSocket(t *testing.T, path string) {
	t.Helper()
	require.Eventually(t, func() bool { _, err := os.Stat(path); return err == nil }, 3*time.Second, 20*time.Millisecond,
		"unix socket did not appear")
}

func startOutageBridge(t *testing.T, endpoint, upstreamPort string, socket bool) *exec.Cmd {
	t.Helper()
	var listen string
	if socket {
		// socat refuses to bind over a stale pathname after an outage.
		_ = os.Remove(endpoint)
		listen = "UNIX-LISTEN:" + endpoint + ",fork"
	} else {
		listen = "TCP-LISTEN:" + endpoint + ",bind=127.0.0.1,reuseaddr,fork"
	}
	cmd := exec.Command("socat", listen, "TCP:127.0.0.1:"+upstreamPort)
	require.NoError(t, cmd.Start())
	require.Eventually(t, func() bool { return processAlive(cmd.Process.Pid) }, time.Second, 20*time.Millisecond, "outage bridge process did not remain alive")
	if socket {
		waitForSocket(t, endpoint)
	}
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	})
	return cmd
}
