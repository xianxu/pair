# #255 M1 terminal qualification

Decision: **reject the pinned backend for unchanged production adoption**. Keep the virtual-terminal architecture, but do not replace either console with this version. Qualification is not a fix for the user's display/selection symptoms and cannot close #255.

Candidate: `github.com/charmbracelet/x/vt@v0.0.0-20260510215043-e3181689be6b`.

Result after M1 review corrections: **53 pass, 15 fail, 14 not-covered; qualified=false**. Failures below are observed mismatches against the declared fixture profile, not proof that every terminal must share every profile choice. Core escape/query expectations derive from [xterm](https://invisible-island.net/xterm/ctlseqs/ctlseqs.html); extended-key requirements from [Kitty](https://sw.kovidgoyal.net/kitty/keyboard-protocol/). Per-case sources are in the emitted report.

Rerun from the repo root:

```sh
go run ./cmd/probes/terminalqualify > /tmp/pair255-terminal-qualification.json
```

Expected exit1 (`go run` prints `exit status 1`) because qualification is negative. Exit2 means infrastructure/invocation failure. `go test` verifies the diagnostic harness, not candidate approval. Every executable case uses literal expected output/cells; passing split cases additionally exercise every split point and byte-at-a-time delivery. A failing case stops at its first mismatch.

## Observed failures

| Case | Evidence |
|---|---|
| combining | `variant=0 chunks=1; cell:0,0: got "e"; want "é"` |
| zwj | `variant=4 chunks=2; cell:0,0: got "👩"; want "👩\u200d💻"` |
| save-csi | `variant=0 chunks=1; cell:2,1: got " "; want "X"` |
| alt-47 | `variant=0 chunks=1; cell:0,0: got "X"; want "A"` |
| hyperlink | `variant=0 chunks=1; link-params:0,0: got "https://example.com/"; want "id=probe"` |
| keyboard-query | `variant=0 chunks=1; replies: got ""; want "\x1b[?1u"` |
| ctrl-return | `variant=0 chunks=1; replies: got ""; want "\x1b[13;5u"` |
| alt-up | `variant=0 chunks=1; replies: got "\x1b\x1b[A"; want "\x1b[1;3A"` |
| key-repeat | `variant=0 chunks=1; replies: got ""; want "\x1b[13;5:2u"` |
| key-release | `variant=0 chunks=1; replies: got ""; want "\x1b[13;5:3u"` |
| mouse-click-suppresses-motion | `variant=0 chunks=1; replies: got "\x1b[<32;3;2M"; want ""` |
| mouse-drag-suppresses-hover | `variant=0 chunks=1; replies: got "\x1b[<35;3;2M"; want ""` |
| mouse-replace-mode | `variant=0 chunks=1; replies: got "\x1b[<35;3;2M"; want ""` |
| status-query | `variant=0 chunks=1; replies: got "\x1b[?0n"; want "\x1b[0n"` |
| cursor-style | `variant=0 chunks=1; cursor-style: got "2,false"; want "2,true"` |

## Required but untested integration obligations

- **device-attributes:** DA replies advertise a specific terminal model/capability set; the production profile and its matching DA1/DA2/XTGETTCAP replies must be chosen together before testing truthfulness.
- **composition-switch:** No production compositor exists: exercise distinct margins/save slots, split switch boundaries and evicted replay through both consumers.
- **hidden-origin:** Disposable endpoint fixtures cannot establish child identity routing while another child is selected.
- **reply-backpressure-routing:** Candidate drain tests prove local teardown; production bounded writer, origin routing and byte ordering remain unimplemented.
- **sync-publication:** The candidate exposes mutable screen state, not an immutable publishable frame boundary.
- **sync-recovery:** No production publication timeout or recovery policy exists to exercise.
- **parser-memory-bound:** Fixed 16KiB malformed/unterminated fixtures cannot prove a memory ceiling for an unending string or parameter stream.
- **wrapper-composition:** Must transport real stdoutChunk/stripCodexOutputMarkers, notification rewriting, Return translation and query tracking output through isolated Zellij, preserving raw/transformed observer semantics.
- **clipboard-policy:** Pinned candidate has no owned clipboard policy callback; existing product policy and origin-bound once-only delivery require integration qualification.
- **notification-origin:** No candidate notification callback connects OSC effects to product policy with exact origin and once-only delivery.
- **partial-parent-write:** No presenter exists to force delayed frames, partial writes, paused admission and release ordering.
- **drag-destination:** Endpoint event encodings cannot prove press-owned destination or switch/close cancellation policy.
- **terminfo-profile:** Production environment/profile has not been selected; synthetic protocol cases do not prove application compatibility.
- **live-display-selection:** M4 requires operator acceptance of continuous highlights and absence of display corruption across both consumers and reattachment.

## Interpretation and next decision

The failures span screen/grapheme parsing, keyboard negotiation/encoding, mode-specific input suppression, alternate-screen handling, hyperlink fields and query/callback semantics. These are broader than a mouse-mode constant change. A production adapter must not hide them by relabeling required behavior unsupported.

The recommended next design compares a maintained backend update/fork against an alternative mature terminal core. Scope the fixes by family and carry this matrix forward unchanged except evidence-backed oracle corrections. A pure adapter is suitable for owned effect routing, frame publication and parent presentation; it must not become a second conflicting screen parser to compensate for core grapheme/cursor defects. No dependency update or broad fork is approved by M1.

The initial architecture review suspected1049 restoration; the current simple1049 fixture passes, so this report does not claim that feature is broken. Likewise the initial raw-vs-split run included a harness x/y-versus-cursor schema mismatch; that was corrected before this report. The ZWJ split remains a real mismatch after correction. Combining marks fail even as one write. These backend failures are not a diagnosis of the current operator runtime, which still uses the existing forwarding path.

## Pair wrapper audit

`wrapcmd.proxy.handleChunk` observes queries from raw bytes, then normalizes notifications and passes rewritten bytes through `stdoutChunk` before queuing parent output. `terminal.Feed` also receives raw bytes. Codex-only filters remove synchronized-output/focus markers by default, with an optional extended-key filter. Thus the raw observer is not a model of the exact delivered stream. M3 must test Return decisions and query tracking against actual transformed delivery through Zellij and the shared terminal boundary, preserving intentional product semantics and removing incompatible filtering through #254. No filters were changed here.

## Qualification cost and provisional bounds

Apple M2 Max, Go1.26.3,10 iterations, `BenchmarkCandidateFeedSnapshot`:80x24 ~1.93ms/op and1.48MB/op;240x80 ~19.40ms/op and18.23MB/op. These figures include diagnostic string-keyed full-screen evidence maps, not a production renderer, and must not become production budget claims. Reuse compact cells/damage rather than those maps for production snapshots. Qualification caps262144cells,1000 history lines,1MiB reply evidence and4KiB mismatch detail; each operation has a2s context and the probe2min. Input fixtures are fixed synthetic data. M2 must budget endpoint memory across representative thread counts and frame latency before implementation; M4 validates those budgets under real use.

## Verification

Harness unit tests and focused race tests pass, including blocked/failing reply transport, isolation, cancellation and joined teardown. CLI tests distinguish fail/not-covered from infrastructure errors and refuse broken report output. Full repository verification and M1 boundary review are pending; append their results below before closing M1. Generated runtime assets are required in the isolated worktree for existing acceptance tests.


### 2026-09-15 final verification

`go test ./...` passed after generating the worktree runtime bundle (`/tmp/pair255-full-go-final.log`). Earlier runs found missing generated assets, two new artifact-inventory omissions (fixed), and one existing `TestOrientationTerminalRepliesDoNotCancel` failure: orientation paste appeared between complete terminal replies. That test passed 30 focused reruns and the final full suite; no wrapper change was made. Focused normal/race tests passed after adding the 64KiB fixture limit, whose failing-before-fix test prevents excessive partition metadata. The comparator mutation test failed as intended. `git diff --check` passed. Probe exit1 reproduced 41 pass / 15 fail / 14 not-covered. M1 boundary review is the remaining checkpoint.

`sdlc actual --issue 255` reported no transcript events in the available harness registry, so measured actuals are unavailable. Use the precise `--no-actual` exception with this reason rather than invent hours.


### 2026-09-15 review round 1 corrections

The first boundary review returned REWORK. BR-1: RunCase now compares complete whole/split observations in both directions in addition to literal expectations; regressions catch changes in otherwise unasserted cells, attributes, hyperlinks, replies and extra keys. BR-2: snapshots now include attribute masks, underline style and underline color, with 12 independent set/reset/preservation fixtures (all pass). BR-3: JSON includes bounded structured expected/observed maps and comparison kind. It prioritizes literal fields or whole/split differences; truncation flags explicitly identify omitted/shortened state. The full observations remain the correctness predicate. BR-4: README documents the runnable probe and exit statuses.

Updated result: 53 pass, 15 fail, 14 not-covered; no unchanged production adoption. Focused race tests pass after the corrections. Failures remain those listed above; the added tests expand the coverage rather than changing the oracle to admit failures.


### 2026-09-15 — M1 review round 2 partition-test correction

Round 2 disposed BR-1 through BR-4 and raised BR-5: partition regression tests counted calls without proving delivered bytes. Added literal partitions for empty, single-byte, multi-byte UTF-8 and CSI inputs, plus independent byte-preservation and every-boundary assertions over all 256 byte values and mixed Unicode/control streams. The production RunCase executor path is checked with the same invariant. Four mutations are detected by failed assertions: repeating whole input, dropping a byte, skipping alternating boundaries and omitting the first boundary. No production implementation changed in this correction. Final full Go suite passed after round 1 corrections; focused race verification covers these additional tests.


### 2026-09-15 — M1 closed: SHIP, round 3

SDLC's third boundary review disposed all five findings with no open blockers. The qualification milestone is complete with a negative adoption decision; #255 remains working. Full repository tests and final focused race checks pass. The next checkpoint is a revised backend choice and detailed production plan, not a runtime rollout.

After adding complete style observation, the same 10-iteration benchmark measured 80x24: 1.73ms/op, 2.90MB/op, 21274 allocations; 240x80: 19.38ms/op, 24.05MB/op, 212267 allocations (`/tmp/pair255-round3-bench.log`). These supersede the earlier diagnostic-map measurements above and remain unrelated to production budgets.

## Revisions

### 2026-09-15 — M2 repaired backend and resource measurements

The unchanged-backend rejection above remains the M1 decision. The checked-in
fork now passes all 68 original executable cases without weakening their
predicates. M2 adds attributed endpoint/presenter cases; remaining live consumer
obligations stay uncovered until their production paths are exercised.

Measured on Apple M2 Max, Go 1.27.1, using the opt-in
`TestTerminalResourceProbe` with 16 instances and post-GC `HeapAlloc` deltas:

| Geometry / workload | Backend heap MiB | Backend Usage estimate MiB | Endpoint + published + caller frame heap MiB |
|---|---:|---:|---:|
| 80×24 empty | 9.08 | 9.67 | 15.87 |
| 80×24 typical | 13.52 | 14.86 | 20.31 |
| 80×24 saturated history | 84.68 | 92.51 | 91.50 |
| 240×80 empty | 68.76 | 77.79 | 134.54 |
| 240×80 typical | 109.85 | 129.61 | 175.64 |
| 240×80 saturated history | 143.64 | 169.27 | 209.44 |

These are live Go heap measurements, not RSS, complete application measurements,
or bounds for simultaneous maximum-sized geometry/metadata. The representative
512MiB target is met here; M4 still must measure the whole composition path.
Saturating the largest scenario allocated 1.108GiB cumulatively over 2.81s;
that is population cost, not input-to-visible latency. `Usage` is a conservative
engineering estimate with documented allocator allowances, not exact heap usage.
The history byte cap continues to use logical retained payload accounting.

Reproduce with:

```sh
PAIR_TERMINAL_RESOURCE_PROBE=1 go test ./cmd/internal/terminal -run '^TestTerminalResourceProbe$' -count=1 -v
```

### 2026-09-15 14:35 PDT — M3 typed history evidence and remaining M4 obligations

The executable probe currently reports **84 pass, 0 fail, 6 not-covered; qualified=false** (`/tmp/pair255-m3-qualification.json`). The six placeholders remain uncovered; the following maps available production seams to the additional evidence required, rather than relabeling discovery or unit results as native acceptance.

| Remaining obligation | Existing production evidence seam | Required completion evidence |
|---|---|---|
| `composition-switch` | Couch `TestEndpointChromeDuringIncompleteChildSequence`, `TestEndpointChromeDoesNotUseChildCursorSave`, `TestEndpointSwitchRestoresFrameWithoutResizingChild`; shell `TestPresentationHiddenStateAndUTF8StayIsolated`, `TestPresentationSwitchRestoresCellsModesAndNeverNudges` | Attribute successful runs through both consumer paths, with distinct margins/save slots, fragmented switching and capture-ring eviction; include native output/history checks. |
| `wrapper-composition` | `TestNativeConsoleWrapperZellij` joins the actual wrapper, PTY and Console; wrapper stdout/notification/Return/query tests provide focused seams | Successful isolated native run with receipt assertions for raw/transformed observations, Return translation, notification rewrite and local query routing; current fixture existence alone does not prove every obligation. |
| `clipboard-policy` | Endpoint/Presenter typed effects and shell `TestPresentationHiddenPhysicalEffectsStaySuppressedAfterSelection`; Couch composer clipboard injection and oversized OSC52 rejection | Both consumers: literal selected/hidden write counts, suppression across subsequent selection, deterministic local query response and no physical read leakage. Composer injection alone is insufficient. |
| `notification-origin` | Couch `TestOutputBatchFocusOrder`, `TestSplitNotificationAcrossTakeover`, `TestHiddenNotificationDoesNotWaitForActivePartialSequence`, `TestConsoleInactiveNotificationCreatesAttentionAndFocusedDoesNot`; Child typed-once notification tests | Attribute current migrated consumer runs and verify exactly one outer delivery with focus-at-delivery attention semantics; shell policy must be exercised explicitly. |
| `terminfo-profile` | `TestTerminalEnvironmentProvidesCompiledProfile`, shell `TestPresentationRealChildFinalOutputAndEnvironment`, profile query cases | Packaged environment/profile load and actual shell, nvim and isolated Zellij compatibility, with truthful advertised capabilities. |
| `live-display-selection` | Typed history serializer production tests compare pinned xterm cells/wraps and disposable native Zellij logical text; no interactive selection claim | Sustained production drawing and actual selection/highlight/copy across both consumers and reattachment, followed by the authorized operator smoke before merge. |

Backend/fork `go test -race ./...` and production `PAIR_TERMINAL_NATIVE=1 go test -race ./cmd/internal/terminal ./cmd/internal/terminalqualify` pass after the M3 history changes (`/tmp/pair255-m3-fork-final.log`, `/tmp/pair255-m3-history-final.log`). Production oracle tests cover explicit trailing spaces, early-wide wrapping, append/rebuild without chrome history, one-host-row geometry, width-one/three/four growth and exact ED2 blank/space history. ED2 follows measured native Zellij: retain the visible prefix through the last printed row, preserving blank hard separators and printed-space soft rows. Pinned xterm ED2 does not save the viewport; tests assert that baseline difference instead of claiming identical direct streams.

An initial whole-publication resource probe exceeded the representative RSS target because ordinary endpoints retained a duplicate history snapshot. The corrected ordinary path transfers one owned backend history snapshot to the caller; only synchronized holds and EOF retain frozen history. No mutable shared cache was introduced. With sixteen 240×80 saturated endpoints plus caller publications, the repeat measured **287,526,792 bytes live Go heap** and **427,704,320 bytes maximum RSS** (`/tmp/pair255-m3-history-resource-final.log`), below 512MiB for this representative probe. Fresh publication allocations fell from 9.40MB to 4.94MB at 80×24 and from 12.79MB to 8.56MB at 240×80 (`/tmp/pair255-m3-history-benchmark-final.log`). These are workload measurements, not universal memory bounds or whole-application sustained-latency acceptance; M4 composition/soak measurements remain necessary.

### 2026-09-15 14:41 PDT — Consumer effect-policy conformance

New tests exercise the migrated consumer paths rather than the shared Presenter alone:

- Couch `TestConsoleClipboardPolicyThroughDelivery` passes endpoint-produced batches through `Console.Deliver` and `onChunk`, asserting one selected clipboard write, zero hidden writes, no replay after selection, and exactly one empty local OSC52 read reply to each originating child with no physical read query.
- Couch `TestConsoleNotificationUsesCapturedDeliveryFocusOnce` queues through the real `Deliver` focus capture, changes selection before processing, and asserts attention follows delivery-time focus in both directions. Each notification reaches the outer terminal exactly once, including after duplicate typed-batch delivery and repaint.
- Shell `TestShellClipboardPolicyThroughOutput` uses admitted tabs and `Child.Feed`/consumer sink delivery, asserting selected/hidden clipboard counts, no selection replay, and origin-local read responses without outer queries.
- Shell `TestShellNotificationsHiddenSelectedAndRedeliveryOnce` covers fragmented hidden and selected notifications, no output from an incomplete envelope, one completed envelope each, no replay on selection, and deduplication of repeated delivery of one typed batch.

The first run found a production Couch regression: `Clipboard:true` forwarded a hidden write that the pre-migration selected-only ordinary-output path suppressed (`/tmp/pair255-m4-effect-policy-red.log`). The production owner restored selected-only bell/title/clipboard emission while preserving notification delivery and hidden-bell attention. The new four tests now pass with `-race` in both consumer packages (`/tmp/pair255-m4-effect-policy-race.log`). Reproduce with:

```sh
go test -race ./cmd/internal/couchtty ./cmd/internal/termcmd -run 'Test(ConsoleClipboardPolicy|ConsoleNotificationUsesCaptured|ShellClipboardPolicy|ShellNotificationsHidden)' -count=1
```

This supplies concrete consumer evidence for the clipboard and notification rows above. The qualification inventory is not automatically promoted by these additional tests; live-display/selection, sustained native behavior and the operator smoke remain explicitly uncovered until their own production evidence is recorded.

### 2026-09-15 — M4 native continuity, mutation and coverage attribution

Frozen candidate native conformance passes three race repetitions (`/tmp/pair255-m4-native-stable.log`,20.910s). The direct Zellij and actual wrapper→Zellij→Couch fixtures retain the same original process PID, random boot nonce and in-memory counter across detach/reattach; a new PTY attachment resumes the counter from1 to2 without creating a replacement session. Independent xterm compares every viewport cell/style/cursor before and after. A mouse drag has visible highlight while held, followed by exact copied text on release; wrapped CJK, enhanced keys, paste/focus/drag and notification delivery survive reattachment. Real nvim content/chrome and scrolling checks also pass. All use disposable sessions.

Coverage attribution remains explicit: these native fixtures use synthetic applications, not a live Codex/Claude conversation. The wrapper native CR receipt exercises pass-through; recognized-composer Return remapping and normalized queued-observer equality are established by focused wrapper stateful tests. Native CPR asserts response shape, not an exact-coordinate/origin proof for every nesting level. The thirty-minute consumer soaks use real PTYs with synthetic helpers, not thirty-minute native Zellij/nvim sessions. Pair term's soak checks mouse destinations; visible selection in both real agent and right panes remains in the operator smoke.

The final inventory remains **84 pass,0 fail,6 not-covered; qualified=false** (`/tmp/pair255-m4-qualification.json`, expected CLI exit1). The production evidence tables above complement those placeholders; they do not relabel them. Operator sustained-use acceptance remains pending. Three causal mutations are detected again on frozen source: discard split UTF-8 parser carry, overwrite the final child row with chrome, and drop mouse motion. Each unmodified targeted suite passes and each scratch-overlay mutation fails behavioral assertions; no production source is mutated (`/tmp/pair255-m4-mutation/manifest.json`).

### 2026-09-15 — M4 final performance and resource evidence

Five timing trials per binary/geometry compare baseline `b11ab67f` with the frozen candidate. The process runs actual `pair term` with synthetic PTY applications and pinned xterm5.5.0 receipts. Both include independent interpreter IPC/parsing; no time is subtracted. Cold startup resets per-trial application data/runtime extraction, not OS filesystem cache. Two production soaks ran concurrently; these are development-machine observations rather than dedicated-machine guarantees.

| Metric |80×24 baseline → candidate |240×80 baseline → candidate |
|---|---:|---:|
| Startup p95 ms |263.832 →660.857 |306.976 →222.145 |
| Startup median ms |171.54 →226.14 |174.95 →204.35 |
| New tab p95 ms |28.688 →49.075 |43.605 →49.203 |
| Input p95 ms |2.843 →19.739 |3.088 →23.454 |
| Switch p95 ms |33.746 →39.386 |41.303 →34.769 |
| Post-history input p95 ms |2.495 →21.051 |2.677 →24.011 |
| Post-history switch p95 ms |187.129 →101.060 |215.472 →115.416 |
| Synthetic output median MiB/s |0.788 →1.040 |0.673 →0.562 |
| Active-output median wrapper CPU % |3.954 →41.378 |4.033 →40.970 |
| Idle median wrapper CPU % |0 →0.498 |0 →0.498 |
|120Hz unsolicited-hover median wrapper CPU % |1.499 →1.500 |1.499 →1.999 |
|16 saturated tabs: maximum observed wrapper RSS MiB |23.234 →203.156 |23.828 →386.594 |

Input meets50ms and representative16-tab sampled RSS meets512MiB. Saturated-history pooled switch p95 **misses100ms** by1.060ms/15.416ms; switches into only the saturated destination have p95105.106ms/150.089ms. All samples remain included. Seven240×80 events above100ms occur in a single trial, with identical44,156-byte redraws; each other trial has none. Candidate history rebuilds preserve logical text/native copy and are smaller than baseline replay, but still cost output/interpretation plus scheduling and compositor work. No single cause is claimed for the extra timing variation. This is a documented provisional-budget exception for boundary review and operator smoke, not a passed threshold or rollout approval. The candidate's active-output CPU cost is substantially higher than the raw-forwarding baseline and is shown explicitly.

RSS is the maximum of49 point-in-time wrapper samples per16-tab run, not OS peak; it includes compositor/endpoints/rings/PTY descriptors and excludes child helpers and independent oracle. Each tab emits1,050,624bytes before sampling, retaining sixteen histories concurrently. CPU is wrapper-only `ps` cumulative accounting;2s idle/hover windows can quantize small deltas. The hover workload sends unsolicited button-none motion while child tracking is disabled; it is not native mouse capture. Throughput covers synthetic payload emission through visible completion.

Evidence: `/tmp/pair255-m4-performance-final.{json,log}`, `/tmp/pair255-memory-final.{json,log}`. Baseline SHA256 `f52764eca185b0147b7e2c60b5f8762c98795dff2e18e4164a0be251a66dccd7`; candidate `3f1f9ecac95017e8f18aaca446a6327cd2aa45ade65729d651ebe34f2b1f7afd`. Failed pre-correction harness evidence remains separately preserved.

### 2026-09-15 — M4 final automated verification

After the test-only startup fence correction, the complete root `go test ./... -count=1` passes (`/tmp/pair255-m4-full-confirmed.log`). The first failed full run remains `/tmp/pair255-m4-full-final.log`; it is not represented as a pass. The local fork normal suite passes. Required native/independent-oracle race suites pass for terminal/qualification/PTY/Pair term/runtime bundle/wrapper (`/tmp/pair255-m4-race-final.log`), as does the full Couch race suite (`/tmp/pair255-focus-prefix-full-race.log`).

`make test-lua`, shell shortcut checks, real retention CLI/editor integration, and migrated paint-gate consumer checks all pass (`/tmp/pair255-m4-{lua,shortcuts,retention,paint-gates}-final.log`). Runtime bundle determinism/drift generation and Linux/amd64 compilation pass. The isolated candidate's copied assets and three binaries are ready; Pair's binary SHA256 matches the measured candidate. Source fingerprinting confirms all421 non-test Go files remain unchanged during sustained qualification. The candidate manifest records pre-commit embedded VCS stamp versus the matching committed production source explicitly.

Environment: Apple M2 Max, macOS26.6.2(25G83), Go1.27.1 darwin/arm64, Node26.0.0/npm11.12.1, Zellij0.45.1, nvim0.11.7/LuaJIT2.1.1741730670, pinned xterm-headless5.5.0. These versions describe the qualification environment, not every supported terminal program. See `000255-terminal-smoke.md` for the remaining operator workflow and source/binary details in `/tmp/pair255-smoke-gfr6g7pd/build.json`.

### 2026-09-15 — Sustained shutdown finding and corrected candidate

The original Pair run **passes**30m0.017s:141,796 iterations,6,778,538,818 parent bytes,1,087,107 writes, maximum input receipt30.244ms and final goroutine count2. The original Couch run **fails** at1800.03s during shutdown after completing operational screen/routing assertions. Its failure remains `/tmp/pair255-couch-soak-30m-before-stop-fix.log`; it is not counted as a passed soak. Short real-PTY repetition reproduces the problem and captures expected owner cancellation as Presenter write failure, with no teardown diagnostic.

Couch now determines the final operational result after Presenter release/join, ignoring only new cancellation-only trees during shutdown. Already-latched live failure, deadlines, mixed host/cancellation errors and every cleanup failure remain fatal. Forced tests include zero/partial accepted bytes, parent parser repair and raw/host cleanup in every outcome. Deterministic race×100 passes, real-PTY250ms soaks race×100 pass(61.559s), and full Couch race passes(17.992s). Red/green logs are `/tmp/pair255-couch-stop-{red,green,full-race}.log`; the actual pre-fix cause is `/tmp/pair255-couch-soak-stop-cause.log`.

The production delta is exactly `couchtty/console.go` and `couchtty/terminal.go`. Pair term's executed path and shared renderer remain unchanged, so its completed soak and the performance measurements above still apply to those paths; the measured Pair binary hash is historical, distinct from the refreshed candidate build. Couch's fresh thirty-minute run is `/tmp/pair255-couch-soak-30m-after-stop-fix.log`. The smoke binaries are rebuilt, and final candidate hashes/source manifest will be recorded after review.

### 2026-09-15 — Corrected Couch sustained run passed

`/tmp/pair255-couch-soak-30m-after-stop-fix.log` passes30m0.015s, including
Stop/join and exit0:78,772 iterations,19,693 attachment replacements,
3,059,548,059 parent bytes,853,486 writes and maximum visible receipt22.293ms.
Operational worker count remains16; final measured heap is5,779,392 bytes.
Together with the earlier passed Pair30m run, this supplies the two sustained
consumer runs. These immutable binaries predate the new notification broker and
OSC9 adapter; the unchanged screen/input stress paths remain attributed to those
binaries, while the new notification path requires its own native stress and
full/race verification. No operator display/highlight acceptance is claimed.

### 2026-09-15 — Final notification revision verification

The final production source has424 non-test Go files in
`/tmp/pair255-notification-production-source.json`, with zero drift through
verification. Full root `go test ./... -count=1` passes
(`/tmp/pair255-m4-full-after-notifications.log`); full wrapper race passes165.300s
(`/tmp/pair255-notify-wrapper-race.log`). Broker/CLI race, full Couch/term race,
shared terminal/qualification race and local fork normal suites pass. The
additional real-PTY blocked/flooding-child output-failure regression passes
race×5; the startup test with the real private PID binding passes race×3.

Strict actual-wrapper/Zellij/nvim qualification passes race×3 in79.614s
(`/tmp/pair255-native-inband-final.log`):96 hook notifications exactly once,
three4096-byte UTF-8 messages,48 hidden/48 focused attention cases,12 resizes,
persistent reattachment with unchanged PID/nonce/counter, held selection/copy,
independent xterm screen comparison and every Console exit0. A delayed
Feed→enqueue overlay also passes, after proving the original highlight assertion
race. These synthetic native checks do not replace the operator's real Codex/
Claude sustained visual acceptance. The probe inventory remains84pass/0fail/
6not-covered, qualified=false; placeholders are not silently promoted.

Artifact coverage and runtime-bundle suites pass; regenerated Zellij config
checks and Linux/amd64 builds pass. The initial cross-build command targeted an
existing file as a directory and failed; corrected output directory succeeds
(`/tmp/pair255-notification-linux-build-final.log`). Previous Lua/shell/retention
and causal mutation evidence remains valid for unchanged paths.

Final smoke Pair hash:
`b4db037d0760d26f4069e4357721be1e7c7df75c589ce1418eae795a3ab7b7ef`,
identical to the native-tested binary. Candidate root
`/tmp/pair255-smoke-gfr6g7pd` holds refreshed runtime assets, Pair/Couch/helper,
private launcher and build manifest. `couch --list` through that launcher reports
no threads; no interactive operator session was launched. See the smoke guide
for exact launch/revert and acceptance steps. M4 boundary review remains pending.
