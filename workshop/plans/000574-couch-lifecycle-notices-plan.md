---
collaboration-mode: discussion
---

# Couch lifecycle system notices implementation plan

> **For agentic workers:** Follow AGENTS.md §3. One atomic full-flow boundary; independent read-only source exploration informs this design, and the binary owns the final fresh review.

**Goal:** Keep each TL hub informed of its slots' confirmed lifecycle changes, with authentic Couch provenance and one logical restart summary.

**Architecture:** Observe completed lifecycle operations and native child exit, normalize them in one bounded pure notice reducer, and publish through the existing broker/queue/batching path. Add a server-owned message origin protected at the wrapper by the kernel identity of its current broker connection. Existing thread lifecycle and delivery status machines remain authoritative.

**Tech stack:** Go, existing Console operation/reattach reducers, Pair lifecycle controller, Unix peer credentials, wrapper registry sessions, #648 FIFO batches and stateful socket/PTY fakes.

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

- [ ] Add failing provenance/receipt/kernel-caller tests; implement typed origin, internal admission, compatibility and exact batch/log preservation. Commit.
- [ ] Add failing native-exit and operation/cohort stateful tests; implement authentic exit reporting, post-adoption and exact park hooks, normalized event reducer and bounded publisher. Commit.
- [ ] Wire startup cohort completion, multi-hub ownership and per-hub opt-out; exercise real socket/Console fake integration and negative controls. Commit.
- [ ] Update README, atlas/index/Couch map, `couch --skill`, and the source-owned TL lead help (an ariadne companion coordinated by the hub, required before this issue closes), add artifact inventory entries, then focused/race and recorded suite verification. Close and land through normal gates.

## Live arrangement requiring hub approval

Propose isolated owned Console/registry/wrapper/socket checks before close, with controlled fake/native shell children and no existing slots/focus/clipboard. The real operator restart then checks a normal start, park/relaunch, native exit and recovery cohort on the rebuilt binary; hub verifies one summary and treats it as Slots information without replying. Request explicit deferral of that operator check to the next hub-run restart, with no :0 staging. If the hub requires a pre-landing real smoke, stage only after it schedules the operator and shared native lane.

## Revisions

- 2026-10-17 — Initial design for approval. Read-only exploration found provisional launch/adoption and attachment/native-exit distinctions, and the direct-endpoint spoofing hole for any new system flag. These are addressed in the plan before code; no implementation changed.
