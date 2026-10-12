# Boundary Review — pair#477 (milestone M1)

| field | value |
|-------|-------|
| issue | 477 — broadcast-web: replay inside a running Couch, for remote viewers (mount ReplayServer on the shared Host) |
| repo | pair |
| issue file | workshop/issues/000477-broadcast-web-replay-inside-a-running-couch-for-remote-viewers-mount-replayserver-on-the-shared-host.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | c5d4d61469037a3d142d379583b4f506cb3016af..5431ed7404605f249b7b14f18d2ec2a312c60072 |
| command | sdlc milestone-close --issue 477 --milestone M1 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T06:04:03-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Reading host.go confirmed lost mounts are removed from the Host; now checking query arg defaults and running the tests.

Tests all pass. Checking whether the "mount completes after Close" branch is actually exercised by reverting it.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M1 does what the plan says and works. Replay shares mount one recording per token on the shared `broadcast.Host`. The `replay-share` / `replay-status` broker ops, both CLIs' share-and-poll, `--local`, the loopback fallbacks and the "length unknown" advisory are all in place. Every affected test package passes under `-race`; to compile `couchcmd` in this checkout I first had to run `runtimebundle-generate`, which writes gitignored assets. The Share/Status split recorded in Revisions is the right call, and `TestReplaySharesFailures` covers it. Nothing blocks the boundary.

Three things should be fixed first:
- **Untested Close race.** The "a mount that completes after Close unmounts at once" branch never runs in any test. When I disabled it, the suite stayed green.
- **No operator visibility.** An agent can publish a recording to the internet, and the operator has no way to see or revoke it.
- **Stale atlas.** The agent-queries section still says the query CLI never needs Couch and doesn't mention `link`.

## 1. Strengths
- `webcast/shares.go:62-172`: the share states are one `ShareState` under one mutex, matching the documented transition table. `Status` only reads and `Share` only starts, so a poll can't restart a failed share (ARCH-ORDER done right on the read side).
- **No stale-token exposure.** `NewRecordingServer` scopes a token to one recording, and `TestRecordingServerServesOnlyItsRecording` covers traversal and encoded-slash 404s. A lost share's old token can't outlive its tunnel: `Host.watch` deletes lost mounts (`broadcast/host.go:248-251`), and a reshare mints a new token.
- **End-to-end chain test.** `TestReplayShareThroughTheBroker` (`couchcmd/replay_share_test.go:612`) round-trips JSON through the real `messageService.handle`, the console, the shares and a `FakeTunnel` Host, then fetches the records through the printed link. Production and test use the same boundary (ARCH-MOCK pass).
- **Tidy-ups.** `predatesOp` pulls the old-Couch detection out of `broadcast_list.go` into a shared helper (ARCH-DRY). The torn-line regex `^\{"t":(\d+)[,}]` comes with a table test of torn, garbage, reordered and overflow last lines.

## 2. Critical findings
None.

## 3. Important findings
- **The late-completion Close branch has no test (ARCH-ORDER)** — `webcast/shares.go:120-127`.
  - **What's wrong:** if `Host.Mount` succeeds between `Close` taking the lock and the goroutine re-locking, the goroutine must unmount. Otherwise the Host (and its cloudflared tunnel) stays open after Couch's teardown.
  - **Evidence:** I replaced `m.Unmount()` with a no-op and `go test -count=3 -run ReplaySharesShares ./cmd/internal/webcast` still passed. `TestReplaySharesCloseWhileOpening` only covers the abandoned-open path (`Host.Mount` returns `ctx.Err`, `m == nil`), and it orders events with `time.Sleep(20ms)`. That is one observed interleaving.
  - **Fix:** add a seam so the test controls when the mount completes, for example an injectable `mountFn`, or a FakeTunnel/probe hook that blocks until released. Then test "probe succeeds → Close → release". Assert `Opens == Closes` and that the Host has no mounts.
- **Agent-initiated public sharing is invisible to the operator and can't be revoked (ARCH-SECURE)** — `couchcmd/replay.go:378`, `couchtty/console_webcast.go:921`.
  - **What's wrong:** an agent's `--broadcast-web-query DIR link` opens a public tunnel serving a recording, which may contain sensitive page content. Nothing shows it: not `--broadcast-list`, not the status bar. It stays up until Couch exits.
  - **Why it matters:** the plan calls an unshare verb a follow-up. But visibility is the minimum that the cast's privacy model (LIVE indicator, #395) implies.
  - **Fix:** report a ready-share count in `BroadcastStatus` / `--broadcast-list`, or in the Chrome/broadcast status cell.
- **The atlas agent-queries section is stale** — `atlas/webcast-format.md:423-433`.
  - It still says the query CLI "needs no running Couch" and that "`Main` is the CLI".
  - The verb table lists only dom, events and notes. It needs a `link [--at T]` row, plus a note on `query.Run` / `Linker`.
  - `atlas/webcast.md` was updated; this section wasn't.

## 4. Minor findings
- `couchcmd/replay.go:392-394`: when the tunnel is off, `replayLinker` appends `"  (Couch's tunnel is off…)"` on the same stdout line as the URL. An agent that reads stdout as a URL gets a malformed value. Put the note on stderr (for example, return it alongside the URL), the way `runReplay` does.
- **Untested `Local` messages:** the `link.local` paths in both CLIs (the `runReplay` stderr note and the `replayLinker` suffix) have no couchcmd test. Only the live test checks `st.Local`.
- `webcast/query/cli.go:945-947`: the new Usage line is far too long and runs into the old "Times are ms…" text. Re-wrap it.
- `query.Main` is now used only by tests (`cli_test.go:16`). Drop it, or document it as a test convenience.
- **Duplicated error classification (ARCH-DRY):** `runReplay` and `replayLinker` each sort the same three sentinel errors (`errNoCouch`, `errCouchPredatesReplay`, `errNotInStore`). A small `canFallBack(err)` helper would keep the two CLIs in step.
- **Real-time sleeps in tests:** `shares_test.go` relies on real time (`OpenDelay`, a 5 ms poll, and a baseline from `runtime.NumGoroutine()`). It works, but it's the flake-prone shape. This is the same seam as the first Important finding.

## 5. Test coverage notes
- Covered: the share states (open, ready, lost then reshared, failed then retried), one mount for concurrent Shares, Close while opening and while ready, the scoped server, `ValidateRequest`, the CLI poll (opening→ready, deadline, closed, failed, no Couch, old Couch, outside the store), the loopback fallback and `--local`, `query link`, the torn-line advisory, and the full broker chain.
- **Gaps:**
  - The probe-completes-after-Close race (above).
  - `Local` output in the CLIs.
  - A lost share observed by the CLI (the poll seeing `lost` after `ready`).
- **Real tunnel:** the live Chrome test uses a `LocalOnly` Host. A test through a real tunnel is still the operator's smoke test, as planned.

## 6. Architectural notes for upcoming work
- ARCH-DRY: pass. `predatesOp` is extracted; the duplicated sentinel switch is noted above.
- ARCH-PURE: pass. `shareReplay` takes `messageCall` and `slotPollClock` as injected parameters; `ReplayShares` logic is small and sits behind the Host seam.
- ARCH-PURPOSE: pass for M1. Notes and strokes are M2, gated on #470. The issue's "menu action on a recording" was an example ("e.g."), and the CLI route satisfies it.
- ARCH-MOCK: pass. `FakeTunnel` sits behind the real Host, a scripted broker stands in for the socket, and the chain test uses the real handler.
- ARCH-CONSTRAINTS: pass. The 2 s socket limit is respected by the async poll (300 ms interval, 45 s cap, past the 30 s probe). Shares are bounded by the number of recordings on disk, and unknown ids insert nothing.
- ARCH-SECURE: flag (Important above). Otherwise sound: ids are checked against the shape regex before any path join, links stay inside the one recording, and the token goes back only over the operator-only socket.
- ARCH-ORDER: flag (Important above). The state enum is explicit, but the late-completion transition has no test and no ordering seam.
- ARCH-FUNERAL: pass with a note. Shares and tokens die at Couch teardown, which the plan states. The tunnel stays open for the rest of the session once anything has been shared. Pair the future unshare/expiry follow-up with the visibility fix.

## 7. Plan revision recommendations
- Add a Revisions entry: "`query.Run(argv, Linker, …)` replaces `Main` as the production entry point; `Main` remains for tests." The plan's CLIs section says "`Main` takes a `Linker`", which isn't what the code does.
- The Tests section claims Close-while-opening coverage of the "mount completes after Close" transition. Either add that test (preferred), or record that only the abandoned-open path is covered.

```findings
findings:
  - id: new
    severity: Important
    family: untested-interleaving-branch
    title: |
      ReplayShares mount-completes-after-Close unmount branch is unexercised; reverting it keeps tests green
    detail: |
      shares.go:120-127. Disabling m.Unmount() in the closed branch passes all share tests (-count=3); CloseWhileOpening only hits the abandoned-open path via a 20ms sleep. Add an ordering seam (blockable mount/probe) and test probe-success -> Close -> release, asserting Opens==Closes.
  - id: new
    severity: Important
    family: public-exposure-without-operator-visibility
    title: |
      Agent-triggered replay shares publish recordings via the tunnel with no operator-visible indicator or listing
    detail: |
      query link lets any agent open a public tunnel serving a recording until Couch exits; neither --broadcast-list nor the status bar shows active shares. Add a ready-share count to BroadcastStatus/status cell (unshare can stay a follow-up).
  - id: new
    severity: Important
    family: atlas-surface-not-updated
    title: |
      atlas/webcast-format.md agent-queries section still says the query CLI needs no Couch and omits the link verb
    detail: |
      Lines 423-433: add a link [--at T] row, note it asks the running Couch, and replace "Main is the CLI" with query.Run/Linker.
  - id: new
    severity: Minor
    family: machine-output-contract
    title: |
      query link appends a tunnel-off note to the URL on stdout, breaking agents that read stdout as a URL
    detail: |
      replay.go:392-394; put the note on stderr as runReplay does.
  - id: new
    severity: Minor
    family: missing-branch-test
    title: |
      ShareStatus.Local paths in runReplay and replayLinker have no couchcmd test
  - id: new
    severity: Minor
    family: usage-text-formatting
    title: |
      query Usage link line is overlong and runs into the existing "Times are ms" sentence
  - id: new
    severity: Minor
    family: duplicated-error-classification
    title: |
      runReplay and replayLinker each classify the same three fallback sentinels; extract a shared predicate
  - id: new
    severity: Minor
    family: dead-production-entrypoint
    title: |
      query.Main is now used only by tests; production calls query.Run
```

---

## Re-review — 2026-10-12T06:14:17-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 477 — broadcast-web: replay inside a running Couch, for remote viewers (mount ReplayServer on the shared Host) |
| repo | pair |
| issue file | workshop/issues/000477-broadcast-web-replay-inside-a-running-couch-for-remote-viewers-mount-replayserver-on-the-shared-host.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | c5d4d61469037a3d142d379583b4f506cb3016af..158f9fd912871682fa7e38a90c319ca0780ca69b |
| command | sdlc milestone-close --issue 477 --milestone M1 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T06:14:17-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Round 1 fix commit is 158f9fd91. Let me look at it first.

Checking how to generate the embedded runtime bundle so couchcmd tests can compile in this checkout.

All green. Now reverting the BR-1 unmount to confirm the new test catches it.

BR-1 confirmed: the new test fails without the unmount. Now checking the BR-2 notice/listing wiring.

Now checking whether the operator notice half of BR-2 is pinned by any test.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

All eight round-1 findings are addressed, and I checked the evidence for each one directly. I did not rely on the commit message. BR-1 is now pinned by a real ordering seam: I replaced `m.Unmount()` with a no-op and `TestReplaySharesMountCompletingAfterClose` failed 3 of 3 runs (`Opens:1 Closes:0`). With the code restored, `go test -race -count=1` passes on webcast/..., couchcmd, couchtty, couchmessage and artifactpath. One caveat: this checkout had to run `go run ./cmd/internal/runtimebundle/generatecmd` first, because the embedded bundle is generated and not committed. That is an environment step, not a defect. Nothing blocks shipping. One Important item remains: `README.md` was not updated for the new `--broadcast-list` output. Two Minor items concern the operator notice that round 1 added.

**1. Strengths**
- `cmd/internal/webcast/shares.go:147-157`: the test-only `mounted` seam lets the test control ordering exactly. The late-mount branch is now driven on purpose instead of hit by a sleep, and it fails when reverted.
- `cmd/internal/couchcmd/replay.go:183-187`: `cantShare` and `tunnelOffNote` remove the duplicated classification and note text, so `runReplay` and `replayLinker` share one source.
- `TestReplayLinkTunnelOff` (`replay_share_test.go:255-272`) asserts that stdout is exactly the URL for both CLIs. That pins the machine-output contract from BR-4 and BR-5.
- Teardown ordering is sound. `replays.Close()` (`console.go:991`) waits for every mount goroutine, including the `onReady` → `GoTracked` call, before `started=false` and `workers.Wait()`. So a late notice can't call `workers.Add` after `Wait`.
- Neither the notice nor the `broadcast-status` listing carries the token. This keeps the "never list the link" rule from #413.

**2. Critical findings**
None.

**3. Important findings**
- **`README.md:400`, `README.md:867-870`: README not updated for the `--broadcast-list` replays line or the `replays` JSON key.**
  - The synopsis still reads "the running broadcast and its viewer count", and the prose (867-870) doesn't mention shared replays. The tool now prints `replays shared until Couch exits: …` and a `replays` JSON field (`broadcast_list.go:41,49`).
  - `README.md:872` still lists `dom|events|notes` with no `link`.
  - **This is the 2nd finding in family `atlas-surface-not-updated`.** The rule: a new user-visible surface (verb, flag, output line, JSON key) must be swept through every place that describes it in the same commit. Those places are the README synopsis, README prose, atlas, `SKILL.md` and the `Usage` text. Round 1 fixed atlas only. Fix all of them by grepping each surface name across these five places.

**4. Minor findings**
- **The new share-ready notice is untested** (`console_webcast.go:257-262`). I replaced `c.GoTracked(func(){ c.setNotice(msg) })` with a no-op and the couchcmd and couchtty Replay/Share/Broadcast tests stayed green. Only the listing half of BR-2 is pinned. This is the 2nd finding in family `missing-branch-test`. The rule: every behavior a fix adds needs its own test that fails without it. Round 1 added two (listing and notice) and pinned one.
- **The notice text is wrong when the tunnel is off** (`console_webcast.go:259`). It says "shared through the broadcast tunnel" even when `ShareStatus.Local` is set, meaning nothing is public. It should branch on `Local`, as both CLIs already do.
- **The no-Couch fallback message now includes the raw transport error** (`replay.go:130` with `216`), e.g. `no running Couch answered (dial unix …: no such file or directory).` It's acceptable, but it's noisier than the old "No running Couch to share it."

**5. Test coverage notes**
- The late-mount-after-Close path, the stdout/stderr split for tunnel-off links in both CLIs, `formatSharedReplays`, and the `Replays` field of `broadcast-status` are all pinned now.
- Two things remain untested:
  - the console notice;
  - the `--json` encoding of `replays` in `runBroadcastListCLI`.

**6. Architectural notes**
- **ARCH-DRY: pass.** Sentinels and the note are consolidated.
- **ARCH-PURE: pass.** Share-and-poll takes an injected `messageCall` and clock, and `formatSharedReplays` is pure.
- **ARCH-PURPOSE: pass.** M1 delivers shares, broker ops, both CLIs, the advisory, atlas and skill. The only miss is the README sweep above.
- **ARCH-MOCK: pass.** The stateful `FakeTunnel` sits behind the real `Host`, and the same seam is used in production.
- **ARCH-CONSTRAINTS: pass.** A 300 ms poll with a 45 s cap stays under the broker's 2 s per-exchange limit.
- **ARCH-SECURE: pass.** Each share has its own token and a scoped server. The ids in the listing and notice are not credentials.
- **ARCH-ORDER: pass.** The state table is documented in `shares.go:37-42`, and the one hard interleaving now has a controllable seam. `onReady` and `mounted` are set without synchronization under a "call before Share" contract. That's fine as long as only `SetReplayShares` sets them. If they ever become runtime-settable, move them into the constructor.
- **ARCH-FUNERAL: pass.** Shares live in memory, are bounded by the recordings requested per run, and end at Couch exit. Unshare and expiry are an explicit follow-up.

**7. Plan revision recommendations**
- Add a `## Revisions` entry recording the operator visibility added in round 1. `Response.Replays` on `broadcast-status`, the `--broadcast-list` line and JSON key, and the `OnReady` notice were not in the accepted design's Lifecycle section, which said "each one prints the link".
- In the Architecture section, change "`Main` takes a `Linker`" (plan line 73) to `Run(argv, Linker, …)`, since `Main` has been removed.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Reverted m.Unmount() in shares.go:154; TestReplaySharesMountCompletingAfterClose fails 3/3 (Opens:1 Closes:0); green restored.
  - id: BR-2
    disposition: addressed
    note: |
      broadcast-status carries Replays, --broadcast-list prints them, pinned in TestReplayShareThroughTheBroker; notice half unpinned (raised separately).
  - id: BR-3
    disposition: addressed
    note: |
      atlas/webcast-format.md:423-432 now lists link, says it asks the running Couch, names Run/Linker.
  - id: BR-4
    disposition: addressed
    note: |
      replay.go:265-267 writes tunnelOffNote to stderr; TestReplayLinkTunnelOff asserts stdout is exactly the URL.
  - id: BR-5
    disposition: addressed
    note: |
      TestReplayLinkTunnelOff covers Local for both runReplay and replayLinker.
  - id: BR-6
    disposition: addressed
    note: |
      query/cli.go Usage reflowed; link sentence separated from the Times sentence.
  - id: BR-7
    disposition: addressed
    note: |
      cantShare predicate at replay.go:183 used by both callers.
  - id: BR-8
    disposition: addressed
    note: |
      query.Main removed; tests call Run(argv, nil, ...), nil linker refuses.
findings:
  - id: new
    severity: Important
    family: atlas-surface-not-updated
    title: |
      README not updated for --broadcast-list shared-replays line, replays JSON key, and link in the query prose
    detail: |
      README.md:400 and 867-870 omit the new replays output; 872 still says dom|events|notes. 2nd in family: rule is every new user-visible surface (verb, flag, output line, JSON key) is swept across README synopsis+prose, atlas, SKILL.md and Usage in the same commit; grep each surface name across all five.
  - id: new
    severity: Minor
    family: missing-branch-test
    title: |
      Share-ready operator notice (console_webcast.go:258-261) has no test; disabling it keeps tests green
    detail: |
      2nd in family: each behavior a fix adds needs its own failing-without-it test; round 1 added listing and notice but pinned only the listing.
  - id: new
    severity: Minor
    family: operator-message-accuracy
    title: |
      Share-ready notice says "through the broadcast tunnel" even when the share is Local (tunnel off)
    detail: |
      console_webcast.go:259; branch on ShareStatus.Local as both CLIs do.
```

---

## Re-review — 2026-10-12T06:19:46-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 477 — broadcast-web: replay inside a running Couch, for remote viewers (mount ReplayServer on the shared Host) |
| repo | pair |
| issue file | workshop/issues/000477-broadcast-web-replay-inside-a-running-couch-for-remote-viewers-mount-replayserver-on-the-shared-host.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | c5d4d61469037a3d142d379583b4f506cb3016af..7f856ae6f6e73f1c6c8e0478d2be8d3420566801 |
| command | sdlc milestone-close --issue 477 --milestone M1 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T06:19:46-07:00 |
| verdict | SHIP |

## Review

Focus first on the open findings: the round-2 commit.

Running the tests for the round-2 fixes needs a generated runtime-asset bundle that this checkout doesn't have. I'm looking for the generator now.

Tests pass. Now confirming the notice test fails when the fix is reverted (BR-10/BR-11).

Both reverts go red. Next, a quick read of the core new code (shares.go and the replay paths) for anything new.

```verdict
verdict: SHIP
confidence: high
```

**Verdict: SHIP.** All three open findings are fixed, and I found nothing new. The round-2 fix commit (7f856ae6f) updates every place the new "shared replays" surface needed to appear. That covers the README synopsis (`README.md:400`) and prose (`README.md:868-882`), the `couch --help` text (`run.go:1315-1316`) and `atlas/broadcast.md:431-433`. The query prose now reads `dom|events|notes|link` and states the stdout/stderr contract. The share-ready notice now checks `ShareStatus.Local`, and a new table-driven test covers both the tunnel and the local-only case.

I confirmed the test is real by reverting each fix in the disposable checkout. Each revert made `TestReplayShareReadyIsANotice` fail. Before the tests would build, I had to generate the embedded runtime assets with `go run ./cmd/internal/runtimebundle/generatecmd`, because this checkout doesn't include them. That is a checkout setup gap, not a problem with the diff. `couchtty`, `couchcmd`, `webcast` and `couchmessage` all pass. `artifactpath` was only run before the assets were generated, and failed then on stale inventory entries for the missing runtime files plus one build failure caused by those missing assets; I did not re-run it after generating them.

### 1. Strengths
- **Notice change:** passing the ready `ShareStatus` into `OnReady` (`shares.go:164-168`) lets the notice report local-only correctly. The status is copied while the lock is held, so the callback can't see a half-updated status.
- **Notice test:** `console_webcast_test.go:299-326` checks three things in both modes:
  - the exact notice text;
  - that the notice never contains the link's token;
  - that `SharedReplays` lists the recording.
- **Untrusted recording id:** it is checked against a fixed shape at the boundary (`replay.go:62-67, 120-126`), and each share is served by a server scoped to that one recording (`NewRecordingServer`). One link can only open the recording it names.
- **Share lifecycle:** the states and transitions are written out in the comment at `shares.go:36-45`. The case where a mount completes after `Close` is handled and tested.

### 2. Critical
None.

### 3. Important
None.

### 4. Minor
None new.

### 5. Test coverage notes
- **BR-10 and BR-11:** I removed the notice call, then separately removed the `Local` branch. Each change made `TestReplayShareReadyIsANotice` fail, and restoring the code brought it back to green.
- **Other packages:** `couchcmd`, `couchtty` and `webcast` all pass once the runtime assets are generated.

### 6. Architectural notes
- **ARCH-DRY:** pass. The shared fallback check from BR-7 is still in place, and nothing new is duplicated.
- **ARCH-PURE:** pass. The listing format (`formatSharedReplays`) is a pure function, and the notice code is a thin layer of glue.
- **ARCH-PURPOSE:** pass. The rule from BR-9 was applied across all five documentation surfaces and recorded in `workshop/lessons.md`.
- **ARCH-MOCK:** pass. Tests run against `broadcast.FakeTunnel` and `LocalOnly` through the production `Host` seam.
- **ARCH-CONSTRAINTS:** pass. `Share` never waits on the mount, and there is at most one mount per recording.
- **ARCH-SECURE:** pass. The recording id is checked on arrival, no link reaches the notice or the listing, and the token is minted per share.
- **ARCH-ORDER:** pass. The share states are written out explicitly. The late-mount case is reproducible through the `mounted` test hook.
- **ARCH-FUNERAL:** pass. Shares exist only in memory, end at `Close` when Couch shuts down, and are bounded at one per recording asked for.

### 7. Plan revision recommendations
None.

```findings
dispose:
  - id: BR-9
    disposition: addressed
    note: |
      README.md:400-401 synopsis and 868-882 prose now name the replays line, the replays JSON key and link (dom|events|notes|link, stdout-only URL); help run.go:1315 and atlas/broadcast.md:431 also swept.
  - id: BR-10
    disposition: addressed
    note: |
      TestReplayShareReadyIsANotice (console_webcast_test.go:299) goes red when setNotice is removed; verified by revert in the review checkout.
  - id: BR-11
    disposition: addressed
    note: |
      console_webcast.go:258-262 branches on ShareStatus.Local passed via OnReady; the LocalOnly test case goes red when the branch is disabled.
```
