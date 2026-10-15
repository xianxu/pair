# Boundary Review — pair#556 (milestone M1)

| field | value |
|-------|-------|
| issue | 556 — Couch confirmed disk recovery and always-reachable fresh startup |
| repo | pair |
| issue file | workshop/issues/000556-confirmed-slot-recovery.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 28d9e6816fd5139ad895ea22d585d7cd6aac9ed9..a2b67ce4127cd90c06961204a4feed4c213dd1b2 |
| command | sdlc milestone-close --issue 556 --milestone M1 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-15T07:31:14-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

M1 shares recovery execution across menu and CLI, preserves uncertain launches, and adds useful adversarial coverage. All six tested packages pass. Two recovery contracts remain incomplete: retained-record Fresh can reconcile a workspace beside foreign survivors, and confirmed retirement does not establish that every named process stopped. The README also omits the new confirmation surface.

1. **Strengths**

   - Confirmation binds actor, source revision, selection and terminal digest, with expiry and single consumption (`recovery_confirmation.go:66`).
   - Uncertain acknowledgment or registration retains the exact launch claim without destructive cleanup (`launch_existing.go:226`).
   - Schema migration preserves read-only compatibility and rejects newer recovery fields in legacy records (`threadrecord/record.go:362`).
   - Cancellation, preservation failure, stale nonces and foreign-owner isolation have focused tests.

2. **Critical findings**

   - Retained-record Fresh bypasses the intended protection against workspace convergence beneath foreign owners.
   - Confirmed retirement checks terminal-server absence without requiring absence of the complete advertised stop set.

3. **Important findings**

   - README documentation was not updated for standalone confirmation and changed recovery behavior.

```findings
findings:
  - id: new
    severity: Critical
    family: recovery-preserves-surviving-workspaces
    title: |
      Retained-record Fresh still reconciles beneath foreign owners
    detail: |
      cmd/internal/couchcore/recovery_prepared.go:330 unconditionally calls prepareTrackedWorkspace, which invokes Workspaces.Ensure for numbered slots. launchRecoveryFreshAddress reaches this call even when confirmRecoveryAbsent failed and the old owners were deliberately left running (lines 414–465). Ensure invokes workspace reconciliation, so allocating a new tag does not isolate those owners from setup/repair effects. Use the read-only readiness adapter already implemented for recordless recovery whenever source absence is unproved. Extend TestPreparedRecoveryForeignFreshLeavesRemnantsAndUsesNewTag to reject any normal readiness call; its fixture currently increments prepared without asserting zero. ARCH-SECURE, ARCH-PURPOSE.
  - id: new
    severity: Critical
    family: retirement-requires-owner-absence
    title: |
      Confirmed retirement does not prove the advertised stop set stopped
    detail: |
      cmd/internal/launcher/recovery_terminal.go:137–141 sends confirmed plans through server-only quiescence. The only signal seam is KillServer, while checkConfirmed returns snapshot.Servers (recovery_terminal_confirmed.go:291) even when accepted wrappers/native agents remain. The existing confirmed-retirement test succeeds although recoveryTerminalWorld.KillServer removes only Servers and leaves Processes intact. A surviving or reparented native owner can therefore strand Resume after the server was killed, and the promised settled-wrapper ParkExpected path is absent. Use normal park where available, implement bounded exact-owner retirement for damaged sources, and require independently observed absence of the named owners before declaring retirement complete. Add survivor/reparenting sequence assertions. ARCH-ORDER, ARCH-PURPOSE, ARCH-MOCK.
  - id: new
    severity: Important
    family: user-facing-docs-follow-surface
    title: |
      README update appears missing for recovery confirmation and changed Fresh behavior
    detail: |
      cmd/internal/couchcmd/cli.go:342 introduces standalone --recovery-confirmation TOKEN, and run.go:1386 changes reboot usage and confirmation semantics, but README.md is unchanged in the pinned range. README.md:452 still advertises reboot --force-unknown, and lines 495–499 and 1410–1412 retain the earlier restricted Fresh/Starting description. Update the CLI reference and recovery instructions to describe exact-stop confirmation, expiry, always-offered Fresh and bounded Starting. Atlas updates are present.
```

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed:

   `go test ./cmd/internal/couchcore ./cmd/internal/couchcmd ./cmd/internal/couchtty ./cmd/internal/couchmessage ./cmd/internal/launcher ./cmd/internal/threadrecord`

   The missing assertions above explain why these gaps remain green. Live operator acceptance and the repository-wide suite were not run during this review. The checkout remains unchanged.

6. **Architectural notes**

   - **ARCH-DRY — pass:** public recovery consumers share the prepared executor.
   - **ARCH-PURE — pass:** validators, confirmation acceptance and stale-start decisions have direct pure tests; IO seams are injected.
   - **ARCH-PURPOSE — flag:** survivor-safe Fresh and complete attributed-owner retirement remain incomplete.
   - **ARCH-MOCK — flag:** retirement tests lack an independent oracle requiring the named processes to stop.
   - **ARCH-CONSTRAINTS — pass:** observations, confirmation capacity and expiry have explicit bounds.
   - **ARCH-SECURE — flag:** retained-record Fresh can reconcile beneath unproved owners.
   - **ARCH-ORDER — flag:** server absence is insufficient evidence of completed owner retirement.
   - **ARCH-FUNERAL — pass:** confirmations are bounded and ephemeral; durable evidence uses existing retention/journal owners.

   Native admission reservations are explicitly assigned to M2; their absence is not an M1 finding.

7. **Plan revision recommendations**

   Add `## Revisions` entries specifying that both recordless and retained-record Fresh use read-only readiness while owners remain unresolved, and that retirement completes only after the accepted owner set is proved absent. Update the core-concepts paths to name the prepared executor and confirmed-attribution implementation.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"2d69997a-f595-4ed7-a0d9-c81cd05e1271","repository":"github.com/xianxu/pair","issue_id":"000556","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/0d4255c869952fe8","reviewed_head":"a2b67ce4127cd90c06961204a4feed4c213dd1b2","slot":"pair:2","requested_at":"2026-10-15T14:25:55.8153Z"},"phase":"finished","pid":14804,"started_at":"2026-10-15T14:25:55.907008Z","finished_at":"2026-10-15T14:31:13.677955Z","session_id":"01a13ff4-9593-7ac2-b900-fea83c073860","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.795789167s
head: "a2b67ce4127cd90c06961204a4feed4c213dd1b2"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
