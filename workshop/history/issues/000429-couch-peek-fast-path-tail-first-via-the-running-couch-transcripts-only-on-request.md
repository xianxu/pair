---
id: 000429
status: done
deps: []
github_issue:
created: 2026-10-10
updated: 2026-10-10
estimate_hours:
card_mirror: 'b3241d4f9c1bf63f767c311ee172ee749b931520' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-10T13:37:15-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
actual_hours: 0.86
---

# couch --peek fast path: tail first via the running Couch, transcripts only on request

## Problem

pair#425 put each slot's tail in wrapper memory so a TL can read "the last N lines of all slots" instantly. Live check (10-10, ops learnings kink 48): the tail reads `source live` with marks, but a single-slot `couch --peek` takes 3.1–3.4s, two slots 1.9s, four 2.4s, against #425's Done-when of well under a second. For comparison, `couch --message-status` (a socket round trip to the running Couch) takes 0.01s, and `couch --list` 0.9s.

Where the time goes (code read):
- `couchcore/peek.go:143` `PeekThread` always runs `SwitchContext.Resolve` (pair prompt log, native transcripts, session binding) **before** the tail read, even when the live tail succeeds. That's the "sent prompts / transcript / native session" footer.
- Each peek is a cold typed operation (`couchcmd/run.go` `runTypedOperationWithConsole`: prepare runtime, namespace, store), about 0.9s before any work.
- The single-reference path also resolves the workspace reference and repository scope, so one slot is slower than two.

## Spec

1. **Tail first and only by default.** Read the live tail; resolve transcripts only with `--transcripts`, or when the live tail fails and the recording fallback needs the agent.
2. **A fast path through the running Couch:** the CLI asks the live Couch over its socket for the slots' tails (as `--message-status` does), with no cold runtime build. Fall back to today's path when no Couch is running.
3. Single- and multi-slot peeks take the same path.

## Done when

- `couch --peek pair:1:2:3,ariadne:1:2:3 --lines 10` returns live tails for 6 slots in under 200ms (measured on the live workbench, binary not shell function), and a single slot in under 100ms. *(Live timing deferred to the TL at rollout — see Revisions.)*
- `--transcripts` restores today's footer; a failed live tail still falls back to the recording and says why.
- A test asserts the default peek does no transcript resolution.

## Plan

Durable plan: `workshop/plans/000429-couch-peek-fast-path-tail-first-via-the-running-couch-transcripts-only-on-request-plan.md`.

- [x] Broker `tail` by slot reference (`Target`, resolved like `--send-to`), answering the thread's tag and agent.
- [x] `PeekThread` transcript-free by default; `--transcripts` restores the footer; record agent for the recording fallback.
- [x] CLI fast path: every `repo:N` ref read concurrently over the broker socket, no Couch build; anything else runs the typed path.
- [x] Docs (atlas/couch.md, couch skill, README).

## Revisions

### 2026-10-10 — Done-when 1's live timing deferred to the TL (ops decision)

- **Reason:** the fast path needs the running Couch on the new build: the broker side (`tail` by slot) lives in Couch. The live Couch predates it and refuses the request, and this slot can't restart Couch. Ops chose to defer, as for pair#421 and pair#425.
- **Delta:** Done-when 1 is checked by the TL at rollout. Restart Couch on the new build (no wrapper relaunch: the wrapper side is #425's unchanged `tail` endpoint), then time `couch --peek pair:1:2:3,ariadne:1:2:3 --lines 10` and `couch --peek pair:1 --lines 10` with the binary. The issue isn't counted done until that passes. Evidence available now: the component measurements and tests in the Log.

### 2026-10-10 — close round 1 findings (FIX-THEN-SHIP)

- **BR-1 (fixed, class: fast and typed peek must answer alike):** the broker now reads the thread record (`authority.record`) and names the tail's agent (`RecordAgent`) and working path exactly as the typed peek does (`TailThread.WorkingPath`). `TestFastPeekMatchesTheTypedPeek` peeks one thread both ways and requires byte-identical JSON (mutation-checked: dropping the path fails it).
- **Plan item 4 prose:** the recording fallback never calls the resolver. `RecordAgent` is the only agent source, and the resolver would return the same value. Default peeks make no `Resolve` call, as the test asserts.
- **Alias and prefix refs:** the fast path resolves a slot as `--send-to` does, so `pa:1` works there. The typed fallback resolves through `ResolveThreadReference`, so the fast path accepts a superset; a ref the fallback can't resolve says so in `unavailable`. Accepted, because the shortcut matches the messaging verbs.
- **Older-Couch fallback:** now tested (`an older couch` case: the refusal leads to the typed peek, with nothing written).
- **Flag re-parse in `fastPeek`:** kept. The flags arrive pre-normalized by `ParseCLI` (`--lines=N`, `--json`, `--transcripts`), and anything unrecognized falls back to the typed path, so a mismatch can't produce a wrong answer, only a slower one.

## Log

### 2026-10-10
- 2026-10-10: closed — Round 2: BR-1 fixed (broker names agent+working_path from the thread record; TestFastPeekMatchesTheTypedPeek requires byte-identical JSON both ways, mutation-checked), older-Couch fallback tested; minors disposed in issue Revisions. Earlier: TestDefaultPeekResolvesNoTranscripts (Done-when 3), TestPeekAnswersFromTheRunningCouch (argv over real socket, Couch build forbidden). make -k test (known TMPDIR/env noise only, reruns pass), go test ./... 84 ok + 2 pre-existing (artifactpath, gcruntime); couchcmd ColdResume flake fails on main too. Done-when 1 live timing deferred to TL at rollout per ops (Revision).; review verdict: SHIP
- 2026-10-10: flow upgraded quick → full — 307 added lines in code files (limit 100); an earlier round of this close already ran the full review

- Design: `Binding.Slot` already names each connected wrapper's `repo:N`, so the broker maps a slot ref to its wrapper with no store read; the CLI fast path is one socket round trip per slot, like `--message-status`.
- Measured where the cold peek's time goes (temporary timers in `runTypedOperationWithConsole`, live, single slot `pair:1`): ~1.55s in the router's `WorkspaceReferencePath`, ~1.64s in the executor (`resolveOperationThread` → `ResolveThreadReference` for `repo:N`, plus the tail). Resolving a `repo:N` to its thread costs ~1.5s each time; the transcript resolution the issue suspected is cheap here (skipping it changed 3.3s little). The fast path avoids both; the typed fallback still pays them (follow-up candidate: the slot resolver's cost, not this issue's).
- Live, against today's Couch (pre-#429 build): the by-slot `tail` is refused ("tail takes only a thread and a line count") in 0.2–0.4ms round trip, and the CLI falls back to the typed peek with correct live output: 3.3s single, 1.7s for four. The Done-when timing needs the running Couch on the new build.
- Measured components of the fast path: binary start + runtime prep ~16ms (a refused run, end to end), socket round trip 0.2–0.4ms (live Couch), end-to-end test over a real socket (`TestPeekAnswersFromTheRunningCouch`) <1ms. A stub broker couldn't stand in for the live one end to end: the machine-wide Couch selection pins the store root.
- Tests: `TestDefaultPeekResolvesNoTranscripts` (Done-when 3: no `Resolve` call by default, including the recording fallback; one with `--transcripts`), `TestResolveTailSlot`, `TestTailBySlotReadsThatSlotsWrapper`, `TestFastPeekAnswersOnlyWhenEverySlotIsLive` (fallback on a failed slot, a non-slot ref, `--transcripts`, over-200 or bad `--lines`; nothing written), `TestPeekAnswersFromTheRunningCouch` (argv → real socket, a runtime that fails any Couch build; mutation-checked: disabling the fast path fails it).

