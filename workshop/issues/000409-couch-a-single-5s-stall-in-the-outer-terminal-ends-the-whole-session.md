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

1. **A slow outer terminal must not end the session.** Treat a timed-out parent-output write as backpressure, not as presenter failure. Mark the presenter as needing a full repaint, drop intermediate frames, and retry when the outer tty becomes writable. Give up (fail) only after a much longer outage, or when the tty is actually gone (EIO/EPIPE/hangup). A partially written frame must be repaired by a full repaint afterwards. The partial-frame state is already tracked by `HistoryState`, which is committed only after a fully emitted frame.
2. **Stop full repaints for small changes.** When the frame is dirty but `reset` is false and no rows were added to history, `HistoryRender.Emit` should send only the rows that changed between `previous` and `next`, keeping the soft-wrap constraints the current code protects (see the comments about ECH/EL2 and wrapped rows). Keep the full repaint for resets, geometry changes, and history appends that shift the screen.

## Done when

- A test with a fake parent writer that blocks for longer than `WriteTimeout` and then drains shows couch staying up and converging to the correct screen.
- A test of a spinner-style one-cell change on the history path shows output bytes in proportion to the changed rows, not the full screen. Measure bytes per frame before and after.
- Rerun the original scenario (couch in the background behind a browser, agent spinner running, capture on): no exit, and repaint volume while idle drops by more than 10×.

## Plan

- [ ]

## Log

### 2026-10-07

- Filed from a brain-session crash investigation. The capture file is 1.2 GB and stays local; the excerpts above are the evidence. Also noted, not investigated: `wrap-events-1-pair-8.jsonl` is 647 MB.
