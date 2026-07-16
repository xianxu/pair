# Boundary Review — pair#96 (whole-issue close)

| field | value |
|-------|-------|
| issue | 96 — route pair-wrap and pair-scribe through dispatcher |
| repo | pair |
| issue file | workshop/issues/000096-route-pty-proxies-dispatcher.md |
| boundary | whole-issue close |
| milestone | — |
| window | b110a39fda999f58d4eeb22fbcfb7b27b800717a..HEAD |
| command | sdlc close --issue 96 |
| reviewer | claude |
| timestamp | 2026-07-01T14:33:40-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

This is a clean, well-executed mechanical repackaging that puts the last two PTY-proxy entrypoints (`pair-wrap`, `pair-scribe`) on #92's streaming dispatch seam. I verified the load-bearing claim — "byte-for-byte identical behavior" — the hard way: a reverse-transform diff of the new `cmd/internal/wrapcmd/wrap.go` and `cmd/internal/scribecmd/scribecmd.go` against the base `main.go` files shows that **every single delta is one of the planned mechanical transforms** (package rename, the 5 injected stdio fields, nil-guards for non-file readers, `writeStartupBanner` receiver, and `run() → (int, error)` return-code threading) — no hidden logic change slipped in. Dispatcher wiring is correct (family flip `planned→implemented`, pair-go gates on `IsImplemented && IsStreaming`, buffered `Dispatch` refuses with the "streaming subcommand … streaming seam" message), the build is green, and `wrapcmd`/`scribecmd`/`dispatcher`/`pair-go` tests all pass (including real-pty child-exit + route-equivalence tests). Nothing blocks SHIP; the findings below are all Minor/informational.

### 1. Strengths
- **Byte-faithful extraction, provably.** The reverse-diff (`wrap.go`, `scribecmd.go`) collapses onto the originals except for the exact planned transforms. This is the "load-bearing verification" the Spec demanded, and it holds.
- **Genuinely good route-equivalence tests** (`cmd/pair-go/pty_proxy_route_test.go:16`): they build both the standalone shim and `pair-go`, then assert the dispatch route matches the standalone binary **byte-for-byte** on exit/stdout/stderr — a real anti-mis-wire check, not a mock reasserting the implementation.
- **Clean IO seam** (`wrap.go:190-205`, `scribecmd.go` `Run`): the `io.Reader→*os.File` type-assert keeps production identical (real `os.Stdin` always asserts) while letting the terminal ops degrade to no-ops under a non-file test reader — improves ARCH-PURE without touching production behavior.
- **scribe gets its first-ever test coverage** (`scribecmd_test.go`), and the pty helpers self-skip on `os.IsPermission` so locked-down CI won't hard-fail.
- **Atlas updated thoroughly** in the same range: inventory contract rows, Coverage Ledger, the streaming-seam paragraph, the `pair-scribe` adjacent section, the `sendKeymapByAgent` file pointer, the how-to `file://` links, and the scribe README all repointed. Atlas gate satisfied.

### 2. Critical findings
None.

### 3. Important findings
None.

### 4. Minor findings
- **Subtle side-effect change on nonzero child exit (`wrap.go` `run`, ~line 2058).** Old code called `os.Exit(exitErr.ExitCode())` *inside* `run()` on a nonzero agent exit, which **skipped** the big `defer` block (tty restore, scrollback/events/wrap-events/adapt FD close+flush, `pair-wrap-pid-<tag>` / `agent-pid-<tag>` removal). The new `return exitErr.ExitCode(), nil` lets those defers run on **both** exit paths. This is benign-to-beneficial (the old skip-on-nonzero was a latent inconsistency; `bin/pair-shell` also cleans the pidfiles, so nothing depended on them surviving), and exit codes are unchanged — but it's a real observable difference not called out under the "byte-for-byte" claim. Worth one line in `## Log`/atlas so it reads as intentional, not an oversight.
- **`pair-scribe` standalone usage string changed** (`scribecmd.go` `fs.Usage`): from `usage: <program-path> -log …` (old `os.Args[0]`) to a fixed `usage: scribe -log …`. Cosmetic; note that the route-equivalence test only compares the two *new* paths to each other, not against the pre-extraction binary, so this delta isn't caught by a test (nor does it need to be).
- **`proxy.stderr` is stored but never read** (`wrap.go:199`). Dead field — kept for symmetry with `stdin`/`stdout`. Harmless; drop it or leave it.
- **Carried-over dead code:** `stdinDone := make(chan struct{})` / `close(stdinDone)` in `run()` is created but never awaited (pre-existing in the original; not introduced here).

### 5. Test coverage notes
- The 14 moved `wrapcmd` unit tests pin the *pure* logic (translateChunk, extractFG, OSC/overlay detection, span LRU, stdout filtering) directly, IO-free — that's where a real regression would show, and they're unchanged. Good.
- `Run`-level coverage is deliberately thin (arg/usage errors + child-exit-code propagation via real pty). Acceptable: `Run` is glue, and the parity tests cross-check it against the shim. The one untested behavior is the changed deferred-cleanup-on-nonzero-exit path (finding #4-a) — not worth a dedicated test given the shell-side belt-and-suspenders cleanup.

### 6. Architectural notes
- **ARCH-DRY — pass.** One implementation per proxy behind two entrypoints (shim + route); the extraction *removes* the future duplication risk rather than adding any. The minor pty/term boilerplate shared between `wrapcmd` and `scribecmd` is generic idiom, not domain logic — not worth a shared helper for two callers with radically different bodies (2359 vs 171 lines).
- **ARCH-PURE — pass.** Business logic was already pure functions with direct unit tests; the diff makes the IO seam (`Run`) injectable, strengthening the pure-core/thin-shell split.
- **ARCH-PURPOSE — pass.** Shadow-sweep of consumers: `wrap` → `bin/pair-wrap` shim (KDL PATH-exec) **and** `pair wrap` route, both deriving from `wrapcmd.Run`; `scribe` → `bin/pair-scribe` shim (`~/.zshrc`) **and** `pair scribe` route, both from `scribecmd.Run`; dispatcher families flipped to `implemented`. Keeping the KDL on the shim is intentional and documented, not a deferred purpose. Every consumer derives from the single source.

### 7. Plan revision recommendations
None required — the plan's Core-concepts/Steps match the delivered code. Optional: add a one-line `## Log` note recording the deferred-cleanup-on-nonzero-exit semantics change (finding #4-a) so the "byte-for-byte" claim is qualified accurately for future readers.
