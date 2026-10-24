---
title: "issuegraph formula"
description: "Manage workflow formulas - the source layer for molecule templates."
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc formula`.

Manage workflow formulas - the source layer for molecule templates.

Formulas are TOML/JSON files that define workflows with composition rules.
Define formulas, cook them into protos, then pour or wisp them into work.

Search paths (in order):
  1. &lt;resolved-issuegraph-dir&gt;/formulas/ (active project)
  2. &lt;checkout-root&gt;/.issuegraph/formulas/ (repo-local formulas)
  3. ~/.issuegraph/formulas/ (user)
  4. $GT_ROOT/.issuegraph/formulas/ (shared workspace root, if GT_ROOT set)

Commands:
  list   List available formulas from all search paths
  show   Show formula details, steps, and composition rules

```
issuegraph formula [flags]
```

## issuegraph formula convert

Convert formula files from JSON to TOML format.

TOML format provides better ergonomics:
  - Multi-line strings without \n escaping
  - Human-readable diffs
  - Comments allowed

The convert command reads a .formula.json file and outputs .formula.toml.
The original JSON file is preserved (use --delete to remove it).

Examples:
  issuegraph formula convert shiny              # Convert shiny.formula.json to .toml
  issuegraph formula convert ./my.formula.json  # Convert specific file
  issuegraph formula convert --all              # Convert all JSON formulas
  issuegraph formula convert shiny --delete     # Convert and remove JSON file
  issuegraph formula convert shiny --stdout     # Print TOML to stdout

```
issuegraph formula convert <formula-name|path> [--all] [flags]
```

**Flags:**

```
      --all      Convert all JSON formulas
      --delete   Delete JSON file after conversion
      --stdout   Print TOML to stdout instead of file
```

## issuegraph formula list

List all formulas from search paths.

Search paths (in order of priority):
  1. &lt;resolved-issuegraph-dir&gt;/formulas/ (active project - highest priority)
  2. &lt;checkout-root&gt;/.issuegraph/formulas/ (repo-local formulas)
  3. ~/.issuegraph/formulas/ (user)
  4. $GT_ROOT/.issuegraph/formulas/ (shared workspace root, if GT_ROOT set)

Formulas in earlier paths shadow those with the same name in later paths.

Examples:
  issuegraph formula list
  issuegraph formula list --json
  issuegraph formula list --type workflow
  issuegraph formula list --type convoy

```
issuegraph formula list [flags]
```

**Flags:**

```
      --type string   Filter by type (workflow, expansion, aspect, convoy)
```

## issuegraph formula show

Show detailed information about a formula.

Displays:
  - Formula metadata (name, type, description)
  - Variables with defaults and constraints
  - Steps with dependencies
  - Composition rules (extends, aspects, expansions)
  - Bond points for external composition

Examples:
  issuegraph formula show shiny
  issuegraph formula show rule-of-five
  issuegraph formula show security-audit --json

```
issuegraph formula show <formula-name> [flags]
```
