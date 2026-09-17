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

---

## Re-review — 2026-09-17T09:33:58-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 256 — Enforce lifecycle transition authority and outcome uncertainty |
| repo | pair |
| issue file | workshop/issues/000256-lifecycle-transition-authority.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | b6a0766ac340596f2f5889f6a183cfcb9f5795ed..0ee032e8b46ebe8340666970ac467d635356484d |
| command | sdlc milestone-close --issue 256 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-09-17T09:33:58-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

Round 1's ten findings are all genuinely disposed, and eight of them are mutation-verified rather than taken on the commit message's word: the orphaned-park abandon, the `Dead`-only re-adoption gate, the `VerifiedPark`-above-`SessionUnresolved` reorder, and the production `SessionPresence` readable/unreadable branch each go red when I revert them in a scratch copy. The docs half is finished and I checked the atlas vocabulary list against `AllThreadReasons()` and the README labels against `Label()` — they agree. What blocks SHIP is that the C1 sweep again fixed the sites the finding named rather than the class: `CommitStartClaim`'s incarnation precondition is still live for every *non-`Dead`* incarnation, and I reproduced end to end that a record with an unobservable launcher plus a surviving session classifies `detached`, is auto-selected at startup, and refuses `couch` in the whole tree with `thread {…} already has 1 incarnation(s)` — the identical wedge, from the branch the `Dead`-only guard deliberately creates. The same scenario succeeds at the base commit, so it is a regression this milestone introduces. Second: the new park abandon runs before `RetireIncarnation`'s own preconditions are checked, so for an `IncarnationUnknown` record (which `soleParkableIncarnation` explicitly permits parking) the park is tombstoned irreversibly and the retire then refuses forever, leaving a thread that can be neither resumed nor archived.

### 1. Strengths

- **The mutation checks are real.** `TestOrphanedParkDoesNotWedgeTheResumeChain` (`cmd/internal/couchcore/sessionevidence_test.go:415`), `TestUnknownLivenessNeverRetiresAnIncarnation` (`:330`), `TestParkedRowSurvivesAnUnresolvedSessionQuestion` (`:375`) and `TestSessionPresenceAnswersThroughTheProductionChecker` (`cmd/internal/couchcore/artifactcollision_zellij_test.go:415`) each fail when I invert the guard they claim to pin. That is four for four on the round's behavioural claims.
- **`cmd/internal/couchcore/actionableinventory.go:330-346`** — the reorder ships with the *reasoning* in the code, not just the branch: "the only two verdicts a session answer can produce for a verified-park record are `detached` and `parked`, both actionable". I checked the third outcome (`ReasonBindingLost`) and it is documented "RECOVERABLE: never retire one", so the reorder cannot widen archive eligibility.
- **`cmd/internal/couchcore/sessionevidence.go:132-159`** — the shared *rule*, not just the shared read. I diffed the before/after of both projectors by hand: `!uniquelyClaimed(...)` is exactly the old `name == "" || claims[name] != 1 || ambiguous[name]`, and `ProjectDetachedSessions`' state map still keeps the first occurrence. Behaviour-preserving extraction.
- **`cmd/internal/couchcore/resume.go:584-600`** names its own non-atomicity in the comment instead of pretending the pair is a transaction — which is what let me find finding 2 by reading it.
- **`cmd/internal/couchcore/artifactcollision_zellij_test.go:456`** distinguishes "readable scope, no index row — asked, and there is none" from "scope could not be read — never asked" *against the production checker*. That is the branch the fake could not model, and it is now two agreeing worlds rather than one.

### 2. Critical findings

**C-A — `cmd/internal/couchcore/resume.go:472` (+ `threadstore.go:506`): the class has a fifth site, and it is the one the `Dead`-only gate creates.** (`classification-not-authority`, 2nd in family)

> **This is the 2nd finding in family `classification-not-authority`.** Earlier rounds fixed instances. Do NOT fix this instance — state the rule that covers all of them, and fix that.

Reproduced end to end in a scratch copy, through the real `StartInteractive`:

```
classify couch-cf90ba5103786743 -> detached/""
StartInteractive err = thread {RepoScope:816fc349d3faebf8 Tag:couch-cf90ba5103786743} already has 1 incarnation(s)
```
The same fixture at base `b6a0766` returns `StartInteractive err = <nil>` — it spawns normally. Shape: one incarnation, `Proc.Exists` returns `Unknown`, session present. `ObserveRecordedProcesses` skips non-`Live`, so `evidence.Live` is empty → `detached`; `SelectResumableRoot` ranks it highest; `DecideResume` permits (`resume_test.go:59` now asserts "unknown incarnation does not veto"); `retireDeadIncarnationBeforeStart` correctly declines (not `Dead`); `CommitStartClaim` refuses with a bare `fmt.Errorf`; `startupResumeRefusal` only decorates coded errors, so it passes through. `len(Incarnations) != 1` reaches the same place.

Two rules, and both are one-place fixes:
1. **No error may leave `ResumeContextWith` without a `ResumeDiagnosticCode`** — decorate at that boundary, not per call site. Today `CommitStartClaim`, `resolveRepoIdentity`, `c.Proc.Current()`, `allocateStartNonce`, the `DetachedSessions` observe error and `"native binding resolver is unavailable"` all escape uncoded. (Alternatively give `startupResumeRefusal` a generic next-step arm for uncoded errors; `warmresume_test.go:162` currently pins pass-through, so that test states the choice.)
2. **The enumeration is not "four sites", it is "every guard that refuses on a record's incarnation or park".** `CommitStartClaim`'s `len(next.Incarnations) != 0` precondition is one, and it is live precisely on the non-`Dead` branch M1 just pinned as correct. Write the enumeration down and sweep it, rather than extending it a fifth time.

Add a startup-path regression test — `startupFixture` + a `SetUnknown` pid reproduces it in ~20 lines.

**C-B — `cmd/internal/couchcore/resume.go:580-600`: the park is abandoned before the retirement's own preconditions are checked, so a failure destroys it permanently.** (`irreversible-step-before-precondition`)

`RetireIncarnation` (`threadstore.go:557`) refuses `incarnation.State != IncarnationLive`, but `retireDeadIncarnationBeforeStart` only screens `Start != nil`, `PID <= 0` and `Identity == ""`. For an `IncarnationUnknown` record the abandon lands first and cannot be undone. Scratch reproduction against the real store:

```
retire err = resume-not-running: stale incarnation could not be retired: retire needs a live incarnation, found "unknown"
park after = <nil>, incarnations = 1, tombstones = 1
```

Reachable: `soleParkableIncarnation` (`park.go:797`) explicitly accepts `IncarnationUnknown`, and `markLiveRecordUnknown` (`couch.go:648`) produces that state whenever a start reached a live incarnation but couch's own attachment did not commit. After this the thread cannot resume (the retire refuses on every retry — the comment's "safe to repeat" holds only when the retirement is *possible*) and cannot archive (`thread.go:360` refuses an unknown helper). That is a new instance of the exact wedge #256 exists to remove, and it costs the `#275` audit trail.

Fix sketch: hoist `RetireIncarnation`'s preconditions ahead of the abandon — return `(nil, nil)` or a coded refusal when `incarnation.State != IncarnationLive`, so the irreversible write only runs once the write it unblocks is known to be admissible. Regression test: the fixture above, asserting `after.Park != nil` when the retire cannot succeed.

Related, same write (ARCH-ORDER): this `AbandonPark` goes direct to the store, bypassing `PairLifecycleController`'s per-thread park worker (`park.go:501`), which is how every other abandon (`park.go:448`) is serialised. CAS prevents a lost update, but the `RecoverActiveParks` goroutine (`couchcmd/run.go:348`) can be mid-`reconcileActive` on the same record during an interactive resume. Worth one sentence saying why the queue is not needed here, or route through it.

### 3. Important findings

**I-A — the two remaining guards inside `retireDeadIncarnationBeforeStart` are unpinned.** (`fail-closed-guard-untested`, 2nd in family)

> **This is the 2nd finding in family `fail-closed-guard-untested`.** Earlier rounds fixed instances. Do NOT fix this instance — state the rule that covers all of them, and fix that.

Measured prevalence in this one function: three fail-closed/degradation guards, one pinned. Reverting `resume.go:601` from `refuseResume(ResumeNotRunning, ...)` to `return nil, err` produces **no** failure anywhere in `couchcore` — `TestResumeFailuresCarryADiagnosticCode` only exercises the `AbandonPark` arm. The park-identity-mismatch refusal (`resume.go:581-586`) has no fixture at all: both park tests set `Park.Identity` equal to the incarnation.

The rule, not the two sites: **every exit from `retireDeadIncarnationBeforeStart` is a fail-closed decision and belongs in one table test** over `{no park, matching park, foreign park} × {Dead, Unknown, Live} × {Live, Unknown incarnation}`, asserting the record's durable state and `ResumeDiagnosticOf(err) != ""` for each. That table also gives C-A and C-B their regressions, and it fails whenever a new exit is added without one.

### 4. Minor findings

- `cmd/internal/couchcmd/run_test.go:1238` — the assertion now checks `ReasonSessionGone` but the failure message still reads *"want unusable/stale-incarnation after the child exited"*. (`test-name-contradicts-assertion`, **2nd in family** — the rule is *a test's prose (name, comment, failure message) must name what it asserts*; sweep it by grepping the diff's touched test files for `stale`/`unrecorded` in strings, which finds this one and nothing else.)
- `cmd/internal/couchcore/artifactcollision.go:262-293` — `DetachedSessions`' doc comment was orphaned by the `resolveScopedBindings` extraction: it now runs straight into `resolveScopedBindings`' own doc with no blank line, so godoc attaches the whole block to the private helper, the fail-closed paragraph appears **twice verbatim**, and `DetachedSessions` (`:408`) has no doc at all. (`stale-wording-after-referent-change`, **2nd in family** — rule: *when a function moves or is split, its doc comment moves with it*; the cheap sweep is `gofmt`-adjacent — grep the diff for a comment block immediately followed by another `//` block with no intervening blank line.)
- The plan's Integration table names `observeSessions` in `sessionevidence.go`; the code ships `SessionPresenceResolver.SessionPresence` (production impl in `artifactcollision.go:370`) plus `ProjectSessionPresence`. `startClaimed`, `indexSessionsByName`, `uniquelyClaimed` and `retireDeadIncarnationBeforeStart` are in no entity table. (`plan-code-divergence`, **2nd in family** — rule: *at each milestone close, re-derive the Core-concepts tables by grepping every row's name and path, and record divergence in `## Revisions`*; round 1 raised this only as a recommendation and it survived.)
- Three resume tests (`TestUnknownLivenessNeverRetiresAnIncarnation`, `TestOrphanedParkDoesNotWedgeTheResumeChain`, `TestResumeFailuresCarryADiagnosticCode`) live in `sessionevidence_test.go` and exercise `resume.go`; they belong in `resume_test.go`.
- `FakeThreadArtifactCollisionChecker.SetDetachedSession(addr, "")` clears presence too, so a test that sets presence then clears a detached session silently loses both.

### 5. Test coverage notes

- **Full-suite state:** `go test ./cmd/internal/couchcore/ ./cmd/internal/couchtty/ ./cmd/internal/couchcmd/` fails only on pty/`ptychild` "operation not permitted" and `mkdir /tmp/pcnotify-*`. I enumerated every `--- FAIL` line: 24 of 24 are that environment class, zero logic failures.
- **Newly covered and verified:** the four mutation checks above. `TestSessionPresenceCountsNoClients` is a good addition — it pins the optimistic-inventory trade at the *production* seam, not just the fake.
- **Gaps:** C-A (no startup-path test for an uncoded resume error; the `Unknown` + surviving-session shape is untested end to end), C-B (no test that a failed retirement leaves the park intact), I-A (two of three guards in one function unpinned).
- Round 1's "add a case to `everyThreadShape` for parked + unresolved" was answered with a standalone mutation-proven test instead. That is fine — the standalone test is stronger.

### 6. Architectural notes for upcoming work

- **ARCH-DRY — pass.** The rule is extracted and I verified the extraction is behaviour-preserving in both projectors. Only the comment-block duplication above remains, and that is prose.
- **ARCH-PURE — pass.** `ClassifyThread`, `startClaimed`, `ProjectSessionPresence`, `indexSessionsByName`, `uniquelyClaimed` are pure and tested with no IO. `retireDeadIncarnationBeforeStart` sits on `*Couch` and calls named store transitions rather than mutating fields — right placement, and it pre-pays M3's Task 11.
- **ARCH-PURPOSE — flag (C-A).** Same failure mode as round 1, one layer out: the Log enumerated three sites, round 1 found a fourth, and the fourth's fix left the fifth. The enumeration still has not been *written* — it is being discovered one reviewer at a time. Write "every guard that refuses on `Incarnations` or `Park`" as a list in the plan and sweep it whole.
- **ARCH-MOCK — pass.** Production `SessionPresence` is now exercised through the stubbed-`zellij` harness including the archive-deciding branch; the fake couples detached ⇒ present. No live conformance check, but `sandboxedChecker` is the standing seam for one when #276 lands.
- **ARCH-CONSTRAINTS — pass with note.** `SessionPresenceQueries() == 1` is pinned across four suites and `DetachedQueries() == 0` in the refresh; `list-clients` has left the refresh entirely. The per-refresh `list-sessions` and the widened `Physical` call remain unmeasured — M2's operator verification owns that.
- **ARCH-SECURE — pass.** No credentials. The only untrusted input is `zellij list-sessions`, parsed in `launcher` and consumed fail-closed on contested and duplicated names; `SessionUnresolved`-as-zero-value keeps "could not ask" unrepresentable as absence.
- **ARCH-ORDER — flag (C-B).** The classifier holding no state between events is a real gain. But the new two-write sequence models the *crash* interleaving and not the *refusal* one, and it performs the irreversible write first. The knowledge model is also asymmetric: `Dead` gates the retirement but nothing gates the abandon, even though both read the same probe.
- **ARCH-FUNERAL — pass.** Nothing durable is created. `ParkHistory` gains a new automatic writer (startup resume), but growth is one tombstone per orphaned park, unchanged per event.

### 7. Plan revision recommendations

1. **The class is not "four sites".** Append the *enumeration* — every guard refusing on `record.Incarnations` or `record.Park` — and record that `CommitStartClaim`'s own precondition is live for the non-`Dead` case the M1 gate creates (C-A). State which layer owns making resume failures coded.
2. **Record the abandon's ordering constraint** (C-B): the park abandon must not precede the retirement's preconditions, and say what the plan expects for an `IncarnationUnknown` record carrying an open park.
3. **Correct the Integration table**: `observeSessions` → `SessionPresenceResolver.SessionPresence` (`artifactcollision.go`) + `ProjectSessionPresence` (`sessionevidence.go`); add `startClaimed`, `indexSessionsByName`, `uniquelyClaimed`, `retireDeadIncarnationBeforeStart` to the Pure/Integration tables.
4. Tick the durable plan's M1 task checkboxes (Tasks 1-3 are still `- [ ]` while the issue file's M1 row is `[x]`) — round 1 asked for this and it was not done.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Mutation-verified: disabling the park-abandon block reds TestOrphanedParkDoesNotWedgeTheResumeChain; un-wrapping the AbandonPark error reds TestResumeFailuresCarryADiagnosticCode. See new finding for the fifth site of the same class.
  - id: BR-2
    disposition: addressed
    note: |
      Mutation-verified: moving the SessionUnresolved arm back above the VerifiedPark branch reds TestParkedRowSurvivesAnUnresolvedSessionQuestion/session_question_failed. Checked the third outcome (ReasonBindingLost) is not archive-eligible.
  - id: BR-3
    disposition: addressed
    note: |
      Mutation-verified: `!= Dead` -> `== Live` reds TestUnknownLivenessNeverRetiresAnIncarnation/unknown_must_not_retire.
  - id: BR-4
    disposition: addressed
    note: |
      Mutation-verified: dropping the readable[scope] check reds TestSessionPresenceAnswersThroughTheProductionChecker on the unreadable-scope case.
  - id: BR-5
    disposition: addressed
    note: |
      atlas/couch.md:251, :579-592 and :961-968 corrected; the vocabulary list now matches AllThreadReasons() element for element. Remaining mentions are the retirement narrative itself.
  - id: BR-6
    disposition: addressed
    note: |
      README.md:517-531 corrected; every label it now names exists in ThreadReason.Label(), and the park claim matches ClassifyThread no longer reading record.Park.
  - id: BR-7
    disposition: addressed
    note: |
      Four of five made (actionableinventory.go:21-30, layout.go:79-82, menu_render.go:432-434, couchcmd/run.go:771). menu.go:1234-1239 remains, which the finding itself scoped to M2/Task 4; the plan's Revisions re-scopes Task 4 off its false premise.
  - id: BR-8
    disposition: addressed
    note: |
      Plan ## Revisions now adopts the durable read explicitly, states the bound (reconcileInterruptedStarts at couch.New) and the Unknown-evidence residual risk, and re-scopes Task 4.
  - id: BR-9
    disposition: addressed
    note: |
      TestStaleLabelDoesNotClaimSupervisorDied deleted; the projector test renamed to TestProjectionAnswersOnlyForBindingsItWasGiven. A third instance of the same rule is raised below.
  - id: BR-10
    disposition: addressed
    note: |
      indexSessionsByName + uniquelyClaimed extracted and used by both projectors; I diffed both call sites by hand and the predicate is exactly equivalent to the two copies.
findings:
  - id: new
    severity: Critical
    family: classification-not-authority
    title: |
      An Unknown-liveness incarnation with a surviving session wedges couch startup in the whole tree
    detail: |
      2nd in family — fix the RULE, not this site. Reproduced end to end through StartInteractive: a record whose launcher is unobservable plus a live session classifies detached, is ranked highest by SelectResumableRoot, passes DecideResume, is correctly declined by the Dead-only retire gate, and then CommitStartClaim (threadstore.go:506) refuses with a bare fmt.Errorf. startupResumeRefusal only decorates coded errors, so couch refuses to start with "thread {...} already has 1 incarnation(s)". The identical fixture at base b6a0766 spawns normally, so this is a regression from this milestone. Two one-place rules — every error leaving ResumeContextWith must carry a ResumeDiagnosticCode (CommitStartClaim, resolveRepoIdentity, Proc.Current, allocateStartNonce, the DetachedSessions observe error and the missing-resolver error all escape uncoded today), and the class enumeration must be written down as "every guard refusing on record.Incarnations or record.Park", which includes CommitStartClaim's own precondition on the non-Dead branch.
  - id: new
    severity: Critical
    family: irreversible-step-before-precondition
    title: |
      The park abandon runs before RetireIncarnation's preconditions, so a failure destroys the park permanently
    detail: |
      retireDeadIncarnationBeforeStart (resume.go:580) screens Start/PID/Identity but not incarnation.State, while RetireIncarnation (threadstore.go:557) refuses anything other than IncarnationLive. Reproduced against the real store with an IncarnationUnknown record carrying a matching open park and a dead process: the abandon lands (park nil, one tombstone appended), the retire then fails with "retire needs a live incarnation, found unknown", and every retry repeats it. The thread can no longer resume and cannot archive either (thread.go:360 refuses an unknown helper) — a new instance of the wedge this issue exists to remove, plus loss of the #275 audit trail. Reachable: soleParkableIncarnation (park.go:797) explicitly permits parking an unknown incarnation, and markLiveRecordUnknown (couch.go:648) produces that state. Fix: hoist the retirement's preconditions ahead of the irreversible write. Related ARCH-ORDER note: this AbandonPark bypasses PairLifecycleController's per-thread park worker, unlike every other abandon.
  - id: new
    severity: Important
    family: fail-closed-guard-untested
    title: |
      Two of the three guards in retireDeadIncarnationBeforeStart are unpinned
    detail: |
      2nd in family — fix the RULE, not these sites. Mutation-checked: reverting resume.go:601 from refuseResume(ResumeNotRunning, ...) to `return nil, err` produces no failure anywhere in couchcore, because TestResumeFailuresCarryADiagnosticCode only reaches the AbandonPark arm. The park-identity-mismatch refusal (resume.go:581-586) has no fixture at all — both park tests give the park the same identity as the incarnation. Measured prevalence: three fail-closed exits in one function, one pinned. The rule is one table test over {no park, matching park, foreign park} x {Dead, Unknown, Live} x {Live, Unknown incarnation} asserting durable state plus a non-empty ResumeDiagnosticOf for every refusal — which also gives the two Critical findings their regressions and fails whenever a new exit is added.
  - id: new
    severity: Minor
    family: test-name-contradicts-assertion
    title: |
      run_test.go failure message still says stale-incarnation while the assertion checks session-gone
    detail: |
      2nd in family — state the rule. cmd/internal/couchcmd/run_test.go:1238 reads "want unusable/stale-incarnation after the child exited" under an assertion on ReasonSessionGone. Rule: a test's prose — name, comment and failure message — must name what it asserts. Sweep by grepping the window's touched test files for "stale"/"unrecorded" inside string literals; that finds this one and nothing else.
  - id: new
    severity: Minor
    family: stale-wording-after-referent-change
    title: |
      DetachedSessions' doc comment was orphaned by the resolveScopedBindings extraction
    detail: |
      2nd in family — state the rule. artifactcollision.go:262-278 is DetachedSessions' doc (cost model, "Pinned by TestDetachedSessionsBindsNothingForAnUnreadableScope") but now runs straight into resolveScopedBindings' own doc with no blank line, so godoc attaches the whole block to the private helper, the fail-closed paragraph appears twice verbatim, and DetachedSessions at :408 has no doc at all. Confirmed against the base version, where the comment sat on DetachedSessions. Rule: when a function moves or is split, its doc comment moves with it.
  - id: new
    severity: Minor
    family: plan-code-divergence
    title: |
      The plan's Integration and Pure tables still name entities the code does not ship
    detail: |
      2nd in family — state the rule. The Integration table names `observeSessions` in sessionevidence.go; the code ships SessionPresenceResolver.SessionPresence (production impl in artifactcollision.go:370) plus ProjectSessionPresence. startClaimed, indexSessionsByName, uniquelyClaimed and retireDeadIncarnationBeforeStart appear in no table. Round 1 raised this as a recommendation and it survived. Rule: at each milestone close, re-derive the Core-concepts tables by grepping every row's name and path, and record any divergence in ## Revisions rather than leaving the plan claiming what the code does not deliver.
```

---

## Re-review — 2026-09-17T10:07:45-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 256 — Enforce lifecycle transition authority and outcome uncertainty |
| repo | pair |
| issue file | workshop/issues/000256-lifecycle-transition-authority.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | b6a0766ac340596f2f5889f6a183cfcb9f5795ed..317732f906e1196054905262ac33ab8e0fad8d26 |
| command | sdlc milestone-close --issue 256 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-09-17T10:07:45-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

All three round-2 Criticals are genuinely closed, and I verified each by mutation in a scratch copy of the pinned head rather than by reading the commit message: deleting the deferred coder in `ResumeContextWith` reddens `TestEveryResumeFailureCarriesADiagnosticCode` on both exits, and moving the `incarnation.State` guard back below the park abandon reddens `TestReAdoptionExitsAreTotalAndCoded/matching-park/dead/unknown` with exactly the "tombstone is permanent" message. The round-3 findings are all consequences of RULE 1's blanket wrap and of enumerations that were extended in code but not in their other homes — none is a wedge, a crash or a data-loss path, so nothing here blocks the boundary; they are cheap and should land before it.

## 1. Strengths

- **The three rules are real rules, not three more patches.** The coder lives at the function boundary (`resume.go:360-365`), so `resolveRepoIdentity`, `Proc.Current`, `allocateStartNonce`, `CommitStartClaim` and the observe errors are covered by construction rather than by an enumeration that keeps missing one. That is the correct response to "not converging".
- **The precondition hoist is complete, not just the one the finding named.** `retireDeadIncarnationBeforeStart` (`resume.go:573-645`) now screens *every* precondition `RetireIncarnation` enforces — count, `Start`, `State`, exact `{PID, Identity}` — before the irreversible `AbandonPark`, and it uses the post-abandon revision for the retire CAS.
- **The table asserts outcomes, not self-consistency.** `sessionevidence_test.go:363` pins `wantRetire := live == dead && state == IncarnationLive` per cell plus "refusal must not have destroyed the park", which is what makes it fail when a new exit appears. The commit's own note that the first version of the table was not mutation-proven is accurate and was fixed.
- **Deleting the unreachable park-identity guard rather than testing it** (`resume.go:617-625`) is the right call, and the comment records *why* it is unrepresentable (`validateLifecycle` + exactly-one incarnation) with the failed fixture attempt as evidence.
- **ARCH-CONSTRAINTS is a net win, and pinned.** The refresh's per-candidate `list-clients` is gone; `TestSessionPresenceCountsNoClients` and `layout_guard_test.go:113-123` pin one host-wide call and zero client queries.

## 2. Critical findings

None.

## 3. Important findings

**I1 — `menu_reattach.go:239-252`: the blanket coder changed what "carries a code" means, and the reattach pass still branches on the old meaning.** *(3rd in family `stale-wording-after-referent-change` — do not fix this site; fix the rule.)* Before RULE 1, `ResumeDiagnosticOf(err) != ""` meant "this is a refusal"; cell 7's comment states that contract explicitly — *"A failure that is not one (a spawn error, a registration timeout) carries only its text, so its row shows that text's first line (decision 11)."* After `resume.go:360-365`, every non-cancellation error out of `ResumeContextWith` carries `resume-unknown`, so `code == ""` is dead for this path and a background reattach that fails on a registration timeout now renders `reattach failed: resume-unknown` (`menu_render.go:710`) instead of the error's first line. The wrap also rebuilds the error from `retErr.Error()`, discarding the chain — `errors.As`, `errors.Is(…, context.DeadlineExceeded)` and `Unwrap() []error` (which `console_completion.go:133` walks) no longer see through it. **The rule:** when you change what a value *means* — not its spelling — enumerate every reader of it and re-derive each, in the same round. The enumeration here is five sites: `grep -rn 'ResumeDiagnosticOf\|\.Diagnostic' --include='*.go'` → `resume.go:138`, `relaunch.go:122`, `startup.go:230`, `console.go:1767`, `menu_reattach.go:239`. The same unswept-referent rule leaves a second live instance in this window: `actionableinventory.go:472-474` still asserts *"a record carrying an incarnation never reached either before, and must not start to"* immediately above the code that now does exactly that.

**I2 — `actionableinventory.go:555`: the new `SessionPresenceResolver` seam is reached only through a silently-failing type assertion.** *(2nd in family `production-seam-only-tested-through-fake` — state the rule.)* `c.Artifacts.(SessionPresenceResolver)` drops the `ok` into an `if`, there is no `var _ SessionPresenceResolver = ScopedThreadArtifactCollisionChecker{}`, and no test constructs a `Couch` over the production checker — `TestSessionPresenceAnswersThroughTheProductionChecker` calls the method directly, and the couchcmd acceptance runs on `rt.artifacts` (the fake). I confirmed both types satisfy the interface *today* by compiling an assertion in a scratch copy, so nothing is broken now; the exposure is that a receiver or signature drift compiles, passes every test, and silently turns every row in every tree into `unusable/checking…`. **The rule:** every interface consumed via a runtime assertion on `c.Artifacts` carries a compile-time `var _ I = ScopedThreadArtifactCollisionChecker{}` alongside the existing fake assertion. Measured prevalence: five such interfaces (`contextPairSessionObserver`, `PairSessionIO`, `DetachedSessionResolver`, `NativeBindingResolver`, `SessionPresenceResolver`); only `PairSessionIO` is pinned (`park.go:812`).

**I3 — `atlas/couch.md:1328`: the atlas records "One class, three sites" while the code records four.** *(2nd in family `atlas-contradicts-code` — state the rule.)* `resume.go:615` says *"This is the FOURTH site of the class… The enumeration is now: ClassifyThread, DecideResume, CommitStartClaim's caller, and this"*, and the plan's `## Revisions` (2026-09-17 round 1) says "The class has FOUR sites, not three". The atlas section written for this milestone still lists three, omitting the one that carries the irreversible-ordering rule — and the issue's `## Log` says three as well. **The rule:** when a boundary round extends an enumeration, every home of that enumeration (code comment, plan Revisions, atlas, `## Log`) is updated in the same round; the atlas is not a milestone-end sweep. Round 1 updated three of the four homes.

## 4. Minor findings

- `classify_test.go:268-276`: `TestClassifyThreadAcceptsExactlyWhatTheOldProjectorAccepted`, its doc (*"except #248 intentionally admits unbound warm sessions"*) and its failure message (`previously %v`, printing `wasActionableBefore` alone) all describe a narrower assertion than the body, which now admits four `newlyActionable` shapes. *(3rd in family `test-name-contradicts-assertion`.)*
- `resume.go:627`: this `AbandonPark` is the only park abandon that bypasses `PairLifecycleController.submit`'s per-thread serialization (`park.go:435-450`); the revision CAS keeps it safe, but there are now two authorities for one durable transition (ARCH-ORDER).
- `resume.go:16`: `ResumeCreating` lost its only producer when `occupiedResumeCode` was deleted. `ThreadReason` has `TestEveryReasonIsProducedBySomeShape` for exactly this; `ResumeDiagnosticCode` has no equivalent guard, which is why nothing noticed.
- `artifactcollision.go:261`: BR-15's fix deleted `resolveScopedBindings`' own doc along with the orphaned one, and `DetachedSessions`' new doc now points at it for "the per-scope fail-closed rule" — which is no longer stated there.
- `actionableinventory.go:560`: `presenceErr` is discarded with no trace or carried field, unlike `PathError` on the same struct; a host-wide zellij failure renders every row `checking…` with the cause recorded nowhere.

## 5. Test coverage notes

The mutation checks I ran: (a) removing the deferred coder → `TestEveryResumeFailureCarriesADiagnosticCode` fails on both exits with the uncoded store text; (b) relocating the `State != IncarnationLive` guard below `AbandonPark` → the table fails on `matching-park/dead/unknown` with the park-destroyed assertion. Both regressions are real. One dimension of `retireDeadIncarnationBeforeStart` is outside the table: the `incarnation.Start != nil` early return is unpinned (removing it broke no test), though it is nearly behaviour-neutral because `Start != nil` implies `State == IncarnationCreating` and the state guard catches the same records one exit later — worth folding into the table as a `{start-claimed}` column rather than a separate finding. The two-write sequence (abandon, then retire) has no seam to inject a crash between them, so the "safe to repeat" claim in the comment is reasoned rather than exercised; the reasoning checks out by inspection (a repeat sees `Park == nil`, `State == Live`, process `Dead`, and retires).

## 6. Architectural notes for upcoming work

ARCH-DRY **pass** — `indexSessionsByName`/`uniquelyClaimed` now serve both projectors and `detachedsessions.go:62-80` is byte-equivalent to its old duplicate. ARCH-PURE **pass** — `ProjectSessionPresence`, `ClassifyThread` and `startClaimed` run with no IO; `retireDeadIncarnationBeforeStart` is tested against the real store rather than a mock. ARCH-PURPOSE **flag (I3)** — the shadow-sweep over the class is complete in code but not in the atlas. ARCH-MOCK **flag (I2)** — the fake/production conformance test exists (`artifactcollision_zellij_test.go:409`) and `SetDetachedSession` now sets presence so the fake cannot model an impossible host, but the assertion binding production to the interface is missing. ARCH-CONSTRAINTS **pass with a note** — the widened `resumeShaped` predicate now runs `Path.Physical` for every profiled record each refresh, where it previously ran only for detach candidates; cheap per call, but it is the one cost this diff *adds* and it is unmeasured. ARCH-SECURE **pass** — the unreadable-scope and contested-name paths fail closed and are tested against real on-disk fixtures. ARCH-ORDER **pass with the Minor above**; the `{park} × {liveness} × {state}` table is the transition enumeration this entry asks for. ARCH-FUNERAL **pass with a note** — `retireDeadIncarnationBeforeStart` is a new writer to the unbounded `ParkHistory`, bounded at one tombstone per orphaned park.

For M2: the last of BR-7's five wording sites (`menu.go:1234-1239`, still reading "mid-park" and "It resolves on its own" — the phrase `actionableinventory.go:27` calls "the bug stated as a comment") is deliberately deferred to Task 4 and recorded in the plan's Revisions, along with the re-scope of Task 4's false premise. That is tracked, not lost, but the busy row carries a *behavioural* consequence of the same false premise — no archive offer — and `ReconcileStart` keeps a claim occupied on Unknown evidence, so the escape M2 owes it is the substantive half.

## 7. Plan revision recommendations

The plan's tables and Revisions match the code at this head; no revision is needed for the plan itself. I3's correction belongs in `atlas/couch.md` and the issue's `## Log`, not in the plan.

```findings
dispose:
  - id: BR-11
    disposition: addressed
    note: |
      Mutation-proven: deleting the deferred coder at resume.go:360 reddens both exits of TestEveryResumeFailureCarriesADiagnosticCode. See I1 for the consumer the widened meaning broke.
  - id: BR-12
    disposition: addressed
    note: |
      Mutation-proven: moving the State guard below AbandonPark reddens matching-park/dead/unknown with the permanent-tombstone assertion. Bypass note carried forward as a Minor.
  - id: BR-13
    disposition: addressed
    note: |
      Twelve cells asserting wantRetire, a non-empty diagnostic on every refusal, and park survival on refusal; the start-in-flight exit remains outside the table (see test notes).
  - id: BR-14
    disposition: addressed
    note: |
      run_test.go:1239 now reads session-gone; swept the window's touched test files for stale/unrecorded string literals and the remaining hits are ordinary prose.
  - id: BR-15
    disposition: addressed
    note: |
      DetachedSessions has its own doc at artifactcollision.go:374 and the duplicated fail-closed paragraph is gone; residual noted as a Minor (resolveScopedBindings now has no doc at all).
  - id: BR-16
    disposition: addressed
    note: |
      Tables re-derived; observeSessions removed, startClaimed / resolveScopedBindings / retireDeadIncarnationBeforeStart added, and every row's name and path greps to the stated file.
findings:
  - id: new
    severity: Important
    family: stale-wording-after-referent-change
    title: |
      The blanket resume coder changed what "carries a code" means and the reattach pass still branches on the old meaning
    detail: |
      3rd in family — fix the RULE, not this site. resume.go:360-365 now gives every non-cancellation error a ResumeDiagnosticCode, so "carries a code" changed from "is a refusal" to "came out of resume". menu_reattach.go:244-252 still reads it the old way: its cell-7 branch falls back to firstErrorLine only when the code is empty, so a background reattach that fails on a spawn error or registration timeout now renders "reattach failed: resume-unknown" (menu_render.go:710) instead of the error's first line, which decision 11 specified. The wrap also rebuilds the error from retErr.Error(), so errors.As, errors.Is(..., context.DeadlineExceeded) and Unwrap() []error no longer see through it (console_completion.go:133 walks joined errors). The rule: when a value's MEANING changes, enumerate every reader and re-derive each in the same round. The enumeration is five sites — resume.go:138, relaunch.go:122, startup.go:230, console.go:1767, menu_reattach.go:239. A second live instance of the same rule: actionableinventory.go:472-474 still asserts that a record carrying an incarnation "must not start to" be physicalized, directly above the code that now does.
  - id: new
    severity: Important
    family: production-seam-only-tested-through-fake
    title: |
      SessionPresenceResolver is reached only through a silently-failing type assertion with no compile-time binding
    detail: |
      2nd in family — state the rule. actionableinventory.go:555 does c.Artifacts.(SessionPresenceResolver) and drops the ok, there is no var _ SessionPresenceResolver = ScopedThreadArtifactCollisionChecker{}, and no test builds a Couch over the production checker (TestSessionPresenceAnswersThroughTheProductionChecker calls the method directly; the couchcmd acceptance uses the fake). Both types satisfy the interface today — verified by compiling the assertion in a scratch copy — so nothing is broken now. The exposure is that a receiver or signature drift compiles, passes every test, and turns every row in every tree into unusable/checking…, because the fake still satisfies it. The rule: every interface consumed via a runtime assertion on c.Artifacts carries a compile-time var _ against the production type, beside the existing fake assertion. Measured prevalence: five such interfaces, one pinned (PairSessionIO at park.go:812).
  - id: new
    severity: Important
    family: atlas-contradicts-code
    title: |
      The atlas records "One class, three sites" while the code and the plan record four
    detail: |
      2nd in family — state the rule. atlas/couch.md:1328 heads the section "One class, three sites" and lists ClassifyThread, DecideResume and CommitStartClaim's caller. resume.go:615 says "This is the FOURTH site of the class" and names the enumeration including the orphaned-park abandon, and the plan's Revisions (2026-09-17 round 1) says "The class has FOUR sites, not three". The issue's Log still says three as well. The omitted member is the one carrying the irreversible-ordering rule, so a reader taking the atlas as the map gets the enumeration minus its most dangerous entry. The rule: when a boundary round extends an enumeration, every home of it — code comment, plan Revisions, atlas, issue Log — is updated in that same round, not at a milestone-end sweep. Round 1 updated three of the four homes.
  - id: new
    severity: Minor
    family: test-name-contradicts-assertion
    title: |
      TestClassifyThreadAcceptsExactlyWhatTheOldProjectorAccepted now admits four deliberately new shapes
    detail: |
      3rd in family. classify_test.go:268-276: the name says "exactly what the old projector accepted", the doc says "except #248 intentionally admits unbound warm sessions", and the failure message prints tc.wasActionableBefore alone — while the body asserts (wasActionableBefore || newlyActionable) over four #256 shapes. The rule is BR-14's, widened: a test's name, doc and failure message must all describe the assertion the body makes, and when the assertion's scope changes all three are restated in the same edit.
  - id: new
    severity: Minor
    family: transition-bypasses-its-owner
    title: |
      The re-adoption's AbandonPark bypasses the per-thread park worker every other abandon goes through
    detail: |
      resume.go:627 calls c.Threads.AbandonPark directly; PairLifecycleController.Abandon (park.go:435-450) routes the same store call through submit, which serializes per thread. The revision CAS keeps this safe — a loser gets a coded refusal, not corruption — but park abandonment now has two authorities, and RecoverActiveParks (run.go:348) runs concurrently with startup resume over the same records (ARCH-ORDER).
  - id: new
    severity: Minor
    family: vocabulary-entry-without-producer
    title: |
      ResumeCreating lost its only producer and the resume diagnostic vocabulary has no produced-by guard
    detail: |
      Deleting occupiedResumeCode removed the only site emitting ResumeCreating (resume.go:16); ResumeLive survives via relaunch.go:107. ThreadReason has TestEveryReasonIsProducedBySomeShape for exactly this class — threadreason.go's own comment cites it as the reason unrecorded-child was deleted rather than kept as a placeholder — but ResumeDiagnosticCode has no equivalent guard, which is why the orphan went unnoticed in the same commit that deleted its producer.
```

---

## Re-review — 2026-09-17T10:47:04-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 256 — Enforce lifecycle transition authority and outcome uncertainty |
| repo | pair |
| issue file | workshop/issues/000256-lifecycle-transition-authority.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | b6a0766ac340596f2f5889f6a183cfcb9f5795ed..54d36c37c5357a177cd4ca65fcfcf97d833d0ea3 |
| command | sdlc milestone-close --issue 256 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-09-17T10:47:04-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

Round 3's inversion is the right call and the code is genuinely better for it: `startupResumeRefusal` decorating *any* failure while `ResumeDiagnosticCode` goes back to meaning "structured refusal" restores the reader distinction `menu_reattach.go` depends on, keeps `errors.Is`/`Unwrap` intact (asserted, not assumed), and the compile-time bindings plus the derived produced-by guard both close the class the prior rounds named. Every logic test in the touched packages passes; the only failures in `couchcore`/`couchtty`/`couchcmd` are pty-child `operation not permitted`, an environment restriction, not this diff. What blocks SHIP is one confirmed Critical: `retireDeadIncarnationBeforeStart` abandons a park transaction after probing a *different* process, on a record shape its own comment (and the plan, and the test table's exclusion note) declares unrepresentable — I built the fixture through the real store and watched a live owner's park get a permanent tombstone. That is the second finding in `irreversible-step-before-precondition`, and the rule round 2 installed ("an irreversible step never precedes a revocable check") was not violated so much as *declared inapplicable on a false reading of the validator*, which is the more durable defect. Beyond that, `BR-19`'s enumeration fix reached the atlas but not the issue `## Log` — the one home round 3's own notes said still needed it.

## 1. Strengths

- `sessionevidence_test.go:365` — `TestReAdoptionExitsAreTotalAndCoded` is a real exit table, not twelve restatements of the implementation: each cell asserts the *expected* outcome (`wantRetire := live == dead && state == IncarnationLive`), a non-empty diagnostic on every refusal, **and** park survival on refusal. That last assertion is what turns BR-12 into a permanent regression rather than a patched incident.
- `artifactcollision_zellij_test.go:414` / `:470` — the production `SessionPresence` is exercised through the stubbed-`zellij` harness, including the readable-scope→`SessionAbsent` vs unreadable-scope→`SessionUnresolved` branch that decides archive-eligibility, plus a no-`list-clients` budget assertion. ARCH-MOCK done properly: production flow and test flow share the boundary.
- `sessionevidence.go:26-31` + `sessionevidence_test.go:80` — `SessionUnresolved` as the **zero value**, pinned by a test that reads like a tautology and is not: it is what makes a gather branch that silently stops running fail closed instead of asserting absence.
- `actionableinventory.go:330-346` — the `VerifiedPark`-above-`SessionUnresolved` reorder ships with its reasoning in the code, and the reasoning is checkable: the only verdicts a session answer can produce for a verified-park record are `detached` and `parked`, and the third outcome (`binding-lost`) is documented never-retire. I confirmed the reorder is load-bearing by reverting it in a scratch copy — `TestParkedRowSurvivesAnUnresolvedSessionQuestion/session_question_failed` reddens.
- `artifactcollision.go:320-331` — the compile-time bindings carry the *why* (a type assertion fails silently and degrades indistinguishably from "could not ask"), and bind four seams on the production type plus two on the fake, rather than pinning only the seam the finding named.

## 2. Critical findings

**C1 — `resume.go:600-612`: the orphaned-park abandon probes one process and writes a permanent tombstone about another.**

`retireDeadIncarnationBeforeStart` probes `thread.Incarnations[0]`, then calls `AbandonPark(..., thread.Park.Identity)`. The comment at `:601-607` omits an identity check deliberately, on the claim that "`validateLifecycle` requires an active park's identity to match one of the record's incarnations … so a park owned by some other process is UNREPRESENTABLE in the store". `sessionevidence_test.go:372` excludes the `foreign` park row from the exit table for the same reason, and the plan repeats it at `workshop/plans/000256-lifecycle-transition-authority-plan.md:842`.

The claim is false. `cmd/internal/threadrecord/lifecycle.go:91` carries an explicit escape:

```go
replacementUnknown := matches == 0 && record.Park.Phase == "unknown" && transactionHasFailure(*record.Park, "replacement_incarnation")
if matches != 1 && !replacementUnknown { … }
```

That shape is produced by real code — `park.go:653-662` records `FailureReplacementIncarnation` when `FinalizePark` hits a revision conflict and `hasExactParkIncarnation` no longer matches — and is pinned as valid by `threadrecord/record_test.go:303`.

Confirmed by execution, not inspection. In a scratch copy of the repo I created, through the production `ThreadStore.CreateThread`, a record with one `live` incarnation `{PID 99, "replacement"}` and an active park owned by `{PID 42, "original-owner"}` (phase `unknown`, `replacement_incarnation` failure), then set `FakeProcOps` so **pid 42 is alive** and pid 99 is not:

```
representable: park owner pid=42, incarnation pid=99
retireErr=<nil> retired=true afterPark=false afterIncarnations=0 tombstones=1
CONFIRMED: the park of a LIVE owner (pid 42) was abandoned after probing only pid 99
```

`AbandonPark`'s tombstone is permanent. After it, the live owner's eventual `FinalizePark` fails with "park abandon identity does not match active transaction", so the park silently never completes while its record says nothing happened — and `#275`'s audit trail for that transaction is gone.

This is the **2nd finding in family `irreversible-step-before-precondition`.** Round 2 fixed the instance (hoisting `State` above `AbandonPark`) and mutation-proved it. Do not just add the identity check. The rule that covers both: **an irreversible step's precondition must be proved about the exact entity the step acts on, and a guard omitted as "unrepresentable" must cite the validator clause that makes it so — read including its exceptions — and be pinned by a test that tries to build the fixture through the real store rather than by prose.** The enumerable siblings are the validator's *other* exception, `lifecycle.go:97-105`'s `resumeOccupied` (a `VerifiedPark` record may carry one `creating`+`Start` or `unknown` incarnation), which is the second place an "unrepresentable" claim about incarnation/park shape can be wrong. Sweep both, restore the `foreign` row to the exit table with the fixture above, and correct `plan:842` and the test's exclusion note.

## 3. Important findings

**I1 — `BR-19` is not addressed: the enumeration reached the atlas and not the issue `## Log`.**

`atlas/couch.md:1330` now reads "One class, four sites" and lists all four — that half is correct and I checked it against `resume.go:597` and the plan's Revisions. But `workshop/issues/000256-lifecycle-transition-authority.md` still says three in all three of its homes: line 123 (`*(Done 2026-09-17; the class had three sites — see Log.)*`), line 133 (the Log heading `M1: one class, three sites`), line 141 (`**The class had three sites, and only running it found the last two.**`), with an enumeration of 1–3 that omits the site carrying the irreversible-ordering rule. BR-19's rule named four homes — code comment, plan Revisions, atlas, issue Log — and round 3 updated one of the two that were outstanding. Round 3's own review notes flagged this (`workshop/plans/000256-lifecycle-transition-authority-m1-review.md:476`: "I3's correction belongs in `atlas/couch.md` **and the issue's `## Log`**").

**I2 — `stale-wording-after-referent-change`, and one member of it is behavioural.**

This is the **4th finding in family `stale-wording-after-referent-change`.** Earlier rounds fixed instances (BR-5 atlas, BR-6 README, BR-7 four of five `ThreadBusy` sites). Do not fix these five sites; fix the rule. Measured prevalence in this window, with the behavioural one first:

1. `resume.go:31-34, 569, 624` — **behavioural.** `ResumeNotRunning`'s own declaration says "a relaunch target that is **not running at all**. It is the OPPOSITE of `ResumeLive`". The re-adoption refuses with it for the inverse condition — "recorded process **could not be proved dead**" — and again for an unrelated store error. `menu_reattach.go:239` skips only `ResumeNotDetached`/`ResumeSessionGone`, so a background reattach of a `#272`-shaped row with an unobservable launcher falls to cell 7 and renders `resume-not-running` on a row whose agent is running. Nothing pins *which* code the re-adoption emits (`TestReAdoptionExitsAreTotalAndCoded` and `TestUnobservableLauncherWithLiveSessionRefusesLegibly` assert only non-empty), and nothing asserts the row text.
2. `startup.go:137-139` — "A candidate outside both sets keeps `ProofUnresolved` and classifies `unknown`." False since Task 1: `SessionPresence` is gathered for every record *after* the `ask` gate (`actionableinventory.go:556`), so an unasked candidate with a live session now classifies `detached`. No reader acts on it today, so this is doc-only — but that comment declares itself "their one home; other comments and docs point here rather than restating it".
3. `actionableinventory.go:419-421` — restates claim (2), in violation of the one-home rule it points at.
4. `actionableinventory.go:376` — `detachedResumeProofMatches` "the warm-session contract shared by **inventory**, execution and the final recheck"; the inventory no longer calls it (`recovery_execute.go:86`, `resume.go:411`, `:541` remain).
5. `artifactcollision_fake.go:59-60` — "An address never set is absent from the answer, so it reads the zero value — unresolved", while `:95-98` returns `SessionAbsent` for unset addresses. Written correct in `a848f37c`, invalidated by `b5fce898`, never re-derived. A test author trusting the doc writes an `unknown` expectation and gets `session-gone`, which is archive-eligible.
6. `startup.go:69` — `occupiedIncarnation` "shared by archive and resume"; resume no longer reads it.

(`menu.go:1234-1239` is the 5th BR-7 site, already recorded as deferred to M2/Task 4 — not counted here.)

Three rounds of hand-enumeration have each missed sites, so the rule needs a mechanism, not more diligence: **a comment that asserts what a code path classifies must name the test that pins it, and a comment that enumerates callers must derive that list.** Both are idiomatic here — `TestEveryResumeDiagnosticCodeIsProducedBySomeSite` already derives identifiers from a declaration, and `startupAsks`' claim is exactly the kind `TestNarrowedStartupAnswersAsAFullProofWould` could assert (extend it to assert the classification of an unasked candidate). For item 1 the mechanism is smaller: pick a code whose declared meaning is true of the refusal, and assert it in the exit table.

**I3 — `plan-code-divergence`: the plan's *task bodies* still direct work the code deliberately did not do.**

This is the **3rd finding in family `plan-code-divergence`.** Round 2's Revisions stated the rule for the Core-concepts tables only ("re-derive them at each milestone close") and BR-16 verified those — they are clean, I grepped every row. The class is wider: every normative statement in the plan, task bodies included. Three live instances:

1. `plan:358-359` — the Task 2 branch table puts `SessionUnresolved` (row 6) above `VerifiedPark` (row 7). The code deliberately inverts them (`actionableinventory.go:330-346`), and that inversion is BR-2's Critical fix. The table still prescribes the bug.
2. `plan:424-426` — "**Keep** `ReasonUnrecordedChild` — #276 will produce it". The code deleted it (`threadreason.go:18-37`), for a good reason recorded in the issue Log; no plan Revision records the reversal.
3. `plan:605-612` — Task 8a's steps describe work landed in M1 (the Log calls it "pulled forward"), and Step 2's red state is `resume-creating`/`resume-live`, naming an identifier deleted in this same window.

State the widened rule in the plan and sweep the task bodies in this round, rather than letting the next milestone open against three instructions that would undo boundary fixes.

## 4. Minor findings

- **`BR-20` not addressed** — `classify_test.go` was untouched in round 3. `TestClassifyThreadAcceptsExactlyWhatTheOldProjectorAccepted` (`:268-276`) still has a name claiming exactness, a doc naming a `#248` exception that no longer exists among the cases, a body admitting three `newlyActionable` `#256` shapes, and a failure message printing `tc.wasActionableBefore` alone. Worth noting: `TestEveryResumeDiagnosticCodeIsProducedBySomeSite` — added *this round* — is a fresh instance of the same rule. Its name claims produced-by; its body counts substring mentions across non-test sources, so a code with two comment mentions and no producer passes (`ResumeLive` has exactly that shape at `resume.go:32` and `relaunch.go:98` besides its producer). The family gaining a member in the round meant to close it is the argument for fixing the rule.
- **`BR-21` not addressed** — `resume.go:608` still calls `c.Threads.AbandonPark` directly, bypassing `PairLifecycleController.Abandon`'s per-thread serialization (`park.go:435-450`). Revision CAS keeps it safe; park abandonment has two authorities, and `RecoverActiveParks` (`couchcmd/run.go:348`) runs concurrently with menu-driven resumes over the same records (ARCH-ORDER).
- **ARCH-CONSTRAINTS: the plan's two budget figures were never recorded.** `plan:294-300` requires "Budget both figures and record them in `## Log`: the startup evidence round, and one steady-state refresh, on the operator's store (7 records, 4 scopes post-cleanup)". The Log has neither. The *envelope* is enforced structurally and well (`SessionPresenceQueries() == 1`, `TestSessionPresenceCountsNoClients`, `DetachedQueries() == 0` at three sites), which is stronger than a timing number — so this is bookkeeping, not risk.
- `actionableinventory.go:472-479` — the round-3 rewrite left both versions of the same sentence in place ("since #256 that is decided by RESUME AUTHORITY rather than by the bookkeeping" and, four lines later, "Resume-shaped is now about RESUME AUTHORITY, not about the bookkeeping").
- `resume.go:570` — "inspect it or **archive the thread**" is offered when the recorded process could not be proved dead; archive refuses an occupied incarnation (`thread.go:362`), so the suggested next step fails.
- `artifactcollision.go:326-329` duplicates `park.go:812`'s `var _ PairSessionIO = ScopedThreadArtifactCollisionChecker{}` — two homes for one fact (ARCH-DRY, trivial).

## 5. Test coverage notes

- The exit table is the right shape and its exclusion note is the one thing wrong with it — see C1. Restoring the `foreign` row makes the table total over what the store can actually hold rather than over what the comment believes it can.
- The two non-atomic writes at `resume.go:608`/`:620` are documented "safe to repeat", and the refusal cells assert the park survives a refusal — but no test exercises the crash-between window (abandon landed, retire failed, retry succeeds). Cheap to add via a `ThreadStore` whose `RetireIncarnation` fails once; the claim is currently prose.
- `BR-18`'s compile-time binding is the right fix and covers the class. Residual, not raised: `contextPairSessionObserver` (`recovery_execute.go:16`) and the anonymous `PairSessionContext` interface (`couchcmd/run.go:125`) are asserted on `c.Artifacts` and unpinned. Both have an explicit non-context fallback, so a drop degrades to an uncancellable observation rather than to universal `unknown` — materially lower exposure than the seams that were pinned.
- Still no live conformance check against the real `zellij` for `SessionPresence`; `sandboxedChecker` is the standing seam for one when `#276` lands.

## 6. Architectural notes

- **ARCH-DRY — pass.** `indexSessionsByName`/`uniquelyClaimed` and `resolveScopedBindings` are extracted and used by both projectors; I diffed the predicate against the two former copies and it is equivalent (ambiguous names are filtered before the state map is read, so first-wins vs skip-on-duplicate cannot diverge).
- **ARCH-PURE — pass.** `ProjectSessionPresence`, `ClassifyThread` and `startClaimed` are pure and tested without IO; the resolver is an interface at the boundary; `gatherThreadEvidence` decides nothing.
- **ARCH-PURPOSE — pass, with the shadow-sweep clean.** The single source is "the session is the liveness authority", and all four consumers derive from it (`ClassifyThread`, `DecideResume`, the store's open-park precondition via its caller, re-adoption). The hand-maintained restatements that remain — `archivableRecord`'s park + occupancy reads, `DecideRecovery`/`ProjectRecoveryChoices`' park gate — are each named to a specific later task (M3/Task 8, M2/Task 5), and `plan:463-466` is explicit that after M1 the wedged row's archive gesture still fails. That is a declared deferral, not a deferred purpose. One wording note: the issue's Plan row claims M1 "Fixes #271 and #272 by deletion", which overstates `#271` — the row is relabelled honestly and the gesture still refuses.
- **ARCH-MOCK — pass.** Fake is stateful and correctly coupled (`SetDetachedSession` sets presence too, with the reason stated); production is exercised through the stubbed-binary seam including the archive-deciding branch. Flagged only that the fake's *doc* now lies about its default (I2/5).
- **ARCH-CONSTRAINTS — pass.** Envelope enforced structurally: one host-wide `list-sessions` per refresh, zero `list-clients`, off the keystroke path in a generation-coalesced worker. Flagged only the unrecorded figures.
- **ARCH-SECURE — pass.** The session-name index is untrusted persisted input under a shared data dir; per-scope reads fail closed and that is pinned against production. `indexSessionsByName` drops empty names and refuses self-contradicting snapshots. No credentials in scope.
- **ARCH-ORDER — flag, C1.** The re-adoption is the only component here carrying state across external events, and the exit table is a good behavioural enumeration of it. The flag is the classic one this entry exists for: an observation about pid 99 is treated as authority to act on the entity identified by pid 42. "An observation is evidence about an *exact entity* at a point in time" — the guard has the exact-identity machinery (`observeExactProcess`, `ProcessIdentity`) and applies it to the wrong entity.
- **ARCH-FUNERAL — pass.** No new durable family. `AbandonPark` gains a second caller, so `ParkHistory` gains at most one tombstone per orphaned-park re-adoption, bounded by park attempts per thread; `ParkHistory` itself is unbounded but pre-existing and owned by `#275`. `SessionObservation` dies with the refresh.

## 7. Plan revision recommendations

Four `## Revisions` entries, dated this round:

1. **C1's premise was false.** Record that `validateLifecycle` permits an active park matching **zero** incarnations via `lifecycle.go:91`'s `replacementUnknown` escape (produced by `park.go:653-662`, pinned by `threadrecord/record_test.go:303`), so `plan:842`'s "the park-identity mismatch branch was **unreachable** … validation already owns it" is wrong and the guard must be restored. State the widened rule and name the second exception (`resumeOccupied`, `lifecycle.go:97-105`) as the other place an "unrepresentable" claim can be wrong.
2. **Correct `plan:358-359`** so the branch table matches the code's deliberate `VerifiedPark`-above-`SessionUnresolved` order, citing BR-2 as why.
3. **Correct `plan:424-426`** — `ReasonUnrecordedChild` was deleted, not kept, because a placeholder would silence `TestEveryReasonIsProducedBySomeShape`; `#276` restores it with a producer.
4. **Re-scope Task 8a** — its Step 3 landed in M1, and Step 2's red state names the deleted `ResumeCreating`. Widen round 2's re-derivation rule from "the Core-concepts tables" to "every normative statement in the plan, task bodies included".

Plus, outside the plan: bring the issue `## Log` and Plan row (lines 123, 133, 141) to four sites, per I1.

```findings
dispose:
  - id: BR-17
    disposition: addressed
    note: |
      Blanket deferred coder gone from resume.go (only the pre-existing retention join at :359 remains, present at base); menu_reattach.go:244-252 reads an empty code correctly again; TestResumeCodeStillMeansAStructuredRefusal and TestEveryStartupResumeFailureIsActionable pin both halves including errors.Is through the decoration. The second instance (actionableinventory.go "must not start to") is rewritten.
  - id: BR-18
    disposition: addressed
    note: |
      artifactcollision.go:320-331 binds SessionPresenceResolver, DetachedSessionResolver, NativeBindingResolver and PairSessionIO on the production type plus two on the fake, with the silent-failure rationale in the comment. Residual not re-raised: contextPairSessionObserver (recovery_execute.go:16) and couchcmd/run.go:125's anonymous interface stay unpinned, but both have an explicit non-context fallback rather than degrading to universal unknown.
  - id: BR-19
    disposition: not-addressed
    note: |
      atlas/couch.md:1330 now says "One class, four sites" — correct. The issue Log, named in the finding as the fourth home and again in round 3's own notes (m1-review.md:476), still says three at issue lines 123, 133 and 141, with an enumeration that omits the site carrying the irreversible-ordering rule. See I1.
  - id: BR-20
    disposition: not-addressed
    note: |
      classify_test.go untouched in 54d36c37; name, doc (still citing a #248 case that no longer exists) and failure message all still disagree with a body admitting three newlyActionable #256 shapes. The guard added this round, TestEveryResumeDiagnosticCodeIsProducedBySomeSite, is a fresh instance of the same rule — it counts substring mentions, so a code with comment mentions and no producer passes.
  - id: BR-21
    disposition: not-addressed
    note: |
      resume.go:608 still calls c.Threads.AbandonPark directly, bypassing PairLifecycleController.Abandon's per-thread worker; no change in park.go or run.go. Minor, CAS-protected.
  - id: BR-22
    disposition: addressed
    note: |
      ResumeCreating deleted and TestEveryResumeDiagnosticCodeIsProducedBySomeSite derives its identifiers from the declaration, so re-adding an unproduced code reddens it. Soundness gap in the guard itself recorded under BR-20 rather than re-raised here.
findings:
  - id: new
    severity: Critical
    family: irreversible-step-before-precondition
    title: |
      The orphaned-park abandon probes one process and writes a permanent tombstone about another
    detail: |
      2nd in family — fix the RULE, not this site. resume.go:600-612 abandons thread.Park after probing thread.Incarnations[0], omitting the identity check on the claim that a park owned by another process is unrepresentable. threadrecord/lifecycle.go:91 permits matches==0 when Phase is "unknown" and the transaction carries a replacement_incarnation failure; park.go:653-662 produces exactly that, and threadrecord/record_test.go:303 pins it as valid. Confirmed by execution in a scratch copy: a record created through the production ThreadStore with a live incarnation {99,"replacement"} and a park owned by {42,"original-owner"}, with pid 42 ALIVE, retires cleanly and abandons pid 42's park — one permanent tombstone, retireErr nil. The live owner's later FinalizePark then fails with "park abandon identity does not match active transaction", so the park silently never completes and #275's audit trail for it is gone. The rule: an irreversible step's precondition must be proved about the exact entity the step acts on, and a guard omitted as "unrepresentable" must cite the validator clause that makes it so, read including its exceptions, and be pinned by a test that tries to build the fixture through the real store. Enumerable sibling: lifecycle.go:97-105's resumeOccupied escape. Also correct sessionevidence_test.go:372's exclusion note and plan:842.
  - id: new
    severity: Important
    family: stale-wording-after-referent-change
    title: |
      A changed referent left six unre-derived sites, one of which renders a false diagnostic to the operator
    detail: |
      4th in family — fix the RULE, not these sites. Behavioural member first: ResumeNotRunning is declared (resume.go:31-34) as "not running at all … the OPPOSITE of ResumeLive" and is emitted at :569 for "could not be proved dead" and at :624 for a store error; menu_reattach.go:239 skips only ResumeNotDetached/ResumeSessionGone, so a background reattach of a #272 row with an unobservable launcher renders "resume-not-running" on a row whose agent is running, and no test pins which code that exit emits. Doc-only members: startup.go:137-139 ("a candidate outside both sets keeps ProofUnresolved and classifies unknown" — presence is now gathered after the ask gate, so it classifies detached); actionableinventory.go:419-421 restating it despite startup.go:124-126 declaring itself the one home; actionableinventory.go:376 ("shared by inventory" — the inventory no longer calls it); artifactcollision_fake.go:59-60 (says unset reads unresolved, code returns SessionAbsent since b5fce898); startup.go:69 (occupiedIncarnation "shared by archive and resume" — resume no longer reads it). Three rounds of hand-enumeration have each missed sites, so the rule needs a mechanism: a comment asserting what a path classifies names the test that pins it, and a comment enumerating callers derives that list — both idiomatic here.
  - id: new
    severity: Important
    family: plan-code-divergence
    title: |
      The plan's task bodies still direct three things the code deliberately did not do
    detail: |
      3rd in family — fix the RULE, not these sites. Round 2's stated rule covered the Core-concepts tables only, and those are clean (I grepped every row). The class is every normative statement in the plan. plan:358-359 prescribes SessionUnresolved above VerifiedPark, which is the bug BR-2 fixed by inverting them. plan:424-426 says "Keep ReasonUnrecordedChild" where the code deleted it, with no Revision recording the reversal. plan:605-612's Task 8a describes work pulled forward into M1 and names the deleted ResumeCreating as its red state. Widen the re-derivation rule to task bodies and sweep them this round, so the next milestone does not open against instructions that undo boundary fixes.
  - id: new
    severity: Minor
    family: declared-measurement-not-recorded
    title: |
      The ARCH-CONSTRAINTS budget figures the plan requires were never recorded in the Log
    detail: |
      plan:294-300 requires both figures in the issue Log — the startup evidence round and one steady-state refresh, on the 7-record 4-scope store. Neither is there. The envelope itself is enforced well and structurally (SessionPresenceQueries()==1, DetachedQueries()==0 at three sites, TestSessionPresenceCountsNoClients), which is stronger evidence than a timing number, so this is bookkeeping rather than risk.
```

---

## Re-review — 2026-09-17T11:15:39-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 256 — Enforce lifecycle transition authority and outcome uncertainty |
| repo | pair |
| issue file | workshop/issues/000256-lifecycle-transition-authority.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | b6a0766ac340596f2f5889f6a183cfcb9f5795ed..4b23a713721488ef13888e2ef59bb12aaa5a90b2 |
| command | sdlc milestone-close --issue 256 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-09-17T11:15:39-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

Round 4's Critical is genuinely closed, and closed the way the rule asked for: `TestForeignOwnedParkIsRepresentableAndRefused` builds the foreign-owned park **through the production `ThreadStore`**, the store accepts it, and deleting the owner probe reddens it — I ran that mutation. The `ResumeUnknown` re-code is likewise mutation-proven, and all six doc-sweep members of BR-24 are corrected, two of them with the "name the test that pins it" mechanism the finding asked for. What blocks SHIP is that the same enumeration is still short one member: `retireDeadIncarnationBeforeStart` declines silently for any incarnation count other than one (`resume.go:555`), so a record the validator accepts — active park, **zero** incarnations, via the very `replacementUnknown` exception round 4 discovered — classifies `detached`, is auto-selected at startup, and makes `couch` refuse to start in the whole tree. I ran the same fixture at base `b6a0766` (`StartInteractive err = <nil>`, couch spawns) and at head (refusal), and confirmed the record can be neither resumed nor archived. Secondary: the disproved premise that caused round 4's Critical is still stated as current in `atlas/couch.md` and in the enclosing comment of the function that was fixed.

### 1. Strengths

- **`sessionevidence_test.go:548-613` is the right shape of proof.** The test does not assert unrepresentability — it *attempts* the fixture through `store.CreateThread` and fails with "the store REFUSED the fixture, so the unrepresentability claim would have held" if it ever becomes impossible. That inverts the failure mode that produced the Critical. Mutation-verified: removing the owner probe at `resume.go:625-629` reddens it on "abandoned a park whose owner is alive and was never probed".
- **`resume.go:625` probes the entity it acts on.** `ProcessIdentity{PID: thread.Park.Identity.PID, Identity: thread.Park.Identity.ProcessIdentity}` — and I checked the guard is sound at its edges: `validateParkIdentity` (`threadrecord/lifecycle.go:185`) requires a non-empty `ProcessIdentity` with no exception, so `observeExactProcess`'s empty-identity→`Dead` path cannot be reached here.
- **`resume_test.go:319-335` pins *which* code each exit emits**, the gap that let "not running at all" be rendered over a live conversation. Mutation-verified: reverting to `ResumeNotRunning` reddens it. `ResumeNotRunning` keeps a producer (`relaunch.go:103`), so `TestEveryResumeDiagnosticCodeIsProducedBySomeSite` stays honest.
- **`startup.go:221-256` is the correct resolution of round 2's over-reach.** Both shapes get a way forward, `ResumeDiagnosticCode` goes back to meaning "structured refusal", and `errors.Is`/`Unwrap` survive. It is why today's uncoded store error degrades legibly instead of muting — including for the Critical below.
- **The doc-sweep grew a mechanism, not more diligence.** `actionableinventory.go:446` and `artifactcollision_fake.go:59-62` now name the test that pins the claim (`TestAbsentLiveEvidenceProvesNothing`, `TestSessionPresenceAnswersThroughTheProductionChecker`), and `actionableinventory.go:424-425` points at `startup.go`'s one home instead of restating it.

### 2. Critical findings

**C1 — `resume.go:555`: the re-adoption declines an open-park record with zero incarnations, and couch then refuses to start in the whole tree.** (`classification-not-authority`, **3rd in family** — do not fix this site; fix the rule.)

Confirmed by execution against the production store, not by inspection:

```
store ACCEPTED: incarnations=0 park=true
classify with present session = "detached"/""
retireDeadIncarnationBeforeStart -> retired=false err=<nil>      # len(Incarnations) != 1
CommitStartClaim err = thread {...} has an open park transaction  (diagnostic="")
```

and end to end through `StartInteractive` on a `startupFixture`:

```
head 4b23a713: StartInteractive err = thread {...} has an open park transaction
                 couch could not resume the thread in this tree ... will not start a second one.
base b6a0766a: StartInteractive err = <nil>      # couch spawns normally
```

The shape is the *same* validator exception round 4 found — `threadrecord/lifecycle.go:91`'s `replacementUnknown` permits `matches == 0` when the phase is `unknown` and the transaction carries a `replacement_incarnation` failure — read at a different incarnation count. `hasExactParkIncarnation` (`park.go:671`) returns false for zero incarnations too, so `park.go:653-662` records that failure code without requiring a replacement to exist; `recovery_execute.go:323` and `continuation_recovery.go:310` both set `next.Incarnations = nil` with no park check, and the result validates. I did not trace that chain to a single reproducible production sequence — that is the honest limit of this finding — but the store accepts the record, which is the standard round 4 itself installed, and ARCH-SECURE treats a persisted record written by another version as untrusted input.

It is also a dead end: `ArchiveThread` refuses ("a park transaction is still open; let lifecycle recovery finish"), `CommitStartClaim` refuses forever, and `RecoverActiveParks` (`couchcmd/run.go:342-350`) runs *after* the dispatch returns, so a failed `start` exits at `run.go:338-340` before park recovery could clear it. Unresumable, unarchivable, and now blocking `couch` in that tree.

Round 2 already stated the rule — *"the enumeration is not 'four sites', it is every guard that refuses on a record's incarnation or park"* — and `resume.go:604-608` writes it down as four **sites** rather than as that predicate, which is why the predicate's other half was never checked. The rule to enforce: **the re-adoption must be total over the record shapes `validateLifecycle` accepts; every `return nil, nil` arm is a claim about what the store cannot hold and needs the same treatment `thread.Park`'s guard just got.** Concretely: hoist the park-abandon above the `len(Incarnations) != 1` gate (its own owner probe already authorizes it independently of any incarnation), and refuse-with-a-code rather than declining silently for the counts it still cannot clear. Mechanism: extend `TestReAdoptionExitsAreTotalAndCoded` (`sessionevidence_test.go:365`) with an incarnation-count dimension `{0, 1}` × the existing park/liveness/state axes, asserting that a record `ClassifyThread` calls resumable is either cleared or refused with a non-empty diagnostic *and* an in-product escape.

### 3. Important findings

**I1 — the premise round 4 disproved is still stated as current in two homes, one of them inside the function that was fixed.** (`atlas-contradicts-code`, **3rd in family** — do not fix these sites; fix the rule.)

- `atlas/couch.md:1343-1345`: *"An orphaned park is abandoned alongside the dead incarnation — **one probe answers both, since the park identity is copied from the incarnation** — and every precondition is screened before that write."* That is precisely the claim `TestForeignOwnedParkIsRepresentableAndRefused` now falsifies, and the atlas is the map AGENTS.md §8 requires be current. The atlas's "Two rules fell out of the sweep" paragraph (`:1349-1354`) also predates round 4's two new rules and does not carry them.
- `cmd/internal/couchcore/resume.go:597-600`: *"But the park identity is COPIED from the incarnation (park.go, soleParkableIncarnation), so the probe above already proved the park's own owner dead."* — contradicted ten lines later by its own correction at `:606-624`. A reader who stops at the outer comment gets the retracted reasoning.

This is one rule with BR-19, which remains open on the site-count homes: **when a boundary round changes a claim, every home of that claim is re-derived in the same round — code comment, plan Revisions, atlas, issue Log.** Three rounds of hand-sweeping have each left homes behind, so the rule needs the mechanism BR-24 introduced applied here too: the claim that justifies a guard's *shape* should cite the test that pins it (`TestForeignOwnedParkIsRepresentableAndRefused`), so the claim and its evidence move together. Measured prevalence in this window: 2 stale homes for the park-identity claim, 3 for the site count (BR-19), 6 for BR-24's referent change, 5 for BR-7's.

### 4. Minor findings

- `actionableinventory.go:567` — `presenceErr` is discarded entirely (`if presence, presenceErr := ...; presenceErr == nil`). A host-wide `zellij` failure or an unreadable scope path renders every row `checking…` with the cause recorded nowhere, while `PathError` on the same struct is carried per record. Fail-closed is right; anonymous is the thing `#181` removed. (`degradation-without-diagnostic`)
- `workshop/plans/...-plan.md:363-364` — round 4's own swap of rows 6/7 left the sentence below the table false: *"Rows 7–9 read only resume authority, which is genuinely durable"*, where row 7 is now `SessionUnresolved`. Row 8 (`binding-lost`) is reachable only *inside* the `VerifiedPark` branch in the code, above row 7, and the `VerifiedPark` + `ProofUnresolved → unknown` sub-case is absent from the table; `:355` still says "(in-memory observation)" and `:213-215` still says *"Ephemeral state stays ephemeral"*. (`plan-code-divergence`, **4th in family** — the rule has now been stated twice and hand-applied twice; the mechanism is to stop restating the branch table in the plan and point at `ClassifyThread` + `classify_test.go`'s `everyThreadShape`, which is derived and tested.)
- `resume_test.go:333` — the failure message prints `ResumeNotRunning` as the value it must *not* be, so under the mutation it reads *"reports "resume-not-running" … not "resume-not-running""*. Name the expected value (`ResumeUnknown`).
- `sessionevidence_test.go` now holds four tests that exercise `resume.go` (`TestUnknownLivenessNeverRetiresAnIncarnation`, `TestOrphanedParkDoesNotWedgeTheResumeChain`, `TestReAdoptionExitsAreTotalAndCoded`, `TestForeignOwnedParkIsRepresentableAndRefused`) while round 4 correctly put the fifth in `resume_test.go`. One file or the other.
- `artifactcollision.go:329` duplicates `park.go:812`'s `var _ PairSessionIO = ScopedThreadArtifactCollisionChecker{}` — two homes for one fact.

### 5. Test coverage notes

- **Suite state:** `couchcore`, `couchtty`, `couchcmd` fail only on `ptychild: operation not permitted` and `mkdir /tmp/pcnotify-*`. I enumerated every `--- FAIL` and ran the same set at base `b6a0766a`: **identical failure set, zero logic failures.** `threadrecord` is green.
- **Mutation checks I ran this round** (scratch copy of the pinned head, both reverted after): deleting the park-owner probe reddens `TestForeignOwnedParkIsRepresentableAndRefused`; reverting `ResumeUnknown` → `ResumeNotRunning` reddens `TestReAdoptionRefusalsClaimOnlyWhatWasProved`. Two for two on the round's behavioural claims.
- **Gap (C1):** `TestReAdoptionExitsAreTotalAndCoded` iterates `{none, matching} park × {dead, unknown, alive} × {Live, Unknown}` but holds incarnation **count** fixed at one — which is exactly the dimension the early return at `resume.go:555` keys on, and the one that is unswept. The `foreign` row is now covered separately, correctly.
- Carried from round 3, still true: the abandon→retire pair has no seam to inject a crash between the two writes, so "safe to repeat" (`resume.go:634-640`) remains reasoned rather than exercised. A `ThreadStore` whose `RetireIncarnation` fails once would pin it.
- No live conformance check against a real `zellij` for `SessionPresence`; `sandboxedChecker` is the standing seam when `#276` lands.

### 6. Architectural notes

- **ARCH-DRY — pass.** `indexSessionsByName`/`uniquelyClaimed`/`resolveScopedBindings` serve both projectors; `detachedsessions.go:62-82` reduces to the shared predicate. Only the duplicated `var _ PairSessionIO` above.
- **ARCH-PURE — pass.** `ClassifyThread`, `startClaimed`, `ProjectSessionPresence`, `indexSessionsByName`, `uniquelyClaimed` run with no IO. `retireDeadIncarnationBeforeStart` sits on `*Couch` and calls named store transitions; its tests run against the real `ThreadStore`, not a mock.
- **ARCH-PURPOSE — flag (C1).** Fifth round, same axis: the enumeration is written as four *sites* rather than as the predicate round 2 named, so the predicate's other half was never checked. The finding names one instance; the deliverable is the class.
- **ARCH-MOCK — pass.** Production `SessionPresence` is exercised through the stubbed-`zellij` harness including the archive-deciding readable/unreadable branch; the fake is stateful and couples detached ⇒ present; the compile-time bindings (`artifactcollision.go:320-331`) close the silent-assertion class. The foreign-park fixture goes through the real store rather than a double — that is the right instinct applied in the right place.
- **ARCH-CONSTRAINTS — pass.** One host-wide `list-sessions`, zero `list-clients`, pinned at four sites and against the production checker; the accounting is now in the issue `## Log` with wall-clock explicitly owned by M2. The park-owner probe adds one `Exists`/`Identity` pair per park-open record — negligible and off the keystroke path.
- **ARCH-SECURE — flag, contributing to C1.** The persisted record is input this process did not necessarily produce, and `validateLifecycle`'s exceptions define what it may hold. Round 4 fixed one reading of `replacementUnknown`; the zero-incarnation reading of the *same* exception is still un-enumerated. Otherwise pass: no credentials, `zellij list-sessions` parsed fail-closed on contested and duplicated names, `SessionUnresolved` as zero value.
- **ARCH-ORDER — pass, with BR-21 open.** The exit table is a real `(state, event) → (state, effects)` enumeration asserting outcomes rather than restating the implementation, and the identity fix is this entry's canonical case handled correctly ("an observation is evidence about an *exact entity*"). Carried: the abandon still bypasses `PairLifecycleController`'s per-thread worker (two authorities for one durable transition), and no test can inject the crash-between-writes interleaving.
- **ARCH-FUNERAL — pass.** No new durable family. `AbandonPark` gains one caller bounded at one tombstone per orphaned-park re-adoption; `ParkHistory` is unbounded but pre-existing and owned by `#275`. `SessionObservation` dies with its refresh.

### 7. Plan revision recommendations

1. **The enumeration is a predicate, not a list of four.** Record C1: `retireDeadIncarnationBeforeStart`'s `len(Incarnations) != 1` early return is an unstated claim about what the store can hold, and `validateLifecycle`'s `replacementUnknown` exception falsifies it at zero incarnations. State which layer clears an orphaned park when there is no incarnation to retire, and name the test dimension that keeps the exits total.
2. **Correct the branch table's surroundings** (`:355`, `:363-364`, row 8, the missing `VerifiedPark`+`ProofUnresolved` sub-case) — or better, delete the restatement and point at `ClassifyThread` plus `everyThreadShape`, so the plan stops being a hand-maintained second copy of the model.
3. **Bring `:213-215` into line with the adopted decision** — "Ephemeral state stays ephemeral" still directs the design the Revisions entry reversed.
4. Outside the plan: correct `atlas/couch.md:1343-1345` and `resume.go:597-600` (I1), and the issue's three "three sites" homes (lines 123, 176, 184 — BR-19, third round open).

```findings
dispose:
  - id: BR-19
    disposition: not-addressed
    note: |
      Round 4 prepended a new Log section mentioning a fourth site but left all three named homes unchanged: issue lines 123, 176 and 184 still say "three sites" with an enumeration of 1-3. The issue file now contradicts itself.
  - id: BR-20
    disposition: not-addressed
    note: |
      classify_test.go was untouched in 4b23a713 (last change b5fce898); name, doc and failure message still disagree with the body. The round's own new test resume_test.go:333 adds a fourth instance — its failure message prints ResumeNotRunning as the value it must not be.
  - id: BR-21
    disposition: not-addressed
    note: |
      resume.go:630 still calls c.Threads.AbandonPark directly; no change in park.go or couchcmd/run.go, and no comment saying why the per-thread worker is not needed here. Minor, CAS-protected.
  - id: BR-23
    disposition: addressed
    note: |
      Mutation-verified: deleting the owner probe at resume.go:625-629 reddens TestForeignOwnedParkIsRepresentableAndRefused, which builds the fixture through the production ThreadStore and the store accepts it. Checked the guard's edge: validateParkIdentity requires a non-empty ProcessIdentity with no exception, so observeExactProcess's empty-identity path is unreachable here. The residual stale premise in two other homes is raised separately.
  - id: BR-24
    disposition: addressed
    note: |
      All six re-derived. Behavioural member mutation-verified (ResumeUnknown -> ResumeNotRunning reddens resume_test.go:319). ResumeNotRunning keeps its producer at relaunch.go:103. Two sites gained the requested mechanism by naming the test that pins the claim.
  - id: BR-25
    disposition: addressed
    note: |
      plan:355-364 branch order inverted, :423-427 records the ReasonUnrecordedChild reversal with its reason, :594-598 marks Task 8a LANDED IN M1 and flags its deleted red state; the widened rule is in the round-4 Revisions entry. Siblings it did not reach are raised as a Minor rather than re-raised here.
  - id: BR-26
    disposition: addressed
    note: |
      The issue Log now carries the ARCH-CONSTRAINTS accounting — 0 client queries where the old path made up to 6, Physical on 4 records rather than 3, four sites asserting SessionPresenceQueries()==1 — and names M2's operator verification as the owner of the wall-clock figure.
findings:
  - id: new
    severity: Critical
    family: classification-not-authority
    title: |
      An open park with zero incarnations makes couch refuse to start in the whole tree
    detail: |
      3rd in family — fix the RULE, not this site. retireDeadIncarnationBeforeStart returns (nil, nil) whenever len(Incarnations) != 1 (resume.go:555), so an open-park record with ZERO incarnations is never cleared. Confirmed by execution against the production store — the same replacementUnknown exception round 4 found (threadrecord/lifecycle.go:91), read at a different incarnation count: CreateThread ACCEPTS the record, ClassifyThread returns detached, the re-adoption declines silently, and CommitStartClaim refuses with an uncoded "has an open park transaction". Through StartInteractive on a startupFixture, head refuses couch in the tree while the identical fixture at base b6a0766a returns err = nil and spawns normally. The record is also unarchivable (ArchiveThread — "a park transaction is still open"), and RecoverActiveParks runs after the dispatch returns (couchcmd/run.go:342) so a failed start exits before it. I did not trace a single reproducible production sequence that writes the shape; the store accepting it is the standard round 4 installed, and ARCH-SECURE treats a record from another version as untrusted input. The rule round 2 stated — "every guard refusing on record.Incarnations or record.Park" — was written into resume.go:604-608 as four SITES rather than as that predicate, which is why its other half went unchecked. Make the re-adoption total over the shapes validateLifecycle accepts — hoist the park abandon above the count gate, since its own owner probe authorizes it independently, and refuse with a code rather than declining silently for counts it cannot clear. Mechanism: add an incarnation-count dimension to TestReAdoptionExitsAreTotalAndCoded.
  - id: new
    severity: Important
    family: atlas-contradicts-code
    title: |
      The premise round 4 disproved is still current in the atlas and in the fixed function's own comment
    detail: |
      3rd in family — fix the RULE, not these sites. atlas/couch.md:1343-1345 still reads "one probe answers both, since the park identity is copied from the incarnation", the exact claim TestForeignOwnedParkIsRepresentableAndRefused now falsifies, and :1349-1354's "two rules fell out of the sweep" predates round 4's two new rules. resume.go:597-600 states the same retracted reasoning ten lines above its own correction at :606-624, so a reader who stops at the outer comment gets the disproved version. This is one rule with BR-19, still open on the site-count homes: when a boundary round changes a claim, every home of it is re-derived in the same round — code comment, plan Revisions, atlas, issue Log. Three rounds of hand-sweeping have each left homes behind, so apply BR-24's mechanism here: a claim that justifies a guard's shape cites the test that pins it, so the claim and its evidence move together. Measured prevalence in this window: 2 stale homes for the park-identity claim, 3 for the site count, 6 for BR-24's referent change, 5 for BR-7's.
  - id: new
    severity: Minor
    family: degradation-without-diagnostic
    title: |
      SessionPresence's error is discarded, so a host-wide failure renders every row checking… with no cause
    detail: |
      actionableinventory.go:567 does `if presence, presenceErr := presenceResolver.SessionPresence(ctx, addresses); presenceErr == nil` and drops the error entirely — no trace, no carried field. A zellij failure or an unreadable scope path turns every row in every tree into unusable/unknown with the reason recorded nowhere, while PathError on the same ThreadEvidence struct is carried per record. Fail-closed is correct and tested; anonymous is the shape #181 removed. Carry it the way PathError is carried, or surface it once in the banner.
  - id: new
    severity: Minor
    family: plan-code-divergence
    title: |
      Round 4's own table edit left the sentence below it false, and two Core-concepts statements still direct the reversed design
    detail: |
      4th in family — the rule has been stated twice and hand-applied twice, so state the mechanism instead. plan:363-364 says "Rows 7-9 read only resume authority, which is genuinely durable" while row 7 is now SessionUnresolved after round 4's swap. Row 8's binding-lost is reachable only INSIDE the VerifiedPark branch in the code, above row 7, and the VerifiedPark + ProofUnresolved -> unknown sub-case is missing from the table entirely. plan:355 still says "(in-memory observation)" and plan:213-215 still says "Ephemeral state stays ephemeral", both describing the design the Revisions entry adopted the opposite of. The mechanism: stop restating the branch table in the plan and point at ClassifyThread plus classify_test.go's everyThreadShape, which is derived and tested — a hand-maintained restatement of the model is a deferred consumer (ARCH-PURPOSE).
```
