# issuegraph — Complete Command Reference

Reference for issuegraph Latest. Generated from `issuegraph help --all`.

## Table of Contents

### Working With Issues:

- [issuegraph assign](#issuegraph-assign) — Assign an issue to someone
- [issuegraph children](#issuegraph-children) — List child issuegraph of a parent
- [issuegraph close](#issuegraph-close) — Close one or more issues
- [issuegraph comment](#issuegraph-comment) — Add a comment to an issue
- [issuegraph comments](#issuegraph-comments) — View or manage comments on an issue
  - [issuegraph comments add](#issuegraph-comments-add) — Add a comment to an issue
  - [issuegraph comments list](#issuegraph-comments-list) — Invalid — use issuegraph comments &lt;issue-id&gt; to list comments
- [issuegraph create](#issuegraph-create) — Create a new issue (or batch from markdown/graph JSON)
- [issuegraph create-form](#issuegraph-create-form) — Create a new issue using an interactive form
- [issuegraph delete](#issuegraph-delete) — Delete one or more issues and clean up references
- [issuegraph edit](#issuegraph-edit) — Edit an issue field in $EDITOR
- [issuegraph gate](#issuegraph-gate) — Manage async coordination gates
  - [issuegraph gate add-waiter](#issuegraph-gate-add-waiter) — Add a waiter to a gate
  - [issuegraph gate check](#issuegraph-gate-check) — Evaluate gates and close resolved ones
  - [issuegraph gate create](#issuegraph-gate-create) — Create a gate that blocks an issue
  - [issuegraph gate discover](#issuegraph-gate-discover) — Discover await_id for gh:run gates
  - [issuegraph gate list](#issuegraph-gate-list) — List gate issues
  - [issuegraph gate resolve](#issuegraph-gate-resolve) — Manually resolve (close) a gate
  - [issuegraph gate show](#issuegraph-gate-show) — Show a gate issue
- [issuegraph label](#issuegraph-label) — Manage issue labels
  - [issuegraph label add](#issuegraph-label-add) — Add a label to one or more issues
  - [issuegraph label list](#issuegraph-label-list) — List labels for an issue
  - [issuegraph label list-all](#issuegraph-label-list-all) — List all unique labels in the database
  - [issuegraph label propagate](#issuegraph-label-propagate) — Propagate a label from a parent issue to all its children
  - [issuegraph label remove](#issuegraph-label-remove) — Remove a label from one or more issues
- [issuegraph link](#issuegraph-link) — Link two issues with a dependency
- [issuegraph list](#issuegraph-list) — List issues
- [issuegraph merge-slot](#issuegraph-merge-slot) — Manage merge-slot gates for serialized conflict resolution
  - [issuegraph merge-slot acquire](#issuegraph-merge-slot-acquire) — Acquire the merge slot
  - [issuegraph merge-slot check](#issuegraph-merge-slot-check) — Check merge slot availability
  - [issuegraph merge-slot create](#issuegraph-merge-slot-create) — Create a merge slot issue for the current rig
  - [issuegraph merge-slot release](#issuegraph-merge-slot-release) — Release the merge slot
- [issuegraph note](#issuegraph-note) — Append a note to an issue
- [issuegraph priority](#issuegraph-priority) — Set the priority of an issue
- [issuegraph promote](#issuegraph-promote) — Promote a wisp to a permanent issue
- [issuegraph q](#issuegraph-q) — Quick capture: create issue and output only ID
- [issuegraph query](#issuegraph-query) — Query issues using a simple query language
- [issuegraph reopen](#issuegraph-reopen) — Reopen one or more closed issues
- [issuegraph search](#issuegraph-search) — Search issues by text query
- [issuegraph set-state](#issuegraph-set-state) — Set operational state (creates event + updates label)
- [issuegraph show](#issuegraph-show) — Show issue details
- [issuegraph state](#issuegraph-state) — Query the current value of a state dimension
  - [issuegraph state list](#issuegraph-state-list) — List all state dimensions on an issue
- [issuegraph tag](#issuegraph-tag) — Add a label to an issue
- [issuegraph todo](#issuegraph-todo) — Manage TODO items (convenience wrapper for task issues)
  - [issuegraph todo add](#issuegraph-todo-add) — Add a new TODO item
  - [issuegraph todo done](#issuegraph-todo-done) — Mark TODO(s) as done
  - [issuegraph todo list](#issuegraph-todo-list) — List TODO items
- [issuegraph update](#issuegraph-update) — Update one or more issues

### Views & Reports:

- [issuegraph count](#issuegraph-count) — Count issues matching filters
- [issuegraph diff](#issuegraph-diff) — Show changes between two commits or branches
- [issuegraph find-duplicates](#issuegraph-find-duplicates) — Find semantically similar issues using text analysis or AI
- [issuegraph history](#issuegraph-history) — Show version history for an issue
- [issuegraph lint](#issuegraph-lint) — Check issues for missing template sections
- [issuegraph stale](#issuegraph-stale) — Show stale issues (not updated recently)
- [issuegraph status](#issuegraph-status) — Show issue database overview and statistics
- [issuegraph statuses](#issuegraph-statuses) — List valid issue statuses
- [issuegraph types](#issuegraph-types) — List valid issue types

### Dependencies & Structure:

- [issuegraph dep](#issuegraph-dep) — Manage dependencies
  - [issuegraph dep add](#issuegraph-dep-add) — Add a dependency
  - [issuegraph dep cycles](#issuegraph-dep-cycles) — Detect dependency cycles
  - [issuegraph dep list](#issuegraph-dep-list) — List dependencies or dependents of one or more issues
  - [issuegraph dep relate](#issuegraph-dep-relate) — Create a bidirectional relates_to link between issues
  - [issuegraph dep remove](#issuegraph-dep-remove) — Remove a dependency
  - [issuegraph dep tree](#issuegraph-dep-tree) — Show dependency tree
  - [issuegraph dep unrelate](#issuegraph-dep-unrelate) — Remove a relates_to link between issues
- [issuegraph duplicate](#issuegraph-duplicate) — Mark an issue as a duplicate of another
- [issuegraph duplicates](#issuegraph-duplicates) — Find and optionally merge duplicate issues
- [issuegraph epic](#issuegraph-epic) — Epic management commands
  - [issuegraph epic close-eligible](#issuegraph-epic-close-eligible) — Close epics where all children are complete
  - [issuegraph epic status](#issuegraph-epic-status) — Show epic completion status
- [issuegraph graph](#issuegraph-graph) — Display issue dependency graph
  - [issuegraph graph check](#issuegraph-graph-check) — Check dependency graph integrity
- [issuegraph supersede](#issuegraph-supersede) — Mark an issue as superseded by a newer one
- [issuegraph swarm](#issuegraph-swarm) — Swarm management for structured epics
  - [issuegraph swarm create](#issuegraph-swarm-create) — Create a swarm molecule from an epic
  - [issuegraph swarm list](#issuegraph-swarm-list) — List all swarm molecules
  - [issuegraph swarm status](#issuegraph-swarm-status) — Show current swarm status
  - [issuegraph swarm validate](#issuegraph-swarm-validate) — Validate epic structure for swarming

### Sync & Data:

- [issuegraph backup](#issuegraph-backup) — Back up your issuegraph database
  - [issuegraph backup init](#issuegraph-backup-init) — Set up a Dolt backup destination
  - [issuegraph backup remove](#issuegraph-backup-remove) — Remove the configured backup destination
  - [issuegraph backup restore](#issuegraph-backup-restore) — Restore database from a Dolt backup
  - [issuegraph backup status](#issuegraph-backup-status) — Show last backup status
  - [issuegraph backup sync](#issuegraph-backup-sync) — Push database to configured Dolt backup
- [issuegraph branch](#issuegraph-branch) — List or create branches
- [issuegraph export](#issuegraph-export) — Export issues to JSONL format
- [issuegraph federation](#issuegraph-federation) — Manage peer-to-peer federation (requires CGO)
- [issuegraph import](#issuegraph-import) — Import issues from a JSONL file or stdin into the database
- [issuegraph restore](#issuegraph-restore) — Restore the pre-compaction content of a compacted issue
- [issuegraph vc](#issuegraph-vc) — Version control operations
  - [issuegraph vc commit](#issuegraph-vc-commit) — Create a commit with all staged changes
  - [issuegraph vc merge](#issuegraph-vc-merge) — Merge a branch into the current branch
  - [issuegraph vc status](#issuegraph-vc-status) — Show current branch and uncommitted changes

### Setup & Configuration:

- [issuegraph bootstrap](#issuegraph-bootstrap) — Non-destructive database setup for fresh clones and recovery
- [issuegraph config](#issuegraph-config) — Manage configuration settings
  - [issuegraph config apply](#issuegraph-config-apply) — Reconcile system state to match configuration
  - [issuegraph config drift](#issuegraph-config-drift) — Detect config-vs-reality inconsistencies
  - [issuegraph config get](#issuegraph-config-get) — Get a configuration value
  - [issuegraph config list](#issuegraph-config-list) — List all configuration
  - [issuegraph config set](#issuegraph-config-set) — Set a configuration value
  - [issuegraph config set-many](#issuegraph-config-set-many) — Set multiple configuration values in one operation
  - [issuegraph config show](#issuegraph-config-show) — Show all effective configuration with provenance
  - [issuegraph config unset](#issuegraph-config-unset) — Delete a configuration value
  - [issuegraph config validate](#issuegraph-config-validate) — Validate sync-related configuration
- [issuegraph context](#issuegraph-context) — Show effective backend identity and repository context
- [issuegraph dolt](#issuegraph-dolt) — Configure Dolt database settings
  - [issuegraph dolt clean-databases](#issuegraph-dolt-clean-databases) — Drop stale test databases from the Dolt server
  - [issuegraph dolt commit](#issuegraph-dolt-commit) — Create a Dolt commit from pending changes
  - [issuegraph dolt killall](#issuegraph-dolt-killall) — Kill all orphan Dolt server processes
  - [issuegraph dolt pull](#issuegraph-dolt-pull) — Pull commits from Dolt remote
  - [issuegraph dolt push](#issuegraph-dolt-push) — Push commits to Dolt remote
  - [issuegraph dolt remote](#issuegraph-dolt-remote) — Manage Dolt remotes
  - [issuegraph dolt set](#issuegraph-dolt-set) — Set a Dolt configuration value
  - [issuegraph dolt show](#issuegraph-dolt-show) — Show current Dolt configuration with connection status
  - [issuegraph dolt start](#issuegraph-dolt-start) — Start the Dolt SQL server for this project
  - [issuegraph dolt status](#issuegraph-dolt-status) — Show Dolt engine status
  - [issuegraph dolt stop](#issuegraph-dolt-stop) — Stop the Dolt SQL server for this project
  - [issuegraph dolt test](#issuegraph-dolt-test) — Test connection to Dolt server
- [issuegraph forget](#issuegraph-forget) — Remove a persistent memory
- [issuegraph hooks](#issuegraph-hooks) — Manage git hooks for issuegraph integration
  - [issuegraph hooks install](#issuegraph-hooks-install) — Install issuegraph git hooks
  - [issuegraph hooks list](#issuegraph-hooks-list) — List installed git hooks status
  - [issuegraph hooks run](#issuegraph-hooks-run) — Execute a git hook (called by thin shims)
  - [issuegraph hooks uninstall](#issuegraph-hooks-uninstall) — Uninstall issuegraph git hooks
- [issuegraph human](#issuegraph-human) — Show essential commands for human users
  - [issuegraph human dismiss](#issuegraph-human-dismiss) — Dismiss a human-needed issue
  - [issuegraph human list](#issuegraph-human-list) — List all human-needed issuegraph
  - [issuegraph human respond](#issuegraph-human-respond) — Respond to a human-needed issue
  - [issuegraph human stats](#issuegraph-human-stats) — Show summary statistics for human-needed issuegraph
- [issuegraph info](#issuegraph-info) — Show database information
- [issuegraph init](#issuegraph-init) — Initialize issuegraph in the current directory
- [issuegraph kv](#issuegraph-kv) — Key-value store commands
  - [issuegraph kv clear](#issuegraph-kv-clear) — Delete a key-value pair
  - [issuegraph kv get](#issuegraph-kv-get) — Get a value by key
  - [issuegraph kv list](#issuegraph-kv-list) — List all key-value pairs
  - [issuegraph kv set](#issuegraph-kv-set) — Set a key-value pair
- [issuegraph memories](#issuegraph-memories) — List or search persistent memories
- [issuegraph onboard](#issuegraph-onboard) — Display minimal snippet for agent instructions file
- [issuegraph prime](#issuegraph-prime) — Output AI-optimized workflow context
- [issuegraph quickstart](#issuegraph-quickstart) — Quick start guide for issuegraph
- [issuegraph recall](#issuegraph-recall) — Retrieve a specific memory
- [issuegraph remember](#issuegraph-remember) — Store a persistent memory
- [issuegraph setup](#issuegraph-setup) — Setup integration with AI editors
- [issuegraph where](#issuegraph-where) — Show active issuegraph location

### Maintenance:

- [issuegraph batch](#issuegraph-batch) — Run multiple write operations in a single database transaction
- [issuegraph compact](#issuegraph-compact) — Squash old Dolt commits to reduce history size
- [issuegraph doctor](#issuegraph-doctor) — Check and fix issuegraph installation health (start here)
- [issuegraph flatten](#issuegraph-flatten) — Squash all Dolt history into a single commit
- [issuegraph gc](#issuegraph-gc) — Garbage collect: decay old issues, compact Dolt commits, run Dolt GC
- [issuegraph migrate](#issuegraph-migrate) — Database migration commands
  - [issuegraph migrate hooks](#issuegraph-migrate-hooks) — Plan or apply git hook migration to marker-managed format
  - [issuegraph migrate issues](#issuegraph-migrate-issues) — Move issues between repositories
  - [issuegraph migrate schema](#issuegraph-migrate-schema) — Apply pending schema migrations (idempotent)
  - [issuegraph migrate sync](#issuegraph-migrate-sync) — Set up sync.branch workflow for multi-clone setups
- [issuegraph ping](#issuegraph-ping) — Check database connectivity
- [issuegraph preflight](#issuegraph-preflight) — Show PR readiness checklist
- [issuegraph prune](#issuegraph-prune) — Delete old closed issuegraph to reclaim space and shrink exports
- [issuegraph purge](#issuegraph-purge) — Delete closed ephemeral issuegraph to reclaim space
- [issuegraph recompute-blocked](#issuegraph-recompute-blocked) — Recompute is_blocked for all issues (repairs stale flags after a pull)
- [issuegraph rename-prefix](#issuegraph-rename-prefix) — Rename the issue prefix for all issues in the database
- [issuegraph rules](#issuegraph-rules) — Audit and compact Claude rules
  - [issuegraph rules audit](#issuegraph-rules-audit) — Scan rules for contradictions and merge opportunities
  - [issuegraph rules compact](#issuegraph-rules-compact) — Merge related rules into composites
- [issuegraph sql](#issuegraph-sql) — Execute raw SQL against the issuegraph database
- [issuegraph upgrade](#issuegraph-upgrade) — Check and manage issuegraph version upgrades
  - [issuegraph upgrade ack](#issuegraph-upgrade-ack) — Acknowledge the current issuegraph version
  - [issuegraph upgrade review](#issuegraph-upgrade-review) — Review changes since last issuegraph version
  - [issuegraph upgrade status](#issuegraph-upgrade-status) — Check if issuegraph version has changed
- [issuegraph worktree](#issuegraph-worktree) — Manage git worktrees for parallel development
  - [issuegraph worktree create](#issuegraph-worktree-create) — Create a worktree
  - [issuegraph worktree info](#issuegraph-worktree-info) — Show worktree info for current directory
  - [issuegraph worktree list](#issuegraph-worktree-list) — List all git worktrees
  - [issuegraph worktree remove](#issuegraph-worktree-remove) — Remove a worktree with safety checks

### Integrations & Advanced:

- [issuegraph admin](#issuegraph-admin) — Administrative commands for database maintenance
  - [issuegraph admin cleanup](#issuegraph-admin-cleanup) — Delete closed issues to reduce database size
  - [issuegraph admin compact](#issuegraph-admin-compact) — Compact old closed issues to save space
  - [issuegraph admin reset](#issuegraph-admin-reset) — Remove all issuegraph data and configuration
- [issuegraph jira](#issuegraph-jira) — Jira integration commands
  - [issuegraph jira pull](#issuegraph-jira-pull) — Pull specific items from Jira
  - [issuegraph jira push](#issuegraph-jira-push) — Push specific issuegraph to Jira
  - [issuegraph jira status](#issuegraph-jira-status) — Show Jira sync status
  - [issuegraph jira sync](#issuegraph-jira-sync) — Synchronize issues with Jira
- [issuegraph linear](#issuegraph-linear) — Linear integration commands
  - [issuegraph linear pull](#issuegraph-linear-pull) — Pull specific items from Linear
  - [issuegraph linear push](#issuegraph-linear-push) — Push specific issuegraph to Linear
  - [issuegraph linear status](#issuegraph-linear-status) — Show Linear sync status
  - [issuegraph linear sync](#issuegraph-linear-sync) — Synchronize issues with Linear
  - [issuegraph linear teams](#issuegraph-linear-teams) — List available Linear teams
- [issuegraph repo](#issuegraph-repo) — Manage multiple repository configuration
  - [issuegraph repo add](#issuegraph-repo-add) — Add an additional repository to sync
  - [issuegraph repo list](#issuegraph-repo-list) — List all configured repositories
  - [issuegraph repo remove](#issuegraph-repo-remove) — Remove a repository from sync configuration
  - [issuegraph repo sync](#issuegraph-repo-sync) — Manually trigger multi-repo sync

### Other Commands:

- [issuegraph ado](#issuegraph-ado) — Azure DevOps integration commands
  - [issuegraph ado projects](#issuegraph-ado-projects) — List accessible Azure DevOps projects
  - [issuegraph ado pull](#issuegraph-ado-pull) — Pull specific items from Azure DevOps
  - [issuegraph ado push](#issuegraph-ado-push) — Push specific issuegraph to Azure DevOps
  - [issuegraph ado status](#issuegraph-ado-status) — Show Azure DevOps sync status
  - [issuegraph ado sync](#issuegraph-ado-sync) — Sync issues with Azure DevOps
- [issuegraph audit](#issuegraph-audit) — Record and label agent interactions (append-only JSONL)
  - [issuegraph audit label](#issuegraph-audit-label) — Append a label entry referencing an existing interaction
  - [issuegraph audit record](#issuegraph-audit-record) — Append an audit interaction entry
- [issuegraph blocked](#issuegraph-blocked) — Show blocked issues
- [issuegraph completion](#issuegraph-completion) — Generate the autocompletion script for the specified shell
  - [issuegraph completion bash](#issuegraph-completion-bash) — Generate the autocompletion script for bash
  - [issuegraph completion fish](#issuegraph-completion-fish) — Generate the autocompletion script for fish
  - [issuegraph completion powershell](#issuegraph-completion-powershell) — Generate the autocompletion script for powershell
  - [issuegraph completion zsh](#issuegraph-completion-zsh) — Generate the autocompletion script for zsh
- [issuegraph cook](#issuegraph-cook) — Compile a formula into a proto (ephemeral by default)
- [issuegraph defer](#issuegraph-defer) — Defer one or more issues for later
- [issuegraph formula](#issuegraph-formula) — Manage workflow formulas
  - [issuegraph formula convert](#issuegraph-formula-convert) — Convert formula from JSON to TOML
  - [issuegraph formula list](#issuegraph-formula-list) — List available formulas
  - [issuegraph formula show](#issuegraph-formula-show) — Show formula details
- [issuegraph github](#issuegraph-github) — GitHub integration commands
  - [issuegraph github pull](#issuegraph-github-pull) — Pull specific items from GitHub
  - [issuegraph github push](#issuegraph-github-push) — Push specific issuegraph to GitHub
  - [issuegraph github repos](#issuegraph-github-repos) — List accessible GitHub repositories
  - [issuegraph github status](#issuegraph-github-status) — Show GitHub sync status
  - [issuegraph github sync](#issuegraph-github-sync) — Sync issues with GitHub
- [issuegraph gitlab](#issuegraph-gitlab) — GitLab integration commands
  - [issuegraph gitlab projects](#issuegraph-gitlab-projects) — List accessible GitLab projects
  - [issuegraph gitlab pull](#issuegraph-gitlab-pull) — Pull specific items from GitLab
  - [issuegraph gitlab push](#issuegraph-gitlab-push) — Push specific issuegraph to GitLab
  - [issuegraph gitlab status](#issuegraph-gitlab-status) — Show GitLab sync status
  - [issuegraph gitlab sync](#issuegraph-gitlab-sync) — Sync issues with GitLab
- [issuegraph help](#issuegraph-help) — Help about any command
- [issuegraph init-safety](#issuegraph-init-safety) — Explain issuegraph init flag semantics and the destroy-token format
- [issuegraph mail](#issuegraph-mail) — Delegate to mail provider (e.g., gt mail)
- [issuegraph metrics](#issuegraph-metrics) — Show or change anonymous usage-metrics settings
  - [issuegraph metrics example](#issuegraph-metrics-example) — Show real examples of the anonymous metrics issuegraph sends
  - [issuegraph metrics off](#issuegraph-metrics-off) — Turn anonymous usage metrics off
  - [issuegraph metrics on](#issuegraph-metrics-on) — Turn anonymous usage metrics on
- [issuegraph mol](#issuegraph-mol) — Molecule commands (work templates)
  - [issuegraph mol bond](#issuegraph-mol-bond) — Bond two protos or molecules together
  - [issuegraph mol burn](#issuegraph-mol-burn) — Delete a molecule without creating a digest
  - [issuegraph mol current](#issuegraph-mol-current) — Show current position in molecule workflow
  - [issuegraph mol distill](#issuegraph-mol-distill) — Extract a formula from an existing epic
  - [issuegraph mol last-activity](#issuegraph-mol-last-activity) — Show last activity timestamp for a molecule
  - [issuegraph mol pour](#issuegraph-mol-pour) — Instantiate a proto as a persistent mol (solid -&gt; liquid)
  - [issuegraph mol progress](#issuegraph-mol-progress) — Show molecule progress summary
  - [issuegraph mol ready](#issuegraph-mol-ready) — Find molecules ready for gate-resume dispatch
  - [issuegraph mol seed](#issuegraph-mol-seed) — Verify formula accessibility
  - [issuegraph mol show](#issuegraph-mol-show) — Show molecule details
  - [issuegraph mol squash](#issuegraph-mol-squash) — Compress molecule execution into a digest
  - [issuegraph mol stale](#issuegraph-mol-stale) — Detect complete-but-unclosed molecules
  - [issuegraph mol wisp](#issuegraph-mol-wisp) — Create or manage wisps (ephemeral molecules)
- [issuegraph notion](#issuegraph-notion) — Notion integration commands
  - [issuegraph notion connect](#issuegraph-notion-connect) — Connect issuegraph to an existing Notion database or data source
  - [issuegraph notion init](#issuegraph-notion-init) — Create a dedicated IssueGraph database in Notion
  - [issuegraph notion pull](#issuegraph-notion-pull) — Pull specific items from Notion
  - [issuegraph notion push](#issuegraph-notion-push) — Push specific issuegraph to Notion
  - [issuegraph notion status](#issuegraph-notion-status) — Show Notion sync status
  - [issuegraph notion sync](#issuegraph-notion-sync) — Sync issues with Notion
- [issuegraph orphans](#issuegraph-orphans) — Identify orphaned issues (referenced in commits but still open)
- [issuegraph ready](#issuegraph-ready) — Show ready work (open, no active blockers)
- [issuegraph rename](#issuegraph-rename) — Rename an issue ID
- [issuegraph ship](#issuegraph-ship) — Publish a capability for cross-project dependencies
- [issuegraph undefer](#issuegraph-undefer) — Undefer one or more issues (restore to open)
- [issuegraph version](#issuegraph-version) — Print version information

---

## Global Flags

These flags apply to all commands:

```
      --actor string              Actor name for audit trail (default: $ISSUEGRAPH_ACTOR, git user.name, $USER)
      --db string                 Database path (default: auto-discover .issuegraph/*.db)
  -C, --directory string          Change to this directory before running the command (like git -C)
      --dolt-auto-commit string   Dolt auto-commit policy (off|on|batch). 'on': commit after each write. 'batch': defer commits to issuegraph dolt commit; uncommitted changes persist in the working set until then. SIGTERM/SIGHUP flush pending batch commits. Default: off. Override via config key dolt.auto-commit
      --global                    Use the global shared-server database (issuegraph_global)
      --ignore-schema-skew        Proceed despite forward schema drift (some queries may fail)
      --json                      Output in JSON format
      --profile                   Generate CPU profile for performance analysis
  -q, --quiet                     Suppress non-essential output (errors only)
      --readonly                  Read-only mode: block write operations (for worker sandboxes)
      --sandbox                   Sandbox mode: disables Dolt auto-push
  -v, --verbose                   Enable verbose/debug output
```

---

## Working With Issues:

### issuegraph assign

Assign an issue to someone.

Shorthand for 'issuegraph update &lt;id&gt; --assignee &lt;name&gt;'.

Examples:
  issuegraph assign issuegraph-123 alice
  issuegraph assign issuegraph-123 ""      # unassign

```
issuegraph assign <id> <name>
```

### issuegraph children

List all issuegraph that are children of the specified parent issue.

This is a convenience alias for 'issuegraph list --parent &lt;id&gt; --status all'.
Unlike plain 'issuegraph list', children includes closed issues by default,
since the primary use case is inspecting all work under a parent.

Examples:
  issuegraph children hq-abc123        # List all children of hq-abc123
  issuegraph children hq-abc123 --json # List children in JSON format
  issuegraph children hq-abc123 --pretty # Show children in tree format

```
issuegraph children <parent-id> [flags]
```

**Flags:**

```
      --pretty   Show children in tree format
```

### issuegraph close

Close one or more issues.

If no issue ID is provided, closes the last touched issue (from most recent
create, update, show, or close operation).

When closing multiple issues, provide one --reason for all IDs or repeat
--reason once per ID. Reasons map positionally: the first --reason applies
to the first ID, the second --reason to the second ID, regardless of where
the flags appear in the command line.

```
issuegraph close [id...] [flags]
```

**Aliases:** done

**Flags:**

```
      --claim-next           Automatically claim the next highest priority available issue
      --continue             Auto-advance to next step in molecule
  -f, --force                Force close pinned issues or unsatisfied gates
      --no-auto              With --continue, show next step but don't claim it
  -r, --reason string        Reason for closing
      --reason-file string   Read close reason from file (use - for stdin)
      --session string       Claude Code session ID (or set CLAUDE_SESSION_ID env var)
      --suggest-next         Show newly unblocked issues after closing
```

### issuegraph comment

Add a comment to an issue.

Shorthand for 'issuegraph comments add &lt;id&gt; "text"'.

Examples:
  issuegraph comment issuegraph-123 "Working on this now"
  issuegraph comment issuegraph-123 Working on this now
  echo "comment from pipe" | issuegraph comment issuegraph-123 --stdin
  issuegraph comment issuegraph-123 --file notes.txt

```
issuegraph comment <id> [text...] [flags]
```

**Flags:**

```
      --file string   Read comment text from file
      --stdin         Read comment text from stdin
```

### issuegraph comments

View or manage comments on an issue.

Examples:
  # List all comments on an issue (issue id is required — there is no "comments list")
  issuegraph comments issuegraph-123

  # List comments in JSON format
  issuegraph comments issuegraph-123 --json

  # Add a comment
  issuegraph comments add issuegraph-123 "This is a comment"

  # Add a comment from a file
  issuegraph comments add issuegraph-123 -f notes.txt

```
issuegraph comments [issue-id] [flags]
```

**Flags:**

```
      --local-time   Show timestamps in local time instead of UTC
```

#### issuegraph comments add

Add a comment to an issue.

Examples:
  # Add a comment
  issuegraph comments add issuegraph-123 "Working on this now"

  # Add a comment from a file
  issuegraph comments add issuegraph-123 -f notes.txt

```
issuegraph comments add [issue-id] [text] [flags]
```

**Flags:**

```
  -a, --author string   Add author to comment
  -f, --file string     Read comment text from file
```

#### issuegraph comments list

Invalid — use issuegraph comments &lt;issue-id&gt; to list comments

```
issuegraph comments list
```

### issuegraph create

Create a new issue (or batch from markdown/graph JSON)

```
issuegraph create [title] [flags]
```

**Aliases:** new

**Flags:**

```
      --acceptance string       Acceptance criteria
      --append-notes string     Append to existing notes (with newline separator)
  -a, --assignee string         Assignee
      --body-file string        Read description from file (use - for stdin)
      --context string          Additional context for the issue
      --defer string            Defer until date (issue hidden from issuegraph ready until then). Same formats as --due
      --deps strings            Dependencies in format 'type:id' or 'id' (e.g., 'discovered-from:issuegraph-20,blocks:issuegraph-15' or 'issuegraph-20')
  -d, --description string      Issue description
      --design string           Design notes
      --design-file string      Read design from file (use - for stdin)
      --dry-run                 Preview what would be created without actually creating
      --due string              Due date/time. Formats: +6h, +1d, +2w, tomorrow, next monday, 2025-01-15
      --ephemeral               Create as ephemeral (short-lived, subject to TTL compaction)
  -e, --estimate int            Time estimate in minutes (e.g., 60 for 1 hour)
      --event-actor string      Entity URI who caused this event (requires --type=event)
      --event-category string   Event category (e.g., patrol.muted, agent.started) (requires --type=event)
      --event-payload string    Event-specific JSON data (requires --type=event)
      --event-target string     Entity URI or issue ID affected (requires --type=event)
      --external-ref string     External reference (e.g., 'gh-9', 'jira-ABC', Linear URL)
  -f, --file string             Create multiple issues from markdown file
      --force                   Force creation even if prefix doesn't match database prefix
      --graph string            Create a graph of issues with dependencies from JSON plan file
      --id string               Explicit issue ID (e.g., 'issuegraph-42' for partitioning)
  -l, --labels strings          Labels (comma-separated)
      --metadata string         Set custom metadata (JSON string or @file.json to read from file)
      --mol-type string         Molecule type: swarm (multi-agent), patrol (recurring ops), work (default)
      --no-history              Skip Dolt commit history without making GC-eligible (for permanent agent issuegraph)
      --no-inherit-labels       Don't inherit labels from parent issue
      --notes string            Additional notes
      --parent string           Parent issue ID for hierarchical child (e.g., 'issuegraph-a3f8e9')
  -p, --priority string         Priority (0-4 or P0-P4, 0=highest) (default "2")
      --repo string             Target repository for issue (overrides auto-routing)
      --silent                  Output only the issue ID (for scripting)
      --skills string           Required skills for this issue
      --spec-id string          Link to specification document
      --stdin                   Read description from stdin (alias for --body-file -)
      --title string            Issue title (alternative to positional argument)
  -t, --type string             Issue type (bug|feature|task|epic|chore|decision); custom types require types.custom config; aliases: enhancement/feat→feature, dec/adr→decision (default "task")
      --validate                Validate description contains required sections for issue type
      --waits-for string        Spawner issue ID to wait for (creates waits-for dependency for fanout gate)
      --waits-for-gate string   Gate type: all-children (wait for all) or any-children (wait for first) (default "all-children")
      --wisp-type string        Wisp type for TTL-based compaction: heartbeat, ping, patrol, gc_report, recovery, error, escalation
```

### issuegraph create-form

Create a new issue using an interactive terminal form.

This command provides a user-friendly form interface for creating issues,
with fields for title, description, type, priority, labels, and more.

Use --parent to create a sub-issue under an existing parent issue.
The child will get an auto-generated hierarchical ID (e.g., parent-id.1).

The form uses keyboard navigation:
  - Tab/Shift+Tab: Move between fields
  - Enter: Submit the form (on the last field or submit button)
  - Ctrl+C: Cancel and exit
  - Arrow keys: Navigate within select fields

```
issuegraph create-form [flags]
```

**Flags:**

```
      --parent string   Parent issue ID for creating a hierarchical child (e.g., 'issuegraph-a3f8e9')
```

### issuegraph delete

Delete one or more issues and clean up all references to them.
This command will:
1. Remove all dependency links (any type, both directions) involving the issues
2. Update text references to "[deleted:ID]" in directly connected issues
3. Permanently delete the issues from the database

This is a destructive operation that cannot be undone. Use with caution.

BATCH DELETION:
Delete multiple issues at once:
  issuegraph delete issuegraph-1 issuegraph-2 issuegraph-3 --force

Delete from file (one ID per line):
  issuegraph delete --from-file deletions.txt --force

Preview before deleting:
  issuegraph delete --from-file deletions.txt --dry-run

DEPENDENCY HANDLING:
Default: Fails if any issue has dependents not in deletion set
  issuegraph delete issuegraph-1 issuegraph-2

Cascade: Recursively delete all dependents
  issuegraph delete issuegraph-1 --cascade --force

Force: Delete and orphan dependents
  issuegraph delete issuegraph-1 --force

```
issuegraph delete <issue-id> [issue-id...] [flags]
```

**Flags:**

```
      --cascade            Recursively delete all dependent issues
      --dry-run            Preview what would be deleted without making changes
  -f, --force              Actually delete (without this flag, shows preview)
      --from-file string   Read issue IDs from file (one per line)
```

### issuegraph edit

Edit an issue field using your configured $EDITOR.

By default, edits the description. Use flags to edit other fields.

Examples:
  issuegraph edit issuegraph-42                    # Edit description
  issuegraph edit issuegraph-42 --title            # Edit title
  issuegraph edit issuegraph-42 --design           # Edit design notes
  issuegraph edit issuegraph-42 --notes            # Edit notes
  issuegraph edit issuegraph-42 --acceptance       # Edit acceptance criteria

```
issuegraph edit [id] [flags]
```

**Flags:**

```
      --acceptance    Edit the acceptance criteria
      --description   Edit the description (default)
      --design        Edit the design notes
      --notes         Edit the notes
      --title         Edit the title
```

### issuegraph gate

Gates are async wait conditions that block workflow steps.

Gates are created automatically when a formula step has a gate field.
They must be closed (manually or via watchers) for the blocked step to proceed.

Gate types:
  human   - Requires manual issuegraph close (Phase 1)
  timer   - Expires after timeout (Phase 2)
  gh:run  - Waits for GitHub workflow (Phase 3)
  gh:pr   - Waits for PR merge (Phase 3)
  issue    - Waits for cross-rig issue to close (Phase 4)

For issue gates, await_id format is &lt;rig&gt;:&lt;issue-id&gt; (e.g., "other-project:op-abc123").

Examples:
  issuegraph gate list           # Show all open gates
  issuegraph gate list --all     # Show all gates including closed
  issuegraph gate check          # Evaluate all open gates
  issuegraph gate check --type=issue  # Evaluate only issue gates
  issuegraph gate resolve &lt;id&gt;   # Close a gate manually

```
issuegraph gate
```

#### issuegraph gate add-waiter

Register an agent as a waiter on a gate issue.

When the gate closes, the waiter will receive a wake notification via 'issuegraph gate wake'.
The waiter is typically the worker's address (e.g., "my-project/workers/agent-1").

This is used by 'issuegraph done --phase-complete' to register for gate wake notifications.

```
issuegraph gate add-waiter <gate-id> <waiter>
```

#### issuegraph gate check

Evaluate gate conditions and automatically close resolved gates.

By default, checks all open gates. Use --type to filter by gate type.

Gate types:
  gh       - Check all GitHub gates (gh:run and gh:pr)
  gh:run   - Check GitHub Actions workflow runs
  gh:pr    - Check pull request merge status
  timer    - Check timer gates (auto-expire based on timeout)
  issue     - Check cross-rig issue gates
  all      - Check all gate types

GitHub gates use the 'gh' CLI to query status:
  - gh:run checks 'gh run view &lt;id&gt; --json status,conclusion'
  - gh:pr checks 'gh pr view &lt;id&gt; --json state,title'

A gate is resolved when:
  - gh:run: status=completed AND conclusion=success
  - gh:pr: state=MERGED
  - timer: current time &gt; created_at + timeout
  - issue: target issue status=closed

A gate is escalated when:
  - gh:run: status=completed AND conclusion in (failure, canceled)
  - gh:pr: state=CLOSED

Examples:
  issuegraph gate check              # Check all gates
  issuegraph gate check --type=gh    # Check only GitHub gates
  issuegraph gate check --type=gh:run # Check only workflow run gates
  issuegraph gate check --type=timer # Check only timer gates
  issuegraph gate check --type=issue  # Check only cross-rig issue gates
  issuegraph gate check --dry-run    # Show what would happen without changes
  issuegraph gate check --escalate   # Escalate expired/failed gates

```
issuegraph gate check [flags]
```

**Flags:**

```
      --dry-run       Show what would happen without making changes
  -e, --escalate      Escalate failed/expired gates
  -l, --limit int     Limit results (default 100) (default 100)
  -t, --type string   Gate type to check (gh, gh:run, gh:pr, timer, issue, all)
```

#### issuegraph gate create

Create an ad-hoc gate issue that blocks another issue until resolved.

The blocked issue will not appear in 'issuegraph ready' until the gate is resolved
via 'issuegraph gate resolve'.

Gate types:
  human   - Requires manual 'issuegraph gate resolve' (default)
  timer   - Auto-resolves after --timeout duration
  gh:run  - Waits for GitHub Actions workflow
  gh:pr   - Waits for PR merge

Examples:
  issuegraph gate create --blocks issuegraph-abc
  issuegraph gate create --type=human --blocks issuegraph-abc --reason="Need design review"
  issuegraph gate create --type=timer --blocks issuegraph-abc --timeout=2h
  issuegraph gate create --type=gh:pr --blocks issuegraph-abc --await-id=42

```
issuegraph gate create [flags]
```

**Flags:**

```
      --await-id string   Condition identifier (run ID, PR number, etc.)
      --blocks string     Issue ID to block (required)
  -r, --reason string     Reason for the gate
      --timeout string    Timeout duration (e.g., 2h, 30m)
  -t, --type string       Gate type (human, timer, gh:run, gh:pr) (default "human")
```

#### issuegraph gate discover

Discovers GitHub workflow run IDs for gates awaiting CI/CD completion.

This command finds open gates with await_type="gh:run" that don't have an await_id,
queries recent GitHub workflow runs, and matches them using heuristics:
  - Branch name matching
  - Commit SHA matching
  - Time proximity (runs within 5 minutes of gate creation)

Once matched, the gate's await_id is updated with the GitHub run ID, enabling
subsequent polling to check the run's status.

Examples:
  issuegraph gate discover           # Auto-discover run IDs for all matching gates
  issuegraph gate discover --dry-run # Preview what would be matched (no updates)
  issuegraph gate discover --branch main --limit 10  # Only match runs on 'main' branch

```
issuegraph gate discover [flags]
```

**Flags:**

```
  -b, --branch string      Filter runs by branch (default: current branch)
  -n, --dry-run            Preview mode: show matches without updating
  -l, --limit int          Max runs to query from GitHub (default 10)
  -a, --max-age duration   Max age for gate/run matching (default 30m0s)
```

#### issuegraph gate list

List all gate issues in the current issuegraph database.

By default, shows only open gates. Use --all to include closed gates.

```
issuegraph gate list [flags]
```

**Flags:**

```
  -a, --all         Show all gates including closed
  -n, --limit int   Limit results (default 50) (default 50)
```

#### issuegraph gate resolve

Close a gate issue to unblock the step waiting on it.

This is equivalent to 'issuegraph close &lt;gate-id&gt;' but with a more explicit name.
Use --reason to provide context for why the gate was resolved.

```
issuegraph gate resolve <gate-id> [flags]
```

**Flags:**

```
  -r, --reason string   Reason for resolving the gate
```

#### issuegraph gate show

Display details of a gate issue including its waiters.

This is similar to 'issuegraph show' but validates that the issue is a gate.

```
issuegraph gate show <gate-id>
```

### issuegraph label

Manage issue labels

```
issuegraph label
```

#### issuegraph label add

Add a label to one or more issues

```
issuegraph label add [issue-id...] [label]
```

#### issuegraph label list

List labels for an issue

```
issuegraph label list [issue-id]
```

#### issuegraph label list-all

List all unique labels in the database

```
issuegraph label list-all
```

#### issuegraph label propagate

Push a label from a parent down to all direct children that don't already have it. Useful for applying branch: labels across an epic's subtasks.

```
issuegraph label propagate [parent-id] [label]
```

#### issuegraph label remove

Remove a label from one or more issues

```
issuegraph label remove [issue-id...] [label]
```

### issuegraph link

Link two issues with a dependency.

Shorthand for 'issuegraph dep add &lt;id1&gt; &lt;id2&gt;'. By default creates a "blocks"
dependency (id2 blocks id1). Use --type to specify a different relationship.

Examples:
  issuegraph link issuegraph-123 issuegraph-456                    # issuegraph-456 blocks issuegraph-123
  issuegraph link issuegraph-123 issuegraph-456 --type related     # issuegraph-123 related to issuegraph-456
  issuegraph link issuegraph-123 issuegraph-456 --type parent-child

```
issuegraph link <id1> <id2> [flags]
```

**Flags:**

```
  -t, --type string   Dependency type (blocks|tracks|related|parent-child|discovered-from) (default "blocks")
```

### issuegraph list

List issues

```
issuegraph list [flags]
```

**Flags:**

```
      --all                          Show all issues including closed (overrides default filter)
  -a, --assignee string              Filter by assignee
      --closed-after string          Filter issues closed after date (YYYY-MM-DD or RFC3339)
      --closed-before string         Filter issues closed before date (YYYY-MM-DD or RFC3339)
      --created-after string         Filter issues created after date (YYYY-MM-DD or RFC3339)
      --created-before string        Filter issues created before date (YYYY-MM-DD or RFC3339)
      --defer-after string           Filter issues deferred after date (supports relative: +6h, tomorrow)
      --defer-before string          Filter issues deferred before date (supports relative: +6h, tomorrow)
      --deferred                     Show only issues with defer_until set
      --desc-contains string         Filter by description substring (case-insensitive)
      --due-after string             Filter issues due after date (supports relative: +6h, tomorrow)
      --due-before string            Filter issues due before date (supports relative: +6h, tomorrow)
      --empty-description            Filter issues with empty or missing description
      --exclude-label strings        Exclude issues that have ANY of these labels
      --exclude-type strings         Exclude issue types from results (comma-separated or repeatable, e.g., --exclude-type=convoy,epic)
      --flat                         Disable tree format and use legacy flat list output
      --format string                Output format: 'digraph' (for golang.org/x/tools/cmd/digraph), 'dot' (Graphviz), or Go template
      --has-metadata-key string      Filter issues that have this metadata key set
      --id string                    Filter by specific issue IDs (comma-separated, e.g., issuegraph-1,issuegraph-5,issuegraph-10)
      --include-gates                Include gate issues in output (normally hidden)
      --include-infra                Include infrastructure issuegraph (agent/role/message) in output
      --include-templates            Include template molecules in output
  -l, --label strings                Filter by labels (AND: must have ALL). Can combine with --label-any
      --label-any strings            Filter by labels (OR: must have AT LEAST ONE). Can combine with --label
      --label-pattern string         Filter by label glob pattern (e.g., 'tech-*' matches tech-debt, tech-legacy)
      --label-regex string           Filter by label regex pattern (e.g., 'tech-(debt|legacy)')
  -n, --limit int                    Limit results (default 50, use 0 for unlimited) (default 50)
      --long                         Show detailed multi-line output for each issue
      --metadata-field stringArray   Filter by metadata field (key=value, repeatable)
      --mol-type string              Filter by molecule type: swarm, patrol, or work
      --no-assignee                  Filter issues with no assignee
      --no-labels                    Filter issues with no labels
      --no-pager                     Disable pager output
      --no-parent                    Exclude child issues (show only top-level issues)
      --no-pinned                    Exclude pinned issues
      --notes-contains string        Filter by notes substring (case-insensitive)
      --offset int                   Skip the first N matching results (0-based). Only supported under --proxied-server.
      --overdue                      Show only issues with due_at in the past (not closed)
      --parent string                Filter by parent issue ID (shows children of specified issue)
      --pinned                       Show only pinned issues
      --pretty                       Display issues in a tree format with status/priority symbols
  -p, --priority string              Priority (0-4 or P0-P4, 0=highest)
      --priority-max string          Filter by maximum priority (inclusive, 0-4 or P0-P4)
      --priority-min string          Filter by minimum priority (inclusive, 0-4 or P0-P4)
      --ready                        Show only ready issues (no active blockers, same semantics as issuegraph ready)
  -r, --reverse                      Reverse sort order
      --skip-labels                  Skip label hydration. The labels field in output will be empty regardless of actual labels. Use only when the caller does not depend on label data. Cannot combine with --label, --label-any, --label-pattern, --label-regex, --exclude-label, or --no-labels.
      --sort string                  Sort by field: priority, created, updated, closed, status, id, title, type, assignee
      --spec string                  Filter by spec_id prefix
  -s, --status string                Filter by stored status (open, in_progress, blocked, deferred, closed). Comma-separated for multiple: --status open,in_progress. Note: repeating -s/--status silently overwrites the previous value — always use the comma-separated form for multi-status filters.
      --title string                 Filter by title text (case-insensitive substring match)
      --title-contains string        Filter by title substring (case-insensitive)
      --tree                         Hierarchical tree format (default: true; use --flat to disable) (default true)
  -t, --type string                  Filter by type (bug, feature, task, epic, chore, decision, merge-request, molecule, gate, convoy). Aliases: mr→merge-request, feat→feature, mol→molecule, dec/adr→decision
      --updated-after string         Filter issues updated after date (YYYY-MM-DD or RFC3339)
      --updated-before string        Filter issues updated before date (YYYY-MM-DD or RFC3339)
  -w, --watch                        Watch for changes and auto-update display (implies --pretty)
      --wisp-type string             Filter by wisp type: heartbeat, ping, patrol, gc_report, recovery, error, escalation
```

### issuegraph merge-slot

Merge-slot gates serialize conflict resolution in the merge queue.

A merge slot is an exclusive access primitive: only one agent can hold it at a time.
This prevents "monkey knife fights" where multiple polecats race to resolve conflicts
and create cascading conflicts.

Each rig has one merge slot issue: &lt;prefix&gt;-merge-slot (labeled gt:slot).
The slot uses:
  - status=open: slot is available
  - status=in_progress: slot is held
  - metadata.holder: who currently holds the slot
  - metadata.waiters: priority-ordered queue of waiters

Examples:
  issuegraph merge-slot create              # Create merge slot for current rig
  issuegraph merge-slot check               # Check if slot is available
  issuegraph merge-slot acquire             # Try to acquire the slot
  issuegraph merge-slot release             # Release the slot

```
issuegraph merge-slot
```

#### issuegraph merge-slot acquire

Attempt to acquire the merge slot for exclusive access.

If the slot is available (status=open), it will be acquired:
  - status set to in_progress
  - holder set to the requester

If the slot is held (status=in_progress), the command fails unless
--wait is passed, which adds the requester to the waiters queue.

Use --holder to specify who is acquiring (default: ISSUEGRAPH_ACTOR env var).

```
issuegraph merge-slot acquire [flags]
```

**Flags:**

```
      --holder string   Who is acquiring the slot (default: ISSUEGRAPH_ACTOR)
      --wait            Add to waiters list if slot is held
```

#### issuegraph merge-slot check

Check if the merge slot is available or held.

Returns:
  - available: slot can be acquired
  - held by &lt;holder&gt;: slot is currently held
  - not found: no merge slot exists for this rig

```
issuegraph merge-slot check
```

#### issuegraph merge-slot create

Create a merge slot issue for serialized conflict resolution.

The slot ID is automatically generated based on the issuegraph prefix (e.g., gt-merge-slot).
The slot is created with status=open (available).

```
issuegraph merge-slot create
```

#### issuegraph merge-slot release

Release the merge slot after conflict resolution is complete.

Sets status back to open and clears the holder field.
If there are waiters, the highest-priority waiter should then acquire.

```
issuegraph merge-slot release [flags]
```

**Flags:**

```
      --holder string   Who is releasing the slot (for verification)
```

### issuegraph note

Append a note to an issue's notes field.

Shorthand for 'issuegraph update &lt;id&gt; --append-notes "text"'.

Examples:
  issuegraph note gt-abc "Fixed the flaky test"
  issuegraph note gt-abc Fixed the flaky test
  echo "note from pipe" | issuegraph note gt-abc --stdin
  issuegraph note gt-abc --file notes.txt

```
issuegraph note <id> [text...] [flags]
```

**Flags:**

```
      --file string   Read note text from file
      --stdin         Read note text from stdin
```

### issuegraph priority

Set the priority of an issue.

Shorthand for 'issuegraph update &lt;id&gt; --priority &lt;n&gt;'.

Priority levels:
  0 - Critical (security, data loss, broken builds)
  1 - High (major features, important bugs)
  2 - Medium (default)
  3 - Low (polish, optimization)
  4 - Backlog (future ideas)

Examples:
  issuegraph priority issuegraph-123 0    # Critical
  issuegraph priority issuegraph-123 2    # Medium

```
issuegraph priority <id> <n>
```

### issuegraph promote

Promote a wisp (ephemeral issue) to a permanent issue.

This copies the issue from the wisps table (dolt_ignored) to the permanent
issues table (Dolt-versioned), preserving labels, dependencies, events, and
comments. The original ID is preserved so all links keep working.

A comment is added recording the promotion and optional reason.

Examples:
  issuegraph promote issuegraph-wisp-abc123
  issuegraph promote issuegraph-wisp-abc123 --reason "Worth tracking long-term"

```
issuegraph promote <wisp-id> [flags]
```

**Flags:**

```
  -r, --reason string   Reason for promotion
```

### issuegraph q

Quick capture creates an issue and outputs only the issue ID.
Designed for scripting and AI agent integration.

Example:
  issuegraph q "Fix login bug"           # Outputs: issuegraph-a1b2
  ISSUE=$(issuegraph q "New feature")    # Capture ID in variable
  issuegraph q "Task" | xargs issuegraph show    # Pipe to other commands

```
issuegraph q [title] [flags]
```

**Flags:**

```
  -l, --labels strings    Labels
  -p, --priority string   Priority (0-4 or P0-P4) (default "2")
  -t, --type string       Issue type (default "task")
```

### issuegraph query

Query issues using a simple query language that supports compound filters,
boolean operators, and date-relative expressions.

The query language enables complex filtering that would otherwise require
multiple flags or piping through jq.

Syntax:
  field=value       Equality comparison
  field!=value      Inequality comparison
  field&gt;value       Greater than
  field&gt;=value      Greater than or equal
  field&lt;value       Less than
  field&lt;=value      Less than or equal

Boolean operators (case-insensitive):
  expr AND expr     Both conditions must match
  expr OR expr      Either condition can match
  NOT expr          Negates the condition
  (expr)            Grouping with parentheses

Supported fields:
  status            Stored status (open, in_progress, blocked, deferred, closed). Note: dependency-blocked issues stay "open"; use 'issuegraph blocked' to find them
  priority          Priority level (0-4)
  type              Issue type (bug, feature, task, epic, chore, decision)
  assignee          Assigned user (use "none" for unassigned)
  owner             Issue owner
  label             Issue label (use "none" for unlabeled)
  title             Search in title (contains)
  description       Search in description (contains, "none" for empty)
  notes             Search in notes (contains)
  created           Creation date/time
  updated           Last update date/time
  started           Date/time issue first transitioned to in_progress
  closed            Close date/time
  id                Issue ID (supports wildcards: issuegraph-*)
  spec              Spec ID (supports wildcards)
  pinned            Boolean (true/false)
  ephemeral         Boolean (true/false)
  template          Boolean (true/false)
  parent            Parent issue ID
  mol_type          Molecule type (swarm, patrol, work)

Date values:
  Relative durations: 7d (7 days ago), 24h (24 hours ago), 2w (2 weeks ago)
  Absolute dates: 2025-01-15, 2025-01-15T10:00:00Z
  Natural language: tomorrow, "next monday", "in 3 days"

Examples:
  issuegraph query "status=open AND priority&gt;1"
  issuegraph query "status=open AND priority&lt;=2 AND updated&gt;7d"
  issuegraph query "(status=open OR status=blocked) AND priority&lt;2"
  issuegraph query "type=bug AND label=urgent"
  issuegraph query "NOT status=closed"
  issuegraph query "assignee=none AND type=task"
  issuegraph query "created&gt;30d AND status!=closed"
  issuegraph query "label=frontend OR label=backend"
  issuegraph query "title=authentication AND priority=0"

```
issuegraph query [expression] [flags]
```

**Flags:**

```
  -a, --all           Include closed issues (default: exclude closed)
  -n, --limit int     Limit results (default: 50, 0 = unlimited) (default 50)
      --long          Show detailed multi-line output for each issue
      --offset int    Skip the first N matching results (0-based). Only supported under --proxied-server.
      --parse-only    Only parse the query and show the AST (for debugging)
  -r, --reverse       Reverse sort order
      --sort string   Sort by field: priority, created, updated, closed, status, id, title, type, assignee
```

### issuegraph reopen

Reopen closed issues by setting status to 'open' and clearing the closed_at timestamp.
This is more explicit than 'issuegraph update --status open' and emits a Reopened event.

```
issuegraph reopen [id...] [flags]
```

**Flags:**

```
  -r, --reason string   Reason for reopening
```

### issuegraph search

Search issues across title and ID (excludes closed issues by default).

ID-like queries (e.g., "issuegraph-123", "hq-319") use fast exact/prefix matching.
Text queries search titles. Use --desc-contains for description search.
Use --status all to include closed issues.

Examples:
  issuegraph search "authentication bug"
  issuegraph search "login" --status open
  issuegraph search "database" --label backend --limit 10
  issuegraph search --query "performance" --assignee alice
  issuegraph search "issuegraph-5q" # Search by partial ID (fast prefix match)
  issuegraph search "security" --priority-min 0 --priority-max 2
  issuegraph search "bug" --created-after 2025-01-01
  issuegraph search "refactor" --status all  # Include closed issues
  issuegraph search "bug" --sort priority
  issuegraph search "task" --sort created --reverse
  issuegraph search "api" --desc-contains "endpoint"
  issuegraph search "cleanup" --no-assignee --no-labels

```
issuegraph search [query] [flags]
```

**Flags:**

```
  -a, --assignee string              Filter by assignee
      --closed-after string          Filter issues closed after date (YYYY-MM-DD or RFC3339)
      --closed-before string         Filter issues closed before date (YYYY-MM-DD or RFC3339)
      --created-after string         Filter issues created after date (YYYY-MM-DD or RFC3339)
      --created-before string        Filter issues created before date (YYYY-MM-DD or RFC3339)
      --desc-contains string         Filter by description substring (case-insensitive)
      --empty-description            Filter issues with empty or missing description
      --external-contains string     Filter by external ref substring (case-insensitive)
      --has-metadata-key string      Filter issues that have this metadata key set
  -l, --label strings                Filter by labels (AND: must have ALL)
      --label-any strings            Filter by labels (OR: must have AT LEAST ONE)
  -n, --limit int                    Limit results (default: 50) (default 50)
      --long                         Show detailed multi-line output for each issue
      --metadata-field stringArray   Filter by metadata field (key=value, repeatable)
      --no-assignee                  Filter issues with no assignee
      --no-labels                    Filter issues with no labels
      --notes-contains string        Filter by notes substring (case-insensitive)
      --priority-max string          Filter by maximum priority (inclusive, 0-4 or P0-P4)
      --priority-min string          Filter by minimum priority (inclusive, 0-4 or P0-P4)
      --query string                 Search query (alternative to positional argument)
  -r, --reverse                      Reverse sort order
      --sort string                  Sort by field: priority, created, updated, closed, status, id, title, type, assignee
  -s, --status string                Filter by stored status (open, in_progress, blocked, deferred, closed, all). Default excludes closed; use 'all' to include closed. Note: dependency-blocked issues use 'issuegraph blocked'
  -t, --type string                  Filter by type (bug, feature, task, epic, chore, decision, merge-request, molecule, gate)
      --updated-after string         Filter issues updated after date (YYYY-MM-DD or RFC3339)
      --updated-before string        Filter issues updated before date (YYYY-MM-DD or RFC3339)
```

### issuegraph set-state

Atomically set operational state on an issue.

This command:
1. Creates an event issue recording the state change (source of truth)
2. Removes any existing label for the dimension
3. Adds the new dimension:value label (fast lookup cache)

State labels follow the convention &lt;dimension&gt;:&lt;value&gt;, for example:
  patrol:active, patrol:muted
  mode:normal, mode:degraded
  health:healthy, health:failing

Examples:
  issuegraph set-state agent-abc patrol=muted --reason "Investigating stuck worker"
  issuegraph set-state agent-abc mode=degraded --reason "High error rate detected"
  issuegraph set-state agent-abc health=healthy

The --reason flag provides context for the event issue (recommended).

```
issuegraph set-state <issue-id> <dimension>=<value> [flags]
```

**Flags:**

```
      --reason string   Reason for the state change (recorded in event)
```

### issuegraph show

Show issue details

```
issuegraph show [id...] [--id=<id>...] [--current] [flags]
```

**Aliases:** view

**Flags:**

```
      --as-of string         Show issue as it existed at a specific commit hash or branch (requires Dolt)
      --children             Show only the children of this issue
      --current              Show the currently active issue (in-progress, hooked, or last touched)
      --id stringArray       Issue ID (use for IDs that look like flags, e.g., --id=gt--xyz)
      --include-comments     Stream full comment bodies in JSON output (--json only; may be slow on issues with many comments)
      --include-dependents   Stream full dependent issues in JSON output (--json only; may be slow on hub issuegraph)
      --local-time           Show timestamps in local time instead of UTC
      --long                 Show all available fields (extended metadata, agent identity, gate fields, etc.)
      --refs                 Show issues that reference this issue (reverse lookup)
      --short                Show compact one-line output per issue
      --thread               Show full conversation thread (for messages)
  -w, --watch                Watch for changes and auto-refresh display
```

### issuegraph state

Query the current value of a state dimension from an issue's labels.

State labels follow the convention &lt;dimension&gt;:&lt;value&gt;, for example:
  patrol:active
  mode:degraded
  health:healthy

This command extracts the value for a given dimension.

Examples:
  issuegraph state witness-abc patrol     # Output: active
  issuegraph state witness-abc mode       # Output: normal
  issuegraph state witness-abc health     # Output: healthy

```
issuegraph state <issue-id> <dimension>
```

#### issuegraph state list

List all state labels (dimension:value format) on an issue.

This filters labels to only show those following the state convention.

Example:
  issuegraph state list witness-abc
  # Output:
  #   patrol: active
  #   mode: normal
  #   health: healthy

```
issuegraph state list <issue-id>
```

### issuegraph tag

Add a label to an issue.

Shorthand for 'issuegraph update &lt;id&gt; --add-label &lt;label&gt;'.

Examples:
  issuegraph tag issuegraph-123 bug
  issuegraph tag issuegraph-123 needs-review

```
issuegraph tag <id> <label>
```

### issuegraph todo

Manage TODO items as lightweight task issues.

TODOs are regular task-type issues with convenient shortcuts:
  issuegraph todo add "Title"    -&gt; issuegraph create "Title" -t task -p 2
  issuegraph todo                -&gt; issuegraph list --type task --status open
  issuegraph todo done &lt;id&gt;      -&gt; issuegraph close &lt;id&gt;

TODOs can be promoted to full issues by changing type or priority:
  issuegraph update todo-123 --type bug --priority 0

```
issuegraph todo
```

#### issuegraph todo add

Add a new TODO item

```
issuegraph todo add <title> [flags]
```

**Flags:**

```
  -d, --description string   Description
  -p, --priority int         Priority (0-4, default 2) (default 2)
```

#### issuegraph todo done

Mark TODO(s) as done

```
issuegraph todo done <id> [<id>...] [flags]
```

**Flags:**

```
      --reason string   Reason for closing (default: Completed)
```

#### issuegraph todo list

List TODO items

```
issuegraph todo list [flags]
```

**Flags:**

```
      --all   Show all TODOs including completed
```

### issuegraph update

Update one or more issues.

If no issue ID is provided, updates the last touched issue (from most recent
create, update, show, or close operation).

```
issuegraph update [id...] [flags]
```

**Flags:**

```
      --acceptance string            Acceptance criteria
      --add-label strings            Add labels (repeatable)
      --allow-empty-description      Allow empty description replacement when reading from stdin or file
      --append-notes string          Append to existing notes (with newline separator)
  -a, --assignee string              Assignee
      --await-id string              Set gate await_id (e.g., GitHub run ID for gh:run gates)
      --body-file string             Read description from file (use - for stdin)
      --claim                        Atomically claim the issue (sets assignee to you, status to in_progress; idempotent if already claimed by you)
      --defer string                 Defer until date (empty to clear). Issue hidden from issuegraph ready until then
  -d, --description string           Issue description
      --design string                Design notes
      --design-file string           Read design from file (use - for stdin)
      --due string                   Due date/time (empty to clear). Formats: +6h, +1d, +2w, tomorrow, next monday, 2025-01-15
      --ephemeral                    Mark issue as ephemeral (wisp) - not exported to JSONL
  -e, --estimate int                 Time estimate in minutes (e.g., 60 for 1 hour)
      --external-ref string          External reference (e.g., 'gh-9', 'jira-ABC', Linear URL)
      --history                      Clear no-history flag (re-enable Dolt commit history)
      --metadata string              Set custom metadata (JSON string or @file.json to read from file)
      --no-history                   Mark issue as no-history (skip Dolt commits, not GC-eligible)
      --notes string                 Additional notes
      --parent string                New parent issue ID (reparents the issue, use empty string to remove parent)
      --persistent                   Mark issue as persistent (promote wisp to regular issue)
  -p, --priority string              Priority (0-4 or P0-P4, 0=highest)
      --remove-label strings         Remove labels (repeatable)
      --session string               Claude Code session ID for status=closed (or set CLAUDE_SESSION_ID env var)
      --set-labels strings           Set labels, replacing all existing (repeatable)
      --set-metadata stringArray     Set metadata key=value (repeatable, e.g., --set-metadata team=platform)
      --spec-id string               Link to specification document
  -s, --status string                New status
      --stdin                        Read description from stdin (alias for --body-file -)
      --title string                 New title
  -t, --type string                  New type (bug|feature|task|epic|chore|decision); custom types require types.custom config
      --unset-metadata stringArray   Remove metadata key (repeatable, e.g., --unset-metadata team)
```

## Views & Reports:

### issuegraph count

Count issues matching the specified filters.

By default, returns the total count of issues matching the filters.
Use --by-* flags to group counts by different attributes.

Examples:
  issuegraph count                          # Count all issues
  issuegraph count --status open            # Count open issues
  issuegraph count --by-status              # Group count by status
  issuegraph count --by-priority            # Group count by priority
  issuegraph count --by-type                # Group count by issue type
  issuegraph count --by-assignee            # Group count by assignee
  issuegraph count --by-label               # Group count by label
  issuegraph count --assignee alice --by-status  # Count alice's issues by status
  issuegraph count --include-infra          # Count issues + wisps tier (matches 'issuegraph list --include-infra --all' cardinality)


```
issuegraph count [flags]
```

**Flags:**

```
  -a, --assignee string         Filter by assignee
      --by-assignee             Group count by assignee
      --by-label                Group count by label
      --by-priority             Group count by priority
      --by-status               Group count by status
      --by-type                 Group count by issue type
      --closed-after string     Filter issues closed after date (YYYY-MM-DD or RFC3339)
      --closed-before string    Filter issues closed before date (YYYY-MM-DD or RFC3339)
      --created-after string    Filter issues created after date (YYYY-MM-DD or RFC3339)
      --created-before string   Filter issues created before date (YYYY-MM-DD or RFC3339)
      --desc-contains string    Filter by description substring
      --empty-description       Filter issues with empty description
      --id string               Filter by specific issue IDs (comma-separated)
      --include-infra           Include infrastructure issuegraph and the wisps tier (matches 'issuegraph list --include-infra --all' cardinality)
  -l, --label strings           Filter by labels (AND: must have ALL)
      --label-any strings       Filter by labels (OR: must have AT LEAST ONE)
      --no-assignee             Filter issues with no assignee
      --no-labels               Filter issues with no labels
      --notes-contains string   Filter by notes substring
  -p, --priority int            Filter by priority (0-4: 0=critical, 1=high, 2=medium, 3=low, 4=backlog)
      --priority-max int        Filter by maximum priority (inclusive)
      --priority-min int        Filter by minimum priority (inclusive)
  -s, --status string           Filter by stored status (open, in_progress, blocked, deferred, closed). Note: dependency-blocked issues use 'issuegraph blocked'
      --title string            Filter by title text (case-insensitive substring match)
      --title-contains string   Filter by title substring
  -t, --type string             Filter by type (bug, feature, task, epic, chore, decision, merge-request, molecule, gate)
      --updated-after string    Filter issues updated after date (YYYY-MM-DD or RFC3339)
      --updated-before string   Filter issues updated before date (YYYY-MM-DD or RFC3339)
```

### issuegraph diff

Show the differences in issues between two commits or branches.

The refs can be:
- Commit hashes (e.g., abc123def)
- Branch names (e.g., main, feature-branch)
- Special refs like HEAD, HEAD~1

Examples:
  issuegraph diff main feature-branch   # Compare main to feature branch
  issuegraph diff HEAD~5 HEAD           # Show changes in last 5 commits
  issuegraph diff abc123 def456         # Compare two specific commits

```
issuegraph diff <from-ref> <to-ref>
```

### issuegraph find-duplicates

Find issues that are semantically similar but not exact duplicates.

Unlike 'issuegraph duplicates' which finds exact content matches, find-duplicates
uses text similarity or AI to find issues that discuss the same topic
with different wording.

Approaches:
  mechanical  Token-based text similarity (default, no API key needed)
  ai          LLM-based semantic comparison (requires ANTHROPIC_API_KEY or ai.api_key)

The mechanical approach tokenizes titles and descriptions, then computes
Jaccard similarity between all issue pairs. It's fast and free but may
miss semantically similar issues with very different wording.

The AI approach sends candidate pairs to Claude for semantic comparison.
It first uses mechanical pre-filtering to reduce the number of API calls,
then asks the LLM to judge whether the remaining pairs are true duplicates.

Examples:
  issuegraph find-duplicates                       # Mechanical similarity (default)
  issuegraph find-duplicates --threshold 0.4       # Lower threshold = more results
  issuegraph find-duplicates --method ai           # Use AI for semantic comparison
  issuegraph find-duplicates --status open         # Only check open issues
  issuegraph find-duplicates --limit 20            # Show top 20 pairs
  issuegraph find-duplicates --json                # JSON output

```
issuegraph find-duplicates [flags]
```

**Aliases:** find-dups

**Flags:**

```
  -n, --limit int         Maximum number of pairs to show (default 50)
      --method string     Detection method: mechanical, ai (default "mechanical")
      --model string      AI model to use (only with --method ai; default from config ai.model)
  -s, --status string     Filter by status (default: non-closed)
      --threshold float   Similarity threshold (0.0-1.0, lower = more results) (default 0.5)
```

### issuegraph history

Show the complete version history of an issue, including all commits
where the issue was modified.

Examples:
  issuegraph history issuegraph-123           # Show all history for issue issuegraph-123
  issuegraph history issuegraph-123 --limit 5 # Show last 5 changes

```
issuegraph history <id> [flags]
```

**Flags:**

```
      --limit int   Limit number of history entries (0 = all)
```

### issuegraph lint

Check issues for missing recommended sections based on issue type.

By default, lints all open issues. Specify issue IDs to lint specific issues.

Section requirements by type:
  bug:      Steps to Reproduce, Acceptance Criteria
  task:     Acceptance Criteria
  feature:  Acceptance Criteria
  epic:     Success Criteria
  chore:    (none)

Examples:
  issuegraph lint                    # Lint all open issues
  issuegraph lint issuegraph-abc             # Lint specific issue
  issuegraph lint issuegraph-abc issuegraph-def      # Lint multiple issues
  issuegraph lint --type bug         # Lint only bugs
  issuegraph lint --status all       # Lint all issues (including closed)


```
issuegraph lint [issue-id...] [flags]
```

**Flags:**

```
  -s, --status string   Filter by status (default: open, use 'all' for all)
  -t, --type string     Filter by issue type (bug, task, feature, epic)
```

### issuegraph stale

Show issues that haven't been updated recently and may need attention.
This helps identify:
- In-progress issues with no recent activity (may be abandoned)
- Open issues that have been forgotten
- Issues that might be outdated or no longer relevant

```
issuegraph stale [flags]
```

**Flags:**

```
  -d, --days int        Issues not updated in this many days (default 30)
  -n, --limit int       Maximum issues to show (default 50)
  -s, --status string   Filter by status (open|in_progress|blocked|deferred)
```

### issuegraph status

Show a quick snapshot of the issue database state and statistics.

This command provides a summary of issue counts by state (open, in_progress,
blocked, closed), ready work, extended statistics (pinned issues,
average lead time), and recent activity over the last 24 hours from git history.

Similar to how 'git status' shows working tree state, 'issuegraph status' gives you
a quick overview of your issue database without needing multiple queries.

Use cases:
  - Quick project health check
  - Onboarding for new contributors
  - Integration with shell prompts or CI/CD
  - Daily standup reference

Examples:
  issuegraph status                    # Show summary with activity
  issuegraph status --no-activity      # Skip git activity (faster)
  issuegraph status --json             # JSON format output
  issuegraph status --assigned         # Show issues assigned to current user
  issuegraph stats                     # Alias for issuegraph status

```
issuegraph status [flags]
```

**Aliases:** stats

**Flags:**

```
      --all           Show all issues (default behavior)
      --assigned      Show issues assigned to current user
      --no-activity   Skip git activity tracking (faster)
```

### issuegraph statuses

List all valid issue statuses and their categories.

Built-in statuses (open, in_progress, blocked, etc.) are always valid.
Additional statuses can be configured via status.custom:

  issuegraph config set status.custom "in_review:active,qa_testing:wip,on_hold:frozen"

Categories control behavior:
  active  — appears in 'issuegraph ready' and default 'issuegraph list'
  wip     — excluded from 'issuegraph ready', visible in default 'issuegraph list'
  done    — excluded from 'issuegraph ready' and default 'issuegraph list'
  frozen  — excluded from 'issuegraph ready' and default 'issuegraph list'

Statuses without a category (legacy format) are valid but excluded from 'issuegraph ready'.

Examples:
  issuegraph statuses            # List all statuses with icons and categories
  issuegraph statuses --json     # Output as JSON


```
issuegraph statuses
```

### issuegraph types

List all valid issue types that can be used with issuegraph create --type.

Core work types (bug, task, feature, chore, epic, decision) are always valid.
Additional types require configuration via types.custom in .issuegraph/config.yaml.

Examples:
  issuegraph types              # List all types with descriptions
  issuegraph types --json       # Output as JSON


```
issuegraph types
```

## Dependencies & Structure:

### issuegraph dep

Manage dependencies between issues.

When called with an issue ID and --blocks flag, creates a blocking dependency:
  issuegraph dep &lt;blocker-id&gt; --blocks &lt;blocked-id&gt;

This is equivalent to:
  issuegraph dep add &lt;blocked-id&gt; &lt;blocker-id&gt;

Examples:
  issuegraph dep issuegraph-xyz --blocks issuegraph-abc    # issuegraph-xyz blocks issuegraph-abc
  issuegraph dep add issuegraph-abc issuegraph-xyz         # Same as above (issuegraph-abc depends on issuegraph-xyz)

```
issuegraph dep [issue-id] [flags]
```

**Flags:**

```
  -b, --blocks string    Issue ID that this issue blocks (shorthand for: issuegraph dep add <blocked> <blocker>)
      --no-cycle-check   Skip per-edge cycle checks for speed (bulk wiring); bulk --file adds still run one final whole-graph check before commit
```

#### issuegraph dep add

Add a dependency between two issues.

The depends-on-id can be provided as:
  - A positional argument: issuegraph dep add issue-123 issue-456
  - A flag: issuegraph dep add issue-123 --blocked-by issue-456
  - A flag: issuegraph dep add issue-123 --depends-on issue-456

The --blocked-by and --depends-on flags are aliases and both mean "issue-123
depends on (is blocked by) the specified issue."

The depends-on-id can be:
  - A local issue ID (e.g., issuegraph-xyz)
  - An external reference: external:&lt;project&gt;:&lt;capability&gt;

For bulk wiring, pass newline-delimited JSON with --file. Each line must be an
object with "from" and "to" fields, and may include "type". The aliases
"issue_id" and "depends_on_id" are also accepted. Use --file - to read stdin.

External references are stored as-is and resolved at query time using
the external_projects config. They block the issue until the capability
is "shipped" in the target project.

Examples:
  issuegraph dep add issuegraph-42 issuegraph-41                              # Positional args
  issuegraph dep add issuegraph-42 --blocked-by issuegraph-41                 # Flag syntax (same effect)
  issuegraph dep add issuegraph-42 --depends-on issuegraph-41                 # Alias (same effect)
  issuegraph dep add gt-xyz external:issuegraph:mol-run-assignee   # Cross-project dependency
  issuegraph dep add issuegraph-42 issuegraph-41 --no-cycle-check             # Skip cycle check (bulk wiring)
  issuegraph dep add --file deps.jsonl                        # Bulk JSONL: &#123;"from":"issuegraph-42","to":"issuegraph-41"&#125;

```
issuegraph dep add [issue-id] [depends-on-id] [flags]
```

**Flags:**

```
      --blocked-by string   Issue ID that blocks the first issue (alternative to positional arg)
      --depends-on string   Issue ID that the first issue depends on (alias for --blocked-by)
      --file string         Read dependency edges from JSONL file, or '-' for stdin
      --no-cycle-check      Skip per-edge cycle checks for speed (bulk wiring); bulk --file adds still run one final whole-graph check before commit
  -t, --type string         Dependency type (blocks|tracks|related|parent-child|discovered-from|until|caused-by|validates|relates-to|supersedes) (default "blocks")
```

#### issuegraph dep cycles

Detect dependency cycles

```
issuegraph dep cycles
```

#### issuegraph dep list

List dependencies or dependents of one or more issues with optional type filtering.

By default shows dependencies (what issues depend on). Use --direction to control:
  - down: Show dependencies (what this issue depends on) - default
  - up:   Show dependents (what depends on this issue)

Multiple IDs can be provided for batch dep listing. With --json, the output
is a flat array of dependency records across all requested issues.

Use --type to filter by dependency type (e.g., tracks, blocks, parent-child).

Examples:
  issuegraph dep list gt-abc                     # Show what gt-abc depends on
  issuegraph dep list gt-abc gt-def              # Batch: deps for both issues
  issuegraph dep list gt-abc --direction=up      # Show what depends on gt-abc
  issuegraph dep list gt-abc --direction=up -t tracks  # Show what tracks gt-abc (convoy tracking)

```
issuegraph dep list [issue-id...] [flags]
```

**Flags:**

```
      --direction string   Direction: 'down' (dependencies), 'up' (dependents) (default "down")
  -t, --type string        Filter by dependency type (e.g., tracks, blocks, parent-child)
```

#### issuegraph dep relate

Create a loose 'see also' relationship between two issues.

The relates_to link is bidirectional - both issues will reference each other.
This enables knowledge graph connections without blocking or hierarchy.

Examples:
  issuegraph relate issuegraph-abc issuegraph-xyz    # Link two related issues
  issuegraph relate issuegraph-123 issuegraph-456    # Create see-also connection

```
issuegraph dep relate <id1> <id2>
```

#### issuegraph dep remove

Remove a dependency

```
issuegraph dep remove [issue-id] [depends-on-id]
```

**Aliases:** rm

#### issuegraph dep tree

Show dependency tree rooted at the given issue.

By default, shows dependencies (what blocks this issue). Use --direction to control:
  - down: Show dependencies (what blocks this issue) - default
  - up:   Show dependents (what this issue blocks)
  - both: Show full graph in both directions

Examples:
  issuegraph dep tree gt-0iqq                    # Show what blocks gt-0iqq
  issuegraph dep tree gt-0iqq --direction=up     # Show what gt-0iqq blocks
  issuegraph dep tree gt-0iqq --status=open      # Only show open issues
  issuegraph dep tree gt-0iqq --depth=3          # Limit to 3 levels deep

```
issuegraph dep tree [issue-id] [flags]
```

**Flags:**

```
      --direction string   Tree direction: 'down' (dependencies), 'up' (dependents), or 'both'
      --format string      Output format: 'mermaid' for Mermaid.js flowchart
  -d, --max-depth int      Maximum tree depth to display (safety limit) (default 50)
      --reverse            Show dependent tree (deprecated: use --direction=up)
      --show-all-paths     Show all paths to nodes (no deduplication for diamond dependencies)
      --status string      Filter to only show issues with this status (open, in_progress, blocked, deferred, closed)
```

#### issuegraph dep unrelate

Remove a relates_to relationship between two issues.

Removes the link in both directions.

Example:
  issuegraph unrelate issuegraph-abc issuegraph-xyz

```
issuegraph dep unrelate <id1> <id2>
```

### issuegraph duplicate

Mark an issue as a duplicate of a canonical issue.

The duplicate issue is automatically closed with a reference to the canonical.
This is essential for large issue databases with many similar reports.

Examples:
  issuegraph duplicate issuegraph-abc --of issuegraph-xyz    # Mark issuegraph-abc as duplicate of issuegraph-xyz

```
issuegraph duplicate <id> --of <canonical> [flags]
```

**Flags:**

```
      --of string   Canonical issue ID (required)
```

### issuegraph duplicates

Find issues with identical content (title, description, design, acceptance criteria).
Groups issues by content hash and reports duplicates with suggested merge targets.
The merge target is chosen by:
1. Reference count (most referenced issue wins)
2. Lexicographically smallest ID if reference counts are equal
Only groups issues with matching status (open with open, closed with closed).
Example:
  issuegraph duplicates                    # Show all duplicate groups
  issuegraph duplicates --auto-merge       # Automatically merge all duplicates
  issuegraph duplicates --dry-run          # Show what would be merged

```
issuegraph duplicates [flags]
```

**Flags:**

```
      --auto-merge   Automatically merge all duplicates
      --dry-run      Show what would be merged without making changes
```

### issuegraph epic

Epic management commands

```
issuegraph epic
```

#### issuegraph epic close-eligible

Close epics where all children are complete

```
issuegraph epic close-eligible [flags]
```

**Flags:**

```
      --dry-run   Preview what would be closed without making changes
```

#### issuegraph epic status

Show epic completion status

```
issuegraph epic status [flags]
```

**Flags:**

```
      --eligible-only   Show only epics eligible for closure
```

### issuegraph graph

Display a visualization of an issue's dependency graph.

For epics, shows all children and their dependencies.
For regular issues, shows the issue and its direct dependencies.

With --all, shows all open issues grouped by connected component.

Display formats:
  (default)        DAG with columns and box-drawing edges (terminal-native)
  --box            ASCII boxes showing layers, more detailed
  --compact        Tree format, one line per issue, more scannable
  --dot            Graphviz DOT format (pipe to dot -Tsvg &gt; graph.svg)
  --html           Self-contained interactive HTML with D3.js visualization

The graph shows execution order:
- Layer 0 / leftmost = no dependencies (can start immediately)
- Higher layers depend on lower layers
- Nodes in the same layer can run in parallel

Status icons: ○ open  ◐ in_progress  ● blocked  ✓ closed  ❄ deferred

Examples:
  issuegraph graph issue-id              # Terminal DAG visualization (default)
  issuegraph graph --box issue-id        # ASCII boxes with layer grouping
  issuegraph graph --dot issue-id | dot -Tsvg &gt; graph.svg  # SVG via Graphviz
  issuegraph graph --dot issue-id | dot -Tpng &gt; graph.png  # PNG via Graphviz
  issuegraph graph --html issue-id &gt; graph.html  # Interactive browser view
  issuegraph graph --all --html &gt; all.html       # All issues, interactive

```
issuegraph graph [issue-id] [flags]
```

**Flags:**

```
      --all       Show graph for all open issues
      --box       ASCII boxes showing layers
      --compact   Tree format, one line per issue, more scannable
      --dot       Output Graphviz DOT format (pipe to: dot -Tsvg > graph.svg)
      --html      Output self-contained interactive HTML (redirect to file)
```

#### issuegraph graph check

Check the dependency graph for cycles, orphans, and other integrity issues.

Returns exit code 0 if the graph is clean, 1 if issues are found.

```
issuegraph graph check
```

### issuegraph supersede

Mark an issue as superseded by a newer version.

The superseded issue is automatically closed with a reference to the replacement.
Useful for design docs, specs, and evolving artifacts.

Examples:
  issuegraph supersede issuegraph-old --with issuegraph-new    # Mark issuegraph-old as superseded by issuegraph-new

```
issuegraph supersede <id> --with <new> [flags]
```

**Flags:**

```
      --with string   Replacement issue ID (required)
```

### issuegraph swarm

Swarm management commands for coordinating parallel work on epics.

A swarm is a structured body of work defined by an epic and its children,
with dependencies forming a DAG (directed acyclic graph) of work.

```
issuegraph swarm
```

#### issuegraph swarm create

Create a swarm molecule to orchestrate parallel work on an epic.

The swarm molecule:
- Links to the epic it orchestrates
- Has mol_type=swarm for discovery
- Specifies a coordinator (optional)
- Can be picked up by any coordinator agent

If given a single issue (not an epic), it will be auto-wrapped:
- Creates an epic with that issue as its only child
- Then creates the swarm molecule for that epic

Examples:
  issuegraph swarm create issuegraph-epic-123                          # Create swarm for epic
  issuegraph swarm create issuegraph-epic-123 --coordinator=observer/   # With specific coordinator
  issuegraph swarm create issuegraph-task-456                          # Auto-wrap single issue

```
issuegraph swarm create [epic-id] [flags]
```

**Flags:**

```
      --coordinator string   Coordinator address (e.g., my-project/witness)
      --force                Create new swarm even if one already exists
```

#### issuegraph swarm list

List all swarm molecules with their status.

Shows each swarm molecule with:
- Progress (completed/total issues)
- Active workers
- Epic ID and title

Examples:
  issuegraph swarm list         # List all swarms
  issuegraph swarm list --json  # Machine-readable output

```
issuegraph swarm list
```

#### issuegraph swarm status

Show the current status of a swarm, computed from issuegraph.

Accepts either:
- An epic ID (shows status for that epic's children)
- A swarm molecule ID (follows the link to find the epic)

Displays issues grouped by state:
- Completed: Closed issues
- Active: Issues currently in_progress (with assignee)
- Ready: Open issues with all dependencies satisfied
- Blocked: Open issues waiting on dependencies

The status is COMPUTED from issuegraph, not stored separately.
If issuegraph changes, status changes.

Examples:
  issuegraph swarm status gt-epic-123       # Show swarm status by epic
  issuegraph swarm status gt-swarm-456      # Show status via swarm molecule
  issuegraph swarm status gt-epic-123 --json  # Machine-readable output

```
issuegraph swarm status [epic-or-swarm-id]
```

#### issuegraph swarm validate

Validate an epic's structure to ensure it's ready for swarm execution.

Checks for:
- Correct dependency direction (requirement-based, not temporal)
- Orphaned issues (roots with no dependents)
- Missing dependencies (leaves that should depend on something)
- Cycles (impossible to resolve)
- Disconnected subgraphs

Reports:
- Ready fronts (waves of parallel work)
- Estimated worker-sessions
- Maximum parallelism
- Warnings for potential issues

Examples:
  issuegraph swarm validate gt-epic-123           # Validate epic structure
  issuegraph swarm validate gt-epic-123 --verbose # Include detailed issue graph

```
issuegraph swarm validate [epic-id] [flags]
```

**Flags:**

```
      --verbose   Include detailed issue graph in output
```

## Sync & Data:

### issuegraph backup

Back up your issuegraph database for off-machine recovery.

This is a Dolt-native database backup. It preserves the database state,
including tables, branches, commit history, and working-set data. This is
different from 'issuegraph export', which writes issue records to JSONL for migration
and interoperability.

Commands:
  issuegraph backup init &lt;path&gt;    Set up a backup destination (filesystem or DoltHub)
  issuegraph backup sync           Push to configured backup destination
  issuegraph backup restore [path] Restore from a backup directory
  issuegraph backup remove         Remove backup destination
  issuegraph backup status         Show backup status

DoltHub is recommended for cloud backup:
  issuegraph backup init https://doltremoteapi.dolthub.com/&lt;user&gt;/&lt;repo&gt;
  Set DOLT_REMOTE_USER and DOLT_REMOTE_PASSWORD for authentication.

```
issuegraph backup
```

#### issuegraph backup init

Configure a filesystem path or URL as a backup destination.

The path can be a local directory (external drive, NAS, Dropbox folder) or a
DoltHub remote URL. If the destination was previously configured, it is
updated to the new path.

Filesystem examples:
  issuegraph backup add /mnt/usb/issuegraph-backup
  issuegraph backup add ~/Dropbox/issuegraph-backup

DoltHub (recommended for cloud backup):
  issuegraph backup add https://doltremoteapi.dolthub.com/myuser/issuegraph-backup

After adding, run 'issuegraph backup sync' to push your data.

```
issuegraph backup init <path>
```

**Aliases:** add

#### issuegraph backup remove

Remove the configured backup destination.

This unregisters the backup remote from Dolt and removes the local
backup configuration. The backup data at the destination is not deleted.

```
issuegraph backup remove
```

**Aliases:** rm

#### issuegraph backup restore

Restore the issuegraph database from a Dolt-native backup.

By default, reads from .issuegraph/backup/ (or the configured backup directory).
Optionally specify a path to a directory containing a Dolt backup.

This restores a full database backup created by 'issuegraph backup sync' or an
equivalent Dolt backup. JSONL files produced by 'issuegraph export' are issue exports,
not restore targets for this command.

Use --force to overwrite an existing database with the backup contents.

The database must already be initialized (run 'issuegraph init' first if needed).
To initialize and restore in one step, use: issuegraph init &amp;&amp; issuegraph backup restore

```
issuegraph backup restore [path] [flags]
```

**Flags:**

```
      --force   Overwrite existing database with backup contents
```

#### issuegraph backup status

Show last backup status

```
issuegraph backup status
```

#### issuegraph backup sync

Sync the current issuegraph database to the configured Dolt backup destination.

This pushes the entire database state (all branches, full history) to the
backup location configured with 'issuegraph backup init'.

The backup is atomic — if the sync fails, the previous backup state is preserved.

Run 'issuegraph backup init &lt;path&gt;' first to configure a destination.

```
issuegraph backup sync
```

### issuegraph branch

List all branches or create a new branch.

This command requires the Dolt storage backend. Without arguments,
it lists all branches. With an argument, it creates a new branch.

Examples:
  issuegraph branch                    # List all branches
  issuegraph branch feature-xyz        # Create a new branch named feature-xyz

```
issuegraph branch [name]
```

### issuegraph export

Export all issues to JSONL (newline-delimited JSON) format.

Each line is a complete JSON object representing one issue, including its
labels, dependencies, and comments.

This command is for issue export, migration, and interoperability. It exports
records from the issues table; it is not a full database backup and does not
capture Dolt branches, commit history, working-set state, or non-issue tables.
For supported full backup/restore flows, use 'issuegraph backup init', 'issuegraph backup sync',
and 'issuegraph backup restore'.

By default, exports only regular issues (excluding infrastructure issuegraph
like agents, roles, and messages). Use --all to include everything.

Memories (from 'issuegraph remember') are excluded by default because they may
contain sensitive agent context. Use --include-memories or --all to
include them.

EXAMPLES:
  issuegraph export                              # Export issues to stdout
  issuegraph export -o issues.jsonl              # Export issues to file
  issuegraph export --include-memories           # Export issues + memories
  issuegraph export --all -o full.jsonl          # Include infra + templates + gates + memories
  issuegraph export --scrub -o clean.jsonl       # Exclude test/pollution records

```
issuegraph export [flags]
```

**Flags:**

```
      --all                Include all records (infra, templates, gates, memories)
      --include-infra      Include infrastructure issuegraph (agents, roles, messages)
      --include-memories   Include persistent memories (from 'issuegraph remember') in the export
  -o, --output string      Output file path (default: stdout)
      --scrub              Exclude test/pollution records
```

### issuegraph federation

Federation commands require CGO and the Dolt storage backend.

This binary was built without CGO support. To use federation features:
  1. Use pre-built binaries from GitHub releases, or
  2. Build from source with CGO enabled

Federation enables synchronized issue tracking across multiple workspaces,
each maintaining their own Dolt database while sharing updates via remotes.

```
issuegraph federation
```

### issuegraph import

Import issues from a JSONL file (newline-delimited JSON) into the database.

If no file is specified, imports from the configured import.path under .issuegraph/
(default: issues.jsonl). Use "-" to read from stdin. This is the incremental counterpart to
'issuegraph export': new issues are created and existing issues are updated (upsert
semantics).

Memory records (lines with "_type":"memory") are automatically detected and
imported as persistent memories (equivalent to 'issuegraph remember'). This makes
'issuegraph export | issuegraph import' a full round-trip for both issues and memories.

Each JSONL line should map to an issue. The importer accepts every field
'issuegraph export' emits — see 'issuegraph export' output for the canonical schema. Only
"title" is required; everything else is optional.

Common fields:
  title                  Required. Short summary.
  description            Long-form body.
  design, notes,         Additional content sections.
    acceptance_criteria
  issue_type             bug | feature | task | epic | chore | ...
  priority               0-4 (0 = critical). 0 is preserved (no omitempty).
  status                 open | in_progress | blocked | closed | ...
                         (rows with status "tombstone" are skipped)
  assignee, owner,       Ownership metadata.
    created_by
  labels                 Array of strings.
  dependencies           Array of &#123;issue_id, depends_on_id, type, ...&#125;.
  comments               Array of comment objects.
  external_ref,          Cross-system identifiers (e.g. "gh-9").
    source_system
  due_at, defer_until    RFC3339 timestamps for scheduling.
  metadata               Arbitrary JSON object preserved verbatim.

Timestamps (created_at, updated_at, started_at, closed_at) are preserved
when present in the JSONL and otherwise filled in by the importer. The
legacy "wisp" boolean is accepted as an alias for "ephemeral".

By default a row only rewrites an existing local issue when its
updated_at is strictly newer. Older rows are skipped (reported as
stale_skipped_ids) and rows with the same updated_at keep every local
column — updated_at has second granularity, so a timestamp tie can be
two distinct same-second updates, and the local row wins the tie
(reported as tie_kept_local_ids; the row's labels/comments/dependencies
still merge). The guard is also enforced inside the upsert itself, so a
local update that lands while the import is running is preserved rather
than overwritten. Existing issues that the import did rewrite are listed
with a field-level summary (updated_issues), so local state changed by
an import is visible. To deliberately restore an older snapshot, pass
--allow-stale, which imports every row even when it overwrites newer
local state.

EXAMPLES:
  issuegraph import                        # Import from configured import.path
  issuegraph import backup.jsonl           # Import from a specific file
  issuegraph import -i backup.jsonl        # Legacy alias for a specific file
  issuegraph import -                      # Read JSONL from stdin
  cat issues.jsonl | issuegraph import -   # Pipe JSONL from another tool
  issuegraph import --dry-run              # Show what would be imported
  issuegraph import --dedup                # Skip issues with duplicate titles
  issuegraph import --allow-stale old.jsonl # Restore an older snapshot (overwrites newer local rows)
  issuegraph import --json                 # Structured output with created and skipped IDs

```
issuegraph import [file|-] [flags]
```

**Flags:**

```
      --allow-stale    Import rows even when older than the local issue (required to restore an older snapshot)
      --dedup          Skip lines whose title matches an existing open issue
      --dry-run        Show what would be imported without importing
  -i, --input string   Read JSONL from a specific file
```

### issuegraph restore

Restore the pre-compaction content of a compacted issue.

When an issue is compacted, its description/design/notes/acceptance criteria
are summarized and the originals are archived to a compaction snapshot. This
command recovers that original content.

By default it is read-only: it displays the archived content without modifying
the database. Pass --apply to write the original content back into the issue
and step its compaction level back down.

If no archived snapshot exists (e.g. the issue was compacted by an older issuegraph
before snapshot archiving), restore falls back to a best-effort reconstruction
from Dolt version history, which can only be displayed, not applied.

```
issuegraph restore <issue-id> [flags]
```

**Flags:**

```
      --apply   Write the restored content back into the issue (default: display only)
      --json    Output restore results in JSON format
```

### issuegraph vc

Version control operations for the issuegraph database.

These commands provide git-like version control for your issue data, including branching, merging, and
viewing history.

Note: 'issuegraph history', 'issuegraph diff', and 'issuegraph branch' also work for quick access.
This subcommand provides additional operations like merge and commit.

```
issuegraph vc
```

#### issuegraph vc commit

Create a new Dolt commit with all current changes.

Examples:
  issuegraph vc commit -m "Added new feature issues"
  issuegraph vc commit --message "Fixed priority on several issues"
  echo "Multi-line message" | issuegraph vc commit --stdin

```
issuegraph vc commit [flags]
```

**Flags:**

```
  -m, --message string   Commit message
      --stdin            Read commit message from stdin
```

#### issuegraph vc merge

Merge the specified branch into the current branch.

If there are merge conflicts, they will be reported. You can resolve
conflicts with --strategy.

Examples:
  issuegraph vc merge feature-xyz                    # Merge feature-xyz into current branch
  issuegraph vc merge feature-xyz --strategy ours    # Merge, preferring our changes on conflict
  issuegraph vc merge feature-xyz --strategy theirs  # Merge, preferring their changes on conflict

```
issuegraph vc merge <branch> [flags]
```

**Flags:**

```
      --strategy string   Conflict resolution strategy: 'ours' or 'theirs'
```

#### issuegraph vc status

Show the current branch, commit hash, and any uncommitted changes.

Examples:
  issuegraph vc status

```
issuegraph vc status
```

## Setup & Configuration:

### issuegraph bootstrap

Bootstrap sets up the issuegraph database without destroying existing data.
Unlike 'issuegraph init --force', bootstrap will never delete existing issues.

Bootstrap auto-detects the right action:
  • If sync.remote is configured: clones from the remote
  • If git origin has Dolt data (refs/dolt/data): clones from git and wires origin for future push/pull
  • If .issuegraph/backup/*.jsonl exists: restores from backup
  • If .issuegraph/issues.jsonl exists: imports from git-tracked JSONL
  • If no database exists: creates a fresh one
  • If database already exists: validates and reports status

This is the recommended command for:
  • Setting up issuegraph on a fresh clone
  • Recovering after moving to a new machine
  • Repairing a broken database configuration

Non-interactive mode (--non-interactive, --yes/-y, or ISSUEGRAPH_NON_INTERACTIVE=1):
  Skips the confirmation prompt before executing the bootstrap plan.
  Also auto-detected when stdin is not a terminal or CI=true is set.

Examples:
  issuegraph bootstrap              # Auto-detect and set up
  issuegraph bootstrap --dry-run    # Show what would be done
  issuegraph bootstrap --json       # Output plan as JSON
  issuegraph bootstrap --yes        # Skip confirmation prompt


```
issuegraph bootstrap [flags]
```

**Flags:**

```
      --dry-run           Show what would be done without doing it
      --non-interactive   Alias for --yes
  -y, --yes               Skip confirmation prompts (for CI/automation)
```

### issuegraph config

Manage configuration settings for external integrations and preferences.

Configuration is stored per-project in the issuegraph database and is version-control-friendly.

Common namespaces:
  - export.*          Auto-export settings (stored in config.yaml)
  - import.*          JSONL import settings (stored in config.yaml)
  - jira.*            Jira integration settings
  - linear.*          Linear integration settings
  - github.*          GitHub integration settings
  - custom.*          Custom integration settings
  - status.*          Issue status configuration
  - doctor.suppress.* Suppress specific issuegraph doctor warnings (GH#1095)

Auto-Export (config.yaml):
  Optional JSONL export to .issuegraph/issues.jsonl after write commands (throttled).
  Useful for viewers (bv), interchange, and issue-level migration; not a backup.
  It is not cross-machine sync; use issuegraph dolt push/pull with a Dolt remote.
  Disabled by default. Enable only for integrations that need fresh JSONL.
  Auto-staging is separate and disabled by default.

  Keys:
    export.auto       Enable/disable auto-export (default: false)
    export.path       Output filename relative to .issuegraph/ (default: issues.jsonl)
    export.interval   Minimum time between exports (default: 60s)
    export.git-add    Auto-stage the export file (default: false)

Auto-Import (config.yaml):
  Reads .issuegraph/issues.jsonl by default when a JSONL import path is implied.
  Use a relative filename/path so the import stays within the project .issuegraph/
  directory and remains portable across machines.

  Keys:
    import.path       Input filename relative to .issuegraph/ (default: issues.jsonl)

Custom Status States:
  You can define custom status states for multi-step pipelines using the
  status.custom config key. Statuses should be comma-separated.

  Example:
    issuegraph config set status.custom "awaiting_review,awaiting_testing,awaiting_docs"

  This enables issues to use statuses like 'awaiting_review' in addition to
  the built-in statuses (open, in_progress, blocked, deferred, closed).

Suppressing Doctor Warnings:
  Suppress specific issuegraph doctor warnings by check name slug:
    issuegraph config set doctor.suppress.pending-migrations true
    issuegraph config set doctor.suppress.git-hooks true
  Check names are converted to slugs: "Git Hooks" → "git-hooks".
  Only warnings are suppressed (errors and passing checks always show).
  To unsuppress: issuegraph config unset doctor.suppress.&lt;slug&gt;

Examples:
  issuegraph config set export.auto true                       # Enable auto-export for viewer integrations
  issuegraph config set export.path "issuegraph.jsonl"              # Custom export filename
  issuegraph config set import.path "issuegraph.jsonl"              # Custom import filename
  issuegraph config set export.git-add true                    # Also stage the export file
  issuegraph config set jira.url "https://company.atlassian.net"
  issuegraph config set jira.project "PROJ"
  issuegraph config set status.custom "awaiting_review,awaiting_testing"
  issuegraph config set doctor.suppress.pending-migrations true
  issuegraph config set dolt.debug true                        # Enable Dolt sql-server debug mode (loglevel=debug, --prof cpu)
  issuegraph config set dolt.local-only true                   # Skip wiring a Dolt sync remote during issuegraph init
  issuegraph config get export.auto
  issuegraph config list
  issuegraph config unset jira.url

```
issuegraph config
```

#### issuegraph config apply

Reconcile actual system state to match declared configuration.

Runs drift detection and then fixes any mismatches it finds:

  - hooks     Reinstall git hooks if missing or outdated
  - remote    Add/update Dolt origin remote to match federation.remote
  - server    Start Dolt server if dolt.shared-server is enabled

This command is idempotent — safe to run multiple times. Use --dry-run
to preview what would change without making modifications.

Examples:
  issuegraph config apply
  issuegraph config apply --dry-run
  issuegraph config apply --json

```
issuegraph config apply [flags]
```

**Flags:**

```
      --dry-run   Show what would change without making modifications
```

#### issuegraph config drift

Detect drift between declared configuration and actual system state.

This is a read-only diagnostic that answers "is my environment consistent
with my config?" — no mutations are performed.

Checks:
  - hooks     Git hooks installed and up-to-date
  - remote    Dolt remote matches federation.remote config
  - server    Server state matches dolt.shared-server config

Exit codes:
  0  No drift detected (all checks ok/info/skipped)
  1  Drift detected (at least one check has status "drift")

Examples:
  issuegraph config drift
  issuegraph config drift --json

```
issuegraph config drift
```

#### issuegraph config get

Get a configuration value

```
issuegraph config get <key>
```

#### issuegraph config list

List all configuration

```
issuegraph config list
```

#### issuegraph config set

Set a configuration value

```
issuegraph config set <key> <value> [flags]
```

**Flags:**

```
      --force-git-tracked   Allow writing secret keys to git-tracked config files (use with caution)
```

#### issuegraph config set-many

Set multiple configuration values at once with a single auto-commit and auto-push.

Each argument must be in key=value format. All values are validated before
any writes occur. This is faster and less noisy than separate 'issuegraph config set'
calls, especially in CI.

Examples:
  issuegraph config set-many ado.state_map.open=New ado.state_map.closed=Closed
  issuegraph config set-many jira.url=https://example.atlassian.net jira.project=PROJ

```
issuegraph config set-many <key=value>... [flags]
```

**Flags:**

```
      --force-git-tracked   Allow writing secret keys to git-tracked config files (use with caution)
```

#### issuegraph config show

Display a unified view of all effective configuration across all sources
with annotations showing where each value comes from.

Sources (by precedence for Viper-managed keys):
  - env          Environment variable (ISSUEGRAPH_* or ISSUEGRAPH_*)
  - config.yaml  Project config file (.issuegraph/config.yaml)
  - default      Built-in default value

Additional sources:
  - metadata     Connection settings from .issuegraph/metadata.json
  - database     Integration config stored in the Dolt database
  - git          Git config (e.g., issuegraph.role)

Examples:
  issuegraph config show
  issuegraph config show --json
  issuegraph config show --source config.yaml

```
issuegraph config show [flags]
```

**Flags:**

```
      --source string   Filter by source (e.g., config.yaml, env, default, metadata, database, git)
```

#### issuegraph config unset

Delete a configuration value

```
issuegraph config unset <key>
```

#### issuegraph config validate

Validate sync-related configuration settings.

Checks:
  - federation.sovereignty is valid (T1, T2, T3, T4, or empty)
  - federation.remote is set for Dolt sync
  - Remote URL format is valid (dolthub://, gs://, s3://, az://, file://)
  - routing.mode is valid (auto, maintainer, contributor, explicit)

	Examples:
	  issuegraph config validate
	  issuegraph config validate --json

```
issuegraph config validate
```

### issuegraph context

Show the effective backend identity information including repository paths,
backend configuration, and sync settings.

This command reads directly from config files and does not require the
database to be open, making it useful for diagnostics in degraded states.

Examples:
  issuegraph context           # Show context information
  issuegraph context --json    # Output in JSON format


```
issuegraph context
```

### issuegraph dolt

Configure and manage Dolt database settings and server lifecycle.

IssueGraph uses a dolt sql-server for all database operations. The server is
auto-started transparently when needed. Use these commands for explicit
control or diagnostics.

Server lifecycle:
  issuegraph dolt start        Start the Dolt server for this project
  issuegraph dolt stop         Stop the Dolt server for this project
  issuegraph dolt status       Show Dolt server status

Configuration:
  issuegraph dolt show         Show current Dolt configuration with connection test
  issuegraph dolt set &lt;k&gt; &lt;v&gt;  Set a configuration value
  issuegraph dolt test         Test server connection

Version control:
  issuegraph dolt commit       Commit pending changes
  issuegraph dolt push         Push commits to Dolt remote
  issuegraph dolt pull         Pull commits from Dolt remote

Remote management:
  issuegraph dolt remote add &lt;name&gt; &lt;url&gt;   Add a Dolt remote
  issuegraph dolt remote list                List configured remotes
  issuegraph dolt remote remove &lt;name&gt;       Remove a Dolt remote

Configuration keys for 'issuegraph dolt set':
  database  Database name (default: issue prefix or "issuegraph")
  host      Server host (default: 127.0.0.1)
  port      Server port (auto-detected; override with issuegraph dolt set port &lt;N&gt;)
  user      MySQL user (default: root)
  data-dir  Custom dolt data directory (absolute path; default: .issuegraph/dolt)

Flags for 'issuegraph dolt set':
  --update-config  Also write to config.yaml for team-wide defaults

Examples:
  issuegraph dolt set database myproject
  issuegraph dolt set host 192.168.1.100 --update-config
  issuegraph dolt set data-dir /home/user/.issuegraph-dolt/myproject
  issuegraph dolt test

```
issuegraph dolt
```

#### issuegraph dolt clean-databases

Identify and drop leftover test and agent databases that accumulate
on the shared Dolt server from interrupted test runs and terminated agents.

Stale database prefixes: testdb_*, doctest_*, doctortest_*, issuegraph_pt*, issuegraph_vr*, issuegraph_t*

These waste server memory and can degrade performance under concurrent load.
Use --dry-run to see what would be dropped without actually dropping.

```
issuegraph dolt clean-databases [flags]
```

**Flags:**

```
      --dry-run   Show what would be dropped without dropping
```

#### issuegraph dolt commit

Create a Dolt commit from any uncommitted changes in the working set.

This is the primary commit point for batch mode. When auto-commit is set to
"batch", changes accumulate in the working set across multiple issuegraph commands and
are committed together here with a descriptive summary message.

Also useful before push operations that require a clean working set, or when
auto-commit was off or changes were made externally.

For more options (--stdin, custom messages), see: issuegraph vc commit

```
issuegraph dolt commit [flags]
```

**Flags:**

```
  -m, --message string   Commit message (default: auto-generated)
```

#### issuegraph dolt killall

Find and kill orphan dolt sql-server processes not tracked by the
canonical PID file for the current repo's Dolt data directory.

Under an orchestrator, the canonical server lives at $GT_ROOT/.issuegraph/. Any other
dolt sql-server processes using that shared data directory are considered
orphans and will be killed.

In standalone mode, only dolt sql-server processes using the current
project's Dolt data directory are eligible for cleanup. Other projects'
servers are preserved.

```
issuegraph dolt killall
```

#### issuegraph dolt pull

Pull commits from the configured Dolt remote into the local database.

Requires a Dolt remote to be configured in the database directory.
For Hosted Dolt, set DOLT_REMOTE_USER and DOLT_REMOTE_PASSWORD environment
variables for authentication.

Use --remote to pull from a specific named remote instead of the default.
The remote must already exist (see 'issuegraph dolt remote add').

```
issuegraph dolt pull [flags]
```

**Flags:**

```
      --remote string   Pull from a specific named remote instead of the default
```

#### issuegraph dolt push

Push local Dolt commits to the configured remote.

Requires a Dolt remote to be configured in the database directory.
For Hosted Dolt, set DOLT_REMOTE_USER and DOLT_REMOTE_PASSWORD environment
variables for authentication.

Use --force to overwrite remote changes (e.g., when the remote has
uncommitted changes in its working set).

Use --remote to push to a specific named remote instead of the default.
The remote must already exist (see 'issuegraph dolt remote add').

```
issuegraph dolt push [flags]
```

**Flags:**

```
      --force           Force push (overwrite remote changes)
      --remote string   Push to a specific named remote instead of the default
```

#### issuegraph dolt remote

Manage Dolt remotes for push/pull replication.

Subcommands:
  add &lt;name&gt; &lt;url&gt;   Add a new remote
  list               List all configured remotes
  remove &lt;name&gt;      Remove a remote

```
issuegraph dolt remote
```

##### issuegraph dolt remote add

Add a Dolt remote

```
issuegraph dolt remote add <name> <url>
```

##### issuegraph dolt remote list

List configured Dolt remotes

```
issuegraph dolt remote list
```

##### issuegraph dolt remote remove

Remove a Dolt remote

```
issuegraph dolt remote remove <name>
```

#### issuegraph dolt set

Set a Dolt configuration value in metadata.json.

Keys:
  database  Database name (default: issue prefix or "issuegraph")
  host      Server host (default: 127.0.0.1)
  port      Server port (auto-detected; override with issuegraph dolt set port &lt;N&gt;)
  user      MySQL user (default: root)
  data-dir  Custom dolt data directory (absolute path; default: .issuegraph/dolt)

Use --update-config to also write to config.yaml for team-wide defaults.

Examples:
  issuegraph dolt set database myproject
  issuegraph dolt set host 192.168.1.100
  issuegraph dolt set port 3307 --update-config
  issuegraph dolt set data-dir /home/user/.issuegraph-dolt/myproject

```
issuegraph dolt set <key> <value> [flags]
```

**Flags:**

```
      --update-config   Also write to config.yaml for team-wide defaults
```

#### issuegraph dolt show

Show current Dolt configuration with connection status

```
issuegraph dolt show
```

#### issuegraph dolt start

Start a dolt sql-server for the current issuegraph project.

The server runs in the background on a per-project port derived from the
project path. PID and logs are stored in .issuegraph/.

The server auto-starts transparently when needed, so manual start is rarely
required. Use this command for explicit control or diagnostics.

```
issuegraph dolt start
```

#### issuegraph dolt status

Show the status of the Dolt engine for the current project.

In embedded mode, reports that the Dolt engine runs in-process and shows
the on-disk data directory. For issuegraph-managed (local) servers, displays
PID, port, and data directory from the local PID file. For externally-
managed servers — either a remote dolt_server_host or a local server
managed outside issuegraph (dolt.auto-start: false, e.g. an orchestrator-shared
sql-server) — pings the configured endpoint via SQL and reports
reachability, server version, and database.

```
issuegraph dolt status
```

#### issuegraph dolt stop

Stop the dolt sql-server managed by issuegraph for the current project.

This sends a graceful shutdown signal. The server will restart automatically
on the next issuegraph command unless auto-start is disabled.

```
issuegraph dolt stop [flags]
```

**Flags:**

```
      --force   Force stop the server
```

#### issuegraph dolt test

Test the connection to the configured Dolt server.

This verifies that:
  1. The server is reachable at the configured host:port
  2. The connection can be established

Use this before switching to server mode to ensure the server is running.

```
issuegraph dolt test
```

### issuegraph forget

Remove a memory by its key.

Use 'issuegraph memories' to see available keys.

Examples:
  issuegraph forget dolt-phantoms
  issuegraph forget auth-jwt

```
issuegraph forget <key>
```

### issuegraph hooks

Install, uninstall, or list git hooks for issuegraph integration.

The hooks provide:
- pre-commit: Run chained hooks before commit
- post-merge: Run chained hooks after pull/merge
- pre-push: Run chained hooks before push
- post-checkout: Run chained hooks after branch checkout
- prepare-commit-msg: Add agent identity trailers for forensics

```
issuegraph hooks
```

#### issuegraph hooks install

Install git hooks for issuegraph integration.

By default, hooks are installed to .git/hooks/ in the current repository.
Use --issuegraph to install to .issuegraph/hooks/ (recommended for Dolt backend).
Use --shared to install to a versioned directory (.issuegraph-hooks/) that can be
committed to git and shared with team members.

Hooks use section markers to coexist with existing hooks — any user content
outside the markers is preserved across installs and upgrades.

Installed hooks:
  - pre-commit: Run chained hooks before commit
  - post-merge: Run chained hooks after pull/merge
  - pre-push: Run chained hooks before push
  - post-checkout: Run chained hooks after branch checkout
  - prepare-commit-msg: Add agent identity trailers (for orchestrator agents)

```
issuegraph hooks install [flags]
```

**Flags:**

```
      --issuegraph    Install hooks to .issuegraph/hooks/ (recommended for Dolt backend)
      --chain    Chain with existing hooks (run them before issuegraph hooks)
      --force    Overwrite existing hooks without backup
      --shared   Install hooks to .issuegraph-hooks/ (versioned) instead of .git/hooks/
```

#### issuegraph hooks list

Show the status of issuegraph git hooks (installed, outdated, missing).

```
issuegraph hooks list
```

#### issuegraph hooks run

Execute the logic for a git hook. This command is typically called by
thin shim scripts installed in .git/hooks/.

Supported hooks:
  - pre-commit: Run chained hooks before commit
  - post-merge: Run chained hooks after pull/merge
  - pre-push: Run chained hooks before push
  - post-checkout: Run chained hooks after branch checkout
  - prepare-commit-msg: Add agent identity trailers for forensics

The thin shim pattern ensures hook logic is always in sync with the
installed issuegraph version - upgrading issuegraph automatically updates hook behavior.

```
issuegraph hooks run <hook-name> [args...]
```

#### issuegraph hooks uninstall

Remove issuegraph git hooks from .git/hooks/ directory.

```
issuegraph hooks uninstall
```

### issuegraph human

Display a focused help menu showing only the most common commands.

issuegraph has 70+ commands - many for AI agents, integrations, and advanced workflows.
This command shows the ~15 essential commands that human users need most often.

For the full command list, run: issuegraph --help

SUBCOMMANDS:
  human list              List all human-needed issuegraph (issues with 'human' label)
  human respond &lt;id&gt;      Respond to a human-needed issue (adds comment and closes)
  human dismiss &lt;id&gt;      Dismiss a human-needed issue permanently
  human stats             Show summary statistics for human-needed issuegraph

```
issuegraph human
```

#### issuegraph human dismiss

Dismiss a human-needed issue permanently without responding.

The issue is closed with a "Dismissed" reason and optional note.

Examples:
  issuegraph human dismiss issuegraph-123
  issuegraph human dismiss issuegraph-123 --reason "No longer applicable"

```
issuegraph human dismiss <issue-id> [flags]
```

**Flags:**

```
      --reason string   Reason for dismissal (optional)
```

#### issuegraph human list

List all issues labeled with 'human' tag.

These are issues that require human intervention or input.

Examples:
  issuegraph human list
  issuegraph human list --status=open
  issuegraph human list --json

```
issuegraph human list [flags]
```

**Flags:**

```
  -s, --status string   Filter by status (open, closed, etc.)
```

#### issuegraph human respond

Respond to a human-needed issue by adding a comment and closing it.

The response is added as a comment and the issue is closed with reason "Responded".

Examples:
  issuegraph human respond issuegraph-123 --response "Use OAuth2 for authentication"
  issuegraph human respond issuegraph-123 -r "Approved, proceed with implementation"

```
issuegraph human respond <issue-id> [flags]
```

**Flags:**

```
  -r, --response string   Response text (required)
```

#### issuegraph human stats

Display summary statistics for human-needed issuegraph.

Shows counts for total, pending (open), responded (closed without dismiss),
and dismissed issuegraph.

Example:
  issuegraph human stats

```
issuegraph human stats
```

### issuegraph info

Display information about the current database.

This command helps debug issues where issuegraph is using an unexpected database. It shows:
  - The absolute path to the database file
  - Database statistics (issue count)
  - Schema information (with --schema flag)
  - What's new in recent versions (with --whats-new flag)

Examples:
  issuegraph info
  issuegraph info --json
  issuegraph info --schema --json
  issuegraph info --whats-new
  issuegraph info --whats-new --json
  issuegraph info --thanks

```
issuegraph info [flags]
```

**Flags:**

```
      --json        Output in JSON format
      --schema      Include schema information in output
      --thanks      Show thank you page for contributors
      --whats-new   Show agent-relevant changes from recent versions
```

### issuegraph init

Initialize issuegraph in the current directory by creating a .issuegraph/ directory
and Dolt database. Optionally specify a custom issue prefix.

Dolt is the default (and only supported) storage backend. The legacy SQLite
backend has been removed. Use --backend=sqlite to see migration instructions.

Use --database to specify an existing server database name, overriding the
default prefix-based naming. This is useful when an external tool (e.g. an orchestrator)
has already created the database.

With --stealth: configures per-repository git settings for invisible issuegraph usage:
  • .git/info/exclude to prevent issuegraph files from being committed
  Perfect for personal use without affecting repo collaborators.
  To set up a specific AI tool, run: issuegraph setup &lt;claude|cursor|aider|...&gt; --stealth

By default, issuegraph uses an embedded Dolt engine (no external server needed).
Pass --server to use an external dolt sql-server instead. In server mode,
set connection details with --server-host, --server-port, and --server-user.
Password should be set via ISSUEGRAPH_DOLT_PASSWORD environment variable.

Auto-export is optional. When enabled, issuegraph exports issues to
.issuegraph/issues.jsonl after write commands (throttled to once per 60s). This is
for viewers (bv), interchange, and issue-level migration; not backup.
Cross-machine sync and backups use Dolt remotes/backups, not JSONL import/export.
To enable: issuegraph config set export.auto true

Non-interactive mode (--non-interactive or ISSUEGRAPH_NON_INTERACTIVE=1):
  Skips all interactive prompts, using sensible defaults:
  • Role defaults to "maintainer" (override with --role)
  • Fork exclude auto-configured when fork detected
  • Auto-export left at default (disabled)
  • --contributor and --team flags are rejected (wizards require interaction)
  Also auto-detected when stdin is not a terminal or CI=true is set.

```
issuegraph init [flags]
```

**Flags:**

```
      --agents-file string                             Custom filename for agent instructions (default: AGENTS.md)
      --agents-profile string                          AGENTS.md profile: 'minimal' (default, pointer to issuegraph prime) or 'full' (complete command reference)
      --agents-template string                         Path to custom AGENTS.md template (overrides embedded default)
      --backend string                                 Storage backend (default: dolt). --backend=sqlite prints deprecation notice.
      --contributor                                    Run OSS contributor setup wizard
      --database string                                Use existing server database name (overrides prefix-based naming)
      --debug                                          Run the managed Dolt sql-server with --loglevel=debug and CPU profiling (--prof cpu). Persisted to config.yaml as dolt.debug. No effect on externally-managed servers.
      --destroy-token string                           Explicit confirmation token for destructive re-init in non-interactive mode (format: 'DESTROY-<prefix>')
      --discard-remote                                 Authorize discarding the configured remote's Dolt history when re-initializing. Requires --destroy-token in non-interactive mode; see 'issuegraph help init-safety'.
      --external                                       Server is externally managed (skip server startup); use with --shared-server or --server
      --force                                          Deprecated alias for --reinit-local. Bypasses only the LOCAL data-safety guard; does NOT authorize remote divergence (see 'issuegraph help init-safety').
      --from-jsonl                                     Import issues from configured import.path; refuses remote history unless --discard-remote authorizes replacement
      --init-if-missing                                If the workspace is already initialized, skip init and exit 0 instead of failing (idempotent init for scaffolds)
      --non-interactive                                Skip all interactive prompts (auto-detected in CI or non-TTY environments)
  -p, --prefix string                                  Issue prefix (default: current directory name)
      --proxied-server                                 [EXPERIMENTAL] Use a per-workspace proxied dolt sql-server (proxy + child dolt) rooted at .issuegraph/proxieddb
      --proxied-server-config-path string              [EXPERIMENTAL] Absolute path to an existing dolt sql-server YAML config (proxied-server mode only). When set, issuegraph uses this file instead of auto-generating one. Relative paths are rejected.
      --proxied-server-external-host string            [EXPERIMENTAL] Hostname or IP of an externally-managed dolt sql-server the proxy should front (proxied-server mode only). Mutually exclusive with --proxied-server-external-socket-path.
      --proxied-server-external-keep-alive duration    [EXPERIMENTAL] TCP keepalive period for the proxy→external connection. Zero uses the package default (30s).
      --proxied-server-external-port int               [EXPERIMENTAL] TCP port of the externally-managed dolt sql-server (proxied-server mode only). Required when --proxied-server-external-host is set.
      --proxied-server-external-socket-path string     [EXPERIMENTAL] Absolute unix socket path of the externally-managed dolt sql-server (proxied-server mode only). Mutually exclusive with --proxied-server-external-host. Relative paths are rejected.
      --proxied-server-external-tls                    [EXPERIMENTAL] Require TLS when connecting to the externally-managed dolt sql-server (proxied-server mode only).
      --proxied-server-external-tls-cert-path string   [EXPERIMENTAL] Absolute path to a client TLS certificate (for mTLS to the externally-managed dolt sql-server). Must be paired with --proxied-server-external-tls-key-path. Relative paths are rejected.
      --proxied-server-external-tls-key-path string    [EXPERIMENTAL] Absolute path to the client TLS private key (for mTLS to the externally-managed dolt sql-server). Must be paired with --proxied-server-external-tls-cert-path. Relative paths are rejected.
      --proxied-server-external-user string            [EXPERIMENTAL] MySQL user for the externally-managed dolt sql-server (proxied-server mode only). Defaults to "root" when empty. Password is read at runtime from $ISSUEGRAPH_PROXIED_SERVER_EXTERNAL_PASSWORD and is never persisted to disk.
      --proxied-server-log-path string                 [EXPERIMENTAL] Absolute path to the proxied dolt sql-server log file (proxied-server mode only). Default: <issuegraphDir>/proxieddb/server.log. Relative paths are rejected.
      --proxied-server-root-path string                [EXPERIMENTAL] Absolute directory holding the proxied dolt sql-server's lockfiles, pidfiles, and child .dolt repository (proxied-server mode only). Default: <issuegraphDir>/proxieddb. May not exist yet — issuegraph will create it. Relative paths are rejected.
  -q, --quiet                                          Suppress output (quiet mode)
      --reinit-local                                   Re-initialize local .issuegraph/ over existing local data. Does NOT authorize remote divergence; see --discard-remote.
      --remote string                                  Dolt remote URL to clone from and persist as sync.remote
      --role string                                    Set issuegraph role without prompting: "maintainer" or "contributor"
      --server                                         Use external dolt sql-server instead of embedded engine
      --server-host string                             Dolt server host (default: 127.0.0.1)
      --server-port int                                Dolt server port (default: 3307)
      --server-socket string                           Unix domain socket path (overrides host/port)
      --server-user string                             Dolt server MySQL user (default: root)
      --setup-exclude                                  Configure .git/info/exclude to keep issuegraph files local (for forks)
      --shared-server                                  Enable shared Dolt server mode (all projects share one server at ~/.issuegraph/shared-server/)
      --skip-agents                                    Skip AGENTS.md and Claude/Codex setup generation
      --skip-hooks                                     Skip git hooks installation
      --stealth                                        Enable stealth mode: global gitattributes and gitignore, no local repo tracking
      --team                                           Run team workflow setup wizard
```

### issuegraph kv

Commands for working with the issuegraph key-value store.

The key-value store is useful for storing flags, environment variables,
or other user-defined data that persists across sessions.

Examples:
  issuegraph kv set mykey myvalue    # Set a value
  issuegraph kv get mykey            # Get a value
  issuegraph kv clear mykey          # Delete a key
  issuegraph kv list                 # List all key-value pairs

```
issuegraph kv
```

#### issuegraph kv clear

Delete a key from the issuegraph key-value store.

Examples:
  issuegraph kv clear feature_flag
  issuegraph kv clear api_endpoint

```
issuegraph kv clear <key>
```

#### issuegraph kv get

Get a value from the issuegraph key-value store.

Examples:
  issuegraph kv get feature_flag
  issuegraph kv get api_endpoint

```
issuegraph kv get <key>
```

#### issuegraph kv list

List all key-value pairs in the issuegraph key-value store.

Examples:
  issuegraph kv list
  issuegraph kv list --json

```
issuegraph kv list
```

#### issuegraph kv set

Set a key-value pair in the issuegraph key-value store.

This is useful for storing flags, environment variables, or other
user-defined data that persists across sessions.

Examples:
  issuegraph kv set feature_flag true
  issuegraph kv set api_endpoint https://api.example.com
  issuegraph kv set max_retries 3

```
issuegraph kv set <key> <value>
```

### issuegraph memories

List all memories, or search by keyword.

Examples:
  issuegraph memories              # list all memories
  issuegraph memories dolt         # search for memories about dolt
  issuegraph memories "race flag"  # search for a phrase

```
issuegraph memories [search]
```

### issuegraph onboard

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
issuegraph onboard
```

### issuegraph prime

Output essential IssueGraph workflow context in AI-optimized markdown format.

Automatically detects if MCP server is active and adapts output:
- MCP mode: Brief workflow reminders (~50 tokens)
- CLI mode: Full command reference (~1-2k tokens)

Designed for Claude Code, Gemini CLI, and Codex SessionStart hooks to prevent
agents from forgetting issuegraph workflow after context compaction.

Config options:
- no-git-ops: When true, outputs stealth mode (no git commands in session close protocol).
  Set via: issuegraph config set no-git-ops true
  Useful when you want to control when commits happen manually.

	Workflow customization:
	- Place a .issuegraph/PRIME.md file in the local clone or resolved workspace to override the default output entirely.
	- Use --export to dump the default content for customization.
	- Use --memories-only for hook contexts that should inject only persistent memories.

```
issuegraph prime [flags]
```

**Flags:**

```
      --export          Output default content (ignores PRIME.md override)
      --full            Force full CLI output (ignore MCP detection)
      --hook-json       Wrap output in the SessionStart hook JSON envelope (Claude Code, Gemini CLI, Codex)
      --mcp             Force MCP mode (minimal output)
      --memories-only   Output only persistent memories for compact hook contexts
      --stealth         Stealth mode (no git operations, flush only)
```

### issuegraph quickstart

Display a quick start guide showing common issuegraph workflows and patterns.

```
issuegraph quickstart
```

### issuegraph recall

Retrieve the full content of a memory by its key.

Examples:
  issuegraph recall dolt-phantoms
  issuegraph recall auth-jwt

```
issuegraph recall <key>
```

### issuegraph remember

Store a memory that persists across sessions and account rotations.

Memories are injected at prime time (issuegraph prime) so you have them
in every session without manual loading.

The positional arg is the memory CONTENT (the key is auto-generated from it
unless --key is given). As a convenience, if the arg is a bare key naming an
existing memory, it is RECALLED instead of stored (same as 'issuegraph recall');
a bare key naming nothing is refused. Use --key to store slug-like content.

Examples:
  issuegraph remember "always run tests with -race flag"
  issuegraph remember "Dolt phantom DBs hide in three places" --key dolt-phantoms
  issuegraph remember "auth module uses JWT not sessions" --key auth-jwt
  issuegraph remember dolt-phantoms        # bare existing key: reads it (= issuegraph recall)

```
issuegraph remember "<insight>" [flags]
```

**Flags:**

```
      --key string   Explicit key for the memory (auto-generated from content if not set). If a memory with this key already exists, it will be updated in place
```

### issuegraph setup

Setup integration files for AI editors and coding assistants.

Recipes define where issuegraph workflow instructions are written. Built-in recipes
include cursor, claude, copilot, gemini, aider, factory, codex, mux, opencode, junie, windsurf, cody, and kilocode.

Examples:
  issuegraph setup cursor          # Install Cursor IDE integration
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
      --global          Install globally (claude/codex/mux; writes to ~/.claude/settings.json, $CODEX_HOME/AGENTS.md or ~/.codex/AGENTS.md, or ~/.mux/AGENTS.md)
      --list            List all available recipes
  -o, --output string   Write template to custom path
      --print           Print the template to stdout
      --project         Install for this project only (gemini/mux)
      --remove          Remove the integration
      --stealth         Use stealth mode (claude/gemini)
```

### issuegraph where

Show the active issuegraph database location, including redirect information.

	This command is useful for debugging when using redirects, to understand
	which issuegraph workspace is actually being used.

Examples:
  issuegraph where           # Show active issuegraph location
  issuegraph where --json    # Output in JSON format


```
issuegraph where
```

## Maintenance:

### issuegraph batch

Run multiple write operations in a single database transaction.

Commands are read from stdin (one per line) or from a file via -f/--file.
All operations execute inside a single dolt transaction: on any error the
whole batch is rolled back, otherwise it is committed with one DOLT_COMMIT.

This is intended for shell scripts that currently invoke 'issuegraph' many times in
a loop, which causes severe write amplification on a dolt sql-server backed
by btrfs+compression. Batching collapses N invocations into one transaction
and one dolt commit.

Grammar (one command per line):
  close &lt;id&gt; [reason...]
  update &lt;id&gt; &lt;key&gt;=&lt;value&gt; [&lt;key&gt;=&lt;value&gt; ...]
  create &lt;type&gt; &lt;priority&gt; &lt;title...&gt;
  dep add &lt;from-id&gt; &lt;to-id&gt; [type]
  dep remove &lt;from-id&gt; &lt;to-id&gt;
  #comment  (blank lines and '# ...' comments are ignored)

Supported 'update' keys: status, priority, title, assignee
Supported dependency types: see 'issuegraph dep add --help' (default: blocks)

Tokens are whitespace-separated. Double-quoted strings ("like this") may
contain spaces; use \" to embed a quote and \\ for a backslash.

Examples:
  # From a pipe
  issuegraph list --status stale -q | awk '&#123;print "close",$1," stale"&#125;' | issuegraph batch

  # From a file
  issuegraph batch -f operations.txt

  # Inline
  printf 'close issuegraph-1 done\nupdate issuegraph-2 status=in_progress\n' | issuegraph batch

On success, exits 0 and prints a summary (or JSON with --json). On any error,
rolls back the entire transaction and exits non-zero with the failing line.

NOTE: This is a narrow subset. Commands like 'show', 'list', 'ready', 'sync',
complex create flows, or any flag not listed above are NOT accepted. Use
normal 'issuegraph' subcommands for interactive/read operations.

```
issuegraph batch [flags]
```

**Flags:**

```
      --dry-run          Parse input and echo commands without executing
  -f, --file string      Read commands from file instead of stdin
  -m, --message string   DOLT_COMMIT message (default: 'issuegraph: batch N ops by <actor>')
```

### issuegraph compact

Squash Dolt commits older than N days into a single commit.

Recent commits (within the retention window) are preserved via cherry-pick.
This reduces Dolt storage overhead from auto-commit history while keeping
recent change tracking intact.

For semantic issue compaction (summarizing closed issues), use 'issuegraph admin compact'.
For full history squash, use 'issuegraph flatten'.

How it works:
  1. Identifies commits older than --days threshold
  2. Creates a squashed base commit from all old history
  3. Cherry-picks recent commits on top
  4. Swaps main branch to the compacted version
  5. Runs Dolt GC to reclaim space

Examples:
  issuegraph compact --dry-run               # Preview: show commit breakdown
  issuegraph compact --force                 # Squash commits older than 30 days
  issuegraph compact --days 7 --force        # Keep only last 7 days of history
  issuegraph compact --days 90 --force       # Conservative: squash 90+ day old commits

```
issuegraph compact [flags]
```

**Flags:**

```
      --days int   Keep commits newer than N days (default 30)
      --dry-run    Preview without making changes
  -f, --force      Confirm commit squash
```

### issuegraph doctor

Sanity check the issuegraph installation for the current directory or specified path.

This command checks:
  - If .issuegraph/ directory exists
  - Database version and migration status
  - Schema compatibility (all required tables and columns present)
  - Whether using hash-based vs sequential IDs
  - If CLI version is current (checks GitHub releases)
  - If Claude plugin is current (when running in Claude Code)
  - File permissions
  - Circular dependencies
  - Git hooks (pre-commit, post-merge, pre-push)
  - .issuegraph/.gitignore up to date
  - Metadata.json version tracking (LastBdVersion field)

Performance Mode (--perf):
  Run performance diagnostics on your database:
  - Times key operations (issuegraph ready, issuegraph list, issuegraph show, etc.)
  - Collects system info (OS, arch, SQLite version, database stats)
  - Generates CPU profile for analysis
  - Outputs shareable report for bug reports

Export Mode (--output):
  Save diagnostics to a JSON file for historical analysis and bug reporting.
  Includes timestamp and platform info for tracking intermittent issues.

Specific Check Mode (--check):
  Run a specific check in detail. Available checks:
  - artifacts: Detect and optionally clean issuegraph classic artifacts
    (stale JSONL, SQLite files, cruft .issuegraph dirs). Use with --clean.
  - conventions: Check for convention drift (lint warnings, stale
    issues, orphaned issues). Advisory only - warns, never blocks.
  - pollution: Detect and optionally clean test issues from database
  - validate: Run focused data-integrity checks (duplicates, orphaned
    deps, test pollution, git conflicts). Use with --fix to auto-repair.

Deep Validation Mode (--deep):
  Validate full graph integrity. May be slow on large databases.
  Additional checks:
  - Parent consistency: All parent-child deps point to existing issues
  - Dependency integrity: All deps reference valid issues
  - Epic completeness: Find epics ready to close (all children closed)
  - Agent issue integrity: Agent issuegraph have valid state values
  - Mail thread integrity: Thread IDs reference existing issues
  - Molecule integrity: Molecules have valid parent-child structures

Server Mode (--server):
  Run health checks for Dolt server mode connections (issuegraph-dolt.2.3):
  - Server reachable: Can connect to configured host:port?
  - Dolt version: Is it a Dolt server (not vanilla MySQL)?
  - Database exists: Does the 'issuegraph' database exist?
  - Schema compatible: Can query issuegraph tables?
  - Connection pool: Pool health metrics

Migration Validation Mode (--migration):
  Run Dolt migration validation checks with machine-parseable output.
  Use --migration=pre before migration to verify readiness:
  - JSONL file exists and is valid (parseable, no corruption)
  - All JSONL issues are present in SQLite (or explains discrepancies)
  - No blocking issues prevent migration
  Use --migration=post after migration to verify completion:
  - Dolt database exists and is healthy
  - All issues from JSONL are present in Dolt
  - No data was lost during migration
  - Dolt database has no locks or uncommitted changes
  Combine with --json for machine-parseable output for automation.

Agent Mode (--agent):
  Output diagnostics designed for AI agent consumption. Instead of terse
  pass/fail messages, each issue includes:
  - Observed state: what the system actually looks like
  - Expected state: what it should look like
  - Explanation: full prose context about the issue and why it matters
  - Commands: exact remediation commands to run
  - Source files: where in the codebase to investigate further
  - Severity: blocking (prevents operation), degraded (partial function),
    or advisory (informational only)
  ZFC-compliant: Go observes and reports, the agent decides and acts.
  Combine with --json for structured agent-facing output.

Suppressing Warnings:
  Suppress specific warnings by setting doctor.suppress.&lt;check-slug&gt; config:
    issuegraph config set doctor.suppress.pending-migrations true
    issuegraph config set doctor.suppress.git-hooks true
  Check names are converted to slugs: "Git Hooks" → "git-hooks".
  Only warnings are suppressed; errors and passing checks always show.
  To unsuppress: issuegraph config unset doctor.suppress.&lt;slug&gt;

Examples:
  issuegraph doctor              # Check current directory
  issuegraph doctor /path/to/repo # Check specific repository
  issuegraph doctor --json       # Machine-readable output
  issuegraph doctor --agent      # Agent-facing diagnostic output
  issuegraph doctor --agent --json  # Structured agent diagnostics (JSON)
  issuegraph doctor --fix        # Automatically fix issues (with confirmation)
  issuegraph doctor --fix --yes  # Automatically fix issues (no confirmation)
  issuegraph doctor --fix -i     # Confirm each fix individually
  issuegraph doctor --fix --fix-child-parent  # Also fix child→parent deps (opt-in)
  issuegraph doctor --fix --force # Force repair even when database can't be opened
  issuegraph doctor --fix --source=jsonl # Rebuild database from a JSONL export
  issuegraph doctor --dry-run    # Preview what --fix would do without making changes
  issuegraph doctor --perf       # Performance diagnostics
  issuegraph doctor --output diagnostics.json  # Export diagnostics to file
  issuegraph doctor --check=artifacts           # Show classic artifacts (JSONL, SQLite, cruft dirs)
  issuegraph doctor --check=artifacts --clean  # Delete safe-to-delete artifacts (with confirmation)
  issuegraph doctor --check=conventions        # Convention drift check (lint, stale, orphans)
  issuegraph doctor --check=pollution          # Show potential test issues
  issuegraph doctor --check=pollution --clean  # Delete test issues (with confirmation)
  issuegraph doctor --check=validate         # Data-integrity checks only
  issuegraph doctor --check=validate --fix   # Auto-fix data-integrity issues
  issuegraph doctor --deep             # Full graph integrity validation
  issuegraph doctor --server           # Dolt server mode health checks
  issuegraph doctor --migration=pre    # Validate readiness for Dolt migration
  issuegraph doctor --migration=post   # Validate Dolt migration completed
  issuegraph doctor --migration=pre --json  # Machine-parseable migration validation

```
issuegraph doctor [path] [flags]
```

**Flags:**

```
      --agent                                   Agent-facing diagnostic mode: rich context for AI agents (ZFC-compliant)
      --check string                            Run specific check in detail (e.g., 'pollution')
      --check-health                            Quick health check for git hooks (silent on success)
      --clean                                   For pollution check: delete detected test issues
      --deep                                    Validate full graph integrity
      --dry-run                                 Preview fixes without making changes
      --fix                                     Automatically fix issues where possible
      --fix-child-parent                        Remove child→parent dependencies (opt-in)
  -i, --interactive                             Confirm each fix individually
      --migration string                        Run Dolt migration validation: 'pre' (before migration) or 'post' (after migration)
      --orchestrator                            Running in orchestrator multi-workspace mode (routes.jsonl is expected, higher duplicate tolerance)
      --orchestrator-duplicates-threshold int   Duplicate tolerance threshold for orchestrator mode (wisps are ephemeral) (default 1000)
  -o, --output string                           Export diagnostics to JSON file
      --perf                                    Run performance diagnostics and generate CPU profile
      --server                                  Run Dolt server mode health checks (connectivity, version, schema)
  -v, --verbose                                 Show all checks (default shows only warnings/errors)
  -y, --yes                                     Skip confirmation prompt (for non-interactive use)
```

### issuegraph flatten

Nuclear option: squash ALL Dolt commit history into a single commit.

This uses the Tim Sehn recipe:
  1. Create a new branch from the current state
  2. Soft-reset to the initial commit (preserving all data)
  3. Commit everything as a single snapshot
  4. Swap main branch to the new flattened branch
  5. Run Dolt GC to reclaim space from old history

This is irreversible — all commit history is lost. The resulting database
has exactly one commit containing all current data.

Use this when:
  - Your .issuegraph/dolt directory has grown very large
  - You don't need commit-level history (time travel)
  - You want to start fresh with minimal storage

Examples:
  issuegraph flatten --dry-run               # Preview: show commit count and disk usage
  issuegraph flatten --force                 # Actually squash all history
  issuegraph flatten --force --json          # JSON output

```
issuegraph flatten [flags]
```

**Flags:**

```
      --dry-run   Preview without making changes
  -f, --force     Confirm irreversible history squash
```

### issuegraph gc

Full lifecycle garbage collection for standalone IssueGraph databases.

Runs three phases in sequence:
  1. DECAY   — Delete closed issues older than N days (default 90)
  2. COMPACT — Squash old Dolt commits into fewer commits (issuegraph compact)
  3. GC      — Run Dolt garbage collection to reclaim disk space

Each phase can be skipped individually. Use --dry-run to preview all phases
without making changes.

Examples:
  issuegraph gc                              # Full GC with defaults (90 day decay)
  issuegraph gc --dry-run                    # Preview what would happen
  issuegraph gc --older-than 30              # Decay issues closed 30+ days ago
  issuegraph gc --skip-decay                 # Skip issue deletion, just compact+GC
  issuegraph gc --skip-dolt                  # Skip Dolt GC, just decay+compact
  issuegraph gc --force                      # Skip confirmation prompt

```
issuegraph gc [flags]
```

**Flags:**

```
      --dry-run          Preview without making changes
  -f, --force            Skip confirmation prompts
      --older-than int   Delete closed issues older than N days (default 90)
      --skip-decay       Skip issue deletion phase
      --skip-dolt        Skip Dolt garbage collection phase
```

### issuegraph migrate

Database migration and data transformation commands.

Without subcommand, checks and updates database metadata to current version.

Subcommands:
  hooks       Plan git hook migration to marker-managed format
  issues      Move issues between repositories
  schema      Apply pending schema migrations (idempotent)
  sync        Set up sync.branch workflow for multi-clone setups


```
issuegraph migrate [flags]
```

**Flags:**

```
      --dry-run          Show what would be done without making changes
      --inspect          Show migration plan and database state for AI agent analysis
      --json             Output migration statistics in JSON format
      --update-repo-id   Update repository ID (use after changing git remote)
      --yes              Auto-confirm prompts
```

#### issuegraph migrate hooks

Analyze git hook files and sidecar artifacts for migration to marker-managed format.

Modes:
  --dry-run  Preview migration operations without changing files
  --apply    Apply migration operations

Examples:
  issuegraph migrate hooks --dry-run
  issuegraph migrate hooks --apply
  issuegraph migrate hooks --apply --yes
  issuegraph migrate hooks --dry-run --json

```
issuegraph migrate hooks [path] [flags]
```

**Flags:**

```
      --apply     Apply planned hook migration changes
      --dry-run   Show what would be done without making changes
      --json      Output in JSON format
      --yes       Skip confirmation prompt for --apply
```

#### issuegraph migrate issues

Move issues from one source repository to another with filtering and dependency preservation.

This command updates the source_repo field for selected issues, allowing you to:
- Move contributor planning issues to upstream repository
- Reorganize issues across multi-phase repositories
- Consolidate issues from multiple repos

Examples:
  # Preview migration from planning repo to current repo
  issuegraph migrate-issues --from ~/.issuegraph-planning --to . --dry-run

  # Move all open P1 bugs
  issuegraph migrate-issues --from ~/repo1 --to ~/repo2 --priority 1 --type bug --status open

  # Move specific issues with their dependencies
  issuegraph migrate-issues --from . --to ~/archive --id issuegraph-abc --id issuegraph-xyz --include closure

  # Move issues with label filter
  issuegraph migrate-issues --from . --to ~/feature-work --label frontend --label urgent

```
issuegraph migrate issues [flags]
```

**Flags:**

```
      --dry-run            Show plan without making changes
      --from string        Source repository (required)
      --id strings         Specific issue IDs to migrate (can specify multiple)
      --ids-file string    File containing issue IDs (one per line)
      --include string     Include dependencies: none/upstream/downstream/closure (default "none")
      --label strings      Filter by labels (can specify multiple)
      --priority int       Filter by priority (0-4) (default -1)
      --status string      Filter by status (open/closed/all)
      --strict             Fail on orphaned dependencies or missing repos
      --to string          Destination repository (required)
      --type string        Filter by issue type (bug/feature/task/epic/chore/decision)
      --within-from-only   Only include dependencies from source repo (default true)
      --yes                Skip confirmation prompt
```

#### issuegraph migrate schema

Apply pending schema migrations idempotently.

Schema migrations also run automatically on store open, so this subcommand
is typically a no-op. It exists to make migration explicit and observable
in CI, release gates, and recovery scenarios.

Example:
  issuegraph migrate schema
  issuegraph migrate schema --json

```
issuegraph migrate schema [flags]
```

**Flags:**

```
      --json   Output in JSON format
```

#### issuegraph migrate sync

Configure separate branch workflow for multi-clone setups.

This sets the sync.branch config value so that issue data is committed
to a dedicated branch, keeping your main branch clean.

Example:
  issuegraph migrate sync issuegraph-sync

```
issuegraph migrate sync <branch> [flags]
```

**Flags:**

```
      --dry-run   Show what would be done without making changes
      --json      Output in JSON format
```

### issuegraph ping

Lightweight health check that confirms issuegraph can reach its database.

Steps:
  1. Resolve the .issuegraph workspace
  2. Open the store (embedded or server)
  3. Run a trivial query (issue count)
  4. Report timing

Exit 0 on success, exit 1 on failure.

Examples:
  issuegraph ping              # Quick connectivity check
  issuegraph ping --json       # Structured output for automation

```
issuegraph ping
```

### issuegraph preflight

Display a checklist of common pre-PR checks for contributors.

This command helps catch common issues before pushing to CI:
- Tests not run locally
- Lint errors
- Unformatted Go files
- .issuegraph/issues.jsonl pollution
- Stale nix vendorHash
- Version mismatches

Examples:
  issuegraph preflight              # Show checklist
  issuegraph preflight --check      # Run checks automatically
  issuegraph preflight --check --json  # JSON output for programmatic use
  issuegraph preflight --check --skip-lint  # Explicitly skip lint check


```
issuegraph preflight [flags]
```

**Flags:**

```
      --check       Run checks automatically
      --fix         Auto-fix issues where possible (not yet implemented)
      --json        Output results as JSON
      --skip-lint   Skip lint check explicitly
```

### issuegraph prune

Permanently delete closed non-ephemeral issuegraph and their associated data.

Use this to trim closed regular issuegraph (tasks, features, bugs, chores, etc.)
that are no longer useful. The common case is a long-lived repo where
closed work has piled up and is bloating auto-export or slowing queries.

Requires --older-than or --pattern. The flag is a safety gate — without
it, a muscle-memory `--force` could wipe every closed issue in the repo.
Use `--pattern '*'` if you really do want to sweep everything closed.

Deletes: issues, dependencies, labels, events, and comments for matching issuegraph.
Skips: pinned issuegraph (protected), open/in-progress issuegraph, and ephemeral issuegraph.

To delete closed ephemeral issuegraph (wisps, transient molecules) use
`issuegraph purge` instead.

For full Dolt storage reclaim after deleting many rows, follow with `issuegraph flatten`
so history can be collapsed and old chunks can be garbage-collected.

EXAMPLES:
  issuegraph prune --older-than 30d              # Preview closed issuegraph &gt;30d old
  issuegraph prune --older-than 30d --force      # Delete them
  issuegraph prune --older-than 90d --dry-run    # Detailed preview with stats
  issuegraph prune --pattern "*" --force         # Delete all closed regular issuegraph
  issuegraph prune --pattern "gm-temp-*" --force # Scope to a pattern

```
issuegraph prune [flags]
```

**Flags:**

```
      --dry-run             Preview what would be pruned with stats
  -f, --force               Actually prune (without this, shows preview)
      --older-than string   Only prune issuegraph closed more than N ago (e.g., 30d, 2w, 60)
      --pattern string      Only prune issuegraph matching ID glob pattern (e.g., 'gm-old-*')
```

### issuegraph purge

Permanently delete closed ephemeral issuegraph and their associated data.

Closed ephemeral issuegraph (wisps, transient molecules) accumulate rapidly and
have no value once closed. This command removes them to reclaim storage.

Deletes: issues, dependencies, labels, events, and comments for matching issuegraph.
Skips: pinned issuegraph (protected).

To delete closed non-ephemeral issuegraph (regular tasks, features, bugs, etc.)
use `issuegraph prune` instead.

For full Dolt storage reclaim after deleting many rows, follow with `issuegraph flatten`
so history can be collapsed and old chunks can be garbage-collected.

EXAMPLES:
  issuegraph purge                           # Preview what would be purged
  issuegraph purge --force                   # Delete all closed ephemeral issuegraph
  issuegraph purge --older-than 7d --force   # Only purge items closed 7+ days ago
  issuegraph purge --pattern "*-wisp-*"      # Only purge matching ID pattern
  issuegraph purge --dry-run                 # Detailed preview with stats

```
issuegraph purge [flags]
```

**Flags:**

```
      --dry-run             Preview what would be purged with stats
  -f, --force               Actually purge (without this, shows preview)
      --older-than string   Only purge issuegraph closed more than N ago (e.g., 7d, 2w, 30)
      --pattern string      Only purge issuegraph matching ID glob pattern (e.g., *-wisp-*)
```

### issuegraph recompute-blocked

Recompute the denormalized is_blocked flag for every issue and wisp.

is_blocked is derived from the dependency graph and maintained automatically by
local writes and by a post-pull recompute scoped to what the merge changed. If
that scoped recompute is skipped — a recompute that failed after its merge
committed, or a conflicted pull resolved by hand — the flag can go stale, and a
later pull that merges nothing will not refresh it (issuegraph-6dnrw.37). 'issuegraph ready'
trusts the flag, so stale values silently hide ready work or surface blocked
work.

This command runs the full recompute unconditionally and commits the result.
It is idempotent: on a consistent database it changes nothing. Works in both
embedded and server mode (unlike 'issuegraph doctor', which is server-mode only).

Examples:
  issuegraph recompute-blocked          # Repair stale is_blocked flags
  issuegraph recompute-blocked --json   # Machine-parseable &#123;"rows_corrected": N&#125;

```
issuegraph recompute-blocked
```

### issuegraph rename-prefix

Rename the issue prefix for all issues in the database.
This will update all issue IDs and all text references across all fields.

USE CASES:
- Shortening long prefixes (e.g., 'knowledge-work-' → 'kw-')
- Rebranding project naming conventions
- Consolidating multiple prefixes after database corruption
- Migrating to team naming standards

Prefix validation rules:
- Max length: 8 characters
- Allowed characters: lowercase letters, numbers, hyphens
- Must start with a letter
- Must end with a hyphen (e.g., 'kw-', 'work-')
- Cannot be empty or just a hyphen

Multiple prefix detection and repair:
If issues have multiple prefixes (corrupted database), use --repair to consolidate them.
The --repair flag will rename all issues with incorrect prefixes to the new prefix,
preserving issues that already have the correct prefix.

EXAMPLES:
  issuegraph rename-prefix kw-                # Rename from 'knowledge-work-' to 'kw-'
  issuegraph rename-prefix mtg- --repair      # Consolidate multiple prefixes into 'mtg-'
  issuegraph rename-prefix team- --dry-run    # Preview changes without applying

NOTE: This is a rare operation. Most users never need this command.

```
issuegraph rename-prefix <new-prefix> [flags]
```

**Flags:**

```
      --dry-run   Preview changes without applying them
      --repair    Repair database with multiple prefixes by consolidating them
```

### issuegraph rules

Audit and compact Claude rules

```
issuegraph rules
```

#### issuegraph rules audit

Scan rules for contradictions and merge opportunities

```
issuegraph rules audit [flags]
```

**Flags:**

```
      --path string       Path to rules directory (default ".claude/rules/")
      --threshold float   Jaccard similarity threshold (default 0.6)
```

#### issuegraph rules compact

Merge related rules into composites

```
issuegraph rules compact [flags]
```

**Flags:**

```
      --auto            Apply audit suggestions
      --dry-run         Preview without applying
      --group strings   Rule names to merge
      --path string     Path to rules directory (default ".claude/rules/")
```

### issuegraph sql

Execute a raw SQL query against the underlying database (SQLite or Dolt).

Useful for debugging, maintenance, and working around bugs in higher-level commands.

Examples:
  issuegraph sql 'SELECT COUNT(*) FROM issues'
  issuegraph sql 'SELECT id, title FROM issues WHERE status = "open" LIMIT 5'
  issuegraph sql 'DELETE FROM dirty_issues WHERE issue_id = "issuegraph-abc123"'
  issuegraph sql --csv 'SELECT id, title, status FROM issues'

The query is passed directly to the database. SELECT queries return results as a
table (or JSON/CSV with --json/--csv). Non-SELECT queries (INSERT, UPDATE, DELETE)
report the number of rows affected.

WARNING: Direct database access bypasses the storage layer. Use with caution.

```
issuegraph sql <query> [flags]
```

**Flags:**

```
      --csv   Output results in CSV format
```

### issuegraph upgrade

Commands for checking issuegraph version upgrades and reviewing changes.

The upgrade command helps you stay aware of issuegraph version changes:
  - issuegraph upgrade status: Check if issuegraph version changed since last use
  - issuegraph upgrade review: Show what's new since your last version
  - issuegraph upgrade ack: Acknowledge the current version

Version tracking is automatic - issuegraph updates metadata.json on every run.

```
issuegraph upgrade
```

#### issuegraph upgrade ack

Mark the current issuegraph version as acknowledged.

This updates metadata.json to record that you've seen the current
version. Mainly useful after reviewing upgrade changes to suppress
future upgrade notifications.

Note: Version tracking happens automatically, so you don't need to
run this command unless you want to explicitly mark acknowledgement.

Examples:
  issuegraph upgrade ack
  issuegraph upgrade ack --json

```
issuegraph upgrade ack
```

#### issuegraph upgrade review

Show what's new in issuegraph since the last version you used.

Unlike 'issuegraph info --whats-new' which shows the last 3 versions,
this command shows ALL changes since your specific last version.

If you're upgrading from an old version, you'll see the complete
changelog of everything that changed since then.

Examples:
  issuegraph upgrade review
  issuegraph upgrade review --json

```
issuegraph upgrade review
```

#### issuegraph upgrade status

Check if issuegraph has been upgraded since you last used it.

This command uses the version tracking that happens automatically
at startup to detect if issuegraph was upgraded.

Examples:
  issuegraph upgrade status
  issuegraph upgrade status --json

```
issuegraph upgrade status
```

### issuegraph worktree

Manage git worktrees with proper issuegraph configuration.

Worktrees allow multiple working directories sharing the same git repository,
enabling parallel development (e.g., multiple agents or features).

Worktrees automatically share the same issuegraph database as the main repository
via git common directory discovery — no manual redirect configuration needed.

Examples:
  issuegraph worktree create feature-auth           # Create worktree
  issuegraph worktree create bugfix --branch fix-1  # Create with specific branch name
  issuegraph worktree list                          # List all worktrees
  issuegraph worktree remove feature-auth           # Remove worktree (with safety checks)
  issuegraph worktree info                          # Show info about current worktree

```
issuegraph worktree
```

#### issuegraph worktree create

Create a git worktree for parallel development.

This command:
1. Creates a git worktree at ./&lt;name&gt; (or specified path)
2. Adds the worktree path to .gitignore (if inside repo root)

The worktree automatically shares the same issuegraph database as the main
repository via git common directory discovery — no redirect file needed.

Examples:
  issuegraph worktree create feature-auth           # Create at ./feature-auth
  issuegraph worktree create bugfix --branch fix-1  # Create with branch name
  issuegraph worktree create ../agents/worker-1     # Create at relative path

```
issuegraph worktree create <name> [--branch=<branch>] [flags]
```

**Flags:**

```
      --branch string   Branch name for the worktree (default: same as name)
```

#### issuegraph worktree info

Show information about the current worktree.

If the current directory is in a git worktree, shows:
- Worktree path and name
- Branch
- IssueGraph configuration (redirect or main)
- Main repository location

Examples:
  issuegraph worktree info          # Show current worktree info
  issuegraph worktree info --json   # JSON output

```
issuegraph worktree info
```

#### issuegraph worktree list

List all git worktrees and their issuegraph configuration state.

Shows each worktree with:
- Name (directory name)
- Path (full path)
- Branch
- IssueGraph state: "redirect" (uses shared db), "shared" (is main), "none" (no issuegraph)

Examples:
  issuegraph worktree list          # List all worktrees
  issuegraph worktree list --json   # JSON output

```
issuegraph worktree list
```

#### issuegraph worktree remove

Remove a git worktree with safety checks.

Before removing, this command checks for:
- Uncommitted changes
- Unpushed commits
- Stashes

Use --force to skip safety checks (not recommended).

Examples:
  issuegraph worktree remove feature-auth         # Remove with safety checks
  issuegraph worktree remove feature-auth --force # Skip safety checks

```
issuegraph worktree remove <name> [flags]
```

**Flags:**

```
      --force   Skip safety checks
```

## Integrations & Advanced:

### issuegraph admin

Administrative commands for issuegraph database maintenance.

These commands are for advanced users and should be used carefully:
  cleanup   Delete closed issues (issue lifecycle)
  compact   Compact old closed issues to save space (storage optimization)
  reset     Remove all issuegraph data and configuration (full reset)

For routine maintenance, prefer 'issuegraph doctor --fix' which handles common repairs
automatically. Use these admin commands for targeted database operations.

```
issuegraph admin
```

#### issuegraph admin cleanup

Delete closed issues to reduce database size.

This command permanently removes closed issues from the database.

NOTE: This command only manages issue lifecycle (closed -&gt; deleted). For general
health checks and automatic repairs, use 'issuegraph doctor --fix' instead.

By default, deletes ALL closed issues. Use --older-than to only delete
issues closed before a certain date.

EXAMPLES:
  issuegraph admin cleanup --force                          # Delete all closed issues
  issuegraph admin cleanup --older-than 30 --force          # Only issues closed 30+ days ago
  issuegraph admin cleanup --ephemeral --force              # Only closed wisps (transient molecules)
  issuegraph admin cleanup --dry-run                        # Preview what would be deleted

SAFETY:
- Requires --force flag to actually delete (unless --dry-run)
- Supports --cascade to delete dependents
- Shows preview of what will be deleted
- Use --json for programmatic output

SEE ALSO:
  issuegraph doctor --fix    Automatic health checks and repairs (recommended for routine maintenance)
  issuegraph admin compact   Compact old closed issues to save space

```
issuegraph admin cleanup [flags]
```

**Flags:**

```
      --cascade          Recursively delete all dependent issues
      --dry-run          Preview what would be deleted without making changes
      --ephemeral        Only delete closed wisps (transient molecules)
  -f, --force            Actually delete (without this flag, shows error)
      --older-than int   Only delete issues closed more than N days ago (0 = all closed issues)
```

#### issuegraph admin compact

Compact old closed issues using semantic summarization.

Compaction reduces database size by summarizing closed issues that are no longer
actively referenced. This is permanent graceful decay - original content is discarded.

Modes:
  - Analyze: Export candidates for agent review (no API key needed)
  - Apply: Accept agent-provided summary (no API key needed)
  - Auto: AI-powered compaction (requires ANTHROPIC_API_KEY or ai.api_key, legacy)
  - Dolt: Run Dolt garbage collection (for Dolt-backend repositories)

Tiers:
  - Tier 1: Semantic compression (30 days closed, 70% reduction)
  - Tier 2: Ultra compression (90 days closed) - planned, not yet implemented

Dolt Garbage Collection:
  With auto-commit per mutation, Dolt commit history grows over time. Use
  --dolt to run Dolt garbage collection and reclaim disk space.

  --dolt: Run Dolt GC on .issuegraph/dolt directory to free disk space.
          This removes unreachable commits and compacts storage.

Examples:
  # Dolt garbage collection
  issuegraph compact --dolt                        # Run Dolt GC
  issuegraph compact --dolt --dry-run              # Preview without running GC

  # Agent-driven workflow (recommended)
  issuegraph compact --analyze --json              # Get candidates with full content
  issuegraph compact --apply --id issuegraph-42 --summary summary.txt
  issuegraph compact --apply --id issuegraph-42 --summary - &lt; summary.txt

  # Legacy AI-powered workflow
  issuegraph compact --auto --dry-run              # Preview candidates
  issuegraph compact --auto --all                  # Compact all eligible issues
  issuegraph compact --auto --id issuegraph-42             # Compact specific issue

  # Statistics
  issuegraph compact --stats                       # Show statistics


```
issuegraph admin compact [flags]
```

**Flags:**

```
      --actor string     Actor name for audit trail (default "agent")
      --all              Process all candidates
      --analyze          Analyze mode: export candidates for agent review
      --apply            Apply mode: accept agent-provided summary
      --auto             Auto mode: AI-powered compaction (legacy)
      --batch-size int   Issues per batch (default 10)
      --dolt             Dolt mode: run Dolt garbage collection on .issuegraph/dolt
      --dry-run          Preview without compacting
      --force            Force compact (bypass checks, requires --id)
      --id string        Compact specific issue
      --json             Output JSON format
      --limit int        Limit number of candidates (0 = no limit)
      --stats            Show compaction statistics
      --summary string   Path to summary file (use '-' for stdin)
      --tier int         Compaction tier (only tier 1 is implemented) (default 1)
      --workers int      Parallel workers (default 5)
```

#### issuegraph admin reset

Reset issuegraph to an uninitialized state, removing all local data.

This command removes:
  - The .issuegraph directory (database, JSONL, config)
  - Git hooks installed by issuegraph
  - Sync branch worktrees

By default, shows what would be deleted (dry-run mode).
Use --force to actually perform the reset.

Examples:
  issuegraph reset              # Show what would be deleted
  issuegraph reset --force      # Actually delete everything

```
issuegraph admin reset [flags]
```

**Flags:**

```
      --force   Actually perform the reset (required)
```

### issuegraph jira

Synchronize issues between issuegraph and Jira.

Configuration:
  issuegraph config set jira.url "https://company.atlassian.net"
  issuegraph config set jira.project "PROJ"
  issuegraph config set jira.projects "PROJ1,PROJ2"   # Multiple projects
  issuegraph config set jira.api_token "YOUR_TOKEN"
  issuegraph config set jira.username "your_email@company.com"  # For Jira Cloud
  issuegraph config set jira.push_prefix "hippo"       # Only push hippo-* issues to Jira
  issuegraph config set jira.push_prefix "proj1,proj2" # Multiple prefixes (comma-separated)

Environment variables (alternative to config):
  JIRA_API_TOKEN  - Jira API token
  JIRA_USERNAME   - Jira username/email
  JIRA_PROJECTS   - Comma-separated project keys

Examples:
  issuegraph jira sync --pull         # Import issues from Jira
  issuegraph jira sync --push         # Export issues to Jira
  issuegraph jira sync                # Bidirectional sync (pull then push)
  issuegraph jira sync --dry-run      # Preview sync without changes
  issuegraph jira status              # Show sync status

```
issuegraph jira
```

#### issuegraph jira pull

Pull one or more items from Jira.

Accepts issue IDs or external references as positional arguments.
Equivalent to: issuegraph jira sync --pull --issues &lt;refs&gt;

```
issuegraph jira pull [refs...] [flags]
```

**Flags:**

```
      --dry-run   Preview pull without making changes
```

#### issuegraph jira push

Push one or more issuegraph issues to Jira.

Accepts issue IDs as positional arguments.
Equivalent to: issuegraph jira sync --push --issues &lt;ids&gt;

```
issuegraph jira push [issue-ids...] [flags]
```

**Flags:**

```
      --dry-run   Preview push without making changes
```

#### issuegraph jira status

Show the current Jira sync status, including:
  - Last sync timestamp
  - Configuration status
  - Number of issues with Jira links
  - Issues pending push (no external_ref)

```
issuegraph jira status
```

#### issuegraph jira sync

Synchronize issues between issuegraph and Jira.

Modes:
  --pull         Import issues from Jira into issuegraph
  --push         Export issues from issuegraph to Jira
  (no flags)     Bidirectional sync: pull then push, with conflict resolution

Conflict Resolution:
  By default, newer timestamp wins. Override with:
  --prefer-local   Always prefer local issuegraph version
  --prefer-jira    Always prefer Jira version

Examples:
  issuegraph jira sync --pull                # Import from Jira
  issuegraph jira sync --push --create-only  # Push new issues only
  issuegraph jira sync --dry-run             # Preview without changes
  issuegraph jira sync --prefer-local        # Bidirectional, local wins

```
issuegraph jira sync [flags]
```

**Flags:**

```
      --create-only       Only create new issues, don't update existing
      --dry-run           Preview sync without making changes
      --issues string     Comma-separated issue IDs to sync selectively (e.g., issuegraph-abc,issuegraph-def). Mutually exclusive with --parent.
      --parent string     Limit push to this issue and its descendants (push only). Mutually exclusive with --issues.
      --prefer-jira       Prefer Jira version on conflicts
      --prefer-local      Prefer local version on conflicts
      --project strings   Project key(s) to sync (overrides configured project/projects)
      --pull              Pull issues from Jira
      --push              Push issues to Jira
      --state string      Issue state to sync: open, closed, all (default "all")
```

### issuegraph linear

Synchronize issues between issuegraph and Linear.

Configuration:
  issuegraph config set linear.api_key "YOUR_API_KEY"
  issuegraph config set linear.team_id "TEAM_ID"
  issuegraph config set linear.team_ids "TEAM_ID1,TEAM_ID2"  # Multiple teams (comma-separated)
  issuegraph config set linear.project_id "PROJECT_ID"  # Optional: sync only this project

Environment variables (alternative to config):
  LINEAR_API_KEY  - Linear API key (for individual developers)
  LINEAR_TEAM_ID  - Linear team ID (UUID, singular)
  LINEAR_TEAM_IDS - Linear team IDs (comma-separated UUIDs)

OAuth (for CI workers / automated sync):
  LINEAR_OAUTH_CLIENT_ID     - OAuth app client ID
  LINEAR_OAUTH_CLIENT_SECRET - OAuth app client secret

  When both OAuth env vars are set, OAuth client_credentials flow is used
  instead of the API key. This allows CI workers to authenticate as an
  application (actor=application) rather than impersonating a user.
  Precedence: OAuth &gt; LINEAR_API_KEY &gt; config file.

Data Mapping (optional, sensible defaults provided):
  Priority mapping (Linear 0-4 to IssueGraph 0-4):
    issuegraph config set linear.priority_map.0 4    # No priority -&gt; Backlog
    issuegraph config set linear.priority_map.1 0    # Urgent -&gt; Critical
    issuegraph config set linear.priority_map.2 1    # High -&gt; High
    issuegraph config set linear.priority_map.3 2    # Medium -&gt; Medium
    issuegraph config set linear.priority_map.4 3    # Low -&gt; Low

  State mapping (Linear state type to IssueGraph status):
    issuegraph config set linear.state_map.backlog open
    issuegraph config set linear.state_map.unstarted open
    issuegraph config set linear.state_map.started in_progress
    issuegraph config set linear.state_map.completed closed
    issuegraph config set linear.state_map.canceled closed
    issuegraph config set linear.state_map.my_custom_state in_progress  # Custom state names

  Label to issue type mapping:
    issuegraph config set linear.label_type_map.bug bug
    issuegraph config set linear.label_type_map.feature feature
    issuegraph config set linear.label_type_map.epic epic

  Relation type mapping (Linear relations to IssueGraph dependencies):
    issuegraph config set linear.relation_map.blocks blocks
    issuegraph config set linear.relation_map.blockedBy blocks
    issuegraph config set linear.relation_map.duplicate duplicates
    issuegraph config set linear.relation_map.related related

  ID generation (optional, hash IDs to match issuegraph/Jira hash mode):
    issuegraph config set linear.id_mode "hash"      # hash (default)
    issuegraph config set linear.hash_length "6"     # hash length 3-8 (default: 6)

Examples:
  issuegraph linear sync --pull         # Import issues from Linear
  issuegraph linear sync --push         # Export issues to Linear
  issuegraph linear sync                # Bidirectional sync (pull then push)
  issuegraph linear sync --dry-run      # Preview sync without changes
  issuegraph create "Fix login" --external-ref https://linear.app/team/issue/TEAM-123
                              # Link a local issue to an existing Linear issue
  issuegraph linear status              # Show sync status

```
issuegraph linear
```

#### issuegraph linear pull

Pull one or more items from Linear.

Accepts issue IDs or external references as positional arguments.
Equivalent to: issuegraph linear sync --pull --issues &lt;refs&gt;

```
issuegraph linear pull [refs...] [flags]
```

**Flags:**

```
      --dry-run     Preview pull without making changes
      --relations   Import Linear relations as issuegraph dependencies when pulling
```

#### issuegraph linear push

Push one or more issuegraph issues to Linear.

Accepts issue IDs as positional arguments.
Equivalent to: issuegraph linear sync --push --issues &lt;ids&gt;

```
issuegraph linear push [issue-ids...] [flags]
```

**Flags:**

```
      --dry-run   Preview push without making changes
```

#### issuegraph linear status

Show the current Linear sync status, including:
  - Last sync timestamp
  - Configuration status
  - Number of issues with Linear links
  - Issues pending push (no external_ref)

```
issuegraph linear status
```

#### issuegraph linear sync

Synchronize issues between issuegraph and Linear.

Modes:
  --pull              Import issues from Linear into issuegraph
  --push              Export issues from issuegraph to Linear
  --pull-if-stale     Pull only if data is stale (skip if fresh)
  (no flags)          Bidirectional sync: pull then push, with conflict resolution

Staleness (--pull-if-stale):
  --threshold 20m     How old data must be before pulling (default 20m)
  A 5-minute debounce prevents agent loops: if a pull completed within 5 minutes,
  data is always treated as fresh regardless of the threshold.

Team Selection:
  --team ID1,ID2  Override configured team IDs for this sync
  Multiple teams can be configured via linear.team_ids (comma-separated).
  Falls back to linear.team_id for backward compatibility.
  Push requires explicit --team when multiple teams are configured.

Pull Options:
  --milestones       Reconstruct Linear project milestones as local epic parents

Type Filtering (--push only):
  --type task,feature       Only sync issues of these types
  --exclude-type wisp       Exclude issues of these types
  --include-ephemeral       Include ephemeral issues (wisps, etc.); default is to exclude
  --parent TICKET           Only push this ticket and its descendants
  --relations               Import Linear relations as issuegraph dependencies on pull

Conflict Resolution:
  By default, newer timestamp wins. Override with:
  --prefer-local    Always prefer local issuegraph version
  --prefer-linear   Always prefer Linear version

Examples:
  issuegraph linear sync --pull                         # Import from Linear
  issuegraph linear sync --pull-if-stale                # Pull only if data is stale
  issuegraph linear sync --pull-if-stale --threshold 5m # Pull if older than 5 minutes
  issuegraph linear sync --pull --relations             # Import Linear blocking relations as issuegraph deps
  issuegraph linear sync --push --create-only           # Push new issues only
  issuegraph linear sync --push --type=task,feature     # Push only tasks and features
  issuegraph linear sync --push --exclude-type=wisp     # Push all except wisps
  issuegraph linear sync --push --parent=issuegraph-abc123      # Push one ticket tree
  issuegraph linear sync --dry-run                      # Preview without changes
  issuegraph linear sync --prefer-local                 # Bidirectional, local wins

```
issuegraph linear sync [flags]
```

**Flags:**

```
      --create-only            Only create new issues, don't update existing
      --dry-run                Preview sync without making changes
      --exclude-type strings   Exclude issues of these types (can be repeated)
      --include-ephemeral      Include ephemeral issues (wisps, etc.) when pushing to Linear
      --issues string          Comma-separated issue IDs to sync selectively (e.g., issuegraph-abc,issuegraph-def). Mutually exclusive with --parent.
      --milestones             Reconstruct Linear project milestones as local epic parents when pulling
      --no-wait                Fail immediately if another sync is running instead of waiting
      --parent string          Limit push to this issuegraph ticket and its descendants
      --prefer-linear          Prefer Linear version on conflicts
      --prefer-local           Prefer local version on conflicts
      --pull                   Pull issues from Linear
      --pull-if-stale          Pull only if Linear data is stale (skip if fresh)
      --push                   Push issues to Linear
      --relations              Import Linear relations as issuegraph dependencies when pulling
      --state string           Issue state to sync: open, closed, all (default "all")
      --team strings           Team ID(s) to sync (overrides configured team_id/team_ids)
      --threshold duration     Staleness threshold for --pull-if-stale (default 20m) (default 20m0s)
      --type strings           Only sync issues of these types (can be repeated)
      --update-refs            Update external_ref after creating Linear issues (default true)
```

#### issuegraph linear teams

List all teams accessible with your Linear API key.

Use this to find the team ID (UUID) needed for configuration.

Example:
  issuegraph linear teams
  issuegraph config set linear.team_id "12345678-1234-1234-1234-123456789abc"

```
issuegraph linear teams
```

### issuegraph repo

Configure and manage multiple repository support for multi-repo hydration.

Multi-repo support allows hydrating issues from multiple issuegraph repositories
into a single database for unified cross-repo issue tracking.

Configuration is stored in .issuegraph/config.yaml under the 'repos' section:

  repos:
    primary: "."
    additional:
      - ~/issuegraph-planning
      - ~/work-repo

Examples:
  issuegraph repo add ~/issuegraph-planning       # Add planning repo
  issuegraph repo add ../other-repo          # Add relative path repo
  issuegraph repo list                       # Show all configured repos
  issuegraph repo remove ~/issuegraph-planning    # Remove by path
  issuegraph repo sync                       # Sync from all configured repos

```
issuegraph repo
```

#### issuegraph repo add

Add a repository path to the repos.additional list in config.yaml.

The path should point to a directory containing a .issuegraph folder.
Paths can be absolute or relative (they are stored as-is).

This modifies .issuegraph/config.yaml, which is version-controlled and
shared across all clones of this repository.

```
issuegraph repo add <path> [flags]
```

**Flags:**

```
      --json   Output JSON
```

#### issuegraph repo list

List all repositories configured in .issuegraph/config.yaml.

Shows the primary repository (always ".") and any additional
repositories configured for hydration.

```
issuegraph repo list [flags]
```

**Flags:**

```
      --json   Output JSON
```

#### issuegraph repo remove

Remove a repository path from the repos.additional list in config.yaml.

The path must exactly match what was added (e.g., if you added "~/foo",
you must remove "~/foo", not "/home/user/foo").

This command also removes any previously-hydrated issues from the database
that came from the removed repository.

```
issuegraph repo remove <path> [flags]
```

**Flags:**

```
      --json   Output JSON
```

#### issuegraph repo sync

Synchronize issues from all configured additional repositories.

Reads issues.jsonl from each additional repository and imports them into
the primary database with their original prefixes and source_repo set.
Uses mtime caching to skip repos whose JSONL hasn't changed.

Also triggers Dolt push/pull if a remote is configured.

```
issuegraph repo sync [flags]
```

**Flags:**

```
      --json      Output JSON
      --verbose   Show detailed sync progress
```

## Other Commands:

### issuegraph ado

Commands for syncing issues between issuegraph and Azure DevOps.

Configuration can be set via 'issuegraph config' or environment variables:
  ado.org / AZURE_DEVOPS_ORG              - Organization name
  ado.project / AZURE_DEVOPS_PROJECT      - Project name (single)
  ado.projects / AZURE_DEVOPS_PROJECTS    - Project names (comma-separated)
  ado.pat / AZURE_DEVOPS_PAT              - Personal access token
  ado.url / AZURE_DEVOPS_URL              - Custom base URL (on-prem)

```
issuegraph ado
```

#### issuegraph ado projects

List Azure DevOps projects that the configured token has access to.

```
issuegraph ado projects
```

#### issuegraph ado pull

Pull one or more items from Azure DevOps.

Accepts issue IDs or external references as positional arguments.
Equivalent to: issuegraph ado sync --pull-only --issues &lt;refs&gt;

```
issuegraph ado pull [refs...] [flags]
```

**Flags:**

```
      --dry-run   Preview pull without making changes
```

#### issuegraph ado push

Push one or more issuegraph issues to Azure DevOps.

Accepts issue IDs as positional arguments.
Equivalent to: issuegraph ado sync --push-only --issues &lt;ids&gt;

```
issuegraph ado push [issue-ids...] [flags]
```

**Flags:**

```
      --dry-run   Preview push without making changes
```

#### issuegraph ado status

Display current Azure DevOps configuration and sync status.

```
issuegraph ado status
```

#### issuegraph ado sync

Synchronize issues between issuegraph and Azure DevOps.

By default, performs bidirectional sync:
- Pulls new/updated work items from Azure DevOps to issuegraph
- Pushes local issuegraph issues to Azure DevOps

Use --pull-only or --push-only to limit direction.

Filters (--area-path, --iteration-path, --types, --states) restrict
which work items are synced. On pull, they limit the WIQL query. On push,
--types and --states filter local issuegraph before pushing to ADO. Use
--no-create with push to skip creating new ADO work items (only update
existing linked items). Filters can also be persisted via config:
  ado.filter.area_path, ado.filter.iteration_path,
  ado.filter.types, ado.filter.states
CLI flags override config values when both are set.

```
issuegraph ado sync [flags]
```

**Flags:**

```
      --area-path string        Filter to ADO area path (e.g., "Project\Team")
      --bootstrap-match         Enable heuristic matching for first sync
      --dry-run                 Show what would be synced without making changes
      --issues string           Comma-separated issue IDs to sync selectively (e.g., issuegraph-abc,issuegraph-def). Mutually exclusive with --parent.
      --iteration-path string   Filter to ADO iteration path (e.g., "Project\Sprint 1")
      --no-create               Never create new items in either direction (pull or push)
      --parent string           Limit push to this issue and its descendants (push only). Mutually exclusive with --issues.
      --prefer-ado              On conflict, use Azure DevOps version
      --prefer-local            On conflict, keep local issuegraph version
      --prefer-newer            On conflict, use most recent version (default)
      --project strings         Project name(s) to sync (overrides configured project/projects)
      --pull-only               Only pull issues from Azure DevOps
      --push-only               Only push issues to Azure DevOps
      --reconcile               Force reconciliation scan for deleted items
      --states string           Filter to ADO states, comma-separated (e.g., "New,Active,Resolved")
      --types string            Filter to work item types, comma-separated (e.g., "Bug,Task,User Story")
```

### issuegraph audit

Audit log entries are appended to .issuegraph/interactions.jsonl.

Each line is one event. This file is intended to be versioned in git and used for:
- auditing ("why did the agent do that?")
- dataset generation (SFT/RL fine-tuning)

Entries are append-only. Labeling creates a new "label" entry that references a parent entry.

```
issuegraph audit
```

#### issuegraph audit label

Append a label entry referencing an existing interaction

```
issuegraph audit label <entry-id> [flags]
```

**Flags:**

```
      --label string    Label value (e.g. "good" or "bad")
      --reason string   Reason for label
```

#### issuegraph audit record

Append an audit interaction entry

```
issuegraph audit record [flags]
```

**Flags:**

```
      --error string       Error string (llm_call/tool_call)
      --exit-code int      Exit code (tool_call) (default -1)
      --issue-id string    Related issue id (issuegraph-...)
      --kind string        Entry kind (e.g. llm_call, tool_call, label)
      --model string       Model name (llm_call)
      --prompt string      Prompt text (llm_call)
      --response string    Response text (llm_call)
      --stdin              Read a JSON object from stdin (must match audit.Entry schema)
      --tool-name string   Tool name (tool_call)
```

### issuegraph blocked

Show blocked issues

```
issuegraph blocked [flags]
```

**Flags:**

```
      --parent string   Filter to descendants of this issue/epic
```

### issuegraph completion

Generate the autocompletion script for issuegraph for the specified shell.
See each sub-command's help for details on how to use the generated script.


```
issuegraph completion
```

#### issuegraph completion bash

Generate the autocompletion script for the bash shell.

This script depends on the 'bash-completion' package.
If it is not installed already, you can install it via your OS's package manager.

To load completions in your current shell session:

	source &lt;(issuegraph completion bash)

To load completions for every new session, execute once:

#### Linux:

	issuegraph completion bash &gt; /etc/bash_completion.d/issuegraph

#### macOS:

	issuegraph completion bash &gt; $(brew --prefix)/etc/bash_completion.d/issuegraph

You will need to start a new shell for this setup to take effect.


```
issuegraph completion bash
```

**Flags:**

```
      --no-descriptions   disable completion descriptions
```

#### issuegraph completion fish

Generate the autocompletion script for the fish shell.

To load completions in your current shell session:

	issuegraph completion fish | source

To load completions for every new session, execute once:

	issuegraph completion fish &gt; ~/.config/fish/completions/issuegraph.fish

You will need to start a new shell for this setup to take effect.


```
issuegraph completion fish [flags]
```

**Flags:**

```
      --no-descriptions   disable completion descriptions
```

#### issuegraph completion powershell

Generate the autocompletion script for powershell.

To load completions in your current shell session:

	issuegraph completion powershell | Out-String | Invoke-Expression

To load completions for every new session, add the output of the above command
to your powershell profile.


```
issuegraph completion powershell [flags]
```

**Flags:**

```
      --no-descriptions   disable completion descriptions
```

#### issuegraph completion zsh

Generate the autocompletion script for the zsh shell.

If shell completion is not already enabled in your environment you will need
to enable it.  You can execute the following once:

	echo "autoload -U compinit; compinit" &gt;&gt; ~/.zshrc

To load completions in your current shell session:

	source &lt;(issuegraph completion zsh)

To load completions for every new session, execute once:

#### Linux:

	issuegraph completion zsh &gt; "$&#123;fpath[1]&#125;/_bd"

#### macOS:

	issuegraph completion zsh &gt; $(brew --prefix)/share/zsh/site-functions/_bd

You will need to start a new shell for this setup to take effect.


```
issuegraph completion zsh [flags]
```

**Flags:**

```
      --no-descriptions   disable completion descriptions
```

### issuegraph cook

Cook transforms a .formula.json file into a proto.

By default, cook outputs the resolved formula as JSON to stdout for
ephemeral use. The output can be inspected, piped, or saved to a file.

Two cooking modes are available:

  COMPILE-TIME (default, --mode=compile):
    Produces a proto with &#123;&#123;variable&#125;&#125; placeholders intact.
    Use for: modeling, estimation, contractor handoff, planning.
    Variables are NOT substituted - the output shows the template structure.

  RUNTIME (--mode=runtime or when --var flags provided):
    Produces a fully-resolved proto with variables substituted.
    Use for: final validation before pour, seeing exact output.
    Requires all variables to have values (via --var or defaults).

Formulas are high-level workflow templates that support:
  - Variable definitions with defaults and validation
  - Step definitions that become issue hierarchies
  - Composition rules for bonding formulas together
  - Inheritance via extends

The --persist flag enables the legacy behavior of writing the proto
to the database. This is useful when you want to reuse the same
proto multiple times without re-cooking.

For most workflows, prefer ephemeral protos: pour and wisp commands
accept formula names directly and cook inline.

Examples:
  issuegraph cook mol-feature.formula.json                    # Compile-time: keep &#123;&#123;vars&#125;&#125;
  issuegraph cook mol-feature --var name=auth                 # Runtime: substitute vars
  issuegraph cook mol-feature --mode=runtime --var name=auth  # Explicit runtime mode
  issuegraph cook mol-feature --dry-run                       # Preview steps
  issuegraph cook mol-release.formula.json --persist          # Write to database
  issuegraph cook mol-release.formula.json --persist --force  # Replace existing

Output (default):
  JSON representation of the resolved formula with all steps.

Output (--persist):
  Creates a proto issue in the database with:
  - ID matching the formula name (e.g., mol-feature)
  - The "template" label for proto identification
  - Child issues for each step
  - Dependencies matching depends_on relationships

```
issuegraph cook <formula-file> [flags]
```

**Flags:**

```
      --dry-run               Preview what would be created
      --force                 Replace existing proto if it exists (requires --persist)
      --mode string           Cooking mode: compile (keep placeholders) or runtime (substitute vars)
      --persist               Persist proto to database (legacy behavior)
      --prefix string         Prefix to prepend to proto ID (e.g., 'gt-' creates 'gt-mol-feature')
      --search-path strings   Additional paths to search for formula inheritance
      --var stringArray       Variable substitution (key=value), enables runtime mode
```

### issuegraph defer

Defer issues to put them on ice for later.

Deferred issues are deliberately set aside - not blocked by anything specific,
just postponed for future consideration. Unlike blocked issues, there's no
dependency keeping them from being worked. Unlike closed issues, they will
be revisited.

Deferred issues don't show in 'issuegraph ready' but remain visible in 'issuegraph list'.

Examples:
  issuegraph defer issuegraph-abc                  # Defer a single issue (status-based)
  issuegraph defer issuegraph-abc --until=tomorrow # Defer until specific time
  issuegraph defer issuegraph-abc --reason="waiting on API access"
  issuegraph defer issuegraph-abc issuegraph-def           # Defer multiple issues

```
issuegraph defer [id...] [flags]
```

**Flags:**

```
      --reason string   Record why this issue is being deferred (appended to notes)
      --until string    Defer until specific time (e.g., +1h, tomorrow, next monday)
```

### issuegraph formula

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
issuegraph formula
```

#### issuegraph formula convert

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

#### issuegraph formula list

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

#### issuegraph formula show

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
issuegraph formula show <formula-name>
```

### issuegraph github

Commands for syncing issues between issuegraph and GitHub.

Configuration can be set via 'issuegraph config' or environment variables:
  github.token / GITHUB_TOKEN           - Personal access token
  github.owner / GITHUB_OWNER           - Repository owner
  github.repo / GITHUB_REPO             - Repository name
  github.repository / GITHUB_REPOSITORY - Combined "owner/repo" format
  github.url / GITHUB_API_URL           - Custom API URL (GitHub Enterprise)

```
issuegraph github
```

#### issuegraph github pull

Pull one or more items from GitHub.

Accepts issue IDs or external references as positional arguments.
Equivalent to: issuegraph github sync --pull-only --issues &lt;refs&gt;

```
issuegraph github pull [refs...] [flags]
```

**Flags:**

```
      --dry-run   Preview pull without making changes
```

#### issuegraph github push

Push one or more issuegraph issues to GitHub.

Accepts issue IDs as positional arguments.
Equivalent to: issuegraph github sync --push-only --issues &lt;ids&gt;

```
issuegraph github push [issue-ids...] [flags]
```

**Flags:**

```
      --dry-run   Preview push without making changes
```

#### issuegraph github repos

List GitHub repositories that the configured token has access to.

```
issuegraph github repos
```

#### issuegraph github status

Display current GitHub configuration and sync status.

```
issuegraph github status
```

#### issuegraph github sync

Synchronize issues between issuegraph and GitHub.

By default, performs bidirectional sync:
- Pulls new/updated issues from GitHub to issuegraph
- Pushes local issuegraph issues to GitHub

Use --pull-only or --push-only to limit direction.

```
issuegraph github sync [flags]
```

**Flags:**

```
      --dry-run         Show what would be synced without making changes
      --issues string   Comma-separated issue IDs to sync selectively (e.g., issuegraph-abc,issuegraph-def). Mutually exclusive with --parent.
      --parent string   Limit push to this issue and its descendants (push only). Mutually exclusive with --issues.
      --prefer-github   On conflict, use GitHub version
      --prefer-local    On conflict, keep local issuegraph version
      --prefer-newer    On conflict, use most recent version (default)
      --pull-only       Only pull issues from GitHub
      --push-only       Only push issues to GitHub
```

### issuegraph gitlab

Commands for syncing issues between issuegraph and GitLab.

Configuration can be set via 'issuegraph config' or environment variables:
  gitlab.url / GITLAB_URL                         - GitLab instance URL
  gitlab.token / GITLAB_TOKEN                     - Personal access token
  gitlab.project_id / GITLAB_PROJECT_ID           - Project ID or path
  gitlab.group_id / GITLAB_GROUP_ID               - Group ID for group-level sync
  gitlab.default_project_id / GITLAB_DEFAULT_PROJECT_ID - Project for creating issues in group mode

```
issuegraph gitlab
```

#### issuegraph gitlab projects

List GitLab projects that the configured token has access to.

```
issuegraph gitlab projects
```

#### issuegraph gitlab pull

Pull one or more items from GitLab.

Accepts issue IDs or external references as positional arguments.
Equivalent to: issuegraph gitlab sync --pull-only --issues &lt;refs&gt;

```
issuegraph gitlab pull [refs...] [flags]
```

**Flags:**

```
      --dry-run   Preview pull without making changes
```

#### issuegraph gitlab push

Push one or more issuegraph issues to GitLab.

Accepts issue IDs as positional arguments.
Equivalent to: issuegraph gitlab sync --push-only --issues &lt;ids&gt;

```
issuegraph gitlab push [issue-ids...] [flags]
```

**Flags:**

```
      --dry-run   Preview push without making changes
```

#### issuegraph gitlab status

Display current GitLab configuration and sync status.

```
issuegraph gitlab status
```

#### issuegraph gitlab sync

Synchronize issues between issuegraph and GitLab.

By default, performs bidirectional sync:
- Pulls new/updated issues from GitLab to issuegraph
- Pushes local issuegraph issues to GitLab

Use --pull-only or --push-only to limit direction.

```
issuegraph gitlab sync [flags]
```

**Flags:**

```
      --assignee string       Filter by assignee username
      --dry-run               Show what would be synced without making changes
      --exclude-type string   Exclude these issue types from sync (comma-separated)
      --issues string         Comma-separated issue IDs to sync selectively (e.g., issuegraph-abc,issuegraph-def). Mutually exclusive with --parent.
      --label string          Filter by labels (comma-separated, AND logic)
      --milestone string      Filter by milestone title
      --no-ephemeral          Exclude ephemeral/wisp issues from push (default: true) (default true)
      --parent string         Limit push to this issue and its descendants (push only). Mutually exclusive with --issues.
      --prefer-gitlab         On conflict, use GitLab version
      --prefer-local          On conflict, keep local issuegraph version
      --prefer-newer          On conflict, use most recent version (default)
      --project string        Filter to issues from this project ID (group mode)
      --pull-only             Only pull issues from GitLab
      --push-only             Only push issues to GitLab
      --type string           Only sync these issue types (comma-separated, e.g. 'epic,feature,task')
```

### issuegraph help

Help provides help for any command in the application.
Simply type issuegraph help [path to command] for full details.

```
issuegraph help [command] [flags]
```

**Flags:**

```
      --all                   Show help for all commands in a single document
      --doc string            Generate markdown docs for a single command
      --docs-root string      Generate repository CLI docs under this root
      --docs-version string   Also refresh one versioned website CLI reference, e.g. 1.0.5
  -h, --help                  help for help
      --list                  List all available commands
```

### issuegraph init-safety

issuegraph init flag safety contract.

Every issuegraph init invocation resolves project_id from exactly one explicitly
named source (local reinit, remote adoption, or a fresh mint). When the
source is ambiguous, issuegraph init refuses.

FLAG SURFACE

  issuegraph init                       Mint a new identity. Bootstraps from
                                origin if it has refs/dolt/data.

  issuegraph init --reinit-local        Re-initialize local .issuegraph/ over existing
                                local data. Does NOT authorize discarding
                                remote history. If origin has Dolt data
                                this will refuse — pair with
                                --discard-remote to override.

  issuegraph init --reinit-local \      Discard the remote's Dolt history and
      --discard-remote          replace it with the local reinit. First
                                issuegraph dolt push after this will be a
                                history-replacing force-push.

  issuegraph init --force               Deprecated alias for --reinit-local.
                                Kept working for ≥2 releases.

  issuegraph init --from-jsonl          Import from configured import.path. If
                                origin has Dolt data, this refuses unless
                                --discard-remote authorizes replacing that
                                remote history.

ADOPTING A REMOTE

  If you want to use the remote's existing history, use:

      issuegraph bootstrap

  issuegraph init will automatically suggest this when a remote is detected.

DESTROY-TOKEN (non-interactive only)

  When running with no TTY (CI, agents, piped input), --discard-remote
  requires an explicit --destroy-token value. The token format is:

      DESTROY-&lt;issue-prefix&gt;

  For example, if your issue prefix is "issuegraph", the token is "DESTROY-issuegraph":

      issuegraph init --reinit-local --discard-remote --destroy-token=DESTROY-issuegraph

  In interactive (TTY) mode you confirm via a typed prompt instead. The
  token is not echoed by issuegraph's runtime error messages — this is a
  deliberate guard against pattern-matched one-liners (see
  docs/adr/0002-init-safety-invariants.md).

EXIT CODES

  10    refused: remote has Dolt history and you selected local history
        without --discard-remote
  11    refused: existing local data and you declined the destroy confirm
  12    refused: --discard-remote passed without a valid --destroy-token
        (non-interactive mode)

RECOVERY

  If you hit a refusal, see docs/RECOVERY.md for step-by-step recovery
  playbooks for each exit code.


```
issuegraph init-safety
```

### issuegraph mail

Delegates mail operations to an external mail provider.

Agents often type 'issuegraph mail' when working with issuegraph, but mail functionality
is typically provided by the orchestrator. This command bridges that gap
by delegating to the configured mail provider.

Configuration (checked in order):
  1. ISSUEGRAPH_MAIL_DELEGATE or ISSUEGRAPH_MAIL_DELEGATE environment variable
  2. 'mail.delegate' config setting (issuegraph config set mail.delegate "gt mail")

Examples:
  # Configure delegation (one-time setup)
  `export ISSUEGRAPH_MAIL_DELEGATE="gt mail"`
  # or
  issuegraph config set mail.delegate "gt mail"

  # Then use issuegraph mail as if it were gt mail
  issuegraph mail inbox                    # Lists inbox
  issuegraph mail send mayor/ -s "Hi"      # Sends mail
  issuegraph mail read msg-123             # Reads a message

```
issuegraph mail [subcommand] [args...]
```

### issuegraph metrics

Show whether anonymous usage metrics are on, see exactly what is sent, and
turn them on or off.

issuegraph shares anonymous usage metrics to learn how people actually use it — just
which commands get run, plus the issuegraph version and OS platform. That's how we decide
what to polish next. We never collect your issues, paths, remotes, identity, or
any user-supplied text.

  issuegraph metrics            show the current status and what is collected
  issuegraph metrics on         turn metrics on
  issuegraph metrics off        turn metrics off
  issuegraph metrics example    show real examples of the events issuegraph sends

```
issuegraph metrics
```

#### issuegraph metrics example

Show real examples of the anonymous metrics issuegraph sends

```
issuegraph metrics example
```

#### issuegraph metrics off

Turn anonymous usage metrics off

```
issuegraph metrics off
```

#### issuegraph metrics on

Turn anonymous usage metrics on

```
issuegraph metrics on
```

### issuegraph mol

Manage molecules - work templates for agent workflows.

Protos are template epics with the "template" label. They define a DAG of work
that can be spawned to create real issues (molecules).

The molecule metaphor:
  - A proto is an uninstantiated template (reusable work pattern)
  - Spawning creates a molecule (real issues) from the proto
  - Variables (&#123;&#123;key&#125;&#125;) are substituted during spawning
  - Bonding combines protos or molecules into compounds
  - Distilling extracts a proto from an ad-hoc epic

Commands:
  show       Show proto/molecule structure and variables
  pour       Instantiate proto as persistent mol (liquid phase)
  wisp       Instantiate proto as ephemeral wisp (vapor phase)
  bond       Polymorphic combine: proto+proto, proto+mol, mol+mol
  squash     Condense molecule to digest
  burn       Discard wisp
  distill    Extract proto from ad-hoc epic

Use "issuegraph formula list" to list available formulas.

```
issuegraph mol
```

**Aliases:** protomolecule

#### issuegraph mol bond

Bond two protos or molecules to create a compound.

The bond command is polymorphic - it handles different operand types:

  formula + formula → cook both, compound proto
  formula + proto   → cook formula, compound proto
  formula + mol     → cook formula, spawn and attach
  proto + proto     → compound proto (reusable template)
  proto + mol       → spawn proto, attach to molecule
  mol + proto       → spawn proto, attach to molecule
  mol + mol         → join into compound molecule

Formula names (e.g., mol-polecat-arm) are cooked inline as ephemeral protos.
This avoids needing pre-cooked proto issuegraph in the database.

Bond types:
  sequential (default) - B runs after A completes
  parallel            - B runs alongside A
  conditional         - B runs only if A fails

Phase control:
  By default, spawned protos follow the target's phase:
  - Attaching to mol (Ephemeral=false) → spawns as persistent (Ephemeral=false)
  - Attaching to ephemeral issue (Ephemeral=true) → spawns as ephemeral (Ephemeral=true)

  Override with:
  --pour  Force spawn as liquid (persistent, Ephemeral=false)
  --ephemeral  Force spawn as vapor (ephemeral, Ephemeral=true, excluded from Dolt sync via dolt_ignore)

Dynamic bonding (Christmas Ornament pattern):
  Use --ref to specify a custom child reference with variable substitution.
  This creates IDs like "parent.child-ref" instead of random hashes.

  Example:
    issuegraph mol bond mol-worker-arm issuegraph-patrol --ref arm-&#123;&#123;worker_name&#125;&#125; --var worker_name=ace
    # Creates: issuegraph-patrol.arm-ace (and children like issuegraph-patrol.arm-ace.capture)

Use cases:
  - Found important bug during patrol? Use --pour to persist it
  - Need ephemeral diagnostic on persistent feature? Use --ephemeral
  - Spawning per-worker arms on a patrol? Use --ref for readable IDs

Examples:
  issuegraph mol bond mol-feature mol-deploy                    # Compound proto
  issuegraph mol bond mol-feature mol-deploy --type parallel    # Run in parallel
  issuegraph mol bond mol-feature issuegraph-abc123                     # Attach proto to molecule
  issuegraph mol bond issuegraph-abc123 issuegraph-def456                       # Join two molecules
  issuegraph mol bond mol-critical-bug wisp-patrol --pour       # Persist found bug
  issuegraph mol bond mol-temp-check issuegraph-feature --ephemeral          # Ephemeral diagnostic
  issuegraph mol bond mol-arm issuegraph-patrol --ref arm-&#123;&#123;name&#125;&#125; --var name=ace  # Dynamic child ID

```
issuegraph mol bond <A> <B> [flags]
```

**Aliases:** fart

**Flags:**

```
      --as string         Custom title for compound proto (proto+proto only)
      --dry-run           Preview what would be created
      --ephemeral         Force spawn as vapor (ephemeral, Ephemeral=true)
      --pour              Force spawn as liquid (persistent, Ephemeral=false)
      --ref string        Custom child reference with {{var}} substitution (e.g., arm-{{polecat_name}})
      --type string       Bond type: sequential, parallel, or conditional (default "sequential")
      --var stringArray   Variable substitution for spawned protos (key=value)
```

#### issuegraph mol burn

Burn a molecule, deleting it without creating a digest.

Unlike squash (which creates a permanent digest before deletion), burn
completely removes the molecule with no trace. Use this for:
  - Abandoned patrol cycles
  - Crashed or failed workflows
  - Test/debug molecules you don't want to preserve

The burn operation differs based on molecule phase:
  - Wisp (ephemeral): Direct delete
  - Mol (persistent): Cascade delete (syncs to remotes)

CAUTION: This is a destructive operation. The molecule's data will be
permanently lost. If you want to preserve a summary, use 'issuegraph mol squash'.

Example:
  issuegraph mol burn issuegraph-abc123              # Delete molecule with no trace
  issuegraph mol burn issuegraph-abc123 --dry-run    # Preview what would be deleted
  issuegraph mol burn issuegraph-abc123 --force      # Skip confirmation
  issuegraph mol burn issuegraph-a1 issuegraph-b2 issuegraph-c3      # Batch delete multiple wisps

```
issuegraph mol burn <molecule-id> [molecule-id...] [flags]
```

**Flags:**

```
      --dry-run   Preview what would be deleted
      --force     Skip confirmation prompt
```

#### issuegraph mol current

Show where you are in a molecule workflow.

If molecule-id is given, show status for that molecule.
If not given, infer from in_progress issues assigned to current agent.

The output shows all steps with status indicators:
  [done]     - Step is complete (closed)
  [current]  - Step is in_progress (you are here)
  [ready]    - Step is ready to start (unblocked)
  [blocked]  - Step is blocked by dependencies
  [pending]  - Step is waiting

For large molecules (&gt;100 steps), a summary is shown instead.
Use --limit or --range to view specific steps:
  issuegraph mol current &lt;id&gt; --limit 50       # Show first 50 steps
  issuegraph mol current &lt;id&gt; --range 100-150  # Show steps 100-150

```
issuegraph mol current [molecule-id] [flags]
```

**Flags:**

```
      --for string     Show molecules for a specific agent/assignee
      --limit int      Maximum number of steps to display (0 = auto, use 'all' threshold)
      --range string   Display specific step range (e.g., '1-50', '100-150')
```

#### issuegraph mol distill

Distill a molecule by extracting a reusable formula from an existing epic.

This is the reverse of pour: instead of formula → molecule, it's molecule → formula.

The distill command:
  1. Loads the existing epic and all its children
  2. Converts the structure to a .formula.json file
  3. Replaces concrete values with &#123;&#123;variable&#125;&#125; placeholders (via --var flags)

Use cases:
  - Team develops good workflow organically, wants to reuse it
  - Capture tribal knowledge as executable templates
  - Create starting point for similar future work

Variable syntax (both work - we detect which side is the concrete value):
  --var branch=feature-auth    Spawn-style: variable=value (recommended)
  --var feature-auth=branch    Substitution-style: value=variable

Output locations (first writable wins):
  1. &lt;resolved-issuegraph-dir&gt;/formulas/ (project-level, default)
  2. &lt;checkout-root&gt;/.issuegraph/formulas/ (repo-local formulas)
  3. ~/.issuegraph/formulas/     (user-level, if project not writable)

Examples:
  issuegraph mol distill issuegraph-o5xe my-workflow
  issuegraph mol distill issuegraph-abc release-workflow --var feature_name=auth-refactor

```
issuegraph mol distill <epic-id> [formula-name] [flags]
```

**Flags:**

```
      --dry-run           Preview what would be created
      --output string     Output directory for formula file
      --var stringArray   Replace value with {{variable}} placeholder (variable=value)
```

#### issuegraph mol last-activity

Show the most recent activity timestamp for a molecule.

Returns the timestamp of the most recent change to any step in the molecule,
making it easy to detect stale or stuck molecules.

Activity sources:
  step_closed      - A step was closed
  step_updated     - A step was updated (claimed, edited, etc.)
  molecule_updated - The molecule root itself was updated

Examples:
  issuegraph mol last-activity hq-wisp-0laki
  issuegraph mol last-activity hq-wisp-0laki --json

```
issuegraph mol last-activity <molecule-id>
```

#### issuegraph mol pour

Pour a proto into a persistent mol - like pouring molten metal into a mold.

This is the chemistry-inspired command for creating PERSISTENT work from templates.
The resulting mol lives in .issuegraph/ (permanent storage) and is synced with git.

Phase transition: Proto (solid) -&gt; pour -&gt; Mol (liquid)

WHEN TO USE POUR vs WISP:
  pour (liquid): Persistent work that needs audit trail
    - Feature implementations spanning multiple sessions
    - Work you may need to reference later
    - Anything worth preserving in git history

  wisp (vapor): Ephemeral work that auto-cleans up
    - Release workflows (one-time execution)
    - Operational loops and recurring cycles
    - Health checks and diagnostics
    - Any operational workflow without audit value

TIP: Formulas can specify phase:"vapor" to recommend wisp usage.
     If you pour a vapor-phase formula, you'll get a warning.

Examples:
  issuegraph mol pour mol-feature --var name=auth    # Persistent feature work
  issuegraph mol pour mol-review --var pr=123        # Persistent code review

```
issuegraph mol pour <proto-id> [flags]
```

**Flags:**

```
      --assignee string      Assign the root issue to this agent/user
      --attach strings       Proto to attach after spawning (repeatable)
      --attach-type string   Bond type for attachments: sequential, parallel, or conditional (default "sequential")
      --dry-run              Preview what would be created
      --var stringArray      Variable substitution (key=value)
```

#### issuegraph mol progress

Show efficient progress summary for a molecule.

This command uses indexed queries to count progress without loading all steps,
making it suitable for very large molecules (millions of steps).

If no molecule-id is given, shows progress for any molecule you're working on.

Output includes:
  - Progress: completed / total (percentage)
  - Current step: the in-progress step (if any)
  - Rate: steps/hour based on closure times
  - ETA: estimated time to completion

Example:
  issuegraph mol progress issuegraph-hanoi-xyz

```
issuegraph mol progress [molecule-id]
```

#### issuegraph mol ready

Find molecules where a gate has closed and the workflow is ready to resume.

This command discovers molecules waiting at a gate step where:
1. The molecule has a gate issue that blocks a step
2. The gate issue is now closed (condition satisfied)
3. The blocked step is now ready to proceed
4. No agent currently has this molecule hooked

This enables discovery-based resume without explicit waiter tracking.
The patrol system uses this to find and dispatch gate-ready molecules.

Examples:
  issuegraph mol ready --gated           # Find all gate-ready molecules
  issuegraph mol ready --gated --json    # JSON output for automation

```
issuegraph mol ready --gated
```

#### issuegraph mol seed

Verify that a formula is accessible and can be cooked.

The seed command checks formula search paths to ensure a formula exists
and can be loaded. This is useful for verifying system health before
attempting to spawn work from a formula.

Formula search paths (checked in order):
  1. &lt;resolved-issuegraph-dir&gt;/formulas/ (active project)
  2. &lt;checkout-root&gt;/.issuegraph/formulas/ (repo-local formulas)
  3. ~/.issuegraph/formulas/ (user level)
  4. $GT_ROOT/.issuegraph/formulas/ (shared workspace root, if GT_ROOT set)

Examples:
  issuegraph mol seed mol-feature                 # Verify specific formula
  issuegraph mol seed mol-review --var name=test  # Verify with variable substitution

```
issuegraph mol seed <formula-name> [flags]
```

**Flags:**

```
      --var stringArray   Variable substitution for condition filtering (key=value)
```

#### issuegraph mol show

Show molecule structure and details.

The --parallel flag highlights parallelizable steps:
  - Steps with no blocking dependencies can run in parallel
  - Shows which steps are ready to start now
  - Identifies parallel groups (steps that can run concurrently)

Example:
  issuegraph mol show issuegraph-patrol --parallel

```
issuegraph mol show <molecule-id> [flags]
```

**Flags:**

```
  -p, --parallel   Show parallel step analysis
```

#### issuegraph mol squash

Squash a molecule's ephemeral children into a single digest issue.

This command collects all ephemeral child issues of a molecule (Ephemeral=true),
generates a summary digest, and promotes the wisps to persistent by
clearing their Wisp flag (or optionally deletes them).

The squash operation:
  1. Loads the molecule and all its children
  2. Filters to only wisps (ephemeral issues with Ephemeral=true)
  3. Generates a digest (summary of work done)
  4. Creates a permanent digest issue (Ephemeral=false)
  5. Clears Wisp flag on children (promotes to persistent)
     OR keeps them with --keep-children (default: delete)

AGENT INTEGRATION:
Use --summary to provide an AI-generated summary. This keeps issuegraph as a pure
tool - the calling agent (orchestrator worker, Claude Code, etc.) is responsible
for generating intelligent summaries. Without --summary, a basic concatenation
of child issue content is used.

This is part of the wisp workflow: spawn creates wisps,
execution happens, squash compresses the trace into an outcome (digest).

Example:
  issuegraph mol squash issuegraph-abc123                    # Squash and promote children
  issuegraph mol squash issuegraph-abc123 --dry-run          # Preview what would be squashed
  issuegraph mol squash issuegraph-abc123 --keep-children    # Keep wisps after digest
  issuegraph mol squash issuegraph-abc123 --summary "Agent-generated summary of work done"

```
issuegraph mol squash <molecule-id> [flags]
```

**Flags:**

```
      --dry-run          Preview what would be squashed
      --keep-children    Don't delete ephemeral children after squash
      --summary string   Agent-provided summary (bypasses auto-generation)
```

#### issuegraph mol stale

Detect molecules (epics with children) that are complete but still open.

A molecule is considered stale if:
  1. All children are closed (Completed == Total)
  2. Root issue is still open
  3. Not assigned to anyone (optional, use --unassigned)
  4. Is blocking other work (optional, use --blocking)

By default, shows all complete-but-unclosed molecules.

Examples:
  issuegraph mol stale              # List all stale molecules
  issuegraph mol stale --json       # Machine-readable output
  issuegraph mol stale --blocking   # Only show those blocking other work
  issuegraph mol stale --unassigned # Only show unassigned molecules
  issuegraph mol stale --all        # Include molecules with 0 children

```
issuegraph mol stale [flags]
```

**Flags:**

```
      --all          Include molecules with 0 children
      --blocking     Only show molecules blocking other work
      --unassigned   Only show unassigned molecules
```

#### issuegraph mol wisp

Create or manage wisps - EPHEMERAL molecules for operational workflows.

When called with a proto-id argument, creates a wisp from that proto.
When called with a subcommand (list, gc), manages existing wisps.

Wisps are issues with Ephemeral=true in the main database. They're stored
locally but NOT synced via git.

WHEN TO USE WISP vs POUR:
  wisp (vapor): Ephemeral work that auto-cleans up
    - Release workflows (one-time execution)
    - Operational loops and recurring cycles
    - Health checks and diagnostics
    - Any operational workflow without audit value

  pour (liquid): Persistent work that needs audit trail
    - Feature implementations spanning multiple sessions
    - Work you may need to reference later
    - Anything worth preserving in git history

TIP: Formulas can specify phase:"vapor" to recommend wisp usage.
     If you use pour on a vapor-phase formula, you'll get a warning.

The wisp lifecycle:
  1. Create: issuegraph mol wisp &lt;proto&gt; or issuegraph create --ephemeral
  2. Execute: Normal issuegraph operations work on wisp issues
  3. Squash: issuegraph mol squash &lt;id&gt; (clears Ephemeral flag, promotes to persistent)
  4. Or burn: issuegraph mol burn &lt;id&gt; (deletes without creating digest)

Examples:
  issuegraph mol wisp issuegraph-release --var version=1.0  # Release workflow
  issuegraph mol wisp mol-my-workflow                  # Ephemeral operational cycle
  issuegraph mol wisp list                             # List all wisps
  issuegraph mol wisp gc                               # Garbage collect old wisps

Subcommands:
  list  List all wisps in current context
  gc    Garbage collect orphaned wisps

```
issuegraph mol wisp [proto-id] [flags]
```

**Flags:**

```
      --dry-run           Preview what would be created
      --root-only         Create only the root issue (no child step issues)
      --var stringArray   Variable substitution (key=value)
```

##### issuegraph mol wisp create

Create a wisp from a proto - sublimation from solid to vapor.

This is the chemistry-inspired command for creating ephemeral work from templates.
The resulting wisp is stored in the main database with Ephemeral=true and NOT synced via git.

Phase transition: Proto (solid) -&gt; Wisp (vapor)

Use wisp for:
  - Operational loops and recurring cycles
  - Health checks and monitoring
  - One-shot orchestration runs
  - Routine operations with no audit value

The wisp will:
  - Be stored in main database with Ephemeral=true flag
  - NOT be synced via git
  - Either evaporate (burn) or condense to digest (squash)

Examples:
  issuegraph mol wisp create mol-patrol                    # Ephemeral patrol cycle
  issuegraph mol wisp create mol-health-check              # One-time health check
  issuegraph mol wisp create mol-diagnostics --var target=db  # Diagnostic run

```
issuegraph mol wisp create <proto-id> [flags]
```

**Flags:**

```
      --dry-run           Preview what would be created
      --root-only         Create only the root issue (no child step issues)
      --var stringArray   Variable substitution (key=value)
```

##### issuegraph mol wisp gc

Garbage collect old or abandoned wisps from the database.

A wisp is considered abandoned if:
  - It hasn't been updated in --age duration and is not closed

Abandoned wisps are deleted without creating a digest. Use 'issuegraph mol squash'
if you want to preserve a summary before garbage collection.

Use --closed to purge ALL closed wisps (regardless of age). This is the
fastest way to reclaim space from accumulated wisp bloat. Safe by default:
requires --force to actually delete.

Note: This uses time-based cleanup, appropriate for ephemeral wisps.
For graph-pressure staleness detection (blocking other work), see 'issuegraph mol stale'.

Examples:
  issuegraph mol wisp gc                                    # Clean abandoned wisps (default: 1h threshold)
  issuegraph mol wisp gc --dry-run                          # Preview what would be cleaned
  issuegraph mol wisp gc --age 24h                          # Custom age threshold
  issuegraph mol wisp gc --all                              # Also clean closed wisps older than threshold
  issuegraph mol wisp gc --closed                           # Preview closed wisp deletion
  issuegraph mol wisp gc --closed --force                   # Delete all closed wisps
  issuegraph mol wisp gc --closed --dry-run                 # Explicit dry-run (same as no --force)
  issuegraph mol wisp gc --exclude-type agent,rig           # Protect agent and rig wisps from GC
  issuegraph mol wisp gc --closed --force --exclude-type mol # Delete closed wisps except mol type

```
issuegraph mol wisp gc [flags]
```

**Flags:**

```
      --age string             Age threshold for abandoned wisp detection (default "1h")
      --all                    Also clean closed wisps older than threshold
      --closed                 Delete all closed wisps (ignores --age threshold)
      --dry-run                Preview what would be cleaned
      --exclude-type strings   Exclude wisps of these types from GC (comma-separated, e.g., agent,rig)
  -f, --force                  Actually delete (default: preview only)
```

##### issuegraph mol wisp list

List all wisps (ephemeral molecules) in the current context.

Wisps are issues with Ephemeral=true in the main database. They are stored
locally but not synced via git.

The list shows:
  - ID: Issue ID of the wisp
  - Title: Wisp title
  - Status: Current status (open, in_progress, closed)
  - Started: When the wisp was created
  - Updated: Last modification time

Old wisp detection:
  - Old wisps haven't been updated in 24+ hours
  - Use 'issuegraph mol wisp gc' to clean up old/abandoned wisps

Examples:
  issuegraph mol wisp list              # List all wisps
  issuegraph mol wisp list --json       # JSON output for programmatic use
  issuegraph mol wisp list --all        # Include closed wisps

```
issuegraph mol wisp list [flags]
```

**Flags:**

```
      --all           Include closed wisps
      --type string   Filter by issue type (e.g., agent, task, patrol)
```

### issuegraph notion

Commands for syncing issues between issuegraph and Notion.

```
issuegraph notion
```

#### issuegraph notion connect

Connect issuegraph to an existing Notion database or data source

```
issuegraph notion connect [flags]
```

**Flags:**

```
      --url string   Existing Notion database or data source URL
```

#### issuegraph notion init

Create a dedicated IssueGraph database in Notion

```
issuegraph notion init [flags]
```

**Flags:**

```
      --parent string   Parent page ID
      --title string    Database title (default "IssueGraph Issues")
```

#### issuegraph notion pull

Pull one or more items from Notion.

Accepts issue IDs or external references as positional arguments.
Equivalent to: issuegraph notion sync --pull --issues &lt;refs&gt;

```
issuegraph notion pull [refs...] [flags]
```

**Flags:**

```
      --dry-run   Preview pull without making changes
```

#### issuegraph notion push

Push one or more issuegraph issues to Notion.

Accepts issue IDs as positional arguments.
Equivalent to: issuegraph notion sync --push --issues &lt;ids&gt;

```
issuegraph notion push [issue-ids...] [flags]
```

**Flags:**

```
      --dry-run   Preview push without making changes
```

#### issuegraph notion status

Show Notion sync status

```
issuegraph notion status
```

#### issuegraph notion sync

Synchronize issues between issuegraph and Notion.

By default this performs bidirectional sync. Use --pull or --push to limit direction.

```
issuegraph notion sync [flags]
```

**Flags:**

```
      --create-only     Only create missing remote pages, do not update existing ones
      --dry-run         Preview changes without making mutations
      --issues string   Comma-separated issue IDs to sync selectively (e.g., issuegraph-abc,issuegraph-def). Mutually exclusive with --parent.
      --parent string   Limit push to this issue and its descendants (push only). Mutually exclusive with --issues.
      --prefer-local    On conflict, keep the local issuegraph version
      --prefer-notion   On conflict, use the Notion version
      --pull            Only pull issues from Notion
      --push            Only push issues to Notion
      --state string    Issue state to sync: open, closed, or all (default "all")
```

### issuegraph orphans

Identify orphaned issues - issues that are referenced in commit messages but remain open or in_progress in the database.

This helps identify work that has been implemented but not formally closed.

Examples:
  issuegraph orphans              # Show orphaned issues
  issuegraph orphans --json       # Machine-readable output
  issuegraph orphans --details    # Show full commit information
  issuegraph orphans --fix        # Close orphaned issues with confirmation
  issuegraph orphans --label theme:personal             # Only orphans with this label
  issuegraph orphans --label-any theme:personal,theme:ventures  # Orphans with either label

```
issuegraph orphans [flags]
```

**Flags:**

```
      --details             Show full commit information
  -f, --fix                 Close orphaned issues with confirmation
  -l, --label strings       Filter by labels (AND: must have ALL). Can combine with --label-any
      --label-any strings   Filter by labels (OR: must have AT LEAST ONE). Can combine with --label
```

### issuegraph ready

Show ready work (open issues with no active blockers).

Excludes in_progress, blocked, deferred, and hooked issues. This uses the
GetReadyWork API which applies blocker-aware semantics to find truly claimable work.

Note: 'issuegraph list --ready' uses the same blocker-aware ready-work semantics.

Use --mol to filter to a specific molecule's steps:
  issuegraph ready --mol issuegraph-patrol   # Show ready steps within molecule

Use --gated to find molecules ready for gate-resume dispatch:
  issuegraph ready --gated           # Find molecules where a gate closed

Use --claim to atomically claim the first ready issue matching the filters:
  issuegraph ready --claim --json

This is useful for agents executing molecules to see which steps can run next.

```
issuegraph ready [flags]
```

**Flags:**

```
  -a, --assignee string              Filter by assignee
      --claim                        Atomically claim the first ready issue matching the filters
      --exclude-label strings        Exclude issues that have ANY of these labels
      --exclude-type strings         Exclude issue types from results (comma-separated or repeatable, e.g., --exclude-type=convoy,epic)
      --explain                      Show dependency-aware reasoning for why issues are ready or blocked
      --gated                        Find molecules ready for gate-resume dispatch
      --has-metadata-key string      Filter issues that have this metadata key set
      --include-deferred             Include issues with future defer_until timestamps
      --include-ephemeral            Include ephemeral issues (wisps) in results
  -l, --label strings                Filter by labels (AND: must have ALL). Can combine with --label-any
      --label-any strings            Filter by labels (OR: must have AT LEAST ONE). Can combine with --label
  -n, --limit int                    Maximum issues to show (use 0 for unlimited) (default 100)
      --metadata-field stringArray   Filter by metadata field (key=value, repeatable)
      --mol string                   Filter to steps within a specific molecule
      --mol-type string              Filter by molecule type: swarm, patrol, or work
      --offset int                   Skip the first N matching results (0-based). Only supported under --proxied-server.
      --parent string                Filter to descendants of this issue/epic
      --plain                        Display issues as a plain numbered list
      --pretty                       Display issues in a tree format with status/priority symbols (default true)
  -p, --priority int                 Filter by priority
  -s, --sort string                  Sort policy: priority (default), hybrid, oldest (default "priority")
  -t, --type string                  Filter by issue type (task, bug, feature, epic, decision, merge-request). Aliases: mr→merge-request, feat→feature, mol→molecule, dec/adr→decision
  -u, --unassigned                   Show only unassigned issues
```

### issuegraph rename

Rename an issue from one ID to another.

This updates:
- The issue's primary ID
- All references in other issues (descriptions, titles, notes, etc.)
- Dependencies pointing to/from this issue
- Labels, comments, and events

Examples:
  issuegraph rename issuegraph-w382l issuegraph-dolt     # Rename to memorable ID
  issuegraph rename gt-abc123 gt-auth    # Use descriptive ID

Note: The new ID must use a valid prefix for this database.

```
issuegraph rename <old-id> <new-id>
```

### issuegraph ship

Ship a capability to satisfy cross-project dependencies.

This command:
  1. Finds issue with export:&lt;capability&gt; label
  2. Validates issue is closed (or --force to override)
  3. Adds provides:&lt;capability&gt; label

External projects can depend on this capability using:
  issuegraph dep add &lt;issue&gt; external:&lt;project&gt;:&lt;capability&gt;

The capability is resolved when the external project has a closed issue
with the provides:&lt;capability&gt; label.

Examples:
  issuegraph ship mol-run-assignee              # Ship the mol-run-assignee capability
  issuegraph ship mol-run-assignee --force      # Ship even if issue is not closed
  issuegraph ship mol-run-assignee --dry-run    # Preview without making changes

```
issuegraph ship <capability> [flags]
```

**Flags:**

```
      --dry-run   Preview without making changes
      --force     Ship even if issue is not closed
```

### issuegraph undefer

Undefer issues to restore them to open status.

This brings issues back from the icebox so they can be worked on again.
Issues will appear in 'issuegraph ready' if they have no blockers.

Examples:
  issuegraph undefer issuegraph-abc        # Undefer a single issue
  issuegraph undefer issuegraph-abc issuegraph-def # Undefer multiple issues

```
issuegraph undefer [id...]
```

### issuegraph version

Print version information

```
issuegraph version
```
