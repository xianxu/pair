# Boundary Review — pair#643 (whole-issue close)

| field | value |
|-------|-------|
| issue | 643 — couch session-sync: frontmatter-scoped read/write that moves the diff baseline only for the frontmatter |
| repo | pair |
| issue file | workshop/issues/000643-couch-session-sync-frontmatter-scoped-read-write-that-moves-the-diff-baseline-only-for-the-frontmatter.md |
| boundary | whole-issue close |
| milestone | — |
| window | f5bf8d2253f0ba0b91fe330f0db126d8bb005e0f..eaada100e0cfb422a3bc3aa1859f2eb5d411b29a |
| command | sdlc close --issue 643 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-17T05:03:06-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned range fulfills the frontmatter scope contract: reads and writes preserve unread body edits, writes retain body bytes, and stale hashes and operator locks refuse correctly. Documentation covers the new CLI surface. No blocking findings.

```findings
{}
```

1. **Strengths**
   - `cmd/internal/sessionsync/file.go:141` defines one byte boundary for projection and replacement, preserving LF/CRLF tails without YAML reserialization.
   - `cmd/internal/sessionsync/sync.go:403` handles absent shadows and ordinary-document baselines conservatively.
   - `cmd/internal/sessionsync/frontmatter_test.go:157` exercises a real publication failure and verifies retained body edits, owed events, and successful reconciliation.
   - CLI scope validation rejects conflicting forms; help uses the existing light cost tag.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**
   - Passed focused frontmatter tests in both affected packages.
   - Passed `go test ./cmd/internal/sessionsync ./cmd/internal/couchcmd -count=1`.
   - Passed range whitespace checks.
   - Pure helpers have direct tests; integration coverage uses disposable stores and exercises production read/write/diff paths. The repository remained unchanged.

6. **Architectural notes**
   - **ARCH-DRY — pass:** shared projection/replacement helpers and existing publication machinery.
   - **ARCH-PURE — pass:** byte transformations remain independent of IO.
   - **ARCH-PURPOSE — pass:** scoped acknowledgement preserves body baselines across both routing modes.
   - **ARCH-MOCK — pass:** no new external dependency or direct call bypasses the existing seams.
   - **ARCH-CONSTRAINTS — pass:** registered limits remain enforced; the inherited legacy envelope is documented.
   - **ARCH-SECURE — pass:** malformed blocks and body-bearing replacement payloads refuse visibly.
   - **ARCH-ORDER — pass:** preparation precedes publication; controlled failure tests validate conservative partial-progress recovery.
   - **ARCH-FUNERAL — pass:** no new durable artifact family or increased per-event residue.

7. **Plan revision recommendations:** None required. The implemented entities and behavior match the plan; close and landing remain explicitly pending.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"a9467607-e570-4d46-8dca-e6d4dddf2608","repository":"github.com/xianxu/pair","issue_id":"000643","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/3643a6e85dbbedf6","reviewed_head":"eaada100e0cfb422a3bc3aa1859f2eb5d411b29a","slot":"pair:6","requested_at":"2026-10-17T12:00:51.47998Z"},"phase":"finished","pid":26149,"started_at":"2026-10-17T12:00:51.705504Z","finished_at":"2026-10-17T12:03:05.359803Z","session_id":"01a149bc-7dc2-7010-9baf-08e3f9a96cd6","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.245772417s
head: "eaada100e0cfb422a3bc3aa1859f2eb5d411b29a"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
