---
id: 000860
status: working
created: 2026-10-23
updated: 2026-10-24
estimate_hours:
github_issue:
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
