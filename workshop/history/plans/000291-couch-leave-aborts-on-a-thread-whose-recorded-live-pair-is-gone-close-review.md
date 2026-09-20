# Boundary Review — pair#291 (whole-issue close)

| field | value |
|-------|-------|
| issue | 291 — Couch leave aborts on a thread whose recorded-live Pair is gone |
| repo | pair |
| issue file | workshop/issues/000291-couch-leave-aborts-on-a-thread-whose-recorded-live-pair-is-gone.md |
| boundary | whole-issue close |
| milestone | — |
| window | 3e290b03027b86217d8b61a4b26dfa98a0c43c1e..96344d790fe98abd1b0d95e0e874165280c7354d |
| command | sdlc close --issue 291 |
| reviewer | codex |
| timestamp | 2026-09-20T14:15:19-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

The #291 fix satisfies the revised contract: confirmed-dead incarnations are retired silently, healthy threads continue detaching, and uncertain or failed operations retain their documented handling. No blocking findings. The broader pinned range also passed targeted review, with one minor documentation correction.

1. **Strengths**
   - `park.go:234` reuses exact-process observation and existing retirement guards.
   - `park_test.go:304` covers stale-first ordering, PID reuse, surviving sessions, and live/unknown preservation.
   - Cancellation and observation-error tests assert that records and signals remain untouched.
   - Fullscreen changes use explicit transitions, stateful doubles, managed artifact cleanup, and updated user documentation.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings**
   - `atlas/architecture.md:1112` and `:1117` still describe “submit/compose”; the current sequencer only submits.

5. **Test coverage**
   - Passed targeted Leave tests, inventory, fullscreen, artifact, shortcut, terminal, wrapper, launcher, Couch UI, and Neovim routing checks.
   - The broader Couch package run was stopped without results; it is not counted as passing.
   - Live fullscreen conformance was inspected but not run. No mutation testing was performed.

6. **Architecture**
   - **ARCH-DRY — pass:** shared observation, retirement, and shortcut definitions.
   - **ARCH-PURE — pass:** pure decisions remain separated from injected IO.
   - **ARCH-PURPOSE — pass:** stale records no longer strand subsequent healthy threads.
   - **ARCH-MOCK — pass:** stateful doubles exercise production seams.
   - **ARCH-CONSTRAINTS — pass:** serial detach retains bounded waits; large-record scanning remains a scalability consideration.
   - **ARCH-SECURE — pass:** exact identities and guarded record writes preserve ownership checks.
   - **ARCH-ORDER — pass:** uncertainty is preserved; post-signal failures stop the sweep.
   - **ARCH-FUNERAL — pass:** new fullscreen artifacts have cleanup and retention ownership.

7. **Plan revision recommendations:** None; #291’s appended revision matches implementation.

```findings
findings:
  - id: new
    severity: Minor
    family: docs-match-executable-contract
    title: |
      Atlas still describes the removed compose-only delivery path
    detail: |
      atlas/architecture.md:1112 and :1117 describe executing or resuming semantic submit/compose, but nvim/draft_send.lua:14 now always emits submit. Replace these present-tense references with submit; historical descriptions of prepared records can remain.
```
