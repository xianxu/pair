---
id: 000085
status: done
deps: ["#82"]
github_issue:
created: 2026-06-29
updated: 2026-06-29
estimate_hours: 1.42
started: 2026-06-29T15:14:39-07:00
actual_hours: 0.18
---

# Batch pair-wrap stdout redraws

## Problem

#82 is tracking a Codex scroll wedge where pair-wrap continues reading,
forwarding, and capturing stdout, but zellij can still get into a bad
scroll/render state. Codex produces dense redraw bursts. Today pair-wrap writes
each filtered PTY stdout chunk directly to `os.Stdout`, so zellij may receive a
redraw storm even though pair-wrap's raw capture and tracing remain healthy.

## Spec

Add a focused experiment in `pair-wrap`: batch only the filtered bytes destined
for `os.Stdout`, flushing at most once every 100ms during active output. Raw
scrollback capture, resize/time event offsets, span extraction, OSC/BEL
detection, picker detection, image capture, and tracing over raw chunks stay on
the immediate PTY-read path.

Design constraints:

- Preserve semantic capture: `--scrollback-log` writes original PTY bytes
  immediately, so renderer offsets and resize/time events remain aligned.
- Preserve detection responsiveness: overlay detection and notify logic still
  observe every PTY chunk immediately.
- Smooth only the visible zellij delivery surface: stdout batches flush on the
  100ms tick, on EOF, and before `masterPump` returns.
- Keep the change local and reversible while #82 remains an experiment
  (`ARCH-PURPOSE`).
- Make the buffering core pure/testable and keep the `os.Stdout` write as a
  thin IO seam (`ARCH-PURE`).
- Reuse the existing stdout filter and wrap-event trace path instead of adding a
  parallel output pipeline (`ARCH-DRY`).

## Done when

- [x] `pair-wrap` buffers filtered stdout bytes and flushes them no more than
      once per 100ms while data is arriving continuously.
- [x] `pair-wrap` flushes any pending stdout bytes before exiting after PTY EOF.
- [x] Raw scrollback writes remain immediate and byte-for-byte original.
- [x] Focused Go tests cover batch timing, EOF flush, stdout filtering, and
      immediate scrollback capture.
- [x] Trace events distinguish queued filtered stdout (`stdout-queue`) from
      actual batched writes (`stdout-batch-flush`), with byte/chunk counts.
- [x] Pair remains buildable/testable, and the log records how to dogfood the
      experiment for #82.

## Estimate

```estimate
model: estimate-logic-v2
familiarity: 0.9
item: smaller-go-module design=0.2 impl=0.6
item: smaller-go-module design=0.1 impl=0.3
item: atlas-docs design=0.1 impl=0.1
total: 1.42
```

## Plan

- [x] Write the durable implementation plan in
      `workshop/plans/000085-pair-wrap-stdout-batching-plan.md`.
- [x] Add failing tests for stdout batching and EOF flush.
- [x] Implement the batching seam in `cmd/pair-wrap/main.go`.
- [x] Extract a small testable stdout pump helper so cadence and EOF flush are
      proven without sleeping in tests.
- [x] Verify focused Go tests, broader Go tests, and build.
- [x] Update atlas/logs with the experiment behavior and dogfood command.

## Log

### 2026-06-29
- 2026-06-29: closed — final re-close after FIX-THEN-SHIP coverage fix; go test ./cmd/pair-wrap; go test ./...; make build; make test; sdlc issue validate workshop/issues/000085-pair-wrap-stdout-batching.md; git diff --check; review verdict: SHIP
- 2026-06-29: closed — re-close after boundary REWORK fix; go test ./cmd/pair-wrap; go test ./...; make build; make test; sdlc issue validate workshop/issues/000085-pair-wrap-stdout-batching.md; git diff --check; review verdict: FIX-THEN-SHIP
- 2026-06-29: closed — go test ./cmd/pair-wrap; go test ./...; make build; make test; sdlc issue validate workshop/issues/000085-pair-wrap-stdout-batching.md; git diff --check; review verdict: REWORK

- Created as a focused follow-up to #82. Decision: batch only filtered
  `os.Stdout` delivery, not raw scrollback or detection, so the experiment lowers
  zellij redraw pressure without damaging pair-wrap's diagnostic substrate.
- Plan-quality review rejected the first plan because it did not explicitly
  test the 100ms cadence or EOF flush. Refined the plan to add a testable stdout
  pump helper and trace field expectations before implementation.
- Implemented `stdoutBatcher` / `stdoutPump` in `cmd/pair-wrap/main.go`.
  `handleChunk` now emits `stdout-queue` and keeps raw scrollback immediate;
  `masterPump` flushes `stdout-batch-flush` on a 100ms tick and at EOF. Live
  dogfood requires `make install` before restarting Pair, because zellij runs
  the installed `pair-wrap`.
- Verification passed: `go test ./cmd/pair-wrap`; `go test ./...`; `make build`;
  `make test`; `sdlc issue validate workshop/issues/000085-pair-wrap-stdout-batching.md`;
  `git diff --check`.
- Boundary review returned `REWORK` for plan documentation only: `stdoutPump`
  was listed as PURE despite writing through an injected `io.Writer`. Revised
  the durable plan to classify `stdoutPump` as an integration point around the
  pure `stdoutBatcher`.
- Second boundary review returned `FIX-THEN-SHIP` for missing `masterPump`
  integration coverage. Added pipe-backed tests for ticker flush and EOF flush
  through `masterPump`, with a short injected flush interval.
