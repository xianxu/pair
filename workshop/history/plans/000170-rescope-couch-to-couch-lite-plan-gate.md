---
gate: plan-quality
issue: 170
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-02T11:59:56-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Important
          title: ctrl+backspace arrival never acknowledges the target actor's pending notification
          detail: |-
            Only finishOperation (console.go:1191-1193, guarded on Operation == "switch")
            and onHotkey's actor-landing arm (console.go:1085-1090) call
            attention.Acknowledge; Task 2 deletes the latter, and switchTo
            (console.go:398-421) does not acknowledge. Task 5's onPreviousHotkey calls
            forceSwitch directly, bypassing runMenuOperation, so landing on A via
            ctrl+backspace leaves A marked as notifying while the operator is attached
            to it. NewestActor() then names the current actor, so the next ctrl-space
            opens the switcher on it instead of the actor that paged.
          family: arrival-must-clear-attention
          round: 1
        - id: PQ-2
          severity: Important
          title: Detached branch position vs the ParkHistory tombstone scan is unspecified
          detail: |-
            resume.go:89-92 refuses on ANY tombstoned ParkHistory entry (the loop has no
            break) inside the VerifiedPark == nil branch, and AbandonPark appends
            tombstones permanently (threadstore.go:414-415). Widening that branch for
            detached records without stating that the detached proof is checked FIRST
            makes any thread with a historical abandoned park permanently
            unreattachable, failing Done-when bullet 3 for a reachable class. Task 10
            Step 1's matrix does not cross tombstoned history either.
          family: resume-gate-enumeration
          round: 1
        - id: PQ-3
          severity: Important
          title: D5 deletes Couch.Spawn, the test seam over the kept-and-rewritten start path
          detail: |-
            couch.go:148-155 is a wrapper over resolveStartResolution + spawnResolved,
            and spawnResolved survives and is rewritten by Task 13 Step 4
            (CommitStartClaim). 24 call sites in couch_test.go plus guard_live_test.go:105
            ride it; M4's "removal of the subject's tests" would drop that coverage in
            the same milestone that rewrites the path. State the migration (or keep
            Spawn as the seam). D5's file list also omits couchcmd/run_test.go, which
            uses Couch.List().
          family: deletion-drops-live-coverage
          round: 1
        - id: PQ-4
          severity: Minor
          title: Leave's verb for an IncarnationUnknown record is left as "parks or skips"
          detail: |-
            Detach requires exactly one IncarnationLive, but hasActiveIncarnation
            (park.go:147-153) admits Unknown too. Parking such a thread kills the agent,
            which is the outcome Decision 3 exists to avoid; skipping leaves it running.
            Pick one and say why.
          family: refusal-fallback-unspecified
          round: 1
        - id: PQ-5
          severity: Minor
          title: M1 documents alt+d before M2 implements it; README's leave prose has no owner
          detail: |-
            Task 5 Step 5 adds "Alt+d detach" to menuControls at M1 while Task 11 wires
            it at M2, and TestREADMEDocumentsEveryPanelControl will pin a key that does
            nothing. Separately, README:350-352 ("parks every active actor sequentially,
            and returns to the parent shell only after all parks are verified") becomes
            false at M2 and no task claims it.
          family: docs-land-at-wrong-milestone
          round: 1
        - id: PQ-6
          severity: Minor
          title: The M2 refresh measurement is prose, not a checklist step
          detail: |-
            "Measure the refresh against the committed BenchmarkMenu100 fixture before
            M2 closes" appears only in the envelope paragraph; Task 11 has no step that
            runs it, so the 2+N subprocess claim ships unverified.
          family: envelope-measurement-unchecked
          round: 1
        - id: PQ-7
          severity: Minor
          title: D1 leaves ariadne's sdlc fleet policy command with no consumer
          detail: |-
            couch is the only external consumer of ariadne/cmd/sdlc/fleet.go's policy
            arm, which carries its own helptext and e2e tests. Say whether it stays
            (operator-facing on its own merits) or gets a peer-repo follow-up; do not
            leave the cross-repo consequence unstated.
          family: peer-surface-undispositioned
          round: 1
        - id: PQ-8
          severity: Minor
          title: The DeleteStart data-loss note over-claims label and description loss
          detail: |-
            deleteThreadIf's predicate already includes threadHasMetadata
            (threadmetadata_model.go:28-30 - Name, Description, PublishedSummary), so a
            named record is refused rather than deleted. The LatestLaunchProfile loss on
            an unnamed detached thread is real and the fix is right; narrow the prose so
            the next reader does not inherit a wrong model of the guard.
          family: claim-overstates-existing-guard
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-02T12:03:01-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: switchTo gains an arrival arg and owns both per-landing rules; forceSwitch routes through it, covering onPreviousHotkey.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: Detached branch ordered before the no-break tombstone scan at resume.go:88-92, with the cross added to Task 10's matrix.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: Spawn moved to "Kept, and why" as the start-path test seam; List's run_test.go caller named in D5.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: Leave skips an IncarnationUnknown thread and reports it, with the Decision-3 rationale stated.
          round: 2
        - id: PQ-5
          disposition: addressed
          note: Alt+d help row deferred to Task 11 Step 4b, which also owns the false README:350-352 leave prose.
          round: 2
        - id: PQ-6
          disposition: addressed
          note: Now Task 11 Step 4c, with numbers recorded in the Log and a stated fallback.
          round: 2
        - id: PQ-7
          disposition: addressed
          note: Task 15 Step 0 records the disposition as a peer-repo note.
          round: 2
        - id: PQ-8
          disposition: addressed
          note: Narrowed to the unnamed-record LatestLaunchProfile exposure; threadHasMetadata credited for named records.
          round: 2
      blocked: false
content_hash: 23adab6416917d6a3ee6564d0bacca38c3aca61e3ca291f92f4a5063a9b56ed6
---

# Gate ledger — pair#170 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-02T11:59:56-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `arrival-must-clear-attention` ctrl+backspace arrival never acknowledges the target actor's pending notification
  Only finishOperation (console.go:1191-1193, guarded on Operation == "switch")
  and onHotkey's actor-landing arm (console.go:1085-1090) call
  attention.Acknowledge; Task 2 deletes the latter, and switchTo
  (console.go:398-421) does not acknowledge. Task 5's onPreviousHotkey calls
  forceSwitch directly, bypassing runMenuOperation, so landing on A via
  ctrl+backspace leaves A marked as notifying while the operator is attached
  to it. NewestActor() then names the current actor, so the next ctrl-space
  opens the switcher on it instead of the actor that paged.
- **PQ-2** [Important] `resume-gate-enumeration` Detached branch position vs the ParkHistory tombstone scan is unspecified
  resume.go:89-92 refuses on ANY tombstoned ParkHistory entry (the loop has no
  break) inside the VerifiedPark == nil branch, and AbandonPark appends
  tombstones permanently (threadstore.go:414-415). Widening that branch for
  detached records without stating that the detached proof is checked FIRST
  makes any thread with a historical abandoned park permanently
  unreattachable, failing Done-when bullet 3 for a reachable class. Task 10
  Step 1's matrix does not cross tombstoned history either.
- **PQ-3** [Important] `deletion-drops-live-coverage` D5 deletes Couch.Spawn, the test seam over the kept-and-rewritten start path
  couch.go:148-155 is a wrapper over resolveStartResolution + spawnResolved,
  and spawnResolved survives and is rewritten by Task 13 Step 4
  (CommitStartClaim). 24 call sites in couch_test.go plus guard_live_test.go:105
  ride it; M4's "removal of the subject's tests" would drop that coverage in
  the same milestone that rewrites the path. State the migration (or keep
  Spawn as the seam). D5's file list also omits couchcmd/run_test.go, which
  uses Couch.List().
- **PQ-4** [Minor] `refusal-fallback-unspecified` Leave's verb for an IncarnationUnknown record is left as "parks or skips"
  Detach requires exactly one IncarnationLive, but hasActiveIncarnation
  (park.go:147-153) admits Unknown too. Parking such a thread kills the agent,
  which is the outcome Decision 3 exists to avoid; skipping leaves it running.
  Pick one and say why.
- **PQ-5** [Minor] `docs-land-at-wrong-milestone` M1 documents alt+d before M2 implements it; README's leave prose has no owner
  Task 5 Step 5 adds "Alt+d detach" to menuControls at M1 while Task 11 wires
  it at M2, and TestREADMEDocumentsEveryPanelControl will pin a key that does
  nothing. Separately, README:350-352 ("parks every active actor sequentially,
  and returns to the parent shell only after all parks are verified") becomes
  false at M2 and no task claims it.
- **PQ-6** [Minor] `envelope-measurement-unchecked` The M2 refresh measurement is prose, not a checklist step
  "Measure the refresh against the committed BenchmarkMenu100 fixture before
  M2 closes" appears only in the envelope paragraph; Task 11 has no step that
  runs it, so the 2+N subprocess claim ships unverified.
- **PQ-7** [Minor] `peer-surface-undispositioned` D1 leaves ariadne's sdlc fleet policy command with no consumer
  couch is the only external consumer of ariadne/cmd/sdlc/fleet.go's policy
  arm, which carries its own helptext and e2e tests. Say whether it stays
  (operator-facing on its own merits) or gets a peer-repo follow-up; do not
  leave the cross-repo consequence unstated.
- **PQ-8** [Minor] `claim-overstates-existing-guard` The DeleteStart data-loss note over-claims label and description loss
  deleteThreadIf's predicate already includes threadHasMetadata
  (threadmetadata_model.go:28-30 - Name, Description, PublishedSummary), so a
  named record is refused rather than deleted. The LatestLaunchProfile loss on
  an unnamed detached thread is real and the fix is right; narrow the prose so
  the next reader does not inherit a wrong model of the guard.

## Round 2 — 2026-09-02T12:03:01-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — switchTo gains an arrival arg and owns both per-landing rules; forceSwitch routes through it, covering onPreviousHotkey.
- PQ-2 — addressed — Detached branch ordered before the no-break tombstone scan at resume.go:88-92, with the cross added to Task 10's matrix.
- PQ-3 — addressed — Spawn moved to "Kept, and why" as the start-path test seam; List's run_test.go caller named in D5.
- PQ-4 — addressed — Leave skips an IncarnationUnknown thread and reports it, with the Decision-3 rationale stated.
- PQ-5 — addressed — Alt+d help row deferred to Task 11 Step 4b, which also owns the false README:350-352 leave prose.
- PQ-6 — addressed — Now Task 11 Step 4c, with numbers recorded in the Log and a stated fallback.
- PQ-7 — addressed — Task 15 Step 0 records the disposition as a peer-repo note.
- PQ-8 — addressed — Narrowed to the unnamed-record LatestLaunchProfile exposure; threadHasMetadata credited for named records.

## Open findings

(none — every finding has been disposed)
