# Boundary Review — pair#838 (whole-issue close)

| field | value |
|-------|-------|
| issue | 838 — Couch: single collaborative mode and the @@chat@@ carve-out; legacy mode keys mapped or ignored |
| repo | pair |
| issue file | workshop/issues/000838-couch-single-collaborative-mode-and-the-chat-carve-out-legacy-mode-keys-mapped.md |
| boundary | whole-issue close |
| milestone | — |
| window | e340924b417db3a9a1e794d2e302708cbfad9774..09f4314e925592fa18e7ce6d9051c475a96a88a4 |
| command | sdlc close --issue 838 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-24T10:59:08-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The single-mode simplification, shared scanner integration, migration safeguards, and historical-round compatibility are well supported. One reproduced ownership defect blocks shipping: an agent can transfer an existing chat tag to a newly inserted heading when legacy chat keeps routing unchanged.

1. **Strengths**

   - Runtime catalog and retired-axis resolution are removed; shared scanner facts drive chat routing.
   - Migration checks heading placement and exact eligible-item projections before returning a candidate ([migrate.go](/Volumes/TestScratch/review-checkouts/e2fb601e2d20d64a/cmd/internal/editorial/migrate.go:155)).
   - Historical presentation and nested unknown fields have round-trip coverage.
   - All 85 provenance entries matched the actual pinned upstream Git objects. README and atlas updates cover the new surface.

2. **Critical findings**

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: operator-authority-association
    title: |
      Chat tags can transfer to another heading without an ownership refusal
    detail: |
      cmd/internal/editorial/edits.go:132 checks declaration position, key and bytes, while lines 153–171 check the original heading's identity and parent; neither verifies that the declaration still belongs to that heading. With frontmatter collaboration-mode: chat followed by @@chat@@, # Questions and 1. Why?, inserting # Injected directly between the tag and # Questions succeeds through ValidateEdits, BuildEdit and Sync.Apply, which publishes the changed document. Whole-document legacy chat keeps effective routing unchanged, so the surviving-byte check cannot detect the transfer. This violates the promised tag-attachment protection (ARCH-SECURE, ARCH-PURPOSE). Compare each candidate declaration's owning heading with the mapped original owner, sweep both tags and heading directives, and add rejection tests through the public write seams.
```

3. **Important findings**

   None.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Existing editorial, session-sync, and couchcmd package tests passed.
   - Editorial race tests, focused public CLI tests, and authored-fixture checks passed.
   - Temporary overlay regressions exposed the defect at three levels: `ValidateEdits` accepted it, `BuildEdit` returned the unauthorized candidate, and `Sync.Apply` published it.
   - The overlays were removed; the repository remains unchanged. Existing ownership tests cover deletion and renaming but miss transfer with unchanged routing.

6. **Architectural notes for upcoming work**

   - **ARCH-DRY — pass:** routing and item grammar reuse shared authority.
   - **ARCH-PURE — pass:** compilation, validation, and migration remain free of IO.
   - **ARCH-PURPOSE — flag:** attachment protection is incomplete despite preserved bytes and routing.
   - **ARCH-MOCK — pass:** public-path tests use portable stateful filesystem rigs and existing synthetic-agent seams.
   - **ARCH-CONSTRAINTS — pass:** counted growth tests exercise the adapter; no new production concurrency is introduced.
   - **ARCH-SECURE — flag:** declaration ownership must include its attachment identity.
   - **ARCH-ORDER — pass:** existing transaction, stale-token, lock, and uncertain-publication handling remain intact.
   - **ARCH-FUNERAL — pass:** no new production artifact family is introduced.
   - **ARCH-EVOLVE — pass:** historical presentation and unknown nested members retain compatibility coverage.

7. **Plan revision recommendations**

   Add a `## Revisions` entry recording attachment transfer as an explicit adversarial case: validate declaration-to-heading identity independently of effective routing, and exercise tags and heading directives under redundant legacy authority across the public write seams.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"fd561ca3-f4a0-4a56-9304-eacc79b9dbb9","repository":"github.com/xianxu/pair","issue_id":"000838","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/e2fb601e2d20d64a","reviewed_head":"09f4314e925592fa18e7ce6d9051c475a96a88a4","slot":"pair:3","requested_at":"2026-10-24T17:52:49.857987Z"},"phase":"finished","pid":71988,"started_at":"2026-10-24T17:52:50.02454Z","finished_at":"2026-10-24T17:58:51.664118Z","session_id":"01a16f0b-3ef3-7a02-8097-32b51b25d57b","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 14.637042708s
head: "09f4314e925592fa18e7ce6d9051c475a96a88a4"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000413

---

## Re-review — 2026-10-24T11:08:28-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 838 — Couch: single collaborative mode and the @@chat@@ carve-out; legacy mode keys mapped or ignored |
| repo | pair |
| issue file | workshop/issues/000838-couch-single-collaborative-mode-and-the-chat-carve-out-legacy-mode-keys-mapped.md |
| boundary | whole-issue close |
| milestone | — |
| window | e340924b417db3a9a1e794d2e302708cbfad9774..dfdfb47c55bd4655d9401712bec710e1b7593b6f |
| command | sdlc close --issue 838 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-24T11:08:28-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The implementation delivers the single collaborative mode, chat protection, legacy compatibility, and highlight-only new rounds. BR-1 is addressed with meaningful regression evidence: removing the owner comparison makes tests fail through the pure validator and public write seams. No blocking findings remain; one help-text omission is minor.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      edits.go:143–159 binds each declaration to its mapped original heading. Regression tests cover tags and heading directives under redundant authority. Removing the owner comparison in a temporary Go overlay made ValidateEdits, BuildEdit, Write, Apply and ApplyFile regressions fail; pinned-head tests pass.
findings:
  - id: new
    severity: Minor
    kind: defect
    family: current-guidance-matches-runtime
    title: |
      CLI help still advertises resolved editorial briefs
    detail: |
      cmd/internal/couchcmd/run.go:1529–1530 still promises resolved editorial briefs and refers to every mode, although this change removes runtime brief resolution and introduces one collaborative mode. Update help to describe chat regions, protected sources and compatibility notices, matching README.md and the bundled skill (ARCH-PURPOSE).
```

1. **Strengths**

   - Shared `sidedoc` facts drive routing and item grammar; the runtime catalog and retired axis resolution are removed.
   - Declaration ownership now survives routing-neutral cases, with refusal tests checking unchanged document, shadow, state and round artifacts.
   - Migration verifies heading structure, chat coverage and exact eligible asks before returning a candidate.
   - Historical presentation values and nested unknown JSON members remain covered by compatibility tests. README and atlas updates explain the changed surface.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** CLI help retains the retired brief/mode description, as recorded above.

5. **Test coverage**

   Passed during review:

   - Full editorial and session-sync packages.
   - Editorial package with the race detector.
   - Focused collaborative CLI fixtures and session-sync editorial tests.
   - Authored acceptance fixture inputs and coarse-chat refusal tests.

   The BR-1 mutation check failed as expected without the fix. Temporary files were removed; the checkout remains unchanged. Native fleet acceptance was inspected but not rerun.

6. **Architectural checks**

   | Marker | Result |
   |---|---|
   | ARCH-DRY | Pass: shared scanner and item grammar remain authoritative. |
   | ARCH-PURE | Pass: compilation, validation and migration logic remain independent of IO. |
   | ARCH-PURPOSE | Flag: minor CLI guidance lag; behavioral purpose is delivered. |
   | ARCH-MOCK | Pass: public seams use portable stateful rigs and the existing native harness. |
   | ARCH-CONSTRAINTS | Pass: adapter growth counters exercise bounded, monotone scans. |
   | ARCH-SECURE | Pass: source authority and declaration ownership are enforced before publication. |
   | ARCH-ORDER | Pass: existing transaction ownership, freshness and uncertain-publication handling remain intact. |
   | ARCH-FUNERAL | Pass: no new runtime durable artifact family. |
   | ARCH-EVOLVE | Pass: historical meanings and unknown nested fields remain preserved. |

7. **Plan revisions**

   No design revision is required. Reconcile completed verification checkboxes with the recorded evidence when closing the tracker.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"19ca5d79-c994-4644-a577-573515e8bc16","repository":"github.com/xianxu/pair","issue_id":"000838","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/514ed0ceff67ffd3","reviewed_head":"dfdfb47c55bd4655d9401712bec710e1b7593b6f","slot":"pair:3","requested_at":"2026-10-24T18:05:25.277492Z"},"phase":"finished","pid":5893,"started_at":"2026-10-24T18:05:25.426849Z","finished_at":"2026-10-24T18:07:49.799322Z","session_id":"01a16f16-c5bb-7033-b135-2bfcf897a92d","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.0411085s
head: "dfdfb47c55bd4655d9401712bec710e1b7593b6f"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000410
