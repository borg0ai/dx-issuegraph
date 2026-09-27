# Agent Instructions

This project uses **issuegraph** (IssueGraph) for issue tracking. Run `issuegraph prime` for full workflow context.

## Quick Reference

```bash
issuegraph ready              # Find available work (open, no blockers)
issuegraph blocked            # Show blocked issues and what blocks them
issuegraph list               # List all issues (with blocker annotations)
issuegraph show <id>          # View issue details
issuegraph update <id> --claim  # Claim work (atomic compare-and-swap)
issuegraph close <id>         # Complete work
issuegraph dolt push          # Push to Dolt remote
```

**Dependency status**: `issuegraph ready` and `issuegraph blocked` are the authoritative
sources for whether work is blocked. `issuegraph list` shows active blocker
annotations but use `issuegraph ready`/`issuegraph blocked` for accurate blocking status.

## Agent Warning: Interactive Commands

**DO NOT use `issuegraph edit`** - it opens an interactive editor ($EDITOR) which AI agents cannot use.

Use `issuegraph update` with flags instead:
```bash
issuegraph update <id> --description "new description"
issuegraph update <id> --title "new title"
issuegraph update <id> --design "design notes"
issuegraph update <id> --notes "additional notes"
issuegraph update <id> --acceptance "acceptance criteria"
```

## Landing the Plane (Session Completion)

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds

