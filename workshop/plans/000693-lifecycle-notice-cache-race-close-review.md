# Boundary Review — pair#693 (whole-issue close)

| field | value |
|-------|-------|
| issue | 693 — flake: lifecycle primary-slot notice can race verified workspace publication |
| repo | pair |
| issue file | workshop/issues/000693-lifecycle-notice-cache-race.md |
| boundary | whole-issue close |
| milestone | — |
| window | f7be53a7fc8f3749d369eb821fbb21579b433712..fdac39f3a0dfc74ccc828ba25211e04faa2a4e0a |
| command | sdlc close --issue 693 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-19T04:26:31-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Reviewing the diff now; checking whether retained ordinary facts change the restart-drain diagnostic, then running tests.

Now reverting the production fix to confirm the regression goes red.

Regression goes red without the fix. Next I'll check concretely whether a non-slot ordinary thread now triggers the restart-drain diagnostic.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The diagnosis holds and the fix lands at the right seam. The publisher's ordinary-target drop in `observeFactLocked` was the actual production bug, not just a gap in the fixture. I confirmed it by reverting the fix: `TestLifecyclePublisherOrdinaryFactWaitsForItsWorkspace` then fails with the new diagnostic (`unresolved=[] identities=[/repo1 /repo2] … workspaces=[pair:1=/repo1 pair:2=/repo2]`), and it passes again with the fix restored. `go test -run TestLifecycle -count=3` passes at head.

One change in behaviour wasn't accounted for. Ordinary facts now go into `unresolved`, so a plain non-slot thread recovered during a restart is counted by the restart-drain diagnostic. That diagnostic is a user-visible Console status notice. A probe test confirmed that head emits `Couch lifecycle: 1 observations lacked verified slot metadata at restart drain` for one ordinary non-slot `recovered` fact, while base emits nothing. This is the only thing between this boundary and SHIP.

**1. Strengths**
- The root cause is correct. Reordering `connected` in the message loop would only have closed the window in the test. The fix in `lifecycle_notices.go:179-186` closes the seconds-long gap in production.
- The regression test forces the widest window: the fact is observed before the wrapper connects at all. It needs no lucky timing, no sleeps and no retries (`lifecycle_notices_test.go:605-620`).
- `retainUnresolvedLocked` (`lifecycle_notices.go:217-241`) pulls the inline bound out into a helper. Its unit test runs without any IO, and evicting the oldest entry is sound because `Observation` is a single Console-wide sequence (`couchtty/console_lifecycle.go:41`).
- `waitWorkspace` (`message_service_test.go:299-310`) waits on exactly what `refresh` reads (`s.workspaces`), as the Done-when requires.
- The diagnostic on timeout is capped at 4096 characters and names every field you need to place the fact.

**2. Critical findings**
None.

**3. Important findings**
- **Restart-drain diagnostic now fires for ordinary non-slot threads** (`lifecycle_notices.go:153-158`).
  - **Before:** ordinary facts never reached `unresolved`, so the drain count only covered untargeted facts.
  - **Now:** every ordinary (non-slot) thread recovered during a restart pass sits unresolved until drain. Each one is then reported to the user as missing slot metadata, which is noise on most restarts that host plain threads.
  - **Fix sketch:** at drain, count only unresolved entries with `Target.Kind == ""` and clear ordinary entries silently. Alternatively, record the target kind and word the diagnostic to match.
  - **Test to add:** an ordinary non-slot fact between `restart-armed` and `restart-drained` produces no "lacked" notice, and an untargeted one still does.
  - **Same family in this window:** the drain branch is the only consumer of `unresolved` that reports to the user. `reducePendingLocked`, `ownershipChanged` and eviction stay silent. The atlas sentence "cleared at restart drain" (`atlas/couch.md:883-888`) should also say what drain reports.

**4. Minor findings**
- Only the two new or changed call sites pass the publisher to `lifecycleNextMessage`. The other roughly ten call sites still time out with the bare "no lifecycle message".
- `unresolved` has no age bound, so an ordinary fact can publish arbitrarily late if its wrapper verifies hours afterwards. Newest-per-path softens this.
- The doc comment for `retainUnresolvedLocked` sits directly after the closing brace of `observeFactLocked`, with no blank line (`lifecycle_notices.go:216-217`).
- `retainUnresolvedLocked` uses `""` as the "nothing chosen yet" marker for `oldest`. That is safe only because `observeFactLocked` rejects empty paths before calling it.

**5. Test coverage notes**
Both orders in the Done-when are covered: fact before identity is the new test, and identity before fact is the primary-slot test using `waitWorkspace`. The cap test covers newest-wins, eviction at 128, and a new path being admitted at the cap. Neither the existing tests nor the new ones cover drain with retained ordinary facts. I did not repeat the `-count=50` run under CPU load.

**6. Architectural notes**
- **ARCH-DRY: pass.** The inline bound became one helper and nothing is duplicated.
- **ARCH-PURE: pass.** The bound logic is tested directly with no IO, and the diagnostic helper exists only in tests.
- **ARCH-PURPOSE: flagged, see the Important finding.** The class "a fact that arrives before its identity" is swept correctly. The one consumer of the widened set that reports to the user (the drain diagnostic) wasn't re-checked.
- **ARCH-EVOLVE: pass.** The change touches no persisted format.

**7. Plan revision recommendations**
- Add to the Fix list how drain treats retained ordinary facts: whether they count toward the restart-drain diagnostic, and why.

```findings
findings:
  - id: new
    severity: Important
    kind: defect
    family: widened-state-consumer-unswept
    title: |
      Restart-drain diagnostic now reports ordinary non-slot threads as lacking slot metadata
    detail: |
      Ordinary facts lacking identity now enter p.unresolved, and the restart-drained branch (lifecycle_notices.go:153-158) counts every unresolved entry into a user-visible Console status notice. Probe confirmed: one ordinary-target recovered fact for a non-slot path between restart-armed and restart-drained emits "Couch lifecycle: 1 observations lacked verified slot metadata at restart drain" at head and nothing at base. Fix: count only untargeted (Target.Kind empty) entries at drain and clear ordinary ones silently, or word the diagnostic by target kind; add a test that an ordinary non-slot fact produces no drain diagnostic and an untargeted one still does. Same family in this window: the drain branch is the only consumer of unresolved that reports to the user; also update atlas/couch.md 883-888 to state what drain reports.
  - id: new
    severity: Minor
    kind: defect
    family: bounded-wait-diagnostic-coverage
    title: |
      Only two lifecycleNextMessage call sites pass the publisher for the timeout diagnostic
    detail: |
      The variadic publisher argument is passed only in the primary-slot and ordinary-wait tests; the other roughly ten call sites still fail with the bare "no lifecycle message".
  - id: new
    severity: Minor
    kind: defect
    family: unbounded-age-retention
    title: |
      Unresolved ordinary facts have no age bound and can publish arbitrarily late
    detail: |
      An ordinary fact waits until drain, ownership change or eviction; if its wrapper verifies hours later, a stale notice is published. Newest-per-path softens this.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"c6955202-c283-4ada-97c4-1ea80abc866f","repository":"github.com/xianxu/pair","issue_id":"000693","recipe":"small-diff-review","round":1,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/c2c736c6873fb820","reviewed_head":"fdac39f3a0dfc74ccc828ba25211e04faa2a4e0a","slot":"pair:2","requested_at":"2026-10-19T11:20:58.569114Z"},"phase":"finished","pid":80464,"started_at":"2026-10-19T11:20:58.785992Z","finished_at":"2026-10-19T11:26:30.180634Z","session_id":"1e081480-d245-4832-a613-87f512cdcad2","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.570461375s
head: "fdac39f3a0dfc74ccc828ba25211e04faa2a4e0a"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-19T04:31:30-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 693 — flake: lifecycle primary-slot notice can race verified workspace publication |
| repo | pair |
| issue file | workshop/issues/000693-lifecycle-notice-cache-race.md |
| boundary | whole-issue close |
| milestone | — |
| window | f7be53a7fc8f3749d369eb821fbb21579b433712..877401b241a651bd16c3dc6c14def7447e0917e5 |
| command | sdlc close --issue 693 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-19T04:31:30-07:00 |
| verdict | SHIP |

## Review

Prior findings are checked and the tests pass. Writing the verdict now.

```verdict
verdict: SHIP
confidence: high
```

All three earlier findings are fixed, and each fix has evidence I checked against the code. **BR-1:** at restart drain, only untargeted facts are counted (`cmd/internal/couchcmd/lifecycle_notices.go:170-180`). The set is cleared without condition, and the new test `TestLifecycleDrainReportsOnlyUntargetedFacts` expects exactly 1 with one ordinary fact and one untargeted fact. Without the fix the count would be 2. `atlas/couch.md` now says what drain reports. **BR-2:** the publisher argument to `lifecycleNextMessage` is now required instead of optional, so the compiler makes every call site pass it, including `native_exit_test.go:173`. **BR-3:** ordinary facts now expire after `lifecycleOrdinaryWait` (2 min). The check runs inside `reducePendingLocked`, which runs on the 1 s refresh ticker (`lifecycle_notices.go:314`), so a stale ordinary notice can be at most about 2 min + 1 s old. `TestLifecycleOrdinaryFactExpires` checks both sides of the limit: a fact exactly at the limit survives, and one second past it is evicted. With `TMPDIR` set to a short path, `go vet` is clean and `go test ./cmd/internal/couchcmd -run 'Lifecycle|NativeExit' -count=3` passes. With the default `TMPDIR`, the run fails on a socket-path length limit that comes from this environment, not the code. The only thing left is a one-line stale comment.

1. **Strengths**
   - The expiry clock comes from the broker clock (`s.authority.now`) and is injected, so the expiry test is pure and needs no sleeps (`lifecycle_notices_test.go:695-715`).
   - The expiry test checks both sides of the boundary (exactly at the limit, then one second past it). It also checks that untargeted facts are not expired.
   - Making the publisher argument required turns BR-2 from a sweep someone has to remember into a compile-time guarantee. That fixes the whole `bounded-wait-diagnostic-coverage` family, not one call site.
   - In `reducePendingLocked`, resolution is checked before expiry. A fact whose identity arrives exactly at the limit still gets published, not dropped.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - `lifecycle_notices.go:241-244`: the doc comment on `retainUnresolvedLocked` still says "Drain and ownership change clear the set". It leaves out the new ordinary-fact expiry, which is the third way entries leave the set. The atlas text at `couch.md:888` already mentions it. This is the only stale comment in the window.

5. **Test coverage**
   - Both new behaviours have tests that would fail without the fix. I worked out the drain count (it would be 2) and the expiry survival from the code; I did not revert and re-run.
   - The drain test covers both target kinds, ordinary and untargeted.
   - Done-when clauses:
     - Fact-before-identity and identity-before-fact are covered by the round-1 tests, which still pass.
     - The diagnostic on timeout is now on every wait.
     - The readiness wait on the cached workspace is unchanged.

6. **Architecture notes**
   - **ARCH-DRY: pass.** `clock()` is a small fallback, and it is the only one in this window.
   - **ARCH-PURE: pass.** Expiry and drain counting stay in-memory reductions under the lock, and their tests need no IO.
   - **ARCH-PURPOSE: pass.** The whole `widened-state-consumer-unswept` family is now covered: all three consumers of `unresolved` (drain, `reducePendingLocked`, the test state dump) handle the new value type and target kinds.
   - **ARCH-EVOLVE: pass.** No persisted format changes; `unresolved` lives only in memory.
   - For future work: untargeted facts outside a restart cycle still wait until a drain or an ownership change, with no age limit. That behaviour predates this diff and is tied to what drain means, so it is not a finding here. If the `unbounded-age-retention` family comes back, the rule to apply is: every entry in `unresolved` either has an age limit or a guaranteed future clearing event.

7. **Plan revisions:** none. The round-1 log and the atlas match the code.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Drain counts only Target.Kind=="" and clears all (lifecycle_notices.go:170-180); TestLifecycleDrainReportsOnlyUntargetedFacts expects 1 (would be 2 without the fix); atlas/couch.md updated.
  - id: BR-2
    disposition: addressed
    note: |
      The publisher parameter is now required, so the compiler forces every call site, including native_exit_test.go:173, to pass it; the whole family is closed.
  - id: BR-3
    disposition: addressed
    note: |
      Ordinary facts expire after lifecycleOrdinaryWait, checked on the 1s refresh ticker; TestLifecycleOrdinaryFactExpires checks exactly-at-limit and one second past it.
findings:
  - id: new
    severity: Minor
    kind: defect
    family: widened-state-consumer-unswept
    title: |
      retainUnresolvedLocked doc comment omits the new ordinary-fact expiry
    detail: |
      lifecycle_notices.go:241-244 says drain and ownership change clear the set; reducePendingLocked now also evicts ordinary facts after lifecycleOrdinaryWait. This is the only stale comment in the window; atlas/couch.md already states the expiry.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"89c16974-7097-45c9-ba89-c85581467dd6","repository":"github.com/xianxu/pair","issue_id":"000693","recipe":"small-diff-review","round":2,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/eb8257215319e8a8","reviewed_head":"877401b241a651bd16c3dc6c14def7447e0917e5","slot":"pair:2","requested_at":"2026-10-19T11:30:32.356393Z"},"phase":"finished","pid":25141,"started_at":"2026-10-19T11:30:32.454742Z","finished_at":"2026-10-19T11:31:29.478162Z","session_id":"87b99c43-0b7a-461a-829f-2db6ec5a1505","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.468136542s
head: "877401b241a651bd16c3dc6c14def7447e0917e5"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
