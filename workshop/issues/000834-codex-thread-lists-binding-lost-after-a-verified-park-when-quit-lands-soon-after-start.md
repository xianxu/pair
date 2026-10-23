---
id: 000834
status: open
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
severity: medium
collaboration-mode: collaboration
card_mirror: '8b4a2796f1265245db2b2366f0863f93969a83b3' # card fields mirrored from issue-cards; edit via sdlc
---

# Codex thread lists binding lost after a verified park when quit lands soon after start

## Problem

## Spec

Found during #827's native full-quit qualification (`make test-couch-fleet`,
`TestDisposableCouchFleetFullQuit`). After one confirmed Alt+x parks a Codex
fleet, every thread record carries `VerifiedPark` with no live incarnation and
no open park, yet `couch --list` can show a Codex row as
`Start fresh: binding lost — repairable` instead of
`Resume from disk (no agent running)`. Claude rows did not show it in the
same runs.

Trace (2026-10-23, branch 000827 at 81b47b374..289ef0c5d, private disposable
fleet):
- Run 2 (codex, 4 slots, quit 2.50s): fixture:3 listed `Start fresh: binding
  lost — repairable`; fixture, fixture:1, fixture:2 listed resumable.
- Run 3 (codex, 4 slots, quit 0.53s; codex, 8 slots, quit 1.40s): one row each
  listed binding lost while the store showed `VerifiedPark` for every record.
- Run 1 (codex, 4 slots, quit 0.43s): all rows resumable.
- The quit lands seconds after `StartWorker` returns; each fixture has already
  written its rollout transcript (`agentfixture` Ready.Transcript) by then.

Classifier: `actionableinventory.go` returns `ThreadUnusable, ReasonBindingLost`
when `VerifiedPark != nil` but `parkedResumeProofMatches` fails, i.e. the parked
resume proof does not match the conversation Couch can resolve. Hypothesis
(unverified): Couch had not yet discovered or bound the Codex native session
when the park captured its resume proof, so the park preserved no matching
binding, although the transcript exists on disk.

Question: is a just-started Codex thread parked before binding discovery
expected to lose resume, or should park (or discovery) bind the existing
transcript first? Either way, the operator sees "binding lost" for a thread
whose conversation exists.

## Done when

- The cause is measured (the binding state at park time for the affected row,
  versus the transcript on disk), not inferred.
- Either a Codex thread with an existing transcript parks resumable, with a
  fixture-backed regression replayed in the normal suite, or the behaviour is
  documented as intended, with the operator-facing label saying why.

## Plan

- [ ]

## Log

Filed from pair#827 (pair:6); unclaimed for TL dispatch.
