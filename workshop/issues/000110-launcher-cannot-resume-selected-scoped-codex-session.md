---
id: 000110
status: working
deps: []
github_issue:
created: 2026-07-07
updated: 2026-07-07
estimate_hours: 0.5
started: 2026-07-07T21:15:56-07:00
---

# launcher cannot resume selected scoped codex session

## Problem

After #107, bare `pair` correctly shows current-repo scoped tag + agent rows,
but resuming a selected historical Codex row can fail at the saved-session step:
the tag/agent is visible, yet the config picker does not find the associated
Codex session to resume.

Root cause found during debugging: `OSRuntime.AgentSessionExists` checks Codex
session files only directly under `~/.codex/sessions`, while Codex writes nested
date paths such as `~/.codex/sessions/YYYY/MM/DD/rollout-...<sid>.jsonl`.

## Spec

Codex session existence checks must recognize the nested rollout path shape that
the rest of Pair already uses for Codex transcript discovery. Keep the change at
the launcher runtime boundary and reuse the existing path convention
(`ARCH-DRY`, `ARCH-PURE`, `ARCH-PURPOSE`).

The intended startup flow is: bare `pair` scans the current repo's scoped
history, lets the user choose a tag + agent row such as `pair/pair-misc  codex`,
then the tag-restart config picker recognizes the saved Codex session id and
offers a resume option. The fix should not invent a new ledger model or picker
path; it should make the existing saved-config/ledger path correctly validate
Codex's on-disk native session artifact.

## Done when

- A regression test fails on the current direct-child Codex glob and passes for
  a nested `~/.codex/sessions/YYYY/MM/DD/rollout-...<sid>.jsonl` file.
- A saved Codex config/ledger entry with that sid is considered resumable, so
  the tag-restart picker can offer "use saved params + session".
- Launcher tests pass.

## Estimate

```estimate
model: estimate-logic-v3.1
item: smaller-go-module design=0.10 impl=0.25
item: milestone-review design=0.00 impl=0.15
total: 0.50
```

## Plan

- [x] Add a failing test for nested Codex session existence.
- [x] Update the Codex native-session probe to search the nested rollout shape.
- [x] Run focused launcher tests and a wider regression command.

## Log

### 2026-07-07
- Created and claimed from a live report: after exiting a `pair-misc` Codex
  session, bare `pair` can select the scoped tag/agent but then finds no
  historical session to attach/resume. Investigation points to Codex
  `AgentSessionExists` using a non-recursive glob, unlike the actual nested
  Codex session layout.
- RED: `go test ./cmd/internal/launcher -run
  TestOSRuntimeAgentSessionExistsFindsNestedCodexRollout -count=1` failed
  because `AgentSessionExists(codex)` did not find the nested rollout file.
- GREEN: changed Codex native-session existence to reuse
  `transcript.Resolve("codex", ...)` (`ARCH-DRY`, `ARCH-PURE`), then the focused
  regression and `go test ./cmd/internal/launcher -count=1` passed.
- Wider verification passed: `go test ./...`; whitespace verification passed:
  `git diff --check`.
