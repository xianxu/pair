# Boundary Review — pair#287 (whole-issue close)

| field | value |
|-------|-------|
| issue | 287 — New sessions die at birth: title poller probes zellij during server startup |
| repo | pair |
| issue file | workshop/issues/000287-new-session-startup-race.md |
| boundary | whole-issue close |
| milestone | — |
| window | 6d378d29732bd65c6065bb559b76424795bc92c8..d1cdf69f86063a1ad4fce45aaef942d720cb203d |
| command | sdlc close --issue 287 |
| reviewer | claude |
| timestamp | 2026-09-18T16:05:26-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

The diagnosis and the fix are sound: the birth evidence (the agent pane's sidecar) is the right signal, the freshness argument rests on ordering rather than clocks, and both measured probers — the title poller and Couch's 10 ms cold-resume poll — are gated behind it, with a live probe showing 4/10 → 0/20. Tests assert *order* (an event log in the poller fake, seam hooks in the Couch fake), not just end state, which is what ARCH-ORDER asks for. What keeps this from a clean SHIP is one new failure path: a cold resume whose thread already has a live session can never observe a birth, so registration now times out and the cold-resume cleanup **deletes that live session** — a case the Couch fake was taught to fabricate a birth for, which is why no test catches it. Everything else is minor. I ran the suites: `titlepoller`, `artifactpath`, `probes/...` pass; `couchcore` (118 s) and `launcher` fail only on `ptychild: operation not permitted`, the known pty/sandbox environment limit, in tests this diff does not touch.

**1. Strengths**

- `cmd/internal/couchcore/panebirth.go:24` — `PaneMarks.BornIn` compares mtimes by **equality only**, never ordering, so the wait needs no clock and cannot be fooled by coarse or skewed timestamps; the table test covers clear-then-rewrite, the stale twin, and the nil baseline.
- `cmd/internal/couchcore/launch_existing.go:140-152` — taking the baseline *while the helper is still blocked* is the right ordering primitive: it is provably before anything Pair does, so "changed since" means "this launch's doing" without a nonce or a contract field.
- `cmd/internal/titlepoller/run_test.go:405-441` and `coldresume_birth_test.go:38-62` — the tests observe the interleaving through a seam (`sleepHook`, `PaneSidecarsHook`, `BeforePairSession`) rather than sampling one lucky ordering; a reverted gate goes red on the event sequence, not on a flake.
- `probes/zellijbirthrace` — real conformance against the real dependency, before and after, with the residual (9/10 under an external hammer) reported rather than buried, and the shared-log attribution caveat stated in SKILL.md:72.
- `cmd/internal/artifactpath/manifest.go:115,422,662` — the new consumer is registered in the binding manifest instead of spelling paths by hand; the fake was cleaned of literal artifact paths.

**2. Critical findings**

None.

**3. Important findings**

- **`cmd/internal/couchcore/launch_existing.go:146` (and `:337`) — a cold resume against an already-live session times out, and cleanup deletes that session.** `DetachedSessions` proves *detached*, so a session that is live **but attached** falls to `StartColdResume`. The helper then runs `pair resume <tag> --resume <id>`, which `createflow.go:341` refuses (`ResumeRequired` with `decision.Action != ActionCreate`), so no pane sidecar is ever rewritten. `awaitPaneBirth` spins to the registration deadline → `failTrackedPostAckStart(shape=cold)` → `quiescePostAckStart` sees `OwnsSession()==true` → `Artifacts.Quiesce` → `QuiesceThreadSession` deletes the live session by name. Pre-#287 the same sequence registered immediately (PairSession present) and destroyed nothing; this diff widens the blast radius to exactly the harm pair#230's comment says must never happen ("would delete a session holding somebody's agent"). Fix sketch: before `h.Acknowledge()` — when no birth of *this* thread can be in flight — observe `PairSessionContext` once and either skip the birth wait when the session is already present (restoring the old outcome) or refuse the cold resume outright, so cleanup never owns a session this start did not create.
- **`cmd/internal/couchcore/artifactcollision_fake.go:238-265` — the fake fabricates a birth the host cannot produce, which is why the above is untested.** `SetPairSession(...,true)`'s live edge and `PairLaunchReleased` both write a sidecar whenever the session is present. Neither holds on the real host: a session is *listed live* during the birth window before any pane runs (that window is this whole issue), and a released cold-resume launch against an already-live session is refused, so no pane is written. The `## Log` records that seven pre-existing tests were kept green this way instead of being corrected — which is the ARCH-MOCK failure mode (tune the double until the tests pass) rather than modelling the dependency. Fix sketch: birth only on the edge that a *create* launch produces (release + the session coming up **after** the release), and update the affected tests; `workshop/lessons.md:24` should likewise stop calling "a session live before its launch ran" an impossible world.

**4. Minor findings**

- `cmd/internal/couchcore/launch_existing.go:347` — the Couch wait aborts the start on any `PaneSidecars` error; the poller treats a failed stat as "not yet", and the plan specifies "not yet, never birth" for both. A transient stat error after a successful create fails registration and quiesces the session this launch just made. No test covers a mid-wait observation failure (only the baseline failure).
- `cmd/internal/couchcore/artifactcollision.go:244` — "A tag that merely shares a prefix is filtered out by AgentFromPane" overstates: for tags `work` and `work-2` in one scope, `pane-work-*.json` matches `pane-work-2-claude.json`, and `componentPattern` allows the hyphen in agent `2-claude`. Unreachable for Couch's fixed-shape `couch-<hex>` tags — but the comment claims a guarantee the code does not give.
- `workshop/issues/000287…md` Done-when — "before the fix most launches panic" against a measured 4/10 plain (10/10 only under the hammer). The atlas and SKILL.md carry the accurate figure; the issue is the artifact the close records as evidence.
- `cmd/internal/titlepoller/run.go:118` — `PaneChecked` error exits the poller with no diagnostic; fail-closed and consistent with the poller's silence, but titles then vanish with no trace.
- `probes/zellijbirthrace/main.go:330` — if the parent is killed or its budget expires, the detached child keeps running trials and the report temp dir survives; per-trial session/process teardown is unaffected.

**5. Test coverage notes**

- Covered well: no-zellij-call-before-birth (poller and Couch), grace expiry with zero probes, the attach one-stat shape, the launcher clear/no-clear pair, `BornIn`'s clear-then-rewrite and stale-twin rows, and the scoped `PaneSidecars` over a temp dir.
- Gaps: (a) the already-live cold resume (Important #1) — currently unrepresentable because of the fake's coupling; (b) a `PaneSidecars` error *during* the wait; (c) nothing ties the path the launcher clears to the path the poller stats — they are resolved in different processes, and a divergence would silently restore the stale-evidence race that the clear exists to prevent. A small cross-process assertion (same `ResolveScoped(dataDir, tag).Pane(agent)` on both sides, like `TestSessionEnvSuppliesEveryPairVariableThePollerReads` does for the env contract) would pin it.

**6. Architectural notes**

- ARCH-DRY: pass. One evidence family serves both consumers, and both resolve it through `artifactpath` rather than spelling names. Note: two `awaitPaneBirth` helpers now exist (`titlepoller/run.go:174`, `couchcore/launch_existing.go:342`) behind different seams — justified today, but #288 adds a third waiter in the launcher, and that is the moment to extract.
- ARCH-PURE: pass. `PaneMarks.BornIn` is pure with an IO-free table test; both gates sit behind existing seams; no new runtime method.
- ARCH-PURPOSE: pass. The sweep was done rather than claimed — the densest prober (Couch's 10 ms poll) was pulled into scope, and residuals are enumerated in the plan and atlas instead of being quietly dropped. Upstream filing is deferred to #288 with the draft verified and the operator's explicit call, and #288 carries the row.
- ARCH-MOCK: **flag** — Important #2. Live conformance via the probe is otherwise exactly right.
- ARCH-CONSTRAINTS: pass. 250 ms stat under a 30 s grace; 10 ms glob+stat replacing a `list-sessions` process inside the unchanged deadline; measured birth 0.65–1.41 s. Watch the Log's observation that birth time climbed monotonically across 20 trials in one scope — per-launch scope state growing without a bound is the ARCH-FUNERAL shape, and #218 is the right home for it.
- ARCH-SECURE: pass. `validateThreadAddress` before the glob, existence rather than content as the signal, probe isolates `XDG_DATA_HOME` and deletes only sessions named in Pair's own index.
- ARCH-ORDER: **flag** — Important #1 is an ordering hole: the wait has no transition for "the state it waits for was already satisfied before the launch", and it collapses that into failure, which cleanup then reads as permission to destroy. Otherwise strong: the create-path edges are pinned, and failure/timeout paths are tested.
- ARCH-FUNERAL: pass. Production creates nothing durable (it removes a file the pane rewrites); the probe removes all three temp dirs and its sessions on every path, enforced by `TestNoProbeExitsPastItsOwnCleanup`, which scans `probes/` and picks the new one up.

**7. Plan revision recommendations**

Add a `## Revisions` entry to `workshop/plans/000287-new-session-startup-race-plan.md` covering:

- **Failed-observation policy.** The ARCH-ORDER section states "A failed stat or glob counts as 'not yet', never as birth"; `awaitPaneBirth` in `couchcore` ends the wait with the error. Either revise the plan to say the Couch wait fails the start on an observation error, or change the code to match the plan (preferred — see Minor #1).
- **Fake surface drift.** The Core-concepts table lists `SetPaneSidecar(addr, agent, mtime)`; the implementation is `SetPaneSidecar(addr, agent)`, and the fake also gained `ClearPaneSidecar`, `PairLaunchReleased`, `PaneQueries`, `PaneSidecarsHook` plus `FakeRunner.OnRelease` — a new test-environment model of Pair that the plan does not mention and that Important #2 is about.
- **Interface location.** The table puts `PaneBirthIO.PaneSidecars` in `artifactcollision.go`; the interface is declared in `panebirth.go` (the method is in `artifactcollision.go`).
- **The already-live cold resume.** Whatever is decided for Important #1, the plan's Sweep table should carry the row: cold resume against a live-but-attached session, and what the birth wait does with it.
