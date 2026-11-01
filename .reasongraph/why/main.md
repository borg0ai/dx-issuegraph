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
Status: discussed
Touches: `go.mod`, `check-versions.sh`

The agent set all 14 version files to 1.3.1 in commit ac80a2b and confirmed that check-versions.sh passes. The Go module version is treated as the source of truth.

Reviewer attention: Confirm the version number is correct given the breaking BEADS removal.

### Tag v1.3.1 on the RFC-close commit rather than the version bump commit
Status: discussed
Touches: `rfc/0020-*.md`, `go.mod`

The annotated v1.3.1 tag points at 8bac9e2, the RFC 0020 close commit that follows the version bump. This means the tag includes the RFC closure.

Reviewer attention: The tag does not include the later BEADS removal in 8425f8f.

### Close RFC 0020 through the specify workflow before release
Status: discussed
Touches: `rfc/0020-*.md`

RFC 0020 was closed through the specify workflow in commit 8bac9e2 before the release was cut.

### Move federation test cases behind a cgo build tag with a no-op counterpart
Status: discussed
Touches: `federation.go`, `**/*federation*_test.go`

go vet failed on HEAD with undefined federationSyncCmd because commit 521a32ed8 removed federation commands from non-test code while tests still referenced them. The agent moved the three federation test cases into a //go:build cgo helper file and added a no-op counterpart for non-cgo builds, leaving the test body unchanged. go vet passes in both cgo modes.

Considered/rejected: Deleting the three federation cases from the test file. That edit was reverted because federation.go defines the commands behind a build tag, so the cases still apply under cgo.
Risk: Under CGO_ENABLED=0 the no-op variant does not exercise federation behavior.
Reviewer attention: Confirm the no-op counterpart is intended. TestDoltPullRefusesInLocalOnlyMode still fails in one mode and its cause was not confirmed.

### Publish each release archive with the issuegraph binary at its root
Status: discussed
Touches: `Makefile`

Each of the six release archives contains a bare issuegraph binary at its root. Package channels install from this layout, so the layout is part of the release contract.

### Verify release assets are publicly downloadable and match SHA256SUMS
Status: discussed
Touches: `SHA256SUMS`

The agent downloaded the published assets without authentication and compared each checksum against SHA256SUMS. All six assets are public and every checksum matches.

### Register the Nix package in packages.nix and the README install table
Status: discussed
Touches: `packages.nix`, `README.md`

The Nix package was added to packages.nix and listed in the README install table, following the existing entries of the base branch.

### Deliver package-channel updates as pull requests and leave them unmerged
Status: discussed — PRs were later merged outside this session
Touches: `packages.nix`, `README.md`

The agent delivered the Homebrew, Nix, and third-channel updates as pull requests. It did not merge them itself.

### Build six Docker targets with CGO_ENABLED=0 and the gms_pure_go tag
Status: discussed
Touches: `Dockerfile*`, `Makefile`

The Docker builds use pure Go with CGO_ENABLED=0 and the gms_pure_go tag, matching make build. The final build status for all six targets was not recorded.

### Remove BEADS fallbacks without migration or legacy support
Status: directed
Touches: `**/legacydir/*`, `apps/cli/mail.go`

The agent deleted the BEADS fallbacks in full with no migration path and no legacy support kept. Directory resolution now reads only ISSUEGRAPH_DIR and the root/.issuegraph workspace, and the BEADS_DIR and .beads lookups are gone.

Risk: Workspaces that only contain .beads or set BEADS_DIR no longer resolve and must be renamed by hand.
Reviewer attention: Confirm the breaking change is intended. Commit 8425f8f touched 14 files, more than the intended scope, including changes from another session. v1.3.1 was tagged before this removal, so the shipped binary still has the fallbacks. A feat! change normally implies 2.0.0, which is still unresolved.

### Drop BEADS_MAIL_DELEGATE from mail delegate resolution
Status: directed
Touches: `apps/cli/mail.go`, `apps/cli/mail*_test.go`

The agent removed the BEADS_MAIL_DELEGATE environment fallback, its help text, and the test case that exercised it. Delegate resolution no longer consults that variable.

Reviewer attention: Confirm no scripts still export BEADS_MAIL_DELEGATE.

### Keep legacydir package name after removing its legacy behavior
Status: discussed — name is now inaccurate; rename left open
Touches: `**/legacydir/*`

The package keeps its legacydir name after the legacy behavior was removed. Renaming was not done in the same change, so the diff stays limited to behavior.

Considered/rejected: Renaming the package in the same commit was not done.
Risk: The name misleads readers about what the package does now.
Reviewer attention: Decide whether to rename the package in a follow-up.

### Record BEADS removal as a breaking change in RFC 0021
Status: discussed
Touches: `rfc/0021-*.md`

RFC 0021 was marked delivered, approved, and closed through the specify workflow. The RFC notes that the removal breaks workspaces that only have .beads or set a BEADS directory variable.

Reviewer attention: Check the RFC wording against the final behavior.

### Supersede RFC 0009 with RFC 0013
Status: discussed
Touches: `rfc/0009-*.md`, `rfc/0013-*.md`

RFC 0009 was marked as replaced by RFC 0013 so the RFC set reflects the current design. The change was committed as c85992a.

### Run CLI mail tests with cgo enabled
Status: discussed — cgo-off run reported no tests
Touches: `apps/cli/mail*_test.go`

With cgo disabled the CLI mail tests reported no tests to run, so the agent reran them with cgo on. Both mail tests pass, and TestMailDelegateFromConfig skips as it did before the change.

Reviewer attention: Confirm CI runs these tests with cgo enabled, since the default run silently skips them.

### Report unverified Nix build instead of claiming success
Status: discussed
Touches: `packages.nix`, `README.md`

Nix is not installed locally, so the Nix build could not be verified. The agent reported that gap and attempted brew audit for the Homebrew recipe instead.

Reviewer attention: Run the Nix build on a machine with Nix before merging the packages.nix change. The brew audit result was not recorded.
