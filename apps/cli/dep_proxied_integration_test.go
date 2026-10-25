//go:build cgo

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func issuegraphProxiedDep(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"dep"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphProxiedEnv(dir)
	stdout, stderr, err := runCommandBuffers(t, cmd)
	if err != nil {
		t.Fatalf("issuegraph dep %s failed: %v\nstdout:\n%s\nstderr:\n%s",
			strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String()
}

func issuegraphProxiedDepFail(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"dep"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphProxiedEnv(dir)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected issuegraph dep %s to fail, but succeeded:\n%s",
			strings.Join(args, " "), out)
	}
	return string(out)
}

func issuegraphProxiedDepJSON(t *testing.T, issuegraph, dir string, args ...string) map[string]interface{} {
	t.Helper()
	fullArgs := append([]string{"dep"}, args...)
	fullArgs = append(fullArgs, "--json")
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphProxiedEnv(dir)
	stdout, stderr, err := runCommandBuffers(t, cmd)
	if err != nil {
		t.Fatalf("issuegraph dep --json %s failed: %v\nstdout:\n%s\nstderr:\n%s",
			strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	s := strings.TrimSpace(stdout.String())
	start := strings.IndexAny(s, "{[")
	if start < 0 {
		t.Fatalf("no JSON in dep output: %s", s)
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(s[start:]), &m); err != nil {
		t.Fatalf("parse dep JSON: %v\n%s", err, s)
	}
	return m
}

func issuegraphProxiedDepJSONArray(t *testing.T, issuegraph, dir string, args ...string) []interface{} {
	t.Helper()
	fullArgs := append([]string{"dep"}, args...)
	fullArgs = append(fullArgs, "--json")
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphProxiedEnv(dir)
	stdout, stderr, err := runCommandBuffers(t, cmd)
	if err != nil {
		t.Fatalf("issuegraph dep --json %s failed: %v\nstdout:\n%s\nstderr:\n%s",
			strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	s := strings.TrimSpace(stdout.String())
	start := strings.Index(s, "[")
	if start < 0 {
		t.Fatalf("no JSON array in dep output: %s", s)
	}
	var arr []interface{}
	if err := json.Unmarshal([]byte(s[start:]), &arr); err != nil {
		t.Fatalf("parse dep JSON array: %v\n%s", err, s)
	}
	return arr
}

func issuegraphProxiedDepWithInput(t *testing.T, issuegraph, dir, input string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"dep"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphProxiedEnv(dir)
	cmd.Stdin = strings.NewReader(input)
	stdout, stderr, err := runCommandBuffers(t, cmd)
	if err != nil {
		t.Fatalf("issuegraph dep %s failed: %v\nstdout:\n%s\nstderr:\n%s",
			strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String()
}

func issuegraphProxiedDepWithInputFail(t *testing.T, issuegraph, dir, input string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"dep"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphProxiedEnv(dir)
	cmd.Stdin = strings.NewReader(input)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected issuegraph dep %s to fail, but succeeded:\n%s",
			strings.Join(args, " "), out)
	}
	return string(out)
}

func TestProxiedServerDep(t *testing.T) {
	requireSharedProxiedServer(t)
	t.Parallel()
	issuegraph := buildEmbeddedBD(t)

	t.Run("Blocks", func(t *testing.T) {
		t.Parallel()
		t.Run("happy_path", func(t *testing.T) {
			t.Parallel()
			p := newSharedProxiedProject(t, issuegraph, "dpb1")
			a := issuegraphProxiedCreate(t, issuegraph, p.dir, "Block A", "--type", "task")
			b := issuegraphProxiedCreate(t, issuegraph, p.dir, "Block B", "--type", "task")
			out := issuegraphProxiedDep(t, issuegraph, p.dir, a.ID, "--blocks", b.ID)
			if !strings.Contains(out, "Added") || !strings.Contains(out, "blocks") {
				t.Errorf("expected 'Added ... blocks' output: %s", out)
			}
		})
	})

	t.Run("Add", func(t *testing.T) {
		t.Parallel()
		t.Run("positional_args", func(t *testing.T) {
			t.Parallel()
			p := newSharedProxiedProject(t, issuegraph, "dpa1")
			a := issuegraphProxiedCreate(t, issuegraph, p.dir, "Pos A", "--type", "task")
			b := issuegraphProxiedCreate(t, issuegraph, p.dir, "Pos B", "--type", "task")
			out := issuegraphProxiedDep(t, issuegraph, p.dir, "add", a.ID, b.ID)
			if !strings.Contains(out, "Added dependency") {
				t.Errorf("expected 'Added dependency': %s", out)
			}
		})

		t.Run("type_parent_child", func(t *testing.T) {
			t.Parallel()
			p := newSharedProxiedProject(t, issuegraph, "dpa2")
			epic := issuegraphProxiedCreate(t, issuegraph, p.dir, "Epic", "--type", "epic")
			c1 := issuegraphProxiedCreate(t, issuegraph, p.dir, "Child 1", "--type", "task")
			c2 := issuegraphProxiedCreate(t, issuegraph, p.dir, "Child 2", "--type", "task")
			issuegraphProxiedDep(t, issuegraph, p.dir, "add", c1.ID, epic.ID, "--type", "parent-child")
			issuegraphProxiedDep(t, issuegraph, p.dir, "add", c2.ID, epic.ID, "--type", "parent-child")
			out := issuegraphProxiedDep(t, issuegraph, p.dir, "list", epic.ID, "--direction", "up")
			if !strings.Contains(out, c1.ID) || !strings.Contains(out, c2.ID) {
				t.Errorf("expected both children in dependents: %s", out)
			}
		})

		t.Run("type_tracks", func(t *testing.T) {
			t.Parallel()
			p := newSharedProxiedProject(t, issuegraph, "dpa3")
			tracker := issuegraphProxiedCreate(t, issuegraph, p.dir, "Tracker", "--type", "task")
			tracked := issuegraphProxiedCreate(t, issuegraph, p.dir, "Tracked", "--type", "task")
			issuegraphProxiedDep(t, issuegraph, p.dir, "add", tracker.ID, tracked.ID, "--type", "tracks")
			out := issuegraphProxiedDep(t, issuegraph, p.dir, "list", tracker.ID)
			if !strings.Contains(out, tracked.ID) {
				t.Errorf("expected tracked issue in deps: %s", out)
			}
		})

		t.Run("type_related", func(t *testing.T) {
			t.Parallel()
			p := newSharedProxiedProject(t, issuegraph, "dpa4")
			r1 := issuegraphProxiedCreate(t, issuegraph, p.dir, "Related 1", "--type", "task")
			r2 := issuegraphProxiedCreate(t, issuegraph, p.dir, "Related 2", "--type", "task")
			out := issuegraphProxiedDep(t, issuegraph, p.dir, "add", r1.ID, r2.ID, "--type", "related")
			if !strings.Contains(out, "Added dependency") {
				t.Errorf("expected related edge accepted: %s", out)
			}
		})

		t.Run("blocked_by_flag", func(t *testing.T) {
			t.Parallel()
			p := newSharedProxiedProject(t, issuegraph, "dpa5")
			x := issuegraphProxiedCreate(t, issuegraph, p.dir, "Blocked", "--type", "task")
			y := issuegraphProxiedCreate(t, issuegraph, p.dir, "Blocker", "--type", "task")
			issuegraphProxiedDep(t, issuegraph, p.dir, "add", x.ID, "--blocked-by", y.ID)
			out := issuegraphProxiedDep(t, issuegraph, p.dir, "list", x.ID)
			if !strings.Contains(out, y.ID) {
				t.Errorf("expected blocker in deps: %s", out)
			}
		})

		t.Run("depends_on_flag", func(t *testing.T) {
			t.Parallel()
			p := newSharedProxiedProject(t, issuegraph, "dpa6")
			x := issuegraphProxiedCreate(t, issuegraph, p.dir, "Dependent", "--type", "task")
			y := issuegraphProxiedCreate(t, issuegraph, p.dir, "Dependency", "--type", "task")
			issuegraphProxiedDep(t, issuegraph, p.dir, "add", x.ID, "--depends-on", y.ID)
			out := issuegraphProxiedDep(t, issuegraph, p.dir, "list", x.ID)
			if !strings.Contains(out, y.ID) {
				t.Errorf("expected dependency in deps: %s", out)
			}
		})

		t.Run("cycle_rejected", func(t *testing.T) {
			t.Parallel()
			p := newSharedProxiedProject(t, issuegraph, "dpa7")
			a := issuegraphProxiedCreate(t, issuegraph, p.dir, "Cyc A", "--type", "task")
			b := issuegraphProxiedCreate(t, issuegraph, p.dir, "Cyc B", "--type", "task")
			issuegraphProxiedDep(t, issuegraph, p.dir, "add", a.ID, b.ID)
			out := issuegraphProxiedDepFail(t, issuegraph, p.dir, "add", b.ID, a.ID)
			if !strings.Contains(strings.ToLower(out), "cycle") {
				t.Errorf("expected 'cycle' error: %s", out)
			}
		})

		t.Run("child_parent_antipattern", func(t *testing.T) {
			t.Parallel()
			p := newSharedProxiedProject(t, issuegraph, "dpa8")
			parent := issuegraphProxiedCreate(t, issuegraph, p.dir, "AP Parent", "--type", "epic")
			child := issuegraphProxiedCreate(t, issuegraph, p.dir, "AP Child", "--type", "task", "--parent", parent.ID)
			out := issuegraphProxiedDepFail(t, issuegraph, p.dir, "add", child.ID, parent.ID)
			if !strings.Contains(out, "child") && !strings.Contains(out, "deadlock") {
				t.Errorf("expected child→parent rejection: %s", out)
			}
		})

		t.Run("json_output", func(t *testing.T) {
			t.Parallel()
			p := newSharedProxiedProject(t, issuegraph, "dpa9")
			j1 := issuegraphProxiedCreate(t, issuegraph, p.dir, "JSON A", "--type", "task")
			j2 := issuegraphProxiedCreate(t, issuegraph, p.dir, "JSON B", "--type", "task")
			m := issuegraphProxiedDepJSON(t, issuegraph, p.dir, "add", j1.ID, j2.ID)
			if m["status"] != "added" {
				t.Errorf("expected status=added, got %v", m["status"])
			}
			if m["issue_id"] != j1.ID || m["depends_on_id"] != j2.ID {
				t.Errorf("unexpected envelope: %v", m)
			}
		})

		t.Run("bulk_file_jsonl", func(t *testing.T) {
			t.Parallel()
			p := newSharedProxiedProject(t, issuegraph, "dpa10")
			b1 := issuegraphProxiedCreate(t, issuegraph, p.dir, "Bulk A", "--type", "task")
			b2 := issuegraphProxiedCreate(t, issuegraph, p.dir, "Bulk B", "--type", "task")
			b3 := issuegraphProxiedCreate(t, issuegraph, p.dir, "Bulk C", "--type", "task")
			path := filepath.Join(t.TempDir(), "deps.jsonl")
			body := fmt.Sprintf("{\"from\":%q,\"to\":%q}\n{\"issue_id\":%q,\"depends_on_id\":%q,\"type\":\"tracks\"}\n",
				b1.ID, b2.ID, b3.ID, b2.ID)
			if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
				t.Fatalf("write deps file: %v", err)
			}
			out := issuegraphProxiedDep(t, issuegraph, p.dir, "add", "--file", path)
			if !strings.Contains(out, "Added 2 dependencies") {
				t.Fatalf("expected bulk add summary: %s", out)
			}
			list1 := issuegraphProxiedDep(t, issuegraph, p.dir, "list", b1.ID)
			if !strings.Contains(list1, b2.ID) {
				t.Fatalf("expected first bulk dep in list: %s", list1)
			}
			list3 := issuegraphProxiedDep(t, issuegraph, p.dir, "list", b3.ID)
			if !strings.Contains(list3, b2.ID) || !strings.Contains(list3, "tracks") {
				t.Fatalf("expected typed bulk dep in list: %s", list3)
			}
		})

		t.Run("bulk_file_validation_atomic", func(t *testing.T) {
			t.Parallel()
			p := newSharedProxiedProject(t, issuegraph, "dpa11")
			v1 := issuegraphProxiedCreate(t, issuegraph, p.dir, "Valid A", "--type", "task")
			v2 := issuegraphProxiedCreate(t, issuegraph, p.dir, "Valid B", "--type", "task")
			path := filepath.Join(t.TempDir(), "bad-deps.jsonl")
			body := fmt.Sprintf("{\"from\":%q,\"to\":%q}\n{\"from\":\"\",\"to\":%q}\n{\"from\":%q,\"to\":\"\"}\n",
				v1.ID, v2.ID, v2.ID, v1.ID)
			if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
				t.Fatalf("write bad deps file: %v", err)
			}
			out := issuegraphProxiedDepFail(t, issuegraph, p.dir, "add", "--file", path)
			if !strings.Contains(out, "line 2: missing from") || !strings.Contains(out, "line 3: missing to") {
				t.Fatalf("expected all validation errors: %s", out)
			}
			list := issuegraphProxiedDep(t, issuegraph, p.dir, "list", v1.ID)
			if strings.Contains(list, v2.ID) {
				t.Fatalf("bulk validation failure should not partial-commit: %s", list)
			}
		})

		t.Run("no_cycle_check_chain", func(t *testing.T) {
			t.Parallel()
			p := newSharedProxiedProject(t, issuegraph, "dpa12")
			const n = 10
			ids := make([]string, n)
			for i := 0; i < n; i++ {
				issue := issuegraphProxiedCreate(t, issuegraph, p.dir, fmt.Sprintf("chain-%d", i), "--type", "task")
				ids[i] = issue.ID
			}
			for i := 1; i < n; i++ {
				out := issuegraphProxiedDep(t, issuegraph, p.dir, "add", ids[i], ids[i-1], "--no-cycle-check")
				if strings.Contains(out, "cycle") {
					t.Errorf("unexpected cycle output with --no-cycle-check: %s", out)
				}
			}
			cyclesOut := issuegraphProxiedDep(t, issuegraph, p.dir, "cycles")
			if strings.Contains(cyclesOut, "Found") {
				t.Errorf("unexpected cycles after acyclic chain wiring: %s", cyclesOut)
			}
			extra := issuegraphProxiedCreate(t, issuegraph, p.dir, "chain-extra", "--type", "task")
			issuegraphProxiedDep(t, issuegraph, p.dir, extra.ID, "--blocks", ids[n-1], "--no-cycle-check")
		})

		t.Run("bulk_no_cycle_check_whole_graph_gate", func(t *testing.T) {
			t.Parallel()
			p := newSharedProxiedProject(t, issuegraph, "dpa13")
			a := issuegraphProxiedCreate(t, issuegraph, p.dir, "Gate A", "--type", "task")
			b := issuegraphProxiedCreate(t, issuegraph, p.dir, "Gate B", "--type", "task")
			c := issuegraphProxiedCreate(t, issuegraph, p.dir, "Gate C", "--type", "task")

			okInput := fmt.Sprintf("{\"from\":%q,\"to\":%q}\n{\"from\":%q,\"to\":%q}\n",
				a.ID, b.ID, b.ID, c.ID)
			out := issuegraphProxiedDepWithInput(t, issuegraph, p.dir, okInput, "add", "--file", "-", "--no-cycle-check")
			if !strings.Contains(out, "Added 2 dependencies") {
				t.Fatalf("expected acyclic bulk add summary: %s", out)
			}

			cycleInput := fmt.Sprintf("{\"from\":%q,\"to\":%q}\n", c.ID, a.ID)
			failOut := issuegraphProxiedDepWithInputFail(t, issuegraph, p.dir, cycleInput, "add", "--file", "-", "--no-cycle-check")
			if !strings.Contains(strings.ToLower(failOut), "cycle") {
				t.Fatalf("expected whole-graph cycle rejection: %s", failOut)
			}

			cycles := issuegraphProxiedDep(t, issuegraph, p.dir, "cycles")
			if !strings.Contains(cycles, "No dependency cycles detected") {
				t.Fatalf("expected rolled-back bulk add to leave graph acyclic: %s", cycles)
			}
		})

		t.Run("commit_visible_to_subsequent_invocation", func(t *testing.T) {
			t.Parallel()
			p := newSharedProxiedProject(t, issuegraph, "dpa14")
			a := issuegraphProxiedCreate(t, issuegraph, p.dir, "Commit A", "--type", "task")
			b := issuegraphProxiedCreate(t, issuegraph, p.dir, "Commit B", "--type", "task")
			issuegraphProxiedDep(t, issuegraph, p.dir, "add", a.ID, b.ID)
			out := issuegraphProxiedDep(t, issuegraph, p.dir, "list", a.ID)
			if !strings.Contains(out, b.ID) {
				t.Fatalf("dep not visible in fresh CLI invocation after add: %s", out)
			}
		})

		t.Run("bulk_external_ref_passthrough", func(t *testing.T) {
			t.Parallel()
			p := newSharedProxiedProject(t, issuegraph, "dpa15")
			src := issuegraphProxiedCreate(t, issuegraph, p.dir, "Ext src", "--type", "task")
			input := fmt.Sprintf("{\"from\":%q,\"to\":\"external:other:capability\"}\n", src.ID)
			cmd := exec.Command(issuegraph, "dep", "add", "--file", "-")
			cmd.Dir = p.dir
			cmd.Env = issuegraphProxiedEnv(p.dir)
			cmd.Stdin = strings.NewReader(input)
			out, err := cmd.CombinedOutput()
			if err != nil {
				if !strings.Contains(string(out), "external") {
					t.Fatalf("external-ref bulk add failed without a clear external-ref error: %v\n%s", err, out)
				}
				return
			}
			if !strings.Contains(string(out), "Added 1 dependencies") {
				t.Fatalf("expected 1-edge bulk add summary or clear external-ref error: %s", out)
			}
		})
	})

}

func TestProxiedServerDep2(t *testing.T) {
	requireSharedProxiedServer(t)
	t.Parallel()
	issuegraph := buildEmbeddedBD(t)

	t.Run("Remove", func(t *testing.T) {
		t.Parallel()
		t.Run("basic", func(t *testing.T) {
			t.Parallel()
			p := newSharedProxiedProject(t, issuegraph, "dpr1")
			a := issuegraphProxiedCreate(t, issuegraph, p.dir, "Rm A", "--type", "task")
			b := issuegraphProxiedCreate(t, issuegraph, p.dir, "Rm B", "--type", "task")
			issuegraphProxiedDep(t, issuegraph, p.dir, "add", a.ID, b.ID)
			out := issuegraphProxiedDep(t, issuegraph, p.dir, "remove", a.ID, b.ID)
			if !strings.Contains(out, "Removed") {
				t.Errorf("expected 'Removed': %s", out)
			}
		})

		t.Run("rm_alias", func(t *testing.T) {
			t.Parallel()
			p := newSharedProxiedProject(t, issuegraph, "dpr2")
			a := issuegraphProxiedCreate(t, issuegraph, p.dir, "Rm A", "--type", "task")
			b := issuegraphProxiedCreate(t, issuegraph, p.dir, "Rm B", "--type", "task")
			issuegraphProxiedDep(t, issuegraph, p.dir, "add", a.ID, b.ID)
			out := issuegraphProxiedDep(t, issuegraph, p.dir, "rm", a.ID, b.ID)
			if !strings.Contains(out, "Removed") {
				t.Errorf("expected 'Removed' via rm alias: %s", out)
			}
		})

		t.Run("json_output", func(t *testing.T) {
			t.Parallel()
			p := newSharedProxiedProject(t, issuegraph, "dpr3")
			a := issuegraphProxiedCreate(t, issuegraph, p.dir, "Rm JSON A", "--type", "task")
			b := issuegraphProxiedCreate(t, issuegraph, p.dir, "Rm JSON B", "--type", "task")
			issuegraphProxiedDep(t, issuegraph, p.dir, "add", a.ID, b.ID)
			m := issuegraphProxiedDepJSON(t, issuegraph, p.dir, "remove", a.ID, b.ID)
			if m["status"] != "removed" {
				t.Errorf("expected status=removed, got %v", m["status"])
			}
			if m["removed"] != true {
				t.Errorf("expected removed=true, got %v", m["removed"])
			}
		})

		t.Run("missing_edge_reports_noop", func(t *testing.T) {
			t.Parallel()
			p := newSharedProxiedProject(t, issuegraph, "dpr4")
			a := issuegraphProxiedCreate(t, issuegraph, p.dir, "Missing edge A", "--type", "task")
			b := issuegraphProxiedCreate(t, issuegraph, p.dir, "Missing edge B", "--type", "task")
			out := issuegraphProxiedDep(t, issuegraph, p.dir, "remove", a.ID, b.ID)
			if !strings.Contains(out, "No dependency found") || strings.Contains(out, "✓") {
				t.Errorf("missing edge should report a no-op, got: %s", out)
			}
			m := issuegraphProxiedDepJSON(t, issuegraph, p.dir, "remove", a.ID, b.ID)
			if m["status"] != "not_found" || m["removed"] != false {
				t.Errorf("missing edge JSON = %v, want status=not_found removed=false", m)
			}
		})
	})

	t.Run("List", func(t *testing.T) {
		t.Parallel()
		t.Run("default_direction_down", func(t *testing.T) {
			t.Parallel()
			p := newSharedProxiedProject(t, issuegraph, "dpl1")
			a := issuegraphProxiedCreate(t, issuegraph, p.dir, "L A", "--type", "task")
			b := issuegraphProxiedCreate(t, issuegraph, p.dir, "L B", "--type", "task")
			issuegraphProxiedDep(t, issuegraph, p.dir, "add", a.ID, b.ID)
			out := issuegraphProxiedDep(t, issuegraph, p.dir, "list", a.ID)
			if !strings.Contains(out, b.ID) {
				t.Errorf("expected dependency in list output: %s", out)
			}
		})

		t.Run("direction_up", func(t *testing.T) {
			t.Parallel()
			p := newSharedProxiedProject(t, issuegraph, "dpl2")
			a := issuegraphProxiedCreate(t, issuegraph, p.dir, "Up A", "--type", "task")
			b := issuegraphProxiedCreate(t, issuegraph, p.dir, "Up B", "--type", "task")
			issuegraphProxiedDep(t, issuegraph, p.dir, "add", a.ID, b.ID)
			out := issuegraphProxiedDep(t, issuegraph, p.dir, "list", b.ID, "--direction", "up")
			if !strings.Contains(out, a.ID) {
				t.Errorf("expected dependent in --direction up: %s", out)
			}
		})

		t.Run("type_filter", func(t *testing.T) {
			t.Parallel()
			p := newSharedProxiedProject(t, issuegraph, "dpl3")
			epic := issuegraphProxiedCreate(t, issuegraph, p.dir, "TF epic", "--type", "epic")
			c1 := issuegraphProxiedCreate(t, issuegraph, p.dir, "TF c1", "--type", "task")
			c2 := issuegraphProxiedCreate(t, issuegraph, p.dir, "TF c2", "--type", "task")
			issuegraphProxiedDep(t, issuegraph, p.dir, "add", c1.ID, epic.ID, "--type", "parent-child")
			issuegraphProxiedDep(t, issuegraph, p.dir, "add", c2.ID, epic.ID, "--type", "parent-child")
			out := issuegraphProxiedDep(t, issuegraph, p.dir, "list", epic.ID, "--direction", "up", "--type", "parent-child")
			if !strings.Contains(out, c1.ID) || !strings.Contains(out, c2.ID) {
				t.Errorf("expected children in type-filtered list: %s", out)
			}
		})

		t.Run("json_output", func(t *testing.T) {
			t.Parallel()
			p := newSharedProxiedProject(t, issuegraph, "dpl4")
			a := issuegraphProxiedCreate(t, issuegraph, p.dir, "LJ A", "--type", "task")
			b := issuegraphProxiedCreate(t, issuegraph, p.dir, "LJ B", "--type", "task")
			issuegraphProxiedDep(t, issuegraph, p.dir, "add", a.ID, b.ID)
			arr := issuegraphProxiedDepJSONArray(t, issuegraph, p.dir, "list", a.ID)
			if len(arr) == 0 {
				t.Fatalf("expected non-empty JSON list array")
			}
			raw, _ := json.Marshal(arr)
			if !strings.Contains(string(raw), b.ID) {
				t.Errorf("expected dependency target in JSON: %s", string(raw))
			}
		})

		t.Run("batch_multi_arg", func(t *testing.T) {
			t.Parallel()
			p := newSharedProxiedProject(t, issuegraph, "dpl5")
			a := issuegraphProxiedCreate(t, issuegraph, p.dir, "B A", "--type", "task")
			b := issuegraphProxiedCreate(t, issuegraph, p.dir, "B B", "--type", "task")
			c := issuegraphProxiedCreate(t, issuegraph, p.dir, "B C", "--type", "task")
			d := issuegraphProxiedCreate(t, issuegraph, p.dir, "B D", "--type", "task")
			issuegraphProxiedDep(t, issuegraph, p.dir, "add", a.ID, b.ID)
			issuegraphProxiedDep(t, issuegraph, p.dir, "add", c.ID, d.ID)
			out := issuegraphProxiedDep(t, issuegraph, p.dir, "list", a.ID, c.ID)
			if !strings.Contains(out, b.ID) || !strings.Contains(out, d.ID) {
				t.Errorf("expected both deps in batch list: %s", out)
			}
			arr := issuegraphProxiedDepJSONArray(t, issuegraph, p.dir, "list", a.ID, c.ID)
			if len(arr) != 2 {
				t.Errorf("expected 2 deps in batch JSON, got %d: %v", len(arr), arr)
			}
		})

		t.Run("batch_partitions_wisp_and_issue_ids", func(t *testing.T) {
			t.Parallel()
			p := newSharedProxiedProject(t, issuegraph, "dpl6")
			issueSrc := issuegraphProxiedCreate(t, issuegraph, p.dir, "WP issue src", "--type", "task")
			issueTgt := issuegraphProxiedCreate(t, issuegraph, p.dir, "WP issue tgt", "--type", "task")
			issuegraphProxiedDep(t, issuegraph, p.dir, "add", issueSrc.ID, issueTgt.ID)

			db := openProxiedDB(t, p)
			wispSrc := fmt.Sprintf("%s-wisp-src", p.prefix)
			wispTgt := fmt.Sprintf("%s-wisp-tgt", p.prefix)
			if _, err := db.ExecContext(context.Background(),
				"INSERT INTO wisps (id, title) VALUES (?, ?), (?, ?)",
				wispSrc, "wisp src", wispTgt, "wisp tgt"); err != nil {
				t.Fatalf("seed wisps: %v", err)
			}
			if _, err := db.ExecContext(context.Background(),
				"INSERT INTO wisp_dependencies (id, issue_id, depends_on_wisp_id, type, created_at, created_by, metadata) VALUES (UUID(), ?, ?, 'blocks', NOW(), 'tester', '{}')",
				wispSrc, wispTgt); err != nil {
				t.Fatalf("seed wisp dep: %v", err)
			}

			arr := issuegraphProxiedDepJSONArray(t, issuegraph, p.dir, "list", issueSrc.ID, wispSrc)
			raw, _ := json.Marshal(arr)
			if !strings.Contains(string(raw), issueTgt.ID) {
				t.Errorf("expected issue target %s in batch list: %s", issueTgt.ID, string(raw))
			}
			if !strings.Contains(string(raw), wispTgt) {
				t.Errorf("expected wisp target %s in batch list (partitioning path): %s", wispTgt, string(raw))
			}
		})
	})

	t.Run("Tree", func(t *testing.T) {
		t.Parallel()
		t.Run("basic", func(t *testing.T) {
			t.Parallel()
			p := newSharedProxiedProject(t, issuegraph, "dpt1")
			root := issuegraphProxiedCreate(t, issuegraph, p.dir, "Tree root", "--type", "task")
			child := issuegraphProxiedCreate(t, issuegraph, p.dir, "Tree child", "--type", "task")
			issuegraphProxiedDep(t, issuegraph, p.dir, "add", root.ID, child.ID)
			out := issuegraphProxiedDep(t, issuegraph, p.dir, "tree", root.ID)
			if !strings.Contains(out, root.ID) {
				t.Errorf("expected root ID in tree: %s", out)
			}
		})

		t.Run("direction_up", func(t *testing.T) {
			t.Parallel()
			p := newSharedProxiedProject(t, issuegraph, "dpt2")
			root := issuegraphProxiedCreate(t, issuegraph, p.dir, "Up tree root", "--type", "task")
			dep := issuegraphProxiedCreate(t, issuegraph, p.dir, "Up tree dep", "--type", "task")
			issuegraphProxiedDep(t, issuegraph, p.dir, "add", root.ID, dep.ID)
			out := issuegraphProxiedDep(t, issuegraph, p.dir, "tree", dep.ID, "--direction", "up")
			if len(strings.TrimSpace(out)) == 0 {
				t.Error("expected non-empty tree output for --direction up")
			}
		})

		t.Run("direction_both", func(t *testing.T) {
			t.Parallel()
			p := newSharedProxiedProject(t, issuegraph, "dpt3")
			root := issuegraphProxiedCreate(t, issuegraph, p.dir, "Both root", "--type", "task")
			dep := issuegraphProxiedCreate(t, issuegraph, p.dir, "Both dep", "--type", "task")
			issuegraphProxiedDep(t, issuegraph, p.dir, "add", root.ID, dep.ID)
			out := issuegraphProxiedDep(t, issuegraph, p.dir, "tree", root.ID, "--direction", "both")
			if len(strings.TrimSpace(out)) == 0 {
				t.Error("expected non-empty tree output for --direction both")
			}
		})

		t.Run("both_does_not_duplicate_root", func(t *testing.T) {
			t.Parallel()
			p := newSharedProxiedProject(t, issuegraph, "dpt4")
			root := issuegraphProxiedCreate(t, issuegraph, p.dir, "Dedup root", "--type", "task")
			depDown := issuegraphProxiedCreate(t, issuegraph, p.dir, "Dedup down", "--type", "task")
			depUp := issuegraphProxiedCreate(t, issuegraph, p.dir, "Dedup up", "--type", "task")
			issuegraphProxiedDep(t, issuegraph, p.dir, "add", root.ID, depDown.ID)
			issuegraphProxiedDep(t, issuegraph, p.dir, "add", depUp.ID, root.ID)
			arr := issuegraphProxiedDepJSONArray(t, issuegraph, p.dir, "tree", root.ID, "--direction", "both")
			count := 0
			for _, node := range arr {
				m, ok := node.(map[string]interface{})
				if !ok {
					continue
				}
				if id, _ := m["id"].(string); id == root.ID {
					count++
				}
			}
			if count != 1 {
				t.Errorf("expected root %s to appear exactly once in both tree, got %d: %v", root.ID, count, arr)
			}
		})

		t.Run("max_depth", func(t *testing.T) {
			t.Parallel()
			p := newSharedProxiedProject(t, issuegraph, "dpt5")
			a := issuegraphProxiedCreate(t, issuegraph, p.dir, "MD A", "--type", "task")
			b := issuegraphProxiedCreate(t, issuegraph, p.dir, "MD B", "--type", "task")
			c := issuegraphProxiedCreate(t, issuegraph, p.dir, "MD C", "--type", "task")
			issuegraphProxiedDep(t, issuegraph, p.dir, "add", a.ID, b.ID)
			issuegraphProxiedDep(t, issuegraph, p.dir, "add", b.ID, c.ID)
			out := issuegraphProxiedDep(t, issuegraph, p.dir, "tree", a.ID, "--max-depth", "1")
			if !strings.Contains(out, a.ID) {
				t.Errorf("expected root in shallow tree: %s", out)
			}
		})

		t.Run("status_filter", func(t *testing.T) {
			t.Parallel()
			p := newSharedProxiedProject(t, issuegraph, "dpt6")
			root := issuegraphProxiedCreate(t, issuegraph, p.dir, "SF root", "--type", "task")
			dep := issuegraphProxiedCreate(t, issuegraph, p.dir, "SF dep", "--type", "task")
			issuegraphProxiedDep(t, issuegraph, p.dir, "add", root.ID, dep.ID)
			_ = issuegraphProxiedDep(t, issuegraph, p.dir, "tree", root.ID, "--status", "open")
		})

		t.Run("format_mermaid", func(t *testing.T) {
			t.Parallel()
			p := newSharedProxiedProject(t, issuegraph, "dpt7")
			root := issuegraphProxiedCreate(t, issuegraph, p.dir, "Mer root", "--type", "task")
			dep := issuegraphProxiedCreate(t, issuegraph, p.dir, "Mer dep", "--type", "task")
			issuegraphProxiedDep(t, issuegraph, p.dir, "add", root.ID, dep.ID)
			out := issuegraphProxiedDep(t, issuegraph, p.dir, "tree", root.ID, "--format", "mermaid")
			if !strings.Contains(out, "flowchart") && !strings.Contains(out, "graph") {
				t.Errorf("expected mermaid flowchart/graph syntax: %s", out)
			}
		})

		t.Run("json_output", func(t *testing.T) {
			t.Parallel()
			p := newSharedProxiedProject(t, issuegraph, "dpt8")
			root := issuegraphProxiedCreate(t, issuegraph, p.dir, "JT root", "--type", "task")
			dep := issuegraphProxiedCreate(t, issuegraph, p.dir, "JT dep", "--type", "task")
			issuegraphProxiedDep(t, issuegraph, p.dir, "add", root.ID, dep.ID)
			arr := issuegraphProxiedDepJSONArray(t, issuegraph, p.dir, "tree", root.ID)
			raw, _ := json.Marshal(arr)
			if !strings.Contains(string(raw), root.ID) {
				t.Errorf("expected root ID in JSON tree: %s", string(raw))
			}
		})

		// bd-lx5iu / Q11: this route resolves a PARTIAL id now. It used to hand the
		// argument to the use case verbatim, so the same abbreviation that worked
		// on a direct workspace failed on a team server.
		t.Run("partial_id_resolves", func(t *testing.T) {
			t.Parallel()
			p := newSharedProxiedProject(t, issuegraph, "dpta")
			root := issuegraphProxiedCreate(t, issuegraph, p.dir, "Partial root", "--type", "task")
			dep := issuegraphProxiedCreate(t, issuegraph, p.dir, "Partial dep", "--type", "task")
			issuegraphProxiedDep(t, issuegraph, p.dir, "add", root.ID, dep.ID)

			// The bare suffix, with the workspace prefix stripped.
			bare := root.ID[strings.Index(root.ID, "-")+1:]
			out := issuegraphProxiedDep(t, issuegraph, p.dir, "tree", bare)
			if !strings.Contains(out, root.ID) || !strings.Contains(out, dep.ID) {
				t.Fatalf("a partial id did not resolve to the full tree: %s", out)
			}
		})

		t.Run("absent_root_is_a_lookup_failure", func(t *testing.T) {
			t.Parallel()
			p := newSharedProxiedProject(t, issuegraph, "dptb")
			out := issuegraphProxiedDepFail(t, issuegraph, p.dir, "tree", "dptb-nosuchissue")
			if !strings.Contains(out, "dptb-nosuchissue") {
				t.Fatalf("the refusal does not name the id the caller gave: %s", out)
			}
		})

		// bd-lx5iu / Q11: --max-rows is HONORED on this route now. It used to be
		// refused outright ("not supported in proxied-server mode") because the
		// unit-of-work path threaded no cap; the cap is on the role's request, so
		// it threads everywhere.
		t.Run("max_rows_is_honored", func(t *testing.T) {
			t.Parallel()
			p := newSharedProxiedProject(t, issuegraph, "dptc")
			root := issuegraphProxiedCreate(t, issuegraph, p.dir, "Cap root", "--type", "task")
			for i := 0; i < 3; i++ {
				child := issuegraphProxiedCreate(t, issuegraph, p.dir, fmt.Sprintf("Cap child %d", i), "--type", "task")
				issuegraphProxiedDep(t, issuegraph, p.dir, "add", root.ID, child.ID)
			}

			out := issuegraphProxiedDepFail(t, issuegraph, p.dir, "tree", root.ID, "--max-rows", "2")
			if strings.Contains(out, "not supported in proxied-server mode") {
				t.Fatalf("the cap is still being refused rather than enforced: %s", out)
			}
			if !strings.Contains(out, "too many rows") {
				t.Fatalf("stderr missing the cap message: %s", out)
			}
			if !strings.Contains(out, "--max-rows=2") {
				t.Fatalf("stderr missing the source attribution: %s", out)
			}

			// And the same walk under a cap it fits inside answers normally, so the
			// case is about the cap rather than about the tree.
			under := issuegraphProxiedDep(t, issuegraph, p.dir, "tree", root.ID, "--max-rows", "4")
			if !strings.Contains(under, root.ID) {
				t.Fatalf("a walk under the cap did not answer: %s", under)
			}
		})

		// --show-all-paths is a DOCUMENTED NO-OP (Q7). It is still accepted so no
		// script breaks, and a diamond is still rendered once.
		t.Run("show_all_paths_is_an_accepted_no_op", func(t *testing.T) {
			t.Parallel()
			p := newSharedProxiedProject(t, issuegraph, "dptd")
			root := issuegraphProxiedCreate(t, issuegraph, p.dir, "Diamond root", "--type", "task")
			left := issuegraphProxiedCreate(t, issuegraph, p.dir, "Diamond left", "--type", "task")
			right := issuegraphProxiedCreate(t, issuegraph, p.dir, "Diamond right", "--type", "task")
			shared := issuegraphProxiedCreate(t, issuegraph, p.dir, "Diamond shared", "--type", "task")
			issuegraphProxiedDep(t, issuegraph, p.dir, "add", root.ID, left.ID)
			issuegraphProxiedDep(t, issuegraph, p.dir, "add", root.ID, right.ID)
			issuegraphProxiedDep(t, issuegraph, p.dir, "add", left.ID, shared.ID)
			issuegraphProxiedDep(t, issuegraph, p.dir, "add", right.ID, shared.ID)

			for _, args := range [][]string{
				{"tree", root.ID},
				{"tree", root.ID, "--show-all-paths"},
			} {
				arr := issuegraphProxiedDepJSONArray(t, issuegraph, p.dir, args...)
				count := 0
				for _, node := range arr {
					m, ok := node.(map[string]interface{})
					if !ok {
						continue
					}
					if id, _ := m["id"].(string); id == shared.ID {
						count++
					}
				}
				if count != 1 {
					t.Errorf("%v: shared node appears %d times, want 1", args, count)
				}
			}
		})

		t.Run("ignores_relates_to", func(t *testing.T) {
			t.Parallel()
			p := newSharedProxiedProject(t, issuegraph, "dpt9")
			root := issuegraphProxiedCreate(t, issuegraph, p.dir, "Rel root", "--type", "task")
			blocker := issuegraphProxiedCreate(t, issuegraph, p.dir, "Real blocker", "--type", "task")
			related := issuegraphProxiedCreate(t, issuegraph, p.dir, "Loose relation", "--type", "task")
			issuegraphProxiedDep(t, issuegraph, p.dir, "add", root.ID, blocker.ID, "--type", "blocks")
			issuegraphProxiedDep(t, issuegraph, p.dir, "add", root.ID, related.ID, "--type", "relates-to")
			issuegraphProxiedDep(t, issuegraph, p.dir, "add", related.ID, root.ID, "--type", "relates-to")

			listOut := issuegraphProxiedDep(t, issuegraph, p.dir, "list", root.ID)
			if !strings.Contains(listOut, related.ID) || !strings.Contains(listOut, "relates-to") {
				t.Fatalf("expected relates-to in dep list: %s", listOut)
			}

			treeOut := issuegraphProxiedDep(t, issuegraph, p.dir, "tree", root.ID)
			if !strings.Contains(treeOut, root.ID) || !strings.Contains(treeOut, blocker.ID) {
				t.Fatalf("expected root + real blocker in tree: %s", treeOut)
			}
			if strings.Contains(treeOut, related.ID) {
				t.Fatalf("relates-to should not render in tree: %s", treeOut)
			}

			upOut := issuegraphProxiedDep(t, issuegraph, p.dir, "tree", related.ID, "--direction", "up")
			if strings.Contains(upOut, root.ID) {
				t.Fatalf("reverse relates-to should not render in dependent tree: %s", upOut)
			}
		})
	})

	t.Run("Cycles", func(t *testing.T) {
		t.Parallel()
		t.Run("detect", func(t *testing.T) {
			t.Parallel()
			p := newSharedProxiedProject(t, issuegraph, "dpc1")
			a := issuegraphProxiedCreate(t, issuegraph, p.dir, "C A", "--type", "task")
			b := issuegraphProxiedCreate(t, issuegraph, p.dir, "C B", "--type", "task")
			issuegraphProxiedDep(t, issuegraph, p.dir, "add", a.ID, b.ID)

			db := openProxiedDB(t, p)
			if _, err := db.ExecContext(context.Background(),
				"INSERT INTO dependencies (id, issue_id, depends_on_issue_id, type, created_at, created_by, metadata) VALUES (UUID(), ?, ?, 'blocks', NOW(), 'tester', '{}')",
				b.ID, a.ID); err != nil {
				t.Fatalf("seed cycle: %v", err)
			}

			out := issuegraphProxiedDep(t, issuegraph, p.dir, "cycles")
			if !strings.Contains(out, a.ID) || !strings.Contains(out, b.ID) {
				t.Errorf("expected cycle members in output: %s", out)
			}
		})

		t.Run("none", func(t *testing.T) {
			t.Parallel()
			p := newSharedProxiedProject(t, issuegraph, "dpc2")
			a := issuegraphProxiedCreate(t, issuegraph, p.dir, "NC A", "--type", "task")
			b := issuegraphProxiedCreate(t, issuegraph, p.dir, "NC B", "--type", "task")
			issuegraphProxiedDep(t, issuegraph, p.dir, "add", a.ID, b.ID)
			out := issuegraphProxiedDep(t, issuegraph, p.dir, "cycles")
			if !strings.Contains(out, "No dependency cycles detected") {
				t.Errorf("expected no-cycle message: %s", out)
			}
		})
	})
}

func TestProxiedServerDepConcurrent(t *testing.T) {
	requireSharedProxiedServer(t)
	t.Parallel()
	issuegraph := buildEmbeddedBD(t)
	p := newSharedProxiedProject(t, issuegraph, "dpcon")

	var ids []string
	for i := 0; i < 16; i++ {
		issue := issuegraphProxiedCreate(t, issuegraph, p.dir, fmt.Sprintf("concurrent-%d", i), "--type", "task")
		ids = append(ids, issue.ID)
	}
	for i := 0; i < 8; i++ {
		issuegraphProxiedDep(t, issuegraph, p.dir, "add", ids[i*2], ids[i*2+1])
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
			id := ids[worker*2]

			cmd := exec.Command(issuegraph, "dep", "list", id, "--json")
			cmd.Dir = p.dir
			cmd.Env = issuegraphProxiedEnv(p.dir)
			if out, err := cmd.CombinedOutput(); err != nil {
				r.err = fmt.Errorf("worker %d dep list: %v\n%s", worker, err, out)
				results[worker] = r
				return
			}

			cmd = exec.Command(issuegraph, "dep", "tree", id)
			cmd.Dir = p.dir
			cmd.Env = issuegraphProxiedEnv(p.dir)
			if out, err := cmd.CombinedOutput(); err != nil {
				r.err = fmt.Errorf("worker %d dep tree: %v\n%s", worker, err, out)
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
}

// TestProxiedServerDepListSingleIDWarnsOnDroppedExternalEdge is the
// proxied-server counterpart to TestEmbeddedDepListSingleIDWarnsOnDroppedExternalEdge
// (dep_list_cross_db_test.go): the regression guard for bd-mtla on the
// runDepListProxiedServer path (modules/cli/dep_proxied_server.go), which gastown's
// live-dolt-server deployment actually runs.
//
// `issuegraph link` correctly writes an edge to an `external:` target (no row in any
// database this session can see), but the single-id `issuegraph dep list <id>` a
// caller runs right after used to show nothing for it — indistinguishable
// from no dependency existing at all. Verifies the fix: stdout/--json stay
// exactly what they were before the fix (no schema change), stderr now names
// the dropped edge, and batch mode (the already-correct path) is unaffected.
func TestProxiedServerDepListSingleIDWarnsOnDroppedExternalEdge(t *testing.T) {
	requireSharedProxiedServer(t)
	t.Parallel()
	issuegraph := buildEmbeddedBD(t)

	p := newSharedProxiedProject(t, issuegraph, "dlw1")
	local := issuegraphProxiedCreate(t, issuegraph, p.dir, "local target", "--type", "task")
	source := issuegraphProxiedCreate(t, issuegraph, p.dir, "source issue", "--type", "task")

	issuegraphProxiedDep(t, issuegraph, p.dir, "add", source.ID, local.ID, "--type", "blocks")

	linkCmd := exec.Command(issuegraph, "link", source.ID, "external:other:capability", "--type", "related")
	linkCmd.Dir = p.dir
	linkCmd.Env = issuegraphProxiedEnv(p.dir)
	if out, err := linkCmd.CombinedOutput(); err != nil {
		t.Fatalf("issuegraph link (external): %v\n%s", err, out)
	}

	cmd := exec.Command(issuegraph, "dep", "list", source.ID)
	cmd.Dir = p.dir
	cmd.Env = issuegraphProxiedEnv(p.dir)
	stdout, stderr, err := runCommandBuffers(t, cmd)
	if err != nil {
		t.Fatalf("issuegraph dep list: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}

	if !strings.Contains(stdout.String(), local.ID) {
		t.Errorf("stdout missing local dep %s:\n%s", local.ID, stdout.String())
	}
	if strings.Contains(stdout.String(), "external:other:capability") {
		t.Errorf("stdout unexpectedly contains the external target (shape must stay unchanged):\n%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "external:other:capability via related") {
		t.Errorf("stderr should warn about the dropped external target with its type, got:\n%s", stderr.String())
	}

	jsonCmd := exec.Command(issuegraph, "dep", "list", source.ID, "--json")
	jsonCmd.Dir = p.dir
	jsonCmd.Env = issuegraphProxiedEnv(p.dir)
	jsonStdout, _, err := runCommandBuffers(t, jsonCmd)
	if err != nil {
		t.Fatalf("issuegraph dep list --json: %v", err)
	}
	if strings.Contains(jsonStdout.String(), "external:other:capability") {
		t.Errorf("--json output unexpectedly contains the external target:\n%s", jsonStdout.String())
	}
	if !strings.Contains(jsonStdout.String(), local.ID) {
		t.Errorf("--json output missing local dep %s:\n%s", local.ID, jsonStdout.String())
	}

	batchCmd := exec.Command(issuegraph, "dep", "list", source.ID, source.ID)
	batchCmd.Dir = p.dir
	batchCmd.Env = issuegraphProxiedEnv(p.dir)
	batchStdout, _, err := runCommandBuffers(t, batchCmd)
	if err != nil {
		t.Fatalf("issuegraph dep list (batch): %v", err)
	}
	if !strings.Contains(batchStdout.String(), "external:other:capability") || !strings.Contains(batchStdout.String(), local.ID) {
		t.Errorf("batch mode should show both deps, got:\n%s", batchStdout.String())
	}
}
