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
continue to point to their original sources. Agents externalize issue evidence under the privacy rule below. Terminal
submission remains separate work; only the common Go submission type is shared.

There are two clocks. A note's `t` is milliseconds since the recording began;
its anchor, context, compact element and `before` actions describe that recorded
moment. `crop.captured_at` is the wall-clock time of a screenshot taken during
submission, and `crop.document` identifies the captured document. Boxes use
`[x, y, width, height]` in page CSS pixels. A crop is the note's recorded box
plus a margin, intersected with the current viewport and captured from the actual
cast tab now. `crop.box` is the actual intersection; `clipped_to_viewport: true`
means the requested crop was narrowed. Wholly offscreen notes keep their recorded
context and report `crop.unavailable: "outside viewport"`. Couch never scrolls the
page to collect missing pixels.

Before each crop, Couch observes document-wide animations/transitions for at most
1500 ms, polling every 50 ms and inspecting at most 512 browser animations across at most
20,000 DOM nodes while locating open shadow roots. It
waits for natural completion; it never finishes or overrides animation styles.
`crop.settlement` records `status: "settled"` when those observations finish, or
`status: "unsettled"` with a reason: `timeout`, `paused animation`,
`infinite animation`, `inspection limit`, or `inspection unavailable`. Missing
settlement metadata in older bundles means unknown. Observation covers the top
document and accessible open-shadow animations conservatively, without target
geometry exclusions: children, pseudo-elements, shadows and filters can paint
outside an animated target's border box. Unrelated off-box animations can consume
the wait budget or mark the crop unsettled;
it is not a guarantee about video, canvas, separate frames or future page changes.

Screenshots use the existing viewport (`captureBeyondViewport: false`): the
beyond-viewport CDP path can restart responsive animations even after they finish.
Document, viewport, scroll or pixel-ratio changes during capture refuse preparation.
The PNG remains current evidence even when its animation is explicitly unsettled.

Version-2 bundles store the actual page in `source.page`: its full URL, title and
`provenance: "cdp"`. Every note's `source_document` references `source.document`.
URL/title are read in the same admitted isolated context as the capture, checked
across the text/pixel fences, and bounded to 16 KiB / 8 KiB. Navigation or changed
metadata rejects capture. An unavailable batch claims no page and stays private.
Version-1 bundles remain readable, including their legacy selector fields. New
bundles omit anchor `sel` and box-text `selector`; recording/query selectors and
the historical `at.node` handle remain available. Text-node granularity and
reading order are preserved, including mixed parent/child content.

Each note's `box_text` captures current characters inside its original unpadded
region (or anchor box), alongside the historical anchor text. `visible_text`
contains visible intersecting text in reading order, without repeating parent
and child text. `accessible_names` separately labels aria-label, alt, title,
and role/name entries. Both include bounded `element` tag/class markup instead of CSS selectors. Hidden, zero-size, off-screen,
and aria-hidden content is excluded. Input values are never read; textarea and
editable descendants are omitted from text and derived names for every origin,
while static field labels remain. This does not redact pixels or historical DOM. Both text and names must intersect a
nonempty note/viewport region and the applicable ancestor clipping rectangles.
Rectangular overflow, scrolling, paint containment and positive axis-aligned scaling
are handled in the composed-tree traversal. A subtree with clip-path, masks,
legacy clip, non-axis-aligned/non-positive transforms, rounded overflow clips or
nonzero overflow-clip-margin is omitted and reported as
`truncated: true`, `skipped: ["unsupported clipping"]`. Its PNG remains complete. The PNG is the definitive appearance;
box text preserves characters and tag/classes help locate source.

The `scope` is `top-document-and-open-shadow-roots`. Separate frame documents,
object/embed content and closed shadow interiors are outside that scope.
Intersecting skipped frames or object/embed nodes set `truncated: true` and name
the kind in `skipped`; closed shadow interiors are not generally detectable.
The PNG retains their visible appearance.

Each text group has an 8 KiB budget including its element metadata and labels; the two
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
  "version": 2,
  "privacy": "private",
  "privacy_rule": null,
  "privacy_origin": "https://example.test",
  "how_to_read": "Read the manifest and PNGs; query the recording as described above.",
  "id": "example-1",
  "target": {"scope": "scope", "tag": "thread", "session": "session", "nonce": "launch"},
  "source": {
    "kind": "dom",
    "page": {"url":"https://example.test/settings", "title":"Settings", "provenance":"cdp"},
    "recording_id": "recording-1",
    "recording_path": "/absolute/couch/webcast/recordings/recording-1",
    "tab_id": "tab-1",
    "document": 1
  },
  "notes": [{
    "id": "note-1",
    "source_document": 1,
    "revision": 0,
    "author": {"id": "viewer-1", "name": "Ana", "color": "#ff0000"},
    "text": "Align this button with the heading.",
    "t": 1530,
    "at": {"node": 7, "kind": "element", "path": ["main", "button#save"], "text": "Save", "box": [20, 80, 100, 32]},
    "context": {"node": 7, "landmark": 3, "text": "Settings Save", "box": [10, 20, 300, 120]},
    "box_text": {"region": [20, 80, 100, 32], "visible_text": [{"element": "<button class=\"save\">", "text": "Save"}], "accessible_names": [{"element": "<button class=\"save\">", "kind": "role", "role": "button", "text": "Save"}], "truncated": false},
    "element": "<button id=\"save\">Save</button>",
    "before": ["click Save"],
    "replay": "https://example.invalid/token/r/recording-1/#t=1530",
    "crop": {"file": "crop-0.png", "box": [10, 70, 120, 52], "captured_at": "2026-10-12T18:30:00Z", "document": 1}
  }],
  "delivery": "prepared"
}
```

Preparation allows 30 seconds overall, including each note's settling allowance,
with a 5-second limit per screenshot.
Each crop is bounded to 4 megapixels and 8 MiB of PNG data; a complete bundle is
bounded to 64 MiB. Reading the source recording has a 128 MiB input budget.
Couch runs one submission preparation or resolution job at a time, capturing
the selected notes sequentially. Exceeding a limit produces a visible failure.

Bundles remain until explicitly removed. The store has a 1 GiB admission cap,
including interrupted staging directories; a full store reports its path for
cleanup and never evicts evidence automatically. Directories are private
(`0700`), files are private (`0600`), and replay tokens stay inside the bundle.

## Origin privacy and issue evidence (#526)

The operator may create `<COUCH_STORE_DIR>/webcast/origin-privacy.json`:

```json
{"origins":[{"origin":"http://localhost:4321","privacy":"shareable"}]}
```

Couch never writes this file; agents must not edit it to obtain a sharing grant.
Rules are exact HTTP(S) origins, with lowercase hosts and default ports normalized;
paths (except a root slash), queries, fragments, wildcards, credentials and duplicate
normalized rules are invalid. Each rule chooses `shareable` or `private`. No rule,
missing file or unknown capture identity means private. The file is limited to
64 KiB and 256 rules and is read once per preparation; operator changes apply to
later submissions. Existing prepared bundles retain their original decision.

Invalid, oversized or unreadable policy does not strand the submission: Couch
uses private for all origins, records `privacy_rule: "invalid policy file: <reason>"`,
and posts a notice naming the file. Reasons are safe categories without policy
contents. Valid matching rules are objects such as
`{"origin":"http://localhost:4321","privacy":"shareable"}`; no match is null.
New bundles always record top-level `privacy` and `privacy_rule`. `privacy_origin`
is derived from `source.page.url` and validated against that same captured URL,
not the recording's initial URL. Missing/unavailable identity cannot grant
sharing. Older bundles without privacy fields are private.

When notes become an issue, write evidence as self-contained text: a page URL
without secrets, element/classes, the operator request, appropriate text evidence
and one line describing what the crop demonstrates. For private evidence,
summarize instead of quoting box text, and attach no crops. A shareable bundle
allows only relevant crops and only when the repository's asset policy permits
its issue asset folder (ariadne#337); absent or unclear repository policy means
text only. This is an agent rule, repeated in the shipped Couch skill and
`how_to_read`; Couch does not copy assets or create issues. A sharing rule never
requires publishing unrelated sensitive content.

Recordings stay local until operator deletion, may expire, and must never be
uploaded or required to understand a durable issue. Replay links are investigation
aids, not a substitute for issue text or permission to publish private evidence.

## Delivery and recovery

The manifest records `prepared → delivering → dispatched → resolved`.
`delivering` is durable intent written before any paste. Idempotent retries confirm directory synchronization again
before reporting success after a failed publication sync. It does **not** prove
that the agent received the message. `dispatched` records the draft transaction's
successful send. Acknowledgment then resolves only the submitted notes whose
recording, document and revision still match. New notes, reopened notes and
notes on a replacement page remain untouched.

The draft status line immediately shows preparation, including “waiting for the
page to settle, crop 2/3”, capture and bundle publication, then “sending to
pair:N…” and “sent” or the failure reason. While that operation is active, only
its initiating buffer is read-only. Another chord reports “already sending…”
without replacing the operation or extending its deadline.

The operation has an absolute 36-second client budget; preparation retains its
30-second server timeout and each RPC its 4-second cap. DOM-only subprocesses
also consume the remaining budget. Success releases the buffer before the normal
clear; every failure or timeout restores editing and keeps the draft. Terminal
status remains until the draft changes or another submission starts.

After completion or failure, Ctrl+Shift+Alt+Return starts a new submission from
the current draft with a new ID and fresh evidence. A changed view prevents its
older preparation from sending. Late completions cannot clear a newer draft;
a late confirmed dispatch can still acknowledge its own notes. Duplicate
messages are acceptable after an uncertain paste. No original text needs to be
reconstructed and no discard key is required.

A new stored bundle marks older same-target `prepared` or `delivering` bundles
`superseded: true` before publishing its own directory. Their files and delivery
states remain intact: superseded does not mean delivered. A partial publication
can leave superseded evidence without a successor, which does not block another
fresh attempt. Already-published same-ID retries confirm only their own
publication and cannot supersede a newer bundle. Superseded bundles cannot begin
another send, but a genuine late acknowledgment can still resolve their captured
note versions idempotently. Store scans ignore foreign files such as Finder's
`.DS_Store`; valid submission directories retain strict validation.

Successful sends clear only their matching draft. Acknowledgments retry
independently of newer drafts (at most 64 active receipts, each with 360 retries
at 100 ms intervals and a 4 s RPC timeout). Busy or failed note resolution can
resume through either `ack` or `status`; the server retains one background worker.
A busy preparation refusal leaves the draft editable for another fresh attempt.

Old authored-text freezes existed only in Neovim memory. Starting a fresh draft
process releases them while preserving the edited draft file. After a process
restart, new submissions supersede old pending evidence instead of being blocked
by it. Acknowledgment receipts themselves are not restored automatically; if
confirmed delivery still needs resolution, inspect the manifest and use the
existing exact-ID commands from the same live Couch thread:

```sh
couch --broadcast-web-submit status example-1
couch --broadcast-web-submit ack example-1
```

Only acknowledge an ID whose send was confirmed. `status` resumes resolution
only for an already `dispatched` manifest; it never infers delivery from silence.
The client reports the exact acknowledgment command when its retry budget is
exhausted. These operations send no agent text and retain the immutable target
guard; replacement launch identities cannot acknowledge an old bundle. Resume
a paused cast before retrying resolution. All evidence, including superseded
bundles, remains under the existing store caps and explicit local cleanup.

## Code map

| Responsibility | Implementation |
| --- | --- |
| Pure origin policy and decision validation | `cmd/internal/submission/privacy.go` |
| Operator policy file reader and cast wiring | `cmd/internal/couchcmd/privacy.go`, `webcast.go` |
| Shared schema, limits, atomic store and delivery transitions | `cmd/internal/submission/` |
| Recorded context and one crop calculation per note | `cmd/internal/webcast/submission/build.go` |
| Frozen open-note selection and revision-guarded resolution | `cmd/internal/webcast/board/submission.go` |
| Current CDP screenshots and document checks | `cmd/internal/webcast/recorder/submission.go` |
| Viewport-preserving capture and bounded animation observations | `cmd/internal/webcast/recorder/capture_ready.go`, `capture_ready.js` |
| Recording inputs and crop assembly | `cmd/internal/webcast/cast_submission.go` |
| Background preparation, replay sharing and acknowledgment | `cmd/internal/couchtty/console_submission.go` |
| Local CLI/RPC and current caller validation | `cmd/internal/couchcmd/submission.go`, `cmd/internal/couchmessage/protocol.go` |
| Bounded draft operation, progress UI, pinned pane and authored send transaction | `nvim/dom_submission_state.lua`, `nvim/dom_submission.lua`, `nvim/dom_transport.lua`, `nvim/submission.lua`, `nvim/init.lua` |

For source recording queries, see [the recording format](webcast-format.md#agent-queries-462).
Replay sharing is mapped in [broadcast-web](webcast.md#replay-471).
