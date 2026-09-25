---
id: 000323
status: working
deps: [ariadne#241]
github_issue:
created: 2026-09-24
updated: 2026-09-24
estimate_hours:
started: 2026-09-24T20:13:00-07:00
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

## Spec

Make `merge-check` green again without losing the ariadne#239 gateway shape.
Two ways, to decide at design:

1. **Wait for ariadne#241.** Once `xianxu/homebrew-ariadne` exists and
   `brew install xianxu/ariadne/weave` works from a clean runner, this issue only
   verifies the run goes green. That makes it a dependency, not work in pair.
2. **Interim fallback in pair.** `bootstrap.sh` keeps the brew path but, when the
   tap is unavailable, falls back to the pre-#239 way of getting weave. The
   fallback must be visible in the log (no silent success on a different
   toolchain) and removed once ariadne#241 lands. `bootstrap.sh` is a seeded
   gateway file owned by ariadne, so a pair-local edit may be overwritten by the
   next weave propagation. Check ariadne's seeding rules before choosing this.

Either way, the error today reads like a credentials problem. The real cause is a
missing tap, so a clearer bootstrap message ("tap xianxu/ariadne not published
yet, see ariadne#241") is worth proposing upstream.

## Done when

- A `merge-check` run on a pair PR passes "Prepare dependencies and compile
  composition" and the job completes green.
- If a fallback was added: its log line is visible in the run, and a follow-up
  (or ariadne#241's close) owns removing it.

## Plan

- [ ]

## Log

### 2026-09-24

- Filed from the #321 ship session after the operator noticed every PR run
  failing. Diagnosis via `gh run view --log-failed`, `gh api
  repos/xianxu/homebrew-ariadne` (404), and ariadne's
  `workshop/issues/000241-publish-weave-startup.md` (status open).
