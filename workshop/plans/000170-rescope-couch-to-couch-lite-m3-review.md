# Boundary Review — pair#170 (milestone M3)

| field | value |
|-------|-------|
| issue | 170 — Rescope couch to couch-lite |
| repo | pair |
| issue file | workshop/issues/000170-rescope-couch-to-couch-lite.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | 9f7d42453898a7768db3fdf81c9dbfb0e14129ae..23e17720db829359d04e14b9ae0f1d2ddc936406 |
| command | sdlc milestone-close --issue 170 --milestone M3 |
| reviewer | claude |
| timestamp | 2026-09-02T17:00:01-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

M3 does what it says at the level it was designed: `SelectUniqueParkedRoot` becomes `SelectUniqueResumableRoot`, the widening is one predicate, exactness is preserved rather than quietly traded for a warm-over-cold ranking, and the rename is recorded as a `## Revisions` entry on #167's archived plan instead of an edit. I mutation-verified the selector (narrowing it back to parked-only reddens four subtests plus the inventory test) and probe-verified both atlas claims about states that are *never* selected. What blocks the boundary is a state the widening newly reaches: `ActionableThreadInventoryContext` binding-filters **parked** candidates through `ResolveEstablished` but not **detached** ones, while `DecideResume`/`ResumeContext` require an established binding for both — so a detached row with a degraded native binding is listed, auto-selected at startup, and refused, and `couch` in that directory exits 1 with no way through (`couch [path] | --list | --show` has no "start new" escape). I reproduced it: `StartInteractive` returns `resume-binding-unbound` and starts nothing. Before M3 the same store state started a new thread. Secondary: no test in `couchcore` pins `StartInteractive`'s detached wiring (mutating the call site to filter parked-only leaves the whole package green — only the pty-gated `couchcmd` acceptance test covers it, and it cannot execute in a pty-less environment), and the durable record misattributes the path-physicalization fix to M3 when it shipped at M2.

## 1. Strengths

- **The rename was treated as the deliverable.** `startup.go:9-24` states the rule, the exclusion of live rows, and the *absence* of a ranking policy; `workshop/history/plans/000167-…-plan.md` gets a Revisions entry rather than a body edit, so #167 keeps saying what it delivered.
- **Exactness held under pressure.** `startup_test.go:41-46` pins "one parked and one detached at the same path is ambiguous" with distinct addresses — the case where warm-over-cold would have been the tempting, wrong answer.
- **The negative twin exists.** `run_test.go:377` (`TestInteractiveLaunchStartsNewWhenNoSessionSurvives`) is the test that catches "startup reattached something it could not prove", and it is the more valuable of the two acceptance tests.
- **The M2 review's explicit carry-forward was honoured.** `TestActionableInventoryPhysicalizesDetachedRowsLikeParkedOnes` (`actionableinventory_test.go:512`) pins the exact-string path dependency the M2 comment called load-bearing and nothing asserted.
- **`Resumable()` is consumed, not re-enumerated** (`startup.go:29`) — one predicate shared with `menu.go:389` (ARCH-DRY), and the stale "proportional to detached threads" comment at `actionableinventory.go:208` was swept to match the corrected M2 cost model.
- Atlas claims verified by probe, not by reading: a stale `IncarnationLive` yields zero rows and startup creates a new thread; an attached-elsewhere session yields no detached observation.

## 2. Critical findings

**C1 — a detached row is offered and auto-selected without the native-binding proof resume requires; `couch` then cannot start in that tree** (`cmd/internal/couchcore/actionableinventory.go:238`, consumed at `startup.go:56`).

In the candidate loop, the parked branch gates on the binding — `resolver.ResolveEstablished(...)`, then `if resolveErr != nil || bindingResumeDiagnostic(binding) != "" { continue }` (`:243-246`). The detached branch appends the address and `continue`s at `:238` **before** any binding is resolved (before even the `resolver == nil` check). But `ResumeContext` calls `bindings.ResolveEstablished` unconditionally and returns its error (`resume.go:212-215`), and `DecideResume` refuses on `bindingResumeDiagnostic` for detached rows exactly as for parked ones (`resume.go:122`). So "listed ⇒ resumable" — the invariant that makes #167's deliberate *no-fallback* startup rule safe — holds for parked rows and not for detached ones.

Reproduced against HEAD (scratch copy, `newTestEnv` + `SetDetachedSession`, no `SetNativeBinding`):

```
rows = [{... State:detached ...}]
StartInteractive err = resume-binding-unbound: native session binding is not one exact established root
start.Record.Thread = {RepoScope: Tag:}     runner ops = []
```

`couchcmd/run.go:285-287` renders that and returns 1. Reachable non-established states for a thread whose session survived: `current.Conflict` → `BindingAmbiguous`, `Binding == nil` or a failed `ValidateBindingProof` → `BindingProvisional` (`sessioninventory/query.go:114-160`) — i.e. the agent's native session data was pruned, rotated, or raced. M2's decision #3 (leave detaches everything) makes *detached* the normal resting state, so this is the ordinary row at the operator's own path, and the only recoveries are `pair resume <tag>` outside couch or running `couch` from a sibling subdirectory. Before M3 the parked-only selector skipped the row and startup created a new thread.

Fix sketch — extend the candidate gate to the class rather than the parked instance (`ARCH-PURPOSE`), which is also what `actionableThreadState`'s own comment demands ("a row that cannot work must not be offered", `actionableinventory.go:154-158`):

```go
if record.VerifiedPark == nil {
    if resolver == nil { continue }
    binding, resolveErr := resolver.ResolveEstablished(ctx, record.Address.RepoScope, string(record.Address.Tag), agent)
    if resolveErr != nil || bindingResumeDiagnostic(binding) != "" { continue }
    detachedCandidates = append(detachedCandidates, record.Address)
    continue
}
```
(The alternative — dropping the binding requirement in `DecideResume` when `Detached`, since `pair resume <tag>` reattaches a surviving session without the native id, per `README.md:511` — is more principled but reopens M2 surface and `RequiredSessionID`/`RequireNativeResumeBinding`. Take it only if the switcher must keep showing such rows.) Pin it with a `couchcore`-level test: a detached row with an unbound binding must not kill startup.

## 3. Important findings

**I1 — `StartInteractive`'s detached wiring is pinned by nothing that runs without a pty** (`cmd/internal/couchcore/startup.go:56`). I replaced the call site with a parked-only pre-filter in a scratch copy and ran the full `couchcore` suite: no new failure. Only `TestInteractiveLaunchReattachesUniqueDetachedRoot` (`couchcmd/run_test.go:315`) covers it, and it hard-fails at `pty.Open()` in any environment without pty access — I could not execute either acceptance test, so the commit's mutation-verification claim for it is unconfirmed here. Fix: a ~15-line `TestStartInteractiveResumesUniqueDetachedRoot` twin of `TestStartInteractiveResumesUniqueExactParkedRoot` (`startup_test.go:74`) using the existing `newTestEnv` fakes — the same test that would have caught C1.

**I2 — the record credits M3 with a fix that shipped at M2.** Plan Step 3b (`plan.md:411`) is ticked as "Extend the physicalization to detached candidates"; `workshop/projects/couch.md:921-926` and the commit message both narrate it as M3's subtle discovery ("M2 had physicalized only parked candidates"). The code at `actionableinventory.go:224-232` is unchanged in this window — `git log -L` puts it in `fac153c9` (`#170 M2`), and M3's new test passes unmodified against the M2 base production source (verified). What M3 delivered is the *proof*, which is what the M2 review asked for. Fix: restate Step 3b as "pin the M2 physicalization with the alias-path test" and correct the project-file paragraph; one sentence each.

**I3 — the M3 operating-envelope claim is stale and unmeasured (`ARCH-CONSTRAINTS`).** `plan.md:402` says the envelope is "unchanged from `#167` — one target resolution, one local actionable snapshot, O(n) pure selection, no fleet scan, no prompt, no goroutine fan-out". Since M2 that snapshot spawns two `list-sessions` runs plus one `action list-clients` per non-exited pair session **on the host** whenever a detach candidate exists (`launcher/zellij.go:35-58`), each bounded by a 5 s `zellijQueryTimeout` — and M3 makes a detach candidate the normal case at startup. `atlas/couch.md:401-406` frames that cost as a *periodic refresh worker* cost only; it is now also on the path between typing `couch` and seeing anything, worst case (2+N)×5 s. Fix: measure one startup with K detached threads, state the budget, and correct both sentences to name the startup path.

**I4 — README documents the switcher's row states as pre-M2** (`README.md:360-361`): "Rows expose only proven `live` and exact verified `parked` states" — detached rows have been listed since M2 and reattaching them is M3's Done-when. `README.md:308` has the same residue: "automatic unique resume reuses the parked thread instead", one paragraph below the sentence M2 correctly widened to "the sole exact resumable thread". Two words; fix both.

## 4. Minor findings

- The selector's rationale is restated near-verbatim in five places (`startup.go:11-24`, `startup_test.go:13-19`, `atlas/couch.md:439-446`, `workshop/projects/couch.md:908-919`, `workshop/history/plans/000167-…:125-144`). Correct today; five copies to keep in sync (`ARCH-DRY` at the prose level).
- `startup_test.go:37-38` — the "ambiguous parked rows" / "ambiguous detached rows" cases pass the same `ThreadAddress` twice, a shape `ProjectActionableThreads` can never emit (one row per record). The realistic distinct-address case exists two rows below, so this is fixture realism only.
- `atlas/couch.md:451` uses an em-dash-and-hyphen mix (`—` at :451, `--` at :447) inside one paragraph; the file is otherwise consistent per section.

## 5. Test coverage notes

- **Verified red:** narrowing `SelectUniqueResumableRoot` to `State == ThreadParked` reddens `TestSelectUniqueResumableRoot/{one exact detached row, one parked and one detached…, one resumable among nonmatches}` and `TestActionableInventoryPhysicalizesDetachedRowsLikeParkedOnes`. The pure selector is genuinely pinned.
- **Verified green under mutation (gap):** filtering `rows` to parked before the selector call in `StartInteractive` leaves the entire `couchcore` package green — I1.
- **Not executable here:** `TestInteractiveLaunchReattachesUniqueDetachedRoot` / `…StartsNewWhenNoSessionSurvives` fail at `pty.Open()` under this environment's restrictions, as do the pre-existing `ptyrunner`/`TestNotificationPTYConformance` cases. Everything else in `couchcore`, `couchtty`, `launcher`, `sessioninventory` passes at HEAD; `go build ./...`, `GOOS=linux go build ./...` and `go vet` are clean.
- **Uncovered:** the C1 state (detached row, non-established binding) and, more generally, any detached row that the projector admits but `DecideResume` refuses. One table-driven test at `StartInteractive` level covers both plus I1.
- `TestStartInteractiveResumeRefusalDoesNotCreateFallbackRoot` still asserts the no-fallback rule for parked rows — keep it; C1's fix should make the *listed* set match the resumable set rather than weaken that rule.

## 6. Architectural notes

- **ARCH-DRY — pass** (Minor only). One selector, not two; `Resumable()` consumed rather than re-enumerated; the corrected cost comment swept to match M2's plan/atlas correction. The five-way prose duplication is the only nit.
- **ARCH-PURE — pass.** `SelectUniqueResumableRoot` is a pure fold over projected rows, tested with no fakes and no IO; `StartInteractive` remains the thin shell and gained no new seam.
- **ARCH-PURPOSE — flag (C1, I2).** C1 is the class/instance failure in its clearest form: the candidate loop's binding gate was written for the parked branch and the detached sibling — added later, now load-bearing at startup — was never brought under it. I2 is the record claiming an instance was fixed here when it was fixed a milestone ago.
- **ARCH-MOCK — pass.** The detached observation crosses the `DetachedSessionResolver` seam in both production and test flow; M2's I3 was delivered (`launcher/session_quiescence_live_test.go:155` `TestSessionDetachLive` conformance-checks the one external behaviour detach rests on).
- **ARCH-CONSTRAINTS — flag (I3).** The startup path inherited an untimed-per-call-but-5 s-bounded 2+N subprocess fan-out and the plan still describes it as "one local actionable snapshot". No startup measurement exists at any milestone.
- **ARCH-SECURE — pass.** M3 adds no new input parsing, no new subprocess argv, no credential path. Session names still reach `exec.Command` as argv elements; the widened selector consumes already-validated projections.
- **For M4:** the plan already flags that `CommitStartClaim` must carry the widened detached precondition forward. Add to that list: whatever C1's fix lands as (candidate binding gate, or a `Detached`-aware `DecideResume`) is now on the startup path, so M4's admission deletion must preserve it or startup regresses to the same fail-to-start with a green suite. Also note that no contract test covers #170's **couchcore** Core-concepts rows (`couchtty/core_concepts_contract_test.go` filters to `cmd/internal/couchtty/`), so M4's much larger `deleted` table will be entirely unenforced — the M2 review named this and the by-construction fix was deliberately backed out; M4 is where it starts to cost.

## 7. Plan revision recommendations

Add to `workshop/plans/000170-rescope-couch-to-couch-lite-plan.md`'s `## Revisions`:

1. **Task 12 Step 3b — restate what M3 delivered.** The physicalization of detached candidates shipped in M2 (`fac153c9`); M3 delivered the alias-path test the M2 review asked for. Record it as "pin the M2 physicalization", and correct the matching paragraph in `workshop/projects/couch.md:921-926`.
2. **Chunk 3 envelope correction.** "Unchanged from `#167` — one local actionable snapshot … no fan-out" is no longer true: since M2 the snapshot costs 2 + N zellij CLI spawns (N = every pair session on the host, 5 s each) whenever a detach candidate exists, and M3 puts that on the interactive startup path. State the measured startup budget and correct `atlas/couch.md:401-406` to name startup alongside the refresh worker.
3. **Chunk 3 — record the listed-vs-resumable invariant explicitly.** Add the rule the widening depends on: a row the projection lists must be one `ResumeContext` can accept, because startup deliberately has no fallback. Name the detached binding gate as the mechanism, and carry it into M4's `CommitStartClaim` note.

```findings
findings:
  - id: new
    severity: Critical
    family: listed-implies-resumable
    title: |
      Detached rows are auto-selected at startup without the binding proof resume requires, so `couch` exits 1 with no way through
    detail: |
      actionableinventory.go:238 appends detached candidates before any ResolveEstablished
      gate, while the parked branch at :243 filters on it and DecideResume/ResumeContext
      require it for both. Reproduced against HEAD: a detached row with an unbound binding
      is listed, selected by SelectUniqueResumableRoot, and refused with
      resume-binding-unbound; couchcmd renders the error and returns 1, and `couch` has no
      new-thread escape hatch. Before M3 the parked-only selector skipped it and startup
      created a new thread.
  - id: new
    severity: Important
    family: seam-untested-at-runnable-level
    title: |
      No couchcore test pins StartInteractive's detached wiring; only the pty-gated couchcmd acceptance test covers it
    detail: |
      Filtering rows to parked before the SelectUniqueResumableRoot call in a scratch copy
      leaves the entire couchcore suite green. The two acceptance tests hard-fail at
      pty.Open() in any environment without pty access, so the commit's mutation claim for
      the reattach test is unconfirmable there. A ~15-line twin of
      TestStartInteractiveResumesUniqueExactParkedRoot closes it and would also catch the
      Critical.
  - id: new
    severity: Important
    family: record-claims-unverified-delivery
    title: |
      Plan Step 3b, the project file and the commit credit M3 with a physicalization fix that shipped at M2
    detail: |
      actionableinventory.go's parked-AND-detached physicalization is unchanged in this
      window; git log -L attributes it to fac153c9 (#170 M2), and M3's new test passes
      against the M2 base production source. M3 delivered the proof, not the fix. Correct
      the step wording and workshop/projects/couch.md:921-926.
  - id: new
    severity: Important
    family: envelope-claim-unmeasured
    title: |
      M3's stated startup envelope contradicts the M2-corrected cost model and is unmeasured
    detail: |
      plan.md:402 says "unchanged from #167 -- one local actionable snapshot ... no
      fan-out", but the snapshot spawns 2 + N zellij CLI processes (N = every pair session
      on the host, 5s timeout each) whenever a detach candidate exists, and M3 makes that
      the normal startup case. atlas/couch.md:401-406 frames the cost as refresh-worker
      only. Measure a startup with K detached threads and correct both sentences.
  - id: new
    severity: Important
    family: readme-stale-for-shipped-surface
    title: |
      README still describes the switcher's row states and unique resume as parked-only
    detail: |
      README.md:360 "Rows expose only proven live and exact verified parked states"
      contradicts M2's detached rows, which M3's Done-when depends on; README.md:308
      "automatic unique resume reuses the parked thread instead" is the same residue one
      paragraph below the sentence M2 correctly widened to "the sole exact resumable
      thread".
  - id: new
    severity: Minor
    family: prose-duplication
    title: |
      The selector's rationale is restated near-verbatim in five artifacts
    detail: |
      startup.go:11-24, startup_test.go:13-19, atlas/couch.md:439-446,
      workshop/projects/couch.md:908-919 and the archived #167 plan all carry the same
      three-paragraph argument. Correct today, five copies to keep in sync.
  - id: new
    severity: Minor
    family: fixture-realism
    title: |
      The selector's ambiguity fixtures pass the same ThreadAddress twice, a shape the projector cannot emit
    detail: |
      startup_test.go:37-38 duplicates one row; ProjectActionableThreads emits one row per
      record. The realistic distinct-address case exists two rows below, so this is
      fixture realism only.
```

---

## Re-review — 2026-09-02T17:22:10-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 170 — Rescope couch to couch-lite |
| repo | pair |
| issue file | workshop/issues/000170-rescope-couch-to-couch-lite.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | a89878c31cd7bee06693257e05440c8c4eee7057..a89878c31cd7bee06693257e05440c8c4eee7057 |
| command | sdlc milestone-close --issue 170 --milestone M3 |
| reviewer | claude |
| timestamp | 2026-09-02T17:22:10-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The Critical from round 3 is genuinely fixed and I proved it both ways: the binding gate now sits ahead of the detached-candidate append (`actionableinventory.go:236-256`), and reverting that reorder in a scratch copy reddens all three non-established subtests of the new `TestStartInteractiveSkipsDetachedRowsWithoutAResumableBinding`. I also ran the class sweep behind it — every `DecideResume` refusal reason is now either filtered in the candidate loop (park transaction, occupied incarnation, missing/unsupported profile, binding) or unreachable from startup selection by construction (`ResumePathMissing` can only fire for a path the selector matched against the operator's own resolved cwd), so "listed ⇒ resumable" actually holds rather than holding for the one state the finding named. What does not block SHIP but is not done: BR-2's test gap survives verbatim — I re-ran its mutation (filter `rows` to `ThreadParked` before the `SelectUniqueResumableRoot` call in `startup.go:53`) and the entire `couchcore` suite is still green, because the new test stops at `ActionableThreadInventoryContext` + selector rather than calling `StartInteractive`; and BR-4 was only half-corrected — the plan sentence was fixed and the cost measured, but `atlas/couch.md:403-406` still frames the 1.49 s snapshot as refresh-worker-only. Both are cheap; neither is a correctness risk at the boundary.

## 1. Strengths

- **The fix is the class, not the site.** One gate covering both resumable kinds (`actionableinventory.go:236-256`) rather than a detached-specific check, and the comment states the invariant it defends instead of describing the mechanics. Mutation-verified in both directions here.
- **The new test reproduces all three degraded binding states** (`startup_test.go:197-251`) — unbound, ambiguous, provisional — not just the one that was reported. That is the enumeration ARCH-PURPOSE asks for at the test level.
- **`TestInteractiveLaunchStartsNewWhenNoSessionSurvives` (`run_test.go:377`)** is the more valuable of the two acceptance tests: it pins "startup must not reattach something it cannot prove", which is the exact failure mode the milestone's widening creates.
- **`Resumable()` is consumed, not re-enumerated** (`actionableinventory.go:75`, used at `startup.go:29` and `menu.go`) — ARCH-DRY held under a widening that invited a second predicate.
- **BR-3's misattribution correction is honest.** Plan Step 3b and `workshop/projects/couch.md:926-934` now say M3 delivered the proof and M2 the fix, with the reason stated. `git log -L` on the physicalization block agrees.
- **The envelope was measured rather than argued** (`BenchmarkZellijSnapshotLive`, issue Log 2026-09-02), and the 1.4 s was filed as #172 with a bounded-concurrency spec instead of being absorbed into M3.

## 2. Critical findings

None. BR-1 is fixed and pinned.

## 3. Important findings

**BR-2 (re-raised, not-addressed) — `StartInteractive`'s detached wiring is still unpinned at the runnable level.** `startup_test.go:197` is named for `StartInteractive` but never calls it; it calls `ActionableThreadInventoryContext` then `SelectUniqueResumableRoot` directly, replicating the two lines under test. I applied BR-2's exact mutation to `startup.go:53` and the full `couchcore` package stayed green. The only tests that catch it are `run_test.go:322/377`, which fail at `pty.Open()` in this environment (`run_test.go:354: operation not permitted`) — so `workshop/projects/couch.md:944-946`'s "the reattach one is mutation-verified" remains unconfirmable from a pty-less shell, exactly as the prior round said. Fix sketch: in `TestStartInteractiveSkipsDetachedRowsWithoutAResumableBinding`, replace the two-line replication with `env.Couch.StartInteractive(ctx, StartArgs{Cwd: "/repo"})` and assert `start.Record.Thread` — that closes the seam and the binding gate in one test.

**BR-4 (re-raised, not-addressed) — the atlas half of the envelope correction did not land, and no startup budget was stated.** `atlas/couch.md:403-406` still reads "Since this now runs on the periodic refresh worker, each query carries `zellijQueryTimeout`" — it does not say M3 put the same snapshot on the *blocking* startup path, which is the only reason the measurement mattered. (It also says "periodic", while the issue Log says refreshes are event-driven with no ticker.) Separately: the measured 1.49 s typical and the plan's own stated `(2+N)×5 s` worst case are now on the path between typing `couch` and the first frame, with no declared startup budget, no progress indication and no skip — ARCH-CONSTRAINTS asks for the bound and the bounded behavior when exceeded, not only the measurement. Deferring the *speedup* to #172 is right; declaring the envelope is this boundary's job.

**New — the inventory's candidate rule is restated in three hand-maintained places and only one was swept.** *This is the 2nd finding in family `readme-stale-for-shipped-surface`.* Per the escalation rule I am not asking for the instances to be patched; here is the rule that covers them: **the inventory's admission predicate has no single source — README, `atlas/couch.md` and the plan each restate it in prose, so any change to the candidate rule must sweep the enumerated set in the same commit, and the enumeration belongs in the plan.** Measured prevalence this round: README.md:361-364 was updated (correct), `atlas/couch.md:399-401` still says candidates are "(no incarnation, no verified park, a saved profile)" with no mention of the established-binding requirement that is now the invariant startup's safety rests on, and `workshop/plans/…-plan.md:258` still says candidates are those three things *and* that bounding the query to them "keeps the refresh cost proportional to detached threads" — the precise claim the M3 code comment at `actionableinventory.go:208-210` was corrected to deny. Two of three restatements drifted from one commit's change; that is the rule failing, not two typos.

## 4. Minor findings

- **BR-6 (not-addressed)** — the selector's three-paragraph rationale is still verbatim in `startup.go:9-24`, `startup_test.go:13-19`, `atlas/couch.md:445-451` and `workshop/projects/couch.md:908-925`.
- **BR-7 (not-addressed)** — `startup_test.go:38-39` still passes the same `ThreadAddress` twice for both ambiguity cases, a shape `ProjectActionableThreads` cannot emit (one row per record).
- **New (family `listed-implies-resumable`, 2nd finding)** — the parked invariant is enforced *inside* the pure projector (`parkedResumeProofMatches` requires `NativeID != ""`), while the detached twin is enforced only in the IO caller's loop; yet `actionableThreadState`'s own comment at `actionableinventory.go:155-156` says it "fails closed on its own, so it does not rely on the caller having filtered candidates." That is now true of the profile checks and false of the binding. Rule, per the escalation protocol: **the evidence a row's `Enter` needs must travel in the observation type the pure projector consumes, so the projector enforces it — a gate that lives only in the IO shell is one refactor or one second caller away from reintroducing BR-1.** The class fix is to give `DetachedSessionObservation` a `NativeID` and require it, mirroring `ParkedResumeObservation`; the loop already holds the binding at that point. No live defect: `ScopedThreadArtifactCollisionChecker` is the only production `Artifacts`, and it is gated.
- `BenchmarkZellijSnapshotLive` measures the real binary but asserts nothing, so it is a measurement harness, not the live conformance check ARCH-MOCK wants for the fake's modeled behavior. Fine for this boundary; worth naming in #172.

## 5. Test coverage notes

- Verified green in this environment: `go build ./...`, `GOOS=linux go build ./...`, `go test ./cmd/internal/couchcore -run 'SelectUnique|StartInteractive|ActionableInventory|Detached|Resume'`, the plan/concept contract tests, `launcher`, `sessioninventory`. The only failures anywhere are `ptychild: operation not permitted` (couchcore pty runner tests, `couchtty` notification-pty, all of `couchcmd/run_test.go`) — environmental, identical shape on untouched tests, not this window's doing.
- Mutation results, run not assumed: reverting the gate reorder → 3 subtests red (BR-1's fix is real); filtering `rows` to parked in `StartInteractive` → suite green (BR-2's gap is real).
- The couchcmd acceptance pair is the right shape (production routing through `dispatchInitialAttach`), but a milestone whose headline behavior is *only* provable where a pty exists has no coverage in any CI or agent context. That is what BR-2 is asking to fix, and it is 15 lines.

## 6. Architectural notes for upcoming work

- ARCH-DRY **pass** (one predicate, one gate) — flagged only for prose duplication (BR-6). ARCH-PURE **pass with the Minor above** — the selector is genuinely pure and table-tested without IO. ARCH-PURPOSE **pass on code, flag on records** — the refusal-reason sweep is complete in the loop; the docs shadow-sweep is not. ARCH-MOCK **pass** — `FakeThreadArtifactCollisionChecker` implements both resolvers behind the same seam as `ScopedThreadArtifactCollisionChecker`, so production and test share the boundary. ARCH-CONSTRAINTS **flag** — see BR-4. ARCH-SECURE **pass** — `bindingResumeDiagnostic`'s `default:` fails closed to unbound, a corrupt manifest errors instead of degrading (`TestStartInteractiveInventoryFailureCreatesNoRoot`), no credential reaches a log or argv.
- Reachability consequence worth carrying to M4/#171: a detached thread whose binding degrades is now hidden from the switcher entirely while its zellij session keeps running an agent. That matches the parked precedent and `ThreadInventory` (the `list` op) still shows it, but it is a new way for a *live* agent to become invisible. The alternative design — list it, refuse its `Enter` with the diagnostic, and gate only startup selection — was not considered in writing; if the operator ever hits it, that is the fork to revisit.
- The marginal cost of the fix is one `ResolveEstablished` (a scoped FS listing, no subprocess) per detached candidate per refresh. Against the parked-era baseline this is a wash — those same records used to pay it as parked candidates — so #172's fan-out remains the only startup-latency term worth optimizing.

## 7. Plan revision recommendations

- Append a `## Revisions` entry `### 2026-09-02 — M3 boundary review`, in the shape the M1 entry already set. The fix commit changed a production admission rule and edited two plan lines *in place* (Step 3b, the envelope paragraph) with inline parentheticals; AGENTS.md §1 asks for an appended timestamped delta instead. The entry should record: the binding gate as an M3 delivery the chunk never described, that the physicalization credit moved to M2, and the measured envelope with #172 filed against it.
- Correct `workshop/plans/…-plan.md:258` (Chunk 2 Core concepts): candidates now also require an established native binding, and "keeps the refresh cost proportional to detached threads" is the claim `actionableinventory.go:208-210` was corrected to deny.
- The M3 Core concepts table itself still matches the code — `SelectUniqueResumableRoot` new at `startup.go`, `SelectUniqueParkedRoot` deleted (no residue outside prose), `Couch.StartInteractive` modified — but `Couch.ActionableThreadInventoryContext` is now modified by M3 too and appears only in M2's table. Add it to M3's Integration points, or the next reader looks for this window's most consequential change in the wrong milestone — the same failure BR-3 named.

---

## Re-review — 2026-09-02T17:36:29-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 170 — Rescope couch to couch-lite |
| repo | pair |
| issue file | workshop/issues/000170-rescope-couch-to-couch-lite.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | a89878c31cd7bee06693257e05440c8c4eee7057..2943799d64dcf4ffa67d413fe6aacf4b757325d4 |
| command | sdlc milestone-close --issue 170 --milestone M3 |
| reviewer | claude |
| timestamp | 2026-09-02T17:36:29-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The two commits in this window do exactly one substantive thing — close BR-2 — and they close it properly. I verified by mutation rather than by reading: applying BR-2's exact mutation (filter `rows` to `ThreadParked` before `SelectUniqueResumableRoot` in `startup.go:53`) in a scratch copy now reddens `TestStartInteractiveResumesUniqueDetachedRoot`, where the previous round's test left the suite green. I also re-verified BR-1's fix from the base commit the same way: reverting the gate reorder in `actionableinventory.go:235-256` reddens all three non-established subtests. BR-3 and BR-5 are genuinely done (plan Step 3b + `workshop/projects/couch.md` credit M2 for the physicalization; README:308 and :361 both rewritten). What blocks a clean SHIP is small and doc-shaped: BR-4 was half-corrected — the plan sentence was fixed and the cost measured, but `atlas/couch.md:403-406` still frames the 1.49 s zellij snapshot as belonging to the refresh worker, and calls that worker "periodic" when there is no ticker anywhere in `cmd/`. Beyond the prior findings, one architectural item is worth the boundary: the binding proof for detached rows is enforced only in the IO shell's candidate loop, while its parked twin is enforced inside the pure projector — and both `actionableThreadState`'s own comment and `DetachedSessionObservation`'s doc currently assert the opposite. None of this is a correctness risk today; the milestone's behavior is right and pinned.

## 1. Strengths

- **BR-2's fix is the real thing, not a plausible-looking one.** `startup_test.go:262-286` calls `env.Couch.StartInteractive` and asserts `start.Record.Thread`, rather than replicating the two lines under test. Mutation-confirmed here in a scratch copy, not taken from the commit message.
- **The negative twin was added unprompted** (`startup_test.go:290-311`): no surviving session ⇒ startup must create a new thread rather than reattach something it cannot prove. That is the failure mode the widening creates, and it traverses the full candidate path (binding resolved, `DetachedSessions` queried, empty answer).
- **BR-1's fix is the class, not the site** — one gate ahead of both appends (`actionableinventory.go:235-256`), with a comment that states the invariant it defends instead of the mechanics.
- **The plan `## Revisions` entry** (`plan.md:748-787`) is an honest appended delta per AGENTS.md §1, and names the delivery that appeared in no task step (the binding gate), the moved credit, and the measured envelope.
- **`Resumable()` is consumed, not re-enumerated** (`actionableinventory.go:73-77`, used at `startup.go:28` and in `menu.go`) — ARCH-DRY held under a widening that invited a second predicate.
- **`FakeThreadArtifactCollisionChecker.DetachedSessions` answers only for addresses asked about**, with the comment explaining why (`artifactcollision_fake.go:78-81`) — the fake would otherwise hide a caller that forgot its candidates. ARCH-MOCK done deliberately.

## 2. Critical findings

None. BR-1 is fixed and mutation-pinned.

## 3. Important findings

**BR-4 (not-addressed) — the atlas half of the envelope correction did not land.** `atlas/couch.md:403-406` still reads "Since this now runs on the periodic refresh worker, each query carries `zellijQueryTimeout`". Two problems: it attributes the cost to the refresh worker alone when M3 put the identical snapshot on the *blocking* startup path — the only reason the 1.49 s measurement mattered — and "periodic" is wrong on its own terms (`grep -rn NewTicker cmd` returns no actionable-refresh ticker; the issue Log says refreshes are event-driven). Adjacent in the same paragraph, `atlas/couch.md:399-401` still enumerates a candidate as "(no incarnation, no verified park, a saved profile)", omitting the established-binding requirement that is now the invariant startup's safety rests on — the gate paragraph 50 lines below states it, so the two now disagree. Fix: one paragraph, three sentences. Also state the startup budget and the bounded behavior when exceeded (worst case `(2+N)×5 s`, mitigated by the timeout, speedup deferred to #172) — ARCH-CONSTRAINTS asks for the bound, not only the measurement.

**New (Important, family `listed-implies-resumable`) — the detached row's binding proof is enforced in the IO shell, not in the pure projector, and two comments claim otherwise.** *This is the 2nd finding in family `listed-implies-resumable`.* Per the escalation protocol I am not asking for this instance to be patched; here is the rule that covers the class:

> **Every proof a row's `Enter` requires must travel to `ProjectActionableThreads` as a field on that row's observation type and be enforced inside `actionableThreadState`. A proof enforced only in `ActionableThreadInventoryContext`'s candidate loop is not part of the row's contract and can be dropped by any refactor or second caller of the exported projector.**

The enumeration the rule implies, swept: `ThreadLive` needs a matching TTY observation — enforced in the projector ✅. `ThreadParked` needs the native binding — enforced in the projector, `parkedResumeProofMatches` requires `observation.NativeID != ""` (`actionableinventory.go:181-187`) ✅. `ThreadDetached` needs a surviving session (`SessionName != ""`, enforced ✅) *and* the native binding — enforced **only** at `actionableinventory.go:250-253` ❌. That asymmetry is precisely how BR-1 shipped. It is also actively mis-documented: `actionableinventory.go:155-158` says "this function's contract is that it fails closed on its own, so it does not rely on the caller having filtered candidates", and `DetachedSessionObservation`'s doc (`actionableinventory.go:44-46`) says "proof arrives as observations, never as persisted lifecycle state". Both are true of the profile checks and false of the binding. Class fix: add `NativeID string` to `DetachedSessionObservation` and require it in `actionableThreadState`'s detached branch, mirroring `ParkedResumeObservation` — the loop already holds `binding` at the append point (`actionableinventory.go:255`), and `ProjectActionableThreads` is exported, so the second caller is a public-API possibility, not a hypothetical. No live defect today: `ScopedThreadArtifactCollisionChecker` is the only production `Artifacts` and it is gated. If the operator judges the field threading not cheap at this boundary, the minimum is correcting the two comments so they stop asserting a contract the code does not hold.

## 4. Minor findings

- **BR-6 (not-addressed)** — the selector rationale is still verbatim in `startup.go:11-24`, `startup_test.go:13-19`, `atlas/couch.md:445-451` and `workshop/projects/couch.md:908-925`.
- **BR-7 (not-addressed)** — `startup_test.go:37-38` still passes the same `ThreadAddress` twice for both ambiguity cases, a shape `ProjectActionableThreads` cannot emit (one row per record). Fixture realism only; the realistic distinct-address case is two rows below.
- **New** — `TestStartInteractiveResumesUniqueDetachedRoot` (`startup_test.go:262-277`) and `TestStartInteractiveStartsNewWhenNoSessionSurvives` (`:290-304`) each hand-rebuild the ~10-line record setup that `seedStartupParked` (`startup_test.go:182-195`) already encapsulates; a `seedStartupResumable(t, env, tag, path, kind)` covers all three sites (ARCH-DRY).
- **New** — `startup_test.go:309` asserts only `start.Record.Thread != stale.Address`, which a zero-valued record would also satisfy. `TestStartInteractiveCreatesNewRootWithoutExactCandidate:109` shows the stronger form (`== (ThreadAddress{}) || Handle == nil`); one extra line closes the vacuous pass.

## 5. Test coverage notes

- Verified in this environment: `go build ./...` ✅, `go vet ./cmd/internal/couchcore` ✅, `go test ./...` — every failure is `ptychild: operation not permitted` / `pty.Open: operation not permitted` (`cmd/couch`, `couchcmd`, `couchcore` pty runners, `couchtty`, `hostty`, `keyscmd`, `ptychild`, terminal mux). Same shape on untouched tests; environmental, not this window's doing. The `fatal error: all goroutines are asleep` in `couchcore` is a cascade of the pty-blocked blocked-runner tests, not a new hang.
- Mutation results, run rather than assumed: filtering `rows` to `ThreadParked` inside `StartInteractive` → `TestStartInteractiveResumesUniqueDetachedRoot` red (BR-2 genuinely closed). Reverting the binding-gate reorder in `actionableinventory.go` → three subtests of `TestStartInteractiveSkipsDetachedRowsWithoutAResumableBinding` red (BR-1 genuinely closed).
- The three `couchcmd` acceptance tests still fail at `pty.Open()` even outside the sandbox, so `workshop/projects/couch.md:942-944`'s "the reattach one is mutation-verified" remains unverifiable from an agent shell. The claim is now *harmless* because the couchcore twin proves the same wiring — but the sentence credits the wrong test; worth one word.

## 6. Architectural notes for upcoming work

- **ARCH-DRY** — pass on production code (one selector, one gate, `Resumable()` consumed not re-enumerated); flagged for prose (BR-6) and the test-setup triple (Minor). **ARCH-PURE** — pass: `SelectUniqueResumableRoot` is genuinely pure and table-tested with no IO; the new tests drive the IO shell entirely through injected fakes. Flagged only by the projector-enforcement finding, which is an ARCH-PURE argument as much as a correctness one — the invariant belongs in the pure layer. **ARCH-PURPOSE** — pass on code (the refusal-reason sweep behind BR-1 is complete), flag on records (BR-4's atlas half). **ARCH-MOCK** — pass: `FakeThreadArtifactCollisionChecker` implements both resolvers behind the same seam as `ScopedThreadArtifactCollisionChecker`, so production and test share the boundary; `BenchmarkZellijSnapshotLive` measures the real binary but asserts nothing, so it is a measurement harness, not a conformance check — fine here, worth naming in #172. **ARCH-CONSTRAINTS** — flag, see BR-4. **ARCH-SECURE** — pass: `bindingResumeDiagnostic` fails closed to unbound by `default:`, a corrupt manifest errors rather than degrading (`TestStartInteractiveInventoryFailureCreatesNoRoot`), no credential reaches a log, argv or fixture.
- Carry to M4/#171: with the gate in place, a detached thread whose binding degrades is hidden from the switcher while its zellij session keeps running an agent. The atlas now records this and names the unexplored fork (list it, refuse `Enter` with the diagnostic, gate only startup selection) — good practice; keep the fork alive rather than letting the recorded consequence become the settled design.

## 7. Plan revision recommendations

The plan matches the code after this window's edits — the `## Revisions` entry, the Chunk 2 candidate-rule correction (`plan.md:258`) and the added `Couch.ActionableThreadInventoryContext` Integration-points row all landed, and the Core concepts cross-check passes (`SelectUniqueResumableRoot` new at `startup.go:25`; `SelectUniqueParkedRoot` deleted with no residue outside archived prose; both integration rows modified as stated). Two small additions:

- Extend the existing `### 2026-09-02 — M3 boundary review` entry with a fifth paragraph recording that the binding gate is enforced in the IO shell rather than the projector, and that `DetachedSessionObservation` carrying `NativeID` is the deferred class fix — otherwise the next reader sees `parkedResumeProofMatches` and assumes symmetry.
- Correct `workshop/projects/couch.md:942-944`: the mutation verification that actually runs belongs to the couchcore twin, not to the pty-gated acceptance test.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Gate reordered ahead of both appends; reverting it in a scratch copy reddens 3 subtests.
  - id: BR-2
    disposition: addressed
    note: |
      New StartInteractive twin; the reviewer's exact parked-only mutation now reddens it.
  - id: BR-3
    disposition: addressed
    note: |
      Plan Step 3b and workshop/projects/couch.md both credit M2 for the physicalization.
  - id: BR-4
    disposition: not-addressed
    note: |
      Plan half corrected and measured; atlas/couch.md:399-406 still says refresh-worker-only, and "periodic" has no ticker.
  - id: BR-5
    disposition: addressed
    note: |
      README.md:308 and :361 both rewritten for detached rows and resumable resume.
  - id: BR-6
    disposition: not-addressed
    note: |
      All five copies of the selector rationale are unchanged in this window.
  - id: BR-7
    disposition: not-addressed
    note: |
      startup_test.go:37-38 still duplicates one address for both ambiguity cases.
findings:
  - id: new
    severity: Important
    family: listed-implies-resumable
    title: |
      The detached row's binding proof is enforced in the IO shell, not the pure projector, and two comments claim the opposite
    detail: |
      2nd finding in this family -- do NOT patch the instance. Rule: every proof a
      row's Enter requires must travel to ProjectActionableThreads as a field on that
      row's observation type and be enforced inside actionableThreadState; a proof
      enforced only in ActionableThreadInventoryContext's candidate loop is not part of
      the row's contract. Swept enumeration: Live needs a TTY observation (in projector),
      Parked needs NativeID (in projector, parkedResumeProofMatches at
      actionableinventory.go:181-187), Detached needs SessionName (in projector) AND the
      native binding (only at actionableinventory.go:250-253). That asymmetry is how BR-1
      shipped. actionableinventory.go:155-158 asserts the function "fails closed on its
      own, so it does not rely on the caller having filtered candidates" and :44-46 says
      "proof arrives as observations" -- both false for the binding. Class fix: add
      NativeID to DetachedSessionObservation and require it in the detached branch; the
      loop already holds binding at line 255, and ProjectActionableThreads is exported so
      a second caller is a public-API possibility. No live defect today
      (ScopedThreadArtifactCollisionChecker is the only production Artifacts, and it is
      gated). Minimum if the field threading is not cheap here: correct the two comments.
  - id: new
    severity: Minor
    family: shared-helper-not-extracted
    title: |
      The two new StartInteractive tests hand-rebuild the record setup seedStartupParked already encapsulates
    detail: |
      startup_test.go:262-277 and :290-304 each repeat ~10 lines that
      seedStartupParked (startup_test.go:182-195) covers, differing only in
      markActionableParked vs SetDetachedSession. A
      seedStartupResumable(t, env, tag, path, kind) covers all three sites (ARCH-DRY).
  - id: new
    severity: Minor
    family: assertion-admits-vacuous-pass
    title: |
      The no-surviving-session negative asserts only inequality, which a zero-valued record also satisfies
    detail: |
      startup_test.go:309 checks start.Record.Thread != stale.Address, so a zero
      ThreadAddress would pass. TestStartInteractiveCreatesNewRootWithoutExactCandidate:109
      shows the stronger form; one line asserting a real new thread closes it.
```

---

## Re-review — 2026-09-02T17:52:49-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 170 — Rescope couch to couch-lite |
| repo | pair |
| issue file | workshop/issues/000170-rescope-couch-to-couch-lite.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | a89878c31cd7bee06693257e05440c8c4eee7057..1d049ec2a967c881a7e65c73deb297ca72db0177 |
| command | sdlc milestone-close --issue 170 --milestone M3 |
| reviewer | claude |
| timestamp | 2026-09-02T17:52:49-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

BR-8's class fix is real and I proved it in both directions rather than reading the commit: dropping the `Agent`/`NativeID` conjuncts from `detachedResumeProofMatches` reddens two subtests of the new `TestProjectActionableThreadsDetachedRequiresTheResumeProof`, and dropping the IO shell's proof-attach loop (`actionableinventory.go:288-294`, restoring `detached = observed`) reddens `TestActionableInventoryPhysicalizesDetachedRowsLikeParkedOnes`, `TestStartInteractiveResumesUniqueDetachedRoot` and the established-binding subtest — so the fix is both enforced and reachable, not a field set at zero call sites. I also re-verified BR-2 myself (narrowing `StartInteractive`'s rows to `ThreadParked` reddens `TestStartInteractiveResumesUniqueDetachedRoot`) and BR-4's atlas half (`atlas/couch.md:400-420` now names startup as the blocking caller, carries the measured 1.49 s, and drops "periodic" — there is no refresh ticker in `cmd/`, so that correction is accurate). Nothing here is a correctness risk. What keeps it off a clean SHIP is that the head commit's own architectural change — the proof now travelling on the observation and being enforced in the pure projector — landed in no durable record: the atlas section written one commit earlier still says the gate is `ActionableThreadInventoryContext`'s, and the plan's two Core-concepts bullets still describe the pre-fix struct and predicate. Plus one design residue the fix creates: `ProjectDetachedSessions`, the package's own pure producer of `DetachedSessionObservation`, emits a value `ProjectActionableThreads` now always rejects.

## 1. Strengths

- **The fix is the rule BR-8 stated, not the site it named.** `DetachedSessionObservation` gained both proof fields, `detachedResumeProofMatches` (`actionableinventory.go:186-193`) enforces them inside the projector, and the misleading comments at the old `:155-158` and `:44-46` are gone or now true. Mutation-verified here, not taken on trust.
- **The new table test enumerates the failure modes rather than the one that motivated it** (`actionableinventory_test.go:568-608`): missing native id, agent disagreeing with the saved profile, missing session name, plus the positive. Two of the four go red under mutation A.
- **Fail-closed on an unknown address.** An observation whose address is not in `detachedProof` picks up the zero value, and `IsSupportedAgent("")` is false, so a misbehaving resolver cannot inject a row. Worth having verified.
- **The atlas records the *cost* of the gate, not just the gate** (`atlas/couch.md:479-487`): a detached thread whose binding degrades is now invisible while its agent keeps running, the alternative design is named, and the fork is assigned to `pair#171`'s family. That is the honest form of an architectural note.
- **The envelope is measured, split by caller, and the slow half is filed** (`atlas/couch.md:400-420`, issue Log 540-556, `pair#172`) — the switcher's 50 ms/16 ms budgets are stated as untouched and the reason (event-driven, last-good render) is given rather than asserted.

## 2. Critical findings

None.

## 3. Important findings

**No durable record describes the layer the fix moved to.** *This is the 2nd finding in family `record-claims-unverified-delivery`.* Per the escalation rule I am not asking for these four lines to be patched. The rule: **a durable record that restates a code contract — a struct's field list, a projector's admission predicate, or which layer enforces an invariant — is a hand-maintained restatement with no derivation, so when the contract changes every record in the set must be swept in the same commit, and the set must be enumerated in the plan rather than rediscovered each round.** Measured prevalence, three rounds: BR-3 (one record crediting the wrong milestone); round 4's unrecorded item (three restatements of the candidate rule, two drifted, one swept last window); and this window, four records describing the pre-`1d049ec2` layering — `atlas/couch.md:470-472` ("`ActionableThreadInventoryContext` resolves `ResolveEstablished` … and drops any whose binding is not one exact established root", true but now half the mechanism), `workshop/projects/couch.md:939-941` ("fixed as one gate for both kinds"), `plan.md:238` (`DetachedSessionObservation` — `{Address ThreadAddress; SessionName string}`, now four fields), and `plan.md:234` (`ThreadDetached` "Emitted only when … exactly one `DetachedSessionObservation` matches its address", which now also requires agent agreement and a non-empty `NativeID`). Eight record sites across three rounds is the enumeration failing to exist, not eight typos. Cheapest durable form: name the enumeration once in the plan's Core concepts preamble ("these bullets restate code contracts; a change to the contract sweeps this list"), and stop transcribing field lists there at all — the table row plus the path is the claim worth keeping.

**`ProjectDetachedSessions` emits a `DetachedSessionObservation` that `ProjectActionableThreads` now always rejects** (`detachedsessions.go:63` vs `actionableinventory.go:186-193`). Both are exported pure functions in one package; the obvious composition of them yields zero detached rows, silently, and no test covers that composition. The type's own doc (`actionableinventory.go:51-54`) now says an observation without `Agent`/`NativeID` "is not evidence of a resumable thread, only of a running session" — which is exactly what its only pure producer produces. Today this is invisible because `ActionableThreadInventoryContext:288-294` decorates in between and `resume.go:222-226` only reads `Address`, but the failure mode for a future caller is "startup silently stops reattaching", which fails open in the annoying direction and would pass every test. Fix sketch (ARCH-SECURE's make-invalid-state-unrepresentable lens): split the type — keep `DetachedSessionObservation{Address, SessionName}` as the session fact that `ProjectDetachedSessions` and `resume.go` share, and give the projector a `DetachedResumeObservation{Address, SessionName, Agent, NativeID}` assembled at the shell boundary. A doc comment on `ProjectDetachedSessions` is the minimum, but BR-8 is the precedent for comments not being the fix.

**M3 produced a Critical and a three-round family and added nothing to `workshop/lessons.md`** (last touched at M2, `9f7d4245`). AGENTS.md §4 asks for the rule that prevents the repeat, and M1 (`dec5928a`) and M2 set the per-milestone precedent. Two entries are owed and both are two lines: (a) *widening an equivalence class widens its gates* — when a new state joins an existing one behind a shared predicate (`Resumable()`), every precondition the old member was gated on must be applied to the new one in the same commit; gating one member and not the other is how BR-1 shipped; (b) *a proof enforced in the IO shell is not part of the row's contract* — every precondition a row's action requires must travel to the pure projector as a field on its observation type, or the next caller reintroduces the bug.

## 4. Minor findings

- **`detachedResumeProofMatches` and `parkedResumeProofMatches` repeat the same four-condition profile guard verbatim** (`actionableinventory.go:187` and `:196`), differing only by `SessionName != ""`. *This is the 2nd finding in family `shared-helper-not-extracted`* (BR-9 is the test-side instance). The rule: **when a second variant of an existing predicate is added, the invariant part is extracted into a shared helper in the same commit — a doc comment asserting the two are "twins" documents the duplication instead of removing it.** Measured prevalence: two production sites here, three test sites in BR-9. `func resumableProfile(record ThreadRecord) (*LaunchProfile, bool)` covers both production sites and preserves the symmetry the comment is defending (`ARCH-DRY`).
- The decoration loop overwrites `observation.Agent`/`NativeID` unconditionally (`actionableinventory.go:292`), so a resolver that ever populated them would be silently clobbered. Harmless today; a comment or a `!= ""` guard would say which layer owns those fields.
- `detachedProof` is typed `map[ThreadAddress]ParkedResumeObservation` (`actionableinventory.go:227`) — the carrier for the *detached* proof is named for the parked one. Reuse is right; the name reads as a bug for a moment.

## 5. Test coverage notes

- Green here: `go build ./...`, `go vet ./cmd/internal/couchcore`, and `go test ./cmd/internal/couchcore -run 'SelectUnique|StartInteractive|ActionableInventory|ProjectActionable|Detached|Resume'`. A full `go test ./...` fails only with `ptychild: operation not permitted` — identical shape on untouched packages (`ptychild`, `hostty`, `keyscmd`, `cmd/couch`), so environmental, not this window's.
- Mutations run, not assumed: (A) drop the two proof conjuncts → 2 subtests red; (B) restore `detached = observed` → 3 tests red; (C) filter `rows` to `ThreadParked` in `StartInteractive:55` → `TestStartInteractiveResumesUniqueDetachedRoot` red. The claims in `workshop/projects/couch.md:944-946` hold.
- Uncovered: the `ProjectDetachedSessions` → `ProjectActionableThreads` composition (Important finding above). Nothing pins that the two pure halves agree on what a valid observation is.
- The couchcmd acceptance pair (`run_test.go:322/377`) still cannot run anywhere without a pty, which remains why the couchcore-level twins matter.

## 6. Architectural notes for upcoming work

- **ARCH-DRY — flag** (Minor: the duplicated profile guard; BR-6's five prose copies still open). **ARCH-PURE — pass**, and improved by this window: the proof moved from the IO shell into the projector, and the new test runs `ProjectActionableThreads` with no IO, no fake and no clock. **ARCH-PURPOSE — pass on code, flag on records**: I ran the shadow-sweep BR-8 asked for and the projector-side enumeration is now complete (Live → TTY observation, Parked → `parkedResumeProofMatches`, Detached → `detachedResumeProofMatches`); the sweep of the *records* that describe it did not happen. **ARCH-MOCK — pass**: `FakeThreadArtifactCollisionChecker.DetachedSessions:82-97` produces the same undecorated shape as `ScopedThreadArtifactCollisionChecker:202-251`, so test and production share the boundary exactly — including, notably, the shape the Important finding above is about. `BenchmarkZellijSnapshotLive` still asserts nothing, so it is a measurement harness rather than the live conformance check; that belongs to `pair#172`. **ARCH-CONSTRAINTS — pass this window**: the envelope is measured (1.49 s / 13 live sessions), split by caller, bounded per query by `zellijQueryTimeout` (`launcher/zellij.go:20,84`), and the aggregate `(2+N)×5 s` worst case is stated with `pair#172` owning it. **ARCH-SECURE — pass**: the projector now parses observations into a validity decision at its boundary instead of trusting provenance, which is the right direction; no credential reaches a log, argv or fixture.
- Carry to M4: `DetachedSessionObservation` is the type M4/#171/#172 will consume. Settling its dual contract now is cheaper than after a third consumer exists.

## 7. Plan revision recommendations

- Extend `### 2026-09-02 — M3 boundary review` with a fifth paragraph: the binding proof now travels on `DetachedSessionObservation` and is enforced in `ProjectActionableThreads`, with `ActionableThreadInventoryContext` reduced to assembling it. Without that, `plan.md:238`'s two-field struct and the atlas's shell-only framing send the next reader to the wrong layer — which is BR-3's failure mode repeating at a different address.
- Correct `plan.md:238` (drop the transcribed field list; point at the symbol) and `plan.md:234` (`ThreadDetached` also requires agent agreement and a non-empty `NativeID`).
- Add the enumeration named in the Important finding: which records restate a code contract, so the next contract change sweeps a written list instead of whichever copy a reviewer happens to grep.
- `atlas/couch.md:470-472` and `workshop/projects/couch.md:939-941` need the same one-clause correction — the gate is enforced in the projector; the inventory supplies its evidence.

```findings
dispose:
  - id: BR-4
    disposition: addressed
    note: |
      atlas:400-420 now names startup as the blocking caller, carries the 1.49 s measurement, and drops "periodic" (no refresh ticker exists in cmd/).
  - id: BR-6
    disposition: not-addressed
    note: |
      Unchanged this window: startup.go:9-24, startup_test.go:12-18, atlas/couch.md, projects/couch.md all still carry the same rationale.
  - id: BR-7
    disposition: not-addressed
    note: |
      startup_test.go:36-38 still passes the same ThreadAddress twice for both ambiguity cases.
  - id: BR-8
    disposition: addressed
    note: |
      Agent+NativeID added and enforced in detachedResumeProofMatches; mutation-verified in both directions (enforcement and reachability).
  - id: BR-9
    disposition: not-addressed
    note: |
      startup_test.go:262-289 and :291-311 still hand-rebuild what seedStartupParked (:172-186) encapsulates.
  - id: BR-10
    disposition: not-addressed
    note: |
      startup_test.go:309 still asserts only inequality; a zero StartResult would pass.
findings:
  - id: new
    severity: Important
    family: record-claims-unverified-delivery
    title: |
      No durable record describes the layer the binding proof moved to; four records still describe the pre-fix shape
    detail: |
      2nd finding in this family -- do NOT patch the four lines. Rule: a record that
      restates a code contract (a struct's fields, a projector's admission predicate,
      which layer enforces an invariant) is a hand-maintained restatement with no
      derivation, so a contract change must sweep the enumerated set in the same
      commit, and the enumeration belongs in the plan. Measured prevalence over three
      rounds: BR-3 (1 site), round 4's unrecorded candidate-rule item (3 sites, 2
      drifted), and this window (4 sites) -- atlas/couch.md:470-472, projects/couch.md:939-941,
      plan.md:238 (two-field struct, now four), plan.md:234 (admission predicate missing
      the agent and NativeID conjuncts). Durable fix: name the enumeration once in the
      plan's Core concepts preamble and stop transcribing field lists there.
  - id: new
    severity: Important
    family: producer-emits-value-its-consumer-rejects
    title: |
      ProjectDetachedSessions emits a DetachedSessionObservation that ProjectActionableThreads now always rejects
    detail: |
      detachedsessions.go:63 sets only Address and SessionName; actionableinventory.go:186-193
      now additionally requires Agent and NativeID. Both are exported pure functions in one
      package, so composing them directly yields zero detached rows -- silently, with no test
      covering it, and the operator-visible effect is "startup stops reattaching". Harmless
      today only because ActionableThreadInventoryContext:288-294 decorates in between and
      resume.go:222-226 reads only Address. Fix: split the type -- DetachedSessionObservation
      {Address, SessionName} for the session fact, DetachedResumeObservation adding the proof,
      assembled at the shell boundary (ARCH-SECURE: make the invalid state unrepresentable).
  - id: new
    severity: Important
    family: lesson-not-recorded-for-boundary-defect
    title: |
      M3 produced a Critical and a three-round family and added no rule to workshop/lessons.md
    detail: |
      lessons.md was last touched at M2 (9f7d4245); M1 has its own lessons commit (dec5928a),
      so per-milestone is this issue's own precedent and AGENTS.md section 4 asks for it. Two
      entries are owed, both two lines: widening an equivalence class widens its gates (gating
      one member of Resumable() and not the other is how BR-1 shipped); and a proof enforced
      in the IO shell is not part of the row's contract (BR-8's rule).
  - id: new
    severity: Minor
    family: shared-helper-not-extracted
    title: |
      detachedResumeProofMatches repeats parkedResumeProofMatches' four-condition profile guard verbatim
    detail: |
      2nd finding in this family -- do NOT patch the instance. Rule: when a second variant of
      an existing predicate is added, the invariant part is extracted into a shared helper in
      the same commit; a doc comment calling the two "twins" documents the duplication rather
      than removing it. Measured prevalence: 2 production sites (actionableinventory.go:187,
      :196) plus BR-9's 3 test sites. A resumableProfile(record) (*LaunchProfile, bool) helper
      covers both production sites and preserves the symmetry the comment defends (ARCH-DRY).
```
