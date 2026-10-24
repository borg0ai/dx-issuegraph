package setup

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/borg0ai/dx-issuegraph/internal/templates/agents"
)

var (
	geminiEnvProvider     = defaultGeminiEnv
	errGeminiHooksMissing = errors.New("gemini hooks not installed")
	errGeminiHooksLegacy  = errors.New("gemini hooks need upgrade")
)

const geminiInstructionsFile = "GEMINI.md"

var geminiAgentsIntegration = agentsIntegration{
	name:         "Gemini CLI",
	setupCommand: "issuegraph setup gemini",
	profile:      agents.ProfileMinimal,
}

type geminiEnv struct {
	stdout     io.Writer
	stderr     io.Writer
	homeDir    string
	projectDir string
	ensureDir  func(string, os.FileMode) error
	readFile   func(string) ([]byte, error)
	writeFile  func(string, []byte) error
}

func defaultGeminiEnv() (geminiEnv, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return geminiEnv{}, fmt.Errorf("home directory: %w", err)
	}
	workDir, err := os.Getwd()
	if err != nil {
		return geminiEnv{}, fmt.Errorf("working directory: %w", err)
	}
	return geminiEnv{
		stdout:     os.Stdout,
		stderr:     os.Stderr,
		homeDir:    home,
		projectDir: workDir,
		ensureDir:  EnsureDir,
		readFile:   os.ReadFile,
		writeFile: func(path string, data []byte) error {
			return atomicWriteFile(path, data)
		},
	}, nil
}

func geminiProjectSettingsPath(base string) string {
	return filepath.Join(base, ".gemini", "settings.json")
}

func geminiGlobalSettingsPath(home string) string {
	return filepath.Join(home, ".gemini", "settings.json")
}

func geminiAgentsEnv(env geminiEnv) agentsEnv {
	return agentsEnv{
		agentsPath: filepath.Join(env.projectDir, geminiInstructionsFile),
		stdout:     env.stdout,
		stderr:     env.stderr,
	}
}

func InstallGemini(project bool, stealth bool) error {
	env, err := geminiEnvProvider()
	if err != nil {
		return HandleError("%v", err)
	}
	return installGemini(env, project, stealth)
}

func installGemini(env geminiEnv, project bool, stealth bool) error {
	var settingsPath string
	if project {
		settingsPath = geminiProjectSettingsPath(env.projectDir)
		_, _ = fmt.Fprintln(env.stdout, "Installing Gemini CLI hooks for this project...")
	} else {
		settingsPath = geminiGlobalSettingsPath(env.homeDir)
		_, _ = fmt.Fprintln(env.stdout, "Installing Gemini CLI hooks globally...")
	}

	if err := env.ensureDir(filepath.Dir(settingsPath), 0o755); err != nil {
		_, _ = fmt.Fprintf(env.stderr, "Error: %v\n", err)
		return err
	}

	settings := make(map[string]interface{})
	if data, err := env.readFile(settingsPath); err == nil {
		if err := json.Unmarshal(data, &settings); err != nil {
			_, _ = fmt.Fprintf(env.stderr, "Error: failed to parse settings.json: %v\n", err)
			return err
		}
	}

	hooks, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		hooks = make(map[string]interface{})
		settings["hooks"] = hooks
	}

	// Gemini CLI (and Claude Code, Codex) require hook stdout to be valid JSON;
	// --hook-json wraps issuegraph prime's markdown in the shared SessionStart envelope.
	// PreCompress is intentionally NOT registered: Gemini's PreCompress event is
	// advisory-only and does not support additionalContext injection — context
	// re-injection after compression is architecturally impossible there.
	command := "issuegraph prime --hook-json"
	if stealth {
		command = "issuegraph prime --stealth --hook-json"
	}

	// Migration sweep: remove all known legacy command variants before registering
	// the canonical command. Re-running setup must be a clean upgrade path —
	// stale entries alongside the new one cause Gemini to invoke both, and any
	// pre-JSON variant emits raw markdown that violates the strict hook contract.
	legacyVariants := []string{"issuegraph prime", "issuegraph prime --stealth"}
	for _, legacy := range legacyVariants {
		if legacy == command {
			continue // never remove the variant we're about to add
		}
		removeHookCommand(hooks, "SessionStart", legacy)
		removeHookCommand(hooks, "PreCompress", legacy)
	}
	// Also clear any --hook-json registration from PreCompress. IssueGraph previously
	// registered issuegraph prime on PreCompress before we discovered that Gemini's
	// PreCompress event is advisory-only and cannot inject additionalContext
	// into the model regardless of output format. Re-running setup migrates cleanly.
	removeHookCommand(hooks, "PreCompress", "issuegraph prime --hook-json")
	removeHookCommand(hooks, "PreCompress", "issuegraph prime --stealth --hook-json")

	if addHookCommand(hooks, "SessionStart", command) {
		_, _ = fmt.Fprintln(env.stdout, "✓ Registered SessionStart hook")
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		_, _ = fmt.Fprintf(env.stderr, "Error: marshal settings: %v\n", err)
		return err
	}

	if err := env.writeFile(settingsPath, data); err != nil {
		_, _ = fmt.Fprintf(env.stderr, "Error: write settings: %v\n", err)
		return err
	}

	// Install minimal issuegraph section in GEMINI.md.
	// Hooks handle the heavy lifting via issuegraph prime; GEMINI.md just needs a pointer.
	if err := installAgents(geminiAgentsEnv(env), geminiAgentsIntegration); err != nil {
		// Non-fatal: hooks are already installed
		_, _ = fmt.Fprintf(env.stderr, "Warning: failed to update %s: %v\n", geminiInstructionsFile, err)
	}

	_, _ = fmt.Fprintln(env.stdout, "\n✓ Gemini CLI integration installed")
	_, _ = fmt.Fprintf(env.stdout, "  Settings: %s\n", settingsPath)
	_, _ = fmt.Fprintln(env.stdout, "\nRestart Gemini CLI for changes to take effect.")
	return nil
}

func CheckGemini() error {
	env, err := geminiEnvProvider()
	if err != nil {
		return HandleError("%v", err)
	}
	return checkGemini(env)
}

func checkGemini(env geminiEnv) error {
	globalSettings := geminiGlobalSettingsPath(env.homeDir)
	projectSettings := geminiProjectSettingsPath(env.projectDir)

	switch {
	case hasCurrentGeminiHooks(globalSettings):
		_, _ = fmt.Fprintf(env.stdout, "✓ Global hooks installed (current): %s\n", globalSettings)
	case hasCurrentGeminiHooks(projectSettings):
		_, _ = fmt.Fprintf(env.stdout, "✓ Project hooks installed (current): %s\n", projectSettings)
	case hasGeminiIssueGraphHooks(globalSettings):
		_, _ = fmt.Fprintf(env.stdout, "⚠ Global hooks installed (legacy format): %s\n", globalSettings)
		_, _ = fmt.Fprintln(env.stdout, "  Legacy 'issuegraph prime' hooks emit raw markdown; Gemini requires JSON stdout.")
		_, _ = fmt.Fprintln(env.stdout, "  Run: issuegraph setup gemini  (upgrades to 'issuegraph prime --hook-json')")
		return errGeminiHooksLegacy
	case hasGeminiIssueGraphHooks(projectSettings):
		_, _ = fmt.Fprintf(env.stdout, "⚠ Project hooks installed (legacy format): %s\n", projectSettings)
		_, _ = fmt.Fprintln(env.stdout, "  Legacy 'issuegraph prime' hooks emit raw markdown; Gemini requires JSON stdout.")
		_, _ = fmt.Fprintln(env.stdout, "  Run: issuegraph setup gemini --project  (upgrades to 'issuegraph prime --hook-json')")
		return errGeminiHooksLegacy
	default:
		_, _ = fmt.Fprintln(env.stdout, "✗ No hooks installed")
		_, _ = fmt.Fprintln(env.stdout, "  Run: issuegraph setup gemini")
		return errGeminiHooksMissing
	}

	return checkAgents(geminiAgentsEnv(env), geminiAgentsIntegration)
}

func RemoveGemini(project bool) error {
	env, err := geminiEnvProvider()
	if err != nil {
		return HandleError("%v", err)
	}
	return removeGemini(env, project)
}

func removeGemini(env geminiEnv, project bool) error {
	var settingsPath string
	if project {
		settingsPath = geminiProjectSettingsPath(env.projectDir)
		_, _ = fmt.Fprintln(env.stdout, "Removing Gemini CLI hooks from project...")
	} else {
		settingsPath = geminiGlobalSettingsPath(env.homeDir)
		_, _ = fmt.Fprintln(env.stdout, "Removing Gemini CLI hooks globally...")
	}

	data, err := env.readFile(settingsPath)
	if err != nil {
		_, _ = fmt.Fprintln(env.stdout, "No settings file found")
	} else {
		var settings map[string]interface{}
		if err := json.Unmarshal(data, &settings); err != nil {
			_, _ = fmt.Fprintf(env.stderr, "Error: failed to parse settings.json: %v\n", err)
			return err
		}

		hooks, ok := settings["hooks"].(map[string]interface{})
		if !ok {
			_, _ = fmt.Fprintln(env.stdout, "No hooks found")
		} else {
			// Remove all known variants from both events. PreCompress is
			// included for migration safety: older installations registered
			// issuegraph prime there before we discovered Gemini's PreCompress hook
			// can't inject additionalContext.
			variants := []string{
				"issuegraph prime",
				"issuegraph prime --stealth",
				"issuegraph prime --hook-json",
				"issuegraph prime --stealth --hook-json",
			}
			for _, cmd := range variants {
				removeHookCommand(hooks, "SessionStart", cmd)
				removeHookCommand(hooks, "PreCompress", cmd)
			}

			data, err = json.MarshalIndent(settings, "", "  ")
			if err != nil {
				_, _ = fmt.Fprintf(env.stderr, "Error: marshal settings: %v\n", err)
				return err
			}

			if err := env.writeFile(settingsPath, data); err != nil {
				_, _ = fmt.Fprintf(env.stderr, "Error: write settings: %v\n", err)
				return err
			}
		}
	}

	if err := removeAgents(geminiAgentsEnv(env), geminiAgentsIntegration); err != nil {
		// Non-fatal
		_, _ = fmt.Fprintf(env.stderr, "Warning: failed to update %s: %v\n", geminiInstructionsFile, err)
	}

	_, _ = fmt.Fprintln(env.stdout, "✓ Gemini CLI hooks removed")
	return nil
}

// geminiSessionStartCommands returns all command strings registered under the
// SessionStart hook event in a settings file. Returns nil on any read/parse
// error. Detection scope is SessionStart only — PreCompress is advisory-only
// in Gemini CLI and does not support additionalContext injection.
func geminiSessionStartCommands(settingsPath string) []string {
	data, err := os.ReadFile(settingsPath) // #nosec G304 -- path constructed from known safe locations (user home/.gemini or cwd/.gemini)
	if err != nil {
		return nil
	}
	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil
	}
	hooks, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		return nil
	}
	eventHooks, ok := hooks["SessionStart"].([]interface{})
	if !ok {
		return nil
	}
	var cmds []string
	for _, hook := range eventHooks {
		hookMap, ok := hook.(map[string]interface{})
		if !ok {
			continue
		}
		hookCmds, ok := hookMap["hooks"].([]interface{})
		if !ok {
			continue
		}
		for _, cmd := range hookCmds {
			cmdMap, ok := cmd.(map[string]interface{})
			if !ok {
				continue
			}
			if s, ok := cmdMap["command"].(string); ok {
				cmds = append(cmds, s)
			}
		}
	}
	return cmds
}

// hasCurrentGeminiHooks reports whether a settings file has a current-format
// issuegraph prime --hook-json command on SessionStart. Returns false for legacy
// "issuegraph prime" installs, which emit raw markdown that violates Gemini's strict
// stdout-must-be-JSON hook contract.
func hasCurrentGeminiHooks(settingsPath string) bool {
	for _, cmd := range geminiSessionStartCommands(settingsPath) {
		if cmd == "issuegraph prime --hook-json" || cmd == "issuegraph prime --stealth --hook-json" {
			return true
		}
	}
	return false
}

// hasGeminiIssueGraphHooks reports whether a settings file has any issuegraph prime hook
// on SessionStart — current format or legacy. Used to detect pre-fix
// installations that need upgrading via issuegraph setup gemini.
func hasGeminiIssueGraphHooks(settingsPath string) bool {
	for _, cmd := range geminiSessionStartCommands(settingsPath) {
		switch cmd {
		case "issuegraph prime", "issuegraph prime --stealth",
			"issuegraph prime --hook-json", "issuegraph prime --stealth --hook-json":
			return true
		}
	}
	return false
}
