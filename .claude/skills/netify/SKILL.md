---
name: netify
description: Orchestrate feature work, bug fixes, refactors, and engineering tasks with focused skills and coding agents. Use when a task needs a coordinated workflow across planning, implementation, issue tracking, code graphs, architecture, design, or review.
license: MIT
metadata:
  version: "0.1.0"
---

# Netify

Route engineering work through the smallest useful set of skills and agents. Keep one accountable lead, one shared scope, and evidence for every completion claim.

Use for code or project work that benefits from routing among Laurenify, Karpathify, Mattify, ReasonGraph, CodeGraph, IssueGraph, ReviewGraph, Designify, Birdify, or Specify. Do not load every tool for every task. Follow [agent-routing.md](references/agent-routing.md) to select only relevant capabilities.

## Workflow

1. **Frame the task.** Read repository instructions and current status. State the requested outcome and observable done conditions. Distinguish facts from assumptions. Ask only about a missing choice that blocks safe progress.
2. **Route capabilities.** Read `references/agent-routing.md`. Activate only matching skills that are installed or explicitly available. A skill name is not proof the skill or its tools are available. Never invent commands, MCP tools, agent results, or repository history.
3. **Recover project context.** For code changes, consult ReasonGraph before editing when available. For multi-session work, blockers, dependencies, or context recovery, use IssueGraph. Use CodeGraph and ReviewGraph only when their CLI/MCP integration is present; graph output narrows investigation but never replaces source and test inspection.
4. **Honor project gates.** For persistent engineering changes, use Specify to deliver and validate one focused RFC before implementation. Present the concrete plan and wait for explicit approval when the repository workflow requires it. If Specify is required but unavailable, stop before implementation and report the missing dependency. When Birdify is active, complete its rendered map and displayed change plan, then wait for confirmation unless the user explicitly waived that gate for this task. Apply Designify's matching workflow without duplicating its RFC or creating a parallel plan.
5. **Choose agent shape.** Delegate only when agent tools exist and work can be separated by clear ownership. Use one lead to own scope, integration, and final verification. Give each worker a bounded task, exact files or output path, dependencies, acceptance criteria, and report format. Keep tightly coupled edits with one implementer. If delegation is unavailable or adds coordination cost, execute sequentially and say so.
6. **Implement in small units.** Use the selected debugging, design, or TDD method. Read relevant implementation and tests before editing. Preserve existing behavior and user changes. Do not expand scope based on neighboring issues or graph suggestions.
7. **Review and verify.** Reconcile worker changes, inspect the complete diff, run relevant project checks, and use ReviewGraph when available for graph-informed change review. Verify the requested behavior directly. Record important non-obvious decisions in ReasonGraph when available. Report exact checks and failures; never infer a pass.
8. **Close the loop.** Update IssueGraph tasks when used. Summarize changed areas, agent contributions, verification evidence, unresolved risks, and remaining work. Do not commit, push, merge, deploy, or publish unless requested or already authorized.

## Agent contract

- Lead agent owns the task outcome and all cross-worker integration.
- Workers do not edit the same files concurrently. If overlap becomes necessary, serialize ownership.
- Workers return changed paths, decisions, checks run with results, and unresolved questions. Lead independently inspects integrated changes.
- Parallelize independent investigation or implementation slices only. Do not spawn agents to imitate tools that are unavailable.
- Keep user-facing updates concise and factual. Surface blockers early; continue independent work while blocked where safe.

## Completion rule

Call work complete only when acceptance conditions are checked against the result. State which checks ran, which did not, and why. Packaging, a successful agent response, or a clean-looking diff alone does not prove behavior.
