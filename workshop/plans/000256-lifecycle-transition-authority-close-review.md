# Boundary Review — pair#256 (whole-issue close)

| field | value |
|-------|-------|
| issue | 256 — Enforce lifecycle transition authority and outcome uncertainty |
| repo | pair |
| issue file | workshop/issues/000256-lifecycle-transition-authority.md |
| boundary | whole-issue close |
| milestone | — |
| window | b6a0766ac340596f2f5889f6a183cfcb9f5795ed..3b4c887c050248221a7db9d78f0f20f50545f9e4 |
| command | sdlc close --issue 256 |
| reviewer | claude |
| timestamp | 2026-09-17T18:13:27-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The implementation is in good shape: the classifier now reads the world instead of couch's bookkeeping, the arbitrary-mutation door is closed and guarded by an AST-derived receiver rule, and the guards that used to re-derive the classification consume it. Most striking is that this round's fixes are genuinely *rules* rather than patches — `lifecycleTransitionsReachingTheDoor` computes its own domain by AST fixed point, `TestReAdoptionExitsAreTotalAndCoded` takes its dimensions from `CreateThread`'s refusal instead of the author's imagination, and `TestFakesWrapEverySentinelTheirProductionSeamWraps` derives the fake/production pairing from a filesystem convention. Eight of the twelve open findings are genuinely disposed; the package tests pass (the only failures in `couchcore`/`couchtty`/`couchcmd` are `ptychild: operation not permitted`, the environment's pty restriction, not logic). What blocks a clean SHIP is narrow and entirely documentary: four of the six sites BR-38 named still state the retired occupancy/receipt rules at HEAD even though the commit message says "All rewritten", and the plan's Core-concepts prose still hand-maintains a claim about `ArchivableState`'s consumers that both the code and the plan's own delivered-table contradict.

### 1. Strengths

- `cmd/internal/couchcore/transition_refusal_test.go:298` — `lifecycleTransitionsReachingTheDoor` derives the transition domain as a call-graph fixed point over `*ThreadStore` methods reaching `updateExistingThread`, with a `< 15` floor that fails when the store's shape changes. A new transition cannot ship without either a refusal case or a written CAS-only exemption. This is the strongest thing in the diff.
- `cmd/internal/couchcore/sessionevidence_test.go:369` — the totality table BR-31 flagged now takes its dimensions from the validator's oracle (`t.Skipf` on `CreateThread` refusal) across park×count×liveness×claim-owner, and asserts three things per cell: a coded refusal, an unchanged revision, and that a refusal never left the park abandoned. Exactly the correction asked for.
- `cmd/internal/couchcore/lifecycledebris.go:53` — every precondition screened before the first write, each write authorized by a probe of *the entity it acts on* (park owner, claim owner, forked process are three different processes). The ordering rationale is written where it is load-bearing, not in a revision entry.
- `cmd/internal/couchcore/mutation_door_test.go:34` — the door guard is receiver-scoped rather than file- or package-scoped, with the reason stated (all three leaking sites were in `couchcore` itself).
- `cmd/internal/couchcmd/livefixture_test.go:20` — acceptance fixtures now drive the real claim → helper-recorded → registered sequence instead of assigning `Incarnations`, so fixture and production agree by construction.

### 2. Critical findings

None.

### 3. Important findings

**I. Four of BR-38's named sites still state the retired rules** (`atlas/couch.md:32`, `:603`, `:887`, `:1733`)

- `atlas/couch.md:887` — "The occupied-incarnation refusal is unchanged, because detach retires the incarnation…". `DecideResume` (`resume.go:113-142`) states the opposite in its own comment: "the park transaction and the incarnation are NOT read here."
- `atlas/couch.md:1733-1734` — the terminology entry still defines a parked thread as one with "an exact verified resume handle and no occupied incarnation". Both halves were retired (M2: the ledger is authority; M1: the occupancy refusal is gone).
- `atlas/couch.md:32-33` — the `live` half of the projection paragraph, which round 11 edited around: "emits only `live` when one durable live PID/start identity exactly matches one observed TTY owner". `ClassifyThread` returns `live` on `len(evidence.Live) != 0`, a union including console pty children — and `TestSwitchAgentRefusesAThreadCouchHostsWithNoIncarnation` (`parkedproducers_test.go:271`) builds exactly the row that has no incarnation at all.
- `atlas/couch.md:603` — "legacy-unverified records", whose referent `ResumeLegacyUnverified` was deleted in this window (`atlas/couch.md:1488` narrates that deletion six hundred lines later).

The mechanism BR-38 asked for *did* land (`retired_referents_test.go`, a checked-in key list), and it passes — which is the finding. Two of these sites contain "occupied", the exact vocabulary the test's own comment says the key-deriving sweep used, so the sweep did not cover the atlas the way the comment claims. The key set is the weak end; the rule BR-38 stated (prose points at `ClassifyThread` / `everyThreadShape` / the named guard rather than paraphrasing it) was applied to the plan's branch table and to nothing else.

**II. The plan's Core-concepts prose still hand-maintains claims the code contradicts** (`workshop/plans/000256-lifecycle-transition-authority-plan.md:243`)

This is the 6th finding in family `plan-code-divergence`. `TestIssue256PlanTablesMatchTheTree` machine-checks the table **rows**; the divergence has simply moved into the **bullets** under them, which nothing checks. Measured, seven instances in that one section:

- `:243-245` — "**ArchivableState** … consumed by both the menu and the store". The only production consumer is `detach.go:271` (`Couch.ArchiveThread`). The store does not call it; the menu deliberately does not (`menu.go:1267-1275` states the rule a second time on purpose, and the plan's own M3-delivered row says "Only the GUARD consumes").
- `:411` "Rows 7–9…" and `:286-287` "`Live` is classification row 4" — both index a branch table the plan deleted at `:395-402`.
- Stale coordinates: `:231` and `:286` cite `actionableinventory.go:413-437` / `:436` for `ObserveRecordedProcesses` (actually `:883`); `:282` cites `:582` for the `Unknown` collapse (actually `:715`); `:224` cites `artifactcollision.go:280-363` for `DetachedSessions` (actually `:402`).

The rule, at the level that covers all seven: a prose bullet that names consumers, branch positions or file:line coordinates is a hand-maintained restatement of the model — the same deferred consumer (ARCH-PURPOSE) the plan already retired the branch table for. Either it cites the test that enumerates the fact (the `ThreadParked` bullet at `:257-262` does this correctly, naming `TestEveryParkedProducerIsAcceptedByResumeSwitchAndArchive`) or it is deleted.

### 4. Minor findings

- `classify_test.go:374-383` — name, doc ("except #248…") and failure message (prints `wasActionableBefore` alone) still describe a narrower assertion than the body, which now ORs six `newlyActionable` shapes.
- `lifecycledebris.go:139` — `AbandonPark` called directly; `PairLifecycleController.Abandon` (`park.go:448`) routes the same store call through the per-thread worker. Two authorities for park abandonment, now reached from three callers (resume, switch-agent, archive).
- `actionableinventory.go:745` — `presenceErr` discarded; a host-wide failure renders every row `unknown` with the cause recorded nowhere, while `PathError` on the same struct is carried per record.
- `resume.go:155` / `threadstore.go:658` — ARCH-FUNERAL: archive now abandons orphaned parks as a matter of course, so `ParkHistory` gains a permanent tombstoned transaction per abandon. No cap, no sweep; `validateLifecycle` (`threadrecord/lifecycle.go:57`) walks it on every write and `DecideResume` scans it on every cold refusal. Per-thread and operator-paced, so not a cliff — and `#275` removes the durable transaction entirely, which is where the bound belongs.

### 5. Test coverage notes

Coverage is the strongest part of this issue. `TestEveryEvidenceFieldIsExercisedByTheCorpus` closes the producer enumeration from the evidence side by reflection; `TestAbsentSourceRecoveryRefusesAnOpenStartClaimBeforeTheStoreDoes` discriminates the *caller's* gate from the store's, which is the assertion that would otherwise silently pass on the wrong refusal. What remains uncovered, and is honestly declared in `## Log`: the cold-side ledger read's growth with store size. `TestWarmRowsAskNoLedgerQuestion` (`classify_test.go:888`) bounds the warm side with `resolveCalls != 0`; the cold side has no equivalent counted invariant, so the declared envelope rests on a prose note rather than an assertion the way `SessionPresenceQueries()==1` does.

### 6. Architectural notes for upcoming work

- ARCH-DRY: pass. `indexSessionsByName`/`uniquelyClaimed` consolidate the fail-closed rule the two session projectors duplicated. The one deliberate duplication (menu offer vs `ArchivableState`) is defended and compared by `TestActionOfferedImpliesPermitted` — a written offer plus a written permission plus an agreement test is the right shape, not a DRY violation.
- ARCH-PURE: pass. `ClassifyThread`, `ArchivableState`, `ResumableState`, `SwitchableState` and `ProjectSessionPresence` are pure and tested without IO; `observeExactProcessOrUnknown` was correctly lowered to take the `ProcOps` seam rather than the `Couch`.
- ARCH-MOCK: pass, notably. The fake/production sentinel conformance guard is derived rather than listed, and `artifactcollision_zellij_test.go` runs the presence seam against real zellij with a call-shape assertion.
- ARCH-SECURE: pass. `clearLifecycleDebris` explicitly treats what `validateLifecycle` accepts as the domain ("a record written by another version is untrusted input"), which is the right provenance reading. No credential surface in this diff.
- ARCH-ORDER: flagged, already recorded. `ThreadEvidence` is now seven independent fields whose legal combinations exist only in `ClassifyThread`'s branch order; the corpus pins the one ordering-sensitive combination (live + unproven) but not the constellation. The plan defers this to `#275` — that is the right place, and `#275` should carry the tagged-enum collapse, not just the park-transaction removal.
- ARCH-CONSTRAINTS: flagged (BR-41, re-raised). ARCH-FUNERAL: flagged (`ParkHistory`, above). ARCH-PURPOSE: flagged (finding II).

### 7. Plan revision recommendations

- A `## Revisions` entry for the close round recording that the `ArchivableState` bullet at `:243-245` was wrong in both directions, and that the Core-concepts **prose** is now held to the same rule as its tables: consumers cite the enumerating test or are deleted. Repoint or delete `:411`, `:286-287`, `:224`, `:231`, `:282`.
- An entry recording that `TestIssue256PlanTablesMatchTheTree`'s tree→rows direction is scoped to `issue256OwnedFiles` ∩ `issue256ConceptSymbols` — so a new concept in a *modified* file (the `PreparedAgentSwitch.state` case that motivated it) is still outside the check, deliberately, with that reason written down.

```findings
dispose:
  - id: BR-19
    disposition: addressed
    note: |
      atlas/couch.md:1342 now heads "One class, four sites"; the issue Log at :477 says four.
  - id: BR-20
    disposition: not-addressed
    note: |
      classify_test.go:374-383 unchanged: doc still says "except #248", six shapes now set newlyActionable, failure message still prints wasActionableBefore alone.
  - id: BR-21
    disposition: not-addressed
    note: |
      lifecycledebris.go:139 still calls Threads.AbandonPark directly, now from three callers (resume.go:517, switchagent.go:322, detach.go:304) rather than one.
  - id: BR-29
    disposition: not-addressed
    note: |
      actionableinventory.go:745 still drops presenceErr with no trace and no carried field.
  - id: BR-30
    disposition: addressed
    note: |
      The branch table is gone (plan:395-402 now points at ClassifyThread + everyThreadShape); the orphaned clause "Rows 7-9 read only resume authority" at plan:411 survives and is folded into the new plan-code-divergence finding.
  - id: BR-31
    disposition: addressed
    note: |
      sessionevidence_test.go:369-478 widens to park{none,matching,foreign} x count{0,1,2} x liveness{dead,unknown,alive} x 6 shapes with CreateThread's refusal as the skip oracle; verified passing.
  - id: BR-32
    disposition: addressed
    note: |
      SessionObservation carries no Name at HEAD (sessionevidence.go:45-54), deleted with the delete-or-justify reason written in place.
  - id: BR-34
    disposition: not-addressed
    note: |
      Three of four sites fixed (ops.go, atlas:498, atlas:33-38 parked half); its fourth named site, the "occupied-incarnation refusal is unchanged" passage, survives at atlas/couch.md:887 and is carried in BR-38's list.
  - id: BR-38
    disposition: not-addressed
    note: |
      The checked-in retired-referent list landed and passes, but four named sites survive at HEAD: atlas/couch.md:32-33 (the live half), :603, :887, :1733-1734.
  - id: BR-39
    disposition: addressed
    note: |
      warm_failure_test.go:168-177 now asserts `else if row.State != ThreadParked` with the premise-determined verdict; the pre-M2 disjunction arm is gone.
  - id: BR-40
    disposition: addressed
    note: |
      plan_contract_256_test.go:190-218 adds the tree->rows direction; its narrowing to issue256OwnedFiles is written down with the reason, so PreparedAgentSwitch.state remains unlisted by design rather than by oversight.
  - id: BR-41
    disposition: not-addressed
    note: |
      Round 11's I4 bounded archive's cost; the cold-side refresh ledger read still has no counted invariant, and the issue Log still declares it an open gap in prose.
findings:
  - id: new
    severity: Important
    family: plan-code-divergence
    title: |
      The plan's Core-concepts PROSE still restates the model, and one bullet names two consumers ArchivableState does not have
    detail: |
      This is the 6th finding in family plan-code-divergence, so the rule is the
      deliverable, not the site. TestIssue256PlanTablesMatchTheTree machine-checks the
      table ROWS; the divergence moved into the bullets beneath them, which nothing
      checks. Measured, seven instances in one section. plan:243-245 says ArchivableState
      is "consumed by both the menu and the store" -- the only production consumer is
      detach.go:271, the store never calls it, and menu.go:1267-1275 states the rule a
      second time ON PURPOSE, which the plan's own M3-delivered row records as "Only the
      GUARD consumes". plan:411 ("Rows 7-9") and plan:286-287 ("Live is classification row
      4") index a branch table the plan deleted at :395-402. Four coordinates are stale:
      :231 and :286 cite actionableinventory.go:413-437 and :436 for ObserveRecordedProcesses
      (actually :883), :282 cites :582 for the Unknown collapse (actually :715), :224 cites
      artifactcollision.go:280-363 for DetachedSessions (actually :402). The rule: a prose
      bullet naming consumers, branch positions or file:line coordinates is a
      hand-maintained restatement of the model -- a deferred consumer (ARCH-PURPOSE) -- so
      it either cites the test that enumerates the fact, the way the ThreadParked bullet at
      :257-262 correctly cites TestEveryParkedProducerIsAcceptedByResumeSwitchAndArchive,
      or it is deleted.
  - id: new
    severity: Minor
    family: unbounded-append-without-removal
    title: |
      Archive now abandons orphaned parks as a matter of course, so ParkHistory gains a permanent tombstone per archive with no removal path
    detail: |
      ARCH-FUNERAL. clearLifecycleDebris (lifecycledebris.go:138-144) is reached from
      resume, switch-agent and archive, and each orphaned park it clears appends a
      tombstoned ParkTransaction at threadstore.go:658. Nothing caps or sweeps
      ParkHistory; validateLifecycle walks every entry on every write
      (threadrecord/lifecycle.go:57) and DecideResume scans it on every cold refusal
      (resume.go:155). Growth is per-thread and operator-paced, so this is a note rather
      than a cliff -- and resume.go's own comment says #275 dissolves the durable park
      transaction entirely, which is where the bound belongs. Recorded so the removal
      path is named rather than inherited.
```

---

## Re-review — 2026-09-17T18:31:41-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 256 — Enforce lifecycle transition authority and outcome uncertainty |
| repo | pair |
| issue file | workshop/issues/000256-lifecycle-transition-authority.md |
| boundary | whole-issue close |
| milestone | — |
| window | b6a0766ac340596f2f5889f6a183cfcb9f5795ed..7447d100c98222aec139b5c5f1790f1c3edaca9b |
| command | sdlc close --issue 256 |
| reviewer | claude |
| timestamp | 2026-09-17T18:31:41-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Round 12 answered all three blocking findings and I can confirm each at HEAD: the four atlas paragraphs BR-34/BR-38 named now state the post-M1/M2 rules (`atlas/couch.md:32-40`, `:605-607`, `:888-894`, `:1739-1743`), five more retired phrases are checked-in data in `issue256RetiredClaims`, and the plan's Core-concepts prose is now machine-checked for coordinates, branch indices and rotted test citations. I ran the new checks (`TestRetiredClaimsAreNotRestatedAsCurrent`, `TestIssue256PlanTablesMatchTheTree`, `TestIssue256CoreConceptsProseCitesTestsNotCoordinates`, `TestColdLedgerReadsAreOnePerColdCandidate`, `TestEveryParkedProducerIsAcceptedByResumeSwitchAndArchive`) — all pass. The remaining `couchcore`/`couchtty`/`couchcmd` failures in this environment are all `ptychild: … operation not permitted`, the pty-spawn restriction, not logic. No Critical and no Important finding survives; what is left is three prior Minors the round deliberately recorded rather than fixed, plus two new Minors — one of them a sentence in the Core-concepts section that the very commit adding the cold-side bound left saying the cold side is unbounded. Nothing here blocks the boundary.

### 1. Strengths

- `cmd/internal/couchcore/retired_referents_test.go:23` — the retired claims are **data**, with the derivation of each key stated (a paragraph-window vocabulary sweep at the retiring commit, not memory), an explicit `narratesRetirement` exemption so history may quote what it retired, a stated scope limit ("recurrence, not novel staleness"), and a `< 50` floor that fails if the walk loses its scope. This is the right shape for a family that failed seven times as prose discipline.
- `cmd/internal/couchcore/plan_contract_256_test.go:247` — the prose check is the correct generalisation of the table check: it bans the two things that rot (coordinates, branch positions) and makes the sanctioned alternative — citing the enumerating test — self-verifying by requiring the cited `Test…` to exist in the couch packages. It found 12 where the reviewer listed 7, and it retires with the plan (`t.Skipf` when the file is archived).
- `cmd/internal/couchcore/classify_test.go:451` — `TestColdLedgerReadsAreOnePerColdCandidate` derives the cold-candidate count from the fixture's own records and fails if the fixture stops separating the two populations (`cold == 0 || cold == len(addresses)`), so it cannot go vacuous the way a hand-written expectation would. It is the honest bound: linear, one read per cold candidate, rather than a constant nobody could hold.
- `atlas/couch.md:891-894` and `:1739-1743` — the two hardest passages are now correct *and* point at the enumeration (`everyThreadShape`) instead of re-deriving it; the "two of the four parked shapes carry an incarnation" sentence explains why the rule is "no LIVE evidence" rather than "no incarnation", which is the distinction three earlier rounds kept losing.
- `cmd/internal/couchcore/classify_test.go:390-394` — the failure message now prints both flags and *why* one alone misread, which is the failure-message discipline BR-14 asked for.

### 2. Critical findings

None.

### 3. Important findings

None. BR-34, BR-38 and BR-42 are disposed `addressed` below, each verified against the pinned before/after diff and the concrete referent (`ArchivableState`'s only production consumer is still `detach.go:271`; `DecideResume` still does not read the incarnation; `ThreadParked`'s doc, `everyThreadShape` and `atlas/couch.md` now all say four producers).

### 4. Minor findings

- `workshop/plans/…-plan.md:323` — Core concepts still reads "the cold side is not (BR-41, recorded)" while `:1087` of the same commit records the bound and the test exists. 7th in `plan-code-divergence`; raised at rule level below, not as a sentence to patch.
- `cmd/internal/couchcore/retired_referents_test.go:23` — the retired-claims map names no end (ARCH-FUNERAL), unlike its sibling check which skips when the plan is archived.
- `classify_test.go:384` — the test's *name* still claims "ExactlyWhatTheOldProjectorAccepted" while its own new doc says "PLUS the shapes a later issue admitted on purpose" (BR-20, disposed `not-addressed`).
- `atlas/couch.md:606` — "Ambiguous and legacy-unverified records now appear in both views" survives from the #181 narration; the phrase is not false (those records still appear and are named), but its diagnostic namesake `ResumeLegacyUnverified` was deleted in this window. Noted, not raised: rewriting it adds a new unchecked paraphrase, which is the cost the family keeps paying.

### 5. Test coverage notes

- The two behaviour-adjacent disposals this round have evidence: BR-41's bound is a real counting assertion against `FakeThreadArtifactCollisionChecker.BindingResolutions()` that would fail if the read became per-record, and it passes. BR-20's is doc/message only, which is the right classification.
- BR-21, BR-29 and BR-43 are unchanged in code and correctly declared as such in `## Log` — no test claims otherwise, which is what keeps the record honest.
- Environment: every failing test in the three touched packages fails on `operation not permitted` from `ptychild`; the close's own `make -k test` (210 packages, exit 0) was recorded at `3b4c887c`, and `7447d100` adds only tests and prose, all of which I ran green.

### 6. Architectural notes

- **ARCH-DRY — pass.** `indexSessionsByName`/`uniquelyClaimed` are now the one attributability rule for both session projectors, and `resolveScopedBindings` is the one index read for `SessionPresence` and `DetachedSessions`.
- **ARCH-PURE — pass.** `ClassifyThread`, `SwitchableState`/`ArchivableState`/`ResumableState`, `ProjectSessionPresence` are pure and tested without IO; the evidence gather is the thin shell.
- **ARCH-PURPOSE — pass on the shadow-sweep.** Consumers of the classification all derive: `PrepareAgentSwitch`, `Couch.ArchiveThread`, `SelectResumableRoot`, `couch --list/--show` (`ThreadSummary.State` is the same classifier). The menu's second statement of the offer is deliberate and compared by `TestActionOfferedImpliesPermitted`.
- **ARCH-MOCK — pass.** Compile-time interface bindings for the production checker and the fake (`artifactcollision.go:305-315`), plus `TestFakesWrapEverySentinelTheirProductionSeamWraps` deriving the seam/fake pairing from the filename convention.
- **ARCH-CONSTRAINTS — pass with a residue.** Warm and cold ledger reads are both bounded now; `observeRecovery`'s widened probe inside `reconcileRecoveryHelper`'s 8-attempt loop is still only measured, not asserted — worth one counted invariant when #275 touches that path.
- **ARCH-SECURE — pass.** `clearLifecycleDebris` treats the validator's accepted domain as the input space (records written by other versions), and each write is authorised by a probe of the entity it acts on.
- **ARCH-ORDER — flag, carried.** `clearLifecycleDebris` screens every precondition before the first write and the CAS makes a loser a coded refusal, but park abandonment now has two authorities (BR-21, still open and recorded). `PreparedAgentSwitch.state` is an observation carried across the prepare→commit boundary; every downstream write is revision-CAS'd, so a stale admission fails closed rather than acting.
- **ARCH-FUNERAL — flag (Minor, below).** BR-43's ParkHistory growth now has a named owner (#275 removes `ParkTransaction`/`ParkHistory` outright). The new guard data introduced this round does not name its end.

### 7. Plan revision recommendations

- A `## Revisions` entry recording that the Core-concepts cost bullet was corrected: the cold side **is** bounded, by `TestColdLedgerReadsAreOnePerColdCandidate`, and that the new prose check cannot see a stale *negative* coverage claim — so the section's rule is extended to "a bullet asserting something is not covered names the test that would cover it, or goes".

```findings
dispose:
  - id: BR-34
    disposition: addressed
    note: |
      All four named sites verified rewritten at HEAD (ops.go:381 resume summary, atlas:32-40, atlas:498, atlas:888-894) and the prescribed grep is now a checked-in list plus TestRetiredClaimsAreNotRestatedAsCurrent, which passes.
  - id: BR-38
    disposition: addressed
    note: |
      ThreadParked's declaration says FOUR producers (actionableinventory.go:26-32), resume.go:250 and relaunch.go:104-107 no longer state the retired occupancy rationale, atlas:1739 cites everyThreadShape; the "legacy-unverified records" sentence at atlas:606 survives but is not a false claim, only an obsolete label.
  - id: BR-42
    disposition: addressed
    note: |
      Verified at HEAD: the ArchivableState bullet now says "Only the GUARD consumes it" (true — detach.go:271 is the sole production caller), the branch-position and coordinate citations are gone, and TestIssue256CoreConceptsProseCitesTestsNotCoordinates enforces all three rules and passes.
  - id: BR-41
    disposition: addressed
    note: |
      TestColdLedgerReadsAreOnePerColdCandidate counts real ResolveEstablished calls against a fixture-derived candidate count and would fail on a second read or a read for a non-candidate; one plan sentence still says otherwise, raised separately.
  - id: BR-43
    disposition: addressed
    note: |
      The finding asked for the removal path to be named rather than inherited; it is named in the issue Log and owned by pair#275, whose Spec removes ParkTransaction and ParkHistory outright.
  - id: BR-20
    disposition: not-addressed
    note: |
      Doc and failure message were corrected; the test's NAME still claims "ExactlyWhatTheOldProjectorAccepted" and is now contradicted by its own doc — the rule asked for all three in the same edit.
  - id: BR-21
    disposition: not-addressed
    note: |
      lifecycledebris.go:139 still calls Threads.AbandonPark directly, now from three callers; recorded in the issue Log as known, unchanged in code.
  - id: BR-29
    disposition: not-addressed
    note: |
      actionableinventory.go:735 still drops presenceErr entirely; recorded in the issue Log as known, unchanged in code.
findings:
  - id: new
    severity: Minor
    family: plan-code-divergence
    title: |
      The plan's Core-concepts prose still says the cold-side ledger read is unbounded, in the commit that bounded it
    detail: |
      This is the 7th finding in family plan-code-divergence. Do NOT fix the sentence
      alone. Measured: workshop/plans/000256-lifecycle-transition-authority-plan.md:323
      reads "the warm side ... is bounded by TestWarmRowsAskNoLedgerQuestion; the cold
      side is not (BR-41, recorded)", while :1087 of the same commit records
      TestColdLedgerReadsAreOnePerColdCandidate, which exists and passes. The new
      guard, TestIssue256CoreConceptsProseCitesTestsNotCoordinates, cannot catch this
      by construction: it checks that a CITED test exists, and a claim that something
      is NOT tested cites nothing. The rule at the level that covers the class: a
      Core-concepts bullet asserting the ABSENCE of coverage is a hand-maintained
      restatement with a shorter half-life than one asserting its presence, so it
      either names the test that would cover it or is deleted — and the section check
      should refuse negative-coverage phrasing ("nothing bounds", "is not bounded",
      "recorded as a known gap") the same way it refuses coordinates. The two
      historical homes (the issue Log at :364 and the plan's M2 review entry at :1295)
      are narration of what was true then and should stay.
  - id: new
    severity: Minor
    family: unbounded-append-without-removal
    title: |
      The retired-claims guard data grows per issue and names no end, unlike its sibling check
    detail: |
      This is the 2nd finding in family unbounded-append-without-removal, so the rule is
      the deliverable. ARCH-FUNERAL: this window created two checked-in guard datasets.
      TestIssue256PlanTablesMatchTheTree names its end — t.Skipf when the plan is
      archived, "this check retires with it". issue256RetiredClaims
      (retired_referents_test.go:23) does not: 25 phrase keys today, walked against
      every .go and .md under cmd/ and atlas/ on every run, with no statement of who
      removes an entry or when. It is not a cliff — the cost is trivial and the claims
      are retired "forever" in the ordinary case — but pair#275 already plans to delete
      ParkTransaction and ParkHistory, at which point keys like "no occupied
      incarnation" and "exact verified resume handle" describe concepts the tree no
      longer has, and a later author writing legitimate prose meets a refusal whose
      rationale has expired. The rule: a per-issue guard dataset states the same thing
      its sibling does — what retires it (the issue's archival, the field's deletion),
      in one line next to the data.
```
