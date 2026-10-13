---
id: 000517
status: working
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours: 8.64
card_mirror: 'a1a4cddf35d8ffa69afaf93f7cfe79705e058041' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T20:05:23-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
---

# couch: an agent switches a slot's coding agent (couch --switch-agent repo:N --agent NAME), with a handoff built from durable state

## Problem

The operator (2026-10-12, Claude quota at 80%) wants an agent-guided agent switcher the TL can use, to move slots to Codex as they free up. Today only the operator can switch a slot's coding agent (the thread's actions, **switch coding agent**), and its handoff feels robotic: Couch launches a fresh conversation and asks the new agent to read the outgoing context and summarize its orientation. Related: pair#414 (a Claude → Grok switch lost its orientation and the attach was refused).

## Spec

Agent-guided: the agents who know the work write the handoff; Couch does the mechanics.
1. **A verb for agents:** `couch --switch-agent repo:N --agent NAME [--handoff-file PATH] --confirm [--json]`, the same operation as the switcher's form, with its guards (refuses a busy or unknown slot). Cost-tagged `[heavy]` in help. The TL can call it for any slot; a slot can call it on itself (`--switch-agent self`).
2. **The handoff is written by an agent, not generated:** before the switch, the outgoing agent (or the TL, for a slot it supervises) writes a short brief the way a colleague would: what the issue is, where the work stands (branch, last commit, what's verified), what's next, open questions, and anything not in the issue file. Couch delivers it verbatim as the new agent's first message. With no handoff (an idle slot on its resting branch), the new agent starts fresh with no orientation prompt.
3. **Durable state stays the source:** the brief points at the issue's Spec, Plan, Log and Revisions rather than restating them; the outgoing agent commits its work first.
4. The switcher's form gains the same path: it asks the outgoing agent for the brief, then switches.
5. Fold in or reference pair#414's attach/orientation failure, so delivery is reliable across agents (Claude, Codex, Grok).

### Agreed design direction

Use the existing broker, park/start, transcript resolver, retention lease and
orientation delivery. Persist one exact previous-session descriptor before the
target starts; it survives successful launch and is selected by `--peek
repo:N --previous`. Full recording reads run in a short-lived CLI. A short
agent-authored brief passes unchanged; a resting idle slot starts with no prompt.
The form asks the outgoing agent to commit and call `--switch-agent self`;
Couch acknowledges before waiting for the same authenticated incarnation to
settle. The brief timeout selects pointers-only content; a longer busy timeout
refuses without interrupting work. The detailed plan proposes limits for review.

## Done when

- The TL switches an idle slot to Codex with one couch call, and a mid-issue slot to Codex with a brief written by its outgoing agent; the new agent resumes the issue from the brief and the issue file.
- The operator approves the handoff's feel on a live switch.
- Tests cover the guards, delivery of a brief verbatim, and the no-handoff fresh start.
- The incoming agent can identify and read the exact previous native transcript
  and preserved tty recording, including an explicit full plain-text read;
  missing evidence is reported rather than silently substituted.
- The form completes without a second click. Authenticated current-incarnation
  callbacks, pointers-only timeout fallback, and a longer busy refusal are tested.
- The TL reviews the switch procedure in ops/skills-draft/xx-tl/SKILL.md and
  coordinates first use on paused ariadne #230/#282/#305 with committed briefs.

## Plan

Durable plan: [agent switch plan](../plans/000517-agent-switch-plan.md).

- [x] M1 — Preserve exact previous-session evidence and expose previous/full tty reads.
- [ ] M2 — Add authenticated asynchronous switch requests, verbatim briefs and fresh starts.
- [ ] M3 — Wire the form, TL procedure, live acceptance and rollout.

## Log

- 2026-10-12 21:42 PDT — continuation writer committed and pushed 9bfe12878
  (workshop/continuation/20261012T214122-agent-switch-517.md), then restart
  refused because prior continuation
  35eff9de95a978dd139c6039c1e49eedacb174877a9d9d877d3f4baab890e0a0 remains failed
  for 1-pair-8. No separate restart or unsafe retry issued. Reported exact
  retained-state blocker to hub, receipt 6c086b9d-1a12-4053-8510-9bbb43d59daf.
  This is additional observed recovery friction, not proof of the underlying
  orientation failure cause. All owner agents stopped safely; M2 remains open.

- 2026-10-12 21:42 PDT — context checkpoint: all M2 owners stopped after atomic
  edits. Recovery schema/projection, exact-native resolver seam and menu
  projection are written; core recovery is an unwired draft. Root targeted
  threadrecord/couchcore/couchtty command matching TestSwitchRecovery,
  TestAgentSwitch, TestMenuAgentSwitch and TestNativeBindingResolverResolveExact
  passed; diff check passed. These selections do not establish transaction,
  restart or receipt integration. Continuation records exact ownership, draft
  defects and next actions; checkpoint is explicitly unverified M2 WIP.

- 2026-10-12 21:39 PDT — revised change-code gate passed: plan-quality round 4
  CLEAN, prior findings remain addressed. Gate commit d1604e0dc. Its estimate
  advisory is addressed below by preserving the original derivation and adding
  approved recovery primitives, current forecast 8.640 hours. M2 implementation
  is authorized; no M2 boundary or full integration success claimed. Recovery
  audit also found latest-ledger native selection would reject an older saved
  session after a same-agent switch; exact named validation must continue through
  the launcher. Context checkpoint underway; in-flight owner edits remain WIP.

- 2026-10-12 21:34 PDT — verified TL approval 60b20071-4145-4fd1-a177-11240322a715:
  durable one-action recovery is the selected invariant. Recovery failure remains
  visible/actionable; unknown presence offers guarded Resume old thread. Record
  before park, restart primary action+notice, tests for every failure row. Audited
  f756c2bf9 read-only with core and UI owners before reuse; documented reusable
  seams and persistence/presentation gaps. Extended executable M2 plan; run its
  plan-quality gate before new implementation. No source-retaining launch added.

- 2026-10-12 21:29 PDT — TL paused M2 for an operator-visible recovery design,
  then clarified that internal states need a small projection rather than forced
  reduction. All implementation agents stopped. M1 remains SHIP at 37707f73b;
  the subsequent full `make test` completed successfully. Unreviewed M2 WIP
  checkpointed at f756c2bf9. Targeted model/coordinator/transport/form tests and
  race checks passed; authored-byte layout/wrapper and fresh-no-prompt readiness
  acceptance passed. Full M2 integration/review has NOT completed. The plan's
  newest Revisions entry maps 8 internal states to 5 visible outcomes and exposes
  park/start failures that can leave neither agent running. Awaiting TL's choice
  between durable one-action recovery and a larger source-retaining replacement
  transaction; no further M2 implementation until approval.

- 2026-10-12 BR-4/BR-5 correction: replay sets explicit row/cell/byte/area
  bounds and uses checked construction/resize. VT's monotonic primary-history
  capacity-loss counter catches eviction and rejected rows even before a later
  erase; bounded replay stops after detecting loss. A 2,000×80-line regression
  is retained in full. Independent cell/byte loss, initial/later geometry,
  malformed event evidence and intentional erase tests pass. Six mutations
  detected missing enforcement and were restored. Direct reply discard removes
  the obsolete drain goroutine. Targeted replay/core/inventory checks and the
  nested VT suite pass; fresh full make test is running.

- 2026-10-12 M1 round 2 disposed BR-1/BR-2/BR-3 and returned REWORK on
  BR-4/BR-5: emulator history cell/byte limits can silently evict lines, and
  accepted geometry can exceed its screen-area limit. The plan now records
  explicit cell/byte/area limits and a monotonic VT capacity-loss counter
  (ARCH-CONSTRAINTS/SECURE) to make complete-or-refuse enforceable.
  Full make test finished with only artifact inventory and lifecycle-refusal
  table failures. Both missing entries are corrected; targeted artifactpath,
  scrollback and lifecycle-refusal tests pass. Replay corrections in progress.

- 2026-10-12 M1 boundary round 1 returned REWORK (BR-1/BR-2/BR-3).
  Reproduced stale previous evidence after A→B, separate park B, B→C;
  reuse now compares the current verified park identity (ARCH-ORDER).
  Updated the plan's actual file owners with a Revisions entry. Added real
  retention collection interleavings through RenderOwnedCaptureLines, covering
  success and replay failure; disabling acquisition or release fails both cases.
  Targeted previous/switch/peek/operation checks and scrollback package pass;
  the scrollback race suite passed. Full make test exposed missing artifact
  inventory classifications; corrected them and the inventory check passes.
  Full suite is still running; M1 re-review follows its outcome.

- 2026-10-12 recovery evidence: after continuation commit d32489907, this
  session received no automatic orientation. TL ops:0 supplied the continuation
  path and next action in verified receipt b75ae41d-6538-4630-ad7c-ad8f751b48d5.
  Recovery required that external pointer despite durable M1 code at 5392b48c0.
  This demonstrates the missing pickup path #517 must cover; it does not yet
  establish the transport cause or prove a fix.

- 2026-10-12 M1 implementation checkpoint: persisted exact previous-session
  descriptor before target launch, retaining it through registration and failed
  retries; added previous/full peek selection and bounded CLI-only replay under
  capture leases. Red tests exposed foreign/missing capture and binding acceptance;
  store now compares each against verified source state. No current fallback.
  Targeted threadrecord/couchcore/couchcmd tests for PreviousSession, SwitchAgent,
  Peek, HelpAndSkill, and Operation pass; scrollback package tests pass separately.
  Limits are 64 MiB raw, 8 MiB events, 100k rows, dimensions 1–1000; README and
  atlas explain retained-history semantics and costs. M1 boundary review pending.

### 2026-10-12
- 2026-10-12: closed M1 — BR-4/BR-5 regressions pass: 2000x80 lines retained; independent history cell/byte/row loss and loss before ED3 refuse; initial/later excessive geometry refuses; strict bounded events refuse malformed evidence. Six enforcement mutations fail and are restored. Scrollback and nested VT full race suites pass; previous-session, lifecycle refusal and artifact inventory targeted tests pass; diff clean. Fresh full make test underway.; review verdict: SHIP

- Claimed in pair:2; start-plan refreshed the local ariadne tools, then started
  the issue branch from main. No implementation changes yet.
- Read #414: its details contain only the title, no reproduction. Existing
  switch tests and live registration/delivery are the regression surfaces.
- Metadata audit: 436 parked raw files and 436 events files, 286 capture metadata
  files, 70 live raw files; latest parked raw approximately 9.4 MB. No transcript
  content read and no live session changed. Large peek requests already fall
  back to current recording replay; the missing capability is previous selection.
- ARCH-DRY/ORDER: reuse current native resolution before park, exact preserved
  capture after park, and existing delivery state; add one source-bound switch
  coordinator rather than duplicate CLI/form lifecycles. Full replay remains
  outside the supervisor because the existing emulator keeps a drain goroutine.
- Durable plan records three actual review boundaries, test commands, proposed
  operating limits and live acceptance. Waiting on TL's plan review, then
  change-code's plan-quality gate; estimate follows plan-quality.

- Plan-quality round 1 refused with PQ-1 (request correlation) and PQ-2
  (function-level testing strategy). Verified both: receipt keys are caller
  scoped and cannot correlate an outgoing callback alone. Plan now specifies
  `--request ID` plus exact source identity and names risk-focused test seams.

## Revisions

- 2026-10-12 21:34 PDT — TL approved the one-action recovery interpretation with
  idempotence, visible failed-recovery fallback, restart presentation and per-row
  tests (receipt 60b20071-4145-4fd1-a177-11240322a715). Appended concrete owners,
  admission-before-ack durability and exact source-native preservation to the
  plan. Earlier pause is lifted subject to the revised plan-quality gate.

- 2026-10-12 21:29 PDT — operator recovery review via ops:0, clarified in receipt
  `5a859539-2444-44a0-b156-6402cfb3227a`: preserve necessary internal states,
  require their projection onto a small set of human outcomes with one recovery
  action each. Added the projection and full failure rows to the durable plan.
  Proposal removes the unmeasured global switch cap. Honest current failure
  evidence requires a scope decision before M2 continues; no approval inferred.

- 2026-10-12 — the operator: the switch is driven by the TL's skill. Done when also: a "switch a slot's agent" procedure in
  ops' `skills-draft/xx-tl/SKILL.md` (when to switch, how to write the brief, the exact couch call, how to check the new
  agent picked it up). The TL reviews that section; the first real use moves the paused ariadne items (#230, #282, #305)
  to Codex with briefs.

- 2026-10-12 — operator scope revision relayed by ops:0 (receipt
  `4f8b1134-cc92-4af2-b4e0-a1deecf71f1a`): the incoming agent must be able to
  find and read the previous session itself. The brief names the previous
  agent-native transcript and per-incarnation tty log, stays short, and points
  at durable issue state. Measure the existing recording and discovery paths
  before designing. Add previous-incarnation peeking and access to the whole
  tty log as readable plain text (all or paged), with cost tags. The current
  live-buffer peek is insufficient even with a large requested line count.

- 2026-10-12 — TL confirms the operator flow (receipt
  `4a85bea3-11b4-435c-b08d-663b6c18c851`): the form requests the outgoing
  agent's brief; that agent commits and calls `couch --switch-agent self`
  with its handoff file. No second operator click. Couch acknowledges before
  waiting for that same incarnation's turn to end. A self request must prove
  the current caller incarnation. A configurable brief deadline (a few
  minutes) selects a pointers-only fallback and reports it on the notice line
  if the agent does not call back. The TL uses the same verb with `repo:N`.
  Deadline behavior while the source remains busy is being clarified.

- 2026-10-12 — TL confirms deadline policy and audit architecture (receipt
  `0106a718-0d2b-46d2-9261-c035f81d798f`): brief timeout selects fallback
  content only; it never interrupts a busy source. A second, longer deadline
  refuses and reports the reason on the notice line. Exact previous descriptor
  plus previous/full peek is approved as the design direction. Added these
  acceptance criteria and the durable three-milestone plan for review.

- 2026-10-12 — implementation plan approved by TL, receipt
  `ebf27554-f0e7-4d4e-8b00-2b46b83c529f`; defaults and full-read limits
  accepted. TL owns live-check scheduling. Proceed to change-code plan-quality;
  no permission to skip any gates. Add dirty-source commit/self guidance to M3.

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md`
against `baseline-v3.1.md`. Method A only.* Source marked stale by
`sdlc estimate-source`; provisional calibration. Derived after plan-quality
round 2 passed. Scope includes design, tests, docs, four reviews (three
milestones plus close), and live discovery for Claude/Codex/Grok.

Decomposition: issue/spec at 0.5 design + 0.2 baseline implementation;
four smaller Go extensions (descriptor/store, peek selection, bounded replay,
wire/CLI) each 0.2 design discounted by 0.2 and 0.5 baseline implementation;
two new Go concerns (pure switch model, coordinator) each 1.0 design discounted
by 0.2 and 0.8 baseline implementation; form at 1.0 design discounted by 0.2
and 1.0 baseline implementation; TL skill at 0.6 design discounted by 0.2 and
0.5 baseline implementation; three live-agent discovery seams each 0.6 baseline
implementation; four review boundaries each 0.1 design discounted by 0.2 and
0.5 baseline implementation; docs at 0.1 design discounted by 0.2 and 0.2
baseline implementation. All implementation values below apply v3.1's 0.4
scale once. Existing Go lifecycle, broker, renderer and TUI seams avoid a new
stack; no external library replaces the source-bound coordination model.
Familiarity 1.0 for this established stack; thorough-plan design buffer 15%.
Design subtotal 1.48; implementation subtotal 3.72; total 5.422 hours.

```estimate-original
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.5 impl=0.08
item: smaller-go-module design=0.04 impl=0.20
item: smaller-go-module design=0.04 impl=0.20
item: smaller-go-module design=0.04 impl=0.20
item: smaller-go-module design=0.04 impl=0.20
item: greenfield-go-module design=0.20 impl=0.32
item: greenfield-go-module design=0.20 impl=0.32
item: tui-screen design=0.20 impl=0.40
item: skill-or-dispatcher design=0.12 impl=0.20
item: real-api-discovery design=0 impl=0.24
item: real-api-discovery design=0 impl=0.24
item: real-api-discovery design=0 impl=0.24
item: milestone-review design=0.02 impl=0.20
item: milestone-review design=0.02 impl=0.20
item: milestone-review design=0.02 impl=0.20
item: milestone-review design=0.02 impl=0.20
item: atlas-docs design=0.02 impl=0.08
design-buffer: 0.15
total: 5.422
```

### 2026-10-12 revised forecast after durable recovery approval

The original 5.422-hour forecast above remains provenance, not the current
scope. Plan-quality round 4 was CLEAN; estimate-quality correctly identified
that the approved recovery delta adds work not represented by that forecast.
Method A and the same provisional v3.1 calibration remain in use. Add three Go
concerns (durable recovery data/projection, proof-based recovery orchestration,
and durable admission/restart receipt integration), each 0.20 design + 0.32
scaled implementation. Add two smaller seams (exact native-session restoration
and atomic launch witnesses), each 0.04 + 0.20. Add the recovery UI projection
and primary-action integration at 0.20 + 0.40, distinct from the original switch
form. Allow two additional review/fix iterations at 0.02 + 0.20 each for the
expanded failure matrix. Existing store, native resolver, resume, and menu
owners are reused; no new external stack is introduced. These are scope
primitives, not measured hours or a backfit to elapsed time.

Delta design 0.92, implementation 2.16; revised design 2.40 and implementation
5.88. Familiarity stays 1.0 and design buffer 15%: 2.40 × 1.15 + 5.88 = 8.640.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.5 impl=0.08
item: smaller-go-module design=0.04 impl=0.20
item: smaller-go-module design=0.04 impl=0.20
item: smaller-go-module design=0.04 impl=0.20
item: smaller-go-module design=0.04 impl=0.20
item: greenfield-go-module design=0.20 impl=0.32
item: greenfield-go-module design=0.20 impl=0.32
item: tui-screen design=0.20 impl=0.40
item: skill-or-dispatcher design=0.12 impl=0.20
item: real-api-discovery design=0 impl=0.24
item: real-api-discovery design=0 impl=0.24
item: real-api-discovery design=0 impl=0.24
item: milestone-review design=0.02 impl=0.20
item: milestone-review design=0.02 impl=0.20
item: milestone-review design=0.02 impl=0.20
item: milestone-review design=0.02 impl=0.20
item: atlas-docs design=0.02 impl=0.08
item: greenfield-go-module design=0.20 impl=0.32
item: greenfield-go-module design=0.20 impl=0.32
item: greenfield-go-module design=0.20 impl=0.32
item: smaller-go-module design=0.04 impl=0.20
item: smaller-go-module design=0.04 impl=0.20
item: tui-screen design=0.20 impl=0.40
item: milestone-review design=0.02 impl=0.20
item: milestone-review design=0.02 impl=0.20
design-buffer: 0.15
total: 8.640
```
