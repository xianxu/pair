---
id: 000090
status: working
deps: []
github_issue:
created: 2026-07-01
updated: 2026-07-01
estimate_hours: 5.44
started: 2026-07-01T00:18:42-07:00
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

- [ ] A release build can produce one `pair` binary that contains the Pair-owned
      runtime assets needed for launch/session flows.
- [ ] Copying only that binary to a clean path works when external dependencies
      are installed.
- [ ] First run extracts or refreshes a versioned runtime root and points
      `PAIR_HOME` at it for the compatibility launch handoff.
- [ ] Adjacent source/Homebrew layouts still work.
- [ ] Upgrade and stale-runtime cleanup behavior is tested.
- [ ] The execution path toward the true native single binary is documented in
      atlas.

## Plan

- [ ] Define the embedded runtime manifest and generated asset list.
- [ ] Implement runtime extraction and version/manifest selection.
- [ ] Wire `cmd/pair-go` to prefer extracted embedded runtime when no adjacent
      asset root exists.
- [ ] Add install/copy smoke tests for clean and upgrade paths.
- [ ] Update README, atlas, and Homebrew packaging notes.

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
