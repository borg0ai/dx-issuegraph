---
title: Factory.ai Droid
description: Set up issuegraph for Factory.ai Droid through a managed IssueGraph section in AGENTS.md
---

Use IssueGraph with Factory.ai Droid through managed `AGENTS.md` guidance.

```bash
issuegraph install factory
issuegraph install factory --check
```

The setup command creates or updates `AGENTS.md` with a managed IssueGraph section. Factory Droid reads `AGENTS.md` automatically when it starts a session.

## Remove

```bash
issuegraph install factory --remove
```
