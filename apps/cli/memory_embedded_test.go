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

	"github.com/borg0ai/dx-issuegraph/internal/configfile"
	"github.com/borg0ai/dx-issuegraph/internal/storage/embeddeddolt"
)

// countMemoryDoltCommits reads dolt_log for an embedded workspace. It is how
// the auto-commit epilogue is observed: a direct-route write that never flags
// it still answers `issuegraph recall` perfectly well, because the row is in the
// working set — dolt_log is the only place the difference shows.
func countMemoryDoltCommits(t *testing.T, issuegraphDir string) int {
	t.Helper()
	cfg, _ := configfile.Load(issuegraphDir)
	database := ""
	if cfg != nil {
		database = cfg.GetDoltDatabase()
	}
	db, cleanup, err := embeddeddolt.OpenSQL(t.Context(), filepath.Join(issuegraphDir, "embeddeddolt"), database, "main")
	if err != nil {
		t.Fatalf("OpenSQL: %v", err)
	}
	defer cleanup()
	var count int
	if err := db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM dolt_log").Scan(&count); err != nil {
		t.Fatalf("query dolt_log: %v", err)
	}
	return count
}

// issuegraphRemember runs "issuegraph remember" with the given args and returns stdout.
func issuegraphRemember(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"remember"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	stdout, stderr, err := runCommandBuffers(t, cmd)
	if err != nil {
		t.Fatalf("issuegraph remember %s failed: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String()
}

// issuegraphRememberFail runs "issuegraph remember" expecting failure.
func issuegraphRememberFail(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"remember"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected issuegraph remember %s to fail, but succeeded:\n%s", strings.Join(args, " "), out)
	}
	return string(out)
}

// issuegraphRecall runs "issuegraph recall" with the given args and returns stdout.
func issuegraphRecall(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"recall"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	stdout, stderr, err := runCommandBuffers(t, cmd)
	if err != nil {
		t.Fatalf("issuegraph recall %s failed: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String()
}

// issuegraphRecallFail runs "issuegraph recall" expecting failure.
func issuegraphRecallFail(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"recall"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected issuegraph recall %s to fail, but succeeded:\n%s", strings.Join(args, " "), out)
	}
	return string(out)
}

// issuegraphMemories runs "issuegraph memories" with the given args and returns stdout.
func issuegraphMemories(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"memories"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	stdout, stderr, err := runCommandBuffers(t, cmd)
	if err != nil {
		t.Fatalf("issuegraph memories %s failed: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String()
}

// issuegraphForget runs "issuegraph forget" with the given args and returns stdout.
func issuegraphForget(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"forget"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	stdout, stderr, err := runCommandBuffers(t, cmd)
	if err != nil {
		t.Fatalf("issuegraph forget %s failed: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String()
}

// issuegraphForgetFail runs "issuegraph forget" expecting failure.
func issuegraphForgetFail(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"forget"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected issuegraph forget %s to fail, but succeeded:\n%s", strings.Join(args, " "), out)
	}
	return string(out)
}

func TestEmbeddedMemory(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "tm")

	// ===== Remember and Recall =====

	t.Run("remember_and_recall", func(t *testing.T) {
		issuegraphRemember(t, issuegraph, dir, "always run tests with -race flag")
		// Recall using auto-generated key — list memories to find it
		out := issuegraphMemories(t, issuegraph, dir)
		if !strings.Contains(out, "always run tests with -race flag") {
			t.Errorf("expected memory in list: %s", out)
		}
	})

	t.Run("remember_with_key", func(t *testing.T) {
		issuegraphRemember(t, issuegraph, dir, "auth module uses JWT not sessions", "--key", "auth-jwt")
		out := issuegraphRecall(t, issuegraph, dir, "auth-jwt")
		if !strings.Contains(out, "JWT") {
			t.Errorf("expected 'JWT' in recall output: %s", out)
		}
	})

	t.Run("remember_overwrite", func(t *testing.T) {
		issuegraphRemember(t, issuegraph, dir, "first version", "--key", "overwrite-test")
		issuegraphRemember(t, issuegraph, dir, "second version", "--key", "overwrite-test")
		out := issuegraphRecall(t, issuegraph, dir, "overwrite-test")
		if !strings.Contains(out, "second version") {
			t.Errorf("expected 'second version' after overwrite: %s", out)
		}
	})

	// Desire path: `issuegraph remember <existing-key>` (treating remember as a getter) is a READ --
	// it recalls the memory instead of overwriting it with its own key as content.
	t.Run("remember_bare_existing_key_recalls", func(t *testing.T) {
		issuegraphRemember(t, issuegraph, dir, "the real insight worth keeping", "--key", "guard-key")
		// Fat-finger a "read": positional arg is the existing bare key. Succeeds as a recall.
		out := issuegraphRemember(t, issuegraph, dir, "guard-key")
		if !strings.Contains(out, "the real insight worth keeping") {
			t.Errorf("expected bare existing key to recall the memory content, got: %s", out)
		}
		// The original content must be intact, not clobbered to "guard-key".
		recalled := issuegraphRecall(t, issuegraph, dir, "guard-key")
		if !strings.Contains(recalled, "the real insight worth keeping") {
			t.Errorf("recall-instead-of-write must preserve original content, got: %s", recalled)
		}
	})

	t.Run("remember_guard_bypass_with_explicit_key", func(t *testing.T) {
		issuegraphRemember(t, issuegraph, dir, "original", "--key", "bypass-key")
		// Explicit --key signals deliberate intent and bypasses the guard.
		issuegraphRemember(t, issuegraph, dir, "bypass-key", "--key", "bypass-key")
		out := issuegraphRecall(t, issuegraph, dir, "bypass-key")
		if !strings.Contains(out, "bypass-key") {
			t.Errorf("explicit --key should bypass guard and overwrite, got: %s", out)
		}
	})

	t.Run("remember_bare_slug_without_memory_refuses", func(t *testing.T) {
		// A bare slug naming NO existing memory is a mistyped read (or junk content) --
		// refuse rather than silently creating a memory whose content is its own key.
		out := issuegraphRememberFail(t, issuegraph, dir, "brand-new-slug-memory")
		if !strings.Contains(out, "memories") {
			t.Errorf("expected refusal to point at `issuegraph memories`, got: %s", out)
		}
		// Nothing may have been stored.
		issuegraphRecallFail(t, issuegraph, dir, "brand-new-slug-memory")
	})

	// The two refusals that sit either side of the desire path, and the reason
	// the bare-slug test is `derived != "" && derived == insight` rather than
	// the shipped `slugify(insight) == insight`: DeriveKey("") is "", so empty
	// and underivable content satisfies derived == insight and would be routed
	// into a "recall" of the empty key instead of being refused. The shipped
	// code was saved from that by an empty-content check that ran BEFORE the
	// branch; that check belongs to the role now, so the branch has to exclude
	// the empty key itself.
	t.Run("remember_refuses_content_no_key_derives_from", func(t *testing.T) {
		for _, tc := range []struct{ insight, want string }{
			{"", "memory content cannot be empty"},
			{"   ", "memory content cannot be empty"},
			{"!!!", "could not generate key from content"},
		} {
			out := issuegraphRememberFail(t, issuegraph, dir, tc.insight)
			if !strings.Contains(out, tc.want) {
				t.Errorf("issuegraph remember %q: expected %q, got: %s", tc.insight, tc.want, out)
			}
			if strings.Contains(out, "a bare existing key READS") || strings.Contains(out, "no memory named") {
				t.Errorf("issuegraph remember %q took the bare-slug desire path, got: %s", tc.insight, out)
			}
		}
	})

	t.Run("remember_new_slug_with_explicit_key_stores", func(t *testing.T) {
		// --key states write intent: slug-like content stores fine.
		issuegraphRemember(t, issuegraph, dir, "brand-new-slug-memory", "--key", "brand-new-slug-memory")
		out := issuegraphRecall(t, issuegraph, dir, "brand-new-slug-memory")
		if !strings.Contains(out, "brand-new-slug-memory") {
			t.Errorf("expected --key to store slug content, got: %s", out)
		}
	})

	// ===== Memories List =====

	t.Run("memories_list", func(t *testing.T) {
		out := issuegraphMemories(t, issuegraph, dir)
		// Should show at least the memories we stored above
		if !strings.Contains(out, "auth-jwt") {
			t.Errorf("expected 'auth-jwt' in memories list: %s", out)
		}
	})

	t.Run("memories_search", func(t *testing.T) {
		issuegraphRemember(t, issuegraph, dir, "dolt phantom DBs hide in three places", "--key", "dolt-phantoms")
		out := issuegraphMemories(t, issuegraph, dir, "dolt")
		if !strings.Contains(out, "dolt") {
			t.Errorf("expected 'dolt' related memory in search: %s", out)
		}
	})

	t.Run("memories_search_no_match", func(t *testing.T) {
		out := issuegraphMemories(t, issuegraph, dir, "zzz_nonexistent_term_xyz")
		// Should succeed but show no results or empty
		_ = out
	})

	// ===== Forget =====

	t.Run("forget", func(t *testing.T) {
		issuegraphRemember(t, issuegraph, dir, "temporary memory to forget", "--key", "forget-me")
		out := issuegraphRecall(t, issuegraph, dir, "forget-me")
		if !strings.Contains(out, "temporary memory") {
			t.Fatalf("expected memory before forget: %s", out)
		}
		issuegraphForget(t, issuegraph, dir, "forget-me")
		// After forget, recall should fail
		issuegraphRecallFail(t, issuegraph, dir, "forget-me")
	})

	// THE AUTO-COMMIT EPILOGUE, which is the trap this convergence was most
	// likely to spring. A direct-route write lands in the Dolt WORKING SET and
	// nothing else commits it: durability is the epilogue in main.go, keyed on
	// commandDidWrite, which noteDirectMemoryWrite is the only thing that sets
	// for this plane. A converged RunE that forgets it produces memories that
	// exist until the process exits and then sit uncommitted — and every other
	// assertion in this file still passes, because a working-set row recalls
	// and lists exactly like a committed one.
	//
	// Its own workspace, so the counts are deterministic (the pattern
	// close_multiple_ids_single_dolt_commit uses).
	t.Run("write_verbs_flag_the_auto_commit_epilogue", func(t *testing.T) {
		edir, eissuegraph, _ := issuegraphInit(t, issuegraph, "--prefix", "ep")

		// Warm-up write: a fresh workspace's first command may commit
		// bookkeeping of its own, which is not what this is measuring.
		issuegraphRemember(t, issuegraph, edir, "warm the workspace up", "--key", "epilogue-warmup")

		before := countMemoryDoltCommits(t, eissuegraph)
		issuegraphRemember(t, issuegraph, edir, "the epilogue is what makes a direct write durable", "--key", "epilogue-subject")
		afterRemember := countMemoryDoltCommits(t, eissuegraph)
		if got := afterRemember - before; got != 1 {
			t.Errorf("issuegraph remember: dolt_log %d -> %d (delta %d), want exactly one auto-commit", before, afterRemember, got)
		}

		issuegraphForget(t, issuegraph, edir, "epilogue-subject")
		afterForget := countMemoryDoltCommits(t, eissuegraph)
		if got := afterForget - afterRemember; got != 1 {
			t.Errorf("issuegraph forget: dolt_log %d -> %d (delta %d), want exactly one auto-commit", afterRemember, afterForget, got)
		}

		// A forget that found nothing deleted nothing, so it must leave no
		// commit either — the other half of the contract, and the one a
		// `noteDirectMemoryWrite()` placed before the not-found branch would
		// break while every output assertion stayed green.
		issuegraphForgetFail(t, issuegraph, edir, "epilogue-subject")
		if got := countMemoryDoltCommits(t, eissuegraph); got != afterForget {
			t.Errorf("issuegraph forget of an absent key: dolt_log %d -> %d, want no commit", afterForget, got)
		}
	})

	// ===== Error Cases =====

	t.Run("recall_missing_key", func(t *testing.T) {
		issuegraphRecallFail(t, issuegraph, dir, "nonexistent-key-xyz")
	})

	t.Run("forget_missing_key", func(t *testing.T) {
		issuegraphForgetFail(t, issuegraph, dir, "nonexistent-key-xyz")
	})

	t.Run("remember_no_args", func(t *testing.T) {
		issuegraphRememberFail(t, issuegraph, dir)
	})

	t.Run("recall_no_args", func(t *testing.T) {
		issuegraphRecallFail(t, issuegraph, dir)
	})

	t.Run("forget_no_args", func(t *testing.T) {
		issuegraphForgetFail(t, issuegraph, dir)
	})
}

// TestEmbeddedMemoryConcurrent exercises memory operations concurrently.
func TestEmbeddedMemoryConcurrent(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "mx")

	// Disable auto-export: this test exercises concurrent memory
	// flock contention, not export behavior. With export.auto=true
	// (the default since GH#2973), 8 concurrent writers also trigger
	// post-write read paths that race with in-flight commits.
	//
	// The underlying race is not flock-level (flock already serializes
	// issuegraph subprocesses) but engine-shutdown-level: Dolt's working-set
	// persistence can lag behind flock release, so the next subprocess
	// sometimes commits with a stale view and overwrites a prior forget.
	// See GH#3260 for the investigation (PR #3269 was the CI probe).
	// Fix would require synchronous flush on engine close or a long-lived
	// engine per store; both are substantial work. Until then, this
	// workaround keeps the test deterministic.
	disableAutoExport := exec.Command(issuegraph, "config", "set", "export.auto", "false")
	disableAutoExport.Dir = dir
	disableAutoExport.Env = issuegraphEnv(dir)
	if out, err := disableAutoExport.CombinedOutput(); err != nil {
		t.Fatalf("disable export.auto: %v\n%s", err, out)
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

			// Each worker stores memories with unique keys
			for i := 0; i < 3; i++ {
				key := fmt.Sprintf("w%d-mem%d", worker, i)
				content := fmt.Sprintf("worker %d memory %d content", worker, i)

				cmd := exec.Command(issuegraph, "remember", content, "--key", key)
				cmd.Dir = dir
				cmd.Env = issuegraphEnv(dir)
				out, err := cmd.CombinedOutput()
				if err != nil {
					r.err = fmt.Errorf("remember %s: %v\n%s", key, err, out)
					results[worker] = r
					return
				}
			}

			// Recall each memory
			for i := 0; i < 3; i++ {
				key := fmt.Sprintf("w%d-mem%d", worker, i)
				expected := fmt.Sprintf("worker %d memory %d content", worker, i)

				cmd := exec.Command(issuegraph, "recall", key)
				cmd.Dir = dir
				cmd.Env = issuegraphEnv(dir)
				out, err := cmd.CombinedOutput()
				if err != nil {
					r.err = fmt.Errorf("recall %s: %v\n%s", key, err, out)
					results[worker] = r
					return
				}
				if !strings.Contains(string(out), expected) {
					r.err = fmt.Errorf("recall %s: expected %q, got %q", key, expected, string(out))
					results[worker] = r
					return
				}
			}

			// Forget one memory
			forgetKey := fmt.Sprintf("w%d-mem0", worker)
			cmd := exec.Command(issuegraph, "forget", forgetKey)
			cmd.Dir = dir
			cmd.Env = issuegraphEnv(dir)
			out, err := cmd.CombinedOutput()
			if err != nil {
				r.err = fmt.Errorf("forget %s: %v\n%s", forgetKey, err, out)
				results[worker] = r
				return
			}

			// List memories
			cmd = exec.Command(issuegraph, "memories")
			cmd.Dir = dir
			cmd.Env = issuegraphEnv(dir)
			out, err = cmd.CombinedOutput()
			if err != nil {
				r.err = fmt.Errorf("memories: %v\n%s", err, out)
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

	// Verify memories only for workers that succeeded (err==nil).
	// With exclusive flock, some workers may fail with "one writer at a time".
	out := issuegraphMemories(t, issuegraph, dir)
	var successCount int
	for _, r := range results {
		if r.err != nil {
			continue
		}
		successCount++
		w := r.worker
		forgottenKey := fmt.Sprintf("w%d-mem0", w)
		if strings.Contains(out, forgottenKey) {
			t.Errorf("expected %s to be forgotten", forgottenKey)
		}
		for i := 1; i < 3; i++ {
			key := fmt.Sprintf("w%d-mem%d", w, i)
			if !strings.Contains(out, key) {
				t.Errorf("expected %s to still exist in memories", key)
			}
		}
	}
	if successCount == 0 {
		t.Fatal("expected at least 1 worker to succeed")
	}
}
