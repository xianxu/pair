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

Codex resume detection must also recognize Codex's own CLI grammar:
`codex [OPTIONS] resume <session>`. A manual or saved Pair invocation such as
`pair-dev codex -- --sandbox danger-full-access resume <sid>` must be treated as
an explicit resume and must persist clean args with the resume binding stripped,
because `session_id` is the canonical stored binding (`ARCH-PURPOSE`,
`ARCH-DRY`).

## Done when

- A regression test fails on the current direct-child Codex glob and passes for
  a nested `~/.codex/sessions/YYYY/MM/DD/rollout-...<sid>.jsonl` file.
- A saved Codex config/ledger entry with that sid is considered resumable, so
  the tag-restart picker can offer "use saved params + session".
- Codex explicit resume is detected and stripped when valid Codex global options
  precede `resume <sid>`, preventing duplicate resume tokens in saved config.
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
- [x] Add a failing test for Codex global options before `resume <sid>`.
- [x] Update Codex resume detection/persistence to parse that valid argv shape.
- [x] Run focused launcher tests and a wider regression command.

## Revisions

- 2026-07-07T21:45:00-07:00 — live retest showed a second root cause: the first
  failed bare `pair` run likely used the pre-fix binary, and the workaround
  `pair-dev codex -- --sandbox danger-full-access resume <sid>` exposed that
  Pair only recognized Codex `resume <sid>` when it was args[0..1]. Expand scope
  to cover Codex global options before the `resume` command.

## Log

### 2026-07-07
- 2026-07-07: closed — Fixed both #107 resume regressions: Codex native session existence uses transcript.Resolve for nested rollout files, and Codex resume detection now supports valid 'codex [OPTIONS] resume <sid>' argv while stripping that binding from saved config args. RED/GREEN covered nested rollout discovery, global-options-before-resume explicit detection, persisted-arg stripping, and an already-polluted saved config composing exactly one resume token. Verified with focused launcher tests, go test ./cmd/internal/launcher -count=1, go test ./..., git diff --check, and rebuilt bin/pair via make build for live retest. No atlas update: this corrects existing documented session identity behavior, no new surface.; review verdict: FIX-THEN-SHIP
- 2026-07-07: closed — Root cause fixed by reusing transcript.Resolve for Codex nested rollout session discovery; RED verified with go test ./cmd/internal/launcher -run TestOSRuntimeAgentSessionExistsFindsNestedCodexRollout -count=1 before fix; GREEN verified with focused regression, go test ./cmd/internal/launcher -count=1, go test ./..., and git diff --check. No atlas update: existing atlas/session-identity.md already documents Codex session identity/storage and this only corrects a runtime probe to match it.; review verdict: SHIP
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
- Reopened from `codecomplete` after live retest. Evidence from
  `/Users/xianxu/.local/share/pair/repos/e108517d46ab4575`: config and ledger
  recorded `session_id=019f3feb-29f5-7940-a976-cb1d1ce13d0f`, but the manual
  workaround persisted args containing `--sandbox danger-full-access resume
  <sid> --no-alt-screen`, proving Codex resume detection/stripping missed valid
  global-options-before-command argv.
- RED: focused launcher argument tests failed for `--sandbox danger-full-access
  resume <sid>`: explicit resume was not detected and persisted config args kept
  the embedded resume token. GREEN: Codex resume command scanning now consumes
  known Codex global options before `resume <sid>` and is reused by explicit
  resume detection plus persisted-arg stripping (`ARCH-DRY`).
- Added live-shape regression for an already-polluted saved Codex config; the
  tag-restart picker now composes exactly `resume <sid> --sandbox
  danger-full-access --no-alt-screen`. Verification passed:
  `go test ./cmd/internal/launcher -run
  'TestStripCodexResumeSubcommand|TestExtractExplicitResumeCodexAllowsGlobalOptionsBeforeCommand|TestPersistedConfigArgsStripsBinding|TestExtractExplicitResume|TestOSRuntimeAgentSessionExistsFindsNestedCodexRollout'
  -count=1`, `go test ./cmd/internal/launcher -count=1`, `go test ./...`, and
  `git diff --check`.
- Boundary review returned FIX-THEN-SHIP for one stale atlas consumer:
  `atlas/architecture.md` still described Codex stripping as args[0..1] only.
  Updated it to document `codex [OPTIONS] resume <sid>`.
