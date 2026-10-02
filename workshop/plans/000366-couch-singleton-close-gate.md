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

## Open findings

- **BR-4** [Critical] `resolved-runtime-root-propagation` Pair children interpret the selected global root as a repository-scoped artifact directory.
