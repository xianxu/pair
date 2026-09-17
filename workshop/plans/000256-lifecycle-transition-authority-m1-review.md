# Boundary Review — pair#256 (milestone M1)

| field | value |
|-------|-------|
| issue | 256 — Enforce lifecycle transition authority and outcome uncertainty |
| repo | pair |
| issue file | workshop/issues/000256-lifecycle-transition-authority.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | b6a0766ac340596f2f5889f6a183cfcb9f5795ed..0d736147913b9e502c36b95c6c63873e14ad1435 |
| command | sdlc milestone-close --issue 256 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-09-17T08:50:28-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The re-cut is the right architecture and most of it landed cleanly: `ClassifyThread` is now a pure, total function of `(record, evidence)` whose branch table reads off the code, `SessionUnresolved` is fail-closed by construction (zero value), the session question costs one host-wide `list-sessions` instead of a per-candidate `list-clients`, and the ~20 restated test expectations carry their reasons rather than being weakened. The re-adoption fix at `resume.go:466` is genuinely covered — I reverted it in a scratch copy and `couchcmd TestRecoveryMenuReachesTerminalAfterActualHelperDeath/warm` goes red. What blocks SHIP is that the "one class, three sites" sweep stopped one site short: a record with an **open park transaction** now classifies `detached`, is ranked highest by `SelectResumableRoot`, passes `DecideResume`, and then hard-errors inside `RetireIncarnation` with a raw store message — which, because startup has no fallback (`startup.go:203-211`), refuses `couch` in that tree entirely. I reproduced this end to end. Secondary but cheap: a single `list-sessions` failure now demotes every *parked* row (which has durable cold-resume authority and needs no session answer) to `unusable/unknown`; the new production IO seam `SessionPresence` has no test at all; and the docs half of the boundary is incomplete — README and three atlas passages still present the retired vocabulary as current, and two comment edits Task 2 explicitly owned were not made.

---

### 1. Strengths

- **`sessionevidence.go:24-31` — `SessionUnresolved` as the zero value**, with `TestUnresolvedIsTheZeroValue` pinning it. This is fail-closed *by construction* rather than by remembering to set a status flag, and it is why deleting `DetachedStatus` didn't cost anything. Confirmed-good pattern; reuse it for the M3 three-valued liveness work.
- **`artifactcollision.go:~290 resolveScopedBindings`** — the two session questions (presence, detached) now share one index read, so the name a thread is *judged* by and the name it is *acted* on cannot drift. Correct ARCH-DRY call, and the right half to extract.
- **The test restatements are honest.** `actionableinventory_test.go:403-426` and `resume_test.go:52-62,176-183` rename each case to say *the old assertion was the bug*, and none was weakened to pass. `classify_test.go:24-28` replacing a name-matched ratchet exception with a `newlyActionable` field is exactly right — a guard keyed to prose stops guarding the moment someone renames a case.
- **The re-adoption is reachable and pinned.** Verified by reverting `resume.go:466-470` in a scratch copy: `TestRecoveryMenuReachesTerminalAfterActualHelperDeath/warm` fails. The `Dead`-only gate at `resume.go:564` is the right rule (see finding 3 for its missing test).
- **`atlas/couch.md:1277-1360`** is a genuinely good write-up of the measurement and the rule — it explains *why* PPID 1 makes the deletion sound, not just that it happened.

### 2. Critical findings

**C1 — `actionableinventory.go:328` / `resume.go:466` + `threadstore.go:551`: an open park makes the new re-adoption path hard-error, and startup has no fallback.** (`classification-not-authority`)

`RetireIncarnation` refuses outright when `next.Park != nil` (`threadstore.go:551-553`), but `retireDeadIncarnationBeforeStart` (`resume.go:555-563`) never checks for a park, and `ResumeContextWith` returns the store's error verbatim (`resume.go:466-468`) instead of a `refuseResume` diagnostic. Reproduced with a scratch test against the real store:

```
park phase = requested
classify            = detached/""
DecideResume        = permitted
retireDeadIncarnationBeforeStart = "cannot retire an incarnation while a park transaction is open"
```

Failure scenario: a thread is parked, couch dies before the park completes (this shape is #271's own live fixture `couch-e1a31510b7033d08`, which sat open for ~18 hours across restarts), and the zellij session survives because `zellij delete-session` never ran. The record now carries `Park != nil` + a stale `live` incarnation + a present session. `ClassifyThread` → `detached`; `SelectResumableRoot` ranks `detached` highest; `startup.go:203` auto-resumes it; the retire errors; `startupResumeRefusal` (`startup.go:217-232`) only decorates errors carrying a `ResumeDiagnosticCode`, and a bare `errors.New` from the store has none — so `couch` in that tree refuses to start with an internal store message and no next step. Before M1 that record classified `busy`, was not resumable, and startup spawned normally. This is a new, worse instance of the exact anti-pattern the issue exists to remove, and `classify_test.go` ("park in flight whose session is still up" → `ThreadDetached`, `newlyActionable`) deliberately admits the shape, so the action path must handle it.

Fix sketch: the class is "a guard reading bookkeeping the classification no longer trusts", and the enumeration the Log wrote has four members, not three. Either (a) have `retireDeadIncarnationBeforeStart` retire the dead incarnation *and* abandon the orphaned park when the park's own `{PID, ProcessIdentity}` is confirmed `Dead` (it is copied from the incarnation, so one probe answers both), or (b) at minimum convert any error out of the re-adoption into a `refuseResume(...)` with a diagnostic code so startup degrades with guidance instead of wedging. Add a regression test for both the classify→resume chain on a park-open record and the startup path. Record the fourth site in the plan's `## Revisions`.

### 3. Important findings

**I1 — `actionableinventory.go:321-324`: `SessionUnresolved` masks a parked thread's durable cold-resume authority.** (`unknown-must-not-mask-durable-authority`)

The `SessionUnresolved` refusal sits *above* the `record.VerifiedPark` branch. Confirmed by scratch test on a fully-proved parked record:

```
absent session     -> parked/""
unresolved session -> unusable/"unknown"
```

Failure scenario: `zellij list-sessions` fails once (or one scope's `session-names.jsonl` is unreadable, or a name is contested) and `gatherThreadEvidence` silently swallows the error (`actionableinventory.go:~535`) — every parked row in the store becomes `unusable/unknown`, even though its resume authority (verified park + resolvable native id) is durable and does not depend on the session existing at all. Before this change a parked record was never a detach candidate and never consulted the session question. Fail-closed is the right instinct where the unknown answer could flip an actionable verdict to a destructive one; here the only two verdicts the session answer can produce for a `VerifiedPark` record are `detached` and `parked`, both actionable, so refusing both is strictly worse than taking the durable one.

Fix sketch: move the `record.VerifiedPark != nil` cold-resume branch above the `SessionUnresolved` case (keep `SessionPresent` first, so a surviving session still wins). Add a case to `everyThreadShape` for parked + unresolved. Note the plan's branch table encodes the current order, so it needs a `## Revisions` entry either way.

**I2 — `resume.go:564`: the fail-closed half of the re-adoption has no test.** (`fail-closed-guard-untested`)

Mutation-checked: changing `!= Dead` to `== Live` — i.e. retiring on an **Unknown** probe — produces no additional test failure across `couchcore`, `couchtty` and `couchcmd`. This is the guard whose own comment says retiring an incarnation whose process is actually alive "would abandon a running agent", and it is the Done-when the issue states ("Unknown observations cannot become confirmed absence or authorize destructive recovery"). Fix: a table test over `FakeProcOps` {Dead, Unknown, Live} asserting the record is untouched for the latter two.

**I3 — `artifactcollision.go:370-410`: the production `SessionPresence` seam is untested.** (`production-seam-only-tested-through-fake`)

Every session-presence test runs against `FakeThreadArtifactCollisionChecker`. Nothing exercises the real `ScopedThreadArtifactCollisionChecker.SessionPresence`, including the branch that decides archive-eligibility: an address in a **readable** scope with no index row → `SessionAbsent`, an address whose scope could **not** be read → left out of the map → `SessionUnresolved` (`artifactcollision.go:394-408`). The fake unconditionally answers `SessionAbsent` for anything unset, so it cannot catch that branch inverting or the `readable` map being dropped. The harness already exists and is cheap: `sandboxedChecker` (`artifactcollision_zellij_test.go:29`) stubs `zellij` on `PATH` and is what `TestDetachedSessionsBindsNothingForAnUnreadableScope` uses. ARCH-MOCK: the fake is stateful and correctly coupled (`SetDetachedSession` also sets presence — good), but a fake with no conformance check against the production implementation is one modelled world, not two agreeing ones.

**I4 — `atlas/couch.md` contradicts itself; the retired vocabulary is still documented as current.** (`atlas-contradicts-code`)

The new section at `:1277` says `stale-incarnation` and `unrecorded-child` are retired, while three earlier passages still present them as live:
- `:584-585` — the "closed vocabulary" list still enumerates both. The same paragraph (`:589-592`) still says "`ThreadEvidence` carries a `ProofStatus` per question", which is now true only of `Parked`.
- `:963-965` — "A stale `IncarnationLive` whose helper is no longer hosted shows as `unusable/stale-incarnation`" — it now shows as `detached`, or `session-gone` when the session is also gone. This is the exact sentence #272 disproves.
- `:251` — "which is the stale-incarnation shape by construction" (passing reference).

AGENTS.md §8 asks the atlas to be *current*, not appended to. Update the three passages (or make them point at the new section).

**I5 — `README.md:517-527` documents removed surface.** (`readme-documents-removed-surface`)

`:527` lists `stale — helper ownership unresolved` among the reason labels an operator sees; that label was deleted in this window. `:517-518` — "Unknown ownership, active clients, and **open start or park transactions** leave a diagnostic instead of guessing that a process died" — is now false for park transactions: nothing in the classification path reads `record.Park`. This is the class the Docs update gate exists to catch at the earliest boundary rather than at the merge-time `specs` judge.

**I6 — `ThreadBusy`'s referent changed but five wording sites still say "park".** (`stale-wording-after-referent-change`)

The plan's Task 2 consumer table calls this out as "the failure mode a rename would have caught", and lists two of these as M1 deliverables. Neither was made:
- `actionableinventory.go:20-22` — the `ThreadBusy` constant's own doc still reads *"a park transaction in flight: not actionable, but not broken either, and it resolves on its own"*. Task 2 Step 3 says to delete it. It now contradicts `startClaimed`'s doc 320 lines below in the same file.
- `layout.go:79-80` — `holdsSession`'s comment still says *"Busy is included because a park in flight can still fail"*. Task 2 lists this file as "comment only" work.

The remaining three are M2/Task 4's and can wait, but note them so the sweep is one enumeration: `menu.go:1234-1239` (busy → `name`/`describe` only, behind the same disproved "it resolves on its own"), `menu_render.go:432` (`"parking…"`), `couchcmd/run.go:770` (`"parking in progress"`). Note also that Task 4's stated premise — *"After Task 2 that row is no longer `busy`, so the branch is dead code"* — is now factually wrong: `startClaimed` keeps `ThreadBusy` reachable, so Task 4 must be re-scoped, not just executed.

**I7 — `actionableinventory.go:288,341-355`: `startClaimed` reads durable state where the plan specified an in-memory observation, with no `## Revisions` entry.** (`plan-code-divergence`)

The plan is explicit: *"`startInFlight` is deleted, not narrowed… It arrives as an observation alongside `Live`… rather than being inferred from a durable `creating` incarnation that outlives the process it describes. Ephemeral state stays ephemeral."* The branch table's row 3 reads "(in-memory observation)". The code instead reads `Incarnation.Start != nil` off the durable record. The Log documents the choice and the justification is sound (without *some* busy branch, a normally-starting thread classifies `session-gone`), but the plan was never revised, so it now claims something the code does not do — and the residual risk is the one the plan was guarding against: `ReconcileStart` keeps a claim `StartKeepOccupied` whenever helper or registration evidence is `Unknown` (`starttransaction.go:186-210`), so such a record reads `busy` indefinitely and `menu.go:1234` offers it only `name`/`describe` — #271's wedge in a new field. Either plumb the observation as designed, or append a `## Revisions` entry adopting the durable read and state the bound (`reconcileInterruptedStarts` runs in `New()`, `couch.go:153`) — and make sure Task 4 gives the busy row an escape.

### 4. Minor findings

- `threadreason_test.go:67-70` — `TestStaleLabelDoesNotClaimSupervisorDied` was repurposed into `ReasonSessionGone.Label() == "session gone"`; its name and failure message still say "stale" and its stated purpose has no subject left. Delete it (`TestEveryReasonHasADistinctOperatorLabel` covers what remains).
- `sessionevidence_test.go:85-93` — `TestAbsentBindingIsAbsentNotUnresolved` asserts the *opposite* of its name (an address absent from the projector's input reads **Unresolved**). Rename to match, e.g. `TestProjectionAnswersOnlyForBindingsItWasGiven`.
- ARCH-DRY: `ProjectSessionPresence` (`sessionevidence.go:95-118`) and `ProjectDetachedSessions` (`detachedsessions.go:62-82`) now duplicate the name-index/ambiguity loop *and* the fail-closed predicate `name == "" || claims[name] != 1 || ambiguous[name]`. The diff correctly extracted the shared **read**; the shared **rule** is still copy-pasted, and divergence in it is silent. Extract `indexSessionsByName(sessions)` plus one `uniquelyClaimed(binding, claims, ambiguous)` helper.
- `resume.go:466-470` — the retire/`CommitStartClaim` pair is two writes with no compensation. Benign today (retiring a provably-dead incarnation is independently correct), but worth one sentence in the comment so the next reader does not assume atomicity.
- `artifactpath/manifest.go:664` — `sessionevidence.go` inserted between `detachedsessions.go` and `git.go`; the list is not sorted anyway, so this is only a note.

### 5. Test coverage notes

- **Covered well:** the two filed bugs have named, measured tests (`TestWedgedParkDoesNotWedgeClassification`, `TestDeadLauncherWithLiveSessionIsDetached`), the crash/clean-detach *identity* claim is asserted rather than asserted twice, the IO budget is pinned (`SessionPresenceQueries() == 1`, `DetachedQueries() == 0` across four suites), and the fail-closed refresh is pinned (`TestSessionPresenceFailureLeavesEveryThreadUnresolved`).
- **Gaps:** I2 (Unknown must not retire — mutation-verified unpinned), I3 (production `SessionPresence` untested), C1 (no test for the classify→resume chain on a park-open record; no startup-path test), I1 (no case for parked + unresolved session).
- **Environment:** `go test ./cmd/internal/couchcore/ ./cmd/internal/couchtty/ ./cmd/internal/couchcmd/` in this session fails only on pty/`ptychild` "operation not permitted" and `mkdir /tmp/pcnotify-*`, all of which reproduce identically at the base commit — no logic failures. `go vet ./cmd/...` is clean apart from one pre-existing unrelated diagnostic in `pairlifecycletest`.

### 6. Architectural notes for upcoming work

- **ARCH-DRY** — flag (Minor, above). The `resolveScopedBindings` extraction is the good half; finish it on the pure side.
- **ARCH-PURE** — pass. `ClassifyThread`, `ProjectSessionPresence` and `startClaimed` are pure and tested with no IO; the seam is an interface on `c.Artifacts` with a fake. The one impurity added, `retireDeadIncarnationBeforeStart`, sits on `*Couch` (the shell) and calls a named store transition rather than mutating fields — correct placement, and it pre-pays M3's Task 11.
- **ARCH-PURPOSE** — flag (C1). The Log's shadow-sweep enumerated three sites of "a guard reading bookkeeping the classification no longer trusts" and stopped. `RetireIncarnation`'s open-park precondition is the fourth, and it is reachable *because of* the fix for site 3. This is the instance-vs-class failure the principle names; the enumeration wants writing down, not extending one more time.
- **ARCH-MOCK** — flag (I3). Seam and stateful fake exist and the fake correctly couples detached⇒present; what is missing is any exercise of the production implementation through the existing stubbed-`zellij` harness.
- **ARCH-CONSTRAINTS** — flag (soft). Two new per-refresh costs went in without measurement: (a) `list-sessions` now runs on **every** refresh unconditionally, where `gatherThreadEvidence` previously skipped it when there were no detach candidates ("a couch with nothing detachable pays nothing"); (b) `resumeShaped` widened (`actionableinventory.go:~458`), so `Physical` is now called for every profiled record including live ones — the test's expected count went 3 → 4 and the comment that bounded it is gone. The plan budgeted both and asked for the figures in `## Log`; M2's operator verification owns that, so this is a reminder, not a blocker. Net direction is good: `list-clients` left the refresh entirely.
- **ARCH-SECURE** — pass. No credentials; the only external input is `zellij list-sessions`, parsed in `launcher` and consumed here with fail-closed rules on contested and duplicated names. `SessionUnresolved`-as-zero-value is the "invalid state unrepresentable" move applied to an observation.
- **ARCH-ORDER** — flag (I7, and a note). The classifier now holds no state between events (pure function of record + evidence) — a real improvement, correctly claimed. But `startClaimed` re-imports durable in-flight state into classification, and the legal-combination question it raises (`Start != nil` × `ReconcileStart` outcomes × what the menu offers) is not written down anywhere. The uncertainty handling in `retireDeadIncarnationBeforeStart` is right — `Dead` only, never `Unknown` — but it is untested (I2), which is precisely the "green run is a sample of size one" case: the only ordering exercised is the one the author happened to write.
- **ARCH-FUNERAL** — pass. Creates nothing durable; `SessionObservation` dies with its refresh. Two reason values were removed from a computed vocabulary, and `ThreadReason` is never persisted (only serialized in `couch --list`/`--show` output), so no migration is owed — worth one line in the close evidence saying so.

### 7. Plan revision recommendations

Append a `## Revisions` entry to `workshop/plans/000256-lifecycle-transition-authority-plan.md` covering:

1. **The class has four sites, not three.** `RetireIncarnation`'s open-park precondition (`threadstore.go:551`) is reachable from the new re-adoption and must be swept in the same round (C1). Add the task, and state which layer abandons the orphaned park.
2. **`ThreadBusy` is produced by a durable `ThreadStartClaim`, not an in-memory observation.** This contradicts *"Ephemeral state stays ephemeral"* and the branch table's row 3 (I7). Record the reasoning from the Log, and state the bound (`reconcileInterruptedStarts` at `couch.go:153`) plus what happens when `ReconcileStart` holds a claim on Unknown evidence.
3. **Branch order: cold-resume authority before the `SessionUnresolved` refusal** (I1), or an explicit justification for masking a parked row on an unanswerable question.
4. **Task 4's premise is wrong.** *"After Task 2 that row is no longer `busy`, so the branch is dead code"* — `ThreadBusy` is still reachable. Re-scope Task 4 to give the busy row an escape hatch rather than to delete a branch as dead.
5. **Task 2's two comment deliverables are not done** (`actionableinventory.go:20-22`, `layout.go:79-80`) — either land them or move them explicitly (I6).
6. **Naming/location corrections:** the plan's integration table names `observeSessions`; the code ships `SessionPresence` + `ProjectSessionPresence`. The Pure-entities table omits `startClaimed`. `AllThreadStates` is listed in `actionableinventory.go` in the table but `threadreason.go` in Task 8 Step 3.
7. The durable plan's M1 task checkboxes (Tasks 1-3) are still `- [ ]` while the issue file's M1 row is `[x]`; tick them or note why.

```findings
findings:
  - id: new
    severity: Critical
    family: classification-not-authority
    title: |
      An open park makes the new re-adoption hard-error and wedges couch startup
    detail: |
      A record with Park != nil, a dead incarnation and a surviving session now classifies detached (actionableinventory.go:316), passes DecideResume, then hits RetireIncarnation's park precondition (threadstore.go:551) via retireDeadIncarnationBeforeStart (resume.go:555-568), which returns the raw store error. startup.go:203-232 has no fallback and only decorates errors carrying a ResumeDiagnosticCode, so couch refuses to start in that tree with an internal message. Reproduced end to end against the real store. This is the fourth site of the class the Log enumerated as three.
  - id: new
    severity: Important
    family: unknown-must-not-mask-durable-authority
    title: |
      SessionUnresolved is checked above VerifiedPark, so one failed list-sessions hides every parked row
    detail: |
      actionableinventory.go:321-324 returns unusable/unknown before the cold-resume branch at :328. Verified: a fully-proved parked record classifies parked under SessionAbsent and unusable/unknown under SessionUnresolved. A parked record's resume authority is durable and needs no session answer, and the only two verdicts the session could produce are parked and detached, both actionable.
  - id: new
    severity: Important
    family: fail-closed-guard-untested
    title: |
      Nothing pins that an Unknown liveness probe must not retire an incarnation
    detail: |
      Mutating resume.go:564 from `!= Dead` to `== Live` (retire on Unknown) produces no test failure across couchcore, couchtty and couchcmd. This is the guard preventing a resume from abandoning a running agent, and it is the issue's own Done-when about Unknown never authorizing destructive recovery.
  - id: new
    severity: Important
    family: production-seam-only-tested-through-fake
    title: |
      ScopedThreadArtifactCollisionChecker.SessionPresence has no test
    detail: |
      Every presence test runs against the fake. The production implementation (artifactcollision.go:370-410), including the readable-scope-no-binding then SessionAbsent versus unreadable-scope then SessionUnresolved branch that decides archive-eligibility, is unexercised. The sandboxedChecker harness (artifactcollision_zellij_test.go:29) already stubs zellij for exactly this.
  - id: new
    severity: Important
    family: atlas-contradicts-code
    title: |
      Three atlas passages still document the retired reason vocabulary as current
    detail: |
      atlas/couch.md:584-585 lists stale-incarnation and unrecorded-child in the closed vocabulary and says ThreadEvidence carries a ProofStatus per question; :963-965 says a stale IncarnationLive shows as unusable/stale-incarnation, the exact sentence this issue disproves; :251 references the shape in passing. The new section at :1277 says the opposite.
  - id: new
    severity: Important
    family: readme-documents-removed-surface
    title: |
      README still lists a deleted reason label and claims parks leave a diagnostic
    detail: |
      README.md:527 names `stale — helper ownership unresolved` among the labels an operator sees; that label was deleted here. README.md:517-518 says open start or park transactions leave a diagnostic instead of guessing a process died, which is no longer true of parks since nothing in the classification path reads record.Park.
  - id: new
    severity: Important
    family: stale-wording-after-referent-change
    title: |
      ThreadBusy changed meaning but five wording sites still describe a park
    detail: |
      Task 2 owned two of these and neither was made: actionableinventory.go:20-22 (the constant's own doc still says "a park transaction in flight ... it resolves on its own") and layout.go:79-80 (holdsSession's rationale). Three more are M2/Task 4's: menu.go:1234-1239, menu_render.go:432 ("parking…"), couchcmd/run.go:770 ("parking in progress"). Task 4's premise that the busy row is now unreachable is also wrong.
  - id: new
    severity: Important
    family: plan-code-divergence
    title: |
      startClaimed reads durable state where the plan specified an in-memory observation
    detail: |
      actionableinventory.go:288,341-355 reads Incarnation.Start off the record; the plan says ephemeral state stays ephemeral and the branch table row 3 says "(in-memory observation)". No ## Revisions entry records the change. Residual risk: ReconcileStart keeps a claim occupied on Unknown evidence (starttransaction.go:186-210), so such a record reads busy indefinitely and menu.go:1234 offers it only name/describe.
  - id: new
    severity: Minor
    family: test-name-contradicts-assertion
    title: |
      Two tests assert something other than what their names say
    detail: |
      threadreason_test.go:67-70 (TestStaleLabelDoesNotClaimSupervisorDied now checks ReasonSessionGone's label; its subject is gone) and sessionevidence_test.go:85-93 (TestAbsentBindingIsAbsentNotUnresolved asserts Unresolved).
  - id: new
    severity: Minor
    family: duplicated-fail-closed-rule
    title: |
      The session name-index and uniqueness predicate are copy-pasted across two projectors
    detail: |
      sessionevidence.go:95-118 and detachedsessions.go:62-82 duplicate the ambiguity loop and `name == "" || claims[name] != 1 || ambiguous[name]`. The diff extracted the shared READ (resolveScopedBindings) but not the shared RULE; divergence in it would be silent.
```
