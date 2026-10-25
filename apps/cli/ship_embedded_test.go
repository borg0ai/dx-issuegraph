//go:build cgo

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// issuegraphShip runs "issuegraph ship" with the given args and returns stdout.
func issuegraphShip(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"ship"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	stdout, stderr, err := runCommandBuffers(t, cmd)
	if err != nil {
		t.Fatalf("issuegraph ship %s failed: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String()
}

// issuegraphShipFail runs "issuegraph ship" expecting failure.
func issuegraphShipFail(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"ship"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected issuegraph ship %s to fail, but succeeded:\n%s", strings.Join(args, " "), out)
	}
	return string(out)
}

func TestEmbeddedShip(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "sh")

	// ===== Basic Ship =====

	t.Run("ship_basic", func(t *testing.T) {
		issue := issuegraphCreate(t, issuegraph, dir, "Ship basic test", "--type", "task")
		// Add export label
		cmd := exec.Command(issuegraph, "label", "add", issue.ID, "export:auth")
		cmd.Dir = dir
		cmd.Env = issuegraphEnv(dir)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("label add failed: %v\n%s", err, out)
		}
		// Close the issue
		cmd = exec.Command(issuegraph, "close", issue.ID)
		cmd.Dir = dir
		cmd.Env = issuegraphEnv(dir)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("close failed: %v\n%s", err, out)
		}

		out := issuegraphShip(t, issuegraph, dir, "auth")
		if !strings.Contains(out, "Shipped") {
			t.Errorf("expected 'Shipped' in output: %s", out)
		}
		if !strings.Contains(out, "provides:auth") {
			t.Errorf("expected 'provides:auth' in output: %s", out)
		}
	})

	// ===== --force (ship unclosed issue) =====

	t.Run("ship_force", func(t *testing.T) {
		issue := issuegraphCreate(t, issuegraph, dir, "Ship force test", "--type", "task")
		cmd := exec.Command(issuegraph, "label", "add", issue.ID, "export:force-cap")
		cmd.Dir = dir
		cmd.Env = issuegraphEnv(dir)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("label add failed: %v\n%s", err, out)
		}

		// Ship without closing — should fail without --force
		issuegraphShipFail(t, issuegraph, dir, "force-cap")

		// Ship with --force — should succeed
		out := issuegraphShip(t, issuegraph, dir, "force-cap", "--force")
		if !strings.Contains(out, "Shipped") {
			t.Errorf("expected 'Shipped' with --force: %s", out)
		}
	})

	// ===== --dry-run =====

	t.Run("ship_dry_run", func(t *testing.T) {
		issue := issuegraphCreate(t, issuegraph, dir, "Ship dry-run test", "--type", "task")
		cmd := exec.Command(issuegraph, "label", "add", issue.ID, "export:dry-cap")
		cmd.Dir = dir
		cmd.Env = issuegraphEnv(dir)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("label add failed: %v\n%s", err, out)
		}
		cmd = exec.Command(issuegraph, "close", issue.ID)
		cmd.Dir = dir
		cmd.Env = issuegraphEnv(dir)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("close failed: %v\n%s", err, out)
		}

		out := issuegraphShip(t, issuegraph, dir, "dry-cap", "--dry-run")
		if !strings.Contains(out, "dry run") && !strings.Contains(out, "Would ship") {
			t.Errorf("expected dry-run message: %s", out)
		}

		// Verify label was NOT added
		cmd = exec.Command(issuegraph, "label", "list", issue.ID, "--json")
		cmd.Dir = dir
		cmd.Env = issuegraphEnv(dir)
		labelOut, labelErr := cmd.CombinedOutput()
		if labelErr != nil {
			t.Fatalf("label list failed: %v\n%s", labelErr, labelOut)
		}
		if strings.Contains(string(labelOut), "provides:dry-cap") {
			t.Errorf("dry-run should not have added provides label: %s", labelOut)
		}
	})

	// ===== Already Shipped =====

	t.Run("ship_already_shipped", func(t *testing.T) {
		issue := issuegraphCreate(t, issuegraph, dir, "Ship already test", "--type", "task")
		cmd := exec.Command(issuegraph, "label", "add", issue.ID, "export:already-cap")
		cmd.Dir = dir
		cmd.Env = issuegraphEnv(dir)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("label add failed: %v\n%s", err, out)
		}
		cmd = exec.Command(issuegraph, "close", issue.ID)
		cmd.Dir = dir
		cmd.Env = issuegraphEnv(dir)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("close failed: %v\n%s", err, out)
		}

		// Ship once
		issuegraphShip(t, issuegraph, dir, "already-cap")
		// Ship again — should report already shipped
		out := issuegraphShip(t, issuegraph, dir, "already-cap")
		if !strings.Contains(out, "already shipped") && !strings.Contains(out, "already_shipped") {
			t.Errorf("expected 'already shipped' message: %s", out)
		}
	})

	// ===== No Export Label =====

	t.Run("ship_no_export_label", func(t *testing.T) {
		issuegraphShipFail(t, issuegraph, dir, "nonexistent-capability-xyz")
	})
}

// TestEmbeddedShipConcurrent exercises ship --dry-run concurrently.
func TestEmbeddedShipConcurrent(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "sx")

	// Pre-create issues with export labels
	for i := 0; i < 8; i++ {
		issue := issuegraphCreate(t, issuegraph, dir, fmt.Sprintf("ship-concurrent-%d", i), "--type", "task")
		cap := fmt.Sprintf("cap-%d", i)
		cmd := exec.Command(issuegraph, "label", "add", issue.ID, "export:"+cap)
		cmd.Dir = dir
		cmd.Env = issuegraphEnv(dir)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("label add failed: %v\n%s", err, out)
		}
		cmd = exec.Command(issuegraph, "close", issue.ID)
		cmd.Dir = dir
		cmd.Env = issuegraphEnv(dir)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("close failed: %v\n%s", err, out)
		}
	}

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
			cap := fmt.Sprintf("cap-%d", worker)

			cmd := exec.Command(issuegraph, "ship", cap, "--dry-run")
			cmd.Dir = dir
			cmd.Env = issuegraphEnv(dir)
			out, err := cmd.CombinedOutput()
			if err != nil {
				r.err = fmt.Errorf("ship --dry-run %s (worker %d): %v\n%s", cap, worker, err, out)
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

func TestEmbeddedExternalCapabilityReadyAfterShip(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	localDir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "xl")
	remoteDir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "xr")

	config := fmt.Sprintf("external_projects:\n  remote: %q\n", remoteDir)
	if err := os.WriteFile(filepath.Join(localDir, ".issuegraph", "config.local.yaml"), []byte(config), 0o600); err != nil {
		t.Fatalf("write external project config: %v", err)
	}

	dependent := issuegraphCreate(t, issuegraph, localDir, "Wait for payments capability", "--type", "task")
	provider := issuegraphCreate(t, issuegraph, remoteDir, "Export payments capability", "--type", "task")
	run := func(dir string, args ...string) string {
		t.Helper()
		cmd := exec.Command(issuegraph, args...)
		cmd.Dir = dir
		cmd.Env = issuegraphEnv(dir)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("issuegraph %s failed: %v: %s", strings.Join(args, " "), err, out)
		}
		return string(out)
	}

	run(remoteDir, "label", "add", provider.ID, "export:payments")
	run(remoteDir, "close", provider.ID)
	run(localDir, "dep", "add", dependent.ID, "external:remote:payments")
	if out := run(localDir, "ready", "--json", "--limit", "0"); strings.Contains(out, dependent.ID) {
		t.Fatalf("dependent became ready before capability publication: %s", out)
	}
	issuegraphShip(t, issuegraph, remoteDir, "payments")
	if out := run(localDir, "ready", "--json", "--limit", "0"); !strings.Contains(out, dependent.ID) {
		t.Fatalf("dependent did not become ready after issuegraph ship: %s", out)
	}
}
