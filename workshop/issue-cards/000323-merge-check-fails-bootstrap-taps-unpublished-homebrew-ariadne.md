---
id: 000323
status: done
started: 2026-09-24T20:13:00-07:00
created: 2026-09-24
updated: 2026-09-29
estimate_hours:
github_issue:
actual_hours: 0.23
tracker:
    version: 1
    completion:
        token: close-a0d3e5608cc9
        repository: github.com/xianxu/pair
        reviewed_head: a5167cf17eaa23101836522d2f71742f85757996
        evidence_commit: 304747b12c159825fafb5a423f94507a82cc8cdc
        landed_commit: dc6599a51ac717a0b5f8032a3e725d9b6c0dc40e
---

# merge-check fails: bootstrap taps unpublished homebrew-ariadne

## Problem

Every PR's `merge-check` run fails in "Prepare dependencies and compile
composition", before any test runs. `bootstrap.sh` runs
`brew install xianxu/ariadne/weave`; Homebrew clones the tap repo
`github.com/xianxu/homebrew-ariadne`, which does not exist (GitHub API 404, even
authenticated), so the clone dies with `fatal: could not read Username for
'https://github.com': terminal prompts disabled`.

- Last pass: 2026-09-24T02:43Z (`47f909f6`). Every run since fails (9 of the last
  100 at filing time, all consecutive).
- Onset: `aff72f82` "build: adopt ariadne#239 seeded gateway files"
  (2026-09-23 19:45 PT) introduced the brew-tap path in `bootstrap.sh`.
- Publishing the tap is ariadne#241 ("Publish weave and cut over startup"),
  still `open`. pair adopted the gateway before its dependency was published.
- Example failing run: 36088953264 (PR #167).
