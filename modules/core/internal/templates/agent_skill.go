// Package templates provides embedded files that issuegraph writes into user workspaces.
package templates

import "github.com/borg0ai/dx-issuegraph/internal/generated/issuegraphassets"

// IssueGraphAgentSkill returns the repo-local IssueGraph agent skill content.
func IssueGraphAgentSkill() string {
	return issuegraphassets.AgentSkill()
}

// IssueGraphAgentSkillOpenAIYAML returns the OpenAI UI metadata for the IssueGraph agent skill.
func IssueGraphAgentSkillOpenAIYAML() string {
	return issuegraphassets.AgentSkillOpenAIYAML()
}
