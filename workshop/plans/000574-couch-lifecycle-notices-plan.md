---
collaboration-mode: discussion
---

# Couch lifecycle system notices implementation plan

> **For agentic workers:** Follow AGENTS.md §3. One atomic full-flow boundary; independent read-only source exploration informs this design, and the binary owns the final fresh review.

**Goal:** Keep each TL hub informed of its slots' confirmed lifecycle changes, with authentic Couch provenance and one logical restart summary.

**Architecture:** Observe completed lifecycle operations and native child exit, normalize them in one bounded pure notice reducer, and publish through the existing broker/queue/batching path. Add a server-owned message origin protected at the wrapper by the kernel identity of its current broker connection. Existing thread lifecycle and delivery status machines remain authoritative.

**Tech stack:** Go, existing Console operation/reattach reducers, Pair lifecycle controller, Unix peer credentials, wrapper registry sessions, #648 FIFO batches and stateful socket/PTY fakes.

> **Superseded initial proposal:** The material below through the first Revisions entry records the initial proposal, not the implementation contract. The appended “Current trimmed plan” replaces it following hub feedback; removed tasks are not acceptance requirements.

## Decisions and alternatives

- Use completion hooks, not request logs or inventory diffs as a substitute for successful start/park/relaunch. Registration alone precedes Console adoption; attach-client exit alone does not prove native agent exit.
- A short debounce guesses when restart reattachments finish. Use the existing `ReattachPass` drain boundary, including empty, failed and cancelled passes. Collect ordinary events while the hub is busy and reuse #648's delivery batching rather than a second paste/submit implementation.
- A textual prefix or a caller-set `System` flag is forgeable. Keep CLI `Request` closed, set `Message.Origin` only through an internal broker entrypoint, and authenticate system-bearing wrapper calls against the current broker's kernel PID/connection lifetime. Canonical receipts retain origin; copied header text remains peer content.
- A same-user peer can currently call a wrapper endpoint directly, so broker-only labeling is insufficient. Protect system-bearing commit/prepare/finalization before state mutation. Ordinary peer protocol stays compatible; broad endpoint hardening is outside this item.
- Add an authenticated terminal child-exit observation on a negotiated newer registry session. Do not relabel transport EOF, attachment exit, native-query failure or wrapper replacement as `agent exited`.

## Core concepts

### Pure entities

| Entity | File | Status |
|---|---|---|
| Message origin and immutable message identity | `cmd/internal/couchmessage/model.go`, `batch.go` | modified |
| Lifecycle event, initiation and normalization | `cmd/internal/couchcore/lifecycle_notice.go` | new |
| Bounded lifecycle notice/cohort reducer | `cmd/internal/couchcore/lifecycle_notice.go` | new |
| Registry terminal child-exit frame | `cmd/internal/couchmessage/session_protocol.go`, `registry.go` | modified |

`Message.Origin` is omitted/zero for peer messages and has one known Couch-system value. Unknown values refuse. `Send` fixes peer origin; `SendSystem` uses the same admission implementation with system origin. Origin participates in in-flight/completed ID deduplication, queue promotion, retained receipt adoption, singleton/batch receipt matching and prepared-head comparison. `Envelope` renders `[Couch system; delivery ID]` from typed origin, never from body text. `BatchEnvelope` preserves each member's typed header and separator, and Pair logging records exactly the submitted combined text.

Lifecycle events carry a supervisor-generated event identity, canonical slot/thread and incarnation, kind/outcome, observed time, known agent/model/branch/issue, and initiation (`operator`, authenticated peer, automatic, unknown). Unknown metadata stays unknown; cached presentation fields never authorize a transition. Normal rows name slot, event, available metadata and initiation. Values are control-free and field-bounded; no nonce, PID-start credential, native transcript, command or private input is put in the body.

The reducer holds a bounded current-slot map, restart cohort and unsent dirty projection. At most the configured actor/slot bound (128) is resident. One accepted immutable summary per hub may be outstanding; newer events dirty the next projection, never mutate accepted broker input. Before admission, repeated updates for a slot coalesce to its latest confirmed state with an explicit change count; finite ordinary bursts retain event distinctions within the bounded projection. No accepted message or receipt is evicted to make room. Overflow produces a visible resnapshot-required count rather than pretending complete history. This is a current-state notification channel, not a durable audit log.

### Integration points

| Integration | File(s) | Status | Boundary |
|---|---|---|---|
| Server-owned admission | `couchmessage/broker.go`, `broker_batch.go` | modified | existing broker workers, capacities, receipts |
| Kernel origin authentication | `couchmessage/transport.go`, `session_transport.go`, `wrapcmd/peer_runtime.go` | modified | current broker Unix connection and endpoint caller PID |
| Confirmed operation/reattach observations | `couchtty/console.go`, `console_menu.go`, `console_reattach.go`, `console_remote.go` | modified | adoption, operation completion and pass drain |
| Exact park completion | `couchcore/park.go` | modified | successful `FinalizePark`, outside store locks |
| Native agent exit | `wrapcmd/wrap.go`, `peer_runtime.go`, registry session | modified | actual owned child's `Wait` result |
| Notice publisher and hub configuration | `couchcmd/lifecycle_notices.go`, `message_service.go`, `run.go`, `supervision.go` | new/modified | current hub ownership, existing broker and lifetime |

## Authoritative event sources

| Notice | Required evidence | Exclusions |
|---|---|---|
| started/resumed | successful `Console.finishOperation` after implicit adoption; initial attach explicitly seeds an equivalent observation | failed registration/adoption, request acceptance, provisional child |
| parked | exact matched completion, exact process dead, successful `Threads.FinalizePark` | quit request, stale/failed completion, cleanup warning alone |
| relaunched | typed `Relaunched` result plus successful replacement adoption | `ParkIncomplete`, `ParkedNotResumed`, failed adoption |
| recovered | successful Continue/Resume/Fresh outcome and adoption, with mode recorded | preview, pending confirmation, `RecoveryLaunchUnconfirmed` |
| agent exited | authenticated current-wrapper terminal frame derived from native child `Wait` | registry EOF/reconnect, detach/client exit, stale incarnation, unknown wait error |
| needs-relaunch | first confirmed transition into the existing actionable inventory reason | unknown evidence, repeated identical inventory, late replaced incarnation |

Correlate substeps by supervisor operation identity and exact incarnation. A high-level relaunch/recovery emits its final result once; its park/start substeps are not independent duplicate notices. A scoped operation owner reserves normalization for its target until final completion; native exit during that operation is retained as evidence. On partial failure, report the actual parked/failed outcome without claiming a successful relaunch. Uncorrelated genuine facts remain facts: a later park cannot retroactively make an earlier unknown exit operator-initiated.

Operator initiation comes only from Console operator origin. Remote origin is derived from the authenticated admitted caller; background reattach is automatic. Existing quit artifacts without initiation proof report unknown. No request/body field may assert that an operator performed an action.

Hooks copy bounded evidence under their existing owners, then call the collector after releasing Console, store, registry and broker locks. They perform no message IO or Git/native discovery. Existing cached context enriches notices asynchronously; missing model/branch/issue is not filled by per-event subprocesses.

## Restart cohort and delivery ordering

1. Arm collection with the existing startup reattach pass before the first inventory is reduced. Seed initial adopted hub/start context when message service is wired; explicit additional hubs use their existing serialized start path.
2. Each background attempt contributes its confirmed result or bounded failure. A completion cannot be published before adoption.
3. Flush one logical summary per owning hub only when the pass reaches `ReattachDone`; zero-candidate, cancellation and failure paths must close the cohort explicitly. Operator operations may delay the pass; a timer does not substitute for its completion.
4. A missing/busy hub retains only bounded latest facts until that exact hub incarnation can admit a summary. Hub reconnection gets a fresh current snapshot, never a replay of possibly submitted text. Ownership is resolved from the canonical workspace and current `HubOwnership` at freeze, with revision revalidation before admission; moves reroute unadmitted facts.
5. Normal summaries use existing broker FIFO and #648 readiness freeze. A restart with twenty ordinary rows fits one body/turn. Respect 8 KiB body, encoded frame and configured character caps: large cohorts become numbered parts of one logical summary, combined by #648 on supported Claude hubs when the configured cap allows. Codex/Qoder batching remains disabled; do not bypass their rollout policy. Explicitly document this bounded multipart case.
6. Submitted receipts retire only the frozen projection; later dirty facts remain pending. Known pre-write cancellation can produce a new current snapshot. Indeterminate input is never replayed. Closing the service cancels and joins its publisher workers and releases all ephemeral state.

### Reducer transitions

| State/event | Result |
|---|---|
| collecting restart / confirmed fact | update cohort/current facts; no send |
| collecting / pass drained or cancelled | ready summary, with successes/failures and completion reason |
| ready / fact | update bounded unsent projection |
| ready / hub absent, opted out or ownership unresolved | retain latest facts or suppress when explicitly muted; no privileged send |
| ready / freeze and successful broker admission | immutable outstanding ID plus remaining dirty projection |
| outstanding / newer fact | dirty next projection; no payload mutation |
| outstanding / Submitted | retire frozen projection; ready only if newer facts exist |
| outstanding / known not-pasted cancellation or replacement | discard old job; produce current snapshot for new live hub |
| outstanding / Indeterminate | retain uncertainty fact; never replay that frozen job |
| any / stale event, duplicate observation or old incarnation exit | ignore without changing current slot state |
| any / shutdown | cancel/join owned publisher work and discard memory |

## System-origin authority and compatibility

The session client already owns a long-lived Unix connection and a read-loss watcher. Pin the kernel broker peer identity to that exact connection generation, clear it on loss/failed handshake/cancellation, and ensure old cleanup cannot clear a newer lease. Install the pin early enough for admission ordering; gating only system-bearing endpoint operations avoids changing legacy Observe/Reserve behavior. Endpoint transport attaches kernel caller identity to its context. Before accepting a system-bearing singleton, prepared head or any batch member, require the current broker identity; JSON binding equality alone cannot authorize system marking. If process-generation proof is needed to exclude PID reuse during loss propagation, reuse the existing native process-identity seam rather than inventing a secret or trusting a supplied PID.

CLI `Request` stays closed and cannot select origin; strict decoding refuses forged fields. Peer messages containing a system-looking header retain the peer outer header and canonical peer receipt. Hub instructions require checking canonical receipt origin and recipient binding before treating a notice as Couch information. The message is information, never a work assignment, permission or operator approval.

Peer origin remains omitted on the wire. System origin is never stripped: old strict wrappers refuse with explicit relaunch advice, without PTY writes or downgrade-to-peer. Unsupported batch fallback may select singleton delivery only with origin preserved. Current `sendGenerated` callers use canonical system origin for startup availability and other existing supervisor notices; their body semantics remain unchanged. Replace ephemeral system classification in delivery accounting with immutable origin so receipt retention and exact Pair logging stay consistent.

Native exit reporting negotiates a newer hello before sending a new terminal frame. The server verifies the kernel wrapper PID as today, exact current session/incarnation, a bounded exit code and one terminal occurrence. A wrapper reports only after its owned child's successful wait (including an observed exit status), before closing its session. Use a bounded flush on the already connected session under the existing admission/write budget; no reconnect or replay during shutdown. Old brokers decline the new hello and retain the current fallback. If no current capable session or wait proof exists, record unknown/unconfirmed locally; later needs-relaunch evidence remains available. Never claim agent exit from a transport failure.

## Configuration and operating envelope

- Default on for configured TL hubs; no explicit hub means no automatic lifecycle recipient. Exclude the recipient hub's own lifecycle to prevent notification recursion.
- Operator opt-out is `<Couch store>/lifecycle-notice-off`, one canonical hub slot per line with comments, matching the existing wait-notice mute-file convention. Bound file input; unreadable/malformed configuration produces a visible warning and conservative suppression for affected notices. No peer RPC edits it. Document exact path, scope and default in README and `couch --skill`.
- Event callbacks are in-memory work bounded by existing slot count, no blocking IO. A service-owned publisher uses existing admission limits and cancellation; at most one frozen summary per hub. A busy/hung hub cannot block other hubs or lifecycle completion. Coalescing explicitly preserves final slot state/counts rather than promising an unbounded audit trail.
- No new durable event log, credentials, retained process or timer loop for discovering Slots. Existing broker receipts keep their bounded retention. New Go source files enter the exhaustive artifact inventory.
- **ARCH-DRY/PURE:** shared message admission/envelope/identity and one pure event reducer. **ARCH-PURPOSE:** every requested event source plus worker startup provenance, opt-out and docs are included. **ARCH-MOCK:** use existing stateful Console/core/registry/PTY fakes and real owned Unix socket subprocesses. **ARCH-CONSTRAINTS:** explicit caps above, no per-event process scan. **ARCH-SECURE:** kernel provenance plus closed caller requests, no header-text authority. **ARCH-ORDER:** completion/reattach/receipt tables and exact incarnation fences. **ARCH-FUNERAL:** service lifetime owns all new memory, leases and workers; no new durable history.

## Function-level verification strategies

| Function/boundary | Adversarial strategy | Independent assertion |
|---|---|---|
| origin validation, Envelope and immutable identity | generate peer/system/unknown origins, copied headers, ID reuse, mixed batches and altered retained receipts | only server-created system origin receives system heading; origin changes never deduplicate/adopt as the original |
| broker-session lease and endpoint guard | real different-PID caller; failed hello, loss, reconnect, old cleanup, missing credentials, forged batch follower | no forged system input/state change; current broker succeeds; peer compatibility preserved |
| terminal exit frame | controlled owned child exits and errors, malformed/out-of-order/duplicate/stale frames, old negotiation | only actual current native child exit produces one fact; EOF/reexec/detach never substitutes |
| completion normalizer | generate operation/adoption success/failure and exit-before/after-completion sequences with replacements | no provisional success, no duplicate substep, no old incarnation overwrites replacement |
| restart reducer/publisher | empty/20/128 rows, out-of-order attempts, failed/cancelled pass, busy/disconnected/replaced hub, ownership move, opted-out hub, cap overflow | one logical drained cohort, no lost final state, bounded memory, no blocking of unrelated hubs or replay after uncertainty |
| runtime wiring | actual Console and registry fake stack over sockets, exact combined PTY input and per-ID receipts/log | hooks are wired to authoritative paths; summary uses #648 rather than a parallel input writer |

Mutate away origin guards, adoption gating and pass-drain gating independently; each associated negative must fail. Retain all failed evidence before restoring green. Run focused/race packages first, then the required recorded full suite on the pushed code head, close, and normal merge/authoritative suite. Standing hub approval allows an early draft PR for CI overlap, with all acceptance gates still before landing.

## Tasks

- Historical proposed task (superseded): Add failing provenance/receipt/kernel-caller tests; implement typed origin, internal admission, compatibility and exact batch/log preservation. Commit.
- Historical proposed task (superseded): Add failing native-exit and operation/cohort stateful tests; implement authentic exit reporting, post-adoption and exact park hooks, normalized event reducer and bounded publisher. Commit.
- Historical proposed task (superseded): Wire startup cohort completion, multi-hub ownership and per-hub opt-out; exercise real socket/Console fake integration and negative controls. Commit.
- Historical proposed task (superseded): Update README, atlas/index/Couch map, `couch --skill`, and the source-owned TL lead help (an ariadne companion coordinated by the hub, required before this issue closes), add artifact inventory entries, then focused/race and recorded suite verification. Close and land through normal gates.

## Live arrangement requiring hub approval

Propose isolated owned Console/registry/wrapper/socket checks before close, with controlled fake/native shell children and no existing slots/focus/clipboard. The real operator restart then checks a normal start, park/relaunch, native exit and recovery cohort on the rebuilt binary; hub verifies one summary and treats it as Slots information without replying. Request explicit deferral of that operator check to the next hub-run restart, with no :0 staging. If the hub requires a pre-landing real smoke, stage only after it schedules the operator and shared native lane.

## Revisions

- 2026-10-17 — Initial design for approval. Read-only exploration found provisional launch/adoption and attachment/native-exit distinctions, and the direct-endpoint spoofing hole for any new system flag. These are addressed in the plan before code; no implementation changed.

- 2026-10-17 — Hub feedback 2b740f16-29c2-4e8a-89ee-6571e0a4dcb2, verified Submitted: simplify before code. The current plan below supersedes the initial architecture, entity tables, tasks and live proposal. Remove wrapper PID authentication, broker-connection leases, the negotiated native-exit frame and per-hub opt-out storage. Native-agent exit is deferred to unclaimed #657 because attachment exit is insufficient evidence. Owned-child checks now and operator smoke at the hub's next restart are approved; the hub owns the ariadne tl-lead companion.

## Current trimmed plan

**Goal:** Send confirmed slot lifecycle information to its existing owning hub, coalescing restart reattachments, through the existing message broker.

**Architecture:** Existing successful lifecycle boundaries feed a bounded collector. Its internal publisher creates system-origin records and sends them through the existing broker and #648 batching. Canonical server receipts establish origin; the closed external request cannot select it. No terminal/session protocol extension or wrapper process authentication is added.

### Core concepts

| Pure entity | Lives in | Status |
|---|---|---|
| Message origin and immutable identity | `cmd/internal/couchmessage/model.go`, `batch.go` | modified |
| Lifecycle fact and bounded collector | `cmd/internal/couchcore/lifecycle_notice.go` | new |

Origin is omitted/zero for peers, with one known system value and unknown values rejected. External sends always construct peer-origin messages; only the server's internal generated-notice producer constructs system origin. Propagate this immutable field through queues, retained receipts, deduplication, batch comparisons and envelopes. The wrapper receives it in the server's delivery record. A copied heading is ordinary peer body text; canonical receipt origin is the authority, not text. A direct endpoint payload cannot create a canonical server system receipt. Preserve origin on singleton fallback; an old strict wrapper refuses unsupported system records without a write or silent downgrade.

The collector owns at most the existing slot bound (128) of current facts and dirty updates, plus one frozen outstanding summary per existing hub. Facts carry exact slot/incarnation, kind, operation identity, known cached metadata and initiation (operator/peer/automatic/unknown). Unknown metadata stays unknown. For repeated unsent slot changes, coalesce to current confirmed state plus change count; this is a Slots update, not an audit history. Freeze accepted payloads: later facts belong to the next projection. Do not change existing thread or receipt status enums.

### Integration points

| Integration | Lives in | Status | Wraps |
|---|---|---|---|
| Internal notice admission and accounting | `cmd/internal/couchcmd/supervision.go`, `message_service.go`; `couchmessage/broker.go`, `endpoint.go`; `wrapcmd/peer_batch.go` | modified | existing broker admission, immutable matching and exact delivery log |
| Confirmed completion observations | `cmd/internal/couchtty/console.go`, `console_menu.go`, `console_reattach.go`; `couchcore/park.go` | modified | final adoption, successful park, existing inventory and restart pass |
| Collector publisher | `cmd/internal/couchcmd/lifecycle_notices.go`, `run.go` | new/modified | existing hub routing, lifetime cancellation, configuration and broker |

Use existing stateful Console/core/registry/PTY fakes and owned Unix sockets/children. No alternate paste writer, new periodic discovery, native scan or per-event Git subprocess. Callbacks copy evidence and invoke the collector outside existing locks. One service-owned worker performs admission and joins on shutdown; preserve the existing hub ownership resolver without a new ownership store. Extend the collector later for #657 without prebuilding a terminal protocol.

### Event evidence and ordering

| Event | Confirmed boundary |
|---|---|
| started/resumed | final successful Console operation after adoption; initial attach explicitly seeds equivalent evidence |
| parked | exact quit completion and successful `FinalizePark`; cleanup warnings do not undo the fact |
| relaunched | typed `Relaunched` plus successful replacement adoption |
| recovered | successful Continue/Resume/Fresh result plus adoption, preserving mode |
| needs-relaunch | transition into the existing confirmed actionable inventory reason; repeated/unknown/stale observations do not notify |

Attachment exit is not an agent-exit event. #657 owns reliable native-agent exit evidence and its tests. Failed adoption never produces success. Normalize a relaunch/recovery's correlated substeps into its final notice, without duplicate park/start notices; partial failures may report confirmed park without claiming successful relaunch. Operator initiation requires existing operator-origin evidence, otherwise say peer, automatic or unknown.

Arm restart collection with the existing reattach pass. Flush at `ReattachDone`, including empty, failed and cancelled outcomes, never a debounce deadline. Twenty normal rows should form one summary. Format bounded fields and respect the existing 8 KiB message body limit; a larger snapshot uses numbered parts and existing #648 capped FIFO batching. Claude batching remains enabled as today; Codex/Qoder remain singleton. Pair logging retains the exact combined submitted turn.

A busy or absent hub retains bounded unsent current facts; it does not block lifecycle completion. Freeze only for the current resolved hub incarnation. Submitted receipts retire the frozen projection, leaving newer changes dirty. Known pre-write cancellation may produce a fresh current snapshot; indeterminate input is never replayed. Stale incarnations cannot overwrite replacements. Shutdown cancels and joins publication and discards ephemeral state.

### Configuration, docs and verification

One process setting: `COUCH_LIFECYCLE_NOTICES`, default on; explicit `off` disables lifecycle notices. Accept `on`/`off` only, with an actionable startup error for other nonempty values. No per-hub mute file. No configured hub means no lifecycle recipient; avoid self-notification loops. Existing worker startup availability notices also use the internal system producer, retaining their body. Do not reclassify unrelated peer sends or expand other generated-notice behavior unnecessarily.

Update README, `couch --skill`, atlas/Couch map and index, and artifact inventory for new files. Hub supplies the ariadne tl-lead companion (Slots information, no reply); record its link and completion before close.

| Risky function | Adversarial strategy and independent guard |
|---|---|
| `Envelope`, `ValidateBatch`, `Advance` | Generate origin/body/identity mutations; only known typed origins render, and identity conflicts leave ownership unchanged. |
| `Broker.send`, `adoptRetained`, `promote`, `StatusContext`, `matchingReceipt`, `finalizeBatch` | Mutate immutable origin across admission and retained-receipt recovery; compare canonical server evidence and assert unknown system records never enter server receipt state. |
| `AdvanceLifecycleNotices` (new sole collector transition) | Generate reordered facts and publication outcomes through bounded state; independently assert size bounds, adopted final state, immutable frozen parts and no replay after uncertain input. |
| `Console.observeLifecycleCompletion` (new), `PairLifecycleController.applyCompletion` | Drive stateful operation/adoption failures and stale exact completions; success requires the existing adoption/finalization authority and attachment exit cannot emit agent exit. |
| `Console.observeLifecycleInventory` and `observeLifecyclePass` (new) | Generate confirmed/unknown inventories and pass progress; require real reason transitions and the drain barrier before publication. |
| `lifecycleNoticesEnabled` (new) | Generate arbitrary setting values; only empty/on/off accepted with documented default. |
| `lifecyclePublisher.run` (new thin shell) | Owned Console/registry/socket stack with controlled receiver readiness/loss; assert actual hook wiring, broker/#648 exact combined turn and per-message receipt provenance. |

Negative controls remove origin matching, adoption gating and pass-drain gating independently; the corresponding tests must fail. ARCH-DRY: existing message path and routing. ARCH-PURE: one collector reducer. ARCH-ORDER: explicit adoption/pass/receipt boundaries. ARCH-SECURE: internal origin, closed external requests and canonical receipts. ARCH-FUNERAL: service owns bounded state and joins its worker.

### Current execution steps (one close boundary)

- [x] Write and run failing origin/admission/receipt/batch tests; implement internal origin and delivery preservation; run targeted couchmessage/wrapcmd tests and commit.
- [x] Write failing pure collector and stateful lifecycle tests; implement confirmed completion hooks, restart drain and bounded publication; run focused core/tty/cmd tests and commit.
- [x] Wire the single setting and startup availability origin; exercise owned socket/Console children, failure cases and negative controls; update docs/inventory and record the hub's tl-lead companion.
- [ ] Run focused/race checks, push and open an early draft PR for CI overlap, then recorded full suite, close review and normal merge gates. Never hand-post gate status.

**Approved live arrangement:** Owned children and isolated sockets now, no existing slot focus/clipboard input. Real start/park/relaunch/recovery and restart-summary smoke runs with the operator at the hub's next restart after landing, with no pair:0 staging. Native-agent exit is outside that smoke and outside this issue. Plan revision goes to the hub before change-code; full-flow plan-quality acceptance precedes the estimate and implementation.

## Revisions — plan-quality round 1

- 2026-10-17 — PQ-1 (ARCH-SECURE): corrected the claim that canonical recovery already excludes endpoint forgery. `Broker.StatusContext` currently imports unknown wrapper-retained receipts (`broker.go:957`); all import paths must enforce the server-origin rule below. No wrapper authentication or new protocol is introduced.
- PQ-2 (ARCH-ORDER): defined publication failure and multipart transitions below. PQ-3: replaced the current verification case inventory with named-function strategies. These refine the approved mechanism and do not restore deferred scope.

### System-origin recovery rule

The broker's existing admitted receipt/active queue is the only system-origin authority; an in-progress caller admission alone is not evidence of an earlier delivery. Unknown retained system records are not imported, after restart or receipt eviction either. `StatusContext` returns existing uncertain/refusal wording without recording such a receipt; `adoptRetained` refuses unknown system records even if the caller now requests system origin. `promote` may adopt a retained system receipt only when its full immutable identity matches the already server-admitted queue head (origin included); otherwise cancel with a conflicting-ID diagnostic and retain the canonical original. Unknown origins reject on every path. Legacy peer receipt recovery remains supported. No recovered system record is silently relabeled peer. Direct endpoint commit followed by status/adopt/promotion must not manufacture server authority. Test the complete class of receipt imports, not only rendering.

### Collector transition contract

`LifecycleNoticeState` contains bounded current facts, dirty revisions, restart collection phase, per-hub frozen parts and receipt outcomes. `LifecycleNoticeEvent` supplies a confirmed fact, pass-drained result, hub-ready change, admission result, receipt result or shutdown. `AdvanceLifecycleNotices(state,event)` is the sole deterministic transition, returning the next state and `LifecycleNoticeEffect` values (publish part or local diagnostic); the service shell supplies IDs, time and broker outcomes.

| Event | Transition/effect |
|---|---|
| confirmed fact | Update at most 128 slot entries and their dirty revisions. Duplicate/stale facts do nothing. When full, keep existing entries and increment a bounded saturating overflow count; the next summary includes the count and a resnapshot instruction. |
| restart drain | End collection and make current dirty projection eligible; empty projection sends nothing. Failure/cancellation is included as bounded summary metadata. |
| ready, live hub, no outstanding part | Freeze current projection into bounded numbered parts (at most 128, one bounded row per slot plus header/counts); send one part. New facts remain separately dirty. |
| admitted part | Keep exact immutable payload and ID until its terminal receipt; do not submit another part for this hub concurrently. |
| admission refusal/unavailable/capacity/unsupported | No receipt owns input: discard that unsent frozen part set back into current dirty facts, emit one bounded local diagnostic, and suppress retries until a new confirmed fact or hub incarnation/readiness change. Unsupported stays suppressed for that hub incarnation until replacement. No timer retry loop. |
| Submitted part | Retire only that part's frozen revisions; publish next unsent part if any. Later dirty updates survive. A partly submitted summary never resends submitted parts. |
| Expired/Cancelled before paste | Already submitted parts stay retired; unsent/known-unpasted revisions return to the current projection. Emit diagnostic and wait for new fact/hub readiness change before a new snapshot; use new ID. |
| Indeterminate or uncertain admission without canonical no-write proof | Retire the affected frozen part as uncertain, emit diagnostic; never replay its bytes/revisions. Continue only with untouched unsent parts or later genuinely new revisions. Unknown admission IDs are polled read-only while service lives; never blindly retry. |
| recipient replacement | Cancel old exact-incarnation job through existing broker rules. Apply its terminal evidence as above; route untouched unsent/current facts through existing ownership to the new live hub. |
| opt-out/shutdown | Suppress/discard unsent projections; shutdown cancels and joins shell work. Already accepted input follows existing broker cancellation semantics. |

The existing body/encoded-frame/cap constraints apply at freeze. Formatting uses bounded fields so each row fits; body parts are complete rows, not truncated events. At most one part is outstanding per hub, preserving FIFO and bounded receipt use; #648 still batches with other queued messages at readiness. A new current snapshot may mention a later changed slot state after an uncertain part, but does not replay that part or claim the earlier event was delivered.

## Revisions — implementation evidence

- 2026-10-17 — Reuse the existing final Console ParkResult carrying VerifiedPark instead of installing a second callback in PairLifecycleController; its success is already produced only after FinalizePark. This removes duplicate event authority and relaunch substep notifications (ARCH-DRY). Callback observations remain post-adoption. At restart drain, facts lacking any typed slot target or verified cached identity are diagnosed locally rather than guessed or allowed to hold the cohort indefinitely. No external discovery is added.
