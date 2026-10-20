---
reviewer_runs:
    - version: 1
      intent:
        id: bb11a21e-bcb7-435f-8d81-079d2c51b80a
        repository: github.com/xianxu/pair
        issue_id: "000717"
        recipe: estimate-quality
        round: 1
        attempt: 1
        agent: claude
        root: /Users/xianxu/workspace/worktree/pair-slot2/pair
        reviewed_head: 67d22553c29f4f8af9f3d880fffd447d86a6593b
        slot: pair:2
        requested_at: "2026-10-20T08:20:13.361356Z"
      phase: finished
      pid: 78729
      started_at: "2026-10-20T08:20:13.44781Z"
      finished_at: "2026-10-20T08:20:35.888743Z"
      session_id: 4e1ec141-8d89-49f0-8665-b8191ffad93e
      outcome: returned
gate: plan-quality
issue: 717
id_prefix: PQ
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            id: 4c85a413-eef9-4636-b60e-e19c5b2d91af
            repository: github.com/xianxu/pair
            issue_id: "000717"
            recipe: plan-quality
            round: 1
            attempt: 1
            agent: claude
            root: /Users/xianxu/workspace/worktree/pair-slot2/pair
            reviewed_head: e89c8ff901f2e0691b8dd3de90b29bc3489aefc6
            slot: pair:2
            requested_at: "2026-10-20T08:15:38.895017Z"
          phase: finished
          pid: 12354
          started_at: "2026-10-20T08:15:39.074629Z"
          finished_at: "2026-10-20T08:16:47.617377Z"
          session_id: ab6148dc-2ee1-41ab-85ce-8be4be326795
          outcome: returned
      "n": 1
      timestamp: "2026-10-20T01:16:47-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      findings:
        - kind: defect
          id: PQ-1
          severity: Important
          title: Admits starts with an unconditional a.Idle clause (recipe.go:86); adding an interrupt clause still admits an interrupt on an idle composer
          detail: The plan's claim that interrupt requests are admitted by nothing else is false if Admits only gains a clause. Make the interrupt check come first and exclusive (if r.InterruptOnly() return a.Interruptible) so Ctrl+C never reaches an idle agent, and name the matching widening of the RecoveryOnly-only pre-write recheck at wrapcmd/agent_drive.go:316.
          family: admission-rule-exclusivity
          round: 1
        - kind: defect
          id: PQ-2
          severity: Minor
          title: Generated ctrl-LETTER rows duplicate existing keys (ctrl-i=tab, ctrl-m=enter, ctrl-u=clear-line) and include flow-control ctrl-s and ctrl-q
          detail: ctrl-u would send the clear-line bytes without the recovery-only exemption. Exclude these rows with reasons, as for c, d and z, or document them as deliberate aliases.
          family: key-table-alias-semantics
          round: 1
        - kind: defect
          id: PQ-3
          severity: Minor
          title: 'The Log''s open question (arbitrary special characters in text: steps, no file form) is neither folded in nor stated as a non-goal'
          family: unstated-non-goals
          round: 1
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            id: b25b75b7-0fd4-4f24-b567-6cdcffab9943
            repository: github.com/xianxu/pair
            issue_id: "000717"
            recipe: plan-quality
            round: 2
            attempt: 1
            agent: claude
            root: /Users/xianxu/workspace/worktree/pair-slot2/pair
            reviewed_head: 27637a86be1ec8c6187f8dfa93d53dd240b66dbf
            slot: pair:2
            requested_at: "2026-10-20T08:17:18.199373Z"
          phase: finished
          pid: 5184
          started_at: "2026-10-20T08:17:18.405221Z"
          finished_at: "2026-10-20T08:17:57.903501Z"
          session_id: b6360913-e03e-428d-8115-fbe8d1d46bde
          outcome: returned
      "n": 2
      timestamp: "2026-10-20T01:17:58-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Admits now checks InterruptOnly first and exclusively; the recheck at agent_drive.go:316 is named.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: ctrl-i, ctrl-m, ctrl-u, ctrl-s and ctrl-q are excluded with reasons.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: Text-step special characters and a --keys file form are stated non-goals with reasons.
          round: 2
      findings:
        - kind: defect
          id: PQ-4
          severity: Important
          title: Widening the per-write recheck (agent_drive.go:316) to all non-model requests breaks key drives; Interruptible is not reconciled with the gates at :262 and :282
          detail: 'This is the 2nd finding in family admission-rule-exclusivity. The recheck runs on every EffectWrite and builds Admission without Idle. Applied to keys, it cancels the first write, or the second once the composer holds text. The interrupt also cannot reach a write: :282 returns on TurnActive unless the state is a dismissal, and :262 common requires no in-flight message and the input lease, which fails for a hub-sent turn. The rule to state in the plan, rather than patching this site: each request shape defines its admitting proof once. The start admission, the turn-active gate and the pre-write recheck all derive from it, and the recheck covers only the write that proof was taken for. For an interrupt that is its single Ctrl+C. The plan must name which parts of common and the lease Interruptible keeps.'
          family: admission-rule-exclusivity
          round: 2
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            id: a67787d8-36dc-4591-b6b0-14ce2e0d8a1c
            repository: github.com/xianxu/pair
            issue_id: "000717"
            recipe: plan-quality
            round: 3
            attempt: 1
            agent: claude
            root: /Users/xianxu/workspace/worktree/pair-slot2/pair
            reviewed_head: 67d22553c29f4f8af9f3d880fffd447d86a6593b
            slot: pair:2
            requested_at: "2026-10-20T08:18:53.179042Z"
          phase: finished
          pid: 67074
          started_at: "2026-10-20T08:18:53.270197Z"
          finished_at: "2026-10-20T08:19:16.859211Z"
          session_id: 5935e750-db21-494a-ae19-7f77ce8a39fc
          outcome: returned
      "n": 3
      timestamp: "2026-10-20T01:19:17-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      dispose:
        - id: PQ-4
          disposition: addressed
          note: The plan adds a shapeProof per request shape. Admits, the turn gate (TurnTolerant) and WriteAdmits all derive from it, and the recheck is limited to the recovery and interrupt shapes. Interruptible names which parts of common and the lease it keeps; verified against agent_drive.go:259-262 and couchmessage/model.go:108.
          round: 3
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            id: b262e5b8-cff8-49a7-b298-cdbb7ce85a19
            repository: github.com/xianxu/pair
            issue_id: "000717"
            recipe: plan-quality
            round: 4
            attempt: 1
            agent: claude
            root: /Users/xianxu/workspace/worktree/pair-slot2/pair
            reviewed_head: 67d22553c29f4f8af9f3d880fffd447d86a6593b
            slot: pair:2
            requested_at: "2026-10-20T08:19:58.011211Z"
          phase: finished
          pid: 77688
          started_at: "2026-10-20T08:19:58.107822Z"
          finished_at: "2026-10-20T08:20:11.632505Z"
          session_id: 7395260c-761a-4990-8b0d-47a6a2d6d972
          outcome: returned
      "n": 4
      timestamp: "2026-10-20T01:20:11-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      blocked: false
      protocol_error: no valid findings block
content_hash: 16fef7699fefa244272a81cc3e97e310536f2ae97eef5bafb83fa1db14dddaf1
---

# Gate ledger — pair#717 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-20T01:16:47-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `admission-rule-exclusivity` (kind: defect) Admits starts with an unconditional a.Idle clause (recipe.go:86); adding an interrupt clause still admits an interrupt on an idle composer
  The plan's claim that interrupt requests are admitted by nothing else is false if Admits only gains a clause. Make the interrupt check come first and exclusive (if r.InterruptOnly() return a.Interruptible) so Ctrl+C never reaches an idle agent, and name the matching widening of the RecoveryOnly-only pre-write recheck at wrapcmd/agent_drive.go:316.
- **PQ-2** [Minor] `key-table-alias-semantics` (kind: defect) Generated ctrl-LETTER rows duplicate existing keys (ctrl-i=tab, ctrl-m=enter, ctrl-u=clear-line) and include flow-control ctrl-s and ctrl-q
  ctrl-u would send the clear-line bytes without the recovery-only exemption. Exclude these rows with reasons, as for c, d and z, or document them as deliberate aliases.
- **PQ-3** [Minor] `unstated-non-goals` (kind: defect) The Log's open question (arbitrary special characters in text: steps, no file form) is neither folded in nor stated as a non-goal

## Round 2 — 2026-10-20T01:17:58-07:00 (claude) — BLOCKED

### Disposed

- PQ-1 — addressed — Admits now checks InterruptOnly first and exclusively; the recheck at agent_drive.go:316 is named.
- PQ-2 — addressed — ctrl-i, ctrl-m, ctrl-u, ctrl-s and ctrl-q are excluded with reasons.
- PQ-3 — addressed — Text-step special characters and a --keys file form are stated non-goals with reasons.

### Raised

- **PQ-4** [Important] `admission-rule-exclusivity` (kind: defect) Widening the per-write recheck (agent_drive.go:316) to all non-model requests breaks key drives; Interruptible is not reconciled with the gates at :262 and :282
  This is the 2nd finding in family admission-rule-exclusivity. The recheck runs on every EffectWrite and builds Admission without Idle. Applied to keys, it cancels the first write, or the second once the composer holds text. The interrupt also cannot reach a write: :282 returns on TurnActive unless the state is a dismissal, and :262 common requires no in-flight message and the input lease, which fails for a hub-sent turn. The rule to state in the plan, rather than patching this site: each request shape defines its admitting proof once. The start admission, the turn-active gate and the pre-write recheck all derive from it, and the recheck covers only the write that proof was taken for. For an interrupt that is its single Ctrl+C. The plan must name which parts of common and the lease Interruptible keeps.

## Round 3 — 2026-10-20T01:19:17-07:00 (claude) — passed

### Disposed

- PQ-4 — addressed — The plan adds a shapeProof per request shape. Admits, the turn gate (TurnTolerant) and WriteAdmits all derive from it, and the recheck is limited to the recovery and interrupt shapes. Interruptible names which parts of common and the lease it keeps; verified against agent_drive.go:259-262 and couchmessage/model.go:108.

## Round 4 — 2026-10-20T01:20:11-07:00 (claude) — passed

**Protocol error:** no valid findings block — this round contributed no findings.

## Open findings

(none — every finding has been disposed)
