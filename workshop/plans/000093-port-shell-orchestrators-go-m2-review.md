# Boundary Review — pair#93 (milestone M2)

| field | value |
|-------|-------|
| issue | 93 — port stateful shell orchestrators to Go |
| repo | pair |
| issue file | workshop/issues/000093-port-shell-orchestrators-go.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | c9d18ce9cc9a1de126c30245e35bd39d58f511b4..HEAD |
| command | sdlc milestone-close --issue 93 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-07-01T15:58:24-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

All remaining references are comments, the correct `explicitAssetPaths` build list, and the e2e tests. I found one real gap in the test wiring. Here's the review.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

This is a faithful, well-structured port of the two floating-pane launchers to Go. The load-bearing awk viewport-scorer, the #63 session keying, and the detached-distiller argv are extracted as pure functions and unit-tested; all zellij/nvim/exec/detach/fs sits behind a `Runtime` seam driven by a scriptable fake. I built both binaries and ran all three e2e smokes (`scrollback-open`, `changelog-open`, `changelog-session-key`) — all PASS driving the Go binaries — plus `go test ./cmd/internal/opener` (green) and `go vet` (clean). Nothing blocks the boundary. The one thing worth fixing before crossing: the changelog e2e — the process-level fake that validates this milestone's new Go detached distiller — can silently SKIP because `bin/pair` isn't a declared prereq of `test-changelog`; it only runs today because a sibling target happens to build it first.

### 1. Strengths
- **Faithful port of the awk scorer** (`cmd/internal/opener/opener.go:34` `matchViewport`) — index/candidate-start/consecutive-score/50%-threshold logic matches the shell line-for-line, with a *documented, deliberate* improvement: the tie-break now prefers the smaller start deterministically (the shell's awk map iteration order was not). Well-covered at `opener_test.go` (hit / sub-threshold reject / top-clamp / empty).
- **Clean ARCH-PURE separation** — pure decisions (`matchViewport`, `resolveSessionID`, `changelogBase`, `distillerEnv`, `distillerInner`, `stripSGR`) in `opener.go`; the full orchestration (`RunScrollback`/`RunChangelog`) is exercised without IO via `fakeRuntime` (`run_test.go`), including re-entrancy, no-scrollback, jump-threading, detached-distiller launch, skip-when-running, and config-key fallback.
- **ARCH-DRY reuse** — the Alt+/ opener renders in-process via `scrollbackcmd.Run` (`runtime.go:RenderScrollback`), dropping both the `bin/pair` subprocess *and* the jq dependency the shell needed; `procutil.Alive` backs the lock liveness checks; a single `distillerInner` const is shared.
- **Replace-in-place done correctly** — shell scripts `git rm`'d, the two `.gitignore` negations dropped, `git ls-files bin/` no longer lists them (verified), `explicitAssetPaths` + the `zellij/config.kdl` `Run "pair-…-open"` callers intact, and the pre-existing changelog e2e tests repurposed unchanged to drive the Go binary.
- **Atlas + lessons discipline** — both atlas files updated (architecture sections, migration-inventory rows, #93 M2 bullet) and a genuinely useful `lessons.md` rule about call-graph prose going stale.

### 2. Critical findings
None.

### 3. Important findings
- **`test-changelog` can silently skip the distiller e2e — missing `bin/pair` prereq** (`Makefile.local:237`). `tests/changelog-open-test.sh:14` does `[ ! -x "$PAIR_HOME/bin/pair" ] && SKIP`, and `RunChangelog` gates the detached distiller on `rt.Executable(pairHome+"/bin/pair")`. But the target's prereqs are `pair-changelog pair-scrollback-render pair-changelog-open pair-scrollback-open` — **not** `$(BIN_DIR)/pair`. So a clean `make test-changelog` no-ops silently, and this milestone's headline new behavior (Go `SysProcAttr.Setsid` detach) goes unverified. It only runs under `make test` because the sibling `test-continue: $(BIN_DIR)/pair` happens to build it just before — a fragile ordering dependency, and the stanza's own rewritten comment promises "no silent no-op." **Fix:** add `$(BIN_DIR)/pair` to the `test-changelog` prereq list. Cheap, non-blocking (mitigated in the standard `make test` path today).

### 4. Minor findings
- Error messages drop the shell's second explanatory line — missing-env loses "This script is meant to run inside a pair session."; no-scrollback loses "(capture starts when the agent pane begins emitting output.)" (`run.go` `RunScrollback`/`RunChangelog`). UX-only, floating pane self-closes after the sleep.
- `runtime.go:firstAgentPaneID` walks a decoded `map[string]interface{}` in Go's random iteration order vs jq's document order — a nondeterministic pane pick *if* >1 candidate ever exists. Safe under the two-pane invariant (draft excluded by title, viewers excluded by floating → exactly one match), so order is moot in practice; noting because determinism was deliberately restored in `matchViewport` but not here.
- `.viewport` write is no longer atomic: the plan specified `WriteAtomic` and the shell used temp+`mv -f`, but `OSRuntime.WriteFile` is a plain `os.WriteFile`. Safe here (synchronous, single writer, completes before nvim reads) — but it's a silent relaxation of a stated property.

### 5. Test coverage notes
- Strong pure + orchestration coverage; I ran it, all green. `resolveSessionID` even covers bad-JSON and no-session-id config.
- Small gap (acceptable): `overlayViewport`'s no-pane-skip and sub-threshold-fallback paths aren't asserted at the `Run` level — but `matchViewport`'s reject/empty are unit-tested and `scrollback-open-test.sh` exercises the no-zellij (no pane) path e2e, so the logic is covered.
- The scrollback smoke is self-sufficient (renders in-process; its SKIP guard is on the declared prereq `pair-scrollback-open`) — no silent-skip risk there, unlike the changelog one above.

### 6. Architectural notes for upcoming work (M3–M5)
- **OSRuntime fs-primitive duplication is trending (ARCH-DRY, forward-looking).** `opener`, `titlepoller`, and `sessionwatch` each define their own `OSRuntime` re-implementing the same trivial `ReadFile`/`WriteFile`/`Remove`/`FileSize`/`Executable`/`ProcessAlive`. M3 (review helpers), M4 (clipboard), M5 (launcher) will each add another. Before this replicates 5–6×, consider a shared `osfs`/`osseam` the per-package `OSRuntime`s embed for the boilerplate, keeping the domain-specific methods (`RenderScrollback`, `AgentPaneID`, `StartDetached`) per-package. Not a flag on this diff — the seam-per-package pattern is the #78 template — just the point where extraction starts paying.
- The re-entrancy-lock idiom (read → `ProcessAlive` → return 0; write pid; `defer Remove`) is duplicated across `RunScrollback`/`RunChangelog` with a *deliberate* timing difference (scrollback locks right before the viewer; changelog locks before the distiller). Fine now; if M3's `review-open` grows a third copy, extract a `withOpenLock` helper parameterized by lock timing.

### 7. Plan revision recommendations
- Add a `## Revisions` entry to `workshop/plans/000093-port-shell-orchestrators-plan.md` reconciling the M2 seam sketch with the shipped surface: `ListAgentPaneID`→`AgentPaneID`; `StartDistiller(argv, statusPath)`→`StartDetached(script, extraEnv, statusPath)`; and `WriteAtomic`/`Stat`→`WriteFile`(non-atomic)/`FileSize`. Call out the `WriteAtomic`→`WriteFile` change as an intentional relaxation (synchronous single-writer `.viewport`) with the rationale, so the plan stops claiming an atomic write the code doesn't perform.
