---
id: 000033
status: open
deps: []
created: 2026-05-31
updated: 2026-05-31
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

- [ ] Reproduce by interrupting or forcing failure between archive moves and the archive commit.
- [ ] Make `sdlc push` recognize its own prepared archive moves on retry and either commit them or print one exact recovery command.
- [ ] Investigate why the retry judge saw stale/open #31 content when the working tree and `sdlc state` showed done/no drift.
- [ ] Add regression coverage for interrupted archive recovery.

## Log

### 2026-05-31

- Filed from live `sdlc push` recovery during pair#31. The push ultimately
  completed only after manually committing archive moves and rerunning
  `sdlc push --yes --no-judge`.
