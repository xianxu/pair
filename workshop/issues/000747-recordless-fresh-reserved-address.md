---
id: 000747
status: working
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'e920f3ab6ad347ff5b92be068d6a10e3a10abe94' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-20T17:17:34-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "1ca39027", done: "018d5795"}
---

# Recordless Fresh worker launch validates a reserved address as established

## Problem

The #744 shared native disposable fleet provisions a genuine numbered worker through real SDLC/Weave, starts a real steward and invokes couch --reboot fixture:1 --confirm --json from that steward's actual inherited environment. Recordless Fresh is admitted but Pair exits before registering: existing Couch thread registration does not match requested established address.

## Spec

Restore genuine recordless Fresh worker startup through its ordinary public authority path. Preserve the reserved/established address distinction and recovery validation; do not bypass checks or synthesize setup/registration records. The #744 harness is an independent consumer from main, not the implementation branch for this bug.

The recordless transaction commits a genuine existing RecoveryFresh intent for its newly allocated address and creating start claim before helper launch. The launcher receives the existing RecoveryAuthority locator and revalidates that committed record, exact helper identity, attempt nonce, revision, agent and cwd before advancing the reserved marker. Reuse launchRecoveryClaim and recoveryExistingWorkspace; do not add a marker exception or another recovery protocol. Keep the descriptive branch/issue orientation and unavailable-screen notice. Ordinary Fresh, cold resume, selected terminal ownership, one-start admission and existing rollback/uncertainty rules retain their contracts.

Actions: start fresh explicitly; inspect an unconfirmed attempt; retry only after existing recovery policy proves it safe.

| Internal state / failure mode | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| New reserved address, committed Fresh intent, matching acknowledged helper | Fresh worker starts and registers | Inspect started worker | Prior unavailable context is not restored |
| Missing/mismatched intent, nonce, helper, agent, cwd or marker | Refused before marker establishment/native launch | Inspect refusal | None; retained evidence remains |
| Helper fails and cleanup proves no surviving owner | Existing confirmed-failure recovery result | Retry through normal explicit recovery | No new conversation established |
| Helper or terminal effect remains unproved | Unconfirmed attempt with existing diagnostic | Inspect attempt; no blind retry | A worker may already have started |
| Foreign/unknown prior owner remains | New address starts beside untouched remnant | Inspect started worker and remnant | Old unavailable context may remain inaccessible |

## Done when

- A real or equivalent executable-boundary regression shows an approved recordless Fresh transaction establishes its exact newly reserved address and registers the numbered worker.
- Unproved, mismatched and existing-address attempts still refuse before effects.
- #744's native consumer starts its three non-hub workers without bypasses.

## Plan

- [ ] Pin the actual recordless producer profile, creating start claim and real reserved marker at the launcher boundary; demonstrate the current refusal without manufacturing registration.
- [ ] Persist the existing tagged Fresh recovery intent with the new record and use the existing authorized recovery launch tail; preserve ordinary Fresh and all input/ownership guards.
- [ ] Verify exact authority mismatches, failure cleanup, foreign owners, unsupported records and ordinary/resume behavior; qualify the independent #744 native consumer.
- [ ] Update atlas/README as needed, run targeted/race and recorded affected checks, then boundary review/CI and landing. Detailed plan: workshop/plans/000747-recordless-fresh-reserved-address-plan.md.

## Log

2026-10-20: #744 native run at clean candidate revision25ac5f6fbf856a337fd03d53108ea4a66f9386ed (both Pair and Couch build metadata verified) reproduced the failure. Genuine steward fixture:0 registered. Its public reboot operation36970b7f-1c25-416e-85d1-22bd8d1af529 failed unconfirmed for fixture:1. The actual target marker was reserved while the steward marker was established. Retained local diagnostics: /Volumes/TestScratch/pc-3857330288/store/launch-diagnostics/failures.json. Full run log /Volumes/TestScratch/tmp/pair-slot5/pair-744-native-ump2qlja/native.log, SHA256 e4ef8082296fb7cbd03b6ad947f4e71fcf8ccf73fdec59256ea5089886c4593a. Likely class seam: startFreshRecoveryWithoutRecord/startFreshSlotWithPrompt passes FreshRequired without the recovery authority needed by createflow.go's EnsureThreadAddress branch; verify rather than weakening RegisterExistingCouchThread.

- 2026-10-20 — Claimed by pair:2 from main while742 remains code-complete, staged and claimed. Root-cause audit confirms recordless startFreshSlotWithPrompt -> claimFreshRecord creates a reserved marker/creating record without RecoveryIntent; launchClaimedThread chooses BuildCouchFreshLaunchProfile from prompt presence, so FreshRequired reaches createflow.go without Recovery authority. RegisterExistingCouchThread correctly refuses reserved. The same missing intent also explains the secondary confirmed-failure cleanup refusal. Existing launchRecoveryFreshAddress already demonstrates tagged Fresh intent publication plus committed-record reread and launchRecoveryClaim reuse. #728's configuration and parked-source admission failures are separate, unproven seams; excluded from747. No production/test code changed; plan submitted to TL before change-code.
