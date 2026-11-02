# Why: main

<!-- reasongraph:v1 generated 2026-11-02 — review before sharing; edit freely, edits are preserved -->

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

### Bump all version files to 1.3.3 and verify with check-versions.sh
Status: directed
Touches: `check-versions.sh`, `.githooks/pre-commit`

The agent synced every version file to 1.3.3 so the release commit matches the 1.3.2 release commit's file set of 17 files (16 release paths plus the 4 stamp lines). It ran check-versions.sh and the Info and Version tests, and both passed.

Reviewer attention: Confirm no version file was missed by the bump.

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
Status: discussed
Touches: `scripts/build-release.sh`

The agent matched the 1.3.2 release layout so each new archive holds a single issuegraph binary (issuegraph.exe on Windows) at its root and is covered by SHA256SUMS.

Reviewer attention: Confirm the archive listing matches the 1.3.2 layout.

### Verify release assets are publicly downloadable and match SHA256SUMS
Status: discussed
Touches: `Formula/issuegraph.rb`, `nix/issuegraph.nix`

The agent re-downloaded the 1.3.3 release archives from the public release and compared their SHA256 values with the published checksums. The channel manifests were written only after the hashes matched.

Reviewer attention: Confirm the per-platform hashes in each channel PR match the published SHA256SUMS.

### Register the Nix package in packages.nix and the README install table
Status: directed
Touches: `nix/packages.nix`, `README.md`

The agent added the issuegraph package to the Nix package list and listed the Nix install command in the README install table. Both entries point at the same release version.

Reviewer attention: Confirm the Nix package attribute name matches the flake output.

### Deliver package-channel updates as pull requests and leave them unmerged
Status: discussed
Touches: `Formula/issuegraph.rb`, `nix/issuegraph.nix`

The Homebrew and Nix channel updates were opened as pull requests and left unmerged for review. The Scoop update was handled the same way.

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
Status: discussed
Touches: `nix/issuegraph.nix`

The agent did not run nix build or nix flake check because no Linux machine was available. It reported the Nix change as untested in the channel PR summary rather than as verified.

Reviewer attention: Treat the Nix channel change as unverified until a Linux build passes.

### Bump all version files to 1.3.2 and verify with check-versions.sh
Status: discussed
Touches: `scripts/update-versions.sh`, `scripts/check-versions.sh`, `apps/cli/info.go`, `README.md`, `CHANGELOG.md`

The agent repeated the version sync for 1.3.2 so the BEADS removal ships in a new release rather than under the existing 1.3.1 tag. The check script passed after the sync.

Reviewer attention: Confirm the 1.3.2 bump touches only release paths.

### Tag v1.3.3 on the release commit
Status: directed
Touches: `.githooks/pre-commit`

The agent created an annotated tag v1.3.3 on the release commit (f372157) after confirming the tag did not already exist. The commit and tag were pushed, and the remote main and tag were checked to point at the same commit.

Reviewer attention: Confirm v1.3.3 points at f372157 on the remote.

### Use main as base branch for Homebrew release PR
Status: discussed — same base branch applied to the Nix and Scoop channel PRs
Touches: `Formula/*`, `nix/issuegraph.nix`

The agent opened the 1.3.3 channel update PRs against the main branch of each channel repository. Each channel repository was checked out and fast-forwarded to main before the update branch was created.

Considered/rejected: Targeting a develop branch for the Homebrew tap, which that repository does not use.
Reviewer attention: Confirm each channel repository's default branch is the intended base.

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

### Add 1.3.3 release notes to README, CHANGELOG and info versionChanges
Status: directed
Touches: `CHANGELOG.md`, `README.md`, `**/info.go`

The agent added a 1.3.3 release-notes entry to the CHANGELOG, the README, and the versionChanges text in info.go. Each was updated with a single entry for this release.

Reviewer attention: Confirm the release notes describe Windows archives as built but not run.

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
Touches: `.birdify/rfc-0023.activity.jsonl`

Plan files that no Birdify target owns are listed in unmappedFiles. The six unowned files are agent_skill.go, agent_skill_test.go, the generator scripts, Makefile, README.md and AGENT_INSTRUCTIONS.md, so they stay visible without being attributed to a target.

### Leave stale Birdify map unregenerated
Status: discussed
Touches: `.birdify/*`

The existing Birdify map is stale and still reflects revision 3. The agent leaves it unregenerated in this change.

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
Status: discussed
Touches: `apps/cli/setup/*`, `main.go`

Dolt initialization, git hooks and AGENTS.md remain per-repository work under `issuegraph init`. Only the global agent skill installation moved to `install`.

### Add global issuegraph install for agent skills
Status: discussed
Touches: `main.go`, `apps/cli/setup/*`

Running `issuegraph install` with no arguments installs the complete issuegraph skill into every detected agent's global skills directory. `install <agent>` installs only the named agent, which is the behavior previously provided by `setup claude`.

Reviewer attention: Confirm detection of installed agents matches the expected agent set on a clean machine.

### Rename setup command to install with no alias
Status: directed
Touches: `main.go`, `capability_registry.go`, `docs/**`, `README.md`

The agent renamed the `setup` command to `install` and removed `setup` entirely, with no alias and no transition period. Running `issuegraph setup` now reports an unknown command. Reference text across docs, tests, doctor hints and skill text was updated to match.

Considered/rejected: Keeping `setup` as an alias during a transition period was not chosen.
Risk: Breaking change for existing scripts and documentation that invoke `setup`.
Reviewer attention: Confirm release notes and external references no longer mention `issuegraph setup`.

### Default global agent install scope with a project flag
Status: discussed
Touches: `main.go`, `apps/cli/setup/*`

Agent skill installation defaults to global scope. A `--project` flag switches it to repository scope. Some recipes that previously defaulted to project-level installation now default to global.

Risk: Users relying on the previous project-level default will now install globally unless they pass `--project`.

### Embed the full issuegraph skill tree into the binary
Status: discussed
Touches: `issuegraph/skills/issuegraph/**`

The entire issuegraph skill tree is embedded into the binary: 46 files, about 290 KB, including SKILL.md, agents/openai.yaml, commands/ and resources/. The previous embed covered only SKILL.md and openai.yaml, which left the other files out of installs. The old duplicate copy under templates had no Go references and was deleted.

Risk: Binary size grows by about 290 KB of embedded skill content.
Reviewer attention: Confirm installed skill directories contain every file from issuegraph/skills/issuegraph.

### Use issuegraph/skills/issuegraph as the single skill source
Status: discussed
Touches: `issuegraph/skills/issuegraph/**`, `modules/core/internal/templates/skills/issuegraph/**`

The issuegraph/skills/issuegraph tree is the only source for the IssueGraph skill. No Go code referenced the older copy under templates, so the agent deleted that copy after confirming its contents differed from the source.

Reviewer attention: Confirm nothing outside Go code still reads the removed templates copy.

### Write agent global skill paths only after verification
Status: discussed
Touches: `apps/cli/setup/*.go`

Each agent's global skills path was checked against the `skills` CLI agent definitions and local directories before being coded. Claude, Codex, Hermes, and Pi match local directories. Gemini and Antigravity rely on the agent table alone.

Reviewer attention: Re-check paths when the upstream `skills` CLI agent table changes.

### Limit first-version install support to four agents
Status: discussed — proposed; agent list pending confirmation
Touches: `apps/cli/setup/*.go`

The first version proposes claude, codex, gemini and hermes, because their global skill directories were observed on the development machine. The list is pending confirmation before the RFC is approved.

Reviewer attention: Confirm the agent list before implementation.

### Defer post-init hint toward install to a follow-up
Status: discussed
Touches: `rfc/0023-*.md`

Whether `issuegraph init` should print a hint toward `install` is left open and deferred to a follow-up change, so init output is unchanged in this work.

### Draft RFC 0023 for global agent install
Status: discussed
Touches: `.spec/rfc/0023-global-agent-install.md`

The install feature is captured as RFC 0023 in Draft status. It covers the setup-to-install rename, global skill embedding, and per-agent path verification. The RFC passes validation and cross-file consistency checks, and no product code changes until it is approved.

### Support six agents in global skill install
Status: discussed
Touches: `apps/cli/setup/skills_install.go`

SkillAgents maps claude, codex, gemini, hermes, antigravity and pi to skill directories under each agent's config home. Claude, codex and hermes honor their home environment overrides. Antigravity detection uses ~/.gemini/antigravity and its skills path matches only the vercel skills CLI table.

Risk: The antigravity and pi paths are not confirmed on a real install on this machine.
Reviewer attention: Confirm the antigravity and pi skill directories on a machine where those agents are installed.

### Honor agent home environment overrides for skill paths
Status: discussed
Touches: `apps/cli/setup/*`

Agent skill paths honor the agent's home environment variables (CLAUDE_CONFIG_DIR, CODEX_HOME, HERMES_HOME) when they are set, and fall back to the default home-relative directory otherwise.

### Remove only the issuegraph skill directory on uninstall
Status: discussed
Touches: `apps/cli/setup/*.go`

The `--remove` flag deletes only the `issuegraph` skill directory for each agent. Sibling skills in the same skills directory are left untouched.

Reviewer attention: Confirm no sibling skill directory is deleted when removal runs.

### Keep recipe default install scope unchanged
Status: discussed — recipe defaults kept after the RFC draft proposed global scope
Touches: `apps/cli/setup.go`

The agent first planned global scope for every recipe. During implementation it kept each existing recipe's original default scope, so only the no-argument install command and the six agent skill installs are global.

Considered/rejected: Switching all recipes to global scope, as the RFC draft proposed.
Reviewer attention: Confirm the RFC text matches this narrower scope.

### Delete stale copy of the issuegraph skill under templates
Status: discussed
Touches: `modules/core/internal/templates/skills/issuegraph/**`

The agent removed the duplicate skill copy under the templates directory because it had drifted from the source tree. Tests in the templates package still pass afterwards.

Reviewer attention: Confirm the templates package has no remaining references to the deleted files.

### Generate embedded skill assets as part of make build
Status: discussed
Touches: `Makefile`, `scripts/generate-issuegraph-assets.mjs`, `modules/core/internal/generated/issuegraphassets/assets_gen.go`

Previously only `pnpm run build:go` generated the embedded skill assets. The agent made `make build` run the generator first, so a fresh clone that runs `make build` gets the skill assets without extra steps.

Risk: The generated file is gitignored, so a build that skips the generator fails to embed the skill.
Reviewer attention: Run `make build` on a clean clone and confirm `issuegraph install` succeeds.

### Fail the build when generated skill assets are stale
Status: discussed
Touches: `modules/core/internal/generated/issuegraphassets/*`, `scripts/generate-issuegraph-assets.mjs`

A test fails when the generated assets no longer match the skill source tree. This catches a skill edit that was not regenerated before a build.

Reviewer attention: Confirm the staleness check runs in CI.

### Run install without opening the store
Status: discussed
Touches: `apps/cli/main.go`, `apps/cli/capability_registry.go`

`install` is added to the no-database command list, so it runs from any directory. Without this, the command failed in directories with no IssueGraph database. The agent found the gap during end-to-end testing after the rename.

Reviewer attention: Confirm `install` never touches the database.

### Reword install status text for check and dry-run modes
Status: agent-initiated — wording change made during verification, not requested in the plan
Touches: `apps/cli/setup.go`

The status text in `--check` and `--dry-run` output read like an installed state even when nothing was installed. The agent changed the wording so both modes state what would happen or what is missing.

Reviewer attention: Confirm the wording is clear to users who read the output without context.

### Leave generated CLI reference unregenerated until release
Status: discussed
Touches: `docs/cli-reference/*`

The CLI reference is not regenerated for the new install command yet. Regeneration is deferred to the release step.

### Inject skill file map into InstallSkills for testability
Status: discussed
Touches: `apps/cli/setup/*`

InstallSkills accepts an injected map of skill files rather than reading the embedded content directly. Tests can therefore exercise installation without depending on the embedded asset contents.

### Cover install output and --check exit code in command tests
Status: discussed
Touches: `apps/cli/setup/*_test.go`

Command-level tests verify the install output and the exit code of `install --check`. Behavior is verified against a temporary HOME directory with fake agent directories, so no real agent configuration is touched.

### Revert three docs files to HEAD after rename
Status: discussed
Touches: `docs/*`

Three docs files modified by the mechanical rename were restored to HEAD byte-for-byte. This keeps only the intended `setup` to `install` changes and leaves unrelated content unchanged.

### Rewrite RFC 0023 to match implemented behavior and close it
Status: discussed
Touches: `rfc/0023-*.md`

RFC 0023 was rewritten to describe the implemented global install behavior and then closed. The open question about whether `init` should hint toward `install` was left for a follow-up.

### Record RFC 0023 Birdify plan after implementation
Status: directed
Touches: `.birdify/rfc-0023.activity.jsonl`, `.birdify/rfc-0023.html`

The agent writes a Birdify activity stream for the already-implemented RFC 0023 and renders it to HTML, so the record reflects shipped behavior rather than a pre-implementation plan.

### Do not disable the Netify hook to write Birdify files
Status: discussed
Touches: `.birdify/rfc-0023.activity.jsonl`

The agent leaves the Netify hook enabled and does not change its configuration. Writes that the hook blocks are not forced through by altering the hook.

### Model RFC 0023 Birdify record on RFC 0022 record format
Status: discussed
Touches: `.birdify/rfc-0023.activity.jsonl`

The agent follows the existing RFC 0022 Birdify record layout so the RFC 0023 record is consistent with prior records and renders with the same tooling.

### Keep setup.go file name and setup identifiers after rename
Status: discussed
Touches: `apps/cli/setup.go`

The install command replaces setup in Use, Short, Long and the event name. The source file and setup-prefixed identifiers keep their names. No explicit rationale was discussed.

Reviewer attention: Decide later whether the file and identifiers should be renamed to match the command.

### Write skill files atomically and report status by byte comparison
Status: discussed
Touches: `apps/cli/setup/skills_install.go`

InstallSkills writes each file through EnsureDir and atomicWriteFile. It compares existing bytes with the embedded skill content to classify each result as installed, updated, or unchanged.

Reviewer attention: Confirm the atomic write leaves no partial skill file when the process is interrupted.

### Create skill directories only for explicitly named agents
Status: discussed
Touches: `apps/cli/setup/skills_install.go`, `apps/cli/install_skills.go`

With no agent flag, the installer skips agents whose detect directory is absent. Agents named explicitly are created even when the detect directory is missing. An unknown agent name returns an error.

Reviewer attention: Confirm that auto mode never creates an agent directory on a machine where that agent is not installed.

### Emit install results as JSON with an agents array
Status: discussed
Touches: `apps/cli/install_skills.go`

The install command with --json prints its per-agent results under a top-level agents key. This gives scripts a stable structure for the install status.

### Exempt install from the no-database check
Status: discussed
Touches: `apps/cli/main.go`, `apps/cli/capability_registry.go`

The no-database allowlist names install in place of setup, so install runs without opening the issuegraph store. The capability registry entry is renamed to match.

Reviewer attention: Confirm that install does not require a database on a fresh checkout.

### Exclude historical docs from the install command rename
Status: discussed
Touches: `CHANGELOG.md`, `.spec/rfc/completed/`

The mass rename of the setup command in docs and source skipped CHANGELOG.md, release-gates, staged-for-removal, completed RFCs and graph outputs. Historical text keeps the old command name.

Reviewer attention: Confirm no live docs still tell readers to run issuegraph setup.

### Document install as a once-per-machine step in README
Status: discussed
Touches: `README.md`

The README adds a step after first init that installs the agent skill once per machine, with issuegraph install examples. Per-repository setup stays under issuegraph init.

### Hold release until blocking issues are resolved
Status: discussed
Touches: `Makefile`

The release is not ready because the shipped binaries cannot run embedded Dolt, the RFC files are uncommitted, and the doc pin is broken. The agent holds the release until those are fixed.

### Exclude reviewgraph-generated agent files and Graphify output from the release commit
Status: discussed
Touches: `graphify-out/*`, `.claude/settings.json.bak`, `.gemini/settings.json.bak`

Generated reviewgraph agent files and Graphify output are left out of the release commit so the commit contains only source and docs changes.

### Treat doc flag check failure as pre-existing, pending pin fix
Status: discussed
Touches: `docs/cli-docs.pin`

The doc flag check fails because the docs pin references a v1.2.2 tag that does not exist. The agent treats this as an existing issue and does not change the pin in this work.

Risk: The doc flag check stays red until the pin points to a real tag.

### Set Birdify targets to cli only
Status: discussed
Touches: `.birdify/rfc-0023.activity.jsonl`, `apps/cli/*`, `templates/agents/*`

The agent sets targets to cli because the whole apps/cli directory belongs to that target. The agent-guides target owns only templates/agents, not agent_skill.go, so it is not listed.

### Mark RFC 0023 planned event as written after implementation
Status: discussed
Touches: `.birdify/rfc-0023.activity.jsonl`

The planned event is recorded as reconstructed after the work was done, not as a plan written beforehand. Page content is an agent declaration rather than an automatic observation of the code.

Risk: Readers may take the planned event as pre-implementation intent when it was written afterward.
Reviewer attention: Confirm the planned event is labelled as after-the-fact and that the page does not present agent declarations as observed facts.

### Record vet, unit and e2e results as Birdify checks
Status: discussed
Touches: `.birdify/rfc-0023.activity.jsonl`

The three checks recorded (vet, unit tests, e2e) are results from runs performed in this session, not assumed passes. Each check is attached to the RFC 0023 record.

### Render RFC 0023 Birdify HTML and validate with no errors or warnings
Status: discussed
Touches: `.birdify/rfc-0023.html`

The agent renders the RFC 0023 Birdify page and runs validation, which reports zero errors and zero warnings. Visual layout was not inspected.

Reviewer attention: Open the rendered page and confirm the layout is readable, since only schema validation was run.

### Pause Birdify capability step on unresolved Netify gate
Status: discussed
Touches: `.birdify/rfc-0023.activity.jsonl`

A Netify gate rejected the IssueGraph capability row even though the graph index directories exist. The agent paused the Birdify capability step until the gate is resolved.

Risk: The capability record stays incomplete and the Netify gate remains unexplained.
Reviewer attention: Confirm the gate rejection cause and that the capability row matches the indexes present in the repository.

### Attribute embedded Dolt init failure to CGO_ENABLED=0 release builds
Status: discussed
Touches: `Makefile`

The Makefile defaults CGO_ENABLED to 1, so the failure comes from the release build command. The 1.3.x release binaries were built with CGO_ENABLED=0, which cannot run embedded Dolt and makes the default init fail.

Considered/rejected: Changing the Makefile default was considered and rejected because the default is already cgo-enabled.
Reviewer attention: Confirm the release build command passes cgo-enabled settings for every shipped target.

### Verify cgo-enabled release builds on macOS, Linux and Windows before tagging
Status: discussed
Touches: `Makefile`

The agent plans a local cgo-enabled macOS build that runs init, followed by cgo build checks on Linux and Windows. This verification has not run yet.

Risk: Cross-platform cgo builds may fail on platforms not yet checked.
Reviewer attention: Confirm init succeeds on each platform's cgo build before the release is tagged.

### Plan release 1.3.3 to ship cgo-enabled binaries
Status: discussed
Touches: `Makefile`

The agent plans a 1.3.3 patch release so that shipped binaries are built with cgo enabled and embedded Dolt works. The version files have not been bumped yet.

Reviewer attention: Confirm the version bump and the cgo build settings match before tagging.

### Keep RFC 0022 and RFC 0023 changes staged for a scoped commit
Status: discussed
Touches: `rfc/*`

The agent proposes committing only the RFC 0022 and RFC 0023 files, keeping generated and tool-output files out of the commit. The commit has not been made.

Reviewer attention: Confirm the staged set contains only the intended RFC files before committing.

### Mark RFC 0023 planned event as written after implementation (updated)
Status: discussed — planned event written after the work, not before
Touches: `.birdify/rfc-0023.activity.jsonl`

The agent wrote the planned event for RFC 0023 after implementation finished, so the activity log records the plan retrospectively. The log holds two events, planned and completed.

Considered/rejected: Writing the planned event before implementation was not possible in this session.
Reviewer attention: Confirm the retrospective planned event is acceptable for the evidence page.

### Render RFC 0023 Birdify HTML and validate with no errors or warnings (updated)
Status: discussed
Touches: `.birdify/rfc-0023.html`, `.birdify/rfc-0023.activity.jsonl`

The agent rendered the RFC 0023 Birdify HTML page from the activity log and validated it, reporting zero errors and zero warnings. The page was checked only by JSON validation and successful rendering, with no visual inspection.

Risk: Visual layout of the page was not checked.
Reviewer attention: Open the rendered page and confirm it displays correctly.

### Set Birdify targets to cli only (updated)
Status: discussed
Touches: `.birdify/rfc-0023.activity.jsonl`, `apps/cli`

The agent limited the Birdify targets for RFC 0023 to cli. The apps/cli directory is owned wholesale by the cli target, while the agent-guides target owns only templates/agents.

Reviewer attention: Confirm the target ownership split matches the module map.

### Keep six unowned plan files in unmappedFiles for Birdify event (updated)
Status: discussed
Touches: `.birdify/rfc-0023.activity.jsonl`, `agent_skill.go`, `agent_skill_test.go`, `Makefile`, `README.md`, `AGENT_INSTRUCTIONS.md`

The agent kept six plan files that no target owns in the unmappedFiles list. These are agent_skill.go, agent_skill_test.go, the generation script, Makefile, README.md and AGENT_INSTRUCTIONS.md.

Reviewer attention: Confirm these files should stay unmapped rather than be assigned to a target.

### Record vet, unit and e2e results as Birdify checks (updated)
Status: discussed
Touches: `.birdify/rfc-0023.activity.jsonl`

The agent recorded three Birdify checks for RFC 0023: vet, unit tests and e2e tests. These results come from runs executed live during this session.

Reviewer attention: Confirm the recorded check results match a fresh run.

### Flag Makefile CGO default mismatch with 1.3.2 release builds
Status: discussed
Touches: `Makefile`

The agent found that the Makefile defaults to CGO_ENABLED=1 while the 1.3.1 and 1.3.2 release artifacts were built with CGO_ENABLED=0. The mismatch explains why released binaries cannot initialize embedded Dolt.

Risk: Local builds and release builds now differ in cgo default, which can confuse future release work.
Reviewer attention: Confirm which CGO default the release pipeline should use.

### Use zig cc for Linux and Windows cgo cross-compilation
Status: discussed
Touches: `Makefile`

The agent cross-compiles the linux and windows cgo targets with go plus zig cc, targeting glibc 2.17 for Linux. Windows output was confirmed as PE32+ binaries and Linux output was exercised in Docker.

Reviewer attention: Confirm the glibc 2.17 floor is acceptable for supported Linux distributions.

### Build darwin-amd64 cgo binary with native Xcode clang
Status: discussed
Touches: `Makefile`

The agent builds the darwin-amd64 cgo binary with the host Xcode clang using -arch x86_64 because zig linking lacked the required macOS system frameworks.

Considered/rejected: zig cc for darwin-amd64 was rejected because linking failed on missing system frameworks.
Reviewer attention: Confirm the darwin-amd64 binary is built on a macOS host with Xcode, which the cross-compile path does not cover.

### Grade cgo binary verification by runtime evidence per target
Status: discussed
Touches: `scripts/build-release.sh`, `issuegraph_1.3.3_*`

Verification was graded per target by what actually ran. Official darwin-arm64, linux-amd64 and linux-arm64 binaries were executed, while Windows amd64 and arm64 were only built. The darwin-amd64 official archive was not executed; only an earlier test build was run under Rosetta.

Risk: Windows and darwin-amd64 release binaries have no runtime evidence for the official archives.
Reviewer attention: Run init, create and list on both Windows archives and on darwin-amd64 natively before treating the release as fully verified.

### Record cgo release binary size increase for distribution
Status: discussed
Touches: `scripts/build-release.sh`

Embedding Dolt makes each compressed cgo archive 71.9 to 91.5 MB: darwin-amd64 91.5 MB, darwin-arm64 85.5 MB, linux-amd64 82.2 MB, linux-arm64 76.1 MB, windows-amd64 78.3 MB, windows-arm64 71.9 MB. The 1.3.2 archives were 44 to 48 MB, so per-archive download size roughly doubles.

Risk: Download and storage size for each release archive roughly doubles compared with 1.3.2.

### Require RFC before changing build pipeline defaults
Status: discussed
Touches: `Makefile`

The agent treats changes to build pipeline defaults, such as the CGO default and release scripts, as needing a standalone RFC under the Specify workflow. No pipeline change is made until that RFC exists.

Reviewer attention: Confirm the RFC gate applies before editing the Makefile defaults.

### Keep issuegraph init scratch output out of repository
Status: discussed
Touches: `.issuegraph`

The agent ran issuegraph init in a temporary directory to verify the cgo binaries, so the created .issuegraph workspace stays outside the repository. It is not part of the repository changes.

Reviewer attention: Confirm no stray .issuegraph directory is staged in the repository.

### Attribute embedded Dolt init failure to CGO_ENABLED=0 release builds (updated)
Status: discussed
Touches: `Makefile`, `.issuegraph/`

The agent attributes the failure of issuegraph init on embedded Dolt to the CGO_ENABLED=0 build setting used for past release binaries, not to runtime code. Init was reproduced against a cgo build on macOS arm64 to confirm the cause.

Reviewer attention: Confirm the failure reproduces on a released no-cgo binary and is absent on the cgo build for each target.

### Verify cgo-enabled release builds on macOS, Linux and Windows before tagging (updated)
Status: discussed
Touches: `Makefile`

The agent verifies cgo release builds per platform before tagging, using runtime init on macOS and Linux and compile-plus-link checks on Windows. The gap in Windows runtime checks is recorded rather than treated as passing.

Reviewer attention: Decide whether compile-only evidence on Windows is enough to tag, or whether a runtime check must come first.

### Hold cgo and Dolt packaging changes until user confirms direction
Status: directed
Touches: `.spec/rfc/0024-init-managed-dolt.md`

The agent holds Dolt packaging and cgo changes until the direction is confirmed. RFC 0024 is written as Draft and no product code is changed before approval.

### Propose installing Dolt via install/init instead of bundling it
Status: directed
Touches: `.spec/rfc/0024-init-managed-dolt.md`

The agent proposed that issuegraph install provision the external dolt binary and that init only create the repository database, rather than bundling Dolt into the release binary. Bundling was rejected because it ties Dolt to the CGO_ENABLED build and increases binary size.

Considered/rejected: Bundling Dolt into the release binary, which requires a cgo build and increases binary size.

### Plan release 1.3.3 to ship cgo-enabled binaries (updated)
Status: discussed
Touches: `scripts/build-release.sh`

Release 1.3.3 is planned to ship cgo-enabled binaries built by the release script. The steps are: sync version files, update README and CHANGELOG, rebuild archives with the script, publish the GitHub Release, and then update the three channel pull requests. The existing 1.3.2 channel pull requests are to be closed with an explanatory note.

Reviewer attention: Confirm the 1.3.3 version sync and channel PR closure steps before tagging.

### Keep Dolt per machine and the database per repository
Status: discussed
Touches: `.spec/rfc/0024-init-managed-dolt.md`, `apps/cli/init.go`

The agent separated the dolt program from repository data. The dolt binary is installed once per machine, while each repository keeps its own database under .issuegraph/ created by init, similar to how git works per repository. Per-repository data lets each repository sync its issues independently with issuegraph dolt push and pull.

Considered/rejected: Installing a separate Dolt program in every repository, which duplicates the binary with no benefit.

### Draft RFC 0024 for Dolt setup via install and init
Status: discussed
Touches: `.spec/rfc/0024-init-managed-dolt.md`, `.spec/ROADMAP.md`, `.spec/TASK_TRACKING.md`

The agent wrote RFC 0024 as a Draft to define Dolt setup across install, init and the release binary. It extends the install step from RFC 0023 and replaces no existing RFC. The RFC is linked from ROADMAP.md and TASK_TRACKING.md and passes specify validate and sync-check.

### Number the Dolt RFC with the next id from specify init
Status: agent-initiated — process choice, not requested in prompts
Touches: `.spec/rfc/0024-init-managed-dolt.md`

The agent took the next free RFC id (0024) from the specify tool's init output so the new RFC does not collide with existing ids 0021 to 0023.

Considered/rejected: Picking an id by hand, which risks collision with existing RFCs.

### Keep release binary on CGO_ENABLED=0 instead of cgo cross-builds
Status: discussed
Touches: `.spec/rfc/0024-init-managed-dolt.md`

The proposal keeps the release binary built with CGO_ENABLED=0 so Dolt is no longer embedded. This reverses the earlier plan to ship cgo-enabled release binaries. Measured sizes were 204 to 283 MB for the cgo build and 101 MB for the no-cgo build.

Considered/rejected: Shipping cgo-enabled release binaries, which were verified to work but increase size and build complexity.
Reviewer attention: Confirm the release size tradeoff and that the earlier cgo verification is superseded.

### Keep embedded Dolt code behind a cgo build tag instead of deleting it
Status: discussed
Touches: `modules/core/internal/doltserver/doltserver.go`, `.spec/rfc/0024-init-managed-dolt.md`

The proposal keeps the embedded-mode code in source behind a cgo build tag rather than removing it. Whether to delete it later is deferred to a separate decision.

Reviewer attention: Confirm the embedded code still builds under the cgo tag.

### Resolve dolt through one helper that checks PATH then ~/.issuegraph/bin
Status: discussed
Touches: `apps/cli/init.go`, `modules/core/internal/doltserver/doltserver.go`

The existing exec.LookPath calls for dolt only search PATH and would miss a Dolt binary installed by issuegraph install. The proposal replaces them with one lookup function that checks PATH first and then ~/.issuegraph/bin.

Risk: A stale binary in PATH could take precedence over the installed copy.

### Default init to shared-server mode in CGO_ENABLED=0 builds instead of erroring
Status: discussed
Touches: `apps/cli/init.go`, `modules/core/internal/doltserver/doltserver.go`

A no-cgo binary's init currently fails and asks for a cgo build because embedded Dolt is unavailable. The proposal makes init default to shared-server mode in that case, so the release binary works with an externally installed dolt.

Risk: Shared-server data lives under ~/.issuegraph/shared-server, outside the repository, so clones need dolt pull or bootstrap to obtain data. Automatic backup is off in shared-server mode, and one server is a single point of failure for all repositories.
Reviewer attention: Confirm which process starts the shared server after a reboot and that concurrent init across several repositories is safe. Both are unverified.

### Verify no-cgo init fails and shared-server init works with external dolt
Status: discussed
Touches: `apps/cli/init.go`

The agent reproduced the failure with a CGO_ENABLED=0 binary, which reports that embedded Dolt requires a cgo build. It then downloaded an official dolt binary into scratch directories and confirmed that init --shared-server, create and list succeed.

### Install fetches the official dolt release into ~/.issuegraph/bin
Status: discussed — pending RFC 0024 approval
Touches: `.spec/rfc/0024-init-managed-dolt.md`

The proposal extends issuegraph install to download the official dolthub/dolt release archive into ~/.issuegraph/bin when dolt is missing, then verify it with dolt version. The --check and --dry-run modes also cover this step.

Considered/rejected: Calling brew or scoop, which depends on system package managers and differs per platform.
Risk: Release archive checksum verification is not yet confirmed.
Reviewer attention: Confirm the release archive can be verified before implementation starts.

### Mark Dolt download checksum, Windows and concurrent init as unverified in RFC 0024
Status: discussed
Touches: `.spec/rfc/0024-init-managed-dolt.md`

RFC 0024 lists unverified items explicitly: whether dolthub/dolt releases ship checksums, shared-server behavior on Windows, concurrent init across repositories, and which process restarts the shared server.

Reviewer attention: Each listed unverified item must be resolved before implementation is marked complete.

### Link RFC 0024 from ROADMAP and TASK_TRACKING
Status: discussed
Touches: `.spec/ROADMAP.md`

The agent corrected the RFC 0024 entry title in the ROADMAP because it still described the earlier no-bundling plan, which contradicted the rewritten RFC.

Reviewer attention: Confirm the ROADMAP title matches the RFC heading.

### Advance RFC 0024 to Approved before implementing
Status: discussed
Touches: `.spec/rfc/0024-init-managed-dolt.md`, `.spec/ROADMAP.md`

The agent advanced RFC 0024 to Approved after validation passed and the sync check reported no inconsistencies, so implementation work could proceed under the RFC.

Reviewer attention: Confirm the ROADMAP status row for 0024 reads Approved.

### Render RFC 0024 Birdify plan before implementation
Status: discussed
Touches: `.birdify/rfc-0024.html`

The agent updated the RFC 0024 Birdify plan to the packaging file set and re-rendered the HTML. The first plan write failed on an empty targets error for an active step, so the agent fixed the step targets before rendering.

Reviewer attention: Confirm the rendered HTML reflects the current file list and that unowned plan files are listed as unmapped.

### Pin Dolt 2.6.2 and verify downloads against pinned SHA256
Status: discussed
Touches: `modules/core/internal/doltbin/doltbin.go`, `modules/core/internal/doltbin/doltbin_test.go`

The doltbin package pins Dolt release 2.6.2 and per-asset SHA256 digests taken from GitHub release asset metadata, instead of querying the release API at runtime. A download whose checksum does not match is rejected before extraction, and the pinned version is at or above the recommended minimum in the doltversion policy.

Considered/rejected: Looking up the latest release at runtime and fetching a separate checksum file. The release archives publish no separate checksum file, and a runtime lookup would make installs non-reproducible.
Risk: Pinned digests go stale when Dolt is upgraded; each version bump requires new digests.
Reviewer attention: Confirm the pinned digests match the published release assets for each supported target.

### Restrict Dolt installation to supported platform targets
Status: discussed
Touches: `apps/cli/install_dolt.go`, `modules/core/internal/doltbin/doltbin.go`

Dolt installation supports only platform targets that have published release archives for the pinned version. Unsupported targets, such as Windows arm64, fail with an error from the install command instead of attempting a download, and the doltbin tests cover the unsupported-platform path.

Reviewer attention: Confirm the error message for unsupported targets is shown by the install command.

### Add ~/.issuegraph/bin to process PATH at CLI startup
Status: discussed
Touches: `apps/cli/main.go`

The CLI prepends the install directory for the managed dolt binary to its own process PATH at startup. Child processes inherit the change, so the existing dolt lookups in the store code resolve the managed binary without changing each lookup call site.

Considered/rejected: Changing each of the three PATH lookups individually. A single startup adjustment keeps the lookup logic in one helper.
Risk: Process-wide PATH mutation affects every child process the CLI starts.
Reviewer attention: Confirm no other binary in ~/.issuegraph/bin shadows the expected tool name.

### Run Dolt installation as the final step of issuegraph install
Status: discussed
Touches: `apps/cli/setup.go`, `apps/cli/install_dolt.go`

The no-argument install command runs the agent skill installation first and then installs dolt into ~/.issuegraph/bin. The skill-install result is preserved as the command result, and remove mode keeps its skill-only behavior.

Reviewer attention: Confirm the exit status when the skill step fails and the dolt step succeeds.

### Cover installDolt paths with command tests
Status: discussed
Touches: `apps/cli/install_dolt_test.go`

Tests cover the installDolt branches: dolt already present, dolt missing and then installed, check-only mode, dry-run mode, and installer failure. The tests run under the CGO_ENABLED=0 build with the gms_pure_go tag and pass.

Reviewer attention: Confirm the tests also run under the cgo-enabled build.

### Select new-workspace init default with cgo build-tagged constants
Status: discussed — init.go wiring was in progress when work paused
Touches: `apps/cli/init_mode_cgo.go`, `apps/cli/init_mode_nocgo.go`, `apps/cli/init.go`

Two build-tagged files define one constant. Builds without cgo, which are the release binaries, default new workspaces to shared-server mode, while cgo builds keep embedded Dolt as the default. The constant is read where init resolves the default mode and does not read or write any data.

Risk: If init.go wiring is incomplete, non-cgo builds fall back to the embedded default.
Reviewer attention: Verify init.go wiring is complete before relying on the default in non-cgo builds.

### Defer promoting the proxied Dolt server to default
Status: discussed
Touches: `rfc/0024-*.md`

The agent compared embedded, shared-server, and per-repository proxied modes. It recommended the shared server first because it is the only non-embedded mode with observed working behavior. The proxied server, which is marked experimental and lacks some commands, is kept as a follow-up pending concurrency and restart verification.

Considered/rejected: Per-repository proxied server as default: data stays in the repository, but the mode is experimental and lacks branch and ado sync commands. Embedded Dolt as default: the cgo binary grows substantially, and bundling Dolt conflicts with the decision to install it separately.
Risk: Shared-server data lives outside the repository, so one server failure affects every repository on the machine. Automatic backup is off, and restart behavior, concurrent init, and Windows support are unverified.
Reviewer attention: Confirm the RFC lists proxied-server promotion as a follow-up item.

### Leave Dolt RFC changes uncommitted for a scoped commit
Status: discussed
Touches: `apps/cli/`, `modules/core/internal/doltbin/`, `rfc/0024-*.md`

The agent left the Dolt install and init changes uncommitted in the current checkout so they can be committed as one scoped change after review.

Reviewer attention: Confirm the staged set contains only RFC 0024 files.

### Run Dolt setup from init per repository instead of install
Status: directed — scope set for this branch
Touches: `setup.go`, `install_dolt.go`, `init.go`

Dolt setup moves out of issuegraph install and into issuegraph init, which handles it when seeding a repository. Install is limited to agent skills, so the Dolt install call in setup.go is removed. Dolt is treated as a per-machine dependency, and each repository keeps its own database.

Considered/rejected: Running Dolt installation as the last step of issuegraph install, and a per-repository proxied server as the init default, were weighed and not adopted.
Risk: Uncommitted install_dolt.go and its call site in setup.go must be removed cleanly, or install behavior drifts.
Reviewer attention: Confirm install no longer touches Dolt and init seeds a new repository with a working embedded store.

### Share one Dolt binary cache in ~/.issuegraph/bin across repositories
Status: discussed
Touches: `doltbin/*`

Init fetches Dolt once into a machine-wide cache under ~/.issuegraph/bin when it is needed, and every repository uses that copy. A per-repository copy was avoided because the Dolt binary is about 112 MB.

Considered/rejected: Copying the Dolt binary into each repository's .issuegraph directory.
Risk: Concurrent first-time downloads from two repositories at once are unverified.
Reviewer attention: Confirm init uses the shared cache path and writes no per-repository copy.

### Revert RFC 0024 to Draft until the Dolt design is confirmed
Status: discussed
Touches: `.spec/rfc/0024-init-managed-dolt.md`

RFC 0024 is moved back from Approved to Draft because its text still describes a proxied-server design that does not match the chosen direction. The RFC body is not yet rewritten.

Considered/rejected: Keeping the proxied-server version of RFC 0024 in Approved status.
Risk: The Draft RFC text is stale until it is rewritten.
Reviewer attention: Confirm the RFC is rewritten to match the confirmed design before it returns to Approved.

### Hold further code and RFC edits until the embedded engine delivery is chosen
Status: discussed
Touches: `doltbin/*`, `.spec/rfc/0024-init-managed-dolt.md`

Code and RFC edits stop because the requested setup depends on how the embedded Dolt engine is delivered. Three options were laid out: a separately downloaded engine program that the small binary calls, two release builds (one small, one embedded), or an external dolt binary. Work waits on that choice.

Risk: Partial helper code remains in the tree while the design is open.
Reviewer attention: Confirm which delivery option is chosen before any further code or RFC changes.

### Record that embedded Dolt engine is compiled into the cgo binary
Status: discussed
Touches: `.issuegraph/embeddeddolt/*`, `init.go`

Embedded init writes only data files, about 2.2 MB, under .issuegraph/embeddeddolt. The Dolt engine is linked into the cgo build, which is about 204 MB, and the embedded storage code has no download path. A no-cgo binary therefore cannot write embedded data, which explains the init failures in the default 1.3.1 and 1.3.2 builds.

Considered/rejected: Assuming init downloads the embedded engine into .issuegraph/embeddeddolt. The code and a cgo run showed no download step.
Reviewer attention: Confirm the no-cgo init failure is the cause before choosing a release build layout.

### Verify proxied-server init with external dolt in an isolated home
Status: discussed
Touches: `.issuegraph/dolt/*`

A no-cgo binary with an external dolt was run through init with the proxied-server option, then create and list, in an isolated HOME directory. Separate repositories stayed isolated from each other, and concurrent creates in one repository all succeeded.

Considered/rejected: Making the per-repository proxied server the default init path, which was not adopted.
Reviewer attention: Confirm the proxied-server behavior holds outside the isolated test environment before relying on it.

### Leave in-progress Dolt code uncommitted and unreverted
Status: discussed
Touches: `doltbin/*`, `install_dolt.go`, `setup.go`, `main.go`

The temporary Dolt helper package, the install hook, PATH wiring at CLI startup, and two build-tag constant files stay in the working tree uncommitted until the design is confirmed. init.go is left unchanged.

Risk: Uncommitted pieces may be committed with unrelated changes if not scoped.
Reviewer attention: Confirm these files are reverted or committed separately from release changes.

### Ship embedded Dolt inside cgo release binaries instead of an external server
Status: discussed
Touches: `scripts/build-release.sh`, `issuegraph_1.3.3_*`

The agent ran the extracted official 1.3.3 archives with no dolt on PATH and confirmed init, create and list complete. This confirms the embedded engine path works from the shipped binaries without a separate Dolt install. The runs covered darwin-arm64, linux-amd64 and linux-arm64.

Reviewer attention: Confirm the no-dolt runs cover the intended first-run path on each shipped target.

### Revert uncommitted external-Dolt scaffolding before bundling
Status: discussed
Touches: `apps/cli/main.go`, `apps/cli/setup.go`, `apps/cli/init.go`

The agent removed the PATH augmentation for ~/.issuegraph/bin in main.go, the post-install Dolt step in setup.go, and the shared-server default in init.go. These were scaffolding for an external Dolt server, which the bundled-embedded-Dolt release path no longer needs.

Considered/rejected: Keeping the external-Dolt wiring behind a flag was not pursued; the scaffolding was reverted outright.
Reviewer attention: Confirm no remaining reference to the removed install or PATH helpers in apps/cli.

### Rewrite RFC 0024 to describe bundled embedded Dolt in release builds
Status: discussed
Touches: `.spec/rfc/0024-init-managed-dolt.md`

The agent rewrote RFC 0024 so it describes release binaries that bundle embedded Dolt through cgo builds, replacing the earlier plan of installing an external Dolt. The RFC was then validated with the specify tooling.

Reviewer attention: Confirm the RFC's stated release-binary behavior matches the build script and the 1.3.2 asset layout.

### Pause revert when Netify skill-check hook blocks git status
Status: discussed — tooling block, not a code decision
Touches: `modules/core/internal/doltbin/*`, `apps/cli/install_dolt.go`, `apps/cli/install_dolt_test.go`, `apps/cli/init_mode_cgo.go`, `apps/cli/init_mode_nocgo.go`

The agent's `git status` check on the scaffolding files was blocked by the Netify skill-check hook, which reported the required graph-index skills as missing. The agent did not get past the block. It stopped retrying, reported the block, and asked for a waiver before continuing the revert.

Considered/rejected: Retrying the same command or bypassing the hook. Both were skipped because the hook's reason did not match the state of the indexes.
Risk: Until the revert completes, the untracked external-Dolt files remain in the working tree.
Reviewer attention: Confirm the revert completed after the hook was resolved and that `git status` on those paths is clean.

### Add scripts/build-release.sh to produce cgo release archives
Status: discussed
Touches: `scripts/build-release.sh`

The agent added a build script that produces the cgo release archives for each target. The script was run to build the 1.3.3 archives in the background.

Reviewer attention: Confirm the script covers all six archive targets.

### Validate build script with a throwaway version before tagging
Status: discussed
Touches: `scripts/build-release.sh`

The agent validated the script with a 0.0.0-test version before any real release. The resulting archives carry that test version in their names and are not meant for publication.

### Re-run setup, templates and ui package tests after the revert
Status: discussed
Touches: `apps/cli/setup/`, `apps/cli/*.go`, `modules/core/internal/templates/`, `modules/core/internal/ui/`

After reverting the external-Dolt wiring, the agent re-ran the affected package tests with both cgo-enabled and cgo-disabled settings. All tests passed in the final run.

Reviewer attention: Confirm the cgo-off embedded Dolt failures are the known environmental case and not a regression from the revert.

### Exclude unrelated staged changes from the release commit
Status: directed
Touches: `CONTRIBUTING*`

The agent moved 19 unrelated paths out of the git index, including reviewgraph-generated files and CONTRIBUTING deletions, so the release commit only holds release changes. The content of those paths was kept in the working tree unstaged, not discarded.

Reviewer attention: Confirm the 19 unstaged paths are handled in a later scoped commit.

### Exclude reviewgraph-generated agent files and Graphify output from the release commit (updated)
Status: discussed
Touches: `.claude/settings.json`, `.claude/skills/explore-codebase/SKILL.md`, `.mcp.json`, `CLAUDE.md`, `AGENTS.md`

Agent-tool and Graphify generated files were kept out of the 1.3.3 release commit so it contains only version, changelog, README and release-note changes. Those files stay in the working tree.

Reviewer attention: Confirm the release commit lists only the 16 release paths plus the pre-commit version stamp.

### Pause archive runtime verification on blocked local hook
Status: discussed
Touches: `scripts/build-release.sh`

A local pre-tool hook blocked the shell commands needed to extract and run the built archives. The agent paused runtime verification and the release commit and tag, and continued only with read-only checks such as checksums and file listings.

Risk: Release sequencing is stalled until the blocking condition is cleared, and the extracted binaries remain unexecuted.
Reviewer attention: Confirm the runtime checks were completed after the hook condition was resolved.

### Verify locally built archives against generated SHA256SUMS
Status: discussed
Touches: `scripts/build-release.sh`

The agent checked the six locally built archives against the generated SHA256SUMS before publishing. All checksums matched.

Reviewer attention: Confirm the SHA256SUMS in the release matches the one verified locally.

### Commit RFC 0022 to 0024 work as one scoped commit
Status: directed
Touches: `scripts/build-release.sh`, `rfc/*.md`

The agent grouped the uncommitted RFC 0022, 0023 and 0024 code, docs, RFC files and Birdify artifacts into one commit (670884b, 84 files) so they land as a single reviewable change. The release build script was included in the same commit.

Reviewer attention: Confirm the 84-file commit contains only the intended RFC work and no unrelated paths.

### Fix pre-commit vet failure by declaring title as var
Status: discussed — surfaced by pre-commit lint during the commit
Touches: `**/install_skills.go`

The pre-commit lint flagged an invalid assignment the agent had introduced in install_skills.go. The agent fixed it before the commit landed.

Reviewer attention: Confirm the install_skills.go change keeps the intended behaviour.

### Stage only version stamp lines in .githooks/pre-commit for release
Status: discussed
Touches: `.githooks/pre-commit`

The release commit includes only the four version-stamp lines in the pre-commit hook. Other staged reviewgraph lines in that file were kept out of the release commit.

Risk: Partial staging of a hook file can leave the committed hook out of sync with the working copy.

### Run info and version tests with cgo enabled before release
Status: directed
Touches: `**/info*.go`, `**/version*.go`

The agent ran the info and version tests with cgo enabled before tagging, because the release binaries are cgo-enabled. The tests passed.

Reviewer attention: Confirm the cgo-enabled test run matches the release build configuration.

### Write release archives to scratch directory outside the repository
Status: discussed
Touches: `scripts/build-release.sh`

The release archives were written to a scratch directory outside the repository, so build output does not dirty the working tree. Only the archives and SHA256SUMS were uploaded to the release.

Risk: Archives exist only outside the repository, so rebuilding requires rerunning the script.

### Defer cli-docs pin update until the release exists
Status: discussed
Touches: `docs/cli-docs.pin`

docs/cli-docs.pin points to v1.2.2, which does not exist as a release. The agent left the pin unchanged and deferred the update until the 1.3.3 release is published.

Risk: Documentation generation keeps referencing a nonexistent release until the pin is updated.
Reviewer attention: Update docs/cli-docs.pin once the 1.3.3 release exists.

### Commit Birdify artifacts for RFC 0023 and 0024 with the code
Status: discussed
Touches: `.birdify/rfc-0023.html`, `.birdify/rfc-0023.activity.jsonl`, `.birdify/rfc-0024.html`, `.birdify/rfc-0024.activity.jsonl`

The generated Birdify HTML and activity logs for RFC 0023 and 0024 were committed alongside the code so the RFC records match the implemented state. They were treated as untracked outputs and added explicitly in the scoped commit.

Reviewer attention: Confirm the Birdify activity logs match the implemented RFC state.

### Verify cgo-enabled release builds on macOS, Linux and Windows before tagging (updated)
Status: discussed
Touches: `scripts/build-release.sh`, `issuegraph_1.3.3_*`

Builds were checked on macOS and Linux by running the official archives. Windows builds were produced by the build script but not executed. Tagging proceeded on the basis of checksum verification and the runtime checks that could run locally.

Risk: Windows cgo binaries have not been run, so a Windows-only embedded Dolt failure would not be caught before release.
Reviewer attention: Confirm Windows runtime checks are completed or accepted as a known gap.

### Publish v1.3.3 GitHub Release with six archives and SHA256SUMS
Status: directed
Touches: `scripts/build-release.sh`, `CHANGELOG.md`

The agent created the v1.3.3 GitHub Release from the tag, attaching six platform archives and a SHA256SUMS file produced by the release build script. The release is published, not left as a draft or prerelease, so channel updates can reference its asset URLs and hashes.

Reviewer attention: Confirm the release is public and that asset names match what the package channel manifests reference.

### Run extracted release binaries with no dolt on PATH before publishing
Status: directed
Touches: `scripts/build-release.sh`

The agent ran the extracted 1.3.3 binaries with no dolt on PATH. On darwin-arm64 locally, and on linux-amd64 and linux-arm64 in a Debian bookworm container, init, create and list succeeded and version reported 1.3.3.

Reviewer attention: Windows amd64 and arm64 binaries were not run; they need a manual init, create and list check.

### Describe Windows archives as built but not run in release notes
Status: discussed
Touches: `CHANGELOG.md`

The 1.3.3 changelog entry states that Windows archives were compiled but not executed during release verification. This keeps the published notes accurate about the level of testing behind the Windows builds.

Reviewer attention: Update the changelog entry once Windows runtime checks are complete.

### Keep RFC 0024 at Approved until Windows runtime checks complete
Status: discussed
Touches: `.spec/rfc/0024-init-managed-dolt.md`

The agent keeps RFC 0024 in Approved status rather than Implemented, because Windows binaries were built but have not been run. The lifecycle change to Implemented and archival are deferred until Windows runtime verification is recorded.

Reviewer attention: Confirm Windows runtime checks are done before advancing the RFC status and archiving it.

### Verify linux release binaries in Debian bookworm container
Status: discussed
Touches: `scripts/build-release.sh`

The agent ran the linux-amd64 and linux-arm64 release binaries in a Debian bookworm container to check runtime behaviour without a system dolt. This was the only Linux distribution tested.

Risk: glibc compatibility on older Linux distributions is not verified.
Reviewer attention: Confirm the Linux binaries run on the distributions users are expected to use.

### Mark darwin-amd64 release archive as not runtime-verified
Status: discussed
Touches: `scripts/build-release.sh`

The official darwin-amd64 archive was not executed in the release verification. Only an earlier test build was run under Rosetta, so the claim covers only that build.

Reviewer attention: Run the official darwin-amd64 archive with init, create and list before relying on it.

### Ship Windows amd64 only in the Scoop manifest
Status: directed
Touches: `bucket/issuegraph.json`

The Scoop manifest for the shennong bucket lists only the windows-amd64 archive. The windows-arm64 archive is still published on the release but is not referenced by the manifest.

Reviewer attention: Confirm the manifest autoupdate and checkver patterns match the windows-amd64 asset name.

### Point channel manifests at 1.3.3 release archives
Status: discussed
Touches: `Formula/*`, `nix/issuegraph.nix`

The Homebrew formula, Nix package and Scoop manifest were updated to the 1.3.3 release URLs, version and SHA256 values. Each hash was re-derived by downloading the published release asset, so the manifests match the live archives.

Reviewer attention: Confirm the SHA256 values in each channel PR match the published SHA256SUMS.

### Add autoPatchelfHook to the Nix package for dynamic glibc linkage
Status: discussed — agent-chosen for the Linux build, no separate request
Touches: `nix/issuegraph.nix`

The 1.3.3 Linux binaries link dynamically against glibc libraries (libc, libresolv, libpthread, libdl) instead of being static. The agent added autoPatchelfHook to the Nix package so those libraries are patched at build time.

Risk: The Nix build was not run because no Linux host was available, so the change is untested.
Reviewer attention: Run nix build on Linux before merging the Nix channel PR.

### Defer brew audit and brew test to the tap's CI
Status: discussed
Touches: `Formula/*`

The local brew audit command rejects file paths, so it could not run against the formula in the working tree. The agent left audit and test to the Homebrew tap's CI and did not modify the tap to work around the local check.

Risk: The formula has not been audited locally.
Reviewer attention: Confirm the tap CI audit and test jobs pass on the channel PR.

### Create channel update branches in channel checkouts without a worktree
Status: discussed
Touches: `Formula/issuegraph.rb`, `nix/issuegraph.nix`

The channel changes were made on branches inside the existing channel checkouts. No new git worktree was created for them.

### Hold RFC 0024 at Approved until channel PRs merge
Status: discussed
Touches: `.birdify/rfc-0024.activity.jsonl`

RFC 0024 stays at Approved while the channel PRs are open. The agent plans to advance it to Implemented and archive it only after those PRs merge.

### Replace nonexistent docs CLI pin v1.2.2 with v1.3.3
Status: directed
Touches: `docs/cli-docs.pin`

The docs CLI pin referenced tag v1.2.2, which does not exist in the repository. The agent changed the pin to v1.3.3 so the docs generation step resolves to a real release tag.

Considered/rejected: Creating a v1.2.2 tag to satisfy the existing pin.
Reviewer attention: Confirm the v1.3.3 tag exists before docs generation runs.

### Commit docs fix directly to main without branch or release
Status: directed
Touches: `docs/cli-docs.pin`, `docs/cli-reference/*`, `scripts/resolve-docs-issuegraph.sh`

The docs pin fix, regenerated CLI reference and build script change were committed straight to main and pushed. No release tag or release was created as part of this docs commit.

Reviewer attention: Confirm the commit contains only the docs pin, reference regeneration and script change.

### Build docs CLI from v1.3.3 tag with embedded skill assets generated first
Status: discussed
Touches: `scripts/resolve-docs-issuegraph.sh`, `scripts/generate-issuegraph-assets.mjs`

Starting with 1.3.3, the tag no longer commits the embedded skill resources. The docs build script now runs generate-issuegraph-assets.mjs before building, because building from the tag otherwise fails on a missing package.

Reviewer attention: Check that the generator runs before every docs CLI build path.

### Build docs CLI with CGO_ENABLED=0 and gms_pure_go tags
Status: discussed — build flags retained from the existing docs script path
Touches: `scripts/resolve-docs-issuegraph.sh`

The docs CLI build from the tag uses CGO_ENABLED=0 and the gms_pure_go build tag. This keeps the docs generation build independent of a C toolchain.

Reviewer attention: Confirm the pure-Go build still produces the docs output the reference expects.

### Regenerate CLI reference docs after the 1.3.3 release exists
Status: discussed
Touches: `docs/cli-reference/*`

The CLI reference was regenerated against the 1.3.3 binary. The stale setup.md page was removed and install.md was added to match the renamed command.

Reviewer attention: Confirm no other docs still link to the removed setup page.

### Verify generated CLI docs are fresh with --check before commit
Status: discussed
Touches: `docs/cli-reference/*`, `scripts/generate-cli-docs.sh`

The agent ran the generate-cli-docs script in check mode before committing. The check passed and reported the generated CLI docs as fresh.

Reviewer attention: Re-run the check on a clean tree to confirm it is reproducible.

### Delete stale standalone cli binary from ~/.local/bin
Status: directed
Touches: `~/.local/bin/cli`

The agent checked that the binary reported itself as issuegraph 1.3.0, an outdated version, before removing it. It then deleted the file and cleared the shell command hash so the old path is no longer resolved. The agent also searched shell profiles, Claude settings and config directories for references to the old path and found none.

Considered/rejected: Keeping the old binary as a fallback was considered and not chosen; the request was to remove it.
Risk: Any script or local tool that calls ~/.local/bin/cli directly will now fail.
Reviewer attention: Confirm no local script or alias still invokes ~/.local/bin/cli.

### Use the Homebrew issuegraph as the only installed binary
Status: directed — follows from the removal request
Touches: `/opt/homebrew/bin/issuegraph`

After removal, the agent confirmed the Homebrew issuegraph at /opt/homebrew/bin is the only issuegraph on PATH and reports version 1.3.3, matching the installed Homebrew formula. Local binaries are no longer maintained alongside the package-manager install.

Reviewer attention: Check that `which -a issuegraph` lists only the Homebrew path on other machines that used the local copy.
