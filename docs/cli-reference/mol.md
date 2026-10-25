---
title: "issuegraph mol"
description: "Molecule commands (work templates)"
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc mol`.

Manage molecules - work templates for agent workflows.

Protos are template epics with the "template" label. They define a DAG of work
that can be spawned to create real issues (molecules).

The molecule metaphor:
  - A proto is an uninstantiated template (reusable work pattern)
  - Spawning creates a molecule (real issues) from the proto
  - Variables (&#123;&#123;key&#125;&#125;) are substituted during spawning
  - Bonding combines protos or molecules into compounds
  - Distilling extracts a proto from an ad-hoc epic

Commands:
  show       Show proto/molecule structure and variables
  pour       Instantiate proto as persistent mol (liquid phase)
  wisp       Instantiate proto as ephemeral wisp (vapor phase)
  bond       Polymorphic combine: proto+proto, proto+mol, mol+mol
  squash     Condense molecule to digest
  burn       Discard wisp
  distill    Extract proto from ad-hoc epic

Use "issuegraph formula list" to list available formulas.

```
issuegraph mol [command]
```

**Aliases:** protomolecule

## issuegraph mol bond

Bond two protos or molecules to create a compound.

The bond command is polymorphic - it handles different operand types:

  formula + formula → cook both, compound proto
  formula + proto   → cook formula, compound proto
  formula + mol     → cook formula, spawn and attach
  proto + proto     → compound proto (reusable template)
  proto + mol       → spawn proto, attach to molecule
  mol + proto       → spawn proto, attach to molecule
  mol + mol         → join into compound molecule

Formula names (e.g., mol-polecat-arm) are cooked inline as ephemeral protos.
This avoids needing pre-cooked proto issuegraph in the database.

Bond types:
  sequential (default) - B runs after A completes
  parallel            - B runs alongside A
  conditional         - B runs only if A fails

Phase control:
  By default, spawned protos follow the target's phase:
  - Attaching to mol (Ephemeral=false) → spawns as persistent (Ephemeral=false)
  - Attaching to ephemeral issue (Ephemeral=true) → spawns as ephemeral (Ephemeral=true)

  Override with:
  --pour  Force spawn as liquid (persistent, Ephemeral=false)
  --ephemeral  Force spawn as vapor (ephemeral, Ephemeral=true, excluded from Dolt sync via dolt_ignore)

Dynamic bonding (Christmas Ornament pattern):
  Use --ref to specify a custom child reference with variable substitution.
  This creates IDs like "parent.child-ref" instead of random hashes.

  Example:
    issuegraph mol bond mol-worker-arm bd-patrol --ref arm-&#123;&#123;worker_name&#125;&#125; --var worker_name=ace
    # Creates: bd-patrol.arm-ace (and children like bd-patrol.arm-ace.capture)

Use cases:
  - Found important bug during patrol? Use --pour to persist it
  - Need ephemeral diagnostic on persistent feature? Use --ephemeral
  - Spawning per-worker arms on a patrol? Use --ref for readable IDs

Examples:
  issuegraph mol bond mol-feature mol-deploy                    # Compound proto
  issuegraph mol bond mol-feature mol-deploy --type parallel    # Run in parallel
  issuegraph mol bond mol-feature bd-abc123                     # Attach proto to molecule
  issuegraph mol bond bd-abc123 bd-def456                       # Join two molecules
  issuegraph mol bond mol-critical-bug wisp-patrol --pour       # Persist found bug
  issuegraph mol bond mol-temp-check bd-feature --ephemeral          # Ephemeral diagnostic
  issuegraph mol bond mol-arm bd-patrol --ref arm-&#123;&#123;name&#125;&#125; --var name=ace  # Dynamic child ID

```
issuegraph mol bond <A> <B> [flags]
```

**Aliases:** fart

**Flags:**

```
      --as string         Custom title for compound proto (proto+proto only)
      --dry-run           Preview what would be created
      --ephemeral         Force spawn as vapor (ephemeral, Ephemeral=true)
      --pour              Force spawn as liquid (persistent, Ephemeral=false)
      --ref string        Custom child reference with {{var}} substitution (e.g., arm-{{polecat_name}})
      --type string       Bond type: sequential, parallel, or conditional (default "sequential")
      --var stringArray   Variable substitution for spawned protos (key=value)
```

## issuegraph mol burn

Burn a molecule, deleting it without creating a digest.

Unlike squash (which creates a permanent digest before deletion), burn
completely removes the molecule with no trace. Use this for:
  - Abandoned patrol cycles
  - Crashed or failed workflows
  - Test/debug molecules you don't want to preserve

The burn operation differs based on molecule phase:
  - Wisp (ephemeral): Direct delete
  - Mol (persistent): Cascade delete (syncs to remotes)

CAUTION: This is a destructive operation. The molecule's data will be
permanently lost. If you want to preserve a summary, use 'issuegraph mol squash'.

Example:
  issuegraph mol burn bd-abc123              # Delete molecule with no trace
  issuegraph mol burn bd-abc123 --dry-run    # Preview what would be deleted
  issuegraph mol burn bd-abc123 --force      # Skip confirmation
  issuegraph mol burn bd-a1 bd-b2 bd-c3      # Batch delete multiple wisps

```
issuegraph mol burn <molecule-id> [molecule-id...] [flags]
```

**Flags:**

```
      --dry-run   Preview what would be deleted
      --force     Skip confirmation prompt
```

## issuegraph mol current

Show where you are in a molecule workflow.

If molecule-id is given, show status for that molecule.
If not given, infer from in_progress issues assigned to current agent.

The output shows all steps with status indicators:
  [done]     - Step is complete (closed)
  [current]  - Step is in_progress (you are here)
  [ready]    - Step is ready to start (unblocked)
  [blocked]  - Step is blocked by dependencies
  [pending]  - Step is waiting

For large molecules (&gt;100 steps), a summary is shown instead.
Use --limit or --range to view specific steps:
  issuegraph mol current &lt;id&gt; --limit 50       # Show first 50 steps
  issuegraph mol current &lt;id&gt; --range 100-150  # Show steps 100-150

```
issuegraph mol current [molecule-id] [flags]
```

**Flags:**

```
      --for string     Show molecules for a specific agent/assignee
      --limit int      Maximum number of steps to display (0 = auto, use 'all' threshold)
      --range string   Display specific step range (e.g., '1-50', '100-150')
```

## issuegraph mol distill

Distill a molecule by extracting a reusable formula from an existing epic.

This is the reverse of pour: instead of formula → molecule, it's molecule → formula.

The distill command:
  1. Loads the existing epic and all its children
  2. Converts the structure to a .formula.json file
  3. Replaces concrete values with &#123;&#123;variable&#125;&#125; placeholders (via --var flags)

Use cases:
  - Team develops good workflow organically, wants to reuse it
  - Capture tribal knowledge as executable templates
  - Create starting point for similar future work

Variable syntax (both work - we detect which side is the concrete value):
  --var branch=feature-auth    Spawn-style: variable=value (recommended)
  --var feature-auth=branch    Substitution-style: value=variable

Output locations (first writable wins):
  1. &lt;resolved-issuegraph-dir&gt;/formulas/ (project-level, default)
  2. &lt;checkout-root&gt;/.issuegraph/formulas/ (repo-local formulas)
  3. ~/.issuegraph/formulas/     (user-level, if project not writable)

Examples:
  issuegraph mol distill bd-o5xe my-workflow
  issuegraph mol distill bd-abc release-workflow --var feature_name=auth-refactor

```
issuegraph mol distill <epic-id> [formula-name] [flags]
```

**Flags:**

```
      --dry-run           Preview what would be created
      --output string     Output directory for formula file
      --var stringArray   Replace value with {{variable}} placeholder (variable=value)
```

## issuegraph mol last-activity

Show the most recent activity timestamp for a molecule.

Returns the timestamp of the most recent change to any step in the molecule,
making it easy to detect stale or stuck molecules.

Activity sources:
  step_closed      - A step was closed
  step_updated     - A step was updated (claimed, edited, etc.)
  molecule_updated - The molecule root itself was updated

Examples:
  issuegraph mol last-activity hq-wisp-0laki
  issuegraph mol last-activity hq-wisp-0laki --json

```
issuegraph mol last-activity <molecule-id> [flags]
```

## issuegraph mol pour

Pour a proto into a persistent mol - like pouring molten metal into a mold.

This is the chemistry-inspired command for creating PERSISTENT work from templates.
The resulting mol is stored as persistent issuegraph in the issue database and
syncs like any other issue (issuegraph dolt push / pull).

Phase transition: Proto (solid) -&gt; pour -&gt; Mol (liquid)

WHEN TO USE POUR vs WISP:
  pour (liquid): Persistent work that needs audit trail
    - Feature implementations spanning multiple sessions
    - Work you may need to reference later
    - Anything worth preserving in git history

  wisp (vapor): Ephemeral work that auto-cleans up
    - Release workflows (one-time execution)
    - Operational loops and recurring cycles
    - Health checks and diagnostics
    - Any operational workflow without audit value

TIP: Formulas can specify phase:"vapor" to recommend wisp usage.
     If you pour a vapor-phase formula, you'll get a warning.

Examples:
  issuegraph mol pour mol-feature --var name=auth    # Persistent feature work
  issuegraph mol pour mol-review --var pr=123        # Persistent code review

```
issuegraph mol pour <proto-id> [flags]
```

**Flags:**

```
      --assignee string      Assign the root issue to this agent/user
      --attach strings       Proto to attach after spawning (repeatable)
      --attach-type string   Bond type for attachments: sequential, parallel, or conditional (default "sequential")
      --dry-run              Preview what would be created
      --var stringArray      Variable substitution (key=value)
```

## issuegraph mol progress

Show efficient progress summary for a molecule.

This command uses indexed queries to count progress without loading all steps,
making it suitable for very large molecules (millions of steps).

If no molecule-id is given, shows progress for any molecule you're working on.

Output includes:
  - Progress: completed / total (percentage)
  - Current step: the in-progress step (if any)
  - Rate: steps/hour based on closure times
  - ETA: estimated time to completion

Example:
  issuegraph mol progress bd-hanoi-xyz

```
issuegraph mol progress [molecule-id] [flags]
```

## issuegraph mol ready

Find molecules where a gate has closed and the workflow is ready to resume.

This command discovers molecules waiting at a gate step where:
1. The molecule has a gate issue that blocks a step
2. The gate issue is now closed (condition satisfied)
3. The blocked step is now ready to proceed
4. No agent currently has this molecule hooked

This enables discovery-based resume without explicit waiter tracking.
The patrol system uses this to find and dispatch gate-ready molecules.

Examples:
  issuegraph mol ready --gated           # Find all gate-ready molecules
  issuegraph mol ready --gated --json    # JSON output for automation

```
issuegraph mol ready --gated [flags]
```

**Flags:**

```
      --gated   Find molecules ready for gate-resume dispatch (always on for this subcommand)
```

## issuegraph mol seed

Verify that a formula is accessible and can be cooked.

The seed command checks formula search paths to ensure a formula exists
and can be loaded. This is useful for verifying system health before
attempting to spawn work from a formula.

Formula search paths (checked in order):
  1. &lt;resolved-issuegraph-dir&gt;/formulas/ (active project)
  2. &lt;checkout-root&gt;/.issuegraph/formulas/ (repo-local formulas)
  3. ~/.issuegraph/formulas/ (user level)
  4. $GT_ROOT/.issuegraph/formulas/ (shared workspace root, if GT_ROOT set)

Examples:
  issuegraph mol seed mol-feature                 # Verify specific formula
  issuegraph mol seed mol-review --var name=test  # Verify with variable substitution

```
issuegraph mol seed <formula-name> [flags]
```

**Flags:**

```
      --var stringArray   Variable substitution for condition filtering (key=value)
```

## issuegraph mol show

Show molecule structure and details.

The --parallel flag highlights parallelizable steps:
  - Steps with no blocking dependencies can run in parallel
  - Shows which steps are ready to start now
  - Identifies parallel groups (steps that can run concurrently)

Example:
  issuegraph mol show bd-patrol --parallel

```
issuegraph mol show <molecule-id> [flags]
```

**Flags:**

```
  -p, --parallel   Show parallel step analysis
```

## issuegraph mol squash

Squash a molecule's ephemeral children into a single digest issue.

This command collects all ephemeral child issues of a molecule (Ephemeral=true),
generates a summary digest, and promotes the wisps to persistent by
clearing their Wisp flag (or optionally deletes them).

The squash operation:
  1. Loads the molecule and all its children
  2. Filters to only wisps (ephemeral issues with Ephemeral=true)
  3. Generates a digest (summary of work done)
  4. Creates a permanent digest issue (Ephemeral=false)
  5. Clears Wisp flag on children (promotes to persistent)
     OR keeps them with --keep-children (default: delete)

AGENT INTEGRATION:
Use --summary to provide an AI-generated summary. This keeps issuegraph as a pure
tool - the calling agent (orchestrator worker, Claude Code, etc.) is responsible
for generating intelligent summaries. Without --summary, a basic concatenation
of child issue content is used.

This is part of the wisp workflow: spawn creates wisps,
execution happens, squash compresses the trace into an outcome (digest).

Example:
  issuegraph mol squash bd-abc123                    # Squash and promote children
  issuegraph mol squash bd-abc123 --dry-run          # Preview what would be squashed
  issuegraph mol squash bd-abc123 --keep-children    # Keep wisps after digest
  issuegraph mol squash bd-abc123 --summary "Agent-generated summary of work done"

```
issuegraph mol squash <molecule-id> [flags]
```

**Flags:**

```
      --dry-run          Preview what would be squashed
      --keep-children    Don't delete ephemeral children after squash
      --summary string   Agent-provided summary (bypasses auto-generation)
```

## issuegraph mol stale

Detect molecules (epics with children) that are complete but still open.

A molecule is considered stale if:
  1. All children are closed (Completed == Total)
  2. Root issue is still open
  3. Not assigned to anyone (optional, use --unassigned)
  4. Is blocking other work (optional, use --blocking)

By default, shows all complete-but-unclosed molecules.

Examples:
  issuegraph mol stale              # List all stale molecules
  issuegraph mol stale --json       # Machine-readable output
  issuegraph mol stale --blocking   # Only show those blocking other work
  issuegraph mol stale --unassigned # Only show unassigned molecules
  issuegraph mol stale --all        # Include molecules with 0 children

```
issuegraph mol stale [flags]
```

**Flags:**

```
      --all          Include molecules with 0 children
      --blocking     Only show molecules blocking other work
      --unassigned   Only show unassigned molecules
```

## issuegraph mol wisp

Create or manage wisps - EPHEMERAL molecules for operational workflows.

When called with a proto-id argument, creates a wisp from that proto.
When called with a subcommand (list, gc), manages existing wisps.

Wisps are issues with Ephemeral=true in the main database. They're stored
locally but NOT synced via git.

WHEN TO USE WISP vs POUR:
  wisp (vapor): Ephemeral work that auto-cleans up
    - Release workflows (one-time execution)
    - Operational loops and recurring cycles
    - Health checks and diagnostics
    - Any operational workflow without audit value

  pour (liquid): Persistent work that needs audit trail
    - Feature implementations spanning multiple sessions
    - Work you may need to reference later
    - Anything worth preserving in git history

TIP: Formulas can specify phase:"vapor" to recommend wisp usage.
     If you use pour on a vapor-phase formula, you'll get a warning.

The wisp lifecycle:
  1. Create: issuegraph mol wisp &lt;proto&gt; or issuegraph create --ephemeral
  2. Execute: Normal issuegraph operations work on wisp issues
  3. Squash: issuegraph mol squash &lt;id&gt; (clears Ephemeral flag, promotes to persistent)
  4. Or burn: issuegraph mol burn &lt;id&gt; (deletes without creating digest)

Examples:
  issuegraph mol wisp mol-my-workflow                  # Ephemeral operational cycle
  issuegraph mol wisp list                             # List all wisps
  issuegraph mol wisp gc                               # Garbage collect old wisps

Subcommands:
  list  List all wisps in current context
  gc    Garbage collect orphaned wisps

```
issuegraph mol wisp [proto-id] [flags]
issuegraph mol wisp [command]
```

**Flags:**

```
      --dry-run           Preview what would be created
      --root-only         Create only the root issue (no child step issues)
      --var stringArray   Variable substitution (key=value)
```

### issuegraph mol wisp create

Create a wisp from a proto - sublimation from solid to vapor.

This is the chemistry-inspired command for creating ephemeral work from templates.
The resulting wisp is stored in the main database with Ephemeral=true and NOT synced via git.

Phase transition: Proto (solid) -&gt; Wisp (vapor)

Use wisp for:
  - Operational loops and recurring cycles
  - Health checks and monitoring
  - One-shot orchestration runs
  - Routine operations with no audit value

The wisp will:
  - Be stored in main database with Ephemeral=true flag
  - NOT be synced via git
  - Either evaporate (burn) or condense to digest (squash)

Examples:
  issuegraph mol wisp create mol-patrol                    # Ephemeral patrol cycle
  issuegraph mol wisp create mol-health-check              # One-time health check
  issuegraph mol wisp create mol-diagnostics --var target=db  # Diagnostic run

```
issuegraph mol wisp create <proto-id> [flags]
```

**Flags:**

```
      --dry-run           Preview what would be created
      --root-only         Create only the root issue (no child step issues)
      --var stringArray   Variable substitution (key=value)
```

### issuegraph mol wisp gc

Garbage collect old or abandoned wisps from the database.

A wisp is considered abandoned if:
  - It hasn't been updated in --age duration and is not closed
  - AND it is not live work: blocked steps (waiting on a dependency), pinned
    issuegraph, and any step whose status category is wip (in_progress, blocked,
    hooked) or frozen (deferred, pinned) are never reclaimed by age, no matter
    how long they have been waiting (GH#4394). Custom statuses count by their
    configured category, so only plain open (active) and closed (done) steps
    are age-reclaimable. If the blocked set or the custom-status list cannot be
    read, the GC aborts rather than risk reclaiming live steps.

Abandoned wisps are deleted without creating a digest. Use 'issuegraph mol squash'
if you want to preserve a summary before garbage collection.

Use --closed to purge ALL closed wisps (regardless of age). This is the
fastest way to reclaim space from accumulated wisp bloat. Safe by default:
requires --force to actually delete.

Note: This uses time-based cleanup, appropriate for ephemeral wisps.
For graph-pressure staleness detection (blocking other work), see 'issuegraph mol stale'.

Examples:
  issuegraph mol wisp gc                                    # Clean abandoned wisps (default: 1h threshold)
  issuegraph mol wisp gc --dry-run                          # Preview what would be cleaned
  issuegraph mol wisp gc --age 24h                          # Custom age threshold
  issuegraph mol wisp gc --all                              # Also clean closed wisps older than threshold
  issuegraph mol wisp gc --closed                           # Preview closed wisp deletion
  issuegraph mol wisp gc --closed --force                   # Delete all closed wisps
  issuegraph mol wisp gc --closed --dry-run                 # Explicit dry-run (same as no --force)
  issuegraph mol wisp gc --exclude-type agent,rig           # Protect agent and rig wisps from GC
  issuegraph mol wisp gc --closed --force --exclude-type mol # Delete closed wisps except mol type

```
issuegraph mol wisp gc [flags]
```

**Flags:**

```
      --age string             Age threshold for abandoned wisp detection (default "1h")
      --all                    Also clean closed wisps older than threshold
      --closed                 Delete all closed wisps (ignores --age threshold)
      --dry-run                Preview what would be cleaned
      --exclude-type strings   Exclude wisps of these types from GC (comma-separated, e.g., agent,rig)
  -f, --force                  Actually delete (default: preview only)
```

### issuegraph mol wisp list

List all wisps (ephemeral molecules) in the current context.

Wisps are issues with Ephemeral=true in the main database. They are stored
locally but not synced via git.

The list shows:
  - ID: Issue ID of the wisp
  - Title: Wisp title
  - Status: Current status (open, in_progress, closed)
  - Started: When the wisp was created
  - Updated: Last modification time

Old wisp detection:
  - Old wisps haven't been updated in 24+ hours
  - Use 'issuegraph mol wisp gc' to clean up old/abandoned wisps

Examples:
  issuegraph mol wisp list              # List all wisps
  issuegraph mol wisp list --json       # JSON output for programmatic use
  issuegraph mol wisp list --all        # Include closed wisps

```
issuegraph mol wisp list [flags]
```

**Flags:**

```
      --all           Include closed wisps
      --type string   Filter by issue type (e.g., agent, task, patrol)
```
