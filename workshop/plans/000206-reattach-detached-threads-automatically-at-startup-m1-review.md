# Boundary Review — pair#206 (milestone M1)

| field | value |
|-------|-------|
| issue | 206 — reattach detached threads automatically at startup |
| repo | pair |
| issue file | workshop/issues/000206-reattach-detached-threads-automatically-at-startup.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | f643ca8e890c638c742219c84562cb97c6eb14fd..df876f7d13ea9ffdbcd8071f05f6d009a69ab660 |
| command | sdlc milestone-close --issue 206 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-09-11T15:16:10-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M1 delivers what the issue's M1 row claims: startup resolves only the cwd candidates and layout-conflicting candidates, the filter sits before the ledger read as well as the zellij ask, and the duplicate-name rule now counts each thread once over the union of index files a call reads. I re-ran the window's tests (green; the only failures in the package are pty-child spawns the sandbox blocks) and reverted five of the seven named mutations in scratch copies. Every one went red on the test the Log names. Nothing blocks the boundary. What keeps this from SHIP is traceability, not code: the plan's equivalence table ships with three of its seven rows and three of its four readers, and Task 4's M1 close criterion, the operator smoke, has no Log entry.

**1. Strengths**

- **Filter placed where the cost is.** The `ask` predicate runs after physicalization and before `ResolveEstablished` (`cmd/internal/couchcore/actionableinventory.go:490-497`), and the fake's `BindingResolutions` counter catches it being moved. Verified by reverting.
- **The fake answers through production's rule** (`cmd/internal/couchcore/artifactcollision_fake.go:110-129`). Fake-backed tests can no longer pass on a proof production refuses. ARCH-MOCK, confirmed good.
- **Union semantics pinned order-independently.** `effectiveBindings` (`artifactcollision.go:159-179`) is tested in both read orders, and the two-scope seam test kills the per-read summing bug the plan gate found.
- **Fixture honesty.** `detachedThreadAt` resolves the real scope key and sets a native binding (`startup_proof_test.go:46-72`), with comments naming the two ways the earlier fixture passed for the wrong reason.
- **Atlas names the exact reader set** the predicate serves (`atlas/couch.md:734-748`), so a fourth reader has a place to be noticed.

**2. Critical findings**

None.

**3. Important findings**

- **Plan Task 2 Step 4 is partially delivered** (`startup_proof_test.go:142-197`). The plan lists seven rows and four readers. The test has four rows and asserts three readers. Missing rows: parked at cwd; cwd session gone; and a cwd thread sharing a session name with an unasked thread at `sandboxedChecker`. Missing reader: `PathHoldsUnreadableThread`. The shared-name row matters most. It is the only place the narrowing and Task 1 are composed end to end. Today each half is pinned alone. Fix: add the three rows and the fourth assertion, or revise the plan to say which rows were dropped and why. ARCH-PURPOSE: the class was enumerated in the plan and the sweep stopped short.
- **The M1 close criterion has no evidence.** Task 4's first bullet makes the operator's smoke ("restart couch with 10 or more threads, report whether startup feels faster") M1's timing evidence, replacing the pulled-forward trace. The issue Log records the suite and the mutation sweep but no smoke. Fix before crossing: run it and log the result, or write a Revisions entry deferring it to M2's smoke. This repo has a recorded lesson about closing without the operator's smoke.

**4. Minor findings**

- **ARCH-DRY, two derivations of one fact.** `DetachedSessions` computes each candidate's newest binding with `lookupSessionName` (`artifactcollision.go:323-331`) and again inside `effectiveBindings` for the claims. Derive `bindings` from `effectiveBindings(reads)[address]` so a candidate's name and its claim are one value by construction.
- **State the rule's real reach.** "The scope's whole index" is the legacy file plus the asked scopes' files. A claimant whose only row is in an unasked scope's file is invisible, and a migrated thread's stale legacy row still counts under its old `pair-` name. Both are theoretical for couch threads because tags are eight random bytes (`threadtag.go:33`), but the comment on `ProjectDetachedSessions` reads wider than the code. One sentence in that comment.
- **Fixture row mislabelled.** The "conflicting layout elsewhere" row uses `layout1`, which `ParseLayoutMode` does not know, so it normalizes to `unknown` and duplicates the "unreadable layout elsewhere" row. Use `layout3` for a known-but-different layout.
- **Test hygiene** in `startup_proof_test.go`: `threadCounters` is written and never read (a data race if any test goes parallel); `_ = cwd` at line 125; `containsAll` reimplements `strings.Contains`; the conflict comparison at line 177 checks lengths, not addresses.
- **Plan bookkeeping.** Tasks 2 and 3 are unticked in the durable plan while the Log says M1 is implemented. The Core concepts prose names `sessionNameClaims`, which does not exist (the code is `effectiveBindings` plus `claimsFromBindings`). Task 2 Step 1 says the count is `list-clients` at the zellij seam with `withOthers`; the test counts candidates at the fake seam, which the Log justifies but the plan does not record.

**5. Test coverage notes**

- Reverted and confirmed red: ask ignored; ask moved after `ResolveEstablished`; layout arm dropped; claims over passed bindings only; claims summed per read. Each failed on the named test with the expected message. The cwd-arm and index-entry mutations were not re-run.
- The windowed tests pass under `go test ./cmd/internal/couchcore/ -run ...`. `go vet` is clean. The full package fails only on pty-child spawns, which the sandbox denies. I did not re-run `make test`.
- PURE entities (`startupAsks`, `effectiveBindings`, `claimsFromBindings`, `ProjectDetachedSessions`) are tested without IO. INTEGRATION goes through the fake or the sandboxed zellij stub.

**6. Architectural notes for upcoming work**

- ARCH-DRY: flag, the double binding derivation above.
- ARCH-PURE: pass. The IO shell builds the claim map; the rule is pure.
- ARCH-PURPOSE: pass for M1's stated purpose. I swept every reader of `rows` in `StartInteractive` and `spawnResolved` and found exactly the four the predicate covers. Flag on the plan's own test enumeration.
- ARCH-MOCK: pass. Production and fake share `ProjectDetachedSessions` and `claimsFromBindings`.
- ARCH-CONSTRAINTS: pass on the counted invariant at two store sizes. The timing half is delegated to the unrecorded smoke.
- ARCH-SECURE: pass. Index entries go through the strict decoder; the layout witness is normalized at projection.
- ARCH-ORDER: pass. M1 holds no state between events: the predicate is evaluated once per snapshot and `DetachedSessions` is a single-shot read.
- For M2: the pass will seed from `ThreadDetached` or resume-shaped `unknown`. After M1, every non-cwd same-layout candidate is `unknown` in startup's rows, so the plan's insistence that the seed comes from the first console inventory, not startup's rows, is load-bearing. Keep the `StartResult carries no rows` invariant pinned when M2 lands.

**7. Plan revision recommendations**

- A `## Revisions` entry recording the Task 2 deviations: candidates counted at the fake seam rather than `list-clients` at the zellij seam, and which equivalence rows and readers were dropped (or that they are being added now).
- Rename `sessionNameClaims` in the Core concepts prose to `effectiveBindings` plus `claimsFromBindings`, and state the rule's reach (asked scopes plus legacy).
- Tick Tasks 2 and 3, and either tick Task 4's smoke with a Log pointer or move it to M2 Task 12 explicitly.
- For the main agent: add a `workshop/lessons.md` rule from this round. A test fixture must use values the code recognizes, or its row labels lie.

```findings
findings:
  - id: new
    severity: Important
    family: plan-test-traceability
    title: |
      TestNarrowedStartupAnswersAsAFullProofWould ships 4 of the plan's 7 rows and 3 of its 4 readers
    detail: |
      Missing rows: parked at cwd; cwd session gone; a cwd thread sharing a session name with an unasked thread at sandboxedChecker. Missing reader: PathHoldsUnreadableThread. The shared-name row is the only end-to-end composition of the narrowing with Task 1's claim count.
  - id: new
    severity: Important
    family: operator-smoke-before-close
    title: |
      Task 4's M1 close criterion, the operator smoke, has no Log entry
    detail: |
      The plan replaced the pulled-forward trace with the operator restarting couch on a 10-plus-thread store and reporting. The Log records the suite and mutations only. Run and log it, or revise the plan to defer it to M2's smoke.
  - id: new
    severity: Minor
    family: dry-duplicate-derivation
    title: |
      DetachedSessions derives each candidate's newest binding twice (lookupSessionName and effectiveBindings)
    detail: |
      Derive bindings from effectiveBindings(reads)[address] so the candidate's name and its claim are one value by construction (artifactcollision.go:323-351).
  - id: new
    severity: Minor
    family: documented-rule-reach
    title: |
      ProjectDetachedSessions' comment says "the scope's whole index"; the reach is legacy plus asked scopes
    detail: |
      A claimant whose only row is in an unasked scope file is invisible, and a migrated thread's stale legacy row still counts under its old name. Theoretical for couch threads (random 8-byte tags), but the comment should say so.
  - id: new
    severity: Minor
    family: test-fixture-honesty
    title: |
      The "conflicting layout elsewhere" row uses layout1, which ParseLayoutMode rejects, so it duplicates the unreadable row
    detail: |
      Use layout3. Also in startup_proof_test.go: threadCounters is written and never read, cwd is discarded, containsAll reimplements strings.Contains, and the conflict comparison checks lengths rather than addresses.
  - id: new
    severity: Minor
    family: plan-drift-from-code
    title: |
      Durable plan lags the code: Tasks 2-3 unticked, prose names sessionNameClaims, Task 2 Step 1 describes a zellij-seam count the test takes at the fake seam
    detail: |
      Tick the delivered steps, rename the helper in the Core concepts prose to effectiveBindings plus claimsFromBindings, and record the seam deviation in a Revisions entry.
```
