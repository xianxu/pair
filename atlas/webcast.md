# Couch broadcast-web: the cast Chrome (#454)

`couch broadcast-web` casts a Chrome tab from Couch as a DOM stream (project: ops
`teams/devinfra/projects/couch-broadcast-web/project.md`). This file maps the browser-control component
and the recorder (#458); the asset proxy plugs into the same seam, and the live viewer (#460) is mapped below. The recording format, its reducer and the wave-1 seams are in
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
`Pause`). It is the record stream's **single writer**: records made elsewhere (#467's control and injected actions, #470's notes
and strokes) enter its queue as `recorder.Emit`, stamped on arrival and checked by its reducer.

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
- **Frames:** a screenshot and layout at each segment start (go-live, resume, navigate, tab) and every 5 s while live
  (`Options.ShotEvery`), never streamed. Screenshots are the viewport at half its CSS size whatever the DPR
  (`recorder.LowRes`), the form #453's fixtures use. The fidelity check (#478, below) compares them with the replay.
  A frame's `t` is its screenshot call's midpoint, but never before its segment: a call that straddles a navigation
  is stamped at the new segment's start, though its pixels may show either page. The layout boxes captured after it
  take the stream's time, which records may have moved past meanwhile (#472).
- **Asset capture** (the recorder's; #459 stores, absolutizes CSS and serves): raw bodies to `assets.Store.Put`, one
  fetch per URL per recording, by the first source to claim it. A Network response claims its URL; so does a resource
  Chrome holds at a segment's start (`Page.getResourceContent`, else a load in the page's context with
  `Network.loadNetworkResource`, never from outside the browser). An element's `src` or a FontFace font's `src` waits
  2 s so responses claim first, then is loaded the same way if nothing did. Images whose source the browser chooses
  (a srcset, a `<picture>`) and lazy images are never loaded from references.
- **Live check:** `TestLiveRecording` (`live_record_test.go`) casts a page with re-renders, open and closed roots, an
  out-of-process iframe, insertRule, a >10k `<style>`, a password, a click that navigates, pause and resume; the
  recording must validate, drop nothing, and hold each case.

## The fidelity check (#478)

Project Principle 6: fidelity is measured, not assumed. Each recording's frames (the real tab) are compared with
what the viewer's renderer draws at the same moments.

- **Pure core:** `webcast/fidelity`. `Diff` is #453's pixelmatch-style YIQ diff (the share of pixels that differ,
  leaving out the cast control's corner). `Summarize` gives median, max and the frames over `Tolerance` (3% of a
  frame's pixels). `DiffBox` and `SideBySide` make each worst moment's image: the tab, the replay, and where they
  differ in red, cropped to the difference. `Write` puts the report in `<recording>/fidelity/`
  (format: [webcast-format.md](webcast-format.md#the-fidelity-report-478)).
- **Shell:** `webcast.CheckFidelity(ctx, chromeBin, dir)`. It serves the recording with a `ReplayServer` on loopback
  and opens `r/<id>/fidelity.html` in a headless Chrome on a throwaway profile. For each frame, in time order, it
  sets the frame's viewport at DPR 1, calls `fidelity.replayAt(t, ReplayWindow)`, screenshots the viewport,
  scales it with `recorder.LowRes` (the tab's own filter) and diffs it.
  - A frame over the tolerance is replayed again at t − `fidelity.CaptureSkew` (50 ms), the best kept. A frame's
    pixels can be from earlier than its t (#472). It is never replayed later than t: #458's recorder reports
    synchronously, so a later match would hide a recorder that is late (pair#475). `recording.FrameTimes` keeps each
    moment in the frame's own segment, the same rule #460's conformance test uses.
  - The report shows the offset and the diff at t, so a timing slip is told apart from a wrong render.
  - Only the 10 worst replays are held in memory.
- **The page:** `web/fidelity.html` + `fidelity.js`, the frame alone at (0,0), unscaled at the operator's viewport.
  `web/replayer.js`'s `Replayer` is the replay. #460's conformance harness uses it too, so there is one
  `replayAt`.
  - Records up to t − 1 s are drawn at once, then the last second in real time, so a transition is in the tab's
    phase.
  - A moment shortly after the last one continues it: the animations paused at the last capture resume, so a
    retry or a close frame is at its phase. A moment before it seeks through the catch-up (`catchUp`, the replay's
    seek rule), so it replays at most one segment.
  - A segment that isn't a navigation snapshots a document that was already running, so its animations are
    finished, not started: on the live blog cast this took the first frame from 2.15% to 0.10%.
- **When it runs:**
  - `CastConfig.Check` runs once a recorded cast with frames has ended, in the background. couchcmd wires
    `CheckFidelity` with one check at a time per Couch, and the summary line goes to the operator's notice.
  - `couch --broadcast-web-fidelity [ID|DIR]` checks a recording (default: the store's newest). It prints the line
    and exits 0 within the tolerance, 3 over it, 1 on an error.
- **Measured** (`Tolerance`'s comment): the fixtures at most 0.74%; a live blog cast by #458's recorder at most 0.10%
  over 17 frames; the blog without its stylesheets 81% median, every frame over.
- **Tests:**
  - `fidelity/fidelity_test.go` and `fidelity_check_test.go`: pure.
  - `record_cast_test.go`: the hook, with a fake check.
  - `couchcmd/fidelity_test.go`: the CLI, with a fake check.
  - `tests/webcast/replayer.test.mjs`: the replay's timing, with a fake clock.
  - `fidelity_live_test.go` (`WEBCAST_LIVE_CHROME=1`): every fixture within the tolerance, and a broken render
    (no stylesheets) caught. `FIDELITY_DIR=` checks any recording in place, e.g. one from `TestLiveRecordURL`.

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
  and the cast is to bind the recording's store to it when the recording opens (first go-live); until then it
  answers 404. That bind, and couchcmd's `NewAssets`, land with #458's recorder (whichever of #458/#459 lands
  second wires them); until then no production cast binds a store. `Bind` takes an `assets.Served` (a store that knows each body's MIME, as `Recorded` does), so the wiring keeps
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
    time so transitions are in phase. The prototype's fixtures' polled observer gets 150 ms of slack; the recorder's
    get none (#472). Any fixture's frame may also be compared 50 ms before its `t`: its pixels are a compositor frame from
    inside the screenshot call. Offsets never leave the frame's own live segment (`recording.FrameTimes`, shared with the
    fidelity tool). `knownGaps` lists frames that differ for a filed reason (#475), and fails once one passes.
  - The harness sizes the drawn page to the segment's viewport, as `viewer.js` does, so a tab switch to a
    different-sized tab compares at that size.
  - `viewer_live_test.go` drives the real page through `Server` and a `Hub` at tablet width: scaling, URL, hover,
    pointer, and `boxOf` against the tab's own layout boxes (215 of 217 on blog).
- **Known transient:** for a moment after a navigation, the title can be Chrome's placeholder (the URL, as in
  Chrome's own tab) until the page reports its `<title>`; it is never the previous page's title (`DocumentChanged`).

## Replay (#471)

A recording plays in the same viewer, with the same model and renderer. Only its source differs: records come from disk
instead of the event stream.

- **Source:** `web/replay.js`. `mount` branches once, on the path (`<token>/r/<id>/`): `mountReplay` instead of
  `connect`. Both return the surface `{model, renderer, onRecord}`, so layout, scaling, the pointer and
  `window.couchViewer` are unchanged.
- **Seek rule:** the state at t is the catch-up of the records up to t, reduced. `catchUp` is a port of
  `recording.CatchUp`, which keeps only the latest `hover` and `ptr` of the latest segment.
  - **Shared corpus:** `tests/webcast/catchup_corpus.json` reaches every branch of the rule and holds Go's answer at
    every cut. `TestCatchUpCorpus` keeps the answers true to Go and requires every special-cased kind; regenerate them
    with `-update-catchup`. The node test holds the port to the same answers, so a change to Go's rule fails at once
    until the port follows.
  - **Invariant:** node tests also check the format's catch-up invariant at every boundary cut of every fixture: the
    same DOM, assets, `DocSeg` and current-document notes as the whole prefix.
  - **Live:** the live check compares the port with Go's, record for record, on the fixtures. A forward seek within a stretch applies the records in
  between. A seek backward or across a `seg` rebuilds from the catch-up, so it replays at most one segment.
- **Player:** `Player` is the replay's state (`t`, playing, speed 0.5–4×, an error state for a refused record). It has
  no clock: the shell ticks it on animation frames. Playback skips pauses, and it stops at the end.
- **The bar** (`#timeline`, `replay.css`) shows play/pause, speed and the track:
  - live segments, with a seam where one starts;
  - pauses hatched, each squeezed to at most 3% of the width (`placer`), so an hour's pause doesn't flatten the rest;
  - who drove, as bands from the `ctl` records' control state;
  - notes and strokes, as ticks.
  Click or drag seeks, and so do ←/→, Home/End and Space.
- **`#t=` links:** `#t=<ms>` opens at that moment, paused. A seek or a pause rewrites the hash
  (`history.replaceState`), and `hashchange` seeks. `webcast.ReplayLink(base, id, t)` builds a link.
- **Marks over the page (#477):** `replayMarks` draws with the live viewer's own modules (#470's `annotate.js`), in an
  `.annotate` layer over `#box`, like the live viewer's:
  - **notes:** `liveNotes(model)` drawn by `pinLayer`. Each pin is at its region, else its element (`boxOf`), else the
    box it carried. A tap shows the note read-only, in a card under its pin (`.replay-note-card`, text
    only). The tap places it at once, and a resize redraws the marks, because a paused replay draws no frame.
  - **strokes:** `strokesAt(model, t)`, the points drawn by t with chunks joined by id, fading over `FADE_MS` after
    the last point. They're painted on a canvas by `paintStrokes`.

  The replay has no `palette.json`, so `mountAnnotate`'s posting UI never mounts there. Every server of the viewer
  serves its files from one table (`serveViewerFile`), the fidelity harness included: `replay.js` now imports
  `annotate.js`, which imports `collab.js`.
- **Server:** `webcast.ReplayServer(token, root)`, a `broadcast.Mountable` with the viewer's CSP. It serves:
  - `<token>/`: the listing, newest first, from each recording's head line and last line;
  - `r/<id>/`: the viewer;
  - `r/<id>/records`: the merged records, read per request, so a recording still being written replays as far as it
    has got;
  - `r/<id>/assets/<hash>`: the cast's `assets.Proxy`, bound to `assets.Reopen(rec)`, a read-only `Recorded` built
    from the asset records.

  An id is one path element (`ValidRecordingID`) naming a real directory directly under the root, never a symlink.
  `NewRecordingServer(token, root, id)` is scoped to one recording: no listing, every other id not found.
  The listing's length is `unknown` when the events file's last line is torn (its `{"t":N,` envelope doesn't parse).
- **Replay shares (#477):** `webcast.ReplayShares`, made in `couchcmd/run.go` beside the shared `broadcast.Host` and
  held by the console, mounts each recording asked for on the Host, under its own token, as a `NewRecordingServer`.
  A link grants the one recording it names (the cast token model).
  - **States:** `opening → ready | failed`, `ready → lost` when the tunnel dies, and `failed | lost → opening` with a
    new token on the next `Share`. `Close` (console teardown) abandons opens, unmounts every share, and so closes the
    tunnel.
  - **Lifecycle:** shares live in memory until Couch exits, one per recording asked for. The tunnel stays open while any
    share is mounted. There's no expiry or unshare verb yet.
  - **The operator sees each one:** a notice when a share is ready (`OnReady`; it never carries the link), and
    `couch --broadcast-list` lists the shared recordings (`Shared`, in `broadcast-status`'s `Replays`). An agent's
    `query link` can't open the tunnel unseen.
  - **The broker ops** are `replay-share` (start it) and `replay-status` (read it). They need no slot identity, like
    `broadcast-status`, because the socket is in the operator-only store. They never wait on the tunnel: the socket gives
    each exchange 2 s, and an open takes up to the Host's 30 s probe. So the CLI asks, then polls every 300 ms, up to
    45 s.
- **Entry point:** `couch --broadcast-web-replay [ID|DIR] [--at MS] [--local]` (`couchcmd/replay.go`).
  - **A recording in Couch's store, with a Couch running:** it asks for the share (`shareReplay`), prints the public
    link and exits.
  - **No Couch, an older Couch, a DIR outside the store, no ID (the listing), or `--local`:** it serves on `127.0.0.1`
    under a fresh token until SIGINT or SIGTERM, and says so on stderr. It writes nothing (ARCH-FUNERAL: the listener
    and token die with the process).
  - **For agents:** `couch --broadcast-web-query DIR link [--at MS]` prints the shared link through the same
    `shareReplay`. The query package takes it as a `query.Linker`. With no Couch it refuses and names the replay CLI.
    It never serves itself.
- **Tests:**
  - `tests/webcast/replay.test.mjs`: the seek rule, timeline, placer, `#t=`, `Player` and `replaySource`, with fakes.
  - `replay_server_test.go`: the routes and confinement to the root.
  - `couchcmd/replay_test.go`: the CLI.
  - `shares_test.go`: the share states on a `FakeTunnel` Host.
  - `couchcmd/replay_share_test.go`: share-and-poll against a scripted broker, the loopback fallbacks, `query link`,
    and the whole chain: CLI → broker handler → console → shares → Host.
  - `replay_live_test.go` (`WEBCAST_LIVE_CHROME=1`): a `#t=` link and shuffled scrubbing draw the reducer's canonical
    state at every pin of every fixture; JS `catchUp` takes the records Go's does; notes and strokes match the reducer
    at their moments; drive bands; nothing is requested outside the server.
