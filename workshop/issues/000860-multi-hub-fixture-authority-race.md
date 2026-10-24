---
id: 000860
status: open
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
card_mirror: 'c4b3214fb979a4f94ea871442ef28f4783171f8b' # card fields mirrored from issue-cards; edit via sdlc
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
