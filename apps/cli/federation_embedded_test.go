//go:build cgo

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"

	"github.com/borg0ai/dx-issuegraph/internal/storage"
)

// issuegraphFederation runs "issuegraph federation" with extra args. Returns combined output.
func issuegraphFederation(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"federation"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	stdout, stderr, err := runCommandBuffers(t, cmd)
	if err != nil {
		t.Fatalf("issuegraph federation %s failed: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String()
}

// issuegraphFederationFail runs "issuegraph federation" expecting failure.
func issuegraphFederationFail(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"federation"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("issuegraph federation %s should have failed, got: %s", strings.Join(args, " "), out)
	}
	return string(out)
}

func TestFormatFederationPeerListJSONPreservesLegacyKeys(t *testing.T) {
	formatted := formatFederationPeerListJSON([]storage.RemoteInfo{{
		Name: "town-beta",
		URL:  "file:///tmp/town-beta",
	}})

	raw, err := json.Marshal(formatted)
	if err != nil {
		t.Fatalf("marshal federation peer JSON: %v", err)
	}
	body := string(raw)
	if !strings.Contains(body, `"Name":"town-beta"`) || !strings.Contains(body, `"URL":"file:///tmp/town-beta"`) {
		t.Fatalf("formatted JSON should preserve legacy Name/URL keys, got %s", body)
	}
	if strings.Contains(body, `"name"`) || strings.Contains(body, `"url"`) {
		t.Fatalf("formatted JSON should not expose lowercase RemoteInfo storage tags, got %s", body)
	}
}

func TestEmbeddedFederation(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt federation tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)

	t.Run("list_peers_empty", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "fdlst0")

		out := issuegraphFederation(t, issuegraph, dir, "list-peers")
		if !strings.Contains(out, "No federation peers") {
			t.Errorf("expected 'No federation peers', got: %s", out)
		}
	})

	t.Run("list_peers_empty_json", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "fdlstj")

		out := issuegraphFederation(t, issuegraph, dir, "list-peers", "--json")
		// Should be valid JSON (empty array or null)
		out = strings.TrimSpace(out)
		if out != "null" && out != "[]" {
			// Try parsing as JSON array
			var result []interface{}
			if err := json.Unmarshal([]byte(out), &result); err != nil {
				t.Fatalf("expected valid JSON array, got: %s", out)
			}
		}
	})

	t.Run("add_peer_simple", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "fdadd")

		out := issuegraphFederation(t, issuegraph, dir, "add-peer", "test-peer", "file:///tmp/fake-peer")
		if !strings.Contains(out, "test-peer") {
			t.Errorf("expected peer name in output, got: %s", out)
		}

		// Verify it appears in list
		listOut := issuegraphFederation(t, issuegraph, dir, "list-peers")
		if !strings.Contains(listOut, "test-peer") {
			t.Errorf("expected 'test-peer' in list, got: %s", listOut)
		}
	})

	t.Run("add_peer_json", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "fdaddj")

		out := issuegraphFederation(t, issuegraph, dir, "add-peer", "json-peer", "file:///tmp/json-peer", "--json")
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(out), &result); err != nil {
			t.Fatalf("failed to parse JSON: %v\n%s", err, out)
		}
		if added, _ := result["added"].(string); added != "json-peer" {
			t.Errorf("expected added='json-peer', got %q", added)
		}
	})

	t.Run("add_peer_with_credentials", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "fdcred")

		out := issuegraphFederation(t, issuegraph, dir, "add-peer", "auth-peer", "file:///tmp/auth-peer",
			"--user", "admin", "--password", "secret", "--json")
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(out), &result); err != nil {
			t.Fatalf("failed to parse JSON: %v\n%s", err, out)
		}
		if hasAuth, _ := result["has_auth"].(bool); !hasAuth {
			t.Error("expected has_auth=true")
		}
	})

	t.Run("add_peer_with_sovereignty", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "fdsov")

		out := issuegraphFederation(t, issuegraph, dir, "add-peer", "sov-peer", "file:///tmp/sov-peer",
			"--sovereignty", "T2", "--json")
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(out), &result); err != nil {
			t.Fatalf("failed to parse JSON: %v\n%s", err, out)
		}
		if sov, _ := result["sovereignty"].(string); sov != "T2" {
			t.Errorf("expected sovereignty='T2', got %q", sov)
		}
	})

	t.Run("remove_peer", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "fdrm")

		issuegraphFederation(t, issuegraph, dir, "add-peer", "removable", "file:///tmp/removable")

		// Verify it exists
		listOut := issuegraphFederation(t, issuegraph, dir, "list-peers")
		if !strings.Contains(listOut, "removable") {
			t.Fatalf("peer should exist before removal, got: %s", listOut)
		}

		// Remove it
		out := issuegraphFederation(t, issuegraph, dir, "remove-peer", "removable")
		if !strings.Contains(out, "removable") {
			t.Errorf("expected peer name in removal output, got: %s", out)
		}

		// Verify it's gone
		listOut = issuegraphFederation(t, issuegraph, dir, "list-peers")
		if strings.Contains(listOut, "removable") {
			t.Errorf("peer should be gone after removal, got: %s", listOut)
		}
	})

	t.Run("remove_peer_json", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "fdrmj")

		issuegraphFederation(t, issuegraph, dir, "add-peer", "rm-json", "file:///tmp/rm-json")

		out := issuegraphFederation(t, issuegraph, dir, "remove-peer", "rm-json", "--json")
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(out), &result); err != nil {
			t.Fatalf("failed to parse JSON: %v\n%s", err, out)
		}
		if removed, _ := result["removed"].(string); removed != "rm-json" {
			t.Errorf("expected removed='rm-json', got %q", removed)
		}
	})

	t.Run("status_no_peers", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "fdst0")

		out := issuegraphFederation(t, issuegraph, dir, "status")
		if !strings.Contains(out, "No federation peers") {
			t.Errorf("expected 'No federation peers', got: %s", out)
		}
	})

	t.Run("status_json_no_peers", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "fdstj")

		out := issuegraphFederation(t, issuegraph, dir, "status", "--json")
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(out), &result); err != nil {
			t.Fatalf("failed to parse JSON: %v\n%s", err, out)
		}
		if _, ok := result["peers"]; !ok {
			t.Error("missing 'peers' in status JSON")
		}
	})
}

func TestEmbeddedFederationConcurrent(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt federation tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "fdconc")

	const numWorkers = 10

	type result struct {
		worker int
		out    string
		err    error
	}

	results := make([]result, numWorkers)
	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for w := 0; w < numWorkers; w++ {
		go func(worker int) {
			defer wg.Done()
			peerName := fmt.Sprintf("conc-peer-%d", worker)
			peerURL := fmt.Sprintf("file:///tmp/conc-peer-%d", worker)
			cmd := exec.Command(issuegraph, "federation", "add-peer", peerName, peerURL)
			cmd.Dir = dir
			cmd.Env = issuegraphEnv(dir)
			out, err := cmd.CombinedOutput()
			results[worker] = result{worker: worker, out: string(out), err: err}
		}(w)
	}
	wg.Wait()

	successes := 0
	for _, r := range results {
		if strings.Contains(r.out, "panic") {
			t.Errorf("worker %d panicked:\n%s", r.worker, r.out)
		}
		if r.err == nil {
			successes++
		} else if !strings.Contains(r.out, "one writer at a time") {
			t.Errorf("worker %d failed with unexpected error: %v\n%s", r.worker, r.err, r.out)
		}
	}
	if successes < 1 {
		t.Errorf("expected at least 1 successful add-peer, got %d", successes)
	}
	t.Logf("%d/%d federation workers succeeded", successes, numWorkers)
}
