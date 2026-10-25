package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadIssueGraphEnvFile(t *testing.T) {
	t.Run("loads env vars from .env file", func(t *testing.T) {
		dir := t.TempDir()
		envFile := filepath.Join(dir, ".env")
		if err := os.WriteFile(envFile, []byte("ISSUEGRAPH_TEST_LOAD_VAR=hello_from_env\n"), 0600); err != nil {
			t.Fatal(err)
		}
		t.Setenv("ISSUEGRAPH_TEST_LOAD_VAR", "") // clear
		os.Unsetenv("ISSUEGRAPH_TEST_LOAD_VAR")

		loadIssueGraphEnvFile(dir)

		if got := os.Getenv("ISSUEGRAPH_TEST_LOAD_VAR"); got != "hello_from_env" {
			t.Errorf("expected ISSUEGRAPH_TEST_LOAD_VAR=hello_from_env, got %q", got)
		}
		os.Unsetenv("ISSUEGRAPH_TEST_LOAD_VAR")
	})

	t.Run("shell env takes precedence over .env", func(t *testing.T) {
		dir := t.TempDir()
		envFile := filepath.Join(dir, ".env")
		if err := os.WriteFile(envFile, []byte("ISSUEGRAPH_TEST_PRECEDENCE=from_file\n"), 0600); err != nil {
			t.Fatal(err)
		}
		t.Setenv("ISSUEGRAPH_TEST_PRECEDENCE", "from_shell")

		loadIssueGraphEnvFile(dir)

		if got := os.Getenv("ISSUEGRAPH_TEST_PRECEDENCE"); got != "from_shell" {
			t.Errorf("expected shell env to win, got %q", got)
		}
	})

	t.Run("no-op when .env does not exist", func(t *testing.T) {
		dir := t.TempDir()
		// Should not panic or error
		loadIssueGraphEnvFile(dir)
	})

	t.Run("no-op when issuegraphDir is empty", func(t *testing.T) {
		// Should not panic or error
		loadIssueGraphEnvFile("")
	})
}
