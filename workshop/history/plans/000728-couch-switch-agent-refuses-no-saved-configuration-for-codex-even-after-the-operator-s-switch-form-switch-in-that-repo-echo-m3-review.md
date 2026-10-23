# Boundary Review — pair#728 (milestone M3)

| field | value |
|-------|-------|
| issue | 728 — couch --switch-agent refuses 'no saved configuration for codex' even after the operator's switch-form switch in that repo (echo) |
| repo | pair |
| issue file | workshop/issues/000728-couch-switch-agent-refuses-no-saved-configuration-for-codex-even-after-the-operator-s-switch-form-switch-in-that-repo-echo.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | e56ead30bf401a0ec70bf6c382a51326cb7d6031..c6967052e028bd896a40daa9f7751a4066984556 |
| command | sdlc milestone-close --issue 728 --milestone M3 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-22T17:34:31-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned M3 range matches the governing revised restart contract. Same-agent restart preserves the accepted native identity; fresh fallback requires positive absence and confirmed cleanup. No blocking findings. This verdict covers M3 implementation; operator acceptance and the final recorded suite remain prerequisites for whole-issue landing.

```findings
{}
```

1. **Strengths**

   - Accepted native identity travels through private typed acceptance, with cloned arguments and fingerprint revalidation (`cmd/internal/couchcore/switchagent.go:83`).
   - The pure restart decision consumes each launch request before effects and fences fallback by attempt, claim, cleanup revision, and nonce (`cmd/internal/couchcore/editedrestart.go:124`).
   - Evidence handling preserves missing versus replaced or uncertain observations; permission and cancellation failures remain refusals (`cmd/internal/couchcore/editedrestart_evidence.go:20`).
   - Console acceptance tests exercise held adoption, failed attachment, preference-publication failure, and warning propagation. README and atlas updates describe the new behavior.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Independently passed:

   - Focused restart, binding, and console tests across couchcore/couchcmd/couchtty.
   - Broader race checks covering edited restart, agent switching, force switching, and transcript binding across those packages.
   - The artifactpath package tests.
   - `make test-affected-inventory`.
   - Pinned-range whitespace checks.

   The disposable native acceptance test compiled and default-skipped because its opt-in environment variable was unset. I did not independently rerun native acceptance or the full repository suite. No checkout edits were made.

6. **Architectural notes**

   - **ARCH-DRY — Pass:** Reuses existing binding, claims, tracked launches, cleanup, and adoption owners.
   - **ARCH-PURE — Pass:** Restart decisions and evidence classification separate policy from IO; pure tests use no external doubles.
   - **ARCH-PURPOSE — Pass:** Both local and remote acceptance carry restart intent through execution and final adoption.
   - **ARCH-MOCK — Pass:** Stateful runner, artifact, session, and console fixtures exercise production seams.
   - **ARCH-CONSTRAINTS — Pass:** Located bindings retain one-stat/no-list behavior; metadata tests count probes and exclude body reads.
   - **ARCH-SECURE — Pass:** Serialized arguments cannot mint private restart/force acceptance; changed identities and uncertain observations refuse.
   - **ARCH-ORDER — Pass:** Explicit phases, exact cleanup receipts, sequence tests, and controlled barriers enforce bounded fallback.
   - **ARCH-FUNERAL — Pass:** New decision state and receipts are request-local; no new durable artifact family is introduced.
   - **ARCH-EVOLVE — Pass:** This range adds no persisted-format change; the public `DeleteStart` API remains compatible.

7. **Plan revision recommendations:** None required. The governing revisions explain the delivered file mapping, metadata completion, and deferred operator acceptance.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"30d68429-2cf4-444d-984a-1e208fee159b","repository":"github.com/xianxu/pair","issue_id":"000728","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/da10127af18cdd3b","reviewed_head":"c6967052e028bd896a40daa9f7751a4066984556","slot":"pair:3","requested_at":"2026-10-23T00:32:03.72371Z"},"phase":"finished","pid":74961,"started_at":"2026-10-23T00:32:03.869703Z","finished_at":"2026-10-23T00:34:30.081379Z","session_id":"01a1662c-08bf-7232-ad96-c8183f0f63dc","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.779274625s
head: "c6967052e028bd896a40daa9f7751a4066984556"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
