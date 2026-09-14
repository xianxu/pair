# Boundary Review — pair#249 (whole-issue close)

| field | value |
|-------|-------|
| issue | 249 — Fix continuation restart for Couch-hosted Pair threads |
| repo | pair |
| issue file | workshop/issues/000249-couch-continuation-restart.md |
| boundary | whole-issue close |
| milestone | — |
| window | 7800e9682e02b59e4b8cd299718b9a08d8e975ed..f5fa755b681606298e6d6e6144e0e4f1a82776d3 |
| command | sdlc close --issue 249 |
| reviewer | codex |
| timestamp | 2026-09-14T12:38:59-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation preserves exact checkpoints, delegates hosted teardown to Couch, and retains recoverable request state. All eight affected-package test suites passed. Two findings remain: a Core-concepts classification contradiction, which this review contract makes blocking, and an asynchronous focus race.

1. **Strengths**

   - Immutable checkpoint snapshots and digest validation protect cross-worktree handoffs.
   - Existing park, tracked-launch, registration, and orientation protocols are reused.
   - Recovery tests cover registration interruption, target death, source reattachment, and retained failure state.
   - Console tests cover both failure/source-exit orders and queue saturation. README and atlas describe the new recovery surface.

2. **Critical findings**

   - [Plan:53](/Users/xianxu/workspace/pair/workshop/plans/000249-couch-continuation-restart-plan.md:53): The Core-concepts table marks `console_switchagent.go` and `switchcontext.go` **modified**, but neither changes in the pinned range. They are reused unchanged. Append a revision correcting this classification and identifying the new callers. This is a traceability blocker under the explicit Core-concepts contract, not a runtime defect.

3. **Important findings**

   - [console_continuation.go:125](/Users/xianxu/workspace/pair/cmd/internal/couchtty/console_continuation.go:125), [console.go:2055](/Users/xianxu/workspace/pair/cmd/internal/couchtty/console.go:2055): `PreserveFocus` captures focus at enqueue time. If the source was focused, acceptance opens the panel; the operator can then select another actor while replacement runs. Completion still attaches the replacement in the foreground, overriding that newer choice. **ARCH-ORDER:** track intervening focus changes and adopt in the background when one occurs. Add a deterministic acceptance → operator switch → completion test.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Full tests passed for `checkpoint`, `threadrecord`, `continuationcmd`, `launcher`, `couchcore`, `couchtty`, `couchcmd`, and `artifactpath`. The pinned diff passes `git diff --check`. Existing focus tests vary focus before acceptance; they do not cover switching during execution. Live Zellij and paid-agent smoke were not rerun.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared checkpoint model and existing lifecycle machinery.
   - **ARCH-PURE — pass:** validation and request transitions separated from IO.
   - **ARCH-PURPOSE — pass:** hosted, warm, cross-worktree, and standalone paths addressed.
   - **ARCH-MOCK — pass:** portable stateful fixtures and scheduled conformance coverage.
   - **ARCH-CONSTRAINTS — pass:** bounded snapshots, submission timeout, and scoped polling.
   - **ARCH-SECURE — pass:** digest, source generation, and process ownership checks.
   - **ARCH-ORDER — flag:** completion can override newer operator focus.
   - **ARCH-FUNERAL — pass:** bounded request slot, derived-file archive cleanup, and Console-owned worker lifetime.

7. **Plan revision recommendations**

   Append a dated `## Revisions` entry identifying orientation delivery as unchanged reuse. Record the focus-ordering correction and its regression test.

```findings
findings:
  - id: new
    severity: Critical
    family: core-concepts-match-diff
    title: |
      Core-concepts table marks unchanged orientation files as modified
    detail: |
      workshop/plans/000249-couch-continuation-restart-plan.md:53 lists console_switchagent.go and switchcontext.go as modified, but neither changes in the pinned range. Append a revision identifying unchanged reuse and the new callers; the explicit Core-concepts contract makes this contradiction blocking.
  - id: new
    severity: Important
    family: async-completion-preserves-user-intent
    title: |
      Continuation completion overrides an intervening operator focus change
    detail: |
      console_continuation.go:125 captures PreserveFocus at enqueue time, and console.go:2055 uses it after asynchronous replacement. Selecting another actor while replacement runs is overridden by foreground adoption. ARCH-ORDER: preserve intervening focus changes and test acceptance, operator switch, then completion.
```
