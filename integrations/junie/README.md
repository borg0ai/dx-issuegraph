# Junie Integration for IssueGraph

Integration for [Junie](https://www.jetbrains.com/junie/) (JetBrains AI Agent) with issuegraph issue tracking.

## Prerequisites

```bash
# Install issuegraph
curl -fsSL https://raw.githubusercontent.com/gastownhall/beads/main/scripts/install.sh | bash

# Initialize issuegraph in your project
issuegraph init
```

## Installation

```bash
issuegraph setup junie
```

This creates:
- `.junie/guidelines.md` - Agent instructions for issuegraph workflow
- `.junie/mcp/mcp.json` - MCP server configuration

## What Gets Installed

### Guidelines (`.junie/guidelines.md`)

Junie automatically reads this file on session start. It contains:
- Core workflow rules for using issuegraph
- Command reference for the `issuegraph` CLI
- Issue types and priorities
- MCP tool documentation

### MCP Config (`.junie/mcp/mcp.json`)

Configures the issuegraph MCP server so Junie can use issuegraph tools directly:

```json
{
  "mcpServers": {
    "beads": {
      "command": "bd",
      "args": ["mcp"]
    }
  }
}
```

## Usage

Once installed, Junie will:
1. Read workflow instructions from `.junie/guidelines.md`
2. Have access to issuegraph MCP tools for direct issue management
3. Be able to use `issuegraph` CLI commands

### MCP Tools Available

- `mcp_beads_ready` - Find tasks ready for work
- `mcp_beads_list` - List issues with filters
- `mcp_beads_show` - Show issue details
- `mcp_beads_create` - Create new issues
- `mcp_beads_update` - Update issue status/priority
- `mcp_beads_close` - Close completed issues
- `mcp_beads_dep` - Manage dependencies
- `mcp_beads_blocked` - Show blocked issues
- `mcp_beads_stats` - Get issue statistics

## Verification

```bash
issuegraph setup junie --check
```

## Removal

```bash
issuegraph setup junie --remove
```

## Related

- `issuegraph prime` - Get full workflow context
- `issuegraph ready` - Find unblocked work
- `issuegraph dolt push` - Push issue changes to Dolt remote (run at session end)

## License

Same as issuegraph (see repository root).
