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

## Revisions

### 2026-10-22 — from capture wait to bounded repaint recovery

Original Spec waited for the first real failed-admission capture, then a recognition fix; original Done when required that first capture and its exact refusing condition. TL now accepts the controlled actual-resize→silence→cursor-hidden reproduction plus operator observations as enough to act. Current Spec/Done when replace that exclusive wait with harmless redraw and re-observation, retaining native capability proof and all input-safety guards. This is a separate recovery mechanism, not a footer recognizer change. Parent #791 dependency and land-first sequencing are explicit; no stacked development.


### 2026-10-22 — parking supersedes approved recovery implementation

The preceding approved SIGWINCH design and five-second recovery contract are historical and unimplemented. Claude's negative native result activates its stop condition. Current Spec/Done when/Plan record the new dependency and evidence-based wait rather than claiming the failed mechanism works or silently replacing it. TL owns any fallback choice and approval.
