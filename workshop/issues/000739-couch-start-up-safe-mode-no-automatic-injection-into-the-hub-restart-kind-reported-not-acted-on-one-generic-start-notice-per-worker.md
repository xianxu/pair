---
id: 000739
status: working
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours: 4.695
collaboration-mode: collaboration
card_mirror: '9ef7a944c1d4a0d67b0bfa18f80f9b80b832fb84' # card fields mirrored from issue-cards; edit via sdlc
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

**Hub automation** controls only Couch-generated hub input and scheduled runs. CLI: `couch --automation off|on|status [hub]`, plus switcher control. It starts on for a positively surviving hub after normal restart; off after full shutdown/unknown startup, and off for missing hubs in a mixed fleet. Only explicit on/off changes it afterward. Human submissions never flip it: the TL can debug with the operator while off. The separate tl-lead rule will run automation on on the first operator message unless told keep off; Couch performs no message parsing.

CLI mutation is own-hub-only; optional hub must name the caller's canonical hub. Only the operator switcher may control another configured hub. While off, suppress automatic hub input and skip scheduled due runs (#737) like missed runs; on resumes at the next due time without replay/catch-up. Already-running scripts may finish (soft stop). Worker/peer reports, operator input, status and desktop output are never gated. The retained line says "hub automation off; say turn it on" and names missing hubs. Focus the launch hub if it survived, otherwise the switcher; extra-hub adoption never steals focus.

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

## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against
baseline-v3.1.md, Method A only; estimate-source flags calibration stale.
Implementation values are v2 primitive hours scaled once by0.4; familiarity1.
The approved thorough design discounts design by0.2 and uses15% design buffer.

Breakdown: issue/spec (v2 design0.5, impl0.3); startup observation extension
(0.3/0.5); one new automation policy owner (1.0/0.8, design halved by reuse of
existing reducer/identity/controller infrastructure before spec discount);
wrapper/session integration (1.0/1.5); three small producer-family extensions
for lifecycle, episode/reminder and resource/stall suppression (each0.3/0.5);
Console/CLI automation controls and focus qualification (TUI1.0/1.0);
scheduler predicate integration (0.3/0.5); docs/inventory (0.2/0.2); one close
review/active landing handling (0.2/0.5); bounded real Zellij/reconnect
qualification (impl0.6, the real-API discovery allowance); two operator-driven
UX/ownership iteration groups (each0.8/0.3). Existing broker, strict codecs,
version fallback and stateful fakes avoid a greenfield protocol/service or
external library. No machine-wait time or hub-owned restart work is counted.
Design1.30 ×1.15 + implementation3.20 =4.695h. #737 wait is not effort; its
injected predicate is an existing-stack integration, not a second scheduler.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.10 impl=0.12
item: smaller-go-module design=0.06 impl=0.20
item: greenfield-go-module design=0.10 impl=0.32
item: api-integration design=0.20 impl=0.60
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: tui-screen design=0.20 impl=0.40
item: smaller-go-module design=0.06 impl=0.20
item: atlas-docs design=0.04 impl=0.08
item: milestone-review design=0.04 impl=0.20
item: real-api-discovery design=0.00 impl=0.24
item: ux-rename-iteration design=0.16 impl=0.12
item: ux-rename-iteration design=0.16 impl=0.12
design-buffer: 0.15
total: 4.695
```

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


### 2026-10-20 23:42 PDT — operator approval: explicit start gate and partial hubs

Operator approval relayed by ops:0 requires the name "start gate", an explicit `couch --start-gate open`/switcher act, read-only status, and no automatic release on operator submission. The TL may debug with the operator while closed. Keep scheduled-task exit-10 wakes behind it with no replay. Reattach surviving hubs with independent closed gates, name missing hubs in one status line, and focus the launch hub only if it survived; otherwise leave the switcher open. The active Spec and durable plan now use these rules. Earlier submission-release text above is superseded historical design. Proceed through change-code; no implementation yet.


### 2026-10-20 23:44–23:48 PDT — final operator policy and plan-quality disposition

Operator renamed the control to hub automation, with explicit off/on/status [hub] and switcher control. Initial on only for positively surviving hubs on normal restart; full shutdown, unknown state and missing mixed-fleet hubs remain off. Off is a soft stop for scheduled runs as well as notices, with missed runs skipped and no replay/catch-up on on; running jobs finish. Human/worker input and status/desktop output stay independent. CLI is own-hub-only, operator switcher may select another hub. tl-lead's first-operator-message default is a separate ariadne rule; no Couch parsing or implicit flip.

First plan-quality round returned PQ-1 Important for prose case inventories instead of named risky functions/adversarial strategies. Active plan compressed into function and producer strategy tables; all latest operator decisions are in the active Spec. No code admitted, no estimate derived. Hub relays #737's predicate seam once pair:1 is back; integration waits for that issue on main.
