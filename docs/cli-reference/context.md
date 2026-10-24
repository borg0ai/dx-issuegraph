---
title: "issuegraph context"
description: "Show the effective backend identity information including repository paths,"
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc context`.

Show the effective backend identity information including repository paths,
backend configuration, and sync settings.

This command reads directly from config files and does not require the
database to be open, making it useful for diagnostics in degraded states.

Examples:
  issuegraph context           # Show context information
  issuegraph context --json    # Output in JSON format


```
issuegraph context [flags]
```
