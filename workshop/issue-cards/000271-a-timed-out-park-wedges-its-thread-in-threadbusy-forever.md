---
id: 000271
status: open
created: 2026-09-16
updated: 2026-09-16
estimate_hours:
github_issue:
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
