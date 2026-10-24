package idgen

import (
	"testing"
	"time"
)

func TestGenerateHashIDMatchesJiraVector(t *testing.T) {
	timestamp := time.Date(2024, 1, 2, 3, 4, 5, 6*1_000_000, time.UTC)
	prefix := "issuegraph"
	title := "Fix login"
	description := "Details"
	creator := "jira-import"

	tests := map[int]string{
		3: "issuegraph-vju",
		4: "issuegraph-8d8e",
		5: "issuegraph-bi3tk",
		6: "issuegraph-8bi3tk",
		7: "issuegraph-r5sr6bm",
		8: "issuegraph-8r5sr6bm",
	}

	for length, expected := range tests {
		got := GenerateHashID(prefix, title, description, creator, timestamp, length, 0)
		if got != expected {
			t.Fatalf("length %d: got %s, want %s", length, got, expected)
		}
	}
}
