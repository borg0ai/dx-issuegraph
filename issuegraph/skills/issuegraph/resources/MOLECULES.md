# Molecules and Wisps Reference

This reference covers issuegraph's molecular chemistry system for reusable work templates and ephemeral workflows.

## The Chemistry Metaphor

issuegraph v0.34.0 introduces a chemistry-inspired workflow system:

| Phase | Name | Storage | Synced? | Use Case |
|-------|------|---------|---------|----------|
| **Solid** | Proto | `.beads/` | Yes | Reusable template (epic with `template` label) |
| **Liquid** | Mol | `.beads/` | Yes | Persistent instance (real issues from template) |
| **Vapor** | Wisp | `.beads-wisp/` | No | Ephemeral instance (operational work, no audit trail) |

**Phase transitions:**
- `spawn` / `pour`: Solid (proto) → Liquid (mol)
- `wisp create`: Solid (proto) → Vapor (wisp)
- `squash`: Vapor (wisp) → Digest (permanent summary)
- `burn`: Vapor (wisp) → Nothing (deleted, no trace)
- `distill`: Liquid (ad-hoc epic) → Solid (proto)

## When to Use Molecules

### Use Protos/Mols When:
- **Repeatable patterns** - Same workflow structure used multiple times (releases, reviews, onboarding)
- **Team knowledge capture** - Encoding tribal knowledge as executable templates
- **Audit trail matters** - Work that needs to be tracked and reviewed later
- **Cross-session persistence** - Work spanning multiple days/sessions

### Use Wisps When:
- **Operational loops** - Patrol cycles, health checks, routine monitoring
- **One-shot orchestration** - Temporary coordination that shouldn't clutter history
- **Diagnostic runs** - Debugging workflows with no archival value
- **High-frequency ephemeral work** - Would create noise in permanent database

**Key insight:** Wisps prevent database bloat from routine operations while still providing structure during execution.

---

## Proto Management

### Creating a Proto

Protos are epics with the `template` label. Create manually or distill from existing work:

```bash
# Manual creation
issuegraph create "Release Workflow" --type epic --label template
issuegraph create "Run tests for {{component}}" --type task
issuegraph dep add task-id epic-id --type parent-child

# Distill from ad-hoc work (extracts template from existing epic)
issuegraph mol distill bd-abc123 --as "Release Workflow" --var version=1.0.0
```

**Proto naming convention:** Use `mol-` prefix for clarity (e.g., `mol-release`, `mol-patrol`).

### Listing Formulas

```bash
issuegraph formula list                 # List all formulas (protos)
issuegraph formula list --json          # Machine-readable
```

### Viewing Proto Structure

```bash
issuegraph mol show mol-release         # Show template structure and variables
issuegraph mol show mol-release --json  # Machine-readable
```

---

## Spawning Molecules

### Basic Spawn (Creates Wisp by Default)

```bash
issuegraph mol spawn mol-patrol                    # Creates wisp (ephemeral)
issuegraph mol spawn mol-feature --pour            # Creates mol (persistent)
issuegraph mol spawn mol-release --var version=2.0 # With variable substitution
```

**Chemistry shortcuts:**
```bash
issuegraph mol pour mol-feature                    # Shortcut for spawn --pour
issuegraph mol wisp mol-patrol                     # Explicit wisp creation
```

### Spawn with Immediate Execution

```bash
issuegraph mol run mol-release --var version=2.0
```

`issuegraph mol run` does three things:
1. Spawns the molecule (persistent)
2. Assigns root issue to caller
3. Pins root issue for session recovery

**Use `mol run` when:** Starting durable work that should survive crashes. The pin ensures `issuegraph ready` shows the work after restart.

### Spawn with Attachments

Attach additional protos in a single command:

```bash
issuegraph mol spawn mol-feature --attach mol-testing --var name=auth
# Spawns mol-feature, then spawns mol-testing and bonds them
```

**Attach types:**
- `sequential` (default) - Attached runs after primary completes
- `parallel` - Attached runs alongside primary
- `conditional` - Attached runs only if primary fails

```bash
issuegraph mol spawn mol-deploy --attach mol-rollback --attach-type conditional
```

---

## Bonding Molecules

### Bond Types

```bash
issuegraph mol bond A B                    # Sequential: B runs after A
issuegraph mol bond A B --type parallel    # Parallel: B runs alongside A
issuegraph mol bond A B --type conditional # Conditional: B runs if A fails
```

### Operand Combinations

| A | B | Result |
|---|---|--------|
| proto | proto | Compound proto (reusable template) |
| proto | mol | Spawn proto, attach to molecule |
| mol | proto | Spawn proto, attach to molecule |
| mol | mol | Join into compound molecule |

### Phase Control in Bonds

By default, spawned protos inherit target's phase. Override with flags:

```bash
# Found bug during wisp patrol? Persist it:
issuegraph mol bond mol-critical-bug wisp-patrol --pour

# Need ephemeral diagnostic on persistent feature?
issuegraph mol bond mol-temp-check bd-feature --wisp
```

### Custom Compound Names

```bash
issuegraph mol bond mol-feature mol-deploy --as "Feature with Deploy"
```

---

## Wisp Lifecycle

### Creating Wisps

```bash
issuegraph mol wisp mol-patrol                       # From proto
issuegraph mol spawn mol-patrol                      # Same (spawn defaults to wisp)
issuegraph mol spawn mol-check --var target=db       # With variables
```

### Listing Wisps

```bash
issuegraph mol wisp list                     # List all wisps
issuegraph mol wisp list --json              # Machine-readable
```

### Ending Wisps

**Option 1: Squash (compress to digest)**
```bash
issuegraph mol squash wisp-abc123                              # Auto-generate summary
issuegraph mol squash wisp-abc123 --summary "Completed patrol" # Agent-provided summary
issuegraph mol squash wisp-abc123 --keep-children              # Keep children, just create digest
issuegraph mol squash wisp-abc123 --dry-run                    # Preview
```

Squash creates a permanent digest issue summarizing the wisp's work, then deletes the wisp children.

**Option 2: Burn (delete without trace)**
```bash
issuegraph mol burn wisp-abc123                    # Delete wisp, no digest
```

Use burn for routine work with no archival value.

### Garbage Collection

```bash
issuegraph mol wisp gc                       # Clean up orphaned wisps
issuegraph mol wisp gc --closed              # Preview closed wisp deletion
issuegraph mol wisp gc --closed --force      # Purge all closed wisps
```

---

## Distilling Protos

Extract a reusable template from ad-hoc work:

```bash
issuegraph mol distill bd-o5xe --as "Release Workflow"
issuegraph mol distill bd-abc --var feature_name=auth-refactor --var version=1.0.0
```

**What distill does:**
1. Loads existing epic and all children
2. Clones structure as new proto (adds `template` label)
3. Replaces concrete values with `{{variable}}` placeholders

**Variable syntax (both work):**
```bash
--var branch=feature-auth      # variable=value (recommended)
--var feature-auth=branch      # value=variable (auto-detected)
```

**Use cases:**
- Team develops good workflow organically, wants to reuse it
- Capture tribal knowledge as executable templates
- Create starting point for similar future work

---

## Cross-Project Dependencies

### Concept

Projects can depend on capabilities shipped by other projects:

```bash
# Project A ships a capability
issuegraph ship auth-api                # Marks capability as available

# Project B depends on it
issuegraph dep add bd-123 external:project-a:auth-api
```

### Shipping Capabilities

```bash
issuegraph ship <capability>            # Ship capability (requires closed issue)
issuegraph ship <capability> --force    # Ship even if issue not closed
issuegraph ship <capability> --dry-run  # Preview
```

**How it works:**
1. Find issue with `export:<capability>` label
2. Validate issue is closed
3. Add `provides:<capability>` label

### Depending on External Capabilities

```bash
issuegraph dep add <issue> external:<project>:<capability>
```

The dependency is satisfied when the external project has a closed issue with `provides:<capability>` label.

**`issuegraph ready` respects external deps:** Issues blocked by unsatisfied external dependencies won't appear in ready list.

---

## Common Patterns

### Pattern: Weekly Review Proto

```bash
# Create proto
issuegraph create "Weekly Review" --type epic --label template
issuegraph create "Review open issues" --type task
issuegraph create "Update priorities" --type task
issuegraph create "Archive stale work" --type task
# Link as children...

# Use each week
issuegraph mol spawn mol-weekly-review --pour
```

### Pattern: Ephemeral Patrol Cycle

```bash
# Patrol proto exists
issuegraph mol wisp mol-patrol

# Execute patrol work...

# End patrol
issuegraph mol squash wisp-abc123 --summary "Patrol complete: 3 issues found, 2 resolved"
```

### Pattern: Feature with Rollback

```bash
issuegraph mol spawn mol-deploy --attach mol-rollback --attach-type conditional
# If deploy fails, rollback automatically becomes unblocked
```

### Pattern: Capture Tribal Knowledge

```bash
# After completing a good workflow organically
issuegraph mol distill bd-release-epic --as "Release Process" --var version=X.Y.Z
# Now team can: issuegraph mol spawn mol-release-process --var version=2.0.0
```

---

## CLI Quick Reference

| Command | Purpose |
|---------|---------|
| `issuegraph formula list` | List available formulas/protos |
| `issuegraph mol show <id>` | Show proto/mol structure |
| `issuegraph mol spawn <proto>` | Create wisp from proto (default) |
| `issuegraph mol spawn <proto> --pour` | Create persistent mol from proto |
| `issuegraph mol run <proto>` | Spawn + assign + pin (durable execution) |
| `issuegraph mol bond <A> <B>` | Combine protos or molecules |
| `issuegraph mol distill <epic>` | Extract proto from ad-hoc work |
| `issuegraph mol squash <mol>` | Compress wisp children to digest |
| `issuegraph mol burn <wisp>` | Delete wisp without trace |
| `issuegraph mol pour <proto>` | Shortcut for `spawn --pour` |
| `issuegraph mol wisp <proto>` | Create ephemeral wisp |
| `issuegraph mol wisp list` | List all wisps |
| `issuegraph mol wisp gc` | Garbage collect orphaned wisps |
| `issuegraph mol wisp gc --closed` | Purge all closed wisps (preview; use --force to delete) |
| `issuegraph ship <capability>` | Publish capability for cross-project deps |

---

## Troubleshooting

**"Proto not found"**
- Check `issuegraph formula list` for available formulas/protos
- Protos need `template` label on the epic

**"Variable not substituted"**
- Use `--var key=value` syntax
- Check proto for `{{key}}` placeholders with `issuegraph mol show`

**"Wisp commands fail"**
- Wisps stored in `.beads-wisp/` (separate from `.beads/`)
- Check `issuegraph mol wisp list` for active wisps

**"External dependency not satisfied"**
- Target project must have closed issue with `provides:<capability>` label
- Use `issuegraph ship <capability>` in target project first
