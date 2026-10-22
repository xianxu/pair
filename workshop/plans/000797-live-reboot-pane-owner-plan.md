---
collaboration-mode: discussion
---

# Live reboot pane ownership diagnosis and correction plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy). Keep the native evidence capture and correction in this issue; the first capture is an investigation checkpoint, not permission to weaken an ownership validator.

**Goal:** Make public confirmed Fresh recovery complete for positively owned disposable Claude and Codex sessions, preserving exact stop authorization and all unknown/foreign/reused refusals.

**Architecture:** First retain the actual native observation at the retirement boundary and reproduce its causal sequence through the existing ownership probe. Then correct the ordering or proof handoff at the source of that failure. Reuse existing recovery authorization and lease machinery; a cached session name, a launch template, or a missing command never grants authority.

**Tech stack:** Go, existing Couch recovery state machine, procutil native identity/argv, Zellij, shared testcouch disposable fleet.

## Current evidence and uncertainty

The incident is recorded in issue #797's Log with its retained native log and SHA256. The stop offer names a registered wrapper, its Claude child and a Zellij session; confirmation fails inside the slot-tree fallback's final QuiesceNamed call.

Code proves this sequence:

1. Prepare and confirm recovery; preserve the prior screen/intent.
2. Optionally park the normal client helper.
3. Call any original terminal retirement.
4. TERM the additional registered wrapper tree and await its exits.
5. Re-prove session ownership from current live pane commands and quiesce it.

Steps2–4 can erase the evidence step5 requires. Zellij can leave exited panes held; command discovery then has no live argv. The existing fake separates process death from pane metadata, concealing this sequence.

The retained incident lacks pane JSON. Zellij also joins argv without quoting and can omit command enrichment after a100ms internal reply failure. These can yield the same generic refusal before any stop. The plan must distinguish the incident's missing/empty/ambiguous data, not infer it from its message.

## Core concepts

These are existing entities to inspect and reuse. No new production authority or state is approved by this diagnosis plan.

### Pure decisions

| Name | Lives in | Planned role |
| --- | --- | --- |
| ClassifySessionOwner / splitObservedCommand | cmd/internal/launcher/session_owner.go | Preserve fail-closed classification; distinguish missing evidence from ambiguous text in diagnostics if needed |
| RecoveryTerminalPlan / checkConfirmed | cmd/internal/launcher/recovery_terminal*.go | Existing immutable server/descendant authorization and survivor checks; candidate reuse, not a second cache |
| AdvanceRecoveryExecution | cmd/internal/couchcore/recovery_plan.go | Preserve screen/intent before retirement and launch only after confirmed retirement |

### Integration points

| Name | Lives in | Wraps / test seam |
| --- | --- | --- |
| withSlotTree | cmd/internal/couchcore/recovery_slot_tree.go | Registered wrapper, exact process signals and session quiescence |
| executePreparedRecovery / parkPreparedRecovery | recovery_prepared.go / recovery_retirement_os.go | Confirmation, screen preservation, normal park and terminal retirement ordering |
| SessionOwnerProbe / ScopedThreadArtifactCollisionChecker | launcher/session_owner_os.go / couchcore/artifactcollision.go | Strict native pane replies, exact server generation and current binding |
| testcouch.Fleet | cmd/internal/testcouch | Isolated genuine agents, public CLI, identity-fenced owned cleanup |

The causal fake must couple process retirement to pane command disappearance. Reuse the real SessionOwnerProbe and strict reply decoder behind the existing fake native IO boundary; do not replace the final ownership decision with a scripted success.

## Investigation checkpoint before production changes

- [ ] Retain a bounded disposable native capture using the shared harness. Record exact tool/version, public reboot offer, confirmation request/outcome, native pane JSON and server/wrapper generations before and at the failing retirement boundary. Retain the original failure's bounded NativeQueryEvidence before private-root cleanup. Credentials and unrelated process argv do not enter committed evidence.
- [ ] Run one personality per Go invocation. Exercise Claude and Codex; retain setup, operation, output, write and cleanup bounds. All direct children and readers join; unknown cleanup prevents acceptance. Do not stage pair:0 or exercise the live fleet.
- [ ] Build a deterministic replay through actual SessionOwnerProbe and recovery composition: valid owned wrapper evidence -> accepted confirmation -> park/retire -> held/exited pane missing command -> current refusal. Also distinguish pre-existing omission and ambiguous joined text.
- [ ] Record which event first loses the witness and why confirmed native terminal attribution selected the slot-tree fallback. If the capture contradicts post-stop evidence loss, stop and revise the diagnosis rather than applying an ordering patch.

## Correction decision and approval boundary

Two viable shapes must be assessed against that capture:

1. **Quiesce while the positive witness still exists.** This is the smallest correction if the only destructive predecessor is the added slot-tree TERM loop. It must also cover the outer normal-park ordering. Moving a single inner call is insufficient if park already erased the witness. Keep normal park's preservation/continuation semantics.
2. **Reuse prepared exact terminal retirement across witness loss.** If normal park must precede terminal retirement, reuse the existing immutable server/descendant plan, lease, current-generation/survivor checks and allow-gone semantics. The initial proof must be positive; missing initial data still refuses. No new persisted permit, no name-only authorization and no expansion after confirmation.

Preferred rule: obtain the positive proof before destroying its witness, and recheck the exact accepted generation at each effect. Choose the smallest existing mechanism that enforces that rule for both normal-park and legacy fallback. Do not broaden ClassifySessionOwner to accept unknown panes. Do not treat Zellij's joined string as structured argv where boundaries matter; do not implement a cross-repository producer change without a separate approved scope.

- [ ] Present the captured cause and the exact chosen correction to the hub; append the approved design as a revision. Run change-code and its plan-quality gate only after that design is concrete and approved. No production edit precedes this boundary.

## Implementation and verification after that approval

- [ ] Add the failing causal regression first, including both applicable park/retirement orderings. Confirm it fails on current main, then implement the approved correction.
- [ ] Cover same-session foreign owner, unknown/missing/ambiguous command, server replacement and PID reuse, newly arrived descendant, cancellation, expired/changed confirmation, stop applied but acknowledgment lost, independent exits and partial retirement. Assert no unauthorized signal and no launch before confirmed absence; never retry uncertain destructive input automatically.
- [ ] Give missing actual-command evidence and parser ambiguity distinct pane-specific diagnostics without leaking raw arbitrary command text into the concise CLI error. Preserve native diagnostic evidence under the existing bounds.
- [ ] Replay the captured native shape in the normal suite. Run public reboot plus its returned confirmation token against genuine disposable Claude and Codex workers; prove the new generation accepts public delivery, the old exact set is absent and cleanup is complete. No fixture-only crash substitutes for live reboot on this issue.
- [ ] Update the recovery atlas and enumerable consumer tests. Run relevant race/ownership tests plus catalog, Godoc, artifact inventory and reader audits. Publish the final head, run the declared affected check/full fallback, close through its mandatory review and merge on SHIP. Report the PR to the hub.

## Constraints, state and trust

This is an explicit recovery operation, not a sampling or keystroke hot path. Preserve existing RPC/query/retirement deadlines, confirmation lifetime, exact-set bounds and serial native test structure. Count the new observations/effect gates at N and2N if the selected correction adds traversal; require linear table/set work and no repeated host census per target.

Actions: inspect, request a new confirmation, acknowledge the current confirmation, resume preserved recovery after inspection.

| Observation / outcome | Visible state | Recovery action | Possible loss |
| --- | --- | --- | --- |
| Complete positive initial proof, same accepted set | confirmation required | acknowledge current offer | source may be retired after acknowledgment |
| Missing/ambiguous/foreign proof before effects | refused with specific diagnostic | inspect | none from this operation |
| Identity or ownership changes | refused; refresh confirmation | request new confirmation | earlier individually confirmed stops may remain applied |
| Unconfirmed stop or query | partial/unconfirmed, no launch | inspect | stopped source work; preserved evidence remains |
| Exact retirement confirmed | new generation started | inspect/deliver | old working turn intentionally ends |

ARCH-DRY: reuse the existing terminal plan, leases, strict parser and fleet harness. ARCH-PURE: classify evidence independently of IO; the shell executes declared recovery order. ARCH-PURPOSE: both public live reboot personalities and negative ownership gates are required; a diagnostic-only change does not finish the issue. ARCH-MOCK: couple fake native metadata to process state and compare it with native capture. ARCH-CONSTRAINTS: no deadline tuning or extra unbounded scans. ARCH-SECURE: argv, pane JSON and persisted bindings are observations, never trusted because they are local. ARCH-ORDER: distinguish initial proof, own applied stops and uncertain completion; preserve the original exact set and current effect vetoes. ARCH-FUNERAL: no new production durable family; bounded diagnostic captures stay in the test log/evidence record, fixture roots and processes have existing cleanup. ARCH-EVOLVE: no persisted format or field-meaning change is proposed; if the concrete correction needs one, return for approval.

## Revisions

### 2026-10-22 — Diagnostic stage approved

Authenticated ops:0 approval9d722e7b-5b6a-4b7e-8907-d5d045742c03 authorizes the bounded throwaway Claude/Codex capture and causal replay first. Retain the refusal's actual NativeQueryEvidence before cleanup removes the private store; retain that diagnostic improvement regardless of the selected correction. No production ownership or ordering change is authorized yet. After capture, bring the concrete correction choice to the hub and append its approved design before changing it.

The initial diagnostic stage changes tests and the existing harness evidence renderer only. Reuse the persisted bounded LifecycleFailure.Zellij* fields; test that the pre-cleanup renderer emits the actual command, version, exit, byte counts, truncation and raw stdout/stderr prefixes, never synthesizing a valid JSON reply from a truncated prefix. The producer's4KiB per-stream cap and existing max-four failure selection remain unchanged. Native runs use the shared testcouch fleet and its disposable stores; no runtime/fleet stage.
