package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/borg0ai/dx-issuegraph/internal/configfile"
)

func TestRunServerHealthChecksSQLiteIsNotAMigrationWarning(t *testing.T) {
	tmpDir := t.TempDir()
	issuegraphDir := filepath.Join(tmpDir, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := (&configfile.Config{Backend: configfile.BackendSQLite, SQLitePath: "issuegraph.db"}).Save(issuegraphDir); err != nil {
		t.Fatalf("save SQLite config: %v", err)
	}

	// The sqlite backend is a removed-backend tombstone: server checks report a
	// clearly-attributed non-Dolt warning with no migration fix, and never edit
	// the workspace.
	result := RunServerHealthChecks(tmpDir)
	if result.OverallOK || len(result.Checks) != 1 {
		t.Fatalf("SQLite server-health result = %#v, want one non-Dolt warning", result)
	}
	check := result.Checks[0]
	if check.Status != StatusWarning || !strings.Contains(check.Message, "sqlite") || check.Fix != "" {
		t.Fatalf("SQLite server-health check = %#v, want non-Dolt warning without migration fix", check)
	}
}
