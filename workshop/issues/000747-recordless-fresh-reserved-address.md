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
- Land on the real emitted-profile executable-boundary proof; #744 then integrates landed main and owns the native three-worker qualification for both issues, as explicitly sequenced by ops:0.

## Plan

- [x] Pin the actual recordless producer profile, creating start claim and real reserved marker at the launcher boundary; demonstrate the current refusal without manufacturing registration.
- [x] Persist the existing tagged Fresh recovery intent with the new record and use the existing authorized recovery launch tail; preserve ordinary Fresh and all input/ownership guards.
- [ ] Verify exact authority mismatches, failure cleanup, foreign owners, unsupported records and ordinary/resume behavior; qualify the independent #744 native consumer.
- [ ] Update atlas/README as needed, run targeted/race and recorded affected checks, then boundary review/CI and landing. Detailed plan: workshop/plans/000747-recordless-fresh-reserved-address-plan.md.

## Log


- 2026-10-20: closed — Actual emitted-profile RunLaunch boundary RED on original established-address error then GREEN in six cases; real OS helper/record authority and reserved marker, no synthesized registration. Boundary race12.083s; recovery/claim race couchcore8.904s/launcher1.468s; broader recovery20.705s/1.172s, vet, reader inventory8 PASS69.473s; helper-generation and missing-start-intent mutations RED. PR498; recorded full suite queued on e8d003b68beb. no-plan-check retains final local/CI/landing steps and TL-explicit post-landing native744 qualification; per ops747 lands on executable-boundary proof then744 integrates landed main for native3 workers.742 stage retained.; review verdict: SHIP
2026-10-20: #744 native run at clean candidate revision25ac5f6fbf856a337fd03d53108ea4a66f9386ed (both Pair and Couch build metadata verified) reproduced the failure. Genuine steward fixture:0 registered. Its public reboot operation36970b7f-1c25-416e-85d1-22bd8d1af529 failed unconfirmed for fixture:1. The actual target marker was reserved while the steward marker was established. Retained local diagnostics: /Volumes/TestScratch/pc-3857330288/store/launch-diagnostics/failures.json. Full run log /Volumes/TestScratch/tmp/pair-slot5/pair-744-native-ump2qlja/native.log, SHA256 e4ef8082296fb7cbd03b6ad947f4e71fcf8ccf73fdec59256ea5089886c4593a. Likely class seam: startFreshRecoveryWithoutRecord/startFreshSlotWithPrompt passes FreshRequired without the recovery authority needed by createflow.go's EnsureThreadAddress branch; verify rather than weakening RegisterExistingCouchThread.

- 2026-10-20 — Claimed by pair:2 from main while742 remains code-complete, staged and claimed. Root-cause audit confirms recordless startFreshSlotWithPrompt -> claimFreshRecord creates a reserved marker/creating record without RecoveryIntent; launchClaimedThread chooses BuildCouchFreshLaunchProfile from prompt presence, so FreshRequired reaches createflow.go without Recovery authority. RegisterExistingCouchThread correctly refuses reserved. The same missing intent also explains the secondary confirmed-failure cleanup refusal. Existing launchRecoveryFreshAddress already demonstrates tagged Fresh intent publication plus committed-record reread and launchRecoveryClaim reuse. #728's configuration and parked-source admission failures are separate, unproven seams; excluded from747. No production/test code changed; plan submitted to TL before change-code.

- 2026-10-20 — TL approved the plan; change-code admitted quick flow. Producer regression RED: actual recordless child lacks committed recovery authority before fake registration. Implementation commits normalized tagged Fresh intent with both record and creating start, rereads committed record, and uses launchRecoveryClaim/recoveryExistingWorkspace. Removed the now-unused prompt-selected Fresh tail from ordinary launchClaimedThread. No launcher/marker validator changes. Focused recordless + ordinary Fresh cases PASS6.122s; broader recovery/claim and executable-boundary qualification in progress.

## Revisions

- 2026-10-20 — TL settled qualification order with pair:5: land747 on its real emitted-profile executable-boundary proof, then744 integrates landed main and its native three-worker matrix serves as both issues' live proof. No candidate overrides or composite builds:744 adds isolated socket-root derivation absent from plain main, so mixing binaries would invalidate isolation. This changes acceptance sequencing, not implementation or authority semantics.

- Verification so far: equivalent RunLaunch boundary RED on original production with the exact established-address error, GREEN on corrected producer (six cases,10.749s), and helper-generation-validation mutation RED on unauthorized marker advancement. Producer start-intent omission mutation also RED at real launcher authority validation. Focused race recovery/claim tests PASS couchcore8.904s/launcher1.468s; vet PASS. Full targeted packages and final boundary race in progress. No native run or registration success claimed.

- Final boundary/race/inventory verification: six-case actual-profile RunLaunch/OS-marker boundary PASS under race12.083s. All8 reader-inventory checks PASS69.473s after auditing the new boundary reader as disposable local fixture input. Production and tests committed at e8d003b68beb9b52c88e13697bb7d4d65dcd613a; PR498 open. Recorded affected check falls back to full suite for known738 runtime-bundle graph gap and queues FIFO position5; no native override or stage change.
