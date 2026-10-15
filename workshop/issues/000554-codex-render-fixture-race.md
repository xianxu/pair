---
id: 000554
status: working
deps: []
github_issue:
created: 2026-10-14
updated: 2026-10-14
estimate_hours:
severity: medium
card_mirror: 'bcaaa030a8e7237b6393cb8059eb75280df3804a' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-14T13:27:30-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
---

# Codex notification fixture races emulator Close against reader

## Problem

Race in unchanged files, found by #544's full-package race run. `TestCodexWorkingNotificationIsSuppressedByCouch` calls `codexCouchRendered`, whose deferred `Emulator.Close` (`codex_working_test.go:249`) races the drain goroutine's `Emulator.Read` (`:252`, via `io.Copy`). The race detector reports concurrent access to the emulator's input pipe in `third_party/vt/emulator.go:289/302`.

Reproduced on a clean detached main checkout at e1cab8eca2041604827a7bda25ed958da5eab7af after runtimebundle generation:

```sh
go test -race ./cmd/internal/wrapcmd -run '^TestCodexWorkingNotificationIsSuppressedByCouch$' -count=10
```

It failed in 7.305s with matching Close/Read stacks. The fixture and emulator files are byte-identical between that main and #544. This issue owns the fixture lifetime repair; #544 changes neither file.

## Spec

Make the renderer fixture own and join its reply-drain lifetime before disposing the emulator. Reuse the existing bounded terminal teardown pattern, preserve notification assertions, and do not hide races with sleeps or disabled instrumentation.

## Done when

- The focused command above passes with `-race`.
- Renderer fixture cleanup cannot access an emulator after disposal; its drain terminates within a stated bound on success and failure.
- Existing Codex-working notification assertions still pass.

## Plan

- [ ] Reuse terminalModel for the Codex render fixture; close its reply pipe, join its drainer, then dispose the emulator (ARCH-DRY, ARCH-ORDER).
- [ ] Add deterministic real-model cleanup regressions for successful rendering and an injected shutdown error, retaining exact rendered text and all notification assertions. Bound fixture rendering/cleanup at 10 seconds or the remaining test budget, whichever is less.
- [ ] Run focused repeated race, full wrapcmd normal/race suites, make -k build and sdlc-local suite; close and land on SHIP.

## Log

### 2026-10-14

- Filed from #544 verification. Main baseline e1cab8eca2041604827a7bda25ed958da5eab7af; focused race failure7.305s. Local diagnostic log `/tmp/pair-baseline-codex-working-race.log` is supplementary; the command and stacks above are the durable reproduction. Runtimebundle generation was the only baseline preparation. Temporary checkout removed after the run. TL requested a separate unassigned issue rather than a fix in #544.

- 2026-10-14: Claimed and started from main 141e055b817765a50ab718224bc3d5c72b7769ff after substrate refresh to ariadne 1643ba28edb46a03318f90c89c225d1a33ac13a0. Reproduced focused -race -count=10 failure: fixture deferred Emulator.Close writes closed while reply drainer Emulator.Read reads it. Existing terminalModel already owns safe pipe-close → drain join → emulator disposal; no library or production change is needed. Plan sent to hub for approval.
