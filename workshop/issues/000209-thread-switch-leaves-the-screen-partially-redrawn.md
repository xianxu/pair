---
id: 000209
status: working
deps: []
github_issue:
created: 2026-09-06
updated: 2026-09-09
estimate_hours:
started: 2026-09-09T18:28:51-07:00
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

## Spec

**A switch must not depend on the retained byte tail being sufficient.** Ask the
child to repaint from its own state instead of reconstructing from history.

The mechanism already exists and is one call: `Child.Resize(Size)`
(`ptychild/child.go:195-196`) — *"The child gets SIGWINCH."* A resize nudge
(dimensions changed and restored) is the portable way to make a full-screen
child repaint from its authoritative model, and zellij — which is what couch's
children actually are — redraws its pane on SIGWINCH. That is the same thing the
operator's mouse click achieves, issued deliberately.

Byte replay stays as the immediate paint so the switch is not visibly blank
while the child responds; the nudge is what makes the result *correct* rather
than *probable*.

### To settle in the plan

1. **Which failure mode is actually firing.** The four above have different
   fixes, and a nudge only obviously fixes 1 and 2. Reproduce first — an idle
   thread left long enough to age its last paint out of 128 KiB is the cheapest
   candidate.
2. **Whether a nudge is safe here.** Resize is not free in this tree: the
   composer recognizers treat resize as *"a latched transaction: authorization
   stays closed from validation until a complete successful resize commits"*, and
   a spurious resize could interact with that latch or with a child mid-render.
   Check before adopting.
3. **Whether zellij offers a cleaner repaint request** than a synthetic resize.
   A dedicated action would avoid the reflow a resize implies.
4. **Where the fix lives.** Per the class above, prefer one answer used by both
   `couchtty` and `termcmd` over two.

Out of scope: enlarging the ring. It moves the threshold without removing the
assumption, and the memory cost is per-child across a fleet of 10+.

## Done when

- Switching to a thread whose last full paint has aged out of the ring produces
  a correct screen with no operator action — reproduced first, then fixed.
- The `cutoff < ringStart` path cannot present a cleared screen with nothing
  drawn.
- `pair term` tab switching gets the same guarantee, from the same code.
- `#196`'s reattach test still passes unmodified — the nudge must not perturb
  the mouse-mode belief on a path that shares this seam.
- A counted invariant lands in `#204`: a switch issues a repaint request, not
  only a replay write.

## Plan

- [ ] Reproduce deliberately: idle a thread until its last full paint is outside
      128 KiB, switch to it, capture the result. Identify which of the four modes
      fires.
- [ ] Settle plan items 2–4 (nudge safety, zellij repaint action, shared home).
- [ ] Implement for both `couchtty` and `termcmd` through one seam.
- [ ] Verify `#196`'s test unmodified; verify the composer resize latch is
      untouched.
- [ ] Add the counted invariant to `#204`.

## Log

### 2026-09-06

Filed from an operator report. The diagnosis is a code read, not a reproduction —
the mechanism is certain (byte-replay-only reconstruction over a bounded ring),
but *which* of the four failure modes produces the operator's "sometimes" is
not, which is why Plan step 1 is reproduction rather than a fix.

Worth recording that the workarounds diagnosed this: clicking the right pane and
scrolling the agent pane both cause zellij to repaint from its own complete
buffer. The operator's fix works because it routes around couch's reconstruction
entirely — so the bug is in trusting the reconstruction, not in the redraw path.

This is `#196`'s shape a second time: a bounded ring cannot re-derive state that
was established before its window, and the component acted as though absence of
evidence were evidence of absence. There it was mouse-mode tracking; here it is
the screen itself.
