---
reviewer_runs:
    - version: 1
      intent:
        id: 243149e9-5dba-48f9-ad91-a644130aabd7
        repository: github.com/xianxu/pair
        issue_id: "000847"
        recipe: estimate-quality
        round: 1
        attempt: 1
        agent: claude
        root: /Users/xianxu/workspace/worktree/pair-slot6/pair
        reviewed_head: 54082b2a126951cf2d5b10f9af7578924f9fa010
        slot: pair:6
        requested_at: "2026-10-25T10:22:08.800046Z"
      phase: finished
      pid: 67785
      started_at: "2026-10-25T10:22:09.011905Z"
      finished_at: "2026-10-25T10:22:44.972113Z"
      session_id: 721080b8-9725-4b1c-b969-a9bc998e7853
      outcome: returned
gate: plan-quality
issue: 847
id_prefix: PQ
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            id: 562c5e24-6d88-4a8f-ad7d-15e9f52826e0
            repository: github.com/xianxu/pair
            issue_id: "000847"
            recipe: plan-quality
            round: 1
            attempt: 1
            agent: claude
            root: /Users/xianxu/workspace/worktree/pair-slot6/pair
            reviewed_head: 13269fb16e7a9ff06718ff82e002159c4bc8b315
            slot: pair:6
            requested_at: "2026-10-25T10:15:23.301906Z"
          phase: finished
          pid: 17430
          started_at: "2026-10-25T10:15:23.493562Z"
          finished_at: "2026-10-25T10:17:30.937981Z"
          session_id: 7ccaca5b-23f5-4421-9574-9989dbf55bae
          outcome: returned
      "n": 1
      timestamp: "2026-10-25T03:17:31-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      findings:
        - kind: defect
          id: PQ-1
          severity: Important
          title: Strict bench-retire relies on Teardown LeftRunning, which is empty once the agent is dead, so retry-after-refusal and prior-generation orphans pass as proof of no survivors
          detail: teardown computes leftRunning only while the agent is listed or its lease is held (couchcore/teardown.go:188-198; leftRunning returns nil when the agent PID is absent, :505-507). After a refused retire has stopped the agent, every retry archives regardless of the survivor, and orphans from earlier bench-start generations live only in orphans.json (:238). RetireBenchSlot must also consult LeftRunningRegistry for this slot's registered orphans with a liveness check. The plan should state what state a refused retire leaves (session stopped, record kept) and test retry with the survivor still alive.
          family: survivor-proof-scoped-to-live-agent
          round: 1
        - kind: defect
          id: PQ-2
          severity: Important
          title: Plan contradicts itself on whether bench-start writes the per-repository saved launch parameters, which production slots of the same repo share
          detail: Saved parameters are keyed by repository identity (couchcore/switchagent.go:335-349, ResolveRepoArguments at repopreferences.go:166), so fixture:2 (bench) and fixture:3 (production) share them. Task 2's test says bench-start replaces the saved repo parameters, while its implementation step says skip the lookup. State that bench-start never writes RepoPreferences, and assert that in the native test.
          family: bench-effects-reach-production-state
          round: 1
        - kind: defect
          id: PQ-3
          severity: Minor
          title: Tasks 1, 3, 4 and 6 enumerate test cases in prose; compress to one strategy line per risky function
          detail: 'E.g. Provisions: table over ownership source enum x caller role, derived from the enum. Also the named hub_ownership_test.go does not exist; the existing test is hub_ownership_sequence_test.go.'
          family: plan-enumerates-test-cases
          round: 1
        - kind: defect
          id: PQ-4
          severity: Minor
          title: ReceiptProvenance.Generation source is given as launch nonce OR Record.Identity; pick one
          family: underspecified-receipt-field
          round: 1
        - kind: defect
          id: PQ-5
          severity: Minor
          title: State that a whole-repo .hub claim grants Provisions over every slot in that repo not pointed elsewhere by an exact claim (accepted, same reach as authorizeHubMutation)
          family: missing-non-goals
          round: 1
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            id: 186d393d-bd82-4222-bad7-cc3ffb81cd7e
            repository: github.com/xianxu/pair
            issue_id: "000847"
            recipe: plan-quality
            round: 2
            attempt: 1
            agent: claude
            root: /Users/xianxu/workspace/worktree/pair-slot6/pair
            reviewed_head: c75c1c104ebdb01354b71efdb70dd9ade06dc516
            slot: pair:6
            requested_at: "2026-10-25T10:18:30.894558Z"
          phase: finished
          pid: 61633
          started_at: "2026-10-25T10:18:31.04566Z"
          finished_at: "2026-10-25T10:19:08.769734Z"
          session_id: 6ba91ec2-f28e-46ca-baa2-a7ca52e00a06
          outcome: returned
      "n": 2
      timestamp: "2026-10-25T03:19:09-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Survivors = proof.LeftRunning plus live registry entries for any of this slot's threads; refused-retire state is stated; retry-with-survivor is tested.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: Stated as a decision, backed by repopreferences_adoption.go:22-25, and asserted byte-identical in Tasks 2 and 6.
          round: 2
        - id: PQ-3
          disposition: not-addressed
          note: Tasks 1/3/4 are compressed, but Tasks 2 and 5 still list cases and Task 1 still names the nonexistent hub_ownership_test.go. Minor, carried forward.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: Generation = StartEvent.Nonce only.
          round: 2
        - id: PQ-5
          disposition: addressed
          round: 2
      findings:
        - kind: defect
          id: PQ-6
          severity: Minor
          title: Strict retire's liveness rule does not say how to treat registry orphans with unknown process identity
          detail: 'This is the 2nd finding in this family, but it is not a fresh instance: it is the remaining edge of the rule PQ-1 fixed (survivors = every registered orphan of the slot that liveness cannot rule out). LeftRunningOrphan carries processUnknown (orphan_registry.go:143). The rule should say an orphan of unknown identity whose PID is alive counts as a survivor, so the proof errs toward refusing.'
          family: survivor-proof-scoped-to-live-agent
          round: 2
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            id: 16a09dbb-b2fd-4e76-86d3-b1984c667bb1
            repository: github.com/xianxu/pair
            issue_id: "000847"
            recipe: plan-quality
            round: 3
            attempt: 1
            agent: claude
            root: /Users/xianxu/workspace/worktree/pair-slot6/pair
            reviewed_head: b130e8d0e608ebd5ae2c9a437b295c27d74c434e
            slot: pair:6
            requested_at: "2026-10-25T10:19:41.261161Z"
          phase: finished
          pid: 7713
          started_at: "2026-10-25T10:19:41.458685Z"
          finished_at: "2026-10-25T10:20:13.076838Z"
          session_id: bb8392e9-7056-406b-b9e9-4a6084504c0c
          outcome: returned
      "n": 3
      timestamp: "2026-10-25T03:20:13-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      dispose:
        - id: PQ-3
          disposition: not-addressed
          note: 'Tasks 1/3/4 compressed; Task 2 and Task 5 still enumerate cases in prose, and Task 1 still cites nonexistent hub_ownership_test.go (only hub_ownership_sequence_test.go exists). Rule: every test step is one strategy line, and existing tests are cited by real path. Minor; carry to close.'
          round: 3
        - id: PQ-6
          disposition: addressed
          note: Strict retirement now counts an orphan with unknown identity and a live PID as a survivor (matches processUnknown, orphan_registry.go:143).
          round: 3
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            id: 13a8617f-b84d-4e0a-84e1-809b2902e471
            repository: github.com/xianxu/pair
            issue_id: "000847"
            recipe: plan-quality
            round: 4
            attempt: 1
            agent: claude
            root: /Users/xianxu/workspace/worktree/pair-slot6/pair
            reviewed_head: 54082b2a126951cf2d5b10f9af7578924f9fa010
            slot: pair:6
            requested_at: "2026-10-25T10:21:37.699537Z"
          phase: finished
          pid: 59528
          started_at: "2026-10-25T10:21:37.86483Z"
          finished_at: "2026-10-25T10:22:02.502533Z"
          session_id: 9a8dd32f-3414-4835-82f4-65ff74399740
          outcome: returned
      "n": 4
      timestamp: "2026-10-25T03:22:03-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      dispose:
        - id: PQ-3
          disposition: not-addressed
          note: 'Tasks 1/3/4 compressed to strategy lines; residual: Task 1 step still names nonexistent hub_ownership_test.go (real: hub_ownership_sequence_test.go), Tasks 2 and 5 still enumerate parser cases in prose. Minor, carried to close.'
          round: 4
      blocked: false
content_hash: f8c82791223df3d419c955bb23ba3534e41442ea608bf178783b5346ca6c9fcf
---

# Gate ledger — pair#847 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-25T03:17:31-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `survivor-proof-scoped-to-live-agent` (kind: defect) Strict bench-retire relies on Teardown LeftRunning, which is empty once the agent is dead, so retry-after-refusal and prior-generation orphans pass as proof of no survivors
  teardown computes leftRunning only while the agent is listed or its lease is held (couchcore/teardown.go:188-198; leftRunning returns nil when the agent PID is absent, :505-507). After a refused retire has stopped the agent, every retry archives regardless of the survivor, and orphans from earlier bench-start generations live only in orphans.json (:238). RetireBenchSlot must also consult LeftRunningRegistry for this slot's registered orphans with a liveness check. The plan should state what state a refused retire leaves (session stopped, record kept) and test retry with the survivor still alive.
- **PQ-2** [Important] `bench-effects-reach-production-state` (kind: defect) Plan contradicts itself on whether bench-start writes the per-repository saved launch parameters, which production slots of the same repo share
  Saved parameters are keyed by repository identity (couchcore/switchagent.go:335-349, ResolveRepoArguments at repopreferences.go:166), so fixture:2 (bench) and fixture:3 (production) share them. Task 2's test says bench-start replaces the saved repo parameters, while its implementation step says skip the lookup. State that bench-start never writes RepoPreferences, and assert that in the native test.
- **PQ-3** [Minor] `plan-enumerates-test-cases` (kind: defect) Tasks 1, 3, 4 and 6 enumerate test cases in prose; compress to one strategy line per risky function
  E.g. Provisions: table over ownership source enum x caller role, derived from the enum. Also the named hub_ownership_test.go does not exist; the existing test is hub_ownership_sequence_test.go.
- **PQ-4** [Minor] `underspecified-receipt-field` (kind: defect) ReceiptProvenance.Generation source is given as launch nonce OR Record.Identity; pick one
- **PQ-5** [Minor] `missing-non-goals` (kind: defect) State that a whole-repo .hub claim grants Provisions over every slot in that repo not pointed elsewhere by an exact claim (accepted, same reach as authorizeHubMutation)

## Round 2 — 2026-10-25T03:19:09-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — Survivors = proof.LeftRunning plus live registry entries for any of this slot's threads; refused-retire state is stated; retry-with-survivor is tested.
- PQ-2 — addressed — Stated as a decision, backed by repopreferences_adoption.go:22-25, and asserted byte-identical in Tasks 2 and 6.
- PQ-3 — not-addressed — Tasks 1/3/4 are compressed, but Tasks 2 and 5 still list cases and Task 1 still names the nonexistent hub_ownership_test.go. Minor, carried forward.
- PQ-4 — addressed — Generation = StartEvent.Nonce only.
- PQ-5 — addressed

### Raised

- **PQ-6** [Minor] `survivor-proof-scoped-to-live-agent` (kind: defect) Strict retire's liveness rule does not say how to treat registry orphans with unknown process identity
  This is the 2nd finding in this family, but it is not a fresh instance: it is the remaining edge of the rule PQ-1 fixed (survivors = every registered orphan of the slot that liveness cannot rule out). LeftRunningOrphan carries processUnknown (orphan_registry.go:143). The rule should say an orphan of unknown identity whose PID is alive counts as a survivor, so the proof errs toward refusing.

## Round 3 — 2026-10-25T03:20:13-07:00 (claude) — passed

### Disposed

- PQ-3 — not-addressed — Tasks 1/3/4 compressed; Task 2 and Task 5 still enumerate cases in prose, and Task 1 still cites nonexistent hub_ownership_test.go (only hub_ownership_sequence_test.go exists). Rule: every test step is one strategy line, and existing tests are cited by real path. Minor; carry to close.
- PQ-6 — addressed — Strict retirement now counts an orphan with unknown identity and a live PID as a survivor (matches processUnknown, orphan_registry.go:143).

## Round 4 — 2026-10-25T03:22:03-07:00 (claude) — passed

### Disposed

- PQ-3 — not-addressed — Tasks 1/3/4 compressed to strategy lines; residual: Task 1 step still names nonexistent hub_ownership_test.go (real: hub_ownership_sequence_test.go), Tasks 2 and 5 still enumerate parser cases in prose. Minor, carried to close.

## Open findings

- **PQ-3** [Minor] `plan-enumerates-test-cases` (kind: defect) Tasks 1, 3, 4 and 6 enumerate test cases in prose; compress to one strategy line per risky function
