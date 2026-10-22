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
