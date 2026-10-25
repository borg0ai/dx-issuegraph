---
title: "issuegraph human"
description: "Show essential commands for human users"
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc human`.

Display a focused help menu showing only the most common commands.

issuegraph has 70+ commands - many for AI agents, integrations, and advanced workflows.
This command shows the ~15 essential commands that human users need most often.

For the full command list, run: issuegraph --help

SUBCOMMANDS:
  human list              List human-needed issuegraph (issues with 'human' label; hides closed by default)
  human respond &lt;id&gt;      Respond to a human-needed issue (adds comment and closes)
  human dismiss &lt;id&gt;      Dismiss a human-needed issue permanently
  human stats             Show summary statistics for human-needed issuegraph

```
issuegraph human [flags]
issuegraph human [command]
```

## issuegraph human dismiss

Dismiss a human-needed issue permanently without responding.

The issue is closed with a "Dismissed" reason and optional note.
The reason can be given as positional arguments or --reason.

Examples:
  issuegraph human dismiss bd-123
  issuegraph human dismiss bd-123 "No longer applicable"
  issuegraph human dismiss bd-123 --reason "No longer applicable"

```
issuegraph human dismiss <issue-id> [reason...] [flags]
```

**Flags:**

```
      --reason string   Reason for dismissal (optional)
```

## issuegraph human list

List issues labeled with 'human' tag.

These are issues that require human intervention or input. Every
human-labeled issue shows regardless of type (including gates and wisps).
By default closed, pinned, and other done/frozen issuegraph are hidden; use
--status to select specific statuses, or --status=all to include every
status.

Examples:
  issuegraph human list
  issuegraph human list --status=closed
  issuegraph human list --status=all
  issuegraph human list --json

```
issuegraph human list [flags]
```

**Flags:**

```
  -s, --status string   Filter by status (open, closed, etc.; comma-separated for multiple, 'all' for every status)
```

## issuegraph human respond

Respond to a human-needed issue by adding a comment and closing it.

The response is added as a comment and the issue is closed with reason "Responded".
The response text can be given as positional arguments, --response, --file, or --stdin.

Examples:
  issuegraph human respond bd-123 "Use OAuth2 for authentication"
  issuegraph human respond bd-123 -r "Approved, proceed with implementation"
  issuegraph human respond bd-123 --file response.md
  echo "Approved" | issuegraph human respond bd-123 --stdin

```
issuegraph human respond <issue-id> [response...] [flags]
```

**Flags:**

```
      --file string       Read response text from file
  -r, --response string   Response text
      --stdin             Read response text from stdin
```

## issuegraph human stats

Display summary statistics for human-needed issuegraph.

Shows counts for total, pending (open), responded (closed without dismiss),
and dismissed issuegraph.

Example:
  issuegraph human stats

```
issuegraph human stats [flags]
```
