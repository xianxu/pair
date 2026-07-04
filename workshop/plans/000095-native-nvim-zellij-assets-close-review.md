# Boundary Review — pair#95 (whole-issue close)

| field | value |
|-------|-------|
| issue | 95 — native nvim and zellij startup assets |
| repo | pair |
| issue file | workshop/issues/000095-native-nvim-zellij-assets.md |
| boundary | whole-issue close |
| milestone | — |
| window | 9d3a1193ca2ae5c144960214c3f198540c78efa6..HEAD |
| command | sdlc close --issue 95 |
| reviewer | claude |
| timestamp | 2026-07-04T00:01:25-07:00 |
| verdict | SHIP |

## Review

Confirmed: `LaunchNative` is only reached after `ResolveAssetRoot` succeeds, so `opts.PairHome` is always a real resolved root — no empty-path edge for the prepend. My review is complete.

```verdict
verdict: SHIP
confidence: high
```

This boundary delivers exactly what issue #95 committed to: a documented decision (keep the digest-versioned extraction, reframed as a content-addressed runtime *cache*, with the residual zero-tree gap honestly recorded) plus the one substantive code change — restoring the `$PAIR_HOME/bin` PATH-prepend the Go launcher dropped in #99 M5c, which silently broke session launch for copied/Homebrew installs. The implementation is a clean pure helper (`prependBinToPath`) wired through a single `SetEnv` at `RunLaunch` entry, and I independently verified every load-bearing claim: the prepend targets the *resolved asset root's* bin/ (not the binary dir), `RunLaunch` genuinely is the sole zellij-spawning path (both `LaunchSession` and `AttachSession` are reachable only through its loop), the augmented PATH flows to zellij via `os.Environ()` inheritance, and the smoke guard bites because the stub's own PATH excludes `$root/bin`. Tests pass (unit 5/5, fake wiring, full launcher package, copied-binary smoke). Nothing blocks SHIP.

### 1. Strengths
- **Textbook pure-core/thin-shell split** (`pathenv.go:18` + `createflow.go:33`): the entire PATH logic is a deterministic string function unit-tested without IO, injected via the one-line `SetEnv` seam — ARCH-PURE exemplar.
- **Single prepend point is provably sufficient** — I verified `LaunchSession` (`createflow.go:300`) and `AttachSession` (`lifecycle.go:40`) are the only zellij session spawns and both flow through `RunLaunch`; placing the prepend at entry (line 33, before the create/attach branch and the restart loop) covers create/attach/resurrect/restart with zero duplication (ARCH-DRY).
- **The smoke is a real regression guard, not a vacuous pass** (`tests/pair-embedded-runtime-test.sh:59-60,129`): launching with `PATH="$bin_dir:/usr/bin:/bin"` (no `$root/bin`) means `command -v copy-on-select` resolves *only* via the launcher's prepend — the test would exit 21 without the fix, as the plan verified.
- **Correct test placement rationale**: wiring tested through the fake `SetEnv` (records into `f.env`) rather than `LaunchNative`'s real `OSRuntime`, avoiding process-env pollution (`createflow_test.go:272`).
- **Doc shadow-sweep is complete**: all live PATH-prepend claims (atlas ×2, `zellij/config.kdl:39` + its regenerated bundle copy, `go-migration-inventory.md`, and the peer `homebrew-pair/Formula/pair.rb:30-34`) now name the Go launcher/#95; remaining `workshop/history/*` hits are archived past-tense narrative.

### 2. Critical findings
None.

### 3. Important findings
None.

### 4. Minor findings
- `pathenv.go:24` — when `binDir` appears *later* in PATH (not first), the function prepends anyway, leaving a duplicate entry. Documented as harmless (first match wins) and covered by the `prepends even if present later` test; intentional, note only.
- `createflow_test.go:279` — the wiring test reads the real ambient `os.Getenv("PATH")` through production code. The assertion is robust to any ambient value (empty → `/pair/bin`; non-empty → `/pair/bin` prefix), so no flakiness, but it's a slight test impurity worth remembering if PATH assertions ever tighten.

### 5. Test coverage notes
- Prepend logic (5 cases), wiring (fake), and end-to-end bare-name resolution (copied-binary smoke) are all covered — this is the exact bug class the diff could ship, and it's caught.
- Source/Homebrew layouts are covered *by argument* (the prepend is layout-agnostic — it prepends whatever `ResolveAssetRoot` returned) rather than a dedicated end-to-end smoke. The argument is sound (those layouts differ only in the resolved root, which the unit/wiring tests exercise generically), and the copied-binary case is the one that was actually broken. Acceptable; noting only that a source/Homebrew smoke doesn't exist if a future regression touched root resolution.

### 6. Architectural notes for upcoming work
- **ARCH-DRY — pass.** One source of truth for the prepend; replaces the retired shell's open-coded inline `export PATH`. No duplication introduced; the single call site is the correct consolidation.
- **ARCH-PURE — pass.** `prependBinToPath` is genuinely pure (uses only `filepath.Join` + the `os.PathListSeparator` package var, no syscalls) and tested without IO; the IO is confined to the injected `SetEnv` seam.
- **ARCH-PURPOSE — pass.** The issue's purpose was to *decide + document* the asset-delivery endpoint and reach (or consciously document the gap to) the native single binary. The diff fulfills both halves: the decision is recorded with weighed/rejected alternatives, the residual zero-tree gap is documented as an accepted limitation (permitted by ARCH-PURPOSE), and the folded-in PATH regression sits directly on Done-when #2 ("works across copied/source/Homebrew") — declared scope per the Log, not creep. Shadow-sweep of the "who prepends PATH" fact confirms every consumer (atlas, config.kdl + generated copy, inventory, peer formula) now derives from/names the Go launcher; no stale restatement survives.

### 7. Plan revision recommendations
None — the plan's Core Concepts table (`prependBinToPath` PURE/new; PATH export at `RunLaunch` modified; smoke modified) matches the code exactly, and every checklist item is delivered. No `## Revisions` entry needed.
