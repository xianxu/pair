---
id: 000829
status: open
deps: []
github_issue:
created: 2026-10-22
updated: 2026-10-22
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'b06703403e579cf4d314cc7b1da15f9451e24dbf' # card fields mirrored from issue-cards; edit via sdlc
---

# Flake: startup prewarm resume batch loses caller workspace

## Problem

During #162's complete affected-package verification on Go1.27.3, TestHubResumeBatchSocketAcceptance/startup-prewarm failed: all three in-process CLI resume calls refused with caller workspace unavailable / unknown message workspace; fresh authority proofs remained zero and each target stayed parked. This is a fixture/authority observation, not evidence that shutdown inherited credentials: the test supplies its own testRT map and calls runMessageCLIWithCall in-process.

## Spec

Retain the intermittent evidence and classify the caller-workspace/prewarm ordering before changing production admission. #162 changes only a slash-command help example in couchcmd production; its wrapper/agentdrive behavior is outside the observed resume admission path. No causal fix or waiver is established here.

## Done when

- A bounded deterministic fixture or retained exact-commit failure identifies why the connected caller's workspace was unavailable.
- Fix the proven owner/order defect without weakening workspace authority or warm-only admission.
- The regression and complete affected package pass; qualify a recorded same-commit fail/pass before using known-flake landing treatment.

## Plan

- [ ]

## Log

2026-10-22: complete command was go test ./cmd/internal/agentdrive ./cmd/internal/wrapcmd ./cmd/internal/couchcmd -count=1 -timeout=5m. agentdrive PASS0.252s; wrapcmd PASS116.345s; couchcmd failed100.410s solely at TestHubResumeBatchSocketAcceptance/startup-prewarm. It reported three caller-workspace refusals, zero fresh proofs and three still-parked targets. Source was #162's uncommitted candidate on HEAD e8f14706307b58bebdfdc218116f6ff6a1bf3206; do not represent that HEAD as containing the candidate patch.

Focused classification on unchanged relevant source: go test ./cmd/internal/couchcmd -run '^TestHubResumeBatchSocketAcceptance/startup-prewarm$' -count=5 -v PASS3.500s, all five cases passed0.61–0.67s. Retained log /Volumes/TestScratch/tmp/pair-slot2/pair162-resume-classify.wCgMpq/focused.log. The initial complete-package failure is retained in the originating #162 terminal output. This establishes an intermittent working-tree observation, but no recorded exact-clean-commit suite fail/pass yet; not registered as a known flake. No resume production/test source was changed while classifying it.
