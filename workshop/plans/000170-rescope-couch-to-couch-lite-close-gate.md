---
gate: boundary-review
issue: 170
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-02T13:01:40-07:00"
      agent: claude
      boundary: M1
      blocked: false
      protocol_error: no valid findings block
    - "n": 2
      timestamp: "2026-09-02T16:05:44-07:00"
      agent: claude
      boundary: M2
      blocked: false
      protocol_error: no valid findings block
    - "n": 3
      timestamp: "2026-09-02T17:00:01-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Critical
          title: Detached rows are auto-selected at startup without the binding proof resume requires, so `couch` exits 1 with no way through
          detail: |-
            actionableinventory.go:238 appends detached candidates before any ResolveEstablished
            gate, while the parked branch at :243 filters on it and DecideResume/ResumeContext
            require it for both. Reproduced against HEAD: a detached row with an unbound binding
            is listed, selected by SelectUniqueResumableRoot, and refused with
            resume-binding-unbound; couchcmd renders the error and returns 1, and `couch` has no
            new-thread escape hatch. Before M3 the parked-only selector skipped it and startup
            created a new thread.
          family: listed-implies-resumable
          round: 3
        - id: BR-2
          severity: Important
          title: No couchcore test pins StartInteractive's detached wiring; only the pty-gated couchcmd acceptance test covers it
          detail: |-
            Filtering rows to parked before the SelectUniqueResumableRoot call in a scratch copy
            leaves the entire couchcore suite green. The two acceptance tests hard-fail at
            pty.Open() in any environment without pty access, so the commit's mutation claim for
            the reattach test is unconfirmable there. A ~15-line twin of
            TestStartInteractiveResumesUniqueExactParkedRoot closes it and would also catch the
            Critical.
          family: seam-untested-at-runnable-level
          round: 3
        - id: BR-3
          severity: Important
          title: Plan Step 3b, the project file and the commit credit M3 with a physicalization fix that shipped at M2
          detail: |-
            actionableinventory.go's parked-AND-detached physicalization is unchanged in this
            window; git log -L attributes it to fac153c9 (#170 M2), and M3's new test passes
            against the M2 base production source. M3 delivered the proof, not the fix. Correct
            the step wording and workshop/projects/couch.md:921-926.
          family: record-claims-unverified-delivery
          round: 3
        - id: BR-4
          severity: Important
          title: M3's stated startup envelope contradicts the M2-corrected cost model and is unmeasured
          detail: |-
            plan.md:402 says "unchanged from #167 -- one local actionable snapshot ... no
            fan-out", but the snapshot spawns 2 + N zellij CLI processes (N = every pair session
            on the host, 5s timeout each) whenever a detach candidate exists, and M3 makes that
            the normal startup case. atlas/couch.md:401-406 frames the cost as refresh-worker
            only. Measure a startup with K detached threads and correct both sentences.
          family: envelope-claim-unmeasured
          round: 3
        - id: BR-5
          severity: Important
          title: README still describes the switcher's row states and unique resume as parked-only
          detail: |-
            README.md:360 "Rows expose only proven live and exact verified parked states"
            contradicts M2's detached rows, which M3's Done-when depends on; README.md:308
            "automatic unique resume reuses the parked thread instead" is the same residue one
            paragraph below the sentence M2 correctly widened to "the sole exact resumable
            thread".
          family: readme-stale-for-shipped-surface
          round: 3
        - id: BR-6
          severity: Minor
          title: The selector's rationale is restated near-verbatim in five artifacts
          detail: |-
            startup.go:11-24, startup_test.go:13-19, atlas/couch.md:439-446,
            workshop/projects/couch.md:908-919 and the archived #167 plan all carry the same
            three-paragraph argument. Correct today, five copies to keep in sync.
          family: prose-duplication
          round: 3
        - id: BR-7
          severity: Minor
          title: The selector's ambiguity fixtures pass the same ThreadAddress twice, a shape the projector cannot emit
          detail: |-
            startup_test.go:37-38 duplicates one row; ProjectActionableThreads emits one row per
            record. The realistic distinct-address case exists two rows below, so this is
            fixture realism only.
          family: fixture-realism
          round: 3
      boundary: M3
      blocked: true
    - "n": 4
      timestamp: "2026-09-02T17:22:10-07:00"
      agent: claude
      boundary: M3
      blocked: true
      protocol_error: no valid findings block
    - "n": 5
      timestamp: "2026-09-02T17:36:29-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: Gate reordered ahead of both appends; reverting it in a scratch copy reddens 3 subtests.
          round: 5
        - id: BR-2
          disposition: addressed
          note: New StartInteractive twin; the reviewer's exact parked-only mutation now reddens it.
          round: 5
        - id: BR-3
          disposition: addressed
          note: Plan Step 3b and workshop/projects/couch.md both credit M2 for the physicalization.
          round: 5
        - id: BR-4
          disposition: not-addressed
          note: Plan half corrected and measured; atlas/couch.md:399-406 still says refresh-worker-only, and "periodic" has no ticker.
          round: 5
        - id: BR-5
          disposition: addressed
          note: README.md:308 and :361 both rewritten for detached rows and resumable resume.
          round: 5
        - id: BR-6
          disposition: not-addressed
          note: All five copies of the selector rationale are unchanged in this window.
          round: 5
        - id: BR-7
          disposition: not-addressed
          note: startup_test.go:37-38 still duplicates one address for both ambiguity cases.
          round: 5
      findings:
        - id: BR-8
          severity: Important
          title: The detached row's binding proof is enforced in the IO shell, not the pure projector, and two comments claim the opposite
          detail: |-
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
          family: listed-implies-resumable
          round: 5
        - id: BR-9
          severity: Minor
          title: The two new StartInteractive tests hand-rebuild the record setup seedStartupParked already encapsulates
          detail: |-
            startup_test.go:262-277 and :290-304 each repeat ~10 lines that
            seedStartupParked (startup_test.go:182-195) covers, differing only in
            markActionableParked vs SetDetachedSession. A
            seedStartupResumable(t, env, tag, path, kind) covers all three sites (ARCH-DRY).
          family: shared-helper-not-extracted
          round: 5
        - id: BR-10
          severity: Minor
          title: The no-surviving-session negative asserts only inequality, which a zero-valued record also satisfies
          detail: |-
            startup_test.go:309 checks start.Record.Thread != stale.Address, so a zero
            ThreadAddress would pass. TestStartInteractiveCreatesNewRootWithoutExactCandidate:109
            shows the stronger form; one line asserting a real new thread closes it.
          family: assertion-admits-vacuous-pass
          round: 5
      boundary: M3
      blocked: true
    - "n": 6
      timestamp: "2026-09-02T17:52:49-07:00"
      agent: claude
      dispose:
        - id: BR-4
          disposition: addressed
          note: atlas:400-420 now names startup as the blocking caller, carries the 1.49 s measurement, and drops "periodic" (no refresh ticker exists in cmd/).
          round: 6
        - id: BR-6
          disposition: not-addressed
          note: 'Unchanged this window: startup.go:9-24, startup_test.go:12-18, atlas/couch.md, projects/couch.md all still carry the same rationale.'
          round: 6
        - id: BR-7
          disposition: not-addressed
          note: startup_test.go:36-38 still passes the same ThreadAddress twice for both ambiguity cases.
          round: 6
        - id: BR-8
          disposition: addressed
          note: Agent+NativeID added and enforced in detachedResumeProofMatches; mutation-verified in both directions (enforcement and reachability).
          round: 6
        - id: BR-9
          disposition: not-addressed
          note: startup_test.go:262-289 and :291-311 still hand-rebuild what seedStartupParked (:172-186) encapsulates.
          round: 6
        - id: BR-10
          disposition: not-addressed
          note: startup_test.go:309 still asserts only inequality; a zero StartResult would pass.
          round: 6
      findings:
        - id: BR-11
          severity: Important
          title: No durable record describes the layer the binding proof moved to; four records still describe the pre-fix shape
          detail: |-
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
          family: record-claims-unverified-delivery
          round: 6
        - id: BR-12
          severity: Important
          title: ProjectDetachedSessions emits a DetachedSessionObservation that ProjectActionableThreads now always rejects
          detail: |-
            detachedsessions.go:63 sets only Address and SessionName; actionableinventory.go:186-193
            now additionally requires Agent and NativeID. Both are exported pure functions in one
            package, so composing them directly yields zero detached rows -- silently, with no test
            covering it, and the operator-visible effect is "startup stops reattaching". Harmless
            today only because ActionableThreadInventoryContext:288-294 decorates in between and
            resume.go:222-226 reads only Address. Fix: split the type -- DetachedSessionObservation
            {Address, SessionName} for the session fact, DetachedResumeObservation adding the proof,
            assembled at the shell boundary (ARCH-SECURE: make the invalid state unrepresentable).
          family: producer-emits-value-its-consumer-rejects
          round: 6
        - id: BR-13
          severity: Important
          title: M3 produced a Critical and a three-round family and added no rule to workshop/lessons.md
          detail: |-
            lessons.md was last touched at M2 (9f7d4245); M1 has its own lessons commit (dec5928a),
            so per-milestone is this issue's own precedent and AGENTS.md section 4 asks for it. Two
            entries are owed, both two lines: widening an equivalence class widens its gates (gating
            one member of Resumable() and not the other is how BR-1 shipped); and a proof enforced
            in the IO shell is not part of the row's contract (BR-8's rule).
          family: lesson-not-recorded-for-boundary-defect
          round: 6
        - id: BR-14
          severity: Minor
          title: detachedResumeProofMatches repeats parkedResumeProofMatches' four-condition profile guard verbatim
          detail: |-
            2nd finding in this family -- do NOT patch the instance. Rule: when a second variant of
            an existing predicate is added, the invariant part is extracted into a shared helper in
            the same commit; a doc comment calling the two "twins" documents the duplication rather
            than removing it. Measured prevalence: 2 production sites (actionableinventory.go:187,
            :196) plus BR-9's 3 test sites. A resumableProfile(record) (*LaunchProfile, bool) helper
            covers both production sites and preserves the symmetry the comment defends (ARCH-DRY).
          family: shared-helper-not-extracted
          round: 6
      boundary: M3
      blocked: false
---

# Gate ledger — pair#170 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-02T13:01:40-07:00 (claude) — passed

**Protocol error:** no valid findings block — this round contributed no findings.

## Round 2 — 2026-09-02T16:05:44-07:00 (claude) — passed

**Protocol error:** no valid findings block — this round contributed no findings.

## Round 3 — 2026-09-02T17:00:01-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Critical] `listed-implies-resumable` Detached rows are auto-selected at startup without the binding proof resume requires, so `couch` exits 1 with no way through
  actionableinventory.go:238 appends detached candidates before any ResolveEstablished
  gate, while the parked branch at :243 filters on it and DecideResume/ResumeContext
  require it for both. Reproduced against HEAD: a detached row with an unbound binding
  is listed, selected by SelectUniqueResumableRoot, and refused with
  resume-binding-unbound; couchcmd renders the error and returns 1, and `couch` has no
  new-thread escape hatch. Before M3 the parked-only selector skipped it and startup
  created a new thread.
- **BR-2** [Important] `seam-untested-at-runnable-level` No couchcore test pins StartInteractive's detached wiring; only the pty-gated couchcmd acceptance test covers it
  Filtering rows to parked before the SelectUniqueResumableRoot call in a scratch copy
  leaves the entire couchcore suite green. The two acceptance tests hard-fail at
  pty.Open() in any environment without pty access, so the commit's mutation claim for
  the reattach test is unconfirmable there. A ~15-line twin of
  TestStartInteractiveResumesUniqueExactParkedRoot closes it and would also catch the
  Critical.
- **BR-3** [Important] `record-claims-unverified-delivery` Plan Step 3b, the project file and the commit credit M3 with a physicalization fix that shipped at M2
  actionableinventory.go's parked-AND-detached physicalization is unchanged in this
  window; git log -L attributes it to fac153c9 (#170 M2), and M3's new test passes
  against the M2 base production source. M3 delivered the proof, not the fix. Correct
  the step wording and workshop/projects/couch.md:921-926.
- **BR-4** [Important] `envelope-claim-unmeasured` M3's stated startup envelope contradicts the M2-corrected cost model and is unmeasured
  plan.md:402 says "unchanged from #167 -- one local actionable snapshot ... no
  fan-out", but the snapshot spawns 2 + N zellij CLI processes (N = every pair session
  on the host, 5s timeout each) whenever a detach candidate exists, and M3 makes that
  the normal startup case. atlas/couch.md:401-406 frames the cost as refresh-worker
  only. Measure a startup with K detached threads and correct both sentences.
- **BR-5** [Important] `readme-stale-for-shipped-surface` README still describes the switcher's row states and unique resume as parked-only
  README.md:360 "Rows expose only proven live and exact verified parked states"
  contradicts M2's detached rows, which M3's Done-when depends on; README.md:308
  "automatic unique resume reuses the parked thread instead" is the same residue one
  paragraph below the sentence M2 correctly widened to "the sole exact resumable
  thread".
- **BR-6** [Minor] `prose-duplication` The selector's rationale is restated near-verbatim in five artifacts
  startup.go:11-24, startup_test.go:13-19, atlas/couch.md:439-446,
  workshop/projects/couch.md:908-919 and the archived #167 plan all carry the same
  three-paragraph argument. Correct today, five copies to keep in sync.
- **BR-7** [Minor] `fixture-realism` The selector's ambiguity fixtures pass the same ThreadAddress twice, a shape the projector cannot emit
  startup_test.go:37-38 duplicates one row; ProjectActionableThreads emits one row per
  record. The realistic distinct-address case exists two rows below, so this is
  fixture realism only.

## Round 4 — 2026-09-02T17:22:10-07:00 (claude) — BLOCKED

**Protocol error:** no valid findings block — this round contributed no findings.

## Round 5 — 2026-09-02T17:36:29-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — Gate reordered ahead of both appends; reverting it in a scratch copy reddens 3 subtests.
- BR-2 — addressed — New StartInteractive twin; the reviewer's exact parked-only mutation now reddens it.
- BR-3 — addressed — Plan Step 3b and workshop/projects/couch.md both credit M2 for the physicalization.
- BR-4 — not-addressed — Plan half corrected and measured; atlas/couch.md:399-406 still says refresh-worker-only, and "periodic" has no ticker.
- BR-5 — addressed — README.md:308 and :361 both rewritten for detached rows and resumable resume.
- BR-6 — not-addressed — All five copies of the selector rationale are unchanged in this window.
- BR-7 — not-addressed — startup_test.go:37-38 still duplicates one address for both ambiguity cases.

### Raised

- **BR-8** [Important] `listed-implies-resumable` The detached row's binding proof is enforced in the IO shell, not the pure projector, and two comments claim the opposite
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
- **BR-9** [Minor] `shared-helper-not-extracted` The two new StartInteractive tests hand-rebuild the record setup seedStartupParked already encapsulates
  startup_test.go:262-277 and :290-304 each repeat ~10 lines that
  seedStartupParked (startup_test.go:182-195) covers, differing only in
  markActionableParked vs SetDetachedSession. A
  seedStartupResumable(t, env, tag, path, kind) covers all three sites (ARCH-DRY).
- **BR-10** [Minor] `assertion-admits-vacuous-pass` The no-surviving-session negative asserts only inequality, which a zero-valued record also satisfies
  startup_test.go:309 checks start.Record.Thread != stale.Address, so a zero
  ThreadAddress would pass. TestStartInteractiveCreatesNewRootWithoutExactCandidate:109
  shows the stronger form; one line asserting a real new thread closes it.

## Round 6 — 2026-09-02T17:52:49-07:00 (claude) — passed

### Disposed

- BR-4 — addressed — atlas:400-420 now names startup as the blocking caller, carries the 1.49 s measurement, and drops "periodic" (no refresh ticker exists in cmd/).
- BR-6 — not-addressed — Unchanged this window: startup.go:9-24, startup_test.go:12-18, atlas/couch.md, projects/couch.md all still carry the same rationale.
- BR-7 — not-addressed — startup_test.go:36-38 still passes the same ThreadAddress twice for both ambiguity cases.
- BR-8 — addressed — Agent+NativeID added and enforced in detachedResumeProofMatches; mutation-verified in both directions (enforcement and reachability).
- BR-9 — not-addressed — startup_test.go:262-289 and :291-311 still hand-rebuild what seedStartupParked (:172-186) encapsulates.
- BR-10 — not-addressed — startup_test.go:309 still asserts only inequality; a zero StartResult would pass.

### Raised

- **BR-11** [Important] `record-claims-unverified-delivery` No durable record describes the layer the binding proof moved to; four records still describe the pre-fix shape
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
- **BR-12** [Important] `producer-emits-value-its-consumer-rejects` ProjectDetachedSessions emits a DetachedSessionObservation that ProjectActionableThreads now always rejects
  detachedsessions.go:63 sets only Address and SessionName; actionableinventory.go:186-193
  now additionally requires Agent and NativeID. Both are exported pure functions in one
  package, so composing them directly yields zero detached rows -- silently, with no test
  covering it, and the operator-visible effect is "startup stops reattaching". Harmless
  today only because ActionableThreadInventoryContext:288-294 decorates in between and
  resume.go:222-226 reads only Address. Fix: split the type -- DetachedSessionObservation
  {Address, SessionName} for the session fact, DetachedResumeObservation adding the proof,
  assembled at the shell boundary (ARCH-SECURE: make the invalid state unrepresentable).
- **BR-13** [Important] `lesson-not-recorded-for-boundary-defect` M3 produced a Critical and a three-round family and added no rule to workshop/lessons.md
  lessons.md was last touched at M2 (9f7d4245); M1 has its own lessons commit (dec5928a),
  so per-milestone is this issue's own precedent and AGENTS.md section 4 asks for it. Two
  entries are owed, both two lines: widening an equivalence class widens its gates (gating
  one member of Resumable() and not the other is how BR-1 shipped); and a proof enforced
  in the IO shell is not part of the row's contract (BR-8's rule).
- **BR-14** [Minor] `shared-helper-not-extracted` detachedResumeProofMatches repeats parkedResumeProofMatches' four-condition profile guard verbatim
  2nd finding in this family -- do NOT patch the instance. Rule: when a second variant of
  an existing predicate is added, the invariant part is extracted into a shared helper in
  the same commit; a doc comment calling the two "twins" documents the duplication rather
  than removing it. Measured prevalence: 2 production sites (actionableinventory.go:187,
  :196) plus BR-9's 3 test sites. A resumableProfile(record) (*LaunchProfile, bool) helper
  covers both production sites and preserves the symmetry the comment defends (ARCH-DRY).

## Open findings

- **BR-6** [Minor] `prose-duplication` The selector's rationale is restated near-verbatim in five artifacts
- **BR-7** [Minor] `fixture-realism` The selector's ambiguity fixtures pass the same ThreadAddress twice, a shape the projector cannot emit
- **BR-9** [Minor] `shared-helper-not-extracted` The two new StartInteractive tests hand-rebuild the record setup seedStartupParked already encapsulates
- **BR-10** [Minor] `assertion-admits-vacuous-pass` The no-surviving-session negative asserts only inequality, which a zero-valued record also satisfies
- **BR-11** [Important] `record-claims-unverified-delivery` No durable record describes the layer the binding proof moved to; four records still describe the pre-fix shape
- **BR-12** [Important] `producer-emits-value-its-consumer-rejects` ProjectDetachedSessions emits a DetachedSessionObservation that ProjectActionableThreads now always rejects
- **BR-13** [Important] `lesson-not-recorded-for-boundary-defect` M3 produced a Critical and a three-round family and added no rule to workshop/lessons.md
- **BR-14** [Minor] `shared-helper-not-extracted` detachedResumeProofMatches repeats parkedResumeProofMatches' four-condition profile guard verbatim
