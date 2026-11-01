package setup

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Skill install statuses reported per agent.
const (
	SkillInstalled = "installed"
	SkillUpdated   = "updated"
	SkillUnchanged = "unchanged"
	SkillRemoved   = "removed"
	SkillMissing   = "missing"
	SkillSkipped   = "skipped"
)

const skillDirName = "issuegraph"

// SkillAgent describes where one coding agent keeps its global skills.
type SkillAgent struct {
	Name      string
	DetectDir string // agent config directory; its existence means the agent is installed
	SkillsDir string // global skills directory; the skill goes in SkillsDir/issuegraph
}

// SkillOptions controls InstallSkills.
type SkillOptions struct {
	Home   string
	Getenv func(string) string
	Files  map[string]string // slash-separated relative path -> content
	Agents []string          // explicit agents; empty means every detected agent
	DryRun bool
	Remove bool
}

// SkillResult is the outcome for one agent.
type SkillResult struct {
	Agent  string `json:"agent"`
	Path   string `json:"path"`
	Status string `json:"status"`
}

func envOr(getenv func(string) string, key, fallback string) string {
	if v := strings.TrimSpace(getenv(key)); v != "" {
		return v
	}
	return fallback
}

// SkillAgents lists the agents that support a global skills directory.
// Paths follow the layout used by the agents' own documentation and the
// vercel-labs/skills installer.
func SkillAgents(home string, getenv func(string) string) []SkillAgent {
	claude := envOr(getenv, "CLAUDE_CONFIG_DIR", filepath.Join(home, ".claude"))
	codex := envOr(getenv, "CODEX_HOME", filepath.Join(home, ".codex"))
	hermes := envOr(getenv, "HERMES_HOME", filepath.Join(home, ".hermes"))
	antigravity := filepath.Join(home, ".gemini", "antigravity")
	pi := filepath.Join(home, ".pi", "agent")
	gemini := filepath.Join(home, ".gemini")
	return []SkillAgent{
		{"claude", claude, filepath.Join(claude, "skills")},
		{"codex", codex, filepath.Join(codex, "skills")},
		{"gemini", gemini, filepath.Join(gemini, "skills")},
		{"hermes", hermes, filepath.Join(hermes, "skills")},
		{"antigravity", antigravity, filepath.Join(antigravity, "skills")},
		{"pi", pi, filepath.Join(pi, "skills")},
	}
}

// SkillAgentNames returns the supported agent names in install order.
func SkillAgentNames() []string {
	agents := SkillAgents("", func(string) string { return "" })
	names := make([]string, len(agents))
	for i, a := range agents {
		names[i] = a.Name
	}
	return names
}

// InstallSkills installs (or removes) the IssueGraph skill for each selected agent.
func InstallSkills(opts SkillOptions) ([]SkillResult, error) {
	agents := SkillAgents(opts.Home, opts.Getenv)
	explicit := len(opts.Agents) > 0
	selected := map[string]bool{}
	for _, name := range opts.Agents {
		name = strings.ToLower(strings.TrimSpace(name))
		found := false
		for _, a := range agents {
			if a.Name == name {
				found = true
			}
		}
		if !found {
			return nil, fmt.Errorf("unknown agent %q (supported: %s)", name, strings.Join(SkillAgentNames(), ", "))
		}
		selected[name] = true
	}

	var results []SkillResult
	for _, a := range agents {
		dest := filepath.Join(a.SkillsDir, skillDirName)
		if explicit && !selected[a.Name] {
			continue
		}
		if !explicit {
			if info, err := os.Stat(a.DetectDir); err != nil || !info.IsDir() {
				results = append(results, SkillResult{a.Name, dest, SkillSkipped})
				continue
			}
		}
		var status string
		var err error
		if opts.Remove {
			status, err = removeSkill(dest, opts.DryRun)
		} else {
			status, err = writeSkill(dest, opts.Files, opts.DryRun)
		}
		if err != nil {
			return results, fmt.Errorf("%s: %w", a.Name, err)
		}
		results = append(results, SkillResult{a.Name, dest, status})
	}
	return results, nil
}

func removeSkill(dest string, dryRun bool) (string, error) {
	if _, err := os.Stat(dest); os.IsNotExist(err) {
		return SkillMissing, nil
	}
	if !dryRun {
		if err := os.RemoveAll(dest); err != nil {
			return "", err
		}
	}
	return SkillRemoved, nil
}

func writeSkill(dest string, files map[string]string, dryRun bool) (string, error) {
	rels := make([]string, 0, len(files))
	for rel := range files {
		rels = append(rels, rel)
	}
	sort.Strings(rels)
	created, changed := 0, 0
	for _, rel := range rels {
		path := filepath.Join(dest, filepath.FromSlash(rel))
		want := []byte(files[rel])
		cur, err := os.ReadFile(path)
		switch {
		case os.IsNotExist(err):
			created++
		case err != nil:
			return "", err
		case !bytes.Equal(cur, want):
			changed++
		default:
			continue
		}
		if dryRun {
			continue
		}
		if err := EnsureDir(filepath.Dir(path), 0o755); err != nil {
			return "", err
		}
		if err := atomicWriteFile(path, want, 0o644); err != nil {
			return "", err
		}
	}
	switch {
	case created == len(rels) && created > 0:
		return SkillInstalled, nil
	case created == 0 && changed == 0:
		return SkillUnchanged, nil
	default:
		return SkillUpdated, nil
	}
}
