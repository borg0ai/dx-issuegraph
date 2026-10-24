package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/subosito/gotenv"

	"github.com/borg0ai/dx-issuegraph/internal/config"
	"github.com/borg0ai/dx-issuegraph/internal/configfile"
	"github.com/borg0ai/dx-issuegraph/internal/debug"
	"github.com/borg0ai/dx-issuegraph/internal/doltserver"
	"github.com/borg0ai/dx-issuegraph/internal/hooks"
	"github.com/borg0ai/dx-issuegraph/internal/issuegraph"
	"github.com/borg0ai/dx-issuegraph/internal/metrics"
	"github.com/borg0ai/dx-issuegraph/internal/migration"
	"github.com/borg0ai/dx-issuegraph/internal/molecules"
	"github.com/borg0ai/dx-issuegraph/internal/remotecache"
	"github.com/borg0ai/dx-issuegraph/internal/routing"
	"github.com/borg0ai/dx-issuegraph/internal/storage"
	"github.com/borg0ai/dx-issuegraph/internal/storage/backends"
	"github.com/borg0ai/dx-issuegraph/internal/storage/dolt"
	dbidentifier "github.com/borg0ai/dx-issuegraph/internal/storage/domain/db"
	"github.com/borg0ai/dx-issuegraph/internal/storage/embeddeddolt"
	"github.com/borg0ai/dx-issuegraph/internal/storage/schema"
	"github.com/borg0ai/dx-issuegraph/internal/storage/uow"
	"github.com/borg0ai/dx-issuegraph/internal/telemetry"
	"github.com/borg0ai/dx-issuegraph/internal/ui"
	"github.com/borg0ai/dx-issuegraph/internal/utils"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

var (
	changeDir    string
	dbPath       string
	databaseFlag string
	actor        string
	store        storage.DoltStorage
	uowProvider  uow.UnitOfWorkProvider
	jsonOutput   bool

	// Signal-aware context for graceful cancellation
	rootCtx    context.Context
	rootCancel context.CancelFunc

	// Hook runner for extensibility
	hookRunner *hooks.Runner

	// Store concurrency protection
	storeMutex  sync.Mutex // Protects store access from background goroutine
	storeActive = false    // Tracks if store is available

	// Version upgrade tracking
	versionUpgradeDetected = false // Set to true if issuegraph version changed since last run
	previousVersion        = ""    // The last issuegraph version user had (empty = first run or unknown)
	upgradeAcknowledged    = false // Set to true after showing upgrade notification once per session
)

type envSnapshotValue struct {
	value string
	ok    bool
}

var changeDirEnvSnapshot map[string]envSnapshotValue

var (
	noColorFlag       bool
	sandboxMode       bool
	globalFlag        bool
	serverMode        bool
	proxiedServerMode bool
	readonlyMode      bool               // Read-only mode: block write operations (for worker sandboxes)
	storeIsReadOnly   bool               // Track if store was opened read-only (for staleness checks)
	ignoreSchemaSkew  bool               // Proceed despite forward schema drift
	lockTimeout       = 30 * time.Second // Dolt open timeout (fixed default)
	cpuProfileEnabled bool
	profileFile       *os.File
	traceFile         *os.File
	memProfilePath    string
	verboseFlag       bool // Enable verbose/debug output
	quietFlag         bool // Suppress non-essential output

	// Dolt auto-commit policy (flag/config). Values: off | on
	doltAutoCommit string

	// commandDidWrite is set when a command performs a write that should trigger
	// auto-flush. Used to decide whether to auto-commit Dolt after the command completes.
	// Thread-safe via atomic.Bool to avoid data races in concurrent flush operations.
	commandDidWrite atomic.Bool

	// commandMayEmptyJSONLExport is set by destructive maintenance commands
	// after they actually delete rows, allowing post-run auto-export to record
	// an intentional empty JSONL artifact instead of treating it as ambiguous.
	commandMayEmptyJSONLExport atomic.Bool

	// commandDeletedIssueIDs is the in-process handoff from the delete
	// commands to post-run auto-export: ids this command hard-deleted are
	// PROVEN gone, so the export's orphan guard skips them instead of
	// mistaking a deliberate delete for a torn store and refusing forever
	// (GH#5896). See deletedIssueIDSet for why it is not persisted.
	commandDeletedIssueIDs deletedIssueIDSet

	// commandDidExplicitDoltCommit is set when a command already created a Dolt commit
	// explicitly (e.g., issuegraph sync in dolt-native mode, hook flows, issuegraph vc commit).
	// This prevents a redundant auto-commit attempt in PersistentPostRun.
	commandDidExplicitDoltCommit bool

	// commandDidWriteTipMetadata is set when a command records a tip as "shown" by writing
	// metadata (tip_*_last_shown). This will be used to create a separate Dolt commit for
	// tip writes, even when the main command is read-only.
	commandDidWriteTipMetadata bool

	// commandTipIDsShown tracks which tip IDs were shown in this command (deduped).
	// This is used for tip-commit message formatting.
	commandTipIDsShown map[string]struct{}

	// commandFreeze is the migration-freeze lookup for this invocation,
	// resolved once in PersistentPreRunE (dc-6jaq). Both hooks read it: the
	// pre-run gate refuses writes from it, and PersistentPostRunE skips its
	// own maintenance writes — auto-commit, tip metadata, backup, export,
	// push — so a read command run during a freeze does not leave a new Dolt
	// commit in the store being migrated. Zero value means "not frozen",
	// which is the right default for the paths that never resolve it.
	commandFreeze migration.Result

	// commandSpan is the root OTel span for the current command execution.
	// All storage and AI spans are nested as children of this span.
	commandSpan oteltrace.Span
)

// skipStoreAnnotation, when set to "1" on a command (or any of its ancestors),
// makes issuegraph skip database/store initialization for that command — the
// annotation-based equivalent of listing the command name in noDbCommands. It
// lets commands defined in other files or build-tagged variants opt out of the
// store gate locally, without editing the central noDbCommands list.
const skipStoreAnnotation = "issuegraph:skip_store"

// commandOptsOutOfStore reports whether cmd or any of its ancestors carries the
// skipStoreAnnotation set to "1". The whole ancestor chain is walked, so
// annotating a command exempts that command and every subcommand beneath it.
// (This is broader than the noDbCommands list, which only matches a command
// name or its direct parent — annotate deliberately, on the specific command
// you want to skip the store.)
func commandOptsOutOfStore(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		if c.Annotations[skipStoreAnnotation] == "1" {
			return true
		}
	}
	return false
}

// readOnlyCommands lists commands that only read from the database.
// These commands open the store in read-only mode. See GH#804.
var readOnlyCommands = map[string]bool{
	"list":       true,
	"ready":      true,
	"show":       true,
	"stats":      true,
	"blocked":    true,
	"count":      true,
	"search":     true,
	"query":      true,
	"graph":      true,
	"duplicates": true,
	"comments":   true, // list comments (not add)
	"current":    true, // issuegraph sync mode current
	"ping":       true,
	"backup":     true, // reads from Dolt, writes only to .issuegraph/backup/
	"export":     true, // reads from Dolt, writes JSONL to file/stdout
	"tail":       true, // issuegraph events tail: reads issuegraph_events_journal, writes nothing
}

// isReadOnlyCommand returns true if the command only reads from the database.
// This is used to open the store in read-only mode, preventing file modifications
// that would trigger file watchers. See GH#804.
func isReadOnlyCommand(cmdName string) bool {
	return readOnlyCommands[cmdName]
}

// isPreviewCommand reports whether cmd was explicitly invoked in a
// non-mutating preview mode. Preview flags are command-local rather than
// persistent, so checking them here after Cobra has parsed the selected
// command is the earliest reliable point to keep the store open read-only.
func isPreviewCommand(cmd *cobra.Command) bool {
	for _, name := range []string{"dry-run", "inspect"} {
		if flag := cmd.Flags().Lookup(name); flag != nil {
			enabled, err := cmd.Flags().GetBool(name)
			if err == nil && enabled {
				return true
			}
		}
	}
	return false
}

type rootStorePolicy struct {
	readOnly         bool
	disableAutoStart bool
	runMaintenance   bool
}

// effectiveRootStorePolicy separates strict --readonly/config policy from
// command classification. Classified reads retain their compatibility
// maintenance and auto-start behavior; strict readonly is mutation-free.
func effectiveRootStorePolicy(cmdName string, strictReadonly bool) rootStorePolicy {
	return rootStorePolicy{
		readOnly:         strictReadonly || isReadOnlyCommand(cmdName),
		disableAutoStart: strictReadonly,
		runMaintenance:   !strictReadonly,
	}
}

// runsPostCommandMaintenance reports whether PersistentPostRunE should run the
// post-command maintenance net — Dolt auto-commit, the tip-metadata commit,
// auto-backup, auto-export and auto-push.
//
// `issuegraph serve` is excluded, and not as an optimization. Those steps are per-
// COMMAND bookkeeping, and a server is not a command: it is a process that ran
// for hours and committed each mutation inside its own transaction as it
// happened. Running them when the operator finally sends SIGTERM would push and
// export on the way out of a signal handler — the worst possible moment — and
// attribute a whole process lifetime of requests to the shutdown. Proxied-mode
// serve never reached this branch at all (PersistentPostRunE only closes the
// provider there); server and shared-server mode do, so the exclusion has to be
// stated rather than inherited.
func runsPostCommandMaintenance(cmdName string, strictReadonly bool) bool {
	if cmdName == serveCmdName {
		return false
	}
	return effectiveRootStorePolicy(cmdName, strictReadonly).runMaintenance
}

// resolveDoltServerConnection fills in how to reach the workspace's Dolt SQL
// server — host, port, socket, user, password, TLS — on doltCfg.
//
// Both consumers of a SQL server in this process go through here: the CLI's own
// store open, and the unit-of-work provider `issuegraph serve` builds for a server-mode
// workspace. That matters more than the deduplication: an HTTP request and a
// CLI command in the same workspace must reach the same server as the same
// identity, and the only way to guarantee that is to resolve it once.
//
// It mirrors dolt.applyResolvedConfig, which this hand-built doltCfg path
// bypasses.
func resolveDoltServerConnection(ctx context.Context, issuegraphDir string, fileCfg *configfile.Config, doltCfg *dolt.Config) error {
	doltCfg.ServerHost = fileCfg.GetDoltServerHost()
	// Port 0 is fine here — auto-start will resolve it. Use the shared helper
	// rather than DefaultConfig(...).Port: this hand-built doltCfg is handed
	// straight to dolt.New, and a port arriving there without its source is
	// read as a caller assertion (see ApplyResolvedServerPort).
	dolt.ApplyResolvedServerPort(issuegraphDir, doltCfg)
	doltCfg.ServerSocket = fileCfg.GetDoltServerSocket()
	// A configured credential command targets an authenticating gateway server:
	// run it for a short-lived token used as the connection username. Fail closed
	// — never fall back to the static/root user when a command was configured but
	// failed. Server mode only: embedded stores never present a username, so the
	// command must not run (or fail) embedded opens even when the env var is set.
	// Dolt-only: the gateway credential command mints a Dolt server
	// username. IsSharedServerMode() forces ServerMode true with no backend
	// guard, so non-Dolt metadata must not try to resolve a server username.
	if doltCfg.ServerMode && fileCfg.GetBackend() == configfile.BackendDolt {
		if _, err := dolt.ApplyGatewayCredential(ctx, fileCfg, doltCfg); err != nil {
			return fmt.Errorf("resolving dolt credential command: %w", err)
		}
	}
	if doltCfg.ServerUser == "" {
		doltCfg.ServerUser = fileCfg.GetDoltServerUser()
	}
	// Use the resolved port for credential lookup — metadata.json port
	// and runtime port can diverge (e.g., tunnel on 3308 vs local on 3307).
	doltCfg.ServerPassword = fileCfg.GetDoltServerPasswordForPort(doltCfg.ServerPort)
	doltCfg.ServerTLS = fileCfg.GetDoltServerTLS()
	return nil
}

var (
	runPostRunAutoCommit = maybeAutoCommit
	runPostRunAutoBackup = maybeAutoBackup
	runPostRunAutoExport = maybeAutoExport
	runPostRunAutoPush   = maybeAutoPush
)

// isWorkingSetReconcileCommand reports whether cmd's whole purpose is to
// reconcile the Dolt working set: "issuegraph dolt commit" or "issuegraph vc commit". These
// commands are the documented recovery from a pending-migration dirty-table
// refusal, but they also open the store, and an open runs the migration -
// hitting that same refusal before the commit that would clear the dirty
// state ever runs. Opening leniently (embeddeddolt.OpenForWorkingSetReconcile)
// breaks that deadlock by skipping the migration instead of failing the open
// (borg0ai/dx-issuegraph#4566).
func isWorkingSetReconcileCommand(cmd *cobra.Command) bool {
	if cmd.Name() != "commit" {
		return false
	}
	parent := cmd.Parent()
	if parent == nil {
		return false
	}
	return parent.Name() == "dolt" || parent.Name() == "vc"
}

// isRemoteSyncCommand reports whether cmd is `issuegraph dolt pull`: the one command
// the #6575 data-behind migrate-gate refusal tells the operator to run.
//
// It is the same deadlock isWorkingSetReconcileCommand breaks for #4566, one
// refusal over. The gate stops a data-behind clone from migrating and names
// `issuegraph dolt pull` as the remedy — but the pull opens the store too, so it hit
// that refusal before it could clear its cause. On an embedded clone there is
// no external `dolt` binary to fall back to, which left the refused clone with
// exactly one exit: ISSUEGRAPH_ALLOW_REMOTE_MIGRATE=1, i.e. performing the migration
// the refusal exists to prevent. Opening via embeddeddolt.OpenForRemoteSync /
// dolt.Config.RemoteSyncOpen tolerates that ONE gate reason and nothing else.
//
// Deliberately just the pull, not `issuegraph sync`: sync also pushes and can write
// issue rows, and a write against a stale schema is the hazard the gate is
// about. The pull only moves the commit graph, which is precisely the
// precondition the refusal is waiting on.
func isRemoteSyncCommand(cmd *cobra.Command) bool {
	if cmd.Name() != "pull" {
		return false
	}
	parent := cmd.Parent()
	return parent != nil && parent.Name() == "dolt"
}

// isForcedMigrate reports whether cmd is `issuegraph migrate` or `issuegraph migrate schema`
// invoked with --force: the operator confirming they are the single designated
// migrator, so the remote-migrate gate (#4259) must not block this run's store
// opens. Consulted in the root PersistentPreRunE because the gate fires during
// store open (and during autoMigrateOnVersionBump), long before the migrate
// command's own RunE.
func isForcedMigrate(cmd *cobra.Command) bool {
	if cmd != migrateCmd && cmd != migrateSchemaCmd {
		return false
	}
	force, _ := cmd.Flags().GetBool("force")
	return force
}

// printGlobalDatabaseConsentHint adds the one thing the gate's own block
// cannot know: which database this invocation was aimed at. Under --global the
// open targets `issuegraph_global`, so the block's `issuegraph migrate schema` would
// migrate the PROJECT database and leave the refusal in place — the working
// remedy is the same verb with the same flag.
//
// It takes the refusal because "the same verb" is not the same verb on every
// arm: the #6575 data-behind stop on a shared store is remote-backed by
// construction, and there the bare verb's consent is never read (see
// schema.SharedConsentCommandForced), so retargeting the bare form would hand
// the operator a global-scoped command that still cannot succeed. This mirrors
// the retarget in handleRemoteMigrateGateJSON. A nil error keeps the
// pre-existing bare-verb wording.
func printGlobalDatabaseConsentHint(w io.Writer, e *schema.RemoteMigrateGateError) {
	if !globalFlag {
		return
	}
	consent := schema.SharedConsentCommandGlobal
	if e != nil && e.IsDataBehind() && e.Shared {
		consent = schema.SharedConsentCommandForcedGlobal
	}
	fmt.Fprintf(w,
		"\n  This command targeted the global database (--global), so run the\n"+
			"  migrate step with the same flag:\n"+
			"        %s\n",
		consent)
}

// renderTypedOpenError prints the actionable block for the store-open failures
// that carry one, honoring --json, and reports whether it handled err. A false
// return means the caller should fall back to its own generic message.
//
// Every open path needs this — the Dolt store, the proxied unit-of-work
// provider, and `issuegraph serve`'s startup. The proxied one used to render every
// failure as `%v` inside "failed to open uow provider", which turned a
// multi-line rebuild or migrate-consent guide into one truncated line.
func renderTypedOpenError(err error) bool {
	// Schema skew gets dedicated UX with actionable rebuild instructions.
	var skewErr *schema.SchemaSkewError
	if errors.As(err, &skewErr) {
		if jsonOutput {
			handleSchemaSkewJSON(skewErr)
		} else {
			fmt.Fprint(os.Stderr, skewErr.UserMessage())
		}
		return true
	}
	// #4259 / #5920: the migrate gate blocks a silent in-place migration and
	// tells the operator to migrate, adopt, or consent.
	var gateErr *schema.RemoteMigrateGateError
	if errors.As(err, &gateErr) {
		if jsonOutput {
			handleRemoteMigrateGateJSON(gateErr)
		} else {
			fmt.Fprint(os.Stderr, gateErr.UserMessage())
			printGlobalDatabaseConsentHint(os.Stderr, gateErr)
		}
		return true
	}
	return false
}

// isSchemaMigrateVerb reports whether cmd is `issuegraph migrate schema` — the one
// invocation in which the operator asked for a schema migration by name. That
// request is the consent the shared-store gate wants for a database with no
// remote (#5920); see schema.SetSharedMigrateConsent.
//
// Deliberately just this one command, not the `issuegraph migrate` tree. Bare
// `issuegraph migrate` reconciles version/repo-id/clone-id metadata and never applies
// a migration itself, and its flag modes are further from schema work still:
// `issuegraph migrate --update-repo-id` is repo-fingerprint surgery after a git remote
// change, and treating it as consent would let a repo-ID update promote the
// schema for every co-resident client as a side effect. `issuegraph migrate sync`,
// `issuegraph migrate hooks`, and the mode-switch verbs consent to nothing either.
// The operator who does want to migrate has the verb this names, and
// `--force` still unlocks from either migrate command.
func isSchemaMigrateVerb(cmd *cobra.Command) bool {
	return cmd == migrateSchemaCmd
}

// forcedMigratePreviewFlag returns the name of a preview flag (--dry-run,
// --inspect) that conflicts with --force on a forced migrate invocation, or ""
// when there is no conflict. The combination must be rejected BEFORE the store
// opens: with the gate override set, the open itself applies pending schema
// migrations, so the preview flag would be honored only after the destructive
// work it exists to prevent had already happened.
func forcedMigratePreviewFlag(cmd *cobra.Command) string {
	for _, name := range []string{"dry-run", "inspect"} {
		if v, err := cmd.Flags().GetBool(name); err == nil && v {
			return name
		}
	}
	return ""
}

// applyNoColorFlag disables colorized output when --no-color is set.
// Complements the NO_COLOR / CLICOLOR=0 env detection in package ui,
// giving callers a per-invocation override.
func applyNoColorFlag() {
	if noColorFlag {
		ui.DisableColors()
	}
}

// loadIssueGraphEnvFile loads .issuegraph/.env into process environment for per-project
// Dolt credentials (GH#2520). Uses gotenv.Load which is non-overriding —
// existing shell env vars always take precedence.
// Safe to call with an empty issuegraphDir (no-op).
func loadIssueGraphEnvFile(issuegraphDir string) {
	if issuegraphDir == "" {
		return
	}
	envFile := filepath.Join(issuegraphDir, ".env")
	if _, err := os.Stat(envFile); err != nil {
		return
	}
	_ = gotenv.Load(envFile)
}

func logConfigDiscovery(issuegraphDir, reason string) {
	metadataPath := filepath.Join(issuegraphDir, configfile.ConfigFileName)
	configYAMLPath := filepath.Join(issuegraphDir, "config.yaml")
	_, metadataErr := os.Stat(metadataPath)
	_, yamlErr := os.Stat(configYAMLPath)
	debug.Logf("Debug: %s at %s -> metadata=%v (%v), config.yaml=%v (%v)\n",
		reason, issuegraphDir, metadataErr == nil, metadataErr, yamlErr == nil, yamlErr)
}

func shouldLogDefaultDoltDatabase(cfg *configfile.Config) bool {
	return cfg != nil && cfg.DoltDatabase == "" && os.Getenv("ISSUEGRAPH_DOLT_SERVER_DATABASE") == ""
}

// loadIssueGraphSelectionEnvFile loads only the selector keys needed for early
// workspace/database discovery. Unlike loadIssueGraphEnvFile, this intentionally
// limits itself to ISSUEGRAPH_DIR / ISSUEGRAPH_DB / ISSUEGRAPH_DB so caller credentials and
// runtime knobs do not leak into explicit-target commands before rebinding.
func loadIssueGraphSelectionEnvFile(issuegraphDir string) {
	if issuegraphDir == "" {
		return
	}
	envFile := filepath.Join(issuegraphDir, ".env")
	pairs, err := gotenv.Read(envFile)
	if err != nil {
		return
	}
	for _, key := range []string{"ISSUEGRAPH_DIR", "ISSUEGRAPH_DB", "ISSUEGRAPH_DB"} {
		if os.Getenv(key) != "" {
			continue
		}
		if value, ok := pairs[key]; ok && strings.TrimSpace(value) != "" {
			_ = os.Setenv(key, value)
		}
	}
}

// loadSelectionEnvironment loads only the selector keys required to discover
// the target workspace/database before the store-init path runs. This preserves
// historical support for .issuegraph/.env files that route commands via ISSUEGRAPH_DB or
// ISSUEGRAPH_DIR without importing the caller workspace's broader runtime settings.
func loadSelectionEnvironment() {
	if os.Getenv("ISSUEGRAPH_DIR") != "" || os.Getenv("ISSUEGRAPH_DB") != "" || os.Getenv("ISSUEGRAPH_DB") != "" {
		return
	}
	if issuegraphDir := issuegraph.FindIssueGraphDir(); issuegraphDir != "" {
		loadIssueGraphSelectionEnvFile(issuegraphDir)
	}
}

// loadEnvironment runs the lightweight, always-needed environment setup that
// must happen before the noDbCommands early return. This ensures commands like
// "issuegraph doctor --server" pick up per-project Dolt credentials from .issuegraph/.env.
//
// This function intentionally does NOT do any store initialization, auto-migrate,
// or telemetry setup — those belong in the store-init phase that runs after the
// noDbCommands check.
func loadEnvironment() {
	// FindIssueGraphDir is lightweight (filesystem walk, no git subprocesses)
	// and resolves ISSUEGRAPH_DIR, redirects, and worktree paths.
	if issuegraphDir := issuegraph.FindIssueGraphDir(); issuegraphDir != "" {
		loadIssueGraphEnvFile(issuegraphDir)
		// Non-fatal warning if .issuegraph/ directory has overly permissive access.
		config.CheckIssueGraphDirPermissions(issuegraphDir)
	}
}

var sharedServerEmbeddedMismatchWarned bool

// warnSharedServerEmbeddedMismatch detects the case where shared-server mode
// is active but metadata.json explicitly pins dolt_mode=embedded. The
// shared-server setting wins for this invocation (GH#2946/2949: stale embedded
// metadata must not hide server-backed issue state), but issuegraph never rewrites the
// committed metadata.json — per-machine environment must not leak into shared
// config (bd-6dnrw.5). Print guidance so the user resolves the conflict
// explicitly.
func warnSharedServerEmbeddedMismatch(cfg *configfile.Config) {
	if cfg == nil || sharedServerEmbeddedMismatchWarned {
		return
	}
	if strings.ToLower(strings.TrimSpace(cfg.DoltMode)) != configfile.DoltModeEmbedded {
		return
	}
	if !doltserver.IsSharedServerMode() {
		return
	}
	sharedServerEmbeddedMismatchWarned = true
	fmt.Fprintln(os.Stderr, "Notice: shared-server mode is enabled (ISSUEGRAPH_DOLT_SHARED_SERVER or dolt.shared-server in config.yaml) but .issuegraph/metadata.json pins dolt_mode=\"embedded\". Using the shared server for this run.")
	fmt.Fprintln(os.Stderr, "  To persist server mode: set dolt_mode to \"server\" in .issuegraph/metadata.json and commit it.")
	fmt.Fprintln(os.Stderr, "  To stay embedded: unset ISSUEGRAPH_DOLT_SHARED_SERVER (or remove dolt.shared-server from config.yaml).")
}

// loadServerModeFromIssueGraphDir loads the storage mode (embedded vs server vs
// proxied-server) from the given issuegraph directory's metadata.json so that
// usesSQLServer() and usesProxiedServer() return the correct values.
//
// A metadata.json that exists but cannot be loaded is a hard error: treating
// it like an absent file silently flips server-mode deployments onto the
// embedded store, where every query answers from an empty relic with exit 0
// (false-empty). Absent metadata.json (cfg == nil) keeps the fresh-repo
// embedded default.
func loadServerModeFromIssueGraphDir(issuegraphDir string) error {
	if issuegraphDir == "" {
		return nil
	}
	cfg, err := configfile.LoadForDiscovery(issuegraphDir)
	if err != nil {
		return fmt.Errorf("load %s: %w; no storage database was opened or modified (storage mode unknown; data commands refuse to fall back to the embedded store)", configfile.ConfigPath(issuegraphDir), err)
	}
	// Absent metadata.json keeps the fresh-repo embedded default unless
	// env/config.yaml supply a remote host (GH#3545) — inference must not
	// depend on metadata existing.
	cfg = normalizeLoadedConfig(cfg)
	warnSharedServerEmbeddedMismatch(cfg)
	psm := cfg.IsDoltProxiedServerMode()
	sm := cfg.IsDoltServerMode()
	// GH#2946: shared-server override for stale metadata.json (no-db commands)
	if !sm && !psm && doltserver.IsSharedServerMode() {
		sm = true
	}
	serverMode = sm
	proxiedServerMode = psm
	if cmdCtx != nil {
		cmdCtx.ServerMode = sm
		cmdCtx.ProxiedServerMode = psm
	}
	return nil
}

// loadServerModeFromConfig loads the storage mode (embedded vs server vs
// proxied-server) from metadata.json so that usesSQLServer() and
// usesProxiedServer() return the correct values. Called for commands that
// skip full DB init but still need to know the mode.
func loadServerModeFromConfig() error {
	return loadServerModeFromIssueGraphDir(issuegraph.FindIssueGraphDir())
}

func preserveRedirectSourceDatabase(issuegraphDir string) {
	if issuegraphDir == "" || os.Getenv("ISSUEGRAPH_DOLT_SERVER_DATABASE") != "" {
		return
	}

	rInfo := issuegraph.ResolveRedirect(issuegraphDir)
	if rInfo.WasRedirected && rInfo.SourceDatabase != "" {
		_ = os.Setenv("ISSUEGRAPH_DOLT_SERVER_DATABASE", rInfo.SourceDatabase)
		if os.Getenv("ISSUEGRAPH_DEBUG_ROUTING") != "" {
			fmt.Fprintf(os.Stderr, "[routing] Preserved source dolt_database %q across redirect\n", rInfo.SourceDatabase)
		}
	}
}

func selectedNoDBIssueGraphDir(cmd *cobra.Command) string {
	if cmd != nil && cmd.Root() != nil && cmd.Root().PersistentFlags().Changed("db") && dbPath != "" {
		if selectedIssueGraphDir := resolveCommandIssueGraphDir(dbPath); selectedIssueGraphDir != "" {
			return selectedIssueGraphDir
		}
	} else if cmd != nil && cmd.PersistentFlags().Changed("db") && dbPath != "" {
		if selectedIssueGraphDir := resolveCommandIssueGraphDir(dbPath); selectedIssueGraphDir != "" {
			return selectedIssueGraphDir
		}
	} else if envDB := os.Getenv("ISSUEGRAPH_DB"); envDB != "" {
		if selectedIssueGraphDir := resolveCommandIssueGraphDir(envDB); selectedIssueGraphDir != "" {
			return selectedIssueGraphDir
		}
	} else if envDB := os.Getenv("ISSUEGRAPH_DB"); envDB != "" {
		if selectedIssueGraphDir := resolveCommandIssueGraphDir(envDB); selectedIssueGraphDir != "" {
			return selectedIssueGraphDir
		}
	}
	if os.Getenv("ISSUEGRAPH_DIR") != "" {
		if selectedIssueGraphDir := issuegraph.FindIssueGraphDir(); selectedIssueGraphDir != "" {
			return selectedIssueGraphDir
		}
	}
	if dbPath != "" {
		if selectedIssueGraphDir := resolveCommandIssueGraphDir(dbPath); selectedIssueGraphDir != "" {
			return selectedIssueGraphDir
		}
	}
	return issuegraph.FindIssueGraphDir()
}

func isSelectedNoDBCommand(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}
	if cmd.Name() == "context" || cmd.Name() == "where" {
		return true
	}
	if cmd.Parent() == nil || cmd.Parent().Name() != "dolt" {
		return false
	}
	switch cmd.Name() {
	case "push", "pull", "commit":
		return false
	default:
		return true
	}
}

// configCommandCanRunWithoutStore returns true for config subcommands whose Run
// path can execute without an opened Dolt store. This lets no-workspace calls
// fail or degrade in the command itself instead of tripping low-level DB init.
func configCommandCanRunWithoutStore(cmd *cobra.Command, args []string) bool {
	if cmd == nil || cmd.Parent() == nil || cmd.Parent().Name() != "config" {
		return false
	}

	switch cmd.Name() {
	case "show", "validate", "drift", "apply":
		return true
	case "set", "get", "unset":
		if len(args) == 0 {
			return true
		}
		key := args[0]
		return config.IsYamlOnlyKey(key) || key == "issuegraph.role"
	case "set-many":
		if len(args) == 0 {
			return true
		}
		for _, arg := range args {
			key, _, ok := strings.Cut(arg, "=")
			if !ok || key == "" {
				return true
			}
			if !config.IsYamlOnlyKey(key) && key != "issuegraph.role" {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func prepareSelectedCommandContext(issuegraphDir string, loadEnv bool) {
	if issuegraphDir == "" {
		return
	}
	_ = os.Setenv("ISSUEGRAPH_DIR", issuegraphDir)
	if loadEnv {
		loadIssueGraphEnvFile(issuegraphDir)
	}
	preserveRedirectSourceDatabase(issuegraphDir)
	if err := config.Initialize(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to reinitialize config for selected issuegraph dir: %v\n", err)
	}
	config.CheckIssueGraphDirPermissions(issuegraphDir)
	if err := loadServerModeFromIssueGraphDir(issuegraphDir); err != nil {
		// Warn, don't fatal: this context also serves no-DB commands —
		// doctor, init, bootstrap, config — which are exactly the repair
		// paths for a corrupt metadata.json. Data commands stay protected
		// by the hard error at store init and in the store factories.
		fmt.Fprintf(os.Stderr, "warning: %v\n", err)
	}
}

func prepareSelectedNoDBContext(issuegraphDir string) {
	prepareSelectedCommandContext(issuegraphDir, true)
}

func commandJSONFlagChanged(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}
	if cmd.Flags().Changed("json") {
		return true
	}
	root := cmd.Root()
	return root != nil && root.PersistentFlags().Changed("json")
}

// refreshBoundCommandConfig reapplies config-backed defaults after the command
// context has been rebound to a resolved target issuegraph directory. This keeps
// explicit flags authoritative while letting rerouted/explicit-db commands use
// the target repo's config rather than the caller's config.
func refreshBoundCommandConfig(cmd *cobra.Command) {
	if cmd == nil {
		return
	}
	root := cmd.Root()
	if root == nil {
		root = cmd
	}
	if !commandJSONFlagChanged(cmd) && !root.PersistentFlags().Changed("format") {
		jsonOutput = config.GetBool("json")
	}
	if !root.PersistentFlags().Changed("readonly") {
		readonlyMode = config.GetBool("readonly")
	}
	if !root.PersistentFlags().Changed("actor") {
		actor = resolveConfiguredActor()
	}
	if !root.PersistentFlags().Changed("dolt-auto-commit") {
		doltAutoCommit = config.GetString("dolt.auto-commit")
	}
}

// resolveCommandIssueGraphDir maps a discovered Dolt data path back to the owning
// .issuegraph directory. filepath.Dir(dbPath) only works when the Dolt data lives
// under .issuegraph/dolt; custom dolt_data_dir values can place it elsewhere.
func resolveCommandIssueGraphDir(dbPath string) string {
	if dbPath == "" {
		return ""
	}

	// Use the same validated candidate logic as the helper/reopen path
	// (GH#2627). This checks filepath.Dir, canonicalized paths, AND
	// FindIssueGraphDir — but only returns a candidate whose metadata.json
	// actually points to dbPath, preventing CWD discovery from overriding
	// an explicit --db flag.
	if issuegraphDir := resolveIssueGraphDirForDBPath(dbPath); issuegraphDir != "" {
		return issuegraphDir
	}

	for dir := filepath.Dir(dbPath); dir != "" && dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
		candidate := filepath.Join(dir, ".issuegraph")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}

	// No candidate matched — fall back to parent directory of the db path.
	// This handles bootstrap/init where no metadata.json exists yet.
	return filepath.Dir(dbPath)
}

// resolveConfiguredActor returns the actor implied by the current environment
// or configuration when no explicit --actor flag was passed.
func resolveConfiguredActor() string {
	if issuegraphActor := os.Getenv("ISSUEGRAPH_ACTOR"); issuegraphActor != "" {
		return issuegraphActor
	}
	return config.GetString("actor")
}

// getActorWithGit returns the actor for audit trails with git config fallback.
// Priority: --actor flag > ISSUEGRAPH_ACTOR env > git config user.name > $USER > "unknown"
// This provides a sensible default for developers: their git identity is used unless
// explicitly overridden
func getActorWithGit() string {
	// If actor is already set (from --actor flag), use it
	if actor != "" {
		return actor
	}

	// Check the actor environment override.
	if issuegraphActor := os.Getenv("ISSUEGRAPH_ACTOR"); issuegraphActor != "" {
		return issuegraphActor
	}

	// Try git config user.name - the natural default for a git-native tool
	if out, err := exec.Command("git", "config", "user.name").Output(); err == nil {
		if gitUser := strings.TrimSpace(string(out)); gitUser != "" {
			return gitUser
		}
	}

	// Fall back to system username
	if user := os.Getenv("USER"); user != "" {
		return user
	}

	return "unknown"
}

// getOwner returns the human owner for CV attribution.
// Priority: GIT_AUTHOR_EMAIL env > git config user.email > "" (empty)
// This is the foundation for HOP CV (curriculum vitae) chains per Decision 008.
// Unlike actor (which tracks who executed), owner tracks the human responsible.
func getOwner() string {
	// Check GIT_AUTHOR_EMAIL first - this is set during git commit operations
	if authorEmail := os.Getenv("GIT_AUTHOR_EMAIL"); authorEmail != "" {
		return authorEmail
	}

	// Fall back to git config user.email - the natural default
	if out, err := exec.Command("git", "config", "user.email").Output(); err == nil {
		if gitEmail := strings.TrimSpace(string(out)); gitEmail != "" {
			return gitEmail
		}
	}

	// Return empty if no email found (owner is optional)
	return ""
}

func init() {
	// Initialize viper configuration
	if err := config.Initialize(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to initialize config: %v\n", err)
	}

	// Register persistent flags
	rootCmd.PersistentFlags().StringVarP(&changeDir, "directory", "C", "", "Change to this directory before running the command (like git -C)")
	rootCmd.PersistentFlags().StringVar(&dbPath, "db", "", "Database path (default: auto-discover .issuegraph/*.db). In proxied-server mode, a value that isn't an existing path is treated as a database name override (see --database)")
	rootCmd.PersistentFlags().StringVar(&databaseFlag, "database", "", "Run against a different server database for this invocation, without changing the project's configured database (proxied-server mode only)")
	rootCmd.PersistentFlags().StringVar(&actor, "actor", "", "Actor name for audit trail (default: $ISSUEGRAPH_ACTOR, git user.name, $USER)")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	rootCmd.PersistentFlags().String("format", "", "Output format (json). Alias for --json")
	_ = rootCmd.PersistentFlags().MarkHidden("format") // Hidden alias for CLI ergonomics
	rootCmd.PersistentFlags().BoolVar(&sandboxMode, "sandbox", false, "Sandbox mode: disables Dolt auto-push")
	rootCmd.PersistentFlags().BoolVar(&readonlyMode, "readonly", false, "Read-only mode: block write operations (for worker sandboxes)")
	rootCmd.PersistentFlags().BoolVar(&globalFlag, "global", false, "Use the global shared-server database (issuegraph_global)")
	rootCmd.PersistentFlags().StringVar(&doltAutoCommit, "dolt-auto-commit", "", "Dolt auto-commit policy (off|on|batch). 'on': commit after each write. 'batch': defer commits to issuegraph dolt commit; uncommitted changes persist in the working set until then (a live batch-mode issuegraph process also flushes on SIGTERM/SIGHUP). Applies to embedded and direct SQL-server modes; proxied-server routes are unaffected. Default: on. Override via config key dolt.auto-commit")
	rootCmd.PersistentFlags().BoolVar(&cpuProfileEnabled, "cpu-profile", false, "Generate CPU profile for performance analysis")
	rootCmd.PersistentFlags().StringVar(&memProfilePath, "mem-profile", "", "Write heap profile to FILE on exit (also respects ISSUEGRAPH_MEM_PROFILE)")
	rootCmd.PersistentFlags().BoolVarP(&verboseFlag, "verbose", "v", false, "Enable verbose/debug output")
	rootCmd.PersistentFlags().BoolVarP(&quietFlag, "quiet", "q", false, "Suppress non-essential output (errors only)")
	rootCmd.PersistentFlags().BoolVar(&ignoreSchemaSkew, "ignore-schema-skew", false, "Proceed despite forward schema drift (some queries may fail)")
	rootCmd.PersistentFlags().BoolVar(&noColorFlag, "no-color", false, "Disable color output (also: NO_COLOR=1 or CLICOLOR=0)")

	// Add --version flag to root command (same behavior as version subcommand)
	rootCmd.Flags().BoolP("version", "V", false, "Print version information")

	// Command groups for organized help output (Tufte-inspired)
	rootCmd.AddGroup(&cobra.Group{ID: "issues", Title: "Working With Issues:"})
	rootCmd.AddGroup(&cobra.Group{ID: "views", Title: "Views & Reports:"})
	rootCmd.AddGroup(&cobra.Group{ID: "deps", Title: "Dependencies & Structure:"})
	rootCmd.AddGroup(&cobra.Group{ID: "sync", Title: "Sync & Data:"})
	rootCmd.AddGroup(&cobra.Group{ID: "setup", Title: "Setup & Configuration:"})
	// NOTE: Many maintenance commands (clean, cleanup, compact, validate, repair-deps)
	// should eventually be consolidated into 'issuegraph doctor' and 'issuegraph doctor --fix' to simplify
	// the user experience. The doctor command can detect issues and offer fixes interactively.
	rootCmd.AddGroup(&cobra.Group{ID: "maint", Title: "Maintenance:"})
	rootCmd.AddGroup(&cobra.Group{ID: "advanced", Title: "Integrations & Advanced:"})

	// Custom help function with semantic coloring (Tufte-inspired)
	// Note: Usage output (shown on errors) is not styled to avoid recursion issues
	rootCmd.SetHelpFunc(colorizedHelpFunc)
}

func resolveChangeDirIssueGraphDir(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", nil
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("cannot resolve -C directory %q: %w", path, err)
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("cannot use -C directory %q: %w", path, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("cannot use -C directory %q: not a directory", path)
	}
	issuegraphDir := issuegraph.FindIssueGraphDirFrom(absPath)
	if issuegraphDir == "" {
		return "", fmt.Errorf("cannot use -C directory %q: no issuegraph project found", path)
	}
	return issuegraphDir, nil
}

func applyChangeDirSelection() error {
	if strings.TrimSpace(changeDir) == "" {
		return nil
	}
	issuegraphDir, err := resolveChangeDirIssueGraphDir(changeDir)
	if err != nil {
		return HandleError("%v", err)
	}
	changeDirEnvSnapshot = make(map[string]envSnapshotValue, 3)
	for _, key := range []string{"ISSUEGRAPH_DIR", "ISSUEGRAPH_DB", "ISSUEGRAPH_DB"} {
		value, ok := os.LookupEnv(key)
		changeDirEnvSnapshot[key] = envSnapshotValue{value: value, ok: ok}
	}
	_ = os.Setenv("ISSUEGRAPH_DIR", issuegraphDir)
	return nil
}

func restoreChangeDirSelection() {
	if changeDirEnvSnapshot == nil {
		return
	}
	for key, snapshot := range changeDirEnvSnapshot {
		if snapshot.ok {
			_ = os.Setenv(key, snapshot.value)
		} else {
			_ = os.Unsetenv(key)
		}
	}
	changeDirEnvSnapshot = nil
}

func guardLegacyNoStoreCommand(cmd *cobra.Command, issuegraphDir string) error {
	if cmd == nil || !cmd.Runnable() || cmd.Parent() == nil || cmd == versionCmd ||
		cmd == doctorCmd || cmd == initCmd || cmd == bootstrapCmd ||
		cmd == legacySQLiteCmd {
		return nil
	}
	if cmd == schemaCmd && cmd.Parent() != nil && cmd.Parent().Parent() == nil {
		return nil
	}
	for current := cmd; current != nil; current = current.Parent() {
		if current == metricsCmd {
			return nil
		}
	}
	switch cmd.Name() {
	case "__complete", "__completeNoDesc", "bash", "completion", "fish", "help", "powershell", "zsh":
		return nil
	}
	if issuegraphDir == "" {
		return guardUndiscoveredLegacyWorkspace()
	}
	return guardLegacyUpgradeWorkspace(issuegraphDir)
}

var rootCmd = &cobra.Command{
	Use:   "issuegraph",
	Short: "issuegraph - Dependency-aware issue tracker",
	Long:  `Issues chained together like issuegraph. A lightweight issue tracker with first-class dependency support.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Handle --version flag on root command
		if v, _ := cmd.Flags().GetBool("version"); v {
			fmt.Printf("issuegraph version %s (%s)\n", Version, Build)
			return
		}
		// No subcommand - show help
		_ = cmd.Help() // Help() always returns nil for cobra commands
	},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) (retErr error) {
		applyNoColorFlag()

		// Initialize CommandContext to hold runtime state (replaces scattered globals)
		initCommandContext()

		// Reset per-command write tracking (used by Dolt auto-commit).
		commandDidWrite.Store(false)
		commandMayEmptyJSONLExport.Store(false)
		commandDeletedIssueIDs.reset()
		commandDidExplicitDoltCommit = false
		commandDidWriteTipMetadata = false
		commandTipIDsShown = make(map[string]struct{})
		commandFreeze = migration.Result{}

		// Set up signal-aware context with batch commit flush on shutdown.
		// Unlike signal.NotifyContext, this also handles SIGHUP and flushes
		// pending batch commits before canceling the context.
		//
		// Publish through setRootContext, not a bare assignment to the
		// globals: cmdCtx exists by now (initCommandContext above), so
		// getRootContext() reads cmdCtx.RootCtx, and the commands that
		// return early from this hook -- every skipsStoreInit command,
		// migrate among them -- never reach syncCommandContext to have it
		// backfilled. A bare assignment leaves those commands reading a nil
		// per-command context and losing Ctrl-C entirely.
		setRootContext(setupGracefulShutdown())

		// Initialize OTel. Telemetry is opt-in — initTelemetry is a noop
		// unless ISSUEGRAPH_OTEL_ENABLED=true or a legacy ISSUEGRAPH_OTEL_* selector is set.
		// Must run before any DB access so SQL spans nest under the command
		// span.
		initTelemetry(rootCtx, Version)

		// Materialize the user-level metrics config only when metrics are
		// actually enabled. When metrics are disabled (ISSUEGRAPH_DISABLE_METRICS or a
		// user-global metrics.disabled), there is nothing to bootstrap. The
		// send-metrics flusher is exempt so it never recurses into bootstrap.
		// This mirrors the resolveMetricsEnabled() gate on the first-run notice
		// below. (~/.config/issuegraph/ lives outside the repo, so this write is not a
		// stealth/per-repository trace; stealth init is handled by suppressing
		// the first-run notice, not by skipping this user-global bootstrap.)
		if cmd.Name() != metrics.SendMetricsSubcommand && resolveMetricsEnabled() {
			if err := metrics.EnsureUserConfigDefaults(); err != nil {
				debug.Logf("warning: ensure user config defaults failed: %v", err)
			}
		}

		if _, err := metrics.Init(Version, resolveMetricsEnabled(), resolveMetricsEndpoint()); err != nil {
			debug.Logf("warning: metrics init failed: %v", err)
		}

		if cmd.Name() == metrics.SendMetricsSubcommand {
			return nil
		}

		// Start root span for this command. rootCtx now carries the span, so
		// all downstream DB and AI calls become child spans automatically.
		rootCtx, commandSpan = startCommandSpan(rootCtx, cmd.Name(), Version, os.Args[1:], secretFlagTokens(cmd))

		// Apply verbosity flags early (before any output)
		debug.SetVerbose(verboseFlag)
		debug.SetQuiet(quietFlag)

		if err := applyChangeDirSelection(); err != nil {
			return err
		}

		// Block dangerous env var overrides that could cause data fragmentation (bd-hevyw).
		if err := checkBlockedEnvVars(); err != nil {
			return HandleError("%v", err)
		}

		loadSelectionEnvironment()

		// Apply viper configuration if flags weren't explicitly set
		// Priority: flags > viper (config file + env vars) > defaults
		// Do this BEFORE early-return so init/version/help respect config

		// Track flag overrides for notification (only in verbose mode)
		flagOverrides := make(map[string]struct {
			Value  interface{}
			WasSet bool
		})

		// Handle --format json alias (desire-path from GH#2612)
		if cmd.Root().PersistentFlags().Changed("format") {
			format, _ := cmd.Root().PersistentFlags().GetString("format")
			if strings.EqualFold(format, "json") {
				jsonOutput = true
			}
		}
		// If flag wasn't explicitly set, use viper value.
		//
		// SHADOWING HAZARD (GH#6278): this reads the ROOT persistent --format
		// only, so a subcommand that registers its own local --format shadows
		// it here and still gets `json: true` promoted over the format it was
		// asked for. `issuegraph list` compensates in gatherListInput; any new local
		// --format registration needs the same treatment or this fires again.
		if !commandJSONFlagChanged(cmd) && !cmd.Root().PersistentFlags().Changed("format") {
			jsonOutput = config.GetBool("json")
		} else {
			flagOverrides["json"] = struct {
				Value  interface{}
				WasSet bool
			}{jsonOutput, true}
		}
		if !cmd.Root().PersistentFlags().Changed("readonly") {
			readonlyMode = config.GetBool("readonly")
		} else {
			flagOverrides["readonly"] = struct {
				Value  interface{}
				WasSet bool
			}{readonlyMode, true}
		}
		var dbNameFromDBFlag string
		if cmd.Name() != "init" && cmd.Root().PersistentFlags().Changed("db") && dbPath != "" &&
			dbidentifier.ValidateIdentifier(dbPath) == nil {
			if _, statErr := os.Stat(dbPath); statErr != nil {
				if !os.IsNotExist(statErr) {
					return HandleError("--db %q: %v", dbPath, statErr)
				}
				dbNameFromDBFlag = dbPath
				dbPath = ""
			}
		}

		if !cmd.Root().PersistentFlags().Changed("db") && dbPath == "" &&
			os.Getenv("ISSUEGRAPH_DB") == "" && os.Getenv("ISSUEGRAPH_DB") == "" && os.Getenv("ISSUEGRAPH_DIR") == "" {
			dbPath = config.GetString("db")
		} else if cmd.Root().PersistentFlags().Changed("db") {
			flagOverrides["db"] = struct {
				Value  interface{}
				WasSet bool
			}{dbPath, true}
		}
		if !cmd.Root().PersistentFlags().Changed("actor") && actor == "" {
			actor = resolveConfiguredActor()
		} else if cmd.Root().PersistentFlags().Changed("actor") {
			flagOverrides["actor"] = struct {
				Value  interface{}
				WasSet bool
			}{actor, true}
		}
		if !cmd.Root().PersistentFlags().Changed("dolt-auto-commit") && strings.TrimSpace(doltAutoCommit) == "" {
			doltAutoCommit = config.GetString("dolt.auto-commit")
		} else if cmd.Root().PersistentFlags().Changed("dolt-auto-commit") {
			flagOverrides["dolt-auto-commit"] = struct {
				Value  interface{}
				WasSet bool
			}{doltAutoCommit, true}
		}

		// --ignore-schema-skew sets ISSUEGRAPH_IGNORE_SCHEMA_SKEW so the env-var escape
		// hatch works uniformly for all store open paths (dolt, embedded).
		if ignoreSchemaSkew {
			_ = os.Setenv("ISSUEGRAPH_IGNORE_SCHEMA_SKEW", "1")
		}

		// Check for and log configuration overrides (only in verbose mode)
		if verboseFlag {
			overrides := config.CheckOverrides(flagOverrides)
			for _, override := range overrides {
				config.LogOverride(override)
			}
		}

		// GH#1093: Check noDbCommands BEFORE expensive operations
		// to avoid spawning git subprocesses for simple commands
		// like "issuegraph version" that don't need database access.
		//
		// A command can also opt out of store init by setting the
		// skipStoreAnnotation on its Command literal instead of being listed
		// here (see commandOptsOutOfStore) — useful for commands defined in
		// other files or build-tagged variants that can't edit this list. The
		// "doctor" command uses that seam and so is intentionally absent below.
		noDbCommands := []string{
			"__complete",       // Cobra's internal completion command (shell completions work without db)
			"__completeNoDesc", // Cobra's completion without descriptions (used by fish)
			"bash",
			"bootstrap",
			"completion",
			"context", // reads config files directly, does not need DB open
			"codex-hook",
			"cursor-hook", // shells out to `issuegraph prime`; never opens the store itself
			// "doctor" opts out via skipStoreAnnotation on its Command literal.
			"dolt", // bare "issuegraph dolt" shows help only; subcommands handled below
			"fish",
			"formula", // parser-only subcommands; add a store-needed guard before adding DB-backed formula subcommands
			"help",
			"hook", // manages its own store lifecycle (#1719)
			"hooks",
			"human",
			"init",
			"merge",
			"metrics", // config-only: status/on/off/example never touch the DB
			"onboard",
			"powershell",
			"prime",
			"quickstart",
			metrics.SendMetricsSubcommand,
			"setup",
			"version",
			"where",
			"zsh",
		}

		// GH#2042: Dolt subcommands that need the store for version-control operations.
		// All other dolt subcommands (show, set, test, start, stop, status) are
		// config/diagnostic commands that skip DB init via the "dolt" parent entry above.
		needsStoreDoltSubcommands := []string{"push", "pull", "commit"}

		// GH#2224: Dolt grandchild subcommands (e.g. "issuegraph dolt remote add") whose
		// Cobra parent is "remote", not "dolt". These need the store but would be
		// silently skipped if "remote" were ever added to noDbCommands.
		needsStoreDoltGrandchildren := []string{"remote"}

		// bd-m7zzd: "human" is listed in noDbCommands for its bare help
		// screen, but list/respond/dismiss/stats are DB-backed. Without this
		// they skip store init entirely, which direct mode papered over by
		// lazily opening a store via ensureStoreActive() — and which in
		// proxied mode left no UOW provider for the proxied duals.
		needsStoreHumanSubcommands := []string{"list", "respond", "dismiss", "stats"}

		skipStoreMigrateSubcommands := []string{"from-server-to-proxied-server", "from-proxied-server-to-server", "from-shared-server-to-proxied-server", "from-proxied-server-to-shared-server"}

		// Check both the command name and parent command name for subcommands
		cmdName := cmd.Name()
		isSubcommand := cmd.Parent() != nil && cmd.Parent().Name() != "issuegraph"
		skipsStoreInit := false
		if cmd.Parent() != nil {
			parentName := cmd.Parent().Name()
			if parentName == "dolt" && slices.Contains(needsStoreDoltSubcommands, cmdName) {
				// GH#2042: dolt push/pull/commit need the store — fall through to init
			} else if slices.Contains(needsStoreDoltGrandchildren, parentName) {
				// GH#2224: dolt remote add/list/remove need the store — fall through to init
			} else if parentName == "human" && slices.Contains(needsStoreHumanSubcommands, cmdName) {
				// bd-m7zzd: human list/respond/dismiss/stats need the store — fall through to init
			} else if parentName == "migrate" && slices.Contains(skipStoreMigrateSubcommands, cmdName) {
				skipsStoreInit = true
			} else if slices.Contains(noDbCommands, parentName) {
				skipsStoreInit = true
			}
		}
		// Only skip for top-level commands in noDbCommands, not subcommands
		// that happen to share names (e.g., "issuegraph backup init" vs "issuegraph init").
		if slices.Contains(noDbCommands, cmdName) && !isSubcommand {
			skipsStoreInit = true
		}

		// Skip for root command with no subcommand (just shows help)
		if cmd.Parent() == nil && cmdName == cmd.Use {
			skipsStoreInit = true
		}

		// Also skip for --version flag on root command (cmdName would be "issuegraph")
		if v, _ := cmd.Flags().GetBool("version"); v {
			skipsStoreInit = true
		}

		// A command may also opt out of store init by declaring the
		// issuegraph:skip_store annotation (see commandOptsOutOfStore), instead of being
		// added to the noDbCommands list above. Commands defined in other files
		// or build-tagged variants use this to exempt themselves without editing
		// the central list.
		if commandOptsOutOfStore(cmd) {
			skipsStoreInit = true
		}

		// One-time friendly heads-up about anonymous usage metrics. Placed after
		// the config-derived json/quiet rebind and command classification above so
		// it can read the real output mode and command identity — that is how it
		// stays suppressed in JSON/hook/protocol/quiet/stealth contexts and never
		// corrupts machine-readable output. No-op after the first run.
		maybeShowMetricsFirstRunNotice(cmd)

		// Commands that skip store initialization still need early config/env
		// setup before they inspect server mode or per-project Dolt settings.
		// Rebind them to the selected workspace so explicit --db / ISSUEGRAPH_DB
		// targets behave consistently across doctor/bootstrap/context/dolt.
		if skipsStoreInit {
			issuegraphDir := selectedNoDBIssueGraphDir(cmd)
			prepareSelectedNoDBContext(issuegraphDir)
			refreshBoundCommandConfig(cmd)
			if os.Getenv("ISSUEGRAPH_DIR") == "" {
				loadEnvironment()
				if err := loadServerModeFromConfig(); err != nil {
					// Warn, don't fatal: skipsStoreInit commands (doctor,
					// init, bootstrap, version, ...) never select a store,
					// and several of them are the repair path for the very
					// corruption being reported.
					fmt.Fprintf(os.Stderr, "warning: %v\n", err)
				}
			}
			if issuegraphDir == "" {
				issuegraphDir = issuegraph.FindIssueGraphDir()
			}
			if commandRegistryPath(cmd) == "doctor" && usesProxiedServer() {
				// Refuse only on a real refusal. The registry validator
				// returns nil for doctor subcommands, and returning early on
				// that would skip the legacy-store guard and autocommit-mode
				// resolution every other skipsStoreInit command still runs.
				if err := validateProxyRegistryBeforeProvider(cmd, resolveProxiedTopology(issuegraphDir)); err != nil {
					return err
				}
			}
			if err := guardLegacyNoStoreCommand(cmd, issuegraphDir); err != nil {
				isMigrationCommand := false
				for current := cmd; current != nil; current = current.Parent() {
					if current.Name() == "migrate" {
						isMigrationCommand = true
						break
					}
				}
				if isMigrationCommand {
					return HandleProxyCapabilityError(&ProxyCapabilityError{Code: "proxy.migrate.invalid_state", Message: err.Error(), ExitCode: 1, Mutates: false})
				}
				return HandleError("%v", err)
			}
			if _, err := getDoltAutoCommitMode(); err != nil {
				return HandleError("%v", err)
			}
		}

		if skipsStoreInit {
			return nil
		}

		// Performance profiling setup
		if cpuProfileEnabled {
			timestamp := time.Now().Format("20060102-150405")
			if f, _ := os.Create(fmt.Sprintf("issuegraph-profile-%s-%s.prof", cmd.Name(), timestamp)); f != nil {
				profileFile = f
				_ = pprof.StartCPUProfile(f) // Best effort: profiling is a debug tool, failure is non-fatal
			}
			if f, _ := os.Create(fmt.Sprintf("issuegraph-trace-%s-%s.out", cmd.Name(), timestamp)); f != nil {
				traceFile = f
				_ = trace.Start(f) // Best effort: profiling is a debug tool, failure is non-fatal
			}
		}

		// Auto-detect sandboxed environment (Phase 2 for GH #353)
		if !cmd.Root().PersistentFlags().Changed("sandbox") {
			if isSandboxed() {
				sandboxMode = true
				fmt.Fprintf(os.Stderr, "ℹ️  Sandbox detected, using direct mode\n")
			}
		}

		// Capture redirect info BEFORE FindDatabasePath() follows the redirect.
		// When .issuegraph/redirect points to a shared directory with a different
		// dolt_database, the source's database name would be lost. Capture it
		// early and set ISSUEGRAPH_DOLT_SERVER_DATABASE so all store opens use it.
		if dbPath == "" {
			preserveRedirectSourceDatabase(issuegraph.GetRedirectInfo().LocalDir)
		}

		if dbPath == "" {
			if issuegraph := issuegraph.FindIssueGraphDir(); issuegraph != "" {
				// Bind the discovered target before admission so the legacy guard
				// honors its config.yaml (including dolt.shared-server), not the
				// caller's. This setup is read-only: metadata discovery below still
				// uses LoadForDiscovery and cannot migrate config.json.
				prepareSelectedCommandContext(issuegraph, true)
				refreshBoundCommandConfig(cmd)
				if guardErr := guardLegacyUpgradeWorkspace(issuegraph); guardErr != nil {
					return HandleError("%v", guardErr)
				}
				cfg, cfgErr := configfile.LoadForDiscovery(issuegraph)
				if cfgErr != nil || cfg != nil && (cfg.IsDoltProxiedServerMode() ||
					registeredBackendWorkspaceIsIssueGraphDir(cfg) ||
					!configfile.IsSupportedBackend(cfg.Backend)) {
					// Proxied-server, registered remote, and removed-backend
					// workspaces may have no local Dolt database file. Invalid
					// or unknown metadata likewise must reach config validation
					// instead of becoming a generic "no database" result.
					dbPath = issuegraph
				} else if cfg == nil && cfgErr == nil &&
					configfile.DefaultConfig().HostImpliesServerMode() {
					// Metadata-less workspace whose server lives at a
					// remote host named by ISSUEGRAPH_DOLT_SERVER_HOST or
					// config.yaml (GH#3545): there is no local database
					// directory to discover, so route the .issuegraph dir as
					// a server workspace instead of "no database found".
					dbPath = issuegraph
				}
			} else if guardErr := guardUndiscoveredLegacyWorkspace(); guardErr != nil {
				return HandleError("%v", guardErr)
			}
		}

		// Initialize database path
		if dbPath == "" {
			// Use public API to find database (same logic as extensions)
			if foundDB := issuegraph.FindDatabasePath(); foundDB != "" {
				dbPath = foundDB
			} else {
				// No database found — allow some commands to run without a database
				// - import: auto-initializes database if missing
				// - setup: creates editor integration files (no DB needed)
				// - config subcommands that operate on config.yaml, git config,
				//   or best-effort diagnostics only (GH#536, bd-934, bd-omc, bd-3rw)
				if configCommandCanRunWithoutStore(cmd, args) {
					// When --db is provided, resolve ISSUEGRAPH_DIR so yaml-only
					// config writes target the correct directory (GH#3348).
					if dbPath != "" {
						if issuegraphDir := resolveCommandIssueGraphDir(dbPath); issuegraphDir != "" {
							prepareSelectedCommandContext(issuegraphDir, false)
						}
					}
					return nil
				}

				// GH#3686: `issuegraph create --repo=<path-or-URL>` targets a different
				// repo's workspace. Without this, PreRun exits with "no issuegraph
				// database found" before create.go's --repo handling runs, even
				// when the target has a valid workspace of its own. Resolve
				// local targets here so store initialization points at them.
				// Remote --repo URLs need no local database at all: create.go
				// opens the remote store itself via the remote cache and never
				// touches the local `store` global on that path (a gap left by
				// #4615, which only handled local paths), so skip local
				// discovery entirely instead of falling through to the "no
				// issuegraph database found" exit below.
				if cmd.Name() == "create" && cmd.Flags().Changed("repo") {
					if repoVal, _ := cmd.Flags().GetString("repo"); repoVal != "" {
						if remotecache.IsRemoteURL(repoVal) {
							return nil
						}
						targetIssueGraphDir := filepath.Join(routing.ExpandPath(repoVal), ".issuegraph")
						dbPath = utils.CanonicalizePath(filepath.Join(targetIssueGraphDir, issuegraph.CanonicalDatabaseName))
					}
				}

				if dbPath == "" && cmd.Name() != "import" && cmd.Name() != "setup" {
					// No database found - provide context-aware error message
					fmt.Fprintf(os.Stderr, "Error: no issuegraph database found\n")
					fmt.Fprintf(os.Stderr, "Hint: %s\n", diagHint())
					fmt.Fprintf(os.Stderr, "      or set ISSUEGRAPH_DIR to point to your .issuegraph directory\n")
					return SilentExit()
				}

				if dbPath == "" {
					// For import/setup commands, set default database path
					// Invariant: dbPath must always be absolute. Use CanonicalizePath for OS-agnostic
					// handling (symlinks, case normalization on macOS).
					//
					// IMPORTANT: Use FindIssueGraphDir() to get the correct .issuegraph directory,
					// which follows redirect files. Without this, a redirected .issuegraph
					// would create a local database instead of using the redirect target.
					// (GH#bd-0qel)
					targetIssueGraphDir := issuegraph.FindIssueGraphDir()
					if targetIssueGraphDir == "" {
						targetIssueGraphDir = ".issuegraph"
					}
					dbPath = utils.CanonicalizePath(filepath.Join(targetIssueGraphDir, issuegraph.CanonicalDatabaseName))
				}
			}
		}

		issuegraphDir := resolveCommandIssueGraphDir(dbPath)
		prepareSelectedCommandContext(issuegraphDir, true)
		refreshBoundCommandConfig(cmd)
		if guardErr := guardLegacyUpgradeWorkspace(issuegraphDir); guardErr != nil {
			return HandleError("%v", guardErr)
		}

		// Workspace operation gate: every command that reaches this point
		// will open the store (the skipsStoreInit early return is above),
		// so take the workspace + physical-root gates now, in the final
		// mode (SHARED for normal commands, EXCLUSIVE for issuegraph backup
		// restore — there is no upgrade path). See workspace_gate.go for
		// the fail-open/fail-closed posture. The handle is released in
		// PersistentPostRunE after store close; if this PreRunE fails
		// later, cobra never runs PostRunE, so the deferred release below
		// covers the PreRunE error paths after acquisition.
		if err := acquireCommandWorkspaceGates(rootCtx, cmd, issuegraphDir); err != nil {
			return err
		}
		defer func() {
			if retErr != nil {
				// Gate-outlives-store: a PreRunE failure AFTER the store
				// opened (cobra will skip PostRunE) must close the
				// store/provider before the gates drop, or maintenance
				// could start against un-quiesced storage.
				closeStoreBeforeGateRelease()
				releaseWorkspaceGates()
			}
		}()
		if _, err := getDoltAutoCommitMode(); err != nil {
			return HandleError("%v", err)
		}

		// Resolve the backend before version tracking, migration, server startup, or
		// any store construction. PostgreSQL/MySQL values are retained as metadata
		// tombstones so an existing workspace fails closed instead of falling through
		// to a new, empty Dolt database.
		cfg, cfgErr := configfile.Load(issuegraphDir)
		if cfgErr != nil {
			return HandleError("failed to load issuegraph config from %s: %v (refusing to fall back to the embedded store; fix or restore metadata.json and retry)", issuegraphDir, cfgErr)
		}
		if backendErr := validateConfiguredBackend(cfg, issuegraphDir); backendErr != nil {
			return HandleError("%v", backendErr)
		}
		// Reject proxy capability combinations before any workspace side effect
		// (version tracking, migration, auto-start, or provider construction).
		// Two validators, one for each half of the policy: flag-keyed rules and
		// the path-keyed capability registry.
		if cfg != nil && cfg.IsDoltProxiedServerMode() {
			if err := validateProxyCapabilitiesBeforeProvider(cmd); err != nil {
				return err
			}
			if err := validateProxyRegistryBeforeProvider(cmd, resolveProxiedTopology(issuegraphDir)); err != nil {
				return err
			}
		}
		// The proxied provider cannot guarantee strict read-only semantics. Refuse
		// before provider construction so no connection, migration, or mutation
		// is attempted; expose the same stable capability code as other proxy
		// front-door refusals.
		if readonlyMode && cfg != nil && cfg.IsDoltProxiedServerMode() {
			return HandleProxyCapabilityError(AssertProxyCapability(ProxyModeProxied, ProxyCapReadonly))
		}

		// Set actor for audit trail
		actor = getActorWithGit()
		// Attach actor to the command span now that we have it.
		if commandSpan != nil {
			commandSpan.SetAttributes(attribute.String("issuegraph.actor", actor))
		}

		// Check if this is a read-only command (GH#804) or an explicitly
		// non-mutating preview. Both must open the store read-only: otherwise
		// schema initialization runs before the command's RunE can honor
		// --dry-run/--inspect or reject invalid arguments. Resolved here,
		// ahead of version tracking, because that is the first step a preview
		// has to change.
		previewMode := isPreviewCommand(cmd)
		policy := effectiveRootStorePolicy(cmd.Name(), readonlyMode)
		useReadOnly := policy.readOnly || previewMode

		// dc-6jaq: consult the migration freeze marker here, before any of
		// this hook's own store-touching side effects — trackBdVersion below
		// (writes .local_version), autoMigrateOnVersionBump (opens its own
		// store connection and can apply a schema migration), and
		// maybeAutoImportJSONL (imports into the store when empty) all run
		// before the command's RunE, where CheckReadonly would otherwise
		// catch a frozen write first. By then the most dangerous writes this
		// gate exists to prevent would already be done. useReadOnly already
		// carries the exact classification this early gate must skip (strict
		// --readonly, or a command on the read-only allowlist) — reusing it
		// here means there is no second, independently-maintained list of
		// "write" commands to drift out of sync with the one useReadOnly is
		// built from. An explicit --dry-run/--inspect preview also sets
		// useReadOnly and so skips this early gate the same way, but is NOT
		// exempt overall: the per-command chokepoint checks again once RunE
		// is reached, with no preview awareness — CheckReadonly for the ~120
		// commands that call it, and runImport's own call for `issuegraph import
		// --dry-run`, which is not one of them. So a preview on a frozen
		// workspace still exits ExitMigrationFrozen there, fail-closed, same
		// as strict --readonly already blocks `create --dry-run` today.
		// Returning the refusal rather than exiting is load-bearing: the
		// gates acquired above are released by this hook's deferred cleanup,
		// which os.Exit would skip.
		//
		// One lookup answers the whole invocation: the walk is a stat per
		// ancestor of both the cwd and the resolved workspace, and resolving
		// it twice a statement apart would not only pay for it twice but let
		// the two readings disagree about a marker that appeared or vanished
		// in between (write allowed but maintenance skipped, or the reverse).
		// PostRunE reads the same answer via commandFreeze.
		commandFreeze = migration.Find(issuegraphDir)
		if !useReadOnly {
			if err := migrationFreezeGate(cmd, strings.TrimPrefix(cmd.CommandPath(), cmd.Root().Name()+" "), commandFreeze); err != nil {
				return err
			}
		}

		// dc-6jaq (review round 2, ask #1): a command classified read-only —
		// or an explicit preview — is deliberately allowed past the gate
		// above; diagnosis must keep working during a freeze. But
		// trackBdVersion/autoMigrateOnVersionBump below are this hook's OWN
		// writes against the (possibly frozen) store, run regardless of the
		// command's own classification — so "the command is a read" must not
		// imply "these side effects may still run". Reproduced pre-fix:
		// freeze the workspace, seed .local_version with a stale version, run
		// `issuegraph list` — exit 0 (correct, it's a read), but .local_version was
		// silently rewritten mid-freeze anyway. Skip both calls under an
		// active freeze without blocking the read itself. Reuses the single
		// lookup resolved above rather than repeating the walk.
		frozenForMaintenance := policy.runMaintenance && commandFreeze.Frozen()

		// Track issuegraph version changes unless strict readonly forbids repository mutation.
		// Best-effort tracking - failures are silent.
		//
		// A preview detects the change but must not consume it: .local_version
		// is the one-shot signal autoMigrateOnVersionBump reads, and a preview
		// skips that reconciliation (see below). See trackBdVersionPreview.
		if policy.runMaintenance && !frozenForMaintenance {
			if previewMode {
				trackBdVersionPreview()
			} else {
				trackBdVersion()
			}
		}

		// If the operator passed --force on `issuegraph migrate` or `issuegraph migrate schema`,
		// set the programmatic gate override before both autoMigrateOnVersionBump
		// and the main store open — both open their own store connections and the
		// gate fires on each.
		forcedMigrate := isForcedMigrate(cmd)
		if forcedMigrate {
			if name := forcedMigratePreviewFlag(cmd); name != "" {
				return HandleError("--force cannot be combined with --%s: opening the store with the gate overridden applies pending migrations before the preview runs", name)
			}
		}
		// Unconditional set-or-clear keeps the override self-clearing should the
		// root command ever be re-run in-process (tests, a future server mode).
		schema.SetForceAllowRemoteMigrate(forcedMigrate)

		// Typing `issuegraph migrate schema` is consent to migrate a shared database
		// that has no remote (#5920): there is no cross-clone fork to risk,
		// only the co-resident lockout the operator is asking to accept. A
		// preview withholds it — `issuegraph migrate schema --dry-run` must not
		// migrate on the way to printing what it would do. Same set-or-clear
		// discipline as the --force override above.
		schema.SetSharedMigrateConsent(isSchemaMigrateVerb(cmd) && !previewMode)

		// Auto-migrate database on version bump (bd-jgxi).
		// Runs for ALL non-preview commands (including read-only ones) because
		// the migration opens its own store connection, writes the version
		// metadata, commits it, and closes BEFORE the main store is opened.
		// This ensures issuegraph doctor and read-only commands see the correct version
		// after a CLI upgrade.
		//
		// Preview paths must never call this helper: it opens a separate
		// writable store before the main read-only store and can therefore
		// apply schema migrations before RunE validates arguments or renders a
		// dry-run plan. frozenForMaintenance excludes it for the same reason
		// as the trackBdVersion call above — see that comment.
		if policy.runMaintenance && !previewMode && !frozenForMaintenance {
			autoMigrateOnVersionBump(issuegraphDir)
		}

		// Initialize direct storage access
		var err error

		// Create Dolt storage config — resolve dolt data dir which may be
		// on a different filesystem (e.g., ext4 for performance on WSL).
		doltPath := doltserver.ResolveDoltDir(issuegraphDir)
		doltCfg := &dolt.Config{
			ReadOnly:         useReadOnly,
			Preview:          previewMode,
			DisableAutoStart: policy.disableAutoStart,
			IssueGraphDir:    issuegraphDir,
			LenientOpen:      isWorkingSetReconcileCommand(cmd),
			RemoteSyncOpen:   isRemoteSyncCommand(cmd),
			// Bulk loads outlive the pool's 10s fast-fail on every server
			// pause (wy-sbgucn); explicit env/config settings still win.
			PoolReadTimeoutFallback: bulkLoadPoolReadTimeout(cmd),
		}

		// Load config to get database name and server connection settings.
		// A present-but-unloadable metadata.json must stop the command here:
		// continuing with the zero-value config silently selects the embedded
		// store with the default database name, and on server-mode
		// deployments that empty relic answers every query with an empty
		// result set and exit 0 (false-empty), which readers misinterpret as
		// "no work". Absent metadata.json (cfg == nil, cfgErr == nil) keeps
		// the fresh-repo embedded default below — unless the env/config.yaml
		// layers already select server mode: that decision must not depend on
		// metadata existing, so substitute the default config and let the
		// normal mode/connection resolution run.
		//
		// The gate asks IsDoltServerMode — the same resolver the branch below
		// uses to set doltCfg.ServerMode — rather than HostImpliesServerMode.
		// Host inference (GH#3545) answers only one layer of that question and
		// deliberately returns false as soon as config.yaml names a
		// `dolt.mode`: correct for inferring FROM a host, wrong as the whole
		// gate. A workspace declaring `dolt.mode: server` in .issuegraph/config.yaml
		// with no metadata.json therefore kept cfg nil, fell through to the
		// embedded branch, and answered every query out of a phantom
		// .issuegraph/embeddeddolt database that same run had just created —
		// exit 0, no rows, nothing to distinguish it from real emptiness.
		// ISSUEGRAPH_DOLT_SERVER_MODE=1 did not rescue it either; the old gate
		// never consulted it. IsDoltServerMode is a superset of
		// HostImpliesServerMode, so nothing that reached server mode before
		// stops reaching it now.
		if cfg == nil && configfile.DefaultConfig().IsDoltServerMode() {
			logConfigDiscovery(issuegraphDir, "no metadata.json; env/config.yaml select server mode")
			cfg = configfile.DefaultConfig()
		}
		if cfg != nil {
			warnSharedServerEmbeddedMismatch(cfg)
			doltCfg.ProxiedServer = cfg.IsDoltProxiedServerMode()
			proxiedServerMode = doltCfg.ProxiedServer
			if cmdCtx != nil {
				cmdCtx.ProxiedServerMode = doltCfg.ProxiedServer
			}

			doltCfg.ServerMode = cfg.IsDoltServerMode()
			// Shared server mode (dolt.shared-server in config.yaml) is a
			// form of server mode. Override metadata.json if it still says
			// embedded — handles installs created before GH#2946 fix. Skip
			// this for proxied-server: it's its own backend, not server.
			if !doltCfg.ServerMode && !doltCfg.ProxiedServer && doltserver.IsSharedServerMode() {
				doltCfg.ServerMode = true
			}
			serverMode = doltCfg.ServerMode
			if cmdCtx != nil {
				cmdCtx.ServerMode = doltCfg.ServerMode
			}

			// Always set database name (needed for bootstrap to find
			// prefix-based databases like "issuegraph_hq"; see #1669)
			doltCfg.Database = cfg.GetDoltDatabase()
			if shouldLogDefaultDoltDatabase(cfg) {
				logConfigDiscovery(issuegraphDir, fmt.Sprintf("metadata loaded without dolt_database; using default database name %q", configfile.DefaultDoltDatabase))
			}

			if err := resolveDoltServerConnection(rootCtx, issuegraphDir, cfg, doltCfg); err != nil {
				return HandleError("%v", err)
			}
		} else if cfgErr == nil {
			logConfigDiscovery(issuegraphDir, "config discovery")
			// Load returned (nil, nil) — no config file found.
			// Fall back to the canonical default database name; matches the
			// behavior of newDoltStoreFromConfig / newReadOnlyStoreFromConfig
			// (see store_factory.go). Without this, embeddeddolt.New rejects
			// the empty database name with "database name must not be empty
			// (caller should default to \"issuegraph\")".
			fmt.Fprintf(os.Stderr, "warning: no issuegraph configuration found in %s; using default database name %q\n", issuegraphDir, configfile.DefaultDoltDatabase)
			doltCfg.Database = configfile.DefaultDoltDatabase
		}
		// Honor shared-server mode even when no project config was found
		// (cfg == nil) or the parse failed. The override inside the
		// cfg != nil branch above is skipped in those cases, so without this
		// an exported ISSUEGRAPH_DOLT_SHARED_SERVER is silently ignored and issuegraph
		// falls through to embeddeddolt.Open, creating a phantom embedded DB
		// that subsequent writes fragment into (GH#3817). This is idempotent:
		// when the override above already ran, ServerMode is already true.
		if !doltCfg.ServerMode && !doltCfg.ProxiedServer && doltserver.IsSharedServerMode() {
			doltCfg.ServerMode = true
			serverMode = doltCfg.ServerMode
			if cmdCtx != nil {
				cmdCtx.ServerMode = doltCfg.ServerMode
			}
		}
		// Defensive: embeddeddolt.New rejects an empty database name, so
		// default it even on paths that never set one.
		if doltCfg.Database == "" {
			doltCfg.Database = configfile.DefaultDoltDatabase
		}
		doltCfg.SyncRemote = resolveSyncRemote()

		// --global flag: switch to the global shared-server database.
		// Must be in shared-server mode; errors otherwise.
		if globalFlag {
			if !doltserver.IsSharedServerMode() {
				return HandleError("--global requires shared-server mode (set ISSUEGRAPH_DOLT_SHARED_SERVER=1 or dolt.shared-server: true in config.yaml)")
			}
			doltCfg.Database = doltserver.GlobalDatabaseName
		}

		// Keep standalone CLI auto-start behavior centralized so doctor and
		// other helper paths stay in lockstep with the main command path.
		dolt.ApplyCLIAutoStart(issuegraphDir, doltCfg)

		databaseOverride := databaseFlag
		if dbNameFromDBFlag != "" {
			if databaseOverride != "" && databaseOverride != dbNameFromDBFlag {
				return HandleError("conflicting database selection: --db=%q vs --database=%q", dbNameFromDBFlag, databaseOverride)
			}
			databaseOverride = dbNameFromDBFlag
		}
		if databaseOverride != "" {
			if !proxiedServerMode {
				return HandleErrorRespectJSON("--database (or a --db value naming a database) is only supported in proxied-server mode")
			}
			if err := dbidentifier.ValidateIdentifier(databaseOverride); err != nil {
				return HandleErrorRespectJSON("%v", err)
			}
		}

		// In proxied mode the CLI short-circuits to the uowProvider path and
		// dispatches through the *_proxied_server.go duals.
		//
		// Preview commands take the same policy here as they do on the
		// embedded and server paths, and for the same reason: the provider
		// open runs CREATE DATABASE and schema.MigrateUpWithLock, and
		// reconcileVersionProxiedServer writes version metadata — all during
		// root pre-run, before --dry-run/--inspect has had any effect. Proxied
		// mode is where that is least visible, not where it is acceptable.
		if proxiedServerMode {
			p, err := newProxiedServerUOWProvider(rootCtx, issuegraphDir, databaseOverride,
				rootProviderOptions(previewMode, useReadOnly)...)
			if err != nil {
				// Same typed rendering the store path gets below: a schema
				// skew or a migration-gate refusal here carries a whole
				// actionable block, and `%v` inside "failed to open uow
				// provider" throws all of it away.
				if rendered := renderTypedOpenError(err); rendered {
					return SilentExit()
				}
				return HandleError("failed to open uow provider: %v", err)
			}
			// Fire the workspace's script hooks after commits on the
			// unit-of-work plumbing, which notified no one: hooks now fire on
			// both write plumbings, from the plumbing rather than from each
			// command. This is the proxied twin of the wireStorageDecorators
			// call below. With hooks disabled the sinks are empty and the
			// provider comes back unwrapped.
			var uowSinks uow.Sinks
			if issuegraphDir != "" && !config.GetBool("no-hooks") {
				hookRunner = hooks.NewRunner(filepath.Join(issuegraphDir, "hooks"))
				uowSinks.Hook = hookRunner
			}
			uowProvider = wireExternalDependencyUOWProvider(uow.NewNotifyingProvider(p, uowSinks))

			if !previewMode {
				reconcileVersionProxiedServer(rootCtx)
			}

			syncCommandContext()
			return nil
		}

		// Default auto-commit to ON when the user hasn't set a value, in both
		// modes — "on" names what each mode already does per write:
		// - Embedded mode: each command writes to the working set and commits
		//   it in PersistentPostRun.
		// - Server mode: the storage layer creates one Dolt commit inside each
		//   write transaction (the post-run flush stays embedded-only). The
		//   default here used to be OFF, but the mode was inert in server mode
		//   — every value behaved like ON — so ON is the compatible default
		//   now that batch/off actually defer version commits (bd-4wamg).
		if strings.TrimSpace(doltAutoCommit) == "" {
			doltAutoCommit = string(doltAutoCommitOn)
		}

		doltCfg.Path = doltPath

		// WARNING: DO NOT remove, delete, or modify files inside Dolt's .dolt/
		// directory — including noms/LOCK files. These are Dolt-internal files.
		// Removing them WILL cause unrecoverable data corruption and data loss.
		// Dolt manages these files itself; external interference is never safe.

		if _, ok := backends.Lookup(cfg.GetBackend()); ok {
			store, err = newRegisteredBackendStore(rootCtx, cfg.GetBackend(), issuegraphDir, useReadOnly)
		} else {
			store, err = newDoltStore(rootCtx, doltCfg)
		}

		// Track final read-only state for staleness checks (GH#1089)
		storeIsReadOnly = doltCfg.ReadOnly

		if err != nil {
			// A failed factory can return a typed-nil concrete pointer,
			// which the interface assignment above makes non-nil; the
			// gate-release cleanup would then call Close on a nil
			// receiver and panic. No store was opened, so drop it.
			store = nil
			// Check for fresh clone scenario
			if handleFreshCloneError(err) {
				return SilentExit()
			}
			if renderTypedOpenError(err) {
				return SilentExit()
			}
			return HandleError("failed to open database: %v", err)
		}

		// Mark store as active for flush goroutine safety
		storeMutex.Lock()
		storeActive = true
		storeMutex.Unlock()

		// Auto-import from issues.jsonl when embedded database is empty (GH#2994).
		// This handles the upgrade path from pre-0.56 (dolt/) to 1.0+ (embeddeddolt/)
		// where the new embedded database starts empty but the git-tracked JSONL
		// still has all the user's data.
		// Skip auto-import when the user is explicitly running "issuegraph import" —
		// the import command handles JSONL files itself and auto-importing
		// first would interfere (double-import / upsert confusion).
		if shouldRunAutoImportJSONL(cmd, store, useReadOnly, globalFlag, doltCfg.ServerMode) &&
			!isDisablingImportAutoViaConfigCommand(cmd, args) {
			maybeAutoImportJSONL(rootCtx, store, issuegraphDir)
		}

		// Validate workspace identity for write commands (GH#2438, GH#2372)
		// Skip for read-only commands since they can't corrupt data.
		// Skip for --global: the global database uses a sentinel project ID
		// that won't match any project's metadata.json.
		if !useReadOnly && !globalFlag && os.Getenv("ISSUEGRAPH_SKIP_IDENTITY_CHECK") != "1" {
			if err := validateWorkspaceIdentity(rootCtx, issuegraphDir); err != nil {
				return err
			}
		}

		// Initialize hook runner using the .issuegraph directory resolved above via
		// resolveCommandIssueGraphDir. Do not use filepath.Dir(dbPath): for a
		// registered WorkspaceIsIssueGraphDir backend dbPath is the .issuegraph directory
		// itself, so filepath.Dir(dbPath) would load hooks from the repo root
		// (<repo>/hooks) instead of .issuegraph/hooks; custom dolt_data_dir layouts
		// can likewise place the Dolt data outside .issuegraph.
		if issuegraphDir != "" {
			hookRunner = hooks.NewRunner(filepath.Join(issuegraphDir, "hooks"))
		}

		// Compose the storage decorator chain: OTel instrumentation (no-op
		// when telemetry is off) wrapped by hook firing (skipped when
		// ISSUEGRAPH_NO_HOOKS=1, which is useful for bulk imports, migrations, or
		// environments where on_create/on_update/on_close hooks should not
		// run). Order matters — see wireStorageDecorators in storage_chain.go.
		store = wireStorageDecorators(store, hookRunner, config.GetBool("no-hooks"))

		// Warn if multiple databases detected in directory hierarchy
		warnMultipleDatabases(dbPath)

		// Load molecule templates from hierarchical catalog locations
		// Templates are loaded after auto-import to ensure the database is up-to-date.
		// Skip for import command to avoid conflicts during import operations.
		if cmd.Name() != "import" && store != nil {
			// Reuse the resolved .issuegraph directory (see the hook runner note
			// above) so a registered WorkspaceIsIssueGraphDir workspace loads
			// .issuegraph/molecules.jsonl rather than <repo>/molecules.jsonl.
			loader := molecules.NewLoader(store)
			if result, err := loader.LoadAll(rootCtx, issuegraphDir); err != nil {
				debug.Logf("warning: failed to load molecules: %v", err)
			} else if result.Loaded > 0 {
				debug.Logf("loaded %d molecules from %v", result.Loaded, result.Sources)
			}
		}

		// Sync all state to CommandContext for unified access.
		syncCommandContext()

		// Tips (including sync conflict proactive checks) are shown via maybeShowTip()
		// after successful command execution, not in PreRun
		return nil
	},
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		// Registered FIRST so it runs LAST: the signal context must outlive
		// the store/gate cleanup below, which passes rootCtx to
		// uowProvider.Close. Canceling in the function body (as this used
		// to) handed those closers a dead context on the way out.
		//
		// Clearing matters as much as canceling. Leaving rootCtx pointing at
		// the context we just canceled is harmless when a real issuegraph process
		// exits here, but every in-process caller that runs Execute() more
		// than once -- the modules/cli test binary, library embedders -- would
		// hand that dead context to the next command, and anything reading
		// it refuses work nobody canceled. nil is the documented "no process
		// signal context yet" state and normalizes back to Background().
		//
		// Deferred rather than inline so the early error returns below clear
		// the globals too.
		defer func() {
			if rootCancel != nil {
				rootCancel()
			}
			setRootContext(nil, nil)
		}()
		defer restoreChangeDirSelection()
		// Give the hooks this command fired their moment before the process
		// exits. Both plumbings run them fire-and-forget on their own
		// goroutines, and a issuegraph command is short enough that returning from main
		// can kill one that has not reached exec yet — a hook that silently did
		// not fire, which is the failure this whole seam exists to stop.
		// Bounded by the runner's own per-hook budget: a script that outlives it
		// is being killed anyway, so waiting longer buys nothing.
		//
		// Deferred order is load-bearing on both sides. It runs AFTER the
		// close-and-release below, because a hook script commonly shells out to
		// issuegraph and an EMBEDDED workspace's Dolt lock is held until this process
		// closes its store — the child would fail to open it. (The workspace
		// gates are not the reason: a normal command holds them SHARED, and the
		// child takes them shared too, so those never contend.) It runs BEFORE
		// restoreChangeDirSelection above, because under `-C` the child inherits
		// this process's environment and must see the workspace the command
		// actually ran against.
		defer waitForCommandHooks()
		// Release the workspace/physical-root gates on EVERY exit from
		// PostRunE — deferred so the early error returns below cannot leak
		// the handle past the function. Ordering is enforced, not assumed:
		// the success path closes uowProvider/store itself (and nils them),
		// making the close call here a no-op; on the early error returns
		// the store is still open, so it is closed HERE, before the gates
		// drop — gates must always outlive the store.
		defer func() {
			closeStoreBeforeGateRelease()
			releaseWorkspaceGates()
		}()

		if proxiedServerMode {
			// Retention maintenance before the provider closes: the journal
			// this workspace just wrote to is reached through it. In the body
			// rather than beside the deferred hook wait, for the reason spelled
			// out at the other trigger site below.
			if shouldAutoPruneEventsJournal(cmd) {
				maybeAutoPruneEventsJournal(rootCtx, issuegraph.FindIssueGraphDir())
			}
			if uowProvider != nil {
				_ = uowProvider.Close(rootCtx)
				uowProvider = nil
			}
		} else {
			// dc-6jaq: the freeze covers this hook's own writes too. A read is
			// allowed through a freeze, but the maintenance that trails it is
			// not the user's command — auto-commit, the tip_*_last_shown
			// metadata write and its separate Dolt commit, auto-backup,
			// auto-export and auto-push all mutate the very store being
			// migrated, and `issuegraph list` on a frozen workspace reaches every one
			// of them. Guarding the outer block keeps that promise in one
			// place instead of six. Writes never get here at all: they were
			// refused in PersistentPreRunE.
			if runsPostCommandMaintenance(cmd.Name(), readonlyMode) && !commandFreeze.Frozen() {
				// Dolt auto-commit: after a successful write command (and after final flush),
				// create a Dolt commit so changes don't remain only in the working set.
				if commandDidWrite.Load() && !commandDidExplicitDoltCommit {
					if err := runPostRunAutoCommit(rootCtx, doltAutoCommitParams{Command: cmd.Name()}); err != nil {
						return HandleError("dolt auto-commit failed: %v", err)
					}
				}

				// Tip metadata auto-commit: if a tip was shown, create a separate Dolt commit for the
				// tip_*_last_shown metadata updates. This may happen even for otherwise read-only commands.
				if commandDidWriteTipMetadata && len(commandTipIDsShown) > 0 {
					// Only applies when dolt auto-commit is enabled and backend is versioned (Dolt).
					if mode, err := getDoltAutoCommitMode(); err != nil {
						return HandleError("dolt tip auto-commit failed: %v", err)
					} else if mode == doltAutoCommitOn {
						// Apply tip metadata writes now (deferred in recordTipShown for Dolt).
						//
						// A store that refuses writes by construction — the
						// preview open, and strict --readonly — must not turn
						// an otherwise successful command into a non-zero exit
						// here. This block is deliberately not gated by the
						// read-only classification, and that has been fine
						// because OpenForReadOnlyCommand is "otherwise a normal
						// writable store"; the write-refusing opens break that
						// assumption. Tip bookkeeping is incidental and
						// recordTipShown's own contract is that it may fail
						// silently, so skip it and carry on.
						tipWritesRefused := false
						for tipID := range commandTipIDsShown {
							key := fmt.Sprintf("tip_%s_last_shown", tipID)
							value := time.Now().Format(time.RFC3339)
							if err := store.SetLocalMetadata(rootCtx, key, value); err != nil {
								if errors.Is(err, embeddeddolt.ErrReadOnly) {
									debug.Logf("tip auto-commit: store is read-only, skipping tip metadata: %v", err)
									tipWritesRefused = true
									break
								}
								return HandleError("dolt tip auto-commit failed: %v", err)
							}
						}

						if !tipWritesRefused {
							ids := make([]string, 0, len(commandTipIDsShown))
							for tipID := range commandTipIDsShown {
								ids = append(ids, tipID)
							}
							msg := formatDoltAutoCommitMessage("tip", getActor(), ids)
							if err := runPostRunAutoCommit(rootCtx, doltAutoCommitParams{Command: "tip", MessageOverride: msg}); err != nil {
								return HandleError("dolt tip auto-commit failed: %v", err)
							}
						}
					}
				}

				// Auto-backup: sync a Dolt-native backup if enabled and due
				runPostRunAutoBackup(rootCtx)

				// Auto-export: write git-tracked JSONL for portability if enabled and due.
				// Read-only commands must not perform post-run maintenance writes or emit
				// sync guidance after machine-readable output.
				if shouldRunPostCommandAutoExport(cmd) {
					if err := runPostRunAutoExport(rootCtx, commandAllowsEmptyAutoExport(cmd)); err != nil {
						return HandleError("%v", err)
					}
				}

				// Auto-push: push to Dolt remote if enabled and due.
				// Skip for read-only commands to avoid unnecessary network operations
				// and metadata writes on commands like issuegraph list/show/ready (GH#2191).
				if !isReadOnlyCommand(cmd.Name()) {
					runPostRunAutoPush(rootCtx)
				}

				// Events-journal retention, LAST in the maintenance net. It is
				// the only step here that serves nobody but the database
				// itself, so everything the user can observe — the commit, the
				// backup, the export, the push — is already done and durable
				// before a maintenance transaction opens. Its failures are
				// logged, never returned.
				//
				// COMBINED ORDERING with the hook teardown above, since both
				// land in this function and each has its own reason:
				// maintenance runs in the BODY, so it is finished before the
				// first defer; the defers then run close-and-release, then
				// waitForCommandHooks, then restoreChangeDirSelection, then the
				// context cancel. That is the only order in which both hold.
				// Auto-prune needs an OPEN store, which the body still has and
				// the hook wait deliberately does not (it is sequenced after
				// the close so a hook that shells out to issuegraph can take the
				// embedded Dolt lock). And it must not be deferred alongside
				// them: it would then either run after the store closed, or
				// delay the close the hook children are waiting on. Its cost is
				// bounded — one indexed query when nothing is due, a 30s pass
				// budget at worst — so the hook wait it precedes starts
				// essentially on time.
				if shouldAutoPruneEventsJournal(cmd) {
					maybeAutoPruneEventsJournal(rootCtx, issuegraph.FindIssueGraphDir())
				}
			}

			// Signal that store is closing (prevents background flush from accessing closed store)
			storeMutex.Lock()
			storeActive = false
			storeMutex.Unlock()

			if store != nil {
				_ = store.Close() // Best effort cleanup
				// Mark closed so the deferred gate-release cleanup above
				// does not double-close it.
				store = nil
			}
		}

		// End the command span and flush OTel data before process exit.
		if commandSpan != nil {
			commandSpan.End()
			commandSpan = nil
		}
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		telemetry.Shutdown(shutdownCtx)
		shutdownCancel()

		if profileFile != nil {
			pprof.StopCPUProfile()
			_ = profileFile.Close() // Best effort cleanup
		}
		if traceFile != nil {
			trace.Stop()
			_ = traceFile.Close() // Best effort cleanup
		}

		// Heap profiling: --mem-profile flag or ISSUEGRAPH_MEM_PROFILE env var.
		// Runs a GC first by default; ISSUEGRAPH_MEM_PROFILE_NOGC=1 skips it to capture peak.
		heapDest := memProfilePath
		if heapDest == "" {
			heapDest = os.Getenv("ISSUEGRAPH_MEM_PROFILE")
		}
		if heapDest != "" {
			if os.Getenv("ISSUEGRAPH_MEM_PROFILE_NOGC") == "" {
				runtime.GC()
			}
			if f, err := os.Create(heapDest); err == nil { // #nosec G304 -- user-supplied profiling path
				_ = pprof.WriteHeapProfile(f)
				_ = f.Close()
			}
		}
		// Optional one-line MemStats summary: ISSUEGRAPH_MEM_STATS=/path/to/stats.txt
		if statsDest := os.Getenv("ISSUEGRAPH_MEM_STATS"); statsDest != "" {
			var ms runtime.MemStats
			runtime.ReadMemStats(&ms)
			if f, err := os.Create(statsDest); err == nil { // #nosec G304 -- user-supplied profiling path
				fmt.Fprintf(f, "HeapAlloc=%d HeapSys=%d HeapInuse=%d HeapObjects=%d\n",
					ms.HeapAlloc, ms.HeapSys, ms.HeapInuse, ms.HeapObjects)
				_ = f.Close()
			}
		}

		// The signal context is canceled and cleared by the deferred hook
		// registered at the top of this function, so that it also covers the
		// early error returns above.
		return nil
	},
}

func shouldRunPostCommandAutoExport(cmd *cobra.Command) bool {
	if cmd == nil {
		return true
	}
	return !isReadOnlyCommand(cmd.Name())
}

func shouldRunAutoImportJSONL(cmd *cobra.Command, s storage.DoltStorage, useReadOnly, globalFlag, serverMode bool) bool {
	if cmd == nil || s == nil || useReadOnly || globalFlag || serverMode {
		return false
	}
	// import.auto=false (or ISSUEGRAPH_IMPORT_AUTO=false) must disable ALL auto-import
	// behavior, not just the git-hook sync path (importJSONLForSync). Without
	// this check, a fresh/empty database would silently auto-import stale
	// issues.jsonl on every write command regardless of the config setting
	// (GH#4304).
	if !config.GetBool("import.auto") {
		return false
	}
	return cmd.Name() != "import"
}

// isDisablingImportAutoViaConfigCommand reports whether the command about to
// run is "issuegraph config set import.auto false" (or an equivalent
// "issuegraph config set-many ... import.auto=false" pair). shouldRunAutoImportJSONL
// runs in PersistentPreRun before configSetCmd/configSetManyCmd write the new
// value to config.yaml, so without this exemption the master switch would
// trigger the very auto-import it is meant to disable on its own invocation
// when a stale .issuegraph/issues.jsonl sits next to an empty database (GH#4304).
func isDisablingImportAutoViaConfigCommand(cmd *cobra.Command, args []string) bool {
	if cmd == nil || cmd.Parent() == nil || cmd.Parent().Name() != "config" {
		return false
	}
	switch cmd.Name() {
	case "set":
		return len(args) >= 2 && args[0] == "import.auto" && isFalsyConfigValue(args[1])
	case "set-many":
		for _, arg := range args {
			key, value, ok := strings.Cut(arg, "=")
			if ok && key == "import.auto" && isFalsyConfigValue(value) {
				return true
			}
		}
	}
	return false
}

// isFalsyConfigValue reports whether a config value string parses as a
// boolean false (e.g. "false", "0", "f").
func isFalsyConfigValue(value string) bool {
	parsed, err := strconv.ParseBool(value)
	return err == nil && !parsed
}

func commandAllowsEmptyAutoExport(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}
	switch cmd.Name() {
	case "prune", "purge":
		return commandMayEmptyJSONLExport.Load()
	default:
		return false
	}
}

// blockedEnvVars lists environment variables that must not be set because they
// could silently override the storage backend via viper's AutomaticEnv, causing
// data fragmentation (bd-hevyw).
var blockedEnvVars = []string{"ISSUEGRAPH_BACKEND", "ISSUEGRAPH_DATABASE_BACKEND"}

// checkBlockedEnvVars returns an error if any blocked env vars are set.
func checkBlockedEnvVars() error {
	for _, name := range blockedEnvVars {
		if os.Getenv(name) != "" {
			return fmt.Errorf("%s env var is not supported and has been removed to prevent data fragmentation.\n"+
				"Unset %s; storage selection comes from .issuegraph/metadata.json. To choose a different supported backend, follow 'issuegraph help init-safety'; do not edit metadata.json by hand", name, name)
		}
	}
	return nil
}

// setupGracefulShutdown creates a context that cancels on SIGINT/SIGTERM/SIGHUP.
// Before cancellation, it flushes pending batch commits so that accumulated
// changes in the Dolt working set are not lost on graceful shutdown.
func setupGracefulShutdown() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background()) //nolint:gosec // G118: cancel is returned and called by caller

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	go func() {
		select {
		case <-sigCh:
			flushBatchCommitOnShutdown()
			cancel()
			// On second signal, force exit
			<-sigCh
			os.Exit(1)
		case <-ctx.Done():
			signal.Stop(sigCh)
		}
	}()

	return ctx, cancel
}

// flushBatchCommitOnShutdown commits any pending batch changes before process exit.
// This prevents data loss when SIGTERM/SIGHUP kills a process with uncommitted
// batch writes sitting in the Dolt working set.
func flushBatchCommitOnShutdown() {
	mode, err := getDoltAutoCommitMode()
	if err != nil || mode != doltAutoCommitBatch {
		return
	}

	storeMutex.Lock()
	active := storeActive
	st := store
	storeMutex.Unlock()

	if !active || st == nil {
		return
	}

	// Use a fresh context with timeout — rootCtx is about to be canceled.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// CommitPending reports atomically whether a commit actually landed, so a
	// clean shutdown stays quiet without spending the 5s flush budget on
	// HEAD-reporting probes before the commit itself (and without racing a
	// concurrent writer's HEAD movement the way a before/after compare would).
	committed, err := st.CommitPending(ctx, getActorWithGit())
	if err != nil {
		if !isDoltNothingToCommit(err) {
			fmt.Fprintf(os.Stderr, "\nWarning: failed to flush batch commit on shutdown: %v\n", err)
		}
		return
	}
	if !committed {
		return
	}

	fmt.Fprintf(os.Stderr, "\nFlushed pending batch commit on shutdown\n")
}

// validateWorkspaceIdentity checks that the project identity from metadata.json
// matches the database's stored project_id. A mismatch indicates configuration
// drift — the CLI may be pointing at the wrong database (GH#2438, GH#2372).
//
// This check only runs for write commands because:
// 1. Read commands are safe even against wrong databases (no data mutation)
// 2. The check requires an open store connection
// 3. New databases won't have _project_id yet (bootstrap case)
func validateWorkspaceIdentity(ctx context.Context, issuegraphDir string) error {
	if store == nil {
		return nil // No store connection, nothing to validate
	}

	// Load project_id from metadata.json
	cfg, err := configfile.Load(issuegraphDir)
	if err != nil || cfg == nil {
		return nil // No config, skip validation (fresh init)
	}
	configProjectID := cfg.ProjectID
	if configProjectID == "" {
		return nil // No project_id in config (pre-identity era)
	}

	// Get project_id from database
	dbProjectID, err := store.GetMetadata(ctx, "_project_id")
	if err != nil || dbProjectID == "" {
		return nil // No project_id in DB (new or pre-identity database)
	}

	// Compare: mismatch means drift
	if configProjectID != dbProjectID {
		fmt.Fprintf(os.Stderr, "Error: workspace identity mismatch detected\n\n")
		fmt.Fprintf(os.Stderr, "  metadata.json project_id: %s\n", configProjectID)
		fmt.Fprintf(os.Stderr, "  database _project_id:     %s\n\n", dbProjectID)
		fmt.Fprintf(os.Stderr, "This means the CLI config and database belong to different projects.\n")
		fmt.Fprintf(os.Stderr, "Possible causes:\n")
		fmt.Fprintf(os.Stderr, "  • ISSUEGRAPH_DIR points to a different project's .issuegraph/\n")
		fmt.Fprintf(os.Stderr, "  • Dolt server endpoint changed and now serves a different database\n")
		fmt.Fprintf(os.Stderr, "  • metadata.json was copied from another project\n\n")
		fmt.Fprintf(os.Stderr, "Recovery: run 'issuegraph doctor --fix' or 'issuegraph bootstrap' to reconcile workspace metadata with the authoritative database when shared-server metadata drifted.\n")
		fmt.Fprintf(os.Stderr, "To diagnose: issuegraph context --json\n")
		fmt.Fprintf(os.Stderr, "To override: set ISSUEGRAPH_SKIP_IDENTITY_CHECK=1\n")
		return SilentExit()
	}
	return nil
}

func main() {
	// ISSUEGRAPH_NAME overrides the binary name in help text (e.g. ISSUEGRAPH_NAME=ops makes
	// "ops --help" show "ops" instead of "issuegraph"). Useful for multi-instance
	// setups where wrapper scripts set ISSUEGRAPH_DIR for routing.
	if name := os.Getenv("ISSUEGRAPH_NAME"); name != "" {
		rootCmd.Use = name
	}

	// Register --all flag on Cobra's auto-generated help command.
	// Must be called after init() so all subcommands are registered and
	// Cobra has created its default help command.
	rootCmd.InitDefaultHelpCmd()
	registerHelpAllFlag()

	executedCmd, err := rootCmd.ExecuteC()

	// Let this command's fire-and-forget hooks finish, for the same
	// every-exit-path reason the metrics flush below is here rather than in
	// PersistentPostRunE: cobra SKIPS PostRunE when RunE returns an error, so a
	// partial batch — `issuegraph close A B` where A commits and B refuses — would exit
	// with A's committed mutation never reaching its hook script. Idempotent, so
	// the PostRunE call on the clean path makes this one free.
	waitForCommandHooks()

	// Finalize queued metrics and detach the uploader. Shared with the os.Exit
	// guards (CheckReadonly and the pre-run gates) so every exit path flushes the
	// same way instead of only the clean RunE/ExecuteC return.
	metrics.CloseAndFlush()

	if err != nil {
		if code, ok := exitCodeFromError(err); ok {
			os.Exit(code)
		}
		if executedCmd != nil && executedCmd.SilenceErrors {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		}
		os.Exit(1)
	}
}

func resolveMetricsEnabled() bool {
	if v, ok := os.LookupEnv(metrics.EnvDisableMetrics); ok {
		return !envTruthyValue(v)
	}
	// DO_NOT_TRACK is a disable-only alias: a truthy value opts out, but a
	// falsey or empty value (DO_NOT_TRACK=0/false/"") must fall through to the
	// user's saved preference instead of forcing metrics back on over a saved
	// `issuegraph metrics off`. Only ISSUEGRAPH_DISABLE_METRICS (checked first) is a
	// bidirectional override.
	if v, ok := os.LookupEnv(metrics.EnvDoNotTrack); ok && envTruthyValue(v) {
		return false
	}
	// Consent is the user's own global choice: resolve it from the user-global
	// config only, never merged project/ISSUEGRAPH_DIR config. Otherwise a
	// repository's .issuegraph/config.yaml (highest viper precedence) could re-enable
	// metrics for a user who ran `issuegraph metrics off`.
	return !config.MetricsDisabledByUserConfig()
}

func resolveMetricsEndpoint() string {
	if v := os.Getenv(metrics.EnvEndpoint); v != "" {
		return v
	}
	// Like enablement, the endpoint is resolved from env + user-global config
	// only so a repository can never redirect where a user's metrics are sent.
	if ep := config.UserMetricsEndpoint(); ep != "" {
		return ep
	}
	return metrics.DefaultEndpoint
}

func envTruthyValue(v string) bool {
	if v == "" {
		return false
	}
	switch strings.ToLower(v) {
	case "0", "false":
		return false
	}
	return true
}

// secretFlagNames are long flag names whose entire value is an opaque credential
// that must never reach the issuegraph.args telemetry span. The flag's value is redacted
// wholesale. Only federation add-peer's --password currently qualifies. Its shorthand (-p) is
// resolved per command via secretFlagTokens so the same letter bound to
// --priority/--prefix/--parallel on other commands is never redacted.
var secretFlagNames = map[string]bool{"password": true}

// secretFlagTokens returns the concrete --long and -short flag tokens that carry a
// secret value for cmd. Resolving against the running command is what makes the
// redaction "by flag identity": -p is treated as secret only on the command that
// actually binds it to a secret flag (federation add-peer), not on the many
// commands that bind -p to a non-secret option.
func secretFlagTokens(cmd *cobra.Command) map[string]bool {
	tokens := make(map[string]bool)
	if cmd == nil {
		return tokens
	}
	for name := range secretFlagNames {
		f := cmd.Flags().Lookup(name)
		if f == nil {
			continue
		}
		tokens["--"+f.Name] = true
		if f.Shorthand != "" {
			tokens["-"+f.Shorthand] = true
		}
	}
	return tokens
}

// scrubArgsForTelemetry joins argv for the issuegraph.args span attribute with any
// credential-bearing values redacted. A secretFlags token's value is redacted
// wholesale across the `--password <v>`,
// `--password=<v>`, `-p <v>`, `-p=<v>`, and `-p<v>` spellings pflag accepts. Every
// other arg gets a conservative DSN/userinfo scrub as defense in depth so a
// positional connection string cannot leak a password.
func scrubArgsForTelemetry(argv []string, secretFlags map[string]bool) string {
	parts := make([]string, len(argv))
	redactNext := false
	for i, a := range argv {
		if redactNext {
			parts[i] = "xxxxx"
			redactNext = false
			continue
		}
		if name, value, ok := strings.Cut(a, "="); ok {
			if secretFlags[name] {
				// --password=<secret> / -p=<secret> — redact the whole value.
				parts[i] = name + "=xxxxx"
				continue
			}
			if strings.HasPrefix(name, "-") {
				scrubbed := scrubUserinfoPassword(scrubPotentialDSNPasswords(value))
				if scrubbed != value {
					// Preserve an arbitrary flag name while parsing its equals-value as
					// a possible DSN. Passing the whole token to url.Parse would treat
					// the flag prefix as the URL scheme and miss query credentials.
					parts[i] = name + "=" + scrubbed
					continue
				}
			}
		}
		if i > 0 {
			if secretFlags[argv[i-1]] {
				// <secret> following a bare --password / -p token.
				parts[i] = "xxxxx"
				continue
			}
		}
		if short, ok := secretShorthandPrefix(a, secretFlags); ok {
			// -p<secret> — pflag's concatenated shorthand spelling.
			parts[i] = short + "xxxxx"
			continue
		}
		if secretShorthandTakesSeparateValue(a, secretFlags) {
			// -qp <secret> — a boolean shorthand cluster ending in the
			// value-taking secret shorthand, with its value in the next token.
			parts[i] = a
			redactNext = true
			continue
		}
		parts[i] = scrubUserinfoPassword(scrubPotentialDSNPasswords(a))
	}
	return strings.Join(parts, " ")
}

// secretShorthandPrefix reports whether a is pflag's concatenated secret-shorthand
// spelling, returning the "-x...-p" prefix to preserve. Long flags cannot concatenate
// a value, so only -X<value> shorthands are matched.
//
// pflag also accepts a CLUSTER of boolean shorthands ending in a value-taking
// shorthand: given boolean flags -q/-v and value flag -p, "-qpSECRET" parses as -q
// followed by -p SECRET, and "-vpSECRET" parses as -v followed by -p SECRET — but the
// raw token still reaches telemetry as one string. Walk the leading run of letters in
// a; the first letter whose "-x" token is a registered secret shorthand ends the
// cluster, and everything after it is that flag's value, regardless of how many
// boolean shorthands preceded it. This mirrors pflag's own grammar (a cluster is zero
// or more boolean shorthands followed by one value-taking shorthand) without needing
// the running command's flag set here: it is conservative in the safe direction,
// since treating a longer prefix as consumed by the secret shorthand only ever
// over-redacts, never under-redacts.
func secretShorthandPrefix(a string, secretFlags map[string]bool) (string, bool) {
	if len(a) < 3 || a[0] != '-' || a[1] == '-' {
		return "", false
	}
	for i := 1; i < len(a); i++ {
		c := a[i]
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')) {
			return "", false
		}
		if secretFlags["-"+string(c)] {
			if i+1 >= len(a) {
				return "", false // no value follows; not the concatenated spelling
			}
			return a[:i+1], true
		}
	}
	return "", false
}

// secretShorthandTakesSeparateValue recognizes a boolean-shorthand cluster that
// ends in a registered secret shorthand with no attached value. For example,
// pflag parses "-qp secret" as -q followed by -p=secret.
func secretShorthandTakesSeparateValue(a string, secretFlags map[string]bool) bool {
	if len(a) < 3 || a[0] != '-' || a[1] == '-' {
		return false
	}
	for i := 1; i < len(a); i++ {
		c := a[i]
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')) {
			return false
		}
		if secretFlags["-"+string(c)] {
			return i == len(a)-1
		}
	}
	return false
}

// scrubUserinfoPassword redacts the password in a URL/DSN userinfo section
// (postgres://user:PASS@host or user:PASS@tcp(...)); args without a user:pass@
// userinfo pass through unchanged, so ordinary text is never mangled.
func scrubUserinfoPassword(a string) string {
	at := strings.LastIndexByte(a, '@')
	if at < 0 {
		return a
	}
	head := a[:at]
	start := 0
	if s := strings.LastIndex(head, "//"); s >= 0 {
		start = s + 2 // userinfo begins after the scheme's "//"
	}
	colon := strings.IndexByte(head[start:], ':')
	if colon < 0 {
		return a // no "user:pass" userinfo, nothing to redact
	}
	return head[:start+colon+1] + "xxxxx" + a[at:]
}
