# OSS Contributor Workflow Example

This example demonstrates how to use issuegraph's contributor workflow to keep your planning issues separate from upstream PRs when contributing to open-source projects.

## Problem

When contributing to OSS projects, you want to:
- Track your planning, todos, and design notes
- Keep experimental work organized
- **NOT** pollute upstream PRs with your personal planning issues

## Solution

Use `issuegraph init --contributor` to set up a separate planning repository that never gets committed to the upstream project.

## Setup

### Step 1: Fork and Clone

```bash
# Fork the project on GitHub, then clone your fork
git clone https://github.com/YOUR_USERNAME/project.git
cd project

# Add upstream remote (important for fork detection!)
git remote add upstream https://github.com/ORIGINAL_OWNER/project.git
```

### Step 2: Initialize IssueGraph with Contributor Wizard

```bash
# Run the contributor setup wizard
issuegraph init --contributor
```

The wizard will:
1. ✅ Detect that you're in a fork (checks for 'upstream' remote)
2. ✅ Prompt you to create a planning repo (`~/.beads-planning` by default)
3. ✅ Configure auto-routing so your planning stays separate
4. ✅ Initialize the planning repo with git

### Step 3: Start Working

```bash
# Create a planning issue
issuegraph create "Plan how to fix bug X" -p 2

# This issue goes to ~/.beads-planning automatically!
```

## How It Works

### Auto-Routing

When you create issues as a contributor:

```bash
issuegraph create "Fix authentication bug" -p 1
```

IssueGraph automatically routes this to your planning repo (`~/.beads-planning/.beads/`), not the current repo.

### Viewing Issues

```bash
# See all issues (from both repos)
issuegraph list

# See only current repo issues
issuegraph list --source-repo .

# See only planning issues
issuegraph list --source-repo ~/.beads-planning
```

### Discovered Work

When you discover work while implementing:

```bash
# The new issue inherits source_repo from parent
issuegraph create "Found edge case in auth" -p 1 --deps discovered-from:bd-42
```

### Committing Code (Not Planning)

Your code changes get committed to the fork, but planning issues stay separate:

```bash
# Only commits to fork (not planning repo)
git add src/auth.go
git commit -m "Fix: authentication bug"
git push origin my-feature-branch
```

Your planning issues in `~/.beads-planning` **never appear in PRs**.

## Example Workflow

```bash
# 1. Create fork and clone
git clone https://github.com/you/upstream-project.git
cd upstream-project
git remote add upstream https://github.com/upstream/upstream-project.git

# 2. Run contributor setup
issuegraph init --contributor
# Wizard detects fork ✓
# Creates ~/.beads-planning ✓
# Configures auto-routing ✓

# 3. Plan your work (routes to planning repo)
issuegraph create "Research how auth module works" -p 2
issuegraph create "Design fix for bug #123" -p 1
issuegraph ready  # Shows planning issues

# 4. Implement (commit code only)
git checkout -b fix-auth-bug
# ... make changes ...
git add . && git commit -m "Fix: auth bug"

# 5. Track discovered work (stays in planning repo)
issuegraph create "Found related issue in logout" -p 2 --deps discovered-from:bd-abc

# 6. Push code (planning never included)
git push origin fix-auth-bug
# Create PR on GitHub - zero planning pollution!

# 7. Clean up after PR merges
issuegraph close bd-abc --reason "PR merged"
```

## Configuration

The wizard configures these settings in `.beads/beads.db`:

```yaml
routing:
  mode: auto
  contributor: ~/.beads-planning
  maintainer: .
```

### Manual Configuration

If you prefer manual setup:

```bash
# Initialize issuegraph normally
issuegraph init

# Configure routing
issuegraph config set routing.mode auto
issuegraph config set routing.contributor ~/.beads-planning
issuegraph config set routing.maintainer .
```

### Legacy Configuration (Deprecated)

Older versions used `contributor.*` keys. These still work for backward compatibility:

```bash
# Old keys (deprecated but functional)
issuegraph config set contributor.planning_repo ~/.beads-planning
issuegraph config set contributor.auto_route true

# New keys (preferred)
issuegraph config set routing.mode auto
issuegraph config set routing.contributor ~/.beads-planning
```

## Multi-Repository View

IssueGraph aggregates issues from multiple repos:

```bash
# List issues from all configured repos
issuegraph list

# Filter by source repository
issuegraph list --source-repo .                    # Current repo only
issuegraph list --source-repo ~/.beads-planning    # Planning repo only
```

## Benefits

✅ **Clean PRs** - No personal todos in upstream contributions
✅ **Private planning** - Experimental work stays local
✅ **Git ledger** - Everything is version controlled
✅ **Unified view** - See all issues with `issuegraph list`
✅ **Auto-routing** - No manual sorting needed

## Common Questions

### Q: What if I want some issues in the upstream repo?

A: Override auto-routing with `--repo` flag:

```bash
issuegraph create "Document new API" -p 2 --repo .
```

### Q: Can I change the planning repo location?

A: Yes, configure it:

```bash
issuegraph config set routing.contributor /path/to/my-planning
```

### Q: What if I have push access to upstream?

A: The wizard will ask if you want a planning repo anyway. You can say "no" to store everything in the current repo.

### Q: How do I disable auto-routing?

A: Change routing mode to explicit:

```bash
issuegraph config set routing.mode explicit
issuegraph config set routing.default .  # Default to current repo
```

## See Also

- [Multi-Repo Migration Guide](../../docs/multi-agent/multi-repo-migration.md)
- [Team Workflow Example](../team-workflow/)
- [Protected Branch Setup](../protected-branch/)
