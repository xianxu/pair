# Owner annotation tools — implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach. Steps use checkbox syntax for tracking.

**Goal:** The authenticated cast operator can use Note and Pointer by default while other viewers still need an Annotation grant and see a clear explanation when it is OFF.

**Architecture:** Reuse the existing isolated-world identity proof, originating generation, Board admission and prepared-note lifecycle. Distinguish trusted operator annotation admission from viewer admission without changing the shared capability state machine or granting authority from UI attributes.

**Tech Stack:** Go Board/recorder/CDP integrations; existing JavaScript annotation policy and mounted controls; stateful fake CDP and bounded native Chrome.

## Evidence and approval boundary

TL9a388ca5 approves owner Note by default with viewer Annotation OFF, requires an exact owner test and a plan before code. TLeed8fb98 clarifies that this applies to both owner Note and Pointer, with no action-specific split. This supersedes the issue's original blanket capability-OFF requirement only for authenticated operator ingress.

The shipped native mount at the supplied1728×997 viewport has a full, native-hit-testable Note button at1664,284.75,48×36; the column ends at324.75 with no overflow. Initially Annotation is OFF, the button is disabled at opacity0.4 with title “Annotation is off.” Pen is enabled. Enabling the existing capability changes Note to enabled/opacity1 with identical geometry. Diagnostic PASS2.342s, one headless private-profile run; source and full output are retained under this plan's canonical `.side/evidence/`, log SHA256fc79f964d1284100bd034ee2a0119701775dd7d00b7d1aa2f27bcab050053401.

The supplied recording confirms1728×997/DPR2 and has no ordinary notes. It excludes private injected control state, so it cannot prove the live button's precise OFF/hidden state. The older screenshot shows the previous injected glyph rail over a mock; it does not prove a missing modern button or asset mismatch. Neither private capture is copied into the repository. The native result and source identify a reproducible default-availability defect; the operator recheck will confirm that it resolves the observed problem.

## Core concepts

Paths below are relative to `cmd/internal/webcast/` unless stated otherwise.

| Pure entity | Lives in | Status |
|---|---|---|
| Annotation authority binding: viewer permit or trusted operator generation | `board/annotation.go` | new private entity in modified existing file |
| Admission and prepared authority binding | `board/annotation.go`, `board/prepared_note.go` | modified |
| Existing annotation availability/mode policy | `web/annotation-policy.js` | reused unchanged |
| Existing capability state/generation/cleanup transitions | `capability/state.go` | reused unchanged |

The authority binding is a private comparable value, not a client field or a second registry. It binds the issuing Board, surface/role and original generation; PreparedNote carries and compares that binding rather than an ordinary viewer permit alone. Empty/zero admission remains invalid, while generation0 is valid when explicitly issued to the owner. A shared locked checker is used at every admission-consuming boundary. Preserve the existing operator-sketch admission's policy explicitly; do not accidentally widen viewer permits or silently change sketch cleanup semantics while generalizing the representation.

| Integration point | Lives in | Status | Owns |
|---|---|---|---|
| Registered isolated-context proof and ordered operator dispatch | `recorder/operator.go`, `recorder/recorder.go` | modified | CDP identity, source generation and repeated writer checks |
| Prepared capture/save handoff | `board/prepared_note.go` (modified); `recorder/operator_notedraft.go` and existing `notedraft` service (reused unchanged) | modified Board binding; existing service/adapters unchanged, tests expanded | scoped preparation, token restoration and unchanged unknown-outcome recovery |
| Direct operator session actions | `board/operator.go` | modified | server attribution, exact request ledger and trusted-surface action dispatch |
| Permission delivery and terminal mode selection | `chrome_operator.go` | modified | active isolated-world delivery, revision fencing and SetAnnotationMode |
| Mounted shared controls and viewer OFF explanation | `web/annotate.js` (modified), `web/annotate.css` (reused unchanged) | reason projection modified; existing layout unchanged | actual buttons, visible reason and existing cancellation paths |
| Viewer HTTP, session proof and capability publication | `server_board.go`, `server_notedraft.go`, `cast_capabilities.go` | reused unchanged; tests expanded | grant-based viewer admission; never report global ON for owner work |
| Disposable native conformance | `operator_owner_live_test.go` using existing launch helpers | new | owner UI default operation and viewer HTTP denial at the same time |

No new cast, permission registry, background worker, timer, storage record or lifecycle state is introduced.

## Exact owner test and authority flow

The trusted owner surface is the cast's own injected `couchcast` isolated world. For each mutation the recorder must prove:

1. The CDP event belongs to the current top session, not another target or iframe.
2. Its actual execution-context ID is registered for that session's top frame and is still stamped with the current loader/document/recording document segment.
3. The recording is opened, live and not ended/paused; a remote viewer is not driving.
4. The request's canonical originating generation equals the current generation, and page/document authority is current at dispatch and final ordered write.

Use `operatorIdentity` and `operatorAdmission`, including document stamps and the final writer callback; never derive ownership from `Viewer.ID`, `author`, Owner, URL/localhost, a cookie, a header, `operator:true`, or a page-supplied execution-context number. The event's execution context comes from CDP, not its payload. A shared/remote HTTP viewer remains a viewer even if it claims “operator.” Attribution is assigned on the trusted path as today.

Mint owner annotation admission only after that identity check on the recorder actor. The immutable parsed request already carries its source generation through the queue: do not replace it with arrival-time authority. Remove or revise the old ordinary-viewer admission capture in `enqueue` coherently so a stale queued request cannot obtain fresh owner authority. Status/recovery paths keep their existing non-mutating reconciliation semantics and identity checks.

Owner annotation admission accepts only a live Board and a settled Annotation phase, OFF or ON, at the matching generation. Viewer admission still requires its actual valid ON permit. Disabling/running/retry/broken cleanup admits no fresh owner Note/Pointer: this preserves the existing ordered clear boundary, pending-reservation cleanup and preparation revocation. After cleanup settles OFF the owner can start new work without granting viewers. Generation changes invalidate already prepared/queued work; a cancelled or uncertain save never turns into a duplicate. Keep the existing command lock and ordered recorder writer; do not hold a Board lock across IO.

Use one shared authority checker in admission Check and every method currently calling `admissionLocked(a.permit)`: prepare/reserve/restore, owner session writes, edits, stroke/state actions and any admission-aware resolution helper. Retain ordinary viewer entry points and their checks. A prepared value must not cross Boards, generations, roles or operator/viewer surfaces; restoration still requires the service's signed token and owner/scope/generation/expiry proof before rebinding local authority. Owner authority never becomes a transferable HTTP credential.

For UI delivery, known current operator contexts receive owner availability independent of viewer OFF, but remain false when inactive or fenced. Reuse the existing monotonic permission receipt/retry mechanism and generation string, including generation0; no new origin payload or role claim is sent. `SetAnnotationMode` selects the known active operator target and uses the same owner-generation admission, then retains its post-call check. The viewer capability snapshot and HTTP headers do not change.

## Visible states and recovery

Actions: select Note/Pointer, save/cancel the existing note editor, wait for readiness, ask owner for a grant, retry cleanup, end/reopen cast.

| Internal condition | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| Current trusted owner, live page, settled OFF or ON | Note and Pointer available | Select tool; annotate | None |
| Annotating viewer, Annotation ON | Existing tools available | Select tool; annotate | None |
| Non-owner viewer, Annotation OFF | Explicit “Annotation is off. Ask the cast owner to enable viewer annotation in Couch.” | Ask owner for a grant | No write admitted |
| View-only link | Explicit read-only reason; no promise that toggling a grant upgrades this link | Ask owner for an annotating link | No write admitted |
| Owner/viewer page pending, inactive, paused or remotely driven | Existing unavailable/paused/driving explanation | Wait/return/resume/release control | Existing transient gesture/draft cancellation rules |
| Capability cleanup running | Existing unavailable state with cleanup reason | Wait | In-progress gestures/preparations may be retired by the existing clear |
| Cleanup retryable/broken | Existing retry/end explanation | Retry cleanup or end/reopen | Existing preparation revocation; retained unknown writes still reconcile |
| Old context/generation/document or ended cast | Refused; no fresh feedback | Start a fresh action on the current page or reopen cast | Unsaved old-scope input may require re-entry; no silent resubmission |

The non-owner OFF explanation is visible text, not only a dim button or tooltip. Reuse the existing hint/status owner, changing text only on availability transitions. Do not introduce a viewer-side “enable” button, automatic grant or a Pointer-to-Note detour. The owner tools remain ordinary labelled controls in the accepted column. Existing page-load guards, unsaved-note submission protection and finished-sketch label rules stay intact.

## Cost, lifetime and formats

Authority tests are constant work per operation: one existing mutex acquisition and fixed-size comparisons. Existing active-context delivery remains one pass over C registered contexts; no new per-note scan, registry or retry loop. Pin C and2C contexts with counted sends/checks under OFF/ON and repeated unchanged updates so unchanged contexts do not amplify delivery. DOM reason updates reuse existing mode/availability transitions and do not scan feedback rows.

The new authority value dies with the existing admission/preparation/session; no persistent authority token is added. Existing signed capture tokens, pending attachments, request ledgers, contexts and draft-service budgets retain their owners, limits and cleanup. No recording, submission, cookie or capability wire schema changes, so no new version or unknown-field policy is needed; existing preserving codecs remain unchanged. UI explanatory text is not an authority carrier.

## Implementation and verification

- [x] **Admission entity:** add pure authority/phase/generation tests for owner OFF/ON, viewer OFF/ON, default invalid admission, stale generations, cleanup running/retry/broken/ended and cross-surface equality. Implement the shared checker and migrate prepared binding consumers; require RED before behavior changes.
- [x] **Trusted ingress:** write stateful recorder tests for current owner Note and Pointer with viewer OFF, wrong world/frame/tab, retired/reused context, navigation, paused/driver/ended and queued Off/On. Mint owner admission only after current CDP proof; retain final ordered writer identity/generation checks.
- [x] **Prepared notes:** qualify ordinary captured preparation and signed unavailable-capture restoration, stale/cross-owner/cross-role tokens, cleanup held across save, late capture completion and lost ACK. Assert exactly one durable note and unchanged reconciliation; never use retry to create a new operation.
- [x] **UI and CLI:** qualify default owner tool delivery and `SetAnnotationMode` at generation0, stale permission receipts/late enables and inactive contexts. Add viewer OFF/read-only reason tests. Keep real mounted control listeners and keyboard access; no permission inferred from their state.
- [x] **Native regression:** one disposable Chrome run at1728×997 drives actual Note hit-testing, box/capture/save and Pointer with viewer OFF. Assert a session-authenticated viewer POST is still forbidden, including forged owner claims; owner drawing does not flip the global viewer capability. Cover Pen expanded and a short viewport for reachability, and one OFF/ON transition without a full native matrix.
- [x] **Documentation and inventory:** update `atlas/webcast-format.md`, `atlas/webcast-submission.md` and the existing README capability explanation for owner versus viewers, exact proof, cleanup fence and viewer recovery. Audit named affected readers individually and run inventory before close.
**Workflow postconditions:** pushed-head `sdlc local-check run --pre-close` through the machine queue; mandatory full close; hub-sequenced operator recheck before landing. No out-of-queue full suite, native matrix, load stress or count above10. #857's combined stage is kept untouched.

Targeted iterations run one command at a time: relevant Board/recorder operator and prepared-note tests, then shared browser annotation/operator tests, then the single native conformance test. Run `make test-affected-inventory` before each close. Close owns fresh review; no duplicate boundary reviewer is dispatched.

## Architectural decisions

- **ARCH-DRY / ARCH-PURE:** one private authority binding/checker shared by all admission consumers; reuse capability transitions and UI policy rather than a second owner permission service.
- **ARCH-PURPOSE / ARCH-SECURE:** every operator ingress and prepared-note restoration uses the exact CDP owner proof; HTTP viewers cannot inherit owner authority through a label, origin or cookie.
- **ARCH-MOCK / ARCH-ORDER:** stateful held capture/clear/write and context retirement tests, plus native hit testing; preserve generation and ordered cleanup/write checks.
- **ARCH-CONSTRAINTS / ARCH-FUNERAL:** fixed-size authority, existing bounded contexts/queues/drafts and joined cleanup; creates no new durable family or background lifetime.
- **ARCH-EVOLVE:** no persistent/wire schema change; authority remains an in-process value.

## Revisions

### 2026-10-24 — Owner/viewer semantics approved; implementation pending plan approval

TL9a388ca5 selects owner Note default availability, keeping other viewers OFF; TLeed8fb98 extends the same rule to owner Pointer. This revises the original preserve-OFF wording only for a proved operator, not a blanket client bypass. Root-cause evidence rejects overflow at the supplied viewport and identifies default shared-capability gating. Exact historical private UI state remains unavailable; a live recheck is still required. No production implementation has started.


### 2026-10-24 — Plan approved

TL4b0ceec1 approves the exact CDP owner proof, one binding through all preparation/final-write consumers, settled OFF/ON owner generation rule, viewer ON grant, unresolved cleanup fence, visible viewer reason and forgery/retirement/race qualification. Proceed to change-code; implementation remains gated on plan quality.

### 2026-10-24 — Implementation qualification

Change-code inferred quick; TL0edcd963 directs keeping it, with automatic boundary-review upgrade for the final diff. Implementation preserves the approved authority and CDP scope. All deliverable checks are implemented and targeted qualification passes: owner/viewer phase/generation/surface, recorder and held capture, actual signed capacity fallback restoration and unknown ACK, viewer guidance, exact-context4/8 counted deliveries, mounted native owner Note/Pointer, and the pre-existing native operator flow updated for owner semantics. Recorded verification, close and the hub-sequenced operator acceptance remain workflow postconditions, not completed claims.

### 2026-10-24 — BR-1 delivery traceability and BR-2 pinned host evidence

BR-1: reconcile every Core concepts table against the delivered diff. The private authority entity lives in existing board/annotation.go; there is no annotation_authority.go. Admission/prepared bindings change in the existing Board files, while recorder/operator_notedraft.go and the production notedraft service are reused unchanged. Viewer explanation changes annotate.js only; CSS and annotation-policy.js stay unchanged. Native qualification is operator_owner_live_test.go. These are traceability corrections, not design or production changes.

BR-2: reran the reviewer's exact native test command on its pinned95f06244a39371e5d3ec8f68183ec22e4b6a2dcd, with tracked webcast source checked clean before running. Headless/private-profile host run with WEBCAST_LIVE_CHROME=1, Go1.27.3 darwin/arm64 PASS4.46s/package4.709s, exit0. Retain the exact head, argv, environment, host and result in evidence/pinned-owner-native-metadata.json and full output in evidence/pinned-owner-native.log. Reviewer-fenced Chrome GPU startup failure is retained as an environment limit, not product evidence. Operator acceptance remains outstanding.
