# Boundary Review — pair#650 (whole-issue close)

| field | value |
|-------|-------|
| issue | 650 — Slug line off by default under Couch (PAIR_SLUG=on to re-enable): no per-turn pair slug spawn or Haiku call |
| repo | pair |
| issue file | workshop/issues/000650-slug-line-off-by-default-under-couch-pair-slug-on-to-re-enable-no-per-turn-pair-slug-spawn-or-haiku-call.md |
| boundary | whole-issue close |
| milestone | — |
| window | c0841e1082d39731781d8a2972eeeaed41d07ade..1ecb83d29deed3d4f82f8ea0a82dffdeb5e2f35f |
| command | sdlc close --issue 650 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-17T15:05:53-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

I've read the diff and checked the other slug entry points. Now I'm running the targeted tests and reverting each gate to confirm the tests go red without it.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The change does what the Spec asks. Both entries that can start a slug are gated: `maybeSpawnSlug` in pair-wrap, and `slugcmd.Run`, which the Claude Stop hook reaches through `bin/pair-slug`. The only other caller of `slugcmd.Run` is `dispatcher.go:282`, so the gate in `Run` covers it too. I reverted each gate in the checkout and both new tests went red: the wrapper test gave `spawns = 1, want 0`, and the slugcmd test reached `native session is unbound` instead of the gate line. With the gates restored, the targeted tests in wrapcmd, slugcmd and dispatcher pass. Three things stand between this and SHIP. The gate's rule is written out twice, in two packages (ARCH-DRY). `## Done when` doesn't describe the Stop-hook gate the Revisions added. And Done-when clause 2, the live check across the fleet, hasn't been done and I can't check it from here.

Done-when: stale — clause 1 only covers the wrapper's spawn. The `slugcmd.Run` gate added in Revisions isn't mentioned, even though it has its own test (`TestRunSkipsUnderCouchHosting`). Clause 2 says "no `pair slug` processes appear", but under Couch the Stop hook still starts a short-lived `pair-slug` process. That process stops at the gate and makes no model call, so the clause is wrong as written.

1. **Strengths**
   - `wrap.go:754`: the gate sits before the debounce bookkeeping, so a skipped turn doesn't touch `lastSlug`. It also writes a `SLUG-skip` line to the debug log, so a quiet slug can be traced.
   - Both gates reuse `launcher.CouchHostedEnv` rather than parsing the Couch env vars again.
   - Every test case sets `COUCH_THREAD_SCOPE`, `COUCH_THREAD_TAG` and `PAIR_SLUG` explicitly, so a test run inside a Couch-hosted session can't leak into the results. The dispatcher slug-route test (`dispatcher_test.go:276`) was updated the same way.
   - `slugcmd_test.go`: each case checks which guard fired by looking for its log line, not just exit code 0. Without that the test couldn't fail, because every path in `Run` returns 0.
   - The README, `atlas/architecture.md` and `atlas/couch.md` all match the code, including the claim that both entries are gated.

2. **Critical:** none.

3. **Important**
   - **ARCH-DRY:** `wrap.go:754` and `slugcmd.go:119` contain the same check, `launcher.CouchHostedEnv(os.Getenv) && os.Getenv("PAIR_SLUG") != "on"`, along with the knob name and its `"on"` value. Those are the only two copies in this window. Fix: add one helper such as `launcher.SlugEnabled(getenv func(string) string) bool` (both packages already import `launcher`), call it from both places, and test the truth table once, including the fail-closed case where a value like `ON` or `1` keeps it off.
   - **Done-when is stale** (see above). Fix: change clause 1 to name both gates, and reword clause 2 to "no slug model calls (`claude -p`) and no wrapper-spawned `pair slug`".
   - **Unverifiable:** Done-when clause 2, the live fleet check, hasn't been done, and the Log says so. A review checkout can't restart Couch, relaunch slots or inspect the live process table.

4. **Minor**
   - No test covers hosting through `COUCH_THREAD_TAG` alone. `CouchHostedEnv` treats scope or tag as hosted, and every test case only sets the scope.
   - Nothing pins that `PAIR_SLUG` values other than exactly `on` keep the slug off. The shared helper from the DRY fix would be the place to test this.

5. **Test coverage:** Each gate has a regression test that I confirmed fails without its fix. Done-when clause 1 is tested in all three states: hosted, hosted with `PAIR_SLUG=on`, and not hosted. The slugcmd test checks the `on` case by seeing the next guard (`no PAIR_TAG`) fire, so it confirms the gate lets the run through without making a model call. I only ran the targeted tests; for the full suite I'm relying on the Log's claim that it passed twice.

6. **Architecture**
   - ARCH-DRY: **flag** (see Important).
   - ARCH-PURE: **pass.** Both gates are thin env checks at the edge of the code. The wrapper test uses the existing `spawnSlug` seam, and nothing new mixes logic with IO.
   - ARCH-PURPOSE: **pass.** The hub's fold-in covered the second entry point, the Stop hook, in the same round, and I found no other slug entry points.

7. **Plan revisions:** Add a `## Revisions` entry that updates Done-when to cover both gates, as worded above, and that records the live fleet check as still pending a Couch restart and slot relaunch.

```findings
findings:
  - id: new
    severity: Important
    kind: defect
    family: single-source-policy-predicate
    title: |
      The slug gate rule is written out twice: wrap.go:754 and slugcmd.go:119
    detail: |
      ARCH-DRY. Both sites write out the same rule, CouchHostedEnv && PAIR_SLUG != "on". Those are the only two copies in this window. Move it into one helper, e.g. launcher.SlugEnabled(getenv), call it from both, and test the truth table once, including that values like ON or 1 keep it off (fail closed).
  - id: new
    severity: Important
    kind: defect
    family: done-when-tracks-revisions
    title: |
      Done when does not cover the slugcmd.Run Stop-hook gate the Revisions added
    detail: |
      Clause 1 only describes the wrapper's spawn. Clause 2, read literally ("no pair slug processes"), is wrong: under Couch the Stop hook still starts a short-lived pair-slug that stops at the gate. Change clause 1 to name both gates, and make clause 2 about model calls and wrapper spawns.
  - id: new
    severity: Important
    kind: unverifiable
    family: live-fleet-acceptance
    title: |
      Done-when clause 2 (live check across the fleet after restart) is not verified
    detail: |
      Attempted: there is nothing to run from here. The review checkout cannot restart Couch, relaunch slots or inspect the live process table, and the issue Log says the check is pending a hub restart. Evidence that would answer it: ps output from the hosted slots over normal turns after relaunch, showing no wrapper-spawned pair slug and no claude -p slug calls.
  - id: new
    severity: Minor
    kind: defect
    family: gate-truth-table-coverage
    title: |
      No test covers COUCH_THREAD_TAG-only hosting, or PAIR_SLUG values other than exactly on
    detail: |
      Both new tests only set COUCH_THREAD_SCOPE, and none tries a PAIR_SLUG value like ON or 1. Cover both when the shared helper is extracted.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"e652dfd7-4c3f-4bd6-98d6-8e457d5a2ab1","repository":"github.com/xianxu/pair","issue_id":"000650","recipe":"small-diff-review","round":1,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/b40f68b20b95d769","reviewed_head":"1ecb83d29deed3d4f82f8ea0a82dffdeb5e2f35f","slot":"pair:8","requested_at":"2026-10-17T22:04:18.647166Z"},"phase":"finished","pid":8520,"started_at":"2026-10-17T22:04:18.888098Z","finished_at":"2026-10-17T22:05:52.354808Z","session_id":"3720a09c-a1ed-409e-af5b-4c10095c5f67","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.247135459s
head: "1ecb83d29deed3d4f82f8ea0a82dffdeb5e2f35f"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-17T15:53:10-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 650 — Slug line off by default under Couch (PAIR_SLUG=on to re-enable): no per-turn pair slug spawn or Haiku call |
| repo | pair |
| issue file | workshop/issues/000650-slug-line-off-by-default-under-couch-pair-slug-on-to-re-enable-no-per-turn-pair-slug-spawn-or-haiku-call.md |
| boundary | whole-issue close |
| milestone | — |
| window | c0841e1082d39731781d8a2972eeeaed41d07ade..7b72b05c05b6484fd33174ba17f011e2bd8a80ec |
| command | sdlc close --issue 650 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-17T15:53:10-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

All four round-1 findings were resolved in code, and I checked each against the diff and the tests, not the commit messages. The one remaining item is the live fleet check (Done-when clause 2). It can only be done after the hub restarts Couch and relaunches the slots, and this review checkout can do neither, so BR-3 stays open as unverifiable until someone records process-table evidence. The code side is ready. I found nothing new to raise. Tests run in this checkout all passed: the launcher, slugcmd and dispatcher packages, the wrapcmd slug tests, and `go vet` on the three changed packages.

1. **Strengths**
   - `launcher.SlugEnabled` (`cmd/internal/launcher/run.go:38`) is now the only place the rule lives. It reuses `CouchHostedEnv` instead of reading the env vars again. A grep shows no other `PAIR_SLUG == "on"` check anywhere in `cmd/`. Both callers use it: `wrap.go:755` and `slugcmd.go:119`.
   - `TestSlugEnabledTruthTable` (`couchhosted_test.go:23`) tests the rule as a pure function through an injected `getenv`. It covers hosting by scope only, tag only and both, and confirms that "ON", "1", "0" and "off" all keep the slug off.
   - `TestRunSkipsUnderCouchHosting` checks the `PAIR_SLUG_LOG` line to show which guard fired, so the Couch gate can't be confused with the later no-session skip. Its hosted case sets `PAIR_TAG=T` on purpose, so it would fail if the gate were removed.
   - In `wrap.go`, the gate runs before the debounce bookkeeping, and the `SLUG-skip` debug line means a hosted session's quiet slug can be explained.
   - The README, `atlas/architecture.md` and `atlas/couch.md` all describe both gates, the `PAIR_SLUG=on` knob, and that nothing changes outside Couch. The wording matches the code.

2. **Critical findings:** none.

3. **Important findings:** none new. BR-3 is still open (see dispositions).

4. **Minor findings:** none.

5. **Test coverage notes:** The Done-when clause 1 matrix is covered by tests for both entry points: Couch-hosted with the knob unset (off), Couch-hosted with `PAIR_SLUG=on` (on), and outside Couch (on). The two per-entry tests only use `COUCH_THREAD_SCOPE`, but tag-only hosting and the odd knob values are now tested once in the shared helper's truth table. That is enough, since both call sites go through that helper.

6. **Architecture:**
   - **ARCH-DRY: pass.** There is one predicate with two callers and no copies.
   - **ARCH-PURE: pass.** `SlugEnabled` is pure, with `getenv` injected; the call sites only pass in `os.Getenv`.
   - **ARCH-PURPOSE: pass in code.** Both ways in are gated: the wrapper's turn-end spawn and the Stop-hook path through `bin/pair-slug`. Clause 2's live acceptance check is the purpose's own measurement, not a separable follow-up, so it stays tracked as BR-3 rather than being called done.

7. **Plan revision recommendations:** none. The Revisions section already records the round-1 fixes and that BR-3 is pending.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      launcher.SlugEnabled (run.go:38) is now the one copy of the rule; wrap.go:755 and slugcmd.go:119 both call it, no other copies under cmd/ (grep); TestSlugEnabledTruthTable passes.
  - id: BR-2
    disposition: addressed
    note: |
      Done-when clause 1 now names both gates (wrapper spawn + slugcmd.Run Stop-hook body); clause 2 is now about wrapper spawns and claude -p calls, not every pair-slug process. Matches the code.
  - id: BR-3
    disposition: not-addressed
    note: |
      No host evidence yet: no dispute filed, no ps output in the Log. The Log and Revisions still say it is pending the hub's Couch restart and slot relaunch. Needed: process-table output from hosted slots over normal turns after relaunch, showing no wrapper-spawned pair slug and no slug claude -p.
  - id: BR-4
    disposition: addressed
    note: |
      TestSlugEnabledTruthTable covers tag-only hosting ({"", "t", ""} -> false, {"", "t", "on"} -> true) and the ON/1/0/off fail-closed values; both call sites go through this helper.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"499163f6-78f7-41c8-8cb7-f720a540ef5b","repository":"github.com/xianxu/pair","issue_id":"000650","recipe":"small-diff-review","round":2,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/9d8ab46598aa95ec","reviewed_head":"7b72b05c05b6484fd33174ba17f011e2bd8a80ec","slot":"pair:8","requested_at":"2026-10-17T22:51:35.667177Z"},"phase":"finished","pid":53269,"started_at":"2026-10-17T22:51:35.927526Z","finished_at":"2026-10-17T22:53:09.685656Z","session_id":"6d50ebd9-077f-455e-afe6-a71dc8c5cdc5","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.700343417s
head: "7b72b05c05b6484fd33174ba17f011e2bd8a80ec"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-17T15:57:15-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 650 — Slug line off by default under Couch (PAIR_SLUG=on to re-enable): no per-turn pair slug spawn or Haiku call |
| repo | pair |
| issue file | workshop/issues/000650-slug-line-off-by-default-under-couch-pair-slug-on-to-re-enable-no-per-turn-pair-slug-spawn-or-haiku-call.md |
| boundary | whole-issue close |
| milestone | — |
| window | c0841e1082d39731781d8a2972eeeaed41d07ade..7b72b05c05b6484fd33174ba17f011e2bd8a80ec |
| command | sdlc close --issue 650 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-17T15:57:15-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The code is done and correct. Both slug entry points check the same rule, `launcher.SlugEnabled`: pair-wrap's turn-end spawn (`maybeSpawnSlug`) and the shared `slugcmd.Run` body, which `pair slug` and the `bin/pair-slug` Claude Stop-hook symlink both reach. I grepped every non-test caller under `cmd/`, and the only invocation of `slugcmd.Run` is `dispatcher.go:282`. `wrap.go:781`'s spawn sits behind the gate, so no path skips the check. The targeted tests pass at the pinned head (launcher, slugcmd, wrapcmd `MaybeSpawnSlug|SlugSpawn`, dispatcher `Slug`). The Done-when clause 1 test covers all three states for both entry points: hosted means off, hosted plus `PAIR_SLUG=on` means on, and outside Couch it runs as before. The one thing between this and SHIP is BR-3, the live fleet check in Done-when clause 2. That evidence can only exist after landing and a restart. No dispute with measured evidence has been filed, so it stays open.

1. **Strengths**
   - `cmd/internal/launcher/run.go:38`: `SlugEnabled` is the single owner of the predicate, the knob name and the exact `on` value. It reuses `CouchHostedEnv` and doesn't parse the env again (ARCH-DRY: pass).
   - `cmd/internal/launcher/couchhosted_test.go:23`: the truth table runs without IO through an injected `getenv`. It covers hosting by scope or by tag, and the fail-closed values `ON`, `1`, `0` and `off` (ARCH-PURE: pass).
   - `cmd/internal/slugcmd/slugcmd_test.go:21`: each case checks the log line, so it pins which guard fired. A test that only checked the exit code would pass whichever guard stopped it.
   - `wrap.go:755`: the gate returns before the debounce bookkeeping. When hosted, `lastSlug` doesn't change, so turning `PAIR_SLUG=on` on later starts with a clean debounce.
   - The tests pin `COUCH_THREAD_SCOPE`, `COUCH_THREAD_TAG` and `PAIR_SLUG` in every case, including the dispatcher test, so a hosted test run can't leak its env into the results.
2. **Critical findings:** none.
3. **Important findings:** no new ones. BR-3 is still open (see dispositions).
4. **Minor findings:** none.
5. **Test coverage notes:** Done-when clause 1 is covered in all three states at both entry points. The truth table covers the edge values of the predicate. I didn't rerun the full suite; the Log records it green on 7b72b05c0.
6. **Architectural notes**
   - ARCH-DRY: pass. The rule lives in one place, and both call sites consult it.
   - ARCH-PURE: pass. The predicate is pure through the injected `getenv`; the call sites only pass `os.Getenv`.
   - ARCH-PURPOSE: pass. Every entry point to the slug body is gated; the only thing outstanding is the live acceptance check.
7. **Plan revision recommendations:** none. The Revisions section already records BR-3 as pending on the restart and relaunch.

```findings
dispose:
  - id: BR-3
    disposition: not-addressed
    note: |
      Still unverifiable from a review checkout and no measured-evidence dispute filed; needs post-relaunch ps output across hosted slots showing no wrapper-spawned pair slug and no slug claude -p call. Code-side gating at both entries is verified (grep: dispatcher.go:282 is the only slugcmd.Run caller; wrap.go:781 spawn sits behind the gate).
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"e133411a-bb88-49f9-8c0b-d5454504cb38","repository":"github.com/xianxu/pair","issue_id":"000650","recipe":"small-diff-review","round":3,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/489fb2a9ba852070","reviewed_head":"7b72b05c05b6484fd33174ba17f011e2bd8a80ec","slot":"pair:8","requested_at":"2026-10-17T22:56:27.801713Z"},"phase":"finished","pid":97622,"started_at":"2026-10-17T22:56:27.971531Z","finished_at":"2026-10-17T22:57:15.12241Z","session_id":"cffdc757-8673-48f8-8250-8650eba33eff","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.635458125s
head: "7b72b05c05b6484fd33174ba17f011e2bd8a80ec"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
