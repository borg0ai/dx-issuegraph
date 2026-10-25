package main

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/borg0ai/dx-issuegraph/internal/config"
)

func TestGetActorWithGit(t *testing.T) {
	originalActor := actor
	t.Cleanup(func() { actor = originalActor })

	gitUser := configuredGitUser(t)
	tests := []struct {
		name              string
		flag              string
		envActor          string
		user              string
		want              string
		requiresGitUser   bool
		requiresNoGitUser bool
	}{
		{name: "flag takes priority", flag: "flag-actor", envActor: "env-actor", user: "system-user", want: "flag-actor"},
		{name: "environment actor", envActor: "env-actor", user: "system-user", want: "env-actor"},
		{name: "git user fallback", user: "system-user", want: gitUser, requiresGitUser: true},
		{name: "system user fallback", user: "fallback-user", want: "fallback-user", requiresNoGitUser: true},
		{name: "unknown fallback", want: "unknown", requiresNoGitUser: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.requiresGitUser && gitUser == "" {
				t.Skip("git user.name is not configured")
			}
			if test.requiresNoGitUser && gitUser != "" {
				t.Skipf("git user.name is configured as %q", gitUser)
			}
			actor = test.flag
			t.Setenv("ISSUEGRAPH_ACTOR", test.envActor)
			t.Setenv("USER", test.user)
			if got := getActorWithGit(); got != test.want {
				t.Errorf("getActorWithGit() = %q, want %q", got, test.want)
			}
		})
	}
}

func configuredGitUser(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("git", "config", "user.name").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func TestResolveConfiguredActorUsesIssueGraphEnvironment(t *testing.T) {
	t.Setenv("ISSUEGRAPH_ACTOR", "issuegraph-actor")
	if err := config.Initialize(); err != nil {
		t.Fatalf("config.Initialize(): %v", err)
	}
	if got := resolveConfiguredActor(); got != "issuegraph-actor" {
		t.Errorf("resolveConfiguredActor() = %q, want %q", got, "issuegraph-actor")
	}
}

func TestResolveConfiguredActorUsesConfigWhenEnvironmentIsEmpty(t *testing.T) {
	t.Setenv("ISSUEGRAPH_ACTOR", "")
	if err := config.Initialize(); err != nil {
		t.Fatalf("config.Initialize(): %v", err)
	}
	want := config.GetString("actor")
	if got := resolveConfiguredActor(); got != want {
		t.Errorf("resolveConfiguredActor() = %q, want configured actor %q", got, want)
	}
}
