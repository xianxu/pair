# Boundary Review — pair#92 (whole-issue close)

| field | value |
|-------|-------|
| issue | 92 — route internal calls through Go dispatcher |
| repo | pair |
| issue file | workshop/issues/000092-go-dispatch-internal-calls.md |
| boundary | whole-issue close |
| milestone | — |
| window | be90754054f437dd1f44dc772acfe5965bf31a33..HEAD |
| command | sdlc close --issue 92 |
| reviewer | claude |
| timestamp | 2026-07-01T12:29:27-07:00 |
| verdict | SHIP |

## Review

I have everything I need. Let me compile the review. Summary of independent verification performed:

- `go build ./...`, `go vet ./cmd/...` — clean (exit 0)
- `go test ./cmd/...` — all packages pass (including new `slugcmd`, `changelogcmd`, `continuationcmd`, and the `pair-go` seam tests)
- `tests/pair-title-poller-test.sh` (drives `pair context`) and `tests/changelog-open-test.sh` (drives `pair scrollback-render` + `pair changelog` through the real `bin/pair`) — PASS
- `nvim -l nvim/scrollback_test.lua` + `changelog_test.lua` — PASS
- Independent ARCH-PURPOSE shadow-sweep — every remaining `pair-<name>` hit is a comment, the deferred `pair-shell → pair-session-watch.sh` chain (#93), or the gitignored bundle mirror
- Verified `bin/pair` is built from `./cmd/pair-go`, assets tree is gitignored (0 tracked), `bin/` is tracked, working tree clean

```verdict
verdict: SHIP
confidence: high
```

**Summary.** pair#92 delivers its stated purpose cleanly: the four internal-call helpers (`slug`, `changelog`, `continuation`, `session-watch`) are now reachable as `pair <sub>` through the dispatcher, each standalone `bin/pair-<name>` binary is a genuine one-line shim over a shared `cmd/internal/<name>cmd` runner (ARCH-DRY), and all five Pair-owned call-sites are repointed. The buffered-vs-streaming split is implemented correctly (the subtle I2 insight that `changelog` needs live stderr despite having no stdin is honored, and I independently confirmed the Alt+l spinner path survives via the passing end-to-end test). Both milestone boundary reviews (M1, M2) already ran FIX-THEN-SHIP and their findings were addressed in the diff — atlas prose updated, plan `## Revisions` records the as-built signatures, `DispatchNames` assertion strengthened, equivalence test added, stale comments fixed, viewer test gap closed. Whole-issue verification (build/vet/all Go tests/two shell suites through the real binary/two lua suites/shadow-sweep) passes with a clean tree. Nothing blocks the boundary; the residual items below are all Minor and already documented-and-accepted in the plan.

**1. Strengths**
- **Single-sourced routing (ARCH-DRY, the heart of the issue).** `DispatchNames`/`StreamingNames`/`IsImplemented`/`IsStreaming` all derive from the one `Families()` table (`dispatcher.go:35-94`); `ClassifyInvocation` takes the reserved set as a parameter fed by `dispatcher.DispatchNames()` (`main.go:54`) — the peel-off set is listed exactly once. Correctly resisted a parallel `Implemented bool` (kept it on `Status`).
- **Shims are real one-liners** (`cmd/pair-slug/main.go`, `pair-changelog/main.go`, `pair-continuation/main.go`, `pair-session-watch/main.go`) over the shared runners — no divergent second implementation. I diffed the moved cores: `distill.go`/`prompt.go`/`slug.go`/`continuation.go` relocated byte-faithfully with only the `package` line changed.
- **Seam tests pin behavior that could actually regress, not mocks.** `TestRunStreamingSubcommandRoutesContinuationStdin` proves stdin threads through (body read → rejected for missing NEXT ACTION), and `...RoutesChangelogToInjectedStderr` proves the live-stderr pass-through the buffered path would break (`cmd/pair-go/main_test.go`).
- **The `Dispatch` streaming guard** (`dispatcher.go:121-129`) gives a precise "streaming subcommand; invoke via the streaming seam" message instead of a misleading "not implemented" for a mis-call — good defensive design.
- **Call-site repointing is lockstep-safe**: `pair-scrollback-open:70-76` updates both the `-x` guard and the invocation to `$PAIR_HOME/bin/pair`, so the "not built" diagnostic can't drift from the real dependency.

**2. Critical findings**
None.

**3. Important findings**
None. (The M1/M2 Important finding — stale `atlas/architecture.md` prose — was addressed: `atlas/architecture.md:78-107` now describes the four routes, the public `pair <sub>` peel-off, and the buffered-vs-streaming seam; `atlas/go-migration-inventory.md` carries the #92 M1/M2 rows; `atlas/index.md:7-8` links both. Atlas gate satisfied.)

**4. Minor findings** (all already recorded in the plan `## Revisions` as accepted as-built decisions — noted for completeness, no action required)
- Flag-**parse** error exit-code drift 2→1: `changelogcmd.run`/`continuationcmd.Run` use `flag.ContinueOnError`+`return 1` where the old binaries defaulted to `ExitOnError` (exit 2). Unreachable by internal callers (fixed valid flags). Documented, plan Revision M1§3.
- `changelogcmd` prints its flag-parse error twice (flag's own `SetOutput(stderr)` + `Run`'s `Fprintf`); `continuationcmd.Run` conversely prints no `pair-continuation:` prefix on a parse error. Cosmetic, never-exercised path.
- `bin/pair-title.sh:105` uses bare `pair context` (PATH-resolved via `pair-shell`'s `$PAIR_HOME/bin` prepend) while the other four call-sites use explicit `$PAIR_HOME/bin/pair`. Parity with the pre-existing bare `pair-context`. Documented, plan Revision M2§2.
- Observation (not a repo defect): the **gitignored** bundle mirror `cmd/internal/runtimebundle/assets/.../bin/pair-title.sh:94` still carries the old `pair-context` comment though the tracked source is fixed — a stale regen artifact that refreshes on the next `make build`; it can't dirty the tree at close (gitignored) or ship (regenerated from the fixed source).

**5. Test coverage notes**
- Well covered: `pair context` (title-poller test), `pair scrollback-render`+`pair changelog` (changelog-open end-to-end through the real binary), `pair slug` command construction (`slug_spawn_test.go`), the streaming seam routing (4 `runStreamingSubcommand` tests), `ClassifyInvocation` grammar (`mode_test.go` — agent names, launcher verbs, bare `pair`, `pair-go`), and `nvim` `renderer_command` invocation form.
- Accepted gaps (documented): `slugcmd.Run()` orchestration is exercised via the relocated `cmd/pair-slug` build+exec integration tests rather than a unit test (env-driven by deliberate decision, plan Revision M1§1); `pair continuation`/`pair session-watch` have no full `bin/pair`-level integration test, but the shim integration tests + seam unit tests cover both the runner and the routing.

**6. Architectural notes for upcoming work**
- **ARCH-DRY — PASS.** No duplicated logic; the reserved set is single-sourced; the streaming seam is ready for #96 (`wrap`/`scribe`) and #93 to reuse by flipping `Status`+`Streaming` in `Families()` with no entrypoint edit. The one cross-language "resolve the pair binary" pattern (Go/shell/lua) is an unavoidable language-boundary repetition, single-sourced within Go in `slugSpawnCmd`.
- **ARCH-PURE — PASS.** Pure cores stay pure and unit-tested without IO; runners are thin IO seams; `runStreamingSubcommand` takes `stdin` as a param for testability. `slugcmd.Run()`'s direct env/fs reads are the only impurity, deliberate and integration-tested.
- **ARCH-PURPOSE — PASS.** All five Done-when items delivered; shadow-sweep independently reproduced. `pair session-watch`/`pair continuation` routes without a repointed production caller are honest Spec-scoped deferrals (session-watch's caller is the shell launcher → #93; continuation is agent-procedure-invoked), documented in `## Log` — symmetry for reuse, not silent under-delivery. The Spec's carve-out of `pair-wrap`/`pair-scribe` to #96 is consistent.

**7. Plan revision recommendations**
None. The plan's two `## Revisions` entries (M1 as-built, M2 as-built) already reconcile every table/code drift the milestone reviews surfaced (runner signatures, equivalence-test scope, flag error-handling, `slugSpawnCmd` signature, `pair-title.sh` invocation form, runtime-bundle gitignore, viewer test gap). The Core Concepts / Integration-points tables no longer claim surfaces the code doesn't expose.
