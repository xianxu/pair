# Boundary Review — pair-claude-resume#676 (whole-issue close)

| field | value |
|-------|-------|
| issue | 676 — Exact Claude resume fails above native tree metadata cap |
| repo | pair-claude-resume |
| issue file | workshop/issues/000676-claude-exact-resume-cap.md |
| boundary | whole-issue close |
| milestone | — |
| window | 8f39e1d9e7d4a26c36261a90bd3674f501ea0c9e..518a5afda71522b491f82a3308e97db9b07d3195 |
| command | sdlc close --issue 676 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-18T15:39:29-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned change matches the issue’s Spec and Plan. Exact Claude lookup avoids unrelated descendant traversal while preserving bounded IO, duplicate detection and existing transcript validation. Latest discovery retains its cap and incomplete-evidence refusal. No blocking findings.

```findings
{}
```

1. **Strengths**

   - The optional visitor preserves existing runtime compatibility and reuses the scanner’s identity and role checks.
   - Directory fingerprints check descriptor and pathname identity, including mutation timestamps.
   - Real filesystem tests cover over-10,000-entry history, relocated sessions, duplicate roots, controlled mutations, cancellation and descriptor closure.
   - Both relevant atlas pages document the lookup change. No new user-facing command, flag or configuration requires a README update.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Independently passed:

   - Focused selector, Claude exact-lookup and fingerprint tests: 4.658 seconds.
   - Full `sessioninventory` package with `-race`: 190.514 seconds.
   - Focused launcher native resume, selector and inventory tests: 16.376 seconds.
   - Pinned-range `git diff --check`.

   These checks do not replace the separately tracked recorded suite, landing and operator acceptance gates.

6. **Architectural notes**

   - **ARCH-DRY — Pass:** Reuses root authorization, budgets and content validation.
   - **ARCH-PURE — Pass:** Fingerprint conversion is tested without IO; filesystem work stays behind the runtime seam.
   - **ARCH-PURPOSE — Pass:** Addresses the global-history failure while preserving cross-project lookup and duplicate refusal.
   - **ARCH-MOCK — Pass:** Controlled filesystem mutations exercise the production helper through an injected adapter.
   - **ARCH-CONSTRAINTS — Pass:** Immediate-entry work remains bounded; descendant growth does not increase lookup work.
   - **ARCH-SECURE — Pass:** Canonical IDs, no-follow opens, regular-file checks and parent revalidation preserve fail-closed behavior.
   - **ARCH-ORDER — Pass:** Synchronous probe ordering is explicit and tested with deterministic mutations; no new state persists between events.
   - **ARCH-FUNERAL — Pass:** No durable artifacts are introduced; descriptor cleanup is tested across success and failure paths.

7. **Plan revision recommendations:** None. The concept tables and completed implementation claims match the code.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"40f867c5-d772-413c-8dce-f98739d4f410","repository":"github.com/xianxu/pair","issue_id":"000676","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/6e906975ecc8d0d4","reviewed_head":"518a5afda71522b491f82a3308e97db9b07d3195","slot":"pair:5","requested_at":"2026-10-18T22:35:28.505506Z"},"phase":"finished","pid":96702,"started_at":"2026-10-18T22:35:28.716966Z","finished_at":"2026-10-18T22:39:28.276748Z","session_id":"01a15127-dc01-79f3-8176-747507c42da4","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.255433417s
head: "518a5afda71522b491f82a3308e97db9b07d3195"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
