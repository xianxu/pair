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
