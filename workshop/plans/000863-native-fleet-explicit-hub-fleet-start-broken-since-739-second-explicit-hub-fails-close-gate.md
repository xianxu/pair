---
gate: boundary-review
issue: 863
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: ea273863-bf00-4614-8b89-8827b75709b6
            repository: github.com/xianxu/pair
            issue_id: "000863"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/8140c6109d7070fe
            reviewed_head: 68bc00e8c2df410b5d53807a9733ef4bb6b0f3ee
            slot: pair:7
            requested_at: "2026-10-24T08:56:28.831565Z"
          phase: finished
          pid: 93491
          started_at: "2026-10-24T08:56:28.99893Z"
          finished_at: "2026-10-24T09:04:06.01835Z"
          session_id: 01a16d20-33e9-7433-8e9e-579382765b6d
          outcome: returned
      "n": 1
      timestamp: "2026-10-24T02:04:07-07:00"
      dispatched: "2026-10-24T01:56:25-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - kind: defect
          id: BR-1
          severity: Critical
          title: Automatic fresh startup can replace an existing stopped conversation
          detail: 'cmd/internal/couchcore/startup.go:147 calls StartFreshSlot, which passes requireEmpty=false (slotrecovery.go:334). The initial observation only discovers local stores for manifest-enrolled repositories (threadstore_location.go:44), so an empty namespace can classify fresh despite a configured numbered hub retaining .couch/thread.json. Startup then archives that stopped conversation and creates another without an explicit Fresh gesture. ARCH-SECURE and ARCH-ORDER: inspect configured local history before granting fresh authority, and enforce an empty-current precondition at mutation time. Add regressions for unenrolled retained history and history appearing after observation.'
          family: automatic-launch-requires-empty-authority
          round: 1
        - kind: unverifiable
          id: BR-2
          severity: Important
          title: The required ExplicitHub native acceptance matrix is not established
          detail: 'Ran PAIR_NATIVE_COUCH_FLEET=1 SDLC_DEP_SUBSTRATE=/Volumes/TestScratch/review-checkouts/5562c74d00a70f30 go test ./cmd/couch -run ''^TestDisposable(FreshHubCaller|EditedRestart|StartupArguments|CouchSYNTHETICLimitAndBusyForcePreserveDirtyFiles)$'' -count=1 -timeout=12m. TestDisposableStartupArguments/claude failed at startup_arguments_fleet_test.go:55 with switch-agent receipt Status:failed, Detail:attach terminal input has ended; wrapper evidence also reported terminal listing omitted a live server. Its Codex case was consequently skipped. ARCH-PURPOSE: provide passing acceptance for both personalities, or measured evidence classifying this failure as independent of the change before claiming the issue''s Done-when condition.'
          family: native-acceptance-evidence
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#863 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-24T02:04:07-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `automatic-launch-requires-empty-authority` (kind: defect) Automatic fresh startup can replace an existing stopped conversation
  cmd/internal/couchcore/startup.go:147 calls StartFreshSlot, which passes requireEmpty=false (slotrecovery.go:334). The initial observation only discovers local stores for manifest-enrolled repositories (threadstore_location.go:44), so an empty namespace can classify fresh despite a configured numbered hub retaining .couch/thread.json. Startup then archives that stopped conversation and creates another without an explicit Fresh gesture. ARCH-SECURE and ARCH-ORDER: inspect configured local history before granting fresh authority, and enforce an empty-current precondition at mutation time. Add regressions for unenrolled retained history and history appearing after observation.
- **BR-2** [Important] `native-acceptance-evidence` (kind: unverifiable) The required ExplicitHub native acceptance matrix is not established
  Ran PAIR_NATIVE_COUCH_FLEET=1 SDLC_DEP_SUBSTRATE=/Volumes/TestScratch/review-checkouts/5562c74d00a70f30 go test ./cmd/couch -run '^TestDisposable(FreshHubCaller|EditedRestart|StartupArguments|CouchSYNTHETICLimitAndBusyForcePreserveDirtyFiles)$' -count=1 -timeout=12m. TestDisposableStartupArguments/claude failed at startup_arguments_fleet_test.go:55 with switch-agent receipt Status:failed, Detail:attach terminal input has ended; wrapper evidence also reported terminal listing omitted a live server. Its Codex case was consequently skipped. ARCH-PURPOSE: provide passing acceptance for both personalities, or measured evidence classifying this failure as independent of the change before claiming the issue's Done-when condition.

## Open findings

- **BR-1** [Critical] `automatic-launch-requires-empty-authority` (kind: defect) Automatic fresh startup can replace an existing stopped conversation
- **BR-2** [Important] `native-acceptance-evidence` (kind: unverifiable) The required ExplicitHub native acceptance matrix is not established
