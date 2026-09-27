# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**IssueGraph** (command: `issuegraph`) is a Dolt-powered issue tracker for AI-supervised coding workflows. Git integration is optional — see `BEADS_DIR` + `--stealth` for git-free operation. We dogfood our own tool.

**IMPORTANT**: See [AGENTS.md](../AGENTS.md) for complete workflow instructions, issuegraph commands, and development guidelines.

## Architecture Overview

### Three-Layer Design

1. **Storage Layer** (`internal/storage/`)
   - **Dolt** in `storage/dolt/` — version-controlled SQL database with cell-level merge
   - Common types and interfaces in `storage.go`

2. **Database Runtime Layer**
   - Embedded mode runs Dolt in-process through `internal/storage/embeddeddolt/`
   - Server mode uses `internal/doltserver/` and `internal/storage/db/`
   - Proxy and pidfile helpers live under `internal/storage/db/`
   - Storage-facing server adapters live under `internal/storage/doltserver/`

3. **CLI Layer** (`modules/cli/`)
   - Cobra-based commands (one file per command: `create.go`, `list.go`, etc.)
   - Direct database access (embedded mode for standalone, server mode for orchestrator)
   - All commands support `--json` for programmatic use
   - Main entry point in `main.go`

### Storage Architecture

IssueGraph uses **Dolt** as its local storage backend — a version-controlled SQL database:

```
Dolt DB (.issuegraph/dolt/)
    ↕ Dolt commits (automatic per write)
Complete Git-tracked snapshot (.issuegraph/issues.jsonl + manifest.json)
    ↕ User-managed Git commit/pull/push
Other clones of the repository
```

- **Write path**: CLI → Dolt → auto-commit to Dolt history
- **Read path**: Direct SQL queries against Dolt
- **Portability**: the complete local snapshot is tracked by Git; IssueGraph never runs Git network commands or Dolt remote sync
- **Hash-based IDs**: Automatic collision prevention (v0.20+)

Core implementation:
- Dolt storage: `internal/storage/dolt/`
- Embedded runtime: `internal/storage/embeddeddolt/`
- Server runtime: `internal/doltserver/`, `internal/storage/db/`, and `internal/storage/doltserver/`
- Legacy remote sync commands are being removed under RFC 0003

### Key Data Types

See `internal/types/types.go`:
- `Issue`: Core work item (title, description, status, priority, etc.)
- `Dependency`: Four types (blocks, related, parent-child, discovered-from)
- `Label`: Flexible tagging system
- `Comment`: Threaded discussions
- `Event`: Full audit trail

## Development Command Source

Use the canonical [TESTING.md](TESTING.md) for test commands, test design, and
PR-readiness gates. This file should not duplicate command matrices or
version-management workflows.

> **Do NOT** use `go build -o issuegraph` or `go install` directly — they create
> stale binaries that shadow `~/.local/bin/issuegraph`. Always use `make install`.

## Testing

Testing guidance lives in [TESTING.md](TESTING.md). Architecture-specific notes
for Claude are limited to where tests touch agent setup, hooks, or
instruction-file generation.

## Important Notes

- **Always read AGENTS.md first** - it has the complete workflow
- Check for duplicates proactively: `issuegraph duplicates --auto-merge`
- Use `--json` flags for all programmatic use

## Key Files

- **AGENTS.md** - Complete workflow and development guide (READ THIS!)
- **README.md** - User-facing documentation
- **ADVANCED.md** - Advanced features (rename, merge, compaction)
- **docs/core-concepts/labels.md** - Complete label system guide
- **docs/reference/configuration.md** - Configuration system

## When Adding Features

See AGENTS.md "Adding a New Command" and "Adding Storage Features" sections for step-by-step guidance.
