---
title: Gemini CLI
description: Set up issuegraph for Gemini CLI with SessionStart hooks that run issuegraph prime and GEMINI.md workflow guidance
---

Use IssueGraph with Gemini CLI through SessionStart hooks and `GEMINI.md` guidance.

```bash
issuegraph install gemini
issuegraph install gemini --check
```

By default, setup installs global hooks in `~/.gemini/settings.json`. For project-local hooks, use:

```bash
issuegraph install gemini --project
```

The hook runs `issuegraph prime --hook-json` so Gemini receives compact IssueGraph workflow context at session start. The setup also writes IssueGraph guidance to `GEMINI.md`.

## Stealth Mode

For CI or other environments where setup should avoid git operations:

```bash
issuegraph install gemini --stealth
issuegraph install gemini --project --stealth
```

## Remove

```bash
issuegraph install gemini --remove
issuegraph install gemini --project --remove
```
