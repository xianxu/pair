---
id: 000821
status: open
deps: []
github_issue:
created: 2026-10-22
updated: 2026-10-22
estimate_hours:
collaboration-mode: collaboration
card_mirror: '45d35347373e482ce4528808cfca95181963a8c6' # card fields mirrored from issue-cards; edit via sdlc
---

# Zellij listing omits a live replacement server during attach

## Problem

Native Zellij list-sessions omitted an independently live replacement server during #728 acceptance. Existing #692 conservative admission refused attachment. #775 standalone reproduction is done; no open issue specifically tracking this false-miss occurrence appeared in the current issue list.

## Spec

Preserve evidence of the known Zellij listing false-miss family (upstream zellij#5751). Determine whether a future upstream release or an approved compatibility change resolves the measured class, preserving conservative ownership and attach proof. No blind retries, deadline increases or speculative absence interpretation.

## Done when

- The retained occurrence is correlated with the known listing behavior and its initiating evidence limits are explicit.
- Any proposed product correction has hub approval and a causal regression retaining conservative attach admission.

## Plan

Unclaimed follow-up; no production investigation dispatched by filing. Reuse #775 reproduction and #692 handoff evidence; do not weaken the guard.

## Log

Authenticated ops:0 54a9ef63-f2de-45dd-9a05-a390818006de classifies this as environmental false-miss rather than #728 switch logic and requests filing if no open false-miss issue exists. On source cf4a3aba8197c7ec961c32c62b33f3229eeba073, native runner40889 joined exit1/152.127s. Claude argv/live/parked sequence PASS56.88s; Codex first two switches passed, then third receipt9cfb3b1a-f1da-4f1e-b88e-69d1b1ac0fa4 failed attach-preflight, operation-b221e2fcaffd34145b3defd0c7bee28e. Target session 📁9223725645807222786-5, launch start-63417f1012c7bfd7, root /Volumes/TestScratch/pc-665541824, run c7572a1b3298d5584b2592081ae78e6e. Raw PTY ends with: terminal listing omitted a live server; attach unverified. Production requireListedLiveTerminal returns ErrListingOmittedLiveServer only after independent live-server evidence. All owned generations absent and child/IO joined after failure; private root removed. No server-log tail available because #806 was not landed in this source.

Raw combined output retained at /Users/xianxu/.local/share/pair/evidence/000728/native-final-20261022/run.log; SHA256 cb0b2cf75c161c0ba3afa92f2ff8a197d136fe19e03865dd57a15fbebd4b1418. Both force cases PASS51.41s (Claude24.70s/Codex21.04s), dirty bytes unchanged, owned cleanup confirmed. No rerun or flake registration: whole argument test has not passed on this head. M2 acceptance is explicitly accepted by the hub; M3 real operator check still required.
