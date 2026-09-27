---
description: Epic management commands
argument-hint: "[command]"
---

Manage epics (large features composed of multiple issues).

## Available Commands

- **status**: Show epic completion status
  - Shows progress for each epic
  - Lists child issues and their states
  - Calculates completion percentage

- **close-eligible**: Close epics where all children are complete
  - Automatically closes epics when all child issues are done
  - Useful for bulk epic cleanup

## Epic Workflow

1. Create epic: `issuegraph create "Large Feature" -t epic -p 1`
2. Link subtasks: `issuegraph dep add ig-20 ig-10 --type parent-child` (task ig-20 is child of epic ig-10)
   - Or at creation: `issuegraph create "Subtask title" -t task --parent ig-10`
   - Children inherit parent labels by default. If the epic uses size/effort
     labels and children should have their own, add `--no-inherit-labels`
     (GH#4562).
3. Track progress: `issuegraph epic status`
4. Auto-close when done: `issuegraph epic close-eligible`

Epics use parent-child dependencies to track subtasks.
