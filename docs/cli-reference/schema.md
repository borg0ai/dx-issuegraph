---
title: "issuegraph schema"
description: "Print the JSON Schema for issuegraph's --json / export output"
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc schema`.

Print the JSON Schema for issuegraph's canonical output record types.

The schema is reflected from the same Go structs issuegraph serializes, so it stays in
lockstep with the actual --json / export output. Use it to generate typed
consumer models (datamodel-code-generator, quicktype, ...) rather than
hand-maintaining them.

  issuegraph schema | jq '.types.issue'        # the issue record schema
  issuegraph schema | jq '.types.dependency'   # the dependency record schema

```
issuegraph schema [flags]
```
