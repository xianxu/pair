---
id: 000881
status: codecomplete
deps: []
github_issue:
created: 2026-10-24
updated: 2026-10-25
estimate_hours:
card_mirror: '2b9c23b76332f74465a85809621529da132bb1d4' # card fields mirrored from issue-cards; edit via sdlc
flake_tests: [github.com/xianxu/pair/cmd/internal/broadcast.TestNamedTunnelServesFromPrivateSocket]
started: 2026-10-25T00:44:15-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "6fec5b1d", done: "ba819485"}
actual_hours: 0.35
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

- [x] Reproduce deterministically with a long inherited TMPDIR; check the complete worst-case derived listener path against the 100-byte budget and retain real Unix/config/mode/cleanup assertions.
- [x] Delegate shortDir to testpair.ShortRoot (ARCH-DRY); add a catalog guard and mutation controls for this fixture allocator.
- [ ] Verify the named tunnel and too-long TCP fallback, the broadcast package and catalog; record the affected check.

## Log


- 2026-10-25: closed — BR-1 fixed: atlas/go-lint-catalog.md maps the only new Pair-local ownership rule, naming broadcast.shortDir, direct shared ShortRoot delegation, alias/shadow behavior, top-level-return syntax limitations and mutation/real-listener regressions; atlas/index.md already links it. Reviewed against actual matcher and tests; documentation-only correction, no behavior change or wording-presence test. Prior RED231-byte path and GREEN named Unix/config/0700/cleanup/TCP count3, broadcast race159.797s, catalog6.826s and audited inventory PASS remain valid. The final Plan row remains unchecked only for recorded-check workflow. Retained pre-close lease-526714052 remains FIFO queued; its exact-code pass still precedes landing. PR606.; review verdict: SHIP
2026-10-24: Found by #865 recorded affected attempt367170267a28fe624cb66a23bf17fa80 on9676dff56fc63b95885f3ec084127d766ca1976e. Sole failure: github.com/xianxu/pair/cmd/internal/broadcast.TestNamedTunnelServesFromPrivateSocket, cloudflared_test.go118: named tunnel origin127.0.0.1:62752, want a unix socket. Other selected Go packages passed; make stopped before the script groups. Duration5m25s. Full failed log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/367170267a28fe624cb66a23bf17fa80.log, SHA256 prefixa79e4f8e79fd. No broadcast source is changed by #865.

TL4b528f6a requested a main control in the same checkout path before filing. After the managed check retired its checkout and scratch, recreated an independent clean main checkout at the identical /Volumes/TestScratch/local-checks/sdlc-local-3304101268 and the same TMPDIR/GOTMPDIR /Volumes/TestScratch/tmp/sdlc-run-affected-64175-1517064309 (59 bytes). HEAD4e3764c65acfff86a9073f1546e38251798005b7, no tracked changes. Command: env TMPDIR=/Volumes/TestScratch/tmp/sdlc-run-affected-64175-1517064309 GOTMPDIR=/Volumes/TestScratch/tmp/sdlc-run-affected-64175-1517064309 go test ./cmd/internal/broadcast -run '^TestNamedTunnelServesFromPrivateSocket$' -count=1 -v. FAIL0.293s at the same line, TCP127.0.0.1:64823. This is a pre-existing fixture failure on main, distinct from #570's review socket.

A diagnostic-only Go overlay added a t.Logf of filepath.Join(pl.dir,"s"), its byte length and maxSocketPath without changing logic. One run passed at100 bytes. A bounded count3 run failed twice at101 bytes and passed once at100 bytes (overall FAIL0.504s). Exact failing path: /Volumes/TestScratch/tmp/sdlc-run-affected-64175-1517064309/cb3769456254/couch-broadcast-1449130918/s (101 bytes); second failing path has cb3378850947/couch-broadcast-3179157916/s (101 bytes). Passing path cb159981107/couch-broadcast-1720210679/s is100 bytes. Cloudflared.Listen's application budget is100, kept below Darwin's104-byte sun_path. The random decimal suffix lengths explain fail/pass on unchanged main. Diagnostic overlay stays outside tracked source at /Volumes/TestScratch/tmp/pair865-broadcast-control.CNseYR/overlay.json; no production or assertion edit.

2026-10-24 registration evidence: original failed #865 head9676dff56fc63b95885f3ec084127d766ca1976e passed the unmodified focused test in0.301s in the identical checkout path when TMPDIR/GOTMPDIR were shortened to /Volumes/TestScratch/tmp. This is focused-versus-recorded evidence with an explicit environment difference, not a full-suite passing retry. Registered the exact test on this unclaimed card via sdlc flake register, using that observation together with the main failure and100/101-byte diagnostic controls above. #865 is using the supported saved affected resume; no passing status is assumed.

2026-10-25: Quick-flow plan posted to hub, proceeding unless objected. Existing production allocation and fallback remain unchanged. A larger production seam or a second allocator would duplicate an existing owner without fixing fixture hermeticity. The regression sets TMPDIR only after checking native Unix availability, so a fixture-created long path cannot turn the intended failure into a skip. The complete path check reserves ten decimal suffix bytes, removing the observed 100/101-byte random-suffix dependence.

2026-10-25 implementation evidence: the exact named-tunnel regression first failed deterministically on the old helper: complete socket path 231 bytes, budget 100 (FAIL0.186s). The assertion includes the private directory prefix, full ten-byte decimal suffix and socket leaf. After delegating shortDir to testpair.ShortRoot, the named Unix tunnel, explicit too-long TCP fallback and directory-removal tests all PASS count3 (0.582s), preserving mode0700 and exact cloudflared argv/config checks. The named test now explicitly closes the tunnel and listener and observes directory removal. No production code changed.

Catalog RED: the new literal-fixture mutation test rejected none of five bad allocator variants before the rule existed. GREEN: shared and aliased imports pass; inherited TMPDIR, t.TempDir, unused shared-root call, wrong import and shadowed package all produce the pair#881 finding. The tracked catalog plus these controls PASS6.826s. The rule is in Pair's extension of pkg/golint and owns the existing broadcast shortDir constructor, rather than adding an allocator policy (ARCH-DRY).

Affected-reader audit: cloudflared_test.go reads only its own generated tunnel config/argv/run records and private fake executables; subprocess inputs are literal fixture commands or this test executable. It reads no repository source. Retained local_or_literal_fixture and individually recertified that one changed reader. make test-affected-inventory PASS. The new lint test uses literal source in the existing fake repository seam and reads no repository source itself.

2026-10-25: Targeted package qualification: go test -race ./cmd/internal/broadcast -run ^Test -count=1 PASS159.797s, including the callers sharing shortDir, process guards and orphan cleanup. No candidate tests remain running. Preparing the recorded affected check on the pushed head.

2026-10-25: PR606 opened. Recorded pre-close on ce4bf5d4b0cf304c1fb7aabe6c5b62729938bb86 selected the full suite because the individually recertified reader inventory is in its window. Retained run lease-526714052, attempt643fd926eb3008e42362d031097e26d9, initially FIFO position6. All implementation qualification is complete; the final Plan row remains unchecked only for the recorded-check workflow. Under tl-worker rules the boundary review may overlap this retained queue wait, with the exact-head recorded pass still mandatory before landing. No further implementation change planned.

2026-10-25 BR-1: round1 FIX-THEN-SHIP/high found only the missing local-catalog atlas mapping; all independent functional checks passed. Verified atlas/go-lint-catalog.md requires mapping every Pair-specific ownership policy. Added the sole new rule in this window, naming its protected constructor, shared allocator, import/shadow behavior, syntactic limitations, mutation tests and real listener regression. atlas/index.md already links the catalog page. Production/tests unchanged; retained suite and FIFO position remain valid. Re-running full close as the ledger requires explicit disposition; no close finalized in round1.
