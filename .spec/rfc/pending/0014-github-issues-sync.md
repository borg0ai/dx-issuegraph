# RFC 0014: Optional GitHub Issues synchronization (child of 0001)

**Status:** Pending

## Summary

Keep IssueGraph local-first and add opt-in, explicit synchronization with one
configured GitHub repository. Local IssueGraph remains usable without GitHub,
credentials, or network access. This RFC proposes changing RFC 0008 only for
users who explicitly configure GitHub sync; RFC 0008 remains authoritative
until this RFC is approved and implemented.

## Problem

Some teams track work in GitHub Issues and want those issues represented in
their local IssueGraph dependency graph. A network integration must not turn
local issue operations into implicit uploads or make GitHub a runtime
requirement.

## Goals

- Keep local-only behavior as the default; do not contact GitHub during init,
  create, read, update, close, hooks, or background processes.
- Add an explicit command to configure a repository and explicit commands to
  import, export, and synchronize issues.
- Support repeatable two-way synchronization for issue title, body,
  open/closed state, labels, and assignees where permissions and API support
  allow it.
- Store remote identity as local metadata keyed by GitHub repository and
  issue number; do not require a core database schema migration if existing
  metadata supports it.
- Detect concurrent changes and report conflicts without silently overwriting
  either side.
- Use the authenticated GitHub CLI credential flow when available; IssueGraph
  must not persist GitHub tokens itself.
- Keep GitHub Pull Requests out of issue import and synchronization.

## Non-goals

- Making GitHub required for local IssueGraph operation.
- Automatic, scheduled, webhook-triggered, or lifecycle-triggered sync.
- Synchronizing comments, reactions, projects, milestones, or GitHub-specific
  workflow policy in the first version.
- Supporting GitLab, Jira, or other trackers in this RFC.
- Replacing Git as transport for IssueGraph's local snapshot files.

## Design

Use GitHub's official REST Issues API for explicit reads and writes. The API
represents pull requests in issue-shaped responses, so every imported item
must be filtered to exclude records carrying a `pull_request` field. The
configured repository is explicit (`OWNER/REPO`); never infer a remote target
from whichever Git remote happens to be first.

Mapping is local metadata on each IssueGraph issue: provider, repository
owner/name, and GitHub issue number. A first import creates local issues and
records this mapping. Export creates a GitHub issue only for an unmapped local
issue selected by the user, then records the returned issue number. Later sync
matches only by this mapping; title text is not an identity key. Import-only
and export-only modes allow controlled adoption before two-way sync.

For mapped issues, compare the last synchronized content with current local
and remote content. If only one side changed, apply that side's change to the
other. If both changed the same synchronized field, report the issue and
conflicting fields, preserve both current versions, and continue other
non-conflicting issues. Never create duplicate issues to resolve a conflict.
Persist enough local sync baseline metadata to detect these conflicts; if the
existing metadata model cannot do that safely, return to RFC planning instead
of adding an unreviewed schema migration.

Commands require an explicit configured repository and explicit operation
(`import`, `export`, or `sync`). Print a dry-run plan first; writes require a
separate `--apply` flag. Authentication and authorization errors fail before
any local or remote mutation. No daemon, Docker service, or background worker
is introduced.

Because approved RFC 0008 currently bans all runtime tracker networking, this
RFC does not silently override it. On approval, amend or supersede 0008 so its
local-only rule becomes the default and GitHub API access is permitted only
through this explicitly configured, manually invoked integration.

## Acceptance

- Fresh install and all existing local workflows make no GitHub network calls.
- GitHub configuration is absent unless the user explicitly sets it.
- Import, export, and sync each show a dry-run plan; no remote writes occur
  without `--apply`.
- Sync excludes pull requests and matches existing issues only by stored
  repository plus issue number.
- One-sided edits propagate; concurrent same-field edits surface as conflicts
  without losing either side's current value.
- Authentication failures and API failures are visible, nonzero, and do not
  corrupt local data or partially conceal completed remote operations.
- Tests prove local-only mode does not load credentials or make HTTP requests.
- No Docker daemon or image is required for build, test, or runtime paths added
  by this RFC.
- RFC 0008 is reconciled before implementation ships; other tracker bans remain
  in force.
