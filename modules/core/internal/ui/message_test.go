package ui

import (
	"strings"
	"testing"
)

// forceRich makes message helpers emit styled output for one test.
func forceRich(t *testing.T) {
	t.Helper()
	old := richOutput
	richOutput = func() bool { return true }
	initColors(true)
	initStyles()
	t.Cleanup(func() {
		richOutput = old
		DisableColors()
	})
}

func forcePlain(t *testing.T) {
	t.Helper()
	old := richOutput
	richOutput = func() bool { return false }
	t.Cleanup(func() { richOutput = old })
}

func TestMessagePlainKeepsExistingText(t *testing.T) {
	forcePlain(t)
	cases := map[string]string{
		"Banner":  Banner("IssueGraph Info"),
		"Section": Section("Schema Information"),
		"Field":   Field("Mode", "direct"),
		"Success": Success("done"),
		"Warn":    Warn("careful"),
		"Error":   Error("broken"),
		"Info":    Info("fyi"),
		"Hint":    Hint("try this"),
	}
	want := map[string]string{
		"Banner":  "IssueGraph Info\n===============",
		"Section": "Schema Information:",
		"Field":   "Mode: direct",
		"Success": "done",
		"Warn":    "careful",
		"Error":   "broken",
		"Info":    "fyi",
		"Hint":    "try this",
	}
	for name, got := range cases {
		if got != want[name] {
			t.Errorf("%s plain = %q, want %q", name, got, want[name])
		}
		if strings.Contains(got, "\x1b[") {
			t.Errorf("%s plain contains ANSI: %q", name, got)
		}
	}
}

func TestMessageRichIsStyled(t *testing.T) {
	forceRich(t)
	cases := []struct {
		name string
		got  string
		text string
		icon string
	}{
		{"Banner", Banner("IssueGraph Info"), "IssueGraph Info", ""},
		{"Section", Section("Schema Information"), "Schema Information", ""},
		{"Field", Field("Mode", "direct"), "direct", ""},
		{"Success", Success("done"), "done", IconPass},
		{"Warn", Warn("careful"), "careful", IconWarn},
		{"Error", Error("broken"), "broken", IconFail},
		{"Info", Info("fyi"), "fyi", IconInfo},
		{"Hint", Hint("try this"), "try this", "→"},
	}
	for _, tc := range cases {
		if !strings.Contains(tc.got, "\x1b[") {
			t.Errorf("%s rich has no ANSI: %q", tc.name, tc.got)
		}
		if !strings.Contains(tc.got, tc.text) {
			t.Errorf("%s rich lost its text %q: %q", tc.name, tc.text, tc.got)
		}
		if tc.icon != "" && !strings.Contains(tc.got, tc.icon) {
			t.Errorf("%s rich missing icon %q: %q", tc.name, tc.icon, tc.got)
		}
	}
}

func TestRichRespectsEnvironment(t *testing.T) {
	t.Setenv("ISSUEGRAPH_AGENT_MODE", "")
	t.Setenv("CLAUDE_CODE", "")
	t.Setenv("ISSUEGRAPH_GIT_HOOK", "")
	t.Setenv("NO_COLOR", "")
	t.Setenv("CLICOLOR", "")
	t.Setenv("CLICOLOR_FORCE", "1")
	if !Rich() {
		t.Error("CLICOLOR_FORCE=1 should enable rich output")
	}
	t.Setenv("NO_COLOR", "1")
	if Rich() {
		t.Error("NO_COLOR must disable rich output")
	}
	t.Setenv("NO_COLOR", "")
	t.Setenv("ISSUEGRAPH_AGENT_MODE", "1")
	if Rich() {
		t.Error("agent mode must disable rich output")
	}
}
