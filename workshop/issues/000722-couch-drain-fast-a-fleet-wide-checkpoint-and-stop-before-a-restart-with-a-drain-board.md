---
id: 000722
status: open
deps: []
github_issue:
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
collaboration-mode: collaboration
card_mirror: '4b68e6466eb0313572a56fc6194f1db96acf50b6' # card fields mirrored from issue-cards; edit via sdlc
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

## Done when

- A fake fleet of 3 slots (one idle, one mid-turn, one with a running suite) reaches all-drained under graceful and
  under `--fast`, with the board showing each transition (tests on the stateful fake).
- `--cancel` restores dispatch delivery and lifts sdlc's pause.
- A slot that never reports shows as stuck, with its reason, at the timeout.

## Plan

- [ ]

## Log

### 2026-10-19
