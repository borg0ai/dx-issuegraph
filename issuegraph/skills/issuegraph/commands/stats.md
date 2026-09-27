---
description: Show project statistics and progress
---

Display statistics about the current issuegraph project.

Use the issuegraph MCP `stats` tool to retrieve project metrics and present them clearly:
- Total issues by status (open, in_progress, blocked, closed)
- Issues by priority level
- Issues by type (bug, feature, task, epic, chore)
- Completion rate
- Recently updated issues

Optionally suggest actions based on the stats:
- High number of blocked issues? Run `/issuegraph:blocked` to investigate
- No in_progress work? Run `/issuegraph:ready` to find tasks
- Many open issues? Consider prioritizing with `/issuegraph:update`
