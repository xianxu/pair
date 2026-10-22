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
