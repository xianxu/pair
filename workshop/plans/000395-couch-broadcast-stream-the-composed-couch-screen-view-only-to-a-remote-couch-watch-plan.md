# Couch Broadcast Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stream the composed Couch screen, view-only, to browser viewers over a
`cloudflared` quick tunnel, started and stopped from a `LIVE` cell in Couch's tab
bar.

**Architecture:** The Presenter gains a frame tap: after each successful paint to
the operator it hands a cloned composed `Frame` and its privacy class to a
subscriber. A new `cmd/internal/broadcast` package turns tapped frames into one
shared stream of rendered VT bytes (via the existing `terminal.Render`), swaps
private frames (the switcher) for a placeholder, withholds every frame on which
the operator's screen did not show the `LIVE` indicator, and serves the stream as
Server-Sent Events to an embedded xterm.js page behind a capability token. Couch
owns the session's lifecycle and the tab-bar control; `couchcmd` wires the real
`cloudflared` tunnel.

**Tech Stack:** Go (`net/http`, `crypto/rand`, `embed`), the existing
`cmd/internal/terminal` renderer, vendored `@xterm/xterm` 5.5.0 (the same
version as the `@xterm/headless` oracle in `tests/terminal-oracle`), and
`cloudflared` quick tunnels.

Issue: `workshop/issues/000395-couch-broadcast-stream-the-composed-couch-screen-view-only-to-a-remote-couch-watch.md`.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `FrameClass` | `cmd/internal/terminal/tap.go` | new |
| `Tap` | `cmd/internal/terminal/tap.go` | new |
| `Presenter.Panel` (gains a `FrameClass`) | `cmd/internal/terminal/presenter.go` | modified |
| `LiveLabel`, `IdleLabel`, `IndicatorShown` | `cmd/internal/broadcast/indicator.go` | new |
| `ViewerFrame` | `cmd/internal/broadcast/privacy.go` | new |
| `Stream`, `Message` | `cmd/internal/broadcast/stream.go` | new |
| `BroadcastCell` in `StatusModel`; `RenderedStatusRow.Control` | `cmd/internal/couchtty/reserve.go` | modified |
| `ActionBroadcast` binding | `cmd/internal/couchkeys/couchkeys.go` | modified |
| `nextFontSize` | `cmd/internal/broadcast/web/viewer.js` | new |

- **FrameClass / Tap** — `FramePublic` or `FramePrivate`; a `Tap` is
  `func(Frame, FrameClass)`, called by the Presenter goroutine after a paint
  succeeds. The contract: a tap must not block and receives an owned clone.
  - **Relationships:** 0..1 tap per Presenter. The class is set by whoever
    selects what is on screen: `Select` sets public, `Panel` takes a class.
  - **DRY rationale:** every paint path (endpoint, panel, chrome update, resize)
    converges on `paintPublication`, so one hook sees all of them.
  - **Future extensions:** per-pane privacy (#395 Spec) is a finer class or a
    mask on the frame; the tap signature widens there.
- **IndicatorShown(f)** — reports whether the frame's last row starts with the
  cells of `LiveLabel`. `RenderStatusRow` draws the same constant, so the drawer
  and the checker can't disagree (ARCH-DRY).
- **ViewerFrame(f, class, showSwitcher)** — returns `f` unchanged for public
  frames or when `showSwitcher` is set; otherwise a placeholder of the same
  geometry: blank rows, the centered line `The operator is switching threads`,
  hidden cursor, and `f`'s last row (the tab bar) copied through.
- **Stream** — the broadcast's one piece of frame state: the last frame sent.
  `Next(f)` returns the `terminal.Render(last, f)` diff as a `Message` (or none if
  nothing changed) and advances; `Join()` renders the current frame from zero for
  a late or resyncing viewer. Pure; tests run without IO.
  - **Relationships:** 1:1 with a Hub; shared by all viewers, so each frame is
    rendered once, not per viewer.
  - **Future extensions:** the row-diff work #262 deferred lands in `Render`
    and this inherits it.
- **BroadcastCell / RenderedStatusRow.Control** — the tab bar's leftmost cell:
  `None` (no broadcaster wired: draw nothing), `Idle` (`⏺`), `Starting` (`⏺…`),
  `Live` (`LIVE ⏸`, bold white on red). `Control` is its zero-based, half-open
  column span; a click there toggles.
- **nextFontSize** — the viewer's pure fit step: given the current font size,
  the rendered screen's pixel size, and the viewport size, returns
  `clamp(floor2(current × min(viewW/screenW, viewH/screenH)), 4, 64)`, where
  `floor2` rounds down to 0.5px.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `Presenter.SetTap` | `cmd/internal/terminal/tap.go` | new | presenter goroutine |
| `Hub` | `cmd/internal/broadcast/hub.go` | new | goroutine, timers, subscriber channels |
| `Server` | `cmd/internal/broadcast/server.go` | new | `net/http`, listener |
| `Session` | `cmd/internal/broadcast/session.go` | new | hub + server + tunnel lifecycle |
| `Tunnel`, `FakeTunnel` | `cmd/internal/broadcast/tunnel.go`, `tunnel_fake.go` | new | external tunnel |
| `Cloudflared` | `cmd/internal/broadcast/cloudflared.go` | new | `cloudflared` binary |
| `Console` broadcast wiring | `cmd/internal/couchtty/console_broadcast.go` | new | Session, Presenter tap, clipboard |
| `couchcmd` wiring | `cmd/internal/couchcmd/run.go` | modified | env options, real tunnel |

- **Hub** — receives offers (latest wins, never blocks), applies
  `IndicatorShown` and `ViewerFrame`, drives the `Stream`, fans messages out to at
  most `MaxViewers` (16) subscribers with an 8-message queue each. A full queue
  marks that viewer for resync: it gets `Join()` at the next frame instead of the
  diffs it missed. Frames without the indicator are withheld; if the indicator
  stays absent for `Grace` (1s; armed at start too), the hub ends with
  `ErrIndicatorHidden`.
  - **Injected into:** `Session`. Timer creation is injected
    (`HubOptions.After`) so grace tests are deterministic.
- **Server** — `http.Handler` rooted at `/<token>/`: `GET /<token>/` (page),
  `/<token>/viewer.js`, `/<token>/xterm.js`, `/<token>/xterm.css` (embedded),
  `/<token>/events` (SSE). Any other method is 405; any other path, or a wrong
  token, is 404 (constant-time compare). Every response sets
  `Cache-Control: no-store`, `Referrer-Policy: no-referrer` (the token is in the
  path), `X-Content-Type-Options: nosniff`, and
  `Content-Security-Policy: default-src 'none'; script-src 'self'; style-src 'self'; connect-src 'self'; img-src 'self'; frame-ancestors 'none'`.
  SSE events: `event: frame` with `data: {"cols":C,"rows":R,"b":"<base64>"}`,
  `event: end` with `data: {"reason":"…"}`, and a `: ping` comment every 15s
  (Cloudflare closes idle streams at 100s).
- **Session** — `Start(ctx, Config) (*Session, error)`: mints a 32-byte
  `crypto/rand` base64url token, listens on `127.0.0.1:0`, opens the tunnel, and
  returns once the public URL is known. `Link()` is `<public>/<token>/`.
  `Offer`, `Stop(reason)`, `Done()`, `Err()`. Phases: `Starting → Live →
  Ended`; `Stop` during `Starting` cancels the tunnel open.
- **Tunnel / FakeTunnel** — `Open(ctx, localURL) (Handle, error)`, where a
  `Handle` has `URL() string`, `Exited() <-chan struct{}`, and `Close() error`.
  The fake is stateful: it records opens and closes, returns the local URL as the
  public URL, can be told to delay `Open` or to exit, and fails a second
  `Close`. A `LocalOnly` tunnel (identity URL) serves `COUCH_BROADCAST_TUNNEL=off`.
- **Cloudflared** — runs
  `cloudflared tunnel --no-autoupdate --url http://127.0.0.1:PORT` in its own
  process group, scans stderr for the first `https://[a-z0-9-]+\.trycloudflare\.com`
  (30s limit), and `Close` sends SIGTERM to the group, then SIGKILL after 3s. It
  writes a pidfile so a later Couch can reap one orphaned by a crash (see
  ARCH-FUNERAL).
- **Console broadcast wiring** — owns a `broadcastPhase` tagged enum
  (`off | starting | live`), toggles it from the click and the key, installs and
  removes the tap, copies the link with `Presenter.Copy`, posts notices, and stops
  the session when it ends on its own (grace expiry, tunnel exit) or when Couch
  shuts down.

### Operating envelope (ARCH-CONSTRAINTS)

- **Path:** the tap runs on the Presenter goroutine (the paint path). It
  clones one frame (≈ cols×rows cells; 200×60 = 12k cells) and stores a pointer;
  no IO, no lock contention beyond one mutex. Without an active broadcast there
  is no tap and no clone.
- **Rate:** bounded by the Presenter's `FrameInterval` coalescing; the hub
  coalesces further (latest wins), so a slow hub drops intermediate frames, never
  stalls the operator.
- **Viewers:** max 16 (503 beyond); 8 queued messages each, then resync. A full
  render of 200×60 is on the order of tens of KB, ×4/3 for base64. Bandwidth is
  not a concern (issue Log, 2026-10-06).
- **Startup:** tunnel URL within 30s or the start fails with a notice.

### Trust boundaries (ARCH-SECURE)

- **Internet → local server (untrusted).** Reached only through the tunnel.
  Accepts GET/HEAD only; never reads a request body; the 256-bit token is the
  only credential, compared in constant time; nothing from the request is
  reflected. The token is never drawn into a frame (the notice says "link
  copied", not the link), so it can't leak through the broadcast itself.
- **Cloudflare edge** sees plaintext frames; #406 closes this.
- **Child output.** Viewers receive only our own `Render` output, the same
  bytes the operator's terminal gets minus parent-mode and effect controls.
  OSC 8 hyperlinks are carried through; xterm.js's default link handler asks
  before opening, and we add no link addon.
- **Viewer page.** Served from the binary; CSP forbids any other origin; no
  storage APIs used.
- **cloudflared binary** is trusted from `PATH`, like `zellij`.

### Ordering (ARCH-ORDER)

Durable state: the session phase (Console) and the hub's last frame and
subscriber set. Events and their orders:

| State | Event | Result |
|-------|-------|--------|
| off | toggle | starting; `Session.Start` runs off the input path |
| starting | toggle | cancel the start; off (a start that completes after cancel is stopped, not adopted) |
| starting | start fails | off + error notice |
| starting | start succeeds | live; `SetTap`, repaint chrome, copy link |
| live | toggle | `SetTap(nil)`, then `Stop`, then off + repaint |
| live | `Done()` (grace, tunnel exit) | `SetTap(nil)`, off, notice with the reason |
| any | Couch shutdown | stop before Presenter release |

In the hub, one goroutine owns the stream and the subscriber set, so a join and
a frame are totally ordered: a joiner gets `Join()` of the frame before the next
diff. Tests inject the order: `FakeTunnel` delays, a manual timer for grace, and
explicit offer/subscribe interleavings.

### Lifetimes (ARCH-FUNERAL)

- **Hub frame and subscribers:** in memory; die with the session.
- **Token:** in memory; dies with the session (the server stops serving it).
- **Listener:** closed by `Stop` (`Server.Shutdown`, 2s, then `Close`).
- **cloudflared process:** created by `Start`; removed by `Stop` (group SIGTERM,
  then SIGKILL). A Couch crash orphans it to PID 1 (lessons.md, #399). The
  pidfile `<pair data dir>/couch/broadcast-tunnel.pid` (pid + process start time)
  is removed by `Stop`. On the next broadcast start, and at Couch startup, a
  surviving pidfile whose pid is alive, whose start time matches, and whose
  command is `cloudflared` gets that process group killed and the file removed.
  At most one file; size bounded.
- **Viewer page:** xterm.js's in-page screen only; closing the tab ends it.

---

## Chunk 1 — M1: frame tap and hub

### Task 1.1: FrameClass and the Presenter tap

**Files:**
- Create: `cmd/internal/terminal/tap.go`, `cmd/internal/terminal/tap_test.go`
- Modify: `cmd/internal/terminal/presenter.go` (struct fields; `paintPublication`
  after `p.previous = f.Clone()`; `Select`; `Panel`)
- Modify: `cmd/internal/couchtty/console_menu.go:228` and every test caller of
  `Panel` (`grep -rn '\.Panel(' cmd`)

- [ ] **Step 1: Write failing tests** in `tap_test.go`, using the existing
  presenter test helpers (`grep -n 'func newTestPresenter\|NewPresenter(' cmd/internal/terminal/*_test.go`):
  - `TestTapSeesEveryPaintedFrame`: select an endpoint, publish output, update
    chrome, resize. The tap receives a frame after each, equal to what was
    painted, with `FramePublic`.
  - `TestTapClassFollowsPanel`: `Panel(ctx, f, FramePrivate)` → the tap gets
    `FramePrivate`; a following `Select` → `FramePublic`.
  - `TestTapGetsAnOwnedClone`: mutating the received frame's cells does not change
    the next diff the presenter writes.
  - `TestTapNotCalledOnFailedPaint`: with a writer that fails, the tap is not
    called.
  - `TestSetTapNilStopsDelivery`.
- [ ] **Step 2:** `go test ./cmd/internal/terminal/ -run Tap` → FAIL (undefined).
- [ ] **Step 3: Implement.**

```go
// tap.go
package terminal

import "context"

// FrameClass says whether a presented frame may leave the operator's screen.
// Private frames (the switcher, which lists the whole fleet) are replaced
// before they reach any viewer.
type FrameClass uint8

const (
	FramePublic FrameClass = iota
	FramePrivate
)

// Tap observes each frame after it has been fully painted to the parent.
// It runs on the Presenter goroutine: it must not block, and it receives an
// owned clone.
type Tap func(Frame, FrameClass)

// SetTap installs or (nil) removes the tap, ordered with paints.
func (p *Presenter) SetTap(ctx context.Context, t Tap) error {
	return p.call(ctx, func(context.Context) error { p.tap = t; return nil })
}
```

  In `Presenter`: add `tap Tap` and `class FrameClass`. In `paintPublication`,
  right after `p.previous = f.Clone()`:

```go
	if p.tap != nil {
		p.tap(p.previous.Clone(), p.class)
	}
```

  `Select` sets `p.class = FramePublic` where it sets `p.selected`. `Panel`
  becomes `Panel(ctx context.Context, f Frame, class FrameClass) error` and sets
  `p.class = class` where it sets `p.selected = nil`. `showMenu` passes
  `terminal.FramePrivate`; test callers pass `terminal.FramePublic`.
- [ ] **Step 4:** `go test ./cmd/internal/terminal/ ./cmd/internal/couchtty/` → PASS.
- [ ] **Step 5: Commit** `#395 M1: terminal: presenter frame tap with privacy class`.

### Task 1.2: Indicator, privacy, stream (pure)

**Files:**
- Create: `cmd/internal/broadcast/{indicator,privacy,stream}.go` and colocated
  `_test.go` files.

- [ ] **Step 1: Write failing tests.**
  - `TestIndicatorShown`: a frame whose last row starts with `LiveLabel` → true;
    label on another row, clipped (`cols < width(LiveLabel)`), absent, or a
    blank resize row → false. Build rows with `terminal.StyledRows`.
  - `TestViewerFramePublicUnchanged`; `TestViewerFrameShowSwitcherUnchanged`.
  - `TestViewerFramePrivateIsPlaceholder`: same geometry; no cell of the
    original body (put a marker `FLEET-SECRET` in it) survives; the last row
    equals the input's last row; the placeholder text is present; cursor hidden.
  - `TestStreamJoinThenDiffsReproduceScreen`: feed frames A, B, C through
    `Next`; separately `Join()` after B and then `Next(C)`. Interpret both byte
    sequences with the vt emulator (`third_party/vt`, see how
    `cmd/internal/terminal` tests build one: `grep -rn 'vt.NewEmulator\|vt.New' cmd/internal/terminal/*_test.go`)
    at the frame geometry; both screens equal C's text.
  - `TestStreamUnchangedFrameYieldsNoMessage`.
  - `TestStreamClearClearsViewer`: A contains a marker; B is blank; the screen
    after A then B has no marker.
  - `TestStreamGeometryChange`: a message carries the new cols/rows, and its bytes
    redraw fully (Render emits `\x1b[2J` on geometry change).
- [ ] **Step 2:** `go test ./cmd/internal/broadcast/` → FAIL.
- [ ] **Step 3: Implement.**

```go
// indicator.go
package broadcast

// LiveLabel is drawn by couchtty's status row while live and checked here on
// every frame; one constant so the drawer and the checker can't disagree.
const (
	LiveLabel = "LIVE ⏸"
	IdleLabel = "⏺"
)

// IndicatorShown reports whether f's last row begins with LiveLabel.
func IndicatorShown(f terminal.Frame) bool { /* walk last-row cells, matching
	each grapheme of LiveLabel in order, skipping width-0 continuation cells */ }
```

```go
// stream.go
// Message is one unit of the broadcast: a rendered update at a geometry.
type Message struct {
	Cols, Rows int
	Data       []byte
}

// Stream holds the last frame sent. Each frame is rendered once and shared.
type Stream struct{ last terminal.Frame; started bool }

func (s *Stream) Next(f terminal.Frame) (Message, bool, error) {
	data, err := terminal.Render(s.last, f)
	if err != nil { return Message{}, false, err }
	s.last, s.started = f, true
	if len(data) == 0 { return Message{}, false, nil }
	return Message{Cols: f.Geometry.Cols, Rows: f.Geometry.Rows, Data: data}, true, nil
}

func (s *Stream) Join() (Message, bool, error) {
	if !s.started { return Message{}, false, nil }
	data, err := terminal.Render(terminal.Frame{}, s.last)
	if err != nil { return Message{}, false, err }
	return Message{Cols: s.last.Geometry.Cols, Rows: s.last.Geometry.Rows, Data: data}, true, nil
}
```

  `ViewerFrame` builds the placeholder with `terminal.StyledRows` for the body
  (`rows-1` rows, text centered) plus `f`'s last-row cells, then
  `terminal.PanelFrame` with a hidden cursor.
- [ ] **Step 4:** tests PASS.
- [ ] **Step 5: Commit** `#395 M1: broadcast: indicator, privacy and stream`.

### Task 1.3: Hub

**Files:**
- Create: `cmd/internal/broadcast/hub.go`, `hub_test.go`

- [ ] **Step 1: Write failing tests** (manual timer via `HubOptions.After`):
  - `TestHubLateJoinerGetsCurrentFrame`: offer A (with indicator), subscribe,
    first message = `Join()` of A; offer B → the diff to B.
  - `TestHubWithholdsFramesWithoutIndicator`: an offered frame without the
    indicator produces no message; a subsequent frame with it does.
  - `TestHubStopsWhenIndicatorHiddenPastGrace`: withheld frames, fire the
    timer → subscribers get `End{Reason: ErrIndicatorHidden}`, channels close,
    `Done()` closes, `Err()` is `ErrIndicatorHidden`. Also at start: no frame
    before the timer fires → same end.
  - `TestHubIndicatorReturnsBeforeGrace`: absent, then present before the
    timer → no end, and the timer is cancelled.
  - `TestHubPrivateFramePlaceholderByDefault` and `…ShownWithOption`:
    frame-level, via the emulator: the viewer screen lacks/has `FLEET-SECRET`.
  - `TestHubSlowViewerResyncs`: never read one subscriber until its queue
    overflows; a fast subscriber still gets every diff; when the slow one reads
    again, its next message is a full render and its screen equals the current
    frame.
  - `TestHubOfferNeverBlocks`: 10k offers with no hub progress return
    promptly (the hub is paused via an option hook).
  - `TestHubMaxViewers`: the 17th `Subscribe` → `ErrTooManyViewers`.
  - `TestHubCloseIdempotent`.
- [ ] **Step 2:** FAIL.
- [ ] **Step 3: Implement.** One goroutine owns `Stream`, the subscriber set
  and the grace timer. `Offer` stores the latest `(frame, class)` under a mutex
  and pokes a `chan struct{}` of capacity 1. `Subscribe` and `Close` are requests
  into the goroutine, so they are ordered with frames. Per-subscriber state:
  `queue chan Message` (cap 8), `resync bool`. Delivery per frame: compute the
  diff once, and `Join()` once if any subscriber is resyncing; non-blocking send;
  on full set `resync`. End: send a final `Message{End: true, Reason: …}` when
  there is room, then close each queue.
- [ ] **Step 4:** PASS, also with `-race -count=20`.
- [ ] **Step 5: Commit** `#395 M1: broadcast: hub with withholding, grace stop and resync`.
- [ ] **M1 close:** `sdlc milestone-close --issue 395 --milestone M1`.

## Chunk 2 — M2: local server and viewer page

### Task 2.1: Vendor xterm.js

**Files:**
- Create: `cmd/internal/broadcast/web/vendor/xterm/{xterm.js,xterm.css,LICENSE,VENDOR.md}`

- [ ] **Step 1:** in the scratchpad: `npm pack @xterm/xterm@5.5.0`; extract it
  into its own empty directory; copy `lib/xterm.js`, `css/xterm.css` and
  `LICENSE`. `VENDOR.md` records the package, version, tarball sha512 (from
  `npm view @xterm/xterm@5.5.0 dist.integrity`) and the copy commands. Same
  version as `tests/terminal-oracle`'s `@xterm/headless`, so the oracle
  that already checks `Render` output checks the parser viewers run.
- [ ] **Step 2:** classify the new files in the production artifact inventory
  (`TestProductionArtifactReferencesAreExactlyClassified`, lessons.md #399).
- [ ] **Step 3: Commit** `#395 M2: broadcast: vendor @xterm/xterm 5.5.0`.

### Task 2.2: Viewer page

**Files:**
- Create: `cmd/internal/broadcast/web/{index.html,viewer.css,viewer.js}`,
  `tests/broadcast-viewer/fit.test.mjs` (run with `node --test`; add it to the
  Makefile's test target next to the terminal oracle)

- [ ] **Step 1: Write the failing node test** for `nextFontSize` (exported from
  `viewer.js` as an ES module): fits the width-bound and height-bound cases,
  clamps to [4, 64], rounds down to 0.5, and is stable (returns `current` when
  the screen already fits within 0.5px).
- [ ] **Step 2:** FAIL.
- [ ] **Step 3: Implement.** `index.html` loads `xterm.css`, `viewer.css`,
  `xterm.js` and `viewer.js` by relative path, with no inline script or style
  (the CSP forbids them). `viewer.js`:
  - `new Terminal({scrollback: 0, disableStdin: true, cursorBlink: false, fontSize: 16, fontFamily: 'ui-monospace, Menlo, monospace'})`;
    no `onData` handler.
  - `new EventSource('events')`. On `frame`: resize the terminal to the
    message's cols×rows if they differ; `term.write(base64 → Uint8Array)`; fit.
  - On `end`: `term.reset()`, close the EventSource, show "Broadcast ended" (and
    the reason). On EventSource error: show "Reconnecting…"; on a successful
    reconnect the first message is a full render, so nothing else is needed.
  - Fit: measure `.xterm-screen`, apply `nextFontSize` up to three times on each
    frame geometry change and each window `resize`.
  - Uses no storage API (`localStorage`, `sessionStorage`, `indexedDB`,
    `caches`, service workers).
- [ ] **Step 4:** node test PASS.
- [ ] **Step 5: Commit** `#395 M2: broadcast: viewer page with auto-fit font`.

### Task 2.3: Server

**Files:**
- Create: `cmd/internal/broadcast/server.go`, `server_test.go`, `web.go`
  (`//go:embed web`)

- [ ] **Step 1: Write failing tests** with `httptest.NewServer` over a real hub:
  - `TestServerRefusesWrongTokenAndPaths`: 404 for a missing or wrong token
    and for unknown paths under the right token.
  - `TestServerIsGetOnly`: POST, PUT, DELETE, PATCH on every route → 405; no
    request body is read (use a body reader that fails the test if read).
  - `TestServerHeaders`: CSP exactly as specified, `no-store`, `no-referrer`,
    `nosniff`, on every route.
  - `TestViewerPageLoadsOnlySameOrigin`: parse `index.html`; every
    `src`/`href` is relative; no inline `<script>` body or `style=` attribute.
    And a static check that `viewer.js` names no storage API.
  - `TestServerSSEStreamsFrames`: subscribe via `/events`; offered frames arrive
    as `event: frame`, the decoded bytes reproduce the screen (emulator); a
    `: ping` arrives within the (test-shortened) interval; closing the hub sends
    `event: end`.
  - `TestServerTooManyViewers` → 503.
- [ ] **Step 2:** FAIL. **Step 3:** implement as specified in Core concepts.
  **Step 4:** PASS. **Step 5: Commit** `#395 M2: broadcast: GET-only SSE server`.

### Task 2.4: Session and tunnel seam

**Files:**
- Create: `cmd/internal/broadcast/{session,tunnel,tunnel_fake}.go` and tests.

- [ ] **Step 1: Write failing tests** (with `FakeTunnel`):
  - `TestSessionLinkServesViewer`: `Start` → `Link()` = fake URL + `/<token>/`;
    GET returns the page; tokens differ across sessions and are 43 base64url
    characters.
  - `TestSessionStopEndsViewersAndRevokesToken`: after `Stop`, the open SSE
    gets `end`, a new GET fails, the fake records the close, the listener is
    closed.
  - `TestSessionCancelDuringStart`: fake delays `Open`; cancel ctx → `Start`
    returns `context.Canceled`; the fake shows open-then-close (a late open is
    closed, not leaked); the listener is closed.
  - `TestSessionTunnelExitEndsSession`: fake exits → `Done()` closes with
    `ErrTunnelExited`; viewers get `end`.
  - `TestSessionPersistsNoFrameData`: point `HOME`, `TMPDIR`, `XDG_*` and
    the working directory at `t.TempDir()` subdirectories; run a session; offer
    frames containing `BROADCAST-MARKER-395`; stream them to an HTTP viewer;
    stop. Walk those directories: no file contains the marker.
- [ ] **Step 2–4:** FAIL → implement → PASS (`-race`).
- [ ] **Step 5: Commit** `#395 M2: broadcast: session lifecycle over a tunnel seam`.
- [ ] **M2 close:** `sdlc milestone-close --issue 395 --milestone M2`.

## Chunk 3 — M3: Couch control

### Task 3.1: Status-row control cell

**Files:**
- Modify: `cmd/internal/couchtty/reserve.go` (`StatusModel`, `RenderedStatusRow`,
  `RenderStatusRow`), `reserve_test.go`

- [ ] **Step 1: Write failing tests:**
  - `BroadcastNone` draws nothing; the existing row tests stay unchanged.
  - `Idle` draws `⏺` first, then the capture badge; `Control` covers exactly its
    columns.
  - `Live` draws `LIVE ⏸` with the red SGR at column 0, and
    `broadcast.IndicatorShown` is true on a frame whose last row is the
    rendered row (`terminal.StyledRows`). This is the drawer↔checker contract.
  - A width narrower than the label clips it, and `IndicatorShown` is false (the
    fail-safe input).
  - Chip spans shift right by the control's width and stay click-accurate.
- [ ] **Step 2–4:** FAIL → add `Broadcast BroadcastCell` to `StatusModel`,
  `Control ColumnSpan` to `RenderedStatusRow`, and draw it before
  `captureBadge` through the same `appendText` → PASS.
- [ ] **Step 5: Commit** `#395 M3: couchtty: broadcast control cell in the status row`.

### Task 3.2: Key binding

**Files:**
- Modify: `cmd/internal/couchkeys/couchkeys.go`, `cmd/internal/couchtty/keys.go`
  (`HitBroadcast`, `seqBroadcast`, `dispatchFor` arm), their tests.

- [ ] **Step 1:** add the binding row:
  `{Action: ActionBroadcast, Scope: ScopeEveryPane, Key: "Ctrl+Alt+b", Help: "start or stop broadcasting this screen (view-only link)", Encodings: [][]byte{[]byte("\x1b\x02"), []byte("\x1b[98;7u")}}`.
  Check the encodings against how `ChordCtrlAltN` is encoded in
  `workbenchshortcut` and match its forms. Ctrl+Alt+b is free in Pair's chord
  table; plain Alt+b is not used because it is the shell's word-back.
- [ ] **Step 2:** `TestEveryCouchActionDispatches` fails until the
  `dispatchFor` arm exists; add it. Help rendering tests update.
- [ ] **Step 3: Commit** `#395 M3: couchkeys: Ctrl+Alt+b toggles broadcast`.

### Task 3.3: Console wiring

**Files:**
- Create: `cmd/internal/couchtty/console_broadcast.go`, `console_broadcast_test.go`
- Modify: `cmd/internal/couchtty/console.go` (fields, `hitHandlers`, teardown
  before Presenter release), `terminal_input.go` (`routeMouseEvent`: check
  `Control` before `ColumnToActor`), `console_presentation.go`
  (`statusModelLocked` fills `Broadcast`), `terminal.go` (`commitChrome` stores
  `Control`).

- [ ] **Step 1: Write failing tests** using the existing Console test harness
  (`grep -n 'func newTestConsole\|func startConsole' cmd/internal/couchtty/*_test.go`)
  and `broadcast.FakeTunnel`:
  - `TestBroadcastClickStartsAndStops`: a click on the control column → the
    session starts, the row shows `LIVE ⏸`, a clipboard copy of the link was
    written, and a viewer GET of the link receives the composed screen
    (emulator text includes an actor's output and the tab bar). A second click
    → the viewer gets `end`, the row shows `⏺`.
  - `TestBroadcastKeyToggles`: the same through Ctrl+Alt+b, from a Pair pane.
  - `TestBroadcastSwitcherPrivate`: open the switcher while live → the viewer
    sees the placeholder, not a thread name; with `ShowSwitcher` → it sees the
    switcher.
  - `TestBroadcastStopsWhenIndicatorCannotBeDrawn`: shrink the host so the
    label clips → after grace the session ends, the notice says why, the phase
    is off.
  - `TestBroadcastToggleWhileStarting`: fake delays `Open`; toggle twice → no
    session left, the fake shows a close.
  - `TestBroadcastStoppedOnShutdown`: stop the Console while live → the viewer
    gets `end` and the fake shows a close, before Presenter release.
  - `TestBroadcastNoBroadcasterNoCell`: with no broadcaster configured, the row
    is unchanged and the key passes through to the child.
- [ ] **Step 2:** FAIL.
- [ ] **Step 3: Implement** the ordering table above. `SetBroadcaster(cfg
  broadcast.Config)` configures it (nil config → `BroadcastNone`). Start runs
  in a goroutine on the Console lifetime; its completion re-enters through the
  Console's existing operation path, checking that the phase is still
  `starting` with the same attempt id. A mismatch stops the late session. The
  tap is `session.Offer`. The notice text is "Broadcast live — link copied" and
  never contains the link.
- [ ] **Step 4:** PASS (`-race`).
- [ ] **Step 5: Commit** `#395 M3: couchtty: broadcast toggle, tap and fail-safe`.

### Task 3.4: couchcmd options

**Files:**
- Modify: `cmd/internal/couchcmd/run.go` (`consoleRunnerFor`), its tests.

- [ ] **Step 1:** read `COUCH_BROADCAST_SWITCHER` (`show` → ShowSwitcher;
  unset → hidden; anything else → startup error naming the variable) and
  `COUCH_BROADCAST_TUNNEL` (`off` → `LocalOnly`; unset → `Cloudflared`;
  anything else → error). Test both parse paths.
- [ ] **Step 2:** local smoke: `COUCH_BROADCAST_TUNNEL=off couch`, Ctrl+Alt+b,
  open the copied `http://127.0.0.1:…/<token>/` in a browser, then switch
  threads, open the switcher, resize, `clear`, and stop. **Ask the operator to
  run this smoke** (memory: dogfood live), and record what they see in the Log.
- [ ] **Step 3: Commit** `#395 M3: couchcmd: broadcast options`.
- [ ] **Atlas:** `atlas/broadcast.md` (tap, hub, server, session, privacy,
  fail-safe, lifetimes), link it from `atlas/index.md`, and add one paragraph to
  `atlas/terminal.md` on the tap and `FrameClass`.
- [ ] **M3 close:** `sdlc milestone-close --issue 395 --milestone M3`.

## Chunk 4 — M4: cloudflared and live smoke

### Task 4.1: Cloudflared tunnel

**Files:**
- Create: `cmd/internal/broadcast/cloudflared.go`, `cloudflared_test.go`

- [ ] **Step 1: Write failing tests** with a fake `cloudflared` script on a
  temp `PATH`. The script prints a quick-tunnel banner to stderr and sleeps;
  variants never print a URL, or exit early.
  - Parses the URL from stderr; times out (shortened in the test) with an error
    naming cloudflared; an early exit is an error.
  - `Close` terminates the process group (the script spawns a child; both are
    gone), escalating to SIGKILL when the script ignores TERM.
  - The pidfile is written on start and removed on close.
  - `ReapOrphan`: a pidfile for a live process with a matching start time and
    command `cloudflared` is killed and removed; a mismatched start time or
    command is left alone and the stale file removed. Each test may remove only
    paths under its own `t.TempDir()` (lessons.md, #399).
  - `cloudflared` not on `PATH` → `ErrNoCloudflared`, with a notice saying how
    to install it (`brew install cloudflared`).
- [ ] **Step 2–4:** FAIL → implement → PASS.
- [ ] **Step 5:** call `ReapOrphan` at Couch startup and before each start.
  **Commit** `#395 M4: broadcast: cloudflared quick tunnel with orphan reaping`.

### Task 4.2: Live smoke and close

- [ ] **Step 1:** `make build` on `pair:0` after a slot move (CLAUDE.local).
  **Ask the operator** to start a broadcast and to open the link on another
  device or network: live updates at the sender's grid with the font fitted, a
  late join shows the current screen, the switcher placeholder appears,
  `clear` clears the viewer, stopping shows "Broadcast ended", and the old link
  no longer loads. Check that no `cloudflared` remains (`pgrep -fl cloudflared`).
- [ ] **Step 2:** full verification per memory: `make -k test`, the changelog
  test with a scratchpad `TMPDIR`, `go test ./...`, and
  `(cd third_party/vt && go test ./...)` if it was touched. Grep any known-failing
  test's output for broadcast files.
- [ ] **Step 3:** `sdlc close --issue 395 --verified '<evidence>'`.
