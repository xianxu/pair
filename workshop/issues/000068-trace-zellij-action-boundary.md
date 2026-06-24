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

#### Deep-dive bisection (2026-06-23 PM) — supersedes the "#30 strip unmasks a fatal un-batched repaint" root-cause above (that hypothesis is **REFUTED**)

Walked the zellij source (cloned at `../zellij`, v0.44.1+36 ≈ installed 0.44.3) plus the
server log (`/var/folders/.../zellij-501/zellij-log/zellij.log`).

- **Real mechanism (confirmed from source + log).** `route.rs:2645` fires when
  `recv_client_msg` (`ipc.rs:364`) returns `None` 1000× in a row. `None` = `read_exact`
  `Err` on a **broken client socket**, spun in a tight loop (the counter resets on any
  valid message, `route.rs:2160`). So the guard is the server **spin-detecting an
  already-dead client connection** — a *symptom*, not the cause. The client exits
  **cleanly** (`lib.rs:1266` "Bye from Zellij!", **no client panic**). The
  `pty.rs:1352 .fatal()` panic (pty-reader's send to the screen thread failed) is
  **incidental collateral** — present in crashes 67-1/2, **absent** in 67-4/5.
- **Channel map.** pane PTY (raw bytes) → in-proc MPSC `ScreenInstruction::PtyBytes` →
  screen thread (VTE→grid→render) → **framed length-prefixed protobuf over a unix socket**
  (`ServerToClientMsg::Render`) → client → terminal. Byte framing is reconstructed
  (length-prefix + `read_exact`), so split writes / sync-block splitting are **not** the issue.
- **Eliminated, each with evidence:** DEC 2026 strip (#30) **and** `PAIR_CODEX_SYNC_PASSTHROUGH`
  passthrough both crash → marker handling irrelevant; `PAIR_CODEX_ALT_SCREEN=1` crashed and
  codex never even emits `1049h`; **pure rate** — synthetic generator
  (`scratchpad/zellij-storm.py`) at 41 *and* 80 fps with a fast-draining headless client
  **survived** (only a SIGKILL teardown faked the guard — a harness artifact I initially
  misread); **codex content** — `--debug` raw byte capture (`zellij-<id>.log`) shows terminal
  queries only at startup, an ordinary inline repaint storm, one 64KB divider mid-stream, no
  killer sequence; client-stdout-panic theory — no panic logged, client exits clean.
- **pair-wrap EXONERATED.** Execed codex directly into the pane (temporary `PAIR_NO_WRAP`
  layout gate, since reverted) — **still crashed**. Verified the bypass actually took effect
  via the frozen `wrap-events-<tag>.jsonl` mtime (pair-wrap `O_TRUNC`s it at startup; it stayed
  at the prior wrapped run's time while the bypass run touched the draft + crashed). [The
  scrollback-file-absence signal I used first was unreliable — those get cleaned on quit.]
- **Bisection points at the nvim draft pane.** codex alone (1 pane, stock config) → survived;
  codex + **idle `sleep`** 2nd pane (pair config, `2 tiled`) → survived ~16.5 min; codex +
  **nvim** draft (pair's real two-pane layout) → **crashes**. So it is **not** the config
  (`scroll_buffer_size 2000` / `pane_frames`) and **not** the split/geometry — it is **nvim
  specifically** in the second pane (working hypothesis: a second active render source racing
  codex's ~41 Hz repaint through the screen→client pipeline, killing the client connection).
- **CAVEAT (gates the conclusion):** the two "survived" runs assume codex was *actively
  churning*; that storm was not independently verified. Re-run single-pane and 2-pane-idle with
  a confirmed ~1 min+ storm before fully trusting the nvim conclusion.
- **Next:** (1) confirm the storm in the survived runs; (2) `--debug` capture on the real
  two-pane layout to see what the nvim pane emits during a codex storm, and test whether a
  minimal / stub periodically-repainting 2nd pane reproduces — pinning nvim's render cadence vs.
  a specific sequence.
- Repro: `pair-dev codex -- --sandbox danger-full-access`, drive codex into sustained work
  (e.g. closing #67); crash signature = `zellij_client lib.rs:1266 "Bye"` + `route.rs:2645`
  guard in the server log, ~50s–2min after start (faster, ~10s, unwrapped).
