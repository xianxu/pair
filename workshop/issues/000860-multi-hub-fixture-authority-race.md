---
id: 000860
status: working
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-24
estimate_hours:
card_mirror: '5a04c92d804322cec1434d02d8be35cc0e113b7c' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-24T00:22:54-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:9
    worktree: /Users/xianxu/workspace/worktree/pair-slot9/pair
    repository: github.com/xianxu/pair
---

# Multi-hub test fixture mutates authority after message readers start

## Problem

A broader targeted race run during #812 integration failed TestHubNotifyNamesExtraHubAndBoundsCustomTitle: multiHubWaitRig starts newWaitRig (and the message service registry loop) before mutating authority.hub and authority.hubs. messageService.execute concurrently reads those fields. This is a fixture race already present in baseline, separate from #812 notice behavior.

Captured race frames:

```text
Write: multiHubWaitRig supervision_test.go:428 (authority.hub)
Write: multiHubWaitRig supervision_test.go:429 (authority.hubs)
Caller: TestHubNotifyNamesExtraHubAndBoundsCustomTitle notify_service_test.go:142
Previous read: messageService.execute message_service.go:900
Read caller: messageService.loop.func1 :836 -> messageService.loop :860
Reader created: newMessageService :727 -> serviceRig.start message_service_test.go:244
 -> serviceRig.restart :218 -> newWaitRig wait_notice_test.go:328
 -> multiHubWaitRig supervision_test.go:414
--- FAIL: TestHubNotifyNamesExtraHubAndBoundsCustomTitle (0.02s)
    testing.go:1867: race detected during execution of test
FAIL couchcmd 83.453s
```

The two test files and the entire execute function are byte-identical across branch base c0dbb32, integrated main cd347333, branch 70185310 and origin/main aafc7c8. This is source comparison, not an executed baseline pass. The broad Hub test expression selected this case; focused HubStartup tests do not. No matching active issue found at filing.

## Spec

Build the final multi-hub authority before starting its service readers, using the existing fixture/composition seam. Preserve the notification routing and bounded-title assertions; no production lock or timing sleep should be needed for fixture initialization.

## Done when

- The captured writer/reader initialization race is prevented structurally.
- The exact notification test and other users of multiHubWaitRig pass repeated race testing without weakening their assertions.
- A bounded suggested repro is go test -race ./cmd/internal/couchcmd -run '^TestHubNotifyNamesExtraHubAndBoundsCustomTitle$' -count=20; this exact isolation has not yet been run by the filer.

## Plan

- [ ]

## Log

Observed during #812 targeted integration checks. Full recorded #812 suite failure on70185310 was a different stale startup display oracle; this race was found only by the broader targeted race run. Unclaimed follow-up; not fixed under #812.

### 2026-10-24 — Claimed and reproduced before implementation

Claimed by pair:9 after #825 and #841 landed, on fresh main99846a5015f753f05edaf4778618849f377a2f71. Start-plan refreshed the substrate and created the issue branch; no generated vocabulary dirt remains. Ran runtimebundle-generate, then the exact notification test under race with count20 and timeout120s. It failed in1.012s with both multiHubWaitRig authority writes racing messageService.execute at896; raw evidence is /tmp/pair860-baseline-race.log. No implementation changes were made before this RED.

The service publishes its connected snapshot before execute finishes reading authority, so waitConnected is not a completion barrier for changing immutable configuration. Existing serviceRig.decorateAuthority runs before newMessageService and survives restart. A read-only fixture audit recommends passing that decorator through newWaitRig and constructing multi-hub ownership before invoking it. A deterministic startup-policy regression can independently require automation status for both configured hubs: late replacement currently leaves /repo3 unknown because startup seeded only /repo1.

The same audit enumerated analogous already-connected fixture assignments in hub_admission_test, hub_guards_test, hub_mutation_payload_test, recovery_confirmation_test, archive_notify_test and lifecycle_notices_test. Quiescent pre-wrapper assignments and local authority/service literals are distinct. Scope and enforcement are being included in the plan to the hub before code; no production lock, timing sleep, retry or assertion weakening is proposed (ARCH-ORDER, ARCH-DRY, ARCH-PURPOSE).
