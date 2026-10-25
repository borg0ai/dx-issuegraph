package scripts_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func sourceRepoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Dir(filepath.Dir(file))
}

func msysPath(path string) string {
	path = filepath.Clean(path)
	path = filepath.ToSlash(path)
	if len(path) >= 3 && path[1] == ':' && path[2] == '/' {
		return "/" + strings.ToLower(path[:1]) + path[2:]
	}
	return path
}

func shellPathUnderEnv(t *testing.T, bash, path string, env []string) string {
	t.Helper()
	clean := filepath.Clean(path)
	dir := clean
	base := ""
	if info, err := os.Stat(clean); err == nil && !info.IsDir() {
		dir = filepath.Dir(clean)
		base = filepath.Base(clean)
	}
	cmd := exec.Command(bash, "--noprofile", "--norc", "-c", "pwd")
	cmd.Dir = dir
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("resolve shell path for %s under script environment: %v\n%s", path, err, out)
	}
	converted := strings.TrimSpace(string(out))
	if converted == "" {
		t.Fatalf("resolve shell path for %s under script environment: empty output", path)
	}
	if base != "" {
		return converted + "/" + filepath.ToSlash(base)
	}
	return converted
}

// shellPathEnv keeps minimal-environment path conversion independent of
// inherited temporary-path mounts. Git for Windows can map /tmp differently
// when TEMP/TMP are omitted, so helpers that replace the child environment
// must resolve paths under that same authority.
func shellPathEnv() []string {
	path := os.Getenv("PATH")
	if runtime.GOOS == "windows" {
		path = "/usr/bin:/bin"
	}
	return []string{
		"PATH=" + path,
		"LC_ALL=C",
		"LANG=C",
		"BASH_ENV=",
		"ENV=",
	}
}

func requireShellCommandPath(t *testing.T, bash, dir string, env []string, name, want string) {
	t.Helper()
	cmd := exec.Command(bash, "--noprofile", "--norc", "-c", `PATH="$ISSUEGRAPH_TEST_COMMAND_PATH"; export PATH; printf '%s\n' "$PATH"; command -v -- "$1"`, "shim-path-check", name)
	cmd.Dir = dir
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("resolve injected %s under script environment: %v\n%s", name, err, out)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	got := lines[len(lines)-1]
	if got != want {
		t.Fatalf("injected %s resolved to %q, want %q; PATH=%q", name, got, want, lines[0])
	}
}

func bashScriptCommand(bash, script string) *exec.Cmd {
	return exec.Command(bash, "--noprofile", "--norc", "-c", `PATH="$ISSUEGRAPH_TEST_COMMAND_PATH"; export PATH; exec "$BASH" --noprofile --norc "$1"`, "script-test", script)
}

func shSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}

func writeExecutable(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS == "windows" {
		chmodPath := shellPath(t, path)
		if wd, err := os.Getwd(); err == nil {
			if rel, relErr := filepath.Rel(wd, path); relErr == nil {
				chmodPath = filepath.ToSlash(rel)
			}
		}
		cmd := exec.Command("bash", "-lc", "chmod +x "+shSingleQuote(chmodPath))
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("chmod +x %s failed: %v\n%s", path, err, out)
		}
	}
}

func shellPath(t *testing.T, path string) string {
	t.Helper()
	clean := filepath.Clean(path)
	dir := clean
	base := ""
	if info, err := os.Stat(clean); err == nil && !info.IsDir() {
		dir = filepath.Dir(clean)
		base = filepath.Base(clean)
	}
	cmd := exec.Command("bash", "-lc", "pwd")
	cmd.Dir = dir
	if out, err := cmd.Output(); err == nil {
		converted := strings.TrimSpace(string(out))
		if converted != "" {
			if base != "" {
				return converted + "/" + filepath.ToSlash(base)
			}
			return converted
		}
	}
	for _, tool := range []string{"wslpath", "cygpath"} {
		out, err := exec.Command(tool, "-u", path).Output()
		if err == nil {
			converted := strings.TrimSpace(string(out))
			if converted != "" {
				return converted
			}
		}
	}
	return msysPath(path)
}
