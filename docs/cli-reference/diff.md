---
title: "issuegraph diff"
description: "Show changes between two commits or branches"
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc diff`.

Show the differences in issues between two commits or branches.

The refs can be:
- Commit hashes (e.g., abc123def)
- Branch names (e.g., main, feature-branch)
- Special refs like HEAD, HEAD~1

Examples:
  issuegraph diff main feature-branch   # Compare main to feature branch
  issuegraph diff HEAD~5 HEAD           # Show changes in last 5 commits
  issuegraph diff abc123 def456         # Compare two specific commits

```
issuegraph diff <from-ref> <to-ref> [flags]
```
