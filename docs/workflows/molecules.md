---
title: Molecules
description: Molecules are epics whose children flow through issuegraph ready as ordered steps; covers creating, executing, bonding, and the molecule lifecycle.
---

Molecules are work graphs: epics whose children flow through `issuegraph ready` as
dependency-ordered steps. They are usually instantiated from formulas, but a
formula is optional — any epic with children is a molecule.

## What is a Molecule?

A molecule is a persistent instance of a proto (a cooked formula):
- Contains steps with dependencies
- Persistent beads in the issue database, synced like any other bead
- Steps map to issues with parent-child relationships

Under the hood, **a molecule is just an epic** — a parent bead with children —
plus workflow semantics:

| Term | Meaning | When to use |
|------|---------|-------------|
| **Epic** | Parent issue with children | General term for hierarchical work |
| **Molecule** | Epic with execution intent | When discussing workflow traversal |
| **Proto** | Epic with the `template` label | Reusable pattern (optional) |

Protos and formulas are optional layers for reusable patterns and complex
composition — most work needs only epics and dependencies.

## Creating Molecules

### From a Formula

```bash
# Cook the formula into a proto, then pour the proto into a molecule
issuegraph cook release.formula.toml
issuegraph mol pour release --var version=1.0.0
```

This creates:
- Parent issue: `bd-xyz` (the molecule root)
- Child issues: `bd-xyz.1`, `bd-xyz.2`, etc. (the steps)

### Without a Formula

Create the epic and wire the dependencies directly:

```bash
issuegraph create "Feature X" -t epic
issuegraph create "Design" -t task --parent <epic-id>
issuegraph create "Implement" -t task --parent <epic-id>
issuegraph create "Test" -t task --parent <epic-id>
issuegraph dep add <implement-id> <design-id>   # implement needs design
issuegraph dep add <test-id> <implement-id>     # test needs implement
```

If the epic carries a size/effort label, see [Labels](/core-concepts/labels) for keeping it off the steps.

If an ad-hoc epic turns out to be worth repeating, extract a reusable formula
from it with `issuegraph mol distill <epic-id> <formula-name>`.

### Finding Molecules

```bash
issuegraph mol current           # Where you are in the molecule you're working
issuegraph mol stale             # Complete-but-still-open molecules
issuegraph mol wisp list         # Ephemeral molecules (wisps)
```

### Viewing a Molecule

```bash
issuegraph mol show <molecule-id>             # Structure and variables
issuegraph mol show <molecule-id> --parallel  # Highlight steps that can run concurrently
issuegraph dep tree <molecule-id>             # Shows full hierarchy
```

## Working with Molecules

### The Execution Model

An agent picks up a molecule and executes ready children in parallel until
everything closes:

```
epic-root (assigned to agent)
├── child.1 (no deps → ready)      ← execute in parallel
├── child.2 (no deps → ready)      ← execute in parallel
├── child.3 (needs child.1) → blocked until child.1 closes
└── child.4 (needs child.2, child.3) → blocked until both close
```

**Children are parallel by default.** Only explicit dependencies create
sequence. The multi-session loop:

1. Get ready work: `issuegraph ready --mol <molecule-id>`
2. Claim it: `issuegraph update <id> --claim`
3. Do the work
4. Close it: `issuegraph close <id>`
5. Repeat until the molecule is done

### Dependency Types

Only some dependency types block execution:

| Type | Semantics | Use case |
|------|-----------|----------|
| `blocks` | B can't start until A closes | Sequencing work |
| `parent-child` | If the parent is blocked, children are blocked | Hierarchy (children parallel by default) |
| `conditional-blocks` | B runs only if A fails | Error-handling paths |
| `waits-for` | B waits for all of A's dynamic children | Fan-in gates — see [Gates](/workflows/gates) |

Non-blocking types (`related`, `discovered-from`, `replies-to`) link issues
without affecting execution.

### Step Dependencies

In a formula, steps declare `needs`:

```toml
[[steps]]
id = "implement"
title = "Implement feature"
needs = ["design"]  # Must complete design first
```

On live issues, add the edge directly — the dependent comes first:

```bash
issuegraph dep add <B-id> <A-id>   # B depends on A (B needs A)
```

The `issuegraph ready` command respects these:

```bash
issuegraph ready --mol <molecule-id>  # Only shows steps with completed dependencies
```

### Progressing Through Steps

```bash
# Start a step
issuegraph update bd-xyz.1 --claim

# Complete a step
issuegraph close bd-xyz.1 --reason "Done"

# Check what's ready next
issuegraph ready --mol bd-xyz
```

### Viewing Progress

```bash
# See blocked steps
issuegraph blocked

# Step-by-step status: [done] / [current] / [ready] / [blocked] / [pending]
issuegraph mol current <molecule-id>

# Progress summary: completed/total, rate, ETA
issuegraph mol progress <molecule-id>
```

## Molecule Lifecycle

```
Formula (template source)
    ↓ issuegraph cook
Proto (template epic)
    ↓ issuegraph mol pour
Molecule (instance)
    ↓ work steps
Completed Molecule
    ↓ optional cleanup
Closed / Squashed / Burned
```

Closing the last child does not close the molecule root — epics stay open as
close-eligible work until closed explicitly (`issuegraph epic close-eligible` sweeps
them). For cleanup of the beads themselves:

- `issuegraph mol squash <id>` condenses a molecule's ephemeral children into a
  permanent digest issue.
- `issuegraph mol burn <id>` deletes a molecule outright, no digest — for abandoned
  or test runs.

See [Wisps](/workflows/wisps) for the ephemeral lifecycle these commands
usually serve.

## Bonding: Connecting Work Graphs

**Bond** means creating a dependency between two work graphs. When molecule A
blocks molecule B, completing A unblocks B and an agent can continue from A
into B — one compound workflow that can span days.

```bash
issuegraph mol bond A B                    # B depends on A (sequential by default)
issuegraph mol bond A B --type parallel    # B runs alongside A
issuegraph mol bond A B --type conditional # B runs only if A fails
```

The command is polymorphic over its operands:

| Operands | What happens |
|----------|--------------|
| proto + proto | Compound proto (reusable template) |
| proto + molecule | Spawns the proto as new issues, attached to the molecule |
| molecule + molecule | Joins them into a compound molecule |
| formula + anything | The formula is cooked inline first |

Spawned issues follow the target's phase (persistent or ephemeral) by
default. Override with `--pour` (force persistent) or `--ephemeral` (force
ephemeral) — see [Wisps](/workflows/wisps).

### Dynamic Bonding

When the number of children isn't known until runtime, bond in a loop with
`--ref` to get readable child IDs instead of random hashes:

```bash
# One arm per discovered worker
issuegraph mol bond mol-worker-arm bd-patrol --ref arm-{{name}} --var name=ace
# Creates: bd-patrol.arm-ace (and children like bd-patrol.arm-ace.capture)
```

## Advanced Features

### Bond Points

Formulas can define bond points — named attachment sites for composition.
Each names a step to attach `before_step` or `after_step` (with optional
`parallel = true`):

```toml
[[compose.bond_points]]
id = "entry"
description = "Attach setup work here"
before_step = "design"
```

### Hooks

Step-completion hooks are not currently exposed as runnable formula
actions. The historical `on_complete.run` example was invalid: `run` is
not a formula field, and `on_complete` runtime expansion is tracked
separately until it is wired end to end.

### Assigning Molecules

Assign the molecule root to an agent at pour time, then track where each
agent is:

```bash
issuegraph mol pour mol-feature --assignee <agent>   # Assign on creation
issuegraph mol current --for <agent>                 # Where that agent is
```

## Agent Pitfalls

1. **Temporal language inverts dependencies.** "Phase 1 comes before Phase 2"
   tempts `issuegraph dep add phase1 phase2` — backwards. Use requirement language:
   "Phase 2 needs Phase 1" is `issuegraph dep add phase2 phase1`. Verify with
   `issuegraph blocked`.
2. **Numbered steps don't create sequence.** Steps named "Step 1/2/3" still
   run in parallel until you add dependencies between them.
3. **Forgetting to close work.** Blocked issues stay blocked forever if their
   blockers aren't closed: `issuegraph close <id> --reason "Done"`.

## Example Workflow

```bash
# 1. Create molecule from formula
issuegraph cook feature-workflow.formula.toml
issuegraph mol pour feature-workflow --var name="dark-mode"

# 2. View structure
issuegraph dep tree bd-xyz

# 3. Start first step
issuegraph update bd-xyz.1 --claim

# 4. Complete and progress
issuegraph close bd-xyz.1
issuegraph ready --mol bd-xyz  # Shows next steps

# 5. Continue until complete
```

## See Also

- [Formulas](/workflows/formulas) - Creating templates
- [Gates](/workflows/gates) - Async coordination
- [Wisps](/workflows/wisps) - Ephemeral workflows
