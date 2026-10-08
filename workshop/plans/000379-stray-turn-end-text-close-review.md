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
