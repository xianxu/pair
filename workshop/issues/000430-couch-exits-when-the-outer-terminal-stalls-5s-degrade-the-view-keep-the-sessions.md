---
id: 000430
status: open
deps: []
github_issue:
created: 2026-10-10
updated: 2026-10-10
estimate_hours:
card_mirror: 'eb992ce9a754bb7760ab4b6d1991e6a18d86df9d' # card fields mirrored from issue-cards; edit via sdlc
---

# couch exits when the outer terminal stalls 5s: degrade the view, keep the sessions

## Problem

Couch keeps shutting itself down under heavy load. Each time, Ghostty stopped
reading Couch's output for 5 seconds. #409 deliberately kept that exit
("failing loudly is how this problem was found"). It added a crash record and
row-diff output. The crash records now show the exit recurring at small write
sizes, so output volume is no longer the cause:

| run started (UTC) | record |
|---|---|
| 2026-10-09 18:09 | wrote 122 of 14086 bytes (before #409 M2) |
| 2026-10-10 04:10 | wrote 45 of 605 bytes |
| 2026-10-10 23:21 (exit ~17:35 local) | wrote 373 of 688 bytes |

(`~/.local/share/pair/couch/crash/*.crash`; the running binary was main
4c147e5d, which includes #409 M2.)

At the 17:35 exit (pair:4 investigation for ops:0, 2026-10-10):

- **Load:** the 12-core host had load averages of 72/87/63.
  - Two ariadne `sdlc.test` runs at about 140% CPU each.
  - Spotlight and FSEvents.
  - About 90 pair processes, plus nvim.
- **Ghostty:** its main thread kept logging through 17:35–17:36 (unified
  log), so the app was alive. Only its pty reader fell behind.
- **Memory:** no jetsam and no OOM. Free memory was 83%.

Ghostty took more than 5s to drain a 600-byte write: the terminal was slow,
not gone.

The policy turns a transient stall in the presentation layer into the loss of
the coordinator, along with all ~10 agent PTYs it hosts (`endpoint-end` for
every `couch-pty-*`, per #409's capture). The restore write then also times
out, which leaves the operator's terminal in a broken mode state.

Mechanism: `terminal.WriteTimeout = 5s` (`cmd/internal/terminal/profile.go`).
`Presenter.write` turns an expired deadline into
`WriteFailure{Op: "parent output"}`, then `Presenter.fail` runs and Couch
exits (`cmd/internal/terminal/presenter.go`).

## Spec

Separate two cases: a slow parent and a gone parent.

- **Gone parent:** EIO, EPIPE or hangup on the parent tty means the terminal
  is gone. Exiting stays correct.
- **Slow parent:** an expired deadline means the terminal is slow. The
  presenter should stop painting and keep the in-flight write blocked, with no
  deadline. Frames coalesce into the dirty state the presenter already keeps,
  so they are skipped rather than queued. Sessions, endpoints and the control
  plane keep running.
  - When the write completes, the presenter repaints the full screen once
    (`reset`) and resumes.
  - The stall is recorded as a notice or diagnostic, not as an exit.
- **Escape hatch:** if a deadline is still wanted, use a much longer one
  (minutes), or an operator action (detach, then reattach). Never 5s of
  scheduler starvation.

This also covers the last Done-when line of #420 ("Couch stays connected
through a multi-second terminal stall"). #420's polling work is a load
reducer; this issue is the resilience fix. Ship it independently.

## Done when

- A test with a fake parent writer that blocks for longer than `WriteTimeout`
  and then drains shows:
  - Couch stays up.
  - The endpoints stay alive.
  - No frames are queued during the stall: one coalesced full repaint follows
    the drain.
  - The end state matches the full rebuild under the xterm oracle.
- A test where the parent writer returns EIO or EPIPE still ends Couch with a
  recorded exit reason (the #409 path).
- Live: a forced Ghostty stall of more than 5s (for example `kill -STOP` on
  Ghostty for 10s, then `-CONT`) leaves Couch and its agents running, and the
  view recovers.

## Plan

- [ ]

## Log

### 2026-10-10

- Filed by pair:4 at the request of ops:0 (TL), during a load diagnosis. Not implemented.
