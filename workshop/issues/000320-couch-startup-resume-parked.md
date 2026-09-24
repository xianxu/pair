---
id: 000320
status: open
deps: []
github_issue:
created: 2026-09-24
updated: 2026-09-24
estimate_hours:
---

# Resume every parked thread when couch starts

## Problem

Parking is the cheap way to put a fleet of threads away (batch park frees
their agents and zellij sessions), but there is no matching way to bring the
fleet back. When couch starts it attaches one thread (`SelectResumableRoot`)
and the #206 background pass reattaches only *detached* threads — agents still
running behind a client-less session. That pass is `warm-only` by design: it
refuses a thread that is parked and never starts an agent. So every parked
thread has to be resumed by hand, one switcher action at a time.

Goal: park a batch, quit couch, start couch later, and have the batch back —
batch park paired with batch recover.

## Spec

Extend couch's startup background pass so parked threads are resumed too, after
the detached ones.

- **Same pass, one more class.** Reuse the #206 reattach pass in
  `menu_reattach.go` (seeded once from the first successful inventory, one
  attempt at a time, behind the operator, transitions owned by `MenuState`)
  rather than adding a second startup loop (ARCH-DRY). Seed order: detached
  threads first (cheap, warm), then parked threads, most recently active first
  within each class.
- **Parked threads resume cold.** A parked entry dispatches the ordinary
  `resume` operation (the same one the switcher's Enter uses), not
  `warm-only`. This deliberately reverses #206's "can only reattach an agent,
  never start one" for the parked class; the detached class keeps `warm-only`.
- **Proof before effect.** Each attempt re-proves its thread at its turn, as
  now. A thread that went live, was archived, or lost its native binding
  between seeding and its turn is skipped silently. Only threads that the
  switcher would list as `parked` (proof-bearing, exact `NativeBindingResolver`
  result) are seeded — unproved, ambiguous or unbound records never start
  agents.
- **Failure is per-row.** A failed cold resume marks the row
  `reattach failed:` exactly like a failed warm reattach; the pass continues
  with the next thread and never stops startup.
- **Bounded.** Resumes stay sequential through the existing capacity-one
  lifecycle worker (parallelism is #205's question, not this one). Each
  resume starts an agent process, so the set of threads resumed is
  what the scope's inventory already shows; no fleet-wide scan beyond it.

### Open questions (settle during planning)

1. **Always, or opt-in?** Resuming every parked thread starts N agents on
   every couch start, including after a deliberate park-to-free-resources. A
   likely shape: resume the parked threads from the *most recent* batch park
   only, or gate on a flag/preference. Decide with the operator before
   building.
2. **Which scope.** Startup is scoped to the requested repository today;
   confirm the parked seed uses that same scope and does not reach other repos.
3. **Interaction with #214** (racing launch operations leave a thread
   unresumable): the pass must not race an operator-initiated resume or
   relaunch of the same thread. The existing "behind the operator" rule
   covers the in-flight slot; confirm it also covers a cold start.

## Done when

- After batch-parking several threads and restarting couch, every one of those
  parked threads comes back as a live, attached row without operator action
  (subject to the opt-in decided in question 1).
- Detached threads are still reattached first and still `warm-only`.
- A parked thread that changed state between startup and its turn (resumed by
  hand, archived, binding gone) is skipped with no second launch.
- A failed cold resume marks only that row `reattach failed:` and the rest of
  the batch still resumes.
- Unproved/ambiguous parked records are never started.
- Tests cover the seeded order, the skip and failure paths through
  `MenuState`'s transitions, and a stateful-fake end-to-end startup with mixed
  detached and parked threads; the atlas's #206 reattach section is updated.

## Plan

- [ ]

## Log

### 2026-09-24

Filed at operator request: "reattach to all parked threads when couch starts
up, so we can batch park and batch recover." Grounding: atlas/couch.md
"Every other detached thread comes back in the background" (#206 M2) and
startup ranking (`SelectResumableRoot`). Related: #175 (resume a unique parked
thread), #205 (parallel batch park/detach), #214 (racing launches).
