# Couch broadcast

`cmd/internal/broadcast` streams the composed Couch screen, view-only, to remote
browser viewers (#395). Status: the frame tap and hub are built (M1); the server,
viewer page, tab-bar control and `cloudflared` tunnel follow in later milestones.

## Source: the Presenter tap

`terminal.Presenter.SetTap` installs a `terminal.Tap`. It is called on the
Presenter goroutine after a frame is fully written to the parent **and**
admitted as presented, with an owned clone and a `terminal.FrameClass`.
`Select` makes the class public; `Panel(ctx, f, class)` takes it. Couch's
switcher (`couchtty.Console.showMenu`) is the one private panel: it lists the
whole fleet. Taps see exactly what the operator saw. They don't see parent bytes,
which are diffs against the operator's previous frame and carry parent-mode
controls.

## Hub

`broadcast.Hub` turns tapped frames into the broadcast. One goroutine owns the
`Stream` (the last frame sent), the subscribers and the grace timer, so a join
and a frame are totally ordered.

- **Withholding:** a frame is forwarded only if `IndicatorShown` holds. That
  means its last row starts with `LiveLabel` drawn in `LiveSGR` (red
  background). The status row draws the same constants. Invariant: viewers get
  only frames the operator saw marked LIVE.
- **Fail-safe:** after `Activate()`, an indicator that is hidden for `Grace`
  (1s) ends the hub with `ErrIndicatorHidden`. `Activate` comes after the tap is
  installed, never while the tunnel is opening.
- **Privacy:** `ViewerFrame` replaces a private frame with a placeholder that
  keeps the tab bar, unless `ShowSwitcher` is set.
- **Shared diffs:** each frame is rendered once with `terminal.Render`, the
  same renderer the xterm oracle checks, and the diff is shared. Late joiners
  get `Join()`, a full render of the current frame.
- **Slow viewers:** a viewer whose 8-deep queue overflows is marked for resync.
  It gets `Join()` as soon as it has room (on a frame or the 100ms tick), and
  never a diff against a frame it missed.
- **Backpressure:** `Offer` never blocks the Presenter; the latest frame wins.

The hub keeps no history and writes nothing to disk; the only frame it retains
is the current one, for late joiners.
