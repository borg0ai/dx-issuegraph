package ado

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/borg0ai/dx-issuegraph/internal/tracker"
	"github.com/borg0ai/dx-issuegraph/internal/types"
)

// IssueToIssueGraph converts an ADO WorkItem (via TrackerIssue) to a issuegraph Issue.
// Returns nil if the TrackerIssue's Raw field is not a *WorkItem.
func (m *adoFieldMapper) IssueToIssueGraph(ti *tracker.TrackerIssue) *tracker.IssueConversion {
	if ti == nil {
		return nil
	}
	wi, ok := ti.Raw.(*WorkItem)
	if !ok || wi == nil {
		return nil
	}

	// Convert description from HTML to Markdown.
	desc, _ := HTMLToMarkdown(wi.GetStringField(FieldDescription))

	// Extract owner from AssignedTo (can be string or identity map).
	owner := extractAssignedTo(wi.GetField(FieldAssignedTo))

	// Parse tags, filtering out internal issuegraph:* tags.
	allTags := parseTags(wi.GetStringField(FieldTags))
	labels := filterIssueGraphTags(allTags)

	issue := &types.Issue{
		Title:       wi.GetStringField(FieldTitle),
		Description: desc,
		Priority:    m.PriorityToIssueGraph(wi.GetField(FieldPriority)),
		Status:      m.StatusToIssueGraph(wi.GetField(FieldState)),
		IssueType:   m.TypeToIssueGraph(wi.GetField(FieldWorkItemType)),
		Owner:       owner,
		Labels:      labels,
	}

	// Restore blocked status from issuegraph:blocked tag (ADO has no blocked state,
	// so blocked maps to Active + tag on push; reverse it here on pull).
	if issue.Status == types.StatusInProgress && hasIssueGraphTag(wi.GetStringField(FieldTags), "issuegraph:blocked") {
		issue.Status = types.StatusBlocked
	}

	// Restore original issuegraph priority from tracker metadata when the mapping
	// is lossy (issuegraph 3 and 4 both map to ADO 4).
	if ti.Metadata != nil {
		if bp, ok := ti.Metadata["issuegraph_priority"]; ok {
			var p int
			var valid bool
			switch v := bp.(type) {
			case string:
				if n, err := strconv.Atoi(v); err == nil {
					p, valid = n, true
				}
			case float64:
				p, valid = int(v), true
			case json.Number:
				if n, err := v.Int64(); err == nil {
					p, valid = int(n), true
				}
			}
			if valid && p >= 0 && p <= 4 {
				issue.Priority = p
			}
		}
	}

	// Build external ref URL.
	ref := buildExternalRef(wi)
	if ref != "" {
		issue.ExternalRef = &ref
	}

	// Preserve ADO-specific metadata for round-trip fidelity.
	meta := buildMetadata(wi)
	// Carry forward issuegraph_priority from TrackerIssue metadata so it survives
	// even when the engine uses conv.Issue.Metadata instead of extIssue.Metadata.
	if ti.Metadata != nil {
		if bp, ok := ti.Metadata["issuegraph_priority"]; ok {
			meta["issuegraph_priority"] = bp
		}
	}
	if len(meta) > 0 {
		raw, err := json.Marshal(meta)
		if err == nil {
			issue.Metadata = json.RawMessage(raw)
		}
	}

	return &tracker.IssueConversion{Issue: issue, Dependencies: ExtractLinkDeps(wi)}
}

// IssueToTracker converts a issuegraph Issue to a map of ADO work item field values.
func (m *adoFieldMapper) IssueToTracker(issue *types.Issue) map[string]interface{} {
	fields := map[string]interface{}{
		FieldTitle:    issue.Title,
		FieldState:    m.StatusToTracker(issue.Status),
		FieldPriority: m.PriorityToTracker(issue.Priority),
	}

	// Convert description from Markdown to HTML.
	if issue.Description != "" {
		htmlDesc, err := MarkdownToHTML(issue.Description)
		if err == nil && htmlDesc != "" {
			fields[FieldDescription] = htmlDesc
		}
	}

	// Build tags: user labels + internal issuegraph tags for round-trip fidelity.
	tags := append([]string{}, issue.Labels...)
	if issue.Status == types.StatusBlocked {
		tags = append(tags, "issuegraph:blocked")
	}
	if len(tags) > 0 {
		fields[FieldTags] = buildTagString(tags)
	}

	// Store original issuegraph priority in metadata for lossy mappings
	// (issuegraph 3 and 4 both map to ADO priority 4).
	if issue.Priority == 3 || issue.Priority == 4 {
		var meta map[string]interface{}
		if len(issue.Metadata) > 0 {
			_ = json.Unmarshal(issue.Metadata, &meta)
		}
		if meta == nil {
			meta = make(map[string]interface{})
		}
		meta["issuegraph_priority"] = strconv.Itoa(issue.Priority)
		if raw, err := json.Marshal(meta); err == nil {
			issue.Metadata = json.RawMessage(raw)
		}
	}

	// Set Severity for Bug-type work items (required by ADO).
	// This is set before restoreMetadata so that a severity value previously
	// pulled from ADO (stored in metadata) takes precedence over the computed one.
	typeName, _ := m.TypeToTracker(issue.IssueType).(string)
	if strings.EqualFold(typeName, "Bug") {
		fields[FieldSeverity] = m.SeverityForBug(issue.Priority)
	}

	// Restore ADO-specific metadata if present (may override computed severity).
	restoreMetadata(issue, fields)

	return fields
}

// extractAssignedTo extracts the display name from an ADO AssignedTo field.
// The field may be a simple string or an identity object with a displayName key.
func extractAssignedTo(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	if m, ok := v.(map[string]interface{}); ok {
		if name, ok := m["displayName"].(string); ok {
			return name
		}
	}
	return ""
}

// buildExternalRef constructs the ADO web URL for a work item.
// Falls back to the API URL if org/project cannot be determined.
func buildExternalRef(wi *WorkItem) string {
	if wi.URL == "" {
		return ""
	}
	// ADO API URL format: https://dev.azure.com/{org}/{project}/_apis/wit/workItems/{id}
	// Web URL format:     https://dev.azure.com/{org}/{project}/_workitems/edit/{id}
	if idx := strings.Index(wi.URL, "/_apis/"); idx > 0 {
		return fmt.Sprintf("%s/_workitems/edit/%d", wi.URL[:idx], wi.ID)
	}
	return wi.URL
}

// buildMetadata extracts ADO-specific fields into a metadata map.
func buildMetadata(wi *WorkItem) map[string]interface{} {
	meta := make(map[string]interface{})

	if v := wi.GetStringField(FieldAreaPath); v != "" {
		meta["ado.area_path"] = v
	}
	if v := wi.GetStringField(FieldIterationPath); v != "" {
		meta["ado.iteration_path"] = v
	}
	if v := wi.GetField(FieldStoryPoints); v != nil {
		meta["ado.story_points"] = v
	}
	if v := wi.GetField(FieldRemainingWork); v != nil {
		meta["ado.remaining_work"] = v
	}
	if v := wi.GetStringField(FieldSeverity); v != "" {
		meta["ado.severity"] = v
	}
	if wi.Rev > 0 {
		meta["ado.rev"] = wi.Rev
	}

	return meta
}

// restoreMetadata copies ADO-specific fields from issue metadata back into the field map.
func restoreMetadata(issue *types.Issue, fields map[string]interface{}) {
	if len(issue.Metadata) == 0 {
		return
	}
	var meta map[string]interface{}
	if err := json.Unmarshal(issue.Metadata, &meta); err != nil {
		return
	}
	if v, ok := meta["ado.area_path"]; ok {
		fields[FieldAreaPath] = v
	}
	if v, ok := meta["ado.iteration_path"]; ok {
		fields[FieldIterationPath] = v
	}
	if v, ok := meta["ado.story_points"]; ok {
		fields[FieldStoryPoints] = v
	}
	if v, ok := meta["ado.severity"]; ok {
		fields[FieldSeverity] = v
	}
}

// parseTags splits an ADO semicolon-separated tag string into a trimmed slice.
// Returns nil for empty input.
func parseTags(tagStr string) []string {
	if strings.TrimSpace(tagStr) == "" {
		return nil
	}
	parts := strings.Split(tagStr, ";")
	var tags []string
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t != "" {
			tags = append(tags, t)
		}
	}
	return tags
}

// buildTagString joins tags with "; " separator (ADO convention).
func buildTagString(tags []string) string {
	return strings.Join(tags, "; ")
}

// filterIssueGraphTags removes internal issuegraph:* tags from a tag slice.
func filterIssueGraphTags(tags []string) []string {
	var out []string
	for _, t := range tags {
		if !strings.HasPrefix(t, "issuegraph:") {
			out = append(out, t)
		}
	}
	return out
}

// hasIssueGraphTag checks if a specific issuegraph:* tag is present in an ADO tag string.
func hasIssueGraphTag(tagStr, tag string) bool {
	for _, t := range parseTags(tagStr) {
		if t == tag {
			return true
		}
	}
	return false
}
