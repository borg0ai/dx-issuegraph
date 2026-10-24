//go:build integration

package issuegraph

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/borg0ai/dx-issuegraph/internal/testutil"
)

func setupIntegrationTestMain(root string) (func(), error) {
	cleanup := func() {
		os.Unsetenv("ISSUEGRAPH_DOLT_PORT")
		os.Unsetenv("ISSUEGRAPH_TEST_ISSUEGRAPH_BINARY")
		os.Unsetenv("ISSUEGRAPH_TEST_MODE")
		os.Unsetenv("ISSUEGRAPH_TEST_SERVER")
	}

	os.Setenv("ISSUEGRAPH_TEST_MODE", "1")
	// AD-01 (be-c5p): allow integration tests to connect to the test container.
	os.Setenv("ISSUEGRAPH_TEST_SERVER", "1")

	if err := testutil.EnsureDoltContainerForTestMain(); err != nil {
		fmt.Fprintf(os.Stderr, "WARN: %v, skipping Dolt tests\n", err)
		return cleanup, nil
	}

	binName := "issuegraph"
	if runtime.GOOS == "windows" {
		binName = "issuegraph.exe"
	}

	modRootCmd := exec.Command("go", "list", "-m", "-f", "{{.Dir}}")
	modRootOut, err := modRootCmd.Output()
	if err != nil {
		testutil.TerminateDoltContainer()
		cleanup()
		return nil, fmt.Errorf("find module root: %w", err)
	}
	modRoot := strings.TrimSpace(string(modRootOut))

	testIssueGraphBinary := filepath.Join(root, binName)
	cmd := exec.Command("go", "build", "-tags", "gms_pure_go", "-o", testIssueGraphBinary, "./modules/cli")
	cmd.Dir = modRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		testutil.TerminateDoltContainer()
		cleanup()
		return nil, fmt.Errorf("build issuegraph binary: %w\n%s", err, out)
	}
	os.Setenv("ISSUEGRAPH_TEST_ISSUEGRAPH_BINARY", testIssueGraphBinary)

	return func() {
		testutil.TerminateDoltContainer()
		cleanup()
	}, nil
}
