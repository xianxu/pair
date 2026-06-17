---
id: 000060
status: done
deps: []
github_issue:
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

- [x] Fix root cause: `tests/autopair-test.sh` driver `qall` → `qall!`.
- [x] Add a shared timeout watchdog (`tests/lib/run-headless.sh`) so a stuck
      headless nvim fails loud instead of hanging the suite; route all 4
      headless-nvim tests through it.
- [x] Audit + fix the latent `qall` in the other 3 drivers (queue-send,
      statusline-pos, changelog-notify).
- [x] Confirm `make test` runs green end-to-end, timed, with streamed output.

## Log

### 2026-06-14

- Filed from #59. Targeted suites + e2e + live test all green; only the aggregate
  `make test` stalls. Leading hypothesis: concurrent in-test `go build`
  (`buildBinary`/`buildRender`) under the recipe's parallel `go test ./...`.
  Note the `| tail` buffering confounder when reproducing — stream the output.

### 2026-06-16 — root cause found; the go-build hypothesis was WRONG
- 2026-06-16: closed — make test green in ~34s (was a 12m54s hang on test-autopair); fix is qall→qall\! in autopair + 3 audited sibling drivers, all routed through the new shared timeout watchdog tests/lib/run-headless.sh whose self-test pins the 124-on-timeout contract; test-infra hygiene with no new architectural surface → --no-atlas; review verdict: FIX-THEN-SHIP

- **The `go build` contention hypothesis is wrong.** `go test ./... -count=1`
  completes in ~14.7s (warm cache); `pair-changelog` is the long pole at ~14s
  (16 redundant in-test `go build`s) but that's *slow, not a hang*.
- **Real culprit: `make test-autopair` hangs forever — confirmed in isolation**
  (>15s, killed). A full `make test` run hung **12m54s** on `test-autopair`
  before I killed it; a **7-day-old** leaked `nvim --headless … init.lua …
  autopair` corpse was still in the process table (this recurs + leaks zombies).
  Note: the issue's "passes directly" list never actually included
  `test-autopair`.
- **Root cause (A/B proven):** `tests/autopair-test.sh`'s driver ends with
  `vim.cmd('qall')` (no bang, line 64) but modifies the buffer via
  `nvim_buf_set_lines` for every case and never saves. `qall` → `E37: No write
  since last change` → refuses to quit → headless nvim blocks in its main loop
  forever (even with stdin=/dev/null). `qall!` exits clean (code 0).
- **Two defects:** (1) the `qall`→E37 hang; (2) no timeout watchdog — *why* a
  one-char bug freezes the whole suite for 13 min and leaks nvims for days.
  Same latent `qall` hazard sits in 3 sibling drivers (queue-send passes today
  only because its send path saves the buffer).
- Fix scope (operator-approved): root cause + shared watchdog + audit all 4.

### 2026-06-16 — fix landed; suite green

- All 4 Plan items done in-tree: `qall`→`qall!` in autopair + the 3 audited
  siblings (queue-send, statusline-pos, changelog-notify), each now routed
  through the new shared watchdog `tests/lib/run-headless.sh`; `make test` wires
  `test-run-headless` first; `tests/run-headless-test.sh` pins the watchdog
  contract (clean exit → 0 quiet; modified-buffer bare-`qall` → `124` ≤ timeout,
  loud, names #60; generic non-terminating child → `124`).
- **Verified green, streamed, timed:** `make -f Makefile.local test` completes in
  **~34s** (was a **12m54s** hang on `test-autopair`); `test-run-headless` and
  `test-autopair` both pass. Done-when satisfied.
