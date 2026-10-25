package linear

import (
	"github.com/borg0ai/dx-issuegraph/internal/tracker"
	"github.com/borg0ai/dx-issuegraph/internal/types"
)

// linearFieldMapper implements tracker.FieldMapper for Linear.
type linearFieldMapper struct {
	config     *MappingConfig
	labelCache *LabelCache // optional; when set, IssueToTracker includes labelIds
}

func (m *linearFieldMapper) PriorityToIssueGraph(trackerPriority interface{}) int {
	if p, ok := trackerPriority.(int); ok {
		return PriorityToIssueGraph(p, m.config)
	}
	return 2
}

func (m *linearFieldMapper) PriorityToTracker(issuegraphPriority int) interface{} {
	return PriorityToLinear(issuegraphPriority, m.config)
}

func (m *linearFieldMapper) StatusToIssueGraph(trackerState interface{}) types.Status {
	if state, ok := trackerState.(*State); ok {
		return StateToIssueGraphStatus(state, m.config)
	}
	return types.StatusOpen
}

func (m *linearFieldMapper) StatusToTracker(issuegraphStatus types.Status) interface{} {
	return StatusToLinearStateType(issuegraphStatus)
}

func (m *linearFieldMapper) TypeToIssueGraph(trackerType interface{}) types.IssueType {
	if labels, ok := trackerType.(*Labels); ok {
		return LabelToIssueType(labels, m.config)
	}
	return types.TypeTask
}

func (m *linearFieldMapper) TypeToTracker(issuegraphType types.IssueType) interface{} {
	return string(issuegraphType)
}

func (m *linearFieldMapper) IssueToIssueGraph(ti *tracker.TrackerIssue) *tracker.IssueConversion {
	li, ok := ti.Raw.(*Issue)
	if !ok {
		return nil
	}

	conv := IssueToIssueGraph(li, m.config)
	if conv == nil {
		return nil
	}

	issue, ok := conv.Issue.(*types.Issue)
	if !ok {
		return nil
	}

	var deps []tracker.DependencyInfo
	for _, d := range conv.Dependencies {
		deps = append(deps, tracker.DependencyInfo{
			FromExternalID: d.FromLinearID,
			ToExternalID:   d.ToLinearID,
			Type:           d.Type,
			Source:         tracker.DependencySource(d.Source),
		})
	}

	return &tracker.IssueConversion{
		Issue:        issue,
		Dependencies: deps,
	}
}

func (m *linearFieldMapper) IssueToTracker(issue *types.Issue) map[string]interface{} {
	updates := map[string]interface{}{
		"title":       issue.Title,
		"description": issue.Description,
		"priority":    PriorityToLinear(issue.Priority, m.config),
	}
	if m.labelCache != nil {
		if ids, _ := ResolveLabelIDs(issue, m.labelCache, m.config); len(ids) > 0 {
			updates["labelIds"] = ids
		}
	}
	return updates
}
