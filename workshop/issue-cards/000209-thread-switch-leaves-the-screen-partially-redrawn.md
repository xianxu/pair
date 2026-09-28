---
id: '000209'
status: done
started: 2026-09-09T18:28:51-07:00
created: 2026-09-06
updated: 2026-09-10
estimate_hours: 1.72
actual_hours: 2.63
---

# thread switch leaves the screen partially redrawn

## Problem

Operator report: after switching threads in couch, the screen is sometimes not
fully redrawn. Clicking the right pane, or scrolling the agent pane a little,
brings it back.

**Both workarounds are the tell.** Neither adds information couch had — they make
*zellij* repaint the pane from its own complete buffer. That is the state couch's
reconstruction failed to reproduce.

### Mechanism

A switch reconstructs the screen from **bytes alone**. `switchTo`
(`couchtty/console.go:498`) calls:

```go
c.takeOverScreen(p.child.ReplayThrough(p.replayCutoff))
```

and `takeOverScreen` (`console.go:981-989`) is exactly two writes:

```go
io.WriteString(c.host, hostty.HomeAndClear)   // blank the screen
c.host.Write(body)                            // replay the retained tail
```

Nothing asks the child to repaint. The screen is correct **iff** the retained
tail happens to contain enough output to repaint it — and the tail is bounded:
`DefaultRingBytes = 128 * 1024` (`ptychild/ring.go:16`), whose own comment is
candid that this is an approximation: *"only the tail can repaint a screen — the
head is a screen nobody will ever see again."*

That assumption holds for a chatty child and breaks in at least four ways:

1. **The last full paint aged out.** A mostly-idle agent painted its frame more
   than 128 KiB ago. Clear-then-replay reproduces only the incremental updates
   since, over a blanked screen.
2. **Replay returns nothing at all.** `ReplayThrough` (`child.go:249-251`) returns
   `nil` when `cutoff < ringStart` — the cutoff is older than what the ring still
   holds. The screen is cleared and nothing is written.
3. **The tail begins mid-sequence.** `Ring` keeps the last N bytes and, as
   `replay.go` records, *"bisects whatever spans that boundary"* — so a replay can
   open inside an escape sequence, and `StripQueries` deliberately emits an
   unterminated escape verbatim.
4. **Mode state is not byte-replayable.** Alt-screen, scrolling region and cursor
   state are the product of the *whole* stream, not its tail. `#196` established
   this class for mouse modes: a bounded ring cannot re-derive a mode set before
   its window. Same defect, different mode.

Which of these dominates is unknown and the report's "sometimes" is consistent
with all four — see Plan step 1.

### It is a class, not a site (`ARCH-PURPOSE`)

`termcmd` does the identical thing for `pair term` tab switching
(`run.go:1021-1024`):

```go
func (m *terminalMux) redrawTab(replay []byte) {
	io.WriteString(m.stdout, hostty.HomeAndClear)
	m.stdout.Write(replay)
}
```

Same two writes, same bounded ring, same assumption. A fix that repairs only
couch leaves the same bug in the right pane's tabs. Both are consumers of the
shared `ptychild`/`hostty` split (`#146`), which is where the answer probably
belongs.
