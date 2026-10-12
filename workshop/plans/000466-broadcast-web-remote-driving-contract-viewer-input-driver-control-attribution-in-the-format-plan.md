# broadcast-web remote-driving contract Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extend #457's recording format and seams, without changing anything existing, so that a viewer can drive the operator's cast tab: viewer input messages, one-driver control, attribution in the recording, an input-injection seam and the reverse channel's contract. Each comes with fakes, contract suites and a driven fixture, so pair#458, #460 and #467 can build against it.

**Architecture:** The format stays the executable spec in `cmd/internal/webcast/recording`. A new `ctl` record carries each control change *and the control state after it*. An embedded `Source` (`by`) attributes interaction records to a viewer, and the reducer refuses a `by` that isn't the current driver. A new package `cmd/internal/webcast/drive` holds the viewer→Couch wire (`Batch`/`Act`, strict parse, limits), the confinement policy (pure), and two seams with stateful fakes and contract suites: `Injector` (dispatches a driver's batch into the cast tab; the real one is CDP `Input.dispatch*`, built in #467) and `Controller` (seats, request/grant/deny/revoke/release/leave, input routing). The HTTP routes and the CDP injector are #467's build; this issue specifies them (atlas) and ships everything they implement against.

**Tech Stack:** Go (stdlib only), JSON Lines.

**Agreed with the TL (ops:0, 2026-10-12):** `by` is optional (absent means the operator) on click, input, focus, scroll, selection, hover and ptr. The validator errors if `by` isn't the current driver. Each `ctl` record carries the state after it. Control survives pause, resume and navigation, and ends at `stop end`. The recorder's own goroutine is the single writer of the record stream (corrected by pair:1, 10-12): `ctl` records and injections enter its FIFO queue as inputs, ordered with DOM events. The routes are `POST /<token>/drive/{hello,control,input}`. The injector refuses stale targets, the right and middle buttons, accelerator keys, the clipboard, file pickers and new windows.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `Source` (embedded `by`) | `cmd/internal/webcast/recording/control.go` | new |
| `Ctl` record, `Control` state, `NextControl` | `cmd/internal/webcast/recording/control.go` | new |
| `Click`, `Input`, `Focus`, `Scroll`, `Selection` | `cmd/internal/webcast/recording/record.go` | modified (embed `Source`) |
| `State` (gains `Control`; attribution check) | `cmd/internal/webcast/recording/reduce.go` | modified |
| `CatchUp` (keeps the latest `ctl`) | `cmd/internal/webcast/recording/catchup.go` | modified |
| `CheckCatchUp` (compares control) | `cmd/internal/webcast/recording/contract.go` | modified |
| `DrivenRecording` | `cmd/internal/webcast/recording/contract.go` | new |
| `driven` fixture + `drivenFixture` | `recording/internal/fixturegen/notes.go`, `testdata/fixtures/driven/` | new |
| `Act`, `Batch`, `ParseBatch` | `cmd/internal/webcast/drive/act.go` | new |
| `Confine` | `cmd/internal/webcast/drive/policy.go` | new |

- **Source**: `struct{ By string `json:"by,omitempty"` }`, embedded in every interaction record. The reducer checks any record implementing `attributed` (`source() string`) generically, before the per-kind switch. #458's new `hover` and `ptr` embed it and get the check for free.
  - **Relationships:** N records → 0..1 viewer (by id; the name and color are in `ctl` records).
  - **DRY rationale:** one field and one check for all seven kinds, instead of a field and a check per kind.
  - **Future extensions:** `by` on DOM changes (effects) if agents ever need them. Today an effect is attributed through the interaction that preceded it.
- **Ctl / Control / NextControl**: `Ctl{Op, Who Author, Driver *Author, Pending []Author}`. `Control{Driver *Author; Pending []Author}`. `NextControl(c, op, who) (Control, error)` is the one transition function: the reducer uses it to check every `ctl` record (the stated state must equal `NextControl(prev, op, who)`), and the fake controller uses it to produce records. Ops: `request` (adds a viewer to pending; refused if that viewer is already pending or driving); `grant` (needs the viewer pending; makes it the driver, and a previous driver loses control); `deny` (needs the viewer pending); `revoke` (needs the viewer to be the driver); `release` (the viewer is the driver, or withdraws its pending request); `leave` (the seat expired: removed from wherever it is, refused if it is nowhere). A no-op transition is refused.
  - **Relationships:** one `Control` per recording (not per segment: it survives `seg`). A `stop end` resets it to empty.
  - **DRY rationale:** the record carries the state so the catch-up keeps just one `ctl`. The reducer recomputes that state, so the carried copy can't drift from the rules.
- **CatchUp change**: the latest `ctl` before the latest `seg` is kept in place. Records from the latest `seg` on are kept as they are today. The FakeHub stays bounded by one segment.
- **Act / Batch / ParseBatch**: the viewer→Couch input wire. `Batch{Seg int, Acts []Act}`. `Seg` is the segment the viewer was showing, so a batch aimed at a document that has since gone is refused as stale. `Act{A, ID, X, Y, Button, Count, DX, DY, Key, Code, Mods, V, Sel}`. Kinds: `down`/`up`/`move`/`click` (Count 2 means double), `wheel`, `key` (`down`/`up` via `Type`), `text` (insert text), `ime` (composition `V` + `Sel`). Coordinates are top-page CSS px, as everywhere in the format. The parse is strict, like `broadcast.ParsePointBatch`: unknown fields are refused, and the limits (`MaxDriveBody` 8 KiB, `MaxActs` 64, `MaxText` 1,000 runes) are named constants.
- **Confine**: `Confine(a Act) error`, a pure allow-list. Only the left button. Keys come from an editing and navigation list (Enter, Tab, Escape, Backspace, Delete, the arrows, Home/End, PageUp/PageDown, Space, printable keys). Shift is always allowed. Ctrl or Meta only with `a`, `z` and `y` (select all, undo, redo). No clipboard keys (c/x/v), no Alt (back/forward), no F-keys, nothing else with Ctrl or Meta (browser accelerators such as Cmd+W, Cmd+L and DevTools). Every refusal wraps `ErrRefused`.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `Injector`, `ErrNotLive`, `ErrStale`, `ErrRefused` | `cmd/internal/webcast/drive/injector.go` | new | CDP `Input.dispatch*` into the cast tab (#467) |
| `FakeInjector` | `cmd/internal/webcast/drive/fake_injector.go` | new | a reduced `recording.State` (hit-test by layout boxes) |
| `RunInjectorContract` | `cmd/internal/webcast/drive/contract.go` | new | — |
| `Controller`, `Seat`, `ErrNoSeat`, `ErrNotDriver` | `cmd/internal/webcast/drive/controller.go` | new | seats, presence and control: the reverse channel's server side (#467 puts it behind HTTP, with its operations entering the recorder's queue) |
| `FakeController` | `cmd/internal/webcast/drive/fake_controller.go` | new | in-memory seats + a `recording.Sink` |
| `RunControllerContract` | `cmd/internal/webcast/drive/contract.go` | new | — |

- **Injector**: `Inject(ctx, by string, b Batch) error`.
  - **Contract:**
    - `ErrNotLive` while paused, or when `b.Seg` isn't the current segment.
    - `Confine` is applied to every act before anything is dispatched; a refused batch dispatches nothing.
    - A `down`, `click` or `wheel` checks that the node at (x, y) in the real tab is `ID` or inside it, and refuses with `ErrStale` otherwise. A `move` or `up` mid-drag isn't checked.
    - What the observer reports while the batch is dispatched carries `by`.
  - **Real (#467):** CDP `Input.dispatchMouseEvent`, `dispatchKeyEvent`, `insertText` and `imeSetComposition`. The hit test is `DOM.getNodeForLocation`, with the backendNodeId mapped to a recording id through #458's map: `Recorder.IDOf(session, backendNodeId)` and `Recorder.NodeOf(id)`, each a request with a reply, answered on the recorder's goroutine. The injection itself runs as a recorder-queue input, so what the observer reports during it gets `by` in queue order. While a viewer drives, file choosers are intercepted (`Page.setInterceptFileChooserDialog`) and cancelled, downloads are denied (`Browser.setDownloadBehavior`), and a target opened by the cast tab (`targetCreated` with that opener) is closed.
  - **Fake:** reduces the page records it is given. Its hit test picks the smallest layout box (`State.Boxes`) containing the point. It emits observed records into a `recording.Sink`, stamped `by`: a click gives `click`, plus `focus` on a form control; `text` gives `input` on the focused control (on `type=password`, `{secret:true}` with no value, following #458's password rule, once #458 lands; until then the fake refuses to type into a password field); a `wheel` gives `scroll` of the top document.
  - **Injected into:** `Controller` (it routes the driver's batches to it).
- **Controller**:
  - **Interface:**
    - `Join(name string) (Seat, error)`: a seat is `{Viewer Author; Key string}`, with Key 32 random bytes in hex.
    - `Attach(viewerID) (detach func(), error)`: presence for a stream's life. A seat with no attached stream for `grace` leaves, emitting `ctl leave` if it was involved.
    - `Request(key)`, `Release(key)` and `Input(ctx, key, Batch)` are viewer calls, authenticated by the key.
    - `Grant(viewerID)`, `Deny(viewerID)` and `Revoke()` are operator calls.
    - `Control() recording.Control` and `Subscribe()` feed Couch's indicator.
  - **Records:** it writes `ctl` records to a `recording.Sink`, with `t` from an injected clock. In the real build the Controller's operations are inputs to the recorder's queue, so the record stream keeps one writer.
  - **Fake:** `FakeController`, the full state machine, with grace timers on a `time.AfterFunc`. Its lifecycle (ARCH-FUNERAL): seats are memory only; they leave after `grace` without a stream, and all go at `Close`.

**ARCH-FUNERAL lines.**
- `ctl` records are written by the Controller. They live as long as the recording, which is kept until the operator deletes it (#457's rule). There is one per human control action, about 100 bytes each, so a cast's control records are bounded by its length.
- Seats are memory only: they die after `grace` without a stream, or with the cast.
- The `driven` fixture is about 10 KB, which counts toward the 4.5 MB fixture budget the test enforces.
- This issue creates no new file families.

**ARCH-CONSTRAINTS.** Limits are 8 KiB per drive body, 64 acts per batch and 1,000 text runes. Rate and in-flight limits (as in `broadcast`'s pointer routes) are #467's HTTP layer; they are specified in the atlas.

## Plan

Single pass (no milestones). TDD per step.

### Task 1: `recording`: Source, Ctl, NextControl, reducer, catch-up

**Files:** create `recording/control.go`, `recording/control_test.go`. Modify `record.go`, `reduce.go`, `catchup.go`, `contract.go`, `reduce_test.go` and `catchup_test.go`.

- [ ] Test `NextControl`: a table of every op × {none, pending, driving, other driving}. Legal rows give the expected state; illegal and no-op rows give an error.
- [ ] Implement `control.go`: `Source`, `attributed`, `Ctl` (`kinds["ctl"]` in `FileEvents`; `check`: op in the set, `who.id`), `Control`, `NextControl` and the op constants.
- [ ] Embed `Source` in `Click`, `Input`, `Focus`, `Scroll` and `Selection`. Encoding is unchanged when `by` is empty (a codec round-trip test).
- [ ] Reducer: `State.Control`. A `ctl` applies anywhere after head (outside a segment too) and is refused unless its stated `driver`/`pending` equals `NextControl(prev)`. `stop end` resets Control. Before the in-segment switch, an `attributed` record with `by` must name the current driver, else error. Tests: attributed click by the driver ok; by a pending viewer → error; after revoke → error; after a navigate seg the driver still drives.
- [ ] `CatchUp` keeps the latest `ctl` before the last seg. `CheckCatchUp` compares `Control`. Test: catch-up at every cut of `DrivenRecording` passes `CheckCatchUp`; a test removing the kept `ctl` fails (mutation check).
- [ ] `DrivenRecording()` in `contract.go`. It goes live with layout. A viewer requests and is granted, then clicks and types (attributed), and navigates (navigate seg, still driving). There is a pause and a resume. The viewer releases; a second viewer is granted and revoked; a request is denied; a viewer leaves. Then stop end.
- [ ] `go test ./cmd/internal/webcast/recording/...` green (existing fixture pins unchanged: no existing record changes meaning). Commit.

### Task 2: `driven` fixture

**Files:** modify `recording/internal/fixturegen/notes.go` and `main.go` (`-driven`), `recording/fixture.go` (`Fixtures` += `driven`), `fixture_test.go` (coverage: grant, release, revoke, an attributed click and input, a navigate seg while driving, a note by the driver) and `testdata/fixtures/README.md`.

- [ ] Coverage test row for `driven` (fails: no fixture).
- [ ] `drivenFixture()`, written with `-driven`, pins via `-update-pins`; `TestFixturesValidateAndReduce` and coverage green. Commit.

### Task 3: `drive` wire and policy

**Files:** create `drive/doc.go`, `drive/act.go`, `drive/act_test.go`, `drive/policy.go`, `drive/policy_test.go`.

- [ ] Tests: `ParseBatch` accepts each act kind and refuses unknown fields, a missing `seg`, empty acts, over-limit acts, text and body, a bad kind, and non-finite coordinates. The errors are fixed values that never echo the input.
- [ ] Tests: `Confine` table. Left ok, right and middle refused. Enter, Tab, arrows and printable ok. Cmd/Ctrl+C/X/V refused; Cmd+W, Cmd+L, Cmd+T, F12, Cmd+Alt+I and Alt+Left refused; Cmd+A, Cmd+Z and Shift+Cmd+Z ok.
- [ ] Implement; commit.

### Task 4: `Injector` seam, fake, contract

**Files:** create `drive/injector.go`, `drive/fake_injector.go`, `drive/contract.go` (`RunInjectorContract`) and `drive/fake_injector_test.go`.

- [ ] `RunInjectorContract(t, newInjector func(page []recording.Record) (Injector, *recording.MemSink))`. The page is `recording.DrivenRecording()`'s first segment.
  - Not live → `ErrNotLive`.
  - A wrong seg → `ErrNotLive`.
  - A refused act → `ErrRefused`, with nothing emitted for the batch.
  - A click on a node whose box doesn't hold the point → `ErrStale`.
  - A good click → a `click` with `by`.
  - `text` after focusing an input → an `input` with `by` and the value.
  - Every emitted record reduces after the page with the driver in control.
- [ ] `FakeInjector` passes it (`fake_injector_test.go` runs the suite). Commit.

### Task 5: `Controller` seam, fake, contract

**Files:** create `drive/controller.go`, `drive/fake_controller.go`, `drive/fake_controller_test.go`; `drive/contract.go` (`RunControllerContract`).

- [ ] `RunControllerContract(t, newController func(sink recording.Sink, inj Injector, clock func() int64, grace time.Duration) Controller)`:
  - Join gives distinct ids and keys.
  - A bad key → `ErrNoSeat`.
  - Request, then grant → `ctl` records whose state matches.
  - Input from a non-driver → `ErrNotDriver`, with nothing injected.
  - The driver's input reaches the injector with `by`.
  - Granting another viewer transfers control.
  - Release, deny and revoke work.
  - With grace at 30 ms, a detached driver leaves and control returns to the operator, while a re-attached one within the grace keeps control.
  - `Subscribe` sees each change.
  - The sink's records, after a page prefix, validate (`Reduce` without error).
- [ ] `FakeController` passes it. Commit.

### Task 6: Atlas, issue, close

- [ ] `atlas/webcast-format.md`:
  - A Context row for `ctl`; `by` in the Context and Observer tables.
  - An "Attribution and control" section.
  - The catch-up rule.
  - A "Remote driving (#466)" section: the wire (routes `POST /<token>/drive/hello|control|input`, bodies, responses, refusal sentences, limits, rate and in-flight), presence (`events?viewer=`), confinement, the recorder's queue as the single writer, the operator's visibility (Couch indicator plus an in-page banner, the same mechanism as the page control), and what stays with the operator (permission prompts, JS dialogs).
  - Components rows for `drive.Injector` and `drive.Controller`.
  - The JS viewer contract gains `nodeAt` and `toPage`.
  - The fixtures table gains `driven`.
- [ ] `atlas/index.md` mentions `drive`.
- [ ] Ask the TL to review (Done when).
- [ ] `go test ./cmd/internal/webcast/...`, full `make -k test`, then `sdlc close`.

## Revisions

### 2026-10-12: single writer is the recorder's queue (pair:1 via TL)
The record stream's single writer is #458's recorder goroutine, not the Chrome loop: the loop forwards CDP events and
live/pause/tab inputs into the recorder's FIFO queue, and the recorder calls CDP synchronously in queue order. `ctl`
records and injections enter that queue. The id map is `Recorder.NodeOf(id) → (session, backendNodeId)` and
`Recorder.IDOf(session, backendNodeId) → id`, request-with-reply on that goroutine. Passwords: `input {id, secret:true}`
(#458); the fake injector follows it.
