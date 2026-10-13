# DOM submission implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach. Steps use checkbox syntax for tracking.

**Goal:** Shift+Ctrl+Alt+Return in the draft sends its text and the cast tab's open notes, with focused recorded context and one current crop per note, to the agent in that view.

**Architecture:** Prepare a durable bundle in Couch, send its absolute path through the existing authored-draft transaction, then acknowledge delivery to resolve exactly the submitted notes. Freeze the bundle and target across retries. Keep historical context separate from current screenshot evidence.

**Tech Stack:** Go, existing CDP recorder and Couch local socket, Lua/nvim draft transaction, JSON and PNG files.

## Scope and decisions

The issue's operator-shaped Log supersedes its initial proposal: one draft hotkey, all open web notes, JSON plus PNG files, no clipboard, no ticket creation, no terminal submission implementation. #488 and #493 are on main; #455 will consume the shared Go type later.

The hub confirmed on 2026-10-12:

- Call `NoteCrops` separately for every note, so overlapping notes still have separate PNGs.
- Context and replay address the note's recorded moment. Pixels are captured now from the actual cast tab, with capture time and document identity.
- A selected note whose document has already gone keeps its recorded context and explicitly reports `crop unavailable: page changed`.
- Navigation during the capture transaction refuses the submission; no delivery, draft consumption or note resolution occurs.

All open notes means those on the cast tab's document at admission, as returned by the Board. Navigation after admission must not silently erase that selection. This does not resurrect notes from earlier documents that were already absent at admission.

The normal Alt+Return path remains the ordinary draft submission. The new chord works with an empty/comment-only draft when notes exist. With no notes, nonempty text follows ordinary submission; with neither, it is a no-op. No separate menu button is added under the operator's later one-hotkey decision.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `Submission`, `Note`, `Crop`, `Source` | `cmd/internal/submission/model.go` | new |
| `Context` JSON fields | `cmd/internal/webcast/notes/model.go` | modified |
| `Build` (recorded note context to submission) | `cmd/internal/webcast/submission/build.go` | new |
| `Delivery` transition model | `cmd/internal/submission/delivery.go` | new |
| Frozen DOM preparation in draft transaction | `nvim/dom_submission.lua`, `nvim/submission.lua` | new / modified |

`Submission` owns a version, ID, source kind, recording absolute path/ID, target thread identity, selected document identity and ordered notes. Each note contains ID, author, text, time, anchor kind/path/selector/box, focused context, compact element text, recent actions, replay link and crop result. A crop result is either a relative `crop-N.png` with bounds/capture time/capture document, or an explicit unavailable reason. Use lowercase JSON names and documented units (CSS pixels and recording-relative milliseconds; wall-clock capture time separately). DOM-specific context lives in an optional web payload; the common envelope, author, anchor kind/path, crop and text do not require terminal implementation. No duplicate rendered submission file.

`Build` reuses `query.Querier.Notes` for note-time context, compact element and preceding actions, and `notes.Structural.NoteCrops` once per note. Filter by the frozen Board selection, not by whatever is latest after IO. Missing selected records are a visible preparation failure, never silently omitted notes. The existing query's missing-element representation becomes explicit unavailable context, retaining anchor text.

`Delivery` owns permitted transitions and retry actions. Lua owns the draft's submission state; Go owns bundle preparation and note acknowledgment. Neither re-derives the other's delivery decision from a missing response.

Precisely, Go `ApplyDelivery(state, event)` owns the persisted states `prepared`, `delivering`, `dispatched`,
`resolved`. `Store.Transition(id, target, event)` is its only writer and atomically replaces/fsyncs the manifest
before answering the RPC. `begin` is prepared -> delivering (idempotent when delivering); nvim must obtain
that durable answer before any paste. `ack` is delivering -> dispatched, persisted BEFORE resolving any notes;
`resolved` follows observed note-state records. Duplicate events return the existing result, and backwards
transitions refuse. There is no claim that a durable `delivering` proves paste or submission occurred.

Lua `DOMSubmission.send` owns only this live draft attempt's frozen payload/target and uses
`PairSubmission.submit_operator_text` for the existing start/written/indeterminate/dispatched transitions.
The new adapter calls prepare once, begin before the transaction, and ack only after the transaction reports
dispatched. `PairSubmission` receives and compares an immutable attachment identity as well as authored text;
it never silently replaces generated text for a pending attempt. Lua retains a separate ack-only pending ID
after PairSubmission clears its log transaction. Failure to persist/receive begin prevents send. Failure to
persist/receive ack after dispatch retains ack-only retry and clears the sent draft normally. Ack/status on a
persisted dispatched bundle can resume resolution after Couch restart; it cannot paste anything.

After nvim restart, a delivering bundle has an unknown delivery outcome: preparing another bundle for that
target refuses and names the existing manifest for manual composer reconciliation. Status never upgrades it
to dispatched from silence. The operator can explicitly acknowledge an already-sent bundle; an unsent one
must be reconciled before removing its preparation. No automatic delivery retry spans nvim restart.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `Store` | `cmd/internal/submission/store.go` | new | portable folder, atomic manifests and PNG files |
| Cast preparation and acknowledgment | `cmd/internal/webcast/cast_submission.go` | new | Board, recording reader, crop source |
| Board selection and guarded resolution | `cmd/internal/webcast/board/submission.go` | new | reducer ownership and ordered recorder emissions |
| Recorder crop capture | `cmd/internal/webcast/recorder/submission.go`, `cmd/internal/webcast/chrome_submission.go` | new | existing CDP connection and recorder queue |
| Console submission binding | `cmd/internal/couchtty/console_submission.go` | new | cast, active thread, replay-share owner |
| Local submission RPC/CLI | `cmd/internal/couchcmd/submission.go`, `message_service.go`, `cli.go`; `cmd/internal/couchmessage/protocol.go` | new / modified | owner-only Couch socket |
| Draft hotkey adapter | `nvim/dom_submission.lua`, `nvim/init.lua`, `nvim/submission.lua` | new / modified | prepare/ack CLI and existing authored send |

Stateful doubles: extend the recorder's `fakeChrome` to return bounded PNGs and change document/connection state at controlled capture boundaries; portable temp-directory Store is the real backend; Couch tests use the existing console fixture and socket harness; nvim tests reuse the stateful fake Zellij agent and authored-log seam. A small opt-in headless Chrome check verifies clip coordinates after scrolling and DPR, with no window/focus changes. Live operator acceptance uses the real blog cast only when the hub schedules it.

## Ordering and failure contract

| State/event | Result/effects |
|-------------|----------------|
| Idle + chord | Pin calling draft's thread and active agent identity; snapshot open notes; begin one preparation. |
| Preparing + repeated chord | Report busy; do not start a second capture. |
| Preparing + document already changed before capture | Keep selected recorded notes; mark their crops unavailable. |
| Capturing + navigation, tab switch, cast close or timeout | Fail preparation; retain draft and notes; no published partial bundle. |
| Prepared + send | Append original authored text using existing log gate, send stripped draft plus absolute manifest pointer. |
| Send refused before paste | Keep same preparation available for retry. |
| Body written + retry | Submit the exact frozen message; never prepare a new bundle. |
| Indeterminate body write | Preserve bundle and draft; require composer reconciliation; no automatic resend. |
| Dispatched + acknowledgment | Resolve only captured IDs on their captured document, then mark completion. |
| Dispatched + acknowledgment lost/fails | Draft may clear because delivery happened; keep acknowledgment-only retry, never resend. |
| Late acknowledgment after navigation/new notes | Do not resolve notes on the new document or notes created after selection. |
| Duplicate prepare/ack with same identity | Return same result; reject reuse with different target/selection. |
| Restart with prepared/uncertain bundle | Keep evidence, do not infer delivery or auto-resend from absence of a receipt. |

Freeze authored body, generated agent text, preparation ID, selected notes, recording/document and recipient for the retry lifetime. Existing `submission.lua` only compares authored text: extend its immutable transaction identity so a new manifest cannot be finalized after an old pasted body was submitted. A commit-only retry stays commit-only. Preserve `!` description behavior without allowing a description-only action to acknowledge an unsent DOM bundle.

The product's chord is an explicit operator send and can use the established draft delivery gesture. Pin and verify its agent pane/thread rather than discovering a different focused slot after preparation. Do not send via peer `Broker.Send`: its eight-message quota and peer envelope are not operator submission semantics. Do not alter the peer breaker or manufacture operator events.

The recorder queue supplies capture identity and checks CDP ordering; merely comparing URLs is insufficient (same-URL reloads count). Capture work happens outside the event-loop critical section; its result is checked after queued navigation evidence has been processed. Snapshot/capture completion channels are buffered and owned; cancellation joins the operation or leaves only a bounded existing CDP call, never a new unowned worker. Use a recorder sequence barrier for the post-capture check, including navigation while a CDP screenshot call is in flight.

Resolution success means the matching note-state records were observed in the recording/Board, not merely accepted by asynchronous `Emit`. Acknowledgment is idempotent. A page that has gone needs no live resolution; its submission still remains valid historical evidence.

## Operating envelope, trust and lifetime

- **ARCH-DRY / ARCH-PURE:** share the submission type, notes model, query formatting and existing authored transaction. Pure build/transition tests take values; the IO shell owns CDP, files and socket calls.
- **ARCH-PURPOSE:** deliver the whole operator gesture, empty draft, PNG evidence and successful-resolution loop. Ticket assets and terminal capture remain the explicitly separate issues.
- **ARCH-MOCK:** the same capture/store/draft seams run with stateful fakes; tests can inject document changes and failures between every commit point.
- **ARCH-CONSTRAINTS:** an operator-triggered batch, one preparation per Couch and sequential PNG capture. Derive the open-note count bound from the Board's palette/color limits. Initial proposed limits: 30 s preparation, 5 s per CDP capture within that deadline, 4 megapixels and 8 MiB per PNG, 64 MiB per bundle. Refuse excessive crops with a visible explanation; never silently omit notes or resize their evidence. Read recording files with a total 128 MiB input budget before reduction; report oversized recordings explicitly. These are conservative co-tenancy assumptions for TL review, not measured performance claims. No heavy test loops or load benchmark.
- **ARCH-SECURE:** treat recordings, socket requests and CDP responses as untrusted typed inputs. Validate JSON version, IDs, finite bounded crop coordinates before arithmetic, PNG format/dimensions before decode/allocation, canonical generated relative crop names and absolute store paths. Reject malformed complete recording lines; #493's incomplete live tail remains retryable. Private directories 0700, files 0600. Replay tokens appear only in private bundle files, never the broadcast notices. Tests use temporary roots, never operator state.
- **ARCH-ORDER:** enforce the transition table above with a single preparation owner, immutable target/selection and controllable late completions. Do not hold Console/Board locks during capture, file reads or socket waits.
- **ARCH-FUNERAL:** Store creates `<store>/webcast/submissions/<id>/submission.json` and crops; the receiving agent/operator is the final consumer until ticket assets are copied. Keep successful bundles until explicit removal, with a 1 GiB total writer-admission cap including interrupted staging directories; show the absolute store and required cleanup when full. Never evict evidence automatically. Preparation stages in private directories, publishes atomically after all crops validate, and removes its own failed staging. Crash remnants count toward the cap and are recognizable for cleanup. Delivery metadata lives in the manifest, not another rendered copy. Durable retry state has the same bundle lifetime. These proposed budgets and retention policy require TL review.

## Implementation tasks (one atomic issue-close review)

### 1. Shared format and pure build

Files: create `cmd/internal/submission/{model,delivery}_test.go` and corresponding implementation; create `cmd/internal/webcast/submission/build_test.go` and implementation; modify `cmd/internal/webcast/notes/model.go`.

- [x] `Validate`/`ValidateCrop`: table-driven malformed versions, coordinate/path domains and boundary values; round-trip canonical JSON fixtures. `Build`: property over frozen selection and recording sequence (output IDs equal selection, each crop index unique), including navigation and overlap. `ApplyDelivery`: generate event sequences and assert monotone durable progress and no delivery effect on ack/status.
- [x] Run `GOMAXPROCS=2 go test -p 1 ./cmd/internal/submission ./cmd/internal/webcast/submission -count=1`; confirm the intended failures.
- [x] Implement the types, validation and pure builder using existing query/model APIs; preserve note ordering and explicitly unavailable context/crops.
- [x] Rerun the targeted command and commit the coherent change with `#483: ...` and author trailer.

### 2. Store, selection and current-tab crop capture

Files: create `cmd/internal/submission/store{,_test}.go`, `cmd/internal/webcast/board/submission{,_test}.go`, `cmd/internal/webcast/recorder/submission{,_test}.go`, `cmd/internal/webcast/chrome_submission.go`, `cmd/internal/webcast/cast_submission{,_test}.go`; extend recorder fake and browser fake at the same seam.

- [x] `Store.Prepare`/`Store.Transition`: inject failure at each write/sync/rename boundary against portable folders; readers see old-or-complete manifests, never partial files, and writer budgets include crash residue. `ValidatePNG`: adversarial headers/dimensions/encoded lengths must refuse before allocation. `Board.Selection`/`Board.ResolveSelection`: interleave note creation, resolution and document replacement; only matching selected identities can change. `Recorder.CaptureCrops`/`captureBarrier`: controllable CDP replies and delayed navigation events prove pre-existing page loss yields unavailable and any in-capture identity change refuses; cancel/close releases ownership.
- [x] Run `GOMAXPROCS=2 go test -p 1 ./cmd/internal/submission ./cmd/internal/webcast/board ./cmd/internal/webcast/recorder -run 'TestSubmission|TestCropCapture' -count=1` and confirm intended failures.
- [x] Implement snapshot selection, bounded capture with CDP clip and sequence validation, portable atomic Store, and guarded note-resolution acknowledgment. Preserve captured note identities through navigation.
- [x] Rerun targeted tests; prove navigation/failed-write tests fail when their guards are removed; commit.

### 3. Couch preparation and acknowledgment adapter

Files: create `cmd/internal/couchtty/console_submission{,_test}.go`, `cmd/internal/couchcmd/submission{,_test}.go`; modify `cmd/internal/couchcmd/{cli,message_service,webcast}.go`, their tests and `cmd/internal/couchmessage/protocol{,_test}.go`.

- [x] `ValidateSubmissionRequest`: cross-product invalid fields and supported verbs proves the grammar closed. `Console.Submission`/`handleSubmission`: real socket harness with controlled thread/cast switches and replay readiness; reject stale recipients and preserve the same request ID after lost responses. `Store.Transition` through RPC: duplicate begin/ack/status cannot repeat paste or resolve unselected notes.
- [x] Run `GOMAXPROCS=2 go test -p 1 ./cmd/internal/couchcmd ./cmd/internal/couchtty ./cmd/internal/couchmessage -run 'TestSubmission|TestDOMSubmission' -count=1` for red.
- [x] Add prepare/status/ack operations and a small CLI adapter. Return structured JSON with ID, absolute manifest path, fixed target and state. Reuse the replay-share owner for per-note replay links. Keep all slow effects outside the UI loop and use its existing notice path.
- [x] Rerun tests and commit. Confirm neither preparation nor status sends agent text or touches the clipboard.

### 4. Draft hotkey and send/resolve transaction

Files: create `nvim/dom_submission.lua`, `nvim/dom_submission_test.lua`; modify `nvim/{init,submission}.lua`, `nvim/submission_test.lua`, `nvim/submission_integration_test.lua`; extend `tests/submission-transaction-nvim-test.sh` and the repository's existing key-map tests as needed.

- [x] `DOMSubmission.send`/`PairSubmission.submit_operator_text`: stateful agent+log+RPC fake injects failure before and after each side effect; assert original authored log text, frozen generated text, no duplicate paste and ack-only recovery after dispatch. `send_and_clear` hotkey adapter: headless nvim drives normal/insert and queue/history states, asserting draft consumption iff dispatch and notes treated as content. `draft_send.send`: fixed target survives retry and refuses replacement incarnation.
- [x] Run `lua nvim/submission_test.lua`, `lua nvim/submission_integration_test.lua`, `lua nvim/dom_submission_test.lua` and `bash tests/submission-transaction-nvim-test.sh`; confirm intended failures under temporary roots.
- [x] Wire Shift+Ctrl+Alt+Return to preparation then the existing authored transaction. Keep original authored text in Pair's log, manifest pointer only in agent text. Clear/consume draft only after dispatch; finalize notes separately and retain finalize-only recovery.
- [x] Rerun the Lua/headless-nvim tests and targeted routing tests. Verify ordinary Alt+Return and `!` behavior. Commit.

### 5. Evidence, documentation and operator acceptance

Files: `README.md`, `atlas/webcast-submission.md` (new), `atlas/index.md`, relevant sections of `atlas/couch.md`, `atlas/webcast-format.md`, and this issue/plan.

- [x] Document the chord, schema example, note/capture time distinction, unavailable crops, storage/lifetime, retry semantics and query/replay pointers. Sweep initial button/clustering wording so current operator guidance reflects the approved spec.
- [x] Run the affected Go packages once with low parallelism and the draft integration tests. Run one bounded headless CDP conformance test (separate profile, no GUI/focus) if permitted by the hub's load constraint; otherwise report that specific gap before acceptance.
- [x] Coordinate the required full-suite gate with the hub's no-machine-load instruction; do not launch a full suite or Chrome workload while it conflicts with that instruction.
- [x] Ask the hub to schedule smoke/staging: two blog viewers create notes; draft hotkey from blog:0 sends all notes and crops; empty draft sends; agent reads only bundle and fixes one correctly; successful notes resolve on every screen; page reload keeps cast working; failure preserves draft/notes. Never stage pair:0 without the hub's instruction.
- [ ] Record actual evidence and outstanding acceptance. Run SDLC close with verified evidence only after operator smoke; the binary owns the fresh-context review. Fix findings, update lessons where needed, then open the PR.
- [ ] On SHIP, merge current main, complete required checks, run `sdlc merge --yes` without validation bypasses, report PR/outcome to hub, and stop.

## Revisions

- 2026-10-12 — Initial implementation plan after claim transfer to pair:6; incorporates hub confirmation of one PNG per note and unavailable crops for a document already gone. Awaiting TL review of plan, limits and retention before implementation.
- 2026-10-12 — TL approved the plan and all proposed limits/retention, and directed entry to change-code (Couch receipt `0a3761ed-b2f7-4044-8a01-fb16fabd915f`). Implement resolve-after-delivery; TL is confirming that policy with the operator and will send any delta.
- 2026-10-12 — PQ-1/PQ-2: named Go's persisted delivery transition authority and Lua's live draft authority, added durable begin-before-paste and conservative restart recovery, and replaced test inventories with function-level adversarial strategies.


- 2026-10-12 — Implementation uses direct pinned Zellij pane writes with revalidation before paste and submit, so the operator's focus remains stationary. The capture barrier spans CDP and recorder event reduction. Note revisions are local conditional-emission guards and are omitted from recording wire records.
- 2026-10-12 — Store metadata reads validate paths/types/sizes without decoding retained PNGs for each status/transition; full Read and publication validate actual PNG bytes. A paused current cast refuses capture. Cumulative crop bytes are checked during collection, before a full batch can exceed 64 MiB.
- 2026-10-12 — TL approved serial full-suite and bounded headless Chrome verification, then close; operator smoke will follow on TL's schedule. #494's document-scoped annotation suppression helper must be integrated by whichever of #483/#494 lands second.

- 2026-10-12 — Operator confirms that submitted notes become resolved: submitting transfers ownership of the notes to the agent; whatever follows (a pensive, an issue, a fix) is the agent's. No code-policy change required.

- 2026-10-12 — Automated verification is complete with main's existing notification test failure tracked as pair#519 and explicitly accepted by TL for local close. Runtime inventory/drift findings were fixed and retested. Guard mutations were tested through Go overlays, keeping working sources unchanged. Actual pure tests use deterministic tables and stateful sequences; no load/fuzz campaign was run. TL requires waiting after close for operator smoke before merge.

- 2026-10-12 21:19 PDT — Operator additions, confirmed by TL (receipts 8ea3022f-ac70-498d-a0e3-033af2f2414a and f9849280-01be-4cef-86d4-d5a6415ada73): preserve current capture-time box text alongside historical anchor text. Use the original unpadded note region, or anchor box when no region, and visible intersection in reading order; skip display:none, visibility:hidden, zero-size, off-screen and aria-hidden content. Label visible text and accessible names (aria-label, alt, title, role + name) separately. Bound each note to 8 KiB visible text, 8 KiB names, 256 entries and 20k visited nodes, explicitly marking truncation. PNG remains definitive appearance; text preserves characters; selectors locate elements. A changed-away document has unavailable current evidence. Include one canonical how_to_read string in submission.json and delivered agent text: manifest/crop meanings, absolute recording path, couch --broadcast-web-query DIR notes|dom|events|link; start with notes, dom --at <t> --sel @N, events --from <t-10000> --to <t>, and couch --skill for the guide. TL authorizes one additional bounded headless conformance run.
- 2026-10-12 21:19 PDT — Close round 1 returned REWORK. BR-1 requires repeated directory synchronization on post-rename durability retries (Prepare and all delivery transitions). BR-2 reclassifies DOMSubmission as INTEGRATION, with a pure state/event transition authority that declares effects and owns all retry state. BR-3 consolidates pane parsing and pair-wrap identity shared by submission transport and pair_poke. Fix and test these before rerunning close; no merge before scheduled operator smoke.

### Additional implementation tasks — approved scope and review corrections

- [ ] Add a bounded BoxText value to the shared schema and original region to note evidence. Extend the single CaptureCrops operation with paired crop/text boxes; capture text in the admitted top-frame isolated context and keep document fences around the whole operation. A pure JS collector consumes DOM reads, orders visible intersecting text and separately labelled accessible entries, and reports truncation; the CDP shell owns deadlines/context identity. Test filtering, ordering, attribute names, limits, missing document and navigation. (ARCH-CONSTRAINTS, ARCH-ORDER, ARCH-MOCK.)
- [ ] Add one pure bundle-reading guide formatter and persist its output as how_to_read; return that same guide through prepared/status RPC and append it in generated agent text. Extend the actual couch --skill guide with web recording usage. Test JSON, RPC and delivered text for absolute recording path and every named command. (ARCH-DRY, ARCH-PURPOSE.)
- [ ] Fix BR-1 post-rename directory sync retries and prove repeated failures cannot authorize paste. Extract the pure Lua transaction model and shared pane parser for BR-2/BR-3; adversarial state sequences retain send-once/ack-only semantics. (ARCH-PURE, ARCH-ORDER, ARCH-DRY.)
- [ ] Update README/atlas, runtime mirrors and inventory, run affected low-parallel tests and the authorized bounded headless conformance check. Commit all evidence, rerun close, report to hub and wait for operator smoke before merging. Durable additions remain under existing bundle/store retention bounds. (ARCH-FUNERAL, ARCH-SECURE.)
