---
id: 000648
status: working
deps: []
github_issue:
created: 2026-10-17
updated: 2026-10-17
estimate_hours: 2.805
collaboration-mode: collaboration
card_mirror: '452033751fac3e86bef2e3497fa75cb839efbfcc' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-17T11:13:12-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
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

- [x] Implement/test shared envelope selection, character cap and atomic pending groups.
- [x] Implement/test writer-owned preparation, freeze-at-readiness and one batch confirmation with per-member receipts, cancellation, recovery and exact Pair-log turn correlation.
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
- Plan report `53a8f43e-1dc1-4041-a3ca-778fbabb1433` was submitted. Verified submitted approval `f937ca8e-4c96-4d1f-90df-2fd2b5d6c9f4` approves the hub-only Claude/Codex scope, readiness-time FIFO freeze, per-group cancellation/confirmation, per-member receipts, singleton legacy fallback, exact combined Pair-log turn and cap/default. The hub explicitly deferred #648's live check to its restart after landing and forbade pair:0 staging. The hub will hold a long turn while this slot and two other slots send ten numbered reports, then verify one or two turns and ten submitted receipts. Proceeding to the full-flow plan-quality gate; no production code yet.
- Full-flow plan-quality round 1 raised Important PQ-1 (function-level test strategy). Round 2 accepted the named production functions/adversarial strategies and disposed PQ-1; no open blockers. No architectural/mechanism change. Deriving the estimate only after that acceptance, as required.
- `change-code --flow full` admitted implementation after the 2.805-hour calibrated estimate. Estimate-quality returned INFO: component allowances include their own test development/debugging; the shared API-integration primitive covers socket/cancellation/compatibility races, representative native conformance and deferred live-check coordination. Issue-spec and the two discovery primitives include already incurred planning/measurements; the review primitive covers the one boundary/check administration, not every component's tests. No estimate bypass.
- Implemented the approved hub-only Claude/Codex path: prepared head cannot paste; one sticky assembly offer from the normal input writer lets the broker freeze the current capped FIFO prefix. The normal writer still checks current eligibility after finalization. All frozen IDs share one job and atomic progress/outcome, while per-ID wrapper retention, admission timestamps, allowance, supervision facts and session-sync ledger rows remain original. One exact combined Pair-log user turn shares the eight-turn/256-KiB launch budget. Ordinary receivers/Qoder and explicitly refusing legacy wrappers retain singleton delivery. No uncertain exchange retries.
- TDD/negative controls: missing API reds preserved in `/tmp/pair648-{pure,model,broker,wrapper}-red.log`; endpoint behavior reds `/tmp/pair648-endpoint-red.log` and `-red2.log`; service behavior red `/tmp/pair648-service-red-corrected.log`. Disabling `FreezeBatch` made the five-receipt model fail immediately (`/tmp/pair648-model-behavior-red.log`); disabling batch registration in the real socket fixture failed on input before freeze (`/tmp/pair648-wrapper-socket-negative.log`), restored before green. A frame-fitting but unqualified >32,768-code-point request initially passed validation (`/tmp/pair648-qualified-cap-red.log`); now the endpoint also enforces the qualified hard maximum independently of configured selection.
- Focused green evidence: pending-group/broker/cap tests `/tmp/pair648-core-green-final2.log`; service rollout/cap/provenance tests 0.549s and race 1.578s (`/tmp/pair648-service-{green,race}.log`); original/new endpoint tests 13.471s, batch race 3.179s and expired-initial-deadline race 2.293s; wrapper package 79.680s then final focused 1.650s. Real-socket broker→remote endpoint→wrapper test passed 1.369s and race 2.477s, with five independent senders, one exact FIFO paste/submit, five receipts, and fresh-broker follower recovery without input replay. All owned servers/clients were cleaned up.
- Fuzz evidence: selection 13,437 executions, validation 2,219, preparation 25,138, endpoint payload isolation 79,816; after hard maximum validation, selection reran against the actual configured domain (1–32,768) for 57,816 executions, PASS (`/tmp/pair648-qualified-cap-fuzz.log`). Character cap and actual JSON/frame bounds are independent; artificial high-cap framing tests remain separate from accepted composer policy.
- Native representative batch conformance: fresh Claude 2.1.302 and Codex 0.163.0 PTYs exercised actual prepare/readiness/finalize/paste without submission; short three-member Unicode scenario passed Claude 3.94s and Codex 3.99s. Codex's first five-tall-member probe pasted but timed out on the fixture's post-paste viewport/composer oracle (cursor below the visible parser range); retained `/tmp/pair648-wrapper-native-codex.log`. It is not evidence of failed retention or a product signal to retry. The representative fixture was narrowed to three short envelopes; full byte retention was measured independently before planning, and five-member submission/recovery is covered through real sockets. Claude's collapsed rendering alone is not a full-text proof. Production confirmation remains the existing composer-clear/turn evidence and fixed post-paste delay. Logs: `/tmp/pair648-wrapper-native-{claude-short,codex-short}.log`.
- Three affected packages PASS together unchanged via `go test ./cmd/internal/couchmessage ./cmd/internal/wrapcmd ./cmd/internal/couchcmd -count=1`: couchmessage 28.000s, wrapcmd 57.794s, couchcmd 72.899s. Log `/tmp/pair648-integrated-packages.log`; this is targeted verification, not the required recorded full suite. No production :0 staging or Couch restart has occurred.
- Code and docs committed/pushed as `d62375f23a5e945e70fb8fd8c87475e7e972f9e8`. Requested the required recorded suite on that exact head: attempt `021626ded6c3e87cba897c78a0787fcd`, detached worktree `/var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/sdlc-local-2496699590`, command `make PAIR_GO_TRIMPATH=-trimpath test`. At the first 60-second queue report it had not started and was position 4; preserving this unchanged attempt and its place. Its printed inflight `run.log` does not exist yet while queued; the live local-check process continues reporting queue state. No suite failure/retry or lost-run report has occurred.

## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against baseline-v3.1.md. Method A only. `sdlc estimate-source` flags the calibration stale, so these numbers are provisional ship wall-clock hours, not operator attention or queue wait.

Primitive decomposition: issue/spec and architecture audit use base design 0.6×0.5 and impl 0.2×0.4. Four well-specced Go extensions cover shared rendering/selection/configuration, actor pending-group transitions, wrapper preparation/retention, and per-message accounting plus exact-turn recording: each design 0.2×0.2 and impl 0.5×0.4. The one batch endpoint/broker integration uses design 1.0×0.2 and impl 1.5×0.4. Native-composer discovery is two bounded integrations (Claude/Codex), each implementation 0.3×0.4 with no design charge. Atlas/protocol docs use design 0.1×0.2 and impl 0.2×0.4; the one close boundary including verification/review uses design 0.1×0.2 and impl 0.5×0.4. These are shared components, not a count of files/tests. Library check: existing reducers, endpoint transport, stateful terminal fakes, and standard Go UTF-8/JSON cover this work; no novel library stack or greenfield subsystem. Familiarity 1.0 from #629/#631/#634/#635. Thorough admitted plan: 15% design buffer. Design subtotal 0.70, implementation subtotal 2.00; 0.70×1.15+2.00=2.805 hours.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.3 impl=0.08
item: smaller-go-module design=0.04 impl=0.2
item: smaller-go-module design=0.04 impl=0.2
item: smaller-go-module design=0.04 impl=0.2
item: smaller-go-module design=0.04 impl=0.2
item: api-integration design=0.2 impl=0.6
item: real-api-discovery design=0 impl=0.12
item: real-api-discovery design=0 impl=0.12
item: atlas-docs design=0.02 impl=0.08
item: milestone-review design=0.02 impl=0.2
design-buffer: 0.15
total: 2.805
```
