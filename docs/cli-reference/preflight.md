---
title: "issuegraph preflight"
description: "Show PR readiness checklist"
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc preflight`.

Display a checklist of common pre-PR checks for contributors.

This command helps catch common issues before pushing to CI:
- Tests not run locally
- Lint errors
- Unformatted Go files
- .issuegraph/issues.jsonl pollution
- Version mismatches

Examples:
  issuegraph preflight              # Show checklist
  issuegraph preflight --check      # Run checks automatically
  issuegraph preflight --check --json  # JSON output for programmatic use
  issuegraph preflight --check --skip-lint  # Explicitly skip lint check


```
issuegraph preflight [flags]
```

**Flags:**

```
      --check       Run checks automatically
      --json        Output results as JSON
      --skip-lint   Skip lint check explicitly
```
