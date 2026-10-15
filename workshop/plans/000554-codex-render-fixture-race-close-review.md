# Boundary Review — pair#554 (whole-issue close)

| field | value |
|-------|-------|
| issue | 554 — Codex notification fixture races emulator Close against reader |
| repo | pair |
| issue file | workshop/issues/000554-codex-render-fixture-race.md |
| boundary | whole-issue close |
| milestone | — |
| window | 141e055b817765a50ab718224bc3d5c72b7769ff..a023e96e3143b7dd928e4cdeb5b9a1d8536e9ca9 |
| command | sdlc close --issue 554 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-15T00:21:48-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned change satisfies the fixture-lifetime repair: rendering finishes through the existing ordered teardown, cleanup errors propagate, and notification assertions remain intact. Focused and adjacent lifecycle tests passed with race instrumentation. No blocking defects found.

```findings
{}
```

1. **Strengths**

   - `cmd/internal/wrapcmd/codex_working_test.go:284` reuses `terminalModel.Close`, whose implementation closes the reply pipe, joins the drainer, then disposes the emulator.
   - `:273` bounds the combined render and cleanup wait by the lesser of ten seconds and the remaining test budget.
   - `:293` exercises real reply-producing rendering, exact screen contents, observable disposal, and shutdown-error propagation in both success and error cases.
   - Existing notification suppression assertions are preserved.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   - Passed notification suppression and both new lifecycle cases under `-race -count=10`.
   - Passed broader Codex and terminal-close tests under `-race`.
   - `git diff --check` passed.
   - Full-package suites, build, and SDLC suite were not independently rerun; their issue-log reports are not treated as measured review evidence.

6. **Architectural notes**

   - **ARCH-DRY — pass:** reuses established teardown and the existing `closeFunc` adapter.
   - **ARCH-PURE — pass:** lifecycle IO remains in the integration fixture; no business logic was added.
   - **ARCH-PURPOSE — pass:** both renderer call sites use the repaired helper, covering the complete affected fixture surface.
   - No new architectural or user-facing surface requires atlas or README changes. No Core concepts table is present.

7. **Plan revision recommendations:** None.

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 8.930412333s
head: "a023e96e3143b7dd928e4cdeb5b9a1d8536e9ca9"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
