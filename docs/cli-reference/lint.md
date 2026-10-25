---
title: "issuegraph lint"
description: "Check issues for missing template sections"
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc lint`.

Check issues for missing recommended sections based on issue type.

By default, lints all open issues. Specify issue IDs to lint specific issues.

Section requirements by type:
  bug:      Steps to Reproduce, Acceptance Criteria
  task:     Acceptance Criteria
  feature:  Acceptance Criteria
  epic:     Success Criteria (or Acceptance Criteria)
  chore:    (none)

Additional per-type sections can be required via config; they are ADDITIVE
to the built-ins above (built-in requirements are never relaxed):

  issuegraph config set lint.sections.epic "Standards scorecard, Cost"

Examples:
  issuegraph lint                    # Lint all open issues
  issuegraph lint bd-abc             # Lint specific issue
  issuegraph lint bd-abc bd-def      # Lint multiple issues
  issuegraph lint --type bug         # Lint only bugs
  issuegraph lint --status all       # Lint all issues (including closed)


```
issuegraph lint [issue-id...] [flags]
```

**Flags:**

```
  -s, --status string   Filter by status (default: open, use 'all' for all)
  -t, --type string     Filter by issue type (bug, task, feature, epic, decision, spike, story, chore, milestone)
```
