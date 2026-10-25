package setup

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/borg0ai/dx-issuegraph/internal/templates"
)

var (
	errAgentSkillMissing = errors.New("issuegraph agent skill not installed")
	errAgentSkillStale   = errors.New("issuegraph agent skill is stale")
)

type agentSkillEnv struct {
	stdout     io.Writer
	stderr     io.Writer
	projectDir string
	ensureDir  func(string, os.FileMode) error
	readFile   func(string) ([]byte, error)
	writeFile  func(string, []byte) error
	removeFile func(string) error
}

func agentSkillRootPath(base string) string {
	return filepath.Join(base, ".agents", "skills", "issuegraph")
}

func agentSkillPath(base string) string {
	return filepath.Join(agentSkillRootPath(base), "SKILL.md")
}

func agentSkillOpenAIYAMLPath(base string) string {
	return filepath.Join(agentSkillRootPath(base), "agents", "openai.yaml")
}

func installAgentSkill(env agentSkillEnv) error {
	_, _ = fmt.Fprintln(env.stdout, "Installing IssueGraph agent skill...")

	skillPath := agentSkillPath(env.projectDir)
	if err := env.ensureDir(filepath.Dir(skillPath), 0o755); err != nil {
		_, _ = fmt.Fprintf(env.stderr, "Error: %v\n", err)
		return err
	}
	if err := env.writeFile(skillPath, []byte(templates.IssueGraphAgentSkill())); err != nil {
		_, _ = fmt.Fprintf(env.stderr, "Error: write skill: %v\n", err)
		return err
	}

	openAIYAMLPath := agentSkillOpenAIYAMLPath(env.projectDir)
	if err := env.ensureDir(filepath.Dir(openAIYAMLPath), 0o755); err != nil {
		_, _ = fmt.Fprintf(env.stderr, "Error: %v\n", err)
		return err
	}
	if err := env.writeFile(openAIYAMLPath, []byte(templates.IssueGraphAgentSkillOpenAIYAML())); err != nil {
		_, _ = fmt.Fprintf(env.stderr, "Error: write skill metadata: %v\n", err)
		return err
	}

	_, _ = fmt.Fprintln(env.stdout, "✓ IssueGraph agent skill installed")
	_, _ = fmt.Fprintf(env.stdout, "  Skill: %s\n", skillPath)
	return nil
}

func checkAgentSkill(env agentSkillEnv, setupCommand string) error {
	skillPath := agentSkillPath(env.projectDir)
	data, err := env.readFile(skillPath)
	if os.IsNotExist(err) {
		_, _ = fmt.Fprintf(env.stdout, "✗ IssueGraph agent skill not found: %s\n", skillPath)
		_, _ = fmt.Fprintf(env.stdout, "  Run: %s\n", setupCommand)
		return errAgentSkillMissing
	}
	if err != nil {
		_, _ = fmt.Fprintf(env.stderr, "Error: read skill: %v\n", err)
		return err
	}
	if string(data) != templates.IssueGraphAgentSkill() {
		_, _ = fmt.Fprintf(env.stdout, "⚠ IssueGraph agent skill installed but stale: %s\n", skillPath)
		_, _ = fmt.Fprintf(env.stdout, "  Run: %s\n", setupCommand)
		return errAgentSkillStale
	}

	openAIYAMLPath := agentSkillOpenAIYAMLPath(env.projectDir)
	data, err = env.readFile(openAIYAMLPath)
	if os.IsNotExist(err) {
		_, _ = fmt.Fprintf(env.stdout, "✗ IssueGraph agent skill metadata not found: %s\n", openAIYAMLPath)
		_, _ = fmt.Fprintf(env.stdout, "  Run: %s\n", setupCommand)
		return errAgentSkillMissing
	}
	if err != nil {
		_, _ = fmt.Fprintf(env.stderr, "Error: read skill metadata: %v\n", err)
		return err
	}
	if string(data) != templates.IssueGraphAgentSkillOpenAIYAML() {
		_, _ = fmt.Fprintf(env.stdout, "⚠ IssueGraph agent skill metadata is stale: %s\n", openAIYAMLPath)
		_, _ = fmt.Fprintf(env.stdout, "  Run: %s\n", setupCommand)
		return errAgentSkillStale
	}

	_, _ = fmt.Fprintf(env.stdout, "✓ IssueGraph agent skill installed: %s\n", skillPath)
	return nil
}

func removeAgentSkill(env agentSkillEnv) error {
	_, _ = fmt.Fprintln(env.stdout, "Removing IssueGraph agent skill...")
	for _, path := range []string{
		agentSkillOpenAIYAMLPath(env.projectDir),
		agentSkillPath(env.projectDir),
	} {
		if err := env.removeFile(path); err != nil && !os.IsNotExist(err) {
			_, _ = fmt.Fprintf(env.stderr, "Error: remove %s: %v\n", path, err)
			return err
		}
	}
	for _, dir := range []string{
		filepath.Join(env.projectDir, ".agents", "skills", "issuegraph", "agents"),
		filepath.Join(env.projectDir, ".agents", "skills", "issuegraph"),
		filepath.Join(env.projectDir, ".agents", "skills"),
		filepath.Join(env.projectDir, ".agents"),
	} {
		entries, err := os.ReadDir(dir) // #nosec G304 -- project-local skill path
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		if len(entries) != 0 {
			continue
		}
		if err := env.removeFile(dir); err != nil && !os.IsNotExist(err) {
			_, _ = fmt.Fprintf(env.stderr, "Error: remove %s: %v\n", dir, err)
			return err
		}
	}
	_, _ = fmt.Fprintln(env.stdout, "✓ IssueGraph agent skill removed")
	return nil
}
