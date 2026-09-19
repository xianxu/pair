---
gate: plan-quality
issue: 288
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-18T19:07:33-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Minor
          title: Test cases enumerated in prose and the diff restated as full code blocks
          detail: Task 1 Step 1 (seven cases), Task 4 Step 2 (five scenarios with per-case assertions) and the full panebirth.go / birthwatch.go / cancellableHandoff / runCreate code blocks will be rewritten as code within the hour. Compress to named functions plus one strategy line per risky function (e.g. watchBirth over probe-answer x client-exit order -> stateful fake with scripted liveness and launch release, guarded by the four mutation checks).
          family: plan-restates-executable-artifacts
          round: 1
        - id: PQ-2
          severity: Minor
          title: A dead verdict overrides a client that exited on its own after the bound (ARCH-ORDER)
          detail: runCreate takes the dead path whenever the verdict is birthDead, whoever ended the client. If a session came up without its sidecar and the operator quits (exit 0) while the 10 s probe is in flight, the launcher reports "never came up", returns 1 and skips runCleanup. Take the dead path only when abort preempted (abort fired before LaunchSession returned) or the client's code is non-zero, and add a sequence test for it.
          family: verdict-must-match-its-cause
          round: 1
        - id: PQ-3
          severity: Minor
          title: The watch's list-sessions probe can land in another thread's birth window (ARCH-CONSTRAINTS)
          detail: 'SessionLiveness is two machine-wide list-sessions runs (zellij.go:60-72) that connect to every socket, which is #287''s kill mechanism, and the plan''s residual only covers the launch''s own birth. It is rare (it fires only after a no-pane-at-10s launch) but should be recorded as a residual with that rarity argument.'
          family: shared-probe-cotenancy
          round: 1
        - id: PQ-4
          severity: Minor
          title: The 10 s bound was not measured for a cold start (after boot or a zellij upgrade)
          detail: The bound's basis is warm idle runs and CPU-saturated runs. The plan names a slow healthy birth as the event most likely to be mishandled, so measure a cold-cache birth with probes/zellijbirthrace (n of about 3), or state why that start-up path does not delay the agent pane sidecar.
          family: envelope-basis-covers-environment
          round: 1
        - id: PQ-5
          severity: Minor
          title: The live conformance check for the fake's death-B model has no re-run trigger (ARCH-MOCK)
          detail: The fake encodes zellij 0.45.1's hung-client behaviour (zero bytes, not listed, exits on SIGTERM), and the plan runs the live probe once. Name the cadence, e.g. rerun `zellijbirthrace launch -hammer` on every zellij version bump.
          family: conformance-cadence
          round: 1
        - id: PQ-6
          severity: Minor
          title: The hung-after-birth exclusion lives only in Task 5, not in Out of scope
          detail: A server that dies after the pane was born leaves the watch at birthBorn, and the operator sees the same hung pane. State this in Out of scope with the reason, plus what happens if the live run shows hung-after-birth greater than 0 (file a follow-up issue).
          family: stated-non-goals
          round: 1
      blocked: false
    - "n": 2
      timestamp: "2026-09-18T19:11:42-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: not-addressed
          note: Author kept the code blocks and prose test lists on purpose (Revisions); Minor, carried to close review.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: Post-probe ctx recheck plus the birthDead-and-code-nonzero guard; test 6 and a mutation check cover the clean-quit race.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: Residual for other threads' births recorded with its rarity argument.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: Cold-cache births measured at 0.44-0.52 s (n=4); the 10 s bound stands.
          round: 2
        - id: PQ-5
          disposition: addressed
          note: Cadence stated in Out of scope; add the line to the Task 5 Step 4 SKILL.md edit and the Task 6 atlas bullet so the claim is backed by a step.
          round: 2
        - id: PQ-6
          disposition: addressed
          note: Hung-after-birth is in Out of scope with its reason and the follow-up rule.
          round: 2
      blocked: false
content_hash: bfa0ed461d1e094791a52af030d60fe1463cc1213d133cf1c4c46ab3794c0bfe
---

# Gate ledger — pair#288 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-18T19:07:33-07:00 (claude) — passed

### Raised

- **PQ-1** [Minor] `plan-restates-executable-artifacts` Test cases enumerated in prose and the diff restated as full code blocks
  Task 1 Step 1 (seven cases), Task 4 Step 2 (five scenarios with per-case assertions) and the full panebirth.go / birthwatch.go / cancellableHandoff / runCreate code blocks will be rewritten as code within the hour. Compress to named functions plus one strategy line per risky function (e.g. watchBirth over probe-answer x client-exit order -> stateful fake with scripted liveness and launch release, guarded by the four mutation checks).
- **PQ-2** [Minor] `verdict-must-match-its-cause` A dead verdict overrides a client that exited on its own after the bound (ARCH-ORDER)
  runCreate takes the dead path whenever the verdict is birthDead, whoever ended the client. If a session came up without its sidecar and the operator quits (exit 0) while the 10 s probe is in flight, the launcher reports "never came up", returns 1 and skips runCleanup. Take the dead path only when abort preempted (abort fired before LaunchSession returned) or the client's code is non-zero, and add a sequence test for it.
- **PQ-3** [Minor] `shared-probe-cotenancy` The watch's list-sessions probe can land in another thread's birth window (ARCH-CONSTRAINTS)
  SessionLiveness is two machine-wide list-sessions runs (zellij.go:60-72) that connect to every socket, which is #287's kill mechanism, and the plan's residual only covers the launch's own birth. It is rare (it fires only after a no-pane-at-10s launch) but should be recorded as a residual with that rarity argument.
- **PQ-4** [Minor] `envelope-basis-covers-environment` The 10 s bound was not measured for a cold start (after boot or a zellij upgrade)
  The bound's basis is warm idle runs and CPU-saturated runs. The plan names a slow healthy birth as the event most likely to be mishandled, so measure a cold-cache birth with probes/zellijbirthrace (n of about 3), or state why that start-up path does not delay the agent pane sidecar.
- **PQ-5** [Minor] `conformance-cadence` The live conformance check for the fake's death-B model has no re-run trigger (ARCH-MOCK)
  The fake encodes zellij 0.45.1's hung-client behaviour (zero bytes, not listed, exits on SIGTERM), and the plan runs the live probe once. Name the cadence, e.g. rerun `zellijbirthrace launch -hammer` on every zellij version bump.
- **PQ-6** [Minor] `stated-non-goals` The hung-after-birth exclusion lives only in Task 5, not in Out of scope
  A server that dies after the pane was born leaves the watch at birthBorn, and the operator sees the same hung pane. State this in Out of scope with the reason, plus what happens if the live run shows hung-after-birth greater than 0 (file a follow-up issue).

## Round 2 — 2026-09-18T19:11:42-07:00 (claude) — passed

### Disposed

- PQ-1 — not-addressed — Author kept the code blocks and prose test lists on purpose (Revisions); Minor, carried to close review.
- PQ-2 — addressed — Post-probe ctx recheck plus the birthDead-and-code-nonzero guard; test 6 and a mutation check cover the clean-quit race.
- PQ-3 — addressed — Residual for other threads' births recorded with its rarity argument.
- PQ-4 — addressed — Cold-cache births measured at 0.44-0.52 s (n=4); the 10 s bound stands.
- PQ-5 — addressed — Cadence stated in Out of scope; add the line to the Task 5 Step 4 SKILL.md edit and the Task 6 atlas bullet so the claim is backed by a step.
- PQ-6 — addressed — Hung-after-birth is in Out of scope with its reason and the follow-up rule.

## Open findings

- **PQ-1** [Minor] `plan-restates-executable-artifacts` Test cases enumerated in prose and the diff restated as full code blocks
