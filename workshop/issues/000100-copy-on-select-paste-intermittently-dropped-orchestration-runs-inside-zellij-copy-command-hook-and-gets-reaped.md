---
id: 000100
status: codecomplete
deps: []
github_issue:
created: 2026-07-05
updated: 2026-07-05
estimate_hours: 1.16
started: 2026-07-05T10:48:20-07:00
plan: workshop/plans/000100-copy-on-select-detach-plan.md
actual_hours: 0.81
---

# copy-on-select paste intermittently dropped — orchestration runs inside zellij copy_command hook and gets reaped

## Problem

Copy-on-select → paste into the nvim draft pane intermittently drops the text.
The source pane flashes green (copy half succeeds) but nothing is inserted.
Worse since the Go migration (#93 M4); operator reports the **first copy after a
pair restart fails, subsequent copies work**.

## Spec

**Root cause (evidence-backed).** The entire paste orchestration runs *inside*
zellij's `copy_command` child (`copy-on-select` → `exec clipboard-to-pane`), and
zellij reaps that child (uncatchable SIGKILL) when it outlives a short deadline
(~1s). The chain is slow: each `zellij action` cold-starts a zellij client
(~350–400ms measured) and the chain makes ~5 of them (list-panes ×2,
set-pane-color, focus-pane-id, write) plus three Go-binary cold-starts → ~1.5–2s.

Diagnostic instrumentation (timestamps + per-call timing + signal logger, added
to `cmd/internal/clipcmd/{runtime,runcli}.go`) proved it:
- Failing runs die at a **variable** point (once deep in clipboard-to-pane's
  pbpaste/list-panes window; once after the flash, before exec'ing
  clipboard-to-pane) with **no catchable signal logged** → time-based SIGKILL
  reap, not a code bug in one call. `ps` shows nothing hung (killed, not stuck).
- A single `zellij action list-panes` measured **395ms**; total was already
  761ms and climbing before clipboard-to-pane even started.

**Why "first copy after restart":** dev mode (`pair-dev` → `PAIR_DEV`) runs
`DevRebuild` (`make build`, whole ~19-binary fleet) on every session
create/restart (`createflow.go:281`). The first copy then execs *freshly-built*
binaries → macOS first-run scan + cold page-in (~130–150ms/binary measured) +
cold zellij client → over the deadline → reaped. Subsequent copies run warm →
under deadline → work. In release (`pair`, no rebuild) binaries stay warm so it
works — but a slow machine on a cold cache could still be reaped (the deadline is
fixed while the in-hook chain's duration is machine/cache dependent).

**Fix — root cause, not prewarm.** Make the `copy_command` hook return to zellij
immediately after mirroring the selection to the clipboard, and run the
flash+paste in a **detached (`setsid`) process** that outlives the hook — the
idiom `ResetPaneColorAfter` (runtime.go:92) and the launcher's `spawnDetached`
already use. Deadline-independent: correct on any machine, cold or warm. Chosen
over prewarm (which only narrows the cold window and stays machine-speed
dependent — the exact fragility the operator flagged for slow machines).

Prior (wrong) hypotheses ruled out by evidence: focus→write race (chain dies
before the write); clipboard round-trip (pbpaste reads fine, log shows clip
bytes read); nvim insert-mode gate (unchanged from bash, not the regression).

## Estimate

Decomposition (v3.1: design hours from the v2.1 primitive table; `impl=` at 40%
of the v2.1 implementation hours; +15% design buffer since a thorough plan doc
exists). The `smaller-go-module` item is bumped above the typical baseline
because it spans ~6 files (the `clipcmd` hook/orchestrator split + `SpawnDetached`
seam + `--orchestrate` dispatch + the Go fake-runtime tests + the async shell
test + removal of the temporary instrumentation).

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module design=0.30 impl=0.45
item: atlas-docs design=0.10 impl=0.05
item: milestone-review design=0.00 impl=0.20
design-buffer: 0.15
total: 1.16
```

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

## Done when

- Copy-on-select's `copy_command` hook does only the clipboard mirror + a
  detached spawn, then returns — the flash + paste run detached and survive a
  reap of the hook's process group.
- First copy after a `pair-dev` restart inserts reliably (cold path).
- The temporary `[trace]` instrumentation is removed.
- A test covers the detach: the hook does not run the slow paste chain inline
  (e.g. fake Runtime asserts the orchestration is spawned detached, not exec'd
  in-process).

## Plan

- [x] Design the detach (self-exec `--orchestrate`; `clipboard-to-pane`/`flash-pane`
      untouched — see plan `workshop/plans/000100-copy-on-select-detach-plan.md`).
- [x] Add a detached-spawn seam to the clipcmd Runtime (`SpawnDetached` via the
      shared `startDetached` helper, ARCH-DRY per plan-quality review).
- [x] Restructure `RunCopyOnSelect`: clipboard mirror → detached orchestrator → return
      (`RunCopyOnSelectOrchestrate` is the detached second half).
- [x] Remove the temporary `[trace]` instrumentation.
- [x] Test the detach boundary + `make test-copy-on-select` (async poll) + clip pkg tests.
- [x] Dogfood verify: cold first copy after `pair-dev` restart inserts (operator
      confirmed across two cold starts — 2026-07-05).

## Log

### 2026-07-05
- 2026-07-05: closed — Full `make test` exit 0 (all Go pkgs + shell tests). clipcmd unit tests assert the hook detaches with zero inline slow-chain calls (listCalls/subprocess/execd all empty) and the orchestrator hands-off / skips-in-nvim / flash-guards. make test-copy-on-select drives the REAL detached chain end-to-end (async poll) green. Operator dogfood: first cold copy inserted across two pair-dev restarts — the reap no longer truncates the paste.; review verdict: SHIP

- Diagnosed via live `clipboard-debug.log` + timing instrumentation (see Spec).
  Confirmed release-mode first copy works; dev-mode first-copy-after-restart
  fails. Operator approved the root-cause detach fix over prewarm.
- Implemented the hook/orchestrator split (`RunCopyOnSelect` hook +
  `RunCopyOnSelectOrchestrate` detached via `SpawnDetached`/`startDetached`).
  Full `make test` green; clip unit tests + `test-copy-on-select` (real detached
  chain, async poll) green. Plan-quality + estimate-quality judges: INFO.
- Operator dogfood: **two cold `pair-dev` restarts, first copy inserted both
  times** — the reap no longer truncates the paste. Fix verified live.
