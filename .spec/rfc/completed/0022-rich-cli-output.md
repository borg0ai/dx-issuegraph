# RFC 0022: Rich colorful CLI output for key commands

**Status:** Implemented

## Summary

Replace plain log-style output (`=====` underlines, bare `Note:` lines, unstyled key/value dumps) with a small shared rendering layer in `modules/core/internal/ui` that produces colored, structured messages: a branded banner, section panels, aligned key-value blocks, leveled status lines (success, warning, error, info, hint). Apply it first to `info`, `version` and the `doctor` embedded-mode notice (`doctor` and `init` already use `ui.Render*`, so only their plain spots change). A spinner is deferred: none of these commands has a slow step to wrap. Output stays plain when stdout is not a TTY, when `NO_COLOR` is set, in agent mode, and under `--json`.

## Problem

Most commands print with raw `fmt.Print*` (about 4,100 call sites versus about 900 through `ui`). The result reads as a log: `issuegraph info` prints a title over a `=====` line and unstyled fields, and unsupported-mode notices are a bare `Note:` line. Errors, warnings and next steps do not stand out, so users miss what to do next. The existing design system (status glyphs `○ ◐ ● ✓ ❄`, P-labels, lipgloss v2 adaptive colors) is only applied to issue lists.

## Goals

One consistent, colorful look for messages: clear hierarchy (banner, sections, key-value), semantic color per level, and visible next-step hints. One small API that new commands use instead of `fmt.Print*`. Zero change for scripts and agents.

## Non-goals

A full-screen interactive Bubble Tea application (open question below). Emoji-style icons: `AGENT_INSTRUCTIONS.md` Visual Design System forbids them, so richness comes from color, Unicode glyphs, borders and spacing. Converting all 4,100 print sites. Changing `--json` output, exit codes, or agent-mode text. New dependencies beyond lipgloss v2 already in `apps/cli/go.mod`.

## Design

1. `modules/core/internal/ui`: add `message.go` with `Banner`, `Section`, `Field`, `Success`, `Warn`, `Error`, `Info` and `Hint` (no `Spinner`), built on the existing `styles.go` palette (`initColors`, adaptive dark and light). Every function degrades to the current plain text when `ShouldUseColor()` is false or `IsAgentMode()` is true, and writes to stderr or stdout exactly where today's text goes.
2. Migrate `info`, `version` and the `doctor` embedded-mode notice in `apps/cli`. Plain output keeps every existing line; the only plain change is the `info` underline, now as long as the title.
3. Add golden tests per function for three modes: color TTY, `NO_COLOR`, and non-TTY or agent.
4. Add tests for `agentDoctorResult` (doctor agent mode), which has no covering tests today, before routing `doctor` through the new API.
5. Extend the Visual Design System section of `AGENT_INSTRUCTIONS.md` with the message levels and their colors and glyphs.

Level glyphs reuse the existing `ui` constants and renderers (`IconPass ✓` green via `RenderPass`, `IconWarn ⚠` amber via `RenderWarn`, `IconFail ✖` red via `RenderFail`, `IconInfo ℹ` blue via `RenderAccent`); only the hint glyph `→` in muted color is new. `styles.go` and `terminal.go` are not modified: `message.go` is additive, which keeps the ReviewGraph blast radius (394 impacted nodes if `styles.go` changed) at zero for existing callers. Section header: bold with a colored rule.

## Open questions

Does "TUI" mean only styled message output, or also a full-screen interactive Bubble Tea view (for example a live `ready` board)? This RFC covers styled output; a full-screen view should be its own RFC.

## Acceptance

Rendering the four commands in a color TTY shows the banner, sections, aligned key-value blocks and leveled lines. The same commands with `NO_COLOR=1`, piped to a file, or in agent mode produce no ANSI escapes and keep today's text. `--json` output is byte-identical to before. Golden tests cover the three modes. `go vet` and the touched package tests pass with CGO on and off.
