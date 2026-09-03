# Boundary Review — pair#170 (milestone M4)

| field | value |
|-------|-------|
| issue | 170 — Rescope couch to couch-lite |
| repo | pair |
| issue file | workshop/issues/000170-rescope-couch-to-couch-lite.md |
| boundary | milestone M4 |
| milestone | M4 |
| window | a89878c31cd7bee06693257e05440c8c4eee7057..f01f5b8afa6bea2013dd30fcfed3f1a8c39649e2 |
| command | sdlc milestone-close --issue 170 --milestone M4 |
| reviewer | claude |
| timestamp | 2026-09-02T21:53:25-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

I have everything I need. Here is the review.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M4 is a strong deletion milestone: ~3,900 lines removed across five subsystems, the razor applied honestly (the *Kept, and why* list is the best part of the plan), and the two persisted-schema hazards were caught by measuring the operator's live store rather than reasoning from Go types. The blocker is one correctness regression: deleting admission also deleted the pristine-start rollback, and the call left in its place (`releaseClaimIfThreadAbsent` on a record that provably still exists) is protection that can never fire — the two tests that pinned the property went with admission and nothing replaced them. Beyond that, the sweep stopped at "the compiler is happy": six symbols and a `make` target are orphaned, and the atlas + issue log both claim the target was deleted. README still documents fleet-policy admission as live behavior. None of it needs redesign; all of it is localized.

## 1. Strengths

- **The tombstone work is exactly right, and measured.** `threadrecord/record.go:44-92` and `threadstore.go:51-60` keep `legacy_actor_id`, `policy`, `claim_generation`, `legacy_cutover`, `legacy_migration_version` as decode-only fields, guarded by `TestPreM4RecordsStillDecode` and `TestPreM4ManifestStillLoads` with fixtures copied from real data. Both redden if the field is removed (strictjson `DisallowUnknownFields`) — I verified they pass and the mechanism is real, not asserted. Ranking the manifest as the worse blast radius and giving it its own guard is the correct call.
- **`StartResolution.CommitArgs` (`startresolution.go:139-155`) is the right response to what D2 surfaced** — three call sites hand-rebuilding "the args that reproduce this resolution", where a wrong guess fails with a drift error nobody drifted into. `ARCH-DRY`, one owner, and `MenuFrame` now holds the resolution instead of a seven-field shadow.
- **The token → fingerprint substitution is argued and pinned, not asserted.** `TestSpawnPreparedRefusesDriftByFingerprint` covers unchanged / drifted-preference / foreign-fingerprint, and the property the fingerprint genuinely does *not* carry (at-most-once) is pinned where it actually lives: `TestStartFormArmedSubmitDispatchesOnce` (`menu_test.go:1129`) drives a second Enter through the reducer and asserts no second `start`. Both pass.
- **`detachedResumeProofMatches` (`actionableinventory.go:186`) moves the binding requirement out of the IO shell into the pure projector**, making the function's own "fails closed on its own" claim true instead of aspirational. Textbook `ARCH-PURE`.
- **The atlas sweep is thorough and honest** — it doesn't just delete the admission prose, it records *what* was deleted and why, and names the one field that survived (`atlas/couch.md`, "Identity" section).

## 2. Critical findings

**C1 — `spawnResolved`'s rollback is dead code at all three post-allocation failure sites (`couch.go:297,301,313`).**

`AllocateThreadTag` (`threadtag.go:40-47`) claims the artifact *and* persists a pristine `Reservation: true` record. The three failure paths after it call `releaseClaimIfThreadAbsent(threadAddress)`, which returns `nil` the moment `GetThread` succeeds (`couch.go:627-636`) — and it always succeeds, because the record was just written. So on a `Proc.Current()` error, an entropy error, or a `CommitStartClaim` write failure, the pristine record and the artifact claim both leak.

Nothing reclaims them: `ProjectActionableThreads` hides reserved records (`actionableinventory.go:145`) and `reconcileInterruptedStarts` skips records with no start transaction (`couch.go:568-571`). The record is invisible in the switcher and permanent.

Pre-M4 this worked two ways: admission's own failure path called `DeletePristineThread` ("Refused pristine reservations are rolled back", `admission.go:90-92`), and the later sites called `rollbackUnforkedStart`. Both are now uncalled — `DeletePristineThread` (`threadstore.go:526`) and `rollbackUnforkedStart` (`couch.go:521`) have zero callers in the tree. The two tests that pinned the property, `TestSpawnCapacityRefusalDoesNotForkAndRollsBackOpaqueReservation` and `TestSpawnPolicyInstabilityDoesNotForkAndRollsBackOpaqueReservation`, were deleted with admission and not replaced; the surviving `assertPreparedStartHadNoEffects` only covers refusals that happen *before* allocation.

Plan Task 13 Step 4 required this explicitly: "plus the pristine-reservation rollback on failure. A single `ThreadStore.CommitStartClaim` CAS covers all four."

Fix sketch: add `rollbackPristineStart(address)` = `Threads.DeletePristineThread(address)` then `releaseClaimIfThreadAbsent(address)`, call it at `couch.go:297,301,313`, and delete `rollbackUnforkedStart`. The regression test is cheap and fails today — `FakeProcOps.CurrentErr` already exists (`procops.go:153`):

```go
env := newTestEnv(t, "/repo")
env.Proc.CurrentErr = errors.New("boom")
_, _, err := env.Couch.Spawn(StartArgs{Worktree: "/repo"})   // want err
assertPreparedStartHadNoEffects(t, env)                       // zero records — fails today
if len(env.Artifacts.Releases()) != 1 { t.Fatal("claim leaked") } // fails today
```

## 3. Important findings

**I1 — The tombstone preserves the record's *shape* but drops the *value*, and the downstream reader hard-errors.**
A pre-M4 incarnation carries `policy.repo_identity` and no top-level `repo_identity`. `fromPersistedThreadRecord` deliberately never reads `DeprecatedPolicy` (`thread.go:184-188`), so such an incarnation loads with `RepoIdentity == ""`. `advanceSuccessfulStart` then refuses: `"successful start has no repository identity"` (`threadstore.go:614`). Reached whenever an *interrupted start* written by the pre-M4 binary is promoted after the upgrade: `reconcileInterruptedStarts` → `StartPromoteLive`/`StartPromoteUnknown` → `AdvanceStart` → `advanceSuccessfulStart` → error → `New()` returns `"reconcile interrupted starts: …"` (`couch.go:110-112`) → **couch will not start at all**, the same whole-store blast radius the manifest tombstone was created to prevent. I checked the operator's live store: 5 records carry `policy`, 0 currently have an open `start`, so it does not fire on this host today — but the milestone's own standard is "'absent from the one store I measured' is not 'absent'". Fix: in `fromPersistedThreadRecord`, when `incarnation.RepoIdentity == ""`, unmarshal `repo_identity` out of `DeprecatedPolicy` — the raw JSON is already retained for exactly this. Extend `TestPreM4RecordsStillDecode`'s fixture with a `"start"` block and assert the identity survives the round trip.

**I2 — The deletion left an enumerable set of orphans; `make test-couch-policy-live` is one and now reports green.**
Deletion is proved by absence (the plan's own `ARCH-PURPOSE` note). Seven survivors, each with exactly one reference — its own definition:

| orphan | site |
|---|---|
| `threadrecord.PolicyCapacity` | `record.go:34` |
| `ThreadStore.DeletePristineThread` | `threadstore.go:526` |
| `Couch.rollbackUnforkedStart` | `couch.go:521` |
| `reflectBytesEqual` | `threadstore.go:905` |
| `ThreadSnapshotConflictError` | `threadstore.go:30-34` |
| `ThreadSnapshot.manifest` / `.raw` | `threadstore.go:39-40` — write-only; `Snapshot()` copies every record's bytes per call for no consumer |
| `test-couch-policy-live` | `Makefile.local:73-81` |

The make target matters most: `PAIR_LIVE_FLEET_POLICY=1 go test … -run '^TestFleetPolicyResolverConformance$'` now prints `ok … [no tests to run]` and **exits 0** — I ran it. A conformance target that silently reports green is worse than the workflow that would have failed loudly, and the workflow is what got deleted instead.

This also makes a **3rd finding in family `record-claims-unverified-delivery`** — `atlas/couch.md` says the target "went with admission in `pair#170` M4, along with the weekly workflow that ran it", and the issue Log says "`.github/workflows/couch-policy-conformance.yml` still invoked the `make test-couch-policy-live` target D1 removed … deleted." Neither is true; the target is still there. Per the escalation rule, do not just fix this instance. **The rule:** a deletion is complete only when the *identifier set* is swept, not the package — for each symbol and target name removed, `grep` the whole tree (Go, `Makefile*`, `.github/`, `atlas/`, `README.md`) before the commit, and no doc or log sentence may claim a removal that grep has not confirmed in the same commit. `workshop/lessons.md` already states this rule verbatim in this very range ("Grep the Makefile, CI workflows, and the atlas for the target and the vocabulary") — it was written and violated in the same boundary, which is the argument for making it mechanical (a `go test` that asserts every `Makefile.local` `go test -run` regex matches at least one test would have caught this one).

**I3 — A 4.5 MB Mach-O arm64 binary is committed at the repo root.**
`couchstartrecovery` landed in `c11f61ea`, almost certainly from `go build ./cmd/probes/couchstartrecovery` while repairing the probe (Task 14 Step 6 — the Makefile target uses `go run`, so nothing wants this file). No `.gitignore` rule covers the repo root. This is the ariadne base layer, so the blob propagates to dependent repos, and it is permanent in history once merged — the cost of fixing it rises discontinuously at this boundary. Drop the blob from the branch and add a root ignore rule (or `/couchstartrecovery`) so a stray probe build cannot recur.

**I4 — README.md still documents fleet-policy admission as live behavior.**
`README.md:307-314` describes new-thread admission coming from `sdlc fleet policy`, bounded keys refusing when occupied, and the typed `provision-worktree` refusal — all deleted by D1. Plan Task 14's own **Files:** list names `README`; it was not touched. This is the **2nd finding in family `readme-stale-for-shipped-surface`**, so per the escalation rule, don't just patch the paragraph. **The rule:** the docs sweep must be driven by an enumeration derived from the diff — the set of deleted identifiers plus changed user-facing behavior — grepped against `README.md` and `atlas/` *together*, rather than from recall of which files felt relevant. The atlas half was done well here and README was simply forgotten, which is the failure mode a shared enumeration removes. The repo already has precedent for mechanical doc pins (`plan_contract_test.go`); a grep-based check over the deleted vocabulary would close the class.

**I5 — The `absent` field is overloaded, silently disabling a pinned-declaration check (`plan_contract_test.go:145-147`).**
`issue151M3ArchitecturalDeclarations` gained `absent: ["…/startgrant.go"]` on `Couch.PrepareStart` and `Couch.SpawnPrepared`. But `absent` is also the guard at line 1481: `if len(declaration.absent) == 0 { issue151M3PinnedDeclarationExists(...) }`. Both declarations now skip the check that they exist at the pinned commit — a check they would still pass, since it reads the frozen git object, not the worktree. Only `StartGrantStore` (whose `source` *is* the absent file) legitimately needs the skip. Plan Task 14 Step 5 warned against exactly this ("the correct repair is a `## Revisions` note on the pinned milestone, not a loosened digest"). Fix: gate line 1481 on whether `declaration.source` itself appears in `absent`, not on `len(absent) != 0`.

**I6 — `ARCH-MOCK`: the new git call has no live conformance case and the fake models only one of its two answer shapes.**
`resolveRepoIdentity` (`couch.go:255`) makes couch depend on `git rev-parse --git-common-dir`, whose answer keys every saved launch preference — the one value D1 had to preserve byte-for-byte. Every fake reply in the tree is `".git"` (`couch_test.go:176,213,409`; `startup_test.go:77,266,296`; `resume_launch_test.go:38,204`; `couchcmd/run_test.go:126`), i.e. the repo-root case. Real git returns a path **relative to the current directory** from a subdirectory — I measured `../../../.git` from `cmd/internal/couchcore` — which is the shape production hits whenever the operator runs `couch` in a subdir. `filepath.Join` happens to handle it correctly, but no test covers it, `conformance_live_test.go` has no git-common-dir case, and the function's own doc comment states the model inaccurately ("relative in a main checkout … and absolute in a linked worktree"), so a future simplification of the join would silently orphan every preference. This is the **2nd finding in family `fixture-realism`**. **The rule:** a fake's canned replies must cover every input shape production can pass the seam — enumerate the production call sites' argument shapes and add a case per shape — and any answer the code *reasons about* gets a live conformance case, not a comment.

## 4. Minor findings

- `RetireIncarnation`'s 14-line doc comment is now attached to `CommitStartClaim` (`threadstore.go:405-419`); `RetireIncarnation` (`:457`) has none.
- `GitRunner`'s doc still says "couch needs exactly one call: rev-parse --show-toplevel" (`git.go:8-9`) — there are two now, and the ARCH-DRY "revisit at the third consumer" note is one consumer closer.
- `resolveStartResolution`'s `ctx` parameter is now unused — `ResolvePolicy` was its only consumer. The bounded 5 s policy subprocess it replaced is now an unbounded `exec.Command` git call on the preview worker (`git.go:27-31`). Pre-existing for `ResolveTree`, so not new exposure, but the last bounded IO on that path is gone.
- `Couch.Spawn`'s doc comment (`couch.go:128-133`) still reads as a production entry point; the plan's *Kept* section promised it would say it is the test seam.
- The `start` op declares `path`/`worktree`/`fingerprint` `Required` but not `ValueRequired` (`ops.go:135-139`), so an empty fingerprint passes schema validation (caught later by the compare). `worktree` is also never read — `resolveStartResolution` uses `WorkingDir()`, which prefers `Cwd`.
- Unrelated gofmt alignment churn swept into the window (`entrypoint/alias_test.go`, `runtimebundle/store_test.go`, `wrapcmd/osc_test.go`, `wrapcmd/extract_fg_test.go`) — harmless, but it adds noise to a deletion diff.

## 5. Test coverage notes

- I could not run the full suite: every failure in `./cmd/internal/{couchcore,couchtty,couchcmd}` is `ptychild: … operation not permitted` from the sandbox, a known environment limit. `go build ./...` is clean; `threadrecord` and `artifactpath` pass; all M4-specific tests I ran individually pass (`TestPreM4ManifestStillLoads`, `TestPreM4RecordsStillDecode`, `TestPathLaunchPreference*`, `TestSpawnPreparedRefusesDriftByFingerprint`, `TestStartFormArmedSubmitDispatchesOnce`, the whole `TestIssue149*`/`TestIssue151*` ledger family).
- **The gap the diff could ship is C1's**: no test exercises a start failure *after* tag allocation. Every "no effects" assertion covers refusals that happen before it.
- `TestPathLaunchPreferenceKeyIsStable` pins the hash of a `{repoIdentity, path}` pair, which is the right guard for the *keying*. It does not touch `resolveRepoIdentity`, so the substitution the milestone actually made — provider value → locally derived value — has no test that would fail if the derivation were wrong. The fake exercises the code path but only at the repo-root shape (see I6).
- Retargeted tests were mutation-checked per the issue Log, and the vacuous-assertion cleanup (`TestTrackedLaunchCancellation…`, the journal test) is genuine work — whole-snapshot assertions instead of a hardcoded address is the right generalization.

## 6. Architectural notes

- **ARCH-DRY — pass.** `CommitArgs` and `CommitStartClaim` each collapse a real duplication rather than adding a second authority. I considered flagging `detachedResumeProofMatches` / `parkedResumeProofMatches` as near-identical twins and decided against it: the shared prefix is four lines, the observation types genuinely differ, and the symmetry is the documented point.
- **ARCH-PURE — pass, and improved.** Moving the binding proof into the pure projector is the principle correctly applied. `CommitStartClaim` keeps the decision in `AdvanceStartTransaction` and only the write in the store method.
- **ARCH-PURPOSE — flag (I2, I4, C1).** Deletion is proved by absence; seven orphans and a green-reporting conformance target are the absence not being checked. C1 is the sharper version: the subsystem's *invariant* was deleted along with its code.
- **ARCH-MOCK — flag (I6).** Obligations correctly shrank when the policy seam went, but a new external dependency arrived in the same commit without the fake-plus-conformance pair the deleted one had.
- **ARCH-CONSTRAINTS — pass.** M4 removes work from every runtime path (two no-op migration passes at `New`, one `sdlc fleet policy` subprocess per resolution, replaced by a much cheaper `git rev-parse`). No new budget is asserted, so no new measurement is owed. See the minor note on the lost timeout.
- **ARCH-SECURE — flag (I1, I3).** I1 is the principle's own case: an artifact written by an older version of this program is trusted to be well-formed, and when it isn't the failure is fatal at `New` rather than visible and local. I3 puts an unsigned locally-built executable into a base-layer repo that propagates. On the positive side, the token→fingerprint change *improves* the trust story — every `start` arg is `Implicit`, and re-deriving beats trusting a held snapshot, as the code comment argues.

## 7. Plan revision recommendations

Add to `workshop/plans/000170-rescope-couch-to-couch-lite-plan.md` `## Revisions`:

1. **"Task 13 Step 4's rollback was not delivered."** State that the pristine-reservation rollback was dropped rather than carried into `CommitStartClaim`'s call sites, that `DeletePristineThread` and `rollbackUnforkedStart` were left uncalled, and name the replacement (`rollbackPristineStart`) plus the regression test, so the Step's checkbox stops claiming what the code doesn't do.
2. **"The tombstone is a decode shim, not a value shim."** Record that keeping a field decodable is not the same as keeping its value reachable, and that `repo_identity` must be read through from `DeprecatedPolicy` for records written before M4 (I1). This is the correction to the revision already titled *"M4: the deletion sweep is a persisted-schema change"*, whose approach section states "records shed them on their next write" without noting that one of the shed fields was still load-bearing.
3. **"Task 14 Step 5's ledger repair overloaded `absent`."** Note that `absent` now means two things and that the pinned-declaration check must key on `declaration.source`, per the Step's own "do not loosen the pinned contract" instruction (I5).
4. **"D1/D5 left orphans; Task 15 needs a mechanical sweep step."** Record the seven-item enumeration and add the grep-the-whole-tree step (including `Makefile*`, `.github/`, `README.md`) as an explicit task step rather than a lesson (I2, I4).

```findings
findings:
  - id: new
    severity: Critical
    family: deleted-subsystem-drops-its-invariant
    title: |
      Pristine-start rollback is gone; the call left in its place can never fire
    detail: |
      AllocateThreadTag persists a reserved record and claims the artifact, so
      releaseClaimIfThreadAbsent at couch.go:297,301,313 always finds the record
      present and returns nil. A Proc.Current, entropy, or CommitStartClaim
      failure therefore leaks a reserved record that ProjectActionableThreads
      hides and reconcileInterruptedStarts never visits, plus its artifact claim.
      DeletePristineThread and rollbackUnforkedStart now have zero callers, and
      the two tests that pinned the property were deleted with admission. Plan
      Task 13 Step 4 required this rollback explicitly. A test setting
      FakeProcOps.CurrentErr and asserting zero records plus one release fails
      today.
  - id: new
    severity: Important
    family: compat-shim-preserves-shape-not-value
    title: |
      Tombstone keeps old records decodable but drops the value they carry
    detail: |
      A pre-M4 incarnation holds policy.repo_identity and no top-level
      repo_identity; fromPersistedThreadRecord (thread.go:184) never reads the
      tombstone, so RepoIdentity is empty and advanceSuccessfulStart
      (threadstore.go:614) refuses. Reached when an interrupted start written by
      the pre-M4 binary is promoted after the upgrade: reconcileInterruptedStarts
      fails, New returns an error, and couch refuses to start at all -- the same
      whole-store blast radius the manifest tombstone was added to prevent. The
      operator's store has 5 records with policy and 0 open starts today, so it
      does not fire on this host. Fix: read repo_identity out of DeprecatedPolicy
      when the new field is empty, and extend the fixture with a start block.
  - id: new
    severity: Important
    family: deletion-leaves-orphaned-surface
    title: |
      Seven orphans survive the sweep, including a conformance target that now reports green
    detail: |
      Each has exactly one reference, its own definition: threadrecord.PolicyCapacity
      (record.go:34), ThreadStore.DeletePristineThread (threadstore.go:526),
      Couch.rollbackUnforkedStart (couch.go:521), reflectBytesEqual
      (threadstore.go:905), ThreadSnapshotConflictError (threadstore.go:30),
      ThreadSnapshot.manifest/.raw (threadstore.go:39, write-only and copied per
      Snapshot call), and Makefile.local:73 test-couch-policy-live. That last one
      runs `go test -run TestFleetPolicyResolverConformance`, which now prints
      "ok [no tests to run]" and exits 0 -- verified. This is also the 3rd
      finding in family record-claims-unverified-delivery, since atlas/couch.md
      and the issue Log both state the target was deleted. Do not fix only these
      instances. The rule: a deletion is complete when the identifier set is
      swept, not the package -- grep every removed symbol and target name across
      Go, Makefile*, .github/, atlas/ and README.md before the commit, and let no
      doc or log sentence claim a removal grep has not confirmed. lessons.md
      states this rule in this very range and the range violates it, which argues
      for making it mechanical.
  - id: new
    severity: Important
    family: build-artifact-committed
    title: |
      4.5 MB Mach-O arm64 binary `couchstartrecovery` committed at the repo root
    detail: |
      Landed in c11f61ea, almost certainly from `go build ./cmd/probes/couchstartrecovery`
      during the Task 14 Step 6 probe repair; the Makefile target uses `go run`, so
      nothing wants the file. No .gitignore rule covers the repo root. This is the
      ariadne base layer, so the blob propagates to dependent repos and becomes
      permanent history once merged -- the cost of removing it rises sharply at
      this boundary. Drop it from the branch and add a root ignore rule.
  - id: new
    severity: Important
    family: readme-stale-for-shipped-surface
    title: |
      README still documents fleet-policy admission, capacity refusal and provision-worktree
    detail: |
      README.md:307-314 describes behavior D1 deleted. Plan Task 14's own Files
      list names README and it was not touched. This is the 2nd finding in this
      family, so do not fix only the paragraph. The rule: the docs sweep is
      driven by an enumeration derived from the diff -- deleted identifiers plus
      changed user-facing behavior -- grepped against README.md and atlas/
      together, not from recall of which files felt relevant. The atlas half was
      done well here and README was simply forgotten, which is exactly what a
      shared enumeration removes. A grep-based check over the deleted vocabulary
      would close the class.
  - id: new
    severity: Important
    family: guard-weakened-not-repaired
    title: |
      Overloading `absent` silently disables the pinned-declaration check for two entries
    detail: |
      plan_contract_test.go:1481 skips issue151M3PinnedDeclarationExists whenever
      len(declaration.absent) != 0. Adding startgrant.go to the absent list of
      Couch.PrepareStart and Couch.SpawnPrepared therefore turned off a check
      both would still pass, since it reads the frozen git object rather than the
      worktree. Only StartGrantStore, whose source IS the absent file, needs the
      skip. Plan Task 14 Step 5 warned against loosening the pinned contract.
      Gate line 1481 on whether declaration.source itself appears in absent.
  - id: new
    severity: Important
    family: fixture-realism
    title: |
      The new git-common-dir seam has no live conformance case and one modeled answer shape
    detail: |
      resolveRepoIdentity (couch.go:255) keys every saved launch preference off
      `git rev-parse --git-common-dir`, the one value D1 had to preserve
      byte-for-byte. Every fake reply in the tree is ".git" -- the repo-root case.
      Real git answers relative to the CURRENT directory from a subdirectory
      (measured: "../../../.git"), which is what production hits when the
      operator runs couch in a subdir; filepath.Join handles it, but nothing
      tests it and the function's doc comment states the model inaccurately
      ("absolute in a linked worktree"). conformance_live_test.go has no
      git-common-dir case. This is the 2nd finding in this family. The rule: a
      fake's canned replies must cover every input shape production can pass the
      seam -- enumerate the call sites' argument shapes and add a case per shape
      -- and any answer the code reasons about earns a live conformance case
      rather than a comment.
  - id: new
    severity: Minor
    family: doc-comment-misattached
    title: |
      RetireIncarnation's doc comment is now attached to CommitStartClaim
    detail: |
      threadstore.go:405-419 -- the new function was inserted between the comment
      and the function it documents, so godoc attributes RetireIncarnation's
      detach-vs-park rationale to CommitStartClaim and RetireIncarnation
      (threadstore.go:457) has no doc at all.
  - id: new
    severity: Minor
    family: stale-doc-after-new-consumer
    title: |
      GitRunner still documents exactly one call; there are two, and its unused ctx went with the policy seam
    detail: |
      git.go:8-9 says "couch needs exactly one call: rev-parse --show-toplevel".
      Separately, resolveStartResolution's ctx parameter is now unused
      (ResolvePolicy was its only consumer), and the bounded 5 s policy
      subprocess it replaced is now an unbounded exec.Command git call on the
      preview worker. Pre-existing for ResolveTree, so not new exposure, but the
      last bounded IO on that path is gone. Couch.Spawn's doc (couch.go:128) also
      still reads as a production entry point where the plan promised "test seam".
  - id: new
    severity: Minor
    family: schema-looser-than-contract
    title: |
      `start` declares path/worktree/fingerprint Required but not ValueRequired, and worktree is never read
    detail: |
      ops.go:135-139 -- an empty fingerprint passes schema validation and is
      caught only by the later compare. resolveStartResolution uses
      WorkingDir(), which prefers Cwd, so the worktree arg CommitArgs emits is
      inert. Also: unrelated gofmt churn in entrypoint/alias_test.go,
      runtimebundle/store_test.go and two wrapcmd tests adds noise to the
      deletion diff.
```

---

## Re-review — 2026-09-02T22:26:20-07:00 (unknown)

| field | value |
|-------|-------|
| issue | 170 — Rescope couch to couch-lite |
| repo | pair |
| issue file | workshop/issues/000170-rescope-couch-to-couch-lite.md |
| boundary | milestone M4 |
| milestone | M4 |
| window | a89878c31cd7bee06693257e05440c8c4eee7057..bb3dcf55b016b6a5cf7685de297eb86e973bebbf |
| command | sdlc milestone-close --issue 170 --milestone M4 |
| reviewer | claude |
| timestamp | 2026-09-02T22:26:20-07:00 |
| verdict | unknown |

## Review

Failed to authenticate: OAuth session expired and could not be refreshed

---

## Re-review — 2026-09-02T22:41:43-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 170 — Rescope couch to couch-lite |
| repo | pair |
| issue file | workshop/issues/000170-rescope-couch-to-couch-lite.md |
| boundary | milestone M4 |
| milestone | M4 |
| window | a89878c31cd7bee06693257e05440c8c4eee7057..bb3dcf55b016b6a5cf7685de297eb86e973bebbf |
| command | sdlc milestone-close --issue 170 --milestone M4 |
| reviewer | claude |
| timestamp | 2026-09-02T22:41:43-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The M4 deletion sweep is sound and the prior round's Critical is genuinely fixed, not narrated: I reverted `rollbackPristineStart` to the old `releaseClaimIfThreadAbsent` in a scratch copy and `TestStartFailuresAfterTagAllocationRollBackTheReservation` went red on both post-allocation failure modes; I removed the `deprecatedPolicyRepoIdentity` fallback and `TestPreM4StartCarriesItsRepositoryIdentityForward` went red; I restored the deleted make target and the pre-fix README and both new mechanical guards fired. `go test ./...` exits 0 in the real checkout (all failures in the sandboxed run were `ptychild: operation not permitted`, i.e. the sandbox). Nine of ten prior findings are addressed. What keeps this off SHIP: BR-23's time bound was **not** restored — `RunContext` propagates cancellation but nothing sets a deadline, and the CLI start path passes `context.Background()`, so a hung `git rev-parse` is now strictly less bounded than the 5 s `ExecPolicyResolver.Timeout` it replaced; and the two new mechanical guards are each narrower than the rule they were written to encode, which is the same recall-driven gap one level up.

## 1. Strengths

- **The Critical fix is real and pinned at the right level.** `couch.go:559` `rollbackPristineStart` correctly undoes both halves of `AllocateThreadTag` in the right order (record first, then artifact release — `threadtag.go:40,47` confirms the claim precedes the record write), and `pristinerollback_test.go:34` uses a `budgetedReader` so the entropy failure lands *after* allocation rather than passing vacuously. That detail is called out in the test comment; it is exactly the trap that would have made the test green for the wrong reason.
- **`TestEveryMakefileTestSelectorMatchesALiveTest` (`maketargets_test.go:27`) is the right shape of guard** — it names the actual mechanism (`go test -run` matching nothing exits 0) rather than the instance, handles Make's `$$` un-doubling, and fails loudly if the scan itself finds no tests. I added a mutant target and it fired.
- **`StartResolution.CommitArgs` (`startresolution.go:148`) is a genuine ARCH-DRY win found by deleting dead code** — three call sites each rebuilt the commit arguments by hand, and the failure mode (passing the *resolved* agent where the operator gave none → different `AgentSource` → phantom drift) is silent. One owner now, and `MenuFrame` holds the resolution instead of a seven-field shadow (`menu.go:77`).
- **The tombstone work is measured, not reasoned.** `threadrecord/record.go:38,52` and `compat_test.go` pin fixtures copied from the operator's live store with the counts stated (17/17 `claim_generation`, 5/5 `policy`), and `manifestcompat_test.go:22` correctly argues the *manifest* deserves its own guard because its blast radius is the whole store rather than one record.
- **The atlas sweep is thorough and historicised properly** — `atlas/couch.md:153,437,674,707` all state the deletions in the past tense with the issue reference, and `index.md`/`architecture.md`/`session-identity.md` were swept for the same vocabulary.

## 2. Critical findings

None.

## 3. Important findings

**(a) `ThreadStore.DeleteUnstartedThread` is orphaned by this round's own deletion — `threadstore.go:533`.** Deleting `rollbackUnforkedStart` removed its only production caller. It now has exactly one reference outside its own definition, in `starttransaction_integration_test.go:58`, and its doc comment still reads "rolls back only the exact creating claim that reached **admission**". **This is the 2nd finding in family `deletion-leaves-orphaned-surface`** — do not just delete this symbol. BR-17's rule ("a deletion is complete when the identifier set is swept") was mechanised for *docs* only; the Go half stayed recall-driven and immediately regressed in the commit that closed BR-17. Make it mechanical: a `deadcode`/`staticcheck U1000` gate over `cmd/`, or a test in the same spirit as `maketargets_test.go` that fails when a `couchcore` production symbol has zero non-test references outside its declaration, with an explicit allowlist for the documented exceptions (`RecoverStoreJournal`, `Couch.Spawn`).

**(b) The new docs guard's file set omits Go sources; two live-voice admission comments survive — `registry.go:19-20`, `conformance_live_test.go:242-243`.** Both describe "normalized provider evidence"/"normalized provider keys" as the *current* admission authority, and the conformance comment's stated consequence ("both could then never host agents concurrently") is the behaviour D1 deleted. `deletedvocabulary_test.go:27-46` targets `README.md`, `atlas/couch.md`, `Makefile*`, `.github/workflows` — a hand-listed subset, while BR-17's stated rule was "grep every removed symbol and target name across **Go**, Makefile\*, .github/, atlas/ and README.md". **This is the 3rd finding in family `readme-stale-for-shipped-surface`.** State and encode the rule: the enumeration's file set is *every tracked text file* (walk from the repo root, skipping `.git`/`workshop/history`), not a per-row list of files someone remembered — a per-row file list reintroduces the exact recall step the guard exists to remove.

**(c) The docs guard matches per line, so a multi-word term wrapped across a line break can never fire — `deletedvocabulary_test.go:67`.** Measured: restoring the pre-fix `README.md` produced hits for `provision-worktree` and `test-couch-policy-live`, but **not** for `sdlc fleet policy`, because the historical text wrapped as ``(`sdlc`` / ``fleet policy`)``. Markdown prose wraps, so every multi-word row in this table is silently optional. **This is the 2nd finding in family `assertion-admits-vacuous-pass`.** The rule: a text guard over prose must match against whitespace-normalised content, not raw lines — join the file, collapse runs of whitespace, then search, and derive the reported line number from the match offset. Otherwise the guard's own regression test (adding a row) proves nothing about whether that row can fire.

**(d) The "restored bound" is cancellation, not a deadline — `git.go:26-31`, `console_menu.go:238`, `couchcmd/run.go:267,282`.** The deleted `ExecPolicyResolver` applied `defaultPolicyTimeout = 5 * time.Second` internally (`policyresolver_exec.go:12,53` at base). The replacement adds `RunContext`, but `startMenuPreview` uses `context.WithCancel(c.lifetime)` with no deadline and the CLI path passes `context.Background()` outright — `grep WithTimeout|WithDeadline` over `couchcore`+`couchtty` finds only `park.go:505` and `launch_existing.go:90`. So a hung `git rev-parse --git-common-dir` blocks an armed submit indefinitely and hangs `couch start <path>` forever, where the deleted code returned in 5 s. `TestRepoIdentityResolutionHonoursCancellation` passes with no deadline anywhere in the tree, so it pins propagation and reads as pinning the bound. **This is the 2nd finding in family `envelope-claim-unmeasured`.** The rule: an operating-envelope claim in a doc comment, commit message or plan Revision must name the *mechanism* that enforces it and be pinned by a test that reddens when that mechanism is removed — "carries a context" is not a bound. Concretely: give the identity resolution its own `context.WithTimeout` at the seam (or at `PrepareStart`/`SpawnPrepared`), and assert the deadline fires against a fake that blocks.

**(e) The plan still says M4 is unbuilt, and one step still describes surface the fix round resurrected — `workshop/plans/000170-…-plan.md:458-474, 494-520`.** All 19 M4 checkboxes are `- [ ]` while the work is committed; the Core concepts tables carry `planned (M4)` for `Admission`/`PolicyResult`/`StartGrantStore`/`MigrateLegacyRecord`/`Actor`/`TreeSummary`/`ActorView` and for the `PolicyResolver` seam, all of which are deleted in the tree; and Task 13 Step 4 (line 497) still asserts `CommitStartClaim` "replaces `CommitThreadReplacements` **and `DeletePristineThread`**, whose only callers were admission" — `DeletePristineThread` is now `rollbackPristineStart`'s core (`couch.go:560`). The plan's own convention (Revisions, line ~744) makes that status column a build tracker: "flipping a row at its milestone is what turns its assertion on". The couchtty contract only pins rows whose declared path is in `cmd/internal/couchtty/`, so the couchcore rows are unasserted and will stay stale silently. This may be in flight as part of Task 15 — dispose it if so.

**(f) Task 15 Step 0's peer-repo note was not filed.** The plan requires the M4 close to record that couch, the only *programmatic* consumer of ariadne's `sdlc fleet policy --path P --json`, no longer calls it, and to let ariadne decide. `../ariadne/cmd/sdlc/internal/fleet/` is still present; grepping `../ariadne/workshop/issues/` and this repo's project/issue files finds no such note (`workshop/projects/couch.md:910` mentions the provider only as something couch deleted). This is exactly the "leaving the cross-repo consequence unstated is how a surface rots" case the step was written for, and it is one file to write.

## 4. Minor findings

- **`.gitignore:44-49` enumerates main packages by hand and already misses two of eight.** `grep -rl '^package main'` finds `cmd/internal/runtimebundle/generatecmd` and `cmd/internal/workbenchshortcut/generatecmd`; neither `/generatecmd` entry is present, so the comment's claim ("named after the main packages") is already false. **2nd in family `build-artifact-committed`** — derive the list in a test rather than extending it.
- **This round's lessons live only in the issue `## Log`, not `workshop/lessons.md`.** The best sentence produced by the round — "when deleting a subsystem, enumerate the *invariants* it enforced, not only the symbols it defined; a deleted guarantee leaves no compile error" — plus the tombstone-must-be-read and `-run`-matches-nothing lessons are in the issue file, which is archived to `workshop/history/` and which AGENTS.md tells the next agent not to read. **2nd in family `lesson-not-recorded-for-boundary-defect`** — the rule: the commit that disposes boundary findings adds one `lessons.md` line per finding *class* in the same commit, so the write is part of the fix rather than a later recall step.
- `resume.go:208-211` and `223-226` compute the same `agent` from `thread.LatestLaunchProfile` twice, the second shadowing the first. Trivial; noting rather than raising.
- `CommitArgs` sends `r.CanonicalPath` as `path` while `ops.go:135` marks it `Required` without `ValueRequired`; an empty `path` still falls through to `WorkingDir()` and surfaces as `ErrStartResolutionChanged` — the same misdiagnosis the fingerprint guard was added to avoid, on the one remaining argument.

## 5. Test coverage notes

- The rollback test covers two of the three post-allocation failure sites; the `CommitStartClaim`-failure site (`couch.go:339`) is unexercised. It is safe by construction (CAS-atomic, so the record stays pristine and `DeletePristineThread` succeeds), so this is a gap in evidence rather than in behaviour.
- `livePreM4RecordWithOpenStart` has a `start` block but no `launch_profile`, and `advanceSuccessfulStart` only reaches the empty-identity refusal when `profile != nil` (`threadstore.go:610`). So the fixture pins the *fix* (value carried forward through `fromPersistedThreadRecord`) but not the *outage* the plan Revision describes. The choke point is covered, so I did not raise it; adding `launch_profile` to that fixture would close the loop for one line.
- `pathpreferencekey_test.go` pinning the digest of a filename that exists in the operator's live store is the right kind of characterization test — it cannot be satisfied by restating the implementation.

## 6. Architectural notes

- **ARCH-DRY — pass.** `CommitArgs` consolidates three hand-rolled arg maps; `CommitStartClaim` reuses `AdvanceStartTransaction` and the existing journal write path; the `GitRunner` doc now records the second call *and* explicitly re-states that the third-consumer threshold for lifting the seam has not moved. Only the trivial `resume.go` shadow above.
- **ARCH-PURE — pass.** `ResolveStartResolution` stays a pure function over `RepoIdentity string`; the git call sits in the `Couch` shell and is injected. `pristinerollback_test.go`'s identity tests construct a bare `&Couch{Git: NewFakeGit(...)}` with no store, no filesystem — that is the pure boundary behaving as advertised.
- **ARCH-PURPOSE — flag** (findings a, b). The shadow-sweep for a deletion is "every consumer of the removed vocabulary derives from its absence". Docs got a mechanism; Go symbols and Go comments did not, and both regressed in the same commit that installed the mechanism.
- **ARCH-MOCK — pass.** `FakeGit.RunContext` honours cancellation before answering, so the fake can prove propagation; `TestGitConformance_LinkedWorktree` now measures all three real `--git-common-dir` shapes against production `resolveRepoIdentity`, and the unit test cans all three so the fake stops modelling only the repo-root case. The deleted seam took its fake and its conformance target with it, which is the correct direction.
- **ARCH-CONSTRAINTS — flag** (finding d). Also worth carrying forward: identity resolution is now an uncached subprocess per preview generation. Cheaper than the `sdlc` subprocess it replaced, so no regression, but it is on a keystroke-adjacent path with no memoisation by `{dir}`.
- **ARCH-SECURE — pass.** `deprecatedPolicyRepoIdentity` (`thread.go:298`) parses persisted, possibly hand-edited data best-effort and degrades to `""`, which `advanceSuccessfulStart` turns into a visible refusal rather than a fabricated identity — the right failure shape. No credentials in the diff; fixtures are anonymised except the preference-key test, where the real path *is* the pinned input.

## 7. Plan revision recommendations

Add one `## Revisions` entry dated 2026-09-02 to `workshop/plans/000170-rescope-couch-to-couch-lite-plan.md` covering:

1. **Tick Chunk 4.** Flip the 19 M4 checkboxes in Tasks 13–15 that are delivered, and flip the Core concepts `Status` column from `planned (M4)` to `deleted` for `Admission`/`AdmissionDecision`/`CapacityExceededError`/`PolicyResult`, `StartGrantStore`/`StartGrantToken`, `MigrateLegacyRecord`, `Actor`, `TreeSummary`/`ActorView`, and the `PolicyResolver` seam; `ThreadIncarnation.Policy → RepoIdentity` and `ThreadStore.CommitStartClaim` become `new`/`modified`.
2. **Correct Task 13 Step 4.** `CommitStartClaim` replaced `CommitThreadReplacements` only. `DeletePristineThread` survives and is now the core of `rollbackPristineStart`; the row `ThreadStore.CommitThreadReplacements, DeletePristineThread` must be split so it stops claiming a deletion the code contradicts.
3. **Record that BR-23's time bound was not delivered.** The Revisions text currently says "so a hung git no longer hangs the start form" — state instead that cancellation was added and a deadline was not, name the CLI's `context.Background()` path, and either schedule the timeout or record the accepted exposure explicitly.
4. **Record the two guards' declared scope.** Both new guards are narrower than the rules they encode (line-oriented matching; a hand-listed file set that excludes Go). Note the intended end state so the next reader does not mistake them for complete.
