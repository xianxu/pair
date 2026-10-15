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

- [ ] Preserve a bounded read-only pair:4 pending/idle observation and receipt status. Coordinate through the TL before any live input, restart or staging.
- [ ] Add failing pure reducer tests for bounded retry and terminal no-op behavior. Add stateful Qoder regressions for dropped first submit, late paste render, occupied composer plus unrelated turn, and mid-turn acceptance. Run targeted tests and record RED.
- [ ] Implement the minimal reducer retry event/count and Qoder evidence projection, then wire through the existing PTY writer. Record GREEN and exact wire counts: body pasted once, bounded CR writes, no writes after confirmation/interruption/deadline.

### 2. Bound failed-transaction recovery

- [ ] After TL decides the terminal policy, add RED tests for unreconciled residue, one report, immediate actionable later refusal, and positive-empty recovery followed by a successful new delivery. Include human drafts, pickers, partial frames, stale epochs and child exit as negative controls.
- [ ] Extend the existing typed stall cause/parser and supervisor notice if needed, preserving mixed-version visible refusal and no automatic release on unknown/occupied input. Reuse the report queue, sanitization, incarnation binding and sequence deduplication. Run parser, reducer and supervisor tests, then race tests.
- [ ] Map the confirmation/retry/recovery behavior in atlas/couch.md; update public guidance only where the operator must act. Add a lesson for any review-discovered rule. Update artifact inventory for a new production source file.

### 3. Verify, live check and land

- [ ] Run focused couchmessage/wrapcmd/couchcmd/artifactpath tests and races; make build. Commit/push candidate and run sdlc local-check run --name suite on the pushed head.
- [ ] Send exact candidate revision to TL; stage/build only as coordinated. TL arranges pair:4 on the candidate wrapper, resolving the pre-existing residual input explicitly. Send harmless unique idle and mid-turn probes; verify receipts Submitted, composer consumed and Qoder acts once. Send a subsequent probe to prove no stale automatic-input latch. Retain failures and do not call a simulated fake a live acceptance.
- [ ] Record live/automated evidence in issue Log; run sdlc close --issue 569 --verified with measured actuals. The gate owns fresh-context review. Fix findings by class and repeat required checks when code changes.
- [ ] On SHIP, integrate main and run sdlc merge --yes; verify landing, report PR/revision to TL and stop. Clear staging as required by the workflow.

## Approval

TL approval of this plan, retry budget and terminal residual-input policy is pending. No implementation begins before approval and sdlc change-code.
