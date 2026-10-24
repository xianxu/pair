# Send cast notes and sketches to the agent (#483, #753)

Press **Shift+Ctrl+Alt+Return** in the draft Neovim, in normal or insert mode,
to send the draft text and eligible saved notes and completed sketches actually
visible on the current console to the agent in that view. Each selected note
carries focused recorded context, a replay link, draw-time box text, and its
saved PNG crop when available. Overlapping notes retain separate captures;
submission never recaptures a changed page for those crops. Off-view notes stay
reachable as **not on this view** and wait for a later visible selection.
The acknowledged visible selection is frozen when Couch admits the request;
new or deferred feedback waits for the next submission.

Confirmed submissions resolve their selected note revisions: acknowledgment
transfers ownership of those notes to the agent; whatever follows (a pensive, an issue, a fix) is the agent's.

The draft is optional: an empty or comment-only draft can send notes or sketches.
With neither kind of evidence, nonempty text follows ordinary draft submission; with no text or evidence, the chord
does nothing. Ordinary Alt+Return keeps its existing behavior. A paused cast
refuses new draw-time captures and note-state writes; resume before drawing a
new note or retrying pending note resolution.

## Completed sketch evidence (#753)

A shape means one finished, labeled sketch, including its accepted visible
strokes. Labeled interrupted sketches retain their interruption/truncation
facts. Unfinished, cleared and entirely undone sketches remain queryable but
are not automatically submitted.

The private operator Chrome UI shows **Pending feedback**, with both the sketch
and annotation counts. Indexed pressure-ink thumbnails and attached sketch notes
show the chosen shapes before the existing draft chord freezes the acknowledged
selection. Only actually painted, finished unsent sketches and placed open-note
revisions are eligible. At most eight sketches are selected in creation order;
other feedback remains visible as **N more sketches** and **N more annotations**.
The eight limit applies only to sketches. Annotation crop cards in this mixed
panel are the separate #805 follow-up.

A stale visible selection, changed note revision, sketch version, mask or private
host refuses preparation and refreshes the preview; draft and ink remain intact.
A held drawing scene, open editor, incomplete paint or partial shape placement
cannot establish a visible selection. There is no browser
send button or viewer agent-dispatch route. Remaining shapes wait for the next
explicit submission.

Confirmed dispatched/resolved manifests also derive a **live display mask**:
exact sketch sequence/digest and note revision disappear from console and viewer
painting, while unsent or newer versions remain. The existing preview worker
loads this projection even without a private console host; full masks travel in
the console OperatorView and the viewer's live `feedback` stream, including
reconnect. This changes no recorded vectors or history, and historical replay
does not consume the mask. Existing conditional note resolution stays separate.
The projection dies with the cast and is rederived from confirmed Store evidence;
there is no acknowledgment sidecar. Its full wire image uses the existing 64 MiB
bundle envelope, with atomic refusal rather than truncation. Historical notes
are not limited by the per-page admission count.

New version-3 manifests retain two typed lists, `notes` and `sketches`, and add
one `sketch_page` referencing **sketch-page.png** with provenance
**replay-rendered-current-view**. At the submission chord, the current recorded
prefix and completed console paint are pinned together. The image reconstructs
that viewport with the selected ink and numbered badges matching preview cards
and sketch `index` values. `sketch_page.current` retains the current projections:
actual painted target locators, boxes and clips. Pointer and private tool UI are
absent. Missing, ambiguous, hidden or partially unprojectable selected paths
refuse the whole batch. Unsupported clipping refuses preparation; ordinary
pressure fringe clipped at the viewport edge is declared explicitly. The replay
image can still differ from live browser pixels in uncaptured visuals and
animation phase.

Each sketch retains its original URL/title and `recorded-dom` begin source,
author/color, note, accepted sequence/digest, phase/reason, bounded literal
element IDs and `data-*` attributes. Original coordinates, pressure and anchors
remain unchanged in `paths`; `finish` records that selected version's completion
context. The root page contains the full-viewport region, scale, dimensions,
clipping and limitations. Reflow projection changes only painted placement.
Missing recorded assets, uncaptured visuals and animation-phase differences
remain explicit limitations.

Legacy shared bundles with replay-rendered finish-time pages and individual
`image`/`sketch-N.png` plans remain readable and immutable. New shared items omit `image`; older decoders requiring
it refuse instead of acknowledging only part of a new submission. The new
codec preserves unknown nested extensions through delivery rewrites while
rejecting mixed or incomplete representations.

`source.record_index` fixes the ordered begin prefix and `history_record_index`
fixes the selected later history. Both are zero-based inclusive indices, so
later changes at the same timestamp cannot replace the selected source/history.
A timestamp-only DOM query may include later same-time records. Each original
anchor retains its recording node handle; bounded ancestor and nearby-element
facts are literal declarations, never inferred application meanings or proof
of current live-node identity. Ordinary notes use the same declaration extractor
at their recorded context or saved-capture moment.

Query locally with `couch --broadcast-web-query DIR sketches [--sketch ID]
[--at T] [--json]`. Here `--at` limits accepted history while source remains at
begin. `sketch-image --sketch ID --out DIR [--at T]` exports the image and
transform/provenance JSON together to an explicit new private directory; it
refuses existing output and paths inside the source recording. Exporting sends
nothing. Both local tools use already-read recording input; single-sketch export keeps
its original begin-page crop. Submission uses the separate current-view full-viewport plan with the same replay
session, pressure painter and cleanup machinery.

The existing private bundle store publishes all images before delivery. Confirmed
`dispatched`/`resolved` manifests suppress only those exact sketch versions from
future selections across targets; no recording event, separate acknowledgment
file or ink clear is added. Unknown delivery does not establish that the agent
received the bundle. A later label/history version remains eligible after an
older acknowledgment. Each note and sketch has its own original-origin privacy
decision. The shared image additionally carries a conservative full-page decision
covering its base URL and every painted sketch; a private contribution makes
the composite private. Replay rendering grants no publication permission.

## Draw, capture and save (#538)

In the viewer or operator overlay, releasing a note box starts a private server
capture and immediately opens its text editor. The editor shows `Capturing…`;
typing remains available while Save waits. A confirmed timestamp shows `✓ Captured`.
Save attaches the text to that capture token and closes only after the note writer
acknowledges it. Pending writer responses are checked by status; they do not prove
that a note exists. A rejected write keeps the text editable for another Save.
An uncertain write retains its request identity until reconciliation proves its
outcome; edited text is never silently substituted into that request.

Capture failures show the reason and Retry capture. Retry preserves authored text
and prepares a new capture. Capacity pressure is different: `crop unavailable:
capture busy` and `capture store full: submit or resolve notes to free space`
produce a signed unavailable-evidence draft, so the note can still be saved.
Cancel, a new note gesture or navigation retires the old draft; late answers cannot
reopen it. Cancelling after an acknowledged Save cannot delete that saved note.

`prepare-note`, `draft-status`, `save-note` and `cancel-draft` are the viewer's POST
routes and the operator adapter's corresponding operations. Each retains the
originating annotation generation. Tokens come from the server; request IDs merely
correlate operations. The shared browser controller polls no faster than once per
second, bounds each RPC to 4 seconds and each operation to 36 seconds. Save alone
briefly makes its editor read-only while its acknowledgment is outstanding; every
failure releases it. These optional dialog hooks do not change terminal notes.

Live pins follow a connected anchor or a confident unique replacement on the same
route, projecting the original region onto its new box. Removed, hidden, clipped,
offscreen, conflicting or ambiguous anchors show no box. The shared list retains
all open notes, including notes on earlier documents, labelled **not on this view**
when placement is unavailable. Its yellow floating panel can be minimized to
Notes (N), moved by its header/button or arrow keys, and restored at that same
session-local position. The list scrolls inside its fixed height. Presentation
changes do not hide anchored feedback or resolve selected revisions. Placed
sketches expose Add note through the existing unfinished-label flow and read-only
View note for completed labels. Pending feedback remains the separate exact
submission preview; its acknowledgment and stale-selection checks are unchanged.
Weak/common text cannot select a replacement;
truncated searches cannot claim uniqueness. Operator originals arrive as actual DOM
objects through a bounded, document-scoped recorder/CDP bridge; weak references
expire with the last open note or document. A mutable path never grants original
identity. Frame replacement requires frame-route
evidence that is not currently recorded, so it stays unavailable. Historical
replay retains recorded placement. Moving a live pin never changes its saved pixels.

The private draw-time store is outside served recording directories, at the recording
parent path with `.note-captures` appended. Drafts expire after 10 minutes; the store
admits at most 128 records and 256 MiB, with at most 8 MiB per PNG. Saved-note evidence
is protected until note resolution or confirmed transfer into a submission bundle;
capacity pressure never silently evicts another open note's pixels. The capture
service owns cleanup and retains durable uncertain write/publication intent.

## Evidence and storage

Couch writes the complete bundle before sending anything:

```text
<COUCH_STORE_DIR>/webcast/submissions/<id>/
  submission.json
  crop-0.png
  crop-1.png
```

The agent receives the draft text, selected and deferred sketch and annotation counts when available,
absolute manifest path, and the manifest's `how_to_read` guide. The guide names the recording path and query commands. Pair's
authored log retains the original draft; generated evidence references belong
to the outgoing prompt. Crops are files, not clipboard attachments. The folder
is portable, with relative crop paths; its recording and replay references
continue to point to their original sources. Agents externalize issue evidence under the privacy rule below. Terminal
submission remains separate work; only the common Go submission type is shared.

There are two clocks. A note's `t` is milliseconds since the recording began;
its anchor, context, compact element and `before` actions describe that recorded
moment. `crop.captured_at` is the wall-clock time of the original draw-time capture,
and `crop.document` identifies that captured document. Boxes use
`[x, y, width, height]` in page CSS pixels. A crop is the note's drawn region,
intersected with the viewport when drawn and captured from the
actual cast tab then. `crop.box` is the actual intersection; `clipped_to_viewport: true`
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
The saved PNG remains draw-time evidence even when its animation is explicitly unsettled.

Version-3 bundles put the actual captured page on each note as `page`: its full
URL, title and `provenance: "cdp"`, beside `capture_id`, `source_document` and that
note's privacy decision. A bundle may span documents; the top-level source document
is the submission context, not a replacement for each note's capture identity.
URL/title are read in the same admitted isolated context as the capture, checked
across the text/pixel fences, and bounded to 16 KiB / 8 KiB. Navigation or changed
metadata rejects capture. An unavailable note capture claims no page and stays private.
Version-1 and version-2 bundles remain readable; version 2 keeps its legacy
batch identity in `source.page`, and version 1 may retain selector fields. New
bundles omit anchor `sel` and box-text `selector`; recording/query selectors and
the historical `at.node` handle remain available. Text-node granularity and
reading order are preserved, including mixed parent/child content.

Each note's `box_text` preserves draw-time characters inside its original unpadded
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
draw-time capture document and interval as its PNG. It is not refreshed at submission.

Read `submission.json`, then its relative `crop-N.png` files. To explore the
recording named in `source.recording_path`, start with
`couch --broadcast-web-query DIR notes`. Use `dom --at <t> --sel @N` for a
recorded element and `events --from <t-10000> --to <t>` for preceding actions;
`link --at <t>` produces a replay link. Replace DIR with the quoted recording
path. Run `couch --skill` for the full query guide. These pointers are also
included in `how_to_read` and the delivered text.

A later navigation, tab switch, hidden anchor or removed element does not change
saved evidence. Submission reads the stored capture and validates its original
document/region; legacy notes or unavailable captures keep recorded context and
an explicit `crop.unavailable` reason instead of taking fresh pixels. Navigation
during the draw-time capture fails that preparation. `CaptureScreenshot` hides
injected Couch UI and restores it on success or failure; missing suppression
confirmation refuses the pixels.

This abbreviated version-3 example shows one note. The target pins the calling
thread and launch incarnation; `revision` pins the note generation acknowledged.

```json
{
  "version": 3,
  "privacy": "private",
  "privacy_rule": null,
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
    "capture_id": "capture-1",
    "page": {"url":"https://example.test/settings", "title":"Settings", "provenance":"cdp"},
    "privacy": "private",
    "privacy_rule": null,
    "privacy_origin": "https://example.test",
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
    "crop": {"file": "crop-0.png", "box": [20, 80, 100, 32], "captured_at": "2026-10-12T18:30:00Z", "document": 1}
  }],
  "delivery": "prepared"
}
```

Draw-time capture has a 36-second service budget, including its settling
allowance, with a 5-second limit per screenshot. Submission preparation retains
its 30-second budget for assembling stored evidence and publishing the bundle.
Each crop is bounded to 4 megapixels and 8 MiB of PNG data; a complete bundle is
bounded to 64 MiB. Reading the source recording has a 128 MiB input budget.
Couch runs one submission preparation or resolution job at a time. Draw-time
captures use the separate bounded capture queue. Exceeding a limit produces a visible failure.

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
Version 3 records `privacy`, `privacy_rule` and any `privacy_origin` on each
note, derived from and validated against that note's captured `page.url`. The
top-level privacy value is a conservative summary: shareable only when every
note is shareable. It cannot grant permission for an individual private note.
Version 2 retains its policy decision against `source.page.url`. Missing/unavailable identity cannot grant
sharing. Older bundles without privacy fields are private.

When notes become an issue, write evidence as self-contained text: a page URL
without secrets, element/classes, the operator request, appropriate text evidence
and one line describing what the crop demonstrates. For each private note or sketch,
summarize sensitive text and attach no crops or sketch images. Explicitly shareable
note evidence allows only relevant crops and only when the repository's asset policy permits
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
recording, original note identity and revision still match, even after a page
change. New or changed note revisions remain untouched.

The draft status line immediately shows submission preparation and bundle
publication, then “sending to pair:N…” and “sent” or the failure reason. While that operation is active, only
its initiating buffer is read-only. Navigation, queue, delete, reset and alternate
send actions on that draft are blocked before changing slot identity or saved
content. Another chord reports “already sending…” without replacing the operation
or extending its deadline. These actions resume when ownership is released.

The operation has an absolute 36-second client budget; preparation retains its
30-second server timeout and each RPC its 4-second cap. DOM-only subprocesses
also consume the remaining budget. Success releases the buffer before the normal
clear; every failure or timeout restores editing and keeps the draft. Terminal
status remains until the draft changes or another submission starts.

After completion or failure, Ctrl+Shift+Alt+Return starts a new submission from
the current draft with a new ID and the saved evidence of its selected notes. A changed view prevents its
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
| Private draw-time capture lifecycle, persistence and ownership transfer | `cmd/internal/webcast/notedraft/` |
| Prepared note capture and acknowledged writes | `cmd/internal/webcast/cast_notedraft.go`, `chrome_prepared.go`, `recorder/prepared_note.go` |
| Shared browser capture workflow and live placement | `web/capture-draft.js`, `web/annotation-placement.js`, `web/annotation-anchor.js` |
| Current CDP screenshots and document checks at draw time | `cmd/internal/webcast/recorder/submission.go` |
| Viewport-preserving capture and bounded animation observations | `cmd/internal/webcast/recorder/capture_ready.go`, `capture_ready.js` |
| Recording inputs and crop assembly | `cmd/internal/webcast/cast_submission.go` |
| Background preparation, replay sharing and acknowledgment | `cmd/internal/couchtty/console_submission.go` |
| Local CLI/RPC and current caller validation | `cmd/internal/couchcmd/submission.go`, `cmd/internal/couchmessage/protocol.go` |
| Bounded draft operation, progress UI, pinned pane and authored send transaction | `nvim/dom_submission_state.lua`, `nvim/dom_submission.lua`, `nvim/dom_transport.lua`, `nvim/submission.lua`, `nvim/init.lua` |

For source recording queries, see [the recording format](webcast-format.md#agent-queries-462).
Replay sharing is mapped in [broadcast-web](webcast.md#replay-471).
