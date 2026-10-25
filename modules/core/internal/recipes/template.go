package recipes

import "fmt"

// Template is the universal issuegraph workflow template.
// This content is written to all file-based recipes.
const Template = `# IssueGraph Issue Tracking

This project uses [IssueGraph (issuegraph)](https://github.com/borg0ai/dx-issuegraph) for issue tracking.

## Core Rules

- Track ALL work in issuegraph (never use markdown TODOs or comment-based task lists)
- Use ` + "`issuegraph ready`" + ` to find available work
- Use ` + "`issuegraph create`" + ` to track new issues/tasks/bugs
- Treat commit, push, and Dolt remote sync as policy-controlled handoff actions
- Run ` + "`issuegraph prime`" + ` for complete workflow context (SSOT for operational commands)
- Default to conservative git authority: report status and proposed commands unless the user, orchestrator, or repository profile explicitly authorizes commit/sync/push

## Quick Reference

` + "```bash" + `
issuegraph prime                              # Load complete workflow context (SSOT)
issuegraph ready                              # Show issues ready to work (no blockers)
issuegraph list --status=open                 # List all open issues
issuegraph create "title" -t task -p 2        # Create new issue
issuegraph update <id> --claim                # Claim work atomically
issuegraph unclaim <id>                       # Release stuck issue (agent crashed)
issuegraph close <id>                         # Mark complete
issuegraph dep add <issue> <depends-on>       # Add dependency
issuegraph dolt push                          # Sync with remote when authorized
` + "```" + `

## Workflow

1. Check for ready work: ` + "`issuegraph ready`" + `
2. Claim an issue atomically: ` + "`issuegraph update <id> --claim`" + `
3. Do the work
4. Mark complete: ` + "`issuegraph close <id>`" + `
5. Handoff: report changed files, validation, issue status, and any proposed commit/sync/push commands

## Issue Types

- ` + "`bug`" + ` - Something broken
- ` + "`feature`" + ` - New functionality
- ` + "`task`" + ` - Work item (tests, docs, refactoring)
- ` + "`epic`" + ` - Large feature with subtasks
- ` + "`chore`" + ` - Maintenance (dependencies, tooling)

## Priorities

- ` + "`0`" + ` - Critical (security, data loss, broken builds)
- ` + "`1`" + ` - High (major features, important bugs)
- ` + "`2`" + ` - Medium (default, nice-to-have)
- ` + "`3`" + ` - Low (polish, optimization)
- ` + "`4`" + ` - Backlog (future ideas)

## Context Loading

Run ` + "`issuegraph prime`" + ` to get complete workflow documentation in AI-optimized format.
` + "`issuegraph prime`" + ` is the single source of truth for operational commands and session workflow.

For detailed docs: see AGENTS.md, the issuegraph quickstart
(https://github.com/borg0ai/dx-issuegraph/blob/main/docs/getting-started/quickstart.md), or run ` + "`issuegraph --help`" + `
`

// CopilotInstructionsTemplate is the repository instructions file used by the
// lightweight Copilot CLI plugin recipe.
const CopilotInstructionsTemplate = `# GitHub Copilot Instructions

This repository uses **IssueGraph (issuegraph)** for issue tracking.

## Core Workflow

- Use ` + "`issuegraph ready`" + ` to find unblocked work
- Use ` + "`issuegraph create`" + ` to track new work
- Use ` + "`issuegraph update <id> --claim`" + ` before starting
- Use ` + "`issuegraph close <id>`" + ` when work is complete
- Treat commit, push, and Dolt remote sync as policy-controlled handoff actions
- Do not commit, push, or run Dolt remote sync unless explicitly authorized

## Context Loading

Run ` + "`issuegraph prime`" + ` for the full workflow context.

If the IssueGraph Copilot plugin is installed, Copilot CLI will automatically run
` + "`issuegraph prime`" + ` on session start and before compaction.
`

// ContentForPath returns the file content that should be written for a recipe path.
func ContentForPath(recipe Recipe, path string) (string, error) {
	switch recipe.Type {
	case TypeFile:
		if recipe.Content != "" {
			return recipe.Content, nil
		}
		return Template, nil
	case TypeMultiFile:
		if len(recipe.Contents) == 0 {
			return "", fmt.Errorf("recipe %q has no file contents", recipe.Name)
		}
		content, ok := recipe.Contents[path]
		if !ok {
			return "", fmt.Errorf("recipe %q has no content for %s", recipe.Name, path)
		}
		return content, nil
	default:
		return "", fmt.Errorf("recipe %q has unsupported type %q", recipe.Name, recipe.Type)
	}
}
