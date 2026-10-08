# Boundary Review — pair#395 (milestone M5)

| field | value |
|-------|-------|
| issue | 395 — Couch broadcast: stream the composed Couch screen, view-only, to a browser viewer |
| repo | pair |
| issue file | workshop/issues/000395-couch-broadcast-stream-the-composed-couch-screen-view-only-to-a-remote-couch-watch.md |
| boundary | milestone M5 |
| milestone | M5 |
| window | 9b8a87e0e55501818567ebd6046a342e7138e43c..9806c45d5a33c5226544fbf92fe97fa218dc28d0 |
| command | sdlc milestone-close --issue 395 --milestone M5 |
| reviewer | claude |
| timestamp | 2026-10-07T23:55:39-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

M5 meets its plan as revised on 2026-10-07 (M5 design) and 2026-10-08 (xterm 6.0.0). The window delivers:
- named tunnels over a private unix socket and quick tunnels over TCP loopback;
- a probe that resolves through 1.1.1.1;
- a guard process tied to Couch by a stdin pipe;
- run records that double as the named-tunnel lock, with startup reaping;
- the README and atlas updates;
- the upgrade to xterm.js 6.0.0, with hashes that match `VENDOR.md`.

The live smoke and the `kill -9` crash test are logged, and the operator confirmed both rounds fixed. `go test ./cmd/internal/broadcast` passes (unsandboxed), and so does the couchcmd `TestBroadcastSettings`. The full couchcmd package still fails one test in this environment, `TestInteractiveLaunchStartsNewWhenNoSessionSurvives` ("operation not permitted"), which isn't in the M5 diff.

Nothing blocks the boundary. Two error-path problems are cheap to fix and worth fixing before close:
- **Quick-tunnel URL match is too loose.** The regex also matches cloudflared's own API host in its failure line, so a failure looks like success.
- **Named-tunnel lock has a race.** The reap step removes records by path after reading them, which can delete a lock another Couch just created.

The xterm.js upgrade and the four-face font wait are outside the original M5 scope, but they are declared in the plan's Revisions and backed by smoke evidence. I accept them.

### 1. Strengths
- **Guard design is correct.** `cloudflared.go:274-297`: Close closes the pipe and never signals the guard, which would orphan cloudflared. `guard.go:70-88` sends TERM, then KILL, then sweeps the process group. The `kill -9` crash test confirms it end to end.
- **Reaping is defensive.** `records.go:134-157` requires a dead owner, a matching `StrictIdentity` and the expected command before killing anything. It also refuses to act when the platform gives no identity. `TestReapOrphans` covers dead owner, live owner, recycled PID, wrong command and foreign directory.
- **The token stays out of errors.** `session.go:194-199` strips the `*url.Error` text, and `TestSessionProbeErrorCarriesNoToken` pins it.
- **Settings are validated before they reach the YAML config.** `couchcmd/broadcast.go:13-14` checks the tunnel name and hostname by regex before `fmt.Sprintf` writes them into `config.yml`, so they can't inject YAML (ARCH-SECURE). The table test covers path-traversal and hostname-with-path cases.
- **`Listen` cleans up after itself** (`cloudflared.go:77-81`). Closing the listener removes the private directory, so an abandoned start leaves nothing behind.

### 2. Critical
None.

### 3. Important
- **`cloudflared.go:37,156`: the quick-tunnel URL regex matches cloudflared's API host.**
  - When cloudflared can't reach its quick-tunnel API (offline, DNS failure), it logs `failed to request quick Tunnel: Post "https://api.trycloudflare.com/tunnel": …` and exits.
  - The regex `https://[a-z0-9-]+\.trycloudflare\.com` matches `https://api.trycloudflare.com`. `found` usually wins the select against `exited`, so `Open` succeeds with the API host as the URL.
  - `Session.Start` then probes for the full `DefaultProbeTimeout` (30s). It sends `/<token>/` to the API host and finally reports a misleading probe error instead of cloudflared's real failure line.
  - Related: `probe` (`session.go:96`) doesn't watch `handle.Exited()`, so a tunnel that dies during the probe also costs the full 30s.
  - Fix: anchor readiness on the banner line, or exclude `api.`. Make `probe` also return on `handle.Exited()`. Add a fake-cloudflared test that prints the real failure line and exits.
- **`records.go:114-143`: the reap step can delete a live named-tunnel lock.**
  - `reap` reads a record, decides its owner is dead, then calls `os.Remove(path)`. Nothing re-checks that the file is still the one it read.
  - Interleaving: Couch A and Couch B both read a stale `named-x.json`. A removes it and claims a fresh one with `O_EXCL`. B, still inside its `ps` calls, removes A's fresh lock and then claims its own.
  - Result: two connectors on one tunnel, which is exactly what `ErrTunnelBusy` exists to prevent. `ReapOrphans` at startup widens the window, since a starting Couch can race another Couch's broadcast start.
  - This violates the ARCH-ORDER guidance about a second actor on the same state.
  - Fix: hold an `flock` on `<records>/.lock` across reap+claim, or rename the stale file to a unique name, re-read it and compare before removing. Add a test with two concurrent claimers.

### 4. Minor
- `records.go:129-133`: if `StrictIdentity` fails briefly when `claim` runs, the record is written with an empty `OwnerID`. `reap` never reaps such a record, so a named tunnel stays `ErrTunnelBusy` for good after a crash, and the message ("another Couch is broadcasting") is misleading. Fix: refuse to claim without an owner identity, or name the record path in the error.
- `records.go:139`: `ReapOrphans` trusts the persisted `PrivateDir`, checking only that its base name has the `couch-broadcast-` prefix. A hand-edited or corrupted record could `RemoveAll` any directory with that name anywhere. Fix: also require it to sit under `os.TempDir()` or `RunDir`.
- `records.go:80-96`: if the `record()` write fails, nothing reports it. The record then has no cloudflared PID and reaping can't kill it, leaving only the guard as backstop. Log the failure or return it.
- `cloudflared_test.go:193`: the "cancelled" fake writes `$$` to `os.TempDir()/unused`, outside `t.TempDir()`, and leaves the file behind. `_ = dir` is dead code. This touches the lessons.md (#399) test-isolation rule.

### 5. Test coverage
- Covered well: parsing settings; Listen falling back to TCP; the listener removing its directory; open failures (missing binary, early exit, timeout, cancel); Close stopping the tunnel through the guard; Exited when the tunnel dies; named-tunnel exclusivity; reap cases; guard escalation and exit passthrough; probe resolver injection.
- Missing: the real cloudflared failure line, which carries the API URL; a tunnel exiting during the probe; concurrent claim/reap on one named tunnel.
- `TestCloudflaredLive` is opt-in, and `atlas/broadcast.md` names it as the live check to rerun after cloudflared upgrades (ARCH-MOCK).

### 6. Architecture
- **ARCH-DRY: pass.** It reuses `procutil` identity and liveness and the `Tunnel`/`Handle` interface. There's no duplicated parsing.
- **ARCH-PURE: pass (mostly).** `parseGuardArgs`, `ParseGuardChild`, `tunnelArgs` and the settings parser are pure. Process and file IO stays in `Open`, `RunGuard` and `reap`, which are necessarily IO glue.
- **ARCH-PURPOSE: pass.** It delivers a reachable link through both tunnel modes, plus crash safety, which is the issue's purpose.
- **ARCH-MOCK: pass.** A stateful fake cloudflared script sits behind the same `Binary` seam that production uses, and there's a live check.
- **ARCH-CONSTRAINTS: pass with a flag.** Open has a 30s timeout, Close a 5s budget and the guard a 3s grace, all enforced. The probe ignoring tunnel exit wastes that 30s budget (Important #1).
- **ARCH-SECURE: pass with Minor flags.** The token stays out of errors, settings are validated by regex before YAML, and directories are 0700. The `PrivateDir` read from a persisted record is trusted too far (Minor).
- **ARCH-ORDER: flag.** Reap followed by claim isn't atomic across Couch processes (Important #2). Within one process, the guard's child PID is recorded before readiness because both come from the same stderr reader, which is correct.
- **ARCH-FUNERAL: pass.**
  - Private directories die with the listener's Close, the guard, or the reap.
  - Records die on release, or on reap once the owner is dead.
  - The `<store>/broadcast/` directory is bounded by the number of live tunnels.

### 7. Plan revisions
None required: the M5 and xterm 6.0.0 revisions match the code. When the Important fixes land, add a short Revisions note recording the readiness anchor and how the lock is made atomic.

```findings
findings:
  - id: new
    severity: Important
    family: tunnel-readiness-signal
    title: |
      Quick-tunnel URL regex matches cloudflared's API host in its failure line; probe ignores tunnel exit
    detail: |
      cloudflared.go:37,156 matches "https://api.trycloudflare.com" in "failed to request quick Tunnel: Post https://api.trycloudflare.com/tunnel" when cloudflared is offline or can't reach its API, so Open succeeds with the API host. session.go:96 probe then sends /<token>/ there for the full 30s DefaultProbeTimeout and reports a misleading error instead of cloudflared's failure line. Fix: anchor on the banner or exclude api., have probe also return on handle.Exited(), and add a fake-cloudflared test that prints the real failure line.
  - id: new
    severity: Important
    family: lock-check-then-act
    title: |
      reap removes records by path after reading them, so a concurrent Couch can delete a fresh named-tunnel lock
    detail: |
      records.go:114-143: two Couches read the same stale named-x.json; A removes it and claims with O_EXCL; B, still inside its ps checks, os.Remove's A's fresh lock and claims too, giving two connectors on one tunnel and defeating ErrTunnelBusy. Fix: hold an flock around reap+claim, or rename-then-verify before removing; add a two-claimer test.
  - id: new
    severity: Minor
    family: lock-check-then-act
    title: |
      A record claimed without an owner identity is never reaped, so the named tunnel stays busy after a crash
    detail: |
      records.go:59,129: if StrictIdentity returns "" at claim time, reap skips the record forever, and the error says "another Couch is broadcasting". Refuse to claim without an identity, or name the record path in the error.
  - id: new
    severity: Minor
    family: persisted-record-trusted
    title: |
      ReapOrphans RemoveAll's a PrivateDir read from a persisted record, checked only by its base-name prefix
    detail: |
      records.go:139: a hand-edited or corrupted record can point at any directory named couch-broadcast-* anywhere. Also require it to sit under os.TempDir() or RunDir.
  - id: new
    severity: Minor
    family: silent-error-swallow
    title: |
      runRecord.record ignores write and rename failures, leaving the reap step unable to kill cloudflared
    detail: |
      records.go:88-95. This is the 2nd finding in this family; the rule is that a failed persistence write must reach a log or a returned error.
  - id: new
    severity: Minor
    family: test-writes-outside-tempdir
    title: |
      The "cancelled" open test writes os.TempDir()/unused and leaves it behind
    detail: |
      cloudflared_test.go:193; the _ = dir is dead code too.
```
