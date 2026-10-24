package notion

import (
	"github.com/borg0ai/dx-issuegraph/internal/tracker"
	"github.com/borg0ai/dx-issuegraph/internal/types"
)

// FieldMapper implements tracker.FieldMapper for Notion.
type FieldMapper struct {
	config *MappingConfig
}

// NewFieldMapper constructs a Notion field mapper with the default mapping when config is nil.
func NewFieldMapper(config *MappingConfig) *FieldMapper {
	if config == nil {
		config = DefaultMappingConfig()
	}
	return &FieldMapper{config: config}
}

func (m *FieldMapper) PriorityToIssueGraph(trackerPriority interface{}) int {
	value, ok := trackerPriority.(string)
	if !ok {
		return 2
	}
	return priorityToIssueGraph(value, m.config)
}

func (m *FieldMapper) PriorityToTracker(issuegraphPriority int) interface{} {
	return priorityToNotion(issuegraphPriority, m.config)
}

func (m *FieldMapper) StatusToIssueGraph(trackerState interface{}) types.Status {
	value, ok := trackerState.(string)
	if !ok {
		return types.StatusOpen
	}
	return statusToIssueGraph(value, m.config)
}

func (m *FieldMapper) StatusToTracker(issuegraphStatus types.Status) interface{} {
	return statusToNotion(issuegraphStatus, m.config)
}

func (m *FieldMapper) TypeToIssueGraph(trackerType interface{}) types.IssueType {
	value, ok := trackerType.(string)
	if !ok {
		return types.TypeTask
	}
	return typeToIssueGraph(value, m.config)
}

func (m *FieldMapper) TypeToTracker(issuegraphType types.IssueType) interface{} {
	return typeToNotion(issuegraphType, m.config)
}

func (m *FieldMapper) IssueToIssueGraph(ti *tracker.TrackerIssue) *tracker.IssueConversion {
	if ti == nil {
		return nil
	}

	switch raw := ti.Raw.(type) {
	case *PulledIssue:
		issue, err := IssueGraphIssueFromPullIssue(*raw, m.config)
		if err != nil {
			return nil
		}
		return &tracker.IssueConversion{Issue: issue}
	case PulledIssue:
		issue, err := IssueGraphIssueFromPullIssue(raw, m.config)
		if err != nil {
			return nil
		}
		return &tracker.IssueConversion{Issue: issue}
	default:
		return nil
	}
}

func (m *FieldMapper) IssueToTracker(issue *types.Issue) map[string]interface{} {
	pushIssue, err := PushIssueFromIssue(issue, m.config)
	if err != nil {
		return map[string]interface{}{}
	}

	return map[string]interface{}{
		"id":           pushIssue.ID,
		"title":        pushIssue.Title,
		"description":  pushIssue.Description,
		"status":       pushIssue.Status,
		"priority":     pushIssue.Priority,
		"issue_type":   pushIssue.IssueType,
		"assignee":     pushIssue.Assignee,
		"labels":       pushIssue.Labels,
		"external_ref": pushIssue.ExternalRef,
	}
}
