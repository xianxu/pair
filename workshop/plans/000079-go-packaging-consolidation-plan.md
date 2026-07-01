# Go Packaging Consolidation Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the installed public `pair` command Go-owned while keeping native assets adjacent and preserving existing launcher behavior.

**Architecture:** Build `cmd/pair-go` into both `bin/pair` and `bin/pair-go`. When invoked as `pair`, the Go binary resolves an asset root and executes `<asset-root>/bin/pair-shell`; when invoked as `pair-go`, it keeps the explicit dispatcher surface and `pair-go launch` handoff. Asset root resolution is pure and ordered: explicit `PAIR_HOME`, then executable sibling root when `<dir>/pair-shell` exists, then build-time `defaultPairHome` injected by Make/Homebrew. `nvim/` and `zellij/` stay adjacent assets because they are native runtime files loaded by Neovim/Zellij and heavily tested in place.

**Tech Stack:** Go command entrypoint, Bash compatibility launcher, GNU/BSD Make install wiring, shell process-level tests, existing zellij/nvim adjacent assets.

---

## Core Concepts

### Pure Entities

| Name | Lives in | Status |
|------|----------|--------|
| `EntrypointMode` | `cmd/internal/entrypoint/mode.go` | new |
| `AssetRoot` | `cmd/internal/entrypoint/asset_root.go` | new |
| `LegacyPairRequest` | `cmd/internal/entrypoint/launch.go` | modified |

**EntrypointMode** — Determines whether one executable invocation should behave as public `pair` or development `pair-go`.
- **Relationships:** 1:1 with the executable basename; `cmd/pair-go/main.go` owns the argv/env IO and calls this pure classifier.
- **DRY rationale:** One classifier prevents `pair` and `pair-go launch` from growing parallel path-resolution rules (ARCH-DRY).
- **Future extensions:** If the shell launcher is later fully ported, this mode decision becomes the dispatch point for native launch instead of compatibility exec.

**AssetRoot** — Pure policy for choosing the root that owns adjacent runtime assets (`bin/pair-shell`, `nvim/`, `zellij/`).
- **Relationships:** N:1 from local source build, local copied install, and Homebrew `libexec` install into one root decision; `LegacyPairRequest` consumes the resolved root.
- **DRY rationale:** Prevents local install and Homebrew install from inventing separate path rules for the same asset layout (ARCH-DRY).
- **Future extensions:** Can add an extracted-embedded asset dir later without changing launcher request construction.

**LegacyPairRequest** — Describes the compatibility exec into the shell launcher.
- **Relationships:** N:1 from `pair` direct mode and `pair-go launch` mode into one request builder; each mode only changes display/diagnostic wording and argv shape. Carries the selected `AssetRoot` and computes `<asset-root>/bin/pair-shell`.
- **DRY rationale:** Keeps legacy shell handoff rules single-sourced while the actual zellij lifecycle remains shell-owned in this issue (ARCH-PURE).
- **Future extensions:** Can be deleted once shell launch is replaced by native Go launch.

### Integration Points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `PairEntrypointRuntime` | `cmd/pair-go/main.go` | modified | `os.Executable`, `os.Stat`, `syscall.Exec`, environment |
| `ShellLauncherShim` | `bin/pair-shell` | new | existing Bash `bin/pair` launcher behavior |
| `InstallLayout` | `Makefile.local` | modified | build/install filesystem layout |
| `HomebrewFormula` | `../homebrew-pair/Formula/pair.rb` | modified | Homebrew libexec/bin install layout |
| `InstallLayoutTest` | `tests/pair-go-install-layout-test.sh` | modified | temp HOME install and process execution |

**PairEntrypointRuntime** — Thin IO shell around the pure entrypoint mode/request logic.
- **Injected into:** `runWithLegacyRuntime` tests via the existing fake runtime shape; runtime supplies `os.Executable`, `PAIR_HOME`, build-time `defaultPairHome`, and stat probes for candidate roots.
- **Future extensions:** Can exec a native Go launch path instead of `bin/pair-shell` without changing tests for mode classification.

**ShellLauncherShim** — The existing Bash launcher, renamed to an internal compatibility target.
- **Injected into:** Go `pair` direct mode and `pair-go launch` handoff.
- **Future extensions:** Shrinks as launch behavior moves to Go; retained explicitly because zellij lifecycle, prompt UI, restart cleanup, title poller, and shell helper orchestration remain shell-owned after #78.

**InstallLayout** — Builds installed `pair` as a Go binary, keeps `pair-go` as the dev dispatcher alias, and keeps native assets source-tree-adjacent for local installs.
- **Injected into:** `make install`, local development and Homebrew-style source checkout installs.
- **Future extensions:** Asset embedding can replace adjacent asset install only after zellij/nvim callers derive from a virtual/extracted asset root.

**HomebrewFormula** — Installs the release layout under Homebrew `libexec` with `bin/`, `nvim/`, and `zellij/` adjacent.
- **Injected into:** `brew install pair` and `brew upgrade pair` through the sibling `xianxu/homebrew-pair` tap.
- **Future extensions:** If Pair later embeds assets, this formula stops installing `nvim/` and `zellij/` trees.

**InstallLayoutTest** — Process-level fake install under temp HOME.
- **Injected into:** `make test-pair-go-install-layout`.
- **Future extensions:** Add Linux smoke assertions if a Linux CI path becomes available.

## Chunk 1: Public Go Entrypoint Compatibility

### Task 1: Protect invocation-mode behavior

**Files:**
- Modify: `cmd/internal/entrypoint/launch.go`
- Create: `cmd/internal/entrypoint/asset_root.go`
- Create: `cmd/internal/entrypoint/asset_root_test.go`
- Create: `cmd/internal/entrypoint/mode.go`
- Modify: `cmd/pair-go/main_test.go`

- [ ] Add tests showing executable basename `pair` resolves to direct public launcher mode.
- [ ] Add tests showing executable basename `pair-go` with `launch` still resolves to explicit launch handoff.
- [ ] Add tests showing `pair-go` helper routes still dispatch without touching the shell launcher.
- [ ] Add pure tests for asset-root resolution:
  - `PAIR_HOME=/repo` wins when `/repo/bin/pair-shell` exists.
  - executable `/repo/bin/pair` resolves sibling root `/repo` when `/repo/bin/pair-shell` exists.
  - copied executable `/home/me/.local/bin/pair` falls back to build-time default root `/repo` when sibling shell is absent and `/repo/bin/pair-shell` exists.
  - missing sibling and missing build-time root produces a diagnostic naming `pair-shell` and `PAIR_HOME`.
- [ ] Run: `go test ./cmd/internal/entrypoint ./cmd/pair-go -count=1`
- [ ] Implement `EntrypointMode`, `AssetRoot`, and shared legacy request construction.
- [ ] Re-run: `go test ./cmd/internal/entrypoint ./cmd/pair-go -count=1`

### Task 2: Move shell launcher behind an internal compatibility name

**Files:**
- Move: `bin/pair` -> `bin/pair-shell`
- Delete tracked source: `bin/pair` (after `git mv`; future `bin/pair` is generated build output only)
- Modify: `.gitignore`
- Modify: `cmd/pair-go/main.go`
- Modify: `bin/pair-dev`

- [ ] Move the existing Bash launcher body to `bin/pair-shell`.
- [ ] Update `.gitignore`: remove the `!bin/pair` tracked-script exception and add `!bin/pair-shell`; `bin/pair` stays ignored as generated Go build output.
- [ ] Update Go direct `pair` mode to exec sibling `pair-shell` with argv[0] presented as `pair`.
- [ ] Update `pair-go launch ...` to exec sibling `pair-shell` with the same argv compatibility as before.
- [ ] Update `pair-dev` to export `PAIR_DEV=1` and exec sibling `pair` (the Go binary), not `pair-shell`, so dev mode exercises the public entrypoint.
- [ ] Run: `bin/pair-go launch --help` after build and confirm it reaches the launcher help.
- [ ] Run: `bin/pair --help` after build and confirm it reaches the same launcher help.

## Chunk 2: Build And Install Layout

### Task 3: Build `pair` from `cmd/pair-go`

**Files:**
- Modify: `Makefile.local`
- Modify: `tests/pair-go-install-layout-test.sh`

- [ ] Update `GO_BINS` so `pair` is a Go-built binary and `pair-go` remains built from the same package.
- [ ] Remove `pair` from `SHELL_BINS`; keep or explicitly drop `pair-dev` based on install behavior.
- [ ] Add a specific `$(BIN_DIR)/pair` build rule using `go build -ldflags "-X main.defaultPairHome=$(CURDIR)" -o $@ ./cmd/pair-go`.
- [ ] Keep `$(BIN_DIR)/pair-go` building from `./cmd/pair-go`; it may use the same `defaultPairHome` ldflag for copied local installs.
- [ ] Update install-layout test: installed `pair` must be executable and not a symlink; installed `pair-go` remains executable; `pair-dev` may remain a symlink if still a dev wrapper.
- [ ] Run: `make build`
- [ ] Run: `make test-pair-go-install-layout`

### Task 4: Adjacent native asset install layout

**Files:**
- Modify: `Makefile.local`
- Modify: `tests/pair-go-install-layout-test.sh`
- Modify: `../homebrew-pair/Formula/pair.rb`

- [ ] Keep `nvim/` and `zellij/` adjacent to `PAIR_HOME` for this issue; do not embed.
- [ ] Local `make install` remains source-tree based for native assets: installed `pair` is copied to `~/.local/bin`, and when it has no sibling `pair-shell`, `AssetRoot` falls back to build-time `defaultPairHome=$(CURDIR)` to find the repo checkout assets.
- [ ] Homebrew install remains `libexec`-adjacent: formula installs `bin/`, `nvim/`, and `zellij/` under `libexec`, then builds Go `pair`, `pair-go`, and required helper binaries into `libexec/bin` with `defaultPairHome=#{libexec}`.
- [ ] Update formula comments and built-binary list so Homebrew surfaces `bin/pair` as the Go-built public command and retains `bin/pair-shell` only as an internal compatibility launcher.
- [ ] Test that local installed `pair --help` reaches the shell help through the Go entrypoint.
- [ ] Run: `make test-pair-go-install-layout`

## Chunk 3: Compatibility Shim Inventory

### Task 5: Retain/remove shims intentionally

**Files:**
- Modify: `atlas/go-migration-inventory.md`
- Modify: `README.md`
- Modify: `atlas/architecture.md`
- Modify: `CHANGELOG.md` if the Homebrew/release note wording needs a packaging entry.
- Modify: `../homebrew-pair/Formula/pair.rb`

- [ ] Document `bin/pair-shell` as a retained compatibility launcher and explain why it is not obsolete yet.
- [ ] Document `pair-dev` as retained dev-mode wrapper that runs the Go public `pair`.
- [ ] Document legacy helper binaries retained because native zellij/nvim/shell callers still reference them.
- [ ] Remove stale wording that says `pair-go launch` is the only Go-owned launch test surface; installed `pair` is now the public Go-owned entrypoint.
- [ ] Update Homebrew wording: formula comments and README/CHANGELOG must say Homebrew installs a Go-built `pair` plus adjacent native assets under `libexec`.

## Chunk 4: Verification And Closure

### Task 6: End-to-end verification

**Files:**
- Modify: `workshop/issues/000079-go-packaging-consolidation.md`
- Modify: `workshop/plans/000079-go-packaging-consolidation-plan.md`

- [ ] Run focused Go tests: `go test ./cmd/internal/entrypoint ./cmd/pair-go -count=1`.
- [ ] Run packaging tests: `make build && make test-pair-go-install-layout`.
- [ ] Run an upgrade-layout test in `tests/pair-go-install-layout-test.sh`: seed `~/.local/bin/pair` as the old symlink-to-source-shell layout, run `make install`, then assert `~/.local/bin/pair` is now a regular executable Go binary, `bin/pair-shell` remains tracked/executable under the source root, `PAIR_HOME` override works, and default-root fallback lets installed `pair --help` reach the shell help.
- [ ] Run a Homebrew formula dry-run/smoke if available: `brew test --formula ../homebrew-pair/Formula/pair.rb` or record the exact local blocker; at minimum run `ruby -c ../homebrew-pair/Formula/pair.rb`.
- [ ] Run launcher smoke: `bin/pair --help`, `bin/pair-go launch --help`, `bin/pair-dev --help`.
- [ ] Run broader impacted tests: `make test-dev-rebuild test-session-watch test-continue`.
- [ ] If practical, run Linux smoke with the available local toolchain; otherwise record why it was not available.
- [ ] Update issue checklist/log with verification evidence.
- [ ] Update atlas/README and run stale-doc grep for old packaging statements.

## Implementation Notes

- This plan deliberately chooses adjacent assets over embedding. `nvim/` and `zellij/` are native runtime surfaces loaded by their own tools; embedding would require extraction or virtual path rewrites across many tested seams. Adjacent assets satisfy #79 now with lower risk and preserve direct edit/test loops.
- Git outcome is explicit: `bin/pair-shell` is tracked source, `bin/pair` is generated Go build output and ignored. The existing blanket `bin/*` ignore remains; the tracked-script exception changes from `!bin/pair` to `!bin/pair-shell`.
- Asset-root outcome is explicit: pure `AssetRoot` chooses `PAIR_HOME`, sibling executable root, or build-time `defaultPairHome`; runtime only probes filesystem existence and execs the resulting shell path.
- `ARCH-PURPOSE`: #79 is not complete if only docs change; installed `pair` must become the Go-owned public command.
- `ARCH-DRY`: direct `pair` and `pair-go launch` must share one compatibility request builder.
- `ARCH-PURE`: mode selection and request construction stay pure; filesystem/exec behavior stays in the `cmd/pair-go/main.go` runtime seam.
