# Multiple Personas Workflow Example

This example demonstrates how to use issuegraph when different roles work on the same project (architect, implementer, reviewer, etc.).

## Problem

Complex projects involve different personas with different concerns:
- **Architect:** System design, technical decisions, high-level planning
- **Implementer:** Write code, fix bugs, implement features
- **Reviewer:** Code review, quality gates, testing
- **Product:** Requirements, priorities, user stories

Each persona needs:
- Different views of the same work
- Clear handoffs between roles
- Track discovered work in context

## Solution

Use issuegraph labels, priorities, and dependencies to organize work by persona, with clear ownership and handoffs.

## Setup

```bash
# Initialize issuegraph
cd my-project
issuegraph init

# Start Dolt server for auto-sync (optional for teams)
issuegraph dolt start
```

## Persona: Architect

The architect creates high-level design and makes technical decisions.

### Create Architecture Epic

```bash
# Main epic
issuegraph create "Design new caching layer" -t epic -p 1
# Returns: bd-a1b2c3

# Add architecture label
issuegraph label add bd-a1b2c3 architecture

# Architecture tasks
issuegraph create "Research caching strategies (Redis vs Memcached)" -p 1 \
  --deps discovered-from:bd-a1b2c3
issuegraph label add bd-xyz architecture

issuegraph create "Write ADR: Caching layer design" -p 1 \
  --deps discovered-from:bd-a1b2c3
issuegraph label add bd-abc architecture

issuegraph create "Design cache invalidation strategy" -p 1 \
  --deps discovered-from:bd-a1b2c3
issuegraph label add bd-def architecture
```

### View Architect Work

```bash
# See only architecture issues
issuegraph list --label architecture

# See architecture issues that are ready
issuegraph list --label architecture --status open | grep -v blocked

# High-priority architecture decisions
issuegraph list --label architecture --priority 0
issuegraph list --label architecture --priority 1
```

### Handoff to Implementer

When design is complete, create implementation tasks:

```bash
# Close architecture tasks
issuegraph close bd-xyz --reason "Decided on Redis with write-through"
issuegraph close bd-abc --reason "ADR-007 published"

# Create implementation tasks with labels
issuegraph create "Implement Redis connection pool" -p 1 \
  --deps discovered-from:bd-a1b2c3
issuegraph label add bd-impl1 implementation

issuegraph create "Add cache middleware to API routes" -p 1 \
  --deps discovered-from:bd-a1b2c3
issuegraph label add bd-impl2 implementation

# Link implementation to architecture
issuegraph dep add bd-impl1 bd-abc --type related  # Based on ADR
issuegraph dep add bd-impl2 bd-abc --type related
```

## Persona: Implementer

The implementer writes code based on architecture decisions.

### View Implementation Work

```bash
# See only implementation tasks
issuegraph list --label implementation --status open

# See what's ready to implement
issuegraph ready | grep implementation

# High-priority bugs to fix
issuegraph list --label implementation --type bug --priority 0
issuegraph list --label implementation --type bug --priority 1
```

### Claim and Implement

```bash
# Claim a task
issuegraph update bd-impl1 --claim

# During implementation, discover issues
issuegraph create "Need connection retry logic" -t bug -p 1 \
  --deps discovered-from:bd-impl1
issuegraph label add bd-bug1 implementation bug

issuegraph create "Add metrics for cache hit rate" -p 2 \
  --deps discovered-from:bd-impl1
issuegraph label add bd-metric1 implementation observability

# Complete implementation
issuegraph close bd-impl1 --reason "Redis pool working, tested locally"
```

### Handoff to Reviewer

```bash
# Mark ready for review
issuegraph create "Code review: Redis caching layer" -p 1
issuegraph label add bd-review1 review

# Link to implementation
issuegraph dep add bd-review1 bd-impl1 --type related
issuegraph dep add bd-review1 bd-impl2 --type related
```

## Persona: Reviewer

The reviewer checks code quality, tests, and approvals.

### View Review Work

```bash
# See all review tasks
issuegraph list --label review --status open

# See what's ready for review
issuegraph ready | grep review

# High-priority reviews
issuegraph list --label review --priority 0
issuegraph list --label review --priority 1
```

### Perform Review

```bash
# Claim review
issuegraph update bd-review1 --claim

# Found issues during review
issuegraph create "Add unit tests for retry logic" -t task -p 1 \
  --deps discovered-from:bd-review1
issuegraph label add bd-test1 implementation testing

issuegraph create "Fix: connection leak on timeout" -t bug -p 0 \
  --deps discovered-from:bd-review1
issuegraph label add bd-bug2 implementation bug critical

issuegraph create "Document Redis config options" -p 2 \
  --deps discovered-from:bd-review1
issuegraph label add bd-doc1 documentation

# Block review until issues fixed
issuegraph dep add bd-review1 bd-test1 --type blocks
issuegraph dep add bd-review1 bd-bug2 --type blocks
```

### Approve or Request Changes

```bash
# After fixes, approve
issuegraph close bd-review1 --reason "LGTM, all tests pass"

# Or request changes
issuegraph update bd-review1 --status blocked
# (blockers will show up in dependency tree)
```

## Persona: Product Owner

The product owner manages priorities and requirements.

### View Product Work

```bash
# See all features
issuegraph list --type feature

# See high-priority work
issuegraph list --priority 0
issuegraph list --priority 1

# See what's in progress
issuegraph list --status in_progress

# See what's blocked
issuegraph list --status blocked
```

### Prioritize Work

```bash
# Bump priority based on customer feedback
issuegraph update bd-impl2 --priority 0

# Lower priority for nice-to-haves
issuegraph update bd-metric1 --priority 3

# Add product label to track customer-facing work
issuegraph label add bd-impl2 customer-facing
```

### Create User Stories

```bash
# User story
issuegraph create "As a user, I want faster page loads" -t feature -p 1
issuegraph label add bd-story1 user-story customer-facing

# Link technical work to user story
issuegraph dep add bd-impl1 bd-story1 --type related
issuegraph dep add bd-impl2 bd-story1 --type related
```

## Multi-Persona Workflow Example

### Week 1: Architecture Phase

**Architect:**

```bash
# Create epic
issuegraph create "Implement rate limiting" -t epic -p 1  # bd-epic1
issuegraph label add bd-epic1 architecture

# Research
issuegraph create "Research rate limiting algorithms" -p 1 \
  --deps discovered-from:bd-epic1
issuegraph label add bd-research1 architecture research

issuegraph update bd-research1 --claim
# ... research done ...
issuegraph close bd-research1 --reason "Chose token bucket algorithm"

# Design
issuegraph create "Write ADR: Rate limiting design" -p 1 \
  --deps discovered-from:bd-epic1
issuegraph label add bd-adr1 architecture documentation

issuegraph close bd-adr1 --reason "ADR-012 approved"
```

### Week 2: Implementation Phase

**Implementer:**

```bash
# See what's ready to implement
issuegraph ready | grep implementation

# Create implementation tasks based on architecture
issuegraph create "Implement token bucket algorithm" -p 1 \
  --deps discovered-from:bd-epic1
issuegraph label add bd-impl1 implementation
issuegraph dep add bd-impl1 bd-adr1 --type related

issuegraph create "Add rate limit middleware" -p 1 \
  --deps discovered-from:bd-epic1
issuegraph label add bd-impl2 implementation

# Claim and start
issuegraph update bd-impl1 --claim

# Discover issues
issuegraph create "Need distributed rate limiting (Redis)" -t bug -p 1 \
  --deps discovered-from:bd-impl1
issuegraph label add bd-bug1 implementation bug
```

**Architect (consulted):**

```bash
# Architect reviews discovered issue
issuegraph show bd-bug1
issuegraph update bd-bug1 --priority 0  # Escalate to critical
issuegraph label add bd-bug1 architecture  # Architect will handle

# Make decision
issuegraph create "Design: Distributed rate limiting" -p 0 \
  --deps discovered-from:bd-bug1
issuegraph label add bd-design1 architecture

issuegraph close bd-design1 --reason "Use Redis with sliding window"
```

**Implementer (continues):**

```bash
# Implement based on architecture decision
issuegraph create "Add Redis sliding window for rate limits" -p 0 \
  --deps discovered-from:bd-design1
issuegraph label add bd-impl3 implementation

issuegraph close bd-impl1 --reason "Token bucket working"
issuegraph close bd-impl3 --reason "Redis rate limiting working"
```

### Week 3: Review Phase

**Reviewer:**

```bash
# See what's ready for review
issuegraph list --label review

# Create review task
issuegraph create "Code review: Rate limiting" -p 1
issuegraph label add bd-review1 review
issuegraph dep add bd-review1 bd-impl1 --type related
issuegraph dep add bd-review1 bd-impl3 --type related

issuegraph update bd-review1 --claim

# Found issues
issuegraph create "Add integration tests for Redis" -t task -p 1 \
  --deps discovered-from:bd-review1
issuegraph label add bd-test1 testing implementation

issuegraph create "Missing error handling for Redis down" -t bug -p 0 \
  --deps discovered-from:bd-review1
issuegraph label add bd-bug2 implementation bug critical

# Block review
issuegraph dep add bd-review1 bd-test1 --type blocks
issuegraph dep add bd-review1 bd-bug2 --type blocks
```

**Implementer (fixes):**

```bash
# Fix review findings
issuegraph update bd-bug2 --claim
issuegraph close bd-bug2 --reason "Added circuit breaker for Redis"

issuegraph update bd-test1 --claim
issuegraph close bd-test1 --reason "Integration tests passing"
```

**Reviewer (approves):**

```bash
# Review unblocked
issuegraph close bd-review1 --reason "Approved, merging PR"
```

**Product Owner (closes epic):**

```bash
# Feature shipped!
issuegraph close bd-epic1 --reason "Rate limiting in production"
```

## Label Organization

### Recommended Labels

```bash
# Role labels
architecture, implementation, review, product

# Type labels
bug, feature, task, chore, documentation

# Status labels
critical, blocked, waiting-feedback, needs-design

# Domain labels
frontend, backend, infrastructure, database

# Quality labels
testing, security, performance, accessibility

# Customer labels
customer-facing, user-story, feedback
```

### View by Label Combination

```bash
# Critical bugs for implementers
issuegraph list --label implementation --label bug --label critical

# Architecture issues needing review
issuegraph list --label architecture --label review

# Customer-facing features
issuegraph list --label customer-facing --type feature

# Backend implementation work
issuegraph list --label backend --label implementation --status open
```

## Filtering by Persona

### Architect View

```bash
# My work
issuegraph list --label architecture --status open

# Design decisions to make
issuegraph list --label architecture --label needs-design

# High-priority architecture
issuegraph list --label architecture --priority 0
issuegraph list --label architecture --priority 1
```

### Implementer View

```bash
# My work
issuegraph list --label implementation --status open

# Ready to implement
issuegraph ready | grep implementation

# Bugs to fix
issuegraph list --label implementation --type bug --priority 0
issuegraph list --label implementation --type bug --priority 1

# Blocked work
issuegraph list --label implementation --status blocked
```

### Reviewer View

```bash
# Reviews waiting
issuegraph list --label review --status open

# Critical reviews
issuegraph list --label review --priority 0

# Blocked reviews
issuegraph list --label review --status blocked
```

### Product Owner View

```bash
# All customer-facing work
issuegraph list --label customer-facing

# Features in progress
issuegraph list --type feature --status in_progress

# Blocked work (needs attention)
issuegraph list --status blocked

# High-priority items across all personas
issuegraph list --priority 0
```

## Handoff Patterns

### Architecture → Implementation

```bash
# Architect creates spec
issuegraph create "Design: New payment API" -p 1
issuegraph label add bd-design1 architecture documentation

# When done, create implementation tasks
issuegraph create "Implement Stripe integration" -p 1
issuegraph label add bd-impl1 implementation
issuegraph dep add bd-impl1 bd-design1 --type related

issuegraph close bd-design1 --reason "Spec complete, ready for implementation"
```

### Implementation → Review

```bash
# Implementer finishes
issuegraph close bd-impl1 --reason "Stripe working, PR ready"

# Create review task
issuegraph create "Code review: Stripe integration" -p 1
issuegraph label add bd-review1 review
issuegraph dep add bd-review1 bd-impl1 --type related
```

### Review → Product

```bash
# Reviewer approves
issuegraph close bd-review1 --reason "Approved, deployed to staging"

# Product tests in staging
issuegraph create "UAT: Test Stripe in staging" -p 1
issuegraph label add bd-uat1 product testing
issuegraph dep add bd-uat1 bd-review1 --type related

# Product approves for production
issuegraph close bd-uat1 --reason "UAT passed, deploying to prod"
```

## Best Practices

### 1. Use Labels Consistently

```bash
# Good: Clear role separation
issuegraph label add bd-123 architecture
issuegraph label add bd-456 implementation
issuegraph label add bd-789 review

# Bad: Mixing concerns
# (same issue shouldn't be both architecture and implementation)
```

### 2. Link Related Work

```bash
# Always link implementation to architecture
issuegraph dep add bd-impl bd-arch --type related

# Link bugs to features
issuegraph dep add bd-bug bd-feature --type discovered-from
```

### 3. Clear Handoffs

```bash
# Document why closing
issuegraph close bd-arch --reason "Design complete, created bd-impl1 and bd-impl2 for implementation"

# Not: "done" (too vague)
```

### 4. Escalate When Needed

```bash
# Implementer discovers architectural issue
issuegraph create "Current design doesn't handle edge case X" -t bug -p 0
issuegraph label add bd-issue architecture  # Tag for architect
issuegraph label add bd-issue needs-design  # Flag as needing design
```

### 5. Regular Syncs

```bash
# Daily: Each persona checks their work
issuegraph list --label architecture --status open  # Architect
issuegraph list --label implementation --status open  # Implementer
issuegraph list --label review --status open  # Reviewer

# Weekly: Team reviews together
issuegraph stats  # Overall progress
issuegraph list --status blocked  # What's stuck?
issuegraph ready  # What's ready to work on?
```

## Common Patterns

### Spike Then Implement

```bash
# Architect creates research spike
issuegraph create "Spike: Evaluate GraphQL vs REST" -p 1
issuegraph label add bd-spike1 architecture research

issuegraph close bd-spike1 --reason "Chose GraphQL, created implementation tasks"

# Implementation follows
issuegraph create "Implement GraphQL API" -p 1
issuegraph label add bd-impl1 implementation
issuegraph dep add bd-impl1 bd-spike1 --type related
```

### Bug Triage

```bash
# Bug reported
issuegraph create "App crashes on large files" -t bug -p 1

# Implementer investigates
issuegraph update bd-bug1 --label implementation
issuegraph update bd-bug1 --claim

# Discovers architectural issue
issuegraph create "Need streaming uploads, not buffering" -t bug -p 0
issuegraph label add bd-arch1 architecture
issuegraph dep add bd-arch1 bd-bug1 --type discovered-from

# Architect designs solution
issuegraph update bd-arch1 --label architecture
issuegraph close bd-arch1 --reason "Designed streaming upload flow"

# Implementer fixes
issuegraph update bd-bug1 --claim
issuegraph close bd-bug1 --reason "Implemented streaming uploads"
```

### Feature Development

```bash
# Product creates user story
issuegraph create "Users want bulk import" -t feature -p 1
issuegraph label add bd-story1 user-story product

# Architect designs
issuegraph create "Design: Bulk import system" -p 1
issuegraph label add bd-design1 architecture
issuegraph dep add bd-design1 bd-story1 --type related

# Implementation tasks
issuegraph create "Implement CSV parser" -p 1
issuegraph label add bd-impl1 implementation
issuegraph dep add bd-impl1 bd-design1 --type related

issuegraph create "Implement batch processor" -p 1
issuegraph label add bd-impl2 implementation
issuegraph dep add bd-impl2 bd-design1 --type related

# Review
issuegraph create "Code review: Bulk import" -p 1
issuegraph label add bd-review1 review
issuegraph dep add bd-review1 bd-impl1 --type blocks
issuegraph dep add bd-review1 bd-impl2 --type blocks

# Product UAT
issuegraph create "UAT: Bulk import" -p 1
issuegraph label add bd-uat1 product testing
issuegraph dep add bd-uat1 bd-review1 --type blocks
```

## See Also

- [Multi-Phase Development](../multi-phase-development/) - Organize work by phase
- [Team Workflow](../team-workflow/) - Collaborate across personas
- [Contributor Workflow](../contributor-workflow/) - External contributions
- [Labels Documentation](../../docs/core-concepts/labels.md) - Label management guide
