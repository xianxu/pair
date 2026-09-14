# Boundary Review — 000207-couch-s-asserted-mouse-mode-has-no-release-path-short-of-restarting-couch#207 (milestone M1)

| field | value |
|-------|-------|
| issue | 207 — couch's asserted mouse mode has no release path short of restarting couch |
| repo | 000207-couch-s-asserted-mouse-mode-has-no-release-path-short-of-restarting-couch |
| issue file | workshop/issues/000207-couch-s-asserted-mouse-mode-has-no-release-path-short-of-restarting-couch.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | e993b68d1a886f89898e8ac089b27717f6090a8c..ff1b0c54247c861dde356e08fa23bac8e0820713 |
| command | sdlc milestone-close --issue 207 --milestone M1 |
| reviewer | codex |
| timestamp | 2026-09-14T09:22:11-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned range delivers the approved M1 diagnostic scope without changing mouse ownership policy or terminal write sequences. Producer tests and race checks pass. No blocking findings; the original recovery acceptance remains explicitly pending M2.

1. **Strengths**

   - `console.go:1243` preserves the existing write gate while exposing deferred, partial, successful, and failed outcomes.
   - `mousetrace_test.go:166` uses a scheduling barrier to verify attribution survives an active-thread change during blocked output.
   - `mousetrace_test.go:208` exercises actual switching and replay across distinct thread identities.
   - `atlas/couch.md:1315` accurately distinguishes scanner belief, accepted bytes, and terminal state.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** The plan’s core-concepts table lacks explicit PURE/INTEGRATION classifications; see the finding below.

5. **Test coverage notes**

   Passed:
   - `go test ./cmd/internal/couchtty ./cmd/internal/couchcmd`
   - `go test -race ./cmd/internal/couchtty -run 'MouseTrace|MouseWriteResult|FormatMouseModes' -count=1`
   - Pinned-range `git diff --check`

   Coverage exercises real producers with captured bytes and controlled write failures. No live recovery verification or mutation testing was performed. No prior findings require disposition.

6. **Architectural notes**

   - **ARCH-DRY — pass:** Shared context, outcome formatting, write gate, and trace sink.
   - **ARCH-PURE — pass:** Outcome formatting is directly tested without IO; producer integration remains in Console.
   - **ARCH-PURPOSE — pass:** Covers the approved diagnostic producers; M2 recovery remains openly unresolved.
   - **ARCH-MOCK — pass:** Production and tests share the Host boundary; tests control accepted bytes and scheduling.
   - **ARCH-CONSTRAINTS — pass:** Bounded metadata, opt-in tracing, and documented capture estimates; no new workers.
   - **ARCH-SECURE — pass:** Free-form fields are bounded and quoted; replay bodies are excluded.
   - **ARCH-ORDER — pass:** Pre-write attribution has deterministic ordering coverage; documentation explicitly disclaims global writer ordering.
   - **ARCH-FUNERAL — pass:** Existing teardown closes the sink; temporary capture ownership and operator cleanup are documented.

   Atlas covers the changed diagnostic format. No new command, flag, keybinding, or configuration key was introduced requiring a README update in this range.

7. **Plan revision recommendation**

   Add a `## Revisions` entry consolidating the entity tables with a Kind column: `mouseTracer` and `Console` are INTEGRATION; `mouseWriteResult` is PURE. Referenced entities and modifications otherwise match the code.

```findings
findings:
  - id: new
    severity: Minor
    family: core-concept-classification
    title: |
      Add explicit kinds to the core-concepts table
    detail: |
      workshop/plans/000207-mouse-diagnostics-plan.md:11 omits the required PURE/INTEGRATION column, while mouseWriteResult appears in a separate table at line 116. Consolidate these classifications in a greppable table and record the clarification under Revisions (ARCH-PURE); this is a documentation omission, not a demonstrated purity violation.
```
