---
id: 000184
status: open
deps: [pair#182]
github_issue:
created: 2026-09-04
updated: 2026-09-04
estimate_hours:
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

## Plan

Plan lands after `sdlc claim` + `sdlc start-plan`; #182's relaunch machinery
(status pane, park-then-launch composition, precondition-before-park ordering)
has to be in the tree before this is designed for real.

- [ ] Design against the landed #182 relaunch operation

## Log

### 2026-09-04

Filed after a sweep of pair's issue set for an existing couch-level agent-switch
task found none: #115 (done) is launcher-level and cold, #135 (open) is the live
handoff, #182 (working) is relaunch with the *saved* profile. Grounding read
while filing: `couchcore/launchprofile.go` (agent/argv independent axes),
`couchcore/resume.go` (`CheckResumePreconditions`, `isBindingDiagnostic`),
`sessionledger/record.go` (binding keyed per agent -- the reason a switch is
cold).
