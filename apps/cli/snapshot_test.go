package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/borg0ai/dx-issuegraph/internal/config"
	"github.com/borg0ai/dx-issuegraph/internal/storage"
	"github.com/borg0ai/dx-issuegraph/internal/types"
)

const snapshotTestJSONL = `{"_type":"issue","id":"t-2","title":"second","status":"open","priority":2,"issue_type":"task"}
{"_type":"issue","id":"t-1","title":"first","status":"open","priority":2,"issue_type":"task"}
`

func writeSnapshotFiles(t *testing.T, jsonl string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "issues.jsonl")
	if err := os.WriteFile(path, []byte(jsonl), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeSnapshotManifest(path); err != nil {
		t.Fatalf("writeSnapshotManifest: %v", err)
	}
	return path
}

func mutateManifest(t *testing.T, path string, fn func(*snapshotManifest)) {
	t.Helper()
	mp := filepath.Join(filepath.Dir(path), snapshotManifestFile)
	raw, err := os.ReadFile(mp)
	if err != nil {
		t.Fatal(err)
	}
	var m snapshotManifest
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	fn(&m)
	out, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mp, out, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSnapshotManifestListsSortedCompleteIDSet(t *testing.T) {
	path := writeSnapshotFiles(t, snapshotTestJSONL)
	manifest, issues, err := readSnapshot(path)
	if err != nil {
		t.Fatalf("readSnapshot: %v", err)
	}
	if manifest.FormatVersion != snapshotFormatVersion || manifest.Count != 2 || len(issues) != 2 {
		t.Fatalf("unexpected manifest %+v with %d issues", manifest, len(issues))
	}
	if strings.Join(manifest.IDs, ",") != "t-1,t-2" {
		t.Fatalf("ids = %v, want sorted [t-1 t-2]", manifest.IDs)
	}
}

func TestSnapshotManifestIsRewrittenToMatchDeletion(t *testing.T) {
	path := writeSnapshotFiles(t, snapshotTestJSONL)
	only := `{"_type":"issue","id":"t-1","title":"first","status":"open","priority":2,"issue_type":"task"}` + "\n"
	if err := os.WriteFile(path, []byte(only), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeSnapshotManifest(path); err != nil {
		t.Fatal(err)
	}
	manifest, _, err := readSnapshot(path)
	if err != nil {
		t.Fatalf("readSnapshot: %v", err)
	}
	if manifest.Count != 1 || manifest.IDs[0] != "t-1" {
		t.Fatalf("deleted id still listed: %+v", manifest)
	}
}

func TestReadSnapshotRejectsInvalidSnapshots(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(t *testing.T, path string)
		wantErr string
	}{
		{"missing manifest", func(t *testing.T, path string) {
			if err := os.Remove(filepath.Join(filepath.Dir(path), snapshotManifestFile)); err != nil {
				t.Fatal(err)
			}
		}, "manifest"},
		{"corrupt manifest", func(t *testing.T, path string) {
			if err := os.WriteFile(filepath.Join(filepath.Dir(path), snapshotManifestFile), []byte("{not json"), 0o644); err != nil {
				t.Fatal(err)
			}
		}, "parse snapshot manifest"},
		{"tampered issues file", func(t *testing.T, path string) {
			extra := snapshotTestJSONL + `{"_type":"issue","id":"t-3","title":"extra"}` + "\n"
			if err := os.WriteFile(path, []byte(extra), 0o644); err != nil {
				t.Fatal(err)
			}
		}, "digest"},
		{"corrupt issues file", func(t *testing.T, path string) {
			if err := os.WriteFile(path, []byte("{broken\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}, "corrupt"},
		{"unsupported format version", func(t *testing.T, path string) {
			mutateManifest(t, path, func(m *snapshotManifest) { m.FormatVersion = 99 })
		}, "format version"},
		{"count disagrees", func(t *testing.T, path string) {
			mutateManifest(t, path, func(m *snapshotManifest) { m.Count = 5 })
		}, "count"},
		{"id list disagrees", func(t *testing.T, path string) {
			mutateManifest(t, path, func(m *snapshotManifest) { m.IDs = []string{"t-1", "t-9"} })
		}, "id list"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := writeSnapshotFiles(t, snapshotTestJSONL)
			tc.mutate(t, path)
			_, issues, err := readSnapshot(path)
			if err == nil {
				t.Fatalf("expected an error, got %d issues", len(issues))
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error %q does not mention %q", err, tc.wantErr)
			}
			if issues != nil {
				t.Fatal("invalid snapshot must not return issues")
			}
		})
	}
}

func TestBuildSnapshotManifestRejectsDuplicateIDs(t *testing.T) {
	dup := `{"_type":"issue","id":"t-1","title":"a"}` + "\n" + `{"_type":"issue","id":"t-1","title":"b"}` + "\n"
	if _, _, err := buildSnapshotManifest([]byte(dup)); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("expected duplicate id error, got %v", err)
	}
}

func TestWriteSnapshotManifestLeavesNoTempFile(t *testing.T) {
	path := writeSnapshotFiles(t, snapshotTestJSONL)
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() != "issues.jsonl" && e.Name() != snapshotManifestFile {
			t.Fatalf("unexpected leftover file %q after atomic write", e.Name())
		}
	}
}

// fakeSnapshotTx records what restore does inside the transaction. Embedding the
// interface leaves every method restore must not call as a nil-pointer panic.
type fakeSnapshotTx struct {
	storage.Transaction
	local     []string
	created   []*types.Issue
	deleted   []string
	createErr error
	searchErr error
	deleteErr error
}

func (f *fakeSnapshotTx) CreateIssues(_ context.Context, issues []*types.Issue, _ string) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.created = append(f.created, issues...)
	return nil
}

func (f *fakeSnapshotTx) SearchIssueIDs(context.Context, string, types.IssueFilter) ([]string, error) {
	return f.local, f.searchErr
}

func (f *fakeSnapshotTx) DeleteIssue(_ context.Context, id string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	f.deleted = append(f.deleted, id)
	return nil
}

type fakeSnapshotRunner struct {
	tx    *fakeSnapshotTx
	calls int
}

func (r *fakeSnapshotRunner) RunInTransaction(_ context.Context, _ string, fn func(storage.Transaction) error) error {
	r.calls++
	return fn(r.tx)
}

func snapshotFixture(t *testing.T, jsonl string) (*snapshotManifest, []*types.Issue) {
	t.Helper()
	path := writeSnapshotFiles(t, jsonl)
	manifest, issues, err := readSnapshot(path)
	if err != nil {
		t.Fatal(err)
	}
	return manifest, issues
}

func TestRestoreSnapshotUpsertsAllAndDeletesOnlyExtras(t *testing.T) {
	manifest, issues := snapshotFixture(t, snapshotTestJSONL)
	runner := &fakeSnapshotRunner{tx: &fakeSnapshotTx{local: []string{"t-1", "t-9", "t-2", "t-8"}}}

	upserted, deleted, err := restoreSnapshot(context.Background(), runner, manifest, issues, types.IssueFilter{}, "tester")
	if err != nil {
		t.Fatalf("restoreSnapshot: %v", err)
	}
	if upserted != 2 || deleted != 2 {
		t.Fatalf("upserted=%d deleted=%d, want 2 and 2", upserted, deleted)
	}
	if got := strings.Join(runner.tx.deleted, ","); got != "t-9,t-8" {
		t.Fatalf("deleted = %q, want only ids absent from the manifest", got)
	}
	if len(runner.tx.created) != 2 || runner.calls != 1 {
		t.Fatalf("created=%d transactions=%d, want 2 issues in one transaction", len(runner.tx.created), runner.calls)
	}
}

func TestRestoreSnapshotEmptySnapshotDeletesEveryLocalIssue(t *testing.T) {
	manifest := &snapshotManifest{FormatVersion: snapshotFormatVersion}
	runner := &fakeSnapshotRunner{tx: &fakeSnapshotTx{local: []string{"t-1", "t-2"}}}

	upserted, deleted, err := restoreSnapshot(context.Background(), runner, manifest, nil, types.IssueFilter{}, "tester")
	if err != nil {
		t.Fatal(err)
	}
	if upserted != 0 || deleted != 2 || len(runner.tx.created) != 0 {
		t.Fatalf("upserted=%d deleted=%d created=%d, want 0, 2, 0", upserted, deleted, len(runner.tx.created))
	}
}

func TestRestoreSnapshotPropagatesTransactionErrors(t *testing.T) {
	manifest, issues := snapshotFixture(t, snapshotTestJSONL)
	boom := errors.New("boom")
	cases := map[string]*fakeSnapshotTx{
		"create": {createErr: boom},
		"search": {searchErr: boom},
		"delete": {local: []string{"t-9"}, deleteErr: boom},
	}
	for name, tx := range cases {
		t.Run(name, func(t *testing.T) {
			_, _, err := restoreSnapshot(context.Background(), &fakeSnapshotRunner{tx: tx}, manifest, issues, types.IssueFilter{}, "tester")
			if !errors.Is(err, boom) {
				t.Fatalf("error %v does not wrap the transaction failure", err)
			}
		})
	}
}

func TestRunSnapshotRestoreValidatesBeforeTouchingTheStore(t *testing.T) {
	path := writeSnapshotFiles(t, snapshotTestJSONL)
	dir := filepath.Dir(path)
	if err := os.Remove(filepath.Join(dir, snapshotManifestFile)); err != nil {
		t.Fatal(err)
	}
	runner := &fakeSnapshotRunner{tx: &fakeSnapshotTx{local: []string{"t-9"}}}

	var out bytes.Buffer
	if err := runSnapshotRestore(context.Background(), runner, dir, types.IssueFilter{}, "tester", &out); err == nil {
		t.Fatal("expected a missing-manifest error")
	}
	if runner.calls != 0 {
		t.Fatalf("store was used %d time(s) for an invalid snapshot", runner.calls)
	}
}

func TestRunSnapshotRestoreReportsCounts(t *testing.T) {
	path := writeSnapshotFiles(t, snapshotTestJSONL)
	runner := &fakeSnapshotRunner{tx: &fakeSnapshotTx{local: []string{"t-1", "t-9"}}}

	var out bytes.Buffer
	if err := runSnapshotRestore(context.Background(), runner, filepath.Dir(path), types.IssueFilter{}, "tester", &out); err != nil {
		t.Fatalf("runSnapshotRestore: %v", err)
	}
	if !strings.Contains(out.String(), "2 issue(s) upserted, 1 deleted") {
		t.Fatalf("unexpected report %q", out.String())
	}
}

func TestRunSnapshotRestorePropagatesStoreFailure(t *testing.T) {
	path := writeSnapshotFiles(t, snapshotTestJSONL)
	runner := &fakeSnapshotRunner{tx: &fakeSnapshotTx{createErr: errors.New("db down")}}

	var out bytes.Buffer
	err := runSnapshotRestore(context.Background(), runner, filepath.Dir(path), types.IssueFilter{}, "tester", &out)
	if err == nil || !strings.Contains(err.Error(), "db down") {
		t.Fatalf("expected the store failure, got %v", err)
	}
	if out.Len() != 0 {
		t.Fatalf("nothing should be reported on failure, got %q", out.String())
	}
}

func TestSnapshotIssuesFileFollowsExportPath(t *testing.T) {
	initConfigForTest(t)
	if got := snapshotIssuesFile(); got != "issues.jsonl" {
		t.Fatalf("default snapshot file = %q, want issues.jsonl", got)
	}
	config.Set("export.path", "custom.jsonl")
	if got := snapshotIssuesFile(); got != "custom.jsonl" {
		t.Fatalf("snapshot file = %q, want custom.jsonl", got)
	}
	config.Set("export.path", "")
	if got := snapshotIssuesFile(); got != "issues.jsonl" {
		t.Fatalf("empty export.path must fall back, got %q", got)
	}
}

func TestWriteSnapshotManifestRejectsUnusableInput(t *testing.T) {
	dir := t.TempDir()
	if err := writeSnapshotManifest(filepath.Join(dir, "absent.jsonl")); err == nil {
		t.Fatal("expected an error for a missing JSONL")
	}
	bad := filepath.Join(dir, "issues.jsonl")
	if err := os.WriteFile(bad, []byte("{broken\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeSnapshotManifest(bad); err == nil {
		t.Fatal("expected an error for unparseable JSONL")
	}
	if _, err := os.Stat(filepath.Join(dir, snapshotManifestFile)); !os.IsNotExist(err) {
		t.Fatalf("a failed build must not leave a manifest, stat err=%v", err)
	}
}
