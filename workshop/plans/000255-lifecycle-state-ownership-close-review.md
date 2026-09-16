# Boundary Review — pair#255 (whole-issue close)

| field | value |
|-------|-------|
| issue | 255 — Establish a faithful terminal abstraction for Couch and Pair |
| repo | pair |
| issue file | workshop/issues/000255-lifecycle-state-ownership.md |
| boundary | whole-issue close |
| milestone | — |
| window | b11ab67ff1d290386cf12177f2dacdccbc8551c0..78922625e5db4ecf21974ccbe7b0501caa51da0e |
| command | sdlc close --issue 255 |
| reviewer | codex |
| timestamp | 2026-09-15T18:34:01-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

BR-22 is addressed, and the inspected terminal ownership and consumer integration are sound. Focused tests and race tests pass. One Important delivery gap blocks closure: the plan claims native terminal conformance is wired into CI, but the committed workflow never runs those tests.

1. **Strengths**

   - Both consumers use the shared Presenter to order presentation and input admission.
   - Endpoint publications preserve coherent frame/history state; child teardown joins publication and input workers.
   - Notification output preserves framing and accounts for accepted bytes.
   - BR-22’s corrected documentation matches the implemented broker route.

2. **Critical findings**

   None.

3. **Important findings**

   **Native terminal conformance is absent from CI.** [Plan:107](/Users/xianxu/workspace/pair/workshop/plans/000255-terminal-abstraction-plan.md:107) marks this delivered. However, [workflow:108](/Users/xianxu/workspace/pair/.github/workflows/couch-zellij-conformance.yml:108) invokes [Makefile.local:83](/Users/xianxu/workspace/pair/Makefile.local:83), which runs only launcher, couchcore and couchcmd lifecycle suites. The new [native test:30](/Users/xianxu/workspace/pair/cmd/internal/couchtty/terminal_native_test.go:30) requires `PAIR_LIVE_COUCH_NATIVE=1` and a candidate binary; no CI entrypoint supplies them.

   Wire the native terminal tests into CI with dependencies, a freshly built candidate, explicit enablement and relevant source-path triggers. Verify the tests execute rather than skip. **ARCH-PURPOSE, ARCH-MOCK.**

4. **Minor findings**

   None remaining.

5. **Test coverage notes**

   Passed focused terminal, qualification, notification, wrapper and fork tests. Race tests passed for terminal, ttyio, notifytransport, ptychild, termcmd and couchtty. Native conformance and extended soaks were not rerun during this review.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared terminal semantics across consumers.
   - **ARCH-PURE — pass:** inspected pure concepts match their classifications and have direct tests.
   - **ARCH-PURPOSE — flag:** promised CI delivery remains incomplete.
   - **ARCH-MOCK — flag:** independent/native fixtures exist, but native terminal checks lack automated execution.
   - **ARCH-CONSTRAINTS — pass:** bounded queues/resources; measured performance exceptions remain explicit.
   - **ARCH-SECURE — pass:** inspected notification boundaries validate ownership and input.
   - **ARCH-ORDER — pass:** serialized admission, partial-write handling and joined teardown.
   - **ARCH-FUNERAL — pass:** inspected workers and temporary artifacts have disposal paths.

7. **Plan revision recommendations**

   Append a `## Revisions` entry recording the missing CI wiring and its correction, including executed tests, dependencies and trigger coverage.

```findings
dispose:
  - id: BR-22
    disposition: addressed
    note: |
      The pinned correction updates atlas/architecture.md:743-748, wrapcmd/wrap.go:10-12 and bin/pair-notify:13 together. Current descriptions name the wrapper broker and serialized pane output; outer-TTY references describe compatibility metadata. notifycmd/run.go:34-40 and wrapcmd/wrap.go:730-736 support those descriptions. This correction changes prose/comments only.
findings:
  - id: new
    severity: Important
    family: conformance-ci-enforcement
    title: |
      Claimed native terminal CI coverage is not wired into any workflow
    detail: |
      workshop/plans/000255-terminal-abstraction-plan.md:107 marks native terminal CI complete, but .github/workflows/couch-zellij-conformance.yml:108 invokes Makefile.local:83-86, which runs only lifecycle suites. cmd/internal/couchtty/terminal_native_test.go:30 skips without PAIR_LIVE_COUCH_NATIVE=1, and no CI entrypoint supplies that flag or PAIR_NATIVE_BINARY. Wire the native terminal suites into CI with dependencies, a freshly built candidate and relevant source triggers; verify actual execution rather than skips. ARCH-PURPOSE and ARCH-MOCK.
```
