package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func resetInstallFlags(t *testing.T) {
	t.Helper()
	oldDry, oldCheck, oldRemove, oldJSON := installDryRun, setupCheck, setupRemove, jsonOutput
	installDryRun, setupCheck, setupRemove, jsonOutput = false, false, false, false
	t.Cleanup(func() { installDryRun, setupCheck, setupRemove, jsonOutput = oldDry, oldCheck, oldRemove, oldJSON })
}

func TestInstallSkillsCommandInstallsFullSkillAndSkipsMissingAgents(t *testing.T) {
	resetInstallFlags(t)
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := installSkills(&out, home, func(string) string { return "" }, nil); err != nil {
		t.Fatalf("installSkills: %v", err)
	}
	for _, rel := range []string{"SKILL.md", "agents/openai.yaml", "commands/ready.md", "resources/CLI_REFERENCE.md"} {
		if _, err := os.Stat(filepath.Join(home, ".claude", "skills", "issuegraph", filepath.FromSlash(rel))); err != nil {
			t.Errorf("full skill must include %s: %v", rel, err)
		}
	}
	text := out.String()
	for _, want := range []string{"claude", "installed", "pi", "skipped (not detected)", "issuegraph init"} {
		if !strings.Contains(text, want) {
			t.Errorf("output missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "\x1b[") {
		t.Errorf("non-TTY output must have no ANSI: %q", text)
	}
}

func TestInstallSkillsCommandCheckFailsWhenMissingAndPassesWhenCurrent(t *testing.T) {
	resetInstallFlags(t)
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".codex"), 0o755); err != nil {
		t.Fatal(err)
	}
	getenv := func(string) string { return "" }
	setupCheck = true
	var checkOut bytes.Buffer
	if err := installSkills(&checkOut, home, getenv, nil); err == nil {
		t.Fatal("--check must fail when the skill is not installed")
	}
	if !strings.Contains(checkOut.String(), "not installed") {
		t.Errorf("--check must say not installed, got:\n%s", checkOut.String())
	}
	if _, err := os.Stat(filepath.Join(home, ".codex", "skills")); !os.IsNotExist(err) {
		t.Fatal("--check must not write files")
	}
	setupCheck = false
	if err := installSkills(&bytes.Buffer{}, home, getenv, nil); err != nil {
		t.Fatal(err)
	}
	setupCheck = true
	if err := installSkills(&bytes.Buffer{}, home, getenv, nil); err != nil {
		t.Fatalf("--check must pass when current: %v", err)
	}
}

func TestInstallSkillsCommandNoAgentDetected(t *testing.T) {
	resetInstallFlags(t)
	var out bytes.Buffer
	if err := installSkills(&out, t.TempDir(), func(string) string { return "" }, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "No supported agent detected") {
		t.Errorf("want guidance, got:\n%s", out.String())
	}
}
