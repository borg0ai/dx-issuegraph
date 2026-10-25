# Agent and skill routing

Select by task need. Load each selected skill's own instructions before following its workflow. These tools evolve; their installed instructions and current CLI help outrank this routing summary.

| Capability | Route when | Boundary |
|---|---|---|
| Laurenify | Need disciplined investigation, architecture, TDD, review, or independent agent swarm | Pick one focused skill such as `how`, `architect`, `tdd`, `thermo-nuclear-code-quality-review`, or `swarm`. Follow that skill's platform mapping and agent protocol. Do not invoke a swarm for serial or tightly coupled work. |
| Karpathify | Need assumption control, minimal implementation, surgical edits, or measurable done conditions | Apply its Think Before Coding, Simplicity First, Surgical Changes, and Goal-Driven Execution principles. Use matching installed skills such as `methodology`, `agentic-engineering`, `minimalism`, or `diagnosing-bugs`. |
| Mattify | Need engineering method for debugging, TDD, or forming a precise spec | Prefer `diagnosing-bugs`, `tdd`, or `to-spec` from `skills/engineering`. Keep its method small and composable; do not let it bypass repository RFC gates. |
| Specify | Any persistent engineering change in an RFC-governed workflow; required by Netify for persistent code changes | Deliver and validate one concern before implementation. Use Specify CLI for lifecycle changes. Wait for required approval. Do not hand-edit its managed index or tracking files when a CLI command exists. |
| Designify | Work governed by Designify's RFC and task workflow, including approved feature implementation, debugging, and verification | Read `designify` plus the matching workflow reference. RFC is the plan; TDD is implementation method, not another approval route. |
| Birdify | User explicitly invokes Birdify, asks for architecture/constraint/change map, or project instructions configure auto mode | Follow its constraint discovery, map rendering and visual review. For code work, show the rendered change plan and wait for approval unless the user explicitly waived that gate for this task. |
| ReasonGraph | Before code changes to retrieve prior decisions; after a consequential non-obvious decision | Run the installed ReasonGraph context/sync workflow. If unavailable, inspect `.reasongraph/why/` directly; never claim a decision was recorded without evidence. |
| CodeGraph | Semantic code navigation is useful and its CLI/MCP is already installed | Use its connected MCP tools or documented CLI. Its index is navigation aid, not source-of-truth. Do not install or reconfigure agents as an incidental task. |
| ReviewGraph | Need graph-informed bug tracing, impact analysis, or change review and MCP tools are connected | For debug/review skills, fetch minimal context first, use targeted graph queries, then inspect source/tests. If MCP is absent, continue with repository tools and disclose unavailable graph review. |
| IssueGraph | Work spans sessions, has dependencies/blockers, or needs recovery after context loss | Check `prime`/`ready` guidance, claim and update tasks using its CLI. Do not initialize or write tracker data unless project setup and user task authorize it. |

## Default sequence

For a normal code change, use this order and omit inapplicable steps:

1. **Understand:** repository instructions and status; ReasonGraph context; targeted CodeGraph or ReviewGraph navigation when available; inspect source and tests.
2. **Specify:** deliver and validate a single RFC; include acceptance conditions and task-level implementation steps in the repository's prescribed location.
3. **Review scope:** if Birdify or Designify applies, follow its rendered plan and approval gates. Do not create duplicate plans.
4. **Implement:** use Mattify or Laurenify debugging/TDD method selected for the problem. Split work across agents only at independent ownership boundaries.
5. **Check:** run relevant tests and project checks; inspect integrated diff; ask ReviewGraph for risk/impact when available; fix verified issues.
6. **Record and close:** sync meaningful rationale to ReasonGraph; update IssueGraph if used; advance/archive RFC through Specify where required.

## Task sizing

- **Small, one-file fix:** one lead; use focused debugging/TDD guidance. Skip swarm. Follow mandatory project RFC and active Birdify gates.
- **Feature with separable modules:** one lead plus workers with non-overlapping ownership. Keep RFC approval and integration with lead.
- **Cross-cutting or multi-session work:** use IssueGraph dependencies; use Designify workflow if configured; delegate investigation and implementation only after boundaries are explicit.
- **Review-only request:** do not edit. Use ReviewGraph when available and inspect the actual diff/source. Return findings with risk, evidence, and recommendation.
- **Unknown or missing tool:** continue with available repository evidence when possible; disclose the missing capability. Stop only if that tool enforces a required approval or repository gate.
