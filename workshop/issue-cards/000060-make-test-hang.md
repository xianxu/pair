---
id: '000060'
status: done
created: 2026-06-14
updated: 2026-06-16
estimate_hours: 1
actual_hours: 0.78
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
