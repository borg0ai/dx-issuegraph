---
title: "issuegraph assign"
description: "Assign an issue to someone"
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc assign`.

Assign an issue to someone.

Shorthand for 'issuegraph update &lt;id&gt; --assignee &lt;name&gt;'.

Refuses to overwrite another actor's live in_progress claim without --force
(bd-98s5c); issues assigned to a claim.pools alias are exempt, matching
--claim. For a holder-aware transfer prefer
'issuegraph update &lt;id&gt; --if-assignee &lt;holder&gt; -a &lt;new&gt;'.

Examples:
  issuegraph assign bd-123 alice
  issuegraph assign bd-123 ""      # unassign

```
issuegraph assign <id> <name> [flags]
```

**Flags:**

```
      --force   Allow overwriting another actor's live in_progress claim (use only for abandoned claims — crashed agent, expired lease; prefer issuegraph reclaim)
```
