---
title: "issuegraph rules"
description: "Audit and compact Claude rules"
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc rules`.

Audit and compact Claude rules

```
issuegraph rules [command]
```

## issuegraph rules audit

Scan rules for contradictions and merge opportunities

```
issuegraph rules audit [flags]
```

**Flags:**

```
      --path string       Path to rules directory (default ".claude/rules/")
      --threshold float   Jaccard similarity threshold (default 0.6)
```

## issuegraph rules compact

Merge related rules into composites

```
issuegraph rules compact [flags]
```

**Flags:**

```
      --auto            Apply audit suggestions
      --dry-run         Preview without applying
      --group strings   Rule names to merge
      --path string     Path to rules directory (default ".claude/rules/")
```
