---
gate: boundary-review
issue: 523
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-13T00:17:27-07:00"
      dispatched: "2026-10-13T00:11:23-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: Replay seeking resurrects annotations cleared before a resume
          detail: 'cmd/internal/webcast/web/replay.js:38-61 lacks the AnnotationClear handling added at recording/catchup.go:88. Using the new capability-events.jsonl fixture, sequential reduction yields notes ["new"] and no strokes, but reduction through JavaScript catchUp yields notes ["old","new"] and stroke ["old"]; an inline assertion reproduced this failure. Port the clear-barrier handling, including clearing kept note IDs, and add the fixture to catch-up/Player seek parity tests. ARCH-PURPOSE: enumerate and update every recording consumer.'
          family: recording-consumer-parity
          round: 1
        - id: BR-2
          severity: Critical
          title: Controller failure handling assumes an acknowledgment the production sink never supplies
          detail: cmd/internal/webcast/drive/controller_mem.go:203-225 treats successful Put as a committed control record. Production wiring uses cast_drive.go:39-43 → Chrome.Emit at chrome.go:109-114, which calls Recorder.Send and immediately returns nil; enqueue refusal and later sink failures cannot reach this branch. Consequently requests/grants can report success and leave authority On after recording failure, contrary to the terminal-failure contract. Introduce an acknowledged recorder control-write seam and exercise it with delayed, closed, and failing recorder outcomes; TestUncertainControlSinkFreezesAuthority currently tests only a synchronous substitute. ARCH-ORDER, ARCH-MOCK.
          family: production-effect-acknowledgment
          round: 1
        - id: BR-3
          severity: Critical
          title: The M1 core-concepts table identifies a nonexistent Board implementation file
          detail: 'workshop/plans/000523-cast-capability-switches-plan.md:82 lists board/capability.go as modified, but that path is absent at pinned HEAD. Admission and cleanup are implemented in newly added board/annotation.go, including AnnotationAdmission at line 15. Correct the location and new/modified status, and record the implementation mapping under ## Revisions as required by the core-concepts gate.'
          family: core-concept-inventory-consistency
          round: 1
        - id: BR-4
          severity: Important
          title: README update appears missing for default-OFF web capabilities
          detail: README.md:1014-1017 and 1063 onward still describe pointer/drive-link holders as able to annotate or request control, while this range changes both capabilities to default OFF and adds server refusals. README.md is unchanged in the range. Document the current admission defaults and capability-versus-link distinction, accurately distinguishing pending M2 controls from implemented behavior.
          family: user-surface-documentation
          round: 1
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 2
      timestamp: "2026-10-13T00:37:42-07:00"
      dispatched: "2026-10-13T00:32:50-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: replay.js:45 retains annotation-clear and clears kept note IDs. The shared-fixture prefix and Player.seek tests pass; removing this handling makes both fail with resurrected annotations.
          round: 2
        - id: BR-2
          disposition: addressed
          note: cast_drive.go now routes control writes through Chrome.WriteControl to recorder/control.go:15, which awaits reducer acceptance and sink completion. Production-adapter tests cover delayed success/failure, closed recorders, cancellation, and shutdown. Replacing acknowledgment with Chrome.Emit makes TestProductionControlRefusesClosedRecorder fail.
          round: 2
        - id: BR-3
          disposition: addressed
          note: The plan table at line 82 now identifies modified board/board.go and new board/annotation.go; Revisions at lines 447–448 records that mapping. The pinned diff confirms both classifications and AnnotationAdmission exists in annotation.go.
          round: 2
        - id: BR-4
          disposition: addressed
          note: README.md:1020–1026 documents independent default-OFF capabilities, persistent links, cleanup behavior, and pending M2 UI. Board/controller zero-value authority and server admission checks substantiate the corrected description.
          round: 2
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      reviewed: 7ff7f694885abb170423df900c6945ff02c8add6
      blocked: false
    - "n": 3
      timestamp: "2026-10-13T09:58:04-07:00"
      dispatched: "2026-10-13T09:51:13-07:00"
      agent: codex
      findings:
        - id: BR-5
          severity: Critical
          title: Pre-OFF annotations can acquire fresh authority after OFF/ON
          detail: 'cmd/internal/webcast/recorder/recorder.go:290 assigns current authority at ingress. A disposable regression retained a valid second stroke before OFF, completed OFF/ON, then delivered it: the recorder accepted it and restored one stroke. web/annotate.js:596 also leaves a held gesture active after annotation-clear when capability delivery is delayed. ARCH-ORDER, ARCH-SECURE, ARCH-PURPOSE: carry the originating capability generation through operator and viewer mutations, reject retired generations at ingress and dispatch, and cancel held gestures at the clear barrier. Add controlled pre-ingress delay regressions across these consumers.'
          family: capability-generation-provenance
          round: 3
        - id: BR-6
          severity: Minor
          title: Permission retry polling contradicts the documented operating envelope
          detail: 'cmd/internal/webcast/chrome_operator.go:107 introduces a 250 ms retry ticker, while workshop/plans/000523-cast-capability-switches-plan.md:177 promises no new periodic polling. ARCH-CONSTRAINTS: document the retry lifetime, bounds, and termination policy in a Revisions entry, or replace polling with readiness-driven delivery.'
          family: operating-envelope-traceability
          round: 3
      boundary: M2
      recipe: milestone-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#523 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-13T00:17:27-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `recording-consumer-parity` Replay seeking resurrects annotations cleared before a resume
  cmd/internal/webcast/web/replay.js:38-61 lacks the AnnotationClear handling added at recording/catchup.go:88. Using the new capability-events.jsonl fixture, sequential reduction yields notes ["new"] and no strokes, but reduction through JavaScript catchUp yields notes ["old","new"] and stroke ["old"]; an inline assertion reproduced this failure. Port the clear-barrier handling, including clearing kept note IDs, and add the fixture to catch-up/Player seek parity tests. ARCH-PURPOSE: enumerate and update every recording consumer.
- **BR-2** [Critical] `production-effect-acknowledgment` Controller failure handling assumes an acknowledgment the production sink never supplies
  cmd/internal/webcast/drive/controller_mem.go:203-225 treats successful Put as a committed control record. Production wiring uses cast_drive.go:39-43 → Chrome.Emit at chrome.go:109-114, which calls Recorder.Send and immediately returns nil; enqueue refusal and later sink failures cannot reach this branch. Consequently requests/grants can report success and leave authority On after recording failure, contrary to the terminal-failure contract. Introduce an acknowledged recorder control-write seam and exercise it with delayed, closed, and failing recorder outcomes; TestUncertainControlSinkFreezesAuthority currently tests only a synchronous substitute. ARCH-ORDER, ARCH-MOCK.
- **BR-3** [Critical] `core-concept-inventory-consistency` The M1 core-concepts table identifies a nonexistent Board implementation file
  workshop/plans/000523-cast-capability-switches-plan.md:82 lists board/capability.go as modified, but that path is absent at pinned HEAD. Admission and cleanup are implemented in newly added board/annotation.go, including AnnotationAdmission at line 15. Correct the location and new/modified status, and record the implementation mapping under ## Revisions as required by the core-concepts gate.
- **BR-4** [Important] `user-surface-documentation` README update appears missing for default-OFF web capabilities
  README.md:1014-1017 and 1063 onward still describe pointer/drive-link holders as able to annotate or request control, while this range changes both capabilities to default OFF and adds server refusals. README.md is unchanged in the range. Document the current admission defaults and capability-versus-link distinction, accurately distinguishing pending M2 controls from implemented behavior.

## Round 2 — 2026-10-13T00:37:42-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — replay.js:45 retains annotation-clear and clears kept note IDs. The shared-fixture prefix and Player.seek tests pass; removing this handling makes both fail with resurrected annotations.
- BR-2 — addressed — cast_drive.go now routes control writes through Chrome.WriteControl to recorder/control.go:15, which awaits reducer acceptance and sink completion. Production-adapter tests cover delayed success/failure, closed recorders, cancellation, and shutdown. Replacing acknowledgment with Chrome.Emit makes TestProductionControlRefusesClosedRecorder fail.
- BR-3 — addressed — The plan table at line 82 now identifies modified board/board.go and new board/annotation.go; Revisions at lines 447–448 records that mapping. The pinned diff confirms both classifications and AnnotationAdmission exists in annotation.go.
- BR-4 — addressed — README.md:1020–1026 documents independent default-OFF capabilities, persistent links, cleanup behavior, and pending M2 UI. Board/controller zero-value authority and server admission checks substantiate the corrected description.

## Round 3 — 2026-10-13T09:58:04-07:00 (codex) — BLOCKED

### Raised

- **BR-5** [Critical] `capability-generation-provenance` Pre-OFF annotations can acquire fresh authority after OFF/ON
  cmd/internal/webcast/recorder/recorder.go:290 assigns current authority at ingress. A disposable regression retained a valid second stroke before OFF, completed OFF/ON, then delivered it: the recorder accepted it and restored one stroke. web/annotate.js:596 also leaves a held gesture active after annotation-clear when capability delivery is delayed. ARCH-ORDER, ARCH-SECURE, ARCH-PURPOSE: carry the originating capability generation through operator and viewer mutations, reject retired generations at ingress and dispatch, and cancel held gestures at the clear barrier. Add controlled pre-ingress delay regressions across these consumers.
- **BR-6** [Minor] `operating-envelope-traceability` Permission retry polling contradicts the documented operating envelope
  cmd/internal/webcast/chrome_operator.go:107 introduces a 250 ms retry ticker, while workshop/plans/000523-cast-capability-switches-plan.md:177 promises no new periodic polling. ARCH-CONSTRAINTS: document the retry lifetime, bounds, and termination policy in a Revisions entry, or replace polling with readiness-driven delivery.

## Open findings

- **BR-5** [Critical] `capability-generation-provenance` Pre-OFF annotations can acquire fresh authority after OFF/ON
- **BR-6** [Minor] `operating-envelope-traceability` Permission retry polling contradicts the documented operating envelope
