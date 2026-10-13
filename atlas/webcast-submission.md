# Send cast notes to the agent (#483)

Press **Shift+Ctrl+Alt+Return** in the draft Neovim, in normal or insert mode,
to send the draft text and all open notes on the cast tab's current document to
the agent in that view. Each note carries focused recorded context, a replay
link, current box text, and its own current PNG crop. Overlapping notes still get separate crops.
The selection is frozen when Couch admits the request; notes added afterward
wait for the next submission.

Submitted notes become resolved: submitting transfers ownership of the notes to
the agent; whatever follows (a pensive, an issue, a fix) is the agent's.

The draft is optional: an empty or comment-only draft can send notes. With no
notes, nonempty text follows ordinary draft submission; with neither, the chord
does nothing. Ordinary Alt+Return keeps its existing behavior. A paused cast
refuses crop capture; resume it before submitting. Pause also blocks note-state
writes, so resume before retrying pending note resolution.

## Evidence and storage

Couch writes the complete bundle before sending anything:

```text
<COUCH_STORE_DIR>/webcast/submissions/<id>/
  submission.json
  crop-0.png
  crop-1.png
```

The agent receives the draft text, absolute manifest path, and the manifest's `how_to_read` guide. The guide names the recording path and query commands. Pair's
authored log retains the original draft; generated evidence references belong
to the outgoing prompt. Crops are files, not clipboard attachments. The folder
is portable, with relative crop paths; its recording and replay references
continue to point to their original sources. Ticket asset copying and terminal
submission are separate work; only the common Go submission type is shared.

There are two clocks. A note's `t` is milliseconds since the recording began;
its anchor, context, compact element and `before` actions describe that recorded
moment. `crop.captured_at` is the wall-clock time of a screenshot taken during
submission, and `crop.document` identifies the captured document. Boxes use
`[x, y, width, height]` in page CSS pixels. A crop is the note's recorded box
plus a margin, captured from the actual cast tab now.

Each note's `box_text` captures current characters inside its original unpadded
region (or anchor box), alongside the historical anchor text. `visible_text`
contains visible intersecting text in reading order, without repeating parent
and child text. `accessible_names` separately labels aria-label, alt, title,
and role/name entries. Both include selectors. Hidden, zero-size, off-screen,
and aria-hidden content is excluded. Both text and names must intersect a
nonempty note/viewport region and the applicable ancestor clipping rectangles.
Rectangular overflow, scrolling, paint containment and positive axis-aligned scaling
are handled in the composed-tree traversal. A subtree with clip-path, masks,
legacy clip, non-axis-aligned/non-positive transforms, rounded overflow clips or
nonzero overflow-clip-margin is omitted and reported as
`truncated: true`, `skipped: ["unsupported clipping"]`. Its PNG remains complete. The PNG is the definitive appearance;
box text preserves characters and selectors locate elements.

The `scope` is `top-document-and-open-shadow-roots`. Separate frame documents,
object/embed content and closed shadow interiors are outside that scope.
Intersecting skipped frames or object/embed nodes set `truncated: true` and name
the kind in `skipped`; closed shadow interiors are not generally detectable.
The PNG retains their visible appearance.

Each text group has an 8 KiB budget including its selectors and labels; the two
groups share a 256-entry limit. Traversal stops at 20,000 nodes and reports
`truncated: true` when evidence reaches a limit. Box text belongs to the same
capture document and interval as its PNG, not the note's historical moment.

Read `submission.json`, then its relative `crop-N.png` files. To explore the
recording named in `source.recording_path`, start with
`couch --broadcast-web-query DIR notes`. Use `dom --at <t> --sel @N` for a
recorded element and `events --from <t-10000> --to <t>` for preceding actions;
`link --at <t>` produces a replay link. Replace DIR with the quoted recording
path. Run `couch --skill` for the full query guide. These pointers are also
included in `how_to_read` and the delivered text.

If the selected document has already gone before capture, the note retains its
recorded context and has `crop: {"unavailable":"page changed"}` instead of a
file. Missing recorded elements have explicit unavailable context and retain
the anchor text. Navigation or tab replacement during capture fails preparation;
it does not silently drop selected notes or publish a partial bundle.
`Recorder.CaptureCrops` uses the document-scoped `CaptureScreenshot` helper (#494):
all injected Couch UI is hidden for each PNG and restored on success or failure.
Missing suppression confirmation fails preparation without publishing the pixels.

This abbreviated one-note example shows the version-1 manifest fields. The
target pins the calling thread and launch incarnation; `revision` pins the
note generation used by acknowledgment.

```json
{
  "version": 1,
  "how_to_read": "Read the manifest and PNGs; query the recording as described above.",
  "id": "example-1",
  "target": {"scope": "scope", "tag": "thread", "session": "session", "nonce": "launch"},
  "source": {
    "kind": "dom",
    "recording_id": "recording-1",
    "recording_path": "/absolute/couch/webcast/recordings/recording-1",
    "tab_id": "tab-1",
    "document": 1
  },
  "notes": [{
    "id": "note-1",
    "revision": 0,
    "author": {"id": "viewer-1", "name": "Ana", "color": "#ff0000"},
    "text": "Align this button with the heading.",
    "t": 1530,
    "at": {"node": 7, "kind": "element", "path": ["main", "button#save"], "sel": "#save", "text": "Save", "box": [20, 80, 100, 32]},
    "context": {"node": 7, "landmark": 3, "text": "Settings Save", "box": [10, 20, 300, 120]},
    "box_text": {"region": [20, 80, 100, 32], "visible_text": [{"selector": "#save", "text": "Save"}], "accessible_names": [{"selector": "#save", "kind": "role", "role": "button", "text": "Save"}], "truncated": false},
    "element": "<button id=\"save\">Save</button>",
    "before": ["click Save"],
    "replay": "https://example.invalid/token/r/recording-1/#t=1530",
    "crop": {"file": "crop-0.png", "box": [10, 70, 120, 52], "captured_at": "2026-10-12T18:30:00Z", "document": 1}
  }],
  "delivery": "prepared"
}
```

Preparation allows 30 seconds overall, with a 5-second limit per screenshot.
Each crop is bounded to 4 megapixels and 8 MiB of PNG data; a complete bundle is
bounded to 64 MiB. Reading the source recording has a 128 MiB input budget.
Couch runs one submission preparation or resolution job at a time, capturing
the selected notes sequentially. Exceeding a limit produces a visible failure.

Bundles remain until explicitly removed. The store has a 1 GiB admission cap,
including interrupted staging directories; a full store reports its path for
cleanup and never evicts evidence automatically. Directories are private
(`0700`), files are private (`0600`), and replay tokens stay inside the bundle.

## Delivery and recovery

The manifest records `prepared → delivering → dispatched → resolved`.
`delivering` is durable intent written before any paste. Idempotent retries confirm directory synchronization again
before reporting success after a failed publication sync. It does **not** prove
that the agent received the message. `dispatched` records the draft transaction's
successful send. Acknowledgment then resolves only the submitted notes whose
recording, document and revision still match. New notes, reopened notes and
notes on a replacement page remain untouched.

Preparation or send failure preserves the draft and notes. Changing the draft
body or view while preparing prevents delivery; return to the original draft
to retry its frozen attempt. The same attempt retains its bundle, outgoing
text and recipient, including after a body was pasted but submit failed. An
indeterminate body write requires inspecting the agent composer before any
further send.

After a successful send, the draft clears normally even if acknowledgment
fails. Press the chord again to retry **only acknowledgment**; it does not
resend the old prompt or consume the next draft. Resume a paused cast first.

After a Neovim restart, an unfinished bundle blocks a new preparation for the
same target and reports its manifest path. Inspect the agent conversation and
composer; no missing response is treated as permission to resend. From a shell
in the same live Couch thread, inspect the exact ID:

```sh
couch --broadcast-web-submit status example-1
```

Only after confirming that the message was sent, acknowledge that same ID:

```sh
couch --broadcast-web-submit ack example-1
```

Both commands return JSON. Repeating `ack` can finish resolution after a Couch
restart and never sends agent text. A replacement launch identity refuses the
old bundle. An unsent `delivering` bundle needs manual reconciliation of the
composer and preserved evidence before removing its preparation; there is no
automatic reset or resend command.

## Code map

| Responsibility | Implementation |
| --- | --- |
| Shared schema, limits, atomic store and delivery transitions | `cmd/internal/submission/` |
| Recorded context and one crop calculation per note | `cmd/internal/webcast/submission/build.go` |
| Frozen open-note selection and revision-guarded resolution | `cmd/internal/webcast/board/submission.go` |
| Current CDP screenshots and document checks | `cmd/internal/webcast/recorder/submission.go` |
| Recording inputs and crop assembly | `cmd/internal/webcast/cast_submission.go` |
| Background preparation, replay sharing and acknowledgment | `cmd/internal/couchtty/console_submission.go` |
| Local CLI/RPC and current caller validation | `cmd/internal/couchcmd/submission.go`, `cmd/internal/couchmessage/protocol.go` |
| Draft attempt, pinned pane and authored send transaction | `nvim/dom_submission.lua`, `nvim/dom_transport.lua`, `nvim/submission.lua` |

For source recording queries, see [the recording format](webcast-format.md#agent-queries-462).
Replay sharing is mapped in [broadcast-web](webcast.md#replay-471).
