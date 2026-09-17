---
id: 000271
status: open
deps: [pair#192]
github_issue:
created: 2026-09-16
updated: 2026-09-16
estimate_hours:
---

# A timed-out park wedges its thread in ThreadBusy forever

## Problem

Split out of `pair#265`, which found this while diagnosing why couch cannot
start usefully in `brain`.

`ClassifyThread` (`couchcore/actionableinventory.go:265`) branches:

```go
if record.Park != nil {
    return ThreadBusy, ""
}
```

Total, unconditional, above every evidence-consulting branch. A park that timed
out leaves `record.Park` set, so its thread is `ThreadBusy` ("parking in
progress") **forever** — and `ThreadBusy` is invisible to every startup
predicate:

- `SelectResumableRoot` ranks only `Detached`/`Parked` → never resumed
- `PathHoldsUsableThread` matches only `Live`/`Detached`/`Parked` → does not hold the path
- `reattachCandidate` wants `Detached`, or `Unusable` with `ReasonUnknown` → the pass skips it

So `couch` in that tree spawns a *fresh* thread on every start and leaves
another record behind, instead of returning the operator to their work.

Observed live (2026-09-16), record
`threadstore/records/2e51fcf9799b1d8f/couch-e1a31510b7033d08.json`, rev 293,
last written 2026-09-15 22:53:

```json
"park": { "phase": "awaiting_completion",
          "identity": { "pid": 64734, "process_identity": "1789535173.46673" },
          "attempts": [ { "number": 1, "timed_out": true,
                          "failure": { "code": "timeout",
                                       "diagnostic": "matching completion was not observed" } } ] }
```

pid 64734 is dead (verified). `couch --list` showed brain carrying this wedged
row plus three `stale — helper ownership unresolved` records dated 09:02, 09:29
and 10:00 that day — one per start attempt.

**This is a missing reconciliation between recorded state and the world, not a
park bug per se.** `ClassifyThread` is built as a reconciler everywhere else:
`ThreadEvidence` is documented as "everything the IO shell resolved about one
record, **and whether it managed to resolve it**"; `ProofStatus` is three-valued
(`ProofUnresolved` = "never asked, or asking failed"); `liveProofMatches`
correlates the record against an observed `ProcessIdentity{PID, Identity}` with
"exact identity match so a recycled PID cannot pass"; and `reattachCandidate`
deliberately consumes that third value. `startInFlight` even carries the comment
that is this issue's missing rule:

> What separates it from a start that DIED is process evidence, not the recorded
> state: a creating incarnation whose process is gone is stale in exactly the way
> the word means.

`ParkIdentity` already persists `{PID, ProcessIdentity}` — the same key
`liveProofMatches` uses, recorded so ownership could be checked — and no
classifier reads it. A park in flight whose process is gone is stale in exactly
the way the word means; couch holds the evidence and never looks (ARCH-ORDER:
observed state vs desired state vs operation outcome).

Recovery is written but disconnected. `PairLifecycleController.ReconcileActive`
exists with tests, and `Couch.ReconcileActiveParks` sits in the dead-symbol
allowlist as `"pair#192: explicit reconciliation pass with no caller"` — hence
the `deps: [pair#192]`. Wiring it is **not sufficient**: for the record above
(attempt 1 `timed_out: true`, `closed` unset) `reconcileActive` re-observes,
finds nothing, falls through to `runActiveAttempt` and *retries*. It has no
"the owner is dead, this transaction is orphaned" rule. `Abandon` is the only
transition that clears a park, and its sole caller is an operator-driven op —
so the wedge survives every automatic path.

## Spec

- Give `ClassifyThread`'s park branch **evidence**, the way every sibling branch
  has it: a park whose owner process is gone is not "in progress". Reuse the
  existing `ProcessIdentity{PID, Identity}` correlation rather than adding a
  second liveness notion (ARCH-DRY) — `ParkIdentity` already persists both
  fields.
- Keep the three-valued discipline: "the owner is gone" and "we could not ask
  whether the owner is gone" are different answers and must not collapse into
  one state. Extend `ThreadEvidence`/`ProofStatus` rather than inventing a
  parallel channel.
- Define the orphaned-park transition. A park whose owner is dead and whose
  attempts are exhausted needs a terminal disposition reachable **without an
  operator gesture**; today only `Abandon` clears one. Decide whether that is an
  automatic abandon, a new classified state the operator can act on from the
  switcher, or a reconciliation pass at startup — and say which, with the
  ordering rules (ARCH-ORDER).
- Decide `ReconcileActiveParks`: wire it with the orphan rule added, or delete
  it and put the rule where it belongs. Either way it leaves `pair#192`'s
  allowlist; do not leave a third state where a written reconciler sits unused.
- The wedged thread must become reachable again: whatever the disposition, the
  operator ends up able to resume, archive, or start fresh in that tree without
  hand-editing a record.
- Bound the residue (ARCH-FUNERAL): a start that refuses to reuse a wedged
  thread currently mints a new record per attempt. Say what removes those.

## Done when

- A park whose owner process is dead no longer classifies as `ThreadBusy`
  indefinitely, and the rule is driven by evidence rather than by the recorded
  phase alone.
- "Owner gone" and "could not determine owner" are distinguishable states, each
  with a defined operator-visible disposition.
- `couch` started in a tree holding a timed-out park returns the operator to
  usable work — resume, archive, or a clean new thread — without a fresh
  orphan record per attempt.
- `Couch.ReconcileActiveParks` is either called from production or deleted, and
  is gone from the `deadsymbols_test.go` allowlist.
- Sequence tests cover: park times out then the owner dies; owner dies mid-park
  with the completion arriving late; liveness probe fails (unresolved); and a
  second couch observing the same wedged record.

## Plan

- [ ]

## Log

### 2026-09-16

- Split out of `pair#265` during its state-machine inspection. `pair#265` fixes
  the input-routing crash and the fatality seam; this issue owns the
  record-versus-world reconciliation. Evidence above captured from the live
  store on this date.

## Revisions

### 2026-09-16 — no operator-reachable escape hatch (severity raised)

Reason: checking whether the operator could unwedge `brain` by hand today, before
this issue ships. They cannot. The wedge is not merely "couch takes a worse
path" — the thread is unreachable by every gesture couch offers.

Delta:

- `menu.go:1238` gives a `ThreadBusy` row exactly two actions — `name` and
  `describe`. No `archive`, no park mode, no resume. Its comment states the
  assumption this issue disproves:

  > Something else is still acting on this thread. Offering archive here would
  > file a record mid-park -- the store refuses it, so the offer is an action
  > that always fails... **It resolves on its own**; metadata still applies.

  It does not resolve on its own. There is no timeout, no expiry and no
  owner-liveness check, so "still acting" is indistinguishable from "the actor
  died 18 hours ago". That comment is the bug stated as a comment, and it should
  be deleted with the fix rather than edited around.

- `menu_render.go:432` renders the row `parking…` forever.

- `Abandon` — the only transition that clears a park — is **unreachable**. No
  menu path constructs `mode=abandon`; the `park` operation is
  `PresentationTUI`, so the `--internal` CLI form refuses it
  (`cli.go:131` requires `PresentationInternal`); and `park` is absent from
  `operationOwnsLive`, so it only runs inside a live couch that the TUI must
  dispatch it from.

- `recover-thread` does not apply: `recovery.go:84` gates on
  `ReasonStaleIncarnation`/`ReasonSessionGone`, and a wedged park is
  `ThreadBusy`, not `ThreadUnusable`.

Net: the only way out today is hand-editing or removing the record file. That is
the strongest argument for this issue — an unreachable durable state with no
gesture is worse than a wrong classification.

Added to `## Spec`: the disposition must be reachable from the switcher for a
row whose park owner is provably gone, and `menu.go:1238`'s two-action list must
widen for that case. Added to `## Done when`: a wedged row offers an action that
clears it, and the "resolves on its own" comment is gone.

Also verified while checking: all four `brain` records carry a dead pid
(1807, 87309, 62226, 64734 — none alive), and the three non-park records have
`last_active_at` at the zero time, i.e. those threads were spawned and never
became active. `PruneDead` (`couch.go:1098`) prunes the *registry*, not
threadstore records, so nothing collects them — which is the ARCH-FUNERAL
residue bullet, now with a measured count.
