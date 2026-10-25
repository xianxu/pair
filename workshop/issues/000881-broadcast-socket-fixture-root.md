---
id: 000881
status: open
deps: []
github_issue:
created: 2026-10-24
updated: 2026-10-24
estimate_hours:
card_mirror: '0ddc8eeba07055c7acb10d972c5fb50289bda1ef' # card fields mirrored from issue-cards; edit via sdlc
flake_tests: [github.com/xianxu/pair/cmd/internal/broadcast.TestNamedTunnelServesFromPrivateSocket]
---

# Broadcast socket fixture exceeds managed TMPDIR path budget

## Problem

TestNamedTunnelServesFromPrivateSocket intermittently fails under the managed affected-check TMPDIR. Its shortDir helper uses os.MkdirTemp with an empty parent, so it inherits the run's longer temporary root. needUnixSockets probes a shorter path successfully, but Cloudflared.Listen adds another couch-broadcast directory and falls back to TCP when the complete socket name exceeds its 100-byte budget.

## Spec

Keep the production Unix-socket/TCP fallback behavior unchanged. Make socket-requiring broadcast fixtures use the shared testpair.ShortRoot allocator, including the complete derived listener path, as the launcher fixture in #866 does. Do not weaken the private-socket assertion, skip it because of a fixture-created long path, or classify the failure as an unexplained retry.

## Done when

- The exact named-tunnel test passes with a long inherited TMPDIR while proving a real private Unix listener, mode0700, expected configuration and cleanup.
- A deterministic regression covers the complete derived path at the budget boundary and fails without the fixture correction.
- Socket fixtures reuse the shared short-root allocator rather than adding another allocator; the explicit too-long production fallback test still passes.

## Plan

- [ ]

## Log

2026-10-24: Found by #865 recorded affected attempt367170267a28fe624cb66a23bf17fa80 on9676dff56fc63b95885f3ec084127d766ca1976e. Sole failure: github.com/xianxu/pair/cmd/internal/broadcast.TestNamedTunnelServesFromPrivateSocket, cloudflared_test.go118: named tunnel origin127.0.0.1:62752, want a unix socket. Other selected Go packages passed; make stopped before the script groups. Duration5m25s. Full failed log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/367170267a28fe624cb66a23bf17fa80.log, SHA256 prefixa79e4f8e79fd. No broadcast source is changed by #865.

TL4b528f6a requested a main control in the same checkout path before filing. After the managed check retired its checkout and scratch, recreated an independent clean main checkout at the identical /Volumes/TestScratch/local-checks/sdlc-local-3304101268 and the same TMPDIR/GOTMPDIR /Volumes/TestScratch/tmp/sdlc-run-affected-64175-1517064309 (59 bytes). HEAD4e3764c65acfff86a9073f1546e38251798005b7, no tracked changes. Command: env TMPDIR=/Volumes/TestScratch/tmp/sdlc-run-affected-64175-1517064309 GOTMPDIR=/Volumes/TestScratch/tmp/sdlc-run-affected-64175-1517064309 go test ./cmd/internal/broadcast -run '^TestNamedTunnelServesFromPrivateSocket$' -count=1 -v. FAIL0.293s at the same line, TCP127.0.0.1:64823. This is a pre-existing fixture failure on main, distinct from #570's review socket.

A diagnostic-only Go overlay added a t.Logf of filepath.Join(pl.dir,"s"), its byte length and maxSocketPath without changing logic. One run passed at100 bytes. A bounded count3 run failed twice at101 bytes and passed once at100 bytes (overall FAIL0.504s). Exact failing path: /Volumes/TestScratch/tmp/sdlc-run-affected-64175-1517064309/cb3769456254/couch-broadcast-1449130918/s (101 bytes); second failing path has cb3378850947/couch-broadcast-3179157916/s (101 bytes). Passing path cb159981107/couch-broadcast-1720210679/s is100 bytes. Cloudflared.Listen's application budget is100, kept below Darwin's104-byte sun_path. The random decimal suffix lengths explain fail/pass on unchanged main. Diagnostic overlay stays outside tracked source at /Volumes/TestScratch/tmp/pair865-broadcast-control.CNseYR/overlay.json; no production or assertion edit.

2026-10-24 registration evidence: original failed #865 head9676dff56fc63b95885f3ec084127d766ca1976e passed the unmodified focused test in0.301s in the identical checkout path when TMPDIR/GOTMPDIR were shortened to /Volumes/TestScratch/tmp. This is focused-versus-recorded evidence with an explicit environment difference, not a full-suite passing retry. Registered the exact test on this unclaimed card via sdlc flake register, using that observation together with the main failure and100/101-byte diagnostic controls above. #865 is using the supported saved affected resume; no passing status is assumed.
