---
gate: boundary-review
issue: 439
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-11T09:31:09-07:00"
      dispatched: "2026-10-11T09:29:14-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Minor
          title: release no-op exits 0 even with --fold, so an old skill believes events were folded
          detail: session_sync.go release branch prints only to stderr and returns 0; folds and the old auto-fold of submits silently do not happen. Consider exiting non-zero when --fold is passed.
          family: retired-verb-silent-success
          round: 1
        - id: BR-2
          severity: Minor
          title: Folding a submit or accepting a delivery-gap now requires a non-empty write
          detail: With release gone, an agent with nothing to change must rewrite a section with the same content to fold or accept a gap; document this in SKILL.md or add a fold-only form.
          family: fold-requires-content-write
          round: 1
        - id: BR-3
          severity: Minor
          title: Restamp can change another section's hash, making the agent's own held hash stale
          detail: If the Updated stamp lives inside a section, writing section A changes section B's hash; fixtures only put the stamp in the preamble.
          family: write-side-effects-outside-target
          round: 1
        - id: BR-4
          severity: Minor
          title: locked() creates .<file>.flock before checking the file is type session-sync
          detail: Running read/status against an ordinary markdown file leaves a stray dot-file behind.
          family: residue-before-validation
          round: 1
        - id: BR-5
          severity: Minor
          title: TestWriteUnderAnOperatorLockRefusesAndChangesNothing comment mentions a legacy agent lock it never writes
          family: test-comment-drift
          round: 1
        - id: BR-6
          severity: Minor
          title: Removing the release no-op is recorded only in the Log, with no tracked issue
          family: untracked-removal-followup
          round: 1
      recipe: milestone-review
      isolation: fenced
      reviewed: e91a498239dbeb89af6b0659677513314130a0a8
      blocked: false
---

# Gate ledger — pair#439 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-11T09:31:09-07:00 (claude) — passed

### Raised

- **BR-1** [Minor] `retired-verb-silent-success` release no-op exits 0 even with --fold, so an old skill believes events were folded
  session_sync.go release branch prints only to stderr and returns 0; folds and the old auto-fold of submits silently do not happen. Consider exiting non-zero when --fold is passed.
- **BR-2** [Minor] `fold-requires-content-write` Folding a submit or accepting a delivery-gap now requires a non-empty write
  With release gone, an agent with nothing to change must rewrite a section with the same content to fold or accept a gap; document this in SKILL.md or add a fold-only form.
- **BR-3** [Minor] `write-side-effects-outside-target` Restamp can change another section's hash, making the agent's own held hash stale
  If the Updated stamp lives inside a section, writing section A changes section B's hash; fixtures only put the stamp in the preamble.
- **BR-4** [Minor] `residue-before-validation` locked() creates .<file>.flock before checking the file is type session-sync
  Running read/status against an ordinary markdown file leaves a stray dot-file behind.
- **BR-5** [Minor] `test-comment-drift` TestWriteUnderAnOperatorLockRefusesAndChangesNothing comment mentions a legacy agent lock it never writes
- **BR-6** [Minor] `untracked-removal-followup` Removing the release no-op is recorded only in the Log, with no tracked issue

## Open findings

- **BR-1** [Minor] `retired-verb-silent-success` release no-op exits 0 even with --fold, so an old skill believes events were folded
- **BR-2** [Minor] `fold-requires-content-write` Folding a submit or accepting a delivery-gap now requires a non-empty write
- **BR-3** [Minor] `write-side-effects-outside-target` Restamp can change another section's hash, making the agent's own held hash stale
- **BR-4** [Minor] `residue-before-validation` locked() creates .<file>.flock before checking the file is type session-sync
- **BR-5** [Minor] `test-comment-drift` TestWriteUnderAnOperatorLockRefusesAndChangesNothing comment mentions a legacy agent lock it never writes
- **BR-6** [Minor] `untracked-removal-followup` Removing the release no-op is recorded only in the Log, with no tracked issue
