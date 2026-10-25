//go:build cgo

package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func snapshotListIDs(t *testing.T, issuegraph, dir string) []string {
	t.Helper()
	var ids []string
	for _, issue := range issuegraphListJSON(t, issuegraph, dir, "--all") {
		ids = append(ids, issue.ID)
	}
	sort.Strings(ids)
	return ids
}

func snapshotManifestIDs(t *testing.T, dir string) []string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, ".issuegraph", snapshotManifestFile))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var m snapshotManifest
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	return m.IDs
}

func copySnapshot(t *testing.T, from, to string) {
	t.Helper()
	for _, name := range []string{"issues.jsonl", snapshotManifestFile} {
		data, err := os.ReadFile(filepath.Join(from, ".issuegraph", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(to, ".issuegraph", name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestEmbeddedSnapshot(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt snapshot tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)

	t.Run("snapshot_tracks_every_write_including_delete", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "snw")
		a := issuegraphCreateSilent(t, issuegraph, dir, "first")
		b := issuegraphCreateSilent(t, issuegraph, dir, "second")

		got := snapshotManifestIDs(t, dir)
		want := []string{a, b}
		sort.Strings(want)
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Fatalf("manifest ids = %v, want %v", got, want)
		}

		issuegraphDelete(t, issuegraph, dir, "--force", a)
		got = snapshotManifestIDs(t, dir)
		if len(got) != 1 || got[0] != b {
			t.Fatalf("after delete manifest ids = %v, want [%s]", got, b)
		}
	})

	t.Run("restore_makes_database_equal_snapshot_and_deletes_extras", func(t *testing.T) {
		src, _, _ := issuegraphInit(t, issuegraph, "--prefix", "snr")
		issuegraphCreateSilent(t, issuegraph, src, "keep one")
		doomed := issuegraphCreateSilent(t, issuegraph, src, "will be deleted")
		issuegraphCreateSilent(t, issuegraph, src, "keep two")
		issuegraphDelete(t, issuegraph, src, "--force", doomed)
		want := snapshotListIDs(t, issuegraph, src)

		dst, _, _ := issuegraphInit(t, issuegraph, "--prefix", "snr")
		extra := issuegraphCreateSilent(t, issuegraph, dst, "local only extra")
		copySnapshot(t, src, dst)

		out, err := issuegraphRunWithFlockRetry(t, issuegraph, dst, "snapshot", "restore")
		if err != nil {
			t.Fatalf("snapshot restore failed: %v\n%s", err, out)
		}
		got := snapshotListIDs(t, issuegraph, dst)
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Fatalf("restored ids = %v, want %v", got, want)
		}
		for _, id := range got {
			if id == extra || id == doomed {
				t.Fatalf("id %s must not exist after restore", id)
			}
		}
	})

	t.Run("invalid_snapshot_fails_and_leaves_database_unchanged", func(t *testing.T) {
		src, _, _ := issuegraphInit(t, issuegraph, "--prefix", "snf")
		issuegraphCreateSilent(t, issuegraph, src, "source issue")
		dst, _, _ := issuegraphInit(t, issuegraph, "--prefix", "snf")
		issuegraphCreateSilent(t, issuegraph, dst, "must survive")
		before := snapshotListIDs(t, issuegraph, dst)

		cases := map[string]func(){
			"missing manifest": func() {
				copySnapshot(t, src, dst)
				_ = os.Remove(filepath.Join(dst, ".issuegraph", snapshotManifestFile))
			},
			"corrupt manifest": func() {
				copySnapshot(t, src, dst)
				_ = os.WriteFile(filepath.Join(dst, ".issuegraph", snapshotManifestFile), []byte("{broken"), 0o644)
			},
			"digest mismatch": func() {
				copySnapshot(t, src, dst)
				f, err := os.OpenFile(filepath.Join(dst, ".issuegraph", "issues.jsonl"), os.O_APPEND|os.O_WRONLY, 0o644)
				if err != nil {
					t.Fatal(err)
				}
				_, _ = f.WriteString(`{"_type":"issue","id":"snf-zzz","title":"tampered"}` + "\n")
				_ = f.Close()
			},
		}
		for name, setup := range cases {
			setup()
			if out, err := issuegraphRunWithFlockRetry(t, issuegraph, dst, "snapshot", "restore"); err == nil {
				t.Fatalf("%s: restore succeeded, want failure\n%s", name, out)
			}
			got := snapshotListIDs(t, issuegraph, dst)
			if strings.Join(got, ",") != strings.Join(before, ",") {
				t.Fatalf("%s: database changed to %v, want %v", name, got, before)
			}
		}
	})

	t.Run("snapshot_flow_needs_no_git", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "sng")
		issuegraphCreateSilent(t, issuegraph, dir, "one")
		noGit := t.TempDir()

		cmd := exec.Command(issuegraph, "create", "--silent", "no git on PATH")
		cmd.Dir = dir
		cmd.Env = append(issuegraphEnv(dir), "PATH="+noGit)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("create without git failed: %v\n%s", err, out)
		}
		if got := snapshotManifestIDs(t, dir); len(got) != 2 {
			t.Fatalf("manifest ids = %v, want 2 after a write without git", got)
		}

		cmd = exec.Command(issuegraph, "snapshot", "restore")
		cmd.Dir = dir
		cmd.Env = append(issuegraphEnv(dir), "PATH="+noGit)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("restore without git failed: %v\n%s", err, out)
		}
	})
}
