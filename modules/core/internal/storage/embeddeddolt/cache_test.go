//go:build cgo

package embeddeddolt_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/borg0ai/dx-issuegraph/internal/storage/embeddeddolt"
)

func TestOpenReturnsCachedStore(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt tests")
	}
	ctx := t.Context()
	issuegraphDir := filepath.Join(t.TempDir(), ".issuegraph")

	// First Open creates a new store.
	store1, err := embeddeddolt.Open(ctx, issuegraphDir, "testdb", "main")
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}

	// Second Open with same issuegraphDir returns the same pointer.
	store2, err := embeddeddolt.Open(ctx, issuegraphDir, "testdb", "main")
	if err != nil {
		t.Fatalf("second Open: %v", err)
	}

	if store1 != store2 {
		t.Fatal("expected second Open to return the same store instance")
	}

	// First Close is a no-op (refcount > 0).
	if err := store1.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}

	// Store should still be usable via store2 — GetIssue exercises withConn.
	_, getErr := store2.GetIssue(t.Context(), "nonexistent")
	if getErr != nil && getErr.Error() != "not found: issue nonexistent" {
		t.Fatalf("GetIssue after first Close should work (got: %v)", getErr)
	}

	// Second Close actually releases the store.
	if err := store2.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

func TestOpenDifferentDirsReturnsDifferentStores(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt tests")
	}
	ctx := t.Context()
	issuegraphDir1 := filepath.Join(t.TempDir(), ".issuegraph")
	issuegraphDir2 := filepath.Join(t.TempDir(), ".issuegraph")

	store1, err := embeddeddolt.Open(ctx, issuegraphDir1, "testdb", "main")
	if err != nil {
		t.Fatalf("Open dir1: %v", err)
	}
	defer store1.Close()

	store2, err := embeddeddolt.Open(ctx, issuegraphDir2, "testdb", "main")
	if err != nil {
		t.Fatalf("Open dir2: %v", err)
	}
	defer store2.Close()

	if store1 == store2 {
		t.Fatal("expected different stores for different directories")
	}
}

func TestOpenAfterCloseCreatesNewStore(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt tests")
	}
	ctx := t.Context()
	issuegraphDir := filepath.Join(t.TempDir(), ".issuegraph")

	store1, err := embeddeddolt.Open(ctx, issuegraphDir, "testdb", "main")
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}

	// Close releases the last reference — evicts from cache.
	store1.Close()

	// Re-opening should create a fresh store, not return the closed one.
	store2, err := embeddeddolt.Open(ctx, issuegraphDir, "testdb", "main")
	if err != nil {
		t.Fatalf("Open after Close: %v", err)
	}
	defer store2.Close()

	if store1 == store2 {
		t.Fatal("expected a new store after full close, got the same pointer")
	}
}
