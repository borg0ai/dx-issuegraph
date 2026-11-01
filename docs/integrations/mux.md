---
title: Mux
description: Set up issuegraph for Mux with a managed AGENTS.md section, optional layered instruction files, and Mux hooks
---

Use IssueGraph with Mux through `AGENTS.md`, optional layered Mux instruction files, and Mux hooks.

```bash
issuegraph install mux
issuegraph install mux --check
```

The default setup writes a managed IssueGraph section to root `AGENTS.md`.

## Workspace and Global Layers

Mux also supports workspace and global instruction layers:

```bash
issuegraph install mux --project
issuegraph install mux --global
issuegraph install mux --project --global
```

Project setup writes `.mux/AGENTS.md` and installs Mux hook files under `.mux/`:

- `.mux/init`
- `.mux/tool_post`
- `.mux/tool_env`

Global setup writes `~/.mux/AGENTS.md`.

## Remove

```bash
issuegraph install mux --remove
issuegraph install mux --project --remove
issuegraph install mux --global --remove
```
