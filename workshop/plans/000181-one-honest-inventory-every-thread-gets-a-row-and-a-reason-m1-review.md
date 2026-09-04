# Boundary Review — pair#181 (milestone M1)

| field | value |
|-------|-------|
| issue | 181 — One honest inventory: every thread gets a row and a reason |
| repo | pair |
| issue file | workshop/issues/000181-one-honest-inventory-every-thread-gets-a-row-and-a-reason.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 88fe1de011b4c6be58e5a8b20eed89dfa4000f5d..5166a2b96d3ed732ac4df5e1d8c046c39087a76e |
| command | sdlc milestone-close --issue 181 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-09-03T18:35:37-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M1 delivers its thesis: `ClassifyThread` is a genuinely total, genuinely pure rule; the IO shell (`gatherThreadEvidence`) gathers and decides nothing; `ProjectActionableThreads` emits one row per manifest record, pinned by an identity test over a real store; `ProofStatus` correctly separates "we asked and the answer was no" from "we could not ask", which is the design that keeps a transient zellij failure from handing M3's retirement a `session-gone` to act on. I verified the two load-bearing claims by mutation rather than by commit message: removing the `nativeID` clean-resolution gate turns `TestStartInteractiveSkipsDetachedRowsWithoutAResumableBinding` red, and adding a tenth reason to `AllThreadReasons()` turns both exhaustiveness guards red (the `label == string(reason)` clause is what makes them bite — that is a well-built guard). What blocks a clean SHIP is that the reason vocabulary is not yet honest for two healthy shapes: a record mid-`start` (a `creating` incarnation) and a record hosted by a *different* couch console both classify `stale-incarnation` and render "stale — couch exited unexpectedly", which is a false diagnosis on the most common path there is; plus the shell-side `ProofStatus` behaviour — the safety property the whole design exists for — is pinned by no test at all, and the atlas edit left a sentence that states the opposite of what the milestone shipped.

## 1. Strengths

- `ClassifyThread` (`cmd/internal/couchcore/actionableinventory.go:189`) is a real pure core: the branch-order comment explains *why* live precedes the resume-shaped refusals, and `TestClassifyThreadDoesNotApplyResumeShapedRefusalsToALiveRow` (`classify_test.go:230`) pins exactly that. `TestClassifyThreadAcceptsExactlyWhatTheOldProjectorAccepted` is a proper characterization test, not a restatement of the new code.
- The `ProofStatus` / `ReasonUnknown` design (`actionableinventory.go:74-81`, `:230`) is the right answer to plan-gate PQ-1, and the shell honours it in both degraded paths — a failed `DetachedSessions` *and* a couch with no `DetachedSessionResolver` both leave candidates unresolved (`:412-430`).
- `TestEveryReasonRendersADistinctLabel` / `TestEveryReasonExplainsItselfOnEnter` really are exhaustiveness guards. I added a `ReasonTranscriptMissing` to `AllThreadReasons()` in a scratch tree and both failed — the `label == string(reason)` fallback check is what saves them from being vacuous.
- `TestEvidencePassAsksOnlyAboutResumeShapedRecords` (`classify_test.go:452`) replaces the mis-cited `BenchmarkMenu100` with a guard that can actually observe per-record cost, using test-local counting wrappers rather than polluting the shared fakes. Good call on the wrapper placement.
- The implementation closed open plan-gate finding **PQ-6** even though the plan never did: `menuThreadActionable` (`menu.go:964`) covers `ThreadBusy` as well as `ThreadUnusable`, so a park-in-flight row no longer offers `resume`/`switch`.

## 2. Critical findings

None.

## 3. Important findings

**I1 — `stale-incarnation` is the catch-all for three causes, two of them healthy** (`cmd/internal/couchcore/actionableinventory.go:199-204`)

`ClassifyThread` returns `ReasonStaleIncarnation` for *any* record with an incarnation that `evidence.Live` does not match. Verified by probe against HEAD: a record with a `creating` incarnation and a valid start claim classifies `("unusable", "stale-incarnation")` — that is the normal state of **every thread between `start` and Pair registration**, and it renders "stale — couch exited unexpectedly" with an Enter notice telling the operator to reconcile pair#171. Second instance: the switcher's live proof is only *this console's* pty children (`run.go:416`, `startup.go:52` pass the console's own observations), so a thread hosted by another couch console — which the measured store makes likely, 13 records across five repo scopes — also reads `stale-incarnation`. `couch --list` will simultaneously call the same thread `live`, because it derives liveness from `ObserveRecordedProcesses`. That is one store telling two stories, which is the defect the milestone claims to close.

Fix sketch: split the class. (a) An incarnation whose `State != IncarnationLive` and which carries a `Start` claim is a start in flight — give it `ThreadBusy` (or a `starting` sibling), not a reason. (b) Union `ObserveRecordedProcesses` into `gatherThreadEvidence`'s live evidence so both views answer from the same proof, or add a distinct reason for "recorded live, process alive, not hosted here". Not a safety risk today (`stale-incarnation` is never-retire and never `Resumable()`), but M2 and M3 are both written over this vocabulary.

**I2 — the shell's `ProofStatus` behaviour is pinned by no test** (`cmd/internal/couchcore/actionableinventory.go:412-430`)

Every `ProofStatus` assertion lives at the pure layer (`classify_test.go:143,163`). I changed `if detachErr == nil {` to `if detachErr != nil || detachErr == nil {` in a scratch copy of HEAD and the entire suite produced zero additional failures (baseline failures only: the git-dependent `TestIssue151M3*` plan-contract tests and the sandboxed pty tests). The same holds for the no-resolver branch that commit `a9e41584` claims to have fixed. The bug this design exists to prevent — a transient subprocess failure asserting `session-gone`, which M3's `DecideRetirement` acts on — can be reintroduced without a single test going red.

Fix sketch: two tests over `ActionableThreadInventoryContext` with a real store holding one detached candidate — a fake whose `DetachedSessions` returns an error, and a `Couch` whose `Artifacts` implements `NativeBindingResolver` but not `DetachedSessionResolver`. Both must classify `unusable/unknown`, never `session-gone`.

**I3 — `atlas/couch.md:295` still asserts the invariant M1 reverses** (`atlas/couch.md:295-296`)

The edit inserted the new paragraph but left the previous sentence's opening clause as context: the file now reads "The ordinary row state is deliberately only `live` or `parked`; unsupported," immediately followed by "**The projection is TOTAL**". A dangling fragment that states the opposite of the paragraph under it. This is precisely the class Task 6 Step 1 caught in README (and fixed, correctly, along with its pinned test string) — the sweep stopped at one of two documents (ARCH-PURPOSE: the instance, not the class).

Fix sketch: delete the orphaned clause; check the paragraph reads as one statement.

**I4 — the plan's Core-concepts table claims a field the code does not add**

The plan says `ActionableThreadSummary` "gains `Reason ThreadReason` **and `Incarnations []ThreadIncarnation`**, the diagnostic detail `couch --list` already prints, so both views can share one population (Task 5)". The code adds only `Reason` (`actionableinventory.go:113`) and instead gives `ThreadSummary` its own `State`/`Reason` (`threadinventory.go:19-21`). The shipped design is better — it keeps the switcher row free of diagnostic detail while still sharing one classifier — but the plan now describes something the tree does not contain, and M3 is written against that table.

Fix sketch: a `## Revisions` entry recording the change of approach and why.

**I5 — Task 6 Step 3 is ticked, but the measurement it demands is not in the issue Log**

The plan step reads "Record the actual counts in the issue Log" and is checked `[x]`. `workshop/issues/000181-…md` `## Log` contains only the 2026-09-03 filing entry. The real-store measurement (13 rows: 2 live, 1 stale-incarnation at pid 67382, 1 detached, 1 parked, 8 binding-lost) exists only in commit `0aab857a`'s message, and the issue's own Done-when asks for it recorded. AGENTS.md §5 wants the boundary evidence in `## Log` too.

Fix sketch: add the 2026-09-03 M1 Log entry with the measured counts, the `pair-couch-24` binding-lost-vs-session-gone distinction, and the review verdict.

**I6 — the third consumer of the reason vocabulary has no exhaustiveness guard** (`cmd/internal/couchcmd/run.go:610-622`)

`threadStateText` switches on state and falls through to `"unusable: " + string(reason)` — raw slug, no operator wording, and nothing iterates `AllThreadReasons()` in `couchcmd`. Both couchtty renderers are guarded (`menu_render_test.go:416`, `menu_test.go:1261`); this one is not. So `couch --list` says `unusable: binding-lost` where the switcher says `binding lost — repairable`, and a reason added in M3 will ship here unstyled and unnoticed. The rule ("every renderer of `ThreadReason` iterates the vocabulary") was written down and applied to two of three sites.

Fix sketch: either add the same iterate-and-assert guard in `couchcmd`, or have `threadStateText` delegate to a shared exported label helper so there is only one switch to guard.

## 4. Minor findings

- `ThreadSummary.Live()` (`threadinventory.go:41`) still derives liveness from incarnations, so `renderThreadRows` dims by one source (`run.go:578`) and labels by another (`threadStateText`): a `stale-incarnation` row prints undimmed above "unusable: stale-incarnation" (ARCH-DRY).
- `couch --show` now runs a full-store `gatherThreadEvidence` — binding resolution per resume-shaped record plus a zellij snapshot (~1.5s, per the plan's own `--list` measurement) — to display one thread (`operationdispatch.go:135-144`). The plan declared that cost for `--list` only.
- `--show` projects `matches` from `ResolveThreadReference`'s own snapshot, so its `WorkingPath` is not physicalized while `--list`'s is — the same field printed two ways by the two views the milestone unified.
- `ThreadInventoryContext` (`threadinventory.go:88`) reads `Threads.Snapshot()` twice per call (once for `preview`, once inside `gatherThreadEvidence`).
- `ReasonAgentUnsupported`'s value is `"unsupported-agent"` while the Spec, the plan and `atlas/couch.md`'s vocabulary list all say `agent-unsupported`. The rename is right (artifact-family token collision, per commit `06b1a2a4`) but is recorded only in that commit message.
- Task 4 asked for a decision on `hiddenThreadNotice` "with the diff in hand, said in the commit message". It was left unchanged (`menu.go:1230`) and no commit mentions it.
- `reduceOperationResult`'s `stillActionable` (`menu.go:1244`) now means "still present in the inventory", since unusable rows are found too.

## 5. Test coverage notes

- The `ClassifyThread` table (`classify_test.go:27-196`) is strong but has no case for an incarnation in `creating` or `unknown` state, nor for a `VerifiedPark` with a resume-occupied incarnation — the shapes behind I1. `TestClassifyThreadAcceptsExactlyWhatTheOldProjectorAccepted` is only as good as `everyThreadShape`, and those rows are absent from it.
- `TestBothInventoriesReportTheSamePopulationAndStates` (`classify_test.go:503`) passes `nil` observations to the switcher, so both views take the same degenerate path. It cannot see the production divergence in I1. Feed the switcher console observations and `--list` its `ProcOps` proof over one store where a thread is genuinely hosted.
- Converting the pre-#181 tests via the `actionableRows()` shim (`classify_test.go:385`) preserves the accepting-set characterization, which is the more valuable half — but it means those cases no longer assert *which* reason the now-visible rows carry, contrary to Task 2 Step 4's "convert each to assert the row's reason". Acceptable given `classify_test.go`'s coverage; worth knowing when reading them.
- Environment note: `TestPtyRunner*`, `TestBlockedRunnerCancellationConformance`, `TestNotificationPTYConformance`, `TestInteractiveLaunch*` and `TestConsoleRunnerDetectsARealPTY` fail here with "operation not permitted" — a sandbox restriction on pty children, not this diff. `go vet ./cmd/...` is clean and all #181-touched tests pass.

## 6. Architectural notes for upcoming work

- **ARCH-DRY — flag.** `ThreadSummary.Live()` vs `State` (Minor), and three parallel reason switches with only two guarded (I6).
- **ARCH-PURE — pass.** The strongest part of the milestone. `ClassifyThread` runs with zero IO; the shell is thin, injected, and its cost contract is now observable.
- **ARCH-PURPOSE — flag.** Shadow-sweep of `AllThreadReasons()`: `unusableReasonText` derives ✓, `unusableThreadNotice` derives ✓, `threadStateText` does not (I6), `atlas/couch.md` restates the ten reasons by hand and the restatement now contradicts itself (I3).
- **ARCH-MOCK — pass, with a gap.** `FakeThreadArtifactCollisionChecker` and `FakePathOps` sit behind the same seams production uses, and the counting wrappers compose over them rather than replacing them. But the fake's *failure* modes are never exercised end-to-end (I2) — a stateful fake that can only be driven down its happy path is half a fake.
- **ARCH-CONSTRAINTS — pass, with a note.** The call-count guard is a real envelope check and correctly replaces the benchmark that could not see the inventory. `--show`'s undeclared full-store fan-out is the outstanding item.
- **ARCH-SECURE — pass.** Persisted records are validated at the boundary and degrade visibly to `invalid`; unresolved external proof degrades to `unknown` rather than a fabricated negative, which is exactly the right failure shape. Untested at the seam (I2), which is why I2 is Important rather than Minor.
- **For M2/M3:** M3's `DecideRetirement` reads this vocabulary, so I1 matters more there than here — if `stale-incarnation` keeps absorbing healthy shapes, the never-retire list has to stay conservative for the wrong reason. Fixing the class now keeps M3's rule narrow and honest.

## 7. Plan revision recommendations

A single `## Revisions` entry on `workshop/plans/000181-…-plan.md` covering:

- **Core concepts table.** Drop `Incarnations` from the `ActionableThreadSummary` row (I4); add the entities that actually landed and are not listed: `ProofStatus`, `gatherThreadEvidence` (integration), `ObserveRecordedProcesses` (integration), `ThreadInventoryContext` (integration), `BuildThreadInventory` (modified — new `evidence` parameter), `ThreadSummary.State`/`.Reason` (modified, `Parked` deleted), `unusableReasonText`, `menuThreadActionable`, `unusableThreadNotice`, `threadStateText`.
- **Reason value rename.** `agent-unsupported` → `unsupported-agent`, with the artifact-family-token reason, in the plan, the issue Spec Revisions table and `atlas/couch.md`.
- **PQ-7, still open after three rounds.** The plan (`:464`) and issue (`:161`, `:192`) say "36 call sites across six files". Measured at `c761b3cd`: 62 references across **14** files. Replace the number or delete the claim — it is the one line in the estimate's risk section that a reader could check.
- **Task 6 Step 3.** Either record the measurement in the issue `## Log` (preferred, I5) or un-tick the step.
