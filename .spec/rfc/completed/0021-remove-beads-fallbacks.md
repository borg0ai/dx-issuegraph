# RFC 0021: Remove BEADS env and .beads directory fallbacks

**Status:** Implemented

## Summary

The product is renamed to IssueGraph. End the RFC 0002 migration period: stop reading `BEADS_DIR`, `BEADS_MAIL_DELEGATE`, and the `.beads` workspace directory. Only `ISSUEGRAPH_*` and `.issuegraph` remain.

## Problem

Legacy BEADS names still work as fallbacks (`modules/core/internal/legacydir`, `apps/cli/mail.go`). The owner wants no BEADS identity left in the product.

## Goals

No code path reads a `BEADS_*` environment variable or a `.beads` directory.

## Non-goals

Historical docs (release-gates, engdocs, CHANGELOG, completed RFCs), other `BEADS_*` names in tests, and renaming the `legacydir` package.

## Design

`legacydir.EnvIssueGraphDir` returns only `ISSUEGRAPH_DIR`; `legacydir.WorkspaceDir` returns `root/.issuegraph`; `apps/cli/mail.go` drops the `BEADS_MAIL_DELEGATE` lookup and help text; update `legacydir_test.go` and `messaging_test.go`. Breaking: workspaces that only have `.beads` or only set `BEADS_DIR` are no longer found.

## Acceptance

`grep -rn 'BEADS_\|\.beads' --include=*.go` outside tests finds nothing; `go vet` and the touched tests pass with CGO on and off.
