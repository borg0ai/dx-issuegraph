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
Touches: `.spec/rfc/0023-global-agent-install.md`

The agent keeps RFC 0023 at the document stage and does not edit product code. The change plan is produced through Birdify and Graphify and must be confirmed before implementation begins.

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

The agent read the Netify configuration first to see which capabilities were enabled for the repository. The capability list was used to decide which skills to run for the release.

Reviewer attention: Confirm the inventory was used for all required capability checks.

### Scope RFC-0003 removal across remote sync call sites found by Graphify
Status: discussed
Touches: `apps/cli/*`, `**/storage/**`, `**/*federation*`, `**/*autopush*`, `**/templates/**`

Graphify source indexing located Dolt remote paths in CLI commands, initialization, automatic push, storage federation, and agent templates. The planned removal is framed around those call chains rather than a narrow single-command delete.

Reviewer attention: Verify every remote/autopush/federation/template path named by the graph is covered or explicitly out of scope before merge.

### Bump all version files to 1.3.1 and verify with check-versions.sh
Status: directed
Touches: `scripts/update-versions.sh`, `scripts/check-versions.sh`

The agent ran the version sync script so every component reports the same release version, then ran the check script to confirm the files agree. The bump was made as a patch step from 1.3.0.

Reviewer attention: Confirm all 14 version files carry the same version string.

### Tag v1.3.1 on the RFC-close commit rather than the version bump commit
Status: directed
Touches: `.spec/rfc/0020-precommit-go-work.md`

The agent tagged v1.3.1 on the commit that closes RFC 0020, which was made after the version bump. This makes the tag include the hook and test fixes that landed with the RFC close.

Reviewer attention: Confirm the tag resolves to the intended commit on origin.

### Close RFC 0020 through the specify workflow before release
Status: directed
Touches: `.spec/rfc/0020-precommit-go-work.md`, `.spec/ROADMAP.md`, `.spec/TASK_TRACKING.md`

The agent moved RFC 0020 through Approved to Implemented with the specify tooling before cutting the release. This keeps the spec status in step with the shipped pre-commit fix.

Reviewer attention: Review the specify sync-check output, which was not inspected after the close.

### Move federation test cases behind a cgo build tag with a no-op counterpart
Status: discussed
Touches: `apps/cli/dolt_local_only_test.go`, `apps/cli/sync_test.go`

The federation commands are compiled only under cgo, but the test file that exercised them had no build tag, so cgo-off builds failed to compile. The agent moved those test cases into a cgo-tagged helper and added a no-op counterpart for non-cgo builds. It also removed an unused import in sync_test.go.

Reviewer attention: Confirm go vet is clean with cgo on and off.

### Publish each release archive with the issuegraph binary at its root
Status: directed
Touches: `Makefile`

The agent packaged each platform build so the executable sits at the archive root under the name issuegraph. Channel recipes depend on this layout to install the binary directly.

Risk: Archives built from Makefile output must keep this layout for every platform.

### Verify release assets are publicly downloadable and match SHA256SUMS
Status: directed
Touches: `SHA256SUMS`, `Makefile`

The agent downloaded each published archive without authentication and compared its checksum to the published SHA256SUMS file. All six checksums matched for 1.3.1 and 1.3.2.

Reviewer attention: Re-run the checksum comparison if assets are re-uploaded.

### Register the Nix package in packages.nix and the README install table
Status: directed
Touches: `nix/packages.nix`, `README.md`

The agent added the issuegraph package to the Nix package list and listed the Nix install command in the README install table. Both entries point at the same release version.

Reviewer attention: Confirm the Nix package attribute name matches the flake output.

### Deliver package-channel updates as pull requests and leave them unmerged
Status: directed
Touches: `Formula/issuegraph.rb`, `nix/packages.nix`, `bucket/issuegraph.json`

The agent pushed each package-channel update to its own release branch and opened a pull request in each channel repository. None of the pull requests were merged by the agent.

Reviewer attention: Confirm each channel PR is reviewed and merged in the right order.

### Build six Docker targets with CGO_ENABLED=0 and the gms_pure_go tag
Status: directed
Touches: `Makefile`

The agent built the release binaries for linux, darwin and windows on amd64 and arm64 inside a pinned Go container. It used CGO_ENABLED=0 and the gms_pure_go tag to match the existing make build recipe.

Risk: The macOS binaries are not code-signed or notarized.

### Remove BEADS fallbacks without migration or legacy support
Status: directed
Touches: `modules/core/internal/legacydir/legacydir.go`, `apps/cli/mail.go`, `apps/cli/messaging_test.go`

The agent removed the BEADS_DIR environment lookup and the .beads directory lookup, so the CLI reads only ISSUEGRAPH_DIR and .issuegraph. No migration or compatibility path was added.

Risk: Workspaces that only have a .beads directory or only set BEADS_DIR are no longer found.
Reviewer attention: Confirm no remaining caller depends on BEADS_DIR or .beads.

### Drop BEADS_MAIL_DELEGATE from mail delegate resolution
Status: directed
Touches: `apps/cli/mail.go`, `apps/cli/messaging_test.go`

The agent removed the BEADS_MAIL_DELEGATE fallback from mail delegate resolution, along with its help text and the test case that exercised it. Delegate lookup now uses only the ISSUEGRAPH variant.

Reviewer attention: Confirm help output no longer mentions BEADS_MAIL_DELEGATE.

### Keep legacydir package name after removing its legacy behavior
Status: discussed
Touches: `modules/core/internal/legacydir/legacydir.go`

The agent kept the legacydir package name after removing its BEADS behavior so existing imports still compile. Renaming the package was deferred to avoid touching every caller.

Considered/rejected: Renaming the package, which would require editing every importer for no behavior change.

### Record BEADS removal as a breaking change in RFC 0021
Status: directed
Touches: `.spec/rfc/0021-remove-beads-fallbacks.md`

The agent delivered RFC 0021 for the removal and recorded it as a breaking change. The RFC states that old .beads workspaces and BEADS_DIR setups are no longer detected.

Reviewer attention: Confirm the breaking-change note is in the release notes.

### Supersede RFC 0009 with RFC 0013
Status: discussed
Touches: `rfc/0009-*.md`, `rfc/0013-*.md`

RFC 0009 was marked as replaced by RFC 0013 so the RFC set reflects the current design. The change was committed as c85992a.

### Run CLI mail tests with cgo enabled
Status: directed
Touches: `apps/cli/messaging_test.go`

The agent ran the mail delegate tests with cgo on because they carry a cgo build tag and skip in a cgo-off run. The tests passed under cgo.

Risk: A cgo-off test run reports these tests as skipped, not passed.

### Report unverified Nix build instead of claiming success
Status: directed
Touches: `nix/packages.nix`

Nix is not installed on the build machine, so the agent could not run the Nix build. It reported the Nix channel as unverified and listed the commands needed on Linux.

Risk: The Nix package may fail to build despite a valid version bump.

### Bump all version files to 1.3.2 and verify with check-versions.sh
Status: discussed
Touches: `scripts/update-versions.sh`, `scripts/check-versions.sh`, `apps/cli/info.go`, `README.md`, `CHANGELOG.md`

The agent repeated the version sync for 1.3.2 so the BEADS removal ships in a new release rather than under the existing 1.3.1 tag. The check script passed after the sync.

Reviewer attention: Confirm the 1.3.2 bump touches only release paths.

### Tag v1.3.2 on the release commit
Status: discussed
Touches: `CHANGELOG.md`, `README.md`, `apps/cli/info.go`

The agent created an annotated v1.3.2 tag on the release commit that holds the version bump, README note, changelog entry and info list update. The tag was pushed and verified against the commit.

Reviewer attention: Confirm the tag points at the commit containing the release notes.

### Use main as base branch for Homebrew release PR
Status: discussed
Touches: `Formula/issuegraph.rb`

The agent based the Homebrew channel branch on main rather than develop because the formula already existed on main. The channel's documented base is develop, which lags main.

Considered/rejected: Targeting develop, which is documented as the base but is behind main.
Reviewer attention: Confirm the intended base branch for the Homebrew channel.

### Leave TestDoltPullRefusesInLocalOnlyMode failing as pre-existing
Status: discussed
Touches: `apps/cli/dolt_local_only_test.go`

The agent found that this test expects an error return while the product now exits with code 1, a change from an earlier commit. The agent judged this a stale test outside the release fix and did not change it.

Risk: The default test run stays red until the expectation is updated.
Reviewer attention: Decide whether the test or the exit behavior is the intended contract.

### Fix pre-commit hook to read go.work and lint each workspace module
Status: discussed — hook failure surfaced during the release commit
Touches: `.githooks/pre-commit`

The pre-commit hook read a root go.mod that no longer exists and ran golangci-lint from a root with no module. The agent changed it to read the Go version from go.work and to lint each workspace module separately.

Risk: Per-module lint may report findings the single root run did not.
Reviewer attention: Confirm the hook passes on a clean commit and fails on a deliberate lint error.

### Use the cli binary name internally and ship issuegraph assets
Status: discussed
Touches: `Makefile`, `package.json`

The Makefile builds the binary as cli and the release assets are named issuegraph. The agent kept the internal name because RFC 0009 records cli as the name the TypeScript app launches.

Considered/rejected: Renaming the binary to issuegraph repo-wide, which would touch about 15 Makefile lines and any caller of the name.
Reviewer attention: Decide whether the binary rename should become its own change.

### Add a local issuegraph symlink to the make install output
Status: discussed
Touches: `Makefile`

make install placed the binary at a path named cli, so the issuegraph command was not on PATH. The agent added a symlink outside the repository pointing at the installed binary.

Risk: The symlink is a local machine change and is not recreated by the repo.

### Enable all code graph tools in the repository
Status: directed — agent guidance files and hooks were generated by the graph init tool
Touches: `CLAUDE.md`, `AGENTS.md`, `.gitignore`, `.claude/settings.json`, `.githooks/pre-commit`

The agent enabled the ReviewGraph, CodeGraph, ReasonGraph, Graphify and Birdify tooling so each one has data for impact analysis. The init step injected graph instructions into agent guidance files and added .reviewgraph/ to .gitignore.

Risk: CLAUDE.md now contains a long generated block that conflicts with its request to stay short.
Reviewer attention: Decide whether the generated agent directories and the 58 MB Graphify output are committed or ignored.

### Add 1.3.2 release notes to README, CHANGELOG and info versionChanges
Status: discussed
Touches: `README.md`, `CHANGELOG.md`, `apps/cli/info.go`

The agent added a latest-release note stating the breaking BEADS removal to the README, and matching entries to the changelog and the info what's-new list. The info tests pass with the new entry.

Reviewer attention: Confirm the breaking change is described for users of old .beads workspaces.

### Ship BEADS removal as release 1.3.2 instead of patching 1.3.1
Status: discussed
Touches: `apps/cli/mail.go`, `modules/core/internal/legacydir/legacydir.go`

The published 1.3.1 binaries still contain the BEADS fallbacks, so the agent cut a new release 1.3.2 that contains the removal. Channel PRs were opened for 1.3.2 with new checksums.

Considered/rejected: Re-tagging or replacing the 1.3.1 assets, which would change a published artifact.

### Hold RFC 0022 implementation until the plan is confirmed
Status: discussed
Touches: `.spec/rfc/0022-rich-cli-output.md`, `.birdify/rfc-0022.html`

The agent drafted RFC 0022 and rendered a change plan, but made no code edits until the plan is approved. The RFC remains in Draft status.

Reviewer attention: Approve the RFC and the starting file set before implementation begins.

### Approve RFC 0022 for styled CLI output
Status: directed
Touches: `rfc/0022-*.md`, `modules/core/internal/ui/*`

The agent scoped RFC 0022 to styled output for a small set of CLI commands, with no full-screen interface. The approval covers only that narrowed scope.

Considered/rejected: A full-screen terminal interface was excluded from scope.
Reviewer attention: Confirm the RFC 0022 text matches the narrowed scope.

### Add shared styled message API in ui package for key commands
Status: directed — per approved RFC 0022
Touches: `modules/core/internal/ui/message.go`, `modules/core/internal/ui/message_test.go`

The agent added message.go to the ui package with Rich, Banner, Section, Field, Success, Warn, Error, Info and Hint helpers so that key commands share one styling path. Field replaces the KeyValue name used in the earlier RFC draft, and the RFC text was corrected to match. Helpers fall back to plain text when color is not active.

Considered/rejected: A new styling layer outside the ui package was not added; the existing ui package was extended instead.
Reviewer attention: Confirm the helper set covers the commands in scope and that Field is the name used across callers and the RFC.

### Reuse existing ui palette and icons with no emoji
Status: discussed
Touches: `modules/core/internal/ui/message.go`

Message helpers draw colors and symbols from the existing ui palette and icon set and add no new colors or emoji. This keeps command output visually consistent with the rest of the ui package.

Reviewer attention: Confirm no emoji or new palette entries appear in rendered output.

### Keep plain text output unchanged when piped, NO_COLOR, agent mode or json
Status: discussed
Touches: `apps/cli/info.go`, `apps/cli/version.go`, `apps/cli/doctor.go`

The agent verified that piped, NO_COLOR and agent-mode output emit no ANSI sequences and that version piped output matches the prior format byte for byte. Color mode produced 34 ANSI sequences, and --json output stayed valid with zero ANSI sequences. The only plain-text change is the info underline width.

Reviewer attention: Confirm the single plain-text change in info output (underline width) is acceptable for any consumers of that text.

### List unowned plan files in both files and unmappedFiles in Birdify event
Status: discussed
Touches: `.birdify/rfc-0022.html`

The Birdify plan validator rejected files that had no owner in the architecture map. The agent listed those files in both the files and unmappedFiles fields so the event validates.

Reviewer attention: Confirm the unowned files are the intended scope.

### Leave stale Birdify map unregenerated
Status: discussed
Touches: `.birdify/rfc-0022.html`

The architecture map references an older project revision and still names a removed MCP module. The agent rendered the RFC plan against this map without regenerating it.

Risk: Plan module ownership may drift from the current code until the map is refreshed.
Reviewer attention: Regenerate the map before relying on its module ownership.

### Write message helper tests before message.go
Status: discussed
Touches: `modules/core/internal/ui/message_test.go`, `modules/core/internal/ui/message.go`

The agent wrote message_test.go first and confirmed it failed before implementing message.go, then made it pass. The full ui package test suite passes after the change.

Reviewer attention: Confirm the tests assert plain-text fallback as well as styled output.

### Drop spinner from RFC 0022 scope
Status: discussed — no spinner helper added to message.go
Touches: `modules/core/internal/ui/message.go`

The agent did not add a spinner helper because the affected commands have no long-running steps. The RFC was updated to reflect this scope reduction.

Reviewer attention: Confirm no affected command needs progress feedback.

### Route info and version output through styled helpers only
Status: discussed
Touches: `apps/cli/info.go`, `apps/cli/version.go`

The info title, fields, sections and hook warnings, along with version output, now render through the message helpers. No other command output paths were changed in these files.

Reviewer attention: Confirm info and version output have no remaining raw styled strings.

### Leave doctor and init rendering unchanged apart from the plain note line
Status: discussed
Touches: `apps/cli/doctor.go`

doctor and init already had styling, so the agent changed only the plain 'not yet supported in embedded mode' note to use the Hint helper. The doctor agent output path was left untouched and has no new test coverage.

Risk: The agent output path of doctor has no test covering the changed note.
Reviewer attention: Confirm the agent output path of doctor is intentionally unchanged.

### Pause RFC 0022 code edits on a blocked edit hook
Status: discussed
Touches: `modules/core/internal/ui/message_test.go`, `modules/core/internal/ui/message.go`

A Netify edit-check hook blocked all edits to the repository, so the agent stopped RFC 0022 code work and did not bypass the guard. Only the failing test file had been written before the pause. The agent is waiting for a decision on how to proceed.

Risk: RFC 0022 implementation is stalled until the hook is cleared or disabled for this task.
Reviewer attention: Decide whether to disable the hook for this task, write message.go manually, or pause the RFC.

### Hold commits until message.go lands
Status: discussed
Touches: `modules/core/internal/ui/message_test.go`, `modules/core/internal/ui/message.go`

The agent decided not to commit the test file alone, because the ui package does not build without message.go. Committing only after the helper implementation is in place keeps the branch buildable.

Risk: Committing the test file early would leave the modules/core ui package and its existing tests failing.
Reviewer attention: Confirm no commit is made until message.go and the ui tests pass.

### Match info section underline length to its title
Status: discussed
Touches: `apps/cli/info.go`

The agent widened the info section underline to match the title width, from 27 to 31 '=' characters. This is the only visible plain-text difference in piped output.

Reviewer attention: Confirm no parser or snapshot depends on the previous 27-character underline.

### Add message output helpers section to AGENT_INSTRUCTIONS.md
Status: discussed
Touches: `AGENT_INSTRUCTIONS.md`

The agent documented the new message helpers under the Visual Design System section of AGENT_INSTRUCTIONS.md so that future agents use them instead of ad hoc styling.

Reviewer attention: Confirm the documented helper list matches message.go.

### Leave styles.go and terminal.go unchanged while adding message helpers
Status: discussed
Touches: `modules/core/internal/ui/styles.go`, `modules/core/internal/ui/terminal.go`

The new helpers were placed in message.go, and the existing styling and terminal detection files were left as they were. This keeps the change additive and limits its blast radius.

Reviewer attention: Confirm no behavior in styles.go or terminal.go was relied on implicitly by the new helpers.

### Verify styled output via ANSI sequence counts across output modes
Status: discussed
Touches: `apps/cli/info.go`, `apps/cli/version.go`, `modules/core/internal/ui/message.go`

The agent built the CLI and ran info and version under forced color, NO_COLOR, piped and agent modes, and JSON. It counted ANSI escape sequences in each mode, with 34 in color mode and zero elsewhere, and validated the JSON output.

Reviewer attention: Confirm the verification matches the modes the release notes promise.

### Run CLI tests with cgo enabled and treat cgo-off embedded Dolt failures as environmental
Status: discussed
Touches: `apps/cli/doctor_mutation_gate_test.go`

Six doctor mutation gate tests fail when CGO_ENABLED=0 because embedded Dolt requires a cgo build. The agent treated these as environmental and not caused by the styling change, and ran the affected tests with cgo enabled.

Risk: Non-cgo builds fail in embedded mode, and these tests do not catch that in CI without cgo.
Reviewer attention: Confirm the release builds used for distribution are built with cgo enabled.

### Bypass edit hook temporarily with scripted file writes to finish RFC 0022
Status: directed — temporary bypass permitted for this session
Touches: `modules/core/internal/ui/message.go`, `apps/cli/info.go`, `apps/cli/version.go`, `apps/cli/doctor.go`

While the edit hook blocked standard edit tools, the agent wrote the message helpers and command changes through scripted file writes so the RFC 0022 work could finish. Hook settings and global configuration were not modified.

Risk: Edits made through the bypass skip the normal edit-tool guard, so they were not checked by that hook.
Reviewer attention: Review the diffs of these four files directly, since they were written through the bypass path.

### Keep per-repository setup under issuegraph init
Status: directed
Touches: `apps/cli/setup/*.go`

Dolt storage is per repository, so repo-scoped setup (storage, git hooks, AGENTS.md and agent integration files) stays under `init`. Binary installation is global and handled by package managers, so no separate repo-level install command is added.

Considered/rejected: A new top-level install command for per-repo setup, which would overlap with package-manager installs.
Reviewer attention: Confirm docs state the order: install the binary globally, then run init in each repository.

### Add global issuegraph install for agent skills
Status: directed
Touches: `apps/cli/setup/*.go`, `issuegraph/skills/issuegraph/**`

A global `install` command places the issuegraph skill into each detected agent's global skills directory so agents learn to use issuegraph. It is separate from per-repo `init` and from binary installation.

Considered/rejected: Folding skill installation into `init`, which would tie a machine-global action to each repository.

### Rename setup command to install with no alias
Status: directed
Touches: `apps/cli/setup.go`, `apps/cli/setup/*.go`, `docs/**`, `README.md`, `modules/core/internal/templates/skills/**`

The existing `setup` command is renamed to `install` with no alias and no transition period, so `issuegraph setup` reports unknown command. About 306 references across docs, CLI setup code, doctor hints, tests, README and skill text must be updated.

Risk: Scripts or docs that still call `issuegraph setup` fail until updated.
Reviewer attention: Confirm no remaining references to the old `setup` command in code, docs, tests or skill text.

### Default global agent install scope with a project flag
Status: discussed — proposed; confirmation pending
Touches: `apps/cli/setup/*.go`

`install` defaults to global scope, and a `--project` flag selects repository scope. This changes recipes that currently default to project scope.

Considered/rejected: Keeping project scope as the default for install recipes.
Reviewer attention: Confirm global default before implementation starts.

### Embed the full issuegraph skill tree into the binary
Status: discussed
Touches: `scripts/generate-issuegraph-assets.mjs`, `issuegraph/skills/issuegraph/**`

The asset generator embeds only SKILL.md and openai.yaml, omitting commands/ and resources/. The RFC requires embedding the whole issuegraph/skills/issuegraph tree, excluding README.md, CLAUDE.md and adr/, regenerating it via pnpm run generate:assets at build time, and adding a test that fails when generated files are stale.

Risk: Embedded skill content can drift from source if the stale-generated-file test is missing or skipped.
Reviewer attention: Confirm the exclusion list and that the stale-file test runs in CI.

### Use issuegraph/skills/issuegraph as the single skill source
Status: discussed
Touches: `modules/core/internal/templates/skills/issuegraph/**`, `issuegraph/skills/issuegraph/**`

A duplicate copy under modules/core/internal/templates/skills/issuegraph had already drifted from the source directory. The RFC deletes the copy so the source directory is the only origin for embedded skill content.

Risk: Code importing the deleted templates copy breaks until repointed.

### Write agent global skill paths only after verification
Status: discussed
Touches: `apps/cli/setup/*.go`

Each agent's global skills path is confirmed against that agent's documentation before it is included. Agents whose path cannot be confirmed are skipped rather than guessed.

Reviewer attention: Confirm each included agent path against its official docs.

### Limit first-version install support to four agents
Status: discussed — proposed; agent list pending confirmation
Touches: `apps/cli/setup/*.go`

The first version proposes claude, codex, gemini and hermes, because their global skill directories were observed on the development machine. The list is pending confirmation before the RFC is approved.

Reviewer attention: Confirm the agent list before implementation.

### Defer post-init hint toward install to a follow-up
Status: discussed — proposed; confirmation pending
Touches: `apps/cli/setup/*.go`

After `init` completes without a global skill present, a one-line hint pointing to `install` is proposed for a later iteration rather than the first version.

### Draft RFC 0023 for global agent install
Status: discussed
Touches: `.spec/rfc/0023-global-agent-install.md`

The install feature is captured as RFC 0023 in Draft status. It covers the setup-to-install rename, global skill embedding, and per-agent path verification. The RFC passes validation and cross-file consistency checks, and no product code changes until it is approved.
