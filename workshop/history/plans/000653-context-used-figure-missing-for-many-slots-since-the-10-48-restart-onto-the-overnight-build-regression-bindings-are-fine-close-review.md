# Boundary Review — pair#653 (whole-issue close)

| field | value |
|-------|-------|
| issue | 653 — Context-used figure missing for many slots since the 10:48 restart onto the overnight build (regression; bindings are fine) |
| repo | pair |
| issue file | workshop/issues/000653-context-used-figure-missing-for-many-slots-since-the-10-48-restart-onto-the-overnight-build-regression-bindings-are-fine.md |
| boundary | whole-issue close |
| milestone | — |
| window | 07b6a897be5e67809b3b476606040460484cccce..64d5798322f7471c617b93761bcd3b958d4d1829 |
| command | sdlc close --issue 653 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-17T14:57:21-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Diff is read. Now running the tests, then reverting the fix to check the regression test goes red.

Fix confirmed red→green. Now checking which of the four re-admit sites the tests actually reach.

Only the `onlyGrew` site is pinned by tests; the other three re-admit sites survive mutation. Quick check for an existing append/growth helper (ARCH-DRY).

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The fix is correct and well aimed. Admission no longer stops for good when the transcript grows during admission, or when a same-launch rebind happens; instead it resets and re-admits on the next Poll. I confirmed the regression in the pinned checkout. With the base `title_projection.go` restored, `TestTitleProjectionGrowthNeverStopsTitle/admit` and the rewritten `TestTitleProjectionAdmissionRejectsSnapshotDrift` both fail with "artifact changed during observation". With the fix they pass, and the full `sessioninventory` and `titlepoller` packages are green. What keeps this from SHIP is coverage. There are four re-admit sites, and I disabled each one in turn. Only the `onlyGrew` path makes any test fail. The other two growth checks inside admission, and the rebind reclassification in `owner()`, can be removed and every test still passes. The rebind change also loosens a check that used to stop the title, and nothing pins its new behavior.

**1. Strengths**
- The re-admit decision is made in one place. `settle` (`title_projection.go:112`) sends every Poll failure either to `reset` or to `stop`, so no call site makes the choice on its own. The fix sits in the shared projection, as the plan requires, so #649 inherits it.
- `reset` (`:102`) puts back the last displayed result. That undoes the usage that `admit` folded into `p.result` before it bailed, and keeps the count visible between polls. It also clears `p.other`, which `admit` appends to without clearing first. That was harmless when admission ran once and would have been a bug once admission repeats.
- Growth only counts when it is `sameTitleIdentity` plus a strictly larger size (`grewFrom`, `:62`). A shrink, a replaced file, or a same-size fingerprint change still falls through to `ErrArtifactChanged`, which matches the classification agreed with pair:6.
- The rewritten drift test keeps its real invariant: no drifted pin, so the next Poll reads from offset 0 (`title_projection_test.go:217`). It drops only the stickiness.
- The atlas paragraph was updated in the same range, with the transient and terminal cases listed accurately.

**2. Critical findings**
None.

**3. Important findings**
- **The three remaining re-admit sites are untested** (`title_projection.go:137`, `:251`, `:265`). I disabled each one in turn and ran `TestTitleProjection*`; all tests stayed green:
  - `:251`: transcript grew between validation and the post-fold `checkArtifacts`.
  - `:265`: transcript grew during the 4KiB boundary read.
  - `:137`: same-launch rebind to a new `RootNativeID` now returns `errTitleReadmit` instead of `ErrArtifactChanged`.

  The drift test's `afterRead` hook (`limit < 64<<10`) fires on the first small read of the tiny fixture, so it only reaches `onlyGrew`. Fix sketch:
  - One fixture per growth site, using an `afterRead` that fires only on the boundary-read call (`offset == rawOffset-n`, `limit <= 4096`) or after it.
  - A rebind test: admit root A, append a binding record for root B under the same launch, then assert the Poll returns no error and the next Poll admits B.
  - A paired negative test showing a binding that disappears (`Binding == nil`) still stops.

**4. Minor findings**
- The type comment at `title_projection.go:29-30` says "only a lost owner or a mutation of admitted authority stops the projection for good". In fact other errors are also terminal: read and stat I/O errors, `ErrReadLimit`, `ErrRootTranscript`, malformed records, failed root validation, and missing invocation identity. The atlas text is more accurate; consider matching it.
- `onlyGrew` (`:200`) re-admits whenever `ValidateTargetWork` fails while any artifact grew. That would mask a real terminal validation failure for as long as the transcript keeps growing every poll. It is bounded (the failure shows up once growth pauses), but it is worth a comment.
- For agy, a growing database in `p.other` is still terminal in `checkArtifacts` (`:305`), both at admission and in the tail. It is the same family of growth-treated-as-mutation, but it is outside this issue's Claude/Codex scope.
- One Done-when clause isn't recorded yet: "after the next restart, every bound slot shows its context figure. Checked live". The Log has a 100-admission probe, not a post-restart fleet check. Add it to the Log before close.

**5. Test coverage notes**
- On the base code, the admit subtest is red and the tail subtest is green. Tail growth was already handled by the `readEnd.Size == readStart.Size` checks (`:341`, `:356`), so the tail subtest is a guard, not the regression test. That matches the plan's note "Red on main (admit)".
- The tests use the in-package `FakeRuntime` with no real I/O, which fits the projection being runtime-injected.

**6. Architectural notes**
- ARCH-DRY: pass. `grewFrom` and `onlyGrew` are new and don't duplicate any existing helper (I grepped for growth/append helpers). `onlyGrew` reuses `equalFingerprint`, `fingerprintFromEntry` and `sameTitleIdentity`.
- ARCH-PURE: pass. The classification is a pure predicate over `FileEntry` values, and the stat I/O goes through the injected `Runtime`.
- ARCH-PURPOSE: pass, with the caveat in the Important finding. The purpose is that ordinary growth never permanently removes the count, and that is delivered at every growth-sensitive admission site. The agy database sibling (Minor) is separable.

**7. Plan revision recommendations**
- Plan item 1 should say the RED test covers only the validation-race site. Add a checklist item for the boundary-read, post-fold and rebind sites (see the Important finding).

```findings
findings:
  - id: new
    severity: Important
    kind: defect
    family: reclassified-error-path-untested
    title: |
      Three of four new errTitleReadmit sites survive mutation: boundary-read growth, post-fold growth, same-launch rebind
    detail: |
      I disabled each site in turn in the pinned checkout and ran TestTitleProjection*. Only the onlyGrew site (title_projection.go:200) fails a test. grewFrom(p.transcript,current) at :251, grewFrom(current,after) at :265, and the rebind reclassification in owner() at :137 can each be removed with every test green. The drift test's afterRead fires on the first small read, so it only reaches onlyGrew. Add a fixture per site: an append timed to the boundary read and to the post-fold stat; a same-launch binding to a new root that must re-admit rather than stop; and a negative test that Binding==nil still stops.
  - id: new
    severity: Minor
    kind: defect
    family: doc-claim-overstates-exhaustiveness
    title: |
      TitleProjection comment says only a lost owner or authority mutation is terminal; I/O, read-limit, malformed-record and identity errors are too
    detail: |
      title_projection.go:29-30. Other errors also stop for good: ReadAt/StatFile errors, ErrReadLimit, ErrRootTranscript, malformed complete records, failed root validation, and missing invocation identity. Reword it to match the atlas list.
  - id: new
    severity: Minor
    kind: defect
    family: growth-treated-as-mutation
    title: |
      For agy, a growing p.other database is still terminal in checkArtifacts during admission and tail
    detail: |
      title_projection.go:305 requires an equal fingerprint for every non-transcript artifact, so agy database growth still stops the title for good. Outside this issue's Claude/Codex scope; note it for a follow-up.
  - id: new
    severity: Minor
    kind: defect
    family: done-when-clause-unrecorded
    title: |
      The Done-when live check after the next restart (every bound slot shows its figure) is not recorded in the Log
    detail: |
      The Log records a 100-admission probe on 1-pair-6, not a fleet check after a restart. Record the live verification before close.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"f5e0085b-1c46-4e2f-a169-2fd9aef817d9","repository":"github.com/xianxu/pair","issue_id":"000653","recipe":"small-diff-review","round":1,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/3ba6e1fd9d7d5bf3","reviewed_head":"64d5798322f7471c617b93761bcd3b958d4d1829","slot":"pair:2","requested_at":"2026-10-17T21:55:22.515509Z"},"phase":"finished","pid":19853,"started_at":"2026-10-17T21:55:22.614269Z","finished_at":"2026-10-17T21:57:21.143556Z","session_id":"63e47e0a-2e5b-49bf-b563-49c9859065d7","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.432064459s
head: "64d5798322f7471c617b93761bcd3b958d4d1829"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-17T15:15:17-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 653 — Context-used figure missing for many slots since the 10:48 restart onto the overnight build (regression; bindings are fine) |
| repo | pair |
| issue file | workshop/issues/000653-context-used-figure-missing-for-many-slots-since-the-10-48-restart-onto-the-overnight-build-regression-bindings-are-fine.md |
| boundary | whole-issue close |
| milestone | — |
| window | 07b6a897be5e67809b3b476606040460484cccce..55c26dbea83dc71be68e161b7f851192778e8799 |
| command | sdlc close --issue 653 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-17T15:15:17-07:00 |
| verdict | REWORK |

## Review

Inspecting whether the proof-validation path inside `ValidateBindingProof` has its own unclassified growth failure.

Writing a temporary probe test: transcript grown past the proof (the normal production state), then an append during the first admission read.

Confirmed a remaining growth site. Now mutation-checking BR-1's three sites.

```verdict
verdict: REWORK
confidence: medium
```

The fix works for the sites it targets, but it leaves out the growth race that production probably hits most often, so this can't ship yet. Classifying growth as transient through `errTitleReadmit`, `reset` and `settle` is the right design. All three admission growth sites are now pinned: I disabled each one in turn in the pinned checkout, and each change turned a test red. Dropping the rebind reclassification also checks out, because `owner()` is identical to the base. The problem is one more growth path. It runs inside `ValidateBindingProof`, before any of the classified sites, and it still returns a terminal `ErrArtifactChanged`. It fires when the file has no generation token. On darwin, `filemeta_darwin.go` sets the token from `stat.Gen`, which is normally 0 for non-root users, so the token is empty on the operator's Macs. It also needs the transcript to have grown since the binding proof, which is the normal state once the agent has taken a turn. I reproduced it with a throwaway test, and the title stopped for good. All the new tests use fixtures where the proof matches the file exactly, so they never reach this path.

1. **Strengths**
   - `settle`/`reset` (`title_projection.go:113-120`) keep one classification point in `Poll`, and `reset` keeps the last displayed figure visible while the projection re-admits.
   - `TestTitleProjectionGrowthAtEachAdmissionSite` finds each site from a clean control admission's stat/read log instead of hard-coding offsets. I mutation-checked it: disabling each of the three sites turns a test red.
   - Removing the unreachable rebind reclassification restores a terminal check that had been loosened, and the existing "conflict" case still pins it.
   - The atlas entry (`atlas/architecture.md:377`) is updated in the same range and correctly lists the same-size fingerprint change as still terminal.

2. **Critical**
   - **`title_projection.go:183-190`: growth during the proof's fallback revalidation still stops the title (ARCH-PURPOSE).**
     - **What happens:** `ValidateBindingProof` (`query.go:144`) calls `AdvanceTargetValidation`. With an empty generation token, that call rejects any growth since the proof (`incremental_inventory.go:226`), so the code falls back to `ValidateTargetWork` (`query.go:216-219`). That function's `ObserveStableArtifact` call returns `ErrArtifactChanged` if the file grows mid-read while the token is empty (`incremental_inventory.go:450-452`). `admit` then returns the error unclassified at `if err != nil { return err }`, and `settle` stops the title.
     - **Reproduction:** take `proofBackedCodexFixtureWithGeneration(…, "")`, append one record before the first Poll, then append once during the first read. Poll returns `artifact changed during observation`, and the next Poll returns it again. With `gen:7` both polls succeed.
     - **Not covered by tests:** every test fixture has the proof equal to the file, so `ValidateBindingProof` takes the "unchanged" early return and never reads.
     - This is the 2nd finding in family `growth-treated-as-mutation`. Don't add a fourth per-site check. The rule that covers every instance: **an admission that fails while the transcript has only grown since admission began is transient.** Stat the transcript and the other proof artifacts once before validating. Then make one classification in `admit` (or in `settle` for errors from `admit`): if the error is `ErrArtifactChanged` and the artifacts only grew, re-admit. Today's three sites and the proof path fold into that one check.
     - **Regression test:** an empty-generation fixture whose transcript has grown past the proof, with an append during the first read.

3. **Important:** none.

4. **Minor**
   - **`title_projection.go:57-59`: stale comment.** The `errTitleReadmit` comment still lists "a rebind of the same launch to a new root", a case this range deliberately removed. This is the 2nd finding in family `doc-claim-overstates-exhaustiveness`. Rule: a comment that lists error classes should describe the one classification point (`settle` or the growth rule above), not restate the cases one by one. If the Critical fix makes the growth rule the single check, both this comment and the `TitleProjection` comment can just state the rule.

5. **Test coverage**
   - The growth tests only use proof-equals-file fixtures. They need at least one fixture where the proof is stale and the generation token is empty, which is the darwin production shape.
   - After my temporary changes were reverted, the full package passes (`go test .`, 16s).

6. **Architectural notes**
   - The per-site `grewFrom`/`onlyGrew` checks spread one rule across four places. The single classification described under Critical is also the base pair#649 should inherit, and it would make pair#656 (agy database growth) a one-line extension.

7. **Plan revisions**
   - The second Plan item says growth racing admission re-admits; it doesn't yet cover growth inside proof revalidation. Add a "## Revisions" entry naming the proof-fallback site and the single-classification rule.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Mutation-checked in the pinned checkout. Disabling onlyGrew fails SnapshotDrift and GrowthNeverStopsTitle; disabling either grewFrom site fails GrowthAtEachAdmissionSite. The rebind reclassification was removed and owner() matches base.
  - id: BR-2
    disposition: addressed
    note: |
      TitleProjection comment (title_projection.go:27-31) now names I/O, read-limit, malformed-record and root-validation failures as terminal.
  - id: BR-3
    disposition: addressed
    note: |
      Out of scope and filed as pair#656, recorded in the Log.
  - id: BR-4
    disposition: not-addressed
    note: |
      The Log says the live post-restart fleet check stays open until the next Couch restart onto a build with this fix. That Done-when clause is still unrecorded.
findings:
  - id: new
    severity: Critical
    kind: defect
    family: growth-treated-as-mutation
    title: |
      Growth during ValidateBindingProof's fallback revalidation still stops the title for good when the generation token is empty (darwin st_gen 0)
    detail: |
      admit at title_projection.go:183-190 returns ValidateBindingProof's error unclassified. With an empty generation token and a transcript grown past its proof, AdvanceTargetValidation (incremental_inventory.go:226) falls back to ValidateTargetWork (query.go:216-219), and ObserveStableArtifact returns ErrArtifactChanged on mid-read growth (incremental_inventory.go:450). Reproduced: fixture with gen "", one append before Poll, one during the first read; both polls error terminally, while gen:7 succeeds. This is the 2nd finding in this family, so fix the rule, not the site. An admission that fails with ErrArtifactChanged while the proof artifacts only grew since admission began (stat once before validation) re-admits. Classify that once and replace the per-site grewFrom/onlyGrew checks. Add an empty-generation, stale-proof regression test.
  - id: new
    severity: Minor
    kind: defect
    family: doc-claim-overstates-exhaustiveness
    title: |
      errTitleReadmit comment still lists a same-launch rebind, a case this range removed
    detail: |
      title_projection.go:57-59. 2nd finding in this family. Rule: describe the one classification point, not a hand-kept list of cases; once the growth rule is a single check, both this comment and the TitleProjection comment should just state it.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"e6e56125-c5aa-4c4e-9fb5-f3b8d96d11af","repository":"github.com/xianxu/pair","issue_id":"000653","recipe":"small-diff-review","round":2,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/bd89c1d09b505b0d","reviewed_head":"55c26dbea83dc71be68e161b7f851192778e8799","slot":"pair:2","requested_at":"2026-10-17T22:13:03.396254Z"},"phase":"finished","pid":94334,"started_at":"2026-10-17T22:13:03.553665Z","finished_at":"2026-10-17T22:15:16.324657Z","session_id":"442ed130-5999-4427-8535-338589eaf7ac","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.248846833s
head: "55c26dbea83dc71be68e161b7f851192778e8799"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-17T15:23:58-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 653 — Context-used figure missing for many slots since the 10:48 restart onto the overnight build (regression; bindings are fine) |
| repo | pair |
| issue file | workshop/issues/000653-context-used-figure-missing-for-many-slots-since-the-10-48-restart-onto-the-overnight-build-regression-bindings-are-fine.md |
| boundary | whole-issue close |
| milestone | — |
| window | 07b6a897be5e67809b3b476606040460484cccce..6ca99b831757b243024f8456d204933f57203761 |
| command | sdlc close --issue 653 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-17T15:23:58-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Probing whether growth during proofless (chosen-id) validation is still terminal.

Confirmed: growth during proofless validation still stops the title. Next, checking whether the BR-5 regression test fails when the fix is reverted.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

BR-5 is fixed for the case it named. `classifyAdmission` (`title_projection.go:86-91`) is now the one place an admission failure gets reclassified. It takes a baseline before proof validation (`:194-199`) and again after validation (`:207`). I checked the regression evidence by reverting parts of the fix in the disposable checkout. Removing the classifier turns `AdmissionRejectsSnapshotDrift`, `GrowthNeverStopsTitle` and `GrowthAtEachAdmissionSite` red. Removing only the proof pre-stat turns `GrowthNeverStopsTitle/stale-proof` red. The package's full test suite passes at head.

One path in the same family is still open. When a binding has no proof, `ValidateNamedContent` runs with no baseline taken first (`p.since = nil`). An append that lands while it reads stops the title permanently. I reproduced this for Claude and Codex chosen-id fixtures: the first poll returns `ErrArtifactChanged`, and the second poll returns the same error. The Log mentions the gap (2 of 103 live bindings) and gives "without a second walk" as the reason. That is a cost reason, not an impossibility. The Done-when clause says *every* bound slot, and the atlas states without qualification that growth during validation is transient. That blocks SHIP, but it is cheap to fix.

**Strengths**
- One classification point, `classifyAdmission`/`onlyGrew` (`title_projection.go:86-110`), replaces the per-site checks. The I/O errors from `checkArtifacts` are no longer folded into `ErrArtifactChanged` (`:263-278`), so real I/O failures still stop the title.
- `reset(last)` (`:122-130`) keeps the figure that was already showing, so a busy slot doesn't flash empty while it re-admits.
- `GrowthAtEachAdmissionSite` finds each site from a clean admission's stat/read ordering instead of hard-coding offsets, and covers both proof and proofless bindings at those sites.
- The `stale-proof` subtest models the darwin case (empty generation token, transcript already past its proof), and it fails without the pre-stat.

**Critical findings**
None.

**Important findings**
- **Proofless admission takes no baseline before validation, so growth during `ValidateNamedContent` is still terminal** (`title_projection.go:193,201-202`). **This is the 3rd finding in family `growth-treated-as-mutation`.**
  - **The rule:** every admission path stats the artifacts it is about to validate before its first read. The proof branch already does this; the proofless branch doesn't.
  - **Fix sketch:** before `ValidateNamedContent`, run `inventory.Observe` and `Select` (`TargetExplicitResume`, `nativeID`) and seed `p.since` from the selected entries. This adds one storage walk, and only during admission, which happens rarely. The other option is to have `ValidateNamedContent` accept a pre-observed snapshot.
  - **Instances in this window:**
    - (a) the proofless validation call at `:202`;
    - (b) the atlas claim at `atlas/architecture.md:377`, "the transcript grew during validation… is transient", which is false for proofless bindings;
    - (c) the `classifyAdmission` comment ("every artifact admission had observed"), which implies a baseline the proofless path doesn't have.
  - **Missing test:** a proofless subtest of `GrowthNeverStopsTitle` that appends after the first read.

**Minor findings**
None new.

**Test coverage notes**
- Proof-path growth is covered at all three sites (validation, post-fold, boundary).
- Proofless growth is covered only after validation. The comment "(Growth during validation is TestTitleProjectionGrowthNeverStopsTitle.)" covers only the proof fixture.

**Architecture**
- **ARCH-DRY: pass.** `grewFrom`/`onlyGrew` is a single helper with one caller.
- **ARCH-PURE: pass.** The classifier does stat I/O only inside the Poll admission path, and the tests drive it through the injected fake runtime.
- **ARCH-PURPOSE: flag.** The issue's purpose is that every bound slot shows its figure. The proofless validation race is a known part of that purpose, documented but not fixed (see the Important finding).

**Plan revision recommendations**
- Plan line 51 says the baseline is "proof artifacts stat'd before validation, else the validated observations". After the fix, it should also name the proofless pre-validation observation. Alternatively, add a `## Revisions` entry that records proofless validation growth as an accepted gap, and qualify the atlas sentence to match.

```findings
dispose:
  - id: BR-5
    disposition: addressed
    note: |
      classifyAdmission plus the proof pre-stat; GrowthNeverStopsTitle/stale-proof goes red when the pre-stat is reverted, and three tests go red when the classifier is reverted. The proofless sibling is raised as a new finding.
  - id: BR-6
    disposition: addressed
    note: |
      The errTitleReadmit comment (title_projection.go:57-58) and the TitleProjection comment now point to the one rule (classifyAdmission) and no longer list cases.
  - id: BR-4
    disposition: withdrawn
    note: |
      The dispute holds. The post-restart live check can only run once the fix is on a deployed build. The Log (issue line 63) records it as open, and pre-landing live evidence is recorded (2/100 terminal vs 0/100).
findings:
  - id: new
    severity: Important
    kind: defect
    family: growth-treated-as-mutation
    title: |
      Proofless admission has no pre-validation baseline, so growth during ValidateNamedContent still stops the title for good
    detail: |
      3rd finding in this family. The rule: every admission path stats the artifacts it is about to validate before its first read. title_projection.go:193 sets p.since=nil and the proofless branch (:202) never seeds it, so onlyGrew(nil) returns false and the error stays terminal. Reproduced with titleProviderFixture (Claude and Codex) and one append after the first read: both polls return ErrArtifactChanged. Instances: the proofless call at :202; atlas/architecture.md:377, which claims growth during validation is transient without qualification; the classifyAdmission comment saying every artifact admission had observed. Fix: run Observe+Select for the named root before validation and seed p.since from it (one extra walk, admission only). Add a proofless subtest to GrowthNeverStopsTitle.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"39b1f75b-ef67-4eec-9359-3e17d293ed80","repository":"github.com/xianxu/pair","issue_id":"000653","recipe":"small-diff-review","round":3,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/548dd285e861dc12","reviewed_head":"6ca99b831757b243024f8456d204933f57203761","slot":"pair:2","requested_at":"2026-10-17T22:21:33.662473Z"},"phase":"finished","pid":98899,"started_at":"2026-10-17T22:21:33.769419Z","finished_at":"2026-10-17T22:23:36.212921Z","session_id":"fc52f0a5-be56-45b7-924d-fb9b788a01ef","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.449266792s
head: "6ca99b831757b243024f8456d204933f57203761"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-17T15:28:25-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 653 — Context-used figure missing for many slots since the 10:48 restart onto the overnight build (regression; bindings are fine) |
| repo | pair |
| issue file | workshop/issues/000653-context-used-figure-missing-for-many-slots-since-the-10-48-restart-onto-the-overnight-build-regression-bindings-are-fine.md |
| boundary | whole-issue close |
| milestone | — |
| window | 07b6a897be5e67809b3b476606040460484cccce..810a4ef4930b21b5729f1898671e363659a3cc9c |
| command | sdlc close --issue 653 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-17T15:28:25-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

This round's one open finding, BR-7, is fixed. Before the first read, `admit` now records a baseline of the artifacts it will validate. A proof-backed binding uses the artifacts named in its proof. A proofless binding uses the artifacts found by the same `Observe`+`Select(TargetExplicitResume)` walk that `ValidateNamedContent` runs (`query.go:616-617`). `IncrementalInventory` is built with an empty `Catalog`, so the catalog loop in `ValidateNamedContent` never runs and the walk is the path it actually takes. The baseline therefore matches what validation reads. I checked that the regression test catches the bug: I removed the proofless branch of `baseline` (`title_projection.go`, `selected := inventory.Select(...)` and the loop after it) in the disposable checkout. Three subtests then failed with `ErrArtifactChanged`: `GrowthNeverStopsTitle/proofless`, `GrowthAtEachAdmissionSite/proofless/post-fold` and `/proofless/boundary`. With the code restored, `sessioninventory` and `titlepoller` both pass, and `go vet` is clean. The atlas sentence (`atlas/architecture.md:377`) and the `classifyAdmission` comment now hold for both binding kinds without qualification. I found nothing new to raise.

1. **Strengths**
   - One rule decides every case: `classifyAdmission`/`onlyGrew` (`title_projection.go:~81-110`) in place of per-site checks. Any `ErrArtifactChanged` where every baseline artifact is unchanged or the same file grown becomes a re-admit. Anything else stays terminal, so a real validation failure still surfaces once the transcript stops growing between polls.
   - `baseline` reuses the inventory's own `Observe`/`Select` for the proofless case instead of a second discovery path (ARCH-DRY).
   - `reset` keeps the last displayed result, so a busy slot's count stays visible while it re-admits. That is the purpose of the issue.
   - `GrowthAtEachAdmissionSite` finds the folded-stat and boundary-read sites from a control run's op trace instead of hard-coded indices. The test therefore stays correct if the read/stat order changes.
   - `checkArtifacts` errors are now returned as they are, not folded into `ErrArtifactChanged`, so I/O errors can't pass for growth.
2. **Critical:** none.
3. **Important:** none.
4. **Minor:** none worth recording. The proofless subtest of `GrowthNeverStopsTitle` uses only the Codex fixture; the Claude proofless path is covered by `GrowthAtEachAdmissionSite`.
5. **Test coverage:** every `## Done when` clause about tests has coverage, in both states:
   - Proof-backed and proofless bindings.
   - Admission and tail.
   - All three admission sites: during validation, post-fold and boundary.

   I confirmed BR-7's fix fails when reverted. The live-check clause was already disposed (BR-4 withdrawn).
6. **Architecture:**
   - ARCH-DRY passes: the baseline reuses `Observe`/`Select`, and `grewFrom` and `onlyGrew` are single helpers.
   - ARCH-PURE passes: the classification lives inside the projection's existing runtime seam, and the tests drive it with injected fakes (`FakeRuntime`/`titleObservedRuntime`), not mocks of the logic.
   - ARCH-PURPOSE passes: every admission path now shares one growth rule, and no part of the issue's purpose has been left as a follow-up.

   For later work: the proofless baseline costs one extra storage walk per admission, which is fine because admission is rare. pair#649 inherits the rule through the shared projection.
7. **Plan revisions:** none. Plan item 3 matches the code.

```findings
dispose:
  - id: BR-7
    disposition: addressed
    note: |
      baseline() now runs Observe+Select for proofless bindings before validation; removing that branch turns GrowthNeverStopsTitle/proofless and GrowthAtEachAdmissionSite/proofless/{post-fold,boundary} red; atlas:377 and classifyAdmission comment now accurate.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"70106634-9de5-424e-bf04-2d48508c580b","repository":"github.com/xianxu/pair","issue_id":"000653","recipe":"small-diff-review","round":4,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/f4377a7efc875f11","reviewed_head":"810a4ef4930b21b5729f1898671e363659a3cc9c","slot":"pair:2","requested_at":"2026-10-17T22:27:09.602705Z"},"phase":"finished","pid":43631,"started_at":"2026-10-17T22:27:09.731222Z","finished_at":"2026-10-17T22:28:23.657533Z","session_id":"ceefbcf9-8744-41b4-b25d-815c73965814","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.031794042s
head: "810a4ef4930b21b5729f1898671e363659a3cc9c"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
