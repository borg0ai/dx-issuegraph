# Release Gate: be-nnqp + be-4b63 — issuegraph stats --no-blocked flag

**PR**: https://github.com/gastownhall/beads/pull/4022
**Branch**: `feat/be-nnqp-be-4b63-stats-no-blocked` (quad341/beads)
**Date**: 2026-05-21
**Deployer**: beads/deployer

## Gate Result: PASS

| # | Criterion | Evidence | Result |
|---|-----------|----------|--------|
| 1 | Review PASS present | be-98y3: PASS — "nil-guarding correct; LOW noted (JSON schema change `int→*int`), non-blocking" | ✅ PASS |
| 2 | Acceptance criteria met | See below | ✅ PASS |
| 3 | Tests pass | CI run 26001556038 — all 40 checks PASS (ubuntu, macOS, Windows smoke, Embedded Dolt cmd 1–20, Storage, lint, fmt, doc freshness, upgrade smokes) | ✅ PASS |
| 4 | No high-severity findings | be-98y3: no blockers — LOW-1 is JSON schema change (`blocked_issues: null` vs `0`) with explicit opt-in signal (`blocked_count_skipped: true`) | ✅ PASS |
| 5 | Final branch is clean | `git status` — clean; only rig artifacts untracked | ✅ PASS |
| 6 | Branch diverges cleanly from main | 3 commits ahead of `origin/main`; merge state CLEAN | ✅ PASS |

## Acceptance Criteria Verification (be-nnqp + be-4b63)

| Criterion | Evidence | Result |
|-----------|----------|--------|
| `issuegraph stats --no-blocked` completes (skips blocked scan) | `noBlocked=true` path calls `store.GetStatisticsNoBlocked(ctx)` instead of `store.GetStatistics(ctx)` | ✅ |
| `issuegraph stats` (default) behavior unchanged | Conditional flag check; non-flag path unchanged | ✅ |
| Human output shows `Blocked: (skipped)` with muted style | `ui.MutedStyle.Render("(skipped)")` on `stats.BlockedIssues == nil` path | ✅ |
| JSON: `blocked_issues` is null when `--no-blocked` | `BlockedIssues *int \`json:"blocked_issues"\`` — pointer, nil when no-blocked | ✅ |
| JSON: `blocked_count_skipped: true` when `--no-blocked` | `BlockedCountSkipped: stats.BlockedIssues == nil` in output struct | ✅ |
| JSON: `blocked_count_skipped` absent (omitempty) in default | `\`json:"blocked_count_skipped,omitempty"\`` on the bool field | ✅ |
| `Blocked: 0` renders in plain text (not red) | `else` branch (not warning) when `BlockedIssues != nil && *BlockedIssues == 0` — be-4b63 fix | ✅ |
| `Blocked: N>0` renders in warning/red style | `ui.RenderFail(...)` when `*BlockedIssues > 0` | ✅ |
| All callers nil-guarded | Reviewer confirmed "nil-guarding correct" | ✅ |
| CLI docs regenerated | `4ea5bd8fd docs(cli): regen CLI reference for bd status --no-blocked` | ✅ |

**Note on LOW-1** (reviewer): `BlockedIssues int→*int` is a JSON schema change: callers now receive `blocked_issues: null` instead of `blocked_issues: 0` when `--no-blocked` is used. Mitigated by `blocked_count_skipped: true` signal. Deliberate opt-in flag — acceptable.

## Commits

| SHA | Description |
|-----|-------------|
| `ddf53438c` | feat(stats): add --no-blocked flag; fix Blocked:0 red render (be-nnqp, be-4b63) |
| `ceb15695b` | fix(stats): F1 BlockedCountSkipped reflects actual nil; F2 Ready shows (skipped) with --no-blocked |
| `4ea5bd8fd` | docs(cli): regen CLI reference for `issuegraph status --no-blocked` |
