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

## Open findings

- **BR-1** [Minor] `claimed-process-orphaned-by-stale-root` (kind: defect) Remembered root death orphans a suite that a live detached root now claims
- **BR-2** [Minor] `transient-error-cached-as-permanent` (kind: defect) Environment read errors cached as no-thread for the generation's lifetime
- **BR-3** [Minor] `duplicated-process-kind-list` (kind: defect) Infrastructure name set duplicates ClassifyUnattributed's names set (ARCH-DRY)
- **BR-4** [Minor] `done-when-evidence-missing` (kind: defect) Live Done-when (full suite raises no orphan alert) not explicitly recorded
