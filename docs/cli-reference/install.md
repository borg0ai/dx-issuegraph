---
title: "issuegraph install"
description: "Install IssueGraph skills and integrations into AI agents and editors"
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc install`.

Install the IssueGraph agent skill and editor integrations.

With no recipe, installs the complete IssueGraph skill into the global skills
directory of every detected coding agent (claude, codex, gemini, hermes,
antigravity, pi). Safe to re-run; nothing is written inside the current repository.
Naming one of those agents installs the skill for that agent only (claude, codex
and gemini also run their integration recipe).

Recipes define where issuegraph workflow instructions are written. Built-in recipes
include cursor, claude, copilot, gemini, aider, factory, codex, mux, opencode, junie, kiro, windsurf, cody, and kilocode.

Examples:
  issuegraph install                 # Install the skill for every detected agent
  issuegraph install --dry-run       # Show what would change
  issuegraph install --check         # Verify the skill is installed and current
  issuegraph install pi              # Install the skill for one agent
  issuegraph install cursor          # Install Cursor IDE integration (rules + agent hooks)
  issuegraph install cursor --global # Install global Cursor hooks (~/.cursor/hooks.json)
  issuegraph install kiro            # Install Kiro steering guidance
  issuegraph install codex           # Install Codex skill + AGENTS.md guidance + native hooks
  issuegraph install codex --global  # Install global Codex skill + guidance + native hooks
  issuegraph install copilot         # Install Copilot CLI plugin + repository instructions
  issuegraph install mux --project   # Install Mux workspace layer (.mux/AGENTS.md)
  issuegraph install mux --global    # Install Mux global layer (~/.mux/AGENTS.md)
  issuegraph install mux --project --global  # Install both Mux layers
  issuegraph install --list          # Show all available recipes
  issuegraph install --print         # Print the template to stdout
  issuegraph install -o rules.md     # Write template to custom path
  issuegraph install --add myeditor .myeditor/rules.md  # Add custom recipe

Use 'issuegraph install &lt;recipe&gt; --check' to verify installation status.
Use 'issuegraph install &lt;recipe&gt; --remove' to uninstall.

```
issuegraph install [recipe] [flags]
```

**Flags:**

```
      --add string      Add a custom recipe with given name
      --check           Check if integration is installed
      --dry-run         Show what would change without writing files
      --global          Install globally (claude/codex/cursor/mux; writes to ~/.claude/settings.json, $CODEX_HOME/AGENTS.md or ~/.codex/AGENTS.md, ~/.cursor/hooks.json, or ~/.mux/AGENTS.md)
      --list            List all available recipes
  -o, --output string   Write template to custom path
      --print           Print the template to stdout
      --project         Install for this project only (gemini/mux)
      --remove          Remove the integration
      --stealth         Use stealth mode (claude/gemini)
```
