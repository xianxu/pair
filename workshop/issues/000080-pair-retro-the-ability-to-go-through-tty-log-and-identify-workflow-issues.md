---
id: 000080
status: open
deps: []
github_issue:
created: 2026-06-26
updated: 2026-06-26
estimate_hours:
---

# pair retro: the ability to go through tty log and identify workflow issues

## Problem

At the end of a session, Pair should be able to go through an agent transcript and identify things that did not work well for that agent. Prefer the TTY log as the source because it is more portable across agents than each agent's native transcript format.

The retro should find concrete workflow/tooling problems: base-layer bugs, agent transcript format drift, tool-call errors, SDLC process friction, permission/environment mismatches, and cases where a rigorous process becomes form without essence.

## Spec

Build a Go script or CLI that reads a Pair TTY log, inspects the work and errors in the session, and suggests ways to improve tooling or instructions.

Initial detection areas:

1. Tool-call errors that could be avoided by better instructions, wrappers, PATH setup, command help, or tool behavior.
2. SDLC workflow friction, especially cases where the agent bypasses the intended process or the process is mechanically followed but misses the underlying purpose.
3. Base-layer compatibility problems, such as transcript format changes, unsupported agent behavior, or environment/permission mismatches.
4. Review/close loops that produce excessive output, invalid intermediate state, or repeated attempts for a predictable reason.

## Done when

- [ ] A Go CLI can analyze a rendered Pair TTY log and emit a markdown retro report.
- [ ] The first version detects tool errors, SDLC friction, and review/close failures from deterministic patterns.
- [ ] The report includes evidence snippets or line references from the log.
- [ ] The implementation has fixtures covering at least one real Pair session failure pattern.
- [ ] The CLI leaves the original log untouched.

## Plan

- [ ] Define the input format: raw TTY log vs `pair-scrollback-render --plain` output.
- [ ] Create a small fixture from a real session log with known workflow issues.
- [ ] Implement deterministic detectors for tool errors and SDLC/review friction.
- [ ] Emit a concise markdown report with evidence and suggested follow-ups.
- [ ] Add tests for the fixture and detector categorization.

## Log

### 2026-06-26

Seeded from the product idea: a Pair session retro should inspect the portable TTY log and surface workflow/tooling problems that are currently only visible after manually reading the transcript.
