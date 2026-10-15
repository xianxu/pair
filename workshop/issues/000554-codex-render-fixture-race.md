---
id: 000554
status: codecomplete
deps: []
github_issue:
created: 2026-10-14
updated: 2026-10-15
estimate_hours:
severity: medium
card_mirror: 'a565297da2e8b5971835b95f94f1e75718572b17' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-14T13:27:30-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "0f1bb428", done: "05c4ceb9"}
actual_hours: 0.84
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

- [x] Reuse terminalModel for the Codex render fixture; close its reply pipe, join its drainer, then dispose the emulator (ARCH-DRY, ARCH-ORDER).
- [x] Add deterministic real-model cleanup regressions for successful rendering and an injected shutdown error, retaining exact rendered text and all notification assertions. Bound fixture rendering/cleanup at 10 seconds or the remaining test budget, whichever is less.
- [x] Run focused repeated race, full wrapcmd normal/race suites, make -k build and sdlc-local suite; close and land on SHIP.

## Log


- 2026-10-15: closed — Fixture now reuses terminalModel pipe-close/drain-join/disposal ownership; exact screen and notification assertions kept. Deterministic real-model success/close-error regression failed with direct-emulator disposal and passes with shared Close. Focused -race x10 passed; full wrapcmd normal passed (178.650s), full wrapcmd -race passed with explicit 30m package watchdog (949.370s; initial default 10m watchdog expired during exhaustive Muse replay). make -k -B build passed with clean embedded revision 3ae889903862fe5d7944dd490bf29aa5bafa9156. Required sdlc-local/suite passed on that commit (exit 0, 10m39s). No atlas: test/docs-only fixture-lifetime repair, no production or vendored changes.; review verdict: SHIP
### 2026-10-14

- Filed from #544 verification. Main baseline e1cab8eca2041604827a7bda25ed958da5eab7af; focused race failure7.305s. Local diagnostic log `/tmp/pair-baseline-codex-working-race.log` is supplementary; the command and stacks above are the durable reproduction. Runtimebundle generation was the only baseline preparation. Temporary checkout removed after the run. TL requested a separate unassigned issue rather than a fix in #544.

- 2026-10-14: Claimed and started from main 141e055b817765a50ab718224bc3d5c72b7769ff after substrate refresh to ariadne 1643ba28edb46a03318f90c89c225d1a33ac13a0. Reproduced focused -race -count=10 failure: fixture deferred Emulator.Close writes closed while reply drainer Emulator.Read reads it. Existing terminalModel already owns safe pipe-close → drain join → emulator disposal; no library or production change is needed. Plan sent to hub for approval.

- 2026-10-14: Hub approved the quick-flow design. Replaced the raw emulator and unjoined drainer with terminalModel; rendering returns only after Close completes, and preserves close errors. The fixture bounds the combined render/cleanup wait at min(10s, testwait.Budget). Original screen/notification assertions are unchanged. Deterministic real-model regression failed with the former direct-emulator teardown on both lifecycle cases (and lost the injected close error); the shared Close path passes. Exact screen expectation includes all 24 rows. Focused notification/lifecycle -race -count=10 passed (3.918s). Full wrapcmd normal/race checks running. No production or vendored file changed.

- 2026-10-14: make -k -B build passed on clean 3ae889903862fe5d7944dd490bf29aa5bafa9156; Pair/Couch embed that revision with vcs.modified=false. Full wrapcmd normal suite passed (178.650s). Full-package race run reached the default 10m test-binary watchdog while actively executing Muse every-byte-split fixture replay (shortcuts.raw), not the Codex renderer, with no race report. Re-running unchanged assertions under an explicit 30m package watchdog; no fixture deadline or product behavior changed. Required sdlc-local suite is queued. Two progress sends to hub returned uncertain; receipt queries report unavailable, so delivery is unproven and neither was automatically retried.

- 2026-10-15: Full wrapcmd -race suite passed with the explicit 30m package watchdog (949.370s), unchanged assertions. The initial 10m run's active fixture-replay stack was a package validation-budget limit, not a remaining renderer race. Required full local suite remains queued second in line.

- 2026-10-15: Required sdlc local-check run --name suite passed on implementation 3ae889903862fe5d7944dd490bf29aa5bafa9156 (exit 0, 10m39s after 1751s queue), recorded as sdlc-local/suite. All packages passed including wrapcmd (44.188s); full log has no failure records. SHA-256: 2cd13a45709ee44cd59244bdae0f1ca879f28be948337fc53cd5fdee576f179c. No atlas update is needed: the change repairs a test fixture's private lifetime and introduces no production surface.
