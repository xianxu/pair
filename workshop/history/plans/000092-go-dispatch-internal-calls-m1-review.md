# Boundary Review — pair#92 (milestone M1)

| field | value |
|-------|-------|
| issue | 92 — route internal calls through Go dispatcher |
| repo | pair |
| issue file | workshop/issues/000092-go-dispatch-internal-calls.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 6ef5b578712991816414f546b2ad209feb2f1799^..HEAD |
| command | sdlc milestone-close --issue 92 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-07-01T12:02:11-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

All clean — vet passes, moved files properly relocated, only shims + build/exec integration tests remain in the `cmd/pair-*` dirs. I have everything I need.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

**Summary.** M1 delivers exactly its stated purpose — dispatcher reachability + runner consolidation, backward-compatible. The four helpers (`slug`, `changelog`, `continuation`, `session-watch`) are now reachable as `pair <sub>`, their logic is consolidated into shared `cmd/internal/<name>cmd` packages, the standalone binaries are genuine thin shims, and the buffered-vs-streaming split is implemented correctly (the I2 insight that `changelog` needs live stderr despite being no-stdin is honored). `go build ./...`, `go vet`, and every affected test package pass; the moved files are cleanly relocated with no forks left behind. Nothing here is a correctness bug or happy-path behavior drift, so the boundary is not blocked. The findings are an atlas-prose gap, plan-table signature mismatches, and a few test-coverage strengthenings — all cheap and non-blocking.

### 1. Strengths
- **Single-sourced routing (ARCH-DRY, the heart of the issue).** `DispatchNames`/`StreamingNames`/`IsImplemented`/`IsStreaming` all derive from the one `Families()` table (`dispatcher.go`), and `ClassifyInvocation` takes the reserved set as a parameter fed by `dispatcher.DispatchNames()` — the peel-off set is listed in exactly one place. Correctly resisted adding a parallel `Implemented bool` (kept it on `Status`).
- **Shims are real one-liners over shared runners** (`cmd/pair-slug/main.go`, `pair-changelog/main.go`, `pair-continuation/main.go`, `pair-session-watch/main.go`) — no divergent second implementation, exactly the Spec's ARCH-DRY intent.
- **Seam tests pin real behavior, not mocks.** `TestRunStreamingSubcommandRoutesContinuationStdin` proves stdin threads through (body read from `strings.NewReader` → rejected for missing NEXT ACTION), and `...RoutesChangelogToInjectedStderr` proves the live-stderr pass-through the buffered path would break. These test the thing that could actually regress.
- **The streaming guard in `Dispatch` (`dispatcher.go`)** gives a precise "streaming subcommand; invoke via the streaming seam" message instead of a misleading "not implemented" — good defensive design for a mis-call.
- **Missing-required-args exit path is byte-faithful** (exit 1 + `pair-changelog: usage:`), matching the old `fail()`.

### 2. Critical findings
None.

### 3. Important findings

**I1 — `atlas/architecture.md` dispatcher section is stale for the new surface (atlas gate).** `atlas/architecture.md:78-84` still says "As of #76, the same dispatcher also has the **first** implemented helper routes: `pair-go context` and `pair-go scrollback-render` … `bin/pair-title.sh` … have not moved to the dispatcher yet." After M1 this is materially incomplete: (a) four more implemented routes exist; (b) the **public `pair <sub>` peel-off** (`ClassifyInvocation` peeling reserved names off the public `pair`) is entirely new *architectural* surface, not just a disposition-table row, and is undescribed; (c) the **buffered-vs-streaming dispatch axis** (`runStreamingSubcommand`) is a new architectural distinction that belongs here. The implementer updated `atlas/go-migration-inventory.md` (good) but not the architecture prose. *Fix:* generalize the "first routes" sentence and add a short paragraph on the `pair <sub>` peel-off + the streaming seam (the detailed status can keep deferring to the inventory per architecture.md:94-95).

**I2 — Core Concepts table signatures contradict the code (plan cross-check → plan revision required).** Plan Integration-points table (`…-plan.md:52-53`) declares `slugcmd.Run(args, env, stdout, stderr) int` and `changelogcmd.Run(args, stdout, stderr) int`. The code ships `func Run() int` (`slugcmd.go`) and `func Run(args []string, stderr io.Writer) int` (`changelogcmd.go`). The entities exist at the stated paths with the stated status, so this is non-blocking, but per the Core-concepts directive a table/code contradiction needs a `## Revisions` entry so the plan stops claiming a surface the code doesn't expose. (`continuationcmd.Run(args, stdin, stdout, stderr, now) int` matches the table — good.)

**I3 — `slugcmd.Run()` dropped the planned dependency injection, leaving the orchestration untested (ARCH-PURE).** The plan (Task 3 Step 2) called for an injected `Env` + writers precisely so the runner's decision flow is unit-testable. The shipped `Run()` reads `os.Getenv`/`os.Getwd` and writes files directly, so `slugcmd_test.go` exercises only the pure helpers (`descendantPIDs`, `codexRolloutRE`, `resolveLiveCodexTranscript`) — the transcript-resolve → model → atomic-write path has **no** unit test. This is not a *regression* (old `pair-slug/main()` was identical), so it doesn't block M1, but it's a missed testability improvement the plan committed to. Either implement the injected `Env`/writers as planned, or revise the plan to record the decision to keep `Run()` env-driven (and lean on the equivalence test in I4 instead).

**I4 — Test coverage the plan committed to is missing / weakened.**
- The planned automated parity `pair-go slug ≡ pair-slug` (Task 3 Step 5, to extend `cmd/pair-go/helper_equivalence_test.go`) was **not** added — the harness still covers only `context`. Low real risk (both paths call the identical `slugcmd.Run()`), but the plan explicitly chose automated over manual verification (constitution §2). Add it or de-scope it in a revision.
- `TestDispatchNamesDeriveFromImplementedStatus` (`dispatcher_test.go`) is weaker than the planned `equalUnordered` assertion: it checks `context`/`scrollback-render` present and `launch`/`wrap`/`scribe` absent, but does **not** assert `slug`/`changelog`/`continuation`/`session-watch` are in the set. The peel-off routing depends on those being in `DispatchNames()`; if one were accidentally left `planned`, `pair changelog` would fall through to `ModePublicPair` (launch a session) and no unit test would catch it. *Fix:* assert the full implemented set as the plan specified.

### 4. Minor findings
- Flag-**parse** error path drifts from exit 2 → exit 1: `changelogcmd.run` and `continuationcmd.Run` use `flag.ContinueOnError` + `return 1`, where the old binaries used the default `flag.CommandLine` (`ExitOnError` → exit 2). Internal callers pass fixed valid flags so no caller/test is affected, but the Spec says "preserve exit codes." For `changelog` the error text also prints twice (once by `flag` via `SetOutput(stderr)`, once by `Run`'s `Fprintf`). `continuationcmd.Run` conversely prints *no* `pair-continuation:` prefix on a parse error (just `return 1`), relying on `flag`'s own output.
- `continuationcmd.run`'s push-failure warning writes `os.Stderr` directly rather than the injected `stderr` — fine in production (seam passes real `os.Stderr`) and pre-existing; the plan (Task 6 Step 2) already noted it as acceptable.

### 5. Test coverage notes
- Positive: `mode_test.go` covers the peel-off grammar thoroughly (agent names, launcher verbs, bare `pair`, `pair-go`); the four `runStreamingSubcommand` tests + `TestDispatchSlugRoutesToRunner` pin the routing; all relocated pure tests (`slug_test`, `distill_test`, `prompt_test`, `continuation_test`) move with their packages and pass.
- Gaps: see I3 (no `slugcmd.Run` orchestration test), I4 (no slug equivalence test; `DispatchNames` set under-asserted). No test asserts `DispatchNames() ⊇ {changelog, continuation, session-watch}` via the real function — `mode_test.go` passes a hand-built `names` slice, so it validates `ClassifyInvocation`'s logic but not that the real reserved set contains those names.

### 6. Architectural notes for upcoming work
- **ARCH-DRY: PASS.** No duplicated logic introduced; the reserved set is single-sourced; the streaming seam is ready for #96 (`wrap`/`scribe`) and #93 to reuse without an entrypoint edit (flip `Status`+`Streaming` in `Families()`).
- **ARCH-PURE: PASS with one note (I3).** Cores stay pure and injected; `runStreamingSubcommand` takes `stdin` as a param for testability. Only `slugcmd.Run` diverged from the injected-seam design.
- **ARCH-PURPOSE: PASS for M1's scoped purpose.** M1 is "reachability + consolidation, backward-compatible" — not the shadow-sweep, which is M2's gate. All four routes exist and old names still work. The `continuation`/`session-watch` routes have no production caller yet; the plan documents this as intentional symmetry for #93/#96, not dead code — the M2 review must run the actual shadow-sweep (`grep -rnE 'pair-(slug|changelog|continuation|context|scrollback-render|session-watch)'`) and confirm every Pair-owned caller derives from `pair <sub>` (plan Task 14).

### 7. Plan revision recommendations
Add a `## Revisions` entry to `workshop/plans/000092-go-dispatch-internal-calls-plan.md` recording:
1. **Runner signatures as-built** — `slugcmd.Run() int` (env-driven, no injected `Env`/writers) and `changelogcmd.Run(args []string, stderr io.Writer) int` (no `stdout`), correcting the Integration-points table (lines 52-53). If keeping `slugcmd.Run()` env-driven is deliberate, state the rationale (buffered route produces no captured output; behavior parity over injectability) so I3 isn't re-flagged later.
2. **Equivalence-test scope** — either the `pair-go slug ≡ pair-slug` harness case (Task 3 Step 5) is still owed, or it's de-scoped because both entry points call the identical `slugcmd.Run()`; record whichever.
3. **Flag error-handling** — note the intentional move to `flag.ContinueOnError` and the resulting exit-code change (2→1) on malformed flags, so the "preserve exit codes" Spec line isn't read as violated by a later reviewer.
