---
id: 000773
status: working
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
severity: high
collaboration-mode: collaboration
card_mirror: 'caeb20cfe4142aba041bb9ee5899985f30a1691d' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-21T08:12:54-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "024ca5fa", done: "12c8a82c"}
---

# startup: terminal readiness truncates the caller to a fixed 10-second bound

## Problem

Production terminal readiness imposes a fixed10s deadline even when its caller has a longer remaining deadline. A cold native agent launch under load can be refused before the caller budget expires. Discovered by #744 final native matrix, independently from #772's attach InputEnded failure. This is a startup-budget design defect; do not classify it away as a flake or increase a numeric timeout.

## Spec

Make terminal readiness honor a coherent caller-owned startup lifetime while preserving cancellation, bounded observation, fail-closed admission and generation-qualified cleanup. Design the production default for callers without deadlines explicitly; do not substitute a larger arbitrary constant. Preserve distinct precreation, readiness and postbirth native-query observation contracts and error evidence.

Approved design: select one absolute startup deadline before the blocked helper is launched; reuse it for helper acknowledgement, pane birth, exact-generation readiness and Couch registration. Preserve an explicit caller deadline; with none, reuse the existing two-minute registration hang ceiling as the shared fallback. Consume the serialized deadline once across exec. Carry startup separately from interactive lifetime and the existing short native-query budgets; startup expiry never ends an admitted interactive client. Full design and adversarial tests: [startup budget plan](../plans/000773-startup-budget-plan.md).

## Done when

- The actual production handoff path no longer truncates a longer caller startup deadline to the fixed birthBound10s; an earlier caller cancellation/deadline still stops promptly.
- A deterministic regression exercises the real production budget-selection boundary with readiness after10s but before a longer caller deadline, plus earlier cancellation/expiry and absent-readiness controls. No retry or blanket timeout bump.
- Cold native launch verification and required recorded checks pass, with original failure evidence retained and no claim that load/AMFI was the measured cause unless proved.

- Existing progress reports the current startup stage and elapsed time; budget-exhaustion failure identifies the consuming stage and duration.

## Plan

- [x] Approve the caller-owned startup/default/transport contract and production-boundary test plan.
- [ ] Record deterministic red probes, implement shared deadline selection and propagate it through helper, launcher, readiness and registration without renewal.
- [ ] Qualify cancellations, uncertain outcomes, exact generation and admitted-client lifetime; update inventory and architecture map.
- [ ] Use #744 shared disposable harness only after it lands for cold native acceptance, then recorded checks, close review and landing.

## Log

2026-10-21: filed unclaimed by pair:5 per TL sequencing for #744. Trace: launcher/birthwatch.go birthBound10*time.Second; session_handoff_os.go passes it directly to coordinateTerminalHandoff; session_handoff.go applies context.WithTimeout(original caller,bound). Fleet caller allowed120s; separate LaunchOptions.BirthBound controls outer birth watch only; Couch registration default2m. CLI15s message is progress, not the budget.

Retained native failure on exact code db791e3b82bb23242ecd5e8fdbebd550c9da8e04: Claude fixture:3 (allocated thread tag1-fixture-4), nonce operation-9923c3b354cb3f27bbbb015f499d4a66; stage registration, Pair before/after exit1. Last output says terminal admission unverified: expected agent-ready-1-fixture-4-claude.json missing, then context deadline exceeded. Operation prepare877ms/operation15.732s; TestDisposableCouchFleet FAIL52.90s; whole command56.322s. Exact cleanup passed; no same-head rerun has occurred. Evidence in #744 PR509 at workshop/evidence/000744/final-outcome-native-fail.log, final-outcome-run.json, final-outcome-result.json; logSHA256355723305828834b91e4c8b91a2374d32d87e3f3c180bc00174f8c9f34058b42. Evidence commit51a07ef3936625a00011008e8d043db10f412efe.

Collision audit found shared scope/tag/agent resolver, unique tag-specific final/tmp paths and exact session/nonce validation; slot address3 versus tag ordinal4 is not a collision. Failed effective wrapper environment and sibling-file census were not retained, so incident inheritance remains unproved. Harness copies executables once per fleet; all workers reuse the same paths. Read-only macOS logs show contemporaneous Gatekeeper assessments with redacted paths; no event attributes this delay to the failed executable. AMFI, machine load, early wrapper failure and missing publication remain possible causes of this particular absent readiness; none is measured as its root cause. #744 is extending pre-cleanup evidence, without changing the production deadline.

### 2026-10-21 — production budget trace and proposed plan

Claimed from clean main after #765 cleanup; bare start-plan refreshed the sibling provider normally (Brewfile check only, no install). Read-only source trace confirms three separate relative startup windows: helper acknowledgement ten seconds, production OS admission birthBound ten seconds, Couch registration two minutes selected only after acknowledgement. Postbirth native observation is a fresh five-second query under interactive lifetime. Existing regression tests intentionally separate precreation-query expiry from readiness and preserve admitted interactive lifetime, so simply reusing queryContext or deadline-bounding LaunchOptions.Context would regress them.

A bounded independent read-only subagent traced Couch/exec/registration callers and confirmed these boundaries; no implementation or test was started. Proposed one absolute startup deadline and explicit phase/lifetime separation in the durable plan. #744 peer reports PR509 still unlanded; its final native matrix passed, recorded suite/close pending. Do not stack, copy that harness or claim its result as #773 acceptance. #739 and #738 remain separately held; #773 does not alter their work.

## Revisions

### 2026-10-21 — one startup owner, not a readiness constant

Reason: the production trace found independently renewed helper/readiness/registration windows, so removing the innermost constant alone cannot fulfill coherent startup ownership. Proposed delta: select one absolute deadline before helper launch, carry it across exec, preserve the existing two-minute fallback only when the caller supplies no deadline, retain shorter observation budgets and separate admitted interactive lifetime. The ten-second birth probe remains a diagnostic threshold. Native acceptance uses #744 after landing; original incident's missing-ready cause remains unknown. Awaiting hub approval before change-code or code.

### 2026-10-21 — hub approval and progress requirement

Hub approved the design and directed change-code. Required UX: existing progress identifies the current stage and elapsed time (waiting for agent readiness, 25 s); final failure identifies the stage that consumed the budget and elapsed time. Added to the durable plan; native acceptance remains on landed #744. No implementation started before the gate.

### 2026-10-21 — first red/green implementation checkpoint

Behavioral RED at the shared production handoff selection boundary: virtual readiness at 12s refused despite a 30s caller, and absent readiness failed at exactly10s. After replacing the admission birthBound with the explicit startup context, both pass in virtual time; default, early cancellation, short-query separation and admitted-client survival controls added. Initial red fixture cleanup had a synctest goroutine-lifetime error; fixed/joined it and repeated RED cleanly before the behavior fix.

Couch transport RED showed no deadline in the helper environment. It now selects before helper preparation, passes the exact timestamp once, and caps registration under that already-selected deadline. Kept registrationDeadline as its registration-only test seam after one existing exit-before-observation regression exposed that moving the seam earlier changed its meaning; no production deadline renewal. The corrected tracked-start and cold late-start/direct-exit/final-look cohort passed1.465s. Added context/error identity protection when caller/client expiry races readiness.

Independent bounded helper work proved expired/invalid timestamps and late acknowledgement incorrectly execed before the fix; now acknowledgement is capped by the absolute deadline, expiry is rechecked before exec, and the read goroutine is joined. Targeted helper controls pass0.274s. Pure value/codec controls pass0.058s, including behavioral ten-second-clipping negative control. CLI consumed-environment RED returned help success and leaked the internal deadline; corrected native entry passes0.191s.

Remote progress RED printed only generic running status. Existing context reporter now reaches an exact-request progress callback and pure receipt transition; same-stage age preserved, late/terminal/reused-ID progress refused. Broad service/authority cohort passed0.170s/9.810s and targeted race cohort1.264s/1.602s. Final failure's consuming-stage/elapsed control and broader core startup/ready/registration/birth/recovery cohort passed22.330s. Full launcher package and final inventory/suite/native acceptance are still pending; no completion or live success claimed.

### 2026-10-21 — pre-recorded-check checkpoint

Full launcher package passed65.149s. After subsequent deadline-race/birth-probe changes, focused production/handoff tests passed0.194s; all relevant core startup/readiness/birth/registration/recovery tests passed22.330s and service/receipt/authority cohort passed0.161s/8.905s. Targeted race controls passed launcher1.236s/core1.392s/message1.195s/command1.433s; that selector did not select pure value tests, whose ordinary package checks passed separately. Birth-probe cancellation RED showed a false dead/abort from the legacy non-contextual query; switched that existing probe to its short contextual observation under startup and the deterministic control passed0.288s. No new probe cadence or timeout bump.

The repository lint disallows even reasoned inline fixed-sleep suppressions. Tests now receive explicit timer events only inside synctest virtual time rather than adding sleep debt; catalog and new narrow admission-budget guard pass2.045s. Fast affected inventory passes; three existing reader certificates re-audited for mechanical callback/context signature changes only (run_test fixed scanner, createflow/workbench fixture-local). No new source reader was waived. Production code exceeds the quick shell, so close must use its full review upgrade; change-code initially inferred quick from the compact design. Native acceptance/close/landing remain pending, no live result claimed.

The actual RunLaunch create/attach boundary control initially assumed the interactive context must be the identical parent pointer. Create correctly owns a cancellable child for its birth watchdog and cancels it after its blocking client returns. Corrected the oracle to require no startup deadline on that child and an uncancelled original parent; control passes0.201s. No production change was made to satisfy the mistaken identity assumption. Short preflight contexts now inherit the original query context directly (synchronous cancellation), are deadline-capped by startup, and recheck startup before Start. OS entry validates startup before constructing the filesystem-backed attempt.
