package notion

import (
	"testing"
	"time"

	"github.com/borg0ai/dx-issuegraph/internal/types"
)

func TestBuildPagePropertiesAndPulledIssueRoundTrip(t *testing.T) {
	t.Parallel()

	externalRef := "https://www.notion.so/Task-0123456789abcdef0123456789abcdef"
	issue := &types.Issue{
		ID:          "bd-123",
		Title:       "Sync from Notion",
		Description: "Short summary",
		Status:      types.StatusInProgress,
		Priority:    1,
		IssueType:   types.TypeFeature,
		Assignee:    "osamu",
		Labels:      []string{"sync", "notion"},
		ExternalRef: &externalRef,
	}

	pushIssue, err := PushIssueFromIssue(issue, nil)
	if err != nil {
		t.Fatalf("PushIssueFromIssue returned error: %v", err)
	}
	if pushIssue.Status != "In Progress" {
		t.Fatalf("status = %q, want %q", pushIssue.Status, "In Progress")
	}
	if pushIssue.ExternalRef != "https://www.notion.so/0123456789abcdef0123456789abcdef" {
		t.Fatalf("external_ref = %q", pushIssue.ExternalRef)
	}
	if len(pushIssue.Labels) != 2 {
		t.Fatalf("push labels = %v, want 2 labels", pushIssue.Labels)
	}

	properties := BuildPageProperties(pushIssue)
	statusProp, ok := properties[PropertyStatus].(map[string]interface{})
	if !ok {
		t.Fatalf("status property type = %T", properties[PropertyStatus])
	}
	selectProp, ok := statusProp["select"].(map[string]interface{})
	if !ok || selectProp["name"] != "In Progress" {
		t.Fatalf("status select = %#v", statusProp["select"])
	}
	labelsProp, ok := properties[PropertyLabels].(map[string]interface{})
	if !ok {
		t.Fatalf("labels property type = %T", properties[PropertyLabels])
	}
	multiSelect, ok := labelsProp["multi_select"].([]map[string]interface{})
	if !ok || len(multiSelect) != 2 {
		t.Fatalf("labels multi_select = %#v", labelsProp["multi_select"])
	}

	createdAt := time.Date(2026, 3, 19, 14, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(5 * time.Minute)
	page := Page{
		ID:             "01234567-89ab-cdef-0123-456789abcdef",
		URL:            "https://www.notion.so/Task-0123456789abcdef0123456789abcdef?pvs=4",
		CreatedTime:    createdAt,
		LastEditedTime: updatedAt,
		Properties: map[string]PageProperty{
			PropertyTitle:        {Title: []RichText{{PlainText: "Sync from Notion"}}},
			PropertyIssueGraphID: {RichText: []RichText{{PlainText: "bd-123"}}},
			PropertyStatus:       {Select: &SelectOption{Name: "In Progress"}},
			PropertyPriority:     {Select: &SelectOption{Name: "High"}},
			PropertyType:         {Select: &SelectOption{Name: "Feature"}},
			PropertyDescription:  {RichText: []RichText{{PlainText: "Short summary"}}},
			PropertyAssignee:     {RichText: []RichText{{PlainText: "osamu"}}},
			PropertyLabels:       {MultiSelect: []SelectOption{{Name: "sync"}, {Name: "notion"}}},
		},
	}

	pulled := PulledIssueFromPage(page)
	if pulled.ExternalRef != "https://www.notion.so/0123456789abcdef0123456789abcdef" {
		t.Fatalf("external ref = %q", pulled.ExternalRef)
	}

	issuegraphIssue, err := IssueGraphIssueFromPullIssue(pulled, nil)
	if err != nil {
		t.Fatalf("IssueGraphIssueFromPullIssue returned error: %v", err)
	}
	if issuegraphIssue.Status != types.StatusInProgress {
		t.Fatalf("status = %q, want %q", issuegraphIssue.Status, types.StatusInProgress)
	}
	if issuegraphIssue.Priority != 1 {
		t.Fatalf("priority = %d, want 1", issuegraphIssue.Priority)
	}
	if issuegraphIssue.IssueType != types.TypeFeature {
		t.Fatalf("issue type = %q, want %q", issuegraphIssue.IssueType, types.TypeFeature)
	}
	if len(issuegraphIssue.Labels) != 2 {
		t.Fatalf("labels = %v, want 2 labels", issuegraphIssue.Labels)
	}
}

func TestValidateDataSourceSchema(t *testing.T) {
	t.Parallel()

	ds := &DataSource{
		Properties: map[string]DataSourceProperty{
			PropertyTitle:        {Type: "title"},
			PropertyIssueGraphID: {Type: "rich_text"},
			PropertyStatus:       {Type: "select"},
			PropertyPriority:     {Type: "select"},
			PropertyType:         {Type: "select"},
			PropertyDescription:  {Type: "rich_text"},
			PropertyAssignee:     {Type: "rich_text"},
		},
	}

	schema := ValidateDataSourceSchema(ds)
	if !schema.Checked {
		t.Fatal("expected schema to be checked")
	}
	if len(schema.Missing) != 1 || schema.Missing[0] != PropertyLabels {
		t.Fatalf("missing = %v, want [%q]", schema.Missing, PropertyLabels)
	}
}

func TestValidateDataSourceSchemaMatchesInitSchema(t *testing.T) {
	t.Parallel()

	properties := BuildInitialDataSourceProperties()
	ds := &DataSource{Properties: map[string]DataSourceProperty{}}
	for name, raw := range properties {
		property, ok := raw.(map[string]interface{})
		if !ok {
			t.Fatalf("property %q type = %T", name, raw)
		}
		for key := range property {
			if key == "type" {
				continue
			}
			ds.Properties[name] = DataSourceProperty{Type: key}
			break
		}
	}

	schema := ValidateDataSourceSchema(ds)
	if len(schema.Missing) != 0 {
		t.Fatalf("missing = %v", schema.Missing)
	}
}

func TestPushIssueFromIssueRejectsUnsupportedIssueType(t *testing.T) {
	t.Parallel()

	issue := &types.Issue{
		ID:        "bd-event-1",
		Title:     "State change: progress -> in_progress",
		Status:    types.StatusOpen,
		Priority:  2,
		IssueType: types.TypeEvent,
	}

	if SupportsIssueType(issue.IssueType, nil) {
		t.Fatal("expected event to be unsupported for Notion schema")
	}

	if _, err := PushIssueFromIssue(issue, nil); err == nil {
		t.Fatal("expected unsupported issue type error")
	}
}
