package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestOnboardCommand(t *testing.T) {
	t.Run("onboard output contains key sections", func(t *testing.T) {
		var buf bytes.Buffer
		if err := renderOnboardInstructions(&buf); err != nil {
			t.Fatalf("renderOnboardInstructions() error = %v", err)
		}
		output := buf.String()

		// Verify output contains expected sections
		expectedSections := []string{
			"issuegraph Onboarding",
			"AGENTS.md",
			"BEGIN AGENTS.MD CONTENT",
			"END AGENTS.MD CONTENT",
			"issuegraph prime",
			"How it works",
		}

		for _, section := range expectedSections {
			if !strings.Contains(output, section) {
				t.Errorf("Expected output to contain '%s', but it was missing", section)
			}
		}
	})

	t.Run("agents content is minimal and points to issuegraph prime", func(t *testing.T) {
		// Verify the agentsContent constant is minimal and points to issuegraph prime
		if !strings.Contains(agentsContent, "issuegraph prime") {
			t.Error("agentsContent should point to 'issuegraph prime' for full workflow")
		}
		if !strings.Contains(agentsContent, "issuegraph ready") {
			t.Error("agentsContent should include quick reference to 'issuegraph ready'")
		}
		if !strings.Contains(agentsContent, "issuegraph create") {
			t.Error("agentsContent should include quick reference to 'issuegraph create'")
		}
		if !strings.Contains(agentsContent, "issuegraph close") {
			t.Error("agentsContent should include quick reference to 'issuegraph close'")
		}
		if !strings.Contains(agentsContent, "issuegraph dolt push") {
			t.Error("agentsContent should include quick reference to 'issuegraph dolt push'")
		}

		// Verify it's actually minimal (less than 500 chars)
		if len(agentsContent) > 500 {
			t.Errorf("agentsContent should be minimal (<500 chars), got %d chars", len(agentsContent))
		}
	})

	t.Run("copilot instructions content is minimal", func(t *testing.T) {
		// Verify copilotInstructionsContent is also minimal
		if !strings.Contains(copilotInstructionsContent, "issuegraph prime") {
			t.Error("copilotInstructionsContent should point to 'issuegraph prime'")
		}

		// Verify it's minimal (less than 500 chars)
		if len(copilotInstructionsContent) > 500 {
			t.Errorf("copilotInstructionsContent should be minimal (<500 chars), got %d chars", len(copilotInstructionsContent))
		}
	})
}
