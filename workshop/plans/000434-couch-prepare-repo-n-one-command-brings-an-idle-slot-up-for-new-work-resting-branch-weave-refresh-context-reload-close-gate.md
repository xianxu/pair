---
gate: boundary-review
issue: 434
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-11T01:27:26-07:00"
      dispatched: "2026-10-11T01:04:35-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: README couch command listing omits the new couch --prepare verb
          detail: README.md:396-416 documents --relaunch and --reload-context with their refusals but not --prepare repo:N --confirm [--force-unknown] [--json] (not-resting, refresh-dirty).
          family: user-surface-docs-sweep
          round: 1
        - id: BR-2
          severity: Minor
          title: Uncertain prepare admission tells the user to verify before resending, though admission has no effect
          detail: slotOperationUncertain's message (messages.go:80) is reused for prepare; re-running prepare is safe and converges.
          family: verb-specific-error-guidance
          round: 1
        - id: BR-3
          severity: Minor
          title: Recheck ignores untracked files while admission counts them dirty
          detail: slot_prepare.go:601 uses --untracked-files=no; ProbeSlotGit includes untracked, so a refresh that creates an untracked file ends as reload refused (dirty) instead of refresh-dirty.
          family: consistent-dirty-predicate
          round: 1
        - id: BR-4
          severity: Minor
          title: Porcelain parsing and substrate FS read are inline in RunPrepare rather than pure helpers or the seam
          detail: Extract dirtyPaths([]byte) as a pure tested function; substrateCommits uses layergraph.OSFS{} directly instead of the injected IO.
          family: pure-core-io-shell
          round: 1
        - id: BR-5
          severity: Minor
          title: prepareLiveRestart comment still says it admits relaunch or reload-context only
          family: stale-doc-comment
          round: 1
      recipe: milestone-review
      blocked: true
    - "n": 2
      timestamp: "2026-10-11T01:42:40-07:00"
      dispatched: "2026-10-11T01:30:20-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: README.md:412-415 now lists couch --prepare repo:N --confirm [--force-unknown] [--json] with the refresh-dirty refusal.
          round: 2
        - id: BR-2
          disposition: addressed
          note: prepare.go:40-44 gives a prepare-specific remedy; the TestPrepareCLI lost-outcome subtest asserts "run couch --prepare again" and no recover-plan text.
          round: 2
        - id: BR-3
          disposition: addressed
          note: The recheck uses slotGitStatusArgs + ParseSlotGitStatus; TestRunPrepareUntrackedOutputIsRefreshDirty fails without the fix (the reload would run).
          round: 2
        - id: BR-4
          disposition: addressed
          note: Porcelain parsing is now the pure, tested SlotGitChangedPaths/ParseSlotGitStatus; the remaining OSFS read in substrateCommits runs on temp-dir fixtures in tests, acceptable.
          round: 2
        - id: BR-5
          disposition: addressed
          note: live_restart.go:245-248 comment now names relaunch, reload-context or prepare.
          round: 2
      recipe: milestone-review
      reviewed: 4686716e8ea06f41ddcad409b99075e6ec5398be
      blocked: false
---

# Gate ledger — pair#434 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-11T01:27:26-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `user-surface-docs-sweep` README couch command listing omits the new couch --prepare verb
  README.md:396-416 documents --relaunch and --reload-context with their refusals but not --prepare repo:N --confirm [--force-unknown] [--json] (not-resting, refresh-dirty).
- **BR-2** [Minor] `verb-specific-error-guidance` Uncertain prepare admission tells the user to verify before resending, though admission has no effect
  slotOperationUncertain's message (messages.go:80) is reused for prepare; re-running prepare is safe and converges.
- **BR-3** [Minor] `consistent-dirty-predicate` Recheck ignores untracked files while admission counts them dirty
  slot_prepare.go:601 uses --untracked-files=no; ProbeSlotGit includes untracked, so a refresh that creates an untracked file ends as reload refused (dirty) instead of refresh-dirty.
- **BR-4** [Minor] `pure-core-io-shell` Porcelain parsing and substrate FS read are inline in RunPrepare rather than pure helpers or the seam
  Extract dirtyPaths([]byte) as a pure tested function; substrateCommits uses layergraph.OSFS{} directly instead of the injected IO.
- **BR-5** [Minor] `stale-doc-comment` prepareLiveRestart comment still says it admits relaunch or reload-context only

## Round 2 — 2026-10-11T01:42:40-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — README.md:412-415 now lists couch --prepare repo:N --confirm [--force-unknown] [--json] with the refresh-dirty refusal.
- BR-2 — addressed — prepare.go:40-44 gives a prepare-specific remedy; the TestPrepareCLI lost-outcome subtest asserts "run couch --prepare again" and no recover-plan text.
- BR-3 — addressed — The recheck uses slotGitStatusArgs + ParseSlotGitStatus; TestRunPrepareUntrackedOutputIsRefreshDirty fails without the fix (the reload would run).
- BR-4 — addressed — Porcelain parsing is now the pure, tested SlotGitChangedPaths/ParseSlotGitStatus; the remaining OSFS read in substrateCommits runs on temp-dir fixtures in tests, acceptable.
- BR-5 — addressed — live_restart.go:245-248 comment now names relaunch, reload-context or prepare.

## Open findings

(none — every finding has been disposed)
