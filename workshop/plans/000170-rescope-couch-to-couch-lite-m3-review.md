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
