---
id: 000833
status: open
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
collaboration-mode: collaboration
card_mirror: '79446da33c86785af8859ff8719b9ec7fae9244f' # card fields mirrored from issue-cards; edit via sdlc
---

# Preserve shutdown cancellation cause across presenter requests

## Problem

A console owner shutdown can be reported as a terminal failure instead of cancellation. #739 recorded suite on eafb27680f56c799d06d18e7639b15b6e5d154c4 failed only TestRunConsoleObservesStartupBeforeArmingProducers: hubs_startup_test.go:457 console startup=1, stderr "couch: terminal: terminal: presenter unavailable". Attempt598a61ed6ef5c7be79c26f6cf168a978, failed log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/598a61ed6ef5c7be79c26f6cf168a978.log, reported SHA prefix4d2073b0e476, duration11m43s. A focused20-run repeat passed before qualification, so no incidence rate is claimed.

## Spec

Preserve the causal error across presentation requests after a presenter fails during owner shutdown. A cancellation-only shutdown must remain normal; real physical failures, including joined physical failure plus cancellation, must remain failures. Do not suppress all "presenter unavailable" errors or expand deadlines.

Trace: FakeHost.Write makes bytes observable before Presenter.Panel returns. The old ordering fixture called Console.Stop immediately after observing startup bytes; Stop cancels Console.lifetime. An in-flight paint can enter Presenter.fail with context cancellation while the presenter lifetime is still live. Presenter.call rejects a failed presenter with the synthetic "presenter unavailable" error (the queued-request path similarly reports "presenter failed"), dropping the causal cancellation. Console.terminalError cannot classify that synthetic error as cancellation and latches a failure. showMenuPublishing performs further chrome publication queries after the write. The suite captured the synthetic failure; this exact internal interleaving is source-derived, not an instrumented stack capture.

A held-write test control in #739 deterministically proves that the old fixture cancels startup paint before publication completes (RED0.398s, "startup paint cancelled before release"). It did not deterministically reproduce the synthetic unavailable error itself; this follow-up must qualify that final interleaving. Existing couchtty/terminal_exit_test.go cancellation-only/physical-error controls cover a shorter command path and must stay meaningful. Console.Stop documents that it is safe from any goroutine, so this is a shared production diagnostic gap rather than just startup policy.

## Done when

- A deterministic held-paint/next-request regression reproduces loss of the cancellation cause and passes after the root fix.
- Cancellation-only shutdown exits normally, while physical failure and mixed errors remain fatal with the original cause preserved.
- No broad ignored-error text matching, timeout increase, or startup-policy change.

## Plan

- [ ]

## Log

2026-10-23: Hubf9476c31 approved separate unclaimed filing; #739 itself changes its startup-ordering fixture to normal switcher Escape with a held-write control. This follow-up owns shared shutdown error classification. No fix or full reproducer is claimed here.
