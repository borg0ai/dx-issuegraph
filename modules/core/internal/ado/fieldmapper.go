package ado

import (
	"strings"

	"github.com/borg0ai/dx-issuegraph/internal/tracker"
	"github.com/borg0ai/dx-issuegraph/internal/types"
)

// adoFieldMapper implements tracker.FieldMapper for Azure DevOps.
type adoFieldMapper struct {
	stateMap map[string]string // issuegraph status → ADO state (from ado.state_map.* config)
	typeMap  map[string]string // issuegraph type → ADO type (from ado.type_map.* config)
}

// Compile-time interface check.
var _ tracker.FieldMapper = (*adoFieldMapper)(nil)

// NewFieldMapper creates a new ADO field mapper with optional custom mappings.
// stateMap overrides default status mapping (issuegraph status → ADO state).
// typeMap overrides default type mapping (issuegraph type → ADO type).
// Pass nil for either to use defaults only.
func NewFieldMapper(stateMap, typeMap map[string]string) tracker.FieldMapper {
	sm := make(map[string]string)
	for k, v := range stateMap {
		sm[k] = v
	}
	tm := make(map[string]string)
	for k, v := range typeMap {
		tm[k] = v
	}
	return &adoFieldMapper{stateMap: sm, typeMap: tm}
}

// PriorityToIssueGraph converts an ADO priority (float64 from JSON: 1-4) to issuegraph (0-4).
// ADO 1→0, 2→1, 3→2, 4→3. Unknown values default to 2 (medium).
func (m *adoFieldMapper) PriorityToIssueGraph(trackerPriority interface{}) int {
	p, ok := trackerPriority.(float64)
	if !ok {
		return 2
	}
	switch int(p) {
	case 1:
		return 0
	case 2:
		return 1
	case 3:
		return 2
	case 4:
		return 3
	default:
		return 2
	}
}

// PriorityToTracker converts a issuegraph priority (0-4) to ADO priority (1-4).
// IssueGraph 0→1, 1→2, 2→3, 3→4, 4→4 (lossy: backlog collapses to low).
func (m *adoFieldMapper) PriorityToTracker(issuegraphPriority int) interface{} {
	switch issuegraphPriority {
	case 0:
		return 1
	case 1:
		return 2
	case 2:
		return 3
	case 3:
		return 4
	case 4:
		return 4
	default:
		return 3
	}
}

// StatusToIssueGraph converts an ADO state string to a issuegraph Status.
// Checks custom stateMap first (inverted lookup), then falls back to Agile defaults.
func (m *adoFieldMapper) StatusToIssueGraph(trackerState interface{}) types.Status {
	state, ok := trackerState.(string)
	if !ok {
		return types.StatusOpen
	}

	// Check custom map first (inverted: ADO state → issuegraph status).
	for issuegraphStatus, adoState := range m.stateMap {
		if strings.EqualFold(state, adoState) {
			return types.Status(issuegraphStatus)
		}
	}

	// Agile defaults.
	switch state {
	case "New":
		return types.StatusOpen
	case "Active":
		return types.StatusInProgress
	case "Resolved":
		return types.StatusClosed
	case "Closed":
		return types.StatusClosed
	case "Removed":
		return types.StatusDeferred
	default:
		return types.StatusOpen
	}
}

// StatusToTracker converts a issuegraph Status to an ADO state string.
// Checks custom stateMap first, then falls back to Agile defaults.
func (m *adoFieldMapper) StatusToTracker(issuegraphStatus types.Status) interface{} {
	if name, ok := m.stateMap[string(issuegraphStatus)]; ok {
		return name
	}
	switch issuegraphStatus {
	case types.StatusOpen:
		return "New"
	case types.StatusInProgress:
		return "Active"
	case types.StatusBlocked:
		return "Active"
	case types.StatusDeferred:
		return "Removed"
	case types.StatusClosed:
		return "Closed"
	default:
		return "New"
	}
}

// TypeToIssueGraph converts an ADO work item type string to a issuegraph IssueType.
// Uses case-insensitive matching. Checks custom typeMap first (inverted), then defaults.
func (m *adoFieldMapper) TypeToIssueGraph(trackerType interface{}) types.IssueType {
	t, ok := trackerType.(string)
	if !ok {
		return types.TypeTask
	}

	// Check custom map first (inverted: ADO type → issuegraph type).
	for issuegraphType, adoType := range m.typeMap {
		if strings.EqualFold(t, adoType) {
			return types.IssueType(issuegraphType)
		}
	}

	// Agile defaults (case-insensitive).
	lower := strings.ToLower(t)
	switch lower {
	case "bug":
		return types.TypeBug
	case "user story":
		return types.TypeFeature
	case "product backlog item":
		return types.TypeFeature
	case "task":
		return types.TypeTask
	case "epic":
		return types.TypeEpic
	default:
		return types.TypeTask
	}
}

// SeverityForBug maps a issuegraph priority (0-4) to an ADO Severity string.
// ADO Bug work items require a Severity field with values like "1 - Critical".
// IssueGraph 0→"1 - Critical", 1→"2 - High", 2→"3 - Medium", 3/4→"4 - Low".
// Returns "3 - Medium" for unknown values.
func (m *adoFieldMapper) SeverityForBug(issuegraphPriority int) string {
	switch issuegraphPriority {
	case 0:
		return "1 - Critical"
	case 1:
		return "2 - High"
	case 2:
		return "3 - Medium"
	case 3:
		return "4 - Low"
	case 4:
		return "4 - Low"
	default:
		return "3 - Medium"
	}
}

// TypeToTracker converts a issuegraph IssueType to an ADO work item type string.
// Checks custom typeMap first, then falls back to Agile defaults.
func (m *adoFieldMapper) TypeToTracker(issuegraphType types.IssueType) interface{} {
	if name, ok := m.typeMap[string(issuegraphType)]; ok {
		return name
	}
	switch issuegraphType {
	case types.TypeBug:
		return "Bug"
	case types.TypeFeature:
		return "User Story"
	case types.TypeEpic:
		return "Epic"
	case types.TypeTask:
		return "Task"
	case types.TypeChore:
		return "Task"
	default:
		return "Task"
	}
}
