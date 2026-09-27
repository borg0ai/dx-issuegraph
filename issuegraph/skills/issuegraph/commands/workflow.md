---
description: Show the AI-supervised issue workflow guide
---

Display the issuegraph workflow for AI agents and developers.

# IssueGraph Workflow

IssueGraph is an issue tracker designed for AI-supervised coding workflows. Here's how to use it effectively:

## 1. Find Ready Work
Use `/issuegraph:ready` or the `ready` MCP tool to see tasks with no blockers.

## 2. Claim Your Task
Claim the issue atomically (assignee + `in_progress` in one step):
- Via command: `/issuegraph:update <id> --claim`
- Via MCP tool: `claim` with `issue_id: "<id>"`

## 3. Work on It
Implement, test, and document the feature or fix.

## 4. Discover New Work
As you work, you'll often find bugs, TODOs, or related work:
- Create issues: `/issuegraph:create` or `create` MCP tool
- Link them: Use `dep` MCP tool with `type: "discovered-from"`
- This maintains context and work history

## 5. Complete the Task
Close the issue when done:
- Via command: `/issuegraph:close <id> "Completed: <summary>"`
- Via MCP tool: `close` with reason

## 6. Check What's Unblocked
After closing, check if other work became ready:
- Use `/issuegraph:ready` to see newly unblocked tasks
- Start the cycle again

## Tips
- **Priority levels**: 0=critical, 1=high, 2=medium, 3=low, 4=backlog
- **Issue types**: bug, feature, task, epic, chore
- **Dependencies**: Use `blocks` for hard dependencies, `related` for soft links
- **Auto-sync**: Changes are stored in Dolt and synced via `issuegraph dolt push` / `issuegraph dolt pull`

## Available Commands
- `/issuegraph:ready` - Find unblocked work
- `/issuegraph:create` - Create new issue
- `/issuegraph:show` - Show issue details
- `/issuegraph:update` - Update issue
- `/issuegraph:close` - Close issue
- `/issuegraph:workflow` - Show this guide (you are here!)

## MCP Tools Available
Use these via the issuegraph MCP server:
- `ready`, `list`, `show`, `create`, `claim`, `update`, `close`
- `dep` (manage dependencies), `blocked`, `stats`
- `init` (initialize issuegraph in a project)

For more details, see the beads README at: https://github.com/gastownhall/beads
