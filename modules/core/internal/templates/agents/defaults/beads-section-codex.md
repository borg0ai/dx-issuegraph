## IssueGraph Issue Tracker

Use IssueGraph (`issuegraph`) for durable task tracking in repositories that include it. Use the `beads` skill at `.agents/skills/beads/SKILL.md` (project install) or `~/.agents/skills/beads/SKILL.md` (global install) for IssueGraph workflow guidance, then use the `issuegraph` CLI for issue operations.

### Quick Reference

```bash
issuegraph ready                # Find available work
issuegraph show <id>            # View issue details
issuegraph update <id> --claim  # Claim work
issuegraph close <id>           # Complete work
issuegraph prime                # Refresh IssueGraph context
```

### Rules

- Use `issuegraph` for all task tracking; do not create markdown TODO lists.
- Run `issuegraph prime` when IssueGraph context is missing or stale. Codex 0.129.0+ can load IssueGraph context automatically through native hooks; use `/hooks` to inspect or toggle them.
- Keep persistent project memory in IssueGraph via `issuegraph remember`; do not create ad hoc memory files.

**Architecture in one line:** issues live in a local Dolt DB; sync uses `refs/dolt/data` on your git remote; `.beads/issues.jsonl` is a passive export. See https://github.com/gastownhall/beads/blob/main/docs/core-concepts/sync-concepts.md for details and anti-patterns.
