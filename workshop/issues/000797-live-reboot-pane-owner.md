---
id: 000797
status: working
deps: []
github_issue:
created: 2026-10-22
updated: 2026-10-22
estimate_hours:
severity: high
collaboration-mode: collaboration
card_mirror: 'e30b4e9437856ae7b4b0037e290957a5381e0823' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-22T09:32:31-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "64c232d4", done: "74ab5a33"}
---

# Live reboot refuses owned pane as ambiguous

## Problem

The owning hub can obtain a live worker's exact reboot confirmation offer, but acknowledging it fails during session quiescence: `ownership unresolved: unknown pane owner: pane has no unambiguous actual command`. The current incarnation remains unreplaceable through the public recovery path. This is separate from #791's busy/banner admission problem, where the same reboot refusal appears only in the Log.

## Spec

Diagnose and correct the native pane-owner proof used by live reboot/recovery confirmation. Preserve fail-closed ownership and exact process-generation fences; do not treat an unknown actual command as permission to kill, weaken the validator, or increase deadlines. Return distinct evidence for unsupported/missing/ambiguous pane data so a legitimate owned session can be recovered through its public confirmation protocol.

## Done when

- A deterministic fixture reproduces the observed live worker and its native pane metadata through the actual ownership parser/decision; the corrected proof succeeds only for positively owned panes.
- Public live reboot followed by the returned recovery-confirmation token succeeds for genuine disposable Claude and Codex workers, while foreign/unknown/reused ownership remains refused.
- Preserve exact stop-set confirmation, cleanup and current caller authority; no numeric deadline tuning or permissive fallback.

## Plan

- [x] Capture the native pane metadata and identify the lost/ambiguous ownership witness before changing the proof.
- [x] Implement and verify the correction through the existing shared disposable-Couch harness and public commands.

## Log

2026-10-22: filed unclaimed per ops:0 approval d2356d3f-461c-4576-9ceb-a71acb4179f6 while qualifying #792. New fixture retained six descendants, proved their genuine ancestry, then released their parent to reparent them. All barriers passed. Public `couch --reboot fixture:1 --confirm --json` returned a valid confirmation-required offer excluding those detached children. Acknowledging its token failed before any #792 orphan-reap effect.

Exact failure: `slot zellij session ... kill not confirmed: session ... ownership unresolved: unknown pane owner: pane has no unambiguous actual command`. Recovery-confirmation receipt ID `1e3f6b40-c5cb-4e3d-8436-8bdb70a0e78e`, failure nonce `operation-c585c70f0788d3a5d29ae5372fb3526c`; queued 0s, prepare 1ms, operation 393ms. The offered wrapper generation was PID80337/start1792659043.371776 and Claude child PID80545/start1792659043.605850. Exact disposable cleanup passed. Overall native command joined in31.258s.

Local retained log: /Volumes/TestScratch/tmp/pair-slot2/pair-792-native-confirmation.kamQRO/native.log, SHA256794f8c0dce57ec187b5ae7eda2c90e80fd35a9974b280c4036e96eb8b40f4d0e (verified with shasum). The private runtime root was removed by normal cleanup; this text retains the actual refusal and correlation independently of scratch retention. Product/harness source is #792's f3f724125 checkpoint; subsequent uncommitted native-test changes only followed the valid public confirmation contract. No validator fix was attempted here. #792 uses a separately approved fixture-only crash to qualify its independent reap behavior.

An earlier native steward start missed its wrapper-ready publication before any retained fixture setup. Later unchanged-product startups succeeded. Cause remains unclassified; see #773's related missing-ready evidence class without assuming that issue's root cause. No flake registration or passing retry is claimed for that startup failure.

### 2026-10-22 — Claimed diagnosis checkpoint

Claimed by pair:2 and started from landed main after #792/PR541 completed; no previous suite or merge runner remains active. start-plan's automatic weave catch-up generated an unrelated merge-check.yml change (drain printf rather than grep -q under pipefail). Preserved that exact generated patch in a named Git stash on main-slot2 rather than mixing a gate configuration change into this issue. The second start-plan succeeded. No production code edited.

The retained incident log preserves the exact refusal and stop offer but not the original pane JSON. Current read-only native list-panes on this slot reports ordinary valid wrapper, term and draft commands; that is current diagnostic evidence, not a reconstruction of the failed disposable session.

Code-grounded causal path: recovery_prepared.go wraps Fresh retirement with withSlotTree, confirms and preserves, may park a client helper, then calls retirement.Retire. recovery_slot_tree.go calls any original retire, TERM-signals added wrapper/children, awaits their exit, and only then calls QuiesceNamed. artifactcollision.go's QuiesceNamed reacquires NamedPairSessionContext -> SessionOwnerProbe -> projectSessionOwnerReply -> ClassifySessionOwner from live panes. Thus the legacy fallback destroys the processes before obtaining the pane evidence needed for its final session stop. Existing slot-tree tests use independent FakeProcOps and FakeThreadArtifactCollisionChecker; process death never changes the fake pane evidence.

A separate producer-contract hazard complicates diagnosis: local clean Zellij source a162323384dff3fbd9c68a4c0971f4f6efca424a at /Users/xianxu/workspace/zellij sets pane_command to command_vec.join(" ") (route.rs3033-3055) and omits it on a100ms query failure; data.rs2418 skips None. Pair parses that lossy string as shell-quoted argv and any missing/unparseable non-plugin pane vetoes the whole observation. The old incident's actual cause cannot be chosen from its generic diagnostic alone. No fallback to terminal_command, ignored unknown pane, validator weakening or timing increase is proposed.

The existing confirmed native terminal path already captures exact servers/descendants before retirement and allows accepted members to disappear while rechecking survivors, argv, ancestry and new arrivals. Reuse is preferable to inventing a second authority cache (ARCH-DRY/ORDER). A narrow reorder must also account for parkPreparedRecovery, which runs before Retire and can remove the same witness; moving one inner call alone is not yet an established complete fix. See the durable capture-first plan.

2026-10-22: authenticated TL approval9d722e7b-5b6a-4b7e-8907-d5d045742c03 authorizes diagnostic capture/replay first and requires retaining judged pane JSON in refusal evidence. Existing production NativeQueryEvidence already propagates into bounded LifecycleFailure.Zellij* fields; the harness's pre-cleanup failure renderer currently prints none of those fields. Diagnostic stage will preserve those existing bytes before cleanup, then capture both personalities and return for concrete correction approval. No parser or deadline change.

2026-10-22 diagnostic implementation checkpoint: change-code admitted the approved test/evidence stage as inferred quick flow (no flow pin or waived gate). The production ownership and recovery ordering are unchanged. nativeFailureEvidence now retains existing bounded NativeQueryEvidence fields before private-store cleanup. Delegated TDD observed missing-field/bounds failures before the renderer fix, then focused tests passed0.520s and the complete testcouch package31.305s. Root's combined characterization/evidence/Godoc/catalog selection passed core1.147s, testcouch0.258s, termcmd0.477s and repolint5.150s.

New real-probe characterization couples FakeProcOps retirement to pane command loss through RecoverSelection/ConfirmRecovery and the actual scoped checker: stop500 -> stop501 -> stop502 -> panes:stopped -> ownership refusal, unchanged exact server, no deletion or replacement launch, unrelated PID600 live. Empty-command, null, ambiguous, foreign and replacement-server probe cases pass. Shared fixture serializer always inserts pane_command; exact native omission still needs the captured producer reply, not an independent fake serializer.

Opt-in TestLiveRecoveryOwnerDiagnostic uses the shared disposable fleet, same retained-six setup, genuine public confirmation and correlated production failure store. Its purpose is to reproduce and retain the refusal, not claim desired successful recovery. Setup180s, operation120s and cleanup15s remain bounded, each personality runs in its own Go process. New test reader is classified non_go_full_fallback because Build consumes executable/runtime repository inputs. Native captures are next; checkpoint precedes those long calls.

### 2026-10-22 — Both native diagnostic captures complete

On unchanged capture candidate725e24c574b2907d79121071ddff26bbe2ddd685, separate Go invocations reproduced the generic ownership failure with complete NativeQueryEvidence and clean owned teardown. Both actual replies show pane0 exited=true/is_held=true with no pane_command, alongside a live correctly owned draft and pair term. The old agent identity is absent. Both queries exited0, stdout untruncated, stderr empty. This is our own stopped witness, not evidence that the strict validator should accept missing data.

- Claude: Go54.627s/sub43.24s; write81326080 of100663296 bytes; query3817 bytes. Log /Volumes/TestScratch/tmp/pair-slot2/pair-797-native-claude-dhksynpq/native.log; SHA2567903f861652f28a3770ef5f7e3ca6d9422326849710c28c2dcb03cb5e7852a55. Complete stdout SHA2562a8cb3eeb139ed5d42c0eaf228ab97209847b57e8754f0f58cb78572815ffc3e.
- Codex: Go54.948s/sub45.24s; write81412096 of100663296 bytes; query3816 bytes. Log /Volumes/TestScratch/tmp/pair-slot2/pair-797-native-codex-g36jmzd8/native.log; SHA25664e3406fd3005630657928f32fbb412c2004e8e4aadfe38e3c50dab87cb7f02f. Complete stdout SHA2564f086b74b8468f0bf1322590c26a90a13f299bb53b7dfc0e40af56a5a7cae3bc.

Both native runners joined; no stage or runtime/check runner remains. Actual replies are retained verbatim in tests/lib/zellij_fixture_recovery_owner_cases.json and replayed through the shared fixture serializer/catalog; strict classification remains Unknown although the captured live draft alone proves its correct owner. Normal capture replay passed launcher0.352s and full fixture catalog14.987s. No missing-ready event, retry or flake registration.

Further call-graph verification narrows the outer-park concern: Pair RunCleanup gates all later stages on session quiescence, and a failed ParkExpected stops before Retire. Successful normal park therefore already removes the session. Proposed correction is the small inner ordering change plus an exact-server handoff through the existing authorized-quiescence loop, closing the current check-then-re-admit generation window without new state. The durable plan lists all three scoped deletion consumers and the replacement/unknown/cancellation tests. Awaiting hub approval of this concrete production correction; no proof/ordering implementation has changed.

2026-10-22: concrete correction approved by authenticated ops:0 receipt a5fddddd-f74f-4737-99de-8bf35d5658b5. Implement the approved ordering and exact-server owned-quiescence path shared by all three scoped deletion consumers. Unsupported deleters refuse, no name fallback. Outer park, signal cadence, bounds and strict classifier decisions remain unchanged; replacement/unknown causal tests and successful native reboot+delivery precede close.

2026-10-22 implementation checkpoint: original retirement remains first, then named-session quiescence precedes the additional wrapper signals/waits. One shared scoped helper requires positive ownership and DeleteOwnedSessionContext at all three deletion sites; foreign observations and unsupported name-only deleters refuse. Native deletion carries the immutable exact server into the existing authorized quiescence loop. No parser permissions, normal park ordering, deadline or signal cadence changed. Diagnostic text now identifies the failing pane and missing/ambiguous command cause without copying command text.

TDD: all six supported/unsupported caller regressions failed on old name-based effects, then passed with exact proof handoff. Native owned-deletion regressions failed for the missing API, then passed with replacement-before-first-census/during-loop, identity/socket changes, absence, cancellation and invalid-proof coverage. Causal recovery regression failed with stop-before-pane-proof, then passed with live proof -> owned quiescence -> stops -> one replacement. Unknown/null/empty/ambiguous/foreign/replacement/quiescence-error and original-retirement-error cause no added wrapper signals or launch. Selected ownership/recovery tests passed launcher1.918s and couchcore3.999s. Disposable success checks follow on this committed candidate, one personality per Go invocation with the unchanged96MiB write cap and time/output bounds.

### 2026-10-22 — Native success acceptance on ecefebc3469bda44d3d28797bcdcb4334998c752

Both separate personality invocations passed genuine public reboot -> returned confirmation -> new exact binding/generation -> public current-worker message acceptance. Old agent identity was absent, every owned generation was absent at cleanup, and direct child plus IO joined. Conservative warnings about unproved retained remnants remain visible; none were authorized by inference. No stage, crash substitution, retry, bound change or flake registration.

- Claude: Go78.150s/sub58.83s, writes81833984/cap100663296. Log /Volumes/TestScratch/tmp/pair-slot2/pair-797-success-claude-12v5rq4u/native.log, SHA2566bd7fbdc78eff02f8b6cf6334a34ab5bc22a00300d6aa54beeb1c453abaa7a77. Old27992/start1792688901.713202 -> current41708/start1792688922.772506, message recovery-owner-current-worker-claude accepted by currentPID. Succeeded receipt8862f39d-be6f-43c2-ba5e-14fed7b07220.
- Codex: Go81.866s/sub65.70s, writes81870848/cap100663296. Log /Volumes/TestScratch/tmp/pair-slot2/pair-797-success-codex-armrrb2m/native.log, SHA2569e7fa3eac2b38ecbc0a6bea7fe90d327631c6e44f05283c6a4a2cc8e0b3eb60d. Old82650/start1792688991.810959 -> current95890/start1792689015.707303, message recovery-owner-current-worker-codex accepted by currentPID. Succeeded receiptcb2aa5c5-e2a9-49f0-9f1e-0ca99da2a42b.

Additional focused evidence/catalog selection passed couchcmd0.392s, testcouch0.343s, zellijtest18.355s; repository/inventory/Godoc selection passed4.574s. Four affected-reader certificates were re-audited: public native executable test remains full fallback, other changed fixtures remain literal/local only. Reader check passed. PR545 opened; local-gates guard passed with no protected paths touched. Recorded affected check selected the full suite on exact candidateecefebc3, run lease-2188589775, retained terminal88486; queued normalFIFO position9 behind existing capacity. Close review may overlap this queue under tl-worker; a recorded pass still precedes landing. Full affected-package and focused race runs remain active independently.

2026-10-22 close r1 REWORK: BR-1 identifies a documentation classification error, repaired by splitting immutable RecoveryTerminalPlan data from IO-dependent checkConfirmed/readRecoveryReady, with a Revisions entry and lesson. BR-2 retains a new reviewer Codex replacement-start failure on003e3ba8 after successful old retirement: Pair exit1 before registration, Lost connection to Zellij server, terminal client exited before admission verified. Reviewer Claude native passed; the earlier own Codex pass is retained but does not dispose of this later failure. Investigating exact reviewer transcript/diagnostics before retry or new production changes. Both complete affected packages passed: couchcore583.465s and launcher108.135s. Focused race selection passed core5.895s/launcher1.421s. Recorded suite remains queued unchanged.

### 2026-10-22 — BR-2 scoped disposition and exact-head diagnostic

Authenticated ops:0 disposition5a670f4e-1ef7-49da-99ba-835f86d2814b directs filing the distinct replacement-before-first-wrapper disconnect unassigned, then re-closing #797 with exact-head success and retained unclassified failure; no safety-test flake registration. Filed #806, replacement Zellij startup disconnect, linking #773 and preserving both logs/hashes plus the early-query hypothesis. The ownership/quiescence ordering correction is established on both agents; no new production change or deadline tuning is made for this startup discovery.

Reviewer failure retained at /Volumes/TestScratch/pair797-br2-transcript-fgmhr1gg/reviewer-codex-native.log,9233bytes, SHA256b86a30bb1b3d2fb642557ee91b7a511807a9ce5926fcf6a2feb95166bf07a99d. Old -2 retired; distinct replacement -3 disconnected before its first wrapper startup. New exact-owned quiescence completes synchronously before Fresh launch and cannot authorize a different name/generation; ordinary park also waits for the helper's death. Thus the failure boundary differs from #797's fixed post-stop ownership refusal. The initiating startup trigger remains unclassified.

Controlled diagnostic uses exact reviewed003e3ba8deb1dd28cedb89b58774a1295b7916e6 source in a detached scratch checkout. A Go overlay only reads at most8192bytes from the existing private zellij.log AFTER confirmation; no production, pre-confirmation timing, acceptance assertion or resource-bound change. Overlay SHA25671e6f37992708b2d6525b562fb66c9b3a06cdd8f5c259f55e9a39e1aaae03c54. Codex public confirmation, changed exact generation, current-worker message acceptance and complete cleanup passed Go30.182s/sub23.82s; writes81977344/cap100663296. Log /Volumes/TestScratch/tmp/pair-slot2/pair797-br2-probe-o0bm5t4z/native2.log, SHA256bd819f74b2424feaa2cf0886e4571d5a4581732bc7b9f72afd547fa3f9155b73. All7236bytes of the private server log retained; Server-not-ready/retry entries but no panic. This passing measured probe does not erase or explain the failed startup. Initial scratch probe compile missed generated runtime bundle before any native run; normal runtimebundle generation resolved that preparation error, preserved separately as native.log.

Source investigation found an independent executable overlap: message ListingPoller immediately and each1s issues global list-sessions without the terminal birth lease; resource-inventory seed/lifecycle probes also query independently. Resource sampling and Git/filesystem prewarm do not. #806 records this as hypothesis, not demonstrated crash causality. These workers are unchanged by #797. BR-1's corrected PURE/INTEGRATION classification and all core-concept rows were checked against code. No behavioral test was added for that prose-only correction. Existing recorded suite lease-2188589775 remains queued on unchanged production/test sourceecefebc3; later issue/plan/lesson/review changes are documentation only.
