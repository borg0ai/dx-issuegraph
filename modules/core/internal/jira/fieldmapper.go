package jira

import (
	"strconv"
	"strings"

	"github.com/borg0ai/dx-issuegraph/internal/tracker"
	"github.com/borg0ai/dx-issuegraph/internal/types"
)

// jiraFieldMapper implements tracker.FieldMapper for Jira.
type jiraFieldMapper struct {
	apiVersion       string                            // "2" or "3" (default: "3")
	statusMap        map[string]string                 // issuegraph status → Jira status name (from jira.status_map.* config)
	typeMap          map[string]string                 // issuegraph type → Jira type (from jira.type_map.* config)
	priorityMap      map[string]string                 // issuegraph priority (as string "0"-"4") → Jira priority name (from jira.priority_map.* config)
	customFields     map[string]interface{}            // Jira field name/id → value (from jira.custom_fields.* config)
	typeCustomFields map[string]map[string]interface{} // Jira issue type → field name/id → value
}

func (m *jiraFieldMapper) PriorityToIssueGraph(trackerPriority interface{}) int {
	if name, ok := trackerPriority.(string); ok {
		// Check custom map first (inverted: Jira name → issuegraph priority).
		for issuegraphPri, jiraName := range m.priorityMap {
			if strings.EqualFold(name, jiraName) {
				if v, err := strconv.Atoi(issuegraphPri); err == nil && v >= 0 && v <= 4 {
					return v
				}
			}
		}
		// Jira defaults.
		switch name {
		case "Highest":
			return 0
		case "High":
			return 1
		case "Medium":
			return 2
		case "Low":
			return 3
		case "Lowest":
			return 4
		}
	}
	return 2
}

func (m *jiraFieldMapper) PriorityToTracker(issuegraphPriority int) interface{} {
	// Check custom map first (issuegraph priority as string key → Jira name).
	if m.priorityMap != nil {
		key := strconv.Itoa(issuegraphPriority)
		if name, ok := m.priorityMap[key]; ok {
			return name
		}
	}
	// Jira defaults.
	switch issuegraphPriority {
	case 0:
		return "Highest"
	case 1:
		return "High"
	case 2:
		return "Medium"
	case 3:
		return "Low"
	case 4:
		return "Lowest"
	default:
		return "Medium"
	}
}

func (m *jiraFieldMapper) StatusToIssueGraph(trackerState interface{}) types.Status {
	if state, ok := trackerState.(string); ok {
		// Check custom map first (inverted: jira name → issuegraph status).
		for issuegraphStatus, jiraName := range m.statusMap {
			if strings.EqualFold(state, jiraName) {
				return types.Status(issuegraphStatus)
			}
		}
		switch state {
		case "To Do", "Open", "Backlog", "New":
			return types.StatusOpen
		case "In Progress", "In Review":
			return types.StatusInProgress
		case "Blocked":
			return types.StatusBlocked
		case "Done", "Closed", "Resolved":
			return types.StatusClosed
		}
	}
	return types.StatusOpen
}

func (m *jiraFieldMapper) StatusToTracker(issuegraphStatus types.Status) interface{} {
	// Check custom map first.
	if name, ok := m.statusMap[string(issuegraphStatus)]; ok {
		return name
	}
	switch issuegraphStatus {
	case types.StatusOpen:
		return "To Do"
	case types.StatusInProgress:
		return "In Progress"
	case types.StatusBlocked:
		return "Blocked"
	case types.StatusClosed:
		return "Done"
	default:
		return "To Do"
	}
}

func (m *jiraFieldMapper) TypeToIssueGraph(trackerType interface{}) types.IssueType {
	t, ok := trackerType.(string)
	if !ok {
		return types.TypeTask
	}

	// Check custom map first (inverted: Jira type → issuegraph type).
	for issuegraphType, jiraType := range m.typeMap {
		if strings.EqualFold(t, jiraType) {
			return types.IssueType(issuegraphType)
		}
	}

	// Jira defaults.
	switch t {
	case "Bug":
		return types.TypeBug
	case "Story", "Feature":
		return types.TypeFeature
	case "Epic":
		return types.TypeEpic
	case "Task", "Sub-task":
		return types.TypeTask
	}
	return types.TypeTask
}

func (m *jiraFieldMapper) TypeToTracker(issuegraphType types.IssueType) interface{} {
	if name, ok := m.typeMap[string(issuegraphType)]; ok {
		return name
	}
	switch issuegraphType {
	case types.TypeBug:
		return "Bug"
	case types.TypeFeature:
		return "Story"
	case types.TypeEpic:
		return "Epic"
	default:
		return "Task"
	}
}

func (m *jiraFieldMapper) IssueToIssueGraph(ti *tracker.TrackerIssue) *tracker.IssueConversion {
	ji, ok := ti.Raw.(*Issue)
	if !ok || ji == nil {
		return nil
	}

	issue := &types.Issue{
		Title:       ji.Fields.Summary,
		Description: DescriptionToPlainText(ji.Fields.Description),
		Priority:    m.PriorityToIssueGraph(priorityName(ji)),
		Status:      m.StatusToIssueGraph(statusName(ji)),
		IssueType:   m.TypeToIssueGraph(typeName(ji)),
	}

	if ji.Fields.Assignee != nil {
		issue.Owner = ji.Fields.Assignee.DisplayName
	}

	if ji.Fields.Labels != nil {
		issue.Labels = ji.Fields.Labels
	}

	// Set external ref from issue URL
	if ji.Self != "" {
		ref := extractBrowseURL(ji)
		issue.ExternalRef = &ref
	}

	return &tracker.IssueConversion{
		Issue: issue,
	}
}

func (m *jiraFieldMapper) IssueToTracker(issue *types.Issue) map[string]interface{} {
	fields := map[string]interface{}{
		"summary": issue.Title,
	}

	// v3 requires ADF (Atlassian Document Format); v2 accepts a plain string.
	if issue.Description != "" {
		if m.apiVersion == "2" {
			fields["description"] = issue.Description
		} else {
			fields["description"] = PlainTextToADF(issue.Description)
		}
	}

	// Set issue type
	typeName := m.TypeToTracker(issue.IssueType)
	if name, ok := typeName.(string); ok {
		fields["issuetype"] = map[string]string{"name": name}
	}

	// Set priority
	priorityName := m.PriorityToTracker(issue.Priority)
	if name, ok := priorityName.(string); ok {
		fields["priority"] = map[string]string{"name": name}
	}

	// Set labels
	if len(issue.Labels) > 0 {
		fields["labels"] = issue.Labels
	}

	for fieldName, value := range m.customFields {
		fields[fieldName] = value
	}

	if name, ok := typeName.(string); ok {
		for jiraType, customFields := range m.typeCustomFields {
			if !strings.EqualFold(jiraType, name) {
				continue
			}
			for fieldName, value := range customFields {
				fields[fieldName] = value
			}
		}
	}

	return fields
}

// Helper functions for safe field extraction from Jira issues.

func priorityName(ji *Issue) string {
	if ji.Fields.Priority != nil {
		return ji.Fields.Priority.Name
	}
	return ""
}

func statusName(ji *Issue) string {
	if ji.Fields.Status != nil {
		return ji.Fields.Status.Name
	}
	return ""
}

func typeName(ji *Issue) string {
	if ji.Fields.IssueType != nil {
		return ji.Fields.IssueType.Name
	}
	return ""
}

// extractBrowseURL builds the human-readable browse URL from a Jira issue.
// Self is "https://company.atlassian.net/rest/api/3/issue/10001";
// we need "https://company.atlassian.net/browse/PROJ-123".
func extractBrowseURL(ji *Issue) string {
	if ji.Self == "" || ji.Key == "" {
		return ""
	}
	if idx := strings.Index(ji.Self, "/rest/api/"); idx > 0 {
		return ji.Self[:idx] + "/browse/" + ji.Key
	}
	return ""
}
