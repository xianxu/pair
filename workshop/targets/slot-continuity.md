---
type: target
slug: slot-continuity
status: active
created: 2026-10-13
updated: 2026-10-13
sources: [the operator and the TL in ops:0, 2026-10-13 evening, after pair#528's first design stranded every slot; pair#517, pair#522, pair#528, pair#531, pair#540]
---

# Target: slot continuity — three outcomes, one action each

In couch/pair, we leverage several different systems to work together, coding agent, nvim as draft pane, pair wraps and monitors TTY, couch as coordinator, zellij as terminal multiplexer that manage processes for us. Those components communicate with each other, sometimes via pipes on the file system. In such environment, the overall state of couch/pair is scattered among those systems. If part of that state is corrupted, or some of those systems aren't functioning, couch's functionality is impacted. Those are mostly the "operational state". 

User cares about much less. Their project is largely in git repos coding agents work on. And a bit of transitional state is in coding agents' transcript file. And there is a bit of information stored by pair, such as TTY log, prompt history (what was sent) and the draft file (what's typed but not yet sent; the draft pane autosaves it to disk on focus loss). Those are about it.

To the end user then, from their point of view, the system operate at three levels. Starting or restarting Couch, recovering a broken slot, switching the agent and compacting it should each land on one of them:

1. **Continue.** A live agent is working with its transcript. Keep using it: attach, talk, nothing torn down. A broken channel to a live, attachable agent is still this outcome, repaired by reconnecting.
2. **Resume from disk.** The live agent is gone, damaged or unreachable, but its conversation can be recovered from disk. Tear down whatever is left (terminal, wrapper, sockets, stale layouts) and relaunch the agent resuming that transcript.
3. **Start fresh.** The transcript can't be recovered. Start a new agent and conversation, oriented from the slot's branch, its issue and the previous terminal log. Ariadne externalizes durable state to git (issues, plans, logs, commits), so the loss is limited to the conversation's working context.

We defend this because the failure mode it prevents is real and recurring: designs built bottom-up from mechanisms accrete states the person can't act on, and a safety reflex ("when unsure, refuse") turns into limbo — on 2026-10-13 a supervisor restart left twenty slots "needing relaunch" with no way forward, and a split session kept a live agent
running where no one could reach it. The **transcript is the asset**; terminals, servers, wrappers and sockets are rebuildable, and the design should spend boldly on rebuilding them to protect the transcript.

## Commitments

- **Map state to outcomes, not outcomes to state.** Internal states may be as many as the mechanics need; they reduce to the three outcomes above, and the row shows one action. A failure that only reports detail is not recovery.
- **Know which reads write.** Every observation names whether it is read-only. `zellij attach` is a write (it resurrects a session from a saved layout when no server answers), so it is never used to probe.
- **Observe before acting, cheapest first.** Startup observes and projects every slot's outcome; it acts on nothing and never refuses to start because of one slot. Action happens per slot, when the person (or the hub) picks the row's action, or when something attaches or sends.
- **Tear down only for outcome 2 or 3,** only with evidence that the live agent can't continue, and confirm first when a running agent would be stopped (the person, or the hub acting for them).
- **Educated guesses over refusals.** When the transcript's identity isn't proven, pick the best-supported candidate with its evidence visible and the alternatives one step away (pair#528's resume ladder); prove when cheap, guess when not, never strand.
- **Proportionality.** Rare, unrecoverable corners (a transcript truly gone) are accepted and land in outcome 3 rather than being designed around; effort goes to the common paths.
- **Start fresh is always reachable.** When no transcript can be found, or no candidate is strong enough to resume, the row offers Start fresh as its action; it never waits for proof that can't come.
- **The hub may act for the person.** The hub (TL) may take a row's action on the person's behalf, e.g. recovering a stranded slot overnight, within these commitments, and reports what it did.

## What this is NOT

- Not a spec of the state machine or the reconciliation table; those live with the issues (pair#528's state manifest) and change as the mechanisms do.
- Not a promise that every combination is recoverable without loss: outcome 3 exists on purpose.
- Not a reason to keep Couch messages durable: messages stay ephemeral; after a hub restart, slots are asked to resend.

