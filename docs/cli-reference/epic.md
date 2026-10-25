---
title: "issuegraph epic"
description: "Epic management commands"
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc epic`.

Epic management commands

```
issuegraph epic [command]
```

## issuegraph epic close-eligible

Close epics where all children are complete

```
issuegraph epic close-eligible [flags]
```

**Flags:**

```
      --dry-run         Preview what would be closed without making changes
  -r, --reason string   Close reason applied to every epic closed (default "All children completed")
```

## issuegraph epic status

Show epic completion status

```
issuegraph epic status [flags]
```

**Flags:**

```
      --eligible-only   Show only epics eligible for closure
```
