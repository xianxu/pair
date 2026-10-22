# Boundary Review — pair#759 (milestone M2)

| field | value |
|-------|-------|
| issue | 759 — remote-session: 👽 mode in the terminal cast behind Cloudflare Access, off by default, kill switch re-enabled only at the console, tagged remote submits |
| repo | pair |
| issue file | workshop/issues/000759-remote-session-mode-in-the-terminal-cast-behind-cloudflare-access-off-by-default-kill-switch-re-enabled-only-at-the-console-tagged-remote-submits.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 8056e8edfd532750dc0cea82c29d6b791c2bd46f..42aeb9140798b7d56d968af0986973da83df344c |
| command | sdlc milestone-close --issue 759 --milestone M2 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T20:06:06-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M2 delivers the documented protected exposure, lifecycle controls and host-only reset boundary. No production security defect was confirmed. Fix the repeatable test-fixture failure before closing M2; document dispatch and provenance remain explicitly assigned to M3.

```findings
findings:
  - id: new
    severity: Important
    family: test-operating-envelope
    title: |
      Allocate a sufficiently short directory for the protected lease test
    detail: |
      cmd/internal/broadcast/protected_test.go:318 uses shortDir(t), which nests beneath the inherited TMPDIR; listenProtected then adds another directory and socket suffix. On this macOS host, go test ./cmd/internal/broadcast -run '^TestProtectedUsesExistingGuardAndNamedLease$' -count=1 consistently fails at line 326 with "protected Unix socket path too long", before exercising the guard or lease. ARCH-CONSTRAINTS: allocate an owned, cleaned-up short directory with room for the complete socket path. The same test passes with TMPDIR=/tmp; retain the production refusal for oversized paths.
  - id: new
    severity: Minor
    family: cli-help-context
    title: |
      Keep notification help beneath its command
    detail: |
      cmd/internal/couchcmd/run.go:1504 inserts --remote between the --notify synopsis and its three explanatory lines. Those lines now appear to describe --remote. Move the remote synopsis after the complete notification help block.
```

1. **Strengths**

   - Protected transport reuses the existing connector, guard and named lease while refusing anonymous and TCP fallback.
   - Supervisor tests control late-open and reset/kill publication ordering, and verify one transport per epoch.
   - The actual broker → wrapper → PTY test verifies child output cannot reset the host latch.
   - README and atlas document the new controls, capture refusal and current editor limitations.

2. **Critical findings**

   None.

3. **Important findings**

   The protected lease fixture fails under this host’s default temporary-directory configuration; see the finding above.

4. **Minor findings**

   Notification help is split by the new remote synopsis.

5. **Test coverage notes**

   Passed:

   - Focused remote/protected/supervisor tests under the race detector across remotesession, broadcast, couchtty and couchcmd.
   - Complete remotesession, repolint and artifactpath suites.
   - Complete broadcast suite.

   These runs used `TMPDIR=/tmp`. The default-temp fixture failure was reproduced separately. The checkout remains unchanged. No live Cloudflare or phone acceptance was performed.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared transport and lifecycle authorities are reused.
   - **ARCH-PURE — pass:** value decisions remain separate from synchronized IO owners; inspected concept-table rows match their entities.
   - **ARCH-PURPOSE — pass:** implemented scope matches the explicitly revised M2 boundary.
   - **ARCH-MOCK — pass:** stateful tunnel and signed-issuer fixtures exercise production seams; live conformance remains M3.
   - **ARCH-CONSTRAINTS — flag:** socket-path fixture violates its environment bound.
   - **ARCH-SECURE — pass:** verified identity, strict ingress, closed routes and isolated reset authority.
   - **ARCH-ORDER — pass:** reducer transitions and controlled interleavings cover key lifecycle races.
   - **ARCH-FUNERAL — pass:** owned transport resources have teardown paths; uncertainty blocks reopening.
   - **ARCH-EVOLVE — pass:** no incompatible persisted-format change identified; preservation tests pass.

7. **Plan revision recommendations**

   None required; correct the fixture and rerun qualification.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"15e65b8e-c3ec-4409-92da-5170e3a6d62d","repository":"github.com/xianxu/pair","issue_id":"000759","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/3a08aa98dd330c57","reviewed_head":"42aeb9140798b7d56d968af0986973da83df344c","slot":"pair:6","requested_at":"2026-10-22T03:02:22.090869Z"},"phase":"finished","pid":127,"started_at":"2026-10-22T03:02:22.243445Z","finished_at":"2026-10-22T03:06:05.390785Z","session_id":"01a1618f-48aa-76d0-9b37-a50490e4a17e","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.829166917s
head: "42aeb9140798b7d56d968af0986973da83df344c"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-21T20:16:52-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 759 — remote-session: 👽 mode in the terminal cast behind Cloudflare Access, off by default, kill switch re-enabled only at the console, tagged remote submits |
| repo | pair |
| issue file | workshop/issues/000759-remote-session-mode-in-the-terminal-cast-behind-cloudflare-access-off-by-default-kill-switch-re-enabled-only-at-the-console-tagged-remote-submits.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 8056e8edfd532750dc0cea82c29d6b791c2bd46f..96378abaf7e6ea5db364bc7c6f0145af9f78e9a8 |
| command | sdlc milestone-close --issue 759 --milestone M2 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T20:16:52-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Both prior findings are addressed, with regressions confirmed to fail when their fixes are reverted. M2’s protected transport, revocation and host-only reset controls passed review checks. One Important recovery gap remains: startup failures lose actionable information before reaching the operator.

```findings
dispose:
  - id: BR-5
    disposition: addressed
    note: |
      cloudflared_test.go:27 now uses testpair.ShortRoot. The long-TMPDIR listener/cleanup regression passes at HEAD and fails with “protected Unix socket path too long” when the allocator fix is reverted in a disposable checkout.
  - id: BR-6
    disposition: addressed
    note: |
      run.go:1503–1507 keeps all notification explanations before --remote. TestRemoteHelpKeepsNotificationDetailsTogether passes at HEAD and fails when the previous ordering is restored in a disposable checkout.
findings:
  - id: new
    severity: Important
    kind: defect
    family: failure-recovery-visibility
    title: |
      Preserve and display actionable remote failure reasons
    detail: |
      supervisor.go:222 collapses ErrTunnelBusy into a generic transport failure, losing the plan’s explicit “stop the existing casts, then retry” recovery. A disposable regression with an already-held exclusive FakeTunnel reproduced Status=disabled and only the generic reset instruction. Separately, couchcmd/remote_cli.go:44 drops RemoteStatus.Detail, and couchtty/console_remote_session.go:90–101 substitutes generic notices, hiding configuration and cleanup details too. ARCH-PURPOSE: preserve safe classified failure reasons and display the secret-free recovery detail through both operator surfaces; cover busy transport, invalid configuration and uncertain cleanup.
```

1. **Strengths**

   - Protected transport requires named Access configuration and private Unix sockets; oversized paths retain production refusal.
   - Held-effect tests verify late-open disposal, joined teardown and reset refusal after uncertain cleanup.
   - The actual broker/wrapper/PTY fixture confirms child input and reset-shaped output cannot clear the host latch.
   - README and atlas describe the new controls and accurately identify document dispatch/provenance as pending M3 work.

2. **Critical findings**

   None.

3. **Important findings**

   The finding above spans [supervisor.go:222](/Volumes/TestScratch/review-checkouts/c23aedc5214837f8/cmd/internal/remotesession/supervisor.go:222), [remote_cli.go:44](/Volumes/TestScratch/review-checkouts/c23aedc5214837f8/cmd/internal/couchcmd/remote_cli.go:44) and [console_remote_session.go:90](/Volumes/TestScratch/review-checkouts/c23aedc5214837f8/cmd/internal/couchtty/console_remote_session.go:90). Fix the failure-to-recovery projection across these surfaces while preserving the current authority fence and keeping capabilities out of diagnostics.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed targeted M2 tests normally and with `-race` across remotesession, broadcast, couchcmd and couchtty; complete remotesession race tests; complete broadcast, repolint and artifactpath tests. Both prior-fix reversions produced the expected regression failures. The new busy-recovery reproduction failed against HEAD. Disposable checkouts were removed; the reviewed checkout remains unchanged. Live Cloudflare/phone acceptance was not exercised.

6. **Architectural notes for upcoming work**

   - **ARCH-DRY — pass:** transport, guard, lease and short-root ownership are reused.
   - **ARCH-PURE — pass:** delivered core-table entities match their locations and classifications; lifecycle decisions remain in the pure reducer.
   - **ARCH-PURPOSE — flag:** actionable startup/recovery behavior remains incomplete as reported above.
   - **ARCH-MOCK — pass for M2:** stateful tunnel and signed issuer fixtures exercise production seams; live conformance remains explicitly assigned to acceptance.
   - **ARCH-CONSTRAINTS — pass:** bounded request lanes, socket budgets and counted lifecycle controls are exercised.
   - **ARCH-SECURE — pass:** authentication, origin/IP checks, private ingress and host-key reset separation are enforced.
   - **ARCH-ORDER — pass:** controlled late completions and reset/kill publication ordering exercise production lifecycle transitions.
   - **ARCH-FUNERAL — pass:** owned listeners, configuration directories and connectors have teardown paths; uncertainty prevents reuse.
   - **ARCH-EVOLVE — pass:** M2 introduces no changed persisted authority format; latch operations reuse the existing codec.

7. **Plan revision recommendations**

   None required. Implement the existing busy-refusal and visible-recovery contracts; retain the documented M2/M3 separation.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"c4744bb2-2cb4-48d9-8820-80d76ea71788","repository":"github.com/xianxu/pair","issue_id":"000759","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/c23aedc5214837f8","reviewed_head":"96378abaf7e6ea5db364bc7c6f0145af9f78e9a8","slot":"pair:6","requested_at":"2026-10-22T03:12:10.811667Z"},"phase":"finished","pid":84106,"started_at":"2026-10-22T03:12:10.974352Z","finished_at":"2026-10-22T03:16:51.981248Z","session_id":"01a16198-446a-7692-a2fc-1ef938d0222a","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.005601792s
head: "96378abaf7e6ea5db364bc7c6f0145af9f78e9a8"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-21T20:27:35-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 759 — remote-session: 👽 mode in the terminal cast behind Cloudflare Access, off by default, kill switch re-enabled only at the console, tagged remote submits |
| repo | pair |
| issue file | workshop/issues/000759-remote-session-mode-in-the-terminal-cast-behind-cloudflare-access-off-by-default-kill-switch-re-enabled-only-at-the-console-tagged-remote-submits.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 8056e8edfd532750dc0cea82c29d6b791c2bd46f..02c31d366994bfe3b8f017e1311c7ce6ed493939 |
| command | sdlc milestone-close --issue 759 --milestone M2 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T20:27:35-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

M2 delivers the protected security surface, console reset boundary, and documented M3 separation. BR-7 is addressed with meaningful regression tests. One blocking lifecycle defect remains: failed cleanup during a canceled opening can be reported as confirmed, allowing the kill latch to reset.

```findings
dispose:
  - id: BR-7
    disposition: addressed
    note: |
      Busy recovery is classified in supervisor.go:222–227, CLI prints Detail at remote_cli.go:45–46, and console notices retain it at console_remote_session.go:90–113. The three corresponding regression tests pass at head and fail when their fixes are independently removed in a disposable copy.
findings:
  - id: new
    severity: Critical
    kind: defect
    family: transport-cleanup-confirmation
    title: |
      Preserve teardown uncertainty when opening returns no mount
    detail: |
      ARCH-ORDER: supervisor.go:213–228 discards MountProtected's error once kill has changed the phase, while cleanupLocked at lines 304–323 assumes no mount means successful cleanup. Reproduced using FakeTunnel with held Open, IgnoreCancel=true and CloseError: On -> wait OpenStarted -> Kill -> release Open -> Join returns nil, CanReset=true, and reset succeeds despite failed connector cleanup; the regression failed on all three race runs. The related host.go:175–211 cancellation path also skips joining an already-closing attempt, allowing MountProtected to return before Close completes. Require confirmed transport teardown independently of whether a Mount was returned, propagate cleanup failures into the supervisor's blocked state, and cover canceled/failed opening plus already-closing cancellation.
```

1. **Strengths**

   - Protected transport validates explicit Access configuration and requires a private Unix origin.
   - Signed HTTP tests exercise authentication, session capacity, sessionless kill, and rejection of generic routes.
   - The actual broker/wrapper/PTY regression checks that child input and output cannot reset the host latch.
   - README and atlas describe the new controls and clearly identify pending document integration.

2. **Critical findings**

   - **Unconfirmed opening cleanup permits reset**, at `supervisor.go:220` and `supervisor.go:304`. Preserve the transport’s cleanup outcome even when opening returns no mount; reset must remain unavailable after uncertain teardown. See the machine-readable finding above.

3. **Important findings**

   None additional.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Passed focused remote/protected tests with race detection, the complete remotesession race suite, repolint, and artifactpath tests.
   - Independently verified BR-7’s red controls.
   - The new canceled-opening cleanup regression failed three times under race detection.
   - Disposable review files were removed; the reviewed checkout remains unchanged.

6. **Architectural notes**

   - **ARCH-DRY — pass:** reuses guarded transport and named-tunnel ownership.
   - **ARCH-PURE — pass:** lifecycle and policy decisions remain separated from IO.
   - **ARCH-PURPOSE — pass:** M2 scope matches the explicit M2/M3 revision.
   - **ARCH-MOCK — pass for M2:** stateful transport and signed issuer share production seams; live qualification remains an explicit later obligation.
   - **ARCH-CONSTRAINTS — pass:** request lanes, deadlines, session capacity, and counted lifecycle checks are bounded.
   - **ARCH-SECURE — pass:** typed identity, protected ingress, secret-free projections, and host-key reset separation are present.
   - **ARCH-ORDER — flag:** cleanup uncertainty is collapsed into success on opening failure/cancellation.
   - **ARCH-FUNERAL — flag:** the same defect permits lifecycle completion before the connector’s end is confirmed.
   - **ARCH-EVOLVE — pass:** this range introduces no incompatible persisted-format rewrite.

7. **Plan revision recommendations**

   Add a `## Revisions` entry requiring teardown confirmation for **every opening outcome**, including no-mount failures and cancellation after teardown has begun. Extend the lifecycle strategy with held-close and failed-close sequences that assert `Join`, blocked status, and reset refusal.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"7aa71a28-18d5-4a60-8d07-4fa81f4bb1a6","repository":"github.com/xianxu/pair","issue_id":"000759","recipe":"milestone-review","round":3,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/41665e02cddc7aab","reviewed_head":"02c31d366994bfe3b8f017e1311c7ce6ed493939","slot":"pair:6","requested_at":"2026-10-22T03:21:42.821237Z"},"phase":"finished","pid":19842,"started_at":"2026-10-22T03:21:42.914556Z","finished_at":"2026-10-22T03:27:31.660446Z","session_id":"01a161a0-fe06-7002-8019-0e312de94603","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.818524959s
head: "02c31d366994bfe3b8f017e1311c7ce6ed493939"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
