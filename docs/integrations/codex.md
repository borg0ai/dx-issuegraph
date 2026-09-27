---
title: Codex
description: Set up issuegraph for Codex with the issuegraph skill, a managed AGENTS.md section, and native hooks that survive compaction
---

Use IssueGraph with Codex through the `issuegraph` skill, managed `AGENTS.md` guidance, and native Codex hooks.

```bash
issuegraph setup codex
issuegraph setup codex --check
```

Project setup writes:

- `.agents/skills/issuegraph/` for the IssueGraph skill.
- `AGENTS.md` with a managed IssueGraph section.
- `.codex/config.toml` with `[features].hooks = true`.
- `.codex/hooks.json` with the IssueGraph hook fallback.

`issuegraph init` runs this project setup by default unless `--skip-agents` or `--stealth` is used. Global setup uses `issuegraph setup codex --global` and writes under `$CODEX_HOME` when set, otherwise `~/.codex`.

Codex 0.129.0+ supports `/hooks`, compact lifecycle hooks, and hook-provided developer context. IssueGraph uses that lifecycle to inject `issuegraph prime` on session start and recover context after compaction. Use `/hooks` to inspect or toggle the installed handlers.

## Hook Lifecycle

- `SessionStart` (`startup|resume|clear`) injects full `issuegraph prime` output.
- `PreCompact` (`manual|auto`) checks `issuegraph prime --memories-only` and warns if IssueGraph context is unavailable.
- `PostCompact` (`manual|auto`) records that the session needs an IssueGraph refresh.
- `UserPromptSubmit` injects full `issuegraph prime` once after compaction, then clears the refresh marker.

`PreCompact` alone does not inject context because Codex ignores plain stdout from compact hooks. The post-compact marker plus first-prompt refresh is the reliable recovery path.

Refresh markers are stored in a user cache/temp directory keyed by Codex `session_id` and workspace path. They are not written to tracked files or to the IssueGraph database.

The IssueGraph Codex plugin stores hooks at `plugins/beads/.codex-plugin/hooks/hooks.json` and declares them in `plugins/beads/.codex-plugin/plugin.json` as `"hooks": "./.codex-plugin/hooks/hooks.json"`. Without the plugin, `issuegraph setup codex` installs the same hook config in `.codex/hooks.json` and enables `[features].hooks = true`.

## Manual Fallback

If you manage `.codex/hooks.json` by hand instead of running `issuegraph setup codex`, the equivalent shape is:

```json
{
  "hooks": {
    "SessionStart": [
      {
        "matcher": "startup|resume|clear",
        "hooks": [{ "type": "command", "command": "issuegraph codex-hook SessionStart", "statusMessage": "Loading IssueGraph context" }]
      }
    ],
    "PreCompact": [
      {
        "matcher": "manual|auto",
        "hooks": [{ "type": "command", "command": "issuegraph codex-hook PreCompact", "statusMessage": "Checking IssueGraph context" }]
      }
    ],
    "PostCompact": [
      {
        "matcher": "manual|auto",
        "hooks": [{ "type": "command", "command": "issuegraph codex-hook PostCompact", "statusMessage": "Scheduling IssueGraph context refresh" }]
      }
    ],
    "UserPromptSubmit": [
      {
        "hooks": [{ "type": "command", "command": "issuegraph codex-hook UserPromptSubmit", "statusMessage": "Refreshing IssueGraph context" }]
      }
    ]
  }
}
```

Then ensure `.codex/config.toml` enables:

```toml
[features]
hooks = true
```
