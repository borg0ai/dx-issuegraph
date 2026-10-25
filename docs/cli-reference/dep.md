---
title: "issuegraph dep"
description: "Manage dependencies"
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc dep`.

Manage dependencies between issues.

When called with an issue ID and --blocks flag, creates a blocking dependency:
  issuegraph dep &lt;blocker-id&gt; --blocks &lt;blocked-id&gt;

This is equivalent to:
  issuegraph dep add &lt;blocked-id&gt; &lt;blocker-id&gt;

Examples:
  issuegraph dep bd-xyz --blocks bd-abc    # bd-xyz blocks bd-abc
  issuegraph dep add bd-abc bd-xyz         # Same as above (bd-abc depends on bd-xyz)

```
issuegraph dep [issue-id] [flags]
issuegraph dep [command]
```

**Flags:**

```
  -b, --blocks string    Issue ID that this issue blocks (shorthand for: issuegraph dep add <blocked> <blocker>)
      --no-cycle-check   Skip the post-add cycle warning (the per-edge cycle check still runs)
```

## issuegraph dep add

Add a dependency between two issues.

The depends-on-id can be provided as:
  - A positional argument: issuegraph dep add issue-123 issue-456
  - A flag: issuegraph dep add issue-123 --blocked-by issue-456
  - A flag: issuegraph dep add issue-123 --depends-on issue-456

The --blocked-by and --depends-on flags are aliases and both mean "issue-123
depends on (is blocked by) the specified issue."

The depends-on-id can be:
  - A local issue ID (e.g., bd-xyz)
  - An external reference: external:&lt;project&gt;:&lt;capability&gt;

For bulk wiring, pass newline-delimited JSON with --file. Each line must be an
object with "from" and "to" fields, and may include "type". The aliases
"issue_id" and "depends_on_id" are also accepted. Use --file - to read stdin.

External references are stored as-is and resolved at query time using
the external_projects config. They block the issue until the capability
is "shipped" in the target project.

With no -t/--type the edge is created as type=blocks, which excludes the
dependent from issuegraph ready. When stderr is an interactive terminal, an advisory
note says so once per command; it is silent for scripted and agent callers
(non-TTY stderr) and can be turned off with --quiet or ISSUEGRAPH_NO_DEP_TYPE_WARNING=1.

Examples:
  issuegraph dep add bd-42 bd-41                              # Positional args
  issuegraph dep add bd-42 --blocked-by bd-41                 # Flag syntax (same effect)
  issuegraph dep add bd-42 --depends-on bd-41                 # Alias (same effect)
  issuegraph dep add gt-xyz external:issuegraph:mol-run-assignee   # Cross-project dependency
  issuegraph dep add bd-42 bd-41 --no-cycle-check             # Skip the post-add cycle warning
  issuegraph dep add --file deps.jsonl                        # Bulk JSONL: &#123;"from":"bd-42","to":"bd-41"&#125;

```
issuegraph dep add [issue-id] [depends-on-id] [flags]
```

**Flags:**

```
      --blocked-by string   Issue ID that blocks the first issue (alternative to positional arg)
      --depends-on string   Issue ID that the first issue depends on (alias for --blocked-by)
      --file string         Read dependency edges from JSONL file, or '-' for stdin
      --no-cycle-check      On bulk --file adds, skip per-edge cycle checks and the post-add cycle warning (one final whole-graph check still runs before commit); on a single-edge add, skip only the post-add cycle warning
  -t, --type string         Dependency type (blocks|tracks|related|parent-child|discovered-from|until|caused-by|validates|relates-to|supersedes); 'blocked-by' and 'depends-on' are accepted as aliases for 'blocks' (default "blocks")
```

## issuegraph dep cycles

Detect dependency cycles

```
issuegraph dep cycles [flags]
```

## issuegraph dep list

List dependencies or dependents of one or more issues with optional type filtering.

By default shows dependencies (what issues depend on). Use --direction to control:
  - down: Show dependencies (what this issue depends on) - default
  - up:   Show dependents (what depends on this issue)

Multiple IDs can be provided for batch dep listing. With --json, the output
is a flat array of dependency records across all requested issues.

Use --type to filter by dependency type (e.g., tracks, blocks, parent-child).

Examples:
  issuegraph dep list gt-abc                     # Show what gt-abc depends on
  issuegraph dep list gt-abc gt-def              # Batch: deps for both issues
  issuegraph dep list gt-abc --direction=up      # Show what depends on gt-abc
  issuegraph dep list gt-abc --direction=up -t tracks  # Show what tracks gt-abc (convoy tracking)

```
issuegraph dep list [issue-id...] [flags]
```

**Flags:**

```
      --direction string   Direction: 'down' (dependencies), 'up' (dependents) (default "down")
  -t, --type string        Filter by dependency type (e.g., tracks, blocks, parent-child)
```

## issuegraph dep relate

Create a loose 'see also' relationship between two issues.

The relates_to link is bidirectional - both issues will reference each other.
This enables knowledge graph connections without blocking or hierarchy.

Examples:
  issuegraph relate bd-abc bd-xyz    # Link two related issues
  issuegraph relate bd-123 bd-456    # Create see-also connection

```
issuegraph dep relate <id1> <id2> [flags]
```

## issuegraph dep remove

Remove a dependency

```
issuegraph dep remove [issue-id] [depends-on-id] [flags]
```

**Aliases:** rm

## issuegraph dep tree

Show dependency tree rooted at the given issue.

By default, shows dependencies (what blocks this issue). Use --direction to control:
  - down: Show dependencies (what blocks this issue) - default
  - up:   Show dependents (what this issue blocks)
  - both: Show full graph in both directions

Examples:
  issuegraph dep tree gt-0iqq                    # Show what blocks gt-0iqq
  issuegraph dep tree gt-0iqq --direction=up     # Show what gt-0iqq blocks
  issuegraph dep tree gt-0iqq --status=open      # Only show open issues
  issuegraph dep tree gt-0iqq --depth=3          # Limit to 3 levels deep

A node reached by two paths is shown ONCE, under the first path that got
there, and a cycle simply ends the descent. --show-all-paths is a deprecated
no-op; use 'issuegraph dep cycles' to find circular dependencies.

--max-rows / ISSUEGRAPH_MAX_ROWS caveat: the tree walk has no query filter to
thread the cap through, so the full tree is always built first and the
node count is checked afterward (post-hoc), not during the walk. The cap is
honored on the --proxied-server route too, which it was not before.

```
issuegraph dep tree [issue-id] [flags]
```

**Flags:**

```
      --direction string   Tree direction: 'down' (dependencies), 'up' (dependents), or 'both'
      --format string      Output format: 'mermaid' for Mermaid.js flowchart
  -d, --max-depth int      Maximum tree depth to display (safety limit) (default 50)
      --max-rows int       Hard upper bound on rows returned. Returns a non-zero exit (code 2) and an error to stderr if exceeded. 0 disables (the default). Overrides ISSUEGRAPH_MAX_ROWS for this invocation. Useful in CI/agent rigs that want a circuit breaker against pathological queries. Honored on both the direct and the --proxied-server route.
      --reverse            Show dependent tree (deprecated: use --direction=up)
      --show-all-paths     Deprecated no-op: accepted and ignored. A node reached by two paths is shown once, under the first.
      --status string      Filter to only show issues with this status (open, in_progress, blocked, deferred, closed)
```

## issuegraph dep unrelate

Remove a relates_to relationship between two issues.

Removes the link in both directions.

Example:
  issuegraph unrelate bd-abc bd-xyz

```
issuegraph dep unrelate <id1> <id2> [flags]
```
