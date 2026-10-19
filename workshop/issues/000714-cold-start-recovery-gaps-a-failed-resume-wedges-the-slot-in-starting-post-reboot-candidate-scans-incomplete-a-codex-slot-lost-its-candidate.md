---
id: 000714
status: open
deps: []
github_issue:
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'fb0fa542f2fe364fcd7adab0bcdf1527b6161ef2' # card fields mirrored from issue-cards; edit via sdlc
---

# Cold-start recovery gaps: a failed resume wedges the slot in starting; post-reboot candidate scans incomplete; a codex slot lost its candidate

## Problem

First real cold start (10-19 ~15:55 hard crash, full machine reboot, 11 working slots down). Recovering the fleet by
`couch --resume-candidates` + `couch --resume` / `--reboot` per slot worked for 9 of 11 on the first try, but surfaced
four gaps:

1. **A failed resume wedges the slot in "starting".** `couch --resume ariadne:2` failed with "Pair exited before
   registering (exit 1)" and recorded no diagnostics (`failures.json` has no entry for nonce start-f65945f19dabb48e).
   Every retry afterwards refused for several minutes with "resume-starting: source is still starting; wait for its
   bounded launch … diagnostics could not be saved: invalid launch failure identity". Only `--reboot` (fresh, losing
   the conversation) got out.
2. **Every candidate scan was incomplete after the reboot:** each slot printed "last activity unknown", "current
   recovery witness names another file generation", "Conversation evidence unavailable", "Candidate scan incomplete".
   Resume still worked, so the warnings were noise and gave the TL nothing to judge by.
3. **pair:3 had no resume candidate at all** (codex, issue pair#712, its suite running at the crash): only "Start
   fresh" was offered, though its transcript should be on disk like the other codex slots'.
4. **No fleet view:** the TL ran 11 `--resume-candidates` calls and joined them with sdlc claims by hand (the "resume
   the fleet" verb is ariadne#456's scope; this issue only needs the per-slot outputs to be trustworthy).

## Spec

## Done when

- A failed resume/launch records its diagnostics and leaves the slot retryable (a test: a launch that exits before
  registering, then a second `--resume` that proceeds instead of refusing as "starting").
- After a reboot, a resumable slot's candidate shows its last activity and no spurious generation/incomplete warning,
  or the warning names what is actually missing (a test from a post-reboot fixture).
- pair:3's missing candidate is explained (root cause in the Log) and fixed, or shown to be a real loss.

## Plan

- [ ]

## Log

### 2026-10-19
