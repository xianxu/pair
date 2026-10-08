# Couch broadcast

`cmd/internal/broadcast` streams the composed Couch screen, view-only, to remote
browser viewers (#395). Status: the frame tap, hub, server, viewer page,
session and Couch control are built (M1–M3). Broadcasts are local-only until
the `cloudflared` tunnel lands (M4).

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
- **Fail-safe:** a tagged watch, `off | shown | hidden`, is driven by the
  indicator on the most recent frame. After `Activate()`, an indicator that
  stays hidden for `Grace` (1s) ends the hub with `ErrIndicatorHidden`.
  `Activate` comes after the tap is installed, never while the tunnel is
  opening. It arms the timer only if the last frame lacked the indicator, so a
  quiet screen already showing LIVE keeps broadcasting.
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

## Server and viewer page

`broadcast.Server` is read-only by construction:
- **GET only:** every other method gets 405, and no request body is read.
- **Token gate:** everything lives under `/<token>/`, and the token is
  compared in constant time. Only five assets and `events` are served; any
  other path, or a wrong token, gets 404.
- **Headers on every response:** `Cache-Control: no-store`,
  `Referrer-Policy: no-referrer` (the token is in the path), `nosniff`, and a
  CSP that confines the page to its own origin. Scripts are strict. Inline
  styles are allowed because xterm.js's DOM renderer inserts `<style>`
  elements.
- **`events`:** a Server-Sent Events stream.
  - `event: frame` carries `{cols, rows, b: base64}`.
  - `event: end` carries `{reason}`.
  - A `: ping` every 15s keeps Cloudflare from closing an idle stream (it does
    at 100s). A failed ping ends that viewer, which frees its slot.

The page (`web/`) and a vendored `@xterm/xterm` 5.5.0
(`web/vendor/xterm/VENDOR.md`, the same version as the headless oracle) are
embedded in the binary. The viewer:
- keeps the sender's grid and scales the font to fit (`nextFontSize`,
  node-tested via `TestViewerNode`);
- sends nothing back and uses no storage;
- on `end`, resets the screen and shows the reason, and never retries;
- never leaves a dead connection looking live. `connect(deps)` is a state
  machine, node-tested against a fake EventSource. A lost connection dims the
  screen and says so; one closed for good is retried at 2, 4, 8, 15 and 30s,
  and then the screen shows "Disconnected".

`TestManualViewerServer` (`BROADCAST_MANUAL=1`) serves a sample broadcast on
loopback for checking the page in a real browser.

## Session and tunnels

`broadcast.Session` (`Start(ctx, Config)`) owns one broadcast:
- **Start:** mints a 32-byte base64url token, serves the Server on the
  tunnel's listener, opens the tunnel, and returns only once the link answers
  200.
- **Stop:** ends viewers at once. The listener (`Shutdown`, 2s) and the tunnel
  close in the background, and `Done` closes when they have.
- **Ends on its own** when the hub ends (indicator hidden), the tunnel exits
  (`ErrTunnelExited`), or its HTTP server stops serving (`ErrServerFailed`).
  The hub publishes its end reason before closing any viewer queue, so every
  viewer is told the real reason.
- **Cancelling a start** closes whatever was opened, including a tunnel that
  finishes opening after the cancel.

A `Tunnel` owns both its `Listen` and its `Open`, because the listener's kind
depends on what exposes it. There are three:
- `LocalOnly`: loopback TCP, a link that works on this machine only.
- `FakeTunnel`: stateful, for tests in other packages too.
- `Cloudflared`: coming in M4.

`TestSessionPersistsNoFrameData` walks home, temp, XDG and the working
directory after a session and finds no frame content.

## Couch control

`couchtty/console_broadcast.go` owns the console's side as a tagged phase:
`off | starting | live | stopping`. The transition table is in the file's
header comment; all transitions happen on the Run loop.

- **Start:** Ctrl+Alt+b (`couchkeys.ActionBroadcast`). It is enhanced-encoding
  only, because the legacy `ESC ^B` is also nvim's Esc-then-page-up.
  `broadcast.Start` runs off the loop (`GoTracked`). A start that completes after
  it was cancelled or superseded is stopped, not adopted (the `attempt` counter).
  Who owns a finished start's session, the loop adopting it or the start
  goroutine stopping it, is decided by one CAS (`startClaim`). The start
  context is released as soon as the start ends; a tunnel must outlive it.
- **Going live:** the tap goes in before the chrome shows `LIVE ⏸`, so the
  first LIVE frame streams. `Activate` comes after, so the grace watch starts
  with the indicator on screen. The link goes to the clipboard (OSC 52) and is
  never drawn; the notice only says "link copied".
- **Status row:** `RenderStatusRow` draws `LIVE …` (starting) or `LIVE ⏸`
  (live) in `broadcast.LiveSGR` before the `REC` badge, through the same
  clipping pass. Its span is `RenderedStatusRow.Control`. A click anywhere in
  it (`routeMouseEvent`, checked before the actor chips) toggles.
  `TestStatusRowLiveCellSatisfiesIndicator` feeds the drawn row to
  `broadcast.IndicatorShown`, so the drawer and the checker can't drift.
- **Stop:** removes the tap (ordered with paints) and stops the session, which
  tells viewers at once. The phase is `stopping` until the listener and
  tunnel are down, off the input path. A toggle during `stopping` gets a
  notice.
- **Ends on its own** (indicator hidden past grace, tunnel exit, server
  failure): back to `off`, with a notice drawn from the same vocabulary
  viewers see (`broadcast.EndReason`).
- **Shutdown:** `teardown` calls `endBroadcastForShutdown` before the presenter
  is released, and waits up to 6s. Background waiters also end on console
  stop, so a tunnel whose close hangs cannot hold Couch's exit beyond that
  bound.
- **Options** (`couchcmd/broadcast.go`): `COUCH_BROADCAST_SWITCHER=show` and
  `COUCH_BROADCAST_TUNNEL=off`. An unknown value refuses startup, and both
  are parsed before anything opens. Without a configured broadcaster the cell
  never shows, and the key explains why.

