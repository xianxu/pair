# Couch broadcast-web: the cast Chrome (#454)

`couch broadcast-web` casts a Chrome tab from Couch as a DOM stream (project: ops
`teams/devinfra/projects/couch-broadcast-web/project.md`). This file maps the browser-control component;
the recorder, asset proxy and DOM viewer come later and plug into the same seam.

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
