# Boundary Review — pair#654 (whole-issue close)

| field | value |
|-------|-------|
| issue | 654 — Qoder slots never show a context figure: ParseTokenUsage has no qoder case |
| repo | pair |
| issue file | workshop/issues/000654-qoder-slots-never-show-a-context-figure-parsetokenusage-has-no-qoder-case.md |
| boundary | whole-issue close |
| milestone | — |
| window | db5ddb4e2bf5af6ec85c3c916e51b17523c12085..07df865f7fdefe2c7243cb888a9c8e8d51662c2b |
| command | sdlc close --issue 654 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-18T11:52:22-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The code fix is correct and covered by tests. `ParseTokenUsage` now has a Qoder branch that reads `context_usage_ratio` and ignores Qoder's token fields, which are always zero. A single `TokenUsage.Figure()` renderer replaces all three `Humanize(usage.InputTokens)` call sites. I ran the tests in this checkout: `sessioninventory`, `contextcmd`, `titlepoller` and `ctxmeter` all pass. `couchcmd` fails in this checkout, but the failure is environmental. A socket path is longer than the 103-byte limit because the review cache directory has a long path. I checked out the base commit `db5ddb4e2` and `TestWorkbenchPerformanceMessageUnchangedEvidence` fails there in the same way, so this diff did not cause it. One thing keeps this from SHIP. The second `## Done when` clause ("A Qoder slot's pane title shows its count. Checked live.") is still an unchecked Plan item, and the issue is being closed anyway.

1. **Strengths**
   - `cmd/internal/sessioninventory/usage.go:22-27`: `Figure()` is the one renderer for the context figure. My sweep found no remaining `Humanize(...InputTokens)` call outside it; `contextcmd`, `couchcmd/titles.go` and `titlepoller/projection.go` all call it now (ARCH-DRY: pass).
   - The parser accepts a record only under strict conditions: it must be an assistant record, not a sidechain, have a usage block, and have a ratio that is present and in (0, 1]. `usage_qoder_test.go:24-33` checks each rejection path, including "tokens but no ratio", so Qoder can never show a constant "0".
   - The fixture uses a usage block copied byte for byte from a real Qoder transcript. The provider parity fixture (`title_projection_test.go:293-296`) now uses Qoder's real shape instead of borrowing Claude's, so the parity test exercises the new path.
   - Keeping `ContextRatio` as a plain `float64` rather than the planned pointer keeps `TokenUsage` comparable with `==`. That departure from the Design is recorded in the Log.
   - The atlas line about the context meter was updated with the percentage behaviour. README needs no change, since this adds no new surface a user runs or types.

2. **Critical findings**: none.

3. **Important findings**
   - **Live check not done.** `## Done when` clause 2 requires a live check of a Qoder pane title. The Plan's last item is still `[ ]`, waiting on a Couch restart. This review cannot verify it. To fix: run the restart, observe e.g. `qoder (40%)` on a Qoder pane title, record it in the Log and tick the item. Alternatively, explicitly move the clause out of this close. This is the only member of its family in the window.

4. **Minor findings**
   - The Design text says the ratio must be "within [0, 1]" and that `ContextRatio` is `*float64`. The code accepts (0, 1] and uses a plain `float64`. Both changes are justified in the Log, but the Design section still states the old contract.
   - `Figure()` renders a small positive ratio (< 0.005) as `0%`. A test pins this, so it is deliberate; it is only noted because the parser rejects a ratio of exactly 0 on the grounds that `0%` would mean nothing.

5. **Test coverage notes**
   - Without the new branch, `ParseTokenUsage` falls to its default and returns false, so `TestParseTokenUsageQoderReadsTheContextRatio` would fail. The tests pin real behaviour.
   - Rounding and the token-count fallback are pinned in `TestTokenUsageFigure`.
   - `TestTokenUsageFromJSONLQoderKeepsTheLastRatio` checks that the last main-chain ratio wins over interleaved bookkeeping records and a trailing sidechain record.
   - None of the three call sites has its own Qoder test. That is acceptable, because each now just calls `Figure()`.

6. **Architectural notes**
   - **ARCH-DRY: pass.** All three renderers derive from `Figure()`.
   - **ARCH-PURE: pass.** The parser and `Figure()` are pure, and their tests do no IO.
   - **ARCH-PURPOSE: pass on the code.** Every consumer derives from the new renderer. The only remaining gap is the live-check clause above.
   - A future agent that reports both a token count and a ratio would always show the percentage, because `Figure()` prefers the ratio. That is fine today; revisit it if such an agent appears.

7. **Plan revision recommendations**
   - Add a `## Revisions` entry: "Design: `ContextRatio` is `float64`, not `*float64` (keeps `TokenUsage` comparable). The accepted ratio range is (0, 1], not [0, 1]."

```findings
findings:
  - id: new
    severity: Important
    kind: unverifiable
    family: done-when-clause-undelivered-at-close
    title: |
      Done-when clause "A Qoder slot's pane title shows its count. Checked live." is not done; its Plan item is still unchecked
    detail: |
      The issue's final Plan item (live check after the next Couch restart) is still [ ], and the window holds no evidence of a live observation. A review checkout cannot run Couch or Qoder. Evidence that would answer this: a Log entry recording a real Qoder pane title showing a percentage (e.g. "qoder (40%)") after the restart, with the Plan item ticked, or an explicit decision to move this clause out of this close. This is the only instance of the family in the window.
  - id: new
    severity: Minor
    kind: defect
    family: plan-design-drift-unrevised
    title: |
      Design text still says ContextRatio is *float64 with range [0, 1]; the code uses float64 with range (0, 1]
    detail: |
      Both changes are justified in the 2026-10-18 Log, but the Design section has no Revisions entry, so the plan still states the old contract. Instances: the field type (usage.go:17) and the accepted range (usage.go:105).
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"ec1863c6-9de3-4574-816f-e7b65161aba9","repository":"github.com/xianxu/pair","issue_id":"000654","recipe":"small-diff-review","round":1,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/b00e82452cc65eec","reviewed_head":"07df865f7fdefe2c7243cb888a9c8e8d51662c2b","slot":"pair:2","requested_at":"2026-10-18T18:50:14.546424Z"},"phase":"finished","pid":8068,"started_at":"2026-10-18T18:50:14.654029Z","finished_at":"2026-10-18T18:52:21.27386Z","session_id":"8a9146c0-d9ee-46e1-b5ba-4389d7e97a84","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.458808666s
head: "07df865f7fdefe2c7243cb888a9c8e8d51662c2b"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-18T11:59:51-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 654 — Qoder slots never show a context figure: ParseTokenUsage has no qoder case |
| repo | pair |
| issue file | workshop/issues/000654-qoder-slots-never-show-a-context-figure-parsetokenusage-has-no-qoder-case.md |
| boundary | whole-issue close |
| milestone | — |
| window | db5ddb4e2bf5af6ec85c3c916e51b17523c12085..d934e9d23fae9ce3b087ad9975fd4fecda292f92 |
| command | sdlc close --issue 654 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-18T11:59:51-07:00 |
| verdict | SHIP |

## Review

I'm re-running the couchcmd tests with a short temp dir. The failures so far all come from Unix socket paths that are too long for this checkout path.

```verdict
verdict: SHIP
confidence: high
```

The code fix is correct and well scoped. `ParseTokenUsage` gets a dedicated `AgentQoder` branch (`cmd/internal/sessioninventory/usage.go:93-108`) that reads `context_usage_ratio` and ignores Qoder's zero token fields. `TokenUsage.Figure()` (`usage.go:22-27`) is now the only renderer: a grep for `Humanize(` in non-test code finds only `Figure` itself and the definition in `ctxmeter`. All three call sites the spec named (`contextcmd.go:72`, `couchcmd/titles.go:342`, `titlepoller/projection.go:53`) go through it. The `sessioninventory`, `titlepoller` and `contextcmd` tests pass. `couchcmd` builds, but its tests fail in this confined checkout for reasons outside this diff: 252 failures are "message socket must be a clean absolute path of at most 103 bytes", because the checkout path is long. With a short TMPDIR, the remaining failures are the sandbox refusing `git init` outside the reviewer's directories (sdlc #339). The one-line change in `couchcmd` is a compile-checked call swap. BR-2 is addressed by the Revisions entry. I withdraw BR-1 on its evidence; the live pane check stays with the TL after landing.

Done-when: current. The Revisions entry changes only the field's representation and the edge of the accepted range. Both Done-when clauses still describe what this diff builds: a parser fixture with negative cases, and a live pane figure. "Count" in the second clause reads as "context figure", which for Qoder is a percentage, as the Design states.

1. **Strengths**
   - The fixture uses a captured Qoder usage block byte for byte (`usage_qoder_test.go:8`). The negative cases cover every guard in the acceptance condition: no ratio, tokens without a ratio, a zero ratio, a ratio above 1, a sidechain record, no usage block, a user record, and malformed input.
   - The ARCH-DRY consolidation is complete: three `Humanize(InputTokens)` sites are now one `Figure()`, and the now-unused `ctxmeter` imports were removed.
   - The parity fixture in `title_projection_test.go:293-296` now uses Qoder's real shape instead of borrowing Claude's token-bearing record, so the title-projection path is actually tested for Qoder.
   - Keeping `ContextRatio` a plain `float64` keeps `TokenUsage` comparable with `==`. The negative-case test relies on that (`got != (TokenUsage{})`).

2. **Critical findings:** none.

3. **Important findings:** none.

4. **Minor findings**
   - The first Plan item still says "a ratio outside [0, 1]", but the Revisions entry only supersedes "the Design bullets above". This is the same range fact BR-2 covered, so I folded it into the BR-2 disposition note rather than raising it again.
   - The Log says a zero ratio is rejected because it "would render as `0%`". Ratios below 0.005 are accepted and still render as `0%` (the `TokenUsageFigure` test case at 0.004 pins this). That is harmless, but the stated reason does not quite hold.

5. **Test coverage notes**
   - The parser, the last-ratio-wins JSONL fold (including a later sidechain record that must be ignored), `Figure` rounding at 0%, 40% and 100% plus the count fallback, and the title projection are all covered.
   - The sidechain check is reachable: the JSONL test's trailing sidechain record with ratio 0.9 would win without it.

6. **Architectural notes**
   - **ARCH-DRY: pass.** There is one renderer and no copy left behind.
   - **ARCH-PURE: pass.** `ParseTokenUsage` and `Figure` are pure and tested without IO; the call sites only call into them.
   - **ARCH-PURPOSE: pass.** Every consumer of the usage figure goes through `Figure()`. The atlas entry at `architecture.md:382` documents the percentage case. The only open item is the live verification, which cannot be done before landing.

7. **Plan revision recommendations:** optional. Widen the existing Revisions line to cover the Plan's first item as well as the Design bullets.

```findings
dispose:
  - id: BR-2
    disposition: addressed
    note: |
      Revisions entry (2026-10-18, BR-2) supersedes the Design's *float64 and [0, 1] claims, matching usage.go:17 and :105. The Plan's first item still says [0, 1]; widening the Revisions line to cover it would finish the sweep.
  - id: BR-1
    disposition: withdrawn
    note: |
      Evidence holds. TestTitleProjectionProviderAppendParity/qoder, rerun here and passing, asserts HasUsage with Figure 40% through the same Usage.Figure() call that titlepoller/projection.go:53 and couchcmd/titles.go:342 make. A pane title can only show the figure after landing and a Couch restart, so the live half is a post-landing TL check (the unchecked Plan item records it), not a defect in this range.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"3d5649d9-e495-4d89-b542-6da6696776c1","repository":"github.com/xianxu/pair","issue_id":"000654","recipe":"small-diff-review","round":2,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/32d165db8e4e1457","reviewed_head":"d934e9d23fae9ce3b087ad9975fd4fecda292f92","slot":"pair:2","requested_at":"2026-10-18T18:53:15.125099Z"},"phase":"finished","pid":27578,"started_at":"2026-10-18T18:53:15.226665Z","finished_at":"2026-10-18T18:59:50.397702Z","session_id":"37738440-0702-4631-a513-5a0b44259580","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.555916958s
head: "d934e9d23fae9ce3b087ad9975fd4fecda292f92"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
