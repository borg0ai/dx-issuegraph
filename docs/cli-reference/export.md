---
title: "issuegraph export"
description: "Export all issues to JSONL (newline-delimited JSON) format."
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc export`.

Export all issues to JSONL (newline-delimited JSON) format.

Each line is a complete JSON object representing one issue, including its
labels, dependencies, and comments.

This command is for issue export, migration, and interoperability. It exports
records from the issues table; it is not a full database backup and does not
capture Dolt branches, commit history, working-set state, or non-issue tables.
For supported full backup/restore flows, use 'issuegraph backup init', 'issuegraph backup sync',
and 'issuegraph backup restore'.

By default, exports only regular issues (excluding infrastructure issuegraph
like agents, roles, and messages). Use --all to include everything.

Memories (from 'issuegraph remember') are excluded by default because they may
contain sensitive agent context. Use --include-memories or --all to
include them.

EXAMPLES:
  issuegraph export                              # Export issues to stdout
  issuegraph export -o issues.jsonl              # Export issues to file
  issuegraph export --include-memories           # Export issues + memories
  issuegraph export --all -o full.jsonl          # Include infra + templates + gates + memories
  issuegraph export --scrub -o clean.jsonl       # Exclude test/pollution records

```
issuegraph export [flags]
```

**Flags:**

```
      --all                Include all records (infra, templates, gates, memories)
      --include-infra      Include infrastructure issuegraph (agents, roles, messages)
      --include-memories   Include persistent memories (from 'issuegraph remember') in the export
  -o, --output string      Output file path (default: stdout)
      --scrub              Exclude test/pollution records
```
