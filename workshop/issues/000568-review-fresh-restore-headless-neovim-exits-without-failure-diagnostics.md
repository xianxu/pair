---
id: 000568
status: open
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
card_mirror: '272924a33afb6194d8f7f0dba3b8ba0076129c69' # card fields mirrored from issue-cards; edit via sdlc
---

# Review fresh-restore headless Neovim exits without failure diagnostics

## Problem

During pair#556 M1 recorded verification, the unchanged shell test tests/review-fresh-restore-test.sh failed at missing-history-does-not-adopt-old-receipt. The Python assertion saw a nonzero headless Neovim return with empty combined output; its assertion omits the return code, so the retained log cannot distinguish a signal from ordinary exit. The isolated full script immediately passed all five cases on the same code. This is not a confirmed root cause or a #556 recovery regression.

## Spec

## Done when

- The failing assertion records return code, bounded Neovim output and enough per-case state to identify a failure without retaining private/user state.
- Repeated portable stateful runs reproduce or bound the failure; fix its demonstrated cause rather than increasing a timeout.
- The test still exercises the real draft/Go opener/review RPC boundary, and temporary children are cleaned by exact identities.

## Plan

- [ ]

## Log

2026-10-15: recorded suite at db86a287f8c74bbda952dfe00356e3b8f11d4601 exited 2 after 3m39s, before the Go phase. Retained common-Git log .git/sdlc/test-stats/failed/eef944b68fc13088226c6a06a1a85b99.log; SHA256 prefix92b0a48d20e5. Earlier cases fresh-restores-history and wrong-conversation-ack-refused passed. Isolated command bash tests/review-fresh-restore-test.sh subsequently passed all five cases. No changes to review editor/opener files in the #556 correction window.
