---
id: 000724
status: open
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
github_issue:
flake_tests: [github.com/xianxu/pair/cmd/internal/wrapcmd.TestTerminalModelControlObserverBoundsStringStorage, github.com/xianxu/pair/cmd/internal/wrapcmd.TestWrapperTitleLeaseNotInherited]
---

# flake: TestTerminalModelControlObserverBoundsStringStorage, TestWrapperTitleLeaseNotInherited

## Problem

Known flaky tests (#394), registered with same-head fail→pass evidence:

- github.com/xianxu/pair/cmd/internal/wrapcmd.TestTerminalModelControlObserverBoundsStringStorage: Same exact head 44d7bdccf8124f25d15082a7286ba1c0ff0f4a46, no source edits: broad go test ./cmd/internal/couchcmd ./cmd/internal/couchmessage ./cmd/internal/wrapcmd -count=1 failed wrapcmd after196.827s (session62009): BoundsStringStorage large OSC allocations26 versus small6; TitleLeaseNotInherited timed out3.07s before leased title appeared. Focused go test ./cmd/internal/wrapcmd -run ^(TestTerminalModelControlObserverBoundsStringStorage|TestWrapperTitleLeaseNotInherited)$ -count=3 -v then passed both three times in3.814s (session76552). Source and test files are unchanged from origin/main. This is observed fail-to-pass evidence, not a root-cause diagnosis; investigate allocation measurement interference and process startup budget under concurrent machine load. Recorded full suite on this head remains queued; no suite failure has been waived.
- github.com/xianxu/pair/cmd/internal/wrapcmd.TestWrapperTitleLeaseNotInherited: Same exact head 44d7bdccf8124f25d15082a7286ba1c0ff0f4a46, no source edits: broad go test ./cmd/internal/couchcmd ./cmd/internal/couchmessage ./cmd/internal/wrapcmd -count=1 failed wrapcmd after196.827s (session62009): BoundsStringStorage large OSC allocations26 versus small6; TitleLeaseNotInherited timed out3.07s before leased title appeared. Focused go test ./cmd/internal/wrapcmd -run ^(TestTerminalModelControlObserverBoundsStringStorage|TestWrapperTitleLeaseNotInherited)$ -count=3 -v then passed both three times in3.814s (session76552). Source and test files are unchanged from origin/main. This is observed fail-to-pass evidence, not a root-cause diagnosis; investigate allocation measurement interference and process startup budget under concurrent machine load. Recorded full suite on this head remains queued; no suite failure has been waived.
