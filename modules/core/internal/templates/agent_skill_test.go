package templates

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSkillFilesMatchSource fails when the generated assets are stale.
// Fix it with: pnpm run generate:assets
func TestSkillFilesMatchSource(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..", "issuegraph", "skills", "issuegraph")
	want := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		if rel == "README.md" || rel == "CLAUDE.md" || strings.HasPrefix(rel, "adr/") || d.Name() == ".DS_Store" {
			return nil
		}
		data, err := os.ReadFile(path)
		want[rel] = string(data)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	got := IssueGraphSkillFiles()
	for rel, content := range want {
		if got[rel] != content {
			t.Errorf("%s is stale or missing in generated assets; run: pnpm run generate:assets", rel)
		}
	}
	for rel := range got {
		if _, ok := want[rel]; !ok {
			t.Errorf("%s is embedded but no longer in the skill source; run: pnpm run generate:assets", rel)
		}
	}
	if _, ok := got["SKILL.md"]; !ok || len(got) < 20 {
		t.Errorf("expected the full skill tree, got %d files", len(got))
	}
}
