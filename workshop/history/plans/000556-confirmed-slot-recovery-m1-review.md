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

---

## Re-review — 2026-10-15T09:00:19-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 556 — Couch confirmed disk recovery and always-reachable fresh startup |
| repo | pair |
| issue file | workshop/issues/000556-confirmed-slot-recovery.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | d6a797451cc6ded68c890f10f294db28b94d1708..a682dc8e4ccfa2ccb6717e9592e30bccc88db27f |
| command | sdlc milestone-close --issue 556 --milestone M1 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-15T09:00:19-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M1 delivers the documented confirmed-recovery path, and all three prior findings are addressed. The fresh package-wide tests pass. One Important issue remains: profile preparation copies the synchronized `Couch` owner, which `go vet` rejects.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      recovery_prepared.go:477–484 selects read-only readiness when source absence is unproved. The strengthened foreign-survivor test asserts zero normal readiness calls and unchanged process/session identities; it passed.
  - id: BR-2
    disposition: addressed
    note: |
      retireConfirmed now signals accepted generations leaf-first and independently observes their absence. Passing regressions cover survivors, reparenting, replacement, false signal success and final-listing races; settled-owner tests exercise ParkExpected and refusal after park failure.
  - id: BR-3
    disposition: addressed
    note: |
      README.md:447–455 documents the standalone token command; lines 493–520 explain exact-stop acceptance, two-minute expiry, Fresh availability and bounded Starting. These passages agree with the CLI, confirmation owner and stale-start decision.
findings:
  - id: new
    severity: Important
    kind: defect
    family: synchronized-owners-must-not-be-copied
    title: |
      Avoid copying the live Couch for read-only profile resolution
    detail: |
      cmd/internal/couchcore/recovery_preflight.go:82 copies the entire Couch, including recoveryMu, regMu and gateOnce; recovery_preflight_test.go:337 repeats the pattern. Copying an active synchronized owner violates its synchronization contract and can race concurrent operations. go vet ./cmd/internal/couchcore exits 1 with both copylocks diagnostics. Pass the read-only ThreadStore into profile-resolution helpers instead of cloning Couch, and construct the test fixture without copying its locks. ARCH-ORDER.
```

1. **Strengths**

   - Retained and recordless Fresh share survivor-safe workspace readiness.
   - Retirement tests assert process absence independently of signal success.
   - Schema migration preserves strict inferred evidence while allowing optional Fresh orientation.
   - README and Atlas cover the new surface and clearly distinguish M1 from M2.

2. **Critical findings:** None.

3. **Important findings:** The synchronized-owner copy above. Remove both copies and rerun `go vet ./cmd/internal/couchcore`.

4. **Minor findings:** None.

5. **Test coverage notes**

   Full tests passed for `couchcore`, `launcher`, `couchcmd`, `couchtty`, `couchmessage` and `threadrecord`; `artifactpath` also passed. Focused recovery regressions passed. The retained whole-suite log’s SHA-256 matches the tracker, and only the issue log changed afterward. Fix-removal mutation checks were not run.

6. **Architectural notes**

   | Principle | Result |
   |---|---|
   | ARCH-DRY | Pass: shared preparation, readiness and launch owners. |
   | ARCH-PURE | Pass: pure decisions have direct tests; IO uses injected seams. |
   | ARCH-PURPOSE | Pass for M1; native admission remains explicitly scoped to M2. |
   | ARCH-MOCK | Pass: stateful lifecycle doubles exercise production boundaries; scratch conformance is documented. |
   | ARCH-CONSTRAINTS | Pass: bounded observations, retirement waits and confirmation capacity. |
   | ARCH-SECURE | Pass: exact generations, strict decoding and repeated authority checks. |
   | ARCH-ORDER | Flag: copying the synchronized owner undermines its ownership boundary. Otherwise, interruption and uncertainty tests are substantive. |
   | ARCH-FUNERAL | Pass: confirmations expire; durable evidence uses existing retention and archival lifecycles. |

7. **Plan revision recommendations:** None required; the existing revisions accurately delimit M1 and M2.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"ae868893-e886-4c83-a394-f1a854b708b0","repository":"github.com/xianxu/pair","issue_id":"000556","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/48fa1712f7ee7662","reviewed_head":"a682dc8e4ccfa2ccb6717e9592e30bccc88db27f","slot":"pair:2","requested_at":"2026-10-15T15:54:27.718592Z"},"phase":"finished","pid":84093,"started_at":"2026-10-15T15:54:27.80741Z","finished_at":"2026-10-15T16:00:19.180974Z","session_id":"01a14045-a338-7a30-9746-d1e554a0e978","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.840719375s
head: "a682dc8e4ccfa2ccb6717e9592e30bccc88db27f"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-15T09:18:44-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 556 — Couch confirmed disk recovery and always-reachable fresh startup |
| repo | pair |
| issue file | workshop/issues/000556-confirmed-slot-recovery.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | d6a797451cc6ded68c890f10f294db28b94d1708..dead1b665bddf4d053df5faba06b54a73420296a |
| command | sdlc milestone-close --issue 556 --milestone M1 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-15T09:18:44-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

BR-4 is addressed, and all seven reviewed packages pass. One Critical defect blocks M1: the advertised keyed Fresh choice encounters native-resume guards before being recognized as Fresh, making it fail where explicit Fresh succeeds.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Survivor-safe readiness remains implemented in recovery_prepared.go; retained-record and recordless Fresh regression tests pass.
  - id: BR-2
    disposition: addressed
    note: |
      Confirmed retirement independently proves accepted process absence; survivor, replacement, reparenting and final-listing regressions pass.
  - id: BR-3
    disposition: addressed
    note: |
      README.md documents standalone confirmation, expiry, Fresh availability and bounded Starting, consistent with the implemented CLI and recovery owners.
  - id: BR-4
    disposition: addressed
    note: |
      recovery_preflight.go:82 copies only ThreadStore and passes it through the existing resolver; the fixture constructs Couch independently. Core vet passes; a temporary pre-fix overlay reproduces both copylocks diagnostics. Profile/preflight race tests pass.
findings:
  - id: new
    severity: Critical
    kind: defect
    family: fresh-selection-precedes-native-guards
    title: |
      Recognize keyed Fresh before applying native-resume admission guards
    detail: |
      cmd/internal/couchcore/recovery_prepared.go:94–115 resolves the native profile and checks park, continuation and start claims before lines 125–130 recognize a Fresh candidate key. ResumeCandidate always enters as RecoveryInferred; menu_recovery.go:109–120 and the printed CLI command use that route for keyed Fresh. A temporary regression matrix reproduces failures for unfinished park, missing saved profile and young start claim; explicit RecoveryFresh succeeds in all three controls. This violates always-reachable Fresh (ARCH-PURPOSE). Normalize the validated choice before mode-specific admission, sweep every native-only guard, and add equivalence regressions through the advertised menu/CLI routes.
```

1. **Strengths**

   - Read-only profile resolution preserves the synchronized `Couch` owner.
   - Retirement tests independently check process disappearance and control ownership changes between effects.
   - Uncertain-launch tests retain exact claims without destructive cleanup or duplicate launch.
   - README and Atlas cover confirmation, schema compatibility and the M1/M2 split.

2. **Critical findings**

   The keyed Fresh routing defect above. Resolve the effective selection before profile and lifecycle admission; preserve candidate validation and exact-stop confirmation.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Package-wide tests passed for `couchcore`, `launcher`, `couchcmd`, `couchtty`, `couchmessage`, `threadrecord` and `artifactpath`. Core vet and focused profile/preflight race tests passed. The recorded suite log matches SHA-256 prefix `e114fbb2b590`.

   Existing tests miss Fresh equivalence across entry points. Temporary overlay tests failed in all three keyed cases described above; their explicit-Fresh controls passed. Overlays were removed; the checkout remains unchanged.

6. **Architectural notes**

   | Principle | Result |
   |---|---|
   | ARCH-DRY | Pass: shared preparation, preservation and launch owners. |
   | ARCH-PURE | Pass: pure decisions and injected IO boundaries remain separated. |
   | ARCH-PURPOSE | **Flag:** advertised Fresh does not consistently fulfill its admission contract. |
   | ARCH-MOCK | Pass: stateful lifecycle doubles exercise production seams; scratch conformance is documented. |
   | ARCH-CONSTRAINTS | Pass: observation, confirmation capacity and retirement waits are bounded. |
   | ARCH-SECURE | Pass: strict decoding, exact generations and authority rechecks. |
   | ARCH-ORDER | Pass: durable claims, fencing and uncertainty retain explicit transition ownership; BR-4 is corrected. |
   | ARCH-FUNERAL | Pass: confirmations expire; durable evidence uses existing retention/archive lifecycles. |

   Cross-slot native exclusion remains explicitly scoped to M2.

7. **Plan revision recommendations**

   Add a `## Revisions` entry stating that every Fresh representation must resolve to Fresh before mode-specific admission. Enumerate profile fallback, unfinished park, continuation, and start-claim guards, with regression coverage for menu, printed candidate command and explicit Fresh.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"1bb5cde0-e4f1-4dd4-bb7e-8e511095b0bd","repository":"github.com/xianxu/pair","issue_id":"000556","recipe":"milestone-review","round":3,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/99fba32e73b76d2a","reviewed_head":"dead1b665bddf4d053df5faba06b54a73420296a","slot":"pair:2","requested_at":"2026-10-15T16:13:15.841739Z"},"phase":"finished","pid":66084,"started_at":"2026-10-15T16:13:15.934778Z","finished_at":"2026-10-15T16:18:43.595767Z","session_id":"01a14056-da01-7461-94a5-7592a3cda1c7","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.838985s
head: "dead1b665bddf4d053df5faba06b54a73420296a"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000409

---

## Re-review — 2026-10-15T09:36:05-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 556 — Couch confirmed disk recovery and always-reachable fresh startup |
| repo | pair |
| issue file | workshop/issues/000556-confirmed-slot-recovery.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | d6a797451cc6ded68c890f10f294db28b94d1708..646c4a5bdb214ee4c755133049c3af386dc679f0 |
| command | sdlc milestone-close --issue 556 --milestone M1 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-15T09:36:05-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned M1 implementation matches the Spec and approved Plan revisions. BR-5 is addressed with demonstrated regression evidence: restoring the pre-fix executor caused eight keyed-Fresh cases to fail, while explicit-Fresh controls passed. No new blocking findings emerged. Cross-slot native exclusion remains explicitly scoped to M2.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Retained and recordless Fresh use read-only checkout readiness beside unproved owners. The retained-owner regression passed and asserts unchanged workspace bytes and no normal provisioning or signals.
  - id: BR-2
    disposition: addressed
    note: |
      Confirmed retirement independently proves every accepted process absent, with generation checks before signals and after final listing. Survivor, replacement, reparenting and delayed-absence regressions passed under race.
  - id: BR-3
    disposition: addressed
    note: |
      The pinned README recovery and Reboot passages document standalone confirmation, expiry and survivor-safe Fresh; atlas/couch.md describes the corresponding prepared executor and retirement owners.
  - id: BR-4
    disposition: addressed
    note: |
      Fresh profile resolution passes a read-only ThreadStore into existing resolvers without copying Couch. Profile regressions and go vet -copylocks passed.
  - id: BR-5
    disposition: addressed
    note: |
      Validated keyed Fresh is normalized before profile and native-only guards. The four-scenario, three-route matrix passes on head; restoring the pre-fix executor in a disposable copy produces eight candidate/operation failures. Menu and printed-CLI transport regressions also pass.
```

1. **Strengths**

   - Shared preparation keeps menu, CLI and confirmation execution on one lifecycle path.
   - Immutable retirement plans preserve exact process authority across effects.
   - Uncertain launches retain their claims and use bounded observation instead of destructive cleanup.
   - Schema migration and optional Fresh orientation preserve strict native-resume evidence requirements.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Focused tests passed across couchcore, couchcmd, couchtty, couchmessage, threadrecord and launcher. Additional race checks covered Fresh equivalence, retirement, uncertain launches and retained evidence. Copylocks and diff whitespace checks passed. The mutation-test copy was removed; the checkout remains clean. I did not rerun the entire repository suite or live operator acceptance.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared executor, journal and launch owners.
   - **ARCH-PURE — pass:** pure selection, validation and transitions separated from injected IO.
   - **ARCH-PURPOSE — pass:** Fresh representations now obey the same admission rule.
   - **ARCH-MOCK — pass:** stateful process, runner and lifecycle doubles exercise production seams; live conformance has a defined release cadence.
   - **ARCH-CONSTRAINTS — pass:** bounded observations, confirmation capacity and expiry.
   - **ARCH-SECURE — pass:** versioned decoding, physical evidence validation and exact identity checks.
   - **ARCH-ORDER — pass:** preservation precedes retirement; stale claims and uncertain outcomes retain explicit authority.
   - **ARCH-FUNERAL — pass:** confirmations expire; durable evidence reuses existing archive and retention lifecycles.

   M2 still requires its planned deterministic cross-slot admission tests before claiming writer exclusion.

7. **Plan revision recommendations:** None required for M1.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"5cfe790c-12a5-4715-8bc0-a34bf76088e3","repository":"github.com/xianxu/pair","issue_id":"000556","recipe":"milestone-review","round":4,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/dc3b0e3198edcea8","reviewed_head":"646c4a5bdb214ee4c755133049c3af386dc679f0","slot":"pair:2","requested_at":"2026-10-15T16:33:46.692911Z"},"phase":"finished","pid":51612,"started_at":"2026-10-15T16:33:46.781371Z","finished_at":"2026-10-15T16:36:05.093062Z","session_id":"01a14069-a1f6-7241-8712-481a63a68263","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.952161375s
head: "646c4a5bdb214ee4c755133049c3af386dc679f0"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
