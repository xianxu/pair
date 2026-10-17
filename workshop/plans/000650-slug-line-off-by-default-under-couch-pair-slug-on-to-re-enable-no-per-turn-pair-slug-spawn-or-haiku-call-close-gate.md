---
gate: boundary-review
issue: 650
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: e652dfd7-4c3f-4bd6-98d6-8e457d5a2ab1
            repository: github.com/xianxu/pair
            issue_id: "000650"
            recipe: small-diff-review
            round: 1
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/b40f68b20b95d769
            reviewed_head: 1ecb83d29deed3d4f82f8ea0a82dffdeb5e2f35f
            slot: pair:8
            requested_at: "2026-10-17T22:04:18.647166Z"
          phase: finished
          pid: 8520
          started_at: "2026-10-17T22:04:18.888098Z"
          finished_at: "2026-10-17T22:05:52.354808Z"
          session_id: 3720a09c-a1ed-409e-af5b-4c10095c5f67
          outcome: returned
      "n": 1
      timestamp: "2026-10-17T15:05:54-07:00"
      dispatched: "2026-10-17T15:04:15-07:00"
      agent: claude
      findings:
        - kind: defect
          id: BR-1
          severity: Important
          title: 'The slug gate rule is written out twice: wrap.go:754 and slugcmd.go:119'
          detail: ARCH-DRY. Both sites write out the same rule, CouchHostedEnv && PAIR_SLUG != "on". Those are the only two copies in this window. Move it into one helper, e.g. launcher.SlugEnabled(getenv), call it from both, and test the truth table once, including that values like ON or 1 keep it off (fail closed).
          family: single-source-policy-predicate
          round: 1
        - kind: defect
          id: BR-2
          severity: Important
          title: Done when does not cover the slugcmd.Run Stop-hook gate the Revisions added
          detail: 'Clause 1 only describes the wrapper''s spawn. Clause 2, read literally ("no pair slug processes"), is wrong: under Couch the Stop hook still starts a short-lived pair-slug that stops at the gate. Change clause 1 to name both gates, and make clause 2 about model calls and wrapper spawns.'
          family: done-when-tracks-revisions
          round: 1
        - kind: unverifiable
          id: BR-3
          severity: Important
          title: Done-when clause 2 (live check across the fleet after restart) is not verified
          detail: 'Attempted: there is nothing to run from here. The review checkout cannot restart Couch, relaunch slots or inspect the live process table, and the issue Log says the check is pending a hub restart. Evidence that would answer it: ps output from the hosted slots over normal turns after relaunch, showing no wrapper-spawned pair slug and no claude -p slug calls.'
          family: live-fleet-acceptance
          round: 1
        - kind: defect
          id: BR-4
          severity: Minor
          title: No test covers COUCH_THREAD_TAG-only hosting, or PAIR_SLUG values other than exactly on
          detail: Both new tests only set COUCH_THREAD_SCOPE, and none tries a PAIR_SLUG value like ON or 1. Cover both when the shared helper is extracted.
          family: gate-truth-table-coverage
          round: 1
      recipe: small-diff-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#650 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-17T15:05:54-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `single-source-policy-predicate` (kind: defect) The slug gate rule is written out twice: wrap.go:754 and slugcmd.go:119
  ARCH-DRY. Both sites write out the same rule, CouchHostedEnv && PAIR_SLUG != "on". Those are the only two copies in this window. Move it into one helper, e.g. launcher.SlugEnabled(getenv), call it from both, and test the truth table once, including that values like ON or 1 keep it off (fail closed).
- **BR-2** [Important] `done-when-tracks-revisions` (kind: defect) Done when does not cover the slugcmd.Run Stop-hook gate the Revisions added
  Clause 1 only describes the wrapper's spawn. Clause 2, read literally ("no pair slug processes"), is wrong: under Couch the Stop hook still starts a short-lived pair-slug that stops at the gate. Change clause 1 to name both gates, and make clause 2 about model calls and wrapper spawns.
- **BR-3** [Important] `live-fleet-acceptance` (kind: unverifiable) Done-when clause 2 (live check across the fleet after restart) is not verified
  Attempted: there is nothing to run from here. The review checkout cannot restart Couch, relaunch slots or inspect the live process table, and the issue Log says the check is pending a hub restart. Evidence that would answer it: ps output from the hosted slots over normal turns after relaunch, showing no wrapper-spawned pair slug and no claude -p slug calls.
- **BR-4** [Minor] `gate-truth-table-coverage` (kind: defect) No test covers COUCH_THREAD_TAG-only hosting, or PAIR_SLUG values other than exactly on
  Both new tests only set COUCH_THREAD_SCOPE, and none tries a PAIR_SLUG value like ON or 1. Cover both when the shared helper is extracted.

## Open findings

- **BR-1** [Important] `single-source-policy-predicate` (kind: defect) The slug gate rule is written out twice: wrap.go:754 and slugcmd.go:119
- **BR-2** [Important] `done-when-tracks-revisions` (kind: defect) Done when does not cover the slugcmd.Run Stop-hook gate the Revisions added
- **BR-3** [Important] `live-fleet-acceptance` (kind: unverifiable) Done-when clause 2 (live check across the fleet after restart) is not verified
- **BR-4** [Minor] `gate-truth-table-coverage` (kind: defect) No test covers COUCH_THREAD_TAG-only hosting, or PAIR_SLUG values other than exactly on
