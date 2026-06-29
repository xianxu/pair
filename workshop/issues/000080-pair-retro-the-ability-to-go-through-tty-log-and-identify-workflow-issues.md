---
id: 000080
status: open
deps: []
github_issue:
created: 2026-06-26
updated: 2026-06-29
estimate_hours:
---

# pair retro: the ability to go through tty log and identify workflow issues

## Problem

At the end of a session, Pair should be able to go through an agent transcript
and identify things that did not work well for that agent. Prefer the rendered
TTY log as the source because it is more portable across agents than each
agent's native transcript format.

The retro should find concrete workflow/tooling problems: base-layer bugs, agent
transcript format drift, tool-call errors, SDLC process friction,
permission/environment mismatches, and cases where a rigorous process becomes
form without essence.

The useful output is not a generic session summary. The useful output is an
evidence-backed list of process/tooling frictions that can become follow-up
issues, instruction changes, or binary improvements.

## Spec

Build a retro skill backed by a binary so the operator can simply ask an agent
to "run a retro" and get a consistent workflow:

- `pair retro <log>` analyzes a rendered Pair TTY log.
- `pair retro current` analyzes the current session's rendered scrollback when
  the needed live paths are available.
- The skill explains when to run the binary, how to interpret the report, and
  how to turn report items into follow-up issues.
- The binary emits a markdown retro report. It should include concrete evidence
  snippets or line references, a classification, and a suggested follow-up.

Initial integration should be manual. This should not be a mandatory
`sdlc close` gate in the first version. Later, repo-level config can choose how
strongly to integrate retros:

- `manual`: retro exists as an operator-invoked command.
- `suggest`: SDLC close/merge prints a hint when no retro report exists.
- `required`: creator/base-layer repos can require a report or an explicit
  `--no-retro` acknowledgement.

If SDLC integration is added, it should be a soft close adjunct by default:
help operators remember to run the retro, but do not impose self-improvement
ceremony on repos that do not want it.

Initial detection areas:

1. Tool-call errors that could be avoided by better instructions, wrappers, PATH setup, command help, or tool behavior.
2. SDLC workflow friction, especially cases where the agent bypasses the intended process or the process is mechanically followed but misses the underlying purpose.
3. Base-layer compatibility problems, such as transcript format changes, unsupported agent behavior, or environment/permission mismatches.
4. Review/close loops that produce excessive output, invalid intermediate state, or repeated attempts for a predictable reason.

## Done when

- [ ] A retro skill documents the operator workflow for "run a retro".
- [ ] A Go CLI can analyze a rendered Pair TTY log and emit a markdown retro report.
- [ ] `pair retro <log>` works for an explicit rendered log path.
- [ ] `pair retro current` works when the current session's Pair paths are available.
- [ ] The first version detects tool errors, SDLC friction, and review/close failures from deterministic patterns.
- [ ] The report includes evidence snippets or line references from the log.
- [ ] The implementation has fixtures covering at least one real Pair session failure pattern.
- [ ] The CLI leaves the original log untouched.
- [ ] SDLC integration is left manual or config-gated; no unconditional close gate is added.

## Plan

- [ ] Define the input format: raw TTY log vs `pair-scrollback-render --plain` output.
- [ ] Define the retro report markdown schema: finding, evidence, impact, suggested follow-up.
- [ ] Add a retro skill that invokes the binary and explains report handling.
- [ ] Create a small fixture from a real session log with known workflow issues.
- [ ] Implement deterministic detectors for tool errors and SDLC/review friction.
- [ ] Add `pair retro <log>` and `pair retro current` entry points.
- [ ] Emit a concise markdown report with evidence and suggested follow-ups.
- [ ] Add tests for the fixture and detector categorization.
- [ ] Leave SDLC close integration as a documented future/configured mode, not a default requirement.

## Log

### 2026-06-26

Seeded from the product idea: a Pair session retro should inspect the portable TTY log and surface workflow/tooling problems that are currently only visible after manually reading the transcript.

### 2026-06-29

Revised scope after dogfooding #84 and discussing whether this belongs in
`sdlc close`. Decision: build a retro skill backed by a binary (`pair retro`)
first, emitting markdown reports from rendered TTY logs. Treat SDLC integration
as a later repo-configured adjunct (`manual` / `suggest` / `required`) rather
than an unconditional close gate.
