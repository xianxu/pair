---
id: 000896
status: open
deps: []
github_issue:
created: 2026-10-25
updated: 2026-10-25
estimate_hours:
card_mirror: '8fb8a28c6c7a9746de120aae51d2e75a05d4d35d' # card fields mirrored from issue-cards; edit via sdlc
---

# couchtty tests retain each Console after the test: RSS grows ~1.25 GB per -count iteration

## Problem

`cmd/internal/couchtty` tests keep each test's Console and its terminal state reachable after the test ends, so one test process grows by about 1.25 GB per `-count` iteration. A `-count=10` proof reached 9.6 GB RSS before the TL stopped it (pair#885 M3, 2026-10-25).

Measured on unmodified main (`4ae4b15e1`), one `couchtty.test` process, `-test.timeout=12m`, `/usr/bin/time -l`:

- `-count=1`: 1.37 GB peak RSS; `-count=2`: 2.61 GB. The pair#885 branch is the same (1.45 / 2.65 GB).
- Heap profile at exit, `-count=2`, `-test.memprofilerate=524288`, `inuse_space`: 975 MB still live, all terminal state: `ultraviolet.NewBuffer` 489 MB, `hostty.NewFakeHost` 175 MB, `terminal.cloneCells` 85 MB, `vt/ansiparser.SetDataSize` 73 MB, `terminal.(*Endpoint).capture` 70 MB, `vt.NewScrollback` 18 MB.
- Spread across many tests, not one: `newRemoteRig` 90 MB, `newWebcastRig` 68 MB, then dozens of Lifecycle/Switcher/Notice/Continuation tests at 5–35 MB each (`TestLifecycleSuccessResultAttentionUsesTypedWarningFacts` 34 MB, `TestSwitcherAutomationWireSplitsAndChildPassThrough` 33 MB, `TestNoDecodedEventKindCanStopThePanelConsole` 30 MB, …).

Likely cause: a goroutine or a package-level reference keeps each Console (and its emulators and fake host) reachable after the test. Not yet traced.

## Spec

Find what keeps a finished test's Console reachable (goroutine dump or a retainer walk at test end) and release it through the Console's own teardown path, so a test process's live heap does not grow with `-count`.

## Done when

- `couchtty.test -test.count=2` peak RSS is within a small margin of `-count=1`, measured the same way.
- A regression test or check fails when a finished test's Console stays reachable.

## Plan

- [ ]

## Log

### 2026-10-25
