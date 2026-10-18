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
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 499163f6-78f7-41c8-8cb7-f720a540ef5b
            repository: github.com/xianxu/pair
            issue_id: "000650"
            recipe: small-diff-review
            round: 2
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/9d8ab46598aa95ec
            reviewed_head: 7b72b05c05b6484fd33174ba17f011e2bd8a80ec
            slot: pair:8
            requested_at: "2026-10-17T22:51:35.667177Z"
          phase: finished
          pid: 53269
          started_at: "2026-10-17T22:51:35.927526Z"
          finished_at: "2026-10-17T22:53:09.685656Z"
          session_id: 6d50ebd9-077f-455e-afe6-a71dc8c5cdc5
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 7b72b05c05b6484fd33174ba17f011e2bd8a80ec
        tracker_head: f42a42dc7c631684422c857636b3c13258f0926d
        branch_point: c0841e1082d39731781d8a2972eeeaed41d07ade
        since: "2026-10-17T12:13:29-07:00"
        until: "2026-10-17T15:09:32-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-17T12:13:29-07:00"
              end: "2026-10-17T15:09:32-07:00"
              began: "2026-10-17T12:13:29-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot8/pair
        sources:
            - harness: qoder
              id: 7febf963-ecac-409f-82f9-c1438842f2a5
              slot: pair:8
              cwd: /Users/xianxu/workspace/worktree/pair-slot8/pair
              digest: c30bdf8f237efccc6ce0ff50bf3e38e5206280272865f48503e4f9e23dfafbf2
              started: "2026-10-17T19:12:56.364Z"
              spawned: false
              bytes: 1698622
              windows:
                - start: "2026-10-17T12:13:29-07:00"
                  end: "2026-10-17T15:09:32-07:00"
        queue_waits:
            - attempt_id: 2a875dd8cd12fd8a37285519857d2012
              slot: pair:8
              record_digest: 98f72abd6cbc5612449e761dac86126812bf008d2529969a01518ef1b117bbf9
              start: "2026-10-17T19:24:30.020404Z"
              end: "2026-10-17T19:48:27.163529Z"
              queue_seconds: 1437.143125
              raw_seconds: 1437.143125
              exact: true
            - attempt_id: 4fbcbff31e39a1c3e56b389edc4f61f0
              slot: pair:8
              record_digest: 3b50aa92fecf6a2ef045dd35581f70536a30a2934d7d2bf108f7985c5cc76ad5
              start: "2026-10-17T20:00:51.830699Z"
              end: "2026-10-17T20:42:07.064748Z"
              queue_seconds: 2475.234049
              raw_seconds: 2475.234049
              exact: true
            - attempt_id: d87364194364dba748abc7b5399f9080
              slot: pair:8
              record_digest: 944a3e4fb71c13597bbd59a22ddd374905ae1003113a835fc1e4fe932fd567d5
              start: "2026-10-17T21:51:56.391712Z"
              end: "2026-10-17T21:51:56.554256Z"
              queue_seconds: 0.162544
              raw_seconds: 0.162544
              exact: true
        excluded_brain_sessions: 1
        excluded_brain_minutes: 51.22708333333333
        cuts:
            - dispatched: "2026-10-17T15:04:15-07:00"
              finalized: "2026-10-17T15:05:54-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-17T15:53:11-07:00"
      dispatched: "2026-10-17T15:51:32-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: launcher.SlugEnabled (run.go:38) is now the one copy of the rule; wrap.go:755 and slugcmd.go:119 both call it, no other copies under cmd/ (grep); TestSlugEnabledTruthTable passes.
          round: 2
        - id: BR-2
          disposition: addressed
          note: Done-when clause 1 now names both gates (wrapper spawn + slugcmd.Run Stop-hook body); clause 2 is now about wrapper spawns and claude -p calls, not every pair-slug process. Matches the code.
          round: 2
        - id: BR-3
          disposition: not-addressed
          note: 'No host evidence yet: no dispute filed, no ps output in the Log. The Log and Revisions still say it is pending the hub''s Couch restart and slot relaunch. Needed: process-table output from hosted slots over normal turns after relaunch, showing no wrapper-spawned pair slug and no slug claude -p.'
          round: 2
        - id: BR-4
          disposition: addressed
          note: TestSlugEnabledTruthTable covers tag-only hosting ({"", "t", ""} -> false, {"", "t", "on"} -> true) and the ON/1/0/off fail-closed values; both call sites go through this helper.
          round: 2
      recipe: small-diff-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: e133411a-bb88-49f9-8c0b-d5454504cb38
            repository: github.com/xianxu/pair
            issue_id: "000650"
            recipe: small-diff-review
            round: 3
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/489fb2a9ba852070
            reviewed_head: 7b72b05c05b6484fd33174ba17f011e2bd8a80ec
            slot: pair:8
            requested_at: "2026-10-17T22:56:27.801713Z"
          phase: finished
          pid: 97622
          started_at: "2026-10-17T22:56:27.971531Z"
          finished_at: "2026-10-17T22:57:15.12241Z"
          session_id: cffdc757-8673-48f8-8250-8650eba33eff
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 7b72b05c05b6484fd33174ba17f011e2bd8a80ec
        tracker_head: f42a42dc7c631684422c857636b3c13258f0926d
        branch_point: c0841e1082d39731781d8a2972eeeaed41d07ade
        since: "2026-10-17T12:13:29-07:00"
        until: "2026-10-17T15:09:32-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-17T12:13:29-07:00"
              end: "2026-10-17T15:09:32-07:00"
              began: "2026-10-17T12:13:29-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot8/pair
        sources:
            - harness: qoder
              id: 7febf963-ecac-409f-82f9-c1438842f2a5
              slot: pair:8
              cwd: /Users/xianxu/workspace/worktree/pair-slot8/pair
              digest: 889f17664d652a0917b9bfb37b3da6a3b92cb011dc8b75bae7ebc177ab73ef40
              started: "2026-10-17T19:12:56.364Z"
              spawned: false
              bytes: 1805340
              windows:
                - start: "2026-10-17T12:13:29-07:00"
                  end: "2026-10-17T15:09:32-07:00"
        queue_waits:
            - attempt_id: 2a875dd8cd12fd8a37285519857d2012
              slot: pair:8
              record_digest: 98f72abd6cbc5612449e761dac86126812bf008d2529969a01518ef1b117bbf9
              start: "2026-10-17T19:24:30.020404Z"
              end: "2026-10-17T19:48:27.163529Z"
              queue_seconds: 1437.143125
              raw_seconds: 1437.143125
              exact: true
            - attempt_id: 4fbcbff31e39a1c3e56b389edc4f61f0
              slot: pair:8
              record_digest: 3b50aa92fecf6a2ef045dd35581f70536a30a2934d7d2bf108f7985c5cc76ad5
              start: "2026-10-17T20:00:51.830699Z"
              end: "2026-10-17T20:42:07.064748Z"
              queue_seconds: 2475.234049
              raw_seconds: 2475.234049
              exact: true
            - attempt_id: d87364194364dba748abc7b5399f9080
              slot: pair:8
              record_digest: 944a3e4fb71c13597bbd59a22ddd374905ae1003113a835fc1e4fe932fd567d5
              start: "2026-10-17T21:51:56.391712Z"
              end: "2026-10-17T21:51:56.554256Z"
              queue_seconds: 0.162544
              raw_seconds: 0.162544
              exact: true
        excluded_brain_sessions: 1
        excluded_brain_minutes: 51.22708333333333
        cuts:
            - dispatched: "2026-10-17T15:04:15-07:00"
              finalized: "2026-10-17T15:05:54-07:00"
              finalizing: false
            - dispatched: "2026-10-17T15:51:32-07:00"
              finalized: "2026-10-17T15:53:11-07:00"
              finalizing: false
      "n": 3
      timestamp: "2026-10-17T15:57:16-07:00"
      dispatched: "2026-10-17T15:56:25-07:00"
      agent: claude
      dispose:
        - id: BR-3
          disposition: not-addressed
          note: 'Still unverifiable from a review checkout and no measured-evidence dispute filed; needs post-relaunch ps output across hosted slots showing no wrapper-spawned pair slug and no slug claude -p call. Code-side gating at both entries is verified (grep: dispatcher.go:282 is the only slugcmd.Run caller; wrap.go:781 spawn sits behind the gate).'
          round: 3
      recipe: small-diff-review
      isolation: fenced
      reviewed: 7b72b05c05b6484fd33174ba17f011e2bd8a80ec
      forced: '--no-ledger (or --force): RED: TestMaybeSpawnSlugOffByDefaultUnderCouch, TestRunSkipsUnderCouchHosting, TestSlugEnabledTruthTable fail on main code (no gate). GREEN: launcher, wrapcmd, slugcmd, dispatcher tests pass. Counterfactual: reverting the slugcmd.Run gate re-reds TestRunSkipsUnderCouchHosting (observed). Both entry points — pair-wrap turn-end spawn and slugcmd.Run via bin/pair-slug, including the Claude Stop hook — gate on launcher.SlugEnabled, one source. Full suite exit 0 on 7b72b05c0 (10m36s) after one load-flake re-run (run-headless-test clean-quit timeout under concurrent fleet suites; passes in isolation). --no-ledger: BR-3 (live-fleet-acceptance, kind: unverifiable) is waived because its evidence — ps across hosted slots after relaunch onto this fix — can only exist post-landing (merge, pair:0 rebuild, hub Couch restart, slot relaunch); close gates the merge, so waiting would deadlock. The live check remains assigned to this slot after the hub restart, recorded in the issue Revisions/Log and flagged in the landing report.'
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

## Round 2 — 2026-10-17T15:53:11-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — launcher.SlugEnabled (run.go:38) is now the one copy of the rule; wrap.go:755 and slugcmd.go:119 both call it, no other copies under cmd/ (grep); TestSlugEnabledTruthTable passes.
- BR-2 — addressed — Done-when clause 1 now names both gates (wrapper spawn + slugcmd.Run Stop-hook body); clause 2 is now about wrapper spawns and claude -p calls, not every pair-slug process. Matches the code.
- BR-3 — not-addressed — No host evidence yet: no dispute filed, no ps output in the Log. The Log and Revisions still say it is pending the hub's Couch restart and slot relaunch. Needed: process-table output from hosted slots over normal turns after relaunch, showing no wrapper-spawned pair slug and no slug claude -p.
- BR-4 — addressed — TestSlugEnabledTruthTable covers tag-only hosting ({"", "t", ""} -> false, {"", "t", "on"} -> true) and the ON/1/0/off fail-closed values; both call sites go through this helper.

## Round 3 — 2026-10-17T15:57:16-07:00 (claude) — BLOCKED

**Forced past** (`--force`): --no-ledger (or --force): RED: TestMaybeSpawnSlugOffByDefaultUnderCouch, TestRunSkipsUnderCouchHosting, TestSlugEnabledTruthTable fail on main code (no gate). GREEN: launcher, wrapcmd, slugcmd, dispatcher tests pass. Counterfactual: reverting the slugcmd.Run gate re-reds TestRunSkipsUnderCouchHosting (observed). Both entry points — pair-wrap turn-end spawn and slugcmd.Run via bin/pair-slug, including the Claude Stop hook — gate on launcher.SlugEnabled, one source. Full suite exit 0 on 7b72b05c0 (10m36s) after one load-flake re-run (run-headless-test clean-quit timeout under concurrent fleet suites; passes in isolation). --no-ledger: BR-3 (live-fleet-acceptance, kind: unverifiable) is waived because its evidence — ps across hosted slots after relaunch onto this fix — can only exist post-landing (merge, pair:0 rebuild, hub Couch restart, slot relaunch); close gates the merge, so waiting would deadlock. The live check remains assigned to this slot after the hub restart, recorded in the issue Revisions/Log and flagged in the landing report.

### Disposed

- BR-3 — not-addressed — Still unverifiable from a review checkout and no measured-evidence dispute filed; needs post-relaunch ps output across hosted slots showing no wrapper-spawned pair slug and no slug claude -p call. Code-side gating at both entries is verified (grep: dispatcher.go:282 is the only slugcmd.Run caller; wrap.go:781 spawn sits behind the gate).

## Open findings

- **BR-3** [Important] `live-fleet-acceptance` (kind: unverifiable) Done-when clause 2 (live check across the fleet after restart) is not verified
