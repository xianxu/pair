# Boundary Review — pair#523 (milestone M1)

| field | value |
|-------|-------|
| issue | 523 — broadcast: 👆 and 👽 are capability switches on both casts (👆 off: no one annotates; 👽 off: no one drives from outside) |
| repo | pair |
| issue file | workshop/issues/000523-broadcast-and-are-capability-switches-on-both-casts-off-no-one-annotates-off-no-one-drives-from-outside.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 45da33e382d7c2564804897098b6fc43414b82fe..56df060998e7dbc55180c55e895b5a0e6712727f |
| command | sdlc milestone-close --issue 523 --milestone M1 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-13T00:17:27-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

M1 has strong admission fencing and cleanup tests, but replay seeking demonstrably resurrects cleared annotations. Production control writes also bypass the acknowledgment assumed by the new failure handling. The core-concepts inventory and README need corrections. M2 controls and overlay integration were treated as pending work, not missing M1 deliverables.

1. **Strengths**

   - Generation-bound admissions cover delayed HTTP bodies and stale requests across OFF→ON.
   - The pure capability reducer explicitly models retryable, broken, and ended states, with sequence tests.
   - Cleanup preserves partial release progress, deduplicates recording operations, and accounts for equal-timestamp replay ordering.
   - Atlas documentation covers the new authority and recording contracts.

2. **Critical findings**

   - JavaScript replay catch-up drops annotation clears before resume, resurrecting old marks.
   - Production control emission reports enqueue success as committed success, preventing recorder failures from reaching the controller.
   - The M1 core-concepts inventory names a nonexistent implementation file. Classified Critical under the requested core-table consistency gate.

3. **Important findings**

   - README documentation still promises annotation and driving from links without describing the new default-OFF admission rules.

4. **Minor findings**

   None.

```findings
findings:
  - id: new
    severity: Critical
    family: recording-consumer-parity
    title: |
      Replay seeking resurrects annotations cleared before a resume
    detail: |
      cmd/internal/webcast/web/replay.js:38-61 lacks the AnnotationClear handling added at recording/catchup.go:88. Using the new capability-events.jsonl fixture, sequential reduction yields notes ["new"] and no strokes, but reduction through JavaScript catchUp yields notes ["old","new"] and stroke ["old"]; an inline assertion reproduced this failure. Port the clear-barrier handling, including clearing kept note IDs, and add the fixture to catch-up/Player seek parity tests. ARCH-PURPOSE: enumerate and update every recording consumer.
  - id: new
    severity: Critical
    family: production-effect-acknowledgment
    title: |
      Controller failure handling assumes an acknowledgment the production sink never supplies
    detail: |
      cmd/internal/webcast/drive/controller_mem.go:203-225 treats successful Put as a committed control record. Production wiring uses cast_drive.go:39-43 → Chrome.Emit at chrome.go:109-114, which calls Recorder.Send and immediately returns nil; enqueue refusal and later sink failures cannot reach this branch. Consequently requests/grants can report success and leave authority On after recording failure, contrary to the terminal-failure contract. Introduce an acknowledged recorder control-write seam and exercise it with delayed, closed, and failing recorder outcomes; TestUncertainControlSinkFreezesAuthority currently tests only a synchronous substitute. ARCH-ORDER, ARCH-MOCK.
  - id: new
    severity: Critical
    family: core-concept-inventory-consistency
    title: |
      The M1 core-concepts table identifies a nonexistent Board implementation file
    detail: |
      workshop/plans/000523-cast-capability-switches-plan.md:82 lists board/capability.go as modified, but that path is absent at pinned HEAD. Admission and cleanup are implemented in newly added board/annotation.go, including AnnotationAdmission at line 15. Correct the location and new/modified status, and record the implementation mapping under ## Revisions as required by the core-concepts gate.
  - id: new
    severity: Important
    family: user-surface-documentation
    title: |
      README update appears missing for default-OFF web capabilities
    detail: |
      README.md:1014-1017 and 1063 onward still describe pointer/drive-link holders as able to annotate or request control, while this range changes both capabilities to default OFF and adds server refusals. README.md is unchanged in the range. Document the current admission defaults and capability-versus-link distinction, accurately distinguishing pending M2 controls from implemented behavior.
```

5. **Test coverage notes**

   - Passed: all webcast Go packages; race tests for webcast, Board, capability, drive, and recorder; 13 JavaScript model tests; 30 replay tests; diff whitespace check.
   - Failed reproduction: the new recording fixture violates JavaScript catch-up parity after resume.
   - Broader verification was limited: couchtty could not compile because generated `runtimebundle/assets/runtime/files` was absent. Artifactpath tests also failed on that missing generated tree. These are checkout prerequisites, not established implementation defects.
   - No prior findings required disposition. The checkout remains unchanged.

6. **Architectural notes for upcoming work**

   | Marker | Result |
   |---|---|
   | ARCH-DRY | Pass: shared capability transitions serve both authorities. |
   | ARCH-PURE | Pass: lifecycle policy remains pure; locking and IO reside in adapters. |
   | ARCH-PURPOSE | Flag: JavaScript replay catch-up was omitted from the recording-consumer sweep. |
   | ARCH-MOCK | Flag: synchronous control-sink tests do not exercise production enqueue semantics. |
   | ARCH-CONSTRAINTS | Pass for inspected M1 mechanisms: single-flight workers and latest-value subscriptions bound added concurrency and buffering. |
   | ARCH-SECURE | Pass: server admission narrows link permissions; internal permits are excluded from decoded input. |
   | ARCH-ORDER | Flag: enqueue success is incorrectly treated as confirmed commitment. |
   | ARCH-FUNERAL | Pass: new state follows cast lifetime; cleanup records use the existing recording lifecycle. |

7. **Plan revision recommendations**

   Add `## Revisions` entries that:

   - Enumerate Go and JavaScript recording/replay consumers, explicitly including `replay.js::catchUp` and `Player.seek`.
   - Specify acknowledged production control writes and failure tests through that actual adapter.
   - Correct the Board implementation path/status and record the M1 README update.

---

## Re-review — 2026-10-13T00:37:42-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 523 — broadcast: 👆 and 👽 are capability switches on both casts (👆 off: no one annotates; 👽 off: no one drives from outside) |
| repo | pair |
| issue file | workshop/issues/000523-broadcast-and-are-capability-switches-on-both-casts-off-no-one-annotates-off-no-one-drives-from-outside.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 45da33e382d7c2564804897098b6fc43414b82fe..7ff7f694885abb170423df900c6945ff02c8add6 |
| command | sdlc milestone-close --issue 523 --milestone M1 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-13T00:37:42-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

M1’s authority, recording, and server behavior match the documented milestone scope. All four prior findings are addressed, with mutation-tested regressions for both behavioral fixes. No new blocking findings. M2’s controls, viewer integration, and overlay work remain pending as documented.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      replay.js:45 retains annotation-clear and clears kept note IDs. The shared-fixture prefix and Player.seek tests pass; removing this handling makes both fail with resurrected annotations.
  - id: BR-2
    disposition: addressed
    note: |
      cast_drive.go now routes control writes through Chrome.WriteControl to recorder/control.go:15, which awaits reducer acceptance and sink completion. Production-adapter tests cover delayed success/failure, closed recorders, cancellation, and shutdown. Replacing acknowledgment with Chrome.Emit makes TestProductionControlRefusesClosedRecorder fail.
  - id: BR-3
    disposition: addressed
    note: |
      The plan table at line 82 now identifies modified board/board.go and new board/annotation.go; Revisions at lines 447–448 records that mapping. The pinned diff confirms both classifications and AnnotationAdmission exists in annotation.go.
  - id: BR-4
    disposition: addressed
    note: |
      README.md:1020–1026 documents independent default-OFF capabilities, persistent links, cleanup behavior, and pending M2 UI. Board/controller zero-value authority and server admission checks substantiate the corrected description.
```

1. **Strengths**
   - Generation-bound admission fences delayed HTTP bodies and queued input across OFF/ON cycles.
   - `recorder/control.go:15` acknowledges actual writes and preserves terminal failure evidence.
   - Replay tests compare notes and strokes across every fixture prefix and nonsequential seeks.
   - Cleanup tests exercise partial progress, lost acknowledgments, explicit retry, and shutdown ownership.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage**
   - Passed all webcast Go packages and 112 JavaScript tests.
   - Passed race tests for webcast, Board, capability, drive, and recorder.
   - Passed artifactpath and couchtty tests after generating the required runtime bundle in a disposable copy.
   - Both behavioral correction mutations failed as expected. Temporary copies were removed; the reviewed checkout remains unchanged.
   - Full `make test` and live Chrome conformance were not rerun in this review.

6. **Architectural notes**
   - **ARCH-DRY — Pass:** Shared capability transitions and acknowledgment helper centralize lifecycle rules.
   - **ARCH-PURE — Pass:** Transition/projection logic is directly tested without IO; integration owners manage effects.
   - **ARCH-PURPOSE — Pass:** Recording-consumer sweep covers Go reduction/catch-up, split-file replay, JavaScript reduction/catch-up, and seeking.
   - **ARCH-MOCK — Pass:** Stateful doubles exercise production seams; existing live conformance harnesses remain available.
   - **ARCH-CONSTRAINTS — Pass:** Single-flight capability work and latest-value subscriptions bound concurrency and buffering.
   - **ARCH-SECURE — Pass:** Server-side role intersection and opaque admission permits prevent client-supplied authority.
   - **ARCH-ORDER — Pass:** Explicit transitions fence stale completions; controlled tests exercise cancellation and uncertain outcomes.
   - **ARCH-FUNERAL — Pass:** Workers/subscriptions terminate with their owners; new records use the existing recording lifecycle.

   M2 should retain these invariants when wiring visible controls and replacement-document overlay permissions.

7. **Plan revision recommendations:** None for M1.
