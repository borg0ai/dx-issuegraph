package audit

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestAppend_CreatesFileAndWritesJSONL(t *testing.T) {
	tmp := t.TempDir()
	issuegraphDir := filepath.Join(tmp, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// issuegraph.FindIssueGraphDir() validates that the directory contains project files.
	// Create metadata.json so ISSUEGRAPH_DIR is accepted by hasIssueGraphProjectFiles.
	metadataPath := filepath.Join(issuegraphDir, "metadata.json")
	if err := os.WriteFile(metadataPath, []byte(`{"backend":"dolt"}`), 0644); err != nil {
		t.Fatalf("write metadata.json: %v", err)
	}
	t.Setenv("ISSUEGRAPH_DIR", issuegraphDir)
	t.Setenv("ISSUEGRAPH_AUDIT_ENABLED", "1")

	id1, err := Append(&Entry{Kind: "llm_call", Model: "test-model", Prompt: "p", Response: "r"})
	if err != nil {
		t.Fatalf("append: %v", err)
	}
	if id1 == "" {
		t.Fatalf("expected id")
	}
	_, err = Append(&Entry{Kind: "label", ParentID: id1, Label: "good", Reason: "ok"})
	if err != nil {
		t.Fatalf("append label: %v", err)
	}

	p := filepath.Join(issuegraphDir, FileName)
	f, err := os.Open(p)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = f.Close() }()

	sc := bufio.NewScanner(f)
	lines := 0
	for sc.Scan() {
		lines++
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if lines != 2 {
		t.Fatalf("expected 2 lines, got %d", lines)
	}
}

func TestAppendIfEnabled_DisabledDoesNotCreateFile(t *testing.T) {
	tmp := t.TempDir()
	issuegraphDir := filepath.Join(tmp, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	metadataPath := filepath.Join(issuegraphDir, "metadata.json")
	if err := os.WriteFile(metadataPath, []byte(`{"backend":"dolt"}`), 0644); err != nil {
		t.Fatalf("write metadata.json: %v", err)
	}
	t.Setenv("ISSUEGRAPH_DIR", issuegraphDir)
	t.Setenv("ISSUEGRAPH_AUDIT_ENABLED", "")

	if _, err := AppendIfEnabled(&Entry{Kind: "llm_call", Model: "test-model"}); err == nil {
		t.Fatalf("expected disabled append to fail")
	}

	if _, err := os.Stat(filepath.Join(issuegraphDir, FileName)); !os.IsNotExist(err) {
		t.Fatalf("disabled append created %s: %v", FileName, err)
	}
}

func TestEnsureFile_DoesNotTruncateWhenConcurrentCreatorWins(t *testing.T) {
	tmp := t.TempDir()
	issuegraphDir := filepath.Join(tmp, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	metadataPath := filepath.Join(issuegraphDir, "metadata.json")
	if err := os.WriteFile(metadataPath, []byte(`{"backend":"dolt"}`), 0644); err != nil {
		t.Fatalf("write metadata.json: %v", err)
	}
	t.Setenv("ISSUEGRAPH_DIR", issuegraphDir)

	wantContents := []byte("{\"id\":\"seed\"}\n")

	oldHook := ensureFileBeforeCreateHook
	defer func() { ensureFileBeforeCreateHook = oldHook }()

	var once sync.Once
	ensureFileBeforeCreateHook = func(path string) {
		once.Do(func() {
			if err := os.WriteFile(path, wantContents, 0644); err != nil {
				t.Fatalf("seed interactions log: %v", err)
			}
		})
	}

	gotPath, err := EnsureFile()
	if err != nil {
		t.Fatalf("EnsureFile: %v", err)
	}

	gotContents, err := os.ReadFile(gotPath)
	if err != nil {
		t.Fatalf("read interactions log: %v", err)
	}
	if !bytes.Equal(gotContents, wantContents) {
		t.Fatalf("EnsureFile truncated seeded content: got %q, want %q", gotContents, wantContents)
	}
}
