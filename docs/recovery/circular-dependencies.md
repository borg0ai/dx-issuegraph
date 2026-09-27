---
title: Circular Dependencies
description: Detect and break dependency cycles
---

This runbook helps you detect and break circular dependency cycles in your issues.

## Symptoms

- "circular dependency detected" errors
- `issuegraph blocked` shows unexpected results
- Issues that should be ready appear blocked

## Diagnosis

```bash
# Check for blocked issues
issuegraph blocked

# View dependencies for a specific issue
issuegraph show <issue-id>

# List all dependencies
issuegraph dep tree
```

## Solution

**Step 1:** Identify the cycle
```bash
issuegraph blocked --verbose
```

**Step 2:** Map the dependency chain
```bash
issuegraph show <issue-a>
issuegraph show <issue-b>
# Follow the chain until you return to <issue-a>
```

**Step 3:** Determine which dependency to remove
Consider: Which dependency is least critical to the workflow?

**Step 4:** Remove the problematic dependency
```bash
issuegraph dep remove <dependent-issue> <blocking-issue>
```

**Step 5:** Verify the cycle is broken
```bash
issuegraph blocked
issuegraph ready
```

## Prevention

- Think "X needs Y" not "X before Y" when adding dependencies
- Use `issuegraph blocked` after adding dependencies to check for cycles
- Keep dependency chains shallow when possible
