# pair#467 — remote driving: a viewer takes control and uses the app through the operator's tab — implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution
> approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or
> superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the remote-driving MVP (internal viewers) on #466's contract. A viewer asks for control and the
operator grants it from Couch. The viewer then clicks, scrolls and types in the operator's cast tab through the scaled
frame, desktop and iPad, and gives control back. The operator can revoke at any moment. The recording attributes the
driver's actions with `by`.

**Architecture:** `drive` already holds the wire, the confinement and the two seams. This issue adds the real
implementations: one injector rule set (`drive.TabInjector`) runs over a `drive.Tab` seam. The real tab is the
recorder's loop, where CDP dispatch happens as a queue input, so `by` is stamped in stream order. The fake tab is a
reduced `recording.State`. The production controller is `drive.MemController`, today's `FakeController` renamed; its
sink is the recorder queue, gated on the cast having gone live. HTTP routes on `webcast.Server`, a drive module in the
viewer shell, and a drive span plus menu on Couch's Chrome cell complete the loop. A real recording of a driven
session replaces the hand-written `driven` fixture.

**Tech Stack:** Go (`cmd/internal/webcast/...`, `cmd/internal/couchtty`, the repo's own `cmd/internal/cdp` over
`--remote-debugging-pipe`), ES modules in `cmd/internal/webcast/web/` (node:test unit tests in `tests/webcast/`,
headless-Chrome live tests behind `WEBCAST_LIVE_CHROME=1`).

**Read first:** `atlas/webcast-format.md` § "Attribution and control (#466)" and § "Remote driving (#466)";
`atlas/webcast.md` § "The recorder (#458)", "In Couch (M2)".

---

## Decisions (made here; told to the TL, ops:0)

1. **Every viewer takes a seat on load.** If `drive/hello` answers 404 (replay server, no controller), the shell has
   no drive UI. The seat is the stream's `?viewer=` identity, so presence (`Attach`) is the seat's life. Watchers hold
   a seat that costs a few hundred bytes; it dies 10 s after their stream does.
2. **With a seat, the notes and strokes author id is the seat's id** (`vN`). The notes name and color stay from
   `identityStore`. A driver's note is then attributable to the driver (fixture coverage "a note by the driver").
3. **Control ops before the recorder has opened are refused** with `ErrNotLive` (409 "The cast is paused."). The
   recorder drops records before its first `Live`, and a controller state that the recording never saw would part
   the controller from the stream's reducer. The gate reads the recorder's own fact (`Recorder.Opened()`, monotonic),
   not Chrome's published phase: `Chrome.publish` can publish `PhaseLive` while `tellRecorder` is still waiting for
   the cast tab's session (`chrome.go` `tellRecorder`'s early return). After opening, a pause leaves control as it
   is (#466: valid while paused).
4. **Touch:** a one-finger tap is a click, and a one-finger drag is down/move/up. A two-finger drag is wheel
   scrolling. On a keyboard device, hover moves are sent (coalesced, ≤ 20/s).
5. **While anyone drives**, file choosers are intercepted in the cast tab and never shown, downloads are denied
   browser-wide, and new targets opened by the cast tab are closed. That covers the operator's own file pickers and
   downloads too while a viewer drives: one rule, no attribution guess. All three are restored when nobody drives.
   Couch says so (TL, 10-12): a notice at each grant ("Ana is driving. Uploads and downloads are off until control
   comes back."), and the drive menu's driver row reads "uploads and downloads off while Ana drives".
6. **Couch UI:** a drive span right after `Chrome ⏸` (and its `💭N`): `🕹Ana` while Ana drives, `✋Bo` when Bo asks
   (`✋2` for two or more), both when both. A click opens the drive menu: a driver row (left click: revoke) and a row
   per asker (left click: grant, right click: deny). The menu reuses the notes menu's box and gesture handling,
   generalized. Revoke is also reachable when the menu is closed: Ctrl+click on the drive span revokes, in any state.
7. **A JavaScript dialog** (alert, confirm, prompt, beforeunload) stops CDP input dispatch and page script until
   the operator answers it. The driver gets a fixed sentence, not a 500 (TL, 10-12): a new `drive.ErrDialog` maps in
   `drive.Answer` to 409 "The page is showing a dialog; the operator must answer it." The Chrome loop doesn't block
   while the recorder's loop does, so it tracks the cast tab's open dialog (`Page.javascriptDialogOpening` /
   `Closed`) and offers it to the recorder as a signal. The CDP tab refuses to dispatch while a dialog is open. An
   in-flight dispatch whose dialog opens returns `ErrDialog` at once and leaves its CDP call to complete later; it
   doesn't wait out the 5 s timeout, so the recorder's loop isn't held either.

## Non-goals

External (non-internal) viewers and their auth; touch dispatch into the page (touch is mapped to mouse acts); hit
tests that descend into out-of-process iframes (`onTarget` accepts the frame owner); answering JavaScript dialogs
or permission prompts for the driver; more than one driver; drag-and-drop of files; remote clipboard.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `Tab` (seam interface) | `cmd/internal/webcast/drive/tab.go` | new |
| `TabInjector` | `cmd/internal/webcast/drive/tab.go` | new |
| `onTarget` (stale rule) | `cmd/internal/webcast/drive/tab.go` | new |
| `FakeInjector` | `cmd/internal/webcast/drive/fake_injector.go` | modified (a `fakeTab` under `TabInjector`) |
| `MemController` (was `FakeController`) | `cmd/internal/webcast/drive/controller_mem.go` | modified (renamed, production) |
| `cdpAct` (act → CDP params) | `cmd/internal/webcast/recorder/drive_cdp.go` | new |
| drive shell logic (`batcher`, `gesture`, `keyAct`) | `cmd/internal/webcast/web/drive.js` | new |
| `driveSpan` / `DriveMenu` model | `cmd/internal/couchtty/drive_menu.go` | new |

- **`Tab`**: the cast tab as an injector sees it, inside one run on the stream's writer:
  `Page() *recording.State` (the followed stream's state: `Live`, `Seg`, `Control`, the DOM tree with parents),
  `At(x, y float64) (recording.NodeID, error)` (the real tab's node at a top-page CSS point, as a recording id; 0 when
  it has none), `Dispatch(a Act) error` (one already-confined act). A `TabRunner` gives tabs:
  `WithTab(ctx, by string, fn func(Tab) error) error`. Records the run causes carry `by`.
  - **Relationships:** `TabInjector` 1:1 `TabRunner`; the real runner is `*recorder.Recorder`, the fake is `fakeTab`.
  - **DRY rationale:** the injector's rules (driver check, live/segment, `confineAll`, stale check, stop at the first
    stale act) are stated once and run in production and in `RunInjectorContract`. Today they are written only in
    `FakeInjector`.
  - **Future extensions:** touch dispatch (`Input.dispatchTouchEvent`) as a new act kind; an OOPIF hit test that
    descends into the child session.
- **`onTarget(st, hit, id) bool`**: the stale rule. It holds when `hit` is `id` or inside it, or when `hit` is a frame
  owner (`iframe`/`frame`) and `id` is inside that frame. The real hit test stops at an out-of-process frame's owner;
  the viewer's `nodeAt` descends into it. Pure, unit-tested on a reduced fixture state.
- **`MemController`**: seats, grace timers, `NextControl` state, `ctl` into a sink, subscriptions. It is the existing
  fake, with the same behavior and a production name. Every caller is renamed; there's no alias.
- **`cdpAct`**: pure mapping from an `Act` plus the top scroll to CDP calls (method and params). Pointer acts map to
  `Input.dispatchMouseEvent` (viewport = page − scroll; `click` = pressed+released with `clickCount`; `move` with
  `buttons` while dragging; `wheel` = `mouseWheel` with deltas). Keys map to `Input.dispatchKeyEvent`
  (`windowsVirtualKeyCode` from `key`, `text` for printable keys on keyDown, `modifiers` bits, and `commands`
  `selectAll`/`undo`/`redo` for the A/Z/Y chords, which Chrome won't run from a synthetic Meta key on macOS). `text`
  maps to `Input.insertText` and `ime` to `Input.imeSetComposition`. Table-tested.
- **drive.js pure parts**: `batcher` (queue acts, coalesce consecutive moves, one request in flight, flush at ≤ 50 ms
  or at once for click/key/text, order preserved), `gesture` (pointer events → acts, touch rules of decision 4),
  `keyAct` (a `KeyboardEvent` → a key act or null, mirroring the chord table's shape only for *what to send*;
  confinement stays the server's), `driveView(control, me)` (button label and state from the latest `ctl`).
- **`DriveMenu`**: laid-out rows for the driver and askers. The layout and drawing are shared with `NotesMenu`
  through a generalized row model (ARCH-DRY: one box, one gesture router).

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `Recorder.WithTab` (+ observer hold/take) | `cmd/internal/webcast/recorder/drive.go`, `observer.js`, `observe.go` | new | CDP input on the recorder loop |
| `Chrome.SetDriving` (guards + banner) | `cmd/internal/webcast/chrome_drive.go`, `inject.js` | new | CDP Page/Browser/Target |
| drive routes | `cmd/internal/webcast/server_drive.go` | new | HTTP |
| `CastConfig.Drive` / `Cast.Drive()` | `cmd/internal/webcast/cast.go` | modified | controller lifetime |
| shell mount (`mountDrive`) | `cmd/internal/webcast/web/drive.js`, `viewer.js`, `annotate.js`, `index.html`, `drive.css` | new/modified | DOM, fetch, EventSource |
| Couch drive state + menu routing | `cmd/internal/couchtty/console_drive.go`, `reserve.go`, `terminal_input.go` | new/modified | tab bar, mouse |
| `driven` scenario recorder | `cmd/internal/webcast/live_record_test.go` | modified | real headless Chrome |

- **`Recorder.WithTab(ctx, by, fn)`**: enqueues `withTab{by, fn, done}` and waits (like `ask`). On the loop it runs:
  (1) **hold**: in every isolated-world context of the cast tab's sessions, evaluate `__couchcastHold()`, which
  returns the observer's pending items and stops it sending. The recorder observes those items unattributed (they
  are the operator's or the page's). (2) `fn(tab)`, whose `Dispatch` calls `r.call(r.top, …)` and whose `At` calls
  `DOM.getNodeForLocation` (viewport point, `includeUserAgentShadowDOM`, `ignorePointerEventsNone`), then `IDOf`.
  (3) **take**: evaluate `__couchcastTake()` in the same contexts. It returns what the observer queued during the
  dispatch (click items included; they no longer flush around the hold) and resumes sending. Those items are
  observed with `by`. Then `afterInput` writes as usual.
  - **Every exit releases the hold.** The take is deferred, so it runs when `fn` returns, fails or is refused. It is
    skipped only when it can't run: a dialog is open, or the context has gone. In the page, the hold expires by
    itself: `__couchcastHold` arms a 1 s timer, and if no take arrives the observer resumes and flushes as usual.
    Items it then sends arrive as ordinary binding reports, without `by`. A late item is never guessed into an
    attribution. A navigation between hold and take is harmless: the new document's observer was never held, and
    the old context's items die with it (the take on a destroyed context fails and is ignored). The fake-CDP harness
    tests `fn` erroring, a take that fails or times out, and a dialog opening mid-dispatch. In each case the next
    ordinary binding report is written, and the observer is not stuck.
  - **Why hold/take rather than an attribution window:** observer reports reach the loop as later CDP events. Any
    window would attribute by arrival time, and would race a revoke (the reducer then refuses the `by` and drops a
    real click). Taking the reports synchronously inside the same queue input orders them exactly.
  - **What stays unattributed (the residual class, written in the atlas):** anything the page reports after the
    take. That covers the `scroll` a wheel causes (scroll events fire on the next frame), page script reacting in a
    later task or frame, and items released by a hold that expired. These are the page's own reactions, which #466
    already leaves unattributed. The driver's direct acts (click, focus, typing as `input`, selection made by keys)
    are taken. Operator input landing in the same milliseconds as a batch is the one mis-attribution the other way.
    The fixture coverage asks for a driver's click and input, not a driver's scroll.
  - `mirror.Observe` gains a `by string` parameter that sets the record's `Source`.
  - **Injected into:** `drive.TabInjector` (as its `TabRunner`). It's tested with a real headless Chrome
    (`WEBCAST_LIVE_CHROME=1`) and through `recorder`'s fake-CDP harness for the hold/take ordering.
- **`Chrome.SetDriving(driver string)`** (with the dialog signal of decision 7): on the loop. When it goes non-empty it sets
  `Page.setInterceptFileChooserDialog{enabled:true}` on the cast tab's session (and on any session that becomes the
  cast tab while driving), `Browser.setDownloadBehavior{behavior:"deny"}`, and a `driving` flag. `handle` closes
  (`Target.closeTarget`) any `targetCreated` page whose `openerId` is the cast tab's while the flag is set. It pushes
  the banner text to `inject.js` (`__couchcastDriver(name)`): "Ana is driving", in the control's closed shadow host,
  which is already outside the recording. When it goes empty, everything is restored (`default` downloads,
  interception off, banner gone).
- **Drive routes** (`server_drive.go`): `POST <cast>/drive/hello|control|input`, admitted like `boardRoutes`: JSON
  content type, a body deadline, `MaxBytesReader(drive.MaxBody)`, and per-seat `broadcast.RateLimit` (input 30/s
  burst 30, control 2/s burst 5) with 4 in flight per seat. `hello` is rate-limited per cast (2/s burst 8) and capped
  at `maxViewers`×2 live seats. Answers go through `drive.Answer`. The events route calls `Controller.Attach` for
  `?viewer=` when a controller is set, replacing `Presence` for that case. An unknown seat still streams; it just
  isn't attached.
- **Cast wiring**: `StartCast` builds `drive.NewMemController(sink, inj, clock, drive.DefaultGrace)`. `sink` is
  `liveGate{rec}`: `Put` returns `drive.ErrNotLive` while `rec.Opened()` is false, else `Emit(r, 0)`. `opened`
  is an atomic flag the loop sets when it processes the first `Live`, and it never resets. A `ctl` enqueued after
  the flag reads true is therefore ordered after the open. `inj` is `drive.NewTabInjector(chrome.Recorder())`. A goroutine
  bounded by the cast forwards `Subscribe()` to `chrome.SetDriving`. `Cast.Close` closes the controller. `Cast`
  exposes `Drive() drive.Controller` for Couch.
- **Couch**: `webcastStarted` subscribes `cast.Drive().Subscribe()` on a tracked goroutine and applies each state on
  the loop (`runTerminalCommand`) into `webcastState.control`. `StatusModel.WebDrive` renders the span, and mouse
  routing opens the menu, grants, denies and revokes.

### ARCH notes

- **ARCH-FUNERAL:** seats and keys are memory only; they end at grace or `Cast.Close`. Per-seat rate buckets live in
  a map keyed by seat id, removed when the seat leaves (the controller reports leaves through `Subscribe`; the route
  table prunes buckets of unknown seats on each request, so the map is bounded by live seats plus one). `ctl` records
  are ~100 B per human action, kept as long as the recording. Download and file-chooser settings are restored when
  driving ends and die with Chrome. The fixture is replaced, not added (budget test).
- **ARCH-CONSTRAINTS:** the per-seat limits above. The recorder loop blocks during a batch: ≤ 64 acts, each one CDP
  round trip (~1 ms), bounded by the 5 s call timeout (decision 7).
- **Ordering:** a control change and an injection are both recorder-queue inputs. Revoke never waits on a batch: the
  controller drops its lock before `Inject` (contract), and the batch re-checks `Page().Control` on the loop.
  `RunInjectorContract` exercises the revoke-overtakes-batch ordering; a recorder test enqueues a `ctl revoke`
  between hold and take to show that the taken items still validate.

## Milestones

Four review boundaries: M1 the Go core, M2 the viewer shell, M3 Couch and the in-page banner, M4 the recorded
fixture and the atlas.

---

## Chunk 1 — M1: the real injector, controller, guards and routes

### Task 1: `Tab`, `TabInjector`, the stale rule; `FakeInjector` over a fake tab

**Files:** create `drive/tab.go`, `drive/tab_test.go`; modify `drive/fake_injector.go`.

- [ ] Test `onTarget` on `DrivenPage` state: self, a descendant hit, an unrelated hit, and an iframe owner hit with
  the id inside its frame (use the fixture's iframe if there is one, else a hand-built `recording.State`). Run it:
  it fails (undefined).
- [ ] Write `Tab`, `TabRunner`, and `TabInjector.Inject`, moving the rule order verbatim from `FakeInjector.Inject`:
  `by` drives in `Page().Control`, else `ErrNotDriver`; live and `b.Seg == Page().Seg`, else `ErrNotLive`;
  `confineAll`; then per act, for down/click/wheel `At` plus `onTarget`, else `ErrStale`; then `Dispatch`.
- [ ] Rewrite `FakeInjector` as `TabInjector` over `fakeTab`: hit testing by the smallest box, and effects as today
  stamped with the run's `by`. `fakeTab.Put` follows the stream. The password case now records
  `input{secret:true}` (#458 landed), and the `pending #458` refusal is removed. `fake_injector_test.go` runs
  `RunInjectorContract` unchanged; it passes.
- [ ] `go test ./cmd/internal/webcast/drive/`, then commit `#467 M1: drive: one injector rule set over a Tab seam`.

### Task 2: `MemController`

**Files:** `git mv drive/fake_controller.go drive/controller_mem.go`; rename the type, the constructor and the tests.

- [ ] Rename `FakeController` to `MemController` and `NewFakeController` to `NewMemController`, and rewrite the doc
  comment: the production controller, whose sink in production is the recorder queue.
- [ ] Add a contract-adjacent test: a sink whose `Put` fails (`ErrNotLive`) leaves the control state and the
  subscribers unchanged, and `Request` returns the sink's error (Answer: 409 paused).
- [ ] `go test ./cmd/internal/webcast/drive/`, then commit.

### Task 3: observer hold/take and `by` in `Observe`

**Files:** `recorder/observer.js`, `recorder/observe.go`, `recorder/recorder.go`, tests in `recorder/recorder_test.go`
(fake CDP harness).

- [ ] `observer.js`: `globalThis.__couchcastHold = () => { held = true; collect scrolls/selection; return
  JSON.stringify(pending.splice(0)) }`. `__couchcastTake = () => { collect; held = false; return JSON of the
  pending items }`. While `held`, `send` (and the immediate click flush) only queues. A `hello` is never held.
- [ ] `mirror.Observe(t, session, frame, it, off, by)`: the built click/input/focus/scroll/selection/hover/ptr record
  gets `Source{By: by}`. Existing callers pass "". Unit test: an item observed with `by` while that viewer drives
  (state from `DrivenPage`) is written with `by`, and with another `by` it is dropped and counted.
- [ ] Commit.

### Task 4: `Recorder.WithTab` and the CDP tab

**Files:** create `recorder/drive.go`, `recorder/drive_cdp.go`, `recorder/drive_cdp_test.go`,
`recorder/drive_test.go`.

- [ ] `cdpAct` table test (pure): each act kind to its CDP method and params, including scroll offset, click count,
  drag buttons, modifiers bits (Alt 1, Ctrl 2, Meta 4, Shift 8), printable text, Enter's `\r`, the selectAll/undo/redo
  commands, insertText and imeSetComposition.
- [ ] `withTab` input on the loop: hold → `fn(tab)` → take, each held/taken item through the `binding` item path
  (refactor `binding`'s per-item body into `observeItem(at, session, ctx, it, by)`). Contexts: every
  `r.contexts[session]` whose session is the cast tab's (`r.top`) or a descendant via `r.parents`.
- [ ] `At`: `DOM.getNodeForLocation{x: x-scrollX, y: y-scrollY, includeUserAgentShadowDOM: true,
  ignorePointerEventsNone: true}` on `r.top`, giving `backendNodeId`, then `IDOf(r.top, backend)`. Not mapped gives
  0 (stale).
- [ ] Fake-CDP test: a `withTab` run where the fake answers `__couchcastTake` with a click item writes
  `click{by:v1}` after the page records; the hold's item is written without `by`. A `ctl revoke` enqueued *before*
  the run makes `Page().Control` show nobody, so `TabInjector` refuses `ErrNotDriver` with no CDP input sent.
  Release paths: `fn` errors; the take call fails; the dialog signal fires mid-dispatch (`ErrDialog`, returned
  without waiting for the call). After each, an ordinary binding report is written, so the recorder isn't stuck.
  `Opened()` is false before the first `Live` and true after it.
- [ ] Commit.

### Task 5: live injection in real Chrome

**Files:** `live_drive_test.go` (new, `WEBCAST_LIVE_CHROME=1`, headless; never focus-stealing).

- [ ] Build a page (data URL or `httptest`) with `#q`, `#save`, a `target=_blank` link, a file input and a download
  link. Go live with `liveRecordingChrome`, emit `ctl request`/`grant` for v1 through `Chrome.Emit`, and call
  `SetDriving("Ana")`. Then through `TabInjector{rec}`: click `#q`, text "invoice", Enter, and check the recording
  holds `click{by:v1}`, `input{v:"invoice",by:v1}`. A click on the file input opens no chooser (no
  `Page.fileChooserOpened` reaches the page; the intercept event is seen). The download link gives no file in a temp
  download dir. The `_blank` link: the new target closes within 2 s. A stale id gives `ErrStale`. Meta+V gives
  `ErrRefused`. Then `SetDriving("")` restores the settings: the file chooser intercept is off.
- [ ] Run: `WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast/ -run TestLiveDrive -count=1` (unsandboxed: Chrome).
- [ ] Commit.

### Task 6: `Chrome.SetDriving` guards and the banner

**Files:** `chrome_drive.go` (new), `chrome.go` (the `handle` target-created hook; reapply on cast-tab change in
`tellRecorder`), `inject.js` (`__couchcastDriver`), test in `chrome_loop_test.go` (fake conn: calls issued on set,
unset and tab switch; an opener-matched target closed, another not).

- [ ] Dialog signal: the loop tracks `Page.javascriptDialogOpening`/`Closed` on the cast tab's session and exposes
  it to the recorder (`recorder.Options.Dialog func() <-chan struct{}`, a channel closed while a dialog is open,
  replaced when it closes). `drive.ErrDialog` and its `Answer` row are added to `drive/injector.go`/`answer.go`,
  with an answer test.
- [ ] Fake-conn test first, then the implementation. Commit.

### Task 7: drive routes, Attach, Cast wiring

**Files:** `server_drive.go`, `server_drive_test.go` (new); `server.go` (route dispatch; events → Attach); `cast.go`
(`liveGate`, controller lifetime, `Drive()`, forwarding to `SetDriving`); `cast_test.go`.

- [ ] Route tests over `MemController` + `FakeInjector` + a `Hub`. Strategy: a table that walks each error
  `drive.Answer` maps through the real route (status and sentence exactly), plus a fuzz of bodies against
  `ParseBatch`/`MaxBody` that may never echo input or reach the injector. Seat presence is checked by attaching and
  detaching streams under a short grace.
- [ ] Cast test (FakeBrowser + a recorder that hasn't opened). Strategy: interleavings. Published live with the
  recorder not yet opened gives 409; after open, the `ctl` lands. A revoke while a fake injection blocks returns at
  once. `SetDriving` follows the controller's states.
- [ ] `go test ./cmd/internal/webcast/... -race`, then commit.
- [ ] **M1 close:** `sdlc milestone-close --issue 467 --milestone M1`.

## Chunk 2 — M2: the viewer shell

### Task 8: drive.js pure parts

**Files:** `web/drive.js`, `tests/webcast/drive.test.mjs`.

- [ ] Tests first. `batcher`: a property test over random act sequences with a fake fetch that answers late and
  sometimes refuses. It checks that order is preserved, at most one request is in flight, consecutive moves
  coalesce, and a refusal stops the queue and shows the sentence (a 401 re-joins). `gesture`: replayed pointer and
  touch traces, including malformed ones (a cancel mid-drag, a second finger landing late). `keyAct` and
  `driveView`: tables.
- [ ] Implement, then `node --test tests/webcast/drive.test.mjs`. Commit.

### Task 9: mount the drive UI

**Files:** `web/drive.js` (`mountDrive`), `web/drive.css`, `web/index.html`, `web/viewer.js` (seat before
EventSource; `couchViewer.drive`), `web/annotate.js` (author id = seat id; mode exclusivity), `server.go` (embed list
+ `assets`), and the served-files closure test.

- [ ] The shell: `hello` (name from `identityStore`), then the stream with `?viewer=<seat>`. On 404 it streams
  without a seat and shows no drive UI. A bar button cycles "Ask to drive" → "Waiting… (cancel)" → "Give back
  control", and shows "Ana is driving" when someone else drives. While this viewer drives, a capture layer over
  `#box` takes `pointer-events:auto; touch-action:none`. A hidden `<textarea>` keeps the focus for keys, text and
  IME (`beforeinput`/`input` → text acts; `compositionupdate` → ime; Enter/Backspace/etc. through `keyAct`), which
  raises the iPad keyboard on tap. Pointing and note modes are off while driving.
- [ ] Live test `viewer_drive_live_test.go` (headless, `WEBCAST_LIVE_CHROME=1`): a `Server` with `MemController` over
  `FakeInjector` fed by the hub at tablet width. The viewer asks; the test grants; mouse click and typing reach the
  injector as the right acts on the right ids; a touch tap is a click; give-back releases; a revoke from the test
  ends the capture layer.
- [ ] `node --test tests/webcast/`, the live suite, then commit.
- [ ] **M2 close:** `sdlc milestone-close --issue 467 --milestone M2`.

## Chunk 3 — M3: Couch

### Task 10: drive span and menu

**Files:** `couchtty/drive_menu.go` (+ test), `couchtty/notes_menu.go` (generalized rows), `couchtty/reserve.go`
(`StatusModel.WebDrive`, span rendering, `RenderedStatusRow.WebDrive`), `couchtty/console_drive.go` (state,
subscription, actions), `couchtty/terminal_input.go` (clicks), `couchtty/console_webcast.go` (subscribe in
`webcastStarted`, forget on end).

- [ ] Render and layout: a table over control states, giving the span text and the menu rows (including the
  driver row's "uploads and downloads off while Ana drives").
- [ ] Console tests drive the real producer (lesson #470 BR-25), a `Cast` over `FakeBrowser` with its controller.
  Strategy: every gesture on the span and the menu, in every control state, ends in the controller state it names.
  Revoke works from each state, with the menu open or closed. The grant notice is shown. Run with
  `-race -count=20`.
- [ ] Commit. **M3 close:** `sdlc milestone-close --issue 467 --milestone M3`.

## Chunk 4 — M4: a real driven fixture, the atlas

### Task 11: record `driven` from a real driven session

**Files:** `live_record_test.go` (`scenarios["driven"]`, with a harness that runs `MemController` + `TabInjector`
over the recording Chrome, grace 1 s); `recording/testdata/fixtures/driven/` (replaced by `fixturegen -recorded
… -name driven`); `recording/internal/fixturegen/main.go` (drop `-driven`); `recording/contract.go`
(`DrivenRecording` stays as the contracts' page builder, documented as no longer the fixture); fixtures README.

- [ ] The scenario: v1 joins and asks; the operator grants. v1 clicks the search box, types, and presses Enter. The
  operator clicks (direct CDP, unattributed). v1 posts a note through the board with author v1. v1 clicks a link
  (navigation while driving). Pause and resume. v1 releases. v2 asks and is denied. v1 asks again, is granted, and
  is revoked. v3 asks, then its stream drops (leave after grace).
- [ ] Re-point consumers to derive from the data (lesson #474): `query/events_test.go`'s `TestEventLinesAttribute`
  picks its moments from the fixture's `ctl`/`by` records; `tests/webcast/replay.test.mjs`'s drive spans are computed
  from the records; `replay_server_test.go`'s duration is read from the fixture. Coverage test unchanged and green.
  Budget test green.
- [ ] `go test ./cmd/internal/webcast/...`, `node --test tests/webcast/`, then commit.

### Task 12: atlas and close

- [ ] `atlas/webcast-format.md` § Remote driving: "as built (#467)": `TabInjector`/`Tab`, hold/take attribution and
  its one residual, `MemController`, `liveGate`, decisions 1–7, and the known dialog limit. Add a `## Revisions`
  entry. `atlas/webcast.md` § In Couch: the drive span and menu, the banner. Check that `atlas/index.md` links them.
- [ ] Full verification: `make -k test`, scratchpad-`TMPDIR` test-changelog, `go test ./...`, and the live suites
  with `WEBCAST_LIVE_CHROME=1`. Then `sdlc close --issue 467 --verified '…'`.
- [ ] Merge `origin/main`, `sdlc pr`, `sdlc merge --yes`. Report to ops:0; the operator's two-device smoke test is
  the TL's to schedule.

## Revisions

- 2026-10-12, plan-quality round 1 and the TL's approval (ops:0, 2efb3d52). PQ-1: the live gate reads
  `Recorder.Opened()`, not Chrome's published phase. PQ-2: every exit releases the observer's hold, the hold expires
  in the page after 1 s, and late items stay unattributed. Minors: the residual class is stated in full; tests are
  written as strategies; non-goals added. TL asks: Couch tells the operator that uploads and downloads are off
  while a viewer drives; a dialog gives the driver a fixed sentence (`drive.ErrDialog`) instead of a 500. The
  smoke test moves after landing (issue Done when, Revisions).
- 2026-10-12, M1 as built and M1 review round 1:
  - File choosers are intercepted on every page session, and on each page attached while driving, not only the cast
    tab. This is decision 5's one rule; input reaches only the cast tab anyway.
  - The guards and the dialog signal follow the active tab in every phase (`castTab`), since control survives a
    pause. BR-1: the dialog signal is a per-session set, recomputed after every loop input, so a dialog opened
    while paused, another tab's dialog, a tab switch and a closed tab all come out right.
  - Route budgets: there's no `maxViewers`×2 hello cap. Seats waiting for a stream are bounded by the hello rate
    times the grace period. The per-seat table drops keys the controller refuses and sweeps keys idle past a
    minute once full (BR-3), instead of pruning on leave.
  - BR-2: the live test asserts the file chooser through a test-only loop event tap. Downloads are checked against
    a positive control (the operator's download lands, the driver's doesn't), and the restore is checked as
    well.
  - The plan's "revoke between hold and take" recorder test is structurally impossible: both are inside one
    queue input. `TestWithTabSeesARevokeQueuedFirst` covers the ordering that exists.
- 2026-10-12, M2 as built and the M2 review round 1. Rule (review family `plan-drift-unrecorded`, second time): a
  Task's named files, fixtures and doubles are contract, and any deviation gets a line here in the milestone's
  closing commit.
  - **Button labels.** The button reads "Cancel request" while asking, not "Waiting… (cancel)", and the bar note says
    "Waiting for the operator…".
  - **Lost seat.** A 401 offers Rejoin, which reloads the page: the stream holds the seat, so a new seat needs a new
    stream.
  - **Live test (BR-9).** `TestDriveLiveTwoViewers` runs MemController over the fake injector (the production rules
    over the fixture's page) through a spy that only notes the batches. It asserts the driver's click lands in the
    stream with `by`. A is at desktop width (1280×800); B is touch at an iPad's portrait size (768×1024, DPR 2).
  - **The wire gate (BR-7, BR-8, and the pacing Minor).** One gate decides what a viewer may send: `batcher`,
    through `wireAct` and `splitText`:
    - coordinates are clamped to [0, 2^20], so a drag captured past the frame's edge still ends with an up;
    - a text act is cut to ≤ 1000 runes and ≤ 2 KiB encoded;
    - a body stays ≤ 8 KiB;
    - requests are spaced 40 ms apart (25/s, under the seat's 30/s).

    A seeded property test checks every sent body against those rules.
  - **Minors.**
    - Key ups are paired (`keyTracker`: an up for every down, also after its modifier is released).
    - There's no hover while a button is held.
    - The toast and the request options come from collab.js (`sayer`, `poster`'s headers).
    - The page's stylesheets are walked by the served-files test.
    - Rejoin stays shown until the page reloads.
- 2026-10-12, M2 review round 2:
  - **BR-16 (input pairing, second in its family).** The cast tab owns letting go. `TabInjector` tracks what the
    driver holds (a button, keys) and releases it in these cases:
    - before another driver's input;
    - before input on another page;
    - when the driver's own batch is refused (the shell drops its queued ups);
    - at once on any control change: `drive.Releaser`, which `MemController` calls off its lock, bounded at 10 s.

    The shell resets its gestures and key tracker whenever control ends or a batch is refused.
  - **BR-15.** Every act carries the segment it was made on, a batch never mixes segments, and moves don't coalesce
    across pages. A late batch is refused (409 paused), never played into the new page.
  - **The Minor.** `driveView` now returns every shell state (none, asking, driving, paused, other, lost) with its
    action and capture, and `render` only paints it.
