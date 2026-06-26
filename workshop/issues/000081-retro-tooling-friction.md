---
id: 000081
status: open
deps: []
github_issue:
created: 2026-06-26
updated: 2026-06-26
estimate_hours:
---

# session retro tooling friction

## Problem

The #72 roadmap session exposed several workflow/tooling problems in the agent loop itself. These are separate from #80's larger product idea of automatically analyzing TTY logs; this issue tracks concrete fixes to friction we already observed.

Evidence came from the live Pair TTY log for `PAIR_TAG=2`, rendered with `pair-scrollback-render --plain`. The session completed, but several failures caused avoidable retries, invalid intermediate state, or noisy review loops.

## Spec

Address or explicitly disposition these findings:

1. `sdlc` was not on PATH in the agent shell.
   - The main agent had to discover `/Users/xianxu/workspace/ariadne/bin/sdlc`.
   - Fresh review subprocesses also attempted `sdlc --help` and failed.
   - Desired outcome: SDLC-invoked subprocesses either inherit a PATH containing the SDLC owner `bin/`, or prompts pass the resolved binary path.

2. Parallel `sdlc issue new` caused git/index/push races.
   - The agent invoked multiple issue creations concurrently.
   - Result: `.git/index.lock`, partial issue sync, and a failed push due to concurrent ref updates.
   - Desired outcome: document/guard that mutating `sdlc` commands must not run concurrently, or add a repo-level lock / batch-create path.

3. `sdlc issue validate` accepts only one file.
   - The agent naturally tried to validate #72-#79 in one call and got `accepts at most 1 arg(s), received 8`.
   - Desired outcome: support multiple issue paths, or add an obvious batch wrapper/help example.

4. `sdlc close --no-actual` produced a schema-invalid done issue.
   - Close flipped `status: done` without `actual_hours`.
   - Boundary review caught that `sdlc issue validate` failed.
   - Desired outcome: `sdlc close` must never write an invalid done issue. Either `--no-actual` must still emit a valid sentinel value accepted by the schema, or close should refuse when the schema requires `actual_hours`.

5. Active-time did not measure a real Pair/Codex session.
   - `sdlc actual --issue 72` reported no measurable activity even after the issue had commits.
   - Desired outcome: determine whether active-time misses the current Pair/Codex transcript location/shape, or whether issue-window attribution missed the session; fix or document the limitation.

6. Boundary review output is too bulky for normal operation.
   - Review output filled thousands of TTY lines and was truncated in the pair log.
   - Desired outcome: persist full structured review output to a sidecar path and print a compact verdict/findings summary in the TTY.

7. Boundary review prompt used the wrong repo label.
   - The review prompt said `ariadne#72` for an issue in `pair`.
   - Desired outcome: prompt should identify the repo that owns the issue being reviewed.

## Done when

- [ ] Each finding above has either a fix, a follow-up issue, or a documented won't-fix rationale.
- [ ] Any SDLC behavior change has a regression test in the SDLC owner repo or an equivalent process-level test in Pair.
- [ ] The `sdlc close --no-actual` path can no longer leave a `done` issue schema-invalid.
- [ ] The review prompt repo label is correct for Pair issues.
- [ ] The PATH behavior for fresh SDLC review subprocesses is fixed or explicitly documented.

## Plan

- [ ] Triage each finding into Pair-local vs ariadne/SDLC-owner work.
- [ ] Open cross-repo follow-up issues where the fix belongs in ariadne.
- [ ] Fix Pair-local issues directly if any are in this repo.
- [ ] Add regression coverage for fixed behavior.
- [ ] Re-run a small retro/close workflow to verify the friction is gone or documented.

## Log

### 2026-06-26

Created after the #72 close. The live Pair TTY log was rendered through `pair-scrollback-render --plain`; key signatures included `command not found: sdlc`, git index/ref lock errors during parallel `sdlc issue new`, `issue validate` arity failure, active-time reporting no measurable activity, and a `REWORK` boundary review caused by `close --no-actual` leaving #72 schema-invalid.
