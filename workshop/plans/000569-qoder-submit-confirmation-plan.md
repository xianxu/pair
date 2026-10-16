# Qoder submission confirmation implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach. Steps use checkbox syntax for tracking.

**Goal:** Make Couch submissions to Qoder complete visibly, retry dropped submit keys without duplicating the body, and expose unreconciled automatic input instead of silently starving subsequent deliveries.

**Architecture:** Extend the existing pure peer-delivery reducer with bounded submit retry, keeping terminal interpretation and PTY writes in the wrapper adapter. Qoder confirmation uses complete rendered composer evidence; occupied or unknown input never becomes success solely from a turn flag. Extend the existing automatic-input tripwire for unreconciled Qoder residue, retaining conservative input ownership and an explicit recovery path.

**Tech Stack:** Go; existing terminal model, peer delivery reducer, harnessSessionFake, input-admission lock, wrapper reporter and Couch broker.

## Evidence and alternatives

Live read-only peek of pair:4 shows the exact restart envelope 44c73545-5fed-4759-9c68-fd4865c10085 in a multiline `*` composer after Qoder completed its earlier task. The original receipt is unavailable following the restart, so its exact terminal outcome cannot be reconstructed. TL reports later deliveries expire behind the previous automatic input.

Source establishes three failures in the current recovery contract: no shared receiver retries a submit; turn-open alone can confirm without fresh rendered output or an empty composer; occupied composers are excluded from the automatic-input tripwire. The native reason that Qoder ignored the first CR is not yet proven. Qoder 1.1.71 uses bare CR for submit; `*` is also a normal input glyph, not an acceptance indicator. #544's native-drive fix paces Codex input and is not an existing shared retry implementation.

Options considered: increase a fixed delay (cannot confirm acceptance); replay the entire message (can duplicate effects); retry only the submit key while the wrapper retains input authority (selected). Never manufacture success or erase a draft to clear a latch.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| PeerDeliveryState / AdvancePeerDelivery | cmd/internal/couchmessage/delivery.go | modified |
| Qoder submission evidence | cmd/internal/wrapcmd/peer_qoder_submission.go | new |
| automaticInputTransaction / advanceAutomaticInput | cmd/internal/wrapcmd/automatic_input.go | modified |
| AutomaticInputStall | cmd/internal/couchmessage/automatic_input_stall.go | modified |

The existing reducer owns legal phases, submit-attempt count and terminal outcomes. Initial submit and retry both pass through its Submitting phase and report exact write completion. Qoder evidence is a pure projection of a complete parsed composer, rendered epoch, paste/submit fences and elapsed clock values. It requests effects rather than writing input. Automatic-input ownership remains one transaction per wrapper; failure never grants permission to paste over residual text.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| dispatchPeer / advancePeer | cmd/internal/wrapcmd/peer_delivery.go | modified | existing PTY writer, terminal snapshot and clock |
| Automatic-input reporter | cmd/internal/wrapcmd/automatic_input_report.go | reused | bounded evidence queue and broker request |
| Supervisor stall handler | cmd/internal/couchcmd/automatic_input_stall.go | modified | authenticated owning-hub and desktop notice |
| harnessSessionFake | cmd/internal/wrapcmd/harness_tty_integration_test.go and peer_qoder_test.go | reused/extended | production input owner and terminal model |

The stateful receiver fixture paints actual captured Qoder idle/pending screen shapes, retains body until a configured submit is accepted, counts each submit and each body paste, and can drop initial submit, delay repaint, accept mid-turn, interrupt with human input or replace the composer with a picker. Every production effect traverses the same writer seam as the fake. Live pair:4 checks qualify that model before landing.

## State and effect contract

- Waiting + existing paste-ready gates -> one bracketed body paste. Existing operator, image, picker, view, terminal-input and deadline fences remain.
- Pasted + existing 150 ms delay -> initial submit. Record rendered paste/submit fences and submit time through the input writer.
- Confirming + complete post-submit rendered empty composer, after the paste was observed -> Submitted. For Qoder an unrelated turn transition while the composer still holds text, or an unknown/incomplete snapshot, is insufficient. Other receivers retain their existing confirmation contract.
- Confirming + positively recognized occupied Qoder composer after observed paste, no admitted human input/overlay, at least 500 ms since the previous submit, and remaining attempt budget -> submit-key-only retry. At most three submit attempts total. Never paste the body again, reset the original 30 s delivery window, or act after a terminal outcome.
- Partial/failed submit write, interruption or confirmation deadline -> existing Indeterminate outcome after a paste. Do not turn an uncertain external effect into Expired or Submitted.
- Unreconciled occupied Qoder input -> one bounded typed stall report and explicit operator recovery requirement. Later deliveries refuse promptly with an actionable reason rather than spending repeated 90 s waits behind the residue. Do not erase, append to or submit an expired message. A complete positively empty post-transaction composer releases the lease and allows a new delivery.

The final unreconciled-input policy is awaiting TL confirmation: safe bounded escalation and prompt refusals versus automatic residual-text reconciliation. The implementation must not invent authority to clear operator text. Live check must include the dropped-first-submit recovery so normal delivery does not reach this terminal state.

## Operating envelope and principles

- ARCH-DRY: reuse AdvancePeerDelivery, existing polling/input writer, terminal parser and authenticated reporter. No second queue or PTY writer.
- ARCH-PURE/ORDER: retry and terminal transitions belong to explicit reducers. IO reports short writes, interruptions and child exit as events. Use rendered epochs, not raw arrival sequence, as evidence that the terminal consumed output. Test late frames and unrelated turn transitions.
- ARCH-PURPOSE: cover idle, mid-turn, retry, confirmation, terminal residue and next-delivery recovery in this item. Native first-key suppression is an uncertainty to measure, not a claimed diagnosis.
- ARCH-MOCK: controlled stateful terminal/receiver backend behind the real writer seam, plus TL-coordinated pair:4 conformance. No operator namespace in automated tests.
- ARCH-CONSTRAINTS: reuse active-only 50 ms poll and 30 s confirmation budget. Three CR attempts, >=500 ms spacing; this initial engineering budget is subject to live evidence and TL approval. No background polling while idle and no extra network IO on the writer. Full suite uses machine gate.
- ARCH-SECURE: Qoder bytes/screens are untrusted; unknown or partial views decline input authority. Human admission revokes retry. Reports remain typed/authenticated, sanitize retained tail and never accept arbitrary notification text. No credentials or auth mutations.
- ARCH-FUNERAL: no new durable artifact family, worker or queue. Existing recent receipts, diagnostic retention and one-element report queue retain their bounds. Retry counters die with the delivery; report occurrence deduplication remains wrapper-incarnation scoped.

## Tasks — one atomic close boundary

### 1. Pin native and state-machine evidence

- [x] Preserve a bounded read-only pair:4 pending/idle observation and receipt status. Coordinate through the TL before any live input, restart or staging.
- [x] Add failing tests using the named function strategies in the revision below. Exercise the production writer with the stateful Qoder receiver and record RED.
- [x] Implement the minimal reducer retry event/count and Qoder evidence projection, then wire through the existing PTY writer. Record GREEN and exact wire counts: body pasted once, bounded CR writes, no writes after confirmation/interruption/deadline.

### 2. Bound failed-transaction recovery

- [x] Under the approved terminal policy, add RED tests for the recovery invariants using the named function strategies below. Keep reporting, input ownership and next-delivery outcomes independently observable.
- [x] Extend the existing typed stall cause/parser and supervisor notice if needed, preserving mixed-version visible refusal and no automatic release on unknown/occupied input. Reuse the report queue, sanitization, incarnation binding and sequence deduplication. Run parser, reducer and supervisor tests, then race tests.
- [x] Map the confirmation/retry/recovery behavior in atlas/couch.md; update public guidance only where the operator must act. Add a lesson for any review-discovered rule. Update artifact inventory for a new production source file.

### 3. Verify, live check and land

- [x] Run focused couchmessage/wrapcmd/couchcmd/artifactpath tests and races; make build. Commit/push candidate and run sdlc local-check run --name suite on the pushed head.
- [ ] Send exact candidate revision to TL; stage/build only as coordinated. TL arranges pair:4 on the candidate wrapper, resolving the pre-existing residual input explicitly. Send harmless unique idle and mid-turn probes; verify receipts Submitted, composer consumed and Qoder acts once. Send a subsequent probe to prove no stale automatic-input latch. Retain failures and do not call a simulated fake a live acceptance.
- [ ] Record live/automated evidence in issue Log; run sdlc close --issue 569 --verified with measured actuals. The gate owns fresh-context review. Fix findings by class and repeat required checks when code changes.
- [ ] On SHIP, integrate main and run sdlc merge --yes; verify landing, report PR/revision to TL and stop. Clear staging as required by the workflow.

## Approval

TL approval of this plan, retry budget and terminal residual-input policy is pending. No implementation begins before approval and sdlc change-code.

## Revisions

### 2026-10-15 — TL approval and exact envelope authority

TL approved plan 967b13cdbf243d945d13986c1d0864c2bc906cad (receipt c4ac5b82-afee-4dfe-8a1e-70a9c5c367db), including three attempts >=500 ms apart inside the existing 30 s window and the explicit terminal-residue policy. Every retry additionally requires that the rendered Qoder composer still holds exactly the envelope Couch typed (earlier refinement 84d5bf3e-f2ba-4f7b-a483-c65b528689f6). Match the full qualified composer, never just a prefix/header or whitespace-stripped substring; if wrapping/truncation makes exact ownership unverifiable, do not retry. Human admission, changed text, unknown/partial frames and overlays revoke retry. The live check must run before landing and be coordinated with TL; Qoder's cheap window is 07:00–17:00 PDT.

### 2026-10-15 — plan-quality round 1 corrections

PQ-1 (ARCH-ORDER): terminal escalation is an event effect, not a future timeout. In advancePeer's PeerPublish effect, before stopping delivery polling, an unconfirmed Qoder submission after a complete body paste and without human interruption emits automaticQuarantine through advanceAutomaticInput. That pure event enters NeedsOperator and emits exactly one report immediately, with duration measured since pastedAt. Exact-envelope residue uses the typed unsubmitted-composer cause; unverifiable residue uses an uncertainty cause and wording, never a claim that execution failed. The existing bounded reporter performs IO off the writer. A subsequent delivery observes this state and the existing peer reducer publishes a no-paste terminal refusal with the actionable reason; it does not spend its admission deadline waiting. Normal output already signals d.wake and calls dispatchPeer through the input owner. Evaluate the automatic-input recovery observation before dispatchPeer's terminal early return, so a complete post-quarantine empty frame clears NeedsOperator even when there is no next delivery. Only that existing render wake or new admission drives observation; no new timer, goroutine or idle poll. An empty snapshot at/before the transaction fence cannot release. Human input, unknown/partial output or an occupied composer never authorizes a retry, deletion or late execution.

PQ-2: the earlier scenario prose is superseded by these function-level adversarial strategies. Individual cases belong in executable tests.

| Risky function | Input class and mechanical guard |
|---------------|----------------------------------|
| AdvancePeerDelivery | Generate retry/interrupt/terminal event sequences; count PTY effects independently, prove at most three submits, exactly one paste and no post-terminal effects. |
| qoderSubmissionEvidence | Mutate complete terminal snapshots, envelope bytes and rendered epochs; exact supported layout and unchanged-input authority are necessary for retry, while only fresh empty evidence confirms. Fuzz malformed geometry/text and reject clipped or unsupported representations. |
| dispatchPeer / advancePeer | Stateful receiver drops keys and reorders paint/turn/input events under fake time; assert receipt outcome against receiver's accepted-envelope count and exact wire writes. |
| advanceAutomaticInput | Enumerate quarantine/render/admission transitions with monotone epochs; prove one report, immediate no-paste refusal and recovery only from later complete empty evidence. |
| ParseAutomaticInputStall | Fuzz bounded JSON and typed-cause mutations; validate required fields, non-release of unsubmitted/unknown causes and visible mixed-version refusal. |
| handleAutomaticInputStall | Existing authenticated fake supervisor receives duplicate/foreign/changed-incarnation reports; assert one owning-hub notice per authorized sequence and no control authority from report content. |

Exact-envelope interpretation: use the complete composer parser and a deterministic expected visual projection of Couch's known envelope. Preserve logical newlines and meaningful spaces; never flatten or normalize arbitrary observed text. Printable/Unicode layout is supported only when its renderer projection can be established by captured Qoder evidence; unknown wrapping/truncation/control sequences fail closed. The no-admitted-human-input fence remains necessary because screen equality alone cannot prove underlying editor bytes. If a native layout is not yet qualified, the safe outcome is no retry plus explicit recovery, not broadened text matching.

### 2026-10-15 — TL defers live activation; review now, hold

TL receipt 172b9532-1e96-4fc7-ab4b-a838f7469fea directs the required suite and close review on candidate code 545ac416d19e3a84905081e3791d8b0a69288fdd, then hold. pair:0 is staged for #544 and pair:4 candidate relaunch is blocked by #584's ownership-guard repair. Do not restage, relaunch or merge this item now. TL plans the combined stage/restart and idle, mid-turn and subsequent-delivery probes in the next 07:00 cheap window after #584 lands. The original prelanding live task remains owed, never ticked; close review may assess the prepared implementation now but does not establish native acceptance. Current suite head 81624e3d60b60189fe823b98e6e9e1210aedafde integrates only new issue details beyond that code.

### 2026-10-15 — implementation checkpoint, final suite queued

Tasks 1–2 are complete. Candidate code a7b330872a80118ff1756edde74104805b07225d includes exact-envelope bounded retries, fresh-empty confirmation, immediate one-report quarantine and recovery, uncertainty-preserving refusal and quarantine precedence over picker holds. Native pending fixture matches the wrapped restart envelope. RED/GREEN and focused race/fuzz/inventory/build evidence is recorded in the issue Log. Required suite is queued on this pushed head (PR #352); no pass claim yet. CI is green. Final full-suite row, close review and all native live/landing rows remain unchecked. TL directs review then HOLD; no staging or relaunch.

### 2026-10-15 — final required suite passed

The recorded suite on a7b330872a80118ff1756edde74104805b07225d passed exit0 in 12m29s (SHA256 4d6a4760b008e2410324de2b8f660f8905cdfb00710ae5b02a2a4d21076e2895); focused/race/fuzz/build/inventory evidence and all CI checks pass. Fresh main integration added issue details only. The full-suite task is complete; close review is next. TL-directed live/landing hold remains in force, and the native live task remains unchecked.

### 2026-10-15 — close round 1 recovery action and README correction

BR-1: earlier prose/example 'press Enter' is superseded by **press Alt+Enter or clear it**. Pair's Qoder profile rewrites plain Enter to a newline; Alt+Enter produces the native submit CR. One shared AutomaticInputRecoveryHint feeds receipt and supervisor instructions. The regression follows the actual refusal's prescribed chord through production translateChunk, verifies a bare native submit and observes the stateful receiver consume the envelope/release quarantine. A string-equality assertion alone is insufficient.

BR-2: sweep README's Qoder-specific confirmation exception, exact-envelope bounded key-only retries, prompt quarantine refusal and correct manual recovery. Replace its generic 'never retries uncertain input' claim with body-replay wording. Atlas and current notice/test strings agree; historical TL quotations remain unchanged. Native acceptance and landing stay held.

### 2026-10-15 — corrected-head required suite passed

Required suite on 8186d8abd3e02528189a1809f3793f57d682bf07 passed exit0 in 14m49s after 1719s queued, including recovery-key and README corrections. Log SHA256 56304faa904ba90b1cc393de569f97a8999bb27c34d72cc82068fb954f757dc9. Focused races and CI pass. Second close review is next; native acceptance and landing remain unchecked under TL's explicit hold.
