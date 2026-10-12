# Couch broadcast-web: the cast Chrome (#454)

`couch broadcast-web` casts a Chrome tab from Couch as a DOM stream (project: ops
`teams/devinfra/projects/couch-broadcast-web/project.md`). This file maps the browser-control component;
the recorder and asset proxy plug into the same seam, and the live viewer (#460) is mapped below. The recording format, its reducer and the wave-1 seams are in
[webcast-format.md](webcast-format.md), which also names where the recorder plugs in here.

## Layers

| Layer | Package | Role |
|---|---|---|
| CDP client | `cmd/internal/cdp` | Chrome DevTools Protocol over `--remote-debugging-pipe` (fd 3 in, fd 4 out, NUL-framed JSON). The only package with CDP wire types. Bounded event queue (4096; `ErrBacklog` past it). |
| Browser seam | `cmd/internal/webcast` `Browser` | State (`Phase` idle/live/ended, active `Tab`), `Subscribe` (latest value, never blocks), `SetLive`, `Close`, `Done`/`Err`. Nothing CDP crosses it. |
| Real Chrome | `webcast.Chrome` (`chrome.go`, `inject.js`) | Launches Chrome on the cast profile, auto-attaches to every page, injects the control, owns all state on one loop goroutine. |
| Fake | `webcast.FakeBrowser` | Stateful double with operator verbs (navigate, open/switch/close tabs, click the control, quit). Drives the same `Tracker`. |

## How the active tab is found

CDP has no "tab activated" event. `Tracker` (pure, `tracker.go`) infers it from target lifecycle plus what each page's
injected script reports (visibility, focus), ordered by arrival sequence: the most recently activated page that is
visible or has never reported; with every page hidden (minimized), the last active one. A page that can't run the
script (`chrome://newtab`) counts as active from its creation, so a `chrome://` page opened in the background is
wrongly taken as active until another page reports — accepted for MVP.

Titles come from the page, not Chrome: `Target.targetInfoChanged` reports a new URL at once but a new `<title>` only
much later (measured, Chrome 154), so the script reports `document.title` and it wins when non-empty.

## The in-page control

`inject.js` runs in the `couchcast` isolated world of every top-level document: it shares the page's DOM, not its
JavaScript, so the page can't see or call the `__couchcast` binding. It draws Go live / Pause in a closed shadow root
on a host marked **`data-couchcast-ui`** — the recorder must leave that element out of the cast. A page can still
click the control (toggling live/pause, which Couch's tab bar shows); accepted for MVP.

Setup per page (`goSetup`): auto-attach holds each new page before its first document runs; the binding, Runtime
and Page domains, and the script (`runImmediately`) are set up, then the page is released — always, since a page
left waiting is a frozen tab. Without `Page.enable` the script never reaches the next document.

## Lifecycle

- **Chrome process:** owned by Couch through the pipe. `Close` → `Browser.close`, killed after 5s. Closing Couch's
  end of the pipe makes Chrome exit by itself, so a crashed Couch leaves no cast Chrome (`TestLiveChromeExitsWhenPipeCloses`).
  The last window closed by hand (macOS keeps Chrome running windowless) ends the cast after 1s with no pages.
- **Profile:** `<couch store>/webcast/chrome-profile`, kept across casts so logins survive; remove it by deleting
  the directory. A second Chrome on it fails with `ErrProfileInUse` (handoff message windowed, exit 21 headless).
- **Chrome's output:** last 8 KiB in memory, quoted only in a launch error.

## Tests

- Pure: `tracker_test.go` (named cases + generated sequences against stated invariants), `translate_test.go`
  (events captured from Chrome 154 in `testdata/`), `publish_test.go`; `tests/webcast/*.test.mjs` for `inject.js`.
- **Conformance:** `runConformance` runs one operator scenario against `FakeBrowser` (always) and real Chrome
  (`WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run Live -v`, outside the agent sandbox — Chrome exits 21
  inside it). Run it at each milestone close that touches `webcast` or `cdp`, and when Chrome's major version
  changes. Live runs are headless and never raise a window (the operator types in Couch meanwhile); real tab
  switching under windows is checked by the smoke test.

## In Couch (M2)

- **Key and cell:** `Shift+Ctrl+Alt+b` (`\x1b[98;8u`, or `66;8u` from a terminal reporting the shifted code;
  `couchkeys.ActionWebcast` → `HitWebcast`) starts the cast Chrome, then toggles live. The tab bar's
  `WebcastCell` draws `Chrome ▶` / `Chrome ⏸` **after** the screen broadcast's `LIVE ⏸ 👆 👽 💭N`, so the
  broadcast's fixed-column indicator check never moves. Click toggles, right-click copies the link, Ctrl with
  either button ends (macOS terminals may send Ctrl+left as a right click with the Ctrl bit).
- **State machine:** `couchtty/console_webcast.go`, mirroring `console_broadcast.go` (off → starting → running →
  ending, `startClaim` adoption, a bounded shutdown beside the broadcast's).
- **Cast:** `webcast.Cast` = Browser + the viewer `Server` mounted on the shared `broadcast.Host`. Chrome comes
  up first; the link opens in the background (`Update.Link` / `LinkErr`). The cast ends when its Browser does;
  viewers get `end`, then the mount goes.
- **Shared Host:** `broadcast.Host` (`host.go`) holds one listener, server and tunnel for every cast, opened by the
  first mount and closed after the last, routing by the token in the first path segment (`Mountable.Owns`). A named
  cloudflared tunnel can be claimed once, so the screen broadcast and the web cast must share it. `couchcmd` builds
  one Host from the `COUCH_BROADCAST_*` tunnel and hands it to both. SSE writing is `broadcast.EventStream`, shared.
- **Events route:** `<token>/events` streams `state` and `end` (the Browser's phase) and, when `CastConfig.Live` is
  set, `rec`: each record line of the recording, a `recording.Hub`'s catch-up first. When the record channel
  closes, the fan-out says why (`Live.Ended`). A viewer dropped as too slow has its stream ended and reconnects for
  a fresh catch-up. Once the fan-out has ended, the viewer keeps its page and the stream goes on with `state` and
  `end`. `?viewer=ID` calls `CastConfig.Presence` for the stream's life (#466's seats). `<token>/assets/<hash>` is
  the asset proxy's (#459, below).
- **Asset store and proxy (#459):** `webcast/assets/`. The recorder (#458) captures bodies over CDP and calls
  `Store.Put`; `assets.Recorded` is the real store: dedup by content hash (a repeated body is processed once),
  images resampled to `Policy.MaxPixels` (`image.go`, WebP through `golang.org/x/image`), CSS URLs made absolute
  (`css.go`'s `RewriteCSS`), bodies written through the recording's `DirWriter` and read back from disk.
  Image decodes are admitted one at a time process-wide (about 256 MB at the 64 MP ceiling), and a body that arrives
  twice is decoded once. `assets.Proxy` serves `<token>/assets/<hash>`: the `Server` owns one (`Server.Assets()`),
  and the cast binds the recording's store to it when the recording opens (first go-live); until then it answers
  404. `Bind` takes an `assets.Served` (a store that knows each body's MIME, as `Recorded` does), so the wiring keeps
  the concrete store, not the `Store` interface. It serves only
  images, stylesheets and fonts (a body a page labelled HTML or script is never served from the viewer's origin),
  rewrites a stylesheet's URLs to `<hash>` at serve time (not cached; bindings change), and marks other bodies
  immutable. The live viewer tests play the fixtures' assets through `Recorded` + `Proxy` (`throughStore`) and
  check the viewer requests nothing outside the cast.
- **Live viewer (#460):** `webcast/web/`, three ES modules plus the shell.
  - `model.js` is the reducer in JS: `recording.Reduce` and `State.Canonical()`, byte for byte. Node tests hold it to
    every fixture's pinned hash.
  - `render.js` draws what the model accepted into the page frame, an iframe with `sandbox="allow-same-origin"`
    (no `allow-scripts`). It never sets `on*` attributes, gives nested frames no `src`, and sends every URL through
    `resolveAsset`; a URL with no asset loads nothing. An image with nothing to load is drawn as not yet loaded (no
    alt text), as a lazy image is in the tab.
  - `css.js` rewrites streamed CSS (`url()`s through the asset table, `:hover` to the stand-in class).
  - `viewer.js` is the shell. It lays the page out once at the operator's viewport and scales it to fit (never
    relayout), shows the title and URL as text, and shows Paused and Cast-ended states. The operator's `hover` is
    drawn by putting the stand-in class on the hovered element and its ancestors, and the sampled `ptr` is a dot
    over the frame. A record the model refuses resyncs: the shell reconnects for a fresh catch-up.
  - The frame takes no input; the shell's overlay does (wave 2's notes, #466/#467's remote driving).
  - The contract is on `window.couchViewer`: `canonical`, `ready`, `resolveAsset`, `boxOf`, `nodeAt`, `toPage`
    and `onRecord`.
- **Viewer fidelity:** `replay_test.go` replays every fixture in headless Chrome (`WEBCAST_LIVE_CHROME=1`).
  - The drawn page's canonical equals the reducer's at every pin, and a join at any pin draws the same page.
  - Renders are within 3% of `frames/` (measured 0.2–2.3%). The last second before each frame is replayed in real
    time so transitions are in phase, and the fixtures' polled observer gets 150 ms of slack.
  - `viewer_live_test.go` drives the real page through `Server` and a `Hub` at tablet width: scaling, URL, hover,
    pointer, and `boxOf` against the tab's own layout boxes (215 of 217 on blog).
- **Known transient:** for a moment after a navigation, the title can be Chrome's placeholder (the URL, as in
  Chrome's own tab) until the page reports its `<title>`; it is never the previous page's title (`DocumentChanged`).
