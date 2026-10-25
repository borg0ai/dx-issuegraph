# GitHub Copilot Instructions for IssueGraph

## Project Overview

**issuegraph** (command: `issuegraph`) is a Dolt-powered issue tracker designed for AI-supervised coding workflows. Git integration is optional. We dogfood our own tool for all task tracking.

**Key Features:**
- Dependency-aware issue tracking
- Auto-sync via Dolt-native replication
- AI-optimized CLI with JSON output
- Dolt server mode for background operations
- MCP server integration for Claude and other AI assistants

## Tech Stack

- **Language**: Go 1.21+
- **Storage**: Dolt (version-controlled SQL database)
- **CLI Framework**: Cobra
- **Testing**: Go standard testing + table-driven tests
- **CI/CD**: GitHub Actions
- **MCP Server**: Python (integrations/issuegraph-mcp/)

## Coding Guidelines

### Testing
- Always write tests for new features
- Use `t.TempDir() in Go tests` to avoid polluting production database
- Run `go test -short ./...` before committing
- Never create test issues in production DB (use temporary DB)

### Code Style
- Run `make ci-pr-lint` before committing changes to Go or lint-controlled files
- Follow existing patterns in `apps/cli/` for new commands
- Add `--json` flag to all commands for programmatic use
- Update docs when changing behavior

### Git Workflow
- Install git hooks: `issuegraph hooks install`
- Use `issuegraph dolt push` / `issuegraph dolt pull` for remote sync
- Before implementing related work, opening a PR, or merging/closing a PR, run:
  `scripts/pr-preflight.sh --search "<topic>" --repo borg0ai/dx-issuegraph` or
  `scripts/pr-preflight.sh <pr-number> --repo borg0ai/dx-issuegraph`
- External contributor PRs have priority: build on them when possible, preserve
  tests and attribution, and never close or replace them silently.

## Issue Tracking with issuegraph

**CRITICAL**: This project uses **issuegraph** for ALL task tracking. Do NOT create markdown TODO lists.

### Essential Commands

```bash
# Find work
issuegraph ready --json                    # Unblocked issues
issuegraph stale --days 30 --json          # Forgotten issues

# Create and manage (ALWAYS include --description)
issuegraph create "Title" --description="Detailed context" -t bug|feature|task -p 0-4 --json
issuegraph update <id> --claim --json
issuegraph close <id> --reason "Done" --json

# Search
issuegraph list --status open --priority 1 --json
issuegraph show <id> --json

# Sync (if remote configured)
issuegraph dolt push                   # Push to Dolt remote
issuegraph dolt pull                   # Pull from Dolt remote
```

### Workflow

1. **Check ready work**: `issuegraph ready --json`
2. **Claim task**: `issuegraph update <id> --claim`
3. **Work on it**: Implement, test, document
4. **Discover new work?** `issuegraph create "Found bug" --description="What was found and why" -p 1 --deps discovered-from:<parent-id> --json`
5. **Complete**: `issuegraph close <id> --reason "Done" --json`
6. **Sync**: `issuegraph dolt push` (push to Dolt remote if configured)

**IMPORTANT**: Always include `--description` when creating issues. Issues without descriptions lack context for future work.

### Priorities

- `0` - Critical (security, data loss, broken builds)
- `1` - High (major features, important bugs)
- `2` - Medium (default, nice-to-have)
- `3` - Low (polish, optimization)
- `4` - Backlog (future ideas)

## Project Structure

```
issuegraph/
├── apps/cli/              # CLI commands (add new commands here)
├── internal/
│   ├── types/           # Core data types
│   └── storage/         # Storage layer
│       └── dolt/        # Dolt implementation
├── integrations/
│   └── issuegraph-mcp/       # MCP server (Python)
├── examples/            # Integration examples
├── docs/                # Documentation
└── .issuegraph/
    └── dolt/            # Dolt database (source of truth)
```

## Available Resources

### MCP Server (Recommended)
Use the issuegraph MCP server for native function calls instead of shell commands:
- Install: `pip install issuegraph-mcp`
- Functions: `mcp__issuegraph__ready()`, `mcp__issuegraph__create()`, etc.
- See `integrations/issuegraph-mcp/README.md`

### Scripts
- `./scripts/update-versions.sh <version>` - Update all version files (no git operations)
- Releases run through the `shipify` skill

### Key Documentation
- **AGENTS.md** - Comprehensive AI agent guide (detailed workflows, advanced features)
- **AGENT_INSTRUCTIONS.md** - Development procedures, testing, releases
- **README.md** - User-facing documentation
- **docs/CLI_REFERENCE.md** - Complete command reference

## Important Rules

- ✅ Use issuegraph for ALL task tracking
- ✅ Always use `--json` flag for programmatic use
- ✅ Use `issuegraph dolt push` / `issuegraph dolt pull` for remote sync
- ✅ Test with `t.TempDir() in Go tests`
- ❌ Do NOT create markdown TODO lists
- ❌ Do NOT create test issues in production DB
- ❌ Do NOT manually modify `.issuegraph/dolt/`

---

**For detailed workflows and advanced features, see [AGENTS.md](../AGENTS.md)**
