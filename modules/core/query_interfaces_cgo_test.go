//go:build cgo

package issuegraph_test

import (
	"github.com/borg0ai/dx-issuegraph"
	"github.com/borg0ai/dx-issuegraph/internal/storage/embeddeddolt"
)

// Compile-time proof that the embedded Dolt store also satisfies each narrow
// public interface (the server Dolt store is asserted in query_interfaces_test.go).
var (
	_ issuegraph.IssueClaimer     = (*embeddeddolt.EmbeddedDoltStore)(nil)
	_ issuegraph.EventQuerier     = (*embeddeddolt.EmbeddedDoltStore)(nil)
	_ issuegraph.DependentQuerier = (*embeddeddolt.EmbeddedDoltStore)(nil)
	_ issuegraph.BlockedQuerier   = (*embeddeddolt.EmbeddedDoltStore)(nil)
)
