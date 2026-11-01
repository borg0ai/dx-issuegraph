package setup

import (
	"os"
	"path/filepath"
	"testing"
)

var testSkillFiles = map[string]string{
	"SKILL.md":            "# issuegraph skill\n",
	"agents/openai.yaml":  "name: issuegraph\n",
	"commands/ready.md":   "ready\n",
	"resources/AGENTS.md": "agents\n",
}

func skillHome(t *testing.T, dirs ...string) string {
	t.Helper()
	home := t.TempDir()
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(home, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return home
}

func noEnv(string) string { return "" }

func statusOf(results []SkillResult, agent string) string {
	for _, r := range results {
		if r.Agent == agent {
			return r.Status
		}
	}
	return "absent"
}

func TestInstallSkillsDetectsAndWritesFullTree(t *testing.T) {
	home := skillHome(t, ".claude", ".codex", ".gemini")
	results, err := InstallSkills(SkillOptions{Home: home, Getenv: noEnv, Files: testSkillFiles})
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range []string{"claude", "codex", "gemini"} {
		if got := statusOf(results, a); got != SkillInstalled {
			t.Errorf("%s status = %q, want installed", a, got)
		}
	}
	for _, a := range []string{"hermes", "pi", "antigravity"} {
		if got := statusOf(results, a); got != SkillSkipped {
			t.Errorf("%s status = %q, want skipped", a, got)
		}
	}
	for rel, want := range testSkillFiles {
		data, err := os.ReadFile(filepath.Join(home, ".claude", "skills", "issuegraph", filepath.FromSlash(rel)))
		if err != nil || string(data) != want {
			t.Errorf("claude %s = %q, %v; want %q", rel, data, err, want)
		}
	}
	if _, err := os.Stat(filepath.Join(home, ".pi")); !os.IsNotExist(err) {
		t.Error("skipped agent directory must not be created")
	}
}

func TestInstallSkillsIsIdempotentAndDetectsUpdates(t *testing.T) {
	home := skillHome(t, ".claude")
	opts := SkillOptions{Home: home, Getenv: noEnv, Files: testSkillFiles}
	if _, err := InstallSkills(opts); err != nil {
		t.Fatal(err)
	}
	results, err := InstallSkills(opts)
	if err != nil {
		t.Fatal(err)
	}
	if got := statusOf(results, "claude"); got != SkillUnchanged {
		t.Errorf("second run = %q, want unchanged", got)
	}
	changed := map[string]string{}
	for k, v := range testSkillFiles {
		changed[k] = v
	}
	changed["SKILL.md"] = "# new\n"
	opts.Files = changed
	results, err = InstallSkills(opts)
	if err != nil {
		t.Fatal(err)
	}
	if got := statusOf(results, "claude"); got != SkillUpdated {
		t.Errorf("changed content = %q, want updated", got)
	}
}

func TestInstallSkillsDryRunWritesNothing(t *testing.T) {
	home := skillHome(t, ".claude")
	results, err := InstallSkills(SkillOptions{Home: home, Getenv: noEnv, Files: testSkillFiles, DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if got := statusOf(results, "claude"); got != SkillInstalled {
		t.Errorf("dry-run status = %q, want installed", got)
	}
	if _, err := os.Stat(filepath.Join(home, ".claude", "skills")); !os.IsNotExist(err) {
		t.Error("dry-run must not create the skills directory")
	}
}

func TestInstallSkillsExplicitAgentCreatesDirectories(t *testing.T) {
	home := skillHome(t)
	results, err := InstallSkills(SkillOptions{Home: home, Getenv: noEnv, Files: testSkillFiles, Agents: []string{"pi", "antigravity"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 || statusOf(results, "pi") != SkillInstalled || statusOf(results, "antigravity") != SkillInstalled {
		t.Fatalf("results = %+v", results)
	}
	for _, p := range []string{".pi/agent/skills/issuegraph/SKILL.md", ".gemini/antigravity/skills/issuegraph/SKILL.md"} {
		if _, err := os.Stat(filepath.Join(home, filepath.FromSlash(p))); err != nil {
			t.Errorf("missing %s: %v", p, err)
		}
	}
}

func TestInstallSkillsUnknownAgentFails(t *testing.T) {
	_, err := InstallSkills(SkillOptions{Home: skillHome(t), Getenv: noEnv, Files: testSkillFiles, Agents: []string{"nope"}})
	if err == nil {
		t.Fatal("unknown agent must fail")
	}
}

func TestInstallSkillsRemoveOnlyDeletesIssueGraph(t *testing.T) {
	home := skillHome(t, ".claude/skills/other")
	opts := SkillOptions{Home: home, Getenv: noEnv, Files: testSkillFiles}
	if _, err := InstallSkills(opts); err != nil {
		t.Fatal(err)
	}
	opts.Remove = true
	results, err := InstallSkills(opts)
	if err != nil {
		t.Fatal(err)
	}
	if got := statusOf(results, "claude"); got != SkillRemoved {
		t.Errorf("remove status = %q", got)
	}
	if _, err := os.Stat(filepath.Join(home, ".claude", "skills", "issuegraph")); !os.IsNotExist(err) {
		t.Error("issuegraph skill directory must be gone")
	}
	if _, err := os.Stat(filepath.Join(home, ".claude", "skills", "other")); err != nil {
		t.Error("sibling skill must stay")
	}
}

func TestInstallSkillsHonorsConfigHomeEnv(t *testing.T) {
	home := skillHome(t)
	custom := t.TempDir()
	env := map[string]string{"CLAUDE_CONFIG_DIR": custom, "CODEX_HOME": filepath.Join(custom, "codex"), "HERMES_HOME": filepath.Join(custom, "hermes")}
	getenv := func(k string) string { return env[k] }
	if err := os.MkdirAll(filepath.Join(custom, "codex"), 0o755); err != nil {
		t.Fatal(err)
	}
	results, err := InstallSkills(SkillOptions{Home: home, Getenv: getenv, Files: testSkillFiles})
	if err != nil {
		t.Fatal(err)
	}
	if statusOf(results, "claude") != SkillInstalled || statusOf(results, "codex") != SkillInstalled || statusOf(results, "hermes") != SkillSkipped {
		t.Fatalf("results = %+v", results)
	}
	if _, err := os.Stat(filepath.Join(custom, "skills", "issuegraph", "SKILL.md")); err != nil {
		t.Errorf("CLAUDE_CONFIG_DIR not honored: %v", err)
	}
}
