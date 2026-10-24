---
title: "issuegraph import"
description: "Import issues from a JSONL file (newline-delimited JSON) into the database."
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc import`.

Import issues from a JSONL file (newline-delimited JSON) into the database.

If no file is specified, imports from the configured import.path under .issuegraph/
(default: issues.jsonl). Use "-" to read from stdin. This is the incremental counterpart to
'issuegraph export': new issues are created and existing issues are updated (upsert
semantics).

Memory records (lines with "_type":"memory") are automatically detected and
imported as persistent memories (equivalent to 'issuegraph remember'). This makes
'issuegraph export | issuegraph import' a full round-trip for both issues and memories.

Each JSONL line should map to an issue. The importer accepts every field
'issuegraph export' emits — see 'issuegraph export' output for the canonical schema. Only
"title" is required; everything else is optional.

Common fields:
  title                  Required. Short summary.
  description            Long-form body.
  design, notes,         Additional content sections.
    acceptance_criteria
  issue_type             bug | feature | task | epic | chore | ...
  priority               0-4 (0 = critical). 0 is preserved (no omitempty).
  status                 open | in_progress | blocked | closed | ...
                         (rows with status "tombstone" are skipped)
  assignee, owner,       Ownership metadata.
    created_by
  labels                 Array of strings.
  dependencies           Array of &#123;issue_id, depends_on_id, type, ...&#125;.
  comments               Array of comment objects.
  external_ref,          Cross-system identifiers (e.g. "gh-9").
    source_system
  due_at, defer_until    RFC3339 timestamps for scheduling.
  metadata               Arbitrary JSON object preserved verbatim.

Timestamps (created_at, updated_at, started_at, closed_at) are preserved
when present in the JSONL and otherwise filled in by the importer. The
legacy "wisp" boolean is accepted as an alias for "ephemeral".

By default a row only rewrites an existing local issue when its
updated_at is strictly newer. Older rows are skipped (reported as
stale_skipped_ids) and rows with the same updated_at keep every local
column — updated_at has second granularity, so a timestamp tie can be
two distinct same-second updates, and the local row wins the tie
(reported as tie_kept_local_ids; the row's labels/comments/dependencies
still merge). The guard is also enforced inside the upsert itself, so a
local update that lands while the import is running is preserved rather
than overwritten. Existing issues that the import did rewrite are listed
with a field-level summary (updated_issues), so local state changed by
an import is visible. To deliberately restore an older snapshot, pass
--allow-stale, which imports every row even when it overwrites newer
local state.

EXAMPLES:
  issuegraph import                        # Import from configured import.path
  issuegraph import backup.jsonl           # Import from a specific file
  issuegraph import -i backup.jsonl        # Legacy alias for a specific file
  issuegraph import -                      # Read JSONL from stdin
  cat issues.jsonl | issuegraph import -   # Pipe JSONL from another tool
  issuegraph import --dry-run              # Show what would be imported
  issuegraph import --dedup                # Skip issues with duplicate titles
  issuegraph import --allow-stale old.jsonl # Restore an older snapshot (overwrites newer local rows)
  issuegraph import --json                 # Structured output with created and skipped IDs

```
issuegraph import [file|-] [flags]
```

**Flags:**

```
      --allow-stale    Import rows even when older than the local issue (required to restore an older snapshot)
      --dedup          Skip lines whose title matches an existing open issue
      --dry-run        Show what would be imported without importing
  -i, --input string   Read JSONL from a specific file
```
