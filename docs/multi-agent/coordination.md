---
title: Agent Coordination
description: Assign and claim issuegraph issues, hand off work, and serialize conflict-prone work with merge slots across multiple agents
---

Patterns for coordinating work between multiple AI agents.

## Work Assignment

### Assigning and Claiming Work

Assign work to a specific agent, or claim it atomically for yourself:

```bash
# Assign issue to an agent
issuegraph assign issuegraph-42 agent-1

# Atomically claim an issue (sets assignee to you, status to in_progress)
issuegraph update issuegraph-42 --claim

# Claim the first ready issue matching your filters
issuegraph ready --claim --json

# Release a claimed issue
issuegraph assign issuegraph-42 ""              # clear the assignee
issuegraph update issuegraph-42 --status open   # make it claimable again
```

### Checking Assigned Work

```bash
# What is agent-1 working on?
issuegraph list --assignee agent-1 --status in_progress

# What is ready for agent-1?
issuegraph ready --assignee agent-1

# JSON output
issuegraph list --assignee agent-1 --json
```

## Handoff Patterns

### Sequential Handoff

Agent A completes work, hands off to Agent B:

```bash
# Agent A
issuegraph comment issuegraph-42 "API complete, ready for review"
issuegraph assign issuegraph-42 agent-b

# Agent B picks up
issuegraph list --assignee agent-b  # Sees issuegraph-42
issuegraph update issuegraph-42 --claim
```

### Parallel Work

Multiple agents work on different issues:

```bash
# Coordinator
issuegraph assign issuegraph-42 agent-a
issuegraph assign issuegraph-43 agent-b
issuegraph assign issuegraph-44 agent-c

# Each agent claims its issue and works independently
issuegraph update issuegraph-42 --claim

# Coordinator monitors progress
issuegraph list --status in_progress --json
```

### Fan-Out / Fan-In

Split work, then merge:

```bash
# Fan-out
issuegraph create "Part A" --parent issuegraph-epic
issuegraph create "Part B" --parent issuegraph-epic
issuegraph create "Part C" --parent issuegraph-epic

issuegraph assign issuegraph-epic.1 agent-a
issuegraph assign issuegraph-epic.2 agent-b
issuegraph assign issuegraph-epic.3 agent-c

# Fan-in: wait for all parts (one dependency per call)
issuegraph dep add issuegraph-merge issuegraph-epic.1
issuegraph dep add issuegraph-merge issuegraph-epic.2
issuegraph dep add issuegraph-merge issuegraph-epic.3
```

If `issuegraph-epic` carries a size/effort label, see [Labels](/core-concepts/labels) for keeping it off the parts.

<Tip>
For structured epic fan-out, `issuegraph swarm` creates and tracks a swarm molecule
from an epic (`issuegraph swarm create`, `issuegraph swarm status`).
</Tip>

## Agent Discovery

IssueGraph has no agent registry — assignees are plain strings. To see which
agents are active, group in-progress work by assignee:

```bash
issuegraph list --status in_progress --json
```

## Conflict Prevention

### Atomic Claims

`--claim` is atomic: when multiple agents pull from the same ready queue,
the first claim wins, and repeating a claim you already hold is idempotent.
Prefer claiming over assigning when agents self-select work:

```bash
issuegraph ready --claim --json
```

### Merge Slots

Serialize conflict-prone work (such as merge-queue conflict resolution) with
a merge slot — an exclusive-access primitive only one agent can hold at a
time. Each project has one merge slot issue, named from the issue prefix
(e.g. `issuegraph-merge-slot`):

```bash
# Create the merge slot for this project
issuegraph merge-slot create

# Check availability
issuegraph merge-slot check

# Acquire before starting; release when done
issuegraph merge-slot acquire
issuegraph merge-slot release
```

## Communication Patterns

### Via Comments

```bash
# Agent A leaves note
issuegraph comment issuegraph-42 "Completed API, needs frontend integration"

# Agent B reads
issuegraph comments issuegraph-42
```

### Via Labels

```bash
# Mark for review
issuegraph update issuegraph-42 --add-label "needs-review"

# Agent B filters
issuegraph list --label-any needs-review
```

## Coordinating Across Repositories

Agents can coordinate work that spans repositories:

```bash
# Depend on a capability delivered by another project
issuegraph dep add issuegraph-42 external:backend:api-ready
```

Multi-repo routing, aggregated views, and contributor/team workflows are
covered in [Routing](/multi-agent/routing) and
[Multi-Repo Migration](/multi-agent/multi-repo-migration).

## Best Practices

1. **Clear ownership** - Assign or claim work so every issue has one owner
2. **Document handoffs** - Use comments to explain context
3. **Use labels for status** - `needs-review`, `blocked`, `ready`
4. **Avoid conflicts** - Claim atomically; use merge slots to serialize
   conflict-prone work
5. **Monitor progress** - Regular status checks
6. **Sync at session end** - Run `issuegraph dolt push` so other agents see your
   updates
