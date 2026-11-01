# Why: main

<!-- reasongraph:v1 generated 2026-11-01 — review before sharing; edit freely, edits are preserved -->

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

### Bump all version files to 1.3.1 and verify with check-versions.sh
Status: directed
Touches: `check-versions.sh`

The agent bumped all 14 version files to 1.3.1 in a single commit (ac80a2b) so every package reports the same version. It ran check-versions.sh to confirm the files agree before tagging.

Reviewer attention: Confirm the 14 version files listed in the bump commit match the release version.

### Tag v1.3.1 on the RFC-close commit rather than the version bump commit
Status: directed — tag placement was a deliberate choice in the session
Touches: `rfc/0020-*.md`

The annotated tag v1.3.1 points at 8bac9e2, the commit that closes RFC 0020 after the version bump commit ac80a2b. The agent treated this placement as intentional, not as an error. No explicit rationale for choosing the later commit was discussed.

Considered/rejected: Tagging the version bump commit ac80a2b directly.
Risk: Anyone referencing the tag gets the RFC-close state, which includes changes beyond the version bump.
Reviewer attention: Confirm the tag should point at 8bac9e2 and that both commits are in the release history.

### Close RFC 0020 through the specify workflow before release
Status: discussed
Touches: `rfc/0020-*.md`, `ROADMAP.md`, `TASK_TRACKING.md`

The agent advanced RFC 0020 to closed using the specify workflow and committed it in 8bac9e2 before tagging. The RFC lifecycle record is kept consistent with the release.

Risk: ROADMAP.md and TASK_TRACKING.md sync was not confirmed in the session.
Reviewer attention: Confirm ROADMAP.md and TASK_TRACKING.md agree with the closed RFC status.

### Move federation test cases behind a cgo build tag with a no-op counterpart
Status: discussed
Touches: `federation.go`, `**/*federation*_test.go`

go vet failed on HEAD with undefined federationSyncCmd because commit 521a32ed8 removed federation commands from non-test code while tests still referenced them. The agent moved the three federation test cases into a //go:build cgo helper file and added a no-op counterpart for non-cgo builds, leaving the test body unchanged. go vet passes in both cgo modes.

Considered/rejected: Deleting the three federation cases from the test file. That edit was reverted because federation.go defines the commands behind a build tag, so the cases still apply under cgo.
Risk: Under CGO_ENABLED=0 the no-op variant does not exercise federation behavior.
Reviewer attention: Confirm the no-op counterpart is intended. TestDoltPullRefusesInLocalOnlyMode still fails in one mode and its cause was not confirmed.

### Publish each release archive with the issuegraph binary at its root
Status: discussed
Touches: `packages.nix`, `README.md`

Each of the six release archives places the bare issuegraph binary at the archive root. Channel install recipes build their download URLs against this layout.

Risk: Channel URLs break if the archive layout changes in a later release.
Reviewer attention: Confirm the archive layout matches what each channel recipe expects.

### Verify release assets are publicly downloadable and match SHA256SUMS
Status: discussed
Touches: `SHA256SUMS`

The agent downloaded all six release assets and compared their checksums against SHA256SUMS. It also confirmed the URLs need no authentication, since package channels cannot install from private assets. All six checksums matched.

Reviewer attention: Re-check asset reachability after any later re-upload.

### Register the Nix package in packages.nix and the README install table
Status: discussed
Touches: `packages.nix`, `README.md`

The agent added the issuegraph Nix package entry to packages.nix and listed it in the README install table on the channel base branch. The Nix build was not verified locally because Nix is not installed on the machine.

Risk: The Nix package has not been built or tested.
Reviewer attention: Run the Nix build before merging the Nix channel PR.

### Deliver package-channel updates as pull requests and leave them unmerged
Status: discussed
Touches: `packages.nix`, `README.md`

The agent opened one pull request per package channel (Homebrew, Nix, and a third channel) instead of pushing directly. None of the three PRs was merged or verified on its own platform in the session.

Risk: Channel install paths stay unverified until each PR is tested on its platform.
Reviewer attention: Confirm each channel PR installs and runs issuegraph on its target platform before merging.

### Build six Docker targets with CGO_ENABLED=0 and the gms_pure_go tag
Status: discussed
Touches: `Makefile`

The agent planned the six-target Docker builds to mirror make build: pure Go, CGO disabled, and the gms_pure_go build tag. The build was interrupted in the session, so its results are unconfirmed.

Risk: Release binaries for the six targets have not been confirmed to build from the Docker path.
Reviewer attention: Re-run the six-target build and confirm it matches make build output.
