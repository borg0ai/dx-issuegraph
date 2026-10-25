---
title: Installation
description: Install the issuegraph CLI with Homebrew on macOS, Nix on Linux, or Scoop on Windows
---

IssueGraph ships through exactly three package managers, one per OS. Each installs
the matching archive from the tagged GitHub Release.

## Components Overview

| Component | What It Is | When You Need It |
|-----------|------------|------------------|
| **issuegraph CLI** | Core command-line tool | Always - this is the foundation |
| **Claude Code Plugin** | Slash commands + enhanced UX | Optional - if you want `/issuegraph:ready`, `/issuegraph:create` commands |

The Plugin enhances Claude Code with slash commands but *requires* the CLI installed.

**Important:** IssueGraph is installed system-wide, not cloned into your project. The `.issuegraph/` directory in your project only contains the issue database.

## Install

### macOS (Homebrew)

```bash
brew install systembugtj/luban-homebrew/issuegraph
```

### Linux (Nix)

Supported systems: `x86_64-linux`, `aarch64-linux`.

```bash
nix profile install github:systembugtj/cangjie-nix#issuegraph
```

### Windows (Scoop)

```powershell
scoop bucket add shennong https://github.com/systembugtj/shennong-bucket
scoop install shennong/issuegraph
```

Verify with `issuegraph version`.

## IDE and Editor Integrations

### CLI + Hooks (Recommended)

The recommended approach for Claude Code, Cursor, Windsurf, and other editors with shell access:

```bash
# 1. Install issuegraph CLI (see Install above)

# 2. Initialize in your project
cd your-project
issuegraph init --quiet

# 3. Setup editor integration (choose one)
issuegraph setup claude   # Claude Code - installs SessionStart hooks
issuegraph setup copilot  # GitHub Copilot CLI - creates .copilot-plugin/plugin.json + .github/copilot-instructions.md
issuegraph setup cursor   # Cursor IDE - creates .cursor/rules/issuegraph.mdc
issuegraph setup aider    # Aider - creates .aider.conf.yml
issuegraph setup codex    # Codex CLI - installs IssueGraph skill, AGENTS.md guidance, and native hooks
issuegraph setup factory  # Factory.ai Droid - creates/updates AGENTS.md
issuegraph setup mux      # Mux - creates/updates AGENTS.md
```

**How it works:**
- `issuegraph init` creates or updates `AGENTS.md` and installs project Claude/Codex integrations by default unless you use `--skip-agents` or `--stealth`
- Editor hooks/rules inject `issuegraph prime` automatically on session start
- Codex 0.129.0+ uses native `/hooks`: SessionStart injects `issuegraph prime`, compact hooks mark context stale, and the next prompt after compaction refreshes IssueGraph context once
- `issuegraph prime` provides ~1-2k tokens of workflow context
- You use `issuegraph` CLI commands directly
- Git hooks (installed by `issuegraph init`) refresh exports and legacy fallbacks; `issuegraph dolt push/pull` syncs the database
- `issuegraph onboard` prints the small manual snippet for unsupported agents or custom instruction files

**Why this is recommended:**
- **Context efficient** - ~1-2k tokens vs 10-50k for MCP tool schemas
- **Lower latency** - Direct CLI calls, no MCP protocol overhead
- **Universal** - Works with any editor that has shell access

**Verify installation:** every recipe supports a check flag, e.g. `issuegraph setup claude --check` or `issuegraph setup copilot --check`.

### Claude Code Plugin (Optional)

For enhanced UX with slash commands:

```bash
# In Claude Code
/plugin marketplace add borg0ai/dx-issuegraph
/plugin install issuegraph
# Restart Claude Code
```

The plugin adds:
- Slash commands: `/issuegraph:ready`, `/issuegraph:create`, `/issuegraph:show`, `/issuegraph:update`, `/issuegraph:close`, etc.
- Task agent for autonomous execution

See [Claude Code Plugin](/integrations/claude-code-plugin) for complete plugin documentation.

### GitHub Copilot

For the **GitHub Copilot CLI** terminal integration:

```bash
issuegraph setup copilot         # Install project Copilot plugin + repository instructions
issuegraph setup copilot --check # Verify the project integration files exist
```

This setup is currently project-scoped only. It writes `.copilot-plugin/plugin.json` and `.github/copilot-instructions.md`; there is no separate `--global` or `--project` mode for Copilot today, and it does not manage `~/.copilot/...` paths. See [Copilot CLI](/integrations/copilot-cli) for the full guide.

## Verifying Installation

```bash
issuegraph version
issuegraph help
```

## Troubleshooting

For additional troubleshooting, see [Troubleshooting](/reference/troubleshooting).

### `issuegraph: command not found`

Check that the package manager's bin directory is on your PATH, then reinstall with the command for your OS above.

## Updating issuegraph

Upgrade checklist:

1. With your current `issuegraph`, sync remote-backed databases before installing the
   new binary:
   `issuegraph dolt push`
   `issuegraph dolt pull`
2. Back up before migration:
   `issuegraph export --all -o .issuegraph/backup/pre-migrate-$(date +%Y%m%d).jsonl`
3. Upgrade using the command for your install method below.
4. After upgrading:
   `issuegraph info --whats-new`
   `issuegraph hooks install`
   `issuegraph version`
5. If crossing a schema migration on a remote-backed database, only the
   designated migrator runs:
   `issuegraph migrate`
   `issuegraph dolt push`

Other clones should install the new binary and run `issuegraph bootstrap`, not
independently migrate. For the full procedure, see [Upgrading](/getting-started/upgrading).

### Upgrade command per OS

| OS | Command |
|----|---------|
| macOS | `brew upgrade issuegraph` |
| Linux | `nix profile upgrade issuegraph` |
| Windows | `scoop update issuegraph` |

Prereleases (e.g. release candidates) are published only as GitHub prereleases
and are not pushed to the package channels.

For post-upgrade steps (hooks, migrations), see [Upgrading](/getting-started/upgrading).

## Uninstalling

Remove the binary with `brew uninstall issuegraph`, `nix profile remove issuegraph`, or
`scoop uninstall issuegraph`. To completely remove IssueGraph from a repository, see
[Uninstalling](/recovery/uninstalling).

## Next Steps

After installation:

1. **Initialize a project**: `cd your-project && issuegraph init`
2. **Learn the basics**: See [Quick Start](/getting-started/quickstart)
3. **Configure your agent**: See [IDE Setup](/getting-started/ide-setup), or run `issuegraph setup --list`
4. **Explore examples**: Check out the [examples/](https://github.com/borg0ai/dx-issuegraph/tree/main/examples) directory
