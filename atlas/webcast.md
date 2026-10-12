# Couch broadcast-web: the cast Chrome (#454)

`couch broadcast-web` casts a Chrome tab from Couch as a DOM stream (project: ops
`teams/devinfra/projects/couch-broadcast-web/project.md`). This file maps the browser-control component
and the recorder (#458); the asset proxy and DOM viewer plug into the same seam. The recording format, its reducer and the wave-1 seams are in
[webcast-format.md](webcast-format.md), which also names where the recorder plugs in here.

## Layers

| Layer | Package | Role |
|---|---|---|
| CDP client | `cmd/internal/cdp` | Chrome DevTools Protocol over `--remote-debugging-pipe` (fd 3 in, fd 4 out, NUL-framed JSON). The only package with CDP wire types. Bounded event queue (4096; `ErrBacklog` past it). |
| Browser seam | `cmd/internal/webcast` `Browser` | State (`Phase` idle/live/ended, active `Tab`), `Subscribe` (latest value, never blocks), `SetLive`, `Close`, `Done`/`Err`. Nothing CDP crosses it. |
| Real Chrome | `webcast.Chrome` (`chrome.go`, `inject.js`) | Launches Chrome on the cast profile, auto-attaches to every page, injects the control, owns all state on one loop goroutine. |
| Fake | `webcast.FakeBrowser` | Stateful double with operator verbs (navigate, open/switch/close tabs, click the control, quit). Drives the same `Tracker`. Records nothing. |
| Recorder | `cmd/internal/webcast/recorder` | Part of the real Browser, on the CDP side of the seam (#458). See [The recorder](#the-recorder-458). |

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

## The recorder (#458)

`Chrome` starts it when `LaunchOptions.Record` is set, forwards **every** CDP event to it (`Send` never blocks the
loop), and tells it the live phase from `publish` (`Live` on the active tab, a switch when the tab changes, `Title`,
`Pause`). It is the record stream's **single writer**; #467's injector adds its inputs to the same queue.

- **Pure core:** `mirror` (`mirror.go`, `observe.go`) holds CDP's DOM per session (`getDocument` pierce, whitespace
  included) and the id map (`backendNodeId` → id per session for one document, keyed by loader id, with a 50 000-node
  graveyard so moves keep ids). Every record goes through its own reducer first, so the output validates by
  construction; refusals are counted (`Stats`). It leaves out the control (`data-couchcast-ui`), user-agent shadow
  trees, and password values. `IDOf`/`NodeOf` expose the map (#467).
- **Shell:** `Recorder` (`recorder.go`): one goroutine, a FIFO of stamped inputs (record `t` = arrival), synchronous
  CDP calls in queue order. CDP events carry `cdp.Event.Seq`; a snapshot drops queued events of its session read
  before its reply (`Conn.CallSeq`). Over 100 000 queued events it drops and resyncs. Side workers (`side.go`) fetch
  asset bodies (`Network.getResponseBody`; already-loaded ones via `Page.getResourceContent`) into the
  `assets.Store`, and take a screenshot plus a `DOMSnapshot` per session every 5 s and at each segment.
- **Observer:** `observer.js` runs in the `couchcast` world of **every frame** (installed with the control; frames
  get it in `goSetupFrame`). It names nodes by path, reports clicks synchronously, coalesces the rest per animation
  frame (scroll is event-driven), scans for values set by script, and reports only while switched on.
- **Frames:** pages auto-attach their out-of-process iframes (`frameAutoAttach`), which are set up and released like
  pages. A frame's document is a `doc` with `host`; click and pointer positions are mapped through the owner
  iframe's box.
- **Where recordings go:** `CastConfig.RecordingsDir` (`<couch store>/webcast/recordings/<id>/`), opened at the first
  go-live, teed into `CastConfig.NewLive`'s hub, whose records `Server` streams as `rec` events after the `state`
  ones. Kept until the operator deletes the directory.
- **Live check:** `TestLiveRecording` (`live_record_test.go`) casts a page with re-renders, open and closed roots, an
  out-of-process iframe, insertRule, a >10k `<style>`, a password, a click that navigates, pause and resume; the
  recording must validate, drop nothing, and hold each case.

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
- **Cast:** `webcast.Cast` = Browser + the stub viewer `Server` mounted on the shared `broadcast.Host`. Chrome comes
  up first; the link opens in the background (`Update.Link` / `LinkErr`). The cast ends when its Browser does;
  viewers get `end`, then the mount goes.
- **Shared Host:** `broadcast.Host` (`host.go`) holds one listener, server and tunnel for every cast, opened by the
  first mount and closed after the last, routing by the token in the first path segment (`Mountable.Owns`). A named
  cloudflared tunnel can be claimed once, so the screen broadcast and the web cast must share it. `couchcmd` builds
  one Host from the `COUCH_BROADCAST_*` tunnel and hands it to both. SSE writing is `broadcast.EventStream`, shared.
- **Viewer (stub):** `webcast/web/` shows Live / Not live, and the title and URL only while live (`viewerState`),
  set as text only. The recorder's DOM stream will replace the state events' content; the route and seams stay.
- **Known transient:** for a moment after a navigation, the title can be Chrome's placeholder (the URL, as in
  Chrome's own tab) until the page reports its `<title>`; it is never the previous page's title (`DocumentChanged`).
