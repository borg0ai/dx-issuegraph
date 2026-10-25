package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"

	"github.com/borg0ai/dx-issuegraph/internal/atomicfile"
	"github.com/borg0ai/dx-issuegraph/internal/config"
	"github.com/borg0ai/dx-issuegraph/internal/issuegraph"
	"github.com/borg0ai/dx-issuegraph/internal/storage"
	"github.com/borg0ai/dx-issuegraph/internal/types"
)

// A snapshot is the pair .issuegraph/issues.jsonl + .issuegraph/manifest.json.
// The manifest names the complete id set, so restore can delete local rows the
// snapshot no longer contains (a bare JSONL upsert cannot express deletion).
// Git carries the two files; IssueGraph itself never runs git (RFC 0004).
const (
	snapshotFormatVersion = 1
	snapshotManifestFile  = "manifest.json"
)

type snapshotManifest struct {
	FormatVersion int      `json:"format_version"`
	Count         int      `json:"count"`
	IDs           []string `json:"ids"`
	// IssuesSHA256 is the hex digest of the exact issues.jsonl bytes, so a
	// manifest that does not match its JSONL is rejected rather than trusted.
	IssuesSHA256 string `json:"issues_sha256"`
}

// buildSnapshotManifest derives the manifest from the JSONL bytes.
func buildSnapshotManifest(jsonl []byte) (*snapshotManifest, []*types.Issue, error) {
	issues, _, err := parseImportRecords(bytes.NewReader(jsonl))
	if err != nil {
		return nil, nil, err
	}
	ids := make([]string, 0, len(issues))
	seen := make(map[string]struct{}, len(issues))
	for _, issue := range issues {
		if _, dup := seen[issue.ID]; dup {
			return nil, nil, fmt.Errorf("duplicate issue id %q in snapshot", issue.ID)
		}
		seen[issue.ID] = struct{}{}
		ids = append(ids, issue.ID)
	}
	sort.Strings(ids)
	sum := sha256.Sum256(jsonl)
	return &snapshotManifest{
		FormatVersion: snapshotFormatVersion,
		Count:         len(ids),
		IDs:           ids,
		IssuesSHA256:  hex.EncodeToString(sum[:]),
	}, issues, nil
}

// writeSnapshotManifest writes manifest.json beside issuesPath, atomically.
// It is written after the JSONL, so a crash between the two leaves a manifest
// whose digest no longer matches and restore refuses it.
func writeSnapshotManifest(issuesPath string) error {
	jsonl, err := os.ReadFile(issuesPath) //nolint:gosec // path is the resolved export path
	if err != nil {
		return fmt.Errorf("read snapshot issues: %w", err)
	}
	manifest, _, err := buildSnapshotManifest(jsonl)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return atomicfile.WriteFile(filepath.Join(filepath.Dir(issuesPath), snapshotManifestFile), append(data, '\n'), 0o644)
}

// readSnapshot loads and fully validates a snapshot. It returns the issues only
// when the format version, digest, count and id set all agree.
func readSnapshot(issuesPath string) (*snapshotManifest, []*types.Issue, error) {
	manifestPath := filepath.Join(filepath.Dir(issuesPath), snapshotManifestFile)
	raw, err := os.ReadFile(manifestPath) //nolint:gosec // sibling of the snapshot JSONL
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil, fmt.Errorf("snapshot manifest %s is missing; refusing to restore", manifestPath)
		}
		return nil, nil, fmt.Errorf("read snapshot manifest: %w", err)
	}
	var manifest snapshotManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return nil, nil, fmt.Errorf("parse snapshot manifest: %w", err)
	}
	if manifest.FormatVersion != snapshotFormatVersion {
		return nil, nil, fmt.Errorf("unsupported snapshot format version %d (this build reads %d)", manifest.FormatVersion, snapshotFormatVersion)
	}
	jsonl, err := os.ReadFile(issuesPath) //nolint:gosec // resolved snapshot path
	if err != nil {
		return nil, nil, fmt.Errorf("read snapshot issues: %w", err)
	}
	derived, issues, err := buildSnapshotManifest(jsonl)
	if err != nil {
		return nil, nil, fmt.Errorf("snapshot issues are corrupt: %w", err)
	}
	if derived.IssuesSHA256 != manifest.IssuesSHA256 {
		return nil, nil, errors.New("snapshot issues do not match the manifest digest; the files are from different exports or corrupt")
	}
	if derived.Count != manifest.Count || len(manifest.IDs) != manifest.Count {
		return nil, nil, fmt.Errorf("snapshot manifest count %d does not match %d issues", manifest.Count, derived.Count)
	}
	for i, id := range derived.IDs {
		if manifest.IDs[i] != id {
			return nil, nil, errors.New("snapshot manifest id list does not match the issues file")
		}
	}
	return &manifest, issues, nil
}

// snapshotTxRunner is the one store capability restore needs.
type snapshotTxRunner interface {
	RunInTransaction(ctx context.Context, commitMsg string, fn func(tx storage.Transaction) error) error
}

// restoreSnapshot makes the store equal to the snapshot in ONE transaction:
// every snapshot issue is upserted and every in-scope local id the manifest
// does not list is deleted. Nothing is written if validation or the
// transaction fails.
func restoreSnapshot(ctx context.Context, st snapshotTxRunner, manifest *snapshotManifest, issues []*types.Issue, scope types.IssueFilter, actor string) (upserted, deleted int, err error) {
	keep := make(map[string]struct{}, len(manifest.IDs))
	for _, id := range manifest.IDs {
		keep[id] = struct{}{}
	}
	err = st.RunInTransaction(ctx, "issuegraph: restore snapshot", func(tx storage.Transaction) error {
		deleted = 0
		if len(issues) > 0 {
			if err := tx.CreateIssues(ctx, issues, actor); err != nil {
				return fmt.Errorf("upsert snapshot issues: %w", err)
			}
		}
		upserted = len(issues)
		local, err := tx.SearchIssueIDs(ctx, "", scope)
		if err != nil {
			return fmt.Errorf("list local issues: %w", err)
		}
		for _, id := range local {
			if _, ok := keep[id]; ok {
				continue
			}
			if err := tx.DeleteIssue(ctx, id); err != nil {
				return fmt.Errorf("delete %s: %w", id, err)
			}
			deleted++
		}
		return nil
	})
	return upserted, deleted, err
}

// snapshotIssuesFile is the snapshot JSONL name, relative to .issuegraph/.
func snapshotIssuesFile() string {
	if name := config.GetString("export.path"); name != "" {
		return name
	}
	return "issues.jsonl"
}

var snapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "Work with the Git-tracked issue snapshot",
	Long: `The snapshot is .issuegraph/issues.jsonl plus .issuegraph/manifest.json.
Commit and move those two files with Git yourself; IssueGraph never runs Git
or contacts a remote.`,
}

var snapshotRestoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Make the local database equal to the snapshot",
	Long: `Validate .issuegraph/issues.jsonl against .issuegraph/manifest.json, then in one
transaction upsert every snapshot issue and delete local issues the manifest
does not list. A missing manifest, corrupt file or digest mismatch fails
without changing the database.`,
	Args:         cobra.NoArgs,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := issuegraph.FindIssueGraphDir()
		if dir == "" {
			return errors.New("no .issuegraph directory found")
		}
		return runSnapshotRestore(rootCtx, store, dir, autoExportFilter(rootCtx), getActorWithGit(), cmd.OutOrStdout())
	},
}

// runSnapshotRestore validates the snapshot in issuegraphDir and applies it.
// The store is not touched unless validation passes.
func runSnapshotRestore(ctx context.Context, st snapshotTxRunner, issuegraphDir string, scope types.IssueFilter, actor string, out io.Writer) error {
	manifest, issues, err := readSnapshot(filepath.Join(issuegraphDir, snapshotIssuesFile()))
	if err != nil {
		return err
	}
	upserted, deleted, err := restoreSnapshot(ctx, st, manifest, issues, scope, actor)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(out, "Restored snapshot: %d issue(s) upserted, %d deleted\n", upserted, deleted)
	return err
}

func init() {
	snapshotCmd.AddCommand(snapshotRestoreCmd)
	rootCmd.AddCommand(snapshotCmd)
}
