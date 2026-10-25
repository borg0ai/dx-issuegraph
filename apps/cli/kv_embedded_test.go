//go:build cgo

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
)

// issuegraphKV runs "issuegraph kv" with the given args and returns stdout.
func issuegraphKV(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"kv"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	stdout, stderr, err := runCommandBuffers(t, cmd)
	if err != nil {
		t.Fatalf("issuegraph kv %s failed: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String()
}

// issuegraphKVFail runs "issuegraph kv" expecting failure.
func issuegraphKVFail(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"kv"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected issuegraph kv %s to fail, but succeeded:\n%s", strings.Join(args, " "), out)
	}
	return string(out)
}

// issuegraphKVListJSON runs "issuegraph kv list --json" and returns parsed map.
func issuegraphKVListJSON(t *testing.T, issuegraph, dir string) map[string]string {
	t.Helper()
	cmd := exec.Command(issuegraph, "kv", "list", "--json")
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	stdout, stderr, err := runCommandBuffers(t, cmd)
	if err != nil {
		t.Fatalf("issuegraph kv list --json failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}
	s := strings.TrimSpace(stdout.String())
	start := strings.Index(s, "{")
	if start < 0 {
		return map[string]string{}
	}
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(s[start:]), &raw); err != nil {
		t.Fatalf("parse kv list JSON: %v\n%s", err, s)
	}
	m := make(map[string]string, len(raw))
	for k, v := range raw {
		if k == "schema_version" {
			continue
		}
		if sv, ok := v.(string); ok {
			m[k] = sv
		}
	}
	return m
}

func TestEmbeddedKV(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "tk")

	// ===== Set and Get =====

	t.Run("kv_set_and_get", func(t *testing.T) {
		issuegraphKV(t, issuegraph, dir, "set", "mykey", "myvalue")
		out := issuegraphKV(t, issuegraph, dir, "get", "mykey")
		if !strings.Contains(out, "myvalue") {
			t.Errorf("expected 'myvalue' in get output: %s", out)
		}
	})

	t.Run("kv_set_overwrite", func(t *testing.T) {
		issuegraphKV(t, issuegraph, dir, "set", "overkey", "first")
		issuegraphKV(t, issuegraph, dir, "set", "overkey", "second")
		out := issuegraphKV(t, issuegraph, dir, "get", "overkey")
		if !strings.Contains(out, "second") {
			t.Errorf("expected 'second' after overwrite: %s", out)
		}
	})

	t.Run("kv_set_special_chars", func(t *testing.T) {
		issuegraphKV(t, issuegraph, dir, "set", "special", "hello world with spaces")
		out := issuegraphKV(t, issuegraph, dir, "get", "special")
		if !strings.Contains(out, "hello world with spaces") {
			t.Errorf("expected value with spaces: %s", out)
		}
	})

	// ===== List =====

	t.Run("kv_list", func(t *testing.T) {
		out := issuegraphKV(t, issuegraph, dir, "list")
		if !strings.Contains(out, "mykey") {
			t.Errorf("expected 'mykey' in list output: %s", out)
		}
	})

	t.Run("kv_list_json", func(t *testing.T) {
		m := issuegraphKVListJSON(t, issuegraph, dir)
		if v, ok := m["mykey"]; !ok || v != "myvalue" {
			t.Errorf("expected mykey=myvalue in JSON, got %q (exists=%v)", v, ok)
		}
	})

	// ===== Clear =====

	t.Run("kv_clear", func(t *testing.T) {
		issuegraphKV(t, issuegraph, dir, "set", "clearme", "temporary")
		out := issuegraphKV(t, issuegraph, dir, "get", "clearme")
		if !strings.Contains(out, "temporary") {
			t.Fatalf("expected 'temporary' before clear: %s", out)
		}
		issuegraphKV(t, issuegraph, dir, "clear", "clearme")
		// Verify it's gone from list
		m := issuegraphKVListJSON(t, issuegraph, dir)
		if _, ok := m["clearme"]; ok {
			t.Error("expected clearme to be absent from kv list after clear")
		}
	})

	// ===== Error Cases =====

	t.Run("kv_get_missing_key", func(t *testing.T) {
		// get on nonexistent key — check behavior
		cmd := exec.Command(issuegraph, "kv", "get", "nonexistent_key_xyz")
		cmd.Dir = dir
		cmd.Env = issuegraphEnv(dir)
		out, err := cmd.CombinedOutput()
		// May succeed with empty or fail — either way no crash
		_ = err
		_ = out
	})

	t.Run("kv_set_no_args", func(t *testing.T) {
		issuegraphKVFail(t, issuegraph, dir, "set")
	})

	t.Run("kv_get_no_args", func(t *testing.T) {
		issuegraphKVFail(t, issuegraph, dir, "get")
	})

	t.Run("kv_clear_no_args", func(t *testing.T) {
		issuegraphKVFail(t, issuegraph, dir, "clear")
	})
}

// TestEmbeddedKVConcurrent exercises kv operations concurrently.
func TestEmbeddedKVConcurrent(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "kx")

	const numWorkers = 8

	type workerResult struct {
		worker int
		err    error
	}

	results := make([]workerResult, numWorkers)
	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for w := 0; w < numWorkers; w++ {
		go func(worker int) {
			defer wg.Done()
			r := workerResult{worker: worker}

			// Each worker sets its own keys
			for i := 0; i < 5; i++ {
				key := fmt.Sprintf("w%d-k%d", worker, i)
				value := fmt.Sprintf("v%d-%d", worker, i)

				cmd := exec.Command(issuegraph, "kv", "set", key, value)
				cmd.Dir = dir
				cmd.Env = issuegraphEnv(dir)
				out, err := cmd.CombinedOutput()
				if err != nil {
					r.err = fmt.Errorf("set %s: %v\n%s", key, err, out)
					results[worker] = r
					return
				}
			}

			// Read back
			for i := 0; i < 5; i++ {
				key := fmt.Sprintf("w%d-k%d", worker, i)
				expected := fmt.Sprintf("v%d-%d", worker, i)

				cmd := exec.Command(issuegraph, "kv", "get", key)
				cmd.Dir = dir
				cmd.Env = issuegraphEnv(dir)
				out, err := cmd.CombinedOutput()
				if err != nil {
					r.err = fmt.Errorf("get %s: %v\n%s", key, err, out)
					results[worker] = r
					return
				}
				if !strings.Contains(string(out), expected) {
					r.err = fmt.Errorf("key %s expected %q, got %q", key, expected, string(out))
					results[worker] = r
					return
				}
			}

			// Clear one key
			clearKey := fmt.Sprintf("w%d-k0", worker)
			cmd := exec.Command(issuegraph, "kv", "clear", clearKey)
			cmd.Dir = dir
			cmd.Env = issuegraphEnv(dir)
			out, err := cmd.CombinedOutput()
			if err != nil {
				r.err = fmt.Errorf("clear %s: %v\n%s", clearKey, err, out)
				results[worker] = r
				return
			}

			results[worker] = r
		}(w)
	}
	wg.Wait()

	for _, r := range results {
		if r.err != nil && !strings.Contains(r.err.Error(), "one writer at a time") {
			t.Errorf("worker %d failed: %v", r.worker, r.err)
		}
	}

	// Verify remaining keys only for workers that succeeded (err==nil).
	// With exclusive flock, some workers may fail with "one writer at a time".
	m := issuegraphKVListJSON(t, issuegraph, dir)
	var successCount int
	for _, r := range results {
		if r.err != nil {
			continue
		}
		successCount++
		w := r.worker
		clearedKey := fmt.Sprintf("w%d-k0", w)
		if _, ok := m[clearedKey]; ok {
			t.Errorf("expected %s to be cleared", clearedKey)
		}
		for i := 1; i < 5; i++ {
			key := fmt.Sprintf("w%d-k%d", w, i)
			expected := fmt.Sprintf("v%d-%d", w, i)
			if v, ok := m[key]; !ok || v != expected {
				t.Errorf("key %s expected %q, got %q (exists=%v)", key, expected, v, ok)
			}
		}
	}
	if successCount == 0 {
		t.Fatal("expected at least 1 worker to succeed")
	}
}
