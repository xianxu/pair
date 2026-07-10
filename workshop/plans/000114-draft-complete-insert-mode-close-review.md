# Boundary Review — pair#114

| field | value |
|-------|-------|
| issue | 114 — draft completion should skip outside insert mode |
| repo | pair |
| boundary | whole-issue close |
| window | d04ca665612c9441492121404c6fb1579cb35364..HEAD |
| reviewer | codex |
| timestamps | 2026-07-10T08:03:27-07:00; 2026-07-10T08:08:xx-07:00 |
| verdicts | FIX-THEN-SHIP; FIX-THEN-SHIP |

## First Review

The implementation guarded the shared `run_completers()` path correctly, but the
review asked for two cheap fixes before shipping:

- Keep the production mode guard local instead of exposing a mutable global that
  `run_completers()` depends on.
- Extend the regression beyond direct Normal-mode invocation so it also covers
  Visual mode and scheduled post-Insert execution.

## Resolution

- Moved the mode check inline inside local `run_completers()`.
- Kept only the test seam global `_G.PairDraftCompleteTest.run_completers`.
- Expanded `tests/draft-complete-mode-test.sh` to cover Normal mode, Visual
  mode, and a scheduled callback after leaving Insert mode.

## Second Review

The follow-up review confirmed the runtime fix and expanded regression satisfy
the issue:

- The guard sits at the single shared completion runner before path, word, or
  spell completion can call `vim.fn.complete()`.
- The debounced callback path reaches the same guarded runner.
- The regression uses the real `nvim/init.lua` with a viable buffer completion
  candidate and covers Normal, Visual, and scheduled post-Insert execution.

The only remaining finding was workflow-artifact cleanup: the generated review
sidecar embedded a full review transcript and should be replaced with this
bounded durable summary.

## Verification

- `bash tests/draft-complete-mode-test.sh`
- `bash tests/autopair-test.sh`
- `bash tests/cr-newline-test.sh`
- `make test-lua`
- `make test-draft-complete`
- `git diff --check`
- `make test`
