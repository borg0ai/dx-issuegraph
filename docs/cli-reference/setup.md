---
title: "issuegraph setup"
description: "Setup integration with AI editors"
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc setup`.

Setup integration files for AI editors and coding assistants.

Recipes define where issuegraph workflow instructions are written. Built-in recipes
include cursor, claude, copilot, gemini, aider, factory, codex, mux, opencode, junie, kiro, windsurf, cody, and kilocode.

Examples:
  issuegraph setup cursor          # Install Cursor IDE integration (rules + agent hooks)
  issuegraph setup cursor --global # Install global Cursor hooks (~/.cursor/hooks.json)
  issuegraph setup kiro            # Install Kiro steering guidance
  issuegraph setup codex           # Install Codex skill + AGENTS.md guidance + native hooks
  issuegraph setup codex --global  # Install global Codex skill + guidance + native hooks
  issuegraph setup copilot         # Install Copilot CLI plugin + repository instructions
  issuegraph setup mux --project   # Install Mux workspace layer (.mux/AGENTS.md)
  issuegraph setup mux --global    # Install Mux global layer (~/.mux/AGENTS.md)
  issuegraph setup mux --project --global  # Install both Mux layers
  issuegraph setup --list          # Show all available recipes
  issuegraph setup --print         # Print the template to stdout
  issuegraph setup -o rules.md     # Write template to custom path
  issuegraph setup --add myeditor .myeditor/rules.md  # Add custom recipe

Use 'issuegraph setup &lt;recipe&gt; --check' to verify installation status.
Use 'issuegraph setup &lt;recipe&gt; --remove' to uninstall.

```
issuegraph setup [recipe] [flags]
```

**Flags:**

```
      --add string      Add a custom recipe with given name
      --check           Check if integration is installed
      --global          Install globally (claude/codex/cursor/mux; writes to ~/.claude/settings.json, $CODEX_HOME/AGENTS.md or ~/.codex/AGENTS.md, ~/.cursor/hooks.json, or ~/.mux/AGENTS.md)
      --list            List all available recipes
  -o, --output string   Write template to custom path
      --print           Print the template to stdout
      --project         Install for this project only (gemini/mux)
      --remove          Remove the integration
      --stealth         Use stealth mode (claude/gemini)
```
