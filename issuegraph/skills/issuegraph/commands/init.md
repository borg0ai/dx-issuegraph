---
description: Initialize issuegraph in the current project
argument-hint: "[prefix]"
---

Initialize issuegraph issue tracking in the current directory.

If a prefix is provided as $1, use it as the issue prefix (e.g., "myproject" creates issues like myproject-1, myproject-2). If not provided, the default is the current directory name.

Use the issuegraph MCP `init` tool with the prefix parameter (if provided) to set up a new issuegraph database.

After initialization:
1. Show the database location
2. Show the issue prefix that will be used
3. Explain the basic workflow (or suggest running `/issuegraph:workflow`)
4. Suggest creating the first issue with `/issuegraph:create`

If issuegraph is already initialized, inform the user and show project stats using the `stats` tool.
