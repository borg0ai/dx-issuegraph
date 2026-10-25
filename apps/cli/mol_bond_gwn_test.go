//go:build cgo

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBdMolBond_ResolvesPlainFormulaName is a regression test for GH#3873:
// issuegraph mol bond fails to resolve formula names that don't match the
// looksLikeFormulaName gatekeeper at apps/cli/mol_bond.go.
//
// The gatekeeper only accepts names starting with "mol-", containing
// ".formula", or containing a path separator. Plain formula names — names
// that are valid formula filenames but match none of those three patterns —
// are rejected even though parser.LoadByName would load them, and every
// other subcommand (issuegraph formula show, issuegraph mol seed, issuegraph mol pour, issuegraph cook)
// resolves them without trouble.
//
// Before the fix, this test reproduces the user-reported error verbatim:
//
//	Error: 'plain-name-formula' not found (not an issue ID or formula name)
func TestBdMolBond_ResolvesPlainFormulaName(t *testing.T) {
	issuegraph := buildEmbeddedBD(t)

	// Set up a workspace with a plain-named formula and an issue to bond
	// against. HOME is pointed at the temp dir, so $HOME/.issuegraph/formulas is the
	// same as the workspace's formulas dir — formulas written here are
	// discoverable through both the workspace and the user-level search paths.
	dir, issuegraphDir, _ := issuegraphInit(t, issuegraph, "--prefix", "gwn")

	formulaDir := filepath.Join(issuegraphDir, "formulas")
	if err := os.MkdirAll(formulaDir, 0o755); err != nil {
		t.Fatalf("mkdir formulas dir: %v", err)
	}

	const formulaTOML = `formula = "plain-name-formula"
version = 1
type = "workflow"

[[steps]]
id = "do-thing"
title = "Do the thing"
type = "task"
`
	formulaPath := filepath.Join(formulaDir, "plain-name-formula.formula.toml")
	if err := os.WriteFile(formulaPath, []byte(formulaTOML), 0o644); err != nil {
		t.Fatalf("write formula: %v", err)
	}

	// Create an issue to use as the second operand of issuegraph mol bond.
	out, err := issuegraphRunWithFlockRetry(t, issuegraph, dir, "create", "target", "-t", "task", "-p", "4", "--json")
	if err != nil {
		t.Fatalf("issuegraph create failed: %v\n%s", err, out)
	}
	targetID := extractFirstJSONStringField(string(out), "id")
	if targetID == "" {
		t.Fatalf("could not parse issue id from issuegraph create output: %s", out)
	}

	// Confirm the formula registry is wired up: issuegraph formula show works for the
	// plain name. If this fails, the test setup is wrong, not the bug.
	out, err = issuegraphRunWithFlockRetry(t, issuegraph, dir, "formula", "show", "plain-name-formula")
	if err != nil {
		t.Fatalf("issuegraph formula show could not resolve the plain formula name (test setup issue, not the gwn bug): %v\n%s", err, out)
	}

	// The critical step. issuegraph mol bond should accept the same plain formula
	// name that issuegraph formula show / issuegraph mol seed / issuegraph mol pour / issuegraph cook all
	// resolve fine.
	out, err = issuegraphRunWithFlockRetry(t, issuegraph, dir, "mol", "bond", "plain-name-formula", targetID, "--dry-run")
	if err != nil {
		t.Fatalf("issuegraph mol bond should resolve a plain formula name; got error: %v\noutput:\n%s", err, out)
	}

	outStr := string(out)
	if !strings.Contains(outStr, "plain-name-formula") {
		t.Errorf("dry-run output did not mention the formula:\n%s", outStr)
	}
	if !strings.Contains(outStr, "formula") {
		t.Errorf("dry-run output did not indicate a formula was resolved:\n%s", outStr)
	}
}

// extractFirstJSONStringField pulls the first occurrence of "<field>": "value"
// from a JSON-ish blob. Good enough for the JSON shape `issuegraph create --json`
// returns; avoids pulling in encoding/json for one field.
func extractFirstJSONStringField(blob, field string) string {
	key := `"` + field + `":`
	idx := strings.Index(blob, key)
	if idx < 0 {
		return ""
	}
	rest := strings.TrimLeft(blob[idx+len(key):], " \t")
	if !strings.HasPrefix(rest, `"`) {
		return ""
	}
	rest = rest[1:]
	end := strings.Index(rest, `"`)
	if end < 0 {
		return ""
	}
	return rest[:end]
}
