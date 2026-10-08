# Couch broadcast

`cmd/internal/broadcast` streams the composed Couch screen, view-only, to remote
browser viewers (#395). Status: the frame tap, hub, server, viewer page and
session are built (M1–M2); the tab-bar control and `cloudflared` tunnel follow.

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
  node-tested via `TestViewerFit`);
- sends nothing back and uses no storage;
- on `end`, resets the screen and shows the reason.

`TestManualViewerServer` (`BROADCAST_MANUAL=1`) serves a sample broadcast on
loopback for checking the page in a real browser.

## Session and tunnels

`broadcast.Session` (`Start(ctx, Config)`) owns one broadcast:
- **Start:** mints a 32-byte base64url token, serves the Server on the
  tunnel's listener, opens the tunnel, and returns only once the link answers
  200.
- **Stop:** ends viewers at once. The listener (`Shutdown`, 2s) and the tunnel
  close in the background, and `Done` closes when they have.
- **Ends on its own** when the hub ends (indicator hidden) or the tunnel exits
  (`ErrTunnelExited`).
- **Cancelling a start** closes whatever was opened, including a tunnel that
  finishes opening after the cancel.

A `Tunnel` owns both its `Listen` and its `Open`, because the listener's kind
depends on what exposes it. There are three:
- `LocalOnly`: loopback TCP, a link that works on this machine only.
- `FakeTunnel`: stateful, for tests in other packages too.
- `Cloudflared`: coming in M4.

`TestSessionPersistsNoFrameData` walks home, temp, XDG and the working
directory after a session and finds no frame content.

