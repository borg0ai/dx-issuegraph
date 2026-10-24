package github

import (
	"fmt"

	"github.com/borg0ai/dx-issuegraph/internal/tracker"
	"github.com/borg0ai/dx-issuegraph/internal/types"
)

// githubFieldMapper implements tracker.FieldMapper for GitHub.
type githubFieldMapper struct {
	config *MappingConfig
}

func (m *githubFieldMapper) PriorityToIssueGraph(trackerPriority interface{}) int {
	// GitHub uses label-based priority (string), not numeric
	if label, ok := trackerPriority.(string); ok {
		if p, exists := m.config.PriorityMap[label]; exists {
			return p
		}
	}
	return 2
}

func (m *githubFieldMapper) PriorityToTracker(issuegraphPriority int) interface{} {
	// Inverse lookup: find the label for this priority
	for label, p := range m.config.PriorityMap {
		if p == issuegraphPriority {
			return label
		}
	}
	return "medium"
}

func (m *githubFieldMapper) StatusToIssueGraph(trackerState interface{}) types.Status {
	if state, ok := trackerState.(string); ok {
		if status, exists := m.config.StateMap[state]; exists {
			return types.Status(status)
		}
		// GitHub-specific defaults
		switch state {
		case "open":
			return types.StatusOpen
		case "closed":
			return types.StatusClosed
		}
	}
	return types.StatusOpen
}

func (m *githubFieldMapper) StatusToTracker(issuegraphStatus types.Status) interface{} {
	switch issuegraphStatus {
	case types.StatusClosed:
		return "closed"
	default:
		return "open"
	}
}

func (m *githubFieldMapper) TypeToIssueGraph(trackerType interface{}) types.IssueType {
	if t, ok := trackerType.(string); ok {
		if issueType, exists := m.config.LabelTypeMap[t]; exists {
			return types.IssueType(issueType)
		}
	}
	return types.TypeTask
}

func (m *githubFieldMapper) TypeToTracker(issuegraphType types.IssueType) interface{} {
	return string(issuegraphType)
}

func (m *githubFieldMapper) IssueToIssueGraph(ti *tracker.TrackerIssue) *tracker.IssueConversion {
	gh, ok := ti.Raw.(*Issue)
	if !ok {
		return nil
	}

	conv := GitHubIssueToIssueGraph(gh, m.config)
	if conv == nil {
		return nil
	}

	// Convert github.DependencyInfo to tracker.DependencyInfo
	var deps []tracker.DependencyInfo
	for _, d := range conv.Dependencies {
		deps = append(deps, tracker.DependencyInfo{
			FromExternalID: fmt.Sprintf("%d", d.FromGitHubNumber),
			ToExternalID:   fmt.Sprintf("%d", d.ToGitHubNumber),
			Type:           d.Type,
		})
	}

	return &tracker.IssueConversion{
		Issue:        conv.Issue,
		Dependencies: deps,
	}
}

func (m *githubFieldMapper) IssueToTracker(issue *types.Issue) map[string]interface{} {
	return IssueGraphIssueToGitHubFields(issue, m.config)
}
