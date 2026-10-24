package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/borg0ai/dx-issuegraph/internal/configfile"
	"github.com/borg0ai/dx-issuegraph/internal/doltserver"
)

func TestCheckManagedHandoffPortWarnsOnManagedPortConflict(t *testing.T) {
	clearResolveIssueGraphDirCache()
	t.Cleanup(clearResolveIssueGraphDirCache)

	repoDir := t.TempDir()
	issuegraphDir := filepath.Join(repoDir, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0o755); err != nil {
		t.Fatalf("mkdir .issuegraph: %v", err)
	}
	if err := (&configfile.Config{}).Save(issuegraphDir); err != nil {
		t.Fatalf("save metadata: %v", err)
	}
	if err := os.WriteFile(filepath.Join(issuegraphDir, doltserver.PortFileName), []byte("37953"), 0o600); err != nil {
		t.Fatalf("write port file: %v", err)
	}

	t.Setenv("ISSUEGRAPH_DOLT_PORT", "54418")
	t.Setenv("ISSUEGRAPH_DOLT_SERVER_PORT", "")
	t.Setenv("GT_ROOT", filepath.Join(repoDir, "city"))

	check := CheckManagedHandoffPort(repoDir)
	if check.Status != StatusWarning {
		t.Fatalf("expected warning, got %s: %s", check.Status, check.Message)
	}
	for _, want := range []string{
		"managed Dolt port override differs",
		"ISSUEGRAPH_DOLT_PORT=54418",
		"contains 37953",
		"GT_ROOT=",
		"standalone store",
	} {
		if !strings.Contains(check.Message+check.Detail+check.Fix, want) {
			t.Fatalf("check missing %q:\n%+v", want, check)
		}
	}
}

func TestCheckManagedHandoffPortOKWithoutConflict(t *testing.T) {
	clearResolveIssueGraphDirCache()
	t.Cleanup(clearResolveIssueGraphDirCache)

	repoDir := t.TempDir()
	issuegraphDir := filepath.Join(repoDir, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0o755); err != nil {
		t.Fatalf("mkdir .issuegraph: %v", err)
	}
	if err := (&configfile.Config{}).Save(issuegraphDir); err != nil {
		t.Fatalf("save metadata: %v", err)
	}
	if err := os.WriteFile(filepath.Join(issuegraphDir, doltserver.PortFileName), []byte("37953"), 0o600); err != nil {
		t.Fatalf("write port file: %v", err)
	}

	t.Setenv("ISSUEGRAPH_DOLT_PORT", "37953")
	t.Setenv("ISSUEGRAPH_DOLT_SERVER_PORT", "")

	check := CheckManagedHandoffPort(repoDir)
	if check.Status != StatusOK {
		t.Fatalf("expected ok, got %s: %s", check.Status, check.Message)
	}
}
