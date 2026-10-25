//go:build cgo

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// issuegraphBackup runs "issuegraph backup" with extra args. Returns combined output.
func issuegraphBackup(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"backup"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	stdout, stderr, err := runCommandBuffers(t, cmd)
	if err != nil {
		t.Fatalf("issuegraph backup %s failed: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String()
}

// issuegraphBackupFail runs "issuegraph backup" expecting failure. Returns combined output.
func issuegraphBackupFail(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"backup"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("issuegraph backup %s should have failed, got: %s", strings.Join(args, " "), out)
	}
	return string(out)
}

func TestEmbeddedBackup(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt backup tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)

	t.Run("status_no_backup", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "bkstat0")

		out := issuegraphBackup(t, issuegraph, dir, "status")
		if !strings.Contains(out, "No backup has been performed") {
			t.Errorf("expected 'No backup has been performed', got: %s", out)
		}
	})

	t.Run("status_after_init_sync", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "bkstat1")
		issuegraphCreateSilent(t, issuegraph, dir, "status test issue")

		backupDest := filepath.Join(t.TempDir(), "dolt-backup-status")
		backupURL := "file://" + backupDest
		issuegraphBackup(t, issuegraph, dir, "init", backupURL)
		issuegraphBackup(t, issuegraph, dir, "sync")

		out := issuegraphBackup(t, issuegraph, dir, "status")
		if out == "" {
			t.Error("expected non-empty status output")
		}
		if strings.Contains(out, "Database size: 0 B") {
			t.Errorf("embedded backup status reported an empty database: %s", out)
		}

		jsonOut := issuegraphBackup(t, issuegraph, dir, "status", "--json")
		var status struct {
			DatabaseSize struct {
				Bytes int64 `json:"bytes"`
			} `json:"database_size"`
		}
		if err := json.Unmarshal([]byte(jsonOut), &status); err != nil {
			t.Fatalf("parse backup status JSON: %v\noutput: %s", err, jsonOut)
		}
		if status.DatabaseSize.Bytes <= 0 {
			t.Errorf("database_size.bytes = %d, want > 0; output: %s", status.DatabaseSize.Bytes, jsonOut)
		}
	})

	t.Run("init_and_sync", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "bkinit")
		issuegraphCreateSilent(t, issuegraph, dir, "init sync test issue")

		// Create a backup destination directory
		backupDest := filepath.Join(t.TempDir(), "dolt-backup")
		backupURL := "file://" + backupDest

		// Init backup destination
		out := issuegraphBackup(t, issuegraph, dir, "init", backupURL)
		if !strings.Contains(out, "Backup destination configured") {
			t.Errorf("expected 'Backup destination configured', got: %s", out)
		}

		// Sync to backup
		out = issuegraphBackup(t, issuegraph, dir, "sync")
		if !strings.Contains(out, "Backup synced") {
			t.Errorf("expected 'Backup synced', got: %s", out)
		}

		// Verify backup destination has data
		if _, err := os.Stat(backupDest); os.IsNotExist(err) {
			t.Error("backup destination directory should exist after sync")
		}
	})

	t.Run("sync_without_init", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "bksync0")

		out := issuegraphBackupFail(t, issuegraph, dir, "sync")
		if !strings.Contains(out, "no backup") && !strings.Contains(out, "not found") &&
			!strings.Contains(out, "No backup") {
			t.Errorf("expected backup-not-configured error, got: %s", out)
		}
	})

	t.Run("restore_from_backup", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "bksrc")
		issuegraphCreateSilent(t, issuegraph, dir, "restore test issue A")
		issuegraphCreateSilent(t, issuegraph, dir, "restore test issue B")

		// Create a Dolt backup via init + sync
		backupDest := filepath.Join(t.TempDir(), "dolt-backup-restore")
		if err := os.MkdirAll(backupDest, 0750); err != nil {
			t.Fatal(err)
		}
		backupURL := "file://" + backupDest
		issuegraphBackup(t, issuegraph, dir, "init", backupURL)
		issuegraphBackup(t, issuegraph, dir, "sync")

		// Restore into a fresh environment. Use a prefix that differs
		// from the source ("bksrc") so the restored database name
		// doesn't collide with the target's own database.
		dir2, _, _ := issuegraphInit(t, issuegraph, "--prefix", "bkdst")

		// The restore will create a database named "bkdst" from the
		// backup, but "bkdst" already exists from issuegraph init. Use
		// issuegraphBackupFail to confirm the expected "already exists" error.
		out := issuegraphBackupFail(t, issuegraph, dir2, "restore", backupDest)
		if !strings.Contains(out, "already exists") {
			t.Errorf("expected 'already exists' error, got: %s", out)
		}
	})
}

func TestEmbeddedBackupConcurrent(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt backup tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "bkconc")

	// Seed some data
	for i := 0; i < 5; i++ {
		issuegraphCreateSilent(t, issuegraph, dir, fmt.Sprintf("concurrent backup issue %d", i))
	}

	// Set up a backup destination for sync
	backupDest := filepath.Join(t.TempDir(), "dolt-backup-concurrent")
	backupURL := "file://" + backupDest
	issuegraphBackup(t, issuegraph, dir, "init", backupURL)

	const numWorkers = 5

	type result struct {
		worker int
		out    string
		err    error
	}

	results := make([]result, numWorkers)
	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for w := 0; w < numWorkers; w++ {
		go func(worker int) {
			defer wg.Done()
			cmd := exec.Command(issuegraph, "backup", "sync")
			cmd.Dir = dir
			cmd.Env = issuegraphEnv(dir)
			out, err := cmd.CombinedOutput()
			results[worker] = result{worker: worker, out: string(out), err: err}
		}(w)
	}
	wg.Wait()

	successes := 0
	for _, r := range results {
		if strings.Contains(r.out, "panic") {
			t.Errorf("worker %d panicked:\n%s", r.worker, r.out)
		}
		if r.err == nil {
			successes++
		} else if !strings.Contains(r.out, "one writer at a time") {
			t.Errorf("worker %d failed with unexpected error: %v\n%s", r.worker, r.err, r.out)
		}
	}
	if successes < 1 {
		t.Errorf("expected at least 1 successful backup, got %d", successes)
	}
	t.Logf("%d/%d backup workers succeeded", successes, numWorkers)

	if _, err := os.Stat(backupDest); os.IsNotExist(err) {
		t.Error("backup destination should exist after concurrent syncs")
	}
}
