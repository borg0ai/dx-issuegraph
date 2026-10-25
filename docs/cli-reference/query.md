---
title: "issuegraph query"
description: "Query issues using a simple query language"
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc query`.

Query issues using a simple query language that supports compound filters,
boolean operators, and date-relative expressions.

The query language enables complex filtering that would otherwise require
multiple flags or piping through jq.

Syntax:
  field=value       Equality comparison
  field!=value      Inequality comparison
  field&gt;value       Greater than
  field&gt;=value      Greater than or equal
  field&lt;value       Less than
  field&lt;=value      Less than or equal

Boolean operators (case-insensitive):
  expr AND expr     Both conditions must match
  expr OR expr      Either condition can match
  NOT expr          Negates the condition
  (expr)            Grouping with parentheses

Supported fields:
  status            Stored status (open, in_progress, blocked, deferred, closed). Note: dependency-blocked issues stay "open"; use 'issuegraph blocked' to find them
  priority          Priority level (0-4)
  type              Issue type (bug, feature, task, epic, chore, decision)
  assignee          Assigned user (use "none" for unassigned)
  owner             Issue owner
  label             Issue label (use "none" for unlabeled)
  title             Search in title (contains)
  description       Search in description (contains, "none" for empty)
  notes             Search in notes (contains)
  created           Creation date/time
  updated           Last update date/time
  started           Date/time issue first transitioned to in_progress
  closed            Close date/time
  id                Issue ID (supports wildcards: bd-*)
  spec              Spec ID (supports wildcards)
  pinned            Boolean (true/false)
  ephemeral         Boolean (true/false)
  template          Boolean (true/false)
  parent            Parent issue ID
  mol_type          Molecule type (swarm, patrol, work)

Date values:
  Relative durations: 7d (7 days ago), 24h (24 hours ago), 2w (2 weeks ago)
  Absolute dates: 2025-01-15, 2025-01-15T10:00:00Z
  Natural language: tomorrow, "next monday", "in 3 days"

Examples:
  issuegraph query "status=open AND priority&gt;1"
  issuegraph query "status=open AND priority&lt;=2 AND updated&gt;7d"
  issuegraph query "(status=open OR status=blocked) AND priority&lt;2"
  issuegraph query "type=bug AND label=urgent"
  issuegraph query "NOT status=closed"
  issuegraph query "assignee=none AND type=task"
  issuegraph query "created&gt;30d AND status!=closed"
  issuegraph query "label=frontend OR label=backend"
  issuegraph query "title=authentication AND priority=0"

```
issuegraph query [expression] [flags]
```

**Flags:**

```
  -a, --all           Include closed issues (default: exclude closed)
  -n, --limit int     Limit results (default: 50, 0 = unlimited) (default 50)
      --long          Show detailed multi-line output for each issue
      --offset int    Skip the first N matching results (0-based). Only supported under --proxied-server.
      --parse-only    Only parse the query and show the AST (for debugging)
  -r, --reverse       Invert the sort field's default direction (created/updated/closed default to newest-first, so --sort updated --reverse is oldest-first)
      --sort string   Sort by field: priority, created, updated, closed, status, id, title, type, assignee
```
