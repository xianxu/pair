# Boundary Review — pair#329 (whole-issue close)

| field | value |
|-------|-------|
| issue | 329 — Detach kills an unbound session watcher |
| repo | pair |
| issue file | workshop/issues/000329-detach-kills-session-watcher.md |
| boundary | whole-issue close |
| milestone | — |
| window | 99405e16f037ad5bbf7f05bbb4e0ed7df75261dd..5d51aff03004b46f8ef3ad2446846a4667234c13 |
| command | sdlc close --issue 329 |
| reviewer | codex |
| timestamp | 2026-09-28T12:14:40-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The ownership change looks sound, and all three relevant package suites pass. Shipping is blocked by missing coverage of the explicit acceptance sequence: detach before the first turn, reattach, complete a turn, then successfully relaunch.

1. **Strengths**
   - Watcher creation is consolidated in wrap; launcher and fresh-restart duplicate paths are removed.
   - The PID freshness bound is captured before agent startup and tested against the PID file’s timestamp.
   - The real process-spawn test verifies an independent process group.

2. **Critical findings:** None.

3. **Important findings**
   - **Acceptance sequence remains untested** — `cmd/internal/wrapcmd/launch_watcher_test.go:56,131`; issue `:98–101`. The wrap test stubs watcher execution; the process-group test launches `sleep` and checks PGID. Neither detaches, reattaches, publishes a completed turn, or verifies relaunch authorization. The recorded smoke explicitly detached after several turns. Add a composed regression exercising the required sequence with stateful native-agent fixtures, covering Codex and Claude, and confirm it fails with the original ownership arrangement. **ARCH-PURPOSE**.

4. **Minor findings**
   - **Watcher lifetime prose omits Codex’s continued observation.** Instances: `atlas/session-identity.md:55–56`, `cmd/internal/wrapcmd/wrap.go:2370–2371`, issue `:86–87`. These say the watcher exits when bound; production enables `FollowLifecycle` (`sessionwatch/runcli.go:144`), and Codex continues after binding (`sessionwatch/run.go:185–196`). Qualify all three statements.

5. **Test coverage notes**
   - Passed: `go test ./cmd/internal/wrapcmd ./cmd/internal/launcher ./cmd/internal/sessionwatch`.
   - Existing watcher tests exercise binding and subsequent Codex lifecycle publication, but do not compose those behaviors with detach.
   - No repository files changed.

6. **Architectural notes**
   - **ARCH-DRY: pass.** One spawn site reuses `sessionwatch.CommandArgs`.
   - **ARCH-PURE: pass.** The change remains small process-boundary glue; binding logic stays in sessionwatch.
   - **ARCH-PURPOSE: flag.** Implementation addresses ownership, but the promised end-to-end outcome lacks regression evidence.
   - Atlas updates are present. No new user-facing command, flag, or configuration requires a README addition.

7. **Plan revision recommendations**
   - Append a dated `## Revisions` entry adding the composed acceptance regression and distinguishing existing mechanism tests from acceptance evidence.
   - Correct the watcher-lifetime description across the enumerated sites.

```findings
findings:
  - id: new
    severity: Important
    family: acceptance-boundary-coverage
    title: |
      Required detach-before-first-turn acceptance sequence is untested
    detail: |
      cmd/internal/wrapcmd/launch_watcher_test.go:56 stubs watcher execution; :131 checks a sleep child's PGID. Neither establishes binding or relaunch after detach and reattach, and workshop/issues/000329-detach-kills-session-watcher.md:98–101 explicitly requires that sequence. The recorded smoke detached only after turns. These are the acceptance-evidence gaps in this window. Add a composed regression covering Codex and Claude with stateful native fixtures, asserting relaunch authorization after the first post-reattach turn, and verify it fails under the original ownership arrangement. ARCH-PURPOSE.
  - id: new
    severity: Minor
    family: lifecycle-documentation-fidelity
    title: |
      Watcher lifetime claims omit continued Codex lifecycle observation
    detail: |
      All instances are atlas/session-identity.md:55–56, cmd/internal/wrapcmd/wrap.go:2370–2371, and workshop/issues/000329-detach-kills-session-watcher.md:86–87. They say the watcher exits when bound, but sessionwatch/runcli.go:144 enables FollowLifecycle and sessionwatch/run.go:185–196 continues Codex observation after binding. Qualify these statements to describe the Codex exception.
```

---

## Re-review — 2026-09-28T12:27:52-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 329 — Detach kills an unbound session watcher |
| repo | pair |
| issue file | workshop/issues/000329-detach-kills-session-watcher.md |
| boundary | whole-issue close |
| milestone | — |
| window | 99405e16f037ad5bbf7f05bbb4e0ed7df75261dd..189fa28f2ac0ee0e5dda0b8acb16df980117c724 |
| command | sdlc close --issue 329 |
| reviewer | codex |
| timestamp | 2026-09-28T12:27:52-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The ownership change works, and both prior corrections are verified. One new issue blocks shipping: the composed acceptance test intermittently fails at final relaunch resolution. Both Codex and Claude exhibited this failure.

1. **Strengths**
   - One wrap-owned spawn site replaces both previous paths.
   - Tests verify process-group isolation and the PID freshness bound.
   - Restoring client-group ownership in a scratch overlay made both acceptance cases fail; production passed with race detection.

2. **Critical findings:** None.

3. **Important findings**
   - `cmd/internal/wrapcmd/detach_acceptance_test.go:200–210`: the test awaits binding, then attempts authorization once. Binding publication precedes remaining watcher writes (`sessionwatch/lifecycle.go:207–219`), so it is not a completion barrier. The full suite failed for Codex with “session inventory storage root is absent”; diagnostic repetitions reproduced this for Claude. Await the final authorization condition with a bounded deadline and retain the last error. Both agent cases need this correction.

4. **Minor findings:** None.

5. **Test coverage**
   - Race-enabled acceptance: passed.
   - Ownership mutation: both agents failed as expected.
   - Relevant package suites: first run failed as above; second passed.
   - Repository files remained unchanged.

6. **Architecture**
   - **ARCH-DRY: pass** — consolidated spawning reuses `CommandArgs`.
   - **ARCH-PURE: pass** — small process-boundary glue; binding logic remains separate.
   - **ARCH-PURPOSE: flag** — intended behavior is demonstrated, but acceptance verification needs reliable completion handling.
   - Atlas updates are present; no new user-facing syntax requires README changes.

7. **Plan revision recommendation**
   - Append a revision requiring bounded observation of final relaunch authorization for both agents, preserving the ownership mutation check.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      The composed Codex/Claude regression exercises real watcher execution, detach, post-reattach binding and relaunch authorization. Independently restoring client-group ownership in a scratch overlay made both cases fail at post-reattach binding.
  - id: BR-2
    disposition: addressed
    note: |
      atlas/session-identity.md:55–57, wrap.go:2370–2372 and the issue's lifecycle paragraph now explicitly preserve Codex observation after binding, matching sessionwatch/runcli.go:144 and run.go:185–196.
findings:
  - id: new
    severity: Important
    family: acceptance-boundary-coverage
    title: |
      Acceptance regression races final relaunch authorization
    detail: |
      cmd/internal/wrapcmd/detach_acceptance_test.go:200–210 waits only for binding before a single resolver call. The full suite failed for Codex at line 208 with “session inventory storage root is absent”; diagnostic repetitions reproduced it for Claude against the Pair data root. Both cases share this site and exhaust the family instances in this window. This is the 2nd finding in family acceptance-boundary-coverage: apply the rule that asynchronous acceptance tests await their final contractual outcome, rather than an intermediate publication. Boundedly await successful resolution and relaunch preconditions for both agents, retaining diagnostic errors and the ownership mutation check. ARCH-PURPOSE.
```

---

## Re-review — 2026-09-28T12:37:38-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 329 — Detach kills an unbound session watcher |
| repo | pair |
| issue file | workshop/issues/000329-detach-kills-session-watcher.md |
| boundary | whole-issue close |
| milestone | — |
| window | 99405e16f037ad5bbf7f05bbb4e0ed7df75261dd..df4a9496db753dd994162cbc352b3f7828a9366b |
| command | sdlc close --issue 329 |
| reviewer | codex |
| timestamp | 2026-09-28T12:37:38-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned change fulfills the issue’s watcher-ownership contract. BR-3 is addressed: both agents await final relaunch authorization with bounded diagnostics. Independently removing that correction reproduced the reported failure for both Codex and Claude.

1. **Strengths**
   - One wrap-owned watcher spawn replaces the launcher and fresh-restart paths.
   - Tests verify process-group isolation, launch identity, and PID freshness.
   - Composed acceptance tests exercise detach before the first turn through successful relaunch authorization.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage**
   - Full wrapcmd, launcher, and sessionwatch suites passed.
   - Race-enabled acceptance passed ten repetitions per agent.
   - A temporary overlay restoring the pre-BR-3 test failed for both agents with “session inventory storage root is absent.”
   - Diff whitespace checks passed; repository files remained unchanged.
   - Acceptance uses native fixtures and simulated terminal attachment, not interactive Zellij.

6. **Architecture**
   - **ARCH-DRY: pass** — consolidated spawning reuses `sessionwatch.CommandArgs`.
   - **ARCH-PURE: pass** — process glue stays small; binding logic remains separate.
   - **ARCH-PURPOSE: pass** — both required agents reach final authorization after detach.
   - Atlas updates cover ownership and lifecycle. No new user-facing syntax requires README changes.

7. **Plan revision recommendations:** None; the existing revision covers BR-3.

```findings
dispose:
  - id: BR-3
    disposition: addressed
    note: |
      cmd/internal/wrapcmd/detach_acceptance_test.go:205–228 boundedly retries resolution and relaunch preconditions for both agents, retaining failure diagnostics. Ten race-enabled repetitions per agent passed; restoring the previous test in a scratch overlay reproduced the storage-root failure for both Codex and Claude.
```
