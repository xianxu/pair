---
gate: boundary-review
issue: 444
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-11T11:10:48-07:00"
      dispatched: "2026-10-11T11:08:53-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: openFlock's fallback (no flock file, check fails, re-open without create) has no test; deleting it leaves all tests green
          detail: 'sync.go:155-160. Every test reads first via r.hash, and that read creates the flock file through locked, so no test starts without a flock file and then fails the check. Removing lines 157-159 keeps the package green, including the race test at 400 runs. Fix: make the check a parameter of openFlock and test a check that creates the flock file and then refuses; assert a handle is returned. Family members in this window: only sync.go:157. The other openFlock branches are covered by the new torn-file test, newRig''s first read, and TestAWrongFileLeavesNoFlockBehind.'
          family: fix-branch-without-failing-test
          round: 1
        - id: BR-2
          severity: Minor
          title: The openFlock comment says the in-place rewriter holds the flock, but that is true only of session-sync writers
          detail: sync.go:147. The operator's editor also rewrites in place but is fenced by .lock, not the flock. Narrow the wording to session-sync writers.
          family: comment-claim-scope
          round: 1
      recipe: small-diff-review
      isolation: checkout
      blocked: true
    - "n": 2
      timestamp: "2026-10-11T11:12:37-07:00"
      dispatched: "2026-10-11T11:11:35-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: TestAFailedCheckDefersToAFlockCreatedMeanwhile (sync_test.go:200) goes red with the fallback (sync.go:155-157) removed and green with it restored; check is now an openFlock parameter.
          round: 2
        - id: BR-2
          disposition: addressed
          note: sync.go:146-148 now scopes the claim to "a couch writer", matching locked's "couch's exclusive flock" (sync.go:124-126), which already says parley is fenced by .lock instead.
          round: 2
      recipe: small-diff-review
      isolation: checkout
      reviewed: fcf20d5bca09db2c82a83e790518b2629f1cf7a2
      blocked: false
---

# Gate ledger — pair#444 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-11T11:10:48-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `fix-branch-without-failing-test` openFlock's fallback (no flock file, check fails, re-open without create) has no test; deleting it leaves all tests green
  sync.go:155-160. Every test reads first via r.hash, and that read creates the flock file through locked, so no test starts without a flock file and then fails the check. Removing lines 157-159 keeps the package green, including the race test at 400 runs. Fix: make the check a parameter of openFlock and test a check that creates the flock file and then refuses; assert a handle is returned. Family members in this window: only sync.go:157. The other openFlock branches are covered by the new torn-file test, newRig's first read, and TestAWrongFileLeavesNoFlockBehind.
- **BR-2** [Minor] `comment-claim-scope` The openFlock comment says the in-place rewriter holds the flock, but that is true only of session-sync writers
  sync.go:147. The operator's editor also rewrites in place but is fenced by .lock, not the flock. Narrow the wording to session-sync writers.

## Round 2 — 2026-10-11T11:12:37-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — TestAFailedCheckDefersToAFlockCreatedMeanwhile (sync_test.go:200) goes red with the fallback (sync.go:155-157) removed and green with it restored; check is now an openFlock parameter.
- BR-2 — addressed — sync.go:146-148 now scopes the claim to "a couch writer", matching locked's "couch's exclusive flock" (sync.go:124-126), which already says parley is fenced by .lock instead.

## Open findings

(none — every finding has been disposed)
