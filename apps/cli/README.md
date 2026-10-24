# @borg0ai/issuegraph - IssueGraph Issue Tracker

[![npm version](https://img.shields.io/npm/v/@borg0ai/issuegraph)](https://www.npmjs.com/package/@borg0ai/issuegraph)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**Give your coding agent a memory upgrade**

IssueGraph is a lightweight memory system for coding agents, using a graph-based issue tracker. This npm package provides easy installation of the native issuegraph binary for Node.js environments, including Claude Code for Web.

## Installation

```bash
npm install -g @borg0ai/issuegraph
```

Or as a project dependency:

```bash
npm install --save-dev @borg0ai/issuegraph
```

## What is IssueGraph?

IssueGraph is an issue tracker designed specifically for AI coding agents. It provides:

- ✨ **Zero setup** - `issuegraph init` creates project-local database
- 🔗 **Dependency tracking** - Four dependency types (blocks, related, parent-child, discovered-from)
- 📋 **Ready work detection** - Automatically finds issues with no open blockers
- 🤖 **Agent-friendly** - `--json` flags for programmatic integration
- 📦 **Version-controlled** - Dolt database with full history and branching
- 🌍 **Distributed by design** - Share one logical database via Dolt remotes

## Quick Start

After installation, initialize issuegraph in your project:

```bash
issuegraph init
```

Then tell your AI agent to use issuegraph for task tracking instead of markdown:

```bash
echo "Use 'issuegraph' commands for issue tracking instead of markdown TODOs" >> AGENTS.md
```

Your agent will automatically:
- Create and track issues during work
- Manage dependencies between tasks
- Find ready work with `issuegraph ready`
- Keep long-term context across sessions

## Common Commands

```bash
# Find ready work
issuegraph ready --json

# Create an issue
issuegraph create "Fix bug" -t bug -p 1

# Show issue details
issuegraph show issuegraph-a1b2

# List all issues
issuegraph list --json

# Start work
issuegraph update issuegraph-a1b2 --claim

# Add dependency
issuegraph dep add issuegraph-f14c issuegraph-a1b2

# Close issue
issuegraph close issuegraph-a1b2 --reason "Fixed"
```

## Claude Code for Web Integration

To auto-install issuegraph in Claude Code for Web sessions, add to your SessionStart hook:

```bash
# .claude/hooks/session-start.sh
npm install -g @borg0ai/issuegraph
issuegraph init --quiet
```

This ensures issuegraph is available in every new session without manual setup.

## Platform Support

This package downloads the appropriate native binary for your platform:

- **macOS**: darwin-amd64, darwin-arm64
- **Linux**: linux-amd64, linux-arm64
- **Windows**: windows-amd64

## Full Documentation

For complete documentation, see the [issuegraph GitHub repository](https://github.com/borg0ai/dx-issuegraph):

- [Complete README](https://github.com/borg0ai/dx-issuegraph#readme)
- [Quick Start (documentation site)](https://github.com/borg0ai/dx-issuegraph/tree/main/docs/getting-started/quickstart) · [in-repo source](https://github.com/borg0ai/dx-issuegraph/blob/main/docs/getting-started/quickstart.md)
- [Installation Guide](https://github.com/borg0ai/dx-issuegraph/blob/main/docs/getting-started/installation.md)
- [FAQ](https://github.com/borg0ai/dx-issuegraph/blob/main/docs/reference/faq.md)
- [Troubleshooting](https://github.com/borg0ai/dx-issuegraph/blob/main/docs/reference/troubleshooting.md)

## Why npm Package vs WASM?

This npm package wraps the native issuegraph binary rather than using WebAssembly because:

- ✅ Full SQLite support (no custom VFS needed)
- ✅ All features work identically to native issuegraph
- ✅ Better performance (native vs WASM overhead)
- ✅ Simpler maintenance

## License

MIT - See [LICENSE](LICENSE) for details.

## Support

- [GitHub Issues](https://github.com/borg0ai/dx-issuegraph/issues)
- [Documentation](https://github.com/borg0ai/dx-issuegraph)
