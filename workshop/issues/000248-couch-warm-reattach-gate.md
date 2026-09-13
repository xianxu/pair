---
id: 000248
status: open
deps: []
github_issue:
created: 2026-09-13
updated: 2026-09-13
estimate_hours:
---

# Couch switcher blocks warm reattach without a native binding

## Problem

Couch's switcher rejects a warm reattachment to an existing live Zellij session
when Pair has not established a native-agent session binding. It labels the row
`binding lost — repairable`; selecting it reports `its resume binding was lost;
the session may still be running (pair#168)`.

Observed 2026-09-13 for Tools, address
`434128d5ad68b26e/couch-253f4266b649cb01`. Read-only investigation around 16:38
America/Los_Angeles confirmed:

- Zellij session `📁tools-couch-3` was alive; `action list-clients` returned
  only the header (zero attached clients).
- Pair wrapper PID 69567 and Codex launcher PID 69577 were alive, started 16:03:36.
- The thread record (revision 5) retained its Codex launch profile and working
  path but had no incarnations or VerifiedPark.
- Its Pair ledger contained a launch and no native binding; rendered scrollback
  showed Codex's initial prompt. Missing binding is not evidence of process death.
- The operator then attempted reattachment in the switcher and supplied a
  screenshot of the refusal above.

Source inspection: `couchcore/actionableinventory.go:detachedResumeProofMatches`
requires `observation.NativeID != ""`. Failed proof with a detached observation
projects to ThreadUnusable/ReasonBindingLost. In contrast, `DecideResume` in
`couchcore/resume.go` allows warm resume without a native binding, with explicit
coverage in `warmresume_test.go`. The UI rejects the thread before that path
can run. These files must be rechecked at implementation time for concurrent work.

## Spec

Align detached inventory, switcher action eligibility, and warm-resume policy:
an exact, uniquely owned, live detached session is eligible for warm reattachment
without a native conversation ID, because reattachment starts no new agent.
Preserve checks for ambiguous ownership, active clients, and stale/dead sessions.
Cold resume must continue requiring the established native binding it consumes.

Use one shared eligibility contract across the consumers (ARCH-DRY,
ARCH-PURPOSE), rather than special-casing Tools or editing its metadata. Recheck
session liveness/ownership at execution so a stale inventory cannot turn a warm
reattach into a cold launch (ARCH-ORDER). Diagnostics must distinguish an absent
native binding from actual loss of the live session.

Why this thread detached remains unknown; finding that cause is distinct from
fixing the evidenced reattachment refusal. No process was restarted or metadata
repaired during investigation. #214 has the same displayed symptom after racing
launches, but those races have not been established in this Tools incident.

## Done when

- A live detached session without a native binding is selectable and reattaches
  successfully through the actual switcher-to-resume path, preserving its agent.
- Regression tests cross inventory classification, UI eligibility, and final
  attachment; calling DecideResume alone is insufficient coverage.
- Tests preserve refusal for dead/stale or ambiguously owned sessions, cover
  session death between inventory and execution, and verify cold resume still
  rejects missing native bindings.
- Operator diagnostics/docs agree with the shared eligibility policy.

## Plan

- [ ] Reproduce the inventory/UI refusal with a portable fake session and no native binding.
- [ ] Align warm eligibility across inventory, switcher, and execution; verify boundary races.
- [ ] Update diagnostics/docs and close through the SDLC review gate.

## Log

### 2026-09-13

Filed at operator request after read-only investigation and confirmation via
switcher screenshot. Evidence lives under
`~/.local/share/pair/repos/434128d5ad68b26e/` in the addressed ledger and
scrollback, and `~/.local/share/pair/couch/threadstore/records/434128d5ad68b26e/`.
These are historical observations; recheck process identities before any repair.
No implementation started.

Follow-up on disconnection cause: all eight current thread records have
LastActiveAt between 16:13:54.437 and 16:13:55.184 on 2026-09-13, with Tools at
16:13:54.549. The other seven have new live incarnations started 16:14:04.968
through 16:14:07.794; Tools alone has none. The current supervisor-owner records
PID 5316 with process identity corresponding to 16:14:04.532. This strongly
supports a Couch-wide detach/restart followed by reattachment of every thread
except Tools, rather than an isolated Tools agent crash. Detach updates
LastActiveAt through RetireIncarnation; its agent and Zellij server survive.
The evidence does not identify what initiated the Couch exit/restart.

## Revisions

### 2026-09-13T16:52:14-07:00 — Clarify native binding and warm-attachment limits

Operator requested preserving the explanation and constraints after confirming
that warm attachment should be offered. This supplements the Spec above.

A native binding is Pair's verified association between its thread and the
agent's own conversation ID/transcript. For a fresh launch, Pair normally
establishes it by uniquely matching a submitted operator prompt and subsequent
agent progress. Opening the agent alone is insufficient. An explicitly
authorized native resume can establish the binding at launch instead; see
`atlas/session-identity.md` for the scanner/authorization contract.

Warm reattachment uses the surviving terminal session, not the transcript.
Removing the native-binding gate must retain these constraints:

- Prove exact, unambiguous ownership of the live detached Zellij session and
  that it has no attached client. Missing transcript evidence does not waive
  session ownership or liveness checks.
- Recheck at execution. If the session dies or becomes occupied after the
  inventory snapshot, refuse visibly; never fall back to starting a fresh
  agent or conversation under a warm-attachment action.
- Successful attachment does not establish or repair the native binding.
  Restarting the agent or recovering its conversation after process death
  still requires an established binding on the cold-resume path.
- Transcript-dependent features may remain unavailable until Pair establishes
  the binding through its normal evidence flow. Present warm attachment as
  restoration of access to the running session, not proof of durable recovery.

Regression coverage must demonstrate both successful warm attachment without a
binding and preservation of these limits, including no fabricated binding and
no fresh-agent launch after the surviving session disappears.
