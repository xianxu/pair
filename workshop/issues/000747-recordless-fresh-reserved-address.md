---
id: 000747
status: open
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'e2d086aa5983a3a50b38a73d7be4a894e6e25ea0' # card fields mirrored from issue-cards; edit via sdlc
---

# Recordless Fresh worker launch validates a reserved address as established

## Problem

The #744 shared native disposable fleet provisions a genuine numbered worker through real SDLC/Weave, starts a real steward and invokes couch --reboot fixture:1 --confirm --json from that steward's actual inherited environment. Recordless Fresh is admitted but Pair exits before registering: existing Couch thread registration does not match requested established address.

## Spec

Restore genuine recordless Fresh worker startup through its ordinary public authority path. Preserve the reserved/established address distinction and recovery validation; do not bypass checks or synthesize setup/registration records. The #744 harness is an independent consumer from main, not the implementation branch for this bug.

## Done when

- A real or equivalent executable-boundary regression shows an approved recordless Fresh transaction establishes its exact newly reserved address and registers the numbered worker.
- Unproved, mismatched and existing-address attempts still refuse before effects.
- #744's native consumer starts its three non-hub workers without bypasses.

## Plan

- [ ]

## Log

2026-10-20: #744 native run at clean candidate revision25ac5f6fbf856a337fd03d53108ea4a66f9386ed (both Pair and Couch build metadata verified) reproduced the failure. Genuine steward fixture:0 registered. Its public reboot operation36970b7f-1c25-416e-85d1-22bd8d1af529 failed unconfirmed for fixture:1. The actual target marker was reserved while the steward marker was established. Retained local diagnostics: /Volumes/TestScratch/pc-3857330288/store/launch-diagnostics/failures.json. Full run log /Volumes/TestScratch/tmp/pair-slot5/pair-744-native-ump2qlja/native.log, SHA256 e4ef8082296fb7cbd03b6ad947f4e71fcf8ccf73fdec59256ea5089886c4593a. Likely class seam: startFreshRecoveryWithoutRecord/startFreshSlotWithPrompt passes FreshRequired without the recovery authority needed by createflow.go's EnsureThreadAddress branch; verify rather than weakening RegisterExistingCouchThread.
