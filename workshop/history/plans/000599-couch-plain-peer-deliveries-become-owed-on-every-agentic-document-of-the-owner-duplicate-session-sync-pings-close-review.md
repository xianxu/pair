# Boundary Review — pair#599 (whole-issue close)

| field | value |
|-------|-------|
| issue | 599 — Couch: plain peer deliveries become owed on every agentic document of the owner (duplicate session-sync pings) |
| repo | pair |
| issue file | workshop/issues/000599-couch-plain-peer-deliveries-become-owed-on-every-agentic-document-of-the-owner-duplicate-session-sync-pings.md |
| boundary | whole-issue close |
| milestone | — |
| window | 53c96ded8854c4da834c594a9901b08c6f8078f2..876f0f80fe55981b58d7fb82a9995e6413cc19a5 |
| command | sdlc close --issue 599 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-16T08:46:20-07:00 |
| verdict | SHIP |

## Review

Still checking where Couch generates its pings and whether one owner can have more than one session-sync file.

Running the sessionsync tests, then reverting the fix to confirm the new tests fail without it.

```verdict
verdict: SHIP
confidence: high
```

The fix delivers what the issue asked for. `OwedForDocument` (`cmd/internal/sessionsync/owed.go:58`) now takes a `plain` flag. `Sync.owed` (`sync.go:289`) is the only production caller, and it sets the flag only when the document's current content is `type: session-sync`. Every status read and write gets its owed list through that one spot, so plain peer deliveries are now owed only on the owner's session file. Submits are still owed only on the document they name. I checked that the new test proves the fix: with `sync.go:289` reverted to always pass `true`, `TestPlainDeliveryOwedOnlyOnSessionFileAndSubmitOnlyOnItsDocument` fails ("plain document owes [report qa-submit], want only its own submit"). With the fix restored, `go test ./cmd/internal/sessionsync/ ./cmd/internal/couchcmd/` passes. Only Minor findings remain, and none of them blocks the gate.

1. **Strengths**
   - The fix lives in one place. `Sync.owed` (`sync.go:285-289`) is what both `Status` (`sync.go:315`) and the write/fold check (`sync.go:472`) call. So status output, the `--fold` check, and Parley's pings (which Parley builds from `status`) all pick up the change, with nothing duplicated.
   - `OwedForDocument` stays pure. The session-file decision is made by the caller and passed in as a bool (`owed.go:58`), and the unit test runs without any IO (`owed_test.go:91-104`).
   - The Sync-level test (`registry_test.go:510-544`) goes through the real `Register`, `Ledger.Append` and `Status` path. It covers both rules at once: a plain delivery is owed only on the session file, and a submit only on its document.
   - The fold-size bounds test was changed to build its entries as submits (`metadata_bounds_test.go:128`), so it still reaches its size limit under the new rule. Silently weakening it would have been easy.
   - The atlas "Owed" rule is updated (`atlas/couch.md:912-916`).

2. **Critical findings:** none.

3. **Important findings:** none.

4. **Minor findings**
   - `owed.go:71-72`: the comment "a malformed one stays owed as a plain delivery rather than vanishing" is now true only on the session file. A plain document drops it, though the session file still owes it, so nothing is lost overall. The comment should say that.
   - Nothing forces an owner to have only one `type: session-sync` file. If an owner has two (say, per-day TL files), each still owes every plain delivery, which was the old behaviour. The spec says "the owner's session file", singular, but neither the code nor the atlas states or enforces that.
   - `sync.go:422` lets a registered plain document gain `type: session-sync`. When that happens, every plain delivery since that document's `Since` becomes owed on it as well as on the existing session file. That is an edge case of the previous point.

5. **Test coverage notes**
   - The unit test and the Sync-level test both pin the rule, and I confirmed the Sync-level test fails without the fix.
   - The "two agentic documents" case in Done-when is tested as one session file plus one plain document. A second plain document follows the same branch, so this is enough.
   - Pings are produced in parley.nvim from `status`, so they aren't tested in this repo. They follow from `Sync.owed`.

6. **Architectural notes**
   - **ARCH-DRY: pass.** There is one predicate (`IsSessionSync`) and one call site.
   - **ARCH-PURE: pass.** The core stays pure and IO stays in `Sync.owed`.
   - **ARCH-PURPOSE: pass.** I checked every consumer of the owed list (status, folds, pings by way of status) and all of them now come from the single call site.
   - **ARCH-MOCK: N/A.** No new external dependency is added.
   - **ARCH-CONSTRAINTS: pass.** The extra cost is one bool check per ledger row.
   - **ARCH-SECURE: pass.** The `type:` frontmatter that decides session-file status is the same trust input already used for routing.
   - **ARCH-ORDER: pass, with a note.** The owed set depends on the document's current `type`, not on its type when a row arrived. So converting a plain document into a session file changes its owed set retroactively (the third Minor finding). This is acceptable as it stands.
   - **ARCH-FUNERAL: pass.** Nothing new is persisted, and fewer owed entries means the `Folded` state grows less.

7. **Plan revision recommendations:** none. The plan still matches the code.

```findings
findings:
  - id: new
    severity: Minor
    kind: defect
    family: stale-invariant-comment
    title: |
      owed.go comment says a malformed submit stays owed as a plain delivery; on non-session documents it is now dropped
    detail: |
      cmd/internal/sessionsync/owed.go:71-72. Update the comment to say a malformed submit is owed as a plain delivery on the owner's session file only.
  - id: new
    severity: Minor
    kind: defect
    family: single-session-file-assumption
    title: |
      Nothing guarantees one type: session-sync file per owner; two session files still each owe every plain delivery
    detail: |
      The spec assumes a single session file per owner. The code neither enforces that nor documents what happens otherwise, and sync.go:422 lets a plain registered document turn into a session-sync file, which brings back retroactive duplicate owed entries for that owner.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"47dd4006-6f41-412d-9641-813dca988f56","repository":"github.com/xianxu/pair","issue_id":"000599","recipe":"milestone-review","round":1,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/ba4ede798a146d00","reviewed_head":"876f0f80fe55981b58d7fb82a9995e6413cc19a5","slot":"pair:2","requested_at":"2026-10-16T15:42:41.585859Z"},"phase":"finished","pid":23266,"started_at":"2026-10-16T15:42:41.739346Z","finished_at":"2026-10-16T15:46:14.003484Z","session_id":"884430a6-24a8-4d61-bb0f-449d92567d1e","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.835756083s
head: "876f0f80fe55981b58d7fb82a9995e6413cc19a5"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
