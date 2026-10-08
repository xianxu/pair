---
id: 000409
status: open
deps: []
github_issue:
created: 2026-10-07
updated: 2026-10-07
estimate_hours:
card_mirror: 'f76904259515b8bb913f65a640aa8f722bed6702' # card fields mirrored from issue-cards; edit via sdlc
---

# couch: a single 5s stall in the outer terminal ends the whole session

## Problem

On 2026-10-07 at 21:31 local time, couch shut itself down while the operator was reading in a browser. Couch was in the background, with agent panes showing Claude Code's "Generating…" spinner. It was not a crash: no `crash/*.crash` file was written, the kernel log shows no kill, and the terminal capture ends with `capture-end status: complete`.

From the terminal capture (`~/.local/share/pair/captures/session-4087359685/events.jsonl`, started 20:16 with `COUCH_CAPTURE_DIR`):

| UTC | event |
|---|---|
| 04:31:29.67 | last successful `host-write` |
| 04:31:34.815 | `host-write` requested 17424, accepted 1024, `context deadline exceeded` after 5001 ms (first error in about 75 minutes of capture) |
| 04:31:34.815–.864 | `endpoint-end` for all 10 `couch-pty-*` |
| 04:31:39.85 | the restore write (leave alt screen, turn off mouse modes, show cursor; 118 bytes) accepted 0, deadline exceeded, so the operator's terminal was left in a broken state |
| 04:32:06 | `capture-end complete` |

Mechanism: `terminal.WriteTimeout = 5s` (`cmd/internal/terminal/profile.go`). `Presenter.write` turns a timed-out write into `WriteFailure{Op: "parent output"}`. The presenter then fails, and couch exits. So a single stall in the outer terminal (Ghostty) ends the whole session.

### Amplifier: every change is a full-screen repaint

Couch's output to the outer terminal is almost entirely full-screen repaints:

- Of 14,876 `host-write`s in the capture, 14,780 (99.4%) were larger than 15 KB, typically 16.7–17.4 KB for a 191×53 screen.
- Rate: about 500 per minute (around 9/s) while agents were streaming. Even with nothing for the operator to do, about 120 per minute (around 2/s), because the selected pane's spinner and elapsed-time counter (`✽ Generating… (53s · ↓ 1.3k tokens)`) redraws a few times per second.
- Ghostty was already slow to accept them: 50–230 ms per 17 KB write in the minutes before the stall.

Cause: the scrollback-history path (`RenderWithHistory` / `HistoryRender.Emit`, `cmd/internal/terminal/history_render.go`) has no row-level diff. When the frame is `dirty` (any changed cell, including a one-character spinner tick), `Emit` inserts blank lines over the whole area below the top row and repaints all `height` rows. The diff-based `Render(prev, next)` path is used only when `history == nil`. So a change of about 50 bytes in a pane costs 17 KB of output to the terminal, roughly 35 KB/s while couch looks idle and about 150 KB/s while agents stream.

Hypothesis (not verified): while Ghostty is in the background or covered by another window, macOS throttles it (App Nap or timer coalescing), so it reads its pty more slowly. Steady full repaints then fill the pty buffer until one write misses the 5 s deadline.

## Spec

Two independent changes:

1. **Say why couch exited when the terminal stopped accepting output.** Keep the exit: failing loudly is how this problem was found. But record the reason the way `crashreport` records a panic. When the presenter fails with `WriteFailure{Op: "parent output"}`, write the cause (operation, bytes accepted/requested, how long it waited) to the run's crash file before exiting. The next start then reports it once, like a panic, for example "previous couch exited: terminal stopped accepting output for 5s". Today this exit leaves no trace: no `.crash` file, no message, and a broken terminal because the restore write times out too. It was diagnosed only because the capture happened to be on.
2. **Send only changed rows on the scrollback path.** When the frame is dirty but `reset` is false and no rows were added to history, `HistoryRender.Emit` should send only the changed rows instead of erasing and repainting all `height` rows. Build the chain-granular row diff already designed in #262 (history: `workshop/history/issues/000262-diagnose-input-screen-flicker.md`, Log 2026-09-17, option 1). The repaint unit is a wrap chain taken as the union of chains in the previous and next frames, using a confined `IL` for multi-row runs and `EL2` for single-row runs. Row 0 keeps `ECH` when it continues from history. Resets, alt-screen transitions and history pushes keep the full rebuild. The cheaper first step from that design also counts: diff only rows that are unwrapped in both frames, and fall back to the full rebuild for everything else.
## Done when

- A test with a fake parent writer that blocks past `WriteTimeout` shows couch exiting with the cause in the crash file. The next `crashreport.Install` reports it as a previous ending with that reason, separate from a Go panic.
- A test of a spinner-style one-cell change on the history path shows output bytes in proportion to the changed rows, not the full screen. Measure bytes per frame before and after. The end state matches the full rebuild under the xterm oracle.
- Rerun the original scenario (couch in the background behind a browser, agent spinner running, capture on): repaint volume while idle drops by more than 10×. If couch still exits, the next start says why.
## Plan

- [ ]

## Revisions

### 2026-10-07: drop "survive a slow terminal"; record the exit reason instead

- **Reason:** the operator decided exiting is the right behavior; the loud exit is how this was discovered. Absorbing stalls would hide the cost. What was wrong is that the exit was silent, not that it happened.
- **Delta:** Spec item 1 changed from "A slow outer terminal must not end the session: treat a timed-out parent-output write as backpressure, mark for full repaint, drop frames, retry when writable, fail only after a much longer outage or on EIO/EPIPE/hangup" to "Say why couch exited". Its Done-when line changed from "couch staying up and converging" to "exit reason recorded and reported on next start". Item 2 now points at #262's existing row-diff design, and its deferral condition ("a measurement shows the bytes matter") is met by this capture.

## Log

### 2026-10-07

- Filed from a brain-session crash investigation. The capture file is 1.2 GB and stays local; the excerpts above are the evidence. Also noted, not investigated: `wrap-events-1-pair-8.jsonl` is 647 MB.
- Row diff was deliberately deferred in #262 (2026-09-17, with the operator) on the premise that a local terminal parses faster than a person types. That missed spinner/streaming repaints with no input from the operator (2–9/s), and a background, throttled Ghostty (50–230 ms per frame, then a 5s stall). The operator reports these exits always happen while not interacting with couch, which supports the throttling hypothesis.
