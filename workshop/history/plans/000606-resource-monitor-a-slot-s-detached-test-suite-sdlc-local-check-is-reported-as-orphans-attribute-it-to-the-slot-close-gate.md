---
gate: boundary-review
issue: 606
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: b56e0004-b567-4068-a672-5f84f649ddc6
            repository: github.com/xianxu/pair
            issue_id: "000606"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/5990f7ff0c46cd0e
            reviewed_head: ee39876a52fc7fc8a51be8fdc89e2e5dd1dc8f84
            slot: pair:2
            requested_at: "2026-10-16T19:26:12.131288Z"
          phase: finished
          pid: 12291
          started_at: "2026-10-16T19:26:12.473072Z"
          finished_at: "2026-10-16T19:38:13.995731Z"
          session_id: 041a3a98-8a40-40bc-a228-68388e8172f1
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: ee39876a52fc7fc8a51be8fdc89e2e5dd1dc8f84
        tracker_head: be1e2db83eafe080efe6f8ef92aa72d7d7bfba78
        branch_point: b666bef0e642caecc1a5caa21169bb823b0d6807
        since: "2026-10-16T12:04:56-07:00"
        until: "2026-10-16T12:24:53-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-16T12:04:56-07:00"
              end: "2026-10-16T12:24:53-07:00"
              began: "2026-10-16T12:04:56-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 9ab7f264f3661b95e17277a5b5f5f5dc905b6800cac2ec74027882ca5bd0f665
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 13130785
              windows:
                - start: "2026-10-16T12:04:56-07:00"
                  end: "2026-10-16T12:24:53-07:00"
      "n": 1
      timestamp: "2026-10-16T12:38:17-07:00"
      dispatched: "2026-10-16T12:26:07-07:00"
      agent: claude
      findings:
        - kind: defect
          id: BR-1
          severity: Minor
          title: Remembered root death orphans a suite that a live detached root now claims
          detail: SelectResourceProcesses keeps previous.ResourceRoot; if the original agent root dies or is replaced while the backgrounded suite runs, rootDead flags it orphan despite DetachedSlotRoots claiming it (resource.go:263-265). Either cover/document as deliberate relaunch tradeoff or prefer a live detached candidate.
          family: claimed-process-orphaned-by-stale-root
          round: 1
        - kind: defect
          id: BR-2
          severity: Minor
          title: Environment read errors cached as no-thread for the generation's lifetime
          detail: resources.go:73-76 caches any err as ok=false forever; a transient failure (e.g. mid-exec read) permanently prevents attribution. Cache only clean reads or known-permanent errors.
          family: transient-error-cached-as-permanent
          round: 1
        - kind: defect
          id: BR-3
          severity: Minor
          title: Infrastructure name set duplicates ClassifyUnattributed's names set (ARCH-DRY)
          detail: resource_detached.go:35-38 and resource.go:312-315 both build zellij+AgentInventory; extract a shared helper so the two exclusion lists cannot drift.
          family: duplicated-process-kind-list
          round: 1
        - kind: defect
          id: BR-4
          severity: Minor
          title: Live Done-when (full suite raises no orphan alert) not explicitly recorded
          detail: The Log records a live probe and claims but not an observed full suite run with zero orphan alerts; add that observation.
          family: done-when-evidence-missing
          round: 1
      recipe: milestone-review
      isolation: fenced
      reviewed: ee39876a52fc7fc8a51be8fdc89e2e5dd1dc8f84
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 6f4c429b-e337-4ddf-89c4-c691ae4114ca
            repository: github.com/xianxu/pair
            issue_id: "000606"
            recipe: fix-review
            round: 2
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/ac17a4a6fd8dac02
            reviewed_head: 63babdc3e4dec229d2035224cc77000da456b7ea
            slot: pair:2
            requested_at: "2026-10-16T19:43:52.361711Z"
          phase: finished
          pid: 20696
          started_at: "2026-10-16T19:43:52.553601Z"
          finished_at: "2026-10-16T19:45:54.34329Z"
          session_id: 36dba369-6014-447a-8c60-0038cee1f06b
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 63babdc3e4dec229d2035224cc77000da456b7ea
        tracker_head: 9d2afc58d5f315486db535a573e5200bb72514c5
        branch_point: b666bef0e642caecc1a5caa21169bb823b0d6807
        since: "2026-10-16T12:04:56-07:00"
        until: "2026-10-16T12:41:52-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-16T12:04:56-07:00"
              end: "2026-10-16T12:41:52-07:00"
              began: "2026-10-16T12:04:56-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: eb8e6ad83cce603879e07112799a9fd38db5801d3a80e35628955190fffc9878
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 13414587
              windows:
                - start: "2026-10-16T12:04:56-07:00"
                  end: "2026-10-16T12:41:52-07:00"
        cuts:
            - dispatched: "2026-10-16T12:26:07-07:00"
              finalized: "2026-10-16T12:38:17-07:00"
              finalizing: true
      "n": 2
      timestamp: "2026-10-16T12:45:56-07:00"
      dispatched: "2026-10-16T12:43:49-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: resource.go:250-261 hands a dead or reused remembered root over to a live same-slot, same-thread candidate. TestDetachedSuiteOutlivesItsAgentsRelaunch fails with the fix reverted (Root PID 5, Orphan true).
          round: 2
        - id: BR-2
          disposition: not-addressed
          note: Only IdentityUnavailableError and ctx errors skip the cache now. Other environmentRead errors (resource.go:170-172, e.g. EINVAL from a mid-exec PROCARGS2 read) are still cached as permanent. The finding asked to cache only clean reads or known-permanent errors (EPERM/ESRCH). The partial fix has a test that goes red without it.
          round: 2
        - id: BR-3
          disposition: not-addressed
          note: The interdiff does not touch resource_detached.go:35-38 or resource.go:324-327. The two zellij+AgentInventory sets are still duplicated.
          round: 2
        - id: BR-4
          disposition: not-addressed
          note: The Log still says live no-orphan-alert acceptance "needs the running Couch on this build (TL after landing)". No full-suite observation is recorded.
          round: 2
      findings:
        - kind: defect
          id: BR-5
          severity: Minor
          title: Handover root-death check restates and diverges from rootDead (ARCH-DRY)
          detail: resource.go:253 treats a live root with an empty identity as dead and ignores duplicate[], while rootDead at resource.go:265 requires a non-empty identity and checks duplicate[]. So a live root whose identity read failed is still handed over. Compute the remembered root's death once and use it for both checks.
          family: duplicated-root-death-predicate
          round: 2
      recipe: fix-review
      isolation: fenced
      reviewed: 63babdc3e4dec229d2035224cc77000da456b7ea
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 8dd0fc99-3a71-42d0-b646-55d296dd890f
            repository: github.com/xianxu/pair
            issue_id: "000606"
            recipe: fix-review
            round: 3
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/461064ffcb106f8e
            reviewed_head: 3532f6d70fdb4d4af46f6fecdbbb7fe591e66664
            slot: pair:2
            requested_at: "2026-10-16T20:21:21.771899Z"
          phase: finished
          pid: 71028
          started_at: "2026-10-16T20:21:21.921543Z"
          finished_at: "2026-10-16T20:22:03.4849Z"
          session_id: 604b4848-8eca-429f-9a55-94b78d0fe224
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 3532f6d70fdb4d4af46f6fecdbbb7fe591e66664
        tracker_head: 25f657b955dbf6b142496a33c3afb644ff037fcb
        branch_point: b666bef0e642caecc1a5caa21169bb823b0d6807
        since: "2026-10-16T12:04:56-07:00"
        until: "2026-10-16T13:20:56-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-16T12:04:56-07:00"
              end: "2026-10-16T13:20:56-07:00"
              began: "2026-10-16T12:04:56-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 166ee04b0ebb8b2ca751353d033c8d069bfbbcb05b95984365cc8bb50a1f1f9f
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 13492315
              windows:
                - start: "2026-10-16T12:04:56-07:00"
                  end: "2026-10-16T13:20:56-07:00"
        cuts:
            - dispatched: "2026-10-16T12:26:07-07:00"
              finalized: "2026-10-16T12:38:17-07:00"
              finalizing: true
            - dispatched: "2026-10-16T12:43:49-07:00"
              finalized: "2026-10-16T12:45:56-07:00"
              finalizing: true
      "n": 3
      timestamp: "2026-10-16T13:22:04-07:00"
      dispatched: "2026-10-16T13:21:20-07:00"
      agent: claude
      dispose:
        - id: BR-2
          disposition: not-addressed
          note: The interdiff does not touch resources.go or resource.go. environmentRead errors other than identity-unavailable and context errors are still cached as permanent.
          round: 3
        - id: BR-3
          disposition: not-addressed
          note: The interdiff does not touch resource_detached.go:35-38 or resource.go. The zellij+AgentInventory name sets are still duplicated.
          round: 3
        - id: BR-4
          disposition: not-addressed
          note: The new issue Log line records only unit tests. There is still no observation of a live full suite run with zero orphan alerts.
          round: 3
        - id: BR-5
          disposition: not-addressed
          note: resource.go is unchanged in this window. The handover root-death check still restates rootDead and diverges from it.
          round: 3
      recipe: fix-review
      isolation: fenced
      reviewed: 3532f6d70fdb4d4af46f6fecdbbb7fe591e66664
      blocked: false
---

# Gate ledger — pair#606 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-16T12:38:17-07:00 (claude) — passed

### Raised

- **BR-1** [Minor] `claimed-process-orphaned-by-stale-root` (kind: defect) Remembered root death orphans a suite that a live detached root now claims
  SelectResourceProcesses keeps previous.ResourceRoot; if the original agent root dies or is replaced while the backgrounded suite runs, rootDead flags it orphan despite DetachedSlotRoots claiming it (resource.go:263-265). Either cover/document as deliberate relaunch tradeoff or prefer a live detached candidate.
- **BR-2** [Minor] `transient-error-cached-as-permanent` (kind: defect) Environment read errors cached as no-thread for the generation's lifetime
  resources.go:73-76 caches any err as ok=false forever; a transient failure (e.g. mid-exec read) permanently prevents attribution. Cache only clean reads or known-permanent errors.
- **BR-3** [Minor] `duplicated-process-kind-list` (kind: defect) Infrastructure name set duplicates ClassifyUnattributed's names set (ARCH-DRY)
  resource_detached.go:35-38 and resource.go:312-315 both build zellij+AgentInventory; extract a shared helper so the two exclusion lists cannot drift.
- **BR-4** [Minor] `done-when-evidence-missing` (kind: defect) Live Done-when (full suite raises no orphan alert) not explicitly recorded
  The Log records a live probe and claims but not an observed full suite run with zero orphan alerts; add that observation.

## Round 2 — 2026-10-16T12:45:56-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — resource.go:250-261 hands a dead or reused remembered root over to a live same-slot, same-thread candidate. TestDetachedSuiteOutlivesItsAgentsRelaunch fails with the fix reverted (Root PID 5, Orphan true).
- BR-2 — not-addressed — Only IdentityUnavailableError and ctx errors skip the cache now. Other environmentRead errors (resource.go:170-172, e.g. EINVAL from a mid-exec PROCARGS2 read) are still cached as permanent. The finding asked to cache only clean reads or known-permanent errors (EPERM/ESRCH). The partial fix has a test that goes red without it.
- BR-3 — not-addressed — The interdiff does not touch resource_detached.go:35-38 or resource.go:324-327. The two zellij+AgentInventory sets are still duplicated.
- BR-4 — not-addressed — The Log still says live no-orphan-alert acceptance "needs the running Couch on this build (TL after landing)". No full-suite observation is recorded.

### Raised

- **BR-5** [Minor] `duplicated-root-death-predicate` (kind: defect) Handover root-death check restates and diverges from rootDead (ARCH-DRY)
  resource.go:253 treats a live root with an empty identity as dead and ignores duplicate[], while rootDead at resource.go:265 requires a non-empty identity and checks duplicate[]. So a live root whose identity read failed is still handed over. Compute the remembered root's death once and use it for both checks.

## Round 3 — 2026-10-16T13:22:04-07:00 (claude) — passed

### Disposed

- BR-2 — not-addressed — The interdiff does not touch resources.go or resource.go. environmentRead errors other than identity-unavailable and context errors are still cached as permanent.
- BR-3 — not-addressed — The interdiff does not touch resource_detached.go:35-38 or resource.go. The zellij+AgentInventory name sets are still duplicated.
- BR-4 — not-addressed — The new issue Log line records only unit tests. There is still no observation of a live full suite run with zero orphan alerts.
- BR-5 — not-addressed — resource.go is unchanged in this window. The handover root-death check still restates rootDead and diverges from it.

## Open findings

- **BR-2** [Minor] `transient-error-cached-as-permanent` (kind: defect) Environment read errors cached as no-thread for the generation's lifetime
- **BR-3** [Minor] `duplicated-process-kind-list` (kind: defect) Infrastructure name set duplicates ClassifyUnattributed's names set (ARCH-DRY)
- **BR-4** [Minor] `done-when-evidence-missing` (kind: defect) Live Done-when (full suite raises no orphan alert) not explicitly recorded
- **BR-5** [Minor] `duplicated-root-death-predicate` (kind: defect) Handover root-death check restates and diverges from rootDead (ARCH-DRY)
