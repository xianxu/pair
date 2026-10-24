---
id: 000860
status: working
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-24
estimate_hours:
card_mirror: '5a04c92d804322cec1434d02d8be35cc0e113b7c' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-24T00:22:54-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:9
    worktree: /Users/xianxu/workspace/worktree/pair-slot9/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "6553758b", done: "0e308a0d"}
---

# Multi-hub test fixture mutates authority after message readers start

## Problem

A broader targeted race run during #812 integration failed TestHubNotifyNamesExtraHubAndBoundsCustomTitle: multiHubWaitRig starts newWaitRig (and the message service registry loop) before mutating authority.hub and authority.hubs. messageService.execute concurrently reads those fields. This is a fixture race already present in baseline, separate from #812 notice behavior.

Captured race frames:

```text
Write: multiHubWaitRig supervision_test.go:428 (authority.hub)
Write: multiHubWaitRig supervision_test.go:429 (authority.hubs)
Caller: TestHubNotifyNamesExtraHubAndBoundsCustomTitle notify_service_test.go:142
Previous read: messageService.execute message_service.go:900
Read caller: messageService.loop.func1 :836 -> messageService.loop :860
Reader created: newMessageService :727 -> serviceRig.start message_service_test.go:244
 -> serviceRig.restart :218 -> newWaitRig wait_notice_test.go:328
 -> multiHubWaitRig supervision_test.go:414
--- FAIL: TestHubNotifyNamesExtraHubAndBoundsCustomTitle (0.02s)
    testing.go:1867: race detected during execution of test
FAIL couchcmd 83.453s
```

The two test files and the entire execute function are byte-identical across branch base c0dbb32, integrated main cd347333, branch 70185310 and origin/main aafc7c8. This is source comparison, not an executed baseline pass. The broad Hub test expression selected this case; focused HubStartup tests do not. No matching active issue found at filing.

## Spec

Build the final multi-hub authority before starting its service readers, using the existing fixture/composition seam. Preserve the notification routing and bounded-title assertions; no production lock or timing sleep should be needed for fixture initialization.

The approved same-class sweep moves hub/hubs configuration in already-connected wait, switch, slot-admission, recovery, archive-warning and lifecycle fixtures through serviceRig.decorateAuthority before newMessageService. Keep intentional mutex-protected hubRuntime.state transitions, local authority construction and in-memory service literals. Quiescent fixtures explicitly modeling cold startup remain separate from the demonstrated live-reader race. No production code, wire format, timeout or retry behavior changes.

## Done when

- Connected worker-cue and native-exit fixtures compose child authority before construction. Repeated races preserve default proof, role-change, committed-report, bounded-proof, unsupported-evidence and lifecycle-notice controls.
- A source enumeration check rejects direct child/hub/hubs writes outside the individually named no-wrapper controls, so rare race scheduling cannot hide a new connected-fixture assignment.

- The captured writer/reader initialization race is prevented structurally.
- The exact notification test and other users of multiHubWaitRig pass repeated race testing without weakening their assertions.
- A bounded suggested repro is go test -race ./cmd/internal/couchcmd -run '^TestHubNotifyNamesExtraHubAndBoundsCustomTitle$' -count=20; this exact isolation has not yet been run by the filer.
- A deterministic regression proves both configured hubs have startup automation policy; it fails before the fix and passes afterward.
- Analogous already-connected fixture hub/hubs assignments are composed before service start, preserving their existing guard, ownership, generation and policy-epoch assertions.

## Plan

- [x] Add the deterministic startup-policy regression and observe RED; the existing exact notification race count20 is already RED.
- [x] Forward the existing authority decorator through the necessary fixture constructors; build final ownership before initialization and remove the enumerated live-service hub/hubs assignments.
- [x] Run exact and consumer repeated races plus the relevant guard/recovery/archive/lifecycle controls, then full couchcmd race qualification. Add the lesson and audit each changed reader certificate individually.
- [x] Push corrected code and initiate the recorded pre-close check; retain its exact run through the mandatory close review under the worker overlap rule.

Landing follows the close verdict and completed recorded check; neither is claimed before it completes.

Quick flow, test-only. ARCH-DRY: reuse decorateAuthority, not a second service-start mechanism. ARCH-ORDER: ownership is final before the constructor launches readers; actual ownership moves still use the existing hubRuntime mutex. ARCH-PURPOSE: enumerate live-wrapper siblings in this round rather than repairing only the two captured lines. ARCH-PURE/MOCK: all changes stay in the existing stateful socket fixtures; production routing is unchanged. ARCH-CONSTRAINTS: no new workload, capacity, timing or cost driver; tests retain existing budgets and use bounded repetitions. ARCH-SECURE/EVOLVE: no new trust boundary, credentials or persisted formats. ARCH-FUNERAL: existing fixture cancellation, joins and private-directory cleanup remain the owners of their resources. Regression plus lesson is the approved enforcement; no catalog follow-up unless another recurrence appears.

## Log

Observed during #812 targeted integration checks. Full recorded #812 suite failure on70185310 was a different stale startup display oracle; this race was found only by the broader targeted race run. Unclaimed follow-up; not fixed under #812.

### 2026-10-24 — Claimed and reproduced before implementation
- 2026-10-24: closed — Test-only fixture repair; no architectural surface. Original hub race RED; deterministic two-hub startup/restart policy RED then GREEN. Native-exit race RED 11.256s and new mechanical enumeration RED 0.439s naming all three writes; complete child controls plus enumeration race count20 PASS 40.086s; prior hub/sibling count20 controls PASS. Final full couchcmd race PASS 211.265s, repolint PASS 20.446s, affected selector tests 13/13 PASS 11.810s. Individually audited fixed-source reader inventory PASS immediately before close. BR-1 connected child family and BR-2 Done-when addressed, assertions preserved. Recorded pre-close lease-1097383963 on corrected source bb826047 remains queued; permitted review overlap, landing waits for pass.; review verdict: SHIP

Claimed by pair:9 after #825 and #841 landed, on fresh main99846a5015f753f05edaf4778618849f377a2f71. Start-plan refreshed the substrate and created the issue branch; no generated vocabulary dirt remains. Ran runtimebundle-generate, then the exact notification test under race with count20 and timeout120s. It failed in1.012s with both multiHubWaitRig authority writes racing messageService.execute at896; raw evidence is /tmp/pair860-baseline-race.log. No implementation changes were made before this RED.

The service publishes its connected snapshot before execute finishes reading authority, so waitConnected is not a completion barrier for changing immutable configuration. Existing serviceRig.decorateAuthority runs before newMessageService and survives restart. A read-only fixture audit recommends passing that decorator through newWaitRig and constructing multi-hub ownership before invoking it. A deterministic startup-policy regression can independently require automation status for both configured hubs: late replacement currently leaves /repo3 unknown because startup seeded only /repo1.

The same audit enumerated analogous already-connected fixture assignments in hub_admission_test, hub_guards_test, hub_mutation_payload_test, recovery_confirmation_test, archive_notify_test and lifecycle_notices_test. Quiescent pre-wrapper assignments and local authority/service literals are distinct. Scope and enforcement are being included in the plan to the hub before code; no production lock, timing sleep, retry or assertion weakening is proposed (ARCH-ORDER, ARCH-DRY, ARCH-PURPOSE).

TL approval276ef60c-ef11-4c6c-b912-fe14febc2767 (submitted): same-class sweep through decorateAuthority, keep locked state tests, RED-first regression and repeated races, then land. Behavioral regression plus lesson is enough; no catalog follow-up unless a second recurrence appears. The generic lint alternative cannot infer publication/reader lifetime without rejecting legitimate local composition or cold-fixture setup.

### 2026-10-24 — Pre-start composition and repeated controls

The deterministic startup-policy regression failed0.407s before the fix: /repo3 was not configured in automation. It now checks both hubs at initial startup and after restart. The focused regression plus all original multiHubWaitRig consumers passed0.592s without race, then20 race repetitions passed4.208s. The audited live-wrapper guard/admission/payload/recovery/archive/lifecycle cases passed20 race repetitions9.189s. Existing assertions remain unchanged; four live ownership transitions now take the existing hubRuntime mutex rather than racing readers.

Forwarded decorateAuthority through wait, switch and slot constructors; the hub-message and manual-lifecycle helpers apply their override before construction too. Direct hub/hubs assignments remaining in the tree are the local pre-construction authority, the no-loop service literal, the pre-wrapper lifecycle setup, and the explicit cold-start acceptance fixture. No production files changed.

Inventory named only wait_notice_test.go. Audited its existing modelConfirmationTail reads: captured Claude2.1.299 and Codex0.162.1 session-status.screen files under wrapcmd/testdata/drive, plus private t.TempDir mute-list IO. No reader path, input set or fallback owner changed, and every test in that fixed-scanner file remains registered in SOURCE_GUARDS. Recorded only the printed exact-file fixed_scanner command. Full couchcmd race and repository lint qualification are running; their passes and the recorded pre-close are not yet claimed.

## Revisions

### 2026-10-24 — Complete child-proof family and mechanical enumeration

Close review BR-1 reproduced all three native-exit child-authority assignments; BR-2 found the added child scope missing from Done-when. TL55cc757d-d197-44f7-b333-56e327bac073 approved their pre-construction composition, unchanged assertions, aligned acceptance criteria, and a small source enumeration check. That check rejects direct service authority.child/hub/hubs writes, with four exact one-write no-wrapper exceptions; a wrapper/connect call before an excepted write also refuses. Local composed authority values remain valid. This replaces reliance on a package race pass alone with mechanical enforcement.

### 2026-10-24 — Include measured child-proof publication

Extend the approved same-class sweep to newWorkerCueRig authority.child and its custom child-proof controls after full package race exposed the identical publication bug. Use the existing pre-construction decorator, preserve proof controls, repeat the coordinator races and full package qualification. Approved by TL f28ac8d5-81f5-4bec-b5eb-4bea676489a3.

### 2026-10-24 — Sweep the demonstrated initialization class

Extend the named multiHubWaitRig repair to analogous fixture assignments after wrappers are live, as approved by the TL. Add deterministic constructor-derived policy coverage alongside race qualification. Preserve the original no-lock/no-sleep and unchanged notification assertions; use one atomic close boundary, not separate milestones.

### 2026-10-24 — Measured child-proof sibling

Full couchcmd race failed in304.234s at newWorkerCueRig's post-connect authority.child replacement, racing the same execute authority read. Exact unknown/uncertain coordinator count20 reproduced RED2.390s on5443413ec. TL approved folding this sibling in (f28ac8d5-81f5-4bec-b5eb-4bea676489a3), and explicitly directed cancellation of old-head queued lease-3482810010; the supported cancel completed successfully. No pass is claimed for that cancelled check. Compose the child-proof closure before construction and supply the three custom proof callbacks through the same fixture constructor, preserving their assertions and atomic proof/ordinal state. The counted no-wrapper fixture remains distinct. Requeue pre-close only after the corrected source is pushed.

The corrected coordinator cases all pass20 race repetitions8.246s, including role-change, committed-report and bounded-proof controls; inventory and diff checks pass. The replacement full package race is running on the corrected source.

### 2026-10-24 — Package race qualified; recorded suite retained

Full couchcmd race on the corrected source passed177.165s (exit0), raw log /tmp/pair860-full-race-final.log. Repository lint previously passed17.395s; repeated hub controls4.208s, analogous consumer controls9.189s and child-proof controls8.246s all passed20 race repetitions. No code changed after the corrected source was pushed.

Pre-close on60e7b5c2fcac6590d65d37c2085f54fc9152868d resolved to affected, then correctly selected the full suite because the audited reader certificate changed. Retained run lease-1964064630 / attempt e517daf3cc672fe0fea983bcb14802c2 in detached checkout /Volumes/TestScratch/local-checks/sdlc-local-2276976659 is queued; no suite pass is claimed. Close review may overlap that queued check under tl-worker's explicit rule, and a recorded pass must precede landing. The final Plan row now records the completed push/check initiation, with close and landing kept as workflow gates rather than a falsely completed checkbox; this is a bookkeeping clarification, no scope change.

### 2026-10-24 — Boundary round1 findings and RED controls

Close returned REWORK: BR-1 Critical child-authority-before-readers and BR-2 Important acceptance-criteria-track-contract. Measured actual0.42h; no close finalized. Own exact native-exit race count20 reproduced RED11.256s; the new source enumeration test failed0.439s naming exactly the three native-exit writes. The previously passing full package run did not expose this schedule. Cancelled queued old-head lease-1964064630 through local-check cancel (exit0), per TL direction; no suite pass claimed.

Addressed the entire connected child assignment inventory: native-exit default, unsupported and lifecycle variants now compose before construction, alongside the already repaired worker-cue helper and custom controls. The sole remaining direct child write is the counted no-wrapper injected-registry control. Extended Done-when and lesson for both findings. Mechanical enumeration reads only package *_test.go sources through os.ReadDir and parser.ParseFile; register its test in the existing SOURCE_GUARDS owner and audit its certificate separately.

The full connected-child family plus mechanical enumeration passed20 race repetitions40.086s. Repository lint passed20.446s. The new reader's inputs were audited as only this package's *_test.go files (directory enumeration plus Go AST parsing); its exact test is registered in SOURCE_GUARDS and its individual certificate is fixed_scanner. The inventory suggested a conservative fallback because its source-reader heuristic did not recognize this scan shape; explicitly selecting fixed_scanner is supported by the audited inputs and registered fixed preflight. Inventory passes after recording only that new certificate. The replacement full package race remains running; no pass claimed yet.

### 2026-10-24 — Completed family qualification before round2

Final full couchcmd race passed211.265s (exit0) on the complete native-exit/enumeration source, raw log /tmp/pair860-full-race-complete-family.log. The affected-selector tests passed13/13 in11.810s after adding the fixed preflight test registration. Inventory reran successfully immediately before close.

Corrected source bb826047d51c4419c8a4bc9cefe5059bb48aaf8b is pushed on PR582. Its new pre-close resolved to affected, then full suite due scripts/test-affected.py; retained lease-1097383963 / attempt8cec4a852c333e3fa50844bd7daaf571, detached checkout /Volumes/TestScratch/local-checks/sdlc-local-284055581, remains queued. Close round2 may overlap the queued run, and landing still requires its completed pass. No recorded suite pass is claimed.
