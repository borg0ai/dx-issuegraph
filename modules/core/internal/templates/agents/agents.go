// Package agents provides embedded AGENTS.md templates for issuegraph init and setup.
package agents

import (
	_ "embed"
	"strings"
)

//go:embed defaults/agents.md.tmpl
var defaultTemplate string

//go:embed defaults/issuegraph-section.md
var issuegraphSection string

// EmbeddedDefault returns the full AGENTS.md template content.
func EmbeddedDefault() string {
	return defaultTemplate
}

// EmbeddedIssueGraphSection returns the issuegraph integration section with markers.
// The returned string is trimmed to match the existing agentsIssueGraphSection behavior
// (no trailing newline after the end marker).
func EmbeddedIssueGraphSection() string {
	return strings.TrimRight(issuegraphSection, "\n") + "\n"
}
