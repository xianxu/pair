---
id: 000585
status: open
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
card_mirror: 'db2c8da7ee9102cbd4519fd684e2a762604bbaa2' # card fields mirrored from issue-cards; edit via sdlc
---

# Preserve cloudflared stderr on early process exit

## Problem

The full recorded suite intermittently loses the fake cloudflared process's stderr failure before reporting its exit. This is independent of #571's Codex adapter; no real tunnel or network is used by this test.

## Spec

Investigate process exit versus stderr-drain ordering in `cmd/internal/broadcast/cloudflared.go`. `StderrPipe`, `watchStderr`, and a concurrent `cmd.Wait` currently feed separate completion paths. The working hypothesis is that the exited branch reads `lastLine` before the reader records the error, or Wait closes the pipe before it drains. This is a hypothesis, not a measured root cause yet. Preserve the original cloudflared failure without accepting its API URL as a tunnel URL.

## Done when

- A deterministic fake-process regression demonstrates the exit/stderr ordering problem and passes after the lifecycle fix, without sleeps or a larger timeout as the fix.
- A fast failing fake retains its stderr error; API failure URLs still cannot become successful tunnel URLs.
- Process cleanup and the broadcast package tests pass, including repeated runs under load.

## Plan

## Log

### 2026-10-15 — first observed during #571 verification

- Test: `TestQuickTunnelAPIFailureIsNotAURL`, `cmd/internal/broadcast/cloudflared_test.go:434`.
- Exact tested head: `8bd4baa138c63cda357c3a447ddb2cba294f1df7`.
- Recorded suite run ID: `b2dd3d164d5caad0fd4f624661697ade`. At filing it is still finishing other packages; log: `/Users/xianxu/workspace/pair/.git/sdlc/test-stats/inflight/b2dd3d164d5caad0fd4f624661697ade/run.log` (failed runs move under `test-stats/failed/<run-id>.log`).
- Failure after 0.13s: `err broadcast: cloudflared exited while opening: no output, want cloudflared's own failure`; broadcast package duration 59.332s.
- Unchanged isolated rerun: `go test ./cmd/internal/broadcast -run '^TestQuickTunnelAPIFailureIsNotAURL$' -count=10` passed, package duration 1.497s; log `/tmp/pair571-cloudflared-check.log`.
- Two earlier complete #571 suites on heads `12223899754159b8aba61548278c2aaacc74df6b` and `ba943894842c638e299efee8bd49be189c92d44d` passed broadcast. These are observations, not a calibrated failure rate.
- Filed unassigned at the TL's direction; keep its fix out of #571. The TL authorized an unchanged recorded-suite rerun on the same head after filing.
