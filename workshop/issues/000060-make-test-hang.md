---
id: 000060
status: open
deps: []
github_issue:
created: 2026-06-14
updated: 2026-06-14
estimate_hours: 1
---

# full make test hangs as an aggregate run

## Problem

Discovered during #59: `make -f Makefile.local test` (the aggregate target) appears
to **hang / run far too long**, while the individual pieces pass fine when run
directly — `go test ./cmd/pair-wrap/ ./cmd/pair-scrollback-render/ ./cmd/pair-changelog/`,
`make test-statusline`, `make test-changelog`, and the e2e all green in seconds.

Two confounders to untangle:

1. **Measurement artifact (partial).** The first observation was a background run
   piped through `… 2>&1 | tail -20`. `tail` buffers until EOF, so the live output
   looked frozen at the `=== full make test ===` header even if `make test` was
   progressing. So "frozen output" ≠ "hung" on its own — but the operator also
   saw it run too long interactively, so there's likely a real stall too.

2. **Suspected real cause — build-in-test under parallel `go test ./...`.** The
   `cmd/pair-changelog` tests shell out to `go build` from inside tests
   (`buildBinary`, and #59's `buildRender` in `TestEndToEndMarkerSurvival`). The
   `make test` recipe ends with `go test ./...`, which compiles+runs packages in
   parallel — so several in-test `go build` invocations can run concurrently with
   (and under) the parent `go test`, contending on the Go build cache/lock. That
   can serialize badly or deadlock. #59 added a second in-test build, which may
   have tipped it over.

Not #59-feature code: the timestamp feature itself is verified (targeted suites +
e2e + a live `Alt+l` test all pass). This is test-infra hygiene.

## Spec

- Find the culprit deterministically: run each `make test` sub-target with a
  per-target timeout (`gtimeout`/a wrapper) to see which stalls; run `go test ./...`
  alone under a timeout to test the build-in-test hypothesis.
- If it's the in-test `go build` contention: build the shared binaries **once**
  (e.g. a `TestMain` that builds `pair-changelog` + `pair-scrollback-render` to a
  temp dir, or a `sync.Once`), or gate the build-heavy e2e behind `testing.Short()`
  so `go test ./... -short` (the aggregate) skips it while it still runs when the
  package is targeted directly. Prefer the build-once approach (keeps coverage).
- If it's a specific shell/lua sub-target: fix or document it.

## Done when

- `make -f Makefile.local test` completes reliably without hanging (timed, with
  visible streaming output — not piped through `tail`), and the root cause is
  fixed (not just worked around) or explicitly documented in the Makefile.

## Plan

- [ ] Bisect: per-target timeout run to identify the stalling target.
- [ ] Fix the root cause (likely build-once for the in-test `go build`s).
- [ ] Confirm `make test` runs green end-to-end, timed.

## Log

### 2026-06-14

- Filed from #59. Targeted suites + e2e + live test all green; only the aggregate
  `make test` stalls. Leading hypothesis: concurrent in-test `go build`
  (`buildBinary`/`buildRender`) under the recipe's parallel `go test ./...`.
  Note the `| tail` buffering confounder when reproducing — stream the output.
