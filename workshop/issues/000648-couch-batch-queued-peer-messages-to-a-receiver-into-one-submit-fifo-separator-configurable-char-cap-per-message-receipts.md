---
id: 000648
status: working
deps: []
github_issue:
created: 2026-10-17
updated: 2026-10-17
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'cdfa43f91f81e7c20a1012857f2707231fbf1fe5' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-17T11:13:12-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
---

# Couch: batch queued peer messages to a receiver into one submit (FIFO, separator, configurable char cap), per-message receipts

## Problem

## Spec

Operator decision, 2026-10-17, prioritized because the TL thread is under pressure. When a receiver's composer becomes available and two or more peer messages are queued for it, Couch submits them as one paste instead of one turn each.

Today (pair#629) each queued message drains as its own submit, so each starts a new agent turn. For the hub that means a full turn of context and overhead per report. Every message already carries its own `[Couch peer from SLOT; delivery ID]` header, which is how the receiver tells senders apart, so turn boundaries carry no information the receiver needs.

Rules:
- **FIFO:** messages keep queue order inside a batch, with a clear separator line between them, and each keeps its header exactly.
- **A size cap:** a batch holds messages up to N characters. N is configurable, for example `COUCH_PEER_BATCH_MAX_CHARS`, with a safe default under the smallest known composer paste limit. pair:3 measures Claude Code's and Codex's paste limits to choose the default. A message larger than N goes alone, as today. Remaining messages wait for the next free composer.
- **Receipts:** every message in a batch gets its own receipt. All of them go to submitted on the one confirmation, and all of them go to indeterminate or refused together if the paste fails. No message is reported delivered unless its batch was.
- **Scope:** the hub first. Any receiver, if it is as safe; say which.
- pair#629's note that "same-receiver input cannot be batched safely": say whether it was about receipts, which the rule above covers, or something else, and handle it.

## Done when

- A fake busy receiver with five queued messages gets one paste with all five in FIFO order, separated, and five submitted receipts. A test covers it.
- A cap smaller than the total splits the messages into ordered batches, and a single over-cap message goes alone. Tests cover both.
- A failed batch paste marks every message in it as not submitted.
- The live check after a restart: the hub, made busy with ten queued reports, receives them in one or two turns, not ten.

## Plan

Durable design: [queued peer-message batches](../plans/000648-peer-message-batches-plan.md). Full flow; one atomic close boundary. Implementation waits for hub approval and the plan-quality gate.

- [ ] Implement/test shared envelope selection, character cap and atomic pending groups.
- [ ] Implement/test writer-owned preparation, freeze-at-readiness and one batch confirmation with per-member receipts, cancellation, recovery and exact Pair-log turn correlation.
- [ ] Run targeted and recorded full-suite verification, coordinate the real busy-hub restart check with the hub, close review and land through the normal gates.

## Log

### 2026-10-17

- Operator priority via verified submitted delivery `5f52a2ca-4b9c-4bd0-ae51-923cf3c9316c`: parked #620 cleanly (draft PR #407, integrated suite still queued and cancelled before starting, unclaimed with durable handoff), then ran `tl-worker --help`, claimed #648 and started this fresh main-based issue branch. #620 park report `52933a9f-5b5a-4d0f-a32a-5546f8ee8c0e` was submitted. No #648 production code changed before plan approval.
- Measured real Claude Code 2.1.302 and Codex CLI 0.163.0 in newly owned isolated PTYs; no prompt submissions or existing-slot input. Bracketed-paste text was exported through native Ctrl+G to an owned editor and compared completely by bytes/SHA256. Claude retained every ASCII size from 8,192 through 1,048,576 and mixed Unicode at 32,768 code points. Codex retained ASCII from 1,024 through 262,144 and mixed Unicode through 65,536. No rejection ceiling was found; these are lower bounds, not claims about arbitrarily large input or context limits. Detailed method, hashes and scratch manifests are in the durable plan. Parent independently verified every Claude input/export pair; corrected Codex manifests exclude an initial last-line-only clearing probe error. All probe PTYs exited, copied Codex auth removed, no repository edits by probe agents.
- Proposed cap: 16,384 Unicode code points including headers/separators, configurable 1–32,768, plus the unchanged actual encoded transport-frame bound. The default is half the common mixed-text measured size and one sixteenth of the smaller ASCII bound. A complete over-cap head goes alone; no follower can be skipped to fit a later message.
- #629's archived issue/plan explicitly excluded batching and explained that shuffling a same-receiver queue cannot bypass the shared composer wait. They contain no measured paste incompatibility. Its implementation's reservation, pending ownership, retained receipt and confirmation are single-ID. #648 must share input ownership/confirmation while preserving each original ID; concatenating bodies alone would break receipt recovery and exact Pair-log turn correlation (#618). Historical screen text is unavailable, so the original conversational wording behind "cannot be batched safely" cannot be verified beyond those artifacts.
- Design: prepare the held head without pasting; use the existing input writer's readiness decision to request a one-time broker freeze of the current FIFO prefix, including reports arriving during the busy turn. Finalize immutable membership once; the same post-paste confirmation decides all member receipts. Hub-only first rollout for Claude/Codex; Qoder and ordinary receivers keep singleton delivery. No new public receipt statuses, retry, timeout increase, second mutable wrapper queue or readiness-policy copy (ARCH-DRY, ARCH-PURE, ARCH-PURPOSE). Durable plan names cancellation races, retained follower lookup, encoded bounds, stateful fakes and exact combined-turn recording.
- Live check/staging is pending fresh hub coordination for #648. Earlier issues' deferred-restart/no-:0-staging instructions are historical and are not reused as authorization. Required check: after the operator-coordinated restart onto the selected build, queue ten numbered small reports while the hub is busy, inspect all ten held/queued receipts, then observe FIFO separated headers in one or two user turns and ten submitted receipts from shared confirmation.
- Verified new TL delivery `3914bb37-e29b-49c9-a2e5-8a34fecab301`: if close refuses actuals for a pre-ariadne#420 suite attempt missing exact queue endpoints, use the precise `--no-actual` acknowledgement and name the legacy attempt/exclusion in verification; do not reconstruct evidence. Applies only if that refusal occurs.
