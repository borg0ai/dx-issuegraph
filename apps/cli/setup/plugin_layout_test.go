package setup

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestPluginLayoutUsesSharedIssueGraphRoot(t *testing.T) {
	root := filepath.Join("..", "..", "..")

	var claudeMarketplace struct {
		Plugins []struct {
			Source string `json:"source"`
		} `json:"plugins"`
	}
	readJSONFile(t, filepath.Join(root, ".claude-plugin", "marketplace.json"), &claudeMarketplace)
	if len(claudeMarketplace.Plugins) != 1 {
		t.Fatalf("expected one Claude marketplace plugin, got %d", len(claudeMarketplace.Plugins))
	}
	if got := claudeMarketplace.Plugins[0].Source; got != "./issuegraph" {
		t.Fatalf("Claude marketplace source = %q, want ./issuegraph", got)
	}

	var claudeManifest struct {
		Skills   string  `json:"skills"`
		Commands string  `json:"commands"`
		Agents   *string `json:"agents"`
	}
	readJSONFile(t, filepath.Join(root, "issuegraph", ".claude-plugin", "plugin.json"), &claudeManifest)
	if claudeManifest.Skills != "./skills/" {
		t.Fatalf("Claude skills path = %q, want ./skills/", claudeManifest.Skills)
	}
	if claudeManifest.Commands != "./skills/issuegraph/commands/" {
		t.Fatalf("Claude commands path = %q, want ./skills/issuegraph/commands/", claudeManifest.Commands)
	}
	if claudeManifest.Agents != nil {
		t.Fatalf("Claude agents path = %q, want unset (default ./agents/) so the loader does not scan codex yaml as agents", *claudeManifest.Agents)
	}

	var codexManifest struct {
		Skills string `json:"skills"`
		Hooks  string `json:"hooks"`
	}
	readJSONFile(t, filepath.Join(root, "issuegraph", ".codex-plugin", "plugin.json"), &codexManifest)
	if codexManifest.Skills != "./skills/" {
		t.Fatalf("Codex manifest skills path = %q, want ./skills/", codexManifest.Skills)
	}
	if codexManifest.Hooks != "./.codex-plugin/hooks/hooks.json" {
		t.Fatalf("Codex manifest hooks path = %q, want ./.codex-plugin/hooks/hooks.json", codexManifest.Hooks)
	}

	requireRepoFile(t, root, "issuegraph", "skills", "issuegraph", "SKILL.md")
	requireRepoFile(t, root, "issuegraph", "skills", "issuegraph", "agents", "openai.yaml")
	requireRepoFile(t, root, "issuegraph", "agents", "task-agent.md")
	requireRepoFile(t, root, "issuegraph", "skills", "issuegraph", "commands", "ready.md")
	requireRepoFile(t, root, "issuegraph", ".codex-plugin", "hooks", "hooks.json")
	requireNoRepoPath(t, root, "issuegraph", "hooks", "hooks.json")
}

func TestPluginCommandArgumentHintsAreStrings(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	commandsDir := filepath.Join(root, "issuegraph", "skills", "issuegraph", "commands")
	entries, err := os.ReadDir(commandsDir)
	if err != nil {
		t.Fatalf("read commands directory %s: %v", commandsDir, err)
	}

	const expectedArgumentHints = 23
	checkedArgumentHints := 0
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}

		path := filepath.Join(commandsDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
		if !bytes.HasPrefix(data, []byte("---\n")) {
			continue
		}

		frontmatter, _, ok := bytes.Cut(data[len("---\n"):], []byte("\n---\n"))
		if !ok {
			t.Fatalf("parse frontmatter in %s: missing closing delimiter", path)
		}

		var metadata map[string]interface{}
		if err := yaml.Unmarshal(frontmatter, &metadata); err != nil {
			t.Fatalf("parse frontmatter in %s: %v", path, err)
		}
		argumentHint, ok := metadata["argument-hint"]
		if !ok {
			continue
		}
		checkedArgumentHints++
		if _, ok := argumentHint.(string); !ok {
			t.Errorf("argument-hint in %s decoded as %T, want string", path, argumentHint)
		}
	}
	if checkedArgumentHints != expectedArgumentHints {
		t.Errorf("checked %d command argument-hint fields, want %d", checkedArgumentHints, expectedArgumentHints)
	}
}

func readJSONFile(t *testing.T, path string, dest interface{}) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if err := json.Unmarshal(data, dest); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
}

func requireRepoFile(t *testing.T, root string, parts ...string) {
	t.Helper()
	path := filepath.Join(append([]string{root}, parts...)...)
	if info, err := os.Stat(path); err != nil {
		t.Fatalf("expected file %s: %v", path, err)
	} else if info.IsDir() {
		t.Fatalf("expected file %s, got directory", path)
	}
}

func requireNoRepoPath(t *testing.T, root string, parts ...string) {
	t.Helper()
	path := filepath.Join(append([]string{root}, parts...)...)
	if _, err := os.Stat(path); err == nil {
		t.Fatalf("expected path %s not to exist", path)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat %s: %v", path, err)
	}
}
