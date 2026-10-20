# Boundary Review — pair#698 (whole-issue close)

| field | value |
|-------|-------|
| issue | 698 — flake: TestListedDetachedTerminalAttachesWithoutNewServer |
| repo | pair |
| issue file | workshop/issues/000698-flake-testlisteddetachedterminalattacheswithoutnewserver.md |
| boundary | whole-issue close |
| milestone | — |
| window | 8f84dd12619483a9be757c5ba6b2ae49d66e1491..50b2d2b03dc9bcb432725bbccd51cd35dd04de7c |
| command | sdlc close --issue 698 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-19T07:57:08-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned change matches the Spec: listing-ready now precedes agent-ready, with production admission and detached reuse behavior unchanged. The regression passes at HEAD and fails on explicit readiness/listing assertions when the original order is restored. No findings.

```findings
{}
```

1. **Strengths**
   - The fixture uses the ordered publisher directly (`terminal_registration_fixture_test.go:69,175`).
   - The regression observes real fixture replies between writes for both supported listing forms (`terminal_registration_reply_test.go:62–78`).
   - Both write failures preserve the error and leave agent readiness absent; success preserves exact readiness bytes.
   - Existing detached reuse assertions retain server identity and one-server/one-create checks.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage**
   - Passed publication, listing forms, empty observations, nested environment isolation, and detached attach tests.
   - Passed artifactpath, testpair, and termcmd packages.
   - Reversed-order overlay failed in 0.31s on semantic assertions, without a timeout oracle. Overlay removed; tracked workspace unchanged.
   - `git diff --check` passed. Full required `make test` suite was not rerun during this review; recorded-suite and landing tasks remain explicitly pending.

6. **Architecture**
   - **ARCH-DRY: Pass.** Reuses atomic filesystem writes and existing fixture reply generation.
   - **ARCH-PURE: Pass.** Small injected IO seam; plan correctly classifies the entities as integration.
   - **ARCH-PURPOSE: Pass.** Fixes the publication race and preserves end-to-end detached attach coverage.
   - **ARCH-EVOLVE: Pass.** No persisted format changes.
   - No new user-facing or architectural surface requiring README or atlas updates.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"35f112a8-3016-43f0-9759-05317c62c973","repository":"github.com/xianxu/pair","issue_id":"000698","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/9fa006b8a0636ced","reviewed_head":"50b2d2b03dc9bcb432725bbccd51cd35dd04de7c","slot":"pair:5","requested_at":"2026-10-19T14:56:07.120968Z"},"phase":"finished","pid":7848,"started_at":"2026-10-19T14:56:07.253172Z","finished_at":"2026-10-19T14:57:07.736832Z","session_id":"01a154a9-a97c-7121-bcaf-9aead89d96ea","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.094420167s
head: "50b2d2b03dc9bcb432725bbccd51cd35dd04de7c"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
