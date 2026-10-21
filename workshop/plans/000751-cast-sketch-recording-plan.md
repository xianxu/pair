---
collaboration-mode: discussion
---

# Cast-sketch recording implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy). Use @superpowers-subagent-driven-development for bounded tasks after agreeing shared types; use @superpowers-executing-plans for the writer/reducer integration. Steps use checkboxes. This slice has one SDLC close boundary, with no Mx tags.

**Goal:** Record and deterministically replay streaming, pressure-aware, anchored sketches with bounded history, Undo/Redo, labels and driving-derived interruption, without changing existing pointer trails.

**Architecture:** Extend version-1 `events.jsonl` with a focused sketch event union. A pure transition core in Go and a matching JavaScript core own sketch history; recording State, both ordered producers and catch-up integrate that core. New sketch codecs preserve additive fields recursively while leaving existing codecs and DOM canonical hashes intact.

**Tech Stack:** Go, existing strictjson helpers, embedded ES modules, Node tests, JSONL and shared JSON fixtures. No new library or service.

**Authority:** #750 and its operator-approved revision are the product contract. ops:0 authorized implementation after the plan-quality gate without another operator round. This slice implements persistence/model/geometry and ordered writer invariants. #752 owns HTTP/privileged binding, transport receipts and source snapshot preparation, live drawing, capability checks and canvas UI. No live drawing or CLI export is claimed here.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|---|---|---|
| Sketch event union and input bounds | `cmd/internal/webcast/recording/sketch.go` | new |
| Sketch wire preservation | `cmd/internal/webcast/recording/sketch_codec.go` | new |
| SketchState / NextSketch | `cmd/internal/webcast/recording/sketch_state.go` | new |
| Sketch projection in recording State | `cmd/internal/webcast/recording/reduce.go` | modified |
| Sketch history byte budget / terminal reserve | `cmd/internal/webcast/recording/sketch_budget.go` | new |
| Source moment ordering | `cmd/internal/webcast/recording/files.go` | modified |
| Sketch geometry / pressure width | `cmd/internal/webcast/sketch/geometry.go` | new |
| Browser sketch transition / projection | `cmd/internal/webcast/web/sketch-model.js` | new |
| Browser affine point projection | `cmd/internal/webcast/web/annotation-anchor.js` | modified |
| Recording catch-up projection contract | `cmd/internal/webcast/recording/catchup.go`, `cmd/internal/webcast/recording/contract.go` | modified |
| Browser recording projection / catch-up | `cmd/internal/webcast/web/model.js`, `cmd/internal/webcast/web/replay.js` | modified |

One recording owns up to 256 sketch identities; one sketch owns one immutable source moment, author/owner, colour, an ordered operation history and up to 64 stroke identities. Each stroke owns one recorded anchor, base width and ordered chunks of original page-coordinate pressure points. The visible and redo stacks reference stroke identities; they never own second copies of paths. Unknown wire members belong to their original record or nested object, not to mutable authoritative state.

`NextSketch` validates a transition completely before publishing any changes. An implementation may return a small validated delta which State applies to its privately owned storage, rather than cloning every historical point. Incoming chunks are copied once into owned storage; lookup is indexed, append visits only new points, Undo/Redo move stack heads without replaying points. Snapshot/query projections may flatten paths outside the append hot path. Expose read-only projections rather than mutable internal maps. This is the first shared history core for the feature, avoids writer/UI-specific stack rules, and leaves rendering and export as consumers (ARCH-PURE, ARCH-DRY, ARCH-ORDER).

An operation receipt contains operation identity, sequence, claimed canonical digest and actual semantic request identity. An identical retry is a no-op even if its producer timestamp differs; a reused operation/sequence with different fields is rejected even if the attacker supplies the same digest. Per-sketch sequence and chunk counters use nonnegative browser-safe integers, not unrestricted Go integers. The digest excludes writer-assigned `t` and `k`; define its canonical known-field projection once and test Go/JS agreement. Persisted unknown fields survive rewrites but do not grant new command effects. #752 can reconstruct transport status from this bounded history.

`SketchBudget` accounts for exact encoded event bytes including the JSONL newline, used bytes and outstanding per-sketch finalization reservations. It is pure admission arithmetic; integration commits budget only when the corresponding ordered record is accepted, and exact retries consume zero additional bytes. Reserves stay assigned to interrupted unlabeled sketches. No per-record history copy or scan is needed to admit a chunk.

Geometry consumes stored anchors and page points. It uses the existing confidence matcher; it adds no selector language or DOM scan. `ProjectPoint` maps x/y between positive anchor boxes while retaining pressure, and `PressureWidth(base,p) = base * (.25 + .75*p)`. Width scales with the viewer, not anchor aspect ratio. Mouse/touch fallback pressure is 1 at capture in #752; explicit persisted points are triples. A helper may default absent caller pressure to 1, but malformed persisted tuples are not silently repaired.

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| Version-1 event registration / codec dispatch | `cmd/internal/webcast/recording/record.go`, `cmd/internal/webcast/recording/codec.go` | modified | JSONL wire objects |
| Production ordered mirror | `cmd/internal/webcast/recorder/mirror.go` | modified | recorder clock, State.Apply and emitted sink records |
| Reference ordered Writer | `cmd/internal/webcast/board/writer.go` | modified | recording clock, State.Apply and recording sink |
| Catch-up Hub | `cmd/internal/webcast/recording/hub.go`, existing hub tests | modified only if needed | stream subscription and segment compaction |
| Embedded browser asset registration | `cmd/internal/webcast/server.go` | modified | go:embed and public module asset map |
| Cross-language contract runner | `cmd/internal/webcast/replay_corpus_test.go`, `tests/webcast/sketch-model.test.mjs` | modified / new | shared fixture input and Node |

Use the real stateful reference Writer and existing recorder fake browser/sink for producer tests. A bounded collecting sink records accepted lines and can refuse a write; do not substitute call-count mocks for the ordered stream. No new external service is introduced. A sink error after State.Apply is an unknown persistence outcome under the existing writer contract: this slice must not claim rollback or authorize #752 to reuse the operation slot. The model is injected into these thin adapters; UI/network admission remains outside it.

## Recording contract

Keep `couch-webcast`, version 1. Register all eight kinds in FileEvents; retain pointer `Stroke` and its `[x,y,dt]` semantics unchanged:

| Kind | Required known fields beyond t,k and operation metadata |
|---|---|
| sketch-begin | id, author, owner, doc, tab, vp |
| sketch-stroke-begin | sketch, id, at, width |
| sketch-points | sketch, id, chunk, pts |
| sketch-stroke-end | sketch, id |
| sketch-undo | sketch, id |
| sketch-redo | sketch, id |
| sketch-label | sketch, label |
| sketch-interrupt | sketch, reason |

Operation metadata is `operation`, per-sketch `sequence`, and canonical request `digest`, represented consistently in Go/JS. Bound sketch/stroke/operation/tab/author IDs to 128 UTF-8 bytes, rejecting control/format characters; digest is exactly lowercase SHA256 hex64. Owner accepts the existing public hex64 session digest or reserved operator identity; this does not tighten legacy notes. Interrupt reasons use a closed enum. Derive the reserve from these actual encoded bounds, including HTML-sensitive identity characters. Do not borrow unlimited existing arbitrary-string fields for terminal records. Server interruptions have stable operation identities. No metadata is added to the shared Header or existing event kinds.

Known payload validation rejects duplicate keys at every depth; missing/null owned members; wrong types; nonintegral/unsafe counters; tuple lengths other than exactly three for points/four for boxes; NaN/Infinity; pressure outside [0,1]; coordinate magnitudes above 1e7; nonpositive anchor box sizes; invalid author/colour/owner; blank or >1024-UTF8-byte labels. Anchors reuse NoteAt's recorded evidence but retain nested unknown fields. Anchors are facts validated against a frozen source at admission in #752: replay must not require historical DOM nodes that CatchUp intentionally strips.

The codec uses strictjson.SplitObject / EncodeObject with explicit owned field sets, plus focused nested handling for author, viewport and anchor objects (including nested stable/path metadata as appropriate). KnownFields does not flatten embedded Header automatically, so do not accidentally treat `Header` as a wire key. Decode/Encode of new records overlays known fields on unknown raw values. Return the completed encoded bytes directly: passing them through json.Marshal again would compact preserved raw JSON. Keep existing shared-record immutability on Encode. Do not promise new preservation for old known kinds.

Sketch phases are drawing, label_pending, complete, interrupted and cleared. Leaving the tool has no durable event; local UI may say label_pending while the recording still has an unfinished drawing. Label is the durable completion event. There is at most one open stroke per active sketch and one active drawing sketch cast-wide. A new accepted stroke-begin clears redo, even before its first points. Seal only nonempty strokes into the visible stack; preserve explicitly empty/incomplete history without inventing points. Interruption seals accepted partial points and marks truncation; a later label retains the interruption evidence.

Reject points/history edits after completion/interruption/clear. Permit the original owner's valid late label on an interrupted sketch, without requiring live source DOM or current driving. Clear invalidates late labels and all old commands. Required labels cannot be overwritten. Undo/Redo require no open stroke and the named stack top. Repeated end after pause/interruption seal acknowledges the already-sealed stroke without another push.

Validate a ctl transition fully, then interrupt the old sketch before installing a changed holder. Pending-only request/deny/withdrawal must not interrupt. Reset retires the current drawing generation even when driver ID is unchanged. Standalone expiry uses sketch-interrupt; navigation/cast end derive interruption from existing seg/stop. Pause seals accepted partial work and blocks drawing; same-document resume keeps sketch/source identity. Define these contextual effects in the sketch core, not separately in both writers. No separate driving permission state machine is added.

## Bounds and complexity

Retain approved limits: 64 points/chunk, 4096 points/stroke, 16384 points/sketch, 64 strokes/sketch including undone, 256 begins/recording, total sketch history <=32 MiB. The 16 KiB command-body limit belongs to #752 but persisted point chunks must already satisfy the point/count bounds. New drawing and Undo/Redo cannot spend reserved terminal bytes. Required label and canonical terminal records can use only their assigned reservation. Label completion releases unused reserve; interruption keeps the remaining label reserve. Clear/end may retire eligibility and release reservations without deleting historical byte accounting. Reject before mutation and before emission.

Terminal reserve calculation must enumerate the maximum encoded legal writer-owned label record and required stroke-end/interrupt combination, including longest bounded identifiers/digests, safe maximum timestamp/sequence, separators and line endings. Use actual Encode outputs for the max-size fixtures. Round up to a power-of-two/whole-KiB allowance at least 8 KiB; increase beyond 8 KiB if the computed maximum needs it. Existing unknown fields on disk are preserved on rewrites; clients cannot inject unbounded unknown members into generated terminal commands. The total remains 32 MiB, not cap plus reserve.

Source begin ordering extends the single OrderedTime helper: when a preceding record exists, `sketch-begin.t > previous.t`; when previous kind is sketch-begin, `next.t > begin.t`; otherwise preserve current split-file rank rule. Both mirror and reference Writer call this helper. Failed validation must not advance lastKind or State.T. Handle timestamp overflow as refusal, not wraparound. Reading raw files never silently restamps timestamps.

CatchUp retains all sketch records and the ctl history that causally derives their interruptions, alongside existing bare seg/stop and clear barriers. Prefer the simple bounded rule of retaining every ctl when sketch history is present; retain current latest-ctl compaction for recordings without sketches. Go and JS choose identically. Do not invent synthetic snapshot events. Historical catch-up does not need old DOM snapshots to validate anchors. Add sketch projection equality to CheckCatchUp independently of legacy DOM Canonical hashes.

Counted work assertions at N and 2N: append visits N/2N newly added points; identical retry appends zero; Undo/Redo visits zero points and updates one stack operation; recording budget admission does no history scan. Rendering scans and source snapshot allocation are #752's work. Sketch history dies with its recording directory; this slice adds no files, leases or background goroutines (ARCH-CONSTRAINTS, ARCH-FUNERAL).

## State and recovery contract

Actions: begin, append points, end stroke, Undo, Redo, supply label, retry the same operation, query receipt, start a new sketch.

| Internal state / failure mode | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| Active sketch, idle/open stroke | Drawing | append points or end stroke | none after acknowledgment |
| No open stroke after local leave | Label needed | supply label | unsaved label text if client dies |
| Same-document pause | Paused | resume cast, then append | unaccepted samples only |
| Driving handoff/reset, expiry or navigation | Interrupted; partial accepted path retained | supply label; current drawer begins a new sketch | unaccepted samples only |
| Label accepted | Saved | begin another sketch | none |
| Annotation clear | Cleared | begin another sketch | old sketch cannot be relabeled or edited |
| Definite validation/cap refusal | Refused; prior projection intact | end stroke and supply reserved label | refused samples |
| Sink may have accepted, acknowledgment missing | Outcome unknown | query receipt and retry only the same identity/body | local samples outside the accepted record |
| Identity reused with another body | Conflict | reconcile original receipt | conflicting command is never applied |

The pure core owns these transitions; caller UI cannot alter its history. The UI's label_pending display is local until a label record completes the sketch. #752 owns network reconciliation, privilege ingress and visible controls; this slice covers their persisted model and deterministic writer evidence. Process death leaves the accepted JSONL prefix; replay reports unfinished/interrupted evidence rather than guessing a label. No new concurrent jobs are created.

## Architectural check

- ARCH-DRY/PURE/ORDER: one Go transition owner and matching browser port, tested through a shared corpus; both producers reuse ordering/budget helpers. Existing strictjson and anchor matcher retain ownership of their concepts.
- ARCH-PURPOSE: all #751 Done-when surfaces are covered; transport/UI/export remain assigned to #752/#753 rather than omitted recording deliverables.
- ARCH-MOCK: existing stateful sink/browser fakes and real reference Writer model ordered integration; no external service is introduced.
- ARCH-CONSTRAINTS: event hot path is O(new points), constant stack updates, one active drawing sketch, fixed chunk/history/byte limits. Snapshot projection is O(history) off the hot path; CPU/time measurements are not the complexity oracle. No new network traffic or background concurrency in this slice.
- ARCH-SECURE: disk JSONL and future ingress payloads are untrusted; typed strict decoding rejects malformed/truncated known records, duplicate keys and invalid numeric tuples. Unknown new members survive; identities are public digests, never cookies. Tests use temporary local recordings.
- ARCH-FUNERAL: new records remain in existing recording directories and die with recording retention; explicit writer caps bound residue. No sidecar files or live handles are added.
- ARCH-EVOLVE: additive version-1 kinds leave fading pointer semantics and DOM hashes unchanged; nested raw fields survive rewrites, and a frozen legacy-reader test proves compatibility.

## Tasks

### Task 1: Pin wire and compatibility behavior before adding the union

**Files:** Create recording/sketch.go, recording/sketch_codec.go, recording/sketch_test.go and recording/sketch_codec_test.go under `cmd/internal/webcast/`; modify recording/{record.go,codec.go,codec_test.go}.

- [ ] Add failing table tests for every kind, required metadata, pressure triple round-trip, exact tuple lengths, safe integer boundaries, invalid labels/boxes/counters and immutable Encode. Extend the existing exhaustive codec `sample()` for every registered kind.
- [ ] Add nested unknown value fixtures at event, author, viewport and anchor levels. Assert raw value preservation, owned-field edits retain unrelated unknown fields, and duplicate keys/null known fields fail before state mutation.
- [ ] Add a frozen pre-feature decoder/registry compatibility test exercising actual DecodeFile/ReadDir/Validate on a whole recording with all new kinds; old semantics keep DOM and existing pointer strokes. Do not implement a second fake legacy reducer merely to make the test green.
- [ ] Run `go test ./cmd/internal/webcast/recording -run 'Sketch|Encode|Decode|Unknown' -count=1`; confirm new tests fail for absent kinds/preservation rather than malformed fixtures.
- [ ] Implement the focused union and raw-preserving codec through existing registry seams and strictjson. Keep existing event kinds untouched. Run the command again and require PASS.
- [ ] Commit the contract and tests with issue/Co-Authored-By/Agent trailers.

### Task 2: Implement pure sketch transitions and contextual ordering

**Files:** Create recording/sketch_state.go and recording/sketch_state_test.go; modify recording/reduce.go, recording/control.go and their tests; create `tests/webcast/sketch-corpus.json` as the shared sequence contract.

- [ ] Write transition tests for chunks, exact retry, forged same-digest conflict, sequence gaps, empty stroke, partial end, multiple strokes, Undo/Redo, new branch, required label and late metadata completion. Snapshot the complete projection before each refused event and assert unchanged afterward.
- [ ] Add ordered context sequences for ctl request/grant/deny/revoke/release/leave/reset; operator and granted driver begin; same-ID reset retirement; pause/resume; navigation/tab/cast end; clear; expiry; repeat end after implicit seal. Include control-record validation failure while a stroke is open to prove interruption is atomic with ctl acceptance.
- [ ] Build corpus cases with expected error indices and normalized sketch projection for every prefix, not only final state. Include a source anchor whose DOM node no longer exists in later segments.
- [ ] Run `go test ./cmd/internal/webcast/recording -run 'Sketch|Control|Annotation|Pause' -count=1`, observe failures, implement the validated transition core and State integration, rerun to PASS.
- [ ] Add instrumented N/2N append and Undo/Redo tests proving linear new-point work and constant-time history updates. Keep counters test-local or expose narrow test hooks; avoid timing assertions.
- [ ] Commit the pure model/context contract.

### Task 3: Implement pressure geometry and the browser model

**Files:** Create `cmd/internal/webcast/sketch/geometry.go`, geometry_test.go, `cmd/internal/webcast/web/sketch-model.js`, `tests/webcast/sketch-model.test.mjs`, `tests/webcast/sketch-geometry.json`; modify web/{model.js,annotation-anchor.js}, existing anchor tests and server.go asset registration.

- [ ] Write shared geometry vectors for affine translation/scaling, unequal aspect ratios, points outside anchor, clipping-independent original paths, pressure 0/.5/1, fallback pressure and invalid boxes. Both Go and JS must consume the same vectors.
- [ ] Write browser pure-core tests using every prefix of the shared sketch corpus, including exact errors, ctl ordering, clear and pause. Test rejected apply leaves complete Model state unchanged (including t, control and paths).
- [ ] Run `go test ./cmd/internal/webcast/sketch -count=1` and `node --test tests/webcast/sketch-model.test.mjs`; confirm missing-module/behavior failures, then implement minimal geometry and JS core.
- [ ] Use named `export function`/`export const` from the new shared module for compatibility with the operator script assembler; do not introduce export class/default dependencies. Add the file to both go:embed and server asset lookup as soon as Model imports it.
- [ ] Route Model sketch events and context transitions through the core; retain old canonical DOM hashes and pointer store behavior. Test browser-safe limits and algorithmic counters against the Go contract.
- [ ] Rerun targeted Node tests plus `go test ./cmd/internal/webcast -run 'TestNode|OperatorScript' -count=1`; require PASS and commit.

### Task 4: Preserve sketch causality through catch-up and live compaction

**Files:** Modify recording/{catchup.go,contract.go,catchup_test.go}, web/replay.js, tests/webcast/replay.test.mjs, cmd/internal/webcast/replay_corpus_test.go and tests/webcast/catchup_corpus.json; extend existing Hub tests.

- [ ] Add a failing recording where an old sketch is interrupted by a historical ctl handoff, then multiple navigations/resumes occur. Compare whole reduction with catch-up at every prefix, including interrupted late label, clear and active pressure stroke.
- [ ] Add repeated Hub segment compaction and reconnect tests proving retained sketch operation/redo history and ctl causality, without historical DOM dependencies. Recordings without sketches retain current compact ctl behavior.
- [ ] Run `go test ./cmd/internal/webcast/recording ./cmd/internal/webcast -run 'CatchUp|Sketch|Hub' -count=1` and `node --test tests/webcast/replay.test.mjs tests/webcast/sketch-model.test.mjs`; observe failures.
- [ ] Implement matching Go/JS retention, extend CheckCatchUp with normalized sketch equality, regenerate shared catch-up corpus through its existing producer, and rerun to PASS. Do not regenerate legacy DOM goldens to hide semantic drift.
- [ ] Commit catch-up contract and fixtures.

### Task 5: Fence exact source time and reserve terminal capacity in both producers

**Files:** Create recording/sketch_budget.go and sketch_budget_test.go; modify recording/{files.go,order_test.go}, recorder/{mirror.go,note_order_test.go}, board/{writer.go,writer_order_test.go}; add focused sketch producer tests where clearer.

- [ ] Add frozen-clock producer fixtures for event/notes/pointer/begin/DOM records in the same millisecond, including same-file neighbors. Merge emitted split files and assert Reduce(begin.t) includes exactly the source prefix and excludes all later changes. Cover both reference Writer and production mirror.
- [ ] Add refusal-before-begin and refusal-after-begin fixtures proving failed events do not advance ordering; test timestamp overflow refusal.
- [ ] Add pure budget tests around exact cap boundaries, duplicate zero-charge, 256 begins, outstanding interrupted labels and clear/end retirement. Near cap, reject new drawing/Undo/Redo while a maximum escaping-heavy label plus required end/interrupt still fit inside 32 MiB.
- [ ] The adversarial label test uses 1024 `<` bytes (6146 JSON bytes before its record envelope under Go JSON escaping) and alternate escaping-heavy/control-permitted characters. Assert the encoded maximum terminal sequence <= computed reserve and reserve >=8 KiB; do not infer encoded bytes from input length.
- [ ] Run `go test ./cmd/internal/webcast/recording ./cmd/internal/webcast/board ./cmd/internal/webcast/recorder -run 'Sketch|Order|Writer' -count=1`, observe failures, implement shared timestamp/budget logic and integrate each producer once, then rerun to PASS.
- [ ] Verify failure after a sink may have accepted is represented as unknown outcome rather than a guaranteed absence. #752 will own transport status/retry behavior atop this state.
- [ ] Commit producer invariants and measured reserve arithmetic.

### Task 6: Document, verify and close the recording slice

**Files:** Modify atlas/webcast-format.md, atlas/webcast.md (only model/format pointers), atlas/index.md if adding a page, workshop/issues/000751-cast-sketch-recording.md, this durable plan and workshop/lessons.md for concrete review lessons.

- [ ] Document new events, pressure tuples, operation identity, source fencing, ctl-derived interruption, catch-up retention, 32 MiB accounting and finalization reserve. Explain unknown fields on new kinds and old-reader limits: old catch-up must not be used to rewrite sketch history.
- [ ] Run `go test ./cmd/internal/webcast/recording ./cmd/internal/webcast/sketch ./cmd/internal/webcast/board ./cmd/internal/webcast/recorder -count=1`, `go test ./cmd/internal/webcast -run 'TestNode|CatchUpCorpus|OperatorScript' -count=1`, and `node --test tests/webcast/sketch-model.test.mjs tests/webcast/replay.test.mjs`. Require PASS, inspect `git diff --check`, and log actual results.
- [ ] Push the reviewed head and run `sdlc local-check run --name affected`. If known affected graph generation failure still applies, record its specific issue and use full `sdlc local-check run --name suite` without bypassing checks. No live device smoke is claimed for this model-only slice.
- [ ] Close once through `sdlc close --issue 751 --verified '<measured commands and results>'`, fixing its fresh-context review findings. Use the hub-owned project policy rather than editing ops project files. Open/update PR and follow tl-worker landing protocol; report landed PR and model contracts to ops:0. Start #752 from landed main, never this unlanded branch.

## Revisions

### 2026-10-20 — encoded terminal reserve correction approved by ops:0

The parent design's provisional 4 KiB finalization reserve is superseded for this implementation. A 1024-byte valid label can require 6146 JSON bytes before the envelope when HTML-sensitive ASCII is escaped. ops:0 explicitly approved deriving the reserve from worst-case encoded terminal records, including the bounded envelope and stroke-end or interrupt, rounding up to at least 8 KiB, retaining the 32 MiB total cap, and adding the adversarial-label regression. Increase above 8 KiB if the complete maximum sequence requires it. This is approved arithmetic within the design and needs no new operator review.
