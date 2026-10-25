package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/borg0ai/dx-issuegraph/apps/cli/doctor"
	"github.com/borg0ai/dx-issuegraph/internal/git"
	"github.com/borg0ai/dx-issuegraph/internal/issuegraph"
	"github.com/borg0ai/dx-issuegraph/internal/metrics"
	"github.com/spf13/cobra"
)

// CheckResult represents the result of a single preflight check.
type CheckResult struct {
	Name    string `json:"name"`
	Passed  bool   `json:"passed"`
	Skipped bool   `json:"skipped,omitempty"`
	Warning bool   `json:"warning,omitempty"`
	Output  string `json:"output,omitempty"`
	Command string `json:"command"`
}

// PreflightResult represents the overall preflight check results.
type PreflightResult struct {
	Checks  []CheckResult `json:"checks"`
	Passed  bool          `json:"passed"`
	Summary string        `json:"summary"`
}

var preflightCmd = &cobra.Command{
	Use:     "preflight",
	GroupID: "maint",
	Short:   "Show PR readiness checklist",
	Long: `Display a checklist of common pre-PR checks for contributors.

This command helps catch common issues before pushing to CI:
- Tests not run locally
- Lint errors
- Unformatted Go files
- .issuegraph/issues.jsonl pollution
- Version mismatches

Examples:
  issuegraph preflight              # Show checklist
  issuegraph preflight --check      # Run checks automatically
  issuegraph preflight --check --json  # JSON output for programmatic use
  issuegraph preflight --check --skip-lint  # Explicitly skip lint check
`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runPreflight,
}

func init() {
	preflightCmd.Flags().Bool("check", false, "Run checks automatically")
	// Bound to the package global like every other command's local --json.
	// commandJSONFlagChanged suppresses the config-driven default whenever a
	// local --json is set, so a flag that does not write the global would
	// leave `issuegraph preflight --json` with jsonOutput=false — inverting JSON mode
	// for everything that reads the global, including the front-door error
	// renderers.
	preflightCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output results as JSON")
	preflightCmd.Flags().Bool("skip-lint", false, "Skip lint check explicitly")

	rootCmd.AddCommand(preflightCmd)
}

func runPreflight(cmd *cobra.Command, args []string) error {
	evt := metrics.NewCommandEvent("preflight")
	defer func() {
		if c := metrics.Global(); c != nil {
			c.CloseEventAndAdd(evt)
		}
	}()

	check, _ := cmd.Flags().GetBool("check")
	skipLint, _ := cmd.Flags().GetBool("skip-lint")

	if check {
		return runChecks(jsonOutput, skipLint)
	}

	// Static checklist mode — tailor the checklist to the detected project
	// stack so non-Go projects don't get a misleading Go checklist (GH#4364).
	root := git.GetRepoRoot()
	if root == "" {
		if wd, err := os.Getwd(); err == nil {
			root = wd
		}
	}

	fmt.Println("PR Readiness Checklist:")
	fmt.Println()
	for _, item := range buildPreflightChecklist(root) {
		fmt.Printf("[ ] %s\n", item)
	}
	fmt.Println()
	fmt.Println("Run 'issuegraph preflight --check' to validate automatically.")
	return nil
}

// fileExists reports whether name exists directly under dir.
func fileExists(dir, name string) bool {
	if dir == "" {
		return false
	}
	_, err := os.Stat(filepath.Join(dir, name))
	return err == nil
}

// buildPreflightChecklist returns PR-readiness checklist items tailored to the
// project's language stack, detected from standard marker files in dir. A
// project with no recognized stack gets a generic reminder rather than a
// misleading Go checklist, and the repo's own Go specific items
// (gms_pure_go build tags, version consistency) only
// appear where they apply (GH#4364).
func buildPreflightChecklist(dir string) []string {
	// Preserve the exact rich checklist when run inside the issuegraph repo itself
	// (its primary audience), so the gms_pure_go build tags and version
	// reminders issuegraph contributors rely on are not lost.
	if isIssueGraphRepo(dir) {
		return []string{
			"Tests pass: go test -tags gms_pure_go -short ./...",
			"Lint passes: golangci-lint run --build-tags=gms_pure_go ./...",
			"Formatting: gofmt -l .",
			"No issuegraph pollution: check .issuegraph/issues.jsonl diff",
			"Version sync: scripts/check-versions.sh passes",
		}
	}

	var items []string
	switch {
	case fileExists(dir, "go.mod"):
		items = append(items,
			"Tests pass: go test ./...",
			"Lint passes: golangci-lint run ./...",
			"Formatting: gofmt -l .",
		)
	case fileExists(dir, "package.json"):
		items = append(items,
			"Tests pass: npm test",
			"Types check: tsc --noEmit",
		)
	case fileExists(dir, "pyproject.toml"), fileExists(dir, "setup.py"):
		items = append(items,
			"Tests pass: pytest",
			"Lint passes: ruff check",
		)
	case fileExists(dir, "Cargo.toml"):
		items = append(items,
			"Tests pass: cargo test",
			"Lint passes: cargo clippy",
		)
	default:
		items = append(items, "Tests pass: run your project's test suite, linter, and formatter")
	}

	// Relevant to any issuegraph workspace, regardless of language.
	if fileExists(dir, ".issuegraph") {
		items = append(items, "No issuegraph pollution: check .issuegraph/issues.jsonl diff")
	}

	return items
}

// isIssueGraphRepo reports whether dir is the issuegraph source repo, detected by the
// module path in go.mod. Used to keep preflight's repo-specific checklist.
func isIssueGraphRepo(dir string) bool {
	data, err := os.ReadFile(filepath.Join(dir, "go.mod")) //nolint:gosec // path is constructed internally (repo root + fixed filename)
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")) == "github.com/borg0ai/dx-issuegraph"
		}
	}
	return false
}

// runChecks executes all preflight checks and reports results.
func runChecks(jsonOutput, skipLint bool) error {
	var results []CheckResult

	// Run test check
	testResult := runTestCheck()
	results = append(results, testResult)

	// Run lint check
	lintResult := runLintCheck(skipLint)
	results = append(results, lintResult)

	// Run formatting check
	fmtResult := runFmtCheck()
	results = append(results, fmtResult)

	// Run issuegraph pollution check
	issuegraphResult := runIssueGraphPollutionCheck()
	results = append(results, issuegraphResult)

	// Run version sync check
	versionResult := runVersionSyncCheck()
	results = append(results, versionResult)

	// Run AGENTS.md / CLAUDE.md divergence check
	divergenceResult := runAgentDocDivergenceCheck()
	results = append(results, divergenceResult)

	// Calculate overall result
	allPassed := true
	passCount := 0
	skipCount := 0
	warnCount := 0
	for _, r := range results {
		if r.Skipped {
			skipCount++
		} else if r.Warning {
			warnCount++
			// Warnings don't fail the overall result but count as "not passed"
		} else if r.Passed {
			passCount++
		} else {
			allPassed = false
		}
	}

	runCount := len(results) - skipCount
	summary := fmt.Sprintf("%d/%d checks passed", passCount, runCount)
	if warnCount > 0 {
		summary += fmt.Sprintf(", %d warning(s)", warnCount)
	}
	if skipCount > 0 {
		summary += fmt.Sprintf(" (%d skipped)", skipCount)
	}

	if jsonOutput {
		result := PreflightResult{
			Checks:  results,
			Passed:  allPassed,
			Summary: summary,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(result); err != nil {
			return HandleError("encoding preflight result: %v", err)
		}
		if !allPassed {
			return SilentExit()
		}
		return nil
	}
	for _, r := range results {
		if r.Skipped {
			fmt.Printf("⚠ %s (skipped)\n", r.Name)
		} else if r.Warning {
			fmt.Printf("⚠ %s\n", r.Name)
		} else if r.Passed {
			fmt.Printf("✓ %s\n", r.Name)
		} else {
			fmt.Printf("✗ %s\n", r.Name)
		}
		fmt.Printf("  Command: %s\n", r.Command)
		if r.Skipped && r.Output != "" {
			fmt.Printf("  Reason: %s\n", r.Output)
		} else if r.Warning && r.Output != "" {
			fmt.Printf("  Warning: %s\n", r.Output)
		} else if !r.Passed && r.Output != "" {
			output := truncateOutput(r.Output, 500)
			fmt.Printf("  Output:\n")
			for _, line := range strings.Split(output, "\n") {
				fmt.Printf("    %s\n", line)
			}
		}
		fmt.Println()
	}
	fmt.Println(summary)

	if !allPassed {
		return SilentExit()
	}
	return nil
}

// runTestCheck runs go test -short ./... and returns the result.
func runTestCheck() CheckResult {
	command := "go test -tags gms_pure_go -short ./..."
	cmd := exec.Command("go", "test", "-tags", "gms_pure_go", "-short", "./...")
	output, err := cmd.CombinedOutput()

	return CheckResult{
		Name:    "Tests pass",
		Passed:  err == nil,
		Output:  string(output),
		Command: command,
	}
}

// runLintCheck runs golangci-lint and returns the result.
func runLintCheck(skipLint bool) CheckResult {
	command := "golangci-lint run --build-tags=gms_pure_go ./..."
	if skipLint {
		return CheckResult{
			Name:    "Lint passes",
			Passed:  false,
			Skipped: true,
			Warning: true,
			Output:  "lint check explicitly skipped by --skip-lint",
			Command: command,
		}
	}

	// Check if golangci-lint is available
	if _, err := exec.LookPath("golangci-lint"); err != nil {
		return CheckResult{
			Name:    "Lint passes",
			Passed:  false,
			Output:  "golangci-lint not found in PATH (install it or rerun with --skip-lint)",
			Command: command,
		}
	}

	cmd := exec.Command("golangci-lint", "run", "--build-tags=gms_pure_go", "./...")
	output, err := cmd.CombinedOutput()

	return CheckResult{
		Name:    "Lint passes",
		Passed:  err == nil,
		Output:  string(output),
		Command: command,
	}
}

// runFmtCheck runs gofmt -l and fails if any files need formatting.
func runFmtCheck() CheckResult {
	command := "gofmt -l ."

	// Check if gofmt is available
	if _, err := exec.LookPath("gofmt"); err != nil {
		return CheckResult{
			Name:    "Formatting",
			Passed:  false,
			Output:  "gofmt not found in PATH (install Go toolchain)",
			Command: command,
		}
	}

	cmd := exec.Command("gofmt", "-l", ".")
	output, err := cmd.CombinedOutput()

	if err != nil {
		return CheckResult{
			Name:    "Formatting",
			Passed:  false,
			Output:  string(output),
			Command: command,
		}
	}

	unformatted := strings.TrimSpace(string(output))
	if unformatted != "" {
		return CheckResult{
			Name:    "Formatting",
			Passed:  false,
			Output:  fmt.Sprintf("Unformatted files:\n%s\nRun: gofmt -w .", unformatted),
			Command: command,
		}
	}

	return CheckResult{
		Name:    "Formatting",
		Passed:  true,
		Command: command,
	}
}

// runIssueGraphPollutionCheck detects .issuegraph/issues.jsonl modifications vs merge base.
func runIssueGraphPollutionCheck() CheckResult {
	command := "git diff -- .issuegraph/issues.jsonl"

	issuegraphDir := issuegraph.FindIssueGraphDir()
	if issuegraphDir == "" {
		return CheckResult{
			Name:    "No issuegraph pollution",
			Passed:  true,
			Command: command,
		}
	}

	// git diff requires a path relative to the worktree root.
	// If issuegraphDir points outside the worktree (shared .issuegraph in a
	// worktree setup), convert to a relative path. When the path is
	// outside the worktree, the pollution check is skipped since git
	// cannot diff paths outside the working tree.
	issuesPath := filepath.Join(issuegraphDir, "issues.jsonl")
	if filepath.IsAbs(issuesPath) {
		repoRoot := git.GetRepoRoot()
		if repoRoot == "" {
			return CheckResult{
				Name:    "No issuegraph pollution",
				Passed:  true,
				Command: command,
			}
		}
		rel, err := filepath.Rel(repoRoot, issuesPath)
		if err != nil || isPathOutsideRepo(rel) {
			return CheckResult{
				Name:    "No issuegraph pollution",
				Passed:  true,
				Command: command,
				Output:  "Skipped: .issuegraph is outside working tree (worktree setup)",
			}
		}
		issuesPath = rel
	}

	branchCmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	branchOut, err := branchCmd.Output()
	if err != nil {
		return CheckResult{
			Name:    "No issuegraph pollution",
			Passed:  false,
			Skipped: true,
			Output:  fmt.Sprintf("Cannot determine branch: %v", err),
			Command: command,
		}
	}
	branch := strings.TrimSpace(string(branchOut))

	var diffOutput []byte
	if branch != "main" && branch != "HEAD" {
		cmd := exec.Command("git", "diff", "origin/main...HEAD", "--", issuesPath)
		diffOutput, _ = cmd.Output()
	} else {
		cmd := exec.Command("git", "diff", "HEAD", "--", issuesPath)
		out1, _ := cmd.Output()
		cmd2 := exec.Command("git", "diff", "--cached", "--", issuesPath)
		out2, _ := cmd2.Output()
		diffOutput = append(out1, out2...)
	}

	if len(strings.TrimSpace(string(diffOutput))) > 0 {
		return CheckResult{
			Name:    "No issuegraph pollution",
			Passed:  false,
			Output:  ".issuegraph/issues.jsonl has been modified — revert changes before pushing",
			Command: command,
		}
	}

	return CheckResult{
		Name:    "No issuegraph pollution",
		Passed:  true,
		Command: command,
	}
}

// isPathOutsideRepo checks if a relative path (from filepath.Rel) points
// outside the base directory by inspecting the first path segment.
func isPathOutsideRepo(rel string) bool {
	if rel == "" {
		return false
	}
	first := rel
	if i := strings.IndexAny(rel, "/\\"); i > 0 {
		first = rel[:i]
	}
	return first == ".."
}

// runVersionSyncCheck checks that all version files are in sync.
// Runs scripts/check-versions.sh and skips when the script is absent.
func runVersionSyncCheck() CheckResult {
	command := "scripts/check-versions.sh"

	// Try using the script (matches CI's check-version-consistency job)
	if _, err := os.Stat("scripts/check-versions.sh"); err == nil {
		cmd := exec.Command("bash", "scripts/check-versions.sh")
		output, err := cmd.CombinedOutput()
		if err != nil {
			return CheckResult{
				Name:    "Version sync",
				Passed:  false,
				Output:  string(output),
				Command: command,
			}
		}
		return CheckResult{
			Name:    "Version sync",
			Passed:  true,
			Output:  string(output),
			Command: command,
		}
	}

	return CheckResult{
		Name:    "Version sync",
		Passed:  true,
		Skipped: true,
		Output:  "scripts/check-versions.sh not found (skipping version sync check)",
		Command: command,
	}
}

// runAgentDocDivergenceCheck flags drift between AGENTS.md and CLAUDE.md
// user-authored regions so the inconsistency is caught pre-PR rather than in
// review.
func runAgentDocDivergenceCheck() CheckResult {
	command := "issuegraph doctor (Agent Doc Divergence)"

	repoRoot := git.GetRepoRoot()
	if repoRoot == "" {
		repoRoot = "."
	}
	check := doctor.CheckAgentDocDivergence(repoRoot)
	if check.Status == doctor.StatusOK {
		return CheckResult{
			Name:    "AGENTS.md/CLAUDE.md in sync",
			Passed:  true,
			Command: command,
		}
	}
	output := check.Message
	if check.Detail != "" {
		output += "\n" + check.Detail
	}
	if check.Fix != "" {
		output += "\n" + check.Fix
	}
	return CheckResult{
		Name:    "AGENTS.md/CLAUDE.md in sync",
		Passed:  false,
		Warning: true,
		Output:  output,
		Command: command,
	}
}

// truncateOutput truncates output to maxLen characters, adding ellipsis if truncated.
func truncateOutput(s string, maxLen int) string {
	if len(s) <= maxLen {
		return strings.TrimSpace(s)
	}
	return strings.TrimSpace(s[:maxLen]) + "\n... (truncated)"
}
