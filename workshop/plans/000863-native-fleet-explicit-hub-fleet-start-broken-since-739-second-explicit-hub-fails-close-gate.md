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
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 7afa2210-0e71-49d9-bf5a-37a5751ec1f2
            repository: github.com/xianxu/pair
            issue_id: "000863"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/a1b688d778015779
            reviewed_head: 71bb4b2e719f929a3873145020110dd138634709
            slot: pair:7
            requested_at: "2026-10-24T10:35:51.33727Z"
          phase: finished
          pid: 68024
          started_at: "2026-10-24T10:35:51.432715Z"
          finished_at: "2026-10-24T10:39:57.239786Z"
          session_id: 01a16d7b-2e29-7410-b27e-d01255a430f4
          outcome: returned
      "n": 2
      timestamp: "2026-10-24T03:39:59-07:00"
      dispatched: "2026-10-24T03:35:49-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: Configured unenrolled history is inspected, and ordinary/numbered creation rechecks emptiness under publication locks. Targeted tests pass; disabling these protections in a disposable pinned-head copy makes the observation and late-history regressions fail.
          round: 2
        - id: BR-2
          disposition: not-addressed
          note: Inspected branch-required-native-7a71f335.log and the pinned-main comparison captures. StartupArguments/claude fails and skips Codex; EditedRestart/codex also fails in the matrix. The main StartupArguments failure has a different signature from the original attach-input-ended failure, while later passing comparisons do not establish its independence. Passing two-personality acceptance or measured classification of the outstanding failures is still needed; the recorded close approval and follow-up issues do not supply that evidence.
          round: 2
      findings:
        - kind: defect
          id: BR-3
          severity: Critical
          title: Post-commit discovery or enrollment failure strands a creating hub transaction
          detail: cmd/internal/couchcore/slotrecovery.go:491-499 returns directly after claimFreshRecord has published the local creating incarnation and native address claim. Discovery failure, identity disagreement, or enrollment failure bypasses launchClaimedThread and its rollback handling. In a disposable pinned-head copy, injecting a discovery failure after artifact Claim reproduced an error with zero launched children but a persisted creating transaction owned by the live supervisor. Finalize every post-publication failure through exact-record/nonce cleanup that also works before enrollment, and add regressions covering these exits. ARCH-ORDER and ARCH-FUNERAL.
          family: committed-starts-require-terminal-disposition
          round: 2
      recipe: milestone-review
      isolation: fenced
      blocked: true
disputes:
    - id: BR-2
      rationale: 'Dispute the broad matrix as a landing blocker for the narrowed startup correction, not the retained failures or their unknown original cause. Fresh two-hub callers and dirty/force pass for both personalities on corrected code; later StartupArguments failure has a measured matching pinned-main Zellij panic now owned by #866. The single Codex idle refusal did not reproduce in either bounded comparison and is preserved in #867, not declared independent or flaky. TL24a27ad0 approves proceeding on this measured scope; TL1d357557 directs this formal dispute instead of further native retries. The original reviewer attach-input-ended/omitted-server signature remains unclassified; neither later passes nor TL approval is offered as proof of its cause.'
      evidence: 'Retained branch-required-native-7a71f335.log: FreshHubCaller PASS Claude14.34s/Codex20.08s and dirty-force PASS both; second StartupArguments switch has Zellij0.45.1 lib.rs1462 panic. Pinned-main4053f3da main-startup-arguments-4053f3da.log has same lib.rs1462 panic before worker start (different phase, not original signature). Exact EditedRestart/Codex pinned-main comparison PASS48.85s and one instrumented branch comparison PASS99.20s, both exact cleanup; branch original refusal retained with receipt dda5ab2f-ae24-4aca-9841-7b19f75d54a8. Full captures and SHA256 are committed under workshop/evidence/000863; #866 and #867 explicitly preserve the limitations. No deadline increase, weakened idle guard, new native repeat or complete-matrix pass claim.'
      after: 2
      timestamp: "2026-10-24T10:45:15Z"
      by: Xian Xu on Xian’s MacBook Pro at pair:7 (/Users/xianxu/workspace/worktree/pair-slot7/pair)
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

## Round 2 — 2026-10-24T03:39:59-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — Configured unenrolled history is inspected, and ordinary/numbered creation rechecks emptiness under publication locks. Targeted tests pass; disabling these protections in a disposable pinned-head copy makes the observation and late-history regressions fail.
- BR-2 — not-addressed — Inspected branch-required-native-7a71f335.log and the pinned-main comparison captures. StartupArguments/claude fails and skips Codex; EditedRestart/codex also fails in the matrix. The main StartupArguments failure has a different signature from the original attach-input-ended failure, while later passing comparisons do not establish its independence. Passing two-personality acceptance or measured classification of the outstanding failures is still needed; the recorded close approval and follow-up issues do not supply that evidence.

### Raised

- **BR-3** [Critical] `committed-starts-require-terminal-disposition` (kind: defect) Post-commit discovery or enrollment failure strands a creating hub transaction
  cmd/internal/couchcore/slotrecovery.go:491-499 returns directly after claimFreshRecord has published the local creating incarnation and native address claim. Discovery failure, identity disagreement, or enrollment failure bypasses launchClaimedThread and its rollback handling. In a disposable pinned-head copy, injecting a discovery failure after artifact Claim reproduced an error with zero launched children but a persisted creating transaction owned by the live supervisor. Finalize every post-publication failure through exact-record/nonce cleanup that also works before enrollment, and add regressions covering these exits. ARCH-ORDER and ARCH-FUNERAL.

## Disputes

Findings the implementer declined (`sdlc finding dispute`), each answered by the
next review round.

- **BR-2** — after round 2 (2026-10-24T10:45:15Z, Xian Xu on Xian’s MacBook Pro at pair:7 (/Users/xianxu/workspace/worktree/pair-slot7/pair))
  rationale: Dispute the broad matrix as a landing blocker for the narrowed startup correction, not the retained failures or their unknown original cause. Fresh two-hub callers and dirty/force pass for both personalities on corrected code; later StartupArguments failure has a measured matching pinned-main Zellij panic now owned by #866. The single Codex idle refusal did not reproduce in either bounded comparison and is preserved in #867, not declared independent or flaky. TL24a27ad0 approves proceeding on this measured scope; TL1d357557 directs this formal dispute instead of further native retries. The original reviewer attach-input-ended/omitted-server signature remains unclassified; neither later passes nor TL approval is offered as proof of its cause.
  evidence: Retained branch-required-native-7a71f335.log: FreshHubCaller PASS Claude14.34s/Codex20.08s and dirty-force PASS both; second StartupArguments switch has Zellij0.45.1 lib.rs1462 panic. Pinned-main4053f3da main-startup-arguments-4053f3da.log has same lib.rs1462 panic before worker start (different phase, not original signature). Exact EditedRestart/Codex pinned-main comparison PASS48.85s and one instrumented branch comparison PASS99.20s, both exact cleanup; branch original refusal retained with receipt dda5ab2f-ae24-4aca-9841-7b19f75d54a8. Full captures and SHA256 are committed under workshop/evidence/000863; #866 and #867 explicitly preserve the limitations. No deadline increase, weakened idle guard, new native repeat or complete-matrix pass claim.

## Open findings

- **BR-3** [Critical] `committed-starts-require-terminal-disposition` (kind: defect) Post-commit discovery or enrollment failure strands a creating hub transaction
