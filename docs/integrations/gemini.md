---
title: Gemini CLI
description: Set up issuegraph for Gemini CLI with SessionStart hooks that run issuegraph prime and GEMINI.md workflow guidance
---

Use IssueGraph with Gemini CLI through SessionStart hooks and `GEMINI.md` guidance.

```bash
issuegraph setup gemini
issuegraph setup gemini --check
```

By default, setup installs global hooks in `~/.gemini/settings.json`. For project-local hooks, use:

```bash
issuegraph setup gemini --project
```

The hook runs `issuegraph prime --hook-json` so Gemini receives compact IssueGraph workflow context at session start. The setup also writes IssueGraph guidance to `GEMINI.md`.

## Stealth Mode

For CI or other environments where setup should avoid git operations:

```bash
issuegraph setup gemini --stealth
issuegraph setup gemini --project --stealth
```

## Remove

```bash
issuegraph setup gemini --remove
issuegraph setup gemini --project --remove
```
