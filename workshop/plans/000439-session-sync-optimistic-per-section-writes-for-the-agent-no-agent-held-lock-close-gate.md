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
    - "n": 2
      timestamp: "2026-10-11T09:34:56-07:00"
      dispatched: "2026-10-11T09:33:37-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: 'Only --fold refuses; release --accept-gap still exits 0 though the old Release persisted GapAccepted (also old bare-release submit auto-fold is dropped). Apply the lesson''s rule: refuse every effect flag except --accept-edits.'
          round: 2
        - id: BR-2
          disposition: addressed
          note: SKILL.md:278-279 documents folding by writing the section back as read printed it; write accepts a non-empty unchanged body.
          round: 2
        - id: BR-3
          disposition: addressed
          note: Documented in SKILL.md:280-282 (restamp in a section makes its held hash stale; read again); by-design stale retry recorded in issue Log.
          round: 2
        - id: BR-4
          disposition: addressed
          note: locked() runs load() before creating .flock (sync.go:129-132); TestAWrongFileLeavesNoFlockBehind goes red when the precheck is reverted.
          round: 2
        - id: BR-5
          disposition: addressed
          note: sync_test.go:91 comment now describes the read-under-operator-lock assertion.
          round: 2
        - id: BR-6
          disposition: addressed
          note: workshop/issues/000440 tracks removal of the release verb with Spec and Done-when.
          round: 2
      recipe: fix-review
      isolation: fenced
      reviewed: 89591551f1a40c6701f584ef8c1b327ac9426cda
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

## Round 2 — 2026-10-11T09:34:56-07:00 (claude) — passed

### Disposed

- BR-1 — not-addressed — Only --fold refuses; release --accept-gap still exits 0 though the old Release persisted GapAccepted (also old bare-release submit auto-fold is dropped). Apply the lesson's rule: refuse every effect flag except --accept-edits.
- BR-2 — addressed — SKILL.md:278-279 documents folding by writing the section back as read printed it; write accepts a non-empty unchanged body.
- BR-3 — addressed — Documented in SKILL.md:280-282 (restamp in a section makes its held hash stale; read again); by-design stale retry recorded in issue Log.
- BR-4 — addressed — locked() runs load() before creating .flock (sync.go:129-132); TestAWrongFileLeavesNoFlockBehind goes red when the precheck is reverted.
- BR-5 — addressed — sync_test.go:91 comment now describes the read-under-operator-lock assertion.
- BR-6 — addressed — workshop/issues/000440 tracks removal of the release verb with Spec and Done-when.

## Open findings

- **BR-1** [Minor] `retired-verb-silent-success` release no-op exits 0 even with --fold, so an old skill believes events were folded
