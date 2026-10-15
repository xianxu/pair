---
id: 000570
status: open
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
severity: medium
card_mirror: 'f5224842a306e4dccf56dd437c72eabb051584eb' # card fields mirrored from issue-cards; edit via sdlc
---

# Review restore socket path exceeds the Unix limit under a long TMPDIR

## Problem

The independent pair#568 close review reproduced a separate restore failure with the review checkout’s longer default TMPDIR: the generated review socket endpoint was 117 bytes and Neovim refused it. All five real restore cases passed when the reviewer shortened TMPDIR to the checkout root. This is distinct from #568’s silent exit-1 watcher teardown defect: that defect reproduces under ordinary TMPDIR and is repaired in PR #345.

## Spec

## Done when

- A deterministic long-TMPDIR fixture captures the socket refusal and proves the endpoint construction respects the platform Unix-domain path limit while remaining unique and owned by the exact review process.
- Real draft/opener/review RPC restore succeeds under the original long TMPDIR without a manual environment workaround, timeout increase, retry, skip or weakened identity checks.
- Document lifecycle/cleanup of any alternate socket location and retain bounded, actionable refusal diagnostics.

## Plan

- [ ]

## Log

2026-10-15: Found by the fresh #568 close reviewer at pinned head 8d114ada12f6af3cd52a27ad1aaae235f93c6dd8. Review sidecar: workshop/plans/000568-review-fresh-restore-headless-neovim-exits-without-failure-diagnostics-close-review.md (archived with #568 on landing). Reviewer returned SHIP for #568; this is an independently filed path-envelope follow-up, not a claim that all TMPDIR lengths are covered by that fix. No implementation started.
