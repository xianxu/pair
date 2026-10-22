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

- [ ] Capture the native pane metadata and identify the lost/ambiguous ownership witness before changing the proof.
- [ ] Implement and verify the correction through the existing shared disposable-Couch harness and public commands.

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
