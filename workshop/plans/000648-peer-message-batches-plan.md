---
collaboration-mode: discussion
---

# Queued peer-message batches implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 for execution strategy. Use @superpowers-executing-plans for the queue/wrapper work, which shares ownership and cancellation context. The SDLC gates own the fresh-context reviews. Plain checkboxes below belong to one atomic close boundary.

**Goal:** A busy hub receives its queued reports in capped FIFO batches, with one paste, one submit and one confirmed outcome per batch, while retaining each message's identity and receipt.

**Architecture:** Keep the existing receiver worker and exclusive wrapper input writer. Prepare the head without allowing it to paste; when that writer recognizes a safe composer, freeze a prefix of the broker's current queue and finalize that batch exactly once. All members share delivery ownership and confirmation, while receipt lookup, allowance accounting and provenance remain per message.

**Tech stack:** Go reducers, existing Unix-socket endpoint protocol, existing stateful wrapper/PTY fakes, opt-in real Claude Code/Codex PTYs.

## Evidence and scope

The operator's #648 Spec is the requirements approval. Implementation waits for the hub to approve this plan and for `sdlc change-code`.

On 2026-10-17 isolated, newly launched clients received bracketed paste without a prompt submission. Native Ctrl+G exported the complete composer to an owned editor; byte equality and SHA256 comparisons established retention, rather than relying on collapsed paste markers. All owned PTYs exited afterwards. No existing conversations, clipboard or slot binaries changed.

| Client | Exact ASCII sizes in characters | Exact mixed Unicode sizes in code points |
|---|---|---|
| Claude Code 2.1.302 | 8,192; 16,384; 32,768; 65,536; 131,072; 262,144; 524,288; 1,048,576 | 32,768 (51,863 UTF-8 bytes; emoji, CJK, combining accents, newlines) |
| Codex CLI 0.163.0 | 1,024; 8,192; 16,384; 32,768; 65,536; 131,072; 262,144 | 16,384; 32,768; 65,536, including a final combining-mark sample of 144,148 UTF-8 bytes |

These are measured lower bounds, not discovered rejection ceilings or model context limits. Choose **16,384 Unicode code points**, including headers and separators, as the default: half the smaller common mixed-text bound and one sixteenth of the smaller ASCII bound. Accept `COUCH_PEER_BATCH_MAX_CHARS` from 1 through 32,768; unset uses the default, invalid/out-of-range values refuse Couch startup with a specific configuration error. Counting code points matches Go's `utf8.RuneCountInString`; a combining mark counts separately. Serialized bytes remain separately bounded by the existing transport limit.

Evidence: Claude manifest `/var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair648-claude-paste.CHhdj9/manifest.json`; default-size input/export SHA256 `a2bb5fe57654d3441963f51c6de37110f16117f16a39fe1951fa8fce095bdf2f`, mixed-text SHA256 `d16d58648023e570b692f03d9258022181ccddbdac5201e87aa0b14b344dc31b`. Codex manifest `/tmp/pair648-codex-paste-measurement.json`, SHA256 `5ef8ad46a74f28d195639e9cfcc79d26ba73cbda762fe86688aabd247716a516`; default-size input/export SHA256 `0a74ed1b9052198663a441cd4ddd701fd48efa3c845fd2e61c408574e7604bb3`; final mixed-text SHA256 `2b85b97c5e02268d37f36d086d5058a69a005a2977ec2eb7a57f4276fe1bfe3b`. The initial Codex clearing probe cleared only the last line; its cumulative results are excluded. Corrected editor-reset runs supply every reported result. Scratch captures contain generated text only; the durable evidence is this table/method/hashes, not a dependency on temporary artifacts.

**#629 note:** Its archived plan says same-receiver messages share composer availability, so shuffling a blocked head cannot make a follower safe; it explicitly excluded batching unrelated envelopes. Its issue Log likewise says no body batching. Neither artifact contains a measured composer incompatibility. The actual implementation reserves, owns, confirms and retains one message ID. Batching therefore requires shared ownership and per-member receipt retention, not merely concatenating bodies. This plan replaces that single-message assumption for the selected hub path.

**Rollout scope (revised with TL approval):** Batch only verified hub bindings running Claude. Ordinary receivers, Codex, Qoder (whose submit retry additionally matches an entire owned envelope), and unsupported profiles retain the existing path. Codex's tall-paste confirmation defect is filed separately as #652; passing retention measurements do not establish confirmation safety. Old wrappers retain singleton delivery through the explicit compatibility refusal described below. Wider rollout is separable; the motivating busy Claude hub is included here.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|---|---|---|
| Peer batch rendering and prefix selection | `cmd/internal/couchmessage/batch.go` | new |
| ActorState pending delivery group | `cmd/internal/couchmessage/model.go` | modified |
| Batch preparation phase | `cmd/internal/couchmessage/batch.go` | new |
| Batch endpoint request validation | `cmd/internal/couchmessage/endpoint.go` | modified |

- **Rendering/selection:** One source owns both singleton envelope rendering and batch rendering. Join complete unchanged envelopes with `\n\n---\n\n`. Select the longest contiguous prefix starting at the active head that fits the code-point cap and the actual encoded finalization request's `MaxFrameBytes` bound. Always include the head alone if it exceeds the character cap; never skip an oversized follower to fit a later message. At most one active head plus `MaxQueued` followers. Unit tests need no IO mocks. The renderer is reused by PTY writes, logs and conformance tests (ARCH-DRY, ARCH-PURE).
- **Pending group:** Keep `Pending()` as the head for existing routing/progress callers; represent joined followers explicitly and expose the full active group for cancellation/accounting. A `FreezeBatch` event accepts only the head plus a contiguous queued prefix, removes those followers from the queue, preserves each body/binding/AcceptedAt and gives newly promoted followers fresh exchange deadlines. Progress/finish emit a receipt for each member in one reducer transition. Admission still charges one allowance unit per original message. Duplicate IDs cannot consume a second unit. Existing receipt statuses do not change.
- **Preparation phase:** Use one explicit internal phase (`preparing`, `ready`, `frozen`), not independent booleans. The writer's ordinary eligibility observation offers assembly once by moving preparing to ready; that permission stays offered even if eligibility subsequently changes. Finalization freezes membership irreversibly and the ordinary writer still checks current eligibility before any paste. These internal phases do not introduce receipt status strings. Unit-test invalid/late transitions and forbid paste in both unfrozen phases.
- **Validation:** Typed prepare/status/finalize operations validate exact receiver binding, matching head, distinct valid IDs, original valid bodies, bounded count and encoded frame size. A singleton `commit` remains byte-compatible. Finalization attaches to the already committed prepared head, so a long composer wait does not revive its expired initial-exchange deadline or expire accepted mail.

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| BatchDeliveryEndpoint and RemoteEndpoint | `cmd/internal/couchmessage/endpoint_batch.go` | new | one prepare and one finalization exchange, read-only status polling |
| Broker group delivery ownership | `cmd/internal/couchmessage/broker.go`, `broker_batch.go` | modified | existing receiver worker, mutex and jobs |
| Wrapper batch preparation and retention | `cmd/internal/wrapcmd/peer_batch.go`, `peer_delivery.go`, `peer_runtime.go` | modified | exclusive PTY input writer and endpoint server |
| Hub policy, configuration and per-message accounting | `cmd/internal/couchcmd/message_service.go`, `message_batch.go` | modified | verified registry bindings, supervision and delivery logs |
| Exact combined Pair-log turn | `cmd/internal/wrapcmd/peer_turn_log.go` | modified | existing locked Pair-log writer |

The stateful `FakeDeliveryEndpoint` gains an optional batch-capable counterpart with explicit busy/ready, prepared/frozen membership, output and confirmation gates; singleton tests keep their current fake. Existing `harnessSessionFake` exercises the real wrapper input loop and terminal model. Real-socket endpoint tests connect those stateful seams; no function-call mocks stand in for composer behavior (ARCH-MOCK).

## Delivery protocol and concurrency

1. Admission/reservation stays immediate and per-message. A selected hub head uses a batch-aware endpoint seam; others call existing `Deliver`.
2. `prepare-batch` installs only the already reserved head in the wrapper. It cannot paste. The normal input writer evaluates the existing composer/draft/picker/image/visibility/input/quarantine rules; at the point it would issue the first paste, it instead publishes preparation readiness and waits. This reuses the existing readiness decision, rather than a second approximation of it.
3. The worker polls a batch-specific read-only status. On readiness, its freeze callback takes the broker mutex, cancels any now-disconnected queued senders, selects the capped current FIFO prefix and applies `FreezeBatch`. All group IDs now point to the same cancellation job. The mutex is released before socket IO. A report arriving after freeze belongs to the next batch.
4. The endpoint sends `commit-batch` **once**. The wrapper validates every member before mutating the prepared delivery, rejects any retained/reused member ID without pasting, stores immutable membership, then enables the ordinary input writer. If the composer changed meanwhile it waits under the existing receiver-lifetime rules, with all members still grouped. Readiness is permission to assemble, not permission to bypass the final input checks.
5. The input writer renders one complete batch, sends one bracketed paste, observes the existing fixed paste-to-submit pause, submits once and requires existing composer/turn confirmation. The same existing 30-second post-paste budget covers the whole group. No timeout bump, automatic retry or body replay is introduced.
6. One confirmed terminal outcome is projected to every original receipt under the broker mutex. Wrapper `status ID` returns the corresponding original member plus the shared outcome, and moving to another delivery retains **all** member receipts in the existing 64-ID bounded recent store. A broker restart can therefore recover a follower by its own ID.
7. `prepare-batch` is also the compatibility probe: only an explicit old-wrapper `unknown endpoint operation` refusal, which occurs before any commit, may fall back to legacy singleton `Deliver`. A transport/decode/identity error is uncertain and never falls back. New fields are omitted from legacy request/response shapes; batch readiness appears only in the new operation's reply. Test both old/new directions without a session hello-version change.

The **single resource** excluding competing input is the existing wrapper input ownership/reservation, held from preparation through confirmation/release; a batch is one owner, not N concurrent owners. Broker actor reservation remains held through bounded Release cleanup outside the shared mutex (#631).

| Event | Policy and result |
|---|---|
| New report while busy / after readiness / after freeze | Queue FIFO; pre-freeze reports may join, post-freeze reports cannot mutate membership. No artificial accumulation delay. |
| Draft, picker, image, scrollback or new operator input | Existing hold/quarantine rules remain authoritative; no paste over a draft. Input after a paste follows existing uncertainty protections for the whole group. |
| Sender disconnect before freeze | Remove only that sender's queued messages; if it owns the prepared head, cancel that attempt and continue remaining queue. |
| Any group sender disconnect after freeze | Cancel the one shared job and conservatively end the entire group together; once handed to the wrapper, use indeterminate, never claim another member submitted. |
| Receiver disconnect/replacement or Couch shutdown | Cancel exact actor lifetime; queued/unhanded mail is cancelled, handed group is indeterminate. Late callbacks cannot rewrite terminal receipts. |
| Partial paste/submit, lost finalization reply or failed confirmation | All members share the existing non-submitted outcome; no retry of an uncertain write. Unjoined queue remains available for the next safe composer. |
| Reused ID or mismatched batch | Reject before any new paste; do not overwrite retained wrapper evidence or replay a retained member. The broker must never infer success for unrelated members. |

No detached batch workers or new persistent queue are created. Prepared state dies with the exact wrapper; each worker's socket polls end with its actor context. Members are bounded by the existing queue; receipts retain existing capacity/expiry rules (ARCH-CONSTRAINTS, ARCH-STATE, ARCH-FUNERAL).

**Pair-log correlation:** Record the exact combined envelope as **one** user turn on submission, using the head ID as the idempotent append ID. Recording each envelope as a separate turn would break transcript hash correlation (#618). Keep the existing eight-turn count bound and add a 256-KiB aggregate payload budget per wrapper launch; the largest qualified batch is at most 32,768×4 UTF-8 bytes and fits. Never persist a body on a failed delivery. Tests compare the combined persisted turn to the pasted bytes and verify the aggregate writer bound.

## Implementation and verification

### Function-level test contract

| Production function | Adversarial strategy and mechanical guard |
|---|---|
| `couchmessage.Envelope`, `BatchEnvelope` | Fuzz valid Unicode message text; independently construct expected unchanged headers and separators and compare complete output. |
| `couchmessage.SelectBatch` | Fuzz bounded valid FIFO queues, character caps and JSON-escaped identities/bodies; assert the result is the maximal contiguous prefix under both the code-point and actual encoded-request bounds, except the documented over-cap singleton. |
| `couchcmd.peerBatchMaxChars` | Table-test the parser's integer domain and absent/malformed input classes; assert the precise startup error or allowed range, independently of batching implementation. |
| `couchmessage.Advance` with `FreezeBatch` | Generate admission/freeze/progress/cancellation/finish/disconnect sequences; assert per-ID allowance, immutable original identity, FIFO membership, one group outcome and terminal-receipt monotonicity. |
| `couchmessage.AdvanceBatchPreparation` | Generate eligibility/finalization/late events through the tagged preparation phase; assert no unfrozen state can produce a paste and frozen membership never changes. |
| `couchmessage.ValidateEndpointRequest`, `ValidateBatch` | Fuzz operation/payload mismatches and mutated identity/membership against a valid batch seed; strict decoding and encoded/count/body bounds must reject before any state or input effect. |
| `Broker.freezeBatch`, group delivery/cancellation path | Use the stateful fake's busy, readiness, freeze, finalization and confirmation gates to force competing arrivals and sender/receiver loss on both sides of freeze; assert complete per-group accounting and independent receiver progress. |
| `RemoteEndpoint.DeliverBatch`, `messageEndpoint.DeliverBatch` | Real Unix sockets plus a stateful recipient drive partial/lost/mismatched replies and explicit legacy refusal; count prepare/finalize effects and prove uncertain exchanges never trigger fallback or replay. |
| `peerDelivery.prepareBatch`, `finalizeBatch`, `retainedLocked`, `proxy.dispatchPeer` | Existing terminal/session fake controls eligibility and input ownership between protocol steps; compare complete output writes with an independent FIFO oracle and require one confirmation before any member can report submitted. |
| `peerTurnRecorder` | Compare the persisted user turn with the exact combined paste; drive enough variable-width turns to hit both writer budgets and assert bounded total persisted bytes/count. |

### Task 1: Pure selection, group transitions and configuration

Files: create `cmd/internal/couchmessage/batch.go`, `batch_test.go`, `model_batch_test.go`; modify `model.go`; create `cmd/internal/couchcmd/message_batch.go`, `message_batch_test.go`.

- [x] Write failing tests for the pure/configuration functions above, using their stated generated-input and transition strategies. Include the issue's cap-split and over-cap singleton acceptance oracles.
- [x] Run `go test ./cmd/internal/couchmessage ./cmd/internal/couchcmd -run 'TestPeerBatch|TestActorBatch|TestPeerBatchConfig' -count=1` and retain the initial failure.
- [x] Implement pure rendering/selection and group reducer transitions; extract singleton rendering into the shared owner. Parse configuration at existing message-service startup and enable batching only for verified Claude hub bindings. Preserve existing public statuses and the eight-unit ordinary allowance.
- [x] Re-run the targeted tests and existing queue/model/routing tests; commit the coherent pure/configuration change with required trailers.

### Task 2: Freeze-at-readiness transport and wrapper ownership

Files: create `cmd/internal/couchmessage/endpoint_batch.go`, `endpoint_batch_test.go`, `broker_batch.go`, `broker_batch_test.go`; modify `endpoint.go`, `broker.go`, `cmd/internal/couchcmd/message_service.go`; create `cmd/internal/wrapcmd/peer_batch.go`, `peer_batch_test.go`; modify `peer_runtime.go`, `peer_delivery.go`, `peer_turn_log.go` and its tests.

- [x] First add the required failing stateful-fake scenario: hold the head, admit four more reports, expose readiness, observe exactly one FIFO paste and one submit, then one confirmation yielding five individually visible submitted receipts. Explicit gates force late-arrival, disconnect and lost-reply interleavings; no sleeps are used to manufacture order.
- [x] Add the controlled-ordering and protocol tests for the integration functions above. The independent invariants are no receipt success before shared confirmation, no uncertain batch replay, preserved per-ID recovery and input ownership, and progress of unrelated receivers.
- [x] Run `go test ./cmd/internal/couchmessage ./cmd/internal/wrapcmd ./cmd/internal/couchcmd -run 'Test.*Batch' -count=1` and retain the failures before implementation.
- [x] Implement the optional batch endpoint seam with one freeze callback; add prepare/status/finalize validation and the wrapper's writer-owned preparation barrier. Share existing readiness, delivery reducer, ownership and post-paste confirmation. Keep old endpoint singleton behavior intact.
- [x] Fan out existing supervision/facts/delivery recorder accounting per original message through one shared accounting helper. Update the Pair-log recorder for one exact batch turn and bounded total bytes.
- [x] Exercise real Unix sockets through the wrapper endpoint, including explicit legacy unknown-operation fallback and no fallback on uncertain errors. Run `go test ./cmd/internal/couchmessage ./cmd/internal/wrapcmd ./cmd/internal/couchcmd -count=1`; resolve reproducible defects, following the flake rule for established intermittent failures. Commit.

### Task 3: Documentation, recorded suite, real hub and landing

Files: `atlas/couch.md`, its existing `atlas/index.md` entry, `cmd/internal/couchcmd/skills/couch/SKILL.md` (embedded protocol source); issue Log; relevant lesson if a class of design mistake is found.

- [x] Document the scope/cap, complete-batch receipt semantics, compatibility and character-vs-byte bound. Extend the existing opt-in `TestPeerLiveConformance` scenario to exercise a complete representative batch in an owned new Claude PTY; use the same renderer as production. Preserve both clients' exact full-composer retention measurements, but decline Codex batch qualification until #652. No live prompt submission is needed for retention; the real hub check below proves submission/turn count.
- [ ] Push the implementation and run `sdlc local-check run --name suite` on the issue branch; keep the required recorded suite and all attempt IDs. Close the tested branch through the normal fresh-context SDLC review, without manually integrating newer main or bypassing gates.
- [x] **Ask the hub to approve the staging/restart arrangement before touching :0.** The live check is required before landing unless the hub explicitly arranges a deferred restart check. Do not infer a #648 exception from earlier issues' historical approvals.
- [ ] With the operator/hub coordinating the restart onto the selected build, make the hub busy, send ten small numbered reports (aggregate complete envelopes below 16,384 code points), retain ten receipt IDs, and verify held/queued positions while busy. When the composer returns, verify FIFO separated headers in one user turn (at most two if a boundary arrival misses freeze), one shared confirmation and ten submitted receipts. Record the running binary revision, receiver version and observed turns; do not restart or drive the live hub unasked.
- [ ] On SHIP and satisfied live-check arrangement, run `sdlc merge --yes`, let the landing line own main integration and authoritative suite/CI, and report the verified landing and PR to the hub. Treat unknown failures as blockers. Do not waive CI or apply operator verification labels.

## Alternatives considered

- Concatenate bodies at admission: misses reports arriving during the held head's long turn and loses independent receipt/provenance and log correlation.
- Drain only the already queued followers after the head submits: still spends an extra turn on every busy period and fails the five-messages/one-paste requirement.
- Move the entire broker queue into a second mutable wrapper queue: duplicates FIFO/cancellation/allowance ownership and permits membership races after a paste. The preparation barrier lets the existing broker own selection and the existing input writer own input.

The chosen design adds a preparation barrier and immutable group; it does not add a second scheduler or copy readiness policy (ARCH-DRY). Atomic receipts and retained member identities fulfill the motivating hub goal rather than only reducing some submits (ARCH-PURPOSE).

## Revisions

- 2026-10-17 11:24 PDT — Plan-quality round 1, PQ-1: replaced test-case inventories with named production functions, one adversarial strategy/mechanical guard per risky function, generated reducer sequences and explicit IO ordering gates. Preserved the required five-message acceptance scenario. Located the embedded Couch protocol's exact source. No mechanism, scope, cap or ownership change from the hub-approved design.
- 2026-10-17 11:29 PDT — Implementation clarification: readiness is a sticky one-time permission to assemble, not ongoing composer eligibility. Otherwise an ordinary repaint between the offered readiness and finalization could reject accepted mail after the broker froze it. The existing writer's final eligibility checks still hold a changed composer before paste, as the approved protocol requires. No new ownership or input authority is added.
- 2026-10-17 11:44 PDT — TL-approved scope reduction (`00eac401-7b14-4ee8-8255-ef5cc30e078e`): Claude hubs only; Codex remains singleton. The native tall Codex failure also affects production confirmation: its 20-row recognizer never records occupied evidence for a 32-line batch while a turn is already active; a subsequent genuine empty composer cannot confirm, so the unchanged 30-second window ends indeterminate. A legacy 26-line singleton reproduces it too. File the recognizer/Return-remapping safety fix independently as unclaimed #652; no parser broadening or timeout/retry change enters #648. Both clients' retention measurements remain valid. Enforce the narrower profile at verified-hub registration and wrapper preparation, then rerun required verification on the revised code.
