---
id: 000033
status: done
deps: []
created: 2026-05-31
updated: 2026-06-01
estimate_hours: 1.5
actual_hours: 1.2
---

# sdlc push archive retry is not idempotent

## Done when

- `sdlc push` can resume or give an exact next-action after its archive step is interrupted after moving issue files but before committing them.
- Retry judges evaluate the real current tracker state and do not report stale/open issue content for archived done issues.
- The recovery path does not require manual `git add` / archive commits outside `sdlc`.

## Spec

During a `sdlc push --yes` for pair#31, the pre-merge judges passed and the code
commits pushed. The archive phase moved completed issues from `workshop/issues/`
to `workshop/history/`, then failed while creating the archive commit because
the sandbox blocked writing `.git/index.lock`.

After rerunning `sdlc push --yes`, the command refused immediately because the
archive files it had just moved were now untracked. After manually staging and
committing those archive moves, another `sdlc push --yes` retry ran judges
against the archive commit and reported that #31 was archived while still
`status: open` with an empty plan. Local inspection and `sdlc state` both showed
#31 was actually `status: done` and no tracker drift existed.

The SDLC bug is not the sandbox denial itself; it is the non-idempotent recovery
after an interrupted archive step, plus the retry judge seeing stale or incorrect
issue content.

## Plan

- [x] Reproduce by interrupting or forcing failure between archive moves and the archive commit.
- [x] Make `sdlc push` recognize its own prepared archive moves on retry and either commit them or print one exact recovery command.
- [x] Investigate why the retry judge saw stale/open #31 content when the working tree and `sdlc state` showed done/no drift.
- [x] Add regression coverage for interrupted archive recovery.

## Log

### 2026-05-31

- Filed from live `sdlc push` recovery during pair#31. The push ultimately
  completed only after manually committing archive moves and rerunning
  `sdlc push --yes --no-judge`.

### 2026-06-01

- Root cause: interrupted archive leaves `D workshop/issues/...` plus
  untracked `workshop/history/...`. `sdlc push` checked generic untracked files
  before archive handling, so retry refused before it could stage the move. When
  manually committed, the retry judge reviewed an archive-only issue deletion
  diff rather than simply finishing the archive operation.
- Fixed in ariadne `cmd/sdlc/push.go`: before the untracked guard, `sdlc push`
  now scans `git status --porcelain --untracked-files=all` for exact prepared
  archive pairs, verifies the history copy has terminal status, stages
  `workshop/issues/` + `workshop/history/`, commits `archive completed issues to
  history`, pushes, and exits. Mixed unrelated dirty files still refuse with a
  concrete next action.
- Added regression coverage in ariadne `cmd/sdlc/push_test.go` for unstaged
  archive-move detection, non-terminal history rejection, and recovery
  stage/commit/push calls.
- Verification: `go test ./cmd/sdlc`, `go test ./...` in ariadne; throwaway git
  repo dry-run showed the recovery plan before the untracked guard; throwaway
  git repo with a bare origin successfully committed and pushed the prepared
  archive move. Rebuilt pair's downstream `bin/sdlc` with `make sdlc-build`.
- Closed: ariadne `go test ./cmd/sdlc` and `go test ./...` pass; throwaway git
  repos verify interrupted archive dry-run recovery and real commit/push
  recovery; pair `bin/sdlc` rebuilt.
