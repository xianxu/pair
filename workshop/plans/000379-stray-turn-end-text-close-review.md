# Boundary Review — pair#379 (whole-issue close)

| field | value |
|-------|-------|
| issue | 379 — Turn-end text flashes at focused pane cursor |
| repo | pair |
| issue file | workshop/issues/000379-stray-turn-end-text.md |
| boundary | whole-issue close |
| milestone | — |
| window | 6986321e216f9da591464e5991bcb03a5675fdf4..1271bf178b102a6b33536781a151943fb96f4b93 |
| command | sdlc close --issue 379 |
| reviewer | codex |
| timestamp | 2026-10-07T20:17:01-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned change fixes UTF-8 control-string leakage at the parser boundary and fulfills the issue’s regression contract. Independent testing confirms the endpoint regression fails with the original parser and passes with this repair. No blocking findings; two imported benchmarks need fixture cleanup.

1. **Strengths**
   - One shared parser repairs OSC/DCS/SOS/PM/APC containment without notification-specific filtering.
   - Endpoint coverage verifies both unchanged screen cells and an intact notification across transport splits.
   - Prefix tracking remains constant-space and preserves existing payload limits.
   - Atlas, licensing, provenance, and fork retirement guidance accompany the implementation.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** `third_party/vt/ansiparser/parser_test.go:185,200` copies benchmarks whose fixture files are absent. Include the licensed fixtures or remove those benchmarks.

5. **Test coverage**
   - Passed nested vt/parser `go test -race ./...`.
   - Passed the full terminal package and focused wrapper observer, boundary, notification-output, and rewriter race tests.
   - Independently reproduced endpoint regression failure using the inspected original-parser overlay.
   - Benchmark execution confirmed both missing-fixture failures. Working tree remained unchanged.

6. **Architecture**
   - **ARCH-DRY — pass:** consumers share the correction and upstream value types.
   - **ARCH-PURE — pass:** deterministic parsing remains separate from IO.
   - **ARCH-PURPOSE — pass:** addresses the control-string family, including production endpoint behavior.
   - **ARCH-MOCK — pass:** no new external dependency; endpoint coverage uses the existing fake transport.
   - **ARCH-CONSTRAINTS — pass:** constant additional state; overflow containment tested.
   - **ARCH-SECURE — pass:** malformed prefixes and cancellation preserve bounded recovery.
   - **ARCH-ORDER — pass:** private parser state transitions cover split input, interruption, termination, and reset.
   - **ARCH-FUNERAL — pass:** creates no durable runtime artifacts or background work.

7. **Plan revisions:** No behavioral revision needed. Add an explicit PURE/INTEGRATION kind column to the concept inventory when next updating it.

```findings
findings:
  - id: new
    severity: Minor
    family: benchmark-fixture-completeness
    title: |
      Imported parser benchmarks reference missing fixtures
    detail: |
      third_party/vt/ansiparser/parser_test.go:185 and :200 read fixtures/demo.vte and fixtures/UTF-8-demo.txt, neither included in the package. Running go test ./ansiparser -run '^$' -bench '^BenchmarkParser(UTF8)?$' -benchtime=1x fails for both. Include appropriately licensed fixtures or remove the unsupported benchmarks.
```

---

## Re-review — 2026-10-07T20:19:38-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 379 — Turn-end text flashes at focused pane cursor |
| repo | pair |
| issue file | workshop/issues/000379-stray-turn-end-text.md |
| boundary | whole-issue close |
| milestone | — |
| window | 6986321e216f9da591464e5991bcb03a5675fdf4..1b6ddc8b671b197cef033ac6eccffa3881f95259 |
| command | sdlc close --issue 379 |
| reviewer | codex |
| timestamp | 2026-10-07T20:19:38-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned changes address the demonstrated UTF-8 control-string leak through a shared parser correction. All three consumers are migrated, compatibility tests pass, and BR-1 is resolved by removing the unsupported benchmarks. No blocking findings.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      The pinned diff removes both fixture-dependent benchmarks and their os import. No fixture references remain in ansiparser; all retained benchmarks pass with -bench . -benchtime=1x.
```

1. **Strengths**
   - The production Endpoint regression checks unchanged screen cells and an intact notification across every split and bytewise delivery (`cmd/internal/terminal/notification_mapping_test.go:60`).
   - Tests cover all five string families, malformed prefixes, cancellation, reset, and bounded overflow (`third_party/vt/ansiparser/string_utf8_test.go:11`).
   - Comparison against upstream v0.11.7 confirms a narrow parser change; shared handler types and transition tables avoid unnecessary divergence.
   - Atlas and fork documentation explain ownership, provenance, and replacement criteria.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None remaining.

5. **Test coverage notes**
   - Passed nested vt/parser suites, including race detection.
   - Passed the full terminal package and focused wrapper observer/output-boundary tests.
   - Passed every retained parser benchmark and diff whitespace checks.
   - Private captured-incident replay and historical red runs were not independently repeated during this review.

6. **Architecture**
   - **ARCH-DRY — pass:** One correction shared across streaming consumers.
   - **ARCH-PURE — pass:** Deterministic parser logic tested without IO.
   - **ARCH-PURPOSE — pass:** Repairs the identified mechanism and related string families.
   - **ARCH-MOCK — pass:** No new external dependency; Endpoint regression uses the existing fake transport.
   - **ARCH-CONSTRAINTS — pass:** Constant-size UTF-8 tracking preserves existing payload bounds; overflow recovery is tested.
   - **ARCH-SECURE — pass:** Untrusted payload continuation bytes remain contained; malformed prefixes preserve control recovery.
   - **ARCH-ORDER — pass:** Private parser state advances through explicit transitions; reset, interruption, and fragmented input are exercised.
   - **ARCH-FUNERAL — pass:** No new durable runtime artifacts or background tasks.

7. **Plan revision recommendations:** None required. Finish the remaining close/publication checklist as delivery proceeds.

---

## Re-review — 2026-10-07T20:24:05-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 379 — Turn-end text flashes at focused pane cursor |
| repo | pair |
| issue file | workshop/issues/000379-stray-turn-end-text.md |
| boundary | whole-issue close |
| milestone | — |
| window | 48d5c69bb558617553dec00b41170e7ee2544683..c3bf44c9c3bf805adceef0b6b53a91041b7e8418 |
| command | sdlc close --issue 379 |
| reviewer | codex |
| timestamp | 2026-10-07T20:24:05-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned change addresses the demonstrated parser defect across the affected control-string families and migrates all three streaming consumers. Regression coverage confirms both containment and preserved notification delivery. No blocking findings; the repository remains unchanged.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Unsupported fixture-dependent benchmarks are removed. The retained self-contained benchmark and parser tests pass with -bench . -benchtime=1x.
```

1. **Strengths**
   - One shared parser correction covers OSC/DCS/SOS/PM/APC without notification-specific filtering.
   - UTF-8 prefix validation preserves standalone controls and bounded payload storage.
   - The production Endpoint regression checks unchanged screen cells and an intact notification across transport splits.
   - Licensed upstream provenance, atlas documentation, and a fork retirement path are retained.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage**
   - Passed full nested vt/parser suites, including race checks.
   - Passed the full terminal suite and focused Endpoint, control-observer, and output-boundary tests, including race checks.
   - Independently restored the original parser through an inspected, existing overlay: the Endpoint regression failed on notification-induced cell changes.
   - All retained parser benchmarks ran successfully; range whitespace checks passed.
   - The private captured-session replay was not independently repeated; the permanent production-path reproducer establishes the defect and repair.

6. **Architecture**
   - **ARCH-DRY — pass:** All three consumers share the correction; upstream handler types and transition table remain shared.
   - **ARCH-PURE — pass:** Parsing remains deterministic and directly testable without IO.
   - **ARCH-PURPOSE — pass:** Repairs the demonstrated cause and related string families; consumer search found no remaining original streaming-parser callers.
   - **ARCH-MOCK — pass:** No new external dependency; Endpoint coverage uses the existing fake IO seam.
   - **ARCH-CONSTRAINTS — pass:** Constant-size prefix tracking preserves existing storage limits; overflow recovery is tested.
   - **ARCH-SECURE — pass:** External bytes receive constrained UTF-8 prefix validation, with malformed-prefix recovery covered.
   - **ARCH-ORDER — pass:** Encapsulated parser transitions retain continuation state across feeds; cancellation, reset, and termination sequences are exercised.
   - **ARCH-FUNERAL — pass:** No new durable runtime artifacts or background tasks.
   
   Atlas changes cover the internal surface. No new commands, flags, configuration, or user workflows require a root README update.

7. **Plan revision recommendations:** None required. The implementation matches the documented repair scope.
