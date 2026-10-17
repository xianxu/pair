---
reviewer_runs:
    - version: 1
      intent:
        id: 67a8bc4e-525f-479c-869f-2af5bca693b2
        repository: github.com/xianxu/pair
        issue_id: "000583"
        recipe: estimate-quality
        round: 1
        attempt: 1
        agent: claude
        root: /Users/xianxu/workspace/worktree/pair-slot4/pair
        reviewed_head: 8793f9f46bccacfaa8017bebc9ad1cfb541f5f20
        slot: pair:4
        requested_at: "2026-10-16T18:01:02.621315Z"
      phase: finished
      pid: 27554
      started_at: "2026-10-16T18:01:03.158954Z"
      finished_at: "2026-10-16T18:01:28.173613Z"
      session_id: a845f300-2968-4cb8-b745-15e1c37f59df
      outcome: returned
gate: plan-quality
issue: 583
id_prefix: PQ
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            id: 46154c02-7ec5-4e94-b4f4-c3c8309078df
            repository: github.com/xianxu/pair
            issue_id: "000583"
            recipe: plan-quality
            round: 1
            attempt: 1
            agent: claude
            root: /Users/xianxu/workspace/worktree/pair-slot4/pair
            reviewed_head: caeaed71410757dd8c862aa0b01405841a42e2ce
            slot: pair:4
            requested_at: "2026-10-16T17:35:31.244483Z"
          phase: finished
          pid: 65340
          started_at: "2026-10-16T17:35:31.61639Z"
          finished_at: "2026-10-16T17:36:52.847178Z"
          session_id: 5c6faf1f-ecbe-4e42-8a3e-5f18673a4399
          outcome: returned
      "n": 1
      timestamp: "2026-10-16T10:36:53-07:00"
      agent: claude
      findings:
        - kind: defect
          id: PQ-1
          severity: Important
          title: 'Changelog kind identity is wrong: log is .md not .log, tag cannot be parsed from hyphenated changelog-<tag>-<agent>[-<session>], and one per-tag sidecar collides across per-agent viewers'
          detail: ChangelogArtifacts.Log = base + ".md" (artifactpath/paths.go:765); componentPattern allows '-' in tag/agent/session (paths.go:14) and session may be empty (paths.go:738-749), so a TagFromNvimEmbedArgv changelog marker is ambiguous; ExpectedArgPath for changelog is undefined. Key the sidecar per agent/session or match argv against each known tag's exact expected paths, and restate the FUNERAL bound.
          family: unverified-existing-behavior-claim
          round: 1
        - kind: defect
          id: PQ-2
          severity: Minor
          title: How ReapNvim/ReapNvimContext set NvimReapRow.TagLive is unspecified; quit-path reap may skip 'owner live'
          detail: At quit the TUI parent may still be alive; if TagLive derives from session liveness the explicit reap no-ops. State that explicit reap passes TagLive=false.
          family: unstated-decision-input
          round: 1
        - kind: defect
          id: PQ-3
          severity: Minor
          title: Tasks 2, 3, 12 and 16 enumerate test cases in prose instead of one strategy line per risky function
          family: test-plan-enumerates-cases
          round: 1
        - kind: defect
          id: PQ-4
          severity: Minor
          title: '''Assert pkill is never invoked'' needs a subprocess seam that does not exist; say kills move onto ProcessTable.Signal'
          family: unstated-decision-input
          round: 1
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            id: 09f2ddf2-aa7d-4371-9bcb-83be1a8b657d
            repository: github.com/xianxu/pair
            issue_id: "000583"
            recipe: plan-quality
            round: 2
            attempt: 1
            agent: claude
            root: /Users/xianxu/workspace/worktree/pair-slot4/pair
            reviewed_head: 07000595a710527eaf4f5b79b1106f1b3b8f053f
            slot: pair:4
            requested_at: "2026-10-16T17:52:15.172815Z"
          phase: finished
          pid: 19020
          started_at: "2026-10-16T17:52:15.390357Z"
          finished_at: "2026-10-16T17:52:49.656348Z"
          session_id: ea13117c-335f-4574-9a74-7f6e505c13e2
          outcome: returned
      "n": 2
      timestamp: "2026-10-16T10:52:50-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Artifact-keyed <base>.nvim-pid beside .openlock (opener/run.go:188 writes opener pid); cross-tag glob hazard measured and rejected; Funeral bound restated.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: TagLive renamed OwnerLive; explicit quit path passes false (Task 3 Step 3).
          round: 2
        - id: PQ-3
          disposition: addressed
          note: Test tasks now carry one adversarial-class strategy line each.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: Kills move onto ProcessTable.Signal; pkillF and lifecycle_os.go:53 pkill deleted, so no subprocess seam remains on the reap path.
          round: 2
      findings:
        - kind: defect
          id: PQ-5
          severity: Minor
          title: Sweep source 3 does not state the verdict when .nvim-pid exists but the .openlock sibling is gone
          detail: '3rd finding in this family. opener/run.go:189 removes .openlock on clean opener exit (defer), which is the root-cause case where the embed outlives its TUI; if the missing pair is read as skip, changelog orphans leak. Rule to apply across every evidence source: state the classifier verdict for absent, present and malformed. Here, an absent openlock means the owner is dead, and the kill stays protected by the identity, nvim name, --embed and exact <base>.md argv checks.'
          family: unstated-decision-input
          round: 2
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            id: 9c21ca93-dfb1-45ac-b1c2-7842b2c0f75b
            repository: github.com/xianxu/pair
            issue_id: "000583"
            recipe: plan-quality
            round: 3
            attempt: 1
            agent: claude
            root: /Users/xianxu/workspace/worktree/pair-slot4/pair
            reviewed_head: 8793f9f46bccacfaa8017bebc9ad1cfb541f5f20
            slot: pair:4
            requested_at: "2026-10-16T18:00:31.469615Z"
          phase: finished
          pid: 14750
          started_at: "2026-10-16T18:00:31.749491Z"
          finished_at: "2026-10-16T18:00:57.825578Z"
          session_id: 1ff324a8-f519-463e-91c6-eed2bb7295d3
          outcome: returned
      "n": 3
      timestamp: "2026-10-16T11:00:59-07:00"
      agent: claude
      dispose:
        - id: PQ-5
          disposition: addressed
          note: Task 5 source 3 now says absent openlock = owner dead = reap (matches the defer remove at opener/run.go:189), alive = skip, malformed = reap gated by row predicates; the rule is applied to every evidence source.
          round: 3
      blocked: false
content_hash: 71ab2c0ebb352010a53c4d18f09cb9623ef2c4356f8894049166fe3698281532
---

# Gate ledger — pair#583 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-16T10:36:53-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `unverified-existing-behavior-claim` (kind: defect) Changelog kind identity is wrong: log is .md not .log, tag cannot be parsed from hyphenated changelog-<tag>-<agent>[-<session>], and one per-tag sidecar collides across per-agent viewers
  ChangelogArtifacts.Log = base + ".md" (artifactpath/paths.go:765); componentPattern allows '-' in tag/agent/session (paths.go:14) and session may be empty (paths.go:738-749), so a TagFromNvimEmbedArgv changelog marker is ambiguous; ExpectedArgPath for changelog is undefined. Key the sidecar per agent/session or match argv against each known tag's exact expected paths, and restate the FUNERAL bound.
- **PQ-2** [Minor] `unstated-decision-input` (kind: defect) How ReapNvim/ReapNvimContext set NvimReapRow.TagLive is unspecified; quit-path reap may skip 'owner live'
  At quit the TUI parent may still be alive; if TagLive derives from session liveness the explicit reap no-ops. State that explicit reap passes TagLive=false.
- **PQ-3** [Minor] `test-plan-enumerates-cases` (kind: defect) Tasks 2, 3, 12 and 16 enumerate test cases in prose instead of one strategy line per risky function
- **PQ-4** [Minor] `unstated-decision-input` (kind: defect) 'Assert pkill is never invoked' needs a subprocess seam that does not exist; say kills move onto ProcessTable.Signal

## Round 2 — 2026-10-16T10:52:50-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — Artifact-keyed <base>.nvim-pid beside .openlock (opener/run.go:188 writes opener pid); cross-tag glob hazard measured and rejected; Funeral bound restated.
- PQ-2 — addressed — TagLive renamed OwnerLive; explicit quit path passes false (Task 3 Step 3).
- PQ-3 — addressed — Test tasks now carry one adversarial-class strategy line each.
- PQ-4 — addressed — Kills move onto ProcessTable.Signal; pkillF and lifecycle_os.go:53 pkill deleted, so no subprocess seam remains on the reap path.

### Raised

- **PQ-5** [Minor] `unstated-decision-input` (kind: defect) Sweep source 3 does not state the verdict when .nvim-pid exists but the .openlock sibling is gone
  3rd finding in this family. opener/run.go:189 removes .openlock on clean opener exit (defer), which is the root-cause case where the embed outlives its TUI; if the missing pair is read as skip, changelog orphans leak. Rule to apply across every evidence source: state the classifier verdict for absent, present and malformed. Here, an absent openlock means the owner is dead, and the kill stays protected by the identity, nvim name, --embed and exact <base>.md argv checks.

## Round 3 — 2026-10-16T11:00:59-07:00 (claude) — passed

### Disposed

- PQ-5 — addressed — Task 5 source 3 now says absent openlock = owner dead = reap (matches the defer remove at opener/run.go:189), alive = skip, malformed = reap gated by row predicates; the rule is applied to every evidence source.

## Open findings

(none — every finding has been disposed)
