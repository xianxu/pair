---
id: 000068
status: working
deps: []
github_issue:
created: 2026-06-23
updated: 2026-06-23
estimate_hours: 1.5
started: 2026-06-23T09:31:06-07:00
---

# Trace zellij action boundary

## Problem

Codex pair sessions can reproduce a zellij logout where the final visible line
is `Bye from Zellij!` and the zellij server log reports more than 1000
consecutive unknown/empty client messages. The existing logs identify zellij's
disconnect guard but do not show which pair-originated zellij action happened
immediately before the failure.

## Spec

Add opt-out-safe diagnostic tracing around the pair-managed zellij action
boundary. The trace must focus on the user prompt path from the draft nvim pane
to the agent pane, because that is the known 100% repro surface. Records should
be JSONL under `$PAIR_DATA_DIR`, scoped by `$PAIR_TAG`, and should avoid writing
full prompt bodies; body-bearing actions record length and a short hash instead.

Initial instrumentation covers:

- draft `send_to_agent`: relative focus, `write-chars`, submit, and focus-back;
- review `pair_poke`: direct `--pane-id` write and submit, for the parallel
  absolute-pane path;
- `pair-wrap`: redacted stdin, master PTY output, stdout-to-zellij writes,
  scrollback writes, resize, and child lifecycle events.

The wrapper records timestamp, component, label, argv summary, duration,
`vim.v.shell_error`, stdout/stderr byte counts when observable, and redacted
argument metadata.

The pair-wrap trace records byte counts, short SHA-256 prefixes, offsets, write
errors, and lifecycle labels, but never raw agent output or prompt text.

## Done when

- A clean repro session leaves `$PAIR_DATA_DIR/zellij-actions-<tag>.jsonl`
  showing the final pair-originated zellij actions before zellij disconnects.
- A clean repro session leaves `$PAIR_DATA_DIR/wrap-events-<tag>.jsonl`
  showing the last pair-wrap stdin/output/stdout boundary events before zellij
  disconnects.
- Full prompt text is not written to the trace.
- Tests cover redaction/logging, review poke integration, and pair-wrap event
  redaction.

## Estimate

```estimate
diagnostic-instrumentation design=0.2 impl=0.8
focused-headless-tests design=0.1 impl=0.4
```

## Plan

- [x] Add a reusable Lua zellij trace wrapper with JSONL output and body redaction.
- [x] Route draft prompt send and review poke zellij calls through the wrapper.
- [x] Add/extend headless tests that fail without trace records and pass with them.
- [x] Run the focused tests and record the repro command/log location.
- [x] Add pair-wrap-side redacted PTY/stdout/lifecycle trace after repro showed
  no pair-originated zellij action immediately before disconnect.

## Log

### 2026-06-23

- Created as follow-up to #67 after log audit showed zellij disconnecting the
  client for repeated unknown/empty IPC messages. `sdlc claim --issue 68`
  changed status to working locally, but could not broadcast because no `main`
  worktree is currently available.
- Added `nvim/zellij_trace.lua` and routed draft `send_to_agent` plus review
  `pair_poke` zellij actions through it. The repro trace lands at
  `$PAIR_DATA_DIR/zellij-actions-<tag>.jsonl`; body-bearing `write-chars`
  records include byte length and a short SHA-256 prefix instead of prompt text.
  Verified with `make test-zellij-trace`, `bash tests/review-poke-test.sh`,
  `bash tests/queue-send-test.sh`, `nvim --headless -u nvim/init.lua +'lua print("loaded")' +'qa!'`,
  and `git diff --check`.
- Repro with tag `67-clean` showed the last zellij action trace at 09:43:43 and
  the zellij disconnect at 09:44:18, so the immediate trigger was not prompt
  injection. Added `pair-wrap` structured trace at
  `$PAIR_DATA_DIR/wrap-events-<tag>.jsonl` to capture redacted stdin, master PTY
  output, stdout writes to zellij, scrollback writes, resize, and child exit
  timing. Verified with `go test ./cmd/pair-wrap -run 'TestTraceWrap|TestHandleChunkTraces' -count=1`,
  `go test ./cmd/pair-wrap -count=1`, `make pair-wrap`, `make test-zellij-trace`,
  and `bash tests/review-poke-test.sh`.
- Repro tag `debug-67` (codex): last nvim zellij action 10:43:28, disconnect
  10:44:36 — a **68s gap**, so again not prompt injection. `wrap-events` shows a
  sustained codex output storm filling that gap: ~50 master-chunks/s for the full
  68s straight into the disconnect, the same ~217-byte frame repeated 750+× (359
  distinct hashes / 3454 chunks), 672 BEL-bearing chunks, **2684/3454 stdout-writes
  `filtered`, 0 write errors**. The 8-byte chunk filtered to `stdout_len:0` and
  repeated 477× is `\x1b[?2026h`.
- Root mechanism: codex runs `--no-alt-screen` (inline mode for scrollback), so it
  repaints in the primary buffer ~50Hz, wrapping each frame in DEC 2026
  synchronized-output markers. `stripCodexSyncOutput` (#30, to stop live-scroll
  interference) strips 2026/1004, so zellij receives an **un-batched 50Hz repaint
  stream** — not byte volume (~8KB/s) but ~50 un-synchronized render updates/s —
  which trips zellij's "1000+ unknown/empty client messages" disconnect guard.
  #30's cosmetic fix likely unmasks this fatal one.
- Added `PAIR_CODEX_SYNC_PASSTHROUGH` gate (`stdoutChunk`): set → forward codex's
  2026/1004 markers untouched, to A/B whether the strip is the trigger (survives →
  honor 2026 / coalesce frames instead of stripping; still dies → throttle codex
  stdout rate). Default off preserves #30. Verified with
  `go test ./cmd/pair-wrap -run TestStdoutChunk -count=1` and `go test ./cmd/pair-wrap -count=1`.
