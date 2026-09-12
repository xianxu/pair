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

---

## Re-review — 2026-09-11T15:55:21-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 206 — reattach detached threads automatically at startup |
| repo | pair |
| issue file | workshop/issues/000206-reattach-detached-threads-automatically-at-startup.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | f643ca8e890c638c742219c84562cb97c6eb14fd..a57fba6ada92c0a099d69c585fd1217186aa936d |
| command | sdlc milestone-close --issue 206 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-09-11T15:55:21-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

M1 delivers the issue's M1 row: startup resolves only the cwd candidates and the layout-conflicting candidates, the filter sits before the ledger read as well as the zellij ask, and the duplicate-name rule counts each thread once over the union of index files a call reads. Every prior Important finding is addressed and I verified the two that mattered by reverting them in a scratch tree: counting the fake's claims over the asked candidates only turns the shared-name equivalence row red, and dropping the scope half of the cwd arm turns the foreign-scope count test red. The operator smoke is logged with an honest reading of what it does and does not show. What remains is Minor: two carried plan-bookkeeping items, and three new instances of families already in play, each with the covering rule stated below rather than a per-site fix.

**1. Strengths**

- **The claim count and the candidate's own name are one value** (`cmd/internal/couchcore/artifactcollision.go:328-336`). `effectiveBindings` feeds both, so the two lookups cannot disagree. BR-6 is closed by construction, not by a comment.
- **The equivalence test now composes the narrowing with Task 1's rule end to end** (`cmd/internal/couchcore/startup_proof_test.go:188-191`). Under mutation A the narrowed inventory selected the cwd thread while the full one refused it, which is exactly the bug the row exists for.
- **A count test where an equivalence test is blind** (`startup_proof_test.go:300-335`). The comment says why over-asking never changes an answer, and the test pins the scope half of the cwd arm with a resolution count. Verified red under mutation B.
- **The smoke was read against the code, not credited to it** (issue Log, "M1 operator smoke"). The unexplained detach speedup was checked function by function, ruled against code, and carried to pair#229 as input rather than claimed.
- **Order independence is tested, not assumed** (`detachedsessions_test.go:283-322`). `effectiveBindings` runs in both read orders in the same test.

**2. Critical findings**

None.

**3. Important findings**

None.

**4. Minor findings**

- **Restatements left unswept by the round-2 decisions** (2nd in `decision-restated-not-swept`). `startup.go:139-140` and `atlas/couch.md:747` say the test asserts three readers and a fourth would widen the predicate; the test asserts four (`startup_proof_test.go:155`, and its own header at line 5 still says three). The plan's envelope (lines 379-381) still promises a measured first-frame time that the Revisions entry replaced with the smoke. The plan's reader list (line 150) has a dangling "and". And the BR-6 fix inserted `scopedIndexRead` between `lookupSessionName`'s doc comment and its function, so the comment now heads the wrong declaration and claims `DetachedSessions` still calls it (`artifactcollision.go:136-140`). Six sites.
- **`DetachedSessions`' fail-closed claim no longer matches its reach** (2nd in `documented-rule-reach`). The doc at `artifactcollision.go:275-278` says a scope whose index cannot be read contributes no bindings. The bindings loop now iterates `scopes`, not `reads` (`:329-336`), so a failed scope's threads are bound and claim-counted from legacy rows replayed in another scope's successful read. Rare (a scope file that exists but will not decode, `launcher/session_index.go:225-235`) and arguably the better answer, but unstated and unpinned.
- **Two derivations of a thread's newest binding in one read** (2nd in `dry-duplicate-derivation`). `lookupSessionName` (`artifactcollision.go:196-204`) and the per-read `latest` map inside `effectiveBindings` (`:165-168`) compute the same fact; `PairSession` uses the first, `DetachedSessions` the second. The test helper `claimsOf` (`detachedsessions_test.go:253-261`) is a third copy of the counting rule.
- **Carried, not addressed:** BR-2 (M2 Task 12's envelope) and the remainder of BR-9 (Task 2 Step 1 still describes `withOthers` and a zellij-seam count; Task 4's delivered rows are unticked).

**5. Test coverage notes**

- Ran `go vet` (clean) and the full `couchcore` package. The only failures are the five pty-child spawn tests the sandbox denies, unrelated to this window. I could not run `make test`; the last logged unsandboxed run predates the round-2 test commits, so the close should re-run it (Task 4's own row).
- Revert-verified in a scratch tree: mutation A (fake counts claims over asked candidates only) fails `TestNarrowedStartupAnswersAsAFullProofWould/the_cwd_thread_shares_its_session_name_with_an_unasked_thread` and `TestFakeDetachedSessionsAppliesTheDuplicateNameRule`; mutation B (scope half of the cwd arm dropped) fails `TestStartupDoesNotProveAForeignScopeRecordAtTheCwdPath`. The round-1 reviewer covered the other five named mutations.
- The `PathHoldsUnreadableThread` assertion is vacuous by construction: unreadable records never reach the gated branch, so no predicate mutation can change its answer. It is a guard for a future change, which is fine, but the test's comment could say so.
- No test pins the fail-closed reach of a scope index read, before or after this window. That is why the union changed it silently.

**6. Architectural notes**

- **ARCH-DRY: flag.** The `lookupSessionName` versus `effectiveBindings` derivation above. BR-6's in-function duplicate is fixed.
- **ARCH-PURE: pass.** `startupAsks`, `effectiveBindings`, `claimsFromBindings` and `ProjectDetachedSessions` are pure and tested without IO; the IO shell only collects reads.
- **ARCH-PURPOSE: pass.** Shadow-sweep of startup's rows: the readers are `ResolveLayoutConflicts` (`layout.go:94`), `SelectResumableRoot` (`startup.go:32`), and the two path guards via `spawnResolved` (`couch.go:379`, `:397`). Nothing else reads `rows`, `StartResult` carries none, and the predicate's layout expression is the same `NormalizeLayout(record.Layout)` the summary carries (`actionableinventory.go:236`).
- **ARCH-MOCK: pass.** Production and fake share `ProjectDetachedSessions` and `claimsFromBindings`, confirmed by mutation A. The fake's map is one binding per thread, so the legacy-replay class is invisible to it and correctly pinned at `sandboxedChecker` instead; the test comment says so.
- **ARCH-CONSTRAINTS: pass.** The invariant is counted in candidates and ledger resolutions at two store sizes, and the smoke is recorded with its unmeasured remainder named. Only the envelope restatement is flagged.
- **ARCH-SECURE: pass.** Index rows go through the strict decoder; the predicate reads typed store records. No secrets in the window.
- **ARCH-ORDER: pass.** M1 carries no state between events. The predicate is evaluated once per snapshot, `DetachedSessions` is a single-shot read, and the merge is order-independent by test.
- **For M2:** the seed must still come from the console's first inventory, never startup's rows, since every non-cwd same-layout candidate is `unknown` there. Keep the "StartResult carries no rows" invariant pinned when M2 lands.

**7. Plan revision recommendations**

- Replace the reader count with a pointer at the three prose sites, and rewrite the envelope bullet to "the counted invariant plus the operator smoke". Fix the dangling "and" at line 150.
- Task 2 Step 1: name `DetachedCandidatesAsked` and `BindingResolutions` at the fake seam, or add the Revisions line BR-9 asked for.
- Tick Task 4's smoke, atlas and mutation rows with Log pointers; re-run and tick `make test` at the close.
- Task 12's measurement bullet: name the per-completion inventory refresh and the two 5 s query bounds (BR-2).
- `workshop/lessons.md`: the branch has no lessons commit. Add the round-1 rule (fixture values the code recognizes) and this round's rule: point at a single-sourced list, never restate its count; a Revisions entry lists the sites it swept.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Task 6 Step 0 adds the generated-sequence invariants in this window; the enumerated prose lists in Tasks 5.1, 8.1 and 9.1 remain, which is the stylistic half.
  - id: BR-2
    disposition: not-addressed
    note: |
      Nothing in the window touched the envelope; M2 Task 12 scope, carry to the M2 close review and name the per-completion refresh and both 5 s bounds in its measurement bullet.
  - id: BR-3
    disposition: addressed
    note: |
      Struct block, Task 6 Step 2, Task 8 Step 4 and the PathHoldsUnreadableThread prose are swept; Task 2 Step 4's sandboxedChecker clause is superseded by the M1-review Revisions entry. The family recurred, see the new finding.
  - id: BR-4
    disposition: addressed
    note: |
      Seven rows and four readers; revert-verified: counting the fake's claims over asked candidates only turns the shared-name row red.
  - id: BR-5
    disposition: addressed
    note: |
      Log entry "M1 operator smoke" records the 1.5x report, what it implies, and what remains unmeasured.
  - id: BR-6
    disposition: addressed
    note: |
      DetachedSessions derives bindings from effectiveBindings(reads)[address]; lookupSessionName remains only for PairSession.
  - id: BR-7
    disposition: addressed
    note: |
      The ProjectDetachedSessions comment now states legacy plus asked scopes and both theoretical gaps.
  - id: BR-8
    disposition: addressed
    note: |
      layout3 row with the refusal test covering both kinds; threadCounter is used; cwd is asserted; strings.Contains; conflicts compared by address.
  - id: BR-9
    disposition: not-addressed
    note: |
      Tasks 2-3 ticked and sessionNameClaims renamed, but Task 2 Step 1 still describes withOthers and a zellij-seam list-clients count no test takes, with no Revisions line; Task 4's delivered rows (smoke, atlas, mutation sweep, make test) are unticked while the Log records each.
findings:
  - id: new
    severity: Minor
    family: decision-restated-not-swept
    title: |
      Round-2 decisions left six restatements unswept: the reader count, the M1 close criterion, and lookupSessionName's orphaned doc comment
    detail: |
      This is the 2nd finding in family decision-restated-not-swept. Sites: startup.go:139-140 and atlas/couch.md:747 say three readers and a fourth would widen the predicate while the test asserts four (startup_proof_test.go:155; its header at line 5 still says three); plan lines 379-381 promise a measured first-frame time the Revisions entry replaced with the smoke; plan line 150 has a dangling "and"; artifactcollision.go:136-140 is lookupSessionName's doc comment now heading scopedIndexRead and claiming DetachedSessions still calls it. BR-3's rule (grep every restatement in the same edit) was stated in round 1 and the next two commits produced these six, so the rule as written is not holding. The rule that covers all of them: do not restate, point. A count or list lives in one place (the test's row table, the reader list in startupAsks' comment) and every other site names it rather than repeating a number; and each Revisions entry ends with a swept line listing every site changed, which a reviewer diffs against a grep of the old term. Measured prevalence this round: six sites across code, atlas and plan.
  - id: new
    severity: Minor
    family: documented-rule-reach
    title: |
      DetachedSessions' comment says a scope whose index cannot be read contributes no bindings; after the union its legacy-bound threads are bound and counted from other reads
    detail: |
      This is the 2nd finding in family documented-rule-reach. artifactcollision.go:275-278 states the old reach; the bindings loop at 329-336 iterates scopes rather than reads, so a failed scope's threads take current[address] from legacy rows replayed in another scope's successful read. A scope read fails only when its own file exists but will not read or decode (launcher/session_index.go:225-235), so this is rare and arguably the better answer, but it is unstated and no test pins either behaviour. The rule that covers BR-7 and this: a comment that names a reach (per scope, the whole index, fail closed) is a claim about which data the code consults, and it is pinned by a test the comment names. The class fix is a test per reach claim in DetachedSessions and ProjectDetachedSessions, including one for a scope whose file will not decode, rather than rewording this comment. Prevalence: two reach claims in the same two functions, neither pinned when raised.
  - id: new
    severity: Minor
    family: dry-duplicate-derivation
    title: |
      lookupSessionName and effectiveBindings are two derivations of a thread's newest binding in one read; PairSession uses one, DetachedSessions the other
    detail: |
      This is the 2nd finding in family dry-duplicate-derivation. artifactcollision.go:196-204 scans one index backwards for one address; effectiveBindings' per-read latest map at 165-168 computes the same fact for every address. If the single-read semantics ever diverge, PairSession and DetachedSessions judge the same thread by different names. The rule: a thread's current session name has exactly one derivation and every reader calls it. The class fix is to delete lookupSessionName and have PairSession call effectiveBindings over its single read, so the rule is enforced by there being one function rather than by two agreeing. The test helper claimsOf (detachedsessions_test.go:253-261) is a third copy of the counting rule and can call claimsFromBindings. Prevalence: three derivations, two production readers.
```
