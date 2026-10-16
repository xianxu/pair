---
id: 000618
status: working
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
card_mirror: '34a2428ab1415719a8f423d302770c1f83c1c2d2' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-16T15:08:44-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
---

# Slots recovered by reboot or resume after a Couch restart can send but not receive

## Problem

After the 2026-10-16 14:57 Couch restart, a slot recovered with `couch --reboot` (pair:3, fresh, tag 1-pair-58) or
`couch --resume` (ariadne:5, tag 1-ariadne-46) can **send** to the hub but **cannot receive**: every
`couch --send-to` to it returns "Query couch --message-status … before any further send", and the status says
"receipt unavailable: no connected recipient retains this message". The agent is live and idle at its prompt.

- ariadne:5 recovered only after `couch --relaunch ariadne:5 --confirm --same-binary` (a plain relaunch refused as
  stale-binary, "changes nothing").
- pair:3 can't be relaunched either: "resume-binding-provisional: its agent has not completed a turn yet", although
  its agent completed a turn (it sent the hub its recovery report).
- The first `--relaunch` of pair:7 and of ariadne:5 each timed out at ~16 s and left a "recover / abandon" choice;
  a second call then succeeded.

So a recovered slot looks healthy in `couch --list` and to the operator, yet silently drops the TL's dispatches.

## Spec

- Find why the recovered wrapper is not a connected recipient (registration on reboot/resume vs. on first turn;
  the binding stays "provisional" after a completed turn).
- One state, one action: a slot that can't receive shows so in `couch --list`, with the one command that fixes it.
- The first relaunch call shouldn't strand a recover/abandon choice on a ~16 s timeout.

## Done when

- Tests: a rebooted and a resumed slot each receive a `--send-to` without a relaunch; a completed first turn makes the
  binding non-provisional.
- Live: after a Couch restart, reboot one slot and resume another; both receive a send.

## Plan

- [ ]

## Log

### 2026-10-16

- 2026-10-16 (pair:2, read-only on pair:3): pair:3's wrapper (pid 36072) is registered and connected to the live
  supervisor (71982); binding, endpoint socket hash, pid file, ready-file nonce/session and session index all match; the
  broker never displaced it (supervision.jsonl subject cf45653c, one incarnation). Its codex rollout shows the TL dispatch
  4a662842 delivered at 15:03:40, after the reboot; my probe d166c04c at 15:23 was submitted. The TL's failed sends to
  pair:3 were at 15:06:46 and 15:07:47 (6ba71ae7, 0803aeee); to ariadne:5 at 15:04:44 and 15:05:07. No file in pair:3's
  scope changed between the failures and the success.
- Cause, measured: every send runs the supervisor's use-time ownership check about three times (sender once; recipient
  before Observe and before Reserve), each calling `zellij list-sessions` twice, inside the 2 s AdmissionTimeout. At load
  average 136 (~50 test processes) one listing took 0.13-1.07 s, so admission times out before Admit, no receipt is
  stored, and `--message-status` can only say "no connected recipient retains this message" (the status path skips the
  check). Hub deliveries in the same 15:05-15:09 window also show uncertain outcomes. Not reboot/resume-specific.
- Provisional: a codex launch has no chosen session ID; only the session watcher confirms the binding, from Pair-log
  turns that only the draft pane writes. A slot driven only by peer messages never confirms, so relaunch refuses.
- ~16 s relaunch: the park polls 15 s (`CompletionTimeout`) for the quit completion and child death, then returns
  ParkIncomplete with recover/abandon.
- Plan sent to the TL for approval:

  > pair:2 #618 PLAN (needs your approval: a new failure record and a cached liveness input). Root cause of can't-receive, measured: NOT reboot/resume-specific, it is load. Every send runs the supervisor's use-time ownership check about 3 times (sender once, recipient before Observe and before Reserve), and each run calls zellij list-sessions twice: ~6 synchronous listings inside the 2 s admission budget. Under today's load (load avg 136, ~50 test processes) one listing takes 0.13-1.07 s, so admission times out before Admit, no receipt exists, and --message-status can only say "no connected recipient retains". pair:3's files were identical at the 15:06/15:07 failures and my 15:23 success; hub deliveries in that same 15:05-15:09 window also show uncertain outcomes. Recovered slots were just the ones you were messaging then.
  > Plan:
  > 1. Root fix: the hot-path authority check answers terminal liveness from the supervisor's polled session presence (bounded freshness), keeping the cheap exact checks (pid file, ready file nonce/session, session index) per request; one check per party per admission, not three. (Deterministic request path, no per-item probes.)
  > 2. Visibility: a failed admission leaves a short-lived failure receipt with its reason, so --message-status says why (e.g. "recipient ownership check timed out") instead of "no connected recipient", and the send prints that reason on its last line.
  > 3. One state, one action: a slot whose last admission failed (with no later success) shows "can't receive: <reason>" in couch --list with the single command that fixes it.
  > 4. Provisional (root-caused): only Pair-log turns written by the draft pane confirm a codex binding, so a slot driven only by peer messages never confirms. Fix: the session watcher also confirms from a native round whose user text matches a message the broker delivered to that wrapper (exact body, after pid-not-before).
  > 5. 16 s relaunch: the park waits a fixed 15 s for the quit and child exit, then strands recover/abandon. Fix: the first call keeps waiting within the operation (async, operation-status) up to a longer bound instead of stranding.
  > Tests: a fake slow zellij (1 s per listing) still admits within budget; a failed admission reports its reason; a peer-message-only codex thread confirms; relaunch with a slow quit completes on the first call. Live: under a full suite, sends to rebooted and resumed slots land.
  > Full flow, durable plan in workshop/plans; I start with 1+2 (the root and the visibility) once you approve.
