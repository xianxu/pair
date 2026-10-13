---
id: 000512
status: open
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: 'dd76545319fe7b0a3a667482dbd45024096644fa' # card fields mirrored from issue-cards; edit via sdlc
---

# dev-prepare: keep the runtime bundle current cheaply and without a build race (pair half of ariadne#299)

## Problem

ariadne#299 (PR #210) lets a repo declare a `dev-prepare` make rule that dev-aliases runs (`make -s dev-prepare`, stdout to stderr) before `go build`. pair needs one: in a fresh pair slot the dev alias for `couch` fails to build because the generated runtime bundle (`cmd/internal/runtimebundle/assets/runtime`) doesn't exist yet. Today `runtimebundle-generate` is phony (always reruns, shells out to `tic`), and `runtimebundlegen.Generate` publishes with `RemoveAll(outRoot)` then `Rename(stage, outRoot)`, so a concurrent `go build` in another couch call in the same slot can see the embed tree missing.

## Spec

From ariadne:2's spec:
1. **A file target, cheap when current:** `dev-prepare` in pair's `Makefile.local` depends on `cmd/internal/runtimebundle/assets/runtime/manifest.json`, whose prerequisites are the generator inputs: `bin/pair-help`, `bin/pair-notify`, `doctor/README.md`, `SKILL.md`, `doctor.sh`, `perf.sh`, `emitter-health.sh`, everything under `bin/lib`, `nvim` and `zellij`, `terminfo/pair-vt-256color.ti`, and the generator sources (`cmd/internal/runtimebundlegen/*.go`, generatecmd, manifestmodel). Give dev-prepare its own non-phony path, or drop the phony prerequisite if `make build` and `make test` keep working. When up to date it runs nothing: no `go run`, no `tic`.
2. **No race with concurrent builds:** publish without a missing-tree window (rename the old tree aside, rename the new one in, then remove the old), or serialize dev-prepare with a slot-local lock (flock or mkdir). Add a test that builds while regenerating.
3. Keep make's stdout quiet (dev-aliases sends it to stderr anyway) and fail non-zero on generator errors.

## Done when

- In a freshly created pair slot, the dev-alias `couch --help` succeeds with no manual make (live check, which also closes ariadne#299's deferred check).
- A second call does no generation (timed or traced); after touching an input, the next call regenerates.
- The concurrent-build test passes and fails with the old publish.

## Plan

- [ ]

## Log

### 2026-10-12
