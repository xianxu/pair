# Boundary Review — pair#862 (whole-issue close)

| field | value |
|-------|-------|
| issue | 862 — session-sync apply advances the diff baseline over the whole file, wiping the operator's unread edits elsewhere |
| repo | pair |
| issue file | workshop/issues/000862-session-sync-apply-advances-the-diff-baseline-over-the-whole-file-wiping-the.md |
| boundary | whole-issue close |
| milestone | — |
| window | f01ccc06b55fd3c5ac383fe172f4a4bb3ab74bcf..95be8dec481a8c3e53c16c4d859269536d91bf25 |
| command | sdlc close --issue 862 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-24T16:21:54-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The implementation preserves unrelated unread edits and conservatively retains the baseline when mapping fails. The session-sync suite and CLI regression pass. One documentation correction remains: the required dependent handoff still describes recovery that the implementation explicitly disproves.

```findings
findings:
  - id: new
    severity: Important
    kind: defect
    family: approved-contract-handoff-consistency
    title: |
      Correct the dependent handoff's obsolete scoped-read recovery
    detail: |
      Pair's plan marks the approved-contract handoff complete at workshop/plans/000862-apply-shadow-plan.md:71, but the pinned Ariadne tracker workshop/issues/000615-session-sync-apply-shadow-fake.md:20 still specifies "scoped-read recovery". TestApplyRetainsUnmappableBaseline proves section reads leave the baseline unchanged; recovery requires reviewing diff and explicitly reading --all. ARCH-PURPOSE: correct the receiving ticket through its owner before considering the handoff complete. This is the sole stale prescriptive recovery instance in the receiving handoff; Pair's README, atlas, help, embedded skill and revised plan already describe the correct recovery.
```

1. **Strengths**
   - `apply_shadow.go:18–58` reuses existing diff logic and checks mapped bytes before replay.
   - `sync.go:593` confines the change to apply preparation, before publication.
   - Integration tests exercise legacy/registered documents, broad records, no-ops, missing baselines, aliases and composed warnings.
   - README, atlas, CLI help and embedded guidance consistently explain the implemented recovery.

2. **Critical findings:** None.

3. **Important findings:** Correct the dependent contract handoff described above. No runtime change is needed.

4. **Minor findings:** None.

5. **Test coverage notes**
   - Passed `go test ./cmd/internal/sessionsync -count=1`.
   - Passed the focused apply tests and fuzz seeds.
   - Passed `TestSessionSyncApplyShadowCLI`.
   - `git diff --check` passed. No reviewed files changed.

6. **Architectural notes**
   - **ARCH-DRY — pass:** existing `changedRuns` supplies both mappings.
   - **ARCH-PURE — pass:** projection remains deterministic and IO-free; publication stays in the existing shell.
   - **ARCH-PURPOSE — flag:** runtime purpose is delivered; the receiving handoff needs the final recovery contract.
   - **ARCH-EVOLVE — pass:** no persisted format changes.

7. **Plan revision recommendations:** No design revision needed. Correct the dependent ticket and record that the approved recovery contract was handed off.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"d669e796-5ecb-45f0-b6a4-3a7cdaecfc00","repository":"github.com/xianxu/pair","issue_id":"000862","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/66b590850f318975","reviewed_head":"95be8dec481a8c3e53c16c4d859269536d91bf25","slot":"pair:3","requested_at":"2026-10-24T23:20:36.234414Z"},"phase":"finished","pid":90082,"started_at":"2026-10-24T23:20:36.353637Z","finished_at":"2026-10-24T23:21:52.614393Z","session_id":"01a17037-5597-7990-8488-0f8205cd0bbe","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.429105666s
head: "95be8dec481a8c3e53c16c4d859269536d91bf25"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
