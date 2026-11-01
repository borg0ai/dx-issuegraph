# RFC 0023: issuegraph install: global agent skill install

**Status:** Implemented

## Summary

Replace the `issuegraph setup` command with `issuegraph install`, with no alias. `issuegraph install` is the global entry point: with no argument it installs the complete IssueGraph agent skill into the global skills directory of every detected coding agent, so agents know how to use issuegraph in any repository. `issuegraph install AGENT` does the same for one agent and takes over everything `setup AGENT` did. Per-repository work stays in `issuegraph init` (Dolt storage under `.issuegraph/`, git hooks, `AGENTS.md`).

## Problem

- Package managers install only the binary, so agents never learn that issuegraph exists.
- Global agent integration is scattered: `setup AGENT --global` covers only claude, codex, cursor and mux, needs one call per agent, and the other agents (gemini, opencode, windsurf, copilot, kiro, qoder, codebuddy, hermes) are project-only.
- The sibling tools `reviewgraph install` and `birdify install` each have one global command; issuegraph has none.
- The embedded skill is incomplete. `scripts/generate-issuegraph-assets.mjs` embeds only `SKILL.md` and `agents/openai.yaml` from `issuegraph/skills/issuegraph`, so `commands/` and `resources/`, which the skill refers to, never reach an agent.
- A second, diverged copy of the skill lives in `modules/core/internal/templates/skills/issuegraph`.

## Goals

- One command after installing the binary: `issuegraph install`.
- The whole skill (`SKILL.md`, `agents/openai.yaml`, `commands/`, `resources/`) is built into the binary and written for every detected agent. Skills are always part of the install; there is no skip flag.
- Idempotent: each agent reports installed, updated, unchanged or skipped. No network access. Nothing is written inside the current repository unless `--project` is given.
- `setup` is gone: one command name, no alias, no deprecation period.

## Non-goals

- Per-repository setup (Dolt database, git hooks, `AGENTS.md` generation), which stays in `issuegraph init`.
- Installing the binary itself, MCP server configuration, and post-install hooks inside Homebrew, Nix or Scoop.
- A `doctor` check for the global skill (follow-up).

## Design

1. Rename the command: the cobra command becomes `Use: "install [recipe]"` and `setup` is gone. The Go file `apps/cli/setup.go` and its `setup*` identifiers keep their names to keep the diff small. Keep the existing flags `--global`, `--project`, `--check`, `--remove`, `--list`, `--print`, `--stealth`, `--add` and `-o`. Add `--dry-run`. There is no `--yes` flag because nothing prompts. Because `install` must work in any directory, it replaces `setup` in the no-database command list in `main.go` and in the capability registry.
2. Semantics: `issuegraph install` with no recipe installs the skill globally for every detected agent; `--check` and `--remove` apply to the same set. `issuegraph install AGENT` for claude, codex, gemini, hermes, antigravity or pi runs that agent's recipe (claude, codex and gemini only) and then installs the global skill for it. Every other recipe behaves exactly as under `setup`, with its previous default scope.
3. Update every reference to `issuegraph setup` (about 306 occurrences in `docs/integrations`, `apps/cli/setup`, `apps/cli`, `apps/cli/doctor`, `docs/cli-reference`, `docs/getting-started`, `README.md`, `modules/core/internal/*`, `integrations/junie` and the skill text) and the tests that assert the old name.
4. Build the full skill into the binary: extend `scripts/generate-issuegraph-assets.mjs` so the generated package embeds the tree `issuegraph/skills/issuegraph` (`SKILL.md`, `agents/openai.yaml`, `commands/`, `resources/`; not `README.md`, `CLAUDE.md` or `adr/`). `issuegraph/skills/issuegraph` is the single source of truth; delete the diverged copy under `modules/core/internal/templates/skills`. Run `pnpm run generate:assets` from the build, and add a test that fails when the generated file is stale.
5. Agent table in `apps/cli/setup/skills_install.go`, with six agents: claude (`$CLAUDE_CONFIG_DIR` or `~/.claude`), codex (`$CODEX_HOME` or `~/.codex`), gemini (`~/.gemini`), hermes (`$HERMES_HOME` or `~/.hermes`), antigravity (`~/.gemini/antigravity`) and pi (`~/.pi/agent`); the skill goes in the `skills/issuegraph` directory under each. Paths come from the vercel-labs/skills agent table and, for claude, codex, gemini, hermes and pi, from directories that exist on the development machine; antigravity is verified only by the skills table. Each entry has each agent has a detect directory and a global skills directory (for example `~/.claude/skills/issuegraph`, `~/.codex/skills/issuegraph`, `~/.gemini/skills/issuegraph`, `~/.hermes/skills/issuegraph`). An agent enters the table only after its global skills path is verified against that agent's documentation; unverified agents are skipped, never guessed. An agent is detected when its config directory exists under `$HOME`; missing directories are not created.
6. Idempotence is a byte comparison per file. `--remove` deletes only the `issuegraph` skill directory it created.
7. Output uses the RFC 0022 helpers; `--json` prints one record per agent.
8. Docs: README install section, `AGENT_INSTRUCTIONS.md`, `docs/cli-reference`.

Breaking: `issuegraph setup` stops working. `docs/cli-reference` is generated from the release pinned in `docs/cli-docs.pin` and is regenerated at the next release, so it still shows `setup` until then.

## Open questions

Resolved by the owner: the first release supports claude, codex, gemini, hermes, antigravity and pi. The hint printed by `issuegraph init` when no global skill is found is a follow-up, as is a `doctor` check.

## Acceptance

With a temporary HOME holding fake `.claude`, `.codex` and `.gemini` directories: `issuegraph install` writes the full skill (`SKILL.md`, `agents/openai.yaml`, `commands/`, `resources/`) under each; a second run reports unchanged and changes no file; `--dry-run` writes nothing; `install claude` touches only claude; `--remove` deletes only the issuegraph skill directories; agents without a config directory are skipped; no file appears in the working repository; `--json` is valid. `issuegraph setup` exits with an unknown-command error and `grep` finds no `issuegraph setup` outside `CHANGELOG.md` and completed RFCs. The generated-assets staleness test passes. Go tests use `t.Setenv("HOME", ...)`. `go vet` passes with cgo on and off.
