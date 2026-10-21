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

## Operator annotation (#494)

The casting page never reaches the recorded scene, bindings or assets.

The cast Chrome uses the viewer's `mountAnnotation` (`web/annotate.js`) and
`collab.js` dialog, swatches and geometry inside the existing excluded closed
shadow host. `operator_script.go` assembles these trusted embedded modules into
isolated-world script scopes; `operator.js` supplies local page coordinates and
the CDP binding transport. No page-origin HTTP or localStorage is involved.
Off/Pen/Note start Off. The shared note gesture previews nothing on pointer-down;
crossing the movement threshold draws a drag box, while a click places its default
box only on release. The completed box stays visible while its draft is edited.
A remote driver disables annotations and hides/unfocuses
an open editor while retaining its draft; control returning leaves the pen Off.
Navigation discards the draft. Pausing holds it on the same document.
Couch’s Chrome Alt-click menu adds `Annotation · Off · Pen · Note` below the
code rows, using their shared layout and hit-testing. It sends the same isolated-world
mode command without focusing Chrome; a refusal appears in Couch’s notice. Cast-row
left actions, right-click link copies, and Alt menus only on LIVE/Chrome are unchanged.
The isolated-world input `__couchcastAnnotationAllowed(boolean)` delegates to the
shared mount's `setAnnotationAllowed`. Every isolated document starts disabled; false forces Off,
cancels gestures and hides/holds the operator editor with a disabled hint. Returning
true leaves the pen Off and does not override driving or pause. View snapshots do
not reset the input. #523 supplies versioned permission to each exact isolated
context after hook installation. A delivery receipt suppresses duplicate resets
and stale late enables; an uncertain enable is fenced false before another enable.
Board permits are captured at binding ingress and checked again at the ordered
recorder write. A fresh save can use a held draft after re-enable; an old queued
save cannot cross generations. Operator processing uses only Board's leaf lock,
never the viewer command lock held while waiting for ordered cleanup.

The Board remains the authority (`board/operator.go`, `operator_view.go`): Go
assigns operator authorship, HTTP cannot claim it, and a bounded latest-value view
carries current open notes plus recent stroke chunks. The recorder's operator
adapter accepts only the current top-frame isolated context. Prepare resolves the
observer's node path once and binds a draft handle to that document and recorded
node. Save never resolves a potentially moved path again. Writes settle from the
recorder sink outcome, not enqueue success. A lost reply is checked using the same
monotonic request ID; an uncertain write cannot create another note. Reconciliation
can only complete its originating action/body; another draft stays open and unsent.
Projection document changes (including inactive tabs and cast end) retire pending
transport state before UI updates, without replaying writes. IDs remain monotonic;
late replies and resolved continuations cannot settle a newer document’s operation.
Same-document pause and driving retain reconciliation.
Operator Escape is captured at the window boundary so a Couch menu selection works
with page focus unchanged; Off and unavailable modes leave page Escape alone.

`recorder.CaptureScreenshot` hides the entire Couch host with a document-local
lease, requires confirmation before and after capture, and restores only that
lease on every exit. Missing/expired confirmation means discard the pixels.
The periodic fidelity worker and `Recorder.CaptureCrops` submission capture both
use it. Capture identity and event barriers still protect the submission text/PNG
interval across navigation and tab changes.

Remote input acquires a non-expiring suppression token before hit testing or
synthetic dispatch. Confirmed completion releases it. A timeout or dialog that
leaves execution uncertain retains the target guard across replacement documents,
refuses further remote batches, and asks the operator in Couch to close/reopen the
cast Chrome. A new document starts hidden until Couch supplies that target's
current guard state. UI state updates carry a sequence so late replies cannot
undo a newer guard. Annotation never delays input to protect an editor: it is
already off while a viewer drives.

The isolated world protects JavaScript binding identity, not the shared DOM: page
code can see or remove the host. The host is not a security boundary. Existing
broadcast/driving controls stay in Couch; #494 adds no browser control authority.

## The recorder (#458)

`Chrome` starts it when `LaunchOptions.Record` is set, forwards **every** CDP event to it (`Send` never blocks the
loop), and tells it the live phase from `publish` (`Live` on the active tab, a switch when the tab changes, `Title`,
`Pause`). It is the record stream's **single writer**: records made elsewhere (#467's control and injected actions, #470's notes
and strokes) enter its queue as `recorder.Emit`, stamped on arrival and checked by its reducer.

- **Pure core:** `mirror` (`mirror.go`, `observe.go`) holds CDP's DOM per session (`getDocument` pierce, whitespace
  included) and the id map (`backendNodeId` → id per session for one document, keyed by loader id, with a 50 000-node
  graveyard so moves keep ids). Every record goes through its own reducer first, so the output validates by
  construction; refusals are counted (`Stats`). It leaves out the control (`data-couchcast-ui`), user-agent shadow
  trees, and password values. `IDOf`/`NodeOf` expose the map (#467). CDP sends a node's children only once they are
  requested: a node that arrives with a count but no children, or that later reports `DOM.childNodeCountUpdated`
  for children the mirror doesn't hold (one inserted empty and filled by script, as asciinema-player fills the
  blog's cast embed: pair#513, measured), goes on `Expand`, and the shell's `DOM.requestChildNodes` brings them as
  `setChildNodes`.
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

## Sketch recording model (#751)

Persistent sketches extend version-1 recording events; the [format contract](webcast-format.md#persistent-sketches-751)
lists all eight kinds, pressure triples, required labels and interruption rules. The recording model provides
persistence, replay state and geometry. [Sketch drawing](#sketch-drawing-752) adds the UI, source preparation,
privilege ingress and transport receipts; #753 owns historical image export.

`recording/sketch.go` and `sketch_codec.go` own bounded wire records and preserving codecs. `sketch_state.go` owns
indexed histories and operation receipts, integrated into `State.Apply` and control transitions. The browser port
is `web/sketch-model.js`, loaded by `model.js`; shared prefix fixtures pin both accepted projections and refusals.
Unknown wire members survive codec rewrites but stay outside authoritative core/snapshot state. Actual request
semantics are reconciled beside the claimed digest, so a reused digest cannot disguise a changed command.
`sketch/geometry.go` and browser `annotation-anchor.js` preserve pressure while projecting recorded anchors and points.

`board.Writer` and `recorder/mirror.go` share `OrderedTime` and `SketchBudget`. Strict timestamp fences on both sides
of `sketch-begin` freeze its preceding source prefix; clear also strictly follows earlier records, and overflow
refuses admission. Exact retries are reconciled before timestamp/document checks and never append or charge twice.
Historical labels use `Emit` with document zero, allowing valid interrupted work to be labeled without current DOM.
The recorder shell admits all eight kinds through its existing ordered queue.

The writer's 32 MiB cap includes exact encoded sketch bytes/newlines and outstanding terminal reservations.
Each sketch initially reserves 16 KiB across label/end/interrupt components, verified against maximal `Encode`
envelopes; reservations cannot be spent by new drawing. Completed labels, clear and recording end release unused
capacity while historical bytes remain charged. Core `history_bytes` separately counts known-field encoding for its
32 MiB bound; the producer budget additionally counts preserved unknown wire members. Admission inspects one entry;
reopening uses the explicit one-pass `RebuildSketchBudget` helper.

Sketch catch-up is derived state: `recording/sketch_snapshot.go` and browser snapshot restoration carry the current
driving holder and bounded sketch histories in one ephemeral `sketch-snapshot` frame. `CatchUp`/browser `catchUp`
reduce through the same model before compacting sketch/control history. Live bootstrap, Player rebuild and Replayer
seek explicitly enable the frame; ordinary disk/source replay and ordered writers refuse it. Restore validates owned
copies before installing state, and future control events retain strict validation. Disk history stays append-only;
control chatter adds no retained history to the snapshot. Shared catch-up tests compare sketch projections separately
from DOM hashes.

Acceptance is not proof of persistence. `board.Writer` retains per-operation sink uncertainty and returns it on
retries without reappending. The pure mirror accepts into its output queue; `Recorder.write` retains sink failure in
`r.err`, and `Close` returns it. Sketch receipt queries and acknowledgments consult persistence outcome before
reporting success. Both accept-then-error and reject-before-write sinks are covered by producer tests.

## Sketch drawing (#752)

`web/sketch.js` mounts the shared toolbar, required-label editor and pointer controller in the viewer and isolated
operator UI. Drawing follows the existing driving holder: the operator when no viewer drives, otherwise only the
granted drive-link seat, with annotation enabled. The server rechecks the authenticated principal, annotation
generation and drawing epoch on ingress; client availability is only an affordance. Pointer and view-only links,
and recording replay, render accepted paths without drawing permission or a palette endpoint.

The trusted console sketches independently of viewer annotation permission when no viewer drives; ordinary
viewer admission still requires annotation and the current driving seat. Console unavailable controls explain
readiness or ask the viewer to release driving control. Paused/ended casts explain resume/new-cast recovery; Done is disabled until
there is a sketch or preparation to finish. Read-only viewer toolbars honor their hidden state.

Choose a fixed colour, then Sketch, and wait for its held recorded scene. Pressure samples stream before pointer-up;
mouse/touch use pressure 1, pen hover is ignored, and cancellation seals a truncated partial stroke. Physical release
does not release a pending logical gesture: the accumulator token stays owned until its anchor and terminal drain
into the FIFO or retirement fences its callback. Successor contacts are visibly consumed without buffering or
allocation; a fresh down is required after readiness. The shared five-second anchor observer bounds this wait.
Undo/Redo also wait for logical gesture release, then wait
for confirmation and affect only the current unfinished sketch. Done, Escape or a tool change first shows
“Saving…” and stops new input while the existing gesture is finished, including an anchor still being resolved
after pointer-up. The required-label prompt opens after that work is confirmed; Escape there resumes the same
scene. Enter or Save label submits through the same settlement guard and completes the requested transition
only after acceptance.
Handoff releases drawing input promptly and leaves an independent old-author label action while its UI credentials
survive. Normal remote input is suspended while the sketch overlay owns input; the driving seat remains granted.

`web/sketch.js` and `web/sketch-policy.js` own this finishing transition for both viewer and console; adapter tool
controls retain the requested exit until it settles. Finishing has a five-second deadline. If it cannot be confirmed,
the UI says “Couldn't save; accepted strokes kept”. An unknown write offers Check confirmation; a failed anchor
leaves label/escape available to acknowledge the unsaved gesture. It does not report completion or
discard an unknown write: the original operation remains available for reconciliation. A late callback cannot
silently turn the failure into success.

`web/sketch-policy.js` also owns one operation in flight per sketch and bounded FIFO chunks. An uncertain
reply stops new geometry: Check confirmation queries the same operation/sequence/epoch, resending its exact body
only after status says it was not observed. A definite refusal leaves the sequence unconsumed. Continue drawing is
offered only for an authoritative drawing status at the expected sequence, the same eligible epoch and a ready held
scene; it confirms a fresh truncated seal for any accepted open prefix before enabling drawing or Undo. Pause can
resume on the same source. Navigation, handoff/reset, clear, end or mutation-idle expiry retire source ownership;
none transfers unfinished work to a new drawer. Reconnect restores accepted history, not an unsent gesture or draft.

### Source preparation and privacy

`board/sketch.go`, `board/writer_sketch.go`, `recorder/sketch.go` and `recording/sketch_source.go` capture the source from
the ordered begin candidate before committing it. The acknowledged descriptor names its exact prefix, time,
document, tab and viewport. `web/sketch-scene.js` reduces that prefix once into an independent Model/Renderer,
freezes animations, and resolves only source-bound recorded asset hashes. It never freezes whichever live DOM the
viewer happened to have when the reply arrived.

The casting Chrome uses `operator-sketch.js`, `chrome_operator_sketch.go`, `chrome_sketch_scene.go` and
`sketch_scene_script.go`. Its closed shadow root hosts the controls, but the security boundary for recorded content
is an opaque-origin `data:` child frame, sandboxed with `allow-same-origin` and without `allow-scripts`. Chrome
installs the shared Model/Renderer into that child's trusted isolated world through CDP. Toolbar, draft and pointer
policy stay in the parent isolated world. After preparation, gesture queries cross the private CDP bridge only as
read-only hit-testing and placement.
Casting-page JavaScript never receives the recorded scene DOM, trusted bindings, lease handles or recorded asset
bytes/blob URLs, including through named-frame targeting. The page can still see or remove the outer control host.
Page CSP remains in force: shell/module preflight precedes begin, and a blocked preparation fails visibly without a
CSP bypass. Parent context, child loader/frame, cast and source retirement fence late calls, revoke asset URLs and
dispose the scene. Scene frames and their nested targets are excluded from recording and ordinary drive input.

### Delivery, rendering and limits

`board/sketch_delivery.go` derives compact display facts after the authoritative reducer accepts a record. The
existing Chrome worker sends ACKed journal batches; a gap or new context installs an atomic render-only bootstrap.
`web/sketch-delivery.js` checks epoch/cursor/chunk continuity, and `sketch-render.js` applies indexed display patches
without another history-legality machine. Its anchor index includes accepted empty and undone strokes, requires
their exact source document and excludes cleared sketches. Point chunks neither rebuild that index nor clone full
history. The shared painter retains accepted ink and paints only a last join plus new samples; geometry,
visibility/Undo/Redo, bootstrap or canvas-size changes trigger a full repaint. Bounded unconfirmed ink uses a
separate canvas and is replaced by accepted echo.

| Resource | Bound |
| --- | --- |
| Sampling and queued points | 50 ms cadence; 64 points/chunk; 16 queued chunks; a pending asynchronous hit-test buffers at most 1024 points. |
| Recorded sketch history | 4096 points/stroke, 16384 points/sketch, 64 strokes/sketch, 256 sketches and the existing 32 MiB history budget, including producer terminal reservations. Labels are nonblank and at most 1024 UTF-8 bytes. |
| Begin source | 8 MiB held DOM/state and 16 MiB complete prefix; exceeded limits refuse before begin commits. |
| Private reads | 32 KiB chunks; recorded assets retain the existing capture limit of 32 MiB per body. |
| Operator delivery | 256-entry/1 MiB journal; batches of at most 32 records/64 KiB; render-only bootstrap at most 36 MiB. |
| Source lifetime | One held source cast-wide; expires after two minutes without an accepted sketch mutation. Reads/status do not renew it. |

Pure and counted tests live in `tests/webcast/sketch-*.test.mjs`; source, ingress, writer and journal tests sit beside
their Go owners. `TestSketchLiveTwoViewers` runs the real server, Board/Writer/Hub, controller and two disposable
headless viewers: pressure ink arrives before pointer-up, echo is unique, Undo/Redo changes canvas pixels, pointer
and view-only links cannot draw, and recording replay agrees with the reducer. Run it with
`WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run '^TestSketchLiveTwoViewers$' -count=1 -v`.
The private console scene has separate owned Chrome checks in `chrome_sketch_scene_test.go` and operator sketch tests.

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
  - The capture at t stops the animations at t, then waits (bounded, `ready`) for the fonts, images and stylesheets
    bound by t to load (#482). The tab painted with bytes it had; an asset bound 1 ms before a frame was still
    loading in the replay, which flaked rec-blog's first frame to 6.75%.
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

## Capability authority (#523)

Recorded casts start with annotation and outside driving OFF. `Board` owns
annotation admission and `drive.MemController` owns drive admission; Cast and
SSE publish their snapshots. Link roles narrow permission independently, and
links/codes stay stable across toggles. HTTP captures an admission generation
before reading the body and rechecks it at mutation/dispatch, including between
input acts. A quick OFF→ON cannot revive an old request.

Turning annotation off acknowledges an ordered `annotation-clear`; driving off
acknowledges one `ctl reset` clearing the active driver and every pending request,
then releases held input. The recorder deduplicates each cleanup generation.
Admission stays closed during cleanup. A confirmed unsubmitted failure offers an
explicit retry; an uncertain external effect or lost recorder/CDP leaves the cast
broken with “end the cast” as its action. Held inputs are removed only after
acknowledgment. Controller work and input callbacks are canceled/joined, including
when the HTTP caller leaves. Cast owns bounded, single-flight button workers.

The `capabilities` SSE event carries annotation/drive support, link permission,
effective availability, visible state, generation and attempt. It replays the
latest snapshot on reconnect and does not expose internal errors. Cast updates
carry the same authority state for the operator's recovery notice. Viewer mutations
carry their request-creation generation in `X-Couch-Annotation-Generation` or
`X-Couch-Drive-Generation`; isolated-world operator requests carry a decimal-string
`generation` from the acknowledged permission delivery. Admission matches that
origin generation under the authority lock, then rechecks the permit at ordered
write/input dispatch. Missing or retired generations fail closed, including payloads
created before off but delivered after re-enable. The shared
`capability.Project` maps lifecycle to one action: enable, disable, wait, retry,
end cast or start a new cast. See `cast_capabilities.go`, `server_capabilities.go`,
`board/annotation.go`, `drive/capability.go` and `recorder/capability.go`.

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
  `WebcastCell` draws `Chrome ▶` / `Chrome ⏸` **after** the screen broadcast's `LIVE ⏸ 👆 💭N`, so the
  broadcast's fixed-column indicator check never moves. Click toggles, right-click copies the link, Ctrl with
  either button ends (macOS terminals may send Ctrl+left as a right click with the Ctrl bit). Live, the cell is
  drawn in `broadcast.LiveSGR`, LIVE's own style, so the two cast indicators read as one family (#501);
  `IndicatorShown` matches LIVE's text, so it doesn't mistake one for the other.
- **The Chrome group (#501)** follows the screen broadcast's `LIVE ⏸ 👆`: `Chrome ⏸ ↗ 👆 👽 Ana 💭N ✋Bo` (its 👽 is real, `WebcastDriveLinkLabel`; #503 removed the broadcast's placeholder), each button its own
  `RenderedStatusRow` span (`WebFocus`, `WebPointerLink`, `WebDriveLink`), so a click knows which link it copies. The
  codes show on hovering the cell itself, not these (#495).
  👆 and 👽 toggle on left-click (enabling copies their link); right-click only copies. A right-click on the cell copies
  the view-only link (the cast row's rule, #495).
  `↗` calls `Browser.Focus`: the real Chrome sends `Target.activateTarget` for the cast tab, then activates its own
  process by PID (`activateProcess`, macOS JXA → `NSRunningApplication`; by PID because the cast Chrome
  shares its bundle id with the operator's own Chrome). Only `Launch` of a windowed Chrome sets the
  activator, so headless and scripted-pipe tests can never take the operator's focus.
- **Remote driving (#467, `couchtty/console_drive.go`):**
  - **The capability chip.** OFF is plain `👽`, ON idle is amber `👽`, and occupied is amber `👽 Ana`.
    The full name belongs to `WebDriveLink`: left-click turns driving off, revoking driver and requests;
    right-click copies its stable link. Driver text is sanitized and clipped with the same hit span.
  - **The request span.** After `💭N`, `✋Bo` names an asker (`✋2` for more).
    It follows `Cast.Drive().Subscribe()`; active driver identity is only in the capability chip.
  - **The drive menu.** A click on the span opens it: the notes menu's box and gesture router, as a third source,
    titled "Remote driving".
    - On the driver's row, "uploads and downloads off while Ana drives", any click takes control back.
    - On an asker's row, a left click grants and a right click denies.
  - **Revoke.** Ctrl with either button on the span revokes, from any state, with the menu open or closed.
  - **Three links (pair#506).** Right-click on the Chrome cell copies the view-only link (watch), 👆 the pointer link
    (watch, point, notes; `copyWebcastPointerLink`, "takes no notes" on an unrecorded cast), and 👽 the drive link,
    the only one that offers a seat; right-click copies without toggling, and the Codes menu (Alt+click on the cell,
    #495) has a row, code and copy action for each of the three (#495 dropped Alt+right-click). Shift+click is the
    terminal's own selection. A pointer-link viewer counts as a `web view` viewer. `couch --broadcast-list` lists `screen`, `web view` and `web drive` shares
    (`Console.Shares`, `couchmessage.ShareStatus`), the drive one with who drives and how many ask.
  - **Acknowledged actions (#523).** Grant, deny, popup revoke and Ctrl-click revoke
    run outside the terminal input loop in one tracked operation slot. Busy clicks
    ask the operator to wait; no action queue accumulates. Admission keeps the
    clicked capability generation through worker dispatch. Terminal completions
    are fenced by cast and generation, and shutdown joins the work while the
    controller owns recorder/CDP deadlines and cast-close cancellation.
  - **Failures.** A control change that fails says why (`driveFailure`): a race, the cast ended, not recording
    yet, or a failure with its cause.
  - **Notices.** Couch says when someone asks, what a grant turns off ("Uploads and downloads are off until control
    comes back"), and when control is back.
  - **The banner.** The in-page control in the operator's tab names the driver.
  - **Tests.** `webcast.FakeDriven` is the Driven fake: the fake injector as the recorder, `Open`, `HoldBatches`.
    `TestDriveFromCouch` drives it through the cast's real HTTP routes.
- **State machine:** `couchtty/console_webcast.go`, mirroring `console_broadcast.go` (off → starting → running →
  ending, `startClaim` adoption, a bounded shutdown beside the broadcast's).
- **Cast:** `webcast.Cast` = Browser + the viewer `Server` mounted on the shared `broadcast.Host`. Chrome comes
  up first; the link opens in the background (`Update.Link` / `LinkErr`). The cast ends when its Browser does;
  viewers get `end`, then the mount goes.
- **Three links (pair#506):** a cast has a view token, a pointer token (only when recorded: no board, no pointing)
  and a drive token (#467). `Update` carries `Link`, `PointerLink` and `DriveLink`. Each grants what the one
  before it does, and more: view watches; pointer also draws strokes and leaves and resolves notes; drive also offers
  a seat. `castLink` (`webcast/link.go`) is the request's link, from `Server.linkOf` (constant-time token compare),
  and each route asks it: `annotates()` for the board, `drives()` for the drive routes.
  - Board POSTs under the view-only link answer 403 "This link is view-only." *before* admission, so they spend no body
    read, in-flight slot or rate budget.
  - `palette.json` answers `{palette, annotate}`. Under the view-only link the viewer's `mountAnnotate` (`readPalette`)
    still draws others' strokes and notes, read-only: the rail never shows and a note opens without Resolve.
  - A pointer-link stream counts with the view streams in `Server.Viewers()`: only drive-link streams hold a seat.
  - The terminal cast's pointer link is a phase switched on and off, so the web cast keeps its own fixed-link model.
- **Shared Host:** `broadcast.Host` (`host.go`) holds one listener, server and tunnel for every cast, opened by the
  first mount and closed after the last, routing by the token in the first path segment (`Mountable.Owns`). A named
  cloudflared tunnel can be claimed once, so the screen broadcast and the web cast must share it. `couchcmd` builds
  one Host from the `COUCH_BROADCAST_*` tunnel and hands it to both. SSE writing is `broadcast.EventStream`, shared.
- **Events route:** `<token>/events` streams `state` and `end` (the Browser's phase) and, when `CastConfig.Live` is
  set, `rec`: each record line of the recording, a `recording.Hub`'s catch-up first. When the record channel
  closes, the fan-out says why (`Live.Ended`). A viewer dropped as too slow has its stream ended and reconnects for
  a fresh catch-up. Once the fan-out has ended, the viewer keeps its page and the stream goes on with `state` and
  `end`. `?viewer=ID` calls `CastConfig.Presence` for the stream's life, and with a controller holds that remote-driving seat (`Controller.Attach`, #467). `<token>/assets/<hash>` is
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
  - What the recording can't hold (pair#513): a canvas's pixels, a video's frames, a plugin's output (`object`,
    `embed`), a frame whose document wasn't recorded. Their size often comes from what isn't recorded (with scripts
    off a browser draws a `<canvas>` as its empty fallback; a video sized by loaded metadata has none, and Safari
    draws no box), so the renderer sizes each to the operator's live layout box (else its `width`/`height`
    attributes) with a zero-specificity rule in the document's viewer sheet, border-box and with the box's
    `aspect-ratio`, so author `height:auto` keeps the operator's height (`liveSizeRule`, `drawLiveSizes`; author CSS
    still wins). The rule never applies to `[hidden]`; once a layout has arrived, an element without a box (not
    rendered) gets none; and it lives in a live-size sheet per tree (document or shadow root), adopted before the
    page's own sheets. `Renderer.uncaptured()` lists them, and `placeholders.js` marks each over the page box: a dashed outline
    at its drawn box and "canvas: not captured", redrawn on every record, so it follows the operator's size. Canvas
    snapshots are #515. Checked by `TestLiveReplacedElements` (video, iframe, canvas resized live, picture, object,
    a node filled after its insert) and `tests/webcast/placeholders.test.mjs`.
  - Paint holding (#475): as a browser's render-blocking CSS does, the renderer draws a new top document into a
    second frame placed before the first in document order, so the first, keeping the previous paint, covers it (and
    the shell's overlay covers both), until the new document's `<link rel=stylesheet>`s (media matching, not alternate
    or disabled) are bound or have failed, or `HOLD_MS` (500 ms) has passed. `#page` is always the frame with the
    current document. The hold is one state on the frame pair (`null` or `{owner, drawn, cancel}`); a renderer that
    starts a hold while one is in flight (a resync, a seek) takes it over. A renderer dirty outside a record flushes
    itself before the next paint (`onDirty`), unless the shell replaces that. So the recording keeps
    when bytes arrived, and the viewer, live or replaying, never draws an unstyled page the tab never showed.
  - `css.js` rewrites streamed CSS (`url()`s through the asset table, `:hover` to the stand-in class).
  - `viewer.js` is the shell. It lays the page out once at the operator's viewport and scales it to fit (never
    relayout), shows the title and URL as text, and shows Paused and Cast-ended states. The operator's `hover` is
    drawn by putting the stand-in class on the hovered element and its ancestors, and the sampled `ptr` is a dot
    over the frame. A record the model refuses resyncs: the shell reconnects for a fresh catch-up.
  - The frame takes no input; the shell's overlay does (wave 2's notes, #466/#467's remote driving).
  - The contract is on `window.couchViewer`: `canonical`, `ready`, `resolveAsset`, `boxOf`, `nodeAt`, `toPage`
    and `onRecord`; `annotate` and `drive` once mounted.
  - **Remote driving (`drive.js`, #467):**
    - **Seat.** The shell takes a seat before it opens the stream (`drive/hello`, the name from `identityStore`).
      The stream then opens with `?viewer=<seat>`, so it holds the seat, and notes and strokes are authored by the
      seat's id (`identityStore.adopt`). A 404 means no driving: the stream opens without a seat and nothing
      mounts.
    - **Control.** A bar button cycles Ask to drive → Cancel request → Give back control, and a note says who
      drives. Both follow the latest `ctl` record.
    - **Input.** While this viewer drives and the cast is live, a capture layer over the page takes pointer and
      touch (`gestures`):
      - a press that doesn't move 4 px is a click (or a double click); further is a drag: down, moves, up;
      - one finger works like the mouse, and two fingers scroll (wheel acts);
      - a mouse with no button hovers, at most every 50 ms.
    - **Keyboard.** A hidden textarea holds the keyboard focus, so an iPad raises its keyboard. Named keys and
      Ctrl/Meta chords go as key acts (`keyAct`); typing goes as text, and compositions as ime.
    - **Batches.** `batcher` is the one gate to the wire:
      - acts go in order, one request in flight, 40 ms apart;
      - moves and wheels coalesce;
      - coordinates are clamped, text is cut by runes and bytes, and a body stays under 8 KiB;
      - each act carries the segment it was made on, and a batch never mixes segments, so an act from a page that
        has gone is refused rather than played into the new one.

      A refusal drops the queue and shows the server's sentence (a 401 offers Rejoin). `driveView` holds every
      shell state (none, asking, driving, paused, other, lost), and the shell only paints it.
    - **Letting go.** The cast tab, not the shell, ends what a driver holds (`drive.TabInjector` tracks a held
      button and keys). It releases them in these cases:
      - before another driver's input;
      - before input on another page;
      - when the driver's own batch is refused;
      - on every control change, through `drive.Releaser`: `MemController` calls it off its lock and names the
        outgoing driver, so a late release never ends the next driver's press.

      The shell resets its gestures and keys when control ends.
    - An annotation/sketch tool can suspend normal remote input while retaining the driving seat; its mode
      coordinator gates leaving a sketch on the required label acknowledgment.
    - `viewer_drive_live_test.go` drives two headless viewers, one by mouse and one by touch, through a real
      `Server` and `MemController`.
- **Viewer fidelity:** `replay_test.go` replays every fixture in headless Chrome (`WEBCAST_LIVE_CHROME=1`).
  - The drawn page's canonical equals the reducer's at every pin, and a join at any pin draws the same page.
  - Renders are within 3% of `frames/` (measured 0.2–2.3%). The last second before each frame is replayed in real
    time so transitions are in phase. The recorder reports synchronously, so no fixture gets
    observer slack (#472, #474). Any fixture's frame may also be compared 50 ms before its `t`: its pixels are a compositor frame from
    inside the screenshot call. Offsets never leave the frame's own live segment (`recording.FrameTimes`, shared with the
    fidelity tool). A frame whose own times fail is tried where the viewer's paint hold over it releases
    (`releaseAt`). `knownGaps` lists frames that differ for a filed reason, each with a ceiling on its best try, and
    fails once one passes or goes over its ceiling (rec-stress frame 0: an image recorded 68 ms after the screenshot).
  - The harness sizes the drawn page to the segment's viewport, as `viewer.js` does, so a tab switch to a
    different-sized tab compares at that size.
  - `viewer_live_test.go` drives the real page through `Server` and a `Hub` at tablet width: scaling, URL, hover,
    pointer, and `boxOf` against the tab's own layout boxes (305 of 306 on rec-blog).
- **Known transient:** for a moment after a navigation, the title can be Chrome's placeholder (the URL, as in
  Chrome's own tab) until the page reports its `<title>`; it is never the previous page's title (`DocumentChanged`).

## Code lock (#495)

The web cast's links are locked like the terminal broadcast's; see
[Code lock](broadcast.md#code-lock-495) for the gate, the session cookie and
the budget.

- **Two locks.** `StartCast` mints the `view` lock (4 digits) before
  anything else, and the `drive` lock (6 digits,
  `broadcast.DriveCodeDigits`) with the drive token.
  - If the drive lock can't be made, there is no drive link: driving is off,
    and the operator is told why.
  - The locks are attached with `Server.WithLocks` before `Mount`, and they
    end after `Unmount` in `run`, so a viewer still receives the cast's end.
- **Codes and rotation.** `Cast.Codes` and `Cast.RotateCode` are Couch's
  handles (M3); `OnCodeRotated` reports each rotation with its `Gen`.
- **Separate codes.** The view code never opens the drive link: each token
  has its own lock and its own cookie path.
- **Cookie coverage.** The page's iframe is sandboxed with
  `allow-same-origin`, so its recorded assets (`assets/<hash>`) carry the
  session cookie. `TestLiveLockPage` checks this in Chrome.
- **Viewer name.** The lock page asks for it with the code.
  - Every cast page keeps it under `couch.viewerName`, by one rule:
    `validName` in `collab.js` and `broadcast.ViewerName` in Go (the board's
    `MaxNameRunes` is the same bound).
  - The note dialog asks only when no name is stored.
  - Terminal notes carry it as `author`, and Couch's notes menu shows
    `name: text`.
- **Not locked.** Replay share links (#477) are a follow-up (pair#509); their
  `LockFor` is nil.

## Replay (#471)

A recording plays in the same viewer, with the same model and renderer. Only its source differs: records come from disk
instead of the event stream.

- **Source:** `web/replay.js`. `mount` branches once, on the path (`<token>/r/<id>/`): `mountReplay` instead of
  `connect`. Both return the surface `{model, renderer, onRecord}`, so layout, scaling, the pointer and
  `window.couchViewer` are unchanged.
- **Seek rule:** the state at t is the catch-up of the records up to t, reduced. `catchUp` is a port of
  `recording.CatchUp`, which keeps only the latest `hover` and `ptr` of the latest segment. Sketch-bearing prefixes
  additionally derive the current control and sketch history as one validated ephemeral snapshot (see Sketch recording model).
  - **Shared corpus:** `tests/webcast/catchup_corpus.json` reaches every branch of the rule and holds Go's answer at
    every cut. `TestCatchUpCorpus` keeps the answers true to Go and requires every special-cased kind; regenerate them
    with `-update-catchup`. The node test holds the port to the same answers, so a change to Go's rule fails at once
    until the port follows.
  - **Invariant:** node tests also check the format's catch-up invariant at every boundary cut of every fixture: the
    same DOM, assets, `DocSeg`, notes and sketch projection as the whole prefix.
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

  The replay has no `palette.json`, so `mountAnnotate`'s posting UI never mounts there. The shared sketch observer
  still mounts and paints historical accepted ink without enabling ingress or source preparation. Every server of
  the viewer serves its files from one table (`serveViewerFile`), the fidelity harness included: `replay.js` now imports
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

The live viewer owns a replay-latest capability subscription, initially closed.
Disconnect/end revoke local availability; retired EventSources cannot update a
replacement or reload it through a delayed session probe. Annotation reuses the
shared mount's cancellation/held-draft policy and clears its transient stroke
cache on annotation-clear. Drive drops queued acts and stale batch completions;
a new capability generation requires fresh consent. Catch-up control and initial
capability events may arrive in either order; projection retains control evidence
while closed without admitting input. See web/viewer.js, annotate.js, drive.js.


### Existing note editors (#537)

Viewer and isolated operator overlay share the note dialog transition model.
An existing own-note pin opens an editor even with the pen Off; other notes and
annotation-disabled/view-only states open read-only. A remote driver suppresses
operator pins and dialogs independently of annotation state. Watch mode keeps
the page background interactive. Revised pins refresh their text and `edited`
marker without replacing note identity.

The operator's `edit-note` request goes through `OperatorSession` and the ordered
recorder. Its fixed trusted owner can edit only operator-created notes. Unknown
writes retain their request identity: the existing target note does not prove an
edit was accepted. See [the format](webcast-format.md#own-note-revisions-537) and
[session ownership](broadcast.md#own-note-editing-537).

Sketch anchor latency can be qualified with the opt-in `TestSketchAnchorLatencyLive` in
`cmd/internal/webcast/sketch_anchor_latency_live_test.go` (`WEBCAST_MEASURE_SKETCH_ANCHOR=1 WEBCAST_LIVE_CHROME=1`).
It owns synthetic local/tunnel pages and disposable Chrome profiles, measures the actual accumulator token drain
separately from writer acknowledgments, and includes console journal traffic. Raw samples go to owned scratch;
small desktop measurements do not qualify mobile or large-document behavior.
