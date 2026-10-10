---
gate: boundary-review
issue: 429
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-10T14:17:17-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: Fast-path peek JSON omits working_path that the typed peek returns
          detail: peek_fast.go builds PeekResult without WorkingPath, but PeekThread fills it, so the couch --peek --json shape depends on whether a Couch is running. Carry it in TailThread, or drop or document it, and add a test that both paths give the same JSON keys.
          family: fast-path-output-parity
          round: 1
        - id: BR-2
          severity: Minor
          title: Plan item 4 says the recording fallback resolves when the record has no agent; the code never resolves
          detail: The behavior is correct, since Resolve would only return RecordAgent again. Add a Revisions entry.
          family: plan-prose-matches-code
          round: 1
        - id: BR-3
          severity: Minor
          title: fastPeek re-parses --lines and --json from inv.args instead of reusing the operation's arg parsing
          family: single-arg-parser
          round: 1
        - id: BR-4
          severity: Minor
          title: Fast path accepts alias or prefix slot refs that the typed fallback may resolve differently
          detail: canonicalTarget accepts pa:1, while ResolveThreadReference may not, so the same ref can succeed only when a Couch is running.
          family: fast-path-output-parity
          round: 1
        - id: BR-5
          severity: Minor
          title: No test for the fallback when an older Couch refuses a by-slot tail request
          family: test-covers-fallback-branch
          round: 1
      recipe: milestone-review
      blocked: true
    - "n": 2
      timestamp: "2026-10-10T14:21:22-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: TailThread carries WorkingPath; TestFastPeekMatchesTheTypedPeek fails (missing working_path) when message_service.go:130 is removed — mutation-verified.
          round: 2
        - id: BR-2
          disposition: addressed
          note: The issue's Revisions "close round 1 findings" records that the fallback never calls the resolver; the code (peek.go:185-197) matches.
          round: 2
        - id: BR-3
          disposition: withdrawn
          note: The flags arrive normalized from ParseCLI, and any flag fastPeek does not recognize falls back to the typed path, so a mismatch can only cost speed, never give a wrong answer.
          round: 2
        - id: BR-4
          disposition: withdrawn
          note: 'Accepted on purpose and recorded in Revisions: the fast path resolves slots the way --send-to does, and the fallback names any reference it cannot resolve in unavailable.'
          round: 2
        - id: BR-5
          disposition: addressed
          note: The "an older couch" case in TestFastPeekAnswersOnlyWhenEverySlotIsLive answers invalid-request and checks the fallback runs with nothing written.
          round: 2
      findings:
        - id: BR-6
          severity: Minor
          title: The plan and atlas still describe TailThread and the agent's source as they were before BR-1
          detail: '2nd finding in this family. The plan''s Design item 1 and atlas/couch.md say TailThread is {slot, tag, agent} (no working_path, and the agent is not said to come from the record). Plan item 4 says "LatestLaunchProfile, else the first incarnation", but RecordAgent prefers the sole incarnation, then the latest launch. Rule: when a fix changes a surface that prose describes, grep every restatement (plan Design, atlas, SKILL) in the same commit, and put a Revisions entry in the durable plan file, not only in the issue.'
          family: plan-prose-matches-code
          round: 2
      recipe: milestone-review
      reviewed: 4cc7d076da3196e3757340c1753029687abc6650
      blocked: false
---

# Gate ledger — pair#429 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-10T14:17:17-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `fast-path-output-parity` Fast-path peek JSON omits working_path that the typed peek returns
  peek_fast.go builds PeekResult without WorkingPath, but PeekThread fills it, so the couch --peek --json shape depends on whether a Couch is running. Carry it in TailThread, or drop or document it, and add a test that both paths give the same JSON keys.
- **BR-2** [Minor] `plan-prose-matches-code` Plan item 4 says the recording fallback resolves when the record has no agent; the code never resolves
  The behavior is correct, since Resolve would only return RecordAgent again. Add a Revisions entry.
- **BR-3** [Minor] `single-arg-parser` fastPeek re-parses --lines and --json from inv.args instead of reusing the operation's arg parsing
- **BR-4** [Minor] `fast-path-output-parity` Fast path accepts alias or prefix slot refs that the typed fallback may resolve differently
  canonicalTarget accepts pa:1, while ResolveThreadReference may not, so the same ref can succeed only when a Couch is running.
- **BR-5** [Minor] `test-covers-fallback-branch` No test for the fallback when an older Couch refuses a by-slot tail request

## Round 2 — 2026-10-10T14:21:22-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — TailThread carries WorkingPath; TestFastPeekMatchesTheTypedPeek fails (missing working_path) when message_service.go:130 is removed — mutation-verified.
- BR-2 — addressed — The issue's Revisions "close round 1 findings" records that the fallback never calls the resolver; the code (peek.go:185-197) matches.
- BR-3 — withdrawn — The flags arrive normalized from ParseCLI, and any flag fastPeek does not recognize falls back to the typed path, so a mismatch can only cost speed, never give a wrong answer.
- BR-4 — withdrawn — Accepted on purpose and recorded in Revisions: the fast path resolves slots the way --send-to does, and the fallback names any reference it cannot resolve in unavailable.
- BR-5 — addressed — The "an older couch" case in TestFastPeekAnswersOnlyWhenEverySlotIsLive answers invalid-request and checks the fallback runs with nothing written.

### Raised

- **BR-6** [Minor] `plan-prose-matches-code` The plan and atlas still describe TailThread and the agent's source as they were before BR-1
  2nd finding in this family. The plan's Design item 1 and atlas/couch.md say TailThread is {slot, tag, agent} (no working_path, and the agent is not said to come from the record). Plan item 4 says "LatestLaunchProfile, else the first incarnation", but RecordAgent prefers the sole incarnation, then the latest launch. Rule: when a fix changes a surface that prose describes, grep every restatement (plan Design, atlas, SKILL) in the same commit, and put a Revisions entry in the durable plan file, not only in the issue.

## Open findings

- **BR-6** [Minor] `plan-prose-matches-code` The plan and atlas still describe TailThread and the agent's source as they were before BR-1
