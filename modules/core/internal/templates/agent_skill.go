// Package templates provides embedded files that bd writes into user workspaces.
package templates

import "github.com/steveyegge/beads/internal/generated/issuegraphassets"

// BeadsAgentSkill returns the repo-local Beads agent skill content.
func BeadsAgentSkill() string {
	return issuegraphassets.AgentSkill()
}

// BeadsAgentSkillOpenAIYAML returns the OpenAI UI metadata for the Beads agent skill.
func BeadsAgentSkillOpenAIYAML() string {
	return issuegraphassets.AgentSkillOpenAIYAML()
}
