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

## Verification strategies (governing PQ-2 revision)

| Risky function | Adversarial input class and mechanical guard |
|---|---|
| Decode / Encode / SketchCanonical | Fuzz nested JSON, duplicate/null/missing owned members and malformed numeric tuples; preserve raw unknown values, owned edits, byte-stable semantic identity and caller immutability. Frozen legacy registry runs real DecodeFile/ReadDir/Validate and keeps old DOM/pointer meaning. |
| NextSketch / State.Apply | Generate legal histories then mutate one operation, counter or context; reject with identical complete before/after state, preserve accepted prefixes, and count only new-point visits at N/2N with zero path visits for Undo/Redo/retries. |
| nextSketch / Model.apply | Consume every prefix of the Go-generated shared corpus, including error prefixes; require identical normalized projection and refusal atomicity. Canonical bytes and asynchronous digest generation agree on shared fixtures. |
| ProjectPoint / PressureWidth | Shared affine/pressure vectors and invalid numeric domains; preserve original triples, return invalid explicitly, and keep width independent of anchor aspect ratio. |
| CatchUp / snapshotSketchState / restoreSketchState | Repeated compaction and reconnect over long control chatter with fixed sketch history; compare complete projection at every prefix, bounded retained snapshot, exact operation/redo authority and no required historical DOM. Malformed bootstrap snapshots refuse before mutation. |
| OrderedTime | Frozen clocks, all file-rank pairs and adjacent source begins, plus near-overflow inputs; merged source at begin includes exactly the preceding prefix, and refused events advance neither time nor kind. Both real producers use the helper. |
| SketchBudget.Next | Generated exact encoded sizes at cap edges and maximum escaping-heavy terminal records; reject before mutation, charge retries zero, retain reservations for interrupted labels, and count zero history scans. |
| board.Writer / recorder.mirror | Stateful collecting/refusing sinks and frozen clocks through production entry points; assert emitted order, no duplicate writes, pre-effect refusal, and unknown persistence after a possibly accepted write. |

## Tasks (governing PQ-2 revision)

### Task 1: Pin and implement the wire contract

**Files:** recording/{sketch.go,sketch_codec.go,sketch_test.go,sketch_codec_test.go,record.go,codec.go,codec_test.go}.

- [ ] Write failing codec/compatibility tests using the named strategies above; run `go test ./cmd/internal/webcast/recording -run 'Sketch|Encode|Decode|Unknown' -count=1` and confirm missing behavior.
- [ ] Implement all eight focused preserving codecs and shared semantic canonicalization through strictjson, extend the exhaustive sample registry, rerun to PASS and commit.

### Task 2: Implement pure sketch history

**Files:** recording/{sketch_state.go,sketch_state_test.go,reduce.go,control.go}, their tests, tests/webcast/sketch-corpus.json.

- [ ] Write failing NextSketch/State.Apply tests and shared prefix corpus from the governing strategies.
- [ ] Implement validated deltas, read-only projections and contextual transitions; run `go test ./cmd/internal/webcast/recording -run 'Sketch|Control|Annotation|Pause' -count=1` to PASS and commit.

### Task 3: Port the browser model and pressure geometry

**Files:** sketch/{geometry.go,geometry_test.go}, web/{sketch-model.js,model.js,annotation-anchor.js}, server.go, tests/webcast/{sketch-model.test.mjs,sketch-geometry.json}.

- [ ] Write failing shared geometry/core tests using the named strategies, then implement named-function exports and both embedded asset registrations.
- [ ] Run `go test ./cmd/internal/webcast/sketch -count=1`, `node --test tests/webcast/sketch-model.test.mjs`, and `go test ./cmd/internal/webcast -run 'TestNode|OperatorScript' -count=1` to PASS and commit.

### Task 4: Snapshot bounded sketch/control catch-up

**Files:** recording/{catchup.go,contract.go,catchup_test.go,sketch_snapshot.go}, web/{replay.js,viewer.js,replayer.js,sketch-model.js}, replay_corpus_test.go, Hub tests, tests/webcast/{replay.test.mjs,catchup_corpus.json}.

- [ ] Write failing snapshot/compaction/prefix equivalence tests from the governing strategies, then implement the approved snapshot revision below through the same reducer.
- [ ] Run `go test ./cmd/internal/webcast/recording ./cmd/internal/webcast -run 'CatchUp|Sketch|Hub' -count=1` and `node --test tests/webcast/replay.test.mjs tests/webcast/sketch-model.test.mjs` to PASS; commit without changing legacy DOM goldens.

### Task 5: Fence source time and reserve capacity

**Files:** recording/{sketch_budget.go,sketch_budget_test.go,files.go,order_test.go}, board/{writer.go,writer_order_test.go}, recorder/{mirror.go,note_order_test.go}, focused producer tests.

- [ ] Write failing OrderedTime/SketchBudget.Next/producer tests from the governing strategies, then integrate each shared helper once per producer.
- [ ] Run `go test ./cmd/internal/webcast/recording ./cmd/internal/webcast/board ./cmd/internal/webcast/recorder -run 'Sketch|Order|Writer' -count=1` to PASS; commit measured reserve arithmetic and uncertainty contract.

### Task 6: Document, verify and land

**Files:** atlas/{webcast-format.md,webcast.md,index.md if needed}, issue, this plan, workshop/lessons.md for review lessons.

- [ ] Document model/format/compatibility and snapshot contract. Run all four feature packages, webcast TestNode/CatchUpCorpus/OperatorScript, focused Node tests, reader/artifact inventories and `git diff --check`.
- [ ] Push and run recorded affected check; use full suite for known pair#738 graph failure without bypass. Run `make test-affected-inventory` before close.
- [ ] Close once with measured evidence; gates own fresh review. Fix findings, land through sdlc, report the outcome to ops:0, and start #752 only from landed main. TL owns project edits.

## Revisions

### 2026-10-20 — encoded terminal reserve correction approved by ops:0

The parent design's provisional 4 KiB finalization reserve is superseded for this implementation. A 1024-byte valid label can require 6146 JSON bytes before the envelope when HTML-sensitive ASCII is escaped. ops:0 explicitly approved deriving the reserve from worst-case encoded terminal records, including the bounded envelope and stroke-end or interrupt, rounding up to at least 8 KiB, retaining the 32 MiB total cap, and adding the adversarial-label regression. Increase above 8 KiB if the complete maximum sequence requires it. This is approved arithmetic within the design and needs no new operator review.

### 2026-10-20 — implementation seam reconnaissance

- Use `SketchOperation{Operation string, Sequence uint64, Digest string}` on the eight concrete new records, keeping Header unchanged. `SketchPoint` is a `[3]float64` with strict raw tuple validation. SketchBegin.Doc follows the existing document-segment identity used by notes, not an arbitrary current root handle.
- Existing note ownership emits lowercase SHA256 hex64 or reserved `operator` (`broadcast/note_owner.go`); use those public identities. Keep note-owner scope separate. Do not change legacy Note validation.
- `NextSketch(state,event,context)` returns an opaque validated delta, committed privately only after all existing contextual validation succeeds. Export snapshot projections for consumers, not mutable storage. Producers call receipt reconciliation before sink append, so duplicate no-op reduction cannot still write duplicate JSONL.
- `SketchBudget.Next(record,encodedBytes,duplicate)` proposes counters/reservations and Commit installs them only after State accepts. Producer reconstruction uses one history pass; chunk admission does no rescans. Reserve label plus stroke-end plus interrupt conservatively, since end then a later handoff can both occur. Maximum escaping-heavy IDs can require more than 8 KiB; calculate and round up (possibly 16 KiB), never assume 8 KiB suffices.
- `OrderedTime` currently has three callers (two producers and its test); extend it to return an overflow error, and consolidate each producer's unchecked annotation-clear `T+1` special case in the same helper. Refusals do not advance State.T or lastKind.
- Geometry imports recording types, never the reverse. Keep browser module exports as named functions/constants. No new transport receipts, visible windows, or source-snapshot IO in this slice.

### 2026-10-20 — PQ-1/PQ-2: reducer-derived snapshot and function strategies

ops:0 delivery 432c999b-f034-4fe0-84f3-f36982901ab4 directs a derived snapshot instead of selective ctl retention or gap-tolerant validation. This supersedes the earlier catch-up paragraph and Task 4 mechanism. Original disk events remain append-only and complete. For sketch-bearing catch-up, reduce the accepted input prefix with the same State/NextSketch owner, retain existing DOM/asset/note/segment records without sketch or ctl history, and append one ephemeral `sketch-snapshot` bootstrap frame containing the current control and bounded sketch state. Non-sketch recordings keep current catch-up behavior. Replay seeking consumes the same frame, not a second history algorithm (ARCH-DRY/PURE/CONSTRAINTS).

The snapshot carries source begin metadata, fixed colour/ownership, phases/interruption evidence, ordered strokes/chunks/pressure paths, open/sealed/truncated flags, visible and redo stacks, next sequence, operation receipts (claimed digest plus actual semantic identity), counts and active identity. Restore validates bounds, stack/reference consistency and owned copies before installing anything. It restores ctlSeen and the current holder atomically; subsequent ctl validates normally. Snapshot copying/flattening is off the append hot path. Repeated compaction can consume a prior snapshot and new records, so it never needs the original full log. Snapshot size is bounded by admitted sketch history (32 MiB plus bounded indexes/projection overhead); arbitrary control chatter adds no retained historical ctl population. Existing DOM/notes retention remains its existing contract.

The bootstrap frame is not a new persisted event kind: no FileEvents registry entry and no writer admission. Add an explicit catch-up-only State/Model snapshot admission option; ordinary disk Decode sees an unknown kind and does not install it, raw browser replay refuses a bootstrap in source records, and ordered writers reject it. Catch-up verification, live viewer bootstrap, Player rebuild and Replayer seek opt into snapshot installation only; this option never relaxes normal ctl validation. A malformed frame refuses atomically. CatchUp keeps its []Record contract and emits the frame through the existing encode/SSE seam; no synthetic frame is written to the recording. For invalid source prefixes, retain the input for normal validation rather than claiming an accepted snapshot.

PQ-2 replaces procedural case inventories with the governing named-function strategy table and concise tasks above. The invariant contracts, state/recovery table and numeric limits remain authoritative. Digest shape is validated in the pure core; receipts compare both the claim and actual semantic identity, so a forged unchanged digest cannot hide a changed body. SketchDigest/SketchCanonical are shared producer/test helpers; browser asynchronous digest generation is pinned to those bytes, avoiding synchronous crypto in Model.apply. Fixed colour is required in SketchBegin.Author.Color; no second colour field is introduced.
