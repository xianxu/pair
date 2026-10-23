---
id: 000836
status: open
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
collaboration-mode: collaboration
card_mirror: '8b179e95beb146dd0c52f0ab61e33891383cfc5f' # card fields mirrored from issue-cards; edit via sdlc
---

# Qualify Qoder slash Return from a trusted checkout

## Problem

The #162 Qoder1.1.78 command-only native qualification reached workspace trust before any input in the pair:2 checkout. Trust was left unchanged. Six-profile normal fixtures and actual Claude/Codex native passes cover #162; ops:0 authorized its close with this conditional Qoder gap and requested this independent unclaimed follow-up.

## Spec

Run a bounded command-only Qoder native Return check from an already-trusted checkout during the operator-approved 07:00–16:00 Pacific window. Never add or modify trust entries. Reuse the existing bounded TTY owner and production composer/Return logic. Use a harmless read-only slash command such as /version; no conversational turn, settings mutation, or permission-question scenario. Confirm the exact command echo, native Return decision, command result and cleanup. If trust or authentication blocks it, stop and record the evidence rather than retrying or changing settings.

## Done when

- A real installed Qoder command-only check from an already-trusted checkout proves positive slash Return and retains version, raw capture and result.
- Existing runtime/output/cleanup bounds remain intact; no trust changes or model conversation are sent.
- Any observed behavior defect is classified with a retained capture before correction; #162's approved grammar and ordinary prose/paths/leading-space/multiline negatives remain covered.

## Plan

- [ ]

## Log

2026-10-23 07:08 Pacific: Qoder1.1.78 on #162 source e8de0bb4082286a2b2bf1dcb4c66100696755516, using a test-only Go overlay to select /version in the existing bounded owner. Ended stage0 at the15s bound (16.136s package total), raw20936 bytes. No command was sent. Final screen explicitly asks whether to trust /Users/xianxu/workspace/worktree/pair-slot2/pair. Cleanup joined normally. The generic classifier reports recognizer-drift because its retained raw-byte trust phrase did not match this rendering; the rendered screen, not that generic label, establishes the trust blocker. No production change or speculative recognizer correction.

Evidence root: /Volumes/TestScratch/tmp/pair-slot2/pair162-qoder-native.9jq5vcma. native.log SHA256 b0e96d576e40990e9e29763d92980608c1dbe042f24ca620d129fb3205ee6b77; native.raw SHA256 e5b925a702044beb7b06d09e25e01baaf46c918332df003988da8ebb0b5ebcb5; probe_test.go SHA256 4d09f577632a3fb1fb3e016217540c5108dbb35cf092068926d8100839cb338c. Overlay changes only the native probe, not production source. Ops:0 authorization2c96c0c9 permits #162 close with the gap recorded and requires this follow-up to remain unclaimed.
