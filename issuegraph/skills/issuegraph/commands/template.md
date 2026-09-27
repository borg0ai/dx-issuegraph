# issuegraph template

Manage issue templates for streamlined issue creation.

## Synopsis

Templates provide pre-filled structures for common issue types, making it faster to create well-formed issues with consistent formatting.

```bash
issuegraph template list
issuegraph template show <template-name>
issuegraph template create <template-name>
```

## Description

Templates can be:
- **Built-in**: Provided by issuegraph (epic, bug, feature)
- **Custom**: Stored in `.beads/templates/` directory

Each template defines default values for:
- Description structure with placeholders
- Issue type (bug, feature, task, epic, chore)
- Priority (0-4)
- Labels
- Design notes structure
- Acceptance criteria structure

## Commands

### list

List all available templates (built-in and custom).

```bash
issuegraph template list
issuegraph template list --json
```

**Examples:**

```bash
$ issuegraph template list
Built-in Templates:
  epic
    Type: epic, Priority: P1
    Labels: epic
  bug
    Type: bug, Priority: P1
    Labels: bug
  feature
    Type: feature, Priority: P2
    Labels: feature
```

### show

Show detailed structure of a specific template.

```bash
issuegraph template show <template-name>
issuegraph template show <template-name> --json
```

**Examples:**

```bash
$ issuegraph template show bug
Template: bug
Type: bug
Priority: P1
Labels: bug

Description:
## Summary

[Brief description of the bug]

## Steps to Reproduce
...
```

### create

Create a custom template in `.beads/templates/` directory.

```bash
issuegraph template create <template-name>
```

This creates a YAML file with default structure that you can edit to customize.

**Examples:**

```bash
$ issuegraph template create performance
✓ Created template: .beads/templates/performance.yaml
Edit the file to customize your template.

$ cat .beads/templates/performance.yaml
name: performance
description: |-
    [Describe the issue]
    
    ## Additional Context
    
    [Add relevant details]
type: task
priority: 2
labels: []
design: '[Design notes]'
acceptance_criteria: |-
    - [ ] Acceptance criterion 1
    - [ ] Acceptance criterion 2

# Edit the template to customize it
$ vim .beads/templates/performance.yaml
```

## Using Templates with `issuegraph create`

Use the `--from-template` flag to create issues from templates:

```bash
issuegraph create --from-template <template-name> "Issue title"
```

Template values can be overridden with explicit flags:

```bash
# Use bug template but override priority
issuegraph create --from-template bug "Login crashes on special chars" -p 0

# Use epic template but add extra labels
issuegraph create --from-template epic "Q4 Infrastructure" -l infrastructure,ops
```

**Examples:**

```bash
# Create epic from template
$ issuegraph create --from-template epic "Phase 3 Features"
✓ Created issue: bd-a3f8e9
  Title: Phase 3 Features
  Priority: P1
  Status: open

# Create bug report from template
$ issuegraph create --from-template bug "Auth token validation fails"
✓ Created issue: bd-42bc7a
  Title: Auth token validation fails
  Priority: P1
  Status: open

# Use custom template
$ issuegraph template create security-audit
$ issuegraph create --from-template security-audit "Review authentication flow"
```

## Template File Format

Templates are YAML files with the following structure:

```yaml
name: template-name
description: |
  Multi-line description with placeholders
  
  ## Section heading
  
  [Placeholder text]

type: bug|feature|task|epic|chore
priority: 0-4
labels:
  - label1
  - label2

design: |
  Design notes structure

acceptance_criteria: |
  - [ ] Acceptance criterion 1
  - [ ] Acceptance criterion 2
```

## Built-in Templates

### epic

For large features composed of multiple issues.

**Structure:**
- Overview and scope
- Success criteria checklist
- Background and motivation
- In-scope / out-of-scope sections
- Architecture design notes
- Component breakdown

**Defaults:**
- Type: epic
- Priority: P1
- Labels: epic

### bug

For bug reports with consistent structure.

**Structure:**
- Summary
- Steps to reproduce
- Expected vs actual behavior
- Environment details
- Root cause analysis (design)
- Proposed fix
- Impact assessment

**Defaults:**
- Type: bug
- Priority: P1
- Labels: bug

### feature

For feature requests and enhancements.

**Structure:**
- Feature description
- Motivation and use cases
- Proposed solution
- Alternatives considered
- Technical design
- API changes
- Testing strategy

**Defaults:**
- Type: feature
- Priority: P2
- Labels: feature

## Custom Templates

Custom templates override built-in templates with the same name. This allows you to customize built-in templates for your project.

**Priority:**
1. Custom templates in `.beads/templates/`
2. Built-in templates

**Example - Override bug template:**

```bash
# Create custom bug template
$ issuegraph template create bug

# Edit to add project-specific fields
$ cat > .beads/templates/bug.yaml << 'EOF'
name: bug
description: |
  ## Bug Report
  
  **Severity:** [critical|high|medium|low]
  **Component:** [auth|api|frontend|backend]
  
  ## Description
  [Describe the bug]
  
  ## Reproduction
  1. Step 1
  2. Step 2
  
  ## Impact
  [Who is affected? How many users?]

type: bug
priority: 0
labels:
  - bug
  - needs-triage

design: |
  ## Investigation Notes
  [Technical details]

acceptance_criteria: |
  - [ ] Bug fixed and verified
  - [ ] Tests added
  - [ ] Monitoring added
EOF

# Now 'issuegraph create --from-template bug' uses your custom template
```

## JSON Output

All template commands support `--json` flag for programmatic use:

```bash
$ issuegraph template list --json
[
  {
    "name": "epic",
    "description": "## Overview...",
    "type": "epic",
    "priority": 1,
    "labels": ["epic"],
    "design": "## Architecture...",
    "acceptance_criteria": "- [ ] All child issues..."
  }
]

$ issuegraph template show bug --json
{
  "name": "bug",
  "description": "## Summary...",
  "type": "bug",
  "priority": 1,
  "labels": ["bug"],
  "design": "## Root Cause...",
  "acceptance_criteria": "- [ ] Bug no longer..."
}
```

## Best Practices

1. **Use templates for consistency**: Establish team conventions for common issue types
2. **Customize built-ins**: Override built-in templates to match your workflow
3. **Version control templates**: Commit `.beads/templates/` to share across team
4. **Keep templates focused**: Create specific templates (e.g., `performance`, `security-audit`) rather than generic ones
5. **Use placeholders**: Mark sections requiring input with `[brackets]` or `TODO`
6. **Include checklists**: Use `- [ ]` for actionable items in description and acceptance criteria

## See Also

- [issuegraph create](create.md) - Create issues
- [issuegraph list](list.md) - List issues
- [README](../README.md) - Main documentation
