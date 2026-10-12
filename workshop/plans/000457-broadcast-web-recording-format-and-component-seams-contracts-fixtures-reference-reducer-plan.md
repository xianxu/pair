# broadcast-web recording format and component seams — Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Give couch-broadcast-web's five wave-1 components (pair#458–#462) one recording format and one interface + stateful fake + contract suite each, so they build in parallel without asking.

**Architecture:** The format spec is `atlas/webcast-format.md`, and the Go package `cmd/internal/webcast/recording` is its executable form. It holds the typed records, a JSONL codec, the directory reader/writer, a pure reducer (state at t), a validator, the catch-up rule for joining mid-cast, and the recorder-side fakes. The other components get one package each (`webcast/assets`, `webcast/notes`, `webcast/query`), holding only the interface, the fake and a contract suite. Their real implementations land there in wave 1. Fixtures are re-recorded with #453's prototype recorder and converted by a Go generator. They are checked in with their reference screenshots and canonical goldens.

**Tech Stack:** Go (stdlib only: `encoding/json`, `crypto/sha256`, `image/png` for frame headers); Node only to run the #453 prototype recorder, offline in the scratchpad, to produce raw inputs.

---

## Decisions (with the reason each was made)

1. **One record stream, split by kind on disk.** The project's sketch had `snapshot` as its own part. Here, snapshots are `doc` records inside `events.jsonl`, because the issue asks for "one shape for the live stream and the on-disk recording". Joining mid-cast is then a filter over the same records (`CatchUp`), not a second format.
2. **No `manifest.json`.** The `head` record is the header, in both the stream and the file (ARCH-DRY: one truth for version and id).
3. **Ids: one integer namespace per recording.** It covers every node of every frame, plus stylesheets. Change records then need no frame field, and an id alone is an anchor for a note. Ids are kept across resume via `backendNodeId` (#458's job; the format states the guarantee).
4. **Every iframe document is a separate `doc` record with `host`**, same-process or not. This gives one rule instead of two, and the reducer and viewer each have one code path.
5. **All coordinates are top-page CSS px.** This covers clicks, layout boxes, note boxes, strokes and crops. They are scroll-independent, and a note's box stays valid while the page scrolls.
6. **DOM keeps original URLs; assets bind absolute URL → content hash.** Agents see the real URLs, and the viewer and proxy resolve them through the table. Capture order doesn't matter, because a URL is resolved at serve time.
7. **Layout boxes are records (`layout`), captured on demand.** The DOM stream has no geometry, but the notes model (#461) needs boxes for context and crops. The recorder captures them with each frame and when a note is made.
8. **Reducer is strict.** A dangling reference is an error, reported with its line. #453's replayer silently counted "missing", and that hid user-agent shadow leakage.
9. **Fakes come with contract suites** (`RunXContract(t, factory)`). The wave-1 implementation runs the same suite as the fake, so the two can't drift (ARCH-MOCK).
10. **Fixture conversion is a generator (`recording/internal/fixturegen`) over the prototype's raw CDP log.** It is not product code. Its CDP→format normalization (node-id mapping, UA-shadow filtering, frame stitching) is a reference #458 may promote. It is not a second recorder.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `Record` (+ one struct per kind) | `cmd/internal/webcast/recording/record.go` | new |
| `Node` (wire node) | `cmd/internal/webcast/recording/record.go` | new |
| `Decode` / `Encode` / `FileOf` | `cmd/internal/webcast/recording/codec.go` | new |
| `State` / `Reduce` / `(*State).Apply` | `cmd/internal/webcast/recording/reduce.go` | new |
| `(*State).Canonical` | `cmd/internal/webcast/recording/canonical.go` | new |
| `Validate` / `Errors` | `cmd/internal/webcast/recording/validate.go` | new |
| `CatchUp` | `cmd/internal/webcast/recording/catchup.go` | new |
| `DecodeFile` / `Merge` / `Split` / `Finding` | `cmd/internal/webcast/recording/files.go` | new |
| `Builder` | `cmd/internal/webcast/recording/builder.go` | new |
| `TextContent` / `Element` | `cmd/internal/webcast/recording/dom.go` | new |
| `CanonicalTimes` / `PinCanonical` / `Pin` | `cmd/internal/webcast/recording/fixture.go` | new |
| `CheckCatchUp` | `cmd/internal/webcast/recording/contract.go` | new |
| `converter` (CDP log → records) | `cmd/internal/webcast/recording/internal/fixturegen/cdpconv.go` | new |

- **Record** — tagged union: `Record` interface (`Kind() string`, `Time() int64`) with a struct per kind (`Head`, `Seg`, `Stop`, `Doc`, `Ins`, `Rm`, `Kids`, `Attr`, `Text`, `Shadow`, `Sheet`, `Adopt`, `Font`, `Asset`, `Input`, `Scroll`, `Selection`, `Nav`, `Title`, `Viewport`, `Click`, `Focus`, `Layout`, `Note`, `NoteState`, `Stroke`, `Frame`) plus `Unknown` (kept raw, for forward compatibility). The kinds are listed once, in a `kinds` table that maps kind to constructor and file family. Decode, `FileOf` and the validator's "known kind" check all derive from it (ARCH-DRY).
  - **Relationships:** a recording is 1:N records. `Doc.Tree`, `Ins.Node`, `Kids.C` and `Shadow.Root` hold `Node` trees.
  - **Future extensions:** new kinds are rows in `kinds`. A compact encoding is a different codec over the same structs.
- **State/Reduce** — the meaning of a recording: an in-memory DOM forest (nodes by id, parent links) plus the observer, context and notes state. `Apply` is atomic: it validates every reference first and mutates only after that, so a failed record leaves the state untouched.
  - **Relationships:** the notes model and the queries consume `*State` read-only. The viewer conforms to its `Canonical()`.
- **Validate** — decodes the lines, checks the envelope (`head` first, a supported `v`, `t` never decreasing), per-kind required fields and segment structure (`seg`, then the top `doc`, before any change), and stable id type and tag. It then reduces everything and attaches each reducer error to its file:line. Unknown kinds produce warnings.
- **CatchUp** — pure join rule: `head` + all `asset` records + records from the latest `seg` on.
- **cdpconv.Converter** — prototype CDP log → records. It maps per-session CDP `nodeId` and `backendNodeId` to recording ids, drops UA shadow roots, splits `contentDocument` into `doc` records with `host`, and stitches OOPIF sessions by `frameId` → owner iframe.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `Sink` / `Tee` / `AssetWriter` / `DirWriter` | `cmd/internal/webcast/recording/dir.go` | new | filesystem |
| `ReadDir` / `ValidateDir` / `Fixture` | `cmd/internal/webcast/recording/dir.go`, `fixture.go` | new | filesystem |
| `MemSink` (fake Sink) | `cmd/internal/webcast/recording/fake.go` | new | — |
| `FakeRecorder` | `cmd/internal/webcast/recording/fake.go` | new | the recorder (#458) |
| `Live` / `FakeHub` / `RunLiveContract` | `cmd/internal/webcast/recording/fake.go`, `contract.go` | new | live fan-out (#460) |
| `assets.Store` / `MemStore` / `RunStoreContract` | `cmd/internal/webcast/assets/` | new | asset store + proxy (#459) |
| `notes.Model` / `FakeModel` / `RunModelContract` | `cmd/internal/webcast/notes/` | new | notes model (#461) |
| `query.Surface` / `FakeSurface` / `RunSurfaceContract` | `cmd/internal/webcast/query/` | new | agent queries (#462) |
| `fixturegen` main, notes spec | `cmd/internal/webcast/recording/internal/fixturegen/main.go`, `notes.go` | new | raw prototype runs on disk |

- **Sink** — `Put(Record) error`, `Close() error`; it carries records only. `DirWriter` is the real file writer. It splits records by `FileOf`, is the one writer of asset bodies (`PutAsset(body)`, the `AssetWriter` the store persists through) and writes frames (`PutFrame`), each atomically. `MemSink` keeps records in memory and enforces the same envelope checks.
  - **Injected into:** the recorder (#458) writes to a Sink (fan-out = `FakeHub` + `DirWriter` via `Tee`).
- **Live** — `Subscribe() (<-chan Record, cancel func())`. The catch-up comes first, then the live records. A slow subscriber is dropped, not blocked (ARCH-CONSTRAINTS): the channel closes and the viewer reconnects and gets a fresh catch-up. `FakeHub` implements Sink + Live with `CatchUp`.
- **assets.Store** — `Put(ctx, Capture{URL, MIME, Body, Declared}) (recording.Asset, error)`, `Get(hash) (Body, error)`, `Lookup(absURL) (recording.Asset, bool)`. `Policy` holds the image resolution budget. `MemStore` dedups by SHA-256 and leaves bodies exact: resampling is #459's, and the contract suite only checks that `orig` is set iff the stored body differs.
- **notes.Model** — `Anchor(st, node) (Anchor, error)`, `Resolve(st, Anchor) (NodeID, bool)`, `Context(st, Anchor) (Context, error)`, `Crops(st, []Anchor, Hints) ([]Box, error)`, `Live(st) []Note` (notes on the current document). `FakeModel` gives deliberately simple answers (the element itself as context, the bbox + margin as crop), enough for #462 to build on before #461 lands.
- **query.Surface** — `DOMAt(t, selector) (string, error)`, `Events(t1, t2) ([]recording.Record, error)`, `Notes(window) ([]NoteView, error)`. `FakeSurface` is backed by `Reduce` over a loaded recording; it only supports an `#id` selector.

### ARCH notes

- **ARCH-PURE:** reducer, validator, catch-up, codec and converter are pure; the IO is `DirWriter`/`ReadDir`/fixturegen main.
- **ARCH-ORDER:** record order is the file order; `t` never reorders. Equal-`t` merge order across files is fixed (events, notes, pointer). Hub subscriptions take their catch-up and register under one lock, so no record can fall between them (contract test with a concurrent writer).
- **ARCH-FUNERAL:** recordings are kept until the operator deletes them (stated in the atlas, size per minute named). Fixtures are static test data, about 1–3 MB in total. A `FakeHub` subscription ends with cancel, or with the hub's Close.
- **ARCH-SECURE:** a recording is untrusted page content. The format keeps scripts, but the viewer contract forbids running them. `ReadDir` refuses asset names that aren't 64-hex, so there is no path traversal from a hash. The reducer bounds nothing yet: recordings are the operator's own.

## Tasks

### Task 1: Format atlas page — DONE before this plan (`atlas/webcast-format.md`), linked from `atlas/index.md` and `atlas/webcast.md`.

### Task 2: Records and codec
**Files:** `recording/doc.go`, `record.go`, `codec.go`, `codec_test.go`.
- [ ] Test first: round-trip each kind (`Decode(Encode(r)) == r`). An unknown kind decodes to `Unknown` and re-encodes to the same bytes. Missing `k` or `t` is an error. `attr` with `"v":null` decodes to a nil `*string`.
- [ ] Implement the `kinds` table and the structs. Node: `{ID, Y, N, NS, A map[string]string, C []Node, SR *Node, TC *Node, V *string, Mode}`.
- [ ] `go test ./cmd/internal/webcast/recording/`, commit `#457: recording: records and codec`.

### Task 3: Reducer and canonical form
**Files:** `reduce.go`, `canonical.go`, `dom.go`, `builder.go`; tests in `reduce_test.go`.
- [ ] Tests, hand-built records, one per rule:
  - `seg` drops the DOM and keeps the assets;
  - the top `doc` must follow `seg`;
  - `ins` with `after` = 0, middle and last;
  - `rm` removes the subtree from the id index;
  - a move (rm, then ins of the same id) keeps the id;
  - `kids` replaces;
  - `attr` with nil removes;
  - `text` on an element is an error;
  - `shadow` attaches;
  - an iframe `doc` with `host` replaces the frame;
  - `input`, `scroll`, `selection` and `focus` on unknown ids are errors;
  - `stop` freezes the state (`Live=false`) and a change after it is an error until the next `seg`;
  - `layout` replaces the boxes;
  - a `note` on an unknown node is still kept (notes outlive DOM), with `Resolved` false;
  - `Apply` is atomic: a failing `ins` leaves the state identical, compared via `Canonical`.
- [ ] `Reduce(recs, t)` applies every record with `Time() <= t`.
- [ ] `Canonical()` is a deterministic JSON document: segment, live, url, title, viewport, the top document tree with frames inlined under their host as `"frame"`, shadow roots under `"sr"`, form state folded into each element as `"$input"`, scroll as `"$scroll"`, and the sheets by owner/adoption. Golden test on a small hand recording.
- [ ] Commit `#457: recording: reference reducer and canonical state`.

### Task 4: Validator, files, catch-up
**Files:** `validate.go`, `files.go`, `catchup.go`, `dir.go`; tests in `validate_test.go` and `catchup_test.go`.
- [ ] Validator tests, one finding each: `head` not first, unsupported `v`, `t` decreasing, a missing required field, a change before the top `doc`, a tag change for an id, a dangling reference (with line), an unknown kind (warning only). A good recording gives zero findings.
- [ ] `FileOf(kind)` from `kinds`; `Merge(events, notes, pointer)` is a stable merge by `t` with the fixed tie order (test the tie order).
- [ ] `CatchUp` tests: assets from earlier segments kept; only the latest segment's DOM; works while paused.
- [ ] `DirWriter` + `ReadDir` round trip (records, asset bodies, frames); asset names validated (`^[0-9a-f]{64}$`).
- [ ] Commit `#457: recording: validator, directory layout, catch-up`.

### Task 5: Recorder-side seams and fakes
**Files:** `fake.go` (Live, MemSink, FakeHub, FakeRecorder), `contract.go` (TB, RunLiveContract, CheckCatchUp), `dir.go` (Sink, Tee); tests in `fake_test.go`.
- [ ] `Sink`, `Tee(sinks...)`, `MemSink`, `Live`, `FakeHub`, `FakeRecorder{Records, Speed}` with `Run(ctx, Sink)` (Speed 0 = instant).
- [ ] `RunLiveContract(t, newHub)`:
  - a late subscriber's first records equal `CatchUp(all so far)`;
  - no gap or duplicate under a concurrent writer (deterministic: a writer goroutine and N subscribe calls; each subscriber's stream is checked against the full log);
  - a slow subscriber is dropped (its channel closes), not blocking `Put`;
  - `Close` ends every subscription.
- [ ] `FakeHub` passes it, and the test runs it.
- [ ] Commit `#457: recording: sink, live fan-out seam, fakes`.

### Task 6: Component interfaces with fakes and contract suites
**Files:** `cmd/internal/webcast/assets/{doc,store,fake,contract,fake_test}.go`, `notes/{doc,model,fake,contract,fake_test}.go`, `query/{doc,surface,fake,contract,fake_test}.go`.
- [ ] assets contract:
  - `Put` of the same bytes twice gives one hash;
  - `Get(hash)` returns the stored bytes, whose SHA-256 is the hash;
  - `Lookup(url)` after `Put`;
  - a rebind of the same URL to new bytes;
  - `Get` of an unknown hash is `ErrNotFound`;
  - `orig` is set iff the body was changed.
- [ ] notes contract, over a fixture `State`:
  - `Anchor` gives node, selector, text and box (when there is layout);
  - `Resolve` of a fresh anchor gives the same node;
  - `Resolve` after the node is removed is false;
  - `Context` includes the node;
  - each crop contains its notes' boxes;
  - `Live` holds only notes since the current segment.
- [ ] query contract, over a fixture:
  - `DOMAt(t,"")` is non-empty and contains the title text;
  - `Events(t1,t2)` are all within the window and in order;
  - `Notes` gives every note, each with its element.
- [ ] Commit `#457: webcast: assets, notes, query seams with fakes`.

### Task 7: Fixtures
**Files:** `recording/internal/fixturegen/{main,cdpconv,cdpconv_test,notes}.go`, `recording/testdata/fixtures/{stress,app,blog,segments}/…` with a README and `record-mjs.patch`, `recording/fixture.go`, `recording/fixture_test.go`.
- [ ] Raw runs: #453's prototype in the scratchpad, mode B, with layout, click and focus capture added and `frameNavigated` logged, headless. Done by a subagent; it records stress, app and blog.
- [ ] `cdpconv` tests on tiny synthetic CDP logs:
  - nodeId mapping;
  - a UA shadow root is dropped, and later events under it are dropped too;
  - `contentDocument` is split into `doc` with `host`;
  - an OOPIF session is stitched to its owner via `frameId`;
  - `setChildNodes` becomes `kids`;
  - `styleText` and `rules` become `sheet` with `owner`;
  - an observer `value` becomes `input`;
  - a top `documentUpdated` or second snapshot becomes `seg(navigate)` + `doc`;
  - stylesheet bodies become `asset` + body;
  - layout bounds map from backend ids.
- [ ] fixturegen writes each fixture: records, assets, frames (the prototype's shots), and notes and strokes from a small per-fixture spec (element by `id` attribute or tag + text at time t; box from the nearest `layout`). It also writes `canonical/<t>.json` goldens at each frame time.
- [ ] The `segments` fixture is hand-written with fixturegen's builder: go-live, pause, resume with ids kept, a tab switch, end.
- [ ] `fixture_test.go`: every fixture validates with zero errors and reduces at every frame time, and its goldens match (`-update` flag to regenerate). Size check: the fixtures total under 4 MB.
- [ ] Commit `#457: fixtures: stress, app, blog, segments`.

### Task 8: Atlas, review, close
- [ ] Fill the fixture table and link `webcast-format.md` from `atlas/index.md` and `atlas/webcast.md`.
- [ ] Tell the TL (ops:0) the format page is ready for the operator's review; fold in the review as a `## Revisions` entry.
- [ ] Full `make test` (memory: default `make -k test`, scratchpad TMPDIR for test-changelog, then `go test ./...`).
- [ ] `sdlc close`, then merge main, `sdlc pr`, `sdlc merge --yes`, report to ops:0.

## Revisions

### 2026-10-11 23:5x — during implementation
- **Goldens become pins.** The plan had `canonical/<t>.json` goldens per fixture. At the stress page's size that is megabytes of JSON, and a second copy of what the reducer already computes. Instead each fixture keeps `canonical.sha256` (t → hash of `State.Canonical()`), and `fixturegen -canonical DIR -at T` prints the expected value on demand (ARCH-DRY: the reducer is its only source). The viewer conformance check compares against that output.
- **Resume means "same document".** `seg.why: resume` is going live again on the document shown before the pause; a different document (navigated while paused) is `navigate`. `State.DocSeg` records where the current document began. `CatchUp` and the notes model's `Live` carry a document's notes across a resume. That is what "notes clear on reload", together with "pause/resume continues from the tab's state", requires.
- **Contract suites take `recording.TB`**, not `*testing.T`, so product packages don't import `testing`.
- **Fixture coverage is asserted** (`TestFixtureCoverage`), and the fixtures have a size budget (4.5 MB; 4.0 MB used). A regenerated fixture therefore can't silently lose a case or grow without notice.
- **Stress fixture:** the FontFace font's body (Apple Chancery, a macOS system font) is excluded from the repository; the `font` record stays.

### 2026-10-12 — close review round 1 (REWORK, BR-1..BR-4)
- **Where the Core-concepts files actually are** (BR-2; the tables above were edited in place in round 2):
  - `Live`, `FakeHub`, `MemSink` and `FakeRecorder` are in `recording/fake.go`;
  - `RunLiveContract`, `ContractRecording` and `TB` are in `recording/contract.go`;
  - `Sink`, `Tee`, `AssetWriter`, `DirWriter`, `ReadDir` and `ValidateDir` are in `recording/dir.go`;
  - `Located`, `DecodeFile`, `Merge`, `Split` and `Finding` are in `recording/files.go`;
  - `Builder` is in `recording/builder.go`, `TextContent`/`Element`/`ParentElement` in `recording/dom.go`, and the pins in `recording/fixture.go`;
  - there are no `live.go` or `sink.go`;
  - the reducer and canonical tests are in `reduce_test.go`, and the file, dir and validator tests in `validate_test.go`;
  - `fixturegen` is `main.go`, `cdpconv.go` and `notes.go`.
- **BR-1:** `CatchUp` lost where the current document began, so after a resume, a late joiner's notes had segment 0 and fell out of `notes.Live`. It now keeps that `seg` as a bare record. `TestCatchUpEquivalence` compares `DocSeg` and the live notes.
- **BR-3 and the kids minor (one namespace, enforced):**
  - a sheet now claims its id, and a sheet can't take a live node's id;
  - `kids` may reuse only ids under the children it replaces, not ids in the parent's shadow root or template content.
- **BR-4 (seam ownership):**
  - a `Sink` carries records only;
  - asset bodies have one writer, `DirWriter` through `recording.AssetWriter`, into which the store persists;
  - a `Live` refuses frames (`ErrNotLive`), and `RunLiveContract` checks it.
- **Minors:**
  - `PutAsset` writes through a temporary file and a rename;
  - `forget` scans sheets only for style and link elements;
  - `RunLiveContract` gained a deterministic sequential join at every point, and keeps the concurrent run as stress;
  - the atlas now says pins (not goldens) and that `Reduce` replays from record 0.

### 2026-10-12 — close review round 2 (SHIP, 3 advisory, fixed by rule)
- Every catch-up equivalence check now goes through `CheckCatchUp`: canonical page, assets, `DocSeg` and live notes. The synthetic cuts and the fixture pins both use it.
- Every `DirWriter` body write (assets and frames) goes through `writeFileAtomic`. A `.tmp-*` file a crash left behind is reported by `ValidateDir`, and it goes with the recording's directory.
- The Core-concepts and Task tables were edited in place; Revisions keeps only the why.
