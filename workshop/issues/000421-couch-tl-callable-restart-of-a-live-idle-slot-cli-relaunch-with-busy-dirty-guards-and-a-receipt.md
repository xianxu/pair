---
id: 000421
status: working
deps: []
github_issue:
created: 2026-10-09
updated: 2026-10-09
estimate_hours: 2.26
card_mirror: 'aa35024978bfcaa4019b284ec69367715b913a2a' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-09T22:53:53-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
---

# couch: TL-callable restart of a live idle slot (CLI relaunch with busy/dirty guards and a receipt)

## Problem

Requested by TL ariadne:1 (operator-approved) on 2026-10-09. After a Pair
rollout, nothing scriptable can restart an idle worker slot onto the new binary.
`couch --reboot repo:N` refuses a live slot ("not-offered: ... is live"), so the
operator pressed Alt+n by hand in 7 slots.

Alt+n is **relaunch** (`couchcore` relaunch; the console's `onRelaunchHotkey`).
It parks the thread and cold-resumes it, which replaces the Pair process with
the current binary and keeps the agent conversation. Reboot, by contrast,
archives the conversation and starts a fresh agent. No CLI exposes relaunch.

## Spec

A TL-callable restart of a live slot, which a dispatcher can script across a
fleet:
- Refuses when the slot is busy: the agent is mid-turn, or its composer is
  occupied (a draft, a suggestion being edited, a dialog).
- Refuses when the slot's checkout is dirty.
- Returns a receipt the caller can verify afterwards, in the shape of
  `--message-status`.

Decided by TL ariadne:1 on 2026-10-09:
- **Relaunch**, keeping the conversation, with the guards and a receipt.
- Also expose **Shift+Alt+N** ("reload context": `pair agent restart`, which
  SIGUSR2s pair-wrap so it starts a fresh agent conversation inside the same
  Pair process) as a second TL-callable operation, with the same guards. It does
  not pick up a new Pair binary; only relaunch does.
- Consider **stale-binary detection**. Tonight a relaunch did not pick up the
  pair#418 fix, because `bin/pair` in pair:0 was built at 11:34 and not
  rebuilt until a manual `make build` at 22:53. A rollout relaunch is only
  worth doing onto a binary newer than the slot's running one.

## Done when

- `couch` has two CLI verbs for a live idle slot: relaunch (the current binary,
  same conversation) and reload-context (a fresh agent conversation). Both
  refuse a busy or dirty slot with an explicit reason.
- Relaunch reports, and by default refuses, a binary that is no newer than
  the one the slot is running (decide the exact rule in design).
- It returns a receipt id; a status query shows the outcome and why.
- Tests cover the guard decisions (pure) and the dispatch through a fake; a
  live check restarts a real idle slot.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: greenfield-go-module   design=0.2 impl=0.3
item: greenfield-go-module   design=0.2 impl=0.3
item: smaller-go-module      design=0.05 impl=0.2
item: atlas-docs             design=0.05 impl=0.08
item: milestone-review       design=0.0 impl=0.2
item: milestone-review       design=0.0 impl=0.2
item: milestone-review       design=0.0 impl=0.2
item: milestone-review       design=0.0 impl=0.2
design-buffer: 0.15
total: 2.26
```

M1 settle signal + hello-v2 negotiation and M2 relaunch admission/freshness/CLI
are greenfield single-concern Go; M3 mirrors M2's path; M4 docs; one review per
milestone. Design is low because the durable plan pre-resolves the decisions.

## Plan

Durable plan: `workshop/plans/000421-couch-tl-callable-restart-of-a-live-idle-slot-cli-relaunch-with-busy-dirty-guards-and-a-receipt-plan.md`.

- [x] M1 — the wrapper reports `Settled` (no open turn, empty composer,
      quiet), plus its own build revision, to the broker
- [x] M2 — `couch --relaunch repo:N --confirm`: busy/dirty/stale-binary guards
      and a receipt
- [ ] M3 — `couch --reload-context repo:N --confirm`: same guards, signals
      pair-wrap like `pair agent restart`
- [ ] M4 — live check on a real idle slot; atlas and `couch --skill` docs

## Log


- 2026-10-10: closed M2 — Round 2. BR-7: RelaunchResult.ReceiptCode keeps park-incomplete / park-ok-resume-failed as the receipt Code, note survives failure (TestRelaunchFailureOutcomeIsTyped). BR-8: TestLiveRestartProbeFactsMapping (settled/busy/legacy/no-session/not-live, git clean/dirty/unreadable, detached probe, cross-scope miss) and probe binary facts. BR-9: README lists --relaunch. BR-10: plan table superseded lines marked inline + M2 deltas revision. Minors: CLI reads IsSlotOperation; version-skew hint names the verb incl. strict-decode unknown field (test). Prior: DecideLiveRestart exhaustive 512, freshness table, TestPrepareSlotOperationRelaunchLive, CLI/protocol override tests. couchcore/couchcmd/couchmessage/couchtty green under clean env. Actual: measured window minus M1.; review verdict: SHIP
### 2026-10-09
- 2026-10-09: closed M1 — Round 2. BR-2 fixed: lifecycleTurnChanged unsettles on a silent open and re-arms on a silent close (TestSettleFollowsSilentTurnTransitions, both directions). Minors fixed: build hashed first in run (~10ms measured); only EOF/reset falls back (TestNegotiationTimeoutDoesNotFallBack). Prior: four hello-v2 pairings, TestRegistryLiveness, wrapperSettled table, interleaved settle timer. wrapcmd/couchmessage/couchcmd green under clean env. Actual is the measured window value.; review verdict: SHIP
- TL finding (2026-10-09): relaunch under Couch never rebuilds, because Couch
  does not propagate `PAIR_DEV`. Propagation is independent work, filed as
  #422. #421's share: the `stale-binary` refusal is skipped when the slot's
  launch environment has `PAIR_DEV` set, because `dev_rebuild` will rebuild
  during the relaunch. Folded into M2.
