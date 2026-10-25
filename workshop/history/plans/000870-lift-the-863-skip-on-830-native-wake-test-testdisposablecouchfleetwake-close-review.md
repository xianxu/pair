# Boundary Review — pair#870 (whole-issue close)

| field | value |
|-------|-------|
| issue | 870 — Lift the #863 skip on #830 native wake test (TestDisposableCouchFleetWake) |
| repo | pair |
| issue file | workshop/issues/000870-lift-the-863-skip-on-830-native-wake-test-testdisposablecouchfleetwake.md |
| boundary | whole-issue close |
| milestone | — |
| window | 63d35ac8295d6b4ff04001c683b038eca1d0e3d7..d1729d36e04386cee75eae1e42c66278d58421e7 |
| command | sdlc close --issue 870 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-25T01:20:55-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Still reviewing. Next I'm finding the testcouch fixture to check how `Workspaces.Workers` is indexed.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

**Summary.** The window touches two files: `cmd/couch/fleet_acceptance_test.go` and its affected-reader record. It removes the pair#863 skip and replaces the old switcher start-form gesture with `Fleet.Start`, which waits for hub 0 and one registration, followed by `WaitAgent(1)` and `WaitActors(2)`. It also fixes three test defects. The resume step now selects the `fixture` row instead of filtering for `fixture:0`. Each hub has one conversation turn before quit, so a resume from disk has something to restore. The asleep map is keyed by fixture slot, not generated tag. All of this matches the Spec and Plan. The affected-reader hash matches the head file (`9b3c3132…`), `go vet` is clean, and the test's gated skip path runs. One gap stops a clean SHIP. The Done-when clause "a repeat starting nothing" is only logged, never asserted, so a regression where the repeat `--wake` starts the plain worker would still pass. I couldn't reproduce the native Claude/Codex run in this sandbox, since it needs unsandboxed real agents. The 73.5 s / 74.4 s passes come from the implementer's Log and I haven't checked them.

1. **Strengths**
   - `fleet_acceptance_test.go:307-317`: the first phase now goes through `Fleet.Start`, which waits for hub 0 and one registration (`testcouch/fleet.go:206-213`). The test then explicitly waits for the second hub and for `WaitActors(2)`. This is the #863 behaviour the issue depends on, with no leftover switcher gesture. `wakeFleetStartHub` was deleted with no remaining callers.
   - `:341-351`: keying the asleep map by workspace slot and failing on an unknown `StartingPath` is stricter than the old tag keying. A stray thread now fails the test instead of quietly inflating the map. `Workspaces.Workers` is a `map[int]string` (`provision.go:21`), so `Workers[1]` and `Workers[2]` can't panic.
   - `:383-386`: the resume step waits for the selected row at the primary path and then for "Resume from disk" before typing. That removes the race in the old filter-then-Tab sequence.
   - `:326-327`, `:433-453`: sending one turn to each hub before quit has a real reason: an empty fixture transcript has nothing to resume from disk, so resume genuinely depends on it.

2. **Critical findings:** none.

3. **Important findings**
   - **`fleet_acceptance_test.go:403-408`: the Done-when clause "a repeat starting nothing" isn't asserted.** The repeat `couch --wake` only checks exit 0 and logs its stdout. If the repeat woke the plain worker or re-woke a hub, the test would still pass. **Fix:** after the repeat, check `asleep()` again (`!w["0"] && !w["1"] && w["2"]`), check that `fleet.Agent(2)` has no live PID, and/or check that stdout contains "no asleep hubs to wake". **Same family in this window:** the first `--wake` at `:390-394` also only logs its stdout, but the asleep map at `:399` pins its effect, so it's covered. The repeat is the only Done-when clause without an assertion.

4. **Minor findings**
   - **ARCH-DRY:** the accept-wait loop in `wakeFleetConverse` (`:439-450`) nearly duplicates loops elsewhere. Same file at `:113-127`, `startup_restart_fleet_test.go:197-211`, and `slash_keys_acceptance_test.go:172`. A shared `waitFleetAccepted(ctx, fleet, n, body)` helper would merge them. This is test-only code, so it isn't blocking.
   - `:442`: when `Accepted` returns a non-`ErrNotExist` error and the slice is empty, the callback returns `false, err`. That's correct, but the shape differs from the sibling at `:115-120`, which returns the error before looping.

5. **Test coverage notes.** Done-when clauses:
   - **Full quit leaves three threads asleep:** asserted at `:353-361`.
   - **Cold start brings nothing back:** asserted at `:368-376`.
   - **Main hub resumes from the switcher:** asserted at `:390-395`.
   - **`--wake` wakes exactly the other hub, plain worker stays asleep:** asserted at `:399-404`.
   - **Repeat starts nothing:** not asserted (see the Important finding).
   - **"Passes natively":** I couldn't check this here because it needs real Claude/Codex unsandboxed. The evidence is the implementer's Log only.

6. **Architectural notes**
   - **ARCH-DRY:** flagged (the Minor finding above).
   - **ARCH-PURE:** passes. This is an integration test going through the injected `testcouch.Fleet` seam, with no business logic.
   - **ARCH-PURPOSE:** passes, apart from the repeat-assertion gap above. The skip is gone, the run covers both agents, and the switcher remains the resume path.
   - **ARCH-EVOLVE:** not applicable. The only persisted-format change is a regenerated reader hash with an unchanged schema.

   The test is now the only native coverage of the T6 wake path, so its assertions should be as strong as its Done-when.

7. **Plan revision recommendations:** none needed. The Log already records the three test defects found during the native run.

```findings
findings:
  - id: new
    severity: Important
    kind: defect
    family: done-when-clause-unasserted
    title: |
      Repeat couch --wake ("a repeat starting nothing") is only logged, never asserted
    detail: |
      fleet_acceptance_test.go:403-408 checks only exit 0 and t.Logf's stdout, so a repeat that woke the plain worker or re-woke a hub would pass. After the repeat, assert that asleep() still shows slot 2 asleep and slots 0 and 1 awake, that fleet.Agent(2) has no live PID, and/or that stdout reports nothing to wake. Family sweep in this window: the first --wake (390-394) also only logs, but the asleep map at 399 pins its effect, so the repeat is the only unasserted clause.
  - id: new
    severity: Minor
    kind: defect
    family: duplicated-accept-wait-loop
    title: |
      wakeFleetConverse re-implements the Accepted-body wait loop already present in sibling tests (ARCH-DRY)
    detail: |
      Near-identical loops: fleet_acceptance_test.go:113-127, :439-450, startup_restart_fleet_test.go:197-211, slash_keys_acceptance_test.go:172. They could become one waitFleetAccepted(ctx, fleet, n, body) helper.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"9ef1b4d1-efbc-4e69-b54f-bb7303c315b9","repository":"github.com/xianxu/pair","issue_id":"000870","recipe":"small-diff-review","round":1,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/f120c53c1adaf278","reviewed_head":"d1729d36e04386cee75eae1e42c66278d58421e7","slot":"pair:6","requested_at":"2026-10-25T08:19:47.654465Z"},"phase":"finished","pid":63906,"started_at":"2026-10-25T08:19:47.753207Z","finished_at":"2026-10-25T08:20:53.708735Z","session_id":"6d0838ec-cfbc-4462-9ccd-f1cfb28ff3f8","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.16964675s
head: "d1729d36e04386cee75eae1e42c66278d58421e7"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000411

---

## Re-review — 2026-10-25T02:13:27-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 870 — Lift the #863 skip on #830 native wake test (TestDisposableCouchFleetWake) |
| repo | pair |
| issue file | workshop/issues/000870-lift-the-863-skip-on-830-native-wake-test-testdisposablecouchfleetwake.md |
| boundary | whole-issue close |
| milestone | — |
| window | 63d35ac8295d6b4ff04001c683b038eca1d0e3d7..5e8db46d9f0ccaeb784a1b9f19943d6b4a0006ad |
| command | sdlc close --issue 870 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-25T02:13:27-07:00 |
| verdict | SHIP |

## Review

**Verdict: SHIP** for pair#870. BR-1 is fixed and BR-2 stays open as a Minor; nothing blocks.

```verdict
verdict: SHIP
confidence: high
```

The skip is gone, and the first phase now uses `Fleet.Start` plus `WaitAgent(1)` and `WaitActors(2)`, as the Spec and Plan say. Every Done-when clause is asserted, including the one BR-1 flagged: the repeat `couch --wake` now has three checks at `cmd/couch/fleet_acceptance_test.go:417-425`:
- stdout contains "no asleep hubs to wake", which is the exact empty-result string from `wakeSummary` (`cmd/internal/couchcmd/wake.go:189`);
- the plain worker has no live PID;
- the asleep state is still slots 0 and 1 awake, slot 2 asleep.

`go vet ./cmd/couch/` is clean. I couldn't run the native Claude and Codex runs in this checkout; those passes come from the implementer's Log, not from me. The only thing still open is BR-2, a Minor test-helper duplication that doesn't block.

1. **Strengths**
   - The asleep map is keyed by slot (from the workspace) and fails on any unknown thread (`fleet_acceptance_test.go:341-348`). That removes the dependence on generated tags and catches stray threads.
   - The switcher gesture waits for the selected `▸ fixture  <primary>` row and for the "Resume from disk" action before typing (`:384-386`). That replaces a typed filter that matched nothing.
   - The unused `wakeFleetStartHub` was deleted rather than left behind.
   - Before parking, each hub the test later brings back accepts one turn (`:326-327`), since a hub with no conversation can't be resumed from disk.

2. **Critical:** none.

3. **Important:** none.

4. **Minor:** BR-2 is still open (details in the findings block).

5. **Test coverage**
   - The full quit is checked to leave 3 asleep threads.
   - The cold start is checked for no live agents.
   - The resume path and the first `--wake` are checked through `WaitAgent(1)`, the worker's PID and the asleep map.
   - The repeat `--wake` is now checked from stdout, the worker's PID and the asleep map.

6. **Architecture**
   - **ARCH-DRY:** flagged, as BR-2 only.
   - **ARCH-PURE:** pass. This is test glue with no business logic.
   - **ARCH-PURPOSE:** pass. All Done-when clauses are delivered.
   - **ARCH-EVOLVE:** pass. No persisted format changes; the affected-reader JSON only updates its hash.

7. **Plan revisions:** none. The Plan and Log match the code.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      fleet_acceptance_test.go:417-425 asserts the "no asleep hubs to wake" stdout (wake.go:189 empty-outcome string), plain worker not live, and asleep map unchanged (0,1 awake; 2 asleep).
  - id: BR-2
    disposition: not-addressed
    note: |
      Still Minor and non-blocking: wakeFleetConverse (fleet_acceptance_test.go:442-461) keeps its own Accepted-body wait loop; the implementer deferred the shared helper because it would touch sibling tests outside 870.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"69c609cf-0d2f-4300-8a95-9aa6ad2d310c","repository":"github.com/xianxu/pair","issue_id":"000870","recipe":"small-diff-review","round":2,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/6cecbadd474c6ad4","reviewed_head":"5e8db46d9f0ccaeb784a1b9f19943d6b4a0006ad","slot":"pair:6","requested_at":"2026-10-25T09:12:44.342449Z"},"phase":"finished","pid":93496,"started_at":"2026-10-25T09:12:44.545634Z","finished_at":"2026-10-25T09:13:25.874012Z","session_id":"e4eb641a-ba03-4dbb-af4b-9f57c1077858","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 12.11588525s
head: "5e8db46d9f0ccaeb784a1b9f19943d6b4a0006ad"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
