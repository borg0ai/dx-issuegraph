package backend_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/borg0ai/dx-issuegraph/backend"
	"github.com/borg0ai/dx-issuegraph/internal/storage"
)

func TestErrCommitIndeterminateAliasesStorageSentinel(t *testing.T) {
	if backend.ErrCommitIndeterminate != storage.ErrCommitIndeterminate {
		t.Fatal("backend ErrCommitIndeterminate must preserve storage sentinel identity")
	}

	err := fmt.Errorf("commit: %w", backend.ErrCommitIndeterminate)
	if !errors.Is(err, storage.ErrCommitIndeterminate) {
		t.Fatalf("errors.Is(err, storage.ErrCommitIndeterminate) = false; err = %v", err)
	}
}
