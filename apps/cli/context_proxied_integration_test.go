//go:build cgo

package main

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/borg0ai/dx-issuegraph/internal/configfile"
)

func TestProxiedServerContext(t *testing.T) {
	requireSharedProxiedServer(t)
	t.Parallel()
	issuegraph := buildEmbeddedBD(t)
	p := newSharedProxiedProject(t, issuegraph, "ctx")

	t.Run("default_text", func(t *testing.T) {
		t.Parallel()
		stdout, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, "context")
		if err != nil {
			t.Fatalf("issuegraph context failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
		}
		if !strings.Contains(stdout, configfile.DoltModeProxiedServer) {
			t.Errorf("expected mode %q in context output:\n%s", configfile.DoltModeProxiedServer, stdout)
		}
		if !strings.Contains(stdout, ".issuegraph") {
			t.Errorf("expected .issuegraph in context output:\n%s", stdout)
		}
	})

	t.Run("json_fields", func(t *testing.T) {
		t.Parallel()
		info := proxiedContextJSON(t, issuegraph, p)

		if info.DoltMode != configfile.DoltModeProxiedServer {
			t.Errorf("dolt_mode = %q, want %q", info.DoltMode, configfile.DoltModeProxiedServer)
		}
		if info.ProxiedDir == "" {
			t.Error("proxied_dir should be populated in proxied-server mode")
		}
		if info.Database != p.database {
			t.Errorf("database = %q, want %q", info.Database, p.database)
		}
		if filepath.Base(info.IssueGraphDir) != ".issuegraph" {
			t.Errorf("issuegraph_dir = %q, want a path ending in .issuegraph", info.IssueGraphDir)
		}
		if info.Backend != configfile.BackendDolt {
			t.Errorf("backend = %q, want %q", info.Backend, configfile.BackendDolt)
		}
		if info.BdVersion == "" {
			t.Error("issuegraph_version should be populated")
		}
		if info.ServerHost != "" || info.ServerPort != 0 {
			t.Errorf("server host/port must be empty in proxied-server mode: host=%q port=%d", info.ServerHost, info.ServerPort)
		}
	})

	t.Run("concurrent", func(t *testing.T) {
		t.Parallel()
		const numWorkers = 8
		errs := make([]error, numWorkers)
		modes := make([]string, numWorkers)

		var wg sync.WaitGroup
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				stdout, _, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, "context", "--json")
				if err != nil {
					errs[idx] = err
					return
				}
				var info ContextInfo
				if jerr := json.Unmarshal([]byte(stdout), &info); jerr != nil {
					errs[idx] = jerr
					return
				}
				modes[idx] = info.DoltMode
			}(i)
		}
		wg.Wait()

		for i := 0; i < numWorkers; i++ {
			if errs[i] != nil {
				t.Errorf("worker %d: issuegraph context failed: %v", i, errs[i])
				continue
			}
			if modes[i] != configfile.DoltModeProxiedServer {
				t.Errorf("worker %d: dolt_mode = %q, want %q", i, modes[i], configfile.DoltModeProxiedServer)
			}
		}
	})
}

func proxiedContextJSON(t *testing.T, issuegraph string, p proxiedProject) ContextInfo {
	t.Helper()
	stdout, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, "context", "--json")
	if err != nil {
		t.Fatalf("issuegraph context --json failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	var info ContextInfo
	if jerr := json.Unmarshal([]byte(stdout), &info); jerr != nil {
		t.Fatalf("decoding context JSON: %v\noutput:\n%s", jerr, stdout)
	}
	return info
}
