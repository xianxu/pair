# Boundary Review — pair#633 (whole-issue close)

| field | value |
|-------|-------|
| issue | 633 — couch --list picks relaunch vs resend from a typed refusal class, not by matching error text |
| repo | pair |
| issue file | workshop/issues/000633-couch-list-picks-relaunch-vs-resend-from-a-typed-refusal-class-not-by-matching-error-text.md |
| boundary | whole-issue close |
| milestone | — |
| window | c9c4daa2147cfdcc3568a330e14eda43ebba25d2..b9071aba1f20096e3a37e08b35e77e52d82b2e2f |
| command | sdlc close --issue 633 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-17T08:21:24-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The implementation delivers the issue’s purpose: refusal sources carry a typed classification, the broker retains and exposes it, and list rendering selects recovery without matching diagnostic text. No blocking defects found in the pinned range.

Done-when: current — the unchanged clauses cover wording-independent relaunch advice, unclassified resend advice, and removal of the phrase list; the revision only separates acceptance from publishing.

```findings
{}
```

1. **Strengths**
   - All nine source sites across the five refusal families carry the typed error, including both registration paths (`message_service.go`).
   - `refusalInfo` preserves cancellation/deadline precedence and error unwrapping (`refusal.go:42`).
   - The unauthenticated projection excludes sender credentials and message bodies; tests verify this.
   - README and atlas explain the new classification and mixed-version fallback.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes:** Independently ran `go test ./cmd/internal/couchmessage ./cmd/internal/couchcmd`; both passed (23.721s and 55.372s). Tests cover reworded classifications, unclassified legacy phrases, unknown classes, cancellation/deadline precedence, source typing, JSON privacy, mixed-version decoding, expiry, eviction, and supersession. `git diff --check` passed. The recorded repository-wide suite was not independently rerun.

6. **Architectural notes**
   - **ARCH-DRY: pass.** Classification is projected centrally; the consumer’s parallel phrase classifier is removed.
   - **ARCH-PURE: pass.** Recovery classification and rendering remain deterministic; existing injected IO boundaries are preserved.
   - **ARCH-PURPOSE: pass.** The source-to-broker-to-list path consumes the typed classification throughout. The consumer sweep found no remaining phrase-based recovery classifier.
   - No Core concepts table is present.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"2f84ae16-d4e7-4f6a-942e-daeb9a1df375","repository":"github.com/xianxu/pair","issue_id":"000633","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/998a9211a3724e24","reviewed_head":"b9071aba1f20096e3a37e08b35e77e52d82b2e2f","slot":"pair:7","requested_at":"2026-10-17T15:19:42.448793Z"},"phase":"finished","pid":51681,"started_at":"2026-10-17T15:19:42.683999Z","finished_at":"2026-10-17T15:21:23.749566Z","session_id":"01a14a72-8aed-75e0-bba9-df48368dad2d","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.693913875s
head: "b9071aba1f20096e3a37e08b35e77e52d82b2e2f"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
