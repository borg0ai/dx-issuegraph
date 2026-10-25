package doctor

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/borg0ai/dx-issuegraph/internal/config"
	"github.com/borg0ai/dx-issuegraph/internal/configfile"
	"github.com/borg0ai/dx-issuegraph/internal/doltserver"
	"github.com/borg0ai/dx-issuegraph/internal/issuegraph"
)

// agentDocFiles returns the list of documentation files to check, including
// the configured agents file (which may differ from the default AGENTS.md).
func agentDocFiles(repoPath string) []string {
	agentsFile := config.SafeAgentsFile()
	files := []string{
		filepath.Join(repoPath, agentsFile),
		filepath.Join(repoPath, "CLAUDE.md"),
		filepath.Join(repoPath, ".github", "copilot-instructions.md"),
		filepath.Join(repoPath, ".claude", "CLAUDE.md"),
		// Local-only variants (not committed to repo)
		filepath.Join(repoPath, "claude.local.md"),
		filepath.Join(repoPath, ".claude", "claude.local.md"),
	}
	// If the configured file isn't the default, also check the default
	// to catch legacy files that haven't been migrated.
	if !strings.EqualFold(agentsFile, config.DefaultAgentsFile) {
		files = append(files, filepath.Join(repoPath, config.DefaultAgentsFile))
	}
	return files
}

// CheckLegacyIssueGraphSlashCommands detects old /issuegraph:* slash commands in documentation
// and recommends migration to issuegraph prime hooks for better token efficiency.
//
// Old pattern: /issuegraph:quickstart, /issuegraph:ready (~10.5k tokens per session)
// New pattern: issuegraph prime hooks (~50-2k tokens per session)
func CheckLegacyIssueGraphSlashCommands(repoPath string) DoctorCheck {
	docFiles := agentDocFiles(repoPath)

	var filesWithLegacyCommands []string
	legacyPattern := "/issuegraph:"

	for _, docFile := range docFiles {
		content, err := os.ReadFile(docFile) // #nosec G304 - controlled paths from repoPath
		if err != nil {
			continue // File doesn't exist or can't be read
		}

		if strings.Contains(string(content), legacyPattern) {
			filesWithLegacyCommands = append(filesWithLegacyCommands, filepath.Base(docFile))
		}
	}

	if len(filesWithLegacyCommands) == 0 {
		return DoctorCheck{
			Name:    "Legacy Commands",
			Status:  StatusOK,
			Message: "No legacy issuegraph slash commands detected",
		}
	}

	return DoctorCheck{
		Name:    "Legacy Commands",
		Status:  StatusWarning,
		Message: fmt.Sprintf("Old issuegraph integration detected in %s", strings.Join(filesWithLegacyCommands, ", ")),
		Detail: "Found: /issuegraph:* slash command references (deprecated)\n" +
			"  These commands are token-inefficient (~10.5k tokens per session)",
		Fix: "Migrate to issuegraph prime hooks for better token efficiency:\n" +
			"\n" +
			"Migration Steps:\n" +
			"  1. Run 'issuegraph setup claude' to add SessionStart hooks\n" +
			"  2. Update " + config.AgentsFile() + "/CLAUDE.md:\n" +
			"     - Remove /issuegraph:* slash command references\n" +
			"     - Add: \"Run 'issuegraph prime' for workflow context\" (for users without hooks)\n" +
			"\n" +
			"Benefits:\n" +
			"  • MCP mode: ~50 tokens vs ~10.5k for full MCP scan (99% reduction)\n" +
			"  • CLI mode: ~1-2k tokens with automatic context recovery\n" +
			"  • Hooks auto-refresh context on session start and before compaction\n" +
			"\n" +
			"See: issuegraph setup claude --help",
	}
}

// CheckLegacyMCPToolReferences detects direct MCP tool name references in documentation
// (e.g., mcp__issuegraph_issuegraph__list, mcp__plugin_issuegraph_issuegraph__show) and recommends
// migration to issuegraph prime hooks for better token efficiency.
//
// Old pattern: Document MCP tool names for direct tool calls (~10.5k tokens per scan)
// New pattern: issuegraph prime hooks with CLI commands (~50-2k tokens)
func CheckLegacyMCPToolReferences(repoPath string) DoctorCheck {
	docFiles := agentDocFiles(repoPath)

	mcpPatterns := []string{
		"mcp__issuegraph_issuegraph__",
		"mcp__plugin_issuegraph_issuegraph__",
		"mcp_issuegraph_",
	}

	var filesWithMCPRefs []string
	for _, docFile := range docFiles {
		content, err := os.ReadFile(docFile) // #nosec G304 - controlled paths from repoPath
		if err != nil {
			continue
		}

		contentStr := string(content)
		for _, pattern := range mcpPatterns {
			if strings.Contains(contentStr, pattern) {
				filesWithMCPRefs = append(filesWithMCPRefs, filepath.Base(docFile))
				break
			}
		}
	}

	if len(filesWithMCPRefs) == 0 {
		return DoctorCheck{
			Name:    "MCP Tool References",
			Status:  StatusOK,
			Message: "No MCP tool references in documentation",
		}
	}

	return DoctorCheck{
		Name:    "MCP Tool References",
		Status:  StatusWarning,
		Message: fmt.Sprintf("MCP tool references found in %s", strings.Join(filesWithMCPRefs, ", ")),
		Detail: "Found: Direct MCP tool name references (e.g., mcp__issuegraph_issuegraph__list)\n" +
			"  MCP tool calls consume ~10.5k tokens per session for tool scanning",
		Fix: "Migrate to issuegraph prime hooks for better token efficiency:\n" +
			"\n" +
			"Migration Steps:\n" +
			"  1. Run 'issuegraph setup claude' to add SessionStart hooks\n" +
			"  2. Replace MCP tool references with CLI commands:\n" +
			"     - mcp__issuegraph_issuegraph__list  → issuegraph list\n" +
			"     - mcp__issuegraph_issuegraph__show  → issuegraph show <id>\n" +
			"     - mcp__issuegraph_issuegraph__ready → issuegraph ready\n" +
			"  3. issuegraph prime hooks auto-inject context on session start\n" +
			"\n" +
			"Benefits:\n" +
			"  • issuegraph prime + hooks: ~50-2k tokens vs ~10.5k for MCP tool scan\n" +
			"  • Automatic context recovery on session start and compaction\n" +
			"\n" +
			"See: issuegraph setup claude --help",
	}
}

// CheckAgentDocumentation checks if agent documentation (AGENTS.md or CLAUDE.md) exists
// and recommends adding it if missing, suggesting issuegraph onboard or issuegraph setup claude.
// Also supports local-only variants (claude.local.md) that are gitignored.
func CheckAgentDocumentation(repoPath string) DoctorCheck {
	docFiles := agentDocFiles(repoPath)

	var foundDocs []string
	for _, docFile := range docFiles {
		if _, err := os.Stat(docFile); err == nil {
			foundDocs = append(foundDocs, filepath.Base(docFile))
		}
	}

	if len(foundDocs) > 0 {
		return DoctorCheck{
			Name:    "Agent Documentation",
			Status:  StatusOK,
			Message: fmt.Sprintf("Documentation found: %s", strings.Join(foundDocs, ", ")),
		}
	}

	return DoctorCheck{
		Name:    "Agent Documentation",
		Status:  StatusWarning,
		Message: "No agent documentation found",
		Detail: "Missing: " + config.AgentsFile() + " or CLAUDE.md\n" +
			"  Documenting workflow helps AI agents work more effectively",
		Fix: "Add agent documentation:\n" +
			"  • Run 'issuegraph onboard' to create " + config.AgentsFile() + " with workflow guidance\n" +
			"  • Or run 'issuegraph setup claude' to add Claude-specific documentation\n" +
			"\n" +
			"For local-only documentation (not committed to repo):\n" +
			"  • Create claude.local.md or .claude/claude.local.md\n" +
			"  • Add 'claude.local.md' to your .gitignore\n" +
			"\n" +
			"Recommended: Include issuegraph workflow in your project documentation so\n" +
			"AI agents understand how to track issues and manage dependencies",
	}
}

// stripIssueGraphIntegrationSection removes the <!-- BEGIN ISSUEGRAPH INTEGRATION ... -->
// ... <!-- END ISSUEGRAPH INTEGRATION --> block (plus an optional trailing newline)
// so the remaining content represents user-authored material. Returns the input
// unchanged if no well-formed marker pair is found.
func stripIssueGraphIntegrationSection(content string) string {
	const beginPrefix = "<!-- BEGIN ISSUEGRAPH INTEGRATION"
	const endMarker = "<!-- END ISSUEGRAPH INTEGRATION -->"

	beginIdx := strings.Index(content, beginPrefix)
	if beginIdx == -1 {
		return content
	}
	endIdx := strings.Index(content, endMarker)
	if endIdx == -1 || endIdx < beginIdx {
		return content
	}
	end := endIdx + len(endMarker)
	if end < len(content) && content[end] == '\n' {
		end++
	}
	return content[:beginIdx] + content[end:]
}

// normalizeUserAuthored canonicalizes content for divergence comparison:
// strips the managed issuegraph section, normalizes line endings, and collapses
// trailing whitespace. Leading/trailing blank lines are removed so cosmetic
// reformatting (e.g. an extra newline at EOF) does not flag as divergence.
func normalizeUserAuthored(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = stripIssueGraphIntegrationSection(content)
	lines := strings.Split(content, "\n")
	for i, ln := range lines {
		lines[i] = strings.TrimRight(ln, " \t")
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

// sameInode reports whether two files refer to the same underlying inode
// (hard-linked or one is a symlink resolving to the other). Falls back to
// comparing resolved paths on platforms where syscall stat is unavailable.
func sameInode(a, b string) (bool, error) {
	resolvedA, err := filepath.EvalSymlinks(a)
	if err != nil {
		return false, err
	}
	resolvedB, err := filepath.EvalSymlinks(b)
	if err != nil {
		return false, err
	}
	if resolvedA == resolvedB {
		return true, nil
	}
	infoA, err := os.Stat(resolvedA)
	if err != nil {
		return false, err
	}
	infoB, err := os.Stat(resolvedB)
	if err != nil {
		return false, err
	}
	return os.SameFile(infoA, infoB), nil
}

// CheckAgentDocDivergence detects when AGENTS.md and CLAUDE.md exist as
// independent regular files whose user-authored regions (everything outside
// the BEGIN/END ISSUEGRAPH INTEGRATION markers) have drifted apart. Hand-edits to
// only one of the pair are a common source of inconsistency; the warning
// recommends symlinking, regenerating via issuegraph setup, or reconciling manually.
//
// Skipped when:
//   - Either file is missing
//   - The files share an inode (hardlink) or one is a symlink to the other
//   - The user-authored content matches after normalization
func CheckAgentDocDivergence(repoPath string) DoctorCheck {
	agentsFile := config.SafeAgentsFile()
	agentsPath := filepath.Join(repoPath, agentsFile)
	claudePath := filepath.Join(repoPath, "CLAUDE.md")

	agentsInfo, errA := os.Lstat(agentsPath)
	claudeInfo, errB := os.Lstat(claudePath)
	if errA != nil || errB != nil {
		return DoctorCheck{
			Name:    "Agent Doc Divergence",
			Status:  StatusOK,
			Message: "N/A (one or both files missing)",
		}
	}

	// If either side is a symlink, treat the pair as intentionally linked
	// and skip the divergence check.
	if agentsInfo.Mode()&os.ModeSymlink != 0 || claudeInfo.Mode()&os.ModeSymlink != 0 {
		return DoctorCheck{
			Name:    "Agent Doc Divergence",
			Status:  StatusOK,
			Message: fmt.Sprintf("%s and CLAUDE.md are linked", agentsFile),
		}
	}

	// Hard-linked to the same inode — same file, no divergence possible.
	if same, err := sameInode(agentsPath, claudePath); err == nil && same {
		return DoctorCheck{
			Name:    "Agent Doc Divergence",
			Status:  StatusOK,
			Message: fmt.Sprintf("%s and CLAUDE.md share an inode", agentsFile),
		}
	}

	agentsContent, err := os.ReadFile(agentsPath) // #nosec G304 - path under repoPath
	if err != nil {
		return DoctorCheck{
			Name:    "Agent Doc Divergence",
			Status:  StatusOK,
			Message: fmt.Sprintf("Cannot read %s: %v", agentsFile, err),
		}
	}
	claudeContent, err := os.ReadFile(claudePath) // #nosec G304 - path under repoPath
	if err != nil {
		return DoctorCheck{
			Name:    "Agent Doc Divergence",
			Status:  StatusOK,
			Message: fmt.Sprintf("Cannot read CLAUDE.md: %v", err),
		}
	}

	// Honor an explicit opt-out marker in either file. Some projects
	// intentionally maintain distinct AGENTS.md and CLAUDE.md (different
	// audiences, different reading orders). The marker lets them silence
	// this check without losing the protection elsewhere.
	const optOutMarker = "<!-- bd-doctor-divergence: ok -->"
	if strings.Contains(string(agentsContent), optOutMarker) || strings.Contains(string(claudeContent), optOutMarker) {
		return DoctorCheck{
			Name:    "Agent Doc Divergence",
			Status:  StatusOK,
			Message: "Divergence check opted out via marker",
		}
	}

	if normalizeUserAuthored(string(agentsContent)) == normalizeUserAuthored(string(claudeContent)) {
		return DoctorCheck{
			Name:    "Agent Doc Divergence",
			Status:  StatusOK,
			Message: fmt.Sprintf("%s and CLAUDE.md user-authored content matches", agentsFile),
		}
	}

	return DoctorCheck{
		Name:    "Agent Doc Divergence",
		Status:  StatusWarning,
		Message: fmt.Sprintf("%s and CLAUDE.md user-authored content has diverged", agentsFile),
		Detail: "Both files exist as independent regular files (not symlinked, different inodes),\n" +
			"  but their content outside the <!-- BEGIN/END ISSUEGRAPH INTEGRATION --> markers differs.\n" +
			"  Hand-edits to only one of the pair are a common cause.",
		Fix: "Reconcile the two files using one of:\n" +
			"\n" +
			"  (a) Symlink one to the other so future edits stay in sync:\n" +
			"      ln -sf " + agentsFile + " CLAUDE.md\n" +
			"\n" +
			"  (b) Regenerate the managed sections (preserves user-authored content\n" +
			"      from AGENTS.md as the source of truth):\n" +
			"      issuegraph setup claude && issuegraph setup codex\n" +
			"\n" +
			"  (c) Reconcile manually — diff the files and copy the intended\n" +
			"      user-authored content into both:\n" +
			"      diff " + agentsFile + " CLAUDE.md\n" +
			"\n" +
			"  (d) If the divergence is intentional (e.g. distinct audiences for\n" +
			"      each file), opt out by adding this HTML comment anywhere in\n" +
			"      either file:\n" +
			"      <!-- bd-doctor-divergence: ok -->",
	}
}

// CheckDatabaseConfig verifies that the configured database path matches what
// actually exists on disk. For Dolt backends, data is on the server. For legacy
// backends, this checks that .db files match the configuration.
func CheckDatabaseConfig(repoPath string) DoctorCheck {
	issuegraphDir := ResolveIssueGraphDirForRepo(repoPath)

	// Load config
	cfg, err := configfile.Load(issuegraphDir)
	if err != nil || cfg == nil {
		// No config or error reading - use defaults
		return DoctorCheck{
			Name:    "Database Config",
			Status:  StatusOK,
			Message: "Using default configuration",
		}
	}

	// Dolt backend stores data on the server — no local .db or .jsonl files expected
	if cfg.GetBackend() == configfile.BackendDolt {
		return DoctorCheck{
			Name:    "Database Config",
			Status:  StatusOK,
			Message: "Dolt backend (data on server)",
		}
	}

	var issues []string

	// Check if configured database exists
	if cfg.Database != "" {
		dbPath := cfg.DatabasePath(issuegraphDir)
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			// Check if other .db files exist
			entries, _ := os.ReadDir(issuegraphDir) // Best effort: nil entries means no legacy files to check
			var otherDBs []string
			for _, entry := range entries {
				if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".db") {
					otherDBs = append(otherDBs, entry.Name())
				}
			}
			if len(otherDBs) > 0 {
				issues = append(issues, fmt.Sprintf("Configured database '%s' not found, but found: %s",
					cfg.Database, strings.Join(otherDBs, ", ")))
			}
		}
	}

	if len(issues) == 0 {
		return DoctorCheck{
			Name:    "Database Config",
			Status:  StatusOK,
			Message: "Configuration matches existing files",
		}
	}

	return DoctorCheck{
		Name:    "Database Config",
		Status:  StatusWarning,
		Message: "Configuration mismatch detected",
		Detail:  strings.Join(issues, "\n  "),
		Fix: "Run 'issuegraph doctor --fix' to auto-detect and fix mismatches, or manually:\n" +
			"  1. Check which files are actually being used\n" +
			"  2. Update metadata.json to match the actual filenames\n" +
			"  3. Or rename the files to match the configuration",
	}
}

// CheckFreshClone detects if this is a fresh clone that needs 'issuegraph init'.
// A fresh clone has legacy JSONL with issues but no database (Dolt or SQLite).
func CheckFreshClone(repoPath string) DoctorCheck {
	backend, issuegraphDir := getBackendAndIssueGraphDir(repoPath)

	// Check if .issuegraph/ exists
	if _, err := os.Stat(issuegraphDir); os.IsNotExist(err) {
		return DoctorCheck{
			Name:    "Fresh Clone",
			Status:  StatusOK,
			Message: "N/A (no .issuegraph directory)",
		}
	}

	// Find the JSONL file
	var jsonlPath string
	var jsonlName string
	for _, name := range []string{"issues.jsonl", "issuegraph.jsonl"} {
		testPath := filepath.Join(issuegraphDir, name)
		if _, err := os.Stat(testPath); err == nil {
			jsonlPath = testPath
			jsonlName = name
			break
		}
	}

	// No JSONL file - not a fresh clone situation
	if jsonlPath == "" {
		return DoctorCheck{
			Name:    "Fresh Clone",
			Status:  StatusOK,
			Message: "N/A (no JSONL file)",
		}
	}

	// Check if database exists (backend-aware)
	switch backend {
	case configfile.BackendDolt:
		// Dolt is directory-backed: treat .issuegraph/dolt as the DB existence signal.
		if info, err := os.Stat(getDatabasePath(issuegraphDir)); err == nil && info.IsDir() {
			return DoctorCheck{
				Name:    "Fresh Clone",
				Status:  StatusOK,
				Message: "Database exists",
			}
		}
		// No local dolt directory — check server mode (FR-020, FR-021).
		if cfg, err := configfile.Load(issuegraphDir); err == nil && cfg != nil && cfg.IsDoltServerMode() {
			host := cfg.GetDoltServerHost()
			// Use DefaultConfig (not deprecated GetDoltServerPort) to resolve
			// the correct port for standalone server mode (ephemeral).
			port := doltserver.DefaultConfig(issuegraphDir).Port
			user := cfg.GetDoltServerUser()
			// Look up the password by the resolved runtime port — cfg.GetDoltServerPassword
			// uses cfg.GetDoltServerPort() which falls back to the 3307 default when
			// metadata.json omits the port, causing credentials lookup to miss when
			// the actual server runs on a different port (e.g. 3306 for
			// externally-hosted Dolt). That mismatch produced spurious
			// "Fresh clone detected" warnings (bd-tzo9).
			password := cfg.GetDoltServerPasswordForPort(port)
			dbName := cfg.GetDoltDatabase()
			result := checkFreshCloneDB(host, port, user, password, dbName, cfg.GetDoltServerTLS())
			if result.Reachable {
				syncRemote := config.GetStringFromDir(issuegraphDir, "sync.remote")
				if syncRemote == "" {
					syncRemote = config.GetStringFromDir(issuegraphDir, "sync.git-remote")
				}
				return freshCloneServerResult(result.Exists, dbName, host, port, syncRemote)
			}
			// Server unreachable in server mode — emit a server-aware warning
			// instead of falling through to the legacy "Fresh clone detected
			// (no database)" message, which is a false positive when
			// dolt_mode=server (the local DB absence is expected). See GH#35.
			// FR-030 only requires that we don't panic on unreachable; it does
			// not mandate the misleading fall-through.
			return freshCloneServerUnreachableResult(dbName, host, port, result.Err)
		}
	default:
		// Non-Dolt (legacy sqlite-era metadata): check configured .db file path.
		var dbPath string
		if cfg, err := configfile.Load(issuegraphDir); err == nil && cfg != nil && cfg.Database != "" {
			dbPath = cfg.DatabasePath(issuegraphDir)
		} else {
			// Fall back to canonical database name
			dbPath = filepath.Join(issuegraphDir, issuegraph.CanonicalDatabaseName)
		}
		if _, err := os.Stat(dbPath); err == nil {
			return DoctorCheck{
				Name:    "Fresh Clone",
				Status:  StatusOK,
				Message: "Database exists",
			}
		}
	}

	// Check if JSONL has any issues (empty JSONL = not really a fresh clone)
	issueCount, _ := countJSONLIssuesAndPrefix(jsonlPath)
	if issueCount == 0 {
		return DoctorCheck{
			Name:    "Fresh Clone",
			Status:  StatusOK,
			Message: fmt.Sprintf("JSONL exists but is empty (%s)", jsonlName),
		}
	}

	// This is a fresh clone! JSONL has issues but no database.
	fixCmd := "issuegraph bootstrap"

	return DoctorCheck{
		Name:    "Fresh Clone",
		Status:  StatusWarning,
		Message: fmt.Sprintf("Fresh clone detected (%d issues in %s, no database)", issueCount, jsonlName),
		Detail: "This appears to be a freshly cloned repository.\n" +
			"  The JSONL file contains issues but no local database exists.\n" +
			"  Run 'issuegraph bootstrap' as the safe entry point for recovering existing state.\n" +
			"  Use '--dry-run' first if you need to inspect whether bootstrap will recover or initialize.\n" +
			"  Use 'issuegraph init' only when creating a brand-new project with no existing .issuegraph data.",
		Fix: fmt.Sprintf("Run '%s' to recover the existing database and import tracked issues", fixCmd),
	}
}

// countJSONLIssuesAndPrefix counts issues in a legacy JSONL file and detects the most common prefix.
func countJSONLIssuesAndPrefix(jsonlPath string) (int, string) {
	file, err := os.Open(jsonlPath) //nolint:gosec
	if err != nil {
		return 0, ""
	}
	defer file.Close()

	count := 0
	prefixCounts := make(map[string]int)

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 1024), 2*1024*1024) // 2MB buffer for large lines
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var issue struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(line, &issue); err != nil {
			continue
		}

		if issue.ID != "" {
			count++
			// Extract prefix (everything before the last dash)
			if lastDash := strings.LastIndex(issue.ID, "-"); lastDash > 0 {
				prefix := issue.ID[:lastDash]
				prefixCounts[prefix]++
			}
		}
	}

	// Find most common prefix
	var mostCommonPrefix string
	maxCount := 0
	for prefix, cnt := range prefixCounts {
		if cnt > maxCount {
			maxCount = cnt
			mostCommonPrefix = prefix
		}
	}

	return count, mostCommonPrefix
}
