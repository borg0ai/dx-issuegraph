---
title: "issuegraph onboard"
description: "Display a minimal snippet to add to your agent instructions file for issuegraph integration."
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc onboard`.

Display a minimal snippet to add to your agent instructions file for issuegraph integration.

By default, the agent instructions file is AGENTS.md. Use 'issuegraph init --agents-file'
to configure a different filename (e.g. ISSUEGRAPH.md).

This outputs a small (~10 line) snippet that points to 'issuegraph prime' for full
workflow context. This is the same minimal profile that 'issuegraph init' generates
by default. This approach:

  • Keeps your agent file lean (doesn't bloat with instructions)
  • issuegraph prime provides dynamic, always-current workflow details
  • Hooks auto-inject issuegraph prime at session start

For agents or environments that do not auto-inject hook output, use
'issuegraph init --agents-profile=full' to embed the complete command reference.

```
issuegraph onboard [flags]
```
