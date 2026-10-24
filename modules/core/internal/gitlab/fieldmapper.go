package gitlab

import (
	"github.com/borg0ai/dx-issuegraph/internal/tracker"
	"github.com/borg0ai/dx-issuegraph/internal/types"
)

// gitlabFieldMapper implements tracker.FieldMapper for GitLab.
type gitlabFieldMapper struct {
	config *MappingConfig
}

func (m *gitlabFieldMapper) PriorityToIssueGraph(trackerPriority interface{}) int {
	// GitLab uses label-based priority (string), not numeric
	if label, ok := trackerPriority.(string); ok {
		if p, exists := m.config.PriorityMap[label]; exists {
			return p
		}
	}
	return 2
}

func (m *gitlabFieldMapper) PriorityToTracker(issuegraphPriority int) interface{} {
	// Inverse lookup: find the label for this priority
	for label, p := range m.config.PriorityMap {
		if p == issuegraphPriority {
			return label
		}
	}
	return "medium"
}

func (m *gitlabFieldMapper) StatusToIssueGraph(trackerState interface{}) types.Status {
	if state, ok := trackerState.(string); ok {
		if status, exists := m.config.StateMap[state]; exists {
			return types.Status(status)
		}
		// GitLab-specific defaults
		switch state {
		case "opened", "reopened":
			return types.StatusOpen
		case "closed":
			return types.StatusClosed
		}
	}
	return types.StatusOpen
}

func (m *gitlabFieldMapper) StatusToTracker(issuegraphStatus types.Status) interface{} {
	switch issuegraphStatus {
	case types.StatusClosed:
		return "closed"
	default:
		return "opened"
	}
}

func (m *gitlabFieldMapper) TypeToIssueGraph(trackerType interface{}) types.IssueType {
	if t, ok := trackerType.(string); ok {
		if issueType, exists := m.config.LabelTypeMap[t]; exists {
			return types.IssueType(issueType)
		}
	}
	return types.TypeTask
}

func (m *gitlabFieldMapper) TypeToTracker(issuegraphType types.IssueType) interface{} {
	return string(issuegraphType)
}

func (m *gitlabFieldMapper) IssueToIssueGraph(ti *tracker.TrackerIssue) *tracker.IssueConversion {
	gl, ok := ti.Raw.(*Issue)
	if !ok {
		return nil
	}

	conv := GitLabIssueToIssueGraph(gl, m.config)
	if conv == nil {
		return nil
	}

	// Convert gitlab.DependencyInfo to tracker.DependencyInfo
	var deps []tracker.DependencyInfo
	for _, d := range conv.Dependencies {
		deps = append(deps, trackerDependencyFromGitLab(d))
	}

	return &tracker.IssueConversion{
		Issue:        conv.Issue,
		Dependencies: deps,
	}
}

func (m *gitlabFieldMapper) IssueToTracker(issue *types.Issue) map[string]interface{} {
	return IssueGraphIssueToGitLabFields(issue, m.config)
}
