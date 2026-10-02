---
gate: boundary-review
issue: 366
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-01T23:21:41-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: Adoption ignores unreadable numbered-slot state.
          detail: cmd/internal/couchsingleton/inspect.go:210 checks only PreviewSnapshot's returned error and snapshot.Unreadable. Numbered-slot failures are instead carried in snapshot.Slots[].Err (cmd/internal/couchcore/threadstore_snapshot.go:31–53). A corrupt slot thread.json therefore produces READY with no blockers; TestReviewCorruptSlotBlocksAdoption reproduces this. Inspect every slot observation and refuse unresolved state. Include slot-local evidence in the digest and publication revalidation, since hashTree currently covers only the global namespace. ARCH-SECURE, ARCH-PURPOSE.
          family: adoption-evidence-completeness
          round: 1
        - id: BR-2
          severity: Critical
          title: Exclusion admits stores with surviving live incarnations.
          detail: cmd/internal/couchsingleton/inspect.go:191 observes only the supervisor lease and discards snapshot.Records; model.go:95 then accepts exclusion of a populated store. With a free supervisor lease and a recorded incarnation whose PID/start identity is still live, preview returns READY. TestReviewExcludedLiveWrapperBlocksAdoption reproduces this. This contradicts the issue's requirement to report surviving wrappers in other namespaces as migration conflicts. Inspect incarnation ownership through the existing process/lifecycle seams, preserve unknown outcomes, and block exclusion until absence is established. ARCH-ORDER, ARCH-PURPOSE.
          family: migration-liveness-evidence
          round: 1
        - id: BR-3
          severity: Critical
          title: Successful adoption can publish an unreadable selection.
          detail: cmd/internal/couchsingleton/manager.go:204–212 publishes serialized Selection without enforcing Read's 65,536-byte limit at line 37. Requests permit up to 4096 exclusions. A valid 600-exclusion request successfully published 139,628 bytes, after which Read refused with "file exceeds 65536 bytes"; TestReviewSelectionWriterReadLimit reproduces this. Share the size contract between writer and reader, reject oversized selections before publication, and test the boundary plus successful round trips. ARCH-CONSTRAINTS, ARCH-SECURE.
          family: persisted-writer-reader-contract
          round: 1
      recipe: milestone-review
      blocked: true
    - "n": 2
      timestamp: "2026-10-01T23:46:32-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: Slot errors, payloads, and topology now participate in locked inspection and publication revalidation. Disabling the slot-error guard makes TestAdoptionRejectsUnreadableSlotEvidence/corrupt fail.
          round: 2
        - id: BR-2
          disposition: addressed
          note: Exclusion checks recorded incarnations and creating owners, preserving unknown liveness and rechecking before publication. Disabling the liveness guard makes live, unknown, and identity-error regression cases fail.
          round: 2
        - id: BR-3
          disposition: addressed
          note: Preview, publication, and reading share maxSelectionBytes. Exact-boundary round trips pass; disabling the encoding limit makes oversize-publication regressions fail.
          round: 2
      findings:
        - id: BR-4
          severity: Critical
          title: Pair children interpret the selected global root as a repository-scoped artifact directory.
          detail: cmd/internal/couchcmd/singleton.go:247 exports roots.PairDataDir as PAIR_DATA_DIR, but cmd/internal/launcher/runcli.go:102–118 treats that variable as an already-scoped directory and derives its global root from HOME/XDG without consuming COUCH_PAIR_DATA_DIR. Hosted artifacts therefore use the flat root; custom selected roots also leave global claim readers pointed at ambient storage. Fix the complete parent/launcher root contract and add actual launcher artifact/read/resume coverage. ARCH-PURPOSE, ARCH-DRY.
          family: resolved-runtime-root-propagation
          round: 2
      recipe: milestone-review
      blocked: true
    - "n": 3
      timestamp: "2026-10-02T00:01:01-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: Slot inspection retains transaction locks and includes unreadable state, topology and payload evidence; corresponding regressions pass.
          round: 3
        - id: BR-2
          disposition: addressed
          note: Excluded inventories require confirmed absence of recorded incarnations and creating owners; live, unknown and revalidation regressions pass.
          round: 3
        - id: BR-3
          disposition: addressed
          note: Serialization and reading share maxSelectionBytes; boundary round-trip and pre-publication overflow regressions pass.
          round: 3
        - id: BR-4
          disposition: addressed
          note: Couch clears PAIR_DATA_DIR and passes COUCH_PAIR_DATA_DIR; launcher and embedded extraction consume the selected global root. The real create/list/resume regression fails when an isolated Go overlay disables selected-root consumption, then passes against unchanged HEAD.
          round: 3
      recipe: milestone-review
      blocked: false
    - "n": 4
      timestamp: "2026-10-02T09:00:06-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: StoreInspection fences numbered-slot sources; unreadable-slot, changed-evidence, and publication-lock regressions pass.
          round: 4
        - id: BR-2
          disposition: addressed
          note: ObserveMigrationProcesses covers incarnations and creating owners; live/unknown exclusion and publication recheck regressions pass.
          round: 4
        - id: BR-3
          disposition: addressed
          note: encodeSelection shares the reader's size limit; exact-boundary round trips and overflow rejection regressions pass.
          round: 4
        - id: BR-4
          disposition: addressed
          note: Selected global roots reach scoped launcher storage and embedded extraction; real create/list/resume and parent runner regressions pass.
          round: 4
      findings:
        - id: BR-5
          severity: Critical
          title: Derived child HOME and TMPDIR can escape explicit isolation through symlinks.
          detail: 'cmd/internal/couchcmd/singleton.go:184-192 substitutes isolated/home without validating its physical destination and creates isolated/tmp without containment validation; line 252 exports both to children. Preexisting symlinks to an outside directory are followed by MkdirAll and descendant writes. A production-path overlay regression fails for both cases. ARCH-SECURE and ARCH-PURPOSE: validate physical containment of every derived runtime root before publication or child effects, and export only validated paths. Add permanent tests using outside temporary sentinels for fallback HOME and TMPDIR through both runner entry points.'
          family: isolation-derived-root-containment
          round: 4
      recipe: milestone-review
      blocked: true
    - "n": 5
      timestamp: "2026-10-02T09:10:51-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: Slot inspection preserves unreadable-state errors and includes slot payloads in adoption evidence.
          round: 5
        - id: BR-2
          disposition: addressed
          note: Nonselected stores retain live and unknown incarnation evidence, preventing unsafe exclusion.
          round: 5
        - id: BR-3
          disposition: addressed
          note: Selection encoding and reading share the encoded-size limit, with boundary regression coverage.
          round: 5
        - id: BR-4
          disposition: addressed
          note: Launcher and embedded runtime extraction consume the selected global root; launcher integration tests pass.
          round: 5
        - id: BR-5
          disposition: addressed
          note: Derived HOME, TMPDIR and XDG roots are validated before publication and exported as physical paths. Tests pass normally and under race detection; disabling validation in a temporary overlay makes HOME/TMPDIR escape tests fail through both runner entry points.
          round: 5
      recipe: milestone-review
      blocked: false
---

# Gate ledger — pair#366 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-01T23:21:41-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `adoption-evidence-completeness` Adoption ignores unreadable numbered-slot state.
  cmd/internal/couchsingleton/inspect.go:210 checks only PreviewSnapshot's returned error and snapshot.Unreadable. Numbered-slot failures are instead carried in snapshot.Slots[].Err (cmd/internal/couchcore/threadstore_snapshot.go:31–53). A corrupt slot thread.json therefore produces READY with no blockers; TestReviewCorruptSlotBlocksAdoption reproduces this. Inspect every slot observation and refuse unresolved state. Include slot-local evidence in the digest and publication revalidation, since hashTree currently covers only the global namespace. ARCH-SECURE, ARCH-PURPOSE.
- **BR-2** [Critical] `migration-liveness-evidence` Exclusion admits stores with surviving live incarnations.
  cmd/internal/couchsingleton/inspect.go:191 observes only the supervisor lease and discards snapshot.Records; model.go:95 then accepts exclusion of a populated store. With a free supervisor lease and a recorded incarnation whose PID/start identity is still live, preview returns READY. TestReviewExcludedLiveWrapperBlocksAdoption reproduces this. This contradicts the issue's requirement to report surviving wrappers in other namespaces as migration conflicts. Inspect incarnation ownership through the existing process/lifecycle seams, preserve unknown outcomes, and block exclusion until absence is established. ARCH-ORDER, ARCH-PURPOSE.
- **BR-3** [Critical] `persisted-writer-reader-contract` Successful adoption can publish an unreadable selection.
  cmd/internal/couchsingleton/manager.go:204–212 publishes serialized Selection without enforcing Read's 65,536-byte limit at line 37. Requests permit up to 4096 exclusions. A valid 600-exclusion request successfully published 139,628 bytes, after which Read refused with "file exceeds 65536 bytes"; TestReviewSelectionWriterReadLimit reproduces this. Share the size contract between writer and reader, reject oversized selections before publication, and test the boundary plus successful round trips. ARCH-CONSTRAINTS, ARCH-SECURE.

## Round 2 — 2026-10-01T23:46:32-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — Slot errors, payloads, and topology now participate in locked inspection and publication revalidation. Disabling the slot-error guard makes TestAdoptionRejectsUnreadableSlotEvidence/corrupt fail.
- BR-2 — addressed — Exclusion checks recorded incarnations and creating owners, preserving unknown liveness and rechecking before publication. Disabling the liveness guard makes live, unknown, and identity-error regression cases fail.
- BR-3 — addressed — Preview, publication, and reading share maxSelectionBytes. Exact-boundary round trips pass; disabling the encoding limit makes oversize-publication regressions fail.

### Raised

- **BR-4** [Critical] `resolved-runtime-root-propagation` Pair children interpret the selected global root as a repository-scoped artifact directory.
  cmd/internal/couchcmd/singleton.go:247 exports roots.PairDataDir as PAIR_DATA_DIR, but cmd/internal/launcher/runcli.go:102–118 treats that variable as an already-scoped directory and derives its global root from HOME/XDG without consuming COUCH_PAIR_DATA_DIR. Hosted artifacts therefore use the flat root; custom selected roots also leave global claim readers pointed at ambient storage. Fix the complete parent/launcher root contract and add actual launcher artifact/read/resume coverage. ARCH-PURPOSE, ARCH-DRY.

## Round 3 — 2026-10-02T00:01:01-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — Slot inspection retains transaction locks and includes unreadable state, topology and payload evidence; corresponding regressions pass.
- BR-2 — addressed — Excluded inventories require confirmed absence of recorded incarnations and creating owners; live, unknown and revalidation regressions pass.
- BR-3 — addressed — Serialization and reading share maxSelectionBytes; boundary round-trip and pre-publication overflow regressions pass.
- BR-4 — addressed — Couch clears PAIR_DATA_DIR and passes COUCH_PAIR_DATA_DIR; launcher and embedded extraction consume the selected global root. The real create/list/resume regression fails when an isolated Go overlay disables selected-root consumption, then passes against unchanged HEAD.

## Round 4 — 2026-10-02T09:00:06-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — StoreInspection fences numbered-slot sources; unreadable-slot, changed-evidence, and publication-lock regressions pass.
- BR-2 — addressed — ObserveMigrationProcesses covers incarnations and creating owners; live/unknown exclusion and publication recheck regressions pass.
- BR-3 — addressed — encodeSelection shares the reader's size limit; exact-boundary round trips and overflow rejection regressions pass.
- BR-4 — addressed — Selected global roots reach scoped launcher storage and embedded extraction; real create/list/resume and parent runner regressions pass.

### Raised

- **BR-5** [Critical] `isolation-derived-root-containment` Derived child HOME and TMPDIR can escape explicit isolation through symlinks.
  cmd/internal/couchcmd/singleton.go:184-192 substitutes isolated/home without validating its physical destination and creates isolated/tmp without containment validation; line 252 exports both to children. Preexisting symlinks to an outside directory are followed by MkdirAll and descendant writes. A production-path overlay regression fails for both cases. ARCH-SECURE and ARCH-PURPOSE: validate physical containment of every derived runtime root before publication or child effects, and export only validated paths. Add permanent tests using outside temporary sentinels for fallback HOME and TMPDIR through both runner entry points.

## Round 5 — 2026-10-02T09:10:51-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — Slot inspection preserves unreadable-state errors and includes slot payloads in adoption evidence.
- BR-2 — addressed — Nonselected stores retain live and unknown incarnation evidence, preventing unsafe exclusion.
- BR-3 — addressed — Selection encoding and reading share the encoded-size limit, with boundary regression coverage.
- BR-4 — addressed — Launcher and embedded runtime extraction consume the selected global root; launcher integration tests pass.
- BR-5 — addressed — Derived HOME, TMPDIR and XDG roots are validated before publication and exported as physical paths. Tests pass normally and under race detection; disabling validation in a temporary overlay makes HOME/TMPDIR escape tests fail through both runner entry points.

## Open findings

(none — every finding has been disposed)
