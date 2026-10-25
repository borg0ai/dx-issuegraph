// memory_contract_test.go — protocol v0 §M (memories).
//
// M1 (model). A memory is a (key, value) pair in a namespace separate from
// issues, with the surface: issuegraph remember / issuegraph memories (list, search) /
// issuegraph recall <key> / issuegraph forget <key>. This file pins that CRUD surface at the
// CLI level — the four verbs, the round trip through the store, and the
// post-forget disappearance.
//
// M4 (reserved namespace). The memory keyspace is reserved to the memory
// commands: a generic KV surface MUST NOT write into it. issuegraph stores memories as
// kv.memory.* rows, so an unbarred `issuegraph kv set memory.x` would mint something
// indistinguishable from a `issuegraph remember` memory — and the merge resolver
// auto-resolves kv.memory.* conflicts with --theirs (GH#2474), so a remote pull
// could silently overwrite the operator's deliberate KV value. The bar is what
// keeps the namespace owned.
package protocol

import (
	"strings"
	"testing"
)

// TestProtocol_MemoryCRUDSurface pins §M1: the remember → recall → list →
// search → forget cycle over the frozen CLI surface.
func TestProtocol_MemoryCRUDSurface(t *testing.T) {
	t.Parallel()
	w := newWorkspace(t)

	const (
		key  = "auth-jwt"
		text = "auth module uses JWT not sessions"
	)

	// Create.
	w.run("remember", text, "--key", key)

	// Read by key.
	recalled := w.run("recall", key)
	if !strings.Contains(recalled, text) {
		t.Errorf("issuegraph recall %s did not return the stored value (§M1):\n%s", key, recalled)
	}

	// List.
	list := w.run("memories")
	if !strings.Contains(list, key) {
		t.Errorf("issuegraph memories does not list the stored key (§M1):\n%s", list)
	}

	// Search (issuegraph memories <term>).
	found := w.run("memories", "JWT")
	if !strings.Contains(found, text) {
		t.Errorf("issuegraph memories JWT did not match the stored memory (§M1):\n%s", found)
	}

	// Update in place: same key, new value.
	const updated = "auth module uses JWT with a 15m TTL"
	w.run("remember", updated, "--key", key)
	recalled = w.run("recall", key)
	if !strings.Contains(recalled, updated) {
		t.Errorf("issuegraph remember --key did not overwrite in place (§M1):\n%s", recalled)
	}

	// Delete.
	w.run("forget", key)
	if _, code := w.runExpectError("recall", key); code == 0 {
		t.Errorf("issuegraph recall after forget exited 0, want nonzero (§M1/§E2)")
	}
	if list := w.run("memories"); strings.Contains(list, key) {
		t.Errorf("forgotten key still listed by issuegraph memories (§M1):\n%s", list)
	}
}

// TestProtocol_MemoryReservedNamespace pins §M4: the generic KV surface must
// refuse to write into the memory namespace, and the refusal must point at the
// commands that own it.
func TestProtocol_MemoryReservedNamespace(t *testing.T) {
	t.Parallel()
	w := newWorkspace(t)

	// Control: an ordinary KV key is accepted, so the refusal below is about the
	// namespace and not about `issuegraph kv set` being broken.
	w.run("kv", "set", "deploy-target", "staging")
	if got := w.run("kv", "get", "deploy-target"); !strings.Contains(got, "staging") {
		t.Fatalf("issuegraph kv set/get round trip failed for an ordinary key:\n%s", got)
	}

	for _, key := range []string{"memory.auth-jwt", "memory.nested.key"} {
		out, code := w.runExpectError("kv", "set", key, "smuggled")
		if code == 0 {
			t.Errorf("issuegraph kv set %s exited 0, want refusal (§M4)", key)
		}
		lower := strings.ToLower(out)
		if !strings.Contains(lower, "reserved") {
			t.Errorf("issuegraph kv set %s: refusal does not say the namespace is reserved (§M4):\n%s", key, out)
		}
		if !strings.Contains(lower, "issuegraph remember") {
			t.Errorf("issuegraph kv set %s: refusal does not name the owning command (§M4):\n%s", key, out)
		}
	}

	// The bar must not have leaked a value into the memory namespace.
	if out := w.run("memories"); strings.Contains(out, "smuggled") {
		t.Errorf("a refused kv set still reached the memory namespace (§M4):\n%s", out)
	}
}
