---
id: 000570
status: open
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
github_issue:
severity: medium
---

# Review restore socket path exceeds the Unix limit under a long TMPDIR

## Problem

The independent pair#568 close review reproduced a separate restore failure with the review checkout’s longer default TMPDIR: the generated review socket endpoint was 117 bytes and Neovim refused it. All five real restore cases passed when the reviewer shortened TMPDIR to the checkout root. This is distinct from #568’s silent exit-1 watcher teardown defect: that defect reproduces under ordinary TMPDIR and is repaired in PR #345.
