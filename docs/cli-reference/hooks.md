---
title: "issuegraph hooks"
description: "Manage git hooks for issuegraph integration"
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc hooks`.

Install, uninstall, or list git hooks for issuegraph integration.

The hooks provide:
- pre-commit: Run chained hooks before commit
- post-merge: Run chained hooks after pull/merge
- pre-push: Run chained hooks before push
- post-checkout: Run chained hooks after branch checkout
- prepare-commit-msg: Add agent identity trailers for forensics

```
issuegraph hooks [command]
```

## issuegraph hooks install

Install git hooks for issuegraph integration.

By default, hooks are installed to .git/hooks/ in the current repository.
Use --issuegraph to install to .issuegraph/hooks/ (recommended for Dolt backend).
Use --shared to install to a versioned directory (.issuegraph-hooks/) that can be
committed to git and shared with team members.

Hooks use section markers to coexist with existing hooks — any user content
outside the markers is preserved across installs and upgrades.

Installed hooks:
  - pre-commit: Run chained hooks before commit
  - post-merge: Run chained hooks after pull/merge
  - pre-push: Run chained hooks before push
  - post-checkout: Run chained hooks after branch checkout
  - prepare-commit-msg: Add agent identity trailers (for orchestrator agents)

```
issuegraph hooks install [flags]
```

**Flags:**

```
      --chain        No-op, kept for compatibility (existing hook content always runs alongside the issuegraph section)
      --force        No-op, kept for compatibility (section markers always preserve non-issuegraph content)
      --issuegraph   Install hooks to .issuegraph/hooks/ (recommended for Dolt backend)
      --shared       Install hooks to .issuegraph-hooks/ (versioned) instead of .git/hooks/
```

## issuegraph hooks list

Show the status of issuegraph git hooks (installed, outdated, missing).

```
issuegraph hooks list [flags]
```

## issuegraph hooks run

Execute the logic for a git hook. This command is typically called by
thin shim scripts installed in .git/hooks/.

Supported hooks:
  - pre-commit: Run chained hooks before commit
  - post-merge: Run chained hooks after pull/merge
  - pre-push: Run chained hooks before push
  - post-checkout: Run chained hooks after branch checkout
  - prepare-commit-msg: Add agent identity trailers for forensics

The thin shim keeps delegated hook logic in sync with the installed issuegraph
version. Upgrading issuegraph updates that delegated behavior. To adopt changes to the
shim's generated shell policy, refresh it with 'issuegraph hooks install'.

```
issuegraph hooks run <hook-name> [args...] [flags]
```

## issuegraph hooks uninstall

Remove issuegraph git hooks from .git/hooks/ directory.

```
issuegraph hooks uninstall [flags]
```
