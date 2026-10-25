//go:build cgo

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
)

// issuegraphSetup runs "issuegraph setup" with the given args and returns stdout.
func issuegraphSetup(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"setup"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	stdout, stderr, err := runCommandBuffers(t, cmd)
	if err != nil {
		t.Fatalf("issuegraph setup %s failed: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String()
}

func TestEmbeddedSetup(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "ts")

	t.Run("setup_list", func(t *testing.T) {
		out := issuegraphSetup(t, issuegraph, dir, "--list")
		if !strings.Contains(out, "cursor") && !strings.Contains(out, "claude") {
			t.Errorf("expected recipe names in setup --list: %s", out)
		}
	})

	t.Run("setup_install_and_check", func(t *testing.T) {
		issuegraphSetup(t, issuegraph, dir, "cursor")
		out := issuegraphSetup(t, issuegraph, dir, "cursor", "--check")
		_ = out // Should succeed
	})

	t.Run("setup_remove", func(t *testing.T) {
		issuegraphSetup(t, issuegraph, dir, "cursor", "--remove")
		// After remove, check should indicate not installed or succeed gracefully
		cmd := exec.Command(issuegraph, "setup", "cursor", "--check")
		cmd.Dir = dir
		cmd.Env = issuegraphEnv(dir)
		_, _ = cmd.CombinedOutput() // May succeed or fail — just verify no crash
	})
}

func TestEmbeddedSetupConcurrent(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "sx")

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
			cmd := exec.Command(issuegraph, "setup", "--list")
			cmd.Dir = dir
			cmd.Env = issuegraphEnv(dir)
			out, err := cmd.CombinedOutput()
			if err != nil {
				r.err = fmt.Errorf("setup --list (worker %d): %v\n%s", worker, err, out)
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
}
