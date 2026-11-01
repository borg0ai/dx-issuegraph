package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/borg0ai/dx-issuegraph/apps/cli/setup"
	"github.com/borg0ai/dx-issuegraph/internal/templates"
	"github.com/borg0ai/dx-issuegraph/internal/ui"
)

var installDryRun bool

// skillAgentHasRecipe reports whether an agent also has a repository/editor recipe.
func skillAgentHasRecipe(name string) bool {
	switch name {
	case "claude", "codex", "gemini":
		return true
	}
	return false
}

func isSkillAgent(name string) bool {
	for _, n := range setup.SkillAgentNames() {
		if n == name {
			return true
		}
	}
	return false
}

// runInstallAgent installs one skill-capable agent: its recipe (when it has one)
// followed by the global skill.
func runInstallAgent(name string) error {
	if skillAgentHasRecipe(name) {
		if err := runRecipe(name); err != nil {
			return err
		}
	}
	if setupProject || setupCheck {
		return nil
	}
	return runInstallSkills([]string{name})
}

// runInstallSkills installs, checks, or removes the global IssueGraph skill.
// With no agents it targets every detected agent.
func runInstallSkills(agents []string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return HandleError("home directory: %v", err)
	}
	return installSkills(os.Stdout, home, os.Getenv, agents)
}

func installSkills(w io.Writer, home string, getenv func(string) string, agents []string) error {
	results, err := setup.InstallSkills(setup.SkillOptions{
		Home:   home,
		Getenv: getenv,
		Files:  templates.IssueGraphSkillFiles(),
		Agents: agents,
		DryRun: installDryRun || setupCheck,
		Remove: setupRemove,
	})
	if err != nil {
		return HandleError("%v", err)
	}
	if jsonOutput {
		return outputJSON(map[string]interface{}{"agents": results})
	}

	var title string
	switch {
	case setupRemove:
		title = "Removing IssueGraph agent skill"
	case setupCheck:
		title = "Checking IssueGraph agent skill"
	case installDryRun:
		title = "Installing IssueGraph agent skill (dry run)"
	default:
		title = "Installing IssueGraph agent skill"
	}
	_, _ = fmt.Fprintln(w, ui.Banner(title))

	detected, stale := 0, false
	for _, r := range results {
		if r.Status == setup.SkillSkipped {
			_, _ = fmt.Fprintln(w, "  "+ui.Info(fmt.Sprintf("%-12s skipped (not detected)", r.Agent)))
			continue
		}
		detected++
		if r.Status != setup.SkillUnchanged && r.Status != setup.SkillMissing {
			stale = true
		}
		_, _ = fmt.Fprintln(w, "  "+ui.Success(fmt.Sprintf("%-12s %-13s %s", r.Agent, skillStatusLabel(r.Status), r.Path)))
	}
	if detected == 0 {
		_, _ = fmt.Fprintln(w, "  "+ui.Warn("No supported agent detected. Name one: issuegraph install "+strings.Join(setup.SkillAgentNames(), "|")))
		return nil
	}
	if setupCheck && stale {
		return SilentExit()
	}
	if !setupRemove && !setupCheck && !installDryRun {
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, ui.Hint("Run 'issuegraph init' in a repository to create its issue database."))
	}
	return nil
}

// skillStatusLabel words the status for the mode: --check reports state,
// --dry-run reports what would happen, and a real run reports what happened.
func skillStatusLabel(status string) string {
	switch {
	case setupCheck:
		switch status {
		case setup.SkillInstalled:
			return "not installed"
		case setup.SkillUpdated:
			return "outdated"
		}
	case installDryRun:
		switch status {
		case setup.SkillInstalled:
			return "would install"
		case setup.SkillUpdated:
			return "would update"
		case setup.SkillRemoved:
			return "would remove"
		}
	}
	return status
}
