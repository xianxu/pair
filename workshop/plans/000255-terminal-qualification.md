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
