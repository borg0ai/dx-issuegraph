package main

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/term"

	"github.com/borg0ai/dx-issuegraph/internal/issuegraph"
)

const lastTouchedFile = "last-touched"

// lastTouchedFallbackEnv overrides the no-ID "last touched issue" fallback
// on mutating commands: 1/true always allows it, 0/false always denies it.
const lastTouchedFallbackEnv = "ISSUEGRAPH_LAST_TOUCHED_FALLBACK"

// AllowLastTouchedFallback reports whether a mutating command (update,
// close) may substitute the last-touched issue when no ID was given.
//
// The fallback is convenient at an interactive prompt but dangerous in
// scripts: `issuegraph update "$ID" ...` with an accidentally empty $ID silently
// mutates whatever issue happened to be touched last (bd-m00pb; a real
// agent session corrupted an unrelated closed issue this way). Read-only
// consumers like `issuegraph show --current` are unaffected.
//
// Precedence mirrors isNonInteractiveBootstrap:
// ISSUEGRAPH_LAST_TOUCHED_FALLBACK (any explicit value wins, only 1/true enables) >
// ISSUEGRAPH_NON_INTERACTIVE / CI (deny) > stdin terminal detection.
func AllowLastTouchedFallback() bool {
	if v := os.Getenv(lastTouchedFallbackEnv); v != "" {
		return v == "1" || v == "true"
	}
	if v := os.Getenv("ISSUEGRAPH_NON_INTERACTIVE"); v == "1" || v == "true" {
		return false
	}
	if v := os.Getenv("CI"); v == "1" || v == "true" {
		return false
	}
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// GetLastTouchedID returns the ID of the last touched issue.
// Returns empty string if no last touched issue exists or the file is unreadable.
func GetLastTouchedID() string {
	issuegraphDir := issuegraph.FindIssueGraphDir()
	if issuegraphDir == "" {
		return ""
	}

	lastTouchedPath := filepath.Join(issuegraphDir, lastTouchedFile)
	data, err := os.ReadFile(lastTouchedPath) // #nosec G304 -- path constructed from issuegraphDir
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(data))
}

// SetLastTouchedID saves the ID of the last touched issue.
// Silently ignores errors (best-effort tracking).
func SetLastTouchedID(issueID string) {
	if issueID == "" {
		return
	}

	issuegraphDir := issuegraph.FindIssueGraphDir()
	if issuegraphDir == "" {
		return
	}

	lastTouchedPath := filepath.Join(issuegraphDir, lastTouchedFile)
	// Write with restrictive permissions (local-only state)
	if err := os.WriteFile(lastTouchedPath, []byte(issueID+"\n"), 0600); err != nil {
		return
	}
	// Always advance mtime, even when the same ID is rewritten, so file-watch
	// fingerprints and cache validators that key on mtime never see an
	// "identical" marker after a write (GH#3965).
	now := time.Now()
	_ = os.Chtimes(lastTouchedPath, now, now)
}

// ClearLastTouched removes the last touched file.
// Silently ignores errors.
func ClearLastTouched() {
	issuegraphDir := issuegraph.FindIssueGraphDir()
	if issuegraphDir == "" {
		return
	}

	lastTouchedPath := filepath.Join(issuegraphDir, lastTouchedFile)
	_ = os.Remove(lastTouchedPath)
}
