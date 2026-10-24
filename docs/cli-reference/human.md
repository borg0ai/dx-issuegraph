---
title: "issuegraph human"
description: "Display a focused help menu showing only the most common commands."
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc human`.

Display a focused help menu showing only the most common commands.

issuegraph has 70+ commands - many for AI agents, integrations, and advanced workflows.
This command shows the ~15 essential commands that human users need most often.

For the full command list, run: issuegraph --help

SUBCOMMANDS:
  human list              List all human-needed issuegraph (issues with 'human' label)
  human respond &lt;id&gt;      Respond to a human-needed issue (adds comment and closes)
  human dismiss &lt;id&gt;      Dismiss a human-needed issue permanently
  human stats             Show summary statistics for human-needed issuegraph

```
issuegraph human [flags]
```

## issuegraph human dismiss

Dismiss a human-needed issue permanently without responding.

The issue is closed with a "Dismissed" reason and optional note.

Examples:
  issuegraph human dismiss issuegraph-123
  issuegraph human dismiss issuegraph-123 --reason "No longer applicable"

```
issuegraph human dismiss <issue-id> [flags]
```

**Flags:**

```
      --reason string   Reason for dismissal (optional)
```

## issuegraph human list

List all issues labeled with 'human' tag.

These are issues that require human intervention or input.

Examples:
  issuegraph human list
  issuegraph human list --status=open
  issuegraph human list --json

```
issuegraph human list [flags]
```

**Flags:**

```
  -s, --status string   Filter by status (open, closed, etc.)
```

## issuegraph human respond

Respond to a human-needed issue by adding a comment and closing it.

The response is added as a comment and the issue is closed with reason "Responded".

Examples:
  issuegraph human respond issuegraph-123 --response "Use OAuth2 for authentication"
  issuegraph human respond issuegraph-123 -r "Approved, proceed with implementation"

```
issuegraph human respond <issue-id> [flags]
```

**Flags:**

```
  -r, --response string   Response text (required)
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
