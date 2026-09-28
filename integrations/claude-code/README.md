# Claude Code Integration for IssueGraph

Slash command for converting [Claude Code](https://docs.anthropic.com/en/docs/claude-code) plans to issuegraph tasks.

## Prerequisites

```bash
# Install issuegraph
curl -fsSL https://raw.githubusercontent.com/gastownhall/beads/main/scripts/install.sh | bash

# Install hooks (auto-injects workflow context on session start)
issuegraph setup claude
```

## Installation

```bash
cp commands/plan-to-beads.md ~/.claude/commands/
```

Optionally add to `~/.claude/settings.json` under `permissions.allow`:

```json
"Bash(issuegraph:*)"
```

## /plan-to-beads

Converts a Claude Code plan file into an issuegraph epic with tasks.

```
/plan-to-beads                    # Convert most recent plan
/plan-to-beads path/to/plan.md    # Convert specific plan
```

**What it does:**
- Parses plan structure (title, summary, phases)
- Creates an epic for the plan
- Creates tasks from each phase
- Sets up sequential dependencies
- Uses Task agent delegation for context efficiency

**Example output:**
```
Created from: peaceful-munching-spark.md

Epic: Standardize ID Generation (bd-abc)
  ├── Add dependency (bd-def) - ready
  ├── Create ID utility (bd-ghi) - blocked by bd-def
  └── Update schema (bd-jkl) - blocked by bd-ghi

Total: 4 tasks
Run `issuegraph ready` to start.
```

## Related

- `issuegraph prime` - Workflow context (auto-injected via hooks)
- `issuegraph setup claude` - Install/manage Claude Code hooks
- `issuegraph ready` - Find unblocked work

## License

Same as issuegraph (see repository root).
