---
id: 000181
status: working
deps: [pair#168, pair#171, pair#179, pair#180]
github_issue:
created: 2026-09-03
updated: 2026-09-03
estimate_hours: 8.64
started: 2026-09-03T16:38:51-07:00
---

# One honest inventory: every thread gets a row and a reason

## Problem

The switcher shows 4 rows. The store holds 13 threads. `couch --list` shows all
13. Nothing reconciles those numbers, and nine threads are invisible with no
notice, no log line and no way to ask why.

Measured on the operator's live store, 2026-09-03:

```
state      resume proof        count
live       established           3     (only 2 are really hosted -- see stale, below)
detached   established           1     tools-couch-2   -- blocked from reattaching (pair#179)
detached   provisional/no-id     1     pair-couch-24   -- hidden (pair#168 lost its binding)
parked     established           1     parley          -- the ONLY resumable park
parked     provisional/no-id     8     brain x5, kbench x2, ariadne -- hidden AND unresumable
live(stale) --                   1     brain-couch-19  -- record says live, no console hosts it (pair#171)
```

**Root cause: "should this row exist" was fused with "can this row be acted
on", and the fused answer is computed half in an IO loop and half in a pure
projector, with `continue` as its only vocabulary for "no" (ARCH-PURE).**

The row set comes from one indexed directory:

```
threadstore/manifest.json  -> [{repo_scope, tag}, ...]
threadstore/records/<scope>/<tag>.json
```

`ThreadStore.Snapshot` (`threadstore.go:504-526`) walks the manifest. Three
proof sources are then joined on: live TTY observations (this console's own
children), the parked proof (pair's `repos/<scope>/ledger-<tag>.jsonl` via
`sessioninventory`), and the detached proof (zellij sessions + pair's
`session-names.jsonl`).

Rows then die in two places. `ActionableThreadInventoryContext:228-272` drops a
record BEFORE the pure function sees it -- reservation, park in flight, any
incarnation, no saved profile, unsupported agent, unphysicalizable path, no
resolver, and `bindingResumeDiagnostic != ""`, which alone accounts for 9 of the
13. Then `ProjectActionableThreads:101-142` / `actionableThreadState:144+` drops
more: invalid record, verified park without a matching parked proof, no
incarnation and no detached proof, an incarnation with no matching TTY
observation.

A pure function cannot be wrong about rows it never receives, and no test can
see the fused answer -- which is why this shipped through four milestones of
review.

Secondary, same root: **detached rows lie**. `menu_render.go:296-299` renders
anything not `Live()` as `parked · <age>`, so the operator's one visible
detached thread is labelled parked.

## Spec

**Every record in the manifest gets a row. State is total. Archive is the only
exit.**

1. **No shell-side filtering.** The IO shell gathers evidence and hands the
   pure projector every record plus whatever proofs it could resolve. It
   decides nothing (ARCH-PURE).
2. **State is a total function** over `(record, live observations, parked
   proof, detached proof)`:
   `live | detached | parked | busy(parking) | unusable(reason)`. There is no
   "absent". The reason taxonomy, each with its own exit:

   | reason | cause | exit |
   | --- | --- | --- |
   | `binding-lost` | pair#168 -- a trailing `launch` ledger row shadows the established binding | RECOVERABLE, repair not archive |
   | `stale-incarnation` | pair#171 -- record claims live, no console hosts it | reconcile to detached or parked |
   | `session-gone` | verified park whose native transcript is gone | archive |
   | `never-started` | reservation or rolled-back start | archive |
   | `invalid` | record fails `ValidateThreadRecord` | archive, inspectable |

3. **The row says why, and Enter explains rather than doing nothing.** An
   unusable row renders its reason where the state label goes.
4. **One inventory.** `couch --list`, the switcher and startup selection read
   the same projection. `--list` may render more detail; it must not see a
   different population.
5. **Archive is a deliberate move** to `threadstore/archive/`, reversed by
   moving the file back and re-adding it to the manifest (`Snapshot` enumerates
   `manifest.Threads`, so a restored file the manifest does not list stays
   invisible), and it is the ONLY way a row leaves the switcher.

Absorbs the display half of pair#171 and all of pair#180; pair#168 and pair#179
become the repairs that turn two reasons back into usable rows.

## Done when

- Rows in the switcher == records in the manifest, always, with no exceptions
  and a test that asserts the equality over a store containing every reason.
- Every reason in `AllThreadReasons()` renders its own distinct label, checked
  by iterating the vocabulary rather than a hand-listed sample, and detached is
  no longer labelled parked.
- `couch --list` and the switcher report the same population.
- pair#179: the operator's `tools-couch-2` reattaches from the switcher.
- pair#168: the eight `binding-lost` parks are re-measured after the repair and
  the count that recovered is reported, not assumed.
- pair#171: `brain-couch-19` shows as itself rather than vanishing.
- pair#180: only `session-gone` / `never-started` / `invalid` reach the archive,
  and archiving is an explicit operator action.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec                 design=1.50 impl=0.12
item: greenfield-go-module       design=0.20 impl=0.24
item: smaller-go-module          design=0.04 impl=0.20
item: cross-cutting-refactor     design=0.10 impl=0.20
item: smaller-go-module          design=0.02 impl=0.12
item: tui-screen                 design=0.16 impl=0.20
item: tui-screen                 design=0.12 impl=0.16
item: smaller-go-module          design=0.06 impl=0.20
item: atlas-docs                 design=0.02 impl=0.05
item: atlas-docs                 design=0.02 impl=0.05
item: milestone-review           design=0.00 impl=0.20
item: smaller-go-module          design=0.02 impl=0.12
item: smaller-go-module          design=0.06 impl=0.20
item: smaller-go-module          design=0.04 impl=0.14
item: smaller-go-module          design=0.06 impl=0.20
item: real-api-discovery         design=0.00 impl=0.18
item: milestone-review           design=0.00 impl=0.20
item: greenfield-go-module       design=0.20 impl=0.24
item: smaller-go-module          design=0.06 impl=0.20
item: smaller-go-module          design=0.04 impl=0.16
item: smaller-go-module          design=0.06 impl=0.20
item: atlas-docs                 design=0.02 impl=0.05
item: issue-spec                 design=0.30 impl=0.04
item: milestone-review           design=0.00 impl=0.20
item: ux-rename-iteration        design=0.15 impl=0.06
item: scope-pivot                design=0.50 impl=0.20
design-buffer: 0.15
total: 8.64
```

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only.*

**One line per instance**, so a close-time miss is attributed to a primitive
rather than to the issue. In order:

| Slug | Instances |
| --- | --- |
| `issue-spec` | this issue + plan authoring (2 review rounds already spent); expanding Chunk 3 after M2's re-measurement |
| `greenfield-go-module` | `ThreadReason` + `AllThreadReasons` in its own file; `DecideRetirement` + `RetirementVerdict` |
| `smaller-go-module` | `ClassifyThread` + `liveProofMatches`; the cost call-count guard; M2's red tests; warm reattach; the startup refusal message; `CurrentLaunch` pending-vs-committed + property test; `ThreadStore.Archive`; the archive reader + restore; `couch prune` |
| `cross-cutting-refactor` | the shell evidence pass + `ProjectActionableThreads`'s new signature across 36 call sites in 6 files |
| `tui-screen` | `rootStateText` + reason labels; Enter-on-unusable + `menuActionItems` |
| `real-api-discovery` | reattaching `tools-couch-2` on the real zellij/pair stack |
| `ux-rename-iteration` | one round on the reason labels — operator-facing text in a single column |
| `scope-pivot` | one, budgeted prospectively (see below) |
| `atlas-docs` | M1 atlas; README + its pinned test string; M3 atlas + project scope event |
| `milestone-review` | M1, M2, M3 boundaries |

**Where judgment entered:**

- **`issue-spec` for plan authoring takes no spec-quality discount**, following
  pair#170's precedent: the spec-authoring primitive cannot be pre-resolved by
  its own output, and this plan's hard question — what the classification
  vocabulary *is* — was open until the plan was written. Two review rounds
  (fresh-eyes reviewer, then two plan-quality rounds) are already spent.
- **Every other design line takes ×0.2.** The plan fixes files, signatures and
  test strategies per task, so the remaining design cost is reading rather than
  deciding.
- **`scope-pivot` ×1 at full design (×1.0).** Budgeted prospectively, not
  retrospectively: this plan has *already* pivoted once during authoring — M2
  collapsed from a `ResumeMode` threaded through Pair's launch flow to "couch
  declines to send a flag" after the operator asked why it was so complex — and
  M3 is explicitly written against a population that does not exist until M2's
  re-measurement. A design with that profile discovers a fourth thing in flight.
- **Design buffer +15%, not +30%** (v2.1 Step 6): thorough plan doc, and +30%
  on top of a ×0.2 discount double-counts the same thoroughness.
- **`impl=` values are already v3.1-scaled** to 40% of the v2/v2.1 table.

**Known risks to this estimate:**

- **M1's refactor is the widest surface.** `ProjectActionableThreads`'s
  signature change reaches 36 call sites across six files, including one
  (`artifactpath/deadsymbols_test.go`) outside the couch packages. One
  `cross-cutting-refactor` line at 0.20 impl is the least defensible number
  here; pair#155 is the precedent where a refactor of that shape ran ~1.8x its
  estimate.
- **M3 is a sketch by construction**, so its four lines are budgeted from the
  primitive table rather than from decomposed steps. If the estimate misses low,
  that is the second place.
- **Direction: more likely low than high.** Two of the three milestones repair
  code whose failure modes were only discovered by measuring the operator's live
  store, and the same measuring is what M2 and M3 both gate on.

## Plan

Plan doc to follow. Three review boundaries:

- [ ] M1 -- one total inventory. State becomes total, the shell stops
      filtering, reasons render, detached gets its own label. No behaviour
      change to what is actionable; the nine hidden threads simply become
      visible with their reasons.
- [ ] M2 -- get back in. pair#179 (warm reattach across the pair create
      boundary + drop the cold gate from the warm path) and pair#168 (stop
      shadowing the binding; fall back to prior binding, then the legacy row's
      session id). Re-measure the store and report what recovered.
- [ ] M3 -- archive as the only exit. pair#180's retirement rule against the
      post-repair population, operator-invoked.

## Revisions

### 2026-09-03 — nine reasons, not six; and two findings from the plan review

Reason: writing the plan and reviewing it against the source widened the
vocabulary and turned up two things the Spec had not accounted for.

Delta:

- **The reason vocabulary is nine, not six.** The Spec table listed the reasons
  with an exit; three more are needed to keep classification total:
  `path-missing` (an unphysicalizable working path -- today dropped at
  `actionableinventory.go:245-247`, and it must stay a refusal because
  `SelectUniqueResumableRoot` compares paths by exact string),
  `profile-missing` and `agent-unsupported`. A tenth, `unrecorded-child`,
  covers the opposite direction of the incarnation disagreement: a hosted child
  for a record with no incarnation. Done-when now iterates
  `AllThreadReasons()` instead of naming a count.
- **Warm reattach has a hazard the Spec did not know about.** Pair's attach
  branch deletes the live zellij session on a layout conflict
  (`createflow.go:250-268`, `rt.DeleteSession`) -- destroying the running agent
  warm reattach exists to preserve, from a couch-spawned non-interactive `pair`
  whose confirmation prompt no human sees. Under `ResumeRequired` it must
  refuse. The same branch also skips `RegisterExistingCouchThread`, so warm
  reattach would be fail-open on couch's address authority. Both are now
  explicit tasks.
- **pair#168 lives in `sessionledger.CurrentLaunch`** (`record.go:499-540`),
  not in `sessioninventory/query.go`: the highest-ordinal `launch` wins and
  only bindings with a matching `LaunchOrdinal` are accepted, so a trailing
  launch orphans every earlier binding. The query layer only reads the result.

## Log

### 2026-09-03

- Filed to hold one design across pair#168, #171, #179 and #180 after the
  operator named the rule: everything in the directory is displayed, and
  whatever cannot be displayed is archived rather than dropped.
