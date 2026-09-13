---
id: 000184
status: codecomplete
deps: [pair#182]
github_issue:
created: 2026-09-04
updated: 2026-09-13
estimate_hours: 6.65
started: 2026-09-13T12:04:32-07:00
actual_hours: 6.03
---

# couch: switch a thread's agent

## Problem

A thread's agent is pinned by its saved launch profile. Every couch path that
brings a thread back -- resume, and the relaunch #182 is adding -- reads
`record.LatestLaunchProfile.Agent` and re-launches that same agent
(`couchcore/resume.go:187`, `CheckResumePreconditions`). There is no couch
gesture that says "same work, different driver".

The operator's escape today is to leave couch: park or quit the thread, drop to
a shell, and run `pair <agent>` -- which #115 taught to route an explicit-agent
request onto existing work. But #159 made the couch TUI the public CLI, so the
one place the operator actually lives is the one place the move is missing.
That matters exactly when it is needed: a degraded provider, an exhausted quota,
or a deliberate choice to put a different model on this particular thread.

Two neighbours bound this issue and neither covers it:

- **#115 (done)** built the launcher-level substrate -- repo-scoped per-agent
  launch defaults and explicit-agent picker routing. It runs from a shell, and
  deliberately refuses to take over a live foreign-agent session.
- **#135 (open)** is the live handoff: quiesce a running source agent and hand
  the tag to a target agent while it is still up. That is the hard, deferred
  case.

This issue is the cold, safe one, at the couch layer: park the thread, then
bring it back under a different agent, in place.

## Spec

### The action

A **switch-agent** action on a thread, alongside detach, park, and #182's
relaunch. It is #182's park-then-resume composition with the agent axis moved:
park the thread to a verified park, then launch a fresh Pair from the CURRENT
binary at the same working path, under the same thread address, with the TARGET
agent.

The launch-profile substrate is already shaped for this.
`couchcore.ResolveLaunchProfile` resolves agent and argv on independent axes:
`AgentSourceExplicit` supplies the target agent, and argv comes from
`PathLaunchPreference.ArgvByAgent[target]` -- "selecting another agent may reuse
that agent's path argv, but can never inherit argv recorded for a different
harness" (`couchcore/launchprofile.go:55`). A switch is an explicit-agent
resolution against the thread's existing path preference. No new storage.

### The conversation does not come along, and the action must say so

The native binding is keyed by `(scope_key, tag, agent)` in the session ledger
(`sessionledger.Record`: `RootNativeID` per agent). A thread that has only ever
run Claude has no `root_native_id` for Codex, so
`bindingResumeDiagnostic` returns `ResumeBindingUnbound` and
`CheckResumePreconditions` refuses -- by construction, for every first switch to
a given agent.

That refusal is correct for relaunch and wrong for this action. A switch is a
**cold start of a new driver under an existing thread**:

- **Survives:** the thread address and its row in the switcher, name and
  description, working path, draft pane, sent-prompt history, future queue,
  artifact state -- everything the tag owns.
- **Does not survive:** the agent's own transcript. The target agent starts with
  no conversation.

So preconditions diverge from relaunch's: `CheckResumePreconditions`'s binding
rule must be skipped for the target agent, and its absence must not read as an
error. It is already factored for this -- `isBindingDiagnostic` exists so "a
caller can apply the rest and skip these". What still holds: the target is
`launcher.IsSupportedAgent`, the working path resolves, the thread is not
occupied, and -- per #182's rule that a park you cannot come back from is worse
than no action at all -- every precondition is checked BEFORE the park.

The operator must be told the transcript is being left behind, before the park,
not after. A switch that silently reads like a relaunch is the failure mode this
issue is most likely to ship.

### Continuation as the bridge (design question, not settled)

The distilled-context answer already exists in the repo: a continuation document
written by the source agent, which the target reads on boot. #115's original
framing had exactly this as the point of switching drivers.

Open question this issue must decide, not assume:

- Ask the source agent for a continuation before parking, offer it to the target
  -- richer, but slow, and it needs a healthy source.
- Switch cold and let the operator carry context by hand -- always available.

The degraded-provider case argues the second must exist: #135 names precisely
the situation where "the current agent can no longer produce the continuation
document itself". A continuation-backed switch is then an enhancement on top of
a cold switch that always works, never a precondition for it.

### Surface

- **Live threads are out of scope.** A live thread routes through park like any
  other couch teardown; taking over a *running* foreign-agent session is #135
  and stays there.
- **Target selection needs a prompt** -- switch-agent is the first couch action
  with an argument. The switcher's edit prompts are the precedent (and #164's
  prefill applies: default to the last agent used at this path that is not the
  current one).
- **Reuse #182's status pane**, the one that outlives its child. A switch has a
  longer gap than a relaunch -- cold agent boot, no session to resume -- so the
  "blank page is indistinguishable from a hang" argument applies with more
  force, not less.
- **Scope follows focus**, as with relaunch: from an actor it switches that
  actor and returns to it; from the panel it switches the highlighted row and
  leaves the operator in the switcher.

## Done when

- A thread whose saved profile names agent A comes back live under agent B, at
  the same thread address, same working path, same row, with its draft/queue/
  history intact -- from inside couch, without dropping to a shell.
- The absence of a native binding for the target agent does not refuse the
  switch, while relaunch's binding precondition is unchanged.
- The operator is told, before the park, that the agent transcript does not
  carry over -- and an acceptance test asserts that warning.
- Every precondition is checked before the park; a switch that cannot complete
  parks nothing. A switch that parks and then fails to launch leaves a verified
  park the switcher can resume, and says so.
- Argv for the target agent comes from that agent's own recorded argv or its
  default -- never from the source agent's argv.
- An unsupported target agent, a missing working path, and an occupied thread
  each refuse with a specific diagnostic.
- Atlas records switch-agent alongside detach/park/relaunch, including what a
  switch keeps and what it drops.

### Absorbed from `#176` (closed as superseded, 2026-09-07)

`#176` approached the same gap from the panel/gesture side. Its design is folded
in here rather than lost; the two turned out to be **complementary**, not
duplicative — this issue establishes that the conversation cannot *resume*, and
`#176`'s carrier is the answer to that.

**One operation, not two.** `switch-agent` takes the target agent as an
argument, and **restart is the same call with target == current**. Do not build
two commands that diverge. Combined with the argv axis this gives one action
with four degenerate cases:

| | agent | argv | conversation |
|---|---|---|---|
| `#182` relaunch | current | current | kept (same binding) |
| restart agent (`Alt+Shift+N`) | current | current | fresh |
| switch agent | **target** | that agent's `ArgvByAgent` entry | carrier (below) |
| new params | current | **new** | per the above |

**Declaration.** Beside the others in `couchcore/ops.go` with a row in the
declaration table (`ops_declarations_test.go`). Shape: `ExecuteLiveOwner`,
`EffectProcess`, **`ConfirmRequired`** — it destroys a live session, the same
grounds on which `stop` is the one existing `ConfirmRequired` operation. `#159`
made the TUI the public CLI, so one declaration yields both surfaces.

**Quiesce is observed, not acknowledged.** couch observes the current driver's
exit itself rather than trusting an acknowledgment from the process being
stopped. A source that dies silently is a case the tests must cover.

**The context carrier, chosen by source HEALTH rather than by trigger.** This
answers this issue's own "the target agent starts with no conversation": the
conversation cannot be *resumed* (no native binding for the target), but a
carrier can be handed over, and the rule covers switch and restart alike:

- **Source healthy** — the ordinary restart-to-refresh, and a switch made by
  choice ⇒ a **continuation**, as `#115` does. Distilled, and the agent is
  present to write it.
- **Source degraded or gone** — quota, provider outage, a wedged session ⇒ the
  **path to the prior transcript**, handed over for the new session to read if
  it wants. Free, requires nothing of the dying agent, raw rather than
  distilled: a worse summary and a strictly more available one.

The fallback is what makes the operation usable in the cases that motivate it
(`#135` lists them: degraded provider, exhausted quota, an agent that can no
longer produce the continuation document itself). Choosing by *health* rather
than by which case triggered it is what keeps it one rule.

**The prior transcript is already preserved — the work is selection, not
rotation.** Per-agent artifacts carry the agent in the filename
(`scrollback-<tag>-<agent>.raw`), so a claude→codex switch writes a different
file by construction, while tag-scoped artifacts (draft, log, ledger, queue) are
deliberately shared — the `#115` model. And pair's quit path already *moves* the
scrollback to a timestamped `parked-scrollback-<tag>-<ts>` base ("move on quit,
copy on compaction", `launcher/osruntime.go:841-856`). Those snapshots
accumulate per tag, so the handover must name **which** one belongs to the
session just replaced: reuse `ParkedScrollbackArtifacts` rather than adding a
second snapshot path (`ARCH-DRY`).

### One thing neither issue decided

**The preference store is keyed by path, not by thread.**
`path-preferences/<digest>.json` digests `RepoIdentity` + `PhysicalPath`, so two
threads at one path share `LastAgent` and `ArgvByAgent`. Switching agent on one
thread therefore nudges the default for the next start at that path. That is
probably wanted — the preference reads as "which agent do I use for this repo" —
but it is a semantic choice about shared state, not an implementation detail, and
it should be stated before building rather than discovered afterwards.

## Plan

`#182`'s relaunch machinery (status pane, park-then-launch composition,
precondition-before-park ordering) has **landed**, so this issue's stated
precondition is cleared. Steps below are `#176`'s, kept because they are the
concrete ones.

- [x] Decide the path-vs-thread keying question above; record the answer in
      `## Spec`.
- [x] Declare `switch-agent` in `ops.go` + the declaration table, with the
      `ConfirmRequired` shape.
- [x] Quiesce-and-observe on couch's side; test a source that dies silently.
- [x] Skip `CheckResumePreconditions`' binding rule for the target agent via the
      existing `isBindingDiagnostic` factoring; assert its absence is not an
      error.
- [x] Carrier selection on source health: continuation when available,
      `ParkedScrollbackArtifacts` transcript path when not.
- [x] Restart = same call, target == current; assert **one** code path, and pin
      subsystem-identity change (zellij/nvim/pair-wrap), not just agent liveness.
- [x] Failure path: assert the thread is still startable after a failed switch.

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md`
against `baseline-v3.1.md`. Method A only.* Calibration is marked stale by
`sdlc estimate-source`, so these are provisional focused ship-hours.

The rows below follow the work in order: issue/spec discussion; bounded argv
parser; structured command transport; park descriptor integration; switch
orchestrator; wrapper delivery state; Couch form; complete-chain acceptance;
operator docs; boundary review; and discovery of each supported harness.
Implementation values are 40% of v2/v2.1 primitives, with familiar-stack factor
1.0. Settled implementation designs use the 0.2 design discount; the issue/spec
row retains the design conversation cost. Thorough-plan design buffer is 15%.
The library check found reusable JSON, terminal recognition, lifecycle, inventory
and runner implementations in the current stack. The small parameter parser is
deliberately limited to quoting without expansion; no new dependency is needed.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=1.00 impl=0.08
item: smaller-go-module design=0.04 impl=0.16
item: cross-cutting-refactor design=0.12 impl=0.20
item: api-integration design=0.40 impl=0.60
item: greenfield-go-module design=0.20 impl=0.32
item: tui-screen design=0.40 impl=0.40
item: tui-screen design=0.20 impl=0.40
item: api-integration design=0.20 impl=0.60
item: atlas-docs design=0.02 impl=0.08
item: milestone-review design=0.00 impl=0.20
item: real-api-discovery design=0.00 impl=0.16
item: real-api-discovery design=0.00 impl=0.16
item: real-api-discovery design=0.00 impl=0.16
item: real-api-discovery design=0.00 impl=0.16
design-buffer: 0.15
total: 6.65
```

Design subtotal 2.58 × 1.15 + implementation subtotal 3.68 = 6.647 hours,
rounded to 6.65. This is an estimate, not a limit or a measured actual.

## Log

### 2026-09-04

Filed after a sweep of pair's issue set for an existing couch-level agent-switch
task found none: #115 (done) is launcher-level and cold, #135 (open) is the live
handoff, #182 (working) is relaunch with the *saved* profile. Grounding read
while filing: `couchcore/launchprofile.go` (agent/argv independent axes),
`couchcore/resume.go` (`CheckResumePreconditions`, `isBindingDiagnostic`),
`sessionledger/record.go` (binding keyed per agent -- the reason a switch is
cold).

### 2026-09-13 — Feature discussion and implementation plan
- 2026-09-13: closed — Operator accepted live smoke: parameter cursor edits, corrected Codex startup, Claude/Agy switching and Copy orientation prompt recovery. Full make test and go test ./... passed before final fixes; final affected package integration and wrapper/lifecycle race suites passed after fixes. Owned live-source acceptance traverses real cleanup/archive metadata through actual launcher/layout/wrapper delivery and preserves draft/log/queue. BR1 simultaneous terminal reply scheduler, BR2 live lifecycle composition and BR3 huge sparse retry counters/cancellation regressions pass. Four-agent disposable live conformance completed. make build and git diff main --check pass; README, atlas, plan and lessons updated.; review verdict: SHIP

Claimed the issue and ran `sdlc start-plan`. Operator decisions are captured in
the revisions below. The current durable implementation plan is
[`000184-couch-switch-thread-agent-plan.md`](../plans/000184-couch-switch-thread-agent-plan.md),
committed locally as `41dd366f`. It supersedes the historical checklist above.
Fresh-context spec review approved planning. Initial plan review found gaps in
park-metadata persistence/retry and operator input between paste and submit;
fixed those, added the lossy argv transport discovered during local inspection,
and received an approved re-review. Prevention rules were added to
`workshop/lessons.md` (ARCH-ORDER, ARCH-DRY, ARCH-PURPOSE).
Validation: `git diff --check` passed; this is documentation only, so no runtime
test result is claimed. Awaiting operator plan approval before `change-code`,
estimate derivation, or implementation.


### 2026-09-13 — Implementation and integration

Operator approved the plan and implementation. `sdlc change-code --issue 184`
passed the plan-quality and estimate gates; branch is
`000184-couch-switch-thread-agent`. Commits `670ea514`, `4a0d0ced`, `41e46549`,
`75eac627` and `fd4b9ec8` carry exact park metadata, fresh argv transport,
Couch UI, source resolution and the full launch-chain test. Root integration
adds exact nonce readiness before preference commit and source revision CAS.

Focused switch, UI, renderer, lifecycle, readiness and context tests pass. The
Couch → launcher → real layout → wrapper → PTY test receives the exact generated
prompt once and preserves source files. Full integrated tests found a stale
launcher environment expectation and an internal-operation presentation mismatch;
corrected both. Live smoke found terminal capability replies incorrectly
cancelling orientation; the wrapper correction and live recheck are in progress.
Final full-suite and boundary review remain pending; this is not completion.


### 2026-09-13 — Verification complete; entering boundary review

Implemented the approved revised contract. Historical Plan checkboxes denote
resolution under those revisions: no source-authored continuation is used, and
the existing agent-only restart chord is preserved. The durable plan records
the delivered flow and integration details.

Verification: `env -u PAIR_SESSION_ID -u PAIR_TAG make test` passed, followed by
`go test ./... -count=1` and a final full integration-package run after the last
changes. Full `-race` orientation/wrapcmd/readiness suites passed (wrapper80.697s).
The actual Couch-profile → launcher → layout → wrapper → PTY acceptance passes.
`git diff --check` is clean. Live disposable-project checks confirmed automatic
orientation submission for Claude2.1.270, Codex0.154.0, Agy1.1.25 and
Muse0.1.0-R708.1. Claude/Codex/Muse read context and replied; Agy's long1,784-byte
prompt submitted once and reached its native permission request for the exact
synthetic log-render command. Manual test setup/approval is distinct from the
app's automatic input. Exact attempts, PIDs and captures are retained locally in
`/tmp/pair-184-orientation-smoke.hWzQJL/summary.json`; direct-wrapper smoke does
not create a Pair native-binding ledger. Boundary acceptance remains pending.


### 2026-09-13 — Live conformance and first boundary verdict

All four installed agents completed context-reading responses. Agy's final
1,784-byte/25-line attempt completed in81.27s and returned to its empty composer;
two one-time native approvals covered only the synthetic renderer command and
its generated temporary-output cleanup, after automatic orientation submission.
All smoke processes were stopped. Final local evidence is in the previously
recorded `summary.json`, including `agy/longcomplete`.

The mandatory boundary gate returned REWORK (window065929e5..445bbefe): BR-1
requires retaining a consumed settle event when a solicited terminal reply wins
input priority; BR-2 requires successful live-source teardown/archive transfer
in the full-chain acceptance. Both are being addressed under the existing plan.
The gate did not finalize close; status remains working.

## Revisions

### 2026-09-13 12:04 PDT — Target owns context reconstruction

Reason: operator clarified the switching behavior during the feature discussion.
These decisions supersede the earlier continuation/source-health carrier rules
and resolve the path-default question; implementation design remains in discussion.

- Every switch starts a fresh target-agent context. Claude → Codex → Claude
  starts a new Claude session that inspects the outgoing Codex session, never
  resumes the earlier Claude conversation.
- The target agent always inspects the source session's transcript or Pair TTY
  log for context. Do not ask the source agent to write a continuation, even
  when healthy. Source health does not select a different handover mechanism.
- Switching changes the default agent for future starts at that working path,
  until it is changed again. This is deliberately shared path preference state,
  not a thread-only setting.
- The Plan's continuation-on-health step is superseded by target-side inspection
  of the outgoing session's transcript or Pair TTY log. Artifact selection and
  target boot behavior remain to be designed.

### 2026-09-13 12:17 PDT — Shared path parameters and orientation prompt

Reason: operator refined the target selection flow and clarified that "thread
preference" meant the existing path preference. These decisions extend the
previous revision and supersede suggestions of separate per-thread storage or
a one-click bypass of parameter review.

- Select the target agent from claude, codex, agy, and muse. Always present a
  startup-parameter screen, prepopulated with that agent's remembered parameters
  for the working path, or its existing defaults when none are remembered.
  The operator can edit the parameters or accept them unchanged.
- Reuse the existing path preference store for both the default agent and each
  agent's startup parameters (ARCH-DRY). These preferences are shared by threads
  at that path; add no per-thread preference storage. Record the selected agent
  and parameters after a successful launch.
- After switching, send the fresh target session a prepared orientation prompt
  identifying the outgoing agent/session and the locations of its Pair log and,
  when available, its native transcript. Prefer the Pair log for its level of
  information. The target reads the prior session's context and replies with an
  orientation summary; it is not instructed to resume unfinished work.
- Bind the prompt's artifact references to the session just replaced, including
  on switches back to a previously used agent. Never resume that target agent's
  old native conversation and never require a source-authored continuation.
- Preserve the thread identity, working path, metadata, draft, sent-prompt
  history, queue, and artifact state as already specified.

Planning still needs to establish the exact log artifact and its lifetime,
target readiness and prompt delivery, and behavior when context artifacts are
unavailable. These are not reasons to reintroduce a dependency on source health.

### 2026-09-13 — Approved feature contract and current-code corrections

Reason: operator approved the feature summary and missing-log recovery rule.
Source inspection also found that two assumptions in the original issue have
been superseded by subsequent work. This is the current contract for planning;
earlier conflicting prose and Plan steps are historical.

1. Offer an explicit switch-agent action on a selected live or verified parked
   thread. A detached/ambiguous owner must first be resolved through the existing
   attach/recovery flow; never take over an unverified foreign live owner.
2. Select a supported agent, then always review editable startup parameters.
   Prefill from that agent's existing path preference, otherwise its repository
   defaults. Canceling either screen leaves the thread and preferences unchanged.
   The final submit names the source and target and explains that this starts a
   fresh conversation; it is the confirmation before destructive work.
3. Revalidate the accepted target, parameters, path, ownership, and launch
   prerequisites before parking. Park an owned live source and observe its exit;
   an already verified parked source needs no second park. Start a fresh target
   on the existing address, with the same tag-owned data. No target native binding
   is required, even when switching back to a previously used agent.
4. After successful launch, update the existing shared path preference for the
   default agent and that agent's parameters. Other agents' parameters remain.
5. Automatically submit a prepared orientation prompt to the fresh target. Name
   the outgoing agent/session and exact available source artifacts. Prefer the
   readable Pair TTY scrollback (includes replies); the shared sent-prompt log
   and native transcript are supporting context. Request a summary of objective,
   progress, decisions, and remaining work, then wait for operator direction.
   Do not involve the source in preparing a continuation, mutate the existing
   draft, consume the future queue, or label this generated prompt as operator
   authored text. The operation does not wait for the model's summary to finish.
6. Missing context artifacts are explicitly reported but do not block switching.
   A preflight refusal parks nothing. Incomplete park uses existing park recovery.
   Failed launch reports the actual parked/occupied state. Failed or uncertain
   prompt delivery leaves the launched session usable with a manual recovery
   instruction; never blindly resend an uncertain submission or relaunch again.
7. Keep panel-origin actions on the panel and return actor-origin actions to
   their replacement, using existing progress notices. A persistent holding
   pane has NOT landed with #182; it belongs to open #186 and is not a dependency.
8. Choosing the current agent in this explicit action uses the same fresh launch
   and orientation flow. Do not remap Alt+Shift+N: it now invokes an agent-only
   restart that promises to retain the running workbench, unlike this action.
   This supersedes the absorbed #176 shortcut/subsystem assertion.


### 2026-09-13 — Operator live acceptance required

Operator instruction: ask them to live-smoke the feature before closing the
issue. Finish review fixes and automated checks, prepare a built candidate and
smoke steps, then wait for their result. Do not rerun close or publish before
that acceptance. The issue remains working after the first review's REWORK.

### 2026-09-13 — Review fixes and candidate verification

BR-1 and BR-2 are implemented with deterministic scheduler and owned-live-source
acceptance regressions. Full wrapper race tests and parked/live acceptance race
tests pass. Root reran uncached couchcmd/launcher/wrapcmd suites successfully
(`/tmp/pair-184-review-fixes-test.log`), checked the diff, and built the candidate
with `make build` (`/tmp/pair-184-smoke-build.log`). Awaiting operator live smoke;
issue stays working and the next close/review is deferred until that result.

### 2026-09-13 — Operator smoke findings

Operator reports accepting Codex defaults failed with unexpected argument
`--sandbox danger-full-access`; switching back to Claude worked, and Agy also
started. Read-only inspection of brain's repository default and shared path
preference confirms Codex argv is stored as one combined element. The editable
form preserves that boundary by quoting it. Correct entry is two unquoted tokens
`--sandbox danger-full-access`; do not globally split stored argv elements since
values may legitimately contain spaces. No user preference files changed.

Agy's orientation reports cancellation from operator input. Investigating actual
readiness/terminal evidence and whether the operator pressed a key before
orientation completed. Live acceptance remains pending; no close or publication.

Agy diagnosis: captured first stdin packet matches the synchronized-output
DECRQM response and Kitty keyboard reply; preceding output contains their
queries. Unsupported DECRQM admission caused false operator-input cancellation
before prompt paste. Implement query-correlated support and regressions; the
initial cancellation does not require a human keypress to explain it.

Exact read-only evidence: attempt `start-d99e8470336e52a8`, child PID 82019;
ready state cancelled with body not written. Initial stdout bytes 0–1224 include
queries for modes 2026/2027 and Kitty keyboard support. Input sequence 17 at
13:54:25.613931 is the 16-byte response with SHA256 prefix `9aed9bfbb79c`,
`ESC[?2026;2$yESC[?1u`; the old classifier rejects it deterministically.

Both smoke defects are fixed: parameter cursor editing and solicited DECRQM
recognition. Focused regressions, uncached couchtty/couchcmd/wrapcmd suites, and
full wrapper race tests pass (81.834s). Candidate rebuilt successfully. Codex's
existing combined argument remains editable without mutating preferences behind
the operator's back. Awaiting fresh operator smoke before close.

### 2026-09-13 — Codex smoke after parameter correction

Operator confirms Left/Right editing works and removing the saved surrounding
quotes permits Claude→Codex startup. Orientation was absent. Exact current ready
record `start-394daa428e9b4b4b` reports cancelled before body with reason
“a dialog interrupted automatic orientation”. Raw bytes 2025–2630, master chunk
95 at 14:04:05.120869, contain the actual directory trust dialog; the resolved
composer appears after an input event at 14:04:09.151182. This is the approved
sticky dialog cancellation, not another false protocol classification. Explain
manual recovery and discuss whether startup-dialog waiting should replace this
behavior. No production change or acceptance/closure inferred from this smoke.

### 2026-09-13 — Operator accepted live smoke

Operator confirmed manual Copy orientation prompt recovery works and stated:
“we can consider smoke test passed. we can move to close this issue now.”
Parameter editing, corrected Codex startup, Claude/Agy switching and orientation
recovery are accepted. Startup trust dialogs continue to cancel automatic
delivery as designed; waiting through them is outside this accepted change.

### 2026-09-13 — Boundary review round 2 REWORK

BR-1/BR-2 were accepted as fixed. New BR-3 requires bounded retry-history lookup
and cancellation: a large persisted counter can cause arbitrary missing-file
reads under the cleanup lock. Fixing this internal recovery path and adding
regressions before rerunning close. Operator smoke acceptance remains recorded.

BR-3 fixed with 32-read descending lookup, explicit budget failure and prompt
cancellation. Nil newer completions do not mask older captures; cancellation
after cleanup still publishes moved capture metadata. Lifecycle regressions
and race suite pass; root uncached lifecycle/core/command/launcher suites pass
and candidate rebuilt. Atlas and lessons updated; rerunning close.
