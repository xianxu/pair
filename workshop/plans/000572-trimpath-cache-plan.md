# Trimpath cache sharing implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach. Steps use checkbox syntax for tracking.

**Goal:** Build and test Pair with trimmed source paths by default, retaining fixture coverage and measuring compile-cache reuse across checkouts.

**Architecture:** Export additive repository-local Go flags from Makefile.local. Replace compiled-source lookup with a shared checkout resolver; test fixtures use cwd ancestry, while diagnostic probes have an explicit checkout input for outside-checkout launches. Declare the recorded suite explicitly, with the operator owning approval of the gate configuration.

**Tech Stack:** Go 1.26, Make, shell, SDLC local checks.

## Core concepts

| Pure entity | Lives in | Status |
|---|---|---|
| No new domain entity; existing Go flags and module identity are configuration values | Makefile.local, internal/reporoot | modified/new |

| Integration point | Lives in | Status | Wraps |
|---|---|---|---|
| Checkout discovery | internal/reporoot/root.go | new | cwd/ancestor filesystem, exact Pair go.mod identity |
| Probe asset checkout | probes/zellijprobe/zellijprobe.go | modified | explicit PAIR_PROBE_REPO or cwd discovery |
| Build/test default | Makefile.local | modified | exported GOFLAGS inherited by nested Go commands |
| Recorded suite | .sdlc/local-checks.json | new | main-owned SDLC suite command |

## Scope and choices

Prefer cwd/ancestor discovery to `go list` subprocesses per fixture lookup or embedding absolute roots. Validate the Pair module identity, not merely existence of an arbitrary go.mod. Test fixtures must not consult PAIR_HOME or live-slot environment, which could silently select a different checkout. Probes may use PAIR_PROBE_REPO because a trimmed binary outside a checkout cannot recover the original source directory. Update the two probe comments promising arbitrary-cwd discovery and document the explicit input. No live probe runs, global go env writes, cache wipes, scratch-volume changes, or live-slot changes.

ARCH-DRY: one filesystem discovery implementation shared by tests, recording fixtures, and the existing probe adapter. ARCH-PURE: discovery is a small IO shell; no new business state machine. ARCH-MOCK: test it with real temporary module trees, including wrong/missing modules, rather than mocked os calls. ARCH-ORDER: resolve assets before tests change cwd or children start; no global memoized root that could outlive cwd changes. ARCH-SECURE: explicit probe override is validated as a Pair checkout, never used as test-fixture authority. ARCH-CONSTRAINTS: preserve existing GOFLAGS (notably -p=3), use the machine suite gate, bound measurement to one small package and two disposable worktrees. ARCH-FUNERAL: remove owned measurement worktrees and isolated cache after retaining the results; never clear shared cache. ARCH-PURPOSE: sweep every runtime.Caller asset lookup, including probes affected by Make's exported flags; do not silently skip oracle tests.

## Tasks

### 1. Baseline and root resolution

- [x] Finish `GOFLAGS='-p=3 -trimpath' make -k test` on unchanged source and retain every failure. The -p value preserves this host's existing GOFLAGS; trimpath is added rather than resetting concurrency limits. If a prerequisite prevents Go tests, run the remaining Go suite with the same flags through the Make environment.
- [x] Add `internal/reporoot/root.go` with `Find(start string) (string, error)` walking ancestors to Pair's module declaration, and `Current() (string, error)` using cwd. Reject filesystem errors and outside-checkout roots explicitly. No compiled source path or global cache.
- [x] Add root tests for repository root, nested package, unrelated nested module, missing checkout, and explicit errors. Run with and without -trimpath.
- [x] Convert the ten affected test files under cmd/internal/{broadcast,checkpoint,couchcmd,launcher,reviewcmd,sessioninventory,terminal,webcast} to cwd-derived paths, preserving fail/skip semantics and resolving before t.Chdir. Convert recording/fixture.go, used across packages, to the same resolver with its existing fail-fast helper contract.
- [x] Add regressions proving recording fixtures resolve from multiple package depths, and missing oracle assets remain visible failures where required. Re-run every failing baseline package.

### 2. Probes and build defaults

- [x] Add a probe checkout adapter in probes/zellijprobe: PAIR_PROBE_REPO, when set, must resolve to a valid checkout; otherwise use cwd ancestry. An invalid override fails with an actionable error. Test both paths with temporary module trees.
- [x] Route the ten Caller-based probes under probes/ and cmd/probes/ through that adapter. Keep positional/re-exec protocols unchanged and pass already-resolved absolute assets to children. Compile all probes; do not run visible/live sessions.
- [x] Register new non-test source files in the artifact inventory and test the inventory package.
- [x] Add the default to Makefile.local via exported additive GOFLAGS, preserving caller flags and avoiding duplicate -trimpath. Allow an explicit Go -trimpath=false override if the caller needs diagnostic paths; test the precise Make semantics before selecting the final assignment.
- [x] Extend Make environment tests to observe default trimpath in nested Go invocations and preservation of an unrelated caller flag. Keep tests free of expensive recompilation where possible.
- [x] Add .sdlc/local-checks.json declaring required suite argv `["make", "test"]`, timeout 30 minutes, weight 1, so Make owns the flag value for both normal and recorded runs. If an explicit flag in recorded argv is required by the TL, use a dedicated Make variable instead of replacing GOFLAGS. Existing main currently uses the same implicit suite command.
- [x] Update atlas/index.md build/test notes, relevant probe comments/docs, and README development instructions. Explain that Pair's intentional per-checkout defaultPairHome linker value still differs; the claim is shared package compilation, not byte-identical installed binaries.

### 3. Measurement and qualification

- [x] Commit a stable candidate, create two detached temporary worktrees at that same commit, and run the same small real package through one isolated GOCACHE sequentially. Use module `go list -export -json` / build action IDs and compiler `-x` output to measure: without trimpath, worktree-specific cache exports; with trimpath, identical export cache path/build ID and no second compile. Retain Go version, flags, package, roots, entry sizes/counts, and commands in the issue Log. Do not use test-result caching as compile-cache evidence.
- [x] Warm dependencies only as needed; the package is small and the run is bounded. Clean up only the worktrees/cache created for this measurement after saving evidence.
- [x] Run make build and the full suite with defaults; preserve all failures and distinguish pre-existing ones from trimpath regressions. Use the recorded suite for final candidate verification where possible to avoid redundant full runs.
- [ ] Close through SDLC's fresh-context review; address findings. Open PR and tell the hub its exact URL/head and .sdlc delta for the operator's local-gates-verified label. Never apply that label ourselves.
- [ ] Once the operator-controlled gate and required checks pass, merge main and land via sdlc merge --yes. Report real merge hash and measurements to hub, then stop.

## Revisions

- 2026-10-15: TL approved 3a4eb910 with an exact recorded suite argv: `make PAIR_GO_TRIMPATH=-trimpath test`. The Make variable defaults to `-trimpath`; `PAIR_GO_TRIMPATH=` is the explicit diagnostic opt-out. Preserve existing GOFLAGS and avoid adding a duplicate flag. This supersedes the optional recorded-command/override alternatives above. Checkout discovery validates the nearest module and rejects foreign nested modules rather than borrowing enclosing assets.
