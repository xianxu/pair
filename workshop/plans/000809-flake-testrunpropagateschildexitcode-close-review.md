# Boundary Review — pair#809 (whole-issue close)

| field | value |
|-------|-------|
| issue | 809 — flake: TestRunPropagatesChildExitCode |
| repo | pair |
| issue file | workshop/issues/000809-flake-testrunpropagateschildexitcode.md |
| boundary | whole-issue close |
| milestone | — |
| window | 7b9440e8c92663f8ee189f54a1260e3043a78f07..15c56a7a55ebe7363d566a640d2e94be3471c41b |
| command | sdlc close --issue 809 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-22T14:16:38-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

I've checked the diff and the issue's spec. Now I'm checking whether every test fixture in scribecmd allocates its pty through `openPTY`, as the atlas says.

The new tests pass. Next I'll temporarily swap the guard for a plain `errors.Is` check to confirm the test goes red without the sign fix.

Mutating the guard turns the test red, and the checkout is restored. One last look at the other injected opener near line 365.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The fix is correct and well evidenced. `openPTYRetry` (`cmd/internal/scribecmd/pty_darwin.go:37-49`) bounds the darwin pty allocation at 3 attempts, retries ENXIO only, and returns the original error unchanged when the bound runs out. `isTransientPTYAlloc` handles ENXIO with either sign, positive or negated. I checked the regression test by replacing the guard with `errors.Is(err, syscall.ENXIO)`: `TestDarwinStartRidesOutTransientAllocationENXIO` then fails ("open called 1 times, want 3 … errno -6"). With the code restored it passes, and `GOOS=linux go vet` is clean. The oracle fix in `runWithPTYReply` does what Spec item 3 asks for. Nothing blocks the gate. Two cheap Important items remain. First, a doc claim is false: one fixture in the same package still calls `pty.Open` directly. Second, the literal wording of `## Done when` contradicts the retry loop this change delivers.

Done-when: stale — it still says "never a retry loop". The Spec's hub ruling allows a bounded production-side retry on darwin, but `## Done when` was never reworded to match. The clause as written is contradicted by the code it gates.

**1. Strengths**
- The sign normalisation (`pty_darwin.go:60-70`) is pinned by tests for both spellings, and the mutation check proves the test catches the guard bug written the obvious way.
- The table test covers every boundary: recovery on attempt 2, recovery on the last attempt (`ptyAllocAttempts-1` failures), exhaustion (`ptyAllocAttempts` failures, original error returned), and an unrelated errno that is not retried (`pty_darwin_test.go:413-419`).
- `TestRunPropagatesChildExitCode` checks for an abort first, as its own assertion, before checking the exit code (`scribecmd_test.go:288-293`). So "scribe aborted" can no longer be misreported as "child's code mis-propagated".
- `TestRunFatalSitesExitOneWithDiagnostic` reuses the existing `run(..., start)` seam (ARCH-DRY). It pins log-open, pty-start and `cmd.Wait`, and it names where the other sites are pinned.
- ARCH-DRY: one retry primitive feeds both the production path and the fixture, and `pty_other.go` stays a plain pass-through.

**2. Critical findings**
None.

**3. Important findings**
- **The claim "test fixtures allocate through `openPTY`" is false within this package** (`pty_darwin_test.go:625`). `TestDarwinRunInteractiveResizeAndRestore` still calls `pty.Open()` directly for its outer terminal, so it is exposed to the same race. The code comment at `pty_darwin.go:32-34` and the atlas paragraph (`atlas/architecture.md:1354-1356`) both state the rule as if it already holds. Instances of this class in the window:
  - (a) `:625`: a direct fixture `pty.Open`. Fix: call `openPTY()`.
  - (b) `:427`: the new test's injected opener calls raw `pty.Open()` inside the counted closure. A real ENXIO there is retried, but it shifts `opened` off by one. In the "recovers on the last attempt" case it uses up the bound, so the test itself can flake. Fix: call `openPTY()` there, so a real-race retry happens below the counter.
  - (c) `:367` is already protected by the wrapper and needs no change.
- **`## Done when` is stale.** The clause "never a retry loop" literally forbids what this diff builds. The reconciliation exists only in the Spec ruling. Reword it, for example: "root cause fixed: no test-side or blind retry and no timeout extension; a bounded darwin ENXIO-only allocation retry per the hub ruling; closing unregisters the tests."

**4. Minor findings**
- The atlas says the errno "arrives negated through the raw syscall path". The Spec says it was measured from `syscall.Open` as well. The code comment states both correctly, but the atlas wording narrows it.
- The same exposure exists in other packages: about 30 direct `pty.Open`/`pty.Start` fixtures, for example in `ttyio`, `couchcmd`, `wrapcmd` and `hostty`. This is out of scope here because `openPTY` is unexported, but see the architectural notes.

**5. Test coverage notes**
- I confirmed the regression test fails without the fix by mutating the guard: 3 of 5 subtests go red.
- Spec Required behavior 3 (the diagnostic survives on abort) is covered by the "pty start" subtest. Exhausting the bound returns the original errno; the test checks this with `errors.Is` on the identical value.
- I did not re-run the concurrent churn harness, which is host-load dependent. I am relying on the Log for that, together with the deterministic tests above.

**6. Architectural notes for upcoming work**
- ARCH-DRY: pass. One retry primitive with two callers, and no duplicated guard.
- ARCH-PURE: pass. `isTransientPTYAlloc` is pure, and `openPTYRetry` takes the allocator as an injected function. Both are tested without real failures.
- ARCH-PURPOSE: flag (the Important finding above). The fixture sweep stopped at the instance that was observed; in-package sibling (a) remains.
- ARCH-EVOLVE: pass, not applicable. No persisted format changed. The affected-readers JSON is a hash refresh under the same schema.
- Every pty-heavy package in the tree is exposed to the same XNU race. A shared exported helper, for example in `ptychild` or a new `ptyalloc` package, would be the follow-up that closes the class tree-wide. That needs its own issue.

**7. Plan revision recommendations**
- Add a `## Revisions` entry recording that the fixture sweep in item 1a also covers `pty_darwin_test.go:625` (and `:427`), once fixed.
- Reword `## Done when` to match the hub ruling, as suggested above.

```findings
findings:
  - id: new
    severity: Important
    kind: defect
    family: fixture-pty-alloc-bypasses-shared-retry
    title: |
      Fixtures in scribecmd still call pty.Open directly despite the doc claim that they use openPTY
    detail: |
      pty_darwin_test.go:625 (TestDarwinRunInteractiveResizeAndRestore) calls pty.Open() directly for its outer terminal, so it is exposed to the #809 race. pty_darwin.go:32-34 and atlas/architecture.md:1354-1356 state that fixtures allocate through openPTY. Second instance in the window: the new test's injected opener at pty_darwin_test.go:427 calls raw pty.Open inside the counted closure, so a real ENXIO shifts the opened count, and in the last-attempt case uses up the bound. Fix both to call openPTY(). Line 367 is already wrapped by openPTYRetry and needs no change.
  - id: new
    severity: Important
    kind: defect
    family: done-when-tracks-contract
    title: |
      Done-when still says never a retry loop, but the change ships a bounded retry
    detail: |
      The Spec records the hub ruling that allows a bounded, ENXIO-only, darwin-only production retry, but Done-when was not updated and its literal text contradicts the delivered openPTYRetry loop. Reword it to ban test-side or blind retries and timeout extensions, and allow the ruled bounded allocation retry.
  - id: new
    severity: Minor
    kind: defect
    family: doc-claim-narrower-than-measurement
    title: |
      Atlas says the negated errno comes from the raw syscall path only
    detail: |
      The Spec and the code comment say the negated errno was measured from syscall.Open as well as from the raw ioctl syscalls. The atlas sentence narrows this to the raw syscall path only.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"2ec51ec3-8281-479f-b400-c95a3f6eb527","repository":"github.com/xianxu/pair","issue_id":"000809","recipe":"small-diff-review","round":1,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/6891b03376a948f0","reviewed_head":"15c56a7a55ebe7363d566a640d2e94be3471c41b","slot":"pair:8","requested_at":"2026-10-22T21:15:09.579029Z"},"phase":"finished","pid":3367,"started_at":"2026-10-22T21:15:09.864261Z","finished_at":"2026-10-22T21:16:36.58377Z","session_id":"22e8b16d-1209-4591-90ca-83e36c1aef15","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.708102042s
head: "15c56a7a55ebe7363d566a640d2e94be3471c41b"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-22T14:23:31-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 809 — flake: TestRunPropagatesChildExitCode |
| repo | pair |
| issue file | workshop/issues/000809-flake-testrunpropagateschildexitcode.md |
| boundary | whole-issue close |
| milestone | — |
| window | 7b9440e8c92663f8ee189f54a1260e3043a78f07..b54e8064574d5962a25811704f3ad4998078a30a |
| command | sdlc close --issue 809 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-22T14:23:31-07:00 |
| verdict | SHIP |

## Review

Running the tests passed on the pinned head. Next I'm reverting the sign normalisation to check that the ENXIO regression test goes red without it.

```verdict
verdict: SHIP
confidence: high
```

This change fixes the root cause of the flake. XNU's ptmx allocator races and intermittently returns ENXIO; the fix retries the allocation at most three times, on ENXIO only and on darwin only, as the hub approved. The guard handles the negated errno that was actually measured. I confirmed the regression test proves it: with the sign normalisation in `isTransientPTYAlloc` removed, `TestDarwinStartRidesOutTransientAllocationENXIO` fails in three cases (raw spelling, recovers on the last attempt, exhausts the bound). Restored, it passes.

On the pinned head, `go test ./cmd/internal/scribecmd -count=1` passes (37.7s) and `GOOS=linux go vet` is clean. The two blocking findings from round 1 are resolved. In scribecmd, the only raw `pty.Open` left is the documented exception at line 367, and `## Done when` now matches the hub ruling.

Round 1's BR-3 was based on a wrong reading. The Spec never measured a negated errno coming from the ioctls, and its fix copied that overclaim into the atlas. I've withdrawn BR-3 and raised the full set of affected comments as one Minor finding. It doesn't block.

1. **Strengths**
   - `pty_darwin.go:64-97`: the retry loop takes the opener as a parameter, so the test can inject failures with no real IO. `isTransientPTYAlloc` is a pure function. The bound of three attempts and the absence of backoff are both justified by the measurement table quoted in the code.
   - `pty_darwin_test.go:134-183`: the cases cover both sign conventions, recovery on the last attempt, running out of attempts, and an unrelated errno that must not be retried. These are exactly the edge cases that matter.
   - `scribecmd_test.go:303-308` fixes the test oracle. It checks Run's stderr is empty before comparing the exit code, so a scribe abort can no longer be reported as a wrong child exit code.
   - `TestRunFatalSitesExitOneWithDiagnostic` reuses the existing `start` seam to pin the log-open, pty-start and `cmd.Wait` sites. Its comment points to where the other sites are pinned.
   - `pty_other.go:12`: the non-darwin `openPTY` just calls `pty.Open`. That keeps the untagged test fixture compiling on Linux without adding a retry there.

2. **Critical findings:** none.

3. **Important findings:** none.

4. **Minor findings**
   - The negated errno's origin is misstated in several places. The Spec says every failure was at `open(/dev/ptmx)` and the ioctls never failed. The comments either say -6 was measured from both paths, or put it on the ioctls alone. The affected comments are listed in the findings block below.

5. **Test coverage**
   - The fix fails without its regression test, which I checked by reverting it.
   - Every Done-when clause that has two states is tested in both: positive and negated spelling, recovery and running out of attempts, retried and not retried.
   - The fixture sweep (BR-1) changes test code to avoid an OS race, so a deterministic regression test isn't possible. That's acceptable evidence for a fixture-only fix.

6. **Architectural notes**
   - **ARCH-DRY: pass.** All production and fixture allocations in the package go through one `openPTYRetry`.
   - **ARCH-PURE: pass.** The classifier is pure and the opener is injected.
   - **ARCH-PURPOSE: pass for this issue.** It fixes production and the registered test, not just the test.
     - Outside this issue: about 25 raw `pty.Open`/`pty.Start` fixtures in other packages are exposed to the same XNU race, e.g. `couchcmd/run_test.go`, `ttyio/writer_test.go`, `wrapcmd/retention_test.go`.
     - That fits the hub ruling (darwin `startPTY` only) and the package-private helper. Fixing those would mean a shared allocation helper and its own issue, which is a reasonable follow-up.
   - **ARCH-EVOLVE: pass.** No persisted format changes; the affected-readers JSON keeps its schema and only the sha changes.

7. **Plan revision recommendations**
   - In the Spec or `## Revisions`: if the comments are corrected, note that the -6 measurement is from `syscall.Open` on `/dev/ptmx` only. The ioctl path is covered by the sign normalisation as a precaution, not because it was observed failing.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      pty_darwin_test.go:630 and :432 now call openPTY; :367 is the documented exception inside the outer openPTYRetry; atlas states the rule. No raw pty.Open remains in scribecmd otherwise.
  - id: BR-2
    disposition: addressed
    note: |
      Done-when now bans test-side or blind retries and names the ruled production bounded retry, citing delivery 4b2538ba; a Revisions entry records it.
  - id: BR-3
    disposition: withdrawn
    note: |
      Mistaken premise: Spec lines on the step classifier say every failure was at the ptmx open and the ioctls never failed, so "measured from both" was itself an overclaim; superseded by the new finding covering every site.
findings:
  - id: new
    severity: Minor
    kind: defect
    family: doc-claim-narrower-than-measurement
    title: |
      Comments misattribute the measured negated errno to the ioctl raw-syscall path
    detail: |
      This is the 2nd finding in family doc-claim-narrower-than-measurement. The Spec measured -6 only at syscall.Open on /dev/ptmx, and TIOCPTYGNAME/GRANT/UNLK never failed. Instances: pty_darwin.go isTransientPTYAlloc comment "measured arriving from BOTH"; atlas/architecture.md:1351-1352 "measured from both ... and the raw ioctl syscalls"; pty_darwin_test.go rawENXIO comment, which says the ioctls are the production spelling's source; pty_darwin_test.go:128-131 "whichever of creack/pty's two syscall conventions happened to lose the race"; lessons.md "a failing raw syscall can hand back a negated errno". Rule: state where the errno came from once, in the isTransientPTYAlloc comment, matching the Spec (measured at the syscall.Open ptmx open; the ioctl path is covered as a precaution only). The other sites should point to that comment instead of restating it.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"61c316c8-c663-46a6-8f26-7379f9068e45","repository":"github.com/xianxu/pair","issue_id":"000809","recipe":"small-diff-review","round":2,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/81202c01342861a5","reviewed_head":"b54e8064574d5962a25811704f3ad4998078a30a","slot":"pair:8","requested_at":"2026-10-22T21:21:37.102485Z"},"phase":"finished","pid":27182,"started_at":"2026-10-22T21:21:37.26276Z","finished_at":"2026-10-22T21:23:29.298624Z","session_id":"8eeca7db-7271-4824-9bd5-133b269d9074","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.717260417s
head: "b54e8064574d5962a25811704f3ad4998078a30a"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-22T14:31:21-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 809 — flake: TestRunPropagatesChildExitCode |
| repo | pair |
| issue file | workshop/issues/000809-flake-testrunpropagateschildexitcode.md |
| boundary | whole-issue close |
| milestone | — |
| window | 7b9440e8c92663f8ee189f54a1260e3043a78f07..1266347a901798d554eb428c2b65801d190923fb |
| command | sdlc close --issue 809 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-22T14:31:21-07:00 |
| verdict | SHIP |

## Review

I'm finishing the review: tests pass and the last open finding (BR-4) checks out, so the verdict follows.

```verdict
verdict: SHIP
confidence: high
```

The only open finding from earlier rounds was BR-4, about doc claims naming the wrong measurement site. It is fixed at all five places it named. `pty_darwin.go:80-92` now gives the measurement once: the negated errno was seen at `syscall.Open` on `/dev/ptmx`, and the ioctls are normalised only as a precaution. That matches the Spec. The `rawENXIO` comment and the ENXIO test's doc comment now point back to that comment instead of repeating it. The atlas (`atlas/architecture.md:1351-1355`) and `workshop/lessons.md` still restate the measurement, but correctly now, with the precaution caveat. The Go-level behaviour is unchanged and I re-verified it here. `go vet` is clean on darwin and with `GOOS=linux`. The targeted tests pass: the ENXIO ride-out, the fatal-site diagnostics, `TestRunPropagatesChildExitCode` and the resize fixture. The only bare `pty.Open` left in the package is `pty_darwin_test.go:367`, the documented exception: it sits inside `startPTYWithOpen`'s own retry, so it is already bounded. I checked the comment's toolchain claim against Go 1.27.1: `syscall/asm_darwin_arm64.s` does issue raw `SVC $0x80`. Nothing new blocks shipping.

1. **Strengths**
   - `isTransientPTYAlloc` (`pty_darwin.go:93-103`) normalises the sign before comparing. It is pinned in both sign conventions, plus last-attempt recovery, running out of attempts, and an unrelated errno not being retried (`pty_darwin_test.go:137-186`).
   - The test fix addresses why the bug could not be diagnosed: `runWithPTYReply` now returns Run's stderr, and `TestRunPropagatesChildExitCode` checks that it is empty before checking the exit code (`scribecmd_test.go:305-314`).
   - There is one allocation primitive, `openPTY`/`openPTYRetry`, shared by production and fixtures. `pty_other.go` gets a two-line pass-through, so non-darwin behaviour is unchanged (ARCH-DRY).
   - `TestRunFatalSitesExitOneWithDiagnostic` pins the log-open, pty-start and `cmd.Wait` fatal sites through the existing `start` seam; no new mocks were needed.
2. **Critical:** none.
3. **Important:** none.
4. **Minor:** none new. The atlas and lessons.md restate the measurement rather than pointing to the code comment, but both are accurate now, so I don't count it as a defect.
5. **Test coverage:** every `## Done when` clause is covered. The retry is production-side only, there is no test-side retry, no timeout was extended and no assertion was weakened. Retry recovery and running out of attempts are both exercised.
6. **Architecture:**
   - **ARCH-DRY: pass.** One shared opener.
   - **ARCH-PURE: pass.** The classifier is pure, and the retry takes the opener as an injected function.
   - **ARCH-PURPOSE: pass.** All fixtures were swept, and production `startPTY` is covered.
   - **ARCH-EVOLVE: pass.** No persisted format changed. The affected-reader JSON only had its hashes refreshed.
7. **Plan revisions:** none.

```findings
dispose:
  - id: BR-4
    disposition: addressed
    note: |
      All five sites verified: pty_darwin.go:80-92 states the measurement once (syscall.Open on /dev/ptmx, ioctls as a precaution only); the rawENXIO comment and test doc point to it; the atlas and lessons.md restate it correctly with the precaution caveat. Comment-only change, nothing behavioural to regress.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"5402068b-f958-4519-afba-a2fea00404ba","repository":"github.com/xianxu/pair","issue_id":"000809","recipe":"small-diff-review","round":3,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/6998f4655fd79299","reviewed_head":"1266347a901798d554eb428c2b65801d190923fb","slot":"pair:8","requested_at":"2026-10-22T21:30:42.471459Z"},"phase":"finished","pid":10146,"started_at":"2026-10-22T21:30:42.591618Z","finished_at":"2026-10-22T21:31:20.699383Z","session_id":"583a2a4e-4f42-46ed-81cb-b0016dc19539","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.462715917s
head: "1266347a901798d554eb428c2b65801d190923fb"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
