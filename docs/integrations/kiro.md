---
id: kiro
title: Kiro CLI
---

# Kiro CLI Integration

Use IssueGraph with Kiro CLI through workspace steering guidance.

```bash
issuegraph setup kiro
issuegraph setup kiro --check
```

The setup command writes IssueGraph workflow guidance to `.kiro/steering/beads.md`.
Kiro reads workspace steering files when it starts a session.

## Remove

```bash
issuegraph setup kiro --remove
```
