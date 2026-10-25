//go:build cgo

package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func issuegraphProxiedKV(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"kv"}, args...)
	stdout, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, dir, fullArgs...)
	if err != nil {
		t.Fatalf("issuegraph kv %s failed: %v\nstdout:\n%s\nstderr:\n%s",
			strings.Join(args, " "), err, stdout, stderr)
	}
	return stdout
}

func issuegraphProxiedKVFail(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"kv"}, args...)
	stdout, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, dir, fullArgs...)
	if err == nil {
		t.Fatalf("expected issuegraph kv %s to fail; got stdout:\n%s\nstderr:\n%s",
			strings.Join(args, " "), stdout, stderr)
	}
	return stdout + stderr
}

// kvListJSONRaw runs `issuegraph kv list --json` and returns the raw JSON object
// text plus the parsed pairs. It pins the wire shape the wyvern wheelhouse
// mail transport parses: a flat JSON object whose only non-string member is
// the outputJSON-injected numeric "schema_version" (present in classic mode
// too — see issuegraphKVListJSON in kv_embedded_test.go); every kv pair is a plain
// string-to-string entry with no envelope.
func kvListJSONRaw(t *testing.T, issuegraph, dir string) (string, map[string]string) {
	t.Helper()
	stdout, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, dir, "kv", "list", "--json")
	if err != nil {
		t.Fatalf("issuegraph kv list --json failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	s := strings.TrimSpace(stdout)
	start := strings.Index(s, "{")
	if start < 0 {
		t.Fatalf("no JSON object in kv list output: %s", s)
	}
	raw := s[start:]
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		t.Fatalf("kv list --json did not parse: %v\n%s", err, raw)
	}
	sv, ok := parsed["schema_version"]
	if !ok {
		t.Fatalf("kv list --json missing schema_version (shape drift): %s", raw)
	}
	if _, isNum := sv.(float64); !isNum {
		t.Fatalf("kv list --json schema_version is not numeric (shape drift): %T %v\n%s", sv, sv, raw)
	}
	m := make(map[string]string, len(parsed))
	for k, v := range parsed {
		if k == "schema_version" {
			continue
		}
		sv, ok := v.(string)
		if !ok {
			t.Fatalf("kv list --json pair %q is not a string (shape drift): %v\n%s", k, v, raw)
		}
		m[k] = sv
	}
	return raw, m
}

// mailEnvelope is a representative wheelhouse mail payload: one argv element
// containing spaces, newlines, and JSON punctuation. kv is the constellation's
// mail transport, so this must round-trip byte-for-byte.
const mailEnvelope = "{\n  \"to\": \"marshal\",\n  \"subject\": \"fleet status\",\n  \"body\": \"line one\nline two with  double spaces\n\ttabbed line\"\n}"

func TestProxiedServerKV(t *testing.T) {
	requireSharedProxiedServer(t)
	t.Parallel()
	issuegraph := buildEmbeddedBD(t)

	t.Run("set_get_roundtrip", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "pkv")

		issuegraphProxiedKV(t, issuegraph, p.dir, "set", "mykey", "myvalue")
		out := issuegraphProxiedKV(t, issuegraph, p.dir, "get", "mykey")
		if out != "myvalue\n" {
			t.Errorf("get after set: expected %q, got %q", "myvalue\n", out)
		}

		// Overwrite
		issuegraphProxiedKV(t, issuegraph, p.dir, "set", "mykey", "second")
		out = issuegraphProxiedKV(t, issuegraph, p.dir, "get", "mykey")
		if out != "second\n" {
			t.Errorf("get after overwrite: expected %q, got %q", "second\n", out)
		}
	})

	t.Run("multiline_json_envelope_roundtrip", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "pkvm")

		// The whole envelope is ONE argv element; no trimming or re-encoding
		// may happen anywhere on the path.
		issuegraphProxiedKV(t, issuegraph, p.dir, "set", "mail/inbox/marshal", mailEnvelope)

		out := issuegraphProxiedKV(t, issuegraph, p.dir, "get", "mail/inbox/marshal")
		if out != mailEnvelope+"\n" {
			t.Errorf("envelope did not round-trip byte-for-byte:\nwant: %q\ngot:  %q", mailEnvelope+"\n", out)
		}

		// The same bytes must come back through list --json.
		_, m := kvListJSONRaw(t, issuegraph, p.dir)
		if got, ok := m["mail/inbox/marshal"]; !ok || got != mailEnvelope {
			t.Errorf("list --json envelope mismatch (exists=%v):\nwant: %q\ngot:  %q", ok, mailEnvelope, got)
		}
	})

	t.Run("list_json_shape", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "pkvl")

		issuegraphProxiedKV(t, issuegraph, p.dir, "set", "alpha", "1")
		issuegraphProxiedKV(t, issuegraph, p.dir, "set", "beta", "two words")

		raw, m := kvListJSONRaw(t, issuegraph, p.dir)
		want := map[string]string{"alpha": "1", "beta": "two words"}
		if len(m) != len(want) {
			t.Errorf("expected exactly %d kv pairs, got %d: %s", len(want), len(m), raw)
		}
		for k, v := range want {
			if got, ok := m[k]; !ok || got != v {
				t.Errorf("list --json: %s expected %q, got %q (exists=%v)", k, v, got, ok)
			}
		}
		// Keys are user keys — the storage prefix must be stripped.
		for k := range m {
			if strings.HasPrefix(k, "kv.") {
				t.Errorf("list --json leaked storage prefix on key %q", k)
			}
		}
	})

	t.Run("list_json_shape_matches_classic", func(t *testing.T) {
		t.Parallel()
		// Same binary, same commands, classic embedded project: the JSON
		// documents (parsed) must be identical — a mail client parses this.
		proxied := newSharedProxiedProject(t, issuegraph, "pkvc")
		classicDir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "kvc")

		for _, kvp := range [][2]string{
			{"plain", "value"},
			{"envelope", mailEnvelope},
		} {
			issuegraphProxiedKV(t, issuegraph, proxied.dir, "set", kvp[0], kvp[1])
			issuegraphKV(t, issuegraph, classicDir, "set", kvp[0], kvp[1])
		}

		_, proxiedMap := kvListJSONRaw(t, issuegraph, proxied.dir)
		classicMap := issuegraphKVListJSON(t, issuegraph, classicDir)

		if len(proxiedMap) != len(classicMap) {
			t.Fatalf("map size mismatch: proxied=%v classic=%v", proxiedMap, classicMap)
		}
		for k, v := range classicMap {
			if got, ok := proxiedMap[k]; !ok || got != v {
				t.Errorf("key %q: classic %q, proxied %q (exists=%v)", k, v, got, ok)
			}
		}
	})

	t.Run("get_missing_key", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "pkvg")
		out := issuegraphProxiedKVFail(t, issuegraph, p.dir, "get", "never_set_key")
		if !strings.Contains(out, "not set") {
			t.Errorf("expected 'not set' for missing key, got: %s", out)
		}
	})

	t.Run("clear", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "pkvd")

		issuegraphProxiedKV(t, issuegraph, p.dir, "set", "clearme", "temporary")
		out := issuegraphProxiedKV(t, issuegraph, p.dir, "clear", "clearme")
		if !strings.Contains(out, "Cleared clearme") {
			t.Errorf("expected 'Cleared clearme', got: %s", out)
		}

		issuegraphProxiedKVFail(t, issuegraph, p.dir, "get", "clearme")
		_, m := kvListJSONRaw(t, issuegraph, p.dir)
		if _, ok := m["clearme"]; ok {
			t.Error("expected clearme absent from kv list after clear")
		}

		// Idempotent-friendly: clearing an already-absent key succeeds,
		// exactly like the classic path (DELETE of a missing row is a no-op
		// and the empty tx commits as nothing-to-commit).
		out = issuegraphProxiedKV(t, issuegraph, p.dir, "clear", "clearme")
		if !strings.Contains(out, "Cleared clearme") {
			t.Errorf("expected idempotent 'Cleared clearme' on second clear, got: %s", out)
		}
	})

	t.Run("invalid_keys_refused", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "pkvi")

		out := issuegraphProxiedKVFail(t, issuegraph, p.dir, "set", "kv.nested", "v")
		if !strings.Contains(out, "nested prefix") {
			t.Errorf("expected nested-prefix refusal, got: %s", out)
		}
		out = issuegraphProxiedKVFail(t, issuegraph, p.dir, "set", "memory.sneaky", "v")
		if !strings.Contains(out, "reserved for persistent memories") {
			t.Errorf("expected memory-namespace refusal, got: %s", out)
		}
		out = issuegraphProxiedKVFail(t, issuegraph, p.dir, "set", "sync.mode", "v")
		if !strings.Contains(out, "reserved prefix") {
			t.Errorf("expected reserved-prefix refusal, got: %s", out)
		}
	})

	t.Run("set_creates_dolt_commit", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "pkvh")
		db := openProxiedDB(t, p)
		before := proxiedDoltHead(t, db)

		issuegraphProxiedKV(t, issuegraph, p.dir, "set", "committed", "v1")

		after := proxiedDoltHead(t, db)
		if after == before {
			t.Errorf("HEAD did not advance on kv set: before=%s after=%s", before, after)
		}
		if n := proxiedDoltCommitCountSince(t, db, before); n != 1 {
			t.Errorf("expected exactly 1 Dolt commit for kv set, got %d", n)
		}
	})
}
