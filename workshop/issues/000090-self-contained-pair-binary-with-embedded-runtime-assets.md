---
id: 000090
status: done
deps: []
github_issue:
created: 2026-07-01
updated: 2026-07-01
estimate_hours: 5.44
started: 2026-07-01T00:18:42-07:00
actual_hours: 0.77
---

# self-contained pair binary with embedded runtime assets

## Problem

After #79, the public `pair` command is Go-owned, but deployment is still an
installed tree: the Go entrypoint must find adjacent Pair-owned runtime assets
such as `bin/pair-shell`, shell helpers, `nvim/`, `zellij/`, and helper
binaries. That is simpler for Homebrew, but it is not the deployment shape we
eventually want: copying one Pair binary around and having it work.

The long-term direction is a true native single binary. Rewriting every
remaining shell and orchestration surface directly into Go is too much risk in
one jump, so the next step should make the current runtime tree derive from one
Go artifact without pretending the shell lifecycle is already gone.

## Spec

Add a self-contained deployment mode for the Go `pair` binary:

- Embed the Pair-owned runtime assets needed by launch/session flows into the Go
  binary.
- On first run, extract those assets to a versioned runtime directory under the
  user's Pair data/cache area, then run the existing launch flow with `PAIR_HOME`
  pointed at that extracted runtime root.
- Preserve the current adjacent-install behavior for source checkout and
  Homebrew layouts unless/until the self-contained mode proves it can replace
  them.
- Keep external programs external: `zellij`, `nvim`, `fzf`, `jq` while the shell
  runtime still needs it, clipboard tools, and agent CLIs are not bundled by this
  issue.
- Make runtime extraction deterministic, idempotent, and upgrade-safe: a new
  binary/runtime version should extract a new directory or refresh only when the
  embedded manifest changes.
- Add a cleanup policy for stale extracted runtimes that cannot delete the
  currently running runtime.

Execution path toward the native single binary:

1. Embed and extract the existing runtime tree. This delivers the "single Pair
   artifact" deployment option while retaining the tested shell/nvim/zellij
   contracts.
2. Route generated internal calls through the Go dispatcher where possible
   (`pair wrap`, `pair slug`, `pair changelog`, `pair continuation`, etc.) while
   keeping compatibility names only as shims.
3. Port stateful shell orchestrators into Go one at a time: launcher/session
   lifecycle, scrollback/changelog openers, title poller, review helpers,
   clipboard helpers.
4. Once shell ownership is gone, stop extracting shell scripts and use embedded
   or generated native assets only for `nvim/` and `zellij/`.
5. Revisit whether `nvim/` and `zellij/` remain extracted native assets or move
   to generated temp files/API-driven startup. The native single binary target
   is one Pair executable, with external platform tools still supplied by the
   system.

Architecture notes:

- `ARCH-PURPOSE`: the copied binary must be enough to provide Pair-owned
  runtime assets; falling back to a source checkout does not satisfy this issue.
- `ARCH-DRY`: the embedded runtime manifest must be the single source of what is
  packaged, installed, and tested. Do not maintain a separate hand-written asset
  list for Homebrew, tests, and extraction.
- `ARCH-PURE`: keep manifest planning, runtime selection, and extraction
  decisions as pure functions with unit tests; keep filesystem writes and
  process exec in thin seams.

## Done when

- [x] A release build can produce one `pair` binary that contains the Pair-owned
      runtime assets needed for launch/session flows.
- [x] Copying only that binary to a clean path works when external dependencies
      are installed.
- [x] First run extracts or refreshes a versioned runtime root and points
      `PAIR_HOME` at it for the compatibility launch handoff.
- [x] Adjacent source/Homebrew layouts still work.
- [x] Upgrade and stale-runtime cleanup behavior is tested.
- [x] The execution path toward the true native single binary is documented in
      atlas.

## Plan

- [x] Define the embedded runtime manifest and generated asset list.
- [x] Implement runtime extraction and version/manifest selection.
- [x] Wire `cmd/pair-go` to prefer extracted embedded runtime when no adjacent
      asset root exists.
- [x] Add install/copy smoke tests for clean and upgrade paths.
- [x] Update README, atlas, and Homebrew packaging notes.

Detailed implementation plan:
`workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md`.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only. `sdlc estimate-source` reports the calibration
source as stale, so the number is provisional but uses the required method.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.20 impl=0.08
item: greenfield-go-module design=0.60 impl=0.56
item: smaller-go-module design=0.35 impl=0.48
item: cross-cutting-refactor design=0.80 impl=1.12
item: atlas-docs design=0.25 impl=0.20
item: milestone-review design=0.00 impl=0.20
design-buffer: 0.15
total: 5.44
```

## Log

### 2026-07-01
- 2026-07-01: closed — embedded runtime bundle verified by runtimebundle tests, drift check, build, PAIR_DATA_DIR copied-binary smoke, adjacent install smoke, full go test, issue validate, and diff check; review verdict: FIX-THEN-SHIP

Created after #79 closed: #79 made `pair` Go-owned but intentionally retained
the adjacent runtime tree. The desired final direction is a true native single
binary; this issue captures the lower-risk next step of embedding/extracting
the current runtime tree first.

Claimed and entered planning. `sdlc start-plan --issue 90` delivered
`ARCH-DRY`, `ARCH-PURE`, and `ARCH-PURPOSE`; the durable plan keeps the runtime
manifest as the packaging source of truth, pure planning/selection functions in
Go, and copied-binary launch as the acceptance path rather than a follow-up.

First `sdlc change-code --issue 90` plan-quality gate returned FAILURE: asset
boundary, generator/staleness contract, and copied-binary smoke were too loose.
Refined the durable plan to name exact runtime asset roots/exclusions, require a
deterministic gitignored generator plus drift check, exercise a fake
launch/session path, and bound Homebrew formula edits to false/conflicting
packaging claims only.

Second `sdlc change-code --issue 90` plan-quality gate returned FAILURE on
remaining precision issues: `bin/pair-title.sh` was referenced by smoke coverage
but missing from the required asset list, extracted runtime naming/version rules
were implicit, and atlas wording could imply a second source. Updated the plan
to include `pair-title.sh`, define `$PAIR_DATA_DIR/runtime/<digest>/pair-home`
plus marker/cleanup rules, and state that automated behavior derives only from
the generated manifest and runtime marker.

Third `sdlc change-code --issue 90` plan-quality gate returned FAILURE because
raw `go test ./cmd/internal/runtimebundle` would fail from a clean checkout once
`//go:embed` references the gitignored generated asset tree. Updated the plan to
add `make test-runtimebundle` as the generated-assets-before-test path after
`embed.go` exists, keep earlier pure tests as raw `go test`, and spell out the
peer-repo `AGENTS.local.md` / `MEMORY.md` requirement before any optional
Homebrew tap edit.

Implemented the embedded runtime path. Added the generated runtime manifest and
bundle generator, pure manifest/extraction/cleanup planning, embedded asset
reader, runtime extraction store, and `cmd/pair-go` fallback that extracts to
`$PAIR_DATA_DIR/runtime/<digest>/pair-home` only after `PAIR_HOME`, executable
sibling assets, and build-time `defaultPairHome` fail. Source/Homebrew adjacent
layouts remain first in the selection order.

Added copied-binary smoke coverage with fake external dependencies for `pair
--help`, `pair resume smoke`, required extracted assets, `PAIR_HOME` handoff,
and stale-runtime cleanup. During verification, parallel smoke runs exposed that
`runtimebundle-generate` rewrote the shared output tree in place; added a
regression test for preserving existing output on failed generation and changed
the generator to stage output in a unique temp directory before replacing the
published bundle (`ARCH-DRY`, `ARCH-PURE`, `ARCH-PURPOSE`).

Updated `README.md`, `atlas/architecture.md`, and
`atlas/go-migration-inventory.md` to document the implemented embedded fallback,
manifest ownership, cleanup behavior, and remaining external dependencies.
No Homebrew tap edit was needed because the adjacent `libexec` packaging path
remains accurate and intentionally precedes embedded fallback.

Verification passed:

- `make test-runtimebundle`
- `make runtimebundle-drift-check`
- `go test ./cmd/internal/entrypoint ./cmd/pair-go -count=1`
- `make build`
- `make test-pair-go-install-layout`
- `make test-pair-embedded-runtime`
- `go test ./... -count=1`
- `sdlc issue validate workshop/issues/000090-self-contained-pair-binary-with-embedded-runtime-assets.md`
- `git diff --check`

First `sdlc close --issue 90` boundary review returned REWORK. Fixed the
blocking findings by honoring `PAIR_DATA_DIR` before XDG/home fallback for
embedded extraction, adding divergent-env copied-binary smoke coverage,
updating the plan's `BuildManifest` path with a `## Revisions` entry, adding
`store.go` to `PAIR_GO_SRCS`, tightening stale-runtime cleanup to 64-character
manifest digests with matching markers, and reusing `runtimebundle` manifest
types from the generator. Also strengthened `runtimebundle-drift-check` to
compare generated file modes and recorded a new lesson for path precedence and
build-prerequisite coverage. A parallel verification run also exposed a
generator publish race; added an interprocess publish lock and concurrent
same-output generator regression test.

Review-fix verification passed:

- `go test ./cmd/internal/runtimebundle ./cmd/internal/runtimebundlegen -count=1`
- `go test ./cmd/pair-go -run 'TestRuntimeDataDir|TestRunDirectPairFallsBackToEmbeddedRuntime' -count=1`
- `make runtimebundle-drift-check`
- `make test-pair-embedded-runtime`
- `make test-runtimebundle`
- `go test ./cmd/internal/entrypoint ./cmd/pair-go -count=1`
- `make build`
- `make test-pair-go-install-layout test-pair-embedded-runtime`
- `go test ./... -count=1`
- `sdlc issue validate workshop/issues/000090-self-contained-pair-binary-with-embedded-runtime-assets.md`
- `git diff --check`

Second `sdlc close --issue 90` boundary review returned FIX-THEN-SHIP and the
close gate finalized the issue. Addressed the remaining important/minor review
items before committing the close: `runtimebundle` atomic writes now use unique
temp files so concurrent copied-binary first-run extraction cannot race on a
shared `.tmp` path, a concurrent `Extract` regression covers that case, and
`README.md` now documents the `PAIR_DATA_DIR`-first runtime path.
