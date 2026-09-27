---
description: Log and label agent interactions (append-only JSONL)
argument-hint: "record|label"
---

Append-only audit logging for agent interactions (prompts, responses, tool calls) in `.beads/interactions.jsonl`.

Each line is one event. Labeling is done by appending a new `"label"` event referencing a previous entry.

## Usage

- **Record an interaction**:
  - `issuegraph audit record --kind llm_call --model "claude-3-5-haiku" --prompt "..." --response "..."`
  - `issuegraph audit record --kind tool_call --tool-name "go test" --exit-code 1 --error "..." --issue-id ig-42`

- **Pipe JSON via stdin**:
  - `cat event.json | issuegraph audit record`

- **Label an entry**:
  - `issuegraph audit label int-a1b2 --label good --reason "Worked perfectly"`
  - `issuegraph audit label int-a1b2 --label bad --reason "Hallucinated a file path"`

## Notes

- Audit entries are **append-only** (no in-place edits).
- `issuegraph dolt push` includes `.beads/interactions.jsonl` in the commit allowlist.

