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
flow: {kind: full, provenance: operator}
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
- Live: after the next full-shutdown/unknown start, the TL sees one report line and no generated hub input until automation is explicitly enabled. The separate tl-lead rule owns the first-operator-message decision; Couch never parses that message.
- Cross-supervisor tests keep one worker alive through a private Couch restart
  and prove no duplicate generic cue; replacing/re-execing the worker creates one
  new opportunity. Unknown/uncertain delivery never authorizes duplicate input.
- Negotiated old/new wrapper compatibility preserves ordinary messaging without
  treating an unsupported cue report as "not yet delivered".

## Plan

- [x] Qualify safe startup classification and attachment-only selection for all hubs.
- [x] Keep startup lifecycle summaries out of hub composers and preserve a usable switcher.
- [x] Add wrapper-owned one-shot worker orientation through negotiated session reporting.
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

### 2026-10-21 — implementation admission and checkpoint

Plan-quality first returned PQ-1 (replace prose scenario enumeration with named
risky-function/adversarial-strategy guards). The revised plan passed CLEAN;
change-code then passed on 6695cb081 after deriving the v3.1 estimate. All unit,
protocol and fake-backed tests are included in the estimate primitives: the
wrapper/session integration row includes negotiation and receiver tests;
supervision nudges belong to the episode/reminder family, and the common send
source guard belongs to the inventory/docs row. The two UX iteration allowances
cover completed operator shaping, not two future implementation rounds.

The hub relayed #737's agreed seam: messageService owns
hubAutomationOn(hubRoot string) bool; scheduler reserveDue, reserved-script
authorization and scheduleNotice.Admit consult it. Whichever lands second wires
the consumer on main, without stacking. The earlier refused query is superseded
by this explicit agreement.

Implementation in progress. Pure hub automation and CLI parser reds were
observed, then focused tests passed. The controller rejects workers, stale or
disconnected callers and foreign hubs; explicit concurrent switcher toggles are
serialized. Unknown startup remains off. Core attachment-only startup, negotiated
wrapper cue receipt and switcher Ctrl+G have separate targeted evidence from
bounded worker tasks. No full suite, close, stage or live acceptance is claimed.

### 2026-10-21 — verified component behavior

Targeted core startup/automation/lifecycle/resource tests passed (5.346 s).
Combined couchcmd automation/startup/producer/supervision/wait/stale tests passed
(17.257 s), and the CLI/controller/source-owner group passed with race checks.
The producer group passed under race, including delayed admission and status
output, off/on during unknown-identity resolution, and resource pending retries.
Suppression is an explicit terminal disposition, never a delivered claim.
Every generated SendSystem now enters through sendAutomaticSystem, pinned by an
AST source-reader guard. Human/worker sends use their unchanged normal path.

Protocol/session/registry/background/peer targeted tests passed in both packages
(5.770/5.889 s); new private-socket cue tests passed race count 3. They count
native input across supervisor replacement before and after confirmation,
receipt eviction, lost acknowledgment, batch followers and same-binding new
launch ordinals. Bounded decoder fuzz passed with two workers for 3 s; it is
bounded qualification, not a failure-rate or performance claim. Counted 20/40
worker tests prove one child proof initially per needed launch, zero extra
ready/process/git work on unchanged ticks, and one additional proof for one
changed ordinal. Unknown role, launch proof and receipt remain conservative.

Changed old expectations intentionally: implicit startup no longer creates or
cold-resumes a missing, parked or orphan-adjacent hub. Explicit lifecycle tests
remain. Lifecycle collector tests now use explicit operator/peer events during
a held startup pass; new tests prove automatic baseline observations never
flush. Background negotiation tests formerly hardcoded hello-v4 as newest and
now name hello-v5, while old-peer strict downgrade behavior remains tested.
The fast reader inventory is audited rather than waived; modified preflight
names follow renamed semantics and the sole-ingress source guard.

### 2026-10-21 — live acceptance dependency and close ordering

Code head 1580bf163bd5869428d5831e3d5d9cef2b18f3ec is published as PR #510.
Recorded affected selected the full suite because the clean runtime bundle graph
is unavailable (#738); attempt 4e8fab1bec9d0e9cb2e5fad70d697437 remains in the
normal machine queue. No duplicate suite, staging or live fleet action occurred.

Hub requires three private real-supervisor scenarios before landing: full
shutdown gives automation off, one status/no hub injection and empty-hub start
guidance, followed by explicit on and a fresh notice; normal restart preserves
hub automation on and reattaches workers without a new generic cue; an actually
started worker receives exactly one cue. pair:5 confirms #744's shared harness
is still unlanded. Hub c23219fb-e84d-4e8a-ae60-c54976d1f4f3 directs waiting for
#744, no duplicated fixture, finishing this suite and close review meanwhile,
and reporting if #744 slips beyond the next morning. Landing stays blocked on
that live evidence; the real fleet first use is the next restart after landing.

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

### 2026-10-21 — concrete #737 provider/consumer ordering

Hub message 752adf46-b8e6-4e99-b76a-dbcdf8e69331 confirms the agreed scheduler
getter and second-lander wiring. #739 may publish its provider first; it does
not claim unlanded scheduler coverage. #737 must wire reserveDue, script
authorization and scheduleNotice before its own landing if it is second. No
stacked development. The implementation's per-hub epoch is a memory-only fence
for delayed notices across off/on, within the approved occurrence-owned policy.

### 2026-10-21 — empty configured hub projection

Final integration proved a never-started configured hub has no ThreadRecord and
therefore no ordinary switcher actor row. Hub approval f7d34e39-4724-43c5-bba1-
32359ac86584 permits a display-only parked/missing hub entry with one explicit
recovery action, without a new selectable hub-row model. The existing Ctrl+Space
Start form accepts the exact hub path and previews before starting. The entry
names that action; no fake conversation address or primary-as-numbered identity
is introduced. Bare-cwd startup separately binds the same verified HubOwnership
to routing and Console, so Ctrl+G works there too; failed resolution stays
unknown with a usable switcher and an explicit diagnostic.

### 2026-10-21 — live wording aligned to final operator policy

The original Done-when phrase "until the operator speaks" is superseded by the
approved explicit automation policy. Full-shutdown/unknown starts stay off until
an explicit control; normal surviving hubs start on. The separate tl-lead rule
chooses the first-message action, and Couch never flips on a submission. Live
acceptance remains owed and must be coordinated with the hub; no stage or
restart is claimed here.

### 2026-10-21 — reviewed code before shared-harness live acceptance

Per hub c23219fb-e84d-4e8a-ae60-c54976d1f4f3, finish recorded checks and the close
review before #744 lands; then integrate its main code and implement the three
approved private-fleet scenarios. This is not permission to land without live
acceptance, nor to use a second fixture. The remaining live Plan checkbox is
intentionally pending at that review boundary; its precise close-plan exception
must say so, rather than falsely ticking acceptance. Any post-review integration
or test code follows the normal narrow re-close and verification gates.
