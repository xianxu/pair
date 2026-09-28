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
