---
id: 000810
status: blocked
deps: [791, 796]
github_issue:
created: 2026-10-22
updated: 2026-10-22
estimate_hours:
collaboration-mode: collaboration
card_mirror: '10f8bd26b8ffe1d339b2a16f7120db4f5657b24d' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-22T13:41:11-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
---

# Couch delivery: recover resize waits without typed input

## Problem

Recurring Couch deliveries stall until operator input, before/after pair#735, perhaps twice daily. Static Claude auto/banner and bypass/banner frames and the Codex blank-line/one-warning frame replay as empty on current main; banner alone cannot explain the intermittent fault.

## Spec

Park recovery implementation until both #796 (collapse tab style by default, avoiding wrap-driven pane resizes) and #791 (delayed refusal/resize capture) land. If stalls recur after both, revisit from the capture evidence. Never type recovery input, invent cursor visibility, relax admission guards or deliver to an unconfirmed composer. A one-row geometry bounce is only a fallback candidate and requires explicit TL approval before implementation.

The controlled resize-then-silence mechanism remains valid, but the first native capability experiment disproved the approved unchanged-size SIGWINCH approach for Claude2.1.308: one owned-group signal after a confirmed idle/quiet baseline produced no repaint within2 seconds. No production recovery was implemented. Historical causation is still unproved because old resize events lack timestamps.

## Done when

- Both #791 and #796 land before recovery work resumes; only a recurring stall with new captured evidence triggers re-investigation.
- Any later proposed mechanism has explicit TL approval, isolated Claude/Codex capability evidence and the controlled resize-then-silence red case.
- A later implementation preserves cursor/composer/input safety, bounded work and owned child lifetime, and passes the normal verification/review/landing gates. No immediate recovery change is authorized while parked.

## Plan

[Durable plan](../plans/000810-resize-repaint-plan.md). Its SIGWINCH design is superseded by the final parking decision; retain the measured negative and test strategy for a future evidence-based re-plan.

- [x] Claim and approve a bounded candidate recovery plan.
- [x] Run the native capability experiment first and stop on the disproved premise.
- [ ] Wait for #791 and #796 landing; investigate only if stalls recur with captured evidence, bringing any fallback mechanism to TL before code.

## Log

2026-10-22: Filed by pair:1 under ops:0 approval of diagnostic-only pair#791. Unclaimed because it waits for a future real capture; no active implementation assigned.


### 2026-10-22 — claimed; recovery plan from main

Authenticated TL9a7378bb-68de-4a95-bb1c-129f827a79c1 directs claim and plan now, based on the operator's observations plus the controlled resize-then-silence mechanism. Claimed pair:1; start-plan created independent branch000810-captured-idle-stall at fresh main651e30092e7ca245e5537ba1450b7ea7eb36ddae. No parent commits imported. Parent #791 is SHIP but its recorded suite is still queued: lease-1905342880/session96595, last position14. Poll it at natural pauses; checkpoint this plan and land #791 first when green. #807 remains independently checkpointed/pushed, implementation not started.

First start-plan refreshed ariadne ffd859eb5d7298b0844bc1c6876edad9b7574fd2→050fa89d520394e538938c04e18325e4c2fa299e, then refused the seeded merge-check workflow diff (ariadne#557). Per TL334838d5-215d-4e30-8be5-1d23196b1556, saved exact diff at /Volumes/TestScratch/tmp/pair810-seeded-workflow-4qhmdf3n/merge-check.patch, restored only that file and reran successfully. No local-gates change included.

Read-only discovery confirms signalChildWinch/ownedGroup/existing stdin peer poll are reusable seams. The unreaped leader is the lifetime fence; add serialized signal admission at that owner rather than PID liveness scans. Shared agentfixture honors every size notification, but real Claude2.1.308/Codex0.163.0 unchanged-geometry SIGWINCH redraw is unverified. Codex's JS launcher does not forward WINCH, so conformance must cover group signaling. No native agent launch, signal or #810 source edit has occurred. Durable plan covers bounded state, recovery/loss, one-request growth, stateful fakes and native capability stop condition. Await TL approval before code.

- Authenticated approval4eb28919-f71c-41c0-ab9f-8fd610dfab6b: exact proposed one owned-group SIGWINCH after2s committed cursor invalidation, no PTY recovery bytes, fixed non-resetting sequence, stop/reap serialization, one request/invalidation. Native capability experiment is explicitly authorized now before #791 lands; production waits for parent. Conduct it as scratch overlay on the existing isolated native conformance harness, retaining joined group/PTY cleanup. If Claude or Codex ignores unchanged-size group SIGWINCH, stop and return to TL. #791 retained runner was rechecked alive after Couch restart, lease1905342880/session96595, FIFO13 at last poll.


### 2026-10-22 — native capability failed; parked by TL

Claude2.1.308 native experiment: `env PAIR810_AGENT=claude PAIR810_CAPTURE_DIR=/Volumes/TestScratch/tmp/pair810-native-redraw-vj17_yvk go test -overlay /Volumes/TestScratch/tmp/pair810-native-redraw-vj17_yvk/overlay.json ./cmd/internal/wrapcmd -run '^TestInvestigation810NativeRedraw$' -count=1 -timeout=60s -v` — behavioral FAIL8.298s, solely “unchanged-size SIGWINCH not honored: ignored”. Confirmed idle at13:57:40.424579-07:00, then2s quiet; one owned unreaped-leader group SIGWINCH at13:57:42.500054-07:00, unchanged120x38. No output/repaint in the following2s; native raw remains3722 bytes, request offset3722. Observer-only synthetic hide marker tested fresh native cursor evidence and was never sent to the PTY. Four setup bytes selected trust only for the disposable empty repo; recovery input and ALL post-request PTY bytes, including protocol replies, were0. Group/PTY cleanup returned nil. The scratch overlay reuses the existing isolated conformance helper and ownedGroup, retaining the leader until teardown before reaping; no real fleet slot was touched.

Private raw SHA2568217851e8384d7be6c7afb877c401c6226994909d6cfb4c4ce09530aeba0432c; raw/JSON/overlay/test under the scratch path above. No raw screen or credentials committed. Native Codex was not launched because the explicit stop condition fired on Claude. No #810 production source changes or change-code gate; no geometry bounce attempted.

Authenticated TL1e66b2d3-7981-4155-9929-76ad686f0bc1 parks this issue until #796 and #791 land. Keep the claim in pair:1 so the evidence/re-plan has an owner. Revisit only if stalls recur after both; one-row geometry bounce is a candidate requiring new explicit TL approval. Meanwhile land #791 when its retained recorded suite passes, then resume #807. Next: keep session96595/lease-1905342880 alive and poll it; no new #810 work. #737 remains parked/unstaged as before.


### 2026-10-22 — further operator recurrence before mitigation rollout

Authenticated TL7bbaa056-5c99-4d47-b9c1-dd36cbadc844: the operator's hub wedged again at approximately14:0x with the two-line tab bar visible. #791's capture was not live, so no refused screen was saved. This is another operator observation consistent with the controlled mechanism, not captured proof of the exact live refusal or resize timing. Plan unchanged: keep #810 parked; #796's default collapse removes wrap-driven pane resizing, and #791 supplies future capture evidence. Revisit only if stalls recur after both land; any geometry-bounce fallback still requires explicit TL approval.


### 2026-10-22 — live ops:1 stall captured read-only; stale-wrapper hypothesis

Authenticated TL bcc470b9-3807-4fc6-9ad9-11b8cc68cda1 requested immediate read-only inspection of ops:1, Claude 2.1.306, tag1-ops-90. Owner-provided full receipt (authenticated path notice0e18ee4e-0739-4c6d-aa6c-933a6cbf33c3) identifies delivery817fd5eb-d6c1-48ef-b29c-b70775e9ec5a as queued position2, “waiting for recognized composer”; AcceptedAt14:39:16.545911-07:00 (the original approximate report said14:34). It is not the wrapper's active head: direct status returns unknown delivery. Receipt visibility is actor-specific; pair:1's receipt refusal must not be interpreted as delivery loss.

Frozen private evidence at /Volumes/TestScratch/tmp/pair810-ops1-live-qhad6_48 includes exact raw2616723 bytes (SHA256cab26061704a57265db146db47de0b675af3f7d5f53b98c2001c3bd1d6f29981), resize events13006 bytes (SHA25620e490eb92b8683b788bdf5373250dcefab2ebe592b89f8f6bb6aa4138829236), wrap trace67769 bytes (SHA2568fd7712f3f1b09c541ba7bd9111c665c8e902e1f4e8c23eb4e208697fe2c4c75), owner receipt, binding, read-only RPC responses, manifests and scratch replay overlays. Recheck found all three streams unchanged. No private raw screen committed or uploaded.

Trace: final native output14:38:07.064004, ending offset2616723, followed by duplicate94x39 winsize at14:38:07.442027 and no later output. Direct production endpoint Observe twice returned sequence16313, LastActivity14:38:07.063125-07:00, Submission0; drive-observe returned Idle/Recoverable/Dismissable/Interruptible all false. Live terminal generation and exact head predicate are not exposed by these older read-only endpoints; activity sequence is not terminal generation. Peek/tail renders the emulator cursor; admission snapshots additionally use synchronized observer visibility, so visible peek alone does not prove admission evidence.

Scratch-overlay TestInvestigation810ReplayFrozen replays all captured resize offsets and bytes through the existing harness/proxy. Current code: PASS0.569s, complete=true, observer visible=true, cursor(2,34),94x39, composer EMPTY, paste=true, picker=false. Final footer includes background shell, PR link and persistent update banner and is recognized. First scratch setup failed because remap=false deliberately constructs no terminal; corrected setup to the existing integration fixture's remap=true. A second controlled replay changes only the no-op resize short-circuit to pre-#735 always-invalidating behavior: PASS0.565s, complete=true, observer visible=false, composer UNKNOWN, detail “cursor is hidden”, while Tail still marks the cursor visible. This reproduces the peek/admission disagreement with the actual frozen stream; it is not proof of the exact running binary. Replay turn=true reflects the fake's retained activity flag and is not a measured live turn flag. No safety guard or production code changed.

Live wrapper PID37129 started October20 at22:25:22; mapped executable inode833972180 differs from current disk inode837844445. Running revision remains unproved. Read-only investigation sent no PTY input, signals or redraw, and did not restart or modify the slot. Outcome943b8af9-777d-4e0c-8900-b17b203081fd was verified submitted to TL: capture complete, ops:1 may be released.

Authenticated TL6e54f49d-cdcf-44db-b1f4-53714a10c64b reports #735 landed October20 at13:25, after ops:0 PID2737 started10:28 and roughly fifteen other wrappers started that morning (ariadne, parley, pair:0 included). TL considers stale pre-#735 wrappers the very likely fleet-wide explanation, has asked the operator to Alt+n ops:0, and expects tonight's full shutdown to refresh the rest. These rollout actions are planned, not yet verified. Filed both follow-ups unclaimed on origin: #817 for actual wrapper-vs-disk revision visibility and #818 for reassessing #810 after all wrappers carry #735 (#818 waits for #791/#796). #810 stays parked: no recovery implementation, native retry or geometry bounce authorized. Next remains #791's retained recorded suite and landing, then #807.


## Revisions

### 2026-10-22 — from capture wait to bounded repaint recovery

Original Spec waited for the first real failed-admission capture, then a recognition fix; original Done when required that first capture and its exact refusing condition. TL now accepts the controlled actual-resize→silence→cursor-hidden reproduction plus operator observations as enough to act. Current Spec/Done when replace that exclusive wait with harmless redraw and re-observation, retaining native capability proof and all input-safety guards. This is a separate recovery mechanism, not a footer recognizer change. Parent #791 dependency and land-first sequencing are explicit; no stacked development.


### 2026-10-22 — parking supersedes approved recovery implementation

The preceding approved SIGWINCH design and five-second recovery contract are historical and unimplemented. Claude's negative native result activates its stop condition. Current Spec/Done when/Plan record the new dependency and evidence-based wait rather than claiming the failed mechanism works or silently replacing it. TL owns any fallback choice and approval.
