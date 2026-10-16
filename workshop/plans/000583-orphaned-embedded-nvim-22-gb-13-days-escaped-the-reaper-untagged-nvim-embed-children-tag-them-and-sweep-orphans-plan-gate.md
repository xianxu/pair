---
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

## Open findings

- **PQ-1** [Important] `unverified-existing-behavior-claim` (kind: defect) Changelog kind identity is wrong: log is .md not .log, tag cannot be parsed from hyphenated changelog-<tag>-<agent>[-<session>], and one per-tag sidecar collides across per-agent viewers
- **PQ-2** [Minor] `unstated-decision-input` (kind: defect) How ReapNvim/ReapNvimContext set NvimReapRow.TagLive is unspecified; quit-path reap may skip 'owner live'
- **PQ-3** [Minor] `test-plan-enumerates-cases` (kind: defect) Tasks 2, 3, 12 and 16 enumerate test cases in prose instead of one strategy line per risky function
- **PQ-4** [Minor] `unstated-decision-input` (kind: defect) 'Assert pkill is never invoked' needs a subprocess seam that does not exist; say kills move onto ProcessTable.Signal
