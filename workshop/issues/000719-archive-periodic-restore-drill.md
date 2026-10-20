---
id: 000719
status: open
deps: []
github_issue:
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
collaboration-mode: collaboration
card_mirror: '721c7d5d0cd8e7e49ab43a5516a7e4a3e74ec222' # card fields mirrored from issue-cards; edit via sdlc
---

# history backup: periodic operator-present Drive restore drill

## Problem

#689 delivered find, sync-index and verified restore, and the operator completed a real Drive restore drill. A repeatable command is needed so future periodic checks exercise recovery rather than merely confirming uploads exist.

## Spec

Add an operator-present `pair archive drill` command. Select a random eligible archived UTC day, restore its bundles from Drive into a fresh private owned scratch directory, fully verify ciphertext and archived members against their labels, and compare restored files with retained originals when those originals remain available. Report verified restoration separately from original comparison; missing originals must never count as a successful original comparison. Keep selection, IO and randomness behind seams usable by a stateful fake.

The drill owns its scratch for its lifetime and deletes its decrypted scratch on success, failure and cancellation, with an explicit diagnostic if cleanup cannot complete. Never delete pre-existing or foreign directories, change agent roots, or expose decrypted contents in logs. Retain only bounded metadata for the outcome. Apply explicit limits to selection, downloads, extraction and execution time using the existing restore mechanisms where suitable.

GPG needs the operator's passphrase through ordinary pinentry: run with the operator present, for example monthly. Do not store a passphrase or add unattended key handling. Make the command suitable for the coming Couch schedule to invoke or prompt for; the scheduler owns recurrence rather than a second timer in the drill command.

## Done when

- The public command chooses a random eligible day, restores from Drive, verifies against labels, compares available originals, reports missing-original uncertainty honestly, and deletes its owned plaintext scratch.
- Stateful fake tests exercise selection, failures, cancellation, verification mismatch and cleanup ownership, plus a throwaway-GPG-key round trip through the production seam.
- The operator completes a real Drive drill while present; docs explain normal pinentry and an example monthly schedule without claiming unattended verification.

## Plan

## Log

### 2026-10-19

Filed unclaimed at the hub's request after #689 landed as PR450 (merge df16e5f535d3fabc46044958030c8f1e1280f007). #689's operator drill restored UTC day 2026-10-16: 63 labels indexed, 2 bundles selected (2,127,158 ciphertext bytes), 339 files and 101,518,369 raw bytes restored; restored/original SHA-256 matched, staging was absent, and the operator/hub deleted the plaintext scratch. This follow-up automates that recovery exercise for an operator-present periodic run.
