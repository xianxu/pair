---
type: target
slug: slot-recovery
status: active
created: 2026-10-14
updated: 2026-10-14
sources: ["pair#528", "pair#553", "pair#556", "pair#557", "pair#561"]
---

# Slot recovery: three outcomes, and a crash-only Couch

What we defend: after **any** interruption (a Couch exit or crash, a restart, a reboot, a forced sleep), every slot
lands in one of a few user-visible states, each with one recovery action, and nothing durable is lost.

## The user-facing states (one action each)

| State | Meaning | Action |
|---|---|---|
| **Continue** | The slot's agent is running (zellij lists it live) | Attach automatically at startup |
| **Resume from disk** | Not running, but its conversation is proven (or confirmed by the operator) from the agent's transcript | Relaunch the agent resuming that transcript |
| **Start fresh** | Not running, and no conversation can be proven | Launch a new session oriented from the preserved TTY log, branch and issue |

Plus **needs relaunch** for dead or resurrected panes with no running command: never shown as live. Uncertain
observations stay uncertain; an inferred conversation is never resumed silently (pair#528).

Internal state may be as detailed as it needs to be, but it maps onto these few states; the operator never needs
to understand the internals to pick the action.

## Couch is crash-only

Couch may exit at any time (a stuck terminal write after a wake, a bug, a kill). We do **not** try to make it survive
every terminal or OS condition. Instead, everything else treats a Couch exit as routine:

- **Durable state lives outside Couch:** in git (issues, branches, suite records) and in the agents' transcripts.
  Couch's in-memory state (receipts, leases, row projections) must be rebuildable from those, or safe to lose.
- **Every exit leaves a crash record** (cause, build, uptime, last events), so we always know why it exited.
- **Restart is cheap and complete:** live slots reattach (Continue); the rest show Resume or Start fresh.
- **Messages reconcile from transcripts:** each delivery is printed in the receiver's transcript with its ID, so after
  a restart, sends whose outcome was lost are matched against the receivers' transcripts and resent if undelivered
  (pair#561).

## How we know

- Acceptance for any Couch change touching startup or recovery: an operator restart with live slots, where every slot
  shows the right state and Continue rows attach without a keypress.
- pair#561's tests: a blocked terminal write yields a crash record; a simulated restart reconciles messages by ID.

## Revisions

- 2026-10-14: created by the TL from the operator's direction after the forced-sleep exit ("design the rest of the
  system to allow Couch to crash out; leave the harder persistence to git and the agents' transcripts").
