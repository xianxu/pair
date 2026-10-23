# Boundary Review — pair#823 (whole-issue close)

| field | value |
|-------|-------|
| issue | 823 — unified-editorial-interface: Couch parses, protects and enforces per-section modes (heading comments, aliases, migration) |
| repo | pair |
| issue file | workshop/issues/000823-unified-editorial-interface-couch-parses-protects-and-enforces-per-section-modes-heading-comments-aliases-migration.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4623c797574cd6aa12d765f6465454f74f02cc98..a859c72eb4600b7c979093b3714e4a7a75334867 |
| command | sdlc close --issue 823 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-23T04:42:53-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The shared compiler, write guard, migration flow, and presentation codecs are substantially implemented and pass focused checks. One reproduced correctness bug blocks shipping: chat protection omits part of an operator’s item when it ends in fenced code. README coverage is also missing for the new user-facing surface.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: protected-source-extent
    title: |
      Chat item protection excludes trailing fenced-code syntax
    detail: |
      cmd/internal/editorial/document.go:222-229 derives item extents from Goldmark block Lines(), which excludes fenced-code delimiters; edits.go:450-454 then trusts that truncated extent for protection and answer placement. For a chat item containing "- Why?\n\n  ```\n  example\n  ```\n", both ValidateEdits and production BuildEdit accept replacing the closing fence with "~~~", violating the operator-item preservation contract (ARCH-PURPOSE, ARCH-SECURE). Derive complete item source extents, including trailing syntax, and add regressions covering nonempty/empty fences and other block children without complete Lines() ranges.
  - id: new
    severity: Important
    kind: defect
    family: user-surface-documentation
    title: |
      README update is missing for editorial settings and migration commands
    detail: |
      cmd/internal/couchcmd/session_sync.go:65-66 introduces migrate-editorial and --confirm, while run.go:1516-1521 adds resolved editorial reads and policy-bound section hashes. README.md is unchanged in the pinned range; its session-sync usage at lines 510-527 omits migration and retains the older settings description. Update that usage with preview/confirmation, canonical settings and heading directives, protected ownership, and opaque section-hash behavior to satisfy the Docs update gate.
```

1. **Strengths**

   - All 39 pinned contract files match the dependency’s source Git objects byte for byte.
   - Protected-setting tests exercise registered, legacy, and standalone write paths and assert refusal leaves document/Shadow/state/round unchanged.
   - Section-hash tests verify policy changes invalidate unchanged section bodies.
   - Preserving codecs test nested unknown metadata, recursive duplicates, required members, and legacy presentation fallback.

2. **Critical findings**

   - `document.go:222-229`: incomplete item extents leave fenced-code closing syntax writable. Two temporary overlay regressions failed on the pinned head, including one through production `BuildEdit`. Fix the complete source-range calculation and retain those regressions.

3. **Important findings**

   - `README.md:510`: document the new surface described above. Atlas updates are present.

4. **Minor findings**

   - None.

5. **Test coverage notes**

   Passed:

   - Go packages: `editorial`, `sessionsync`, `reviewapply`.
   - Focused editorial Couch CLI tests.
   - Lua reconstruct, round-watch, and record tests.
   - `tests/review-applied-round-test.sh`.
   - `make test-affected-inventory`.
   - `git diff --check`.

   Existing fenced-code tests cover masking and link recognition, but miss complete chat-item protection. The two additional regressions demonstrated the gap. Temporary overlays were removed; the checkout remains unchanged. Native qualification and the full recorded suite were not rerun.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared compiler and marker grammar serve the inspected consumers.
   - **ARCH-PURE — pass:** policy compilation and edit validation are separated from transaction IO.
   - **ARCH-PURPOSE — flag:** incomplete chat-item extents underdeliver preservation.
   - **ARCH-MOCK — pass:** migration tests exercise temporary registry/filesystem state through production seams.
   - **ARCH-CONSTRAINTS — pass:** counted growth tests and document/round caps are present.
   - **ARCH-SECURE — flag:** the truncated range incorrectly grants authority over operator-owned syntax.
   - **ARCH-ORDER — pass:** migration uses the existing lock/CAS transaction; stale confirmation is tested.
   - **ARCH-FUNERAL — pass:** compiled policy is transaction-local; existing round lifecycle is reused.
   - **ARCH-EVOLVE — pass:** inspected codecs preserve unknown nested values and reject malformed known members.

7. **Plan revision recommendations**

   Add a `## Revisions` entry specifying that chat-item ownership covers complete source syntax, including trailing fenced-code delimiters and block children without full line ranges. Add the corresponding adversarial tests and README update to the completion checklist.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"e1063d65-ffa9-4199-96e1-d0f18f6eb7c7","repository":"github.com/xianxu/pair","issue_id":"000823","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/52487aa9f600139f","reviewed_head":"a859c72eb4600b7c979093b3714e4a7a75334867","slot":"pair:9","requested_at":"2026-10-23T11:38:35.700277Z"},"phase":"finished","pid":53375,"started_at":"2026-10-23T11:38:35.887383Z","finished_at":"2026-10-23T11:42:52.003239Z","session_id":"01a1688e-43ca-7641-8932-7764fd263afe","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.291445792s
head: "a859c72eb4600b7c979093b3714e4a7a75334867"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
