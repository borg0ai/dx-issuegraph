# IssueGraph Skill Maintenance Guide

## Architecture Decisions

ADRs in `adr/` document key decisions. These are NOT loaded during skill invocation—they're reference material for maintainers making changes.

| ADR | Decision |
|-----|----------|
| [ADR-0001](adr/0001-bd-prime-as-source-of-truth.md) | Use `issuegraph prime` as CLI reference source of truth |

## Key Principle: DRY via issuegraph prime

**NEVER duplicate CLI documentation in SKILL.md or resources.**

- `issuegraph prime` outputs AI-optimized workflow context
- `issuegraph <command> --help` provides specific usage
- Both auto-update with issuegraph releases

**SKILL.md should only contain:**
- Decision frameworks (issuegraph vs TodoWrite)
- Prerequisites (install verification)
- Resource index (progressive disclosure)
- Pointers to `issuegraph prime` and `--help`

## Keeping the Skill Updated

### When issuegraph releases new version:

1. **Check for new features**: `issuegraph --help` for new commands
2. **Update SKILL.md frontmatter**: `version: "X.Y.Z"`
3. **Add resources for conceptual features** (agents, gates, chemistry patterns)
4. **Don't add CLI reference** — that's `issuegraph prime`'s job

### What belongs in resources:

| Content Type | Belongs in Resources? | Why |
|--------------|----------------------|-----|
| Conceptual frameworks | ✅ Yes | issuegraph prime doesn't explain "when to use" |
| Decision trees | ✅ Yes | Cognitive guidance, not CLI reference |
| Advanced patterns | ✅ Yes | Depth beyond `--help` |
| CLI command syntax | ❌ No | Use `issuegraph <cmd> --help` |
| Workflow checklists | ❌ No | `issuegraph prime` covers this |

### Resource update checklist:

```
[ ] Check if issuegraph prime now covers this content
[ ] If yes, remove from resources (avoid duplication)
[ ] If no, update resource for new issuegraph version
[ ] Update version compatibility in README.md
```

## File Roles

| File | Purpose | When to Update |
|------|---------|----------------|
| SKILL.md | Entry point, resource index | New features, version bumps |
| README.md | Human docs, installation | Structure changes |
| CLAUDE.md | This file, maintenance guide | Architecture changes |
| adr/*.md | Decision records | When making architectural decisions |
| resources/*.md | Deep-dive guides | New conceptual content |

## Testing Changes

After skill updates:

```bash
# Verify SKILL.md is within token budget
wc -w plugins/issuegraph/skills/issuegraph/SKILL.md  # Target: 400-600 words

# Verify links resolve
# (Manual check: ensure all resource links in SKILL.md exist)

# Verify issuegraph prime still works
issuegraph prime | head -20
```

## Attribution

Resources adapted from other sources should include attribution header:

```markdown
# Resource Title

> Adapted from [source]
```
