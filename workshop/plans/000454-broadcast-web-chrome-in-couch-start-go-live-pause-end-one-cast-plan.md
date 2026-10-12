# broadcast-web: Chrome in Couch — Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Shift+Ctrl+Alt+B in Couch starts a dedicated Chrome driven over CDP. The operator can go live, pause, resume and end one cast from Couch's tab bar or from a button inside the page. A stub viewer page shows the cast's live state and the active tab's URL and title, over the broadcast's transport.

**Architecture:** There are three layers, each with one seam:
1. `cdp` is a small Chrome DevTools Protocol client over Chrome's `--remote-debugging-pipe` (no TCP port, no websocket dependency).
2. `webcast` is the browser-control component. Its `Browser` interface carries state, live, active tab and end, and no CDP type crosses it. Behind it are the real Chrome and a stateful `FakeBrowser`. `webcast.Cast` composes a `Browser` with a viewer endpoint and the stub stream.
3. Couch's console only talks to `Cast` (and through it, `Browser`), the same way it talks to `broadcast.Session`.

The tunnel/listener/probe half of `broadcast.Start` is extracted as `broadcast.Endpoint`, so both casts share it.

**Tech Stack:** Go standard library (os/exec, encoding/json, net/http, SSE), Chrome ≥ 125 (CDP `Page.addScriptToEvaluateOnNewDocument{runImmediately}`, `Runtime.addBinding{executionContextName}`); verified on Chrome 154. Viewer page is plain JS, no vendored libraries.

---

## Spike results (2026-10-11, Chrome 154.0.8037.98)

A throwaway Go program (scratchpad, not committed) launched Chrome with `--remote-debugging-pipe` (fd 3 = commands
in, fd 4 = replies/events out, each message NUL-terminated JSON) and measured:

- `Target.setDiscoverTargets` reports `page` targets and also `browser_ui`, `background_page`, `service_worker` and
  `other` targets. Only `type == "page"` is a tab.
- `Target.attachToTarget{flatten:true}` gives a `sessionId`; later commands carry `sessionId` at the top level.
- `Runtime.addBinding{name, executionContextName:"couchcast"}` plus `Page.addScriptToEvaluateOnNewDocument{worldName:
  "couchcast", runImmediately:true}` injects into the **current** `about:blank` and into every later document. The
  binding fires `Runtime.bindingCalled` with the payload. In the page's own world, `typeof window.__couchcast` is
  `"undefined"`: the page cannot see or call the binding.
- `Runtime.enable` reports each isolated context (`name:"couchcast"`, `auxData.frameId`), which is the `contextId`
  used to push state into the page.
- `Target.targetInfoChanged` carries the URL and title on every navigation; no injection is needed for those.
- **Closing our end of the pipe makes Chrome exit** (exit status 0 within the 5s window). Chrome therefore dies with
  its owner: a crashed Couch cannot leave a cast Chrome running.
- Inside the agent sandbox Chrome exits at once (status 21); the live check runs unsandboxed.

The spike ran headless, so visibility under a real window (background tab `hidden`, two windows both `visible`) is
confirmed by the live check (Task 9), not assumed.

## Key-decoding check (done before design)

Couch pushes kitty keyboard flags 3 (`\x1b[>3u`, presenter.go). Its decoder plus `productKey` yields
`\x1b[98;7u` for Ctrl+Alt+b and `\x1b[98;8u` for Shift+Ctrl+Alt+B. A terminal that reports the shifted code sends
`\x1b[66;8u`, and the event-type and alternate-key forms (`98;8:1u`, `98:66;8u`) also canonicalise to `\x1b[98;8u`. All
were measured through `terminal.Decoder` + `productKey`. The legacy `ESC ^B` is not a broadcast binding at all
(#395), so the two chords are distinct. The operator's Ghostty config binds nothing with ctrl+alt+shift. The key on
the real terminal is confirmed in the smoke test.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `cdp.Message` / framing (`encode`, `splitFrames`) | `cmd/internal/cdp/frame.go` | new |
| `webcast.Tab`, `webcast.State`, `webcast.Phase` | `cmd/internal/webcast/state.go` | new |
| `webcast.Tracker` (active-tab reducer) | `cmd/internal/webcast/tracker.go` | new |
| `webcast.translate` (CDP event → tracker input) | `cmd/internal/webcast/translate.go` | new |
| `webcast.viewerState` (what a viewer may see) | `cmd/internal/webcast/stream.go` | new |
| `ChromeCell` + its span in `RenderStatusRow` | `cmd/internal/couchtty/reserve.go` | modified |

- **cdp framing** turns NUL-delimited bytes into `Message{ID, Method, Params, Result, Error, SessionID}` and back.
  Unit tests cover split reads, multiple messages in one read and a frame over the size cap.
- **State / Phase:** `Phase` is a tagged enum, `PhaseIdle` (Chrome running, not cast) | `PhaseLive` | `PhaseEnded`,
  never a `running && live` boolean pair. `State{Phase, Tab Tab, HasTab bool}` is the whole published state of a
  Browser. `Tab{ID, URL, Title string}`; `ID` is an opaque string (Chrome's targetId), not a CDP type.
  - **Relationships:** a Browser publishes 1 State at a time. Cast and Couch read it.
  - **Future extensions:** the Recorder adds nothing here; it subscribes to the same State to know when to snapshot
    (Live edge, Tab.ID change, navigation).
- **Tracker** is a pure reducer over ordered inputs: `TargetCreated{ID,URL,Title}`, `TargetChanged{ID,URL,Title}`,
  `TargetDestroyed{ID}` and `PageReport{ID, Visible, Focused}`. `Active()` returns the active tab and `Pages()` the
  page count. Rule, ordered by input sequence (a counter, not wall time, per ARCH-ORDER):
  - each page has `lastActivated` (the seq of its creation, or of a report that is visible-and-focused, or of a
    hidden→visible transition) and `reported`/`visible`;
  - the active tab is the page with the greatest `lastActivated` among those that are visible, or that never
    reported. Never-reported covers uninjectable pages such as `chrome://newtab`: a page Cmd+T opens is active until
    it reports otherwise.
  - Tests cover: single tab; Cmd+T new tab; switching back (old tab's report visible); background-opened tab (created,
    then reports hidden → not active); two windows, last focused wins; active tab destroyed → falls back to the
    previous; NTP never reports; and the order sensitivity of "hidden then visible" vs "visible then hidden".
  - **DRY rationale:** one place decides "which tab is cast"; the Recorder task reuses it unchanged.
- **translate** maps a decoded CDP event (method, params JSON, sessionId→targetId map) to zero or more Tracker
  inputs. It drops non-`page` targets and parses binding payloads. Its tests use JSON captured from the spike.
- **viewerState** maps State to the wire shape: `{live:false}` when not live, with **no URL or title**, so nothing the
  operator does while paused reaches a viewer. When live it is `{live:true,url,title}`; Ended is the `end` event.
  Tested for both phases.
- **ChromeCell:** `ChromeOff` (draw nothing) | `ChromeIdle` (`Chrome ▶`) | `ChromeLive` (`Chrome ⏸`). It is drawn
  **after** the screen broadcast's `LIVE ⏸ 👆 👽 💭N` cells, so the broadcast's fixed-column indicator check is
  untouched, and before the capture badge. `RenderedStatusRow.Chrome ColumnSpan` is its hit region, recorded
  only from the drawn frame (lesson #447 M4).

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `cdp.Conn` | `cmd/internal/cdp/conn.go` | new | Chrome's debugging pipe |
| `webcast.Browser` (interface) | `cmd/internal/webcast/browser.go` | new | the cast Chrome |
| `webcast.Chrome` (real Browser) | `cmd/internal/webcast/chrome.go`, `inject.js` | new | Chrome process + CDP |
| `webcast.FakeBrowser` | `cmd/internal/webcast/fake.go` (non-test, like `broadcast.FakeTunnel`) | new | stateful fake of Browser |
| `broadcast.Endpoint` | `cmd/internal/broadcast/endpoint.go` | new (extracted from `Session.Start`) | listener + http.Server + Tunnel + probe |
| `webcast.Server` + `web/` | `cmd/internal/webcast/server.go`, `web/index.html`, `web/viewer.js` | new | HTTP/SSE |
| `webcast.Cast` | `cmd/internal/webcast/cast.go` | new | Browser + Endpoint + stream |
| Couch webcast state machine | `cmd/internal/couchtty/console_webcast.go` | new | Cast |
| `ActionWebcast` key | `cmd/internal/couchkeys/couchkeys.go`, `couchtty/keys.go` | modified | input |
| launcher wiring | `cmd/internal/couchcmd/webcast.go`, `run.go` | new/modified | env, data dir, Chrome binary |

- **cdp.Conn:** `New(r io.Reader, w io.WriteCloser) *Conn`; `Call(ctx, sessionID, method string, params, result any) error`;
  `Events() <-chan Event` (`Event{SessionID, Method string; Params json.RawMessage}`); `Close()`; `Done()`; `Err()`.
  - A single reader goroutine routes replies by id and queues events in an internal FIFO that the pump hands to
    `Events()`. The queue is capped at `maxQueuedEvents = 4096`; past that the Conn fails with `ErrBacklog`, which
    a stuck consumer would cause and which is never silently dropped.
  - Messages are capped at 16 MiB (`ErrFrameTooLarge`).
  - Pending calls fail with the Conn's error when it ends (EOF = Chrome exited).
  - **Injected into:** `webcast.Chrome` only. Tests use `io.Pipe` pairs with a scripted peer.
  - **Future extensions:** the Recorder's DOM/Network domains ride the same Conn, through an accessor the Recorder
    task adds on `Chrome` (deliberately not now: YAGNI, and it would leak CDP past the seam before a consumer exists).
- **Browser:**
  ```go
  type Browser interface {
      State() State
      // Subscribe returns a channel that always holds the latest State (capacity 1,
      // replaced, never blocks the publisher) and a cancel func.
      Subscribe() (<-chan State, func())
      SetLive(live bool) error  // Idle<->Live; error once Ended
      Close() error             // end the cast and close Chrome; idempotent
      Done() <-chan struct{}    // closed when Chrome is gone (by Close, by hand, or crashed)
      Err() error               // why it ended: nil for Close, ErrClosedByOperator, or the failure
  }
  ```
  The in-page button and the Couch controls both go through `SetLive`. The real Browser calls it on itself from
  `bindingCalled{toggle}`.
- **Chrome:** `Launch(ctx, LaunchOptions{Binary, ProfileDir string; Stderr io.Writer}) (*Chrome, error)`.
  - Starts Chrome with `--remote-debugging-pipe --user-data-dir=<ProfileDir> --no-first-run
    --no-default-browser-check about:blank`, then runs `setDiscoverTargets`.
  - Per page target: `attachToTarget{flatten}`, then `Runtime.addBinding`, `Page.enable`, `Runtime.enable` and
    `Page.addScriptToEvaluateOnNewDocument{worldName, runImmediately}`.
  - `executionContextCreated` events with name `couchcast` on a main frame are tracked, so that `SetLive` can
    `Runtime.evaluate("__couchcastSet(true|false)", contextId)` in every attached page.
  - One owner goroutine (`run`) owns all mutable state. CDP events and `SetLive`/`Close` requests are both inputs to
    its select, which is the ARCH-ORDER answer: no lock-ordered constellations.
  - **Ends** on any of three things:
    - Chrome exits (cmd.Wait / Conn EOF) → `ErrClosedByOperator` if we didn't ask.
    - `Close()`: `Browser.close`, then wait for exit up to `closeWait = 5s`, then `Process.Kill`.
    - The page count drops to zero after having been ≥1 and stays at zero for `noPagesGrace = 1s`: on macOS, closing
      the last window leaves Chrome running windowless, so we then close it ourselves.
  - **Profile-in-use:** if Chrome exits within the start window before the first page target, `Launch` returns
    `ErrProfileInUse`-wrapped text naming the profile dir. Chrome's singleton hands a second launch with the same
    profile to the existing process and exits.
- **inject.js** runs in the `couchcast` isolated world and the top frame only (`window === window.top`). It draws a
  small fixed-position control inside a **closed** shadow root on a host element carrying
  `data-couchcast-ui`, so page CSS can't restyle it. The Recorder must skip that attribute; this is noted in atlas.
  - The control shows "● Go live" when idle and "❚❚ Pause" when live. Clicking calls
    `__couchcast('{"t":"toggle"}')`.
  - It reports `{"t":"vis","visible":…,"focused":…}` on load, `visibilitychange`, `focus` and `blur`.
  - It exposes `__couchcastSet(live)` in its own world.
  - The in-page pause is the same control; the spec's "stop control inside the browser" is that button while live.
- **FakeBrowser** is a stateful model with operator verbs `Navigate(url,title)`, `OpenTab(url,title, background bool)`,
  `Activate(id)`, `ClickPageControl()` and `QuitByHand()`. It drives the same `Tracker`, so the active-tab rule is not
  re-implemented in the fake. It also has `FailNextSetLive(err)` for error paths. A conformance suite (Task 9) runs
  one scenario against both the fake and real Chrome.
- **broadcast.Endpoint:** `OpenEndpoint(ctx, EndpointConfig{Tunnel, ProbeTimeout, Resolve}, handler func(token string)
  http.Handler) (*Endpoint, error)`.
  - Methods: `Link()` (`base/token/`), `Base()`, `Mode()`, `Exited() <-chan error` (a tunnel exit or server
    failure) and `Close()`.
  - Holds Listen → Serve → Tunnel.Open → probe → teardown, moved verbatim from `Session.Start`/`teardown`.
  - `Session` keeps its hub, pointer and notes and calls `OpenEndpoint`. Existing broadcast tests must stay green
    unchanged (ARCH-DRY: one tunnel lifecycle).
  - Also exported: `broadcast.SetSecurityHeaders(h http.Header, csp string)`, used by both servers.
- **webcast.Server** serves only under `/<token>/`: `""` → index.html, `viewer.js`, `events` (SSE). It sends events
  `state` (viewerState JSON) on join and on change, `: ping` every `broadcast.DefaultPing`, and `end {reason}`.
  - GET only, the same security headers, CSP `default-src 'none'; script-src 'self'; style-src 'self'
    'unsafe-inline'; connect-src 'self'; frame-ancestors 'none'`.
  - Viewer cap `broadcast.DefaultMaxViewers` (16); past it the stream answers 503.
- **Cast:** `StartCast(ctx, CastConfig{Launch func(ctx) (Browser, error); Endpoint broadcast.EndpointConfig})
  (*Cast, error)`.
  - Launches the Browser first: the operator sees Chrome at once. It then opens the Endpoint in the background, so
    `Link()` is `""` with `LinkErr()` nil while it opens.
  - It forwards Browser states to the Server's subscribers and ends everything when the Browser ends.
  - Methods: `State()`, `Subscribe()` (forwarded, plus link readiness as `Update{State, LinkReady bool, LinkErr error}`),
    `ToggleLive()`, `End()`, `Done()`, `Err()`.
- **Couch state machine** (`console_webcast.go`) mirrors `console_broadcast.go`'s phases and `startClaim`:
  `webcastOff → webcastStarting → webcastRunning → webcastEnding → webcastOff`.
  - Key/click in Running: ToggleLive. Key in Starting: a notice "Chrome is starting".
  - Ctrl+click (Ctrl with left or right button; macOS terminals may turn Ctrl+left into a right-click with the Ctrl
    bit still set): End.
  - Right-click: copy the link via `copyLink`, or the notice "viewer link not ready" / the link error.
  - Cast done → Off, plus a notice when the cause wasn't our End ("Chrome closed").
  - Couch shutdown → End, waiting at most `webcastShutdownWait = 6s`, beside `endBroadcastForShutdown`.

### Operating envelope (ARCH-CONSTRAINTS)

- **One** cast Chrome per Couch (the Couch state machine refuses a second start). There is one tab-tracker entry per
  open page, a few hundred bytes each; operator tab counts (tens) are fine.
- CDP event volume here is target lifecycle plus bindings, a handful per second. The Conn's 4096-event queue cap
  fails loudly; the Recorder task revisits it when DOM events arrive.
- Viewers: at most 16, as for the screen broadcast. One state event is <1 KiB, so the fan-out cost is negligible.
- Start latency: Chrome's window appears in about 1s. The link is ready after the tunnel probe, which takes 2–10s
  for a quick tunnel (the same as the screen broadcast).
- **Security (ARCH-SECURE):**
  - The pipe has no network surface, so other local processes cannot drive the operator's Chrome, unlike
    `--remote-debugging-port`.
  - The page's main world cannot see the binding. A page *can* see and click the injected DOM host. The worst it can
    do is toggle live/pause, which is visible in Couch's tab bar; accepted for MVP.
  - Viewers receive URL and title only while live. The link token is the only credential, as for the screen
    broadcast, and the link is never drawn.

### Lifecycles (ARCH-FUNERAL)

- **Chrome process:** created by `Launch`. Its owner is the Couch process through the pipe. It is removed by
  `Close()`, by the operator quitting Chrome, or by Chrome exiting when the pipe closes because Couch died
  (measured). Bound: 1.
- **Chrome profile dir** `<couch store dir>/webcast/chrome-profile`: created on the first cast and reused on purpose,
  so logins to the app under test survive across casts. One per store dir. Chrome bounds its own caches; the
  operator removes it by deleting the dir (documented in README). Cost: one profile on disk, typically tens to a few
  hundred MB.
- **Endpoint / tunnel:** created by `Cast` and removed with the Cast. The cloudflared guard and orphan reaper already
  cover a crashed Couch.
- **Viewer streams:** end with the Cast (an `end` event, then the server closes).
- **Nothing else is written to disk.** The stub stream is not recorded; recording is the Recorder task's.

## Task list

Milestone **M1 — browser-control component** (Tasks 1–5, 9a): `cdp`, `webcast` state/tracker/translate, Chrome,
inject, FakeBrowser, conformance + live check. **M2 — Couch integration and stub stream** (Tasks 6–8, 9b, 10): the
Endpoint extraction, server/viewer, Cast, Couch key/indicator/controls, launcher wiring, docs, smoke test.

### Task 1: cdp framing and Conn  (M1)

**Files:** Create `cmd/internal/cdp/frame.go`, `frame_test.go`, `conn.go`, `conn_test.go`, `doc.go`.

- [ ] Write failing tests:
  - `TestSplitFramesAcrossReads`: feeding `{"id":1}\x00{"id"` then `:2}\x00` yields ids 1 and 2.
  - `TestFrameTooLarge`.
  - `TestCallRoutesReplyByID`: the scripted peer answers id 2 before id 1, and both calls get their own results.
  - `TestCallErrorReply`: `{"id":1,"error":{"code":-32000,"message":"x"}}` → `*cdp.Error` with code and message.
  - `TestEventsInOrderWithSession`.
  - `TestPendingCallsFailOnEOF`: the peer closes, and Call returns an error wrapping `io.EOF`; `Done()` closes.
  - `TestBacklogFailsLoudly`: with `maxQueuedEvents` lowered via an unexported var in the test, an unread Events()
    → `Err()` is `ErrBacklog`.
  - `TestCallContextCancel`: ctx cancelled → Call returns `ctx.Err()`, and a late reply is discarded without panic.
- [ ] Run `go test ./cmd/internal/cdp/` and expect a compile failure.
- [ ] Implement: `encode(Message) []byte` (JSON + `\x00`), a `bufio.Reader.ReadSlice(0)`-based reader with a size cap,
  `Conn` with a `pending map[int64]chan Message` under a mutex, an atomic id counter, an event FIFO
  (slice + `sync.Cond`) and a pump goroutine to the `events` chan.
- [ ] Run the tests until they pass, then `go vet ./cmd/internal/cdp/`.
- [ ] Commit `#454 M1: cdp: pipe client with framing, call routing, bounded event queue`.

### Task 2: State, Tracker, translate (pure)  (M1)

**Files:** Create `cmd/internal/webcast/{doc.go,state.go,tracker.go,tracker_test.go,translate.go,translate_test.go,testdata/spike_events.jsonl}`.

- [ ] Tracker table test `TestTrackerActiveTab`. Each case is an input sequence plus the expected `Active().ID`:
  - single tab;
  - new tab (created) → active;
  - switch back (`PageReport{A,visible:true,focused:true}` after B created, then `PageReport{B,visible:false}`) → A;
  - background open (B created, B reports hidden) → A;
  - two windows (A and B both visible, B focused last) → B, then A focused → A;
  - active destroyed → previous;
  - all destroyed → none, `Pages()==0`;
  - NTP never reports → active by creation.
  Also `TestTrackerURLTitleFollowChanges`.
- [ ] translate tests over `testdata/spike_events.jsonl` (events captured in the spike: targetCreated for page and
  browser_ui, targetInfoChanged, bindingCalled vis/toggle, executionContextCreated):
  - `browser_ui`/`service_worker` produce nothing;
  - a page targetCreated produces `TargetCreated`;
  - bindingCalled `vis` produces `PageReport` for the targetId mapped from the sessionId;
  - bindingCalled `toggle` produces a `ToggleRequested` input;
  - a malformed payload produces nothing and no panic.
- [ ] Implement, run the tests until they pass, and commit `#454 M1: webcast: tab tracker and CDP event translation`.

### Task 3: Browser interface, FakeBrowser  (M1)

**Files:** Create `cmd/internal/webcast/{browser.go,fake.go,fake_test.go,subscribe.go}`.

- [ ] `subscribe.go`: `latest[T]` is a capacity-1 replace-on-send channel helper, shared by Fake, Chrome and Cast
  (ARCH-DRY). Test `TestLatestNeverBlocksAndKeepsNewest`.
- [ ] Fake tests:
  - start Idle with one `about:blank` tab;
  - `SetLive(true)` → Live; `ClickPageControl` toggles back;
  - `Navigate` updates the active tab's URL and title;
  - `OpenTab(background:false)` switches the active tab; `OpenTab(background:true)` doesn't;
  - `QuitByHand` → `Done` closes, `Err()==ErrClosedByOperator`;
  - `Close` → `Err()==nil`, and Close is idempotent;
  - `SetLive` after end → `ErrEnded`;
  - `FailNextSetLive`.
- [ ] Implement, run the tests until they pass, and commit `#454 M1: webcast: Browser seam and stateful FakeBrowser`.

### Task 4: inject.js  (M1)

**Files:** Create `cmd/internal/webcast/inject.js` (`//go:embed`); test `tests/webcast-inject/inject.test.mjs` and
`cmd/internal/webcast/inject_test.go`. The test runs `node --test` and skips if node is missing, as
`broadcast/viewer_test.go` does.

- [ ] Node test with a minimal DOM stub (document, window, shadow root stub) asserting:
  - nothing runs when `window !== window.top`;
  - one `vis` report on load and one per visibilitychange;
  - `__couchcastSet(true)` changes the label to Pause, and a click sends `toggle`;
  - the host has `data-couchcast-ui` and uses `attachShadow({mode:'closed'})`.
- [ ] Implement inject.js. It must be idempotent: re-injection on the same document (runImmediately plus a reattach)
  must not add a second control. Guard on an isolated-world global.
- [ ] Commit `#454 M1: webcast: in-page Go live / Pause control and visibility reporter`.

### Task 5: Chrome (real Browser)  (M1)

**Files:** Create `cmd/internal/webcast/{chrome.go,chrome_find.go,chrome_find_test.go}`.

- [ ] `chrome_find.go`: `FindChrome(getenv, lookPath, stat)`. Order: `COUCH_WEBCAST_CHROME` if set (it must exist,
  else error naming the variable, per the #451 lesson). Then the macOS app bundles (`/Applications/Google
  Chrome.app/…`, `$HOME/Applications/…`, Chromium). Then `$PATH` names `google-chrome`, `google-chrome-stable`,
  `chromium`, `chromium-browser`. Table-tested with fakes.
- [ ] `chrome.go`: the Launch/run loop as described in Core concepts.
  - Errors are wrapped and never retried silently.
  - `SetLive` sends a request to the loop. The loop updates Phase, publishes, then evaluates `__couchcastSet` in each
    tracked context. Evaluation errors on pages mid-navigation are ignored on purpose: the new document's script
    asks for state on load via a `{"t":"hello"}` report, and the loop answers it with the current state.
  - Include the `hello` round trip in inject.js and translate (amend Task 4's test).
- [ ] Unit-testable parts stay in Task 2's pure functions. Chrome's process/protocol behavior is covered by Task 9's
  live check, which is the conformance oracle for the fake.
- [ ] Commit `#454 M1: webcast: Chrome browser over the debugging pipe`.

### Task 9a: conformance suite + live check  (M1)

**Files:** Create `cmd/internal/webcast/conformance_test.go`, `live_test.go`.

- [ ] `runConformance(t, h harness)`, where `harness` has `browser() Browser`, `navigate(url)`, `newTab(url)`,
  `activate(id)`, `clickControl()` and `quitByHand()`. Scenario:
  1. Idle with one tab.
  2. SetLive(true) → Live.
  3. Navigate to a `data:` page with title "one" → Tab.Title "one".
  4. New tab "two" → active "two".
  5. Activate the first → "one".
  6. clickControl → Idle.
  7. clickControl → Live.
  8. quitByHand → Done, ErrClosedByOperator.
  9. Separately: Close() → Done, Err nil, process gone.
- [ ] Fake harness runs in plain `go test`.
- [ ] Live harness: gated by `WEBCAST_LIVE_CHROME=1`; Chrome is headful because visibility semantics are the point,
  with a temp profile dir.
  - Operator actions use unexported test hooks on `*Chrome` that send CDP over the same Conn: `Page.navigate`,
    `Target.createTarget`, `Target.activateTarget`, `Runtime.evaluate` of a click on the shadow control via the
    isolated world, and `Browser.close` for "by hand", flagged as not ours.
  - Plus `TestLiveChromeExitsWhenPipeCloses` (the funeral property) and `TestLiveProfileInUse`.
  - Run: `WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run Live -v` (unsandboxed).
- [ ] **M1 boundary:** update the atlas (new `atlas/webcast.md`, linked from `atlas/index.md`), then run
  `sdlc milestone-close --issue 454 --milestone M1`.

### Task 6: Extract broadcast.Endpoint  (M2)

**Files:** Create `cmd/internal/broadcast/endpoint.go`, `endpoint_test.go`; Modify `session.go` (Start, watch,
teardown), `server.go` (export `SetSecurityHeaders`).

- [ ] Endpoint tests with `FakeTunnel`:
  - the link answers the handler built from the minted token;
  - ctx cancelled during Open closes everything;
  - a tunnel exit shows on `Exited()`;
  - Close is idempotent and shuts the server down.
- [ ] Move the code, then run the whole broadcast package (`go test ./cmd/internal/broadcast/ ./cmd/internal/couchtty/`).
  Its tests must stay green unchanged.
- [ ] Commit `#454 M2: broadcast: extract Endpoint (tunnel + server + probe) for reuse`.

### Task 7: webcast Server, viewer page, Cast  (M2)

**Files:** Create `cmd/internal/webcast/{server.go,server_test.go,stream.go,stream_test.go,cast.go,cast_test.go,web/index.html,web/viewer.js,web.go}`;
`tests/webcast-viewer/viewer.test.mjs`.

- [ ] Server tests (httptest):
  - a wrong token gets 404;
  - security headers are set;
  - SSE sends `state` on join (redacted while idle), then on change, then `end`;
  - the 17th viewer gets 503;
  - POST gets 405.
- [ ] Viewer (node test): renders "Not live" / URL + title / "Cast ended"; never inserts URL or title as HTML
  (textContent only).
- [ ] Cast tests with FakeBrowser + FakeTunnel:
  - start: the Browser is up before the link is ready;
  - the link becomes ready;
  - ToggleLive forwards to the Browser;
  - Browser QuitByHand → Cast Done with the cause and viewers get `end`;
  - End → Browser closed and endpoint closed;
  - Endpoint fails to open → Cast stays up, with LinkErr set;
  - a tunnel exit mid-cast → LinkErr set, the browser keeps running.
- [ ] Implement, then commit `#454 M2: webcast: stub viewer stream and Cast`.

### Task 8: Couch key, indicator, controls, wiring  (M2)

**Files:**
- Modify: `couchkeys/couchkeys.go` (`ActionWebcast`, `Key: "Shift+Ctrl+Alt+b"`, Encodings `\x1b[98;8u`, `\x1b[66;8u`).
- Modify: `couchtty/keys.go` (`HitWebcast`), `console.go` (fields, hit table, shutdown hook), `reserve.go`
  (ChromeCell, span), `terminal.go` (`commitChrome` stores `statusChrome`), `terminal_input.go` (clicks).
- Create: `couchtty/console_webcast.go`, `console_webcast_test.go`, `reserve_webcast_test.go`,
  `couchcmd/webcast.go`, `couchcmd/webcast_test.go`.
- Modify: `couchcmd/run.go` (SetWebcast).

- [ ] Key tests:
  - `\x1b[98;8u` and `\x1b[66;8u` → `HitWebcast`;
  - `\x1b[98;7u` is still `HitBroadcast`;
  - a release form never fires.
- [ ] Render tests: idle and live labels; position after the broadcast cells (with and without a live screen
  broadcast); clipped at narrow widths → empty span.
- [ ] Console tests with `newFixtureBeforeRun` + a Cast config whose `Launch` returns a FakeBrowser and whose tunnel
  is FakeTunnel. Each asserts the **drawn cells** (lesson #447 M4) with a control:
  - the key starts Chrome → `Chrome ▶`;
  - the key again → `Chrome ⏸`;
  - a click on the cell → `▶`, and again → `⏸`;
  - a right-click copies the link (OSC 52 seen on the presenter output), or the not-ready notice before the link
    is ready;
  - Ctrl+left and Ctrl+right on the cell → FakeBrowser closed, the cell gone;
  - a plain click elsewhere on the row does not toggle (control);
  - `QuitByHand` → the cell gone plus the notice;
  - the key during Starting → notice, no second Launch (the Launch count is asserted);
  - a Launch that completes after Couch shut down is closed (startClaim);
  - Couch shutdown ends a running cast.
- [ ] `couchcmd/webcast.go`: `webcastSettings(getenv, storeDir, lookPath, stat)` builds the CastConfig: FindChrome,
  the profile dir `<storeDir>/webcast/chrome-profile`, and the same tunnel choice as `broadcastSettings`
  (reuse its tunnel builder; do not restate the env parsing). Chrome missing → configured-but-refusing: the key
  shows the notice "Chrome not found (set COUCH_WEBCAST_CHROME)".
- [ ] Commit `#454 M2: couch: Shift+Ctrl+Alt+B Chrome cast with tab-bar controls`.

### Task 9b: live end-to-end (M2)

- [ ] Extend the live test: a `Cast` with the real Chrome and the `LocalOnly` tunnel; an SSE client on the link sees
  `state{live:true,url,title}` after SetLive plus a navigation, and `{live:false}` without a URL after pause.

### Task 10: docs, build, smoke test  (M2)

- [ ] README: add the `Chrome ▶/⏸` cell to the tab-bar description (lesson #447: grep `LIVE ⏸ 👆 👽`), the key in
  the key table, `COUCH_WEBCAST_CHROME`, and the profile dir and how to remove it.
- [ ] `atlas/webcast.md`: components, seams, and the `data-couchcast-ui` note for the Recorder.
- [ ] Full verification per memory: `make -k test`, then `go test ./...`. Compare any failure with main.
- [ ] Move the branch to pair:0 (`sdlc move`), `make build` there, check HEAD; ask the operator to smoke-test
  (restart Couch from pair:0), naming the commit. Steps: Shift+Ctrl+Alt+B → Chrome opens; again → `Chrome ⏸`;
  right-click → link pasted in a phone/another browser shows Live + URL; navigate and switch tabs (including a new
  one); click → paused (viewer shows not-live, no URL); in-page button resumes; Ctrl+click closes Chrome; restart
  and quit Chrome by hand → the cell clears.
- [ ] `sdlc close --issue 454 --verified '…'`, then merge main, `sdlc pr`, `sdlc merge --yes`.

## Revisions

### 2026-10-11 — plan-quality round 1

**PQ-1 (Important): concurrent screen broadcast and web cast on a named tunnel.** `Cloudflared.Open` claims
`named-<Name>` exclusively (`records.go` → `ErrTunnelBusy`), so two casts that each open the configured tunnel
collide. Policy: **one shared, reference-counted host per Couch, mounting each cast under its own token.** Both casts
then run at once on one hostname, and the Recorder and later casts inherit it. Mutual exclusion would forbid
demoing a terminal and a web app together, and a separate webcast hostname is more operator setup. This replaces
the `broadcast.Endpoint` extraction (Task 6 and the Core concepts row) with:

- **`broadcast.Host`** (`cmd/internal/broadcast/host.go`, new):
  - **API:** `NewHost(HostConfig{Tunnel, ProbeTimeout, Resolve})` and
    `Mount(ctx, token string, h http.Handler) (*Mount, error)`.
  - **Mount:** `Link()` (`base/token/`), `Base()`, `Mode()`, `Exited() <-chan error` (the shared tunnel or server
    died; every mount sees it), and `Unmount()`.
  - **Mux:** the Host routes `/<segment>/…` to the handler mounted under that segment (constant-time compare, per
    mount). An unknown segment gets 404 with the same security headers.
  - **Ordering:** one mutex-guarded tagged state, `hostClosed | hostOpening{done chan} | hostOpen | hostClosing{done
    chan}`. A Mount while opening waits on the same open. The probe runs once per open, against the first mount's
    link; later mounts are reachable as soon as they're in the mux. The last Unmount moves to closing and tears down.
    A Mount while closing waits for the close, then reopens. An open that fails reports to every waiter and returns
    to closed. A cancelled waiter leaves the open running for the others; if no mount remains when the open
    completes, it closes.
  - **Tests** use `FakeTunnel` (`OpenDelay`, `CloseBlock`, stats) with a seam to order the interleavings (FakeTunnel's
    delays, plus a gate channel the test releases). Each asserts against the stated invariant: *Opens − Closes ≤ 1
    at all times, and a mounted handler is reachable iff its Mount is live.* Cases:
    - two mounts share one Open;
    - Mount during Opening;
    - Mount during Closing reopens;
    - the last Unmount closes;
    - a failed open reaches all waiters;
    - a cancelled waiter;
    - a tunnel exit reaches all mounts.
    Plus a collision regression with a named-style claim (a FakeTunnel mode that refuses a second concurrent Open with
    `ErrTunnelBusy`, mirroring `records.go`): a screen broadcast and a web cast mounted together both answer.
- **`broadcast.Session`** takes a `*Host` (`Config.Host`). When it is nil, `Start` makes a private Host from
  `Config.Tunnel`, so the existing session tests keep their shape. The session mounts its hub server and its
  pointer-token routes under their tokens. The pointer token is a second Mount on the same Host, made by
  `EnablePointer` and removed by `DisablePointer`.
- **Couch:** `couchcmd` builds one Host from the tunnel choice and hands it to both `broadcast.Config` and the webcast
  `CastConfig`. Task 8's "reuse its tunnel builder" becomes an explicit extraction:
  `broadcastTunnel(getenv, store, guard) (broadcast.Tunnel, error)` is split out of `broadcastSettings` (Minor 2),
  and `webcastSettings` takes the built Host instead of re-reading env.
- `Cast` takes a `*broadcast.Host` and Mounts on start, so `CastConfig.Endpoint` becomes `CastConfig.Host`.

**Minors:**
- **Renames** (naming collision): `ChromeCell` → `WebcastCell` (`WebcastOff|WebcastIdle|WebcastLive`),
  `RenderedStatusRow.Chrome` → `.Webcast`, and the console field `statusWebcast`. "chrome" already means status-row UI
  chrome (`commitChrome`).
- **Test strategy** (in place of enumerated prose, which stays as the case list):
  - `splitFrames` gets a fuzz test seeded with spike captures and truncated or oversized frames. Invariant: it never
    panics, and re-encoding the decoded messages reproduces the input prefix.
  - Tracker gets a generated-sequence property test against two invariants: the active tab is always a live page, and
    when one exists it is a visible or never-reported one. Hand cases are kept for the named scenarios.
- **Active-tab edge:** an uninjectable page (`chrome://…`) opened in the background counts as active until something
  else reports, because it can never report "hidden". Accepted for MVP and stated in atlas: Chrome opens `chrome://`
  pages in the background only on unusual paths (Cmd+T/NTP opens in the foreground). The Recorder task revisits it if
  observed.
- **Chrome stderr:** `LaunchOptions.Stderr` is a bounded in-memory ring (last 8 KiB). Its contents are quoted in a
  `Launch` error only; nothing is written to disk.
- **Live conformance cadence:** `WEBCAST_LIVE_CHROME=1` runs at each milestone close of an issue that touches
  `cmd/internal/webcast` or `cmd/internal/cdp`, and after a Chrome major version change is noticed. This is stated in
  atlas next to `BROADCAST_LIVE_CLOUDFLARED`.

**Non-goals (this issue):**
- recording anything: no DOM, no files (the Recorder task);
- a CDP accessor on `Chrome` for the Recorder;
- protecting the injected control from a page that forges a click (it can only toggle live/pause, which is visible
  in the tab bar);
- casting `file://` pages;
- more than one cast Chrome;
- remote driving;
- privacy masking.

### 2026-10-11 — M1 as built (live findings and the M1 review)

Divergences from the tasks above, all measured against Chrome 154:
- **Attach:** browser-level `Target.setAutoAttach{waitForDebuggerOnStart, flatten, filter: page}` replaces the
  per-target `attachToTarget`. It covers the already-open page too, and holds each new page until `goSetup` releases
  it with `Runtime.runIfWaitingForDebugger`, always, even when setup failed. Setup after attach had raced the first
  navigation. `Launch` returns once the first page's control is set up, not when the page appears.
- **Titles:** `Target.targetInfoChanged` brings a new `<title>` only much later, so the script reports
  `document.title` and the Tracker gains `PageTitle`. A main-frame `Page.frameNavigated` (`DocumentChanged`) clears
  it, so an uninjectable new document doesn't keep the old title (BR-5).
- **Setup steps:** `Page.enable` is required; without it the script never reaches the next document.
- **Naming:** `publisher[T]` (`publish.go`) is what Task 3 called `latest[T]`.
- **LaunchOptions:** gained `StartURL` and `Headless` (tests only); the planned `Stderr` became an internal 8 KiB
  tail.
- **Process seam:** the process sits behind a small `process` interface, so the run loop is tested against a
  scripted CDP peer and a fake process (`chrome_loop_test.go`, BR-3). The tests cover per-page setup and release,
  the page toggle, last-window grace and its cancel, end-cause precedence, kill after `closeWait`, SetLive while
  ending, and exit reasons. Each was mutation-checked.
- **End reason:** `endError` (pure) decides the reason: a cause the loop chose, then a failed connection (BR-2), then
  a failing status, then the operator. `SetLive` while ending returns `ErrEnded`. `cmd.WaitDelay` bounds `Wait` when
  helpers hold the output pipe (BR-4).
- **Profile in use** is read from Chrome's output (`SingletonLock: File exists`, or the windowed handoff message),
  not from exit status 21, which other lock-setup failures share (BR-6).
- **Live harness:** headless, and never raises a window (operator instruction: don't take focus from Couch). Tab
  switching is emulated by focus; real switching under windows moves to the smoke test. The node tests live in
  `tests/webcast/`.
- **M1 review round 2:**
  - **Drain (BR-11):** the cdp reader drains raw bytes after every failure that leaves the pipe open, so Chrome never
    blocks writing and exits cleanly when its command pipe closes. Each `Conn` doc-comment promise is pinned
    (`TestReaderDrainsAfterProtocolFailure`, `TestErrClasses`).
  - **End reason (BR-12):** `cdp.ErrProtocol` classes what Chrome *sent* (oversized, malformed, backlog), and
    `endError` counts only that class as "lost Chrome". The pipe closing (EOF, a closed reader, a write after exit)
    never is, since our own teardown produces it too. The scripted peer gains a mode where the reaper closes the
    reader first.
  - **Reaping (BR-4):** `startProcess` sets `WaitDelay` and is tested with a shell whose background child keeps the
    output pipe open.

### 2026-10-11 — M2 as built

- **Host:** `broadcast.Host` routes by `Mountable.Owns(segment)` rather than one mount per token, since the screen
  broadcast's server already answers two tokens (view and pointer); one mount per cast.
- **Host fallback:** an unknown token answers like a mount does: 405 for any non-GET, 404 for GET. The existing
  route-table test pins it.
- **Cancel:** the last waiter to cancel an open waits for it, and for the close that follows, before returning.
  This keeps `broadcast.Start`'s contract that a cancelled start closes what it opened. A late-succeeding open is
  closed by the single "nobody wants it" rule.
- **No `broadcastTunnel` extraction (Task 8):** `run.go` builds one Host from the tunnel `broadcastSettings`
  already parsed, so the env is parsed in one place.
- **Shared helpers:** SSE writing moved to `broadcast.EventStream` (both servers use it), and the token minter is
  exported as `broadcast.NewToken`.
- **Live end-to-end** (`TestLiveCastEndToEnd`): a real headless Chrome, Cast, the Host on `LocalOnly`, and an SSE
  viewer. It found Chrome's title placeholder during loads, which is accepted and documented in atlas; the old
  title is never paired with the new URL.
- **M2 review (BR-14):** every loss of interest in the Host now goes through `releaseIfUnwantedLocked`: an open
  completing, a waiter abandoning (`abandonWait`), or an Unmount. The cancel/complete race could leak a tunnel 1 in
  300 times. `OwnsToken` is exported and used by both casts (a constant-time compare).
