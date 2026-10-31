# Why: main

<!-- reasongraph:v1 generated 2026-10-31 — review before sharing; edit freely, edits are preserved -->

## Intent
Advance the next approved RFC by removing Dolt remote issue synchronization from the CLI, storage, and agent-facing surfaces while leaving Pending RFCs unimplemented.

## Decisions

### Treat RFC-0003 as the next implementable approved RFC
Status: directed
Touches: `.birdify/rfc-0003.html`, `apps/cli/*`

RFC-0014 validated successfully but remains Pending, so it cannot be implemented yet. The next approved executable item is RFC-0003, which removes Dolt remote issue sync. Existing remote, autopush, and federation call sites mean the RFC is still incomplete.

Considered/rejected: Implementing Pending RFC-0014 immediately was rejected because Pending status blocks implementation despite a passing validate/sync-check.
Reviewer attention: Confirm RFC-0003 is still Approved and is the intended next landing target versus any newer approved RFC.

### Keep product code unchanged until the Birdify plan is confirmed
Status: discussed
Touches: `.birdify/rfc-0003.html`

Birdify and Netify gates require an explicit plan confirmation before multi-file implementation. The agent rendered and browser-checked the RFC-0003 change plan, listed verification commands without running them, and left product code untouched pending approval.

Risk: A long hold after planning can drift from a moving codebase before implementation starts.
Reviewer attention: Confirm the rendered plan at .birdify/rfc-0003.html matches the intended removal surface before any code edits.

### Correct Birdify module mapping for RFC-0003 before planning
Status: discussed — mapping revised to rev 3 during planning
Touches: `.birdify/*`, `.birdify/rfc-0003.html`

Graphify tracing showed remote sync still exposed through CLI, initialization, autopush, storage federation, and agent templates. The prior Birdify map still referenced outdated bd naming and stale path coverage, so the mapping was revised to rev 3 and re-rendered to avoid an incorrect implementation plan.

Considered/rejected: Reusing the stale Birdify architecture map without correction was rejected because it would mis-scope the change plan.

### Initialize ReasonGraph in the repository
Status: discussed
Touches: `.reasongraph/`, `.gitignore`, `AGENTS.md`

ReasonGraph was required for Netify context recovery and decision capture. The agent initialized it successfully, writing .reasongraph/ and updating .gitignore and AGENTS.md.

Risk: AGENTS.md edits can collide with intentional local divergence markers already present in the repo.

### Preserve existing local .issuegraph workspace changes
Status: agent-initiated — not requested in plan or prompts
Touches: `.issuegraph/*`

The checkout already contained local .issuegraph/ modifications. The agent chose to leave those changes intact while investigating RFC state and Netify capability readiness.

Risk: Unrelated local issue snapshot edits may remain mixed with later RFC work unless separated before commit.

### Stay on the current checkout without creating a branch or worktree
Status: discussed
Touches: `.birdify/rfc-0003.html`

netify-worktree-pr ask-first forbids creating a branch or worktree without explicit approval. Planning and capability seeding therefore continue in the existing checkout.

Considered/rejected: Creating a feature worktree or branch for RFC-0003 was deferred until explicit approval.

### Record default Netify capability inventory when .netify.json is missing
Status: discussed
Touches: `.netify.json`

The workspace had no .netify.json. Under Netify requirements the agent recorded the default capability set rather than inventing a reduced subset, while marking unavailable tools with evidence instead of skipping them.

Risk: A freshly written capability manifest may disagree with what is actually installed on the machine until installs catch up.

### Scope RFC-0003 removal across remote sync call sites found by Graphify
Status: discussed
Touches: `apps/cli/*`, `**/storage/**`, `**/*federation*`, `**/*autopush*`, `**/templates/**`

Graphify source indexing located Dolt remote paths in CLI commands, initialization, automatic push, storage federation, and agent templates. The planned removal is framed around those call chains rather than a narrow single-command delete.

Reviewer attention: Verify every remote/autopush/federation/template path named by the graph is covered or explicitly out of scope before merge.
