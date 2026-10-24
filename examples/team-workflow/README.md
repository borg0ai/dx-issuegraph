# Team Workflow Example

This example demonstrates how to use issuegraph for team collaboration with shared repositories.

## Problem

When working as a team on a shared repository, you want to:
- Track issues collaboratively
- Keep everyone in sync via git
- Handle protected main branches
- Maintain clean git history

## Solution

Use `issuegraph init --team` to set up team collaboration with automatic sync and optional protected branch support.

## Setup

### Step 1: Initialize Team Workflow

```bash
# In your shared repository
cd my-project

# Run the team setup wizard
issuegraph init --team
```

The wizard will:
1. ✅ Detect your git configuration
2. ✅ Ask if main branch is protected
3. ✅ Configure sync branch (if needed)
4. ✅ Set up automatic sync
5. ✅ Enable team mode

### Step 2: Protected Branch Configuration

If your main branch is protected (GitHub/GitLab), the wizard will:
- Create a separate `issuegraph-metadata` branch for issue updates
- Configure issuegraph to commit to this branch automatically
- Set up periodic PR workflow for merging to main

### Step 3: Team Members Join

Other team members just need to:

```bash
# Clone the repository
git clone https://github.com/org/project.git
cd project

# Initialize issuegraph (auto-imports existing issues)
issuegraph init

# Start working!
issuegraph ready
```

## How It Works

### Direct Commits (No Protected Branch)

If main isn't protected:

```bash
# Create issue
issuegraph create "Implement feature X" -p 1

# Dolt server auto-commits to main
# (or run 'issuegraph dolt push' manually)

# Pull to see team's issues
git pull
issuegraph list
```

### Protected Branch Workflow

If main is protected:

```bash
# Create issue
issuegraph create "Implement feature X" -p 1

# Auto-commits to issuegraph-metadata branch
# (or run 'issuegraph dolt push' manually)

# Push issuegraph-metadata
git push origin issuegraph-metadata

# Periodically: merge issuegraph-metadata to main via PR
```

## Configuration

The wizard configures:

```yaml
team:
  enabled: true
  sync_branch: issuegraph-metadata  # or main if not protected

dolt:
  auto-commit: on
```

### Manual Configuration

```bash
# Enable team mode
issuegraph config set team.enabled true

# Set sync branch
issuegraph config set team.sync_branch issuegraph-metadata

# Enable auto-commit
issuegraph config set dolt.auto-commit on
```

## Example Workflows

### Scenario 1: Unprotected Main

```bash
# Alice creates an issue
issuegraph create "Fix authentication bug" -p 1

# Auto-commits and pushes to main
# (auto-sync enabled)

# Bob pulls changes
git pull
issuegraph list  # Sees Alice's issue

# Bob claims it
issuegraph update issuegraph-abc --claim

# Auto-commits Bob's update
# Alice pulls and sees Bob is working on it
```

### Scenario 2: Protected Main

```bash
# Alice creates an issue
issuegraph create "Add new API endpoint" -p 1

# Auto-commits to issuegraph-metadata
git push origin issuegraph-metadata

# Bob pulls issuegraph-metadata
git pull origin issuegraph-metadata
issuegraph list  # Sees Alice's issue

# Later: merge issuegraph-metadata to main via PR
git checkout main
git pull origin main
git merge issuegraph-metadata
# Create PR, get approval, merge
```

## Team Workflows

### Daily Standup

```bash
# See what everyone's working on
issuegraph list --status in_progress

# See what's ready for work
issuegraph ready

# See recently closed issues
issuegraph list --status closed --limit 10
```

### Sprint Planning

```bash
# Create sprint issues
issuegraph create "Implement user auth" -p 1
issuegraph create "Add profile page" -p 1
issuegraph create "Fix responsive layout" -p 2

# Assign to team members
issuegraph update issuegraph-abc --assignee alice
issuegraph update issuegraph-def --assignee bob

# Track dependencies
issuegraph dep add issuegraph-def issuegraph-abc --type blocks
```

### PR Integration

```bash
# Create issue for PR work
issuegraph create "Refactor auth module" -p 1

# Work on it
issuegraph update issuegraph-abc --claim

# Open PR with issue reference
git push origin feature-branch
# PR title: "feat: refactor auth module (issuegraph-abc)"

# Close when PR merges
issuegraph close issuegraph-abc --reason "PR #123 merged"
```

## Sync Strategies

### Auto-Sync (Recommended)

The Dolt server commits and pushes automatically when auto-commit is enabled:

```bash
issuegraph config set dolt.auto-commit on
issuegraph dolt start
```

Benefits:
- ✅ Always in sync
- ✅ No manual intervention
- ✅ Real-time collaboration

### Manual Sync

Push and pull when you want:

```bash
issuegraph dolt push  # Push local changes to remote
issuegraph dolt pull  # Pull remote changes locally
```

Benefits:
- ✅ Full control
- ✅ Batch updates
- ✅ Review before push

## Conflict Resolution

Hash-based IDs prevent most conflicts. Dolt handles merges natively using three-way merge, similar to git. If conflicts occur during `issuegraph dolt pull`:

```bash
# View conflicts
issuegraph sql "SELECT * FROM dolt_conflicts"

# Resolve by accepting ours or theirs
issuegraph sql "CALL dolt_conflicts_resolve('--ours')"
# OR
issuegraph sql "CALL dolt_conflicts_resolve('--theirs')"

# Complete the sync
issuegraph dolt push
```

## Protected Branch Best Practices

### For Protected Main:

1. **Create issuegraph-metadata branch**
   ```bash
   git checkout -b issuegraph-metadata
   git push origin issuegraph-metadata
   ```

2. **Configure protection rules**
   - Allow direct pushes to issuegraph-metadata
   - Require PR for main

3. **Periodic PR workflow**
   ```bash
   # Once per day/sprint
   git checkout main
   git pull origin main
   git checkout issuegraph-metadata
   git pull origin issuegraph-metadata
   git checkout main
   git merge issuegraph-metadata
   # Create PR, get approval, merge
   ```

4. **Keep issuegraph-metadata clean**
   ```bash
   # After PR merges
   git checkout issuegraph-metadata
   git rebase main
   git push origin issuegraph-metadata --force-with-lease
   ```

## Common Questions

### Q: How do team members see each other's issues?

A: Issues are stored in Dolt, which supports distributed sync. Use `issuegraph dolt pull` to fetch and `issuegraph dolt push` to share changes.

```bash
issuegraph dolt pull
issuegraph list  # See everyone's issues
```

### Q: What if two people create issues at the same time?

A: Hash-based IDs prevent collisions. Even if created simultaneously, they get different IDs.

### Q: How do I disable auto-sync?

A: Turn it off:

```bash
issuegraph config set dolt.auto-commit off

# Sync manually
issuegraph dolt push
issuegraph dolt pull
```

### Q: Can we use different sync branches per person?

A: Not recommended. Use a single shared branch for consistency. If needed:

```bash
issuegraph config set sync.branch my-custom-branch
```

### Q: What about CI/CD integration?

A: Add to your CI pipeline:

```bash
# In .github/workflows/main.yml
- name: Sync issuegraph issues
  run: |
    issuegraph dolt push
    git push origin issuegraph-metadata
```

## Troubleshooting

### Issue: Server not committing

Check server status:

```bash
issuegraph doctor
```

Verify config:

```bash
issuegraph config get dolt.auto-commit
```

Restart Dolt server:

```bash
issuegraph dolt stop
issuegraph dolt start
```

### Issue: Merge conflicts

Dolt handles merges natively. If conflicts occur during sync:

```bash
issuegraph sql "SELECT * FROM dolt_conflicts"
issuegraph sql "CALL dolt_conflicts_resolve('--ours')"
issuegraph dolt push
```

See [GIT_INTEGRATION.md](../../docs/reference/git-integration.md) for details.

### Issue: Issues not syncing

Manually sync:

```bash
issuegraph dolt push
issuegraph dolt pull
```

Check for conflicts:

```bash
git status
issuegraph validate --checks=conflicts
```

## See Also

- [Protected Branch Setup](../protected-branch/)
- [Contributor Workflow](../contributor-workflow/)
- [Multi-Repo Migration Guide](../../docs/multi-agent/multi-repo-migration.md)
- [Git Integration Guide](../../docs/reference/git-integration.md)
