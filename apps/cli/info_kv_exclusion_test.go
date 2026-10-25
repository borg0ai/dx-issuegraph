package main

import (
	"strings"
	"testing"

	"github.com/borg0ai/dx-issuegraph/internal/storage/kvkeys"
	"github.com/borg0ai/dx-issuegraph/internal/workapi"
)

// TestInfoConfigExcludesTheKVPlane pins that `issuegraph info --json` does not serve
// memories.
//
// It asserts the FILTER both info routes now apply, rather than shelling out:
// the two call sites differ only in which seam supplies the map, and the thing
// that can regress is somebody re-inlining GetAllConfig at one of them.
//
// Why it matters more than an ordinary config read: the issuegraph MCP server's
// get_schema_info tool runs `issuegraph info --schema --json` and returns the parsed
// dict whole, so every memory key and VALUE landed in the transcript of any
// agent that asked a schema question — and `issuegraph info` is the diagnostic people
// paste into bug reports.
func TestInfoConfigExcludesTheKVPlane(t *testing.T) {
	stored := map[string]string{
		"issue_prefix":    "issuegraph",
		"custom.statuses": "awaiting_review",
		kvkeys.MemoryConfigKeyPrefix + "deploy-notes": "the staging deploy token is sk-live-000",
		kvkeys.Prefix + "release.channel":             "beta",
		// Near misses that must SURVIVE: the rule is a prefix, not a substring.
		"kvetch":                    "not under the kv prefix",
		"custom.mentions.kv.inside": "kept",
	}

	got := workapi.FilterSettingsEnumeration(stored)

	for key := range got {
		if strings.HasPrefix(key, kvkeys.Prefix) {
			t.Errorf("issuegraph info would serve %q: the kv plane is not settings, and this map reaches "+
				"agent transcripts through the MCP get_schema_info tool", key)
		}
	}
	for _, want := range []string{"issue_prefix", "custom.statuses", "kvetch", "custom.mentions.kv.inside"} {
		if _, ok := got[want]; !ok {
			t.Errorf("%q was dropped; only keys under %q may be", want, kvkeys.Prefix)
		}
	}
}
