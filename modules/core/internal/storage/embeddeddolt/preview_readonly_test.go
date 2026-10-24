//go:build cgo

package embeddeddolt_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/borg0ai/dx-issuegraph/internal/storage/embeddeddolt"
)

// TestPreviewStoreRefusesLocalMetadataWrite pins the classification the CLI's
// PersistentPostRun tip auto-commit keys on. That block writes tip metadata
// unconditionally — read-only commands have always tolerated it because
// OpenForReadOnlyCommand is deliberately writable — so the write-refusing
// preview store must return an error that errors.Is recognizes, or a
// successful dry-run exits non-zero after the fact.
func TestPreviewStoreRefusesLocalMetadataWrite(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt tests")
	}

	ctx := t.Context()
	issuegraphDir := filepath.Join(t.TempDir(), ".issuegraph")

	store, err := embeddeddolt.Open(ctx, issuegraphDir, "previewdb", "main")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := store.SetLocalMetadata(ctx, "tip_example_last_shown", "2026-01-01T00:00:00Z"); err != nil {
		t.Fatalf("SetLocalMetadata on writable store: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	preview, err := embeddeddolt.OpenForPreviewCommand(ctx, issuegraphDir, "previewdb", "main")
	if err != nil {
		t.Fatalf("OpenForPreviewCommand: %v", err)
	}
	defer func() { _ = preview.Close() }()

	// Reads still work.
	if _, err := preview.GetLocalMetadata(ctx, "tip_example_last_shown"); err != nil {
		t.Fatalf("GetLocalMetadata on preview store: %v", err)
	}

	err = preview.SetLocalMetadata(ctx, "tip_example_last_shown", "2026-02-02T00:00:00Z")
	if err == nil {
		t.Fatal("SetLocalMetadata on preview store = nil error, want refusal")
	}
	if !errors.Is(err, embeddeddolt.ErrReadOnly) {
		t.Fatalf("SetLocalMetadata on preview store = %v, want an error matching embeddeddolt.ErrReadOnly", err)
	}
}
