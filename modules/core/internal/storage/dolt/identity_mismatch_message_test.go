package dolt

import (
	"strings"
	"testing"
)

// TestProjectIdentityMismatchError_InitAdvice pins GH#5558: when the identity
// check fires inside issuegraph init (Config.OpenedByInit), the refusal must not tell
// the user not to run the command they are running, and must name remedies that
// work from there. Every other open keeps its original advice — including a
// CreateIfMissing open, which is not by itself issuegraph init.
func TestProjectIdentityMismatchError_InitAdvice(t *testing.T) {
	const localID, dbID, database = "project-AAAA", "project-BBBB", "shared_db"

	t.Run("opened-by-init", func(t *testing.T) {
		msg := projectIdentityMismatchError(localID, dbID, database, true).Error()
		if strings.Contains(msg, "Do NOT run 'issuegraph init'") {
			t.Errorf("init-time refusal tells the user not to run issuegraph init:\n%s", msg)
		}
		for _, want := range []string{
			"PROJECT IDENTITY MISMATCH",
			localID,
			dbID,
			`"` + database + `"`,
			"issuegraph dolt status",
			"issuegraph init --database <other-name>",
			"issuegraph init --server-host <host> --server-port <port>",
			"issuegraph doctor --fix",
			"issuegraph bootstrap",
		} {
			if !strings.Contains(msg, want) {
				t.Errorf("init-time refusal missing %q:\n%s", want, msg)
			}
		}
	})

	t.Run("normal open", func(t *testing.T) {
		msg := projectIdentityMismatchError(localID, dbID, database, false).Error()
		for _, want := range []string{
			"PROJECT IDENTITY MISMATCH — refusing to connect",
			localID,
			dbID,
			"issuegraph dolt status",
			"Do NOT run 'issuegraph init' — your data likely exists, just on a different server.",
		} {
			if !strings.Contains(msg, want) {
				t.Errorf("normal-open refusal missing %q:\n%s", want, msg)
			}
		}
	})
}
