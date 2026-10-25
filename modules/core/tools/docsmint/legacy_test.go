package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const legacyShowPage = `---
id: show
title: issuegraph show
slug: /cli-reference/show
sidebar_position: 40
---

<!-- AUTO-GENERATED: do not edit manually -->
Generated from ` + "`issuegraph help --doc show`" + `

## issuegraph show

Show issue details

### issuegraph show sub

Nested subcommand help.

` + "```" + `
### fenced heading stays put
` + "```" + `
`

const legacyIndexPage = `---
id: index
title: CLI Reference
sidebar_position: 0
---

# CLI Reference

<!-- AUTO-GENERATED: do not edit manually -->
Reference for issuegraph Latest. Generated from ` + "`issuegraph help --docs-root`" + `.

This reference covers all 2 live top-level ` + "`issuegraph`" + ` commands.

## Commands

- [` + "`issuegraph show`" + `](./show.md)
`

func TestConvertLegacyPageProducesGenericForm(t *testing.T) {
	got, err := convertLegacyPage(legacyShowPage)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"title: \"issuegraph show\"\n",
		"description: \"Show issue details\"\n",
		"Generated from `issuegraph help --doc show`.\n",
		"\n## issuegraph show sub\n",
		"\n### fenced heading stays put\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("converted page missing %q\n--- got ---\n%s", want, got)
		}
	}
	if strings.Contains(got, "sidebar_position") || strings.Contains(got, "slug:") {
		t.Errorf("legacy frontmatter leaked into converted page:\n%s", got)
	}
	if strings.Contains(got, "\n## issuegraph show\n") {
		t.Errorf("top command heading should be dropped:\n%s", got)
	}
}

func TestConvertLegacyIndexProducesGenericForm(t *testing.T) {
	got, err := convertLegacyIndex(legacyIndexPage)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"title: CLI Reference\n",
		"description: Generated reference for every issuegraph command\n",
		"Generated from `issuegraph help --docs-root`.\n",
		"- [`issuegraph show`](./show.md)",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("converted index missing %q\n--- got ---\n%s", want, got)
		}
	}
	if strings.Contains(got, "# CLI Reference\n\n<!--") {
		t.Errorf("H1 should be dropped (frontmatter title is the H1):\n%s", got)
	}
	if strings.Contains(got, "Reference for issuegraph Latest") {
		t.Errorf("versioned intro line should be replaced:\n%s", got)
	}
}

func TestConvertLegacyPageFailsOnUnexpectedShape(t *testing.T) {
	for name, content := range map[string]string{
		"no frontmatter":  "## issuegraph show\n",
		"no marker":       "---\nid: x\n---\n\nGenerated from `issuegraph help --doc x`\n\n## issuegraph x\n",
		"no command head": "---\nid: x\n---\n\n<!-- AUTO-GENERATED: do not edit manually -->\nGenerated from `issuegraph help --doc x`\n\nprose\n",
	} {
		if _, err := convertLegacyPage(content); err == nil {
			t.Errorf("%s: expected error, got none", name)
		}
	}
}

func TestRunConsumesLegacyStagingTree(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "website", "docs", "cli-reference", "index.md"), legacyIndexPage)
	writeFile(t, filepath.Join(root, "website", "docs", "cli-reference", "show.md"), legacyShowPage)
	writeFile(t, filepath.Join(root, "docs", "CLI_REFERENCE.md"), "# issuegraph — Complete Command Reference\n")
	writeFile(t, filepath.Join(root, "docs", "docs.json"), `{
  "navigation": {
    "groups": [
      {
        "group": "CLI Reference",
        "pages": [
          "cli-reference/index"
        ]
      }
    ]
  }
}
`)

	if err := run(root); err != nil {
		t.Fatal(err)
	}

	out, err := os.ReadFile(filepath.Join(root, "docs", "cli-reference", "show.md"))
	if err != nil {
		t.Fatal(err)
	}
	page := string(out)
	if !strings.Contains(page, "{/* AUTO-GENERATED: do not edit manually */}") {
		t.Errorf("legacy page should get the Mintlify comment transform:\n%s", page)
	}
	if !strings.Contains(page, "title: \"issuegraph show\"") {
		t.Errorf("legacy page should carry generic frontmatter:\n%s", page)
	}

	idx, err := os.ReadFile(filepath.Join(root, "docs", "cli-reference", "index.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(idx), "](/cli-reference/show)") {
		t.Errorf("legacy index links should become extensionless routes:\n%s", idx)
	}

	nav, err := os.ReadFile(filepath.Join(root, "docs", "docs.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(nav), `"cli-reference/show"`) {
		t.Errorf("nav should include the legacy-converted page:\n%s", nav)
	}
}
