# Boundary Review — pair#101 (whole-issue close)

| field | value |
|-------|-------|
| issue | 101 — review nvim search should be case-insensitive (smartcase) |
| repo | pair |
| issue file | workshop/issues/000101-review-nvim-search-should-be-case-insensitive-smartcase.md |
| boundary | whole-issue close |
| milestone | — |
| window | 6957b0ad5d91990a760d273a8523e7a9c45716fe..HEAD |
| command | sdlc close --issue 101 |
| reviewer | claude |
| timestamp | 2026-07-06T14:59:31-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

A trivial, fully-specced config change that delivers exactly its stated purpose: `nvim/review.lua` now sets `ignorecase`+`smartcase` so `/foo` matches case-insensitively while `/Foo` stays case-sensitive. The change matches the Spec/Plan line-for-line, the atlas is updated in the same range, and the test coverage is the good kind — it asserts real search *behavior* (`vim.fn.search`) not just flag state, so a future regression that flips only one option is caught. I ran the suite (`bash tests/review-window-test.sh`) and both new assertions plus the whole file are green (`review-window-test ok`). Nothing blocks SHIP.

**1. Strengths**
- `tests/review-window-test.sh:133-139` — behavioral assertion is genuinely robust. I verified the negative cases logically: only-`ignorecase` → `cs_miss=2≠0` fails; only-`smartcase` (no effect without ignorecase) → `ci_hit=0≠2` fails; neither → both fail. So the one-flag-regression claim in the Log holds.
- Buffer mutation from the smartcase test is safely reset at `tests/review-window-test.sh:154` before downstream diagnostic checks — no state leak into later assertions.
- `nvim/review.lua:21-25` — well-placed in the top `vim.opt` block with an accurate comment about pane-locality (self-contained `nvim -u` init, no draft/user config touched).
- Brings `review.lua` into consistency with `nvim/scrollback.lua:27-28`, which already carries the same idiom.

**2. Critical findings** — none.

**3. Important findings** — none.

**4. Minor findings**
- The `ignorecase`+`smartcase` pair is now duplicated across `nvim/scrollback.lua` and `nvim/review.lua` (and the wrap/linebreak/breakindent lines are similarly repeated). This is **not** a finding — it's the deliberate "each file is the entire, plugin-free, self-contained `nvim -u` init, no shared rtp" architecture stated in both headers. Extracting a shared options module would require the very rtp coupling these files avoid. Noted only so it's on record as intentional.

**5. Test coverage notes**
- Coverage is appropriate and exceeds the minimum: both a flag assertion (`review-smartcase-opt`) and a behavioral one (`review-smartcase-search`). The behavioral test is the real guard and is correctly constructed with the `nW` flags that honor `ignorecase`/`smartcase`. Verified passing on nvim v0.11.7.

**6. Architectural notes**
- ARCH-DRY — **pass.** The two-line repeat across self-contained inits is consistent with the established, deliberate no-shared-rtp pattern (see `scrollback.lua` header rationale). No new duplication introduced that should be consolidated.
- ARCH-PURE — **pass.** This is inherently the thin IO/UI boundary (editor config); there is no business logic to separate. The test exercises real behavior via `vim.fn.search` rather than mocking.
- ARCH-PURPOSE — **pass.** Shadow-sweep: the issue's sole purpose is smartcase search in the review pane; the diff sets both flags and pins the behavior. No deferred "follow-up" that is actually the point; nothing under-delivered.

**7. Plan revision recommendations** — none. The plan checkboxes (`review.lua` opts, `review-smartcase-opt`+`review-smartcase-search` assertions, negative-test verification) all correspond to delivered code, and I confirmed the tests pass. No `## Revisions` entry needed.

Note on the Docs gate: no README update is warranted — this changes a default search behavior of a standard vim `/`, not a new user-facing keybinding, subcommand, flag, or config key. Atlas coverage (`atlas/review-workbench.md`) is the correct and sufficient doc surface, and it was updated.
