package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/borg0ai/dx-issuegraph/internal/config"
	"github.com/borg0ai/dx-issuegraph/internal/configfile"
	"github.com/borg0ai/dx-issuegraph/internal/doltserver"
	"github.com/borg0ai/dx-issuegraph/internal/issuegraph"
	"github.com/borg0ai/dx-issuegraph/internal/lockfile"
	"github.com/borg0ai/dx-issuegraph/internal/metrics"
	"github.com/borg0ai/dx-issuegraph/internal/storage/dbproxy/proxy"
	"github.com/borg0ai/dx-issuegraph/internal/storage/dbproxy/server"
	"github.com/borg0ai/dx-issuegraph/internal/storage/dbproxy/util"
	"github.com/borg0ai/dx-issuegraph/internal/ui"
)

const migrateLockFileName = "migrate.lock"

const migrateJournalFileName = "dolt-mode-migration.json"

type migratePhase string

const (
	migratePrepared           migratePhase = "prepared"
	migrateTargetConfigured   migratePhase = "target_configured"
	migrateOldControlsRetired migratePhase = "old_controls_retired"
	migrateVerified           migratePhase = "verified"
	migrateCommitted          migratePhase = "committed"
)

type migrateJournal struct {
	Version    int                                 `json:"version"`
	SourceMode string                              `json:"source_mode"`
	TargetMode string                              `json:"target_mode"`
	Shared     bool                                `json:"shared"`
	RootPath   string                              `json:"root_path,omitempty"`
	External   *configfile.ExternalDoltConfig      `json:"external,omitempty"`
	Sidecar    *configfile.ProxiedServerClientInfo `json:"sidecar,omitempty"`
	LogAssets  []string                            `json:"log_assets,omitempty"`
	Ownership  string                              `json:"ownership"`
	Attempt    int                                 `json:"attempt"`
	Phase      migratePhase                        `json:"phase"`
}
type migrateDryRunOutput struct {
	DryRun     bool   `json:"dry_run"`
	SourceMode string `json:"source_mode"`
	TargetMode string `json:"target_mode"`
	Shared     bool   `json:"shared"`
	RootPath   string `json:"root_path"`
}

func migrateJournalPath(issuegraphDir string) string {
	return filepath.Join(issuegraphDir, migrateJournalFileName)
}

func loadMigrateJournal(issuegraphDir string) (*migrateJournal, error) {
	b, err := os.ReadFile(migrateJournalPath(issuegraphDir))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", migrateJournalFileName, err)
	}
	var j migrateJournal
	if err := json.Unmarshal(b, &j); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", migrateJournalFileName, err)
	}
	if j.Version != 1 || j.SourceMode == "" || j.TargetMode == "" || j.Phase == "" || j.Attempt < 1 {
		return nil, fmt.Errorf("invalid %s: missing required migration state", migrateJournalFileName)
	}
	switch j.Phase {
	case migratePrepared, migrateTargetConfigured, migrateOldControlsRetired, migrateVerified, migrateCommitted:
	default:
		return nil, fmt.Errorf("invalid %s: unknown phase %q", migrateJournalFileName, j.Phase)
	}
	if j.RootPath == "" || !filepath.IsAbs(j.RootPath) || filepath.Clean(j.RootPath) != j.RootPath {
		return nil, fmt.Errorf("invalid %s: root_path must be a clean absolute path", migrateJournalFileName)
	}
	if j.SourceMode == configfile.DoltModeServer && j.TargetMode == configfile.DoltModeProxiedServer {
		if j.Shared { /* shared-server uses server mode plus shared topology */
		}
	} else if j.SourceMode == configfile.DoltModeProxiedServer && (j.TargetMode == configfile.DoltModeServer || j.TargetMode == "shared-server") {
		if j.TargetMode == "shared-server" && !j.Shared {
			return nil, fmt.Errorf("invalid %s: shared target requires shared topology", migrateJournalFileName)
		}
		if j.TargetMode == configfile.DoltModeServer && j.Shared {
			return nil, fmt.Errorf("invalid %s: server target cannot use shared topology", migrateJournalFileName)
		}
	} else {
		return nil, fmt.Errorf("invalid %s: unsupported mode transition %s -> %s", migrateJournalFileName, j.SourceMode, j.TargetMode)
	}
	if j.Sidecar == nil {
		return nil, fmt.Errorf("invalid %s: missing sidecar topology", migrateJournalFileName)
	}
	if j.Sidecar.RootPath == "" || j.Sidecar.ResolvedRootPath(issuegraphDir) == "" {
		return nil, fmt.Errorf("invalid %s: sidecar root_path is required", migrateJournalFileName)
	}
	if j.Ownership != "managed-local" && j.Ownership != "external" {
		return nil, fmt.Errorf("invalid %s: unknown ownership %q", migrateJournalFileName, j.Ownership)
	}
	// Journals written before the External field was added recorded the
	// endpoint only in the sidecar. Infer it for compatibility so recovery
	// still takes the typed, fail-closed external path.
	if j.External == nil && j.Ownership == "external" && j.Sidecar.External != nil {
		j.External = j.Sidecar.External
	}
	if j.Ownership == "external" && j.Sidecar.External == nil {
		return nil, fmt.Errorf("invalid %s: external ownership missing endpoint", migrateJournalFileName)
	}
	if j.Ownership == "managed-local" && j.Sidecar.External != nil {
		return nil, fmt.Errorf("invalid %s: managed-local ownership has external endpoint", migrateJournalFileName)
	}
	if !reflect.DeepEqual(j.External, j.Sidecar.External) {
		return nil, fmt.Errorf("invalid %s: external topology disagrees with sidecar", migrateJournalFileName)
	}
	if j.Sidecar.External != nil {
		if err := j.Sidecar.External.Validate(); err != nil {
			return nil, fmt.Errorf("invalid %s external topology: %w", migrateJournalFileName, err)
		}
	}
	if rp := j.Sidecar.ResolvedRootPath(issuegraphDir); rp != "" && filepath.Clean(rp) != j.RootPath {
		return nil, fmt.Errorf("invalid %s: sidecar root does not match root_path", migrateJournalFileName)
	}
	return &j, nil
}

func saveMigrateJournal(issuegraphDir string, j *migrateJournal) error {
	b, err := json.MarshalIndent(j, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(issuegraphDir, ".dolt-mode-migration-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if err = tmp.Chmod(0o600); err == nil {
		_, err = tmp.Write(b)
	}
	if err == nil {
		err = tmp.Sync()
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("writing %s: %w", migrateJournalFileName, err)
	}
	if err = os.Rename(tmpName, migrateJournalPath(issuegraphDir)); err != nil {
		return err
	}
	d, err := os.Open(issuegraphDir) // #nosec G304 -- issuegraphDir is the discovered workspace directory
	if err != nil {
		return fmt.Errorf("syncing %s directory: %w", migrateJournalFileName, err)
	}
	if err = d.Sync(); err != nil {
		_ = d.Close()
		return fmt.Errorf("syncing %s directory: %w", migrateJournalFileName, err)
	}
	if err = d.Close(); err != nil {
		return fmt.Errorf("closing %s directory: %w", migrateJournalFileName, err)
	}
	return nil
}

func removeMigrateJournal(issuegraphDir string) error {
	if err := os.Remove(migrateJournalPath(issuegraphDir)); err != nil && !os.IsNotExist(err) {
		return err
	}
	d, err := os.Open(issuegraphDir) // #nosec G304 -- issuegraphDir is the discovered workspace directory
	if err != nil {
		return fmt.Errorf("opening migration directory: %w", err)
	}
	if err = d.Sync(); err != nil {
		_ = d.Close()
		return fmt.Errorf("syncing migration directory: %w", err)
	}
	if err = d.Close(); err != nil {
		return fmt.Errorf("closing migration directory: %w", err)
	}
	return nil
}

func migrateFault(phase migratePhase) error {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("ISSUEGRAPH_MIGRATION_FAIL_PHASE")), string(phase)) {
		return fmt.Errorf("migration fault injected after phase %s", phase)
	}
	return nil
}

// externalMigrationRefusal is a policy refusal rather than a registry row: it
// depends on the workspace's endpoint, not on which command was typed. The
// reason is design because someone else owns that endpoint and its schema —
// no amount of plumbing on this side makes migrating it issuegraph's business.
func externalMigrationRefusal(message string) error {
	return HandleProxyCapabilityError(&ProxyCapabilityError{
		Code:     "proxy.migrate.external_endpoint",
		Message:  message,
		ExitCode: 1,
		Mutates:  false,
		Reason:   ProxyReasonDesign,
	})
}

func migrationInvalidState(message string) error {
	return HandleProxyCapabilityError(&ProxyCapabilityError{
		Code:     "proxy.migrate.invalid_state",
		Message:  message,
		ExitCode: 1,
		Mutates:  false,
	})
}

// validateCompletedProxyControls verifies that an already-proxied target has
// no live process or stale lifecycle artifacts. Empty, unlocked lock markers
// are harmless leftovers from a prior lifecycle and are intentionally
// accepted; a held marker still proves active ownership and is refused.
func validateCompletedProxyControls(root string) error {
	if running, _ := proxy.IsRunning(root); running {
		return migrationInvalidState("completed migration still has a running proxy")
	}
	for _, control := range proxy.ControlFilePaths(root) {
		if _, statErr := os.Stat(control); statErr != nil {
			if os.IsNotExist(statErr) {
				continue
			}
			return migrationInvalidState(fmt.Sprintf("completed migration cannot inspect proxy control %s: %v", control, statErr))
		}
		base := filepath.Base(control)
		if base == proxy.LockFileName || base == server.LockFileName {
			lock, lockErr := util.TryLock(control)
			if lockErr == nil {
				lock.Unlock()
				continue
			}
			return migrationInvalidState(fmt.Sprintf("completed migration has a held proxy control %s", base))
		}
		return migrationInvalidState(fmt.Sprintf("completed migration has stale proxy control %s", base))
	}
	return nil
}

func validateCompletedLockMarker(path string, label string) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return migrationInvalidState(fmt.Sprintf("completed migration cannot inspect %s: %v", label, err))
	}
	lock, err := util.TryLock(path)
	if err != nil {
		return migrationInvalidState(fmt.Sprintf("completed migration has a held %s", label))
	}
	lock.Unlock()
	return nil
}

func migrationSidecarsEquivalent(issuegraphDir string, a, b *configfile.ProxiedServerClientInfo) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.ResolvedRootPath(issuegraphDir) == b.ResolvedRootPath(issuegraphDir) &&
		a.ResolvedConfigPath(issuegraphDir) == b.ResolvedConfigPath(issuegraphDir) &&
		a.ResolvedLogPath(issuegraphDir) == b.ResolvedLogPath(issuegraphDir) &&
		a.Port == b.Port && a.IdleTimeout == b.IdleTimeout &&
		reflect.DeepEqual(a.External, b.External)
}

func checkpointMigration(issuegraphDir string, j *migrateJournal, phase migratePhase) error {
	j.Phase = phase
	if err := saveMigrateJournal(issuegraphDir, j); err != nil {
		return err
	}
	return migrateFault(phase)
}

func validateMigrationJournalAgainstConfig(j *migrateJournal, cfg *configfile.Config) error {
	if j == nil || cfg == nil {
		return nil
	}
	if j.Phase == migratePrepared {
		if cfg.GetDoltMode() != j.SourceMode && cfg.GetDoltMode() != j.TargetMode && !(j.TargetMode == "shared-server" && cfg.IsDoltServerMode()) {
			return fmt.Errorf("migration journal phase %s disagrees with metadata mode %q", j.Phase, cfg.GetDoltMode())
		}
		return nil
	}
	if j.TargetMode == configfile.DoltModeProxiedServer && !cfg.IsDoltProxiedServerMode() {
		return fmt.Errorf("migration journal phase %s requires proxied-server metadata", j.Phase)
	}
	if j.TargetMode != configfile.DoltModeProxiedServer && !cfg.IsDoltServerMode() {
		return fmt.Errorf("migration journal phase %s requires server metadata", j.Phase)
	}
	return nil
}

// sameMigrationRoot reports whether two spellings name the same Dolt root.
//
// A recorded journal/sidecar root and the canonical root derived at run time
// can be lexically different yet identical on disk: on macOS the temp and
// data roots reach the same directory through the /var -> /private/var
// symlink, so a filepath.Clean comparison alone rejects a valid topology.
// Resolving the deepest existing component canonicalizes that prefix while
// still working when the leaf itself has not been created yet.
func sameMigrationRoot(a, b string) bool {
	if filepath.Clean(a) == filepath.Clean(b) {
		return true
	}
	return resolveExistingMigrationPath(a) == resolveExistingMigrationPath(b)
}

func resolveExistingMigrationPath(path string) string {
	if path == "" || !filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	current := filepath.Clean(path)
	var trailing string
	for {
		if resolved, err := filepath.EvalSymlinks(current); err == nil {
			return filepath.Join(resolved, trailing)
		}
		parent := filepath.Dir(current)
		if parent == current {
			return filepath.Clean(path)
		}
		trailing = filepath.Join(filepath.Base(current), trailing)
		current = parent
	}
}

func validateMigrationTopology(issuegraphDir string, j *migrateJournal, shared bool, forward bool) error {
	if j == nil {
		return nil
	}
	canonical := doltserver.DoltDirPath(issuegraphDir)
	if shared {
		var err error
		canonical, err = doltserver.SharedDoltPath()
		if err != nil {
			return fmt.Errorf("resolve shared Dolt root: %w", err)
		}
	}
	if !sameMigrationRoot(j.RootPath, canonical) {
		return fmt.Errorf("journal root %s does not match canonical %s", j.RootPath, canonical)
	}
	if shared {
		want := "true"
		if j.Phase != migratePrepared {
			want = "false"
		}
		if !forward {
			want = "true"
			if j.Phase == migratePrepared {
				want = "false"
			}
		}
		if v, ok := config.WorkspaceYamlValue(issuegraphDir, "dolt.shared-server"); !ok || strings.ToLower(v) != want {
			return fmt.Errorf("shared-server YAML is %q, want %s for phase %s", v, want, j.Phase)
		}
	}
	return nil
}

func validateDoltRoot(root string) error {
	b, err := os.ReadFile(filepath.Join(root, ".dolt", "repo_state.json")) // #nosec G304 -- root is validated migration workspace path
	if err != nil {
		return fmt.Errorf("read Dolt identity: %w", err)
	}
	var obj struct {
		Head     string                 `json:"head"`
		Remotes  map[string]interface{} `json:"remotes"`
		Backups  map[string]interface{} `json:"backups"`
		Branches map[string]interface{} `json:"branches"`
	}
	if err := json.Unmarshal(b, &obj); err != nil || obj.Head == "" || !strings.HasPrefix(obj.Head, "refs/heads/") || strings.TrimPrefix(obj.Head, "refs/heads/") == "" || obj.Remotes == nil || obj.Backups == nil || obj.Branches == nil {
		return fmt.Errorf("invalid Dolt identity repo_state.json")
	}
	return nil
}

func migrateToProxiedRunE(metricName, checkName string, shared bool) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		evt := metrics.NewCommandEvent(metricName)
		defer func() {
			if c := metrics.Global(); c != nil {
				c.CloseEventAndAdd(evt)
			}
		}()

		dryRun, _ := cmd.Flags().GetBool("dry-run")
		if !dryRun {
			CheckReadonly(checkName)
		}

		idleTimeout, err := resolveMigrateIdleTimeout(cmd)
		if err != nil {
			return err
		}
		return runMigrateToProxiedServer(dryRun, idleTimeout, shared)
	}
}

func migrateFromProxiedRunE(metricName, checkName string, shared bool) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		evt := metrics.NewCommandEvent(metricName)
		defer func() {
			if c := metrics.Global(); c != nil {
				c.CloseEventAndAdd(evt)
			}
		}()

		dryRun, _ := cmd.Flags().GetBool("dry-run")
		if !dryRun {
			CheckReadonly(checkName)
		}
		return runMigrateFromProxiedServer(dryRun, shared)
	}
}

var migrateToProxiedServerCmd = &cobra.Command{
	Use:           "from-server-to-proxied-server",
	Short:         "[EXPERIMENTAL] Switch a server-mode repo to proxied-server mode",
	SilenceUsage:  true,
	SilenceErrors: true,
	Long: `Switch a repo from server mode (issuegraph init --server) to proxied-server mode.

Both modes root their dolt sql-server at the same .issuegraph/dolt directory, so this
only rewrites .issuegraph/metadata.json (dolt_mode) and writes the proxied-server
sidecar — no Dolt data is copied or moved. Stop the running server first with
'issuegraph dolt stop'.

Note: dolt_mode lives in the committed metadata.json, so this change propagates
to clones on the next push.`,
	Args: cobra.NoArgs,
	RunE: migrateToProxiedRunE("migrate-to-proxied-server", "migrate from-server-to-proxied-server", false),
}

var migrateSharedToProxiedServerCmd = &cobra.Command{
	Use:           "from-shared-server-to-proxied-server",
	Short:         "[EXPERIMENTAL] Switch a shared-server repo to proxied-server mode",
	SilenceUsage:  true,
	SilenceErrors: true,
	Long: `Switch a repo from shared-server mode to proxied-server mode.

The proxied server is rooted at the shared dolt directory
(~/.issuegraph/shared-server/dolt), so no Dolt data is copied or moved; this rewrites
.issuegraph/metadata.json (dolt_mode), turns off dolt.shared-server for this repo, and
writes the proxied-server sidecar. Stop the running shared server first with
'issuegraph dolt stop' — note that stops it for every project sharing it.`,
	Args: cobra.NoArgs,
	RunE: migrateToProxiedRunE("migrate-shared-to-proxied-server", "migrate from-shared-server-to-proxied-server", true),
}

var migrateToServerCmd = &cobra.Command{
	Use:           "from-proxied-server-to-server",
	Short:         "[EXPERIMENTAL] Switch a proxied-server repo to server mode",
	SilenceUsage:  true,
	SilenceErrors: true,
	Long: `Switch a repo from proxied-server mode to server mode (issuegraph init --server).

Both modes root their dolt sql-server at the same .issuegraph/dolt directory, so this
only rewrites .issuegraph/metadata.json (dolt_mode) and removes the proxied-server
sidecar — no Dolt data is copied or moved. Stop the running proxy first with
'issuegraph dolt stop'.

Note: dolt_mode lives in the committed metadata.json, so this change propagates
to clones on the next push.`,
	Args: cobra.NoArgs,
	RunE: migrateFromProxiedRunE("migrate-to-server", "migrate from-proxied-server-to-server", false),
}

var migrateToSharedServerCmd = &cobra.Command{
	Use:           "from-proxied-server-to-shared-server",
	Short:         "[EXPERIMENTAL] Switch a proxied-server repo back to shared-server mode",
	SilenceUsage:  true,
	SilenceErrors: true,
	Long: `Switch a repo from proxied-server mode back to shared-server mode.

Only applies to a proxied-server repo rooted at the shared dolt directory
(~/.issuegraph/shared-server/dolt) — the reverse of from-shared-server-to-proxied-server.
This rewrites .issuegraph/metadata.json (dolt_mode), re-enables dolt.shared-server, and
removes the proxied-server sidecar; no Dolt data is copied or moved. Stop the
running proxy first with 'issuegraph dolt stop'.`,
	Args: cobra.NoArgs,
	RunE: migrateFromProxiedRunE("migrate-to-shared-server", "migrate from-proxied-server-to-shared-server", true),
}

func resolveMigrateIdleTimeout(cmd *cobra.Command) (time.Duration, error) {
	if !cmd.Flags().Changed("idle-timeout") {
		return 0, nil
	}
	v, _ := cmd.Flags().GetDuration("idle-timeout")
	if v < 0 {
		return 0, HandleError("--idle-timeout must be 0 (never) or a positive duration, got %s", v)
	}
	if v == 0 {
		return proxy.IdleTimeoutNever, nil
	}
	return v, nil
}

// formatMigrateIdleTimeout renders a resolved idle timeout for operator-facing
// messages. resolveMigrateIdleTimeout maps an omitted flag to 0 and an explicit
// "--idle-timeout 0" to IdleTimeoutNever, so both sentinels need names.
func formatMigrateIdleTimeout(d time.Duration) string {
	switch d {
	case proxy.IdleTimeoutNever:
		return "never"
	case 0:
		return "default"
	default:
		return d.String()
	}
}

func migrateModeIssueGraphDir() (string, error) {
	issuegraphDir := issuegraph.FindIssueGraphDir()
	if issuegraphDir == "" {
		return "", HandleErrorWithHint(activeWorkspaceNotFoundError(), diagHint())
	}
	return issuegraphDir, nil
}

func loadMigrateModeConfig(issuegraphDir string) (*configfile.Config, error) {
	cfg, err := configfile.Load(issuegraphDir)
	if err != nil {
		return nil, HandleProxyCapabilityError(&ProxyCapabilityError{Code: "proxy.migrate.invalid_state", Message: fmt.Sprintf("failed to load config: %v", err), ExitCode: 1, Mutates: false})
	}
	if cfg == nil {
		return nil, HandleProxyCapabilityError(&ProxyCapabilityError{Code: "proxy.migrate.invalid_state", Message: fmt.Sprintf("no issuegraph database found in %s — run 'issuegraph init' first", issuegraphDir), ExitCode: 1, Mutates: false})
	}
	return cfg, nil
}

// migrateGateDestRoot resolves the physical root the DESTINATION mode of a
// dolt-mode migration will use, so the migration can gate it alongside the
// source's roots. Both proxied-server migration directions keep the data
// directory in place, so this is the shared dolt dir for --shared flows and
// the per-project dolt data dir otherwise — resolved side-effect-free (no
// mkdir) because gate planning must not create the tree it is guarding.
func migrateGateDestRoot(shared bool, issuegraphDir string) (string, error) {
	if shared {
		return doltserver.SharedDoltPath()
	}
	return doltserver.DoltDirPath(issuegraphDir), nil
}

// acquireMigrateGates takes the workspace gate plus the physical-root gates
// for BOTH the source mode's roots (via ResolvePhysicalRoots inside
// acquireExclusiveWorkspaceGates) and the destination mode's root, all
// EXCLUSIVE in one AcquireAll. It must run BEFORE acquireMigrateLock:
// the normative lock ordering is workspace gate(s) → physical-root gate(s)
// → migrate.lock → embedded .lock → proxy locks → dolt-server.lock.
func acquireMigrateGates(issuegraphDir string, shared bool, reason string) (func(), error) {
	destRoot, err := migrateGateDestRoot(shared, issuegraphDir)
	if err != nil {
		return nil, HandleError("failed to resolve migration destination root: %v", err)
	}
	// getRootContext(), not the rootCtx global: it honors the per-command
	// context when globals are disabled, and normalizes the not-yet-set case
	// to context.Background().
	h, err := acquireExclusiveWorkspaceGates(getRootContext(), issuegraphDir, reason, destRoot)
	if err != nil {
		return nil, HandleErrorWithHint(
			fmt.Sprintf("cannot migrate while other issuegraph activity holds this workspace: %v", err),
			"wait for running issuegraph commands to finish, then retry")
	}
	return func() { _ = h.Release() }, nil
}

// acquireMigrateLock takes the legacy per-workspace migrate.lock. Known
// hazard, deliberately left as-is: this lock file lives INSIDE .issuegraph and is
// removed on release, which is exactly the split-inode pattern the
// workspacegate package documents as unsafe for operations that replace
// directories. The workspace/physical-root gates acquired above (see
// acquireMigrateGates) are the durable fence; replacing migrate.lock itself
// is out of scope here (PR-B2+).
func acquireMigrateLock(issuegraphDir string) (func(), error) {
	lockPath := filepath.Join(issuegraphDir, migrateLockFileName)
	lock, err := util.TryLock(lockPath)
	if err != nil {
		if lockfile.IsLocked(err) {
			return nil, HandleErrorWithHint("another issuegraph migrate is in progress on this workspace", "wait for it to finish, then retry")
		}
		return nil, HandleError("failed to acquire migration lock: %v", err)
	}
	return func() {
		lock.Unlock()
		_ = os.Remove(lockPath)
	}, nil
}

func migrateLockErr(what string, err error) error {
	if lockfile.IsLocked(err) {
		return HandleErrorWithHint(fmt.Sprintf("%s is still running", what), "stop it first: issuegraph dolt stop")
	}
	return HandleError("failed to acquire %s lock: %v", what, err)
}

func runMigrateToProxiedServer(dryRun bool, idleTimeout time.Duration, shared bool) error {
	issuegraphDir, err := migrateModeIssueGraphDir()
	if err != nil {
		return err
	}
	if !dryRun {
		releaseGates, err := acquireMigrateGates(issuegraphDir, shared, "issuegraph migrate to proxied-server")
		if err != nil {
			return err
		}
		defer releaseGates()
		releaseMigrateLock, err := acquireMigrateLock(issuegraphDir)
		if err != nil {
			return err
		}
		defer releaseMigrateLock()
	}
	cfg, err := loadMigrateModeConfig(issuegraphDir)
	if err != nil {
		return err
	}
	j, jerr := loadMigrateJournal(issuegraphDir)
	if jerr != nil {
		return HandleProxyCapabilityError(&ProxyCapabilityError{Code: "proxy.migrate.invalid_state", Message: fmt.Sprintf("migration state is unreadable: %v", jerr), ExitCode: 1, Mutates: false})
	}
	if err := validateMigrationJournalAgainstConfig(j, cfg); err != nil {
		return HandleError("migration state is inconsistent: %v", err)
	}
	liveShared, yamlShared, yamlSharedSet, err := migrationSharedTopology(issuegraphDir)
	if err != nil {
		return HandleProxyCapabilityError(&ProxyCapabilityError{Code: "proxy.migrate.invalid_state", Message: err.Error(), ExitCode: 1, Mutates: false})
	}
	if j != nil {
		wantShared := shared && j.Phase == migratePrepared
		if !shared && yamlSharedSet && yamlShared {
			return HandleError("migration journal topology disagrees with persisted shared-server YAML (true)")
		}
		if liveShared != wantShared {
			if yamlSharedSet && !yamlShared && !wantShared {
				// The environment may still advertise shared mode after the
				// forward transition persisted YAML=false; trust persisted state.
				liveShared = wantShared
			} else {
				return HandleError("migration journal topology does not match live shared-server configuration")
			}
		}
	} else if shared && !liveShared && !cfg.IsDoltProxiedServerMode() {
		return HandleError("migration command topology does not match live shared-server configuration")
	}
	if j != nil && j.Sidecar != nil && j.Sidecar.External != nil {
		return externalMigrationRefusal("cannot resume migration for externally hosted proxied Dolt endpoint")
	}
	if cfg.IsDoltProxiedServerMode() && j == nil {
		info, ierr := configfile.LoadProxiedServerClientInfo(issuegraphDir)
		if ierr != nil {
			return migrationInvalidState(fmt.Sprintf("migration sidecar is unreadable: %v", ierr))
		}
		if info != nil {
			canonicalRoot := doltserver.DoltDirPath(issuegraphDir)
			if shared {
				canonicalRoot, ierr = doltserver.SharedDoltPath()
				if ierr != nil {
					return migrationInvalidState(fmt.Sprintf("failed to resolve shared migration root: %v", ierr))
				}
			}
			if info.External != nil {
				return migrationInvalidState("completed migration has an externally hosted proxied Dolt endpoint")
			}
			if !sameMigrationRoot(info.ResolvedRootPath(issuegraphDir), canonicalRoot) {
				return migrationInvalidState("proxied sidecar does not match canonical migration target")
			}
			if !shared && yamlSharedSet && yamlShared {
				return migrationInvalidState("completed migration topology is shared-server; use the shared migration command")
			}
			if err := validateCompletedProxyControls(canonicalRoot); err != nil {
				return err
			}
			stateDir := issuegraphDir
			if shared {
				stateDir, ierr = doltserver.SharedServerDir()
				if ierr != nil {
					return migrationInvalidState(fmt.Sprintf("failed to resolve shared server state: %v", ierr))
				}
			}
			if err := validateCompletedLockMarker(doltserver.LockPath(stateDir), "dolt-server lock"); err != nil {
				return err
			}
			fmt.Printf("%s\n", ui.RenderPass("✓ Already in proxied-server mode"))
			return nil
		}
		return migrationInvalidState(fmt.Sprintf("repo is marked proxied-server but %s is missing; rerun migration with a recoverable journal", configfile.ProxiedServerClientInfoFileName))
	}

	var rootPath string
	if j != nil {
		rootPath = j.RootPath
	} else if shared {
		if !doltserver.IsSharedServerMode() {
			return HandleError("repo is not in shared-server mode; this command only migrates shared-server repos")
		}
		rootPath, err = doltserver.SharedDoltPath()
		if err != nil {
			return HandleError("failed to resolve shared dolt directory: %v", err)
		}
	} else {
		rootPath = doltserver.DoltDirPath(issuegraphDir)
		if !cfg.IsDoltServerMode() {
			return HandleError("repo is not in server mode (dolt_mode=%q); this command only migrates server-mode repos", cfg.GetDoltMode())
		}
		// This migration only re-points the proxy at the LOCAL Dolt root;
		// it does not copy data. A workspace whose server mode comes from a
		// remote host (GH#3545 inference or explicit config) would silently
		// switch from the remote data to an empty/stale local database.
		if host := cfg.GetDoltServerHost(); !configfile.IsLocalHostString(host) {
			return HandleError("the configured Dolt server host is remote (%s); this migration only re-points the proxy at the local Dolt root and would abandon the remote data.\nMigrate on the server host itself, or clear dolt_server_host / dolt.host / ISSUEGRAPH_DOLT_SERVER_HOST first", host)
		}
		if doltserver.IsSharedServerMode() {
			return HandleErrorWithHint("repo is in shared-server mode", "use 'issuegraph migrate from-shared-server-to-proxied-server'")
		}
	}
	if j == nil {
		// A sidecar without its migration journal is ambiguous state. Refuse
		// to overwrite it: an externally-owned endpoint cannot be safely
		// converted, and a mismatched local root may point at another repo.
		info, ierr := configfile.LoadProxiedServerClientInfo(issuegraphDir)
		if ierr != nil {
			return HandleError("migration sidecar is unreadable: %v", ierr)
		}
		if info != nil {
			if info.External != nil {
				return externalMigrationRefusal("cannot migrate an externally hosted proxied Dolt endpoint; reconfigure the endpoint on its owner first")
			}
			resolvedSidecarRoot := info.ResolvedRootPath(issuegraphDir)
			if resolvedSidecarRoot != "" && !sameMigrationRoot(resolvedSidecarRoot, rootPath) {
				return HandleError("migration sidecar root %s does not match canonical %s", info.RootPath, rootPath)
			}
			return HandleError("migration sidecar exists without a recoverable journal; remove it only after verifying ownership")
		}
	}

	serverDir := doltserver.ResolveServerDir(issuegraphDir)
	if j != nil {
		if err := validateMigrationTopology(issuegraphDir, j, shared, true); err != nil {
			return HandleError("migration topology is inconsistent: %v", err)
		}
	}
	if state, _ := doltserver.IsRunning(serverDir); state != nil && state.Running {
		return HandleErrorWithHint("dolt server is still running", "stop it first: issuegraph dolt stop")
	}

	if dryRun {
		if jsonOutput {
			b, _ := json.Marshal(migrateDryRunOutput{true, configfile.DoltModeServer, configfile.DoltModeProxiedServer, shared, rootPath})
			fmt.Println(string(b))
			return nil
		}
		fmt.Println("Dry run mode - no changes will be made")
		fmt.Printf("Would set dolt_mode: %s → %s\n", configfile.DoltModeServer, configfile.DoltModeProxiedServer)
		if shared {
			fmt.Println("Would disable dolt.shared-server")
			fmt.Printf("Would root the proxy at %s\n", rootPath)
		}
		fmt.Printf("Would write %s\n", configfile.ProxiedServerClientInfoFileName)
		for _, p := range doltserver.StateFilePaths(serverDir) {
			fmt.Printf("Would remove %s\n", p)
		}
		return nil
	}
	if st, e := os.Stat(rootPath); e != nil || !st.IsDir() {
		return HandleError("Dolt root %s does not exist or is not a directory", rootPath)
	}
	if st, e := os.Stat(filepath.Join(rootPath, ".dolt")); e != nil || !st.IsDir() {
		return HandleError("Dolt root %s is not a valid Dolt directory", rootPath)
	}
	if st, e := os.Stat(filepath.Join(rootPath, ".dolt", "repo_state.json")); e != nil || st.IsDir() {
		return HandleError("Dolt root %s is not a valid Dolt repository", rootPath)
	}
	if err := validateDoltRoot(rootPath); err != nil {
		return HandleError("Dolt root validation failed: %v", err)
	}

	if j == nil {
		sidecar := &configfile.ProxiedServerClientInfo{RootPath: rootPath, IdleTimeout: idleTimeout}
		j = &migrateJournal{Version: 1, SourceMode: cfg.GetDoltMode(), TargetMode: configfile.DoltModeProxiedServer, Shared: shared, RootPath: rootPath, Sidecar: sidecar, Ownership: "managed-local", Attempt: 1, Phase: migratePrepared}
		if err := saveMigrateJournal(issuegraphDir, j); err != nil {
			return HandleError("failed to prepare migration: %v", err)
		}
		if err := migrateFault(migratePrepared); err != nil {
			return err
		}
	} else if j.TargetMode != configfile.DoltModeProxiedServer || j.Shared != shared {
		return HandleError("migration journal belongs to a different migration")
	} else {
		j.Attempt++
		if err := saveMigrateJournal(issuegraphDir, j); err != nil {
			return HandleError("failed to update migration attempt: %v", err)
		}
	}

	if j.Phase == migratePrepared {
		cfg.DoltMode = configfile.DoltModeProxiedServer
		if err := cfg.Save(issuegraphDir); err != nil {
			return HandleError("failed to save metadata.json: %v", err)
		}
		if shared {
			if err := config.SetYamlConfigInDir(issuegraphDir, "dolt.shared-server", "false"); err != nil {
				return HandleError("failed to disable dolt.shared-server: %v", err)
			}
		}
		// Replay the recorded plan instead of rebuilding it from the current
		// argv. A resume that omits or changes --idle-timeout would otherwise
		// write a sidecar the target_configured check below then rejects, and
		// because that check runs only after the phase has advanced past
		// migratePrepared, this block never runs again — the workspace would be
		// wedged in proxied-server mode with no in-tool recovery.
		info := j.Sidecar
		if info == nil {
			info = &configfile.ProxiedServerClientInfo{RootPath: rootPath, IdleTimeout: idleTimeout}
		} else if info.IdleTimeout != idleTimeout {
			fmt.Fprintf(os.Stderr, "Warning: resuming migration with the recorded --idle-timeout (%s); the value supplied now (%s) is ignored\n", formatMigrateIdleTimeout(info.IdleTimeout), formatMigrateIdleTimeout(idleTimeout))
		}
		if err := configfile.SaveProxiedServerClientInfo(issuegraphDir, info); err != nil {
			return HandleError("failed to write %s: %v", configfile.ProxiedServerClientInfoFileName, err)
		}
		if err := checkpointMigration(issuegraphDir, j, migrateTargetConfigured); err != nil {
			return err
		}
	}
	if j.Phase == migrateTargetConfigured {
		if info, e := configfile.LoadProxiedServerClientInfo(issuegraphDir); e != nil {
			return HandleError("migration sidecar is unreadable: %v", e)
		} else if info == nil && j.Sidecar != nil {
			if e := configfile.SaveProxiedServerClientInfo(issuegraphDir, j.Sidecar); e != nil {
				return HandleError("failed to repair %s: %v", configfile.ProxiedServerClientInfoFileName, e)
			}
		} else if j.Sidecar != nil && !reflect.DeepEqual(info, j.Sidecar) {
			return HandleError("migration sidecar does not match prepared state")
		}
		if errs := doltserver.RemoveStateFiles(serverDir); len(errs) > 0 {
			return HandleError("failed to retire Dolt server controls: %v", errs[0])
		}
		if err := checkpointMigration(issuegraphDir, j, migrateOldControlsRetired); err != nil {
			return err
		}
	}
	if j.Phase == migrateOldControlsRetired {
		if c, e := configfile.Load(issuegraphDir); e != nil || c == nil || !c.IsDoltProxiedServerMode() {
			return HandleError("migration verification failed: metadata mode is not proxied-server")
		}
		if info, e := configfile.LoadProxiedServerClientInfo(issuegraphDir); e != nil || info == nil {
			return HandleError("migration verification failed: proxied sidecar is missing or unreadable")
		}
		if err := checkpointMigration(issuegraphDir, j, migrateVerified); err != nil {
			return err
		}
	}
	if j.Phase == migrateVerified {
		if err := checkpointMigration(issuegraphDir, j, migrateCommitted); err != nil {
			return err
		}
		if err := removeMigrateJournal(issuegraphDir); err != nil {
			return HandleError("failed to finalize migration: %v", err)
		}
	}
	if j.Phase == migrateCommitted {
		if err := removeMigrateJournal(issuegraphDir); err != nil {
			return HandleError("failed to finalize migration: %v", err)
		}
	}

	dataDir := proxiedServerRoot(issuegraphDir)
	if shared {
		dataDir = rootPath
	}
	commandDidWrite.Store(true)
	fmt.Printf("%s\n\n", ui.RenderPass("✓ Switched to proxied-server mode"))
	fmt.Printf("  Data directory unchanged: %s\n", dataDir)
	fmt.Println("  The proxy starts automatically on the next issuegraph command.")
	return nil
}

func runMigrateFromProxiedServer(dryRun bool, shared bool) error {
	issuegraphDir, err := migrateModeIssueGraphDir()
	if err != nil {
		return err
	}
	if !dryRun {
		releaseGates, err := acquireMigrateGates(issuegraphDir, shared, "issuegraph migrate from proxied-server")
		if err != nil {
			return err
		}
		defer releaseGates()
		releaseMigrateLock, err := acquireMigrateLock(issuegraphDir)
		if err != nil {
			return err
		}
		defer releaseMigrateLock()
	}
	cfg, err := loadMigrateModeConfig(issuegraphDir)
	if err != nil {
		return err
	}
	j, jerr := loadMigrateJournal(issuegraphDir)
	if jerr != nil {
		return HandleProxyCapabilityError(&ProxyCapabilityError{Code: "proxy.migrate.invalid_state", Message: fmt.Sprintf("migration state is unreadable: %v", jerr), ExitCode: 1, Mutates: false})
	}
	if err := validateMigrationJournalAgainstConfig(j, cfg); err != nil {
		return HandleError("migration state is inconsistent: %v", err)
	}
	liveShared, yamlShared, yamlSharedSet, err := migrationSharedTopology(issuegraphDir)
	if err != nil {
		return HandleProxyCapabilityError(&ProxyCapabilityError{Code: "proxy.migrate.invalid_state", Message: err.Error(), ExitCode: 1, Mutates: false})
	}
	var diskSidecar *configfile.ProxiedServerClientInfo
	if diskSidecar, err = configfile.LoadProxiedServerClientInfo(issuegraphDir); err != nil {
		return HandleProxyCapabilityError(&ProxyCapabilityError{Code: "proxy.migrate.invalid_state", Message: fmt.Sprintf("migration sidecar is unreadable: %v", err), ExitCode: 1, Mutates: false})
	}
	if j != nil {
		wantShared := shared && j.Phase != migratePrepared
		if liveShared != wantShared {
			return HandleError("migration journal topology does not match live shared-server configuration")
		}
	}
	if j != nil && j.Sidecar != nil && j.Sidecar.External != nil {
		return externalMigrationRefusal("cannot resume migration for externally hosted proxied Dolt endpoint")
	}
	if j != nil {
		// SCOPED TO REVERSE JOURNALS, and the scope is load-bearing. At
		// migratePrepared the two directions disagree about whether the
		// sidecar is on disk: a reverse journal records a sidecar that
		// already exists, while a forward journal records the plan it is
		// about to write and only advances past migratePrepared after the
		// write. So a crash in a forward migration's pre-write window leaves
		// diskSidecar nil against a non-nil j.Sidecar, and an unscoped guard
		// refuses this command — the documented escape hatch — by naming a
		// file whose absence is the correct state for that phase. Forward
		// journals fall through to the journal-ownership check below, which
		// reports the real situation. Testing TargetMode against the forward
		// value rather than for the reverse one keeps the guard live for
		// BOTH reverse targets, "server" and "shared-server".
		if j.Phase == migratePrepared && j.TargetMode != configfile.DoltModeProxiedServer &&
			!migrationSidecarsEquivalent(issuegraphDir, diskSidecar, j.Sidecar) {
			return HandleError("migration sidecar does not match prepared state")
		}
		if j.Phase == migrateTargetConfigured && diskSidecar != nil && !migrationSidecarsEquivalent(issuegraphDir, diskSidecar, j.Sidecar) {
			return HandleError("migration sidecar does not match prepared state")
		}
		if (j.Phase == migrateOldControlsRetired || j.Phase == migrateVerified || j.Phase == migrateCommitted) && diskSidecar != nil {
			return HandleError("migration sidecar must be absent after controls are retired")
		}
	}
	if j == nil {
		alreadyTarget := cfg.IsDoltServerMode() && ((shared && yamlSharedSet && yamlShared) || (!shared && (!yamlSharedSet || !yamlShared)))
		if alreadyTarget {
			if diskSidecar != nil {
				return HandleProxyCapabilityError(&ProxyCapabilityError{Code: "proxy.migrate.invalid_state", Message: "completed migration has a stale proxied-server sidecar", ExitCode: 1, Mutates: false})
			}
			root := doltserver.DoltDirPath(issuegraphDir)
			if shared {
				if sharedRoot, sharedErr := doltserver.SharedDoltPath(); sharedErr == nil {
					root = sharedRoot
				}
			}
			if running, _ := proxy.IsRunning(root); running {
				return HandleError("completed migration still has a running proxy")
			}
			for _, path := range proxy.ControlFilePaths(root) {
				_, statErr := os.Stat(path)
				if os.IsNotExist(statErr) {
					continue
				}
				if statErr != nil {
					return HandleProxyCapabilityError(&ProxyCapabilityError{Code: "proxy.migrate.invalid_state", Message: fmt.Sprintf("completed migration cannot inspect proxy control %s: %v", path, statErr), ExitCode: 1, Mutates: false})
				}
				if filepath.Base(path) == proxy.LockFileName || filepath.Base(path) == server.LockFileName {
					// Migration lock files are created as part of acquiring the
					// lifecycle gate and may remain as empty, unlocked markers.
					// Probe only an existing file; TryLock must not create state
					// during an idempotent no-op.
					lock, lockErr := util.TryLock(path)
					if lockErr == nil {
						lock.Unlock()
						continue
					}
				}
				if statErr == nil {
					return HandleProxyCapabilityError(&ProxyCapabilityError{Code: "proxy.migrate.invalid_state", Message: fmt.Sprintf("completed migration has stale proxy control %s", path), ExitCode: 1, Mutates: false})
				}
			}
			return nil
		}
		if liveShared {
			return HandleError("migration command topology does not match live shared-server configuration")
		}
	}
	var sourceSidecar *configfile.ProxiedServerClientInfo
	if j == nil {
		sourceSidecar = diskSidecar
		if sourceSidecar == nil {
			return HandleError("migration sidecar is missing")
		}
		if sourceSidecar.External != nil {
			return externalMigrationRefusal("cannot migrate an externally hosted proxied Dolt endpoint to local server mode; reconfigure the endpoint on its owner first")
		}
	}
	if j == nil {
		if shared {
			// Persisted workspace YAML is authoritative over ambient process
			// environment. A completed forward migration deliberately leaves
			// ISSUEGRAPH_DOLT_SHARED_SERVER stale in some shells while writing
			// dolt.shared-server: false; do not misclassify that proxied repo as
			// already migrated to shared-server mode.
			if !cfg.IsDoltProxiedServerMode() && ((!yamlSharedSet && doltserver.IsSharedServerMode()) || (yamlSharedSet && yamlShared)) {
				fmt.Printf("%s\n", ui.RenderPass("✓ Already in shared-server mode"))
				return nil
			}
		} else if cfg.IsDoltServerMode() && !doltserver.IsSharedServerMode() {
			fmt.Printf("%s\n", ui.RenderPass("✓ Already in server mode"))
			return nil
		}
	}
	if j == nil && !cfg.IsDoltProxiedServerMode() {
		return HandleError("repo is not in proxied-server mode (dolt_mode=%q); this command only migrates proxied-server repos", cfg.GetDoltMode())
	}
	expectedTarget := configfile.DoltModeServer
	if shared {
		expectedTarget = "shared-server"
	}
	if j != nil && (j.TargetMode != expectedTarget || j.Shared != shared) {
		return HandleError("migration journal belongs to a different migration")
	}
	// Leaving proxied mode would make dolt_team_server inert and resume
	// bd-driven schema migrations on the bts-owned database.
	if cfg.DoltTeamServer {
		return HandleErrorWithHint("workspace is team-server managed (dolt_team_server in metadata.json); the shared database's schema is owned by issuegraph-team-server",
			"if the database is no longer bts-managed, remove dolt_team_server from .issuegraph/metadata.json first")
	}

	rootDir := ""
	if j != nil {
		rootDir = j.RootPath
	} else {
		rootDir, err = resolveProxiedServerRootPath(issuegraphDir)
		if err != nil {
			return HandleError("%v", err)
		}
	}
	if sourceSidecar != nil && sourceSidecar.RootPath == "" {
		sourceSidecar.RootPath = rootDir
	}
	if j != nil {
		if err := validateMigrationTopology(issuegraphDir, j, shared, false); err != nil {
			return HandleError("migration topology is inconsistent: %v", err)
		}
	}

	sharedDolt, sharedErr := doltserver.SharedDoltPath()
	if shared {
		if sharedErr != nil {
			return HandleError("failed to resolve shared dolt directory: %v", sharedErr)
		}
		if rootDir != sharedDolt {
			return HandleErrorWithHint(fmt.Sprintf("proxied-server root %s is not the shared dolt directory", rootDir), "use 'issuegraph migrate from-proxied-server-to-server'")
		}
	} else if sharedErr == nil && rootDir == sharedDolt {
		return HandleErrorWithHint("proxied-server is rooted at the shared dolt directory", "use 'issuegraph migrate from-proxied-server-to-shared-server'")
	}

	if running, _ := proxy.IsRunning(rootDir); running {
		return HandleErrorWithHint("proxied-server is still running", "stop it first: issuegraph dolt stop")
	}

	var logAssets []string
	if j != nil {
		logAssets = append(logAssets, j.LogAssets...)
	} else {
		logAssets, err = proxiedLogAssets(issuegraphDir)
		if err != nil {
			return HandleError("%v", err)
		}
	}

	if dryRun {
		if jsonOutput {
			b, _ := json.Marshal(migrateDryRunOutput{true, configfile.DoltModeProxiedServer, expectedTarget, shared, rootDir})
			fmt.Println(string(b))
			return nil
		}
		fmt.Println("Dry run mode - no changes will be made")
		fmt.Printf("Would set dolt_mode: %s → %s\n", configfile.DoltModeProxiedServer, configfile.DoltModeServer)
		if shared {
			fmt.Println("Would enable dolt.shared-server")
		}
		fmt.Printf("Would remove %s\n", configfile.ProxiedServerClientInfoFileName)
		for _, p := range proxy.ControlFilePaths(rootDir) {
			fmt.Printf("Would remove %s\n", p)
		}
		for _, p := range logAssets {
			fmt.Printf("Would remove %s\n", p)
		}
		return nil
	}
	if st, e := os.Stat(rootDir); e != nil || !st.IsDir() {
		return HandleError("Dolt root %s does not exist or is not a directory", rootDir)
	}
	if st, e := os.Stat(filepath.Join(rootDir, ".dolt")); e != nil || !st.IsDir() {
		return HandleError("Dolt root %s is not a valid Dolt directory", rootDir)
	}
	if st, e := os.Stat(filepath.Join(rootDir, ".dolt", "repo_state.json")); e != nil || st.IsDir() {
		return HandleError("Dolt root %s is not a valid Dolt repository", rootDir)
	}
	if err := validateDoltRoot(rootDir); err != nil {
		return HandleError("Dolt root validation failed: %v", err)
	}

	serverStateDir := issuegraphDir
	if shared {
		serverStateDir, err = doltserver.SharedServerDir()
		if err != nil {
			return HandleError("failed to resolve shared server directory: %v", err)
		}
	}

	proxyLock, err := util.TryLock(filepath.Join(rootDir, proxy.LockFileName))
	if err != nil {
		return migrateLockErr("proxy", err)
	}
	defer proxyLock.Unlock()

	childLock, err := util.TryLock(filepath.Join(rootDir, server.LockFileName))
	if err != nil {
		return migrateLockErr("proxied dolt sql-server", err)
	}
	defer childLock.Unlock()

	serverLock, err := util.TryLock(doltserver.LockPath(serverStateDir))
	if err != nil {
		return migrateLockErr("dolt sql-server", err)
	}
	defer serverLock.Unlock()
	if j == nil {
		j = &migrateJournal{Version: 1, SourceMode: configfile.DoltModeProxiedServer, TargetMode: expectedTarget, Shared: shared, RootPath: rootDir, Sidecar: sourceSidecar, LogAssets: logAssets, Ownership: "managed-local", Attempt: 1, Phase: migratePrepared}
		if err := saveMigrateJournal(issuegraphDir, j); err != nil {
			return HandleError("failed to prepare migration: %v", err)
		}
		if err := migrateFault(migratePrepared); err != nil {
			return err
		}
	} else {
		j.Attempt++
		if err := saveMigrateJournal(issuegraphDir, j); err != nil {
			return HandleError("failed to update migration attempt: %v", err)
		}
	}
	if j.Phase == migratePrepared {
		if err := doltserver.MarkDoltDirCompatible(rootDir); err != nil {
			return HandleError("failed to mark dolt directory compatible: %v", err)
		}
		cfg.DoltMode = configfile.DoltModeServer
		if err := cfg.Save(issuegraphDir); err != nil {
			return HandleError("failed to save metadata.json: %v", err)
		}
		if shared {
			if err := config.SetYamlConfigInDir(issuegraphDir, "dolt.shared-server", "true"); err != nil {
				return HandleError("failed to enable dolt.shared-server: %v", err)
			}
		}
		if err := checkpointMigration(issuegraphDir, j, migrateTargetConfigured); err != nil {
			return err
		}
	}
	if j.Phase == migrateTargetConfigured {
		if err := os.Remove(configfile.ProxiedServerClientInfoPath(issuegraphDir)); err != nil && !os.IsNotExist(err) {
			return HandleError("failed to remove %s: %v", configfile.ProxiedServerClientInfoFileName, err)
		}
		if errs := proxy.PurgeControlFiles(rootDir); len(errs) > 0 {
			return HandleError("failed to retire proxy controls: %v", errs[0])
		}
		if errs := removeMigrateAssets(logAssets); len(errs) > 0 {
			return HandleError("failed to retire proxy logs: %v", errs[0])
		}
		if err := checkpointMigration(issuegraphDir, j, migrateOldControlsRetired); err != nil {
			return err
		}
	}
	if j.Phase == migrateOldControlsRetired {
		if c, e := configfile.Load(issuegraphDir); e != nil || c == nil || !c.IsDoltServerMode() {
			return HandleError("migration verification failed: metadata mode is not server")
		}
		if shared {
			if v, ok := config.WorkspaceYamlValue(issuegraphDir, "dolt.shared-server"); !ok || strings.ToLower(v) != "true" {
				return HandleError("migration verification failed: shared-server is not enabled")
			}
		}
		if err := checkpointMigration(issuegraphDir, j, migrateVerified); err != nil {
			return err
		}
	}
	if j.Phase == migrateVerified {
		if err := checkpointMigration(issuegraphDir, j, migrateCommitted); err != nil {
			return err
		}
		if err := removeMigrateJournal(issuegraphDir); err != nil {
			return HandleError("failed to finalize migration: %v", err)
		}
	}
	if j.Phase == migrateCommitted {
		if err := removeMigrateJournal(issuegraphDir); err != nil {
			return HandleError("failed to finalize migration: %v", err)
		}
	}

	commandDidWrite.Store(true)
	if shared {
		fmt.Printf("%s\n\n", ui.RenderPass("✓ Switched to shared-server mode"))
	} else {
		fmt.Printf("%s\n\n", ui.RenderPass("✓ Switched to server mode"))
	}
	fmt.Printf("  Data directory unchanged: %s\n", rootDir)
	fmt.Println("  The dolt sql-server starts automatically on the next issuegraph command.")
	return nil
}

// migrationSharedTopology resolves the target workspace's shared-server
// setting. An explicit workspace YAML value is authoritative over ambient
// process environment so retries cannot be misclassified by stale env state.
func migrationSharedTopology(issuegraphDir string) (live, yaml bool, yamlSet bool, err error) {
	live = doltserver.IsSharedServerMode()
	raw, yamlSet, readErr := config.WorkspaceYamlValueStrict(issuegraphDir, "dolt.shared-server")
	if readErr != nil {
		return false, false, false, readErr
	}
	if !yamlSet {
		return live, false, false, nil
	}
	parsed, parseErr := strconv.ParseBool(strings.TrimSpace(raw))
	if parseErr != nil {
		return false, false, true, fmt.Errorf("workspace dolt.shared-server is invalid: %q", raw)
	}
	return parsed, parsed, true, nil
}

func proxiedLogAssets(issuegraphDir string) ([]string, error) {
	logPath, isCustomLog, err := resolveProxiedServerLogPath(issuegraphDir)
	if err != nil {
		return nil, err
	}
	if isCustomLog {
		return nil, nil
	}
	return []string{logPath}, nil
}

func removeMigrateAssets(paths []string) []error {
	var errs []error
	for _, p := range paths {
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			errs = append(errs, err)
		}
	}
	return errs
}

func warnMigrateRemovalErrors(errs []error) {
	for _, err := range errs {
		fmt.Fprintf(os.Stderr, "Warning: could not remove migration asset: %v\n", err)
	}
}
