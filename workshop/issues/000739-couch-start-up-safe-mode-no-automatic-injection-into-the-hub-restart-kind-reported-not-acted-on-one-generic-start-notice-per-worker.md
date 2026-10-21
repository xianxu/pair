---
id: 000739
status: working
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: '120ea18e2370654d8ccb7181f0493cfc9483432e' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-20T22:40:29-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
---

# couch: start-up safe mode: no automatic injection into the hub, restart kind reported not acted on, one generic start notice per worker

## Problem

After a restart, Couch acts on its own: it injects system notices into every slot, the hub included ("Couch
restarted and the hub is available…"), and recovery has been attempted automatically at times. The operator (10-20,
drain redesign, ops side doc `2026-10-20-17-drain-redesign.md`): no automatic recovery; a restart may be for
something else, or the setup may be wedged; the operator tells the TL "start recovery" in one line.

## Spec

- **Hub safe mode:** on start, the hub's TL gets no automatic system message injection and nothing is auto-started
  inside the hub.
- **Restart kind, reported only:** Couch tells a normal Couch restart (an attachable Zellij session for the fleet
  exists) from a full shutdown (none), and reports it to the TL once, in one line ("full shutdown detected: N slots
  parked; recovery waits for the operator"), without acting on it.
- **Worker start notice:** when a worker slot starts (resume or fresh), Couch sends one short generic notice: "you
  restarted; reread your issue Log and continue where you left off; report to your hub". The TL may follow with its
  own message; Couch never sends more than the one generic notice per start.

### Confirmed interpretation for operator review

Safe startup attaches surviving hubs/workers only; absent hubs or slots stay
listed, with no cold resume or Fresh, including an empty installation. Publish
the restart-kind report as one retained Console status line, never a composer
message. Unknown or orphaned evidence is not a full-shutdown proof. All explicit
operator-authorized lifecycle actions keep their existing guards.

Use a **per-hub memory-only startup gate**: until a genuine operator submission
to that hub, discard all hub-directed automatic system messages. Ordinary peer
reports stay deliverable. Opening the gate never replays suppressed notices; the
TL can inspect current state with peek/status. Initial counter reconciliation,
worker/system input and reconnect cannot open it. The one restart-kind line is
retained Console status, not injected agent input. The plan enumerates lifecycle,
wait, session-file, resource, stall and report-nudge producers and their tests.

The generic worker cue is once per **wrapper launch incarnation**, including
Fresh, resume and relaunch; same-incarnation reconnect across a Couch restart
does not repeat it. Its memory lives in the wrapper, not a file or a supervisor
attempt set. Keep the **separate existing Couch-restart resend prompt** once per
worker address per Couch lifetime: that asks for still-relevant unanswered
reports, rather than claiming the worker's agent restarted. The operator reviews
this distinction and the per-Couch-lifetime alternative in the durable plan.

## Done when

- Tests pin: no injection into the hub on start; restart-kind classification from fixture Zellij state; exactly one
  generic notice per worker start.
- Live: after the next full restart, the TL sees one report line and nothing else until the operator speaks.
- Cross-supervisor tests keep one worker alive through a private Couch restart
  and prove no duplicate generic cue; replacing/re-execing the worker creates one
  new opportunity. Unknown/uncertain delivery never authorizes duplicate input.
- Negotiated old/new wrapper compatibility preserves ordinary messaging without
  treating an unsupported cue report as "not yet delivered".

## Plan

- [ ] Qualify safe startup classification and attachment-only selection for all hubs.
- [ ] Keep startup lifecycle summaries out of hub composers and preserve a usable switcher.
- [ ] Add wrapper-owned one-shot worker orientation through negotiated session reporting.
- [ ] Verify counts, compatibility and coordinated live behavior; close and land once.

Durable plan: [startup safe mode](../plans/000739-startup-safe-mode-plan.md).

## Log

### 2026-10-20

Read the fleet-drain project's design.md before design and claimed/start-planned
from fresh main. Source-only exploration confirmed implicit cold/Fresh hub
fallbacks in StartInteractive and hub injection via lifecycle startup summaries.
The existing resend coordinator already skips verified hubs; it is not the
producer of those hub summaries. A bounded read-only subagent traced message
producers and existing tests; no live process, production code or test changed.

Hub confirmed attachment-only startup and per-wrapper start-cue ownership, then
clarified that the separate Couch-restart resend prompt stays. A wrapper-held
constant-size delivery witness and negotiated reporting preserve this across
supervisor replacement. Existing launch ordinal distinguishes a native restart
through wrapper re-exec even if PID/birth survives. Operator review is pending;
no change-code or estimate has been run. The hub owns project edits and rollout.

The producer sweep found that filtering lifecycle startup alone still permits
inherited wait episodes, a hub's own session-file reminder, resource episodes
and stall reports to wake the TL. Hub chose the per-hub quiet gate until genuine
operator submission; all suppressed occurrences are retired, not queued for
release. The narrower baseline-only option stays in the operator-review table.
Archive warnings are passive status/attention/desktop paths, not broker agent
input. The unlanded #737 scheduler must share the gate for any future hub input;
a peer seam query was refused without a receipt, so no agreement is claimed.

## Revisions

### 2026-10-20 — operator-review startup boundary and two notice lifetimes

Per ops:0's dispatch and subsequent clarification, make startup attachment-only
for all hubs and workers, retain the report in Console without waking the TL,
and keep unknown evidence explicit. The agent-start cue is wrapper-owned and
once per launch incarnation; a Couch-only restart must not call a surviving
agent restarted. Preserve the older per-Couch-lifetime resend prompt as an
independent worker-only mechanism. The plan presents the simpler per-lifetime
alternative for the operator to accept or overrule before code. Existing
explicit recovery, ordinary later lifecycle notices and warm worker attachment
remain, with no new durable recovery state or automatic action.

### 2026-10-20 — gate every automatic hub-input producer, discard without replay

Source audit found automatic producers outside startup lifecycle summaries.
Per ops:0's explicit decision, all automatic system input to each hub stays
suppressed until genuine operator submission to that hub, and suppressed
notices never replay. The worker resend prompt remains distinct and deliverable
to workers while the hub is quiet. Add producer-by-producer tests, current-hub
submission negatives, protocol negotiation and cross-supervisor worker-cue
qualification to the operator-review plan. This is still design-only work.
