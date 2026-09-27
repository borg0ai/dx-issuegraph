---
title: Junie
description: Set up issuegraph for Junie, the JetBrains AI agent, with a guidelines file and an MCP server configuration
---

How to use issuegraph with Junie (JetBrains AI Agent).

## Setup

### Quick Setup

```bash
issuegraph setup junie
```

This creates:
- **`.junie/guidelines.md`** - Agent instructions for issuegraph workflow
- **`.junie/mcp/mcp.json`** - MCP server configuration

### Verify Setup

```bash
issuegraph setup junie --check
```

## How It Works

1. **Session starts** → Junie reads `.junie/guidelines.md` for workflow context
2. **MCP tools available** → Junie can use issuegraph MCP tools directly
3. **You work** → Use `issuegraph` CLI commands or MCP tools
4. **Session ends** → Run `issuegraph dolt push` to push changes to Dolt remote

## Configuration Files

### Guidelines (`.junie/guidelines.md`)

Contains workflow instructions that Junie reads automatically:
- Core workflow rules
- Command reference
- Issue types and priorities
- MCP tool documentation

### MCP Config (`.junie/mcp/mcp.json`)

<Warning>
`issuegraph setup junie` currently writes an MCP config that invokes `issuegraph mcp`, a
command that does not exist in current issuegraph builds — that config will not
start a server. Until the recipe is fixed, point Junie at the standalone
`beads-mcp` server instead:
</Warning>

```json
{
  "mcpServers": {
    "beads": {
      "command": "uvx",
      "args": ["beads-mcp"]
    }
  }
}
```

See [MCP Server](/integrations/mcp-server) for the server's tool catalog
and other install options (pip/pipx).

## CLI Commands

You can also use the `issuegraph` CLI directly:

### Creating Issues

```bash
# Always include description for context
issuegraph create "Fix authentication bug" \
  --description="Login fails with special characters in password" \
  -t bug -p 1 --json

# Link discovered issues
issuegraph create "Found SQL injection" \
  --description="User input not sanitized in query builder" \
  --deps discovered-from:bd-42 --json
```

### Working on Issues

```bash
# Find ready work
issuegraph ready --json

# Start work
issuegraph update bd-42 --claim --json

# Complete work
issuegraph close bd-42 --reason "Fixed in commit abc123" --json
```

### Querying

```bash
# List open issues
issuegraph list --status open --json

# Show issue details
issuegraph show bd-42 --json

# Check blocked issues
issuegraph blocked --json
```

### Syncing

```bash
# ALWAYS run at session end
issuegraph dolt push
```

## Best Practices

### Always Use `--json`

```bash
issuegraph list --json          # Parse programmatically
issuegraph create "Task" --json # Get issue ID from output
issuegraph show bd-42 --json    # Structured data
```

### Always Include Descriptions

```bash
# Good
issuegraph create "Fix auth bug" \
  --description="Login fails when password contains quotes" \
  -t bug -p 1 --json

# Bad - no context for future work
issuegraph create "Fix auth bug" -t bug -p 1 --json
```

### Link Related Work

```bash
# When you discover issues during work
issuegraph create "Found related bug" \
  --deps discovered-from:bd-current --json
```

### Push Before Session End

```bash
# ALWAYS run before ending
issuegraph dolt push
```

## Troubleshooting

### Guidelines not loaded

```bash
# Check setup
issuegraph setup junie --check

# Reinstall if needed
issuegraph setup junie
```

### MCP tools not available

```bash
# Verify MCP config exists and points at the beads-mcp server
cat .junie/mcp/mcp.json

# Verify the server package is installed
pip show beads-mcp
```

### Changes not syncing

```bash
# Force push
issuegraph dolt push

# Check system health
issuegraph doctor
```

### Database not found

```bash
# Initialize issuegraph
issuegraph init --quiet
```

## Removing Integration

```bash
issuegraph setup junie --remove
```

This removes:
- `.junie/guidelines.md`
- `.junie/mcp/mcp.json`
- Empty `.junie/mcp/` and `.junie/` directories

## See Also

- [MCP Server](/integrations/mcp-server) - MCP server details
- [Claude Code](/integrations/claude-code) - Similar hook-based integration
- [IDE Setup](/getting-started/ide-setup) - Other editors
