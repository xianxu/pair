---
id: 000722
status: working
deps: []
github_issue:
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
collaboration-mode: collaboration
card_mirror: '69ac046214912d497164675c93e78dd8e22e39db' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-19T23:16:03-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
---

# couch --drain [--fast]: a fleet-wide checkpoint-and-stop before a restart, with a drain board

## Problem

There is no safe, on-demand moment to restart the machine or Couch. Work never ends, so "safe to restart" has to be a
state the fleet can reach on request. Today a restart is a crash (10-19 15:55) or the TL pinging slots one by one,
and nobody can read off whether it's safe. Operator item 15 (10-19), discussion:
ops `teams/devinfra/tl-status-xian-xu.side/2026-10-19-15-fleet-drain.md`.

## Spec

`couch --drain [--reason TEXT] [--fast]` and `couch --drain --cancel` (steward hub or operator only):
- **Graceful (default):** every live slot gets one system notice ("checkpoint and stop: finish the current atomic
  step, commit, push, write where you stopped in the issue Log, start nothing new, reply drained"). Couch stops
  delivering dispatches meanwhile.
- **`--fast`:** for work that can be restarted later. Couch interrupts running agent turns (Esc), and the notice says
  "stop now: write where you stopped, don't finish the step". Suites and merges are cancelled through sdlc's pause
  (ariadne counterpart) and recorded as interrupted, and rerun on the same head after the restart. Nothing durable is
  lost: the working tree and the transcript are on disk; only in-flight turn context goes.
- **Drain board** (`couch --drain --status`, plus the tab bar): per slot **draining / drained / stuck (why)**. Drained
  means the agent is idle at its prompt, no running suite, merge or landing is held, and the slot's last report says
  drained. A timeout (`--timeout`) ends in a list of the stuck slots, never "go anyway".
- After the restart, "resume the fleet" (ariadne#456) brings back the slots that had work.

The [durable implementation plan](../plans/000722-fleet-drain-plan.md) defines the draft transition model, state/recovery table, exact v1 notice text and shared SDLC argv/JSON/fake contract. TL-approved refinements below supersede the original operator-entry/free-text-report wording; remaining design proposals await TL approval before code.

## Done when

- A fake fleet of 3 slots (one idle, one mid-turn, one with a running suite) reaches all-drained under graceful and
  under `--fast`, with the board showing each transition (tests on the stateful fake).
- `--cancel` restores dispatch delivery and lifts sdlc's pause.
- A slot that never reports shows as stuck, with its reason, at the timeout.
- A matching per-drain ack plus fresh exact-incarnation idle and authoritative SDLC quiescence is required for drained; quiet without ack is stuck: no ack. Old incarnations/acks, incomplete queue status and late effects cannot manufacture completion.
- Steward-only control, dispatch holding/release, owner-isolated queue cancellation, fast interruption identity checks and restart/uncertain-effect recovery pass deterministic sequence tests. The real ariadne#481 queue adapter conforms in an isolated portable state root before production enablement/close.

## Plan

- [ ] Approve/checkpoint the design and pass full-flow change-code before implementation.
- [ ] Implement the pure model/board/notices and owner-scoped queue/store seams with stateful fakes and sequence tests.
- [ ] Wire steward CLI, authenticated ack, dispatch gate, fast wrapper interrupt and tab/status board.
- [ ] Verify real ariadne#481 conformance, isolated wrapper integration, races, docs and required recorded suite; stage/live board check with hub authorization.
- [ ] Close through the mandatory SDLC review, land through the ordinary gates and report the outcome.

## Log

### 2026-10-19

Claimed by pair:5 under canonical TL dispatch c1f508a1. Read the operator's fleet-drain discussion and ariadne#481's existing Spec. start-plan delivered ARCH principles and created the independent issue branch from main. Its weave refresh regenerated the two CI workflows; reverted only those generated files per the standing #699 instruction, then start-plan succeeded. No implementation code, real pause, interrupt, restart or live drain was performed.

Read-only code exploration confirmed HubOwnership.Main as steward authority, exact liveCaller/Registry session evidence, Broker.SendSystem, console event publication and existing stateful fake patterns. Park/continuation execution does not provide an agent-authored checkpoint and is not reused as the drain's stop mechanism. Drafted the durable plan with all nine ARCH lenses; no estimate is derived before plan-quality.

## Revisions

### 2026-10-19 — authenticated acknowledgement and shared queue ownership

TL b2e72204 explicitly approves `couch --drain --ack ID` instead of free-text drained reports, as the only worker protocol action. The board derives idle, incarnation and SDLC quiescence; otherwise-quiet/no-ack shows stuck immediately. Workers still checkpoint as directed by the system notice; no separate hub report is required for drain completion.

TL 205542db approves the stable ariadne#481 seam: `sdlc queue pause --reason TEXT --owner TOKEN [--cancel-running]`, `resume --owner TOKEN`, and `status --json` with paused/unknown plus active suite, merge and landing owners by workspace. Resume releases only its owner's pause. The plan specifies exact argv/schema and a stateful fake; TL will copy the approved contract into #481's Spec. #481 is not implemented by this pair issue.

TL 126cc5c8 narrows the original “steward hub or operator” entry to steward CLI only: the operator asks the TL to drain. Read-only status is available to anyone. Bare outside-Couch shell mutation and a console action are out of scope; a console action can be filed separately if requested. Steward remains the first hub on the command line, reusing existing HubOwnership.Main.

The remaining draft proposes asynchronous start (the steward must also checkpoint and become idle), a supervisor-owned deadline with no automatic unpause, owner-intent persistence, fail-closed unknown effects and bounded fast interruption. These details and the user-visible state table are submitted for TL review before any code.

### 2026-10-19 — one advisory review clarifies control ordering

A fresh-context read found that the existing peer FIFO could hold the drain notice itself, that a delayed Esc could hit the checkpoint turn, and that cancellation must revoke admitted notices rather than only coordinator goroutines. Accepted all three; the durable plan's governing Revisions specify one typed generation control lane, original-turn fencing before notice admission, and wrapper-lease revocation/reconciliation before cancellation releases dispatch. Added named adversarial tests for each. This was one advisory read, not an approval loop or a replacement for change-code's plan-quality review. No scope approval, implementation or live mutation is inferred from it.

### 2026-10-19 — TL approval to enter change-code

Canonical submitted receipt 0a9b7377 approves the complete plan, four visible states (draining/stuck/drained/cancelled) and proposed async/deadline/persistence choices. TL copied the notice and SDLC seam verbatim to ariadne#481. Run full-flow change-code and its plan-quality gate before implementation; real #481 conformance still gates enablement/close. No bare operator-shell or console mutation entry is added.

### 2026-10-19 — PQ-1 test strategy correction

Full-flow change-code round1 refused only Important PQ-1 (function-level-test-strategy). Compressed task test inventories into production-function/adversarial-strategy/mechanical-guard descriptions across the plan. The approved design/seam is unchanged; implementation still waits for change-code admission.
