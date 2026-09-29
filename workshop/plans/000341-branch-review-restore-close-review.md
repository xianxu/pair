# Boundary Review — pair#341 (whole-issue close)

| field | value |
|-------|-------|
| issue | 341 — Alt+C restores review target from branch |
| repo | pair |
| issue file | workshop/issues/000341-branch-review-restore.md |
| boundary | whole-issue close |
| milestone | — |
| window | 41ab4f9add1908e854c3f41e491e4a64d53793db..af18ceaad04fb3862bf803f679b6435c78c060f3 |
| command | sdlc close --issue 341 |
| reviewer | codex |
| timestamp | 2026-09-28T21:33:20-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation covers branch restoration, authenticated activation, and bounded recovery well. A reproduced handoff-loss race blocks shipping. Routine typing also performs synchronous history scans, and the new operator commands are missing from README.

1. **Strengths**

   - The shared Go resolver pins HEAD, validates exact round subjects and safe paths, and bounds history reads.
   - Real-process tests verify A → B → A restoration, fresh-session isolation, and recovery across process exit.
   - Activation preserves retained buffers and restores the previous owner when setup fails.
   - Recovery snapshots have private permissions, atomic replacement, capacity limits, and explicit removal rules.

2. **Critical findings**

   - **Handoff consumption precedes final acceptance.** [handoff.lua:90](/Users/xianxu/workspace/pair/nvim/review/handoff.lua:90) deletes the payload before calling `on_agent_round`, whose [authorization check](/Users/xianxu/workspace/pair/nvim/review/init.lua:127) can still refuse. A checkout change between these checks loses the response without applying it. A controlled probe using the production functions produced `payload_exists=false text=before`. Require explicit apply/defer acceptance before consuming the payload; preserve it on refusal. **ARCH-ORDER, ARCH-PURPOSE.**

3. **Important findings**

   - **Typing synchronously scans Git history.** [review.lua:848](/Users/xianxu/workspace/pair/nvim/review.lua:848) runs recovery checks on every `TextChanged`/`TextChangedI`. Modified buffers reach `guard → resolve →` [identity.lua:23](/Users/xianxu/workspace/pair/nvim/review/identity.lua:23), blocking Neovim for up to 2.5 seconds per resolver call. Use asynchronous observation for proactive recovery, retaining authoritative checks at write/apply boundaries. Add a delayed-resolver responsiveness test. **ARCH-CONSTRAINTS.**
   - **README update missing.** [README.md:139](/Users/xianxu/workspace/pair/README.md:139) retains the old target-based Alt+C description and omits `:PairReviewRecover` and `:PairReviewDiscardRecovery`. Document branch restoration, blocked transitions, and recovery commands alongside the existing atlas update.

4. **Minor findings**

   - [review.lua:790](/Users/xianxu/workspace/pair/nvim/review.lua:790) registers rendering autocmds on every activation without removing earlier registrations. A real A → B → A probe increased A’s `TextChanged` callback count from one to two. Use an owned, clearable group and test stable registration counts. **ARCH-FUNERAL.**

5. **Test coverage notes**

   Passed: Go review package tests; all six new Lua review test modules; branch-restoration, fresh-restoration, and producer-context acceptance tests.

   `make test-lua` stopped in `scrollback_test.lua` with an editor-storage sandbox permission error. The full suite was therefore not verified. Existing handoff tests cover refusal before admission, but miss refusal after admission and before application.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared identity resolver replaces independent history interpretation.
   - **ARCH-PURE — pass:** declared pure entities are separated from integration code.
   - **ARCH-PURPOSE — flag:** handoff loss violates preservation of pending work.
   - **ARCH-MOCK — pass:** stateful host/producer fixtures and real Git/Neovim tests exercise integration boundaries.
   - **ARCH-CONSTRAINTS — flag:** synchronous history scans enter the typing path.
   - **ARCH-SECURE — pass:** identity, paths, context, and recovery storage receive explicit validation.
   - **ARCH-ORDER — flag:** deletion occurs before final acceptance.
   - **ARCH-FUNERAL — flag:** activation-owned autocmd registrations accumulate.

7. **Plan revision recommendations**

   Append a timestamped `## Revisions` entry specifying acceptance-before-consumption, asynchronous recovery observation, and autocmd ownership. Add their regression cases and README coverage to the active checklist.

```findings
findings:
  - id: new
    severity: Critical
    family: consume-after-acceptance
    title: |
      Handoff deletion precedes final application authorization
    detail: |
      nvim/review/handoff.lua:90 deletes before the callback; nvim/review/init.lua:127 can subsequently refuse after checkout movement. A controlled production-function probe confirmed payload loss without application. Consume only after explicit apply/defer acceptance and test this interleaving (ARCH-ORDER, ARCH-PURPOSE).
  - id: new
    severity: Important
    family: nonblocking-editor-observation
    title: |
      Routine typing synchronously resolves full Git history
    detail: |
      nvim/review.lua:848 routes TextChanged/TextChangedI through preserve_mismatch and guard to nvim/review/identity.lua:23, synchronously waiting up to 2.5 seconds per resolver call. Make proactive recovery observation asynchronous and test responsiveness with delayed resolution (ARCH-CONSTRAINTS).
  - id: new
    severity: Important
    family: user-surface-documentation
    title: |
      README omits branch restoration and recovery commands
    detail: |
      README.md:139 retains the previous target-based Alt+C description. This range introduces PairReviewRecover and PairReviewDiscardRecovery without any README update; document the changed behavior and recovery workflow.
  - id: new
    severity: Minor
    family: activation-resource-ownership
    title: |
      Repeated activation accumulates rendering autocmds
    detail: |
      nvim/review.lua:790 registers callbacks on every activation without corresponding cleanup. A real A-to-B-to-A probe increased A's TextChanged callback count from one to two. Use a clearable owned group and assert stable counts (ARCH-FUNERAL).
```
