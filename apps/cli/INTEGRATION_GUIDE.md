# Complete Integration Guide: @borg0ai/issuegraph + Claude Code for Web

This guide shows the complete end-to-end setup for using issuegraph (IssueGraph) in Claude Code for Web via the npm package.

## 🎯 Goal

Enable automatic issue tracking with issuegraph in every Claude Code for Web session with zero manual setup.

## 📋 Prerequisites

- A git repository with your project
- Claude Code for Web access

## 🚀 Quick Setup (5 Minutes)

### Step 1: Create SessionStart Hook

Create the file `.claude/hooks/session-start.sh` in your repository:

```bash
#!/bin/bash
# Auto-install issuegraph in every Claude Code for Web session

# Install issuegraph globally from npm
npm install -g @borg0ai/issuegraph

# Initialize issuegraph if not already done
if [ ! -d .issuegraph ]; then
  issuegraph init --quiet
fi

# Show current work
echo ""
echo "📋 Ready work:"
issuegraph ready --limit 5 || echo "No ready work found"
```

### Step 2: Make Hook Executable

```bash
chmod +x .claude/hooks/session-start.sh
```

### Step 3: Update AGENTS.md

Add issuegraph usage instructions to your AGENTS.md file:

```markdown
## Issue Tracking with issuegraph

This project uses issuegraph (IssueGraph) for issue tracking. It's automatically installed in each session via SessionStart hook.

### Finding Work

```bash
# See what's ready to work on
issuegraph ready --json | jq '.[0]'

# Get issue details
issuegraph show <issue-id> --json
```

### Creating Issues

```bash
# Create a new issue
issuegraph create "Task description" -t task -p 1 --json

# Create a bug
issuegraph create "Bug description" -t bug -p 0 --json
```

### Working on Issues

```bash
# Start work (atomic claim)
issuegraph update <issue-id> --claim

# Add a comment
issuegraph comments add <issue-id> "Progress update"

# Close when done
issuegraph close <issue-id> --reason "Description of what was done"
```

### Managing Dependencies

```bash
# Issue A blocks issue B
issuegraph dep add <issue-b> <issue-a>

# Show dependency tree
issuegraph dep tree <issue-id>
```

### Best Practices

1. **Always use --json**: Makes output easy to parse programmatically
2. **Create issues proactively**: When you notice work, file it immediately
3. **Link discovered work**: Use `issuegraph dep add --type discovered-from`
4. **Close with context**: Always provide --reason when closing
5. **Sync changes**: Run `issuegraph dolt push` to push changes to the Dolt remote
```

### Step 4: Commit and Push

```bash
git add .claude/hooks/session-start.sh AGENTS.md
git commit -m "Add issuegraph auto-install for Claude Code for Web"
git push
```

## 🎬 How It Works

### First Session in Claude Code for Web

1. **Session starts** → Claude Code for Web creates fresh Linux VM
2. **Hook runs** → `.claude/hooks/session-start.sh` executes automatically
3. **npm install** → Downloads @borg0ai/issuegraph package from npm
4. **Postinstall** → Downloads native issuegraph binary for platform (~17MB)
5. **issuegraph init** → Sets up the .issuegraph directory and Dolt database
6. **Ready** → `issuegraph` command is available, shows ready work

**Time: ~5-10 seconds**

### Subsequent Sessions

Same process, but:
- `issuegraph init --quiet` sets up Dolt and syncs existing data
- Agent picks up right where it left off

**Time: ~5-10 seconds**

## 💡 Usage Patterns

### Pattern 1: Agent Starts Session

```
Agent: Starting new session...
System: Running SessionStart hook...
System: Installing issuegraph v0.21.5...
System: ✓ issuegraph installed successfully
System:
System: 📋 Ready work:
System: issuegraph-a1b2 [P1] Fix authentication bug
System: issuegraph-f14c [P1] Add user profile page
System:
Agent: I can see there are 2 ready tasks. Let me work on issuegraph-a1b2 (Fix authentication bug).
Agent: [runs] issuegraph update issuegraph-a1b2 --claim
```

### Pattern 2: Discovering New Work

```
Agent: While fixing issuegraph-a1b2, I noticed the password reset flow is also broken.
Agent: [runs] issuegraph create "Fix password reset flow" -t bug -p 1 --json
System: {"id": "issuegraph-3e7a", ...}
Agent: [runs] issuegraph dep add issuegraph-3e7a issuegraph-a1b2 --type discovered-from
Agent: I've filed issuegraph-3e7a for the password reset issue and linked it to the current work.
```

### Pattern 3: Completing Work

```
Agent: I've fixed the authentication bug. Tests are passing.
Agent: [runs] issuegraph close issuegraph-a1b2 --reason "Fixed session validation logic. Added tests."
Agent: [runs] git add . && git commit -m "Fix authentication bug (issuegraph-a1b2)"
Agent: Work completed. Ready for next task.
```

## 🔧 Advanced Configuration

### Custom Installation Path

Install to project instead of global:

```bash
#!/bin/bash
# .claude/hooks/session-start.sh

# Install as project dependency
npm install --save-dev @borg0ai/issuegraph

# Use via npx
npx issuegraph init --quiet

echo "Use 'npx issuegraph' for all issuegraph commands"
```

### Conditional Installation

Only install if not already present:

```bash
#!/bin/bash
# .claude/hooks/session-start.sh

if ! command -v issuegraph &> /dev/null; then
    echo "Installing issuegraph..."
    npm install -g @borg0ai/issuegraph
else
    echo "issuegraph already available: $(issuegraph version)"
fi

# Rest of setup...
```

### Silent Installation

Minimal output for cleaner logs:

```bash
#!/bin/bash
# .claude/hooks/session-start.sh

npm install -g @borg0ai/issuegraph --silent 2>&1 | grep -v "npm WARN"
issuegraph init --quiet 2>&1 | grep -v "already initialized"
```

## 📊 Benefits

### For Agents

- ✅ **Persistent memory**: Issue context survives session resets
- ✅ **Structured planning**: Dependencies create clear work order
- ✅ **Automatic setup**: No manual intervention needed
- ✅ **Dolt-backed**: All issues are version controlled
- ✅ **Fast queries**: `issuegraph ready` finds work instantly

### For Humans

- ✅ **Visibility**: See what agents are working on
- ✅ **Auditability**: Full history of issue changes
- ✅ **Collaboration**: Multiple agents share same issue database
- ✅ **Portability**: Works locally and in cloud sessions
- ✅ **No servers**: Everything is git and SQLite

### vs Markdown TODOs

| Feature | issuegraph Issues | Markdown TODOs |
|---------|-----------|----------------|
| Dependencies | ✅ 4 types | ❌ None |
| Ready work detection | ✅ Automatic | ❌ Manual |
| Status tracking | ✅ Built-in | ❌ Manual |
| History/audit | ✅ Full trail | ❌ Git only |
| Queries | ✅ SQL-backed | ❌ Text search |
| Cross-session | ✅ Persistent | ⚠️ Markdown only |
| Agent-friendly | ✅ JSON output | ⚠️ Parsing required |

## 🐛 Troubleshooting

### "issuegraph: command not found"

**Cause**: SessionStart hook didn't run or installation failed

**Fix**:
```bash
# Manually install
npm install -g @borg0ai/issuegraph

# Verify
issuegraph version
```

### "Database not found"

**Cause**: `issuegraph init` wasn't run

**Fix**:
```bash
issuegraph init
```

### "Merge conflict during sync"

**Cause**: Two sessions modified issues concurrently

**Fix**: Run `issuegraph dolt pull` to resolve via Dolt's merge. See the main issuegraph TROUBLESHOOTING.md for details.

### Slow Installation

**Cause**: Network latency downloading binary

**Optimize**:
```bash
# Use npm cache
npm config set cache ~/.npm-cache

# Or install as dependency (cached by package-lock.json)
npm install --save-dev @borg0ai/issuegraph
```

## 📚 Next Steps

1. **Read the full docs**: https://github.com/borg0ai/dx-issuegraph
2. **Try the quickstart**: `issuegraph quickstart` (interactive tutorial)
3. **Set up MCP**: For local Claude Desktop integration
4. **Explore examples**: https://github.com/borg0ai/dx-issuegraph/tree/main/examples

## 🔗 Resources

- [issuegraph GitHub](https://github.com/borg0ai/dx-issuegraph)
- [npm package](https://www.npmjs.com/package/@borg0ai/issuegraph)
- [Claude Code docs](https://docs.claude.com/claude-code)
- [SessionStart hooks](https://docs.claude.com/claude-code/hooks)

## 💬 Example Agent Prompt

Add this to your project's system prompt or AGENTS.md:

```
You have access to issuegraph (IssueGraph) for issue tracking. It's automatically installed in each session.

WORKFLOW:
1. Start each session: Check `issuegraph ready --json` for available work
2. Choose a task: Pick highest priority with no blockers
3. Start work: `issuegraph update <id> --claim`
4. Work on it: Implement, test, document
5. File new issues: Create issues for any work discovered
6. Link issues: Use `issuegraph dep add` to track relationships
7. Close when done: `issuegraph close <id> --reason "what you did"`
8. Sync changes: Run `issuegraph dolt push` at end of session

ALWAYS:
- Use --json flags for programmatic parsing
- Create issues proactively (don't let work be forgotten)
- Link related issues with dependencies
- Close issues with descriptive reasons
- Run `issuegraph dolt push` at end of sessions

NEVER:
- Use markdown TODOs (use issuegraph instead)
- Work on blocked issues (check `issuegraph show <id>` for blockers)
- Close issues without --reason
- Forget to run `issuegraph dolt push` at end of sessions
```

## 🎉 Success Criteria

After setup, you should see:

✅ New sessions automatically have `issuegraph` available
✅ Agents use `issuegraph` for all issue tracking
✅ Issues persist across sessions via git
✅ Multiple agents can collaborate on same issues
✅ No manual installation required

## 🆘 Support

- [File an issue](https://github.com/borg0ai/dx-issuegraph/issues)
- [Read the FAQ](https://github.com/borg0ai/dx-issuegraph/blob/main/FAQ.md)
- [Check troubleshooting](https://github.com/borg0ai/dx-issuegraph/blob/main/TROUBLESHOOTING.md)
