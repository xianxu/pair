---
gate: boundary-review
issue: 206
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-11T15:16:10-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Minor
          title: Compress the prose test-case lists into per-function strategy lines, and add generated event-sequence invariant tests for ReattachPass
          detail: |-
            Tasks 2.4, 5.1, 7.1, 8.1 and 9.1 enumerate cases. Invariants to check: at most one attempt, root never queued, queue never grows, no emit while an operator op holds the slot, Queue, Attached and Failed disjoint. Hand-picked rows sample one interleaving each.
            (carried from plan-quality PQ-5, deferred to the boundary review)
          family: test-plan-enumerates-cases
          round: 1
        - id: BR-2
          severity: Minor
          title: Worst-case attempt bound and pass duration omit repeated 5 s zellij queries and the O(N squared) per-completion inventory refresh
          detail: |-
            DetachedSessions runs in ResumeContext and again in confirmStillDetached, each query bounded at 5 s (zellij.go:20). finishOperation's requestMenuRefresh (console.go:1884) re-asks every remaining candidate. Task 12 should measure it.
            (carried from plan-quality PQ-6, deferred to the boundary review)
          family: envelope-omits-cost-source
          round: 1
        - id: BR-3
          severity: Minor
          title: Revisions changed the prose but not the code block, the task steps and the file lists that restate them
          detail: |-
            The ReattachPass struct still has Attached map[...]bool, commented "awaiting an inventory that shows them Live", and Task 6 Step 2 names clearAttachedWhenLive; both contradict cell 13. Task 2 Step 4 says the fake does not model the duplicate-name rule, which is false since PQ-8. Task 8 omits declaring background on attach in ops.go and bumping the attach arity of 2 (run_test.go:586). Core concepts also says PathHoldsUnreadableThread reads only cwd rows; it checks the whole scope (startup.go:112-117). That is harmless, because unreadable records never reach the part of the scan the filter controls. Rule: when a decision changes, grep every restatement (code blocks, tables, steps, file lists, test names) and change them in the same edit. PQ-7's three sites belong to the same class.
            (carried from plan-quality PQ-10, deferred to the boundary review)
          family: decision-restated-not-swept
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-09-11T15:16:10-07:00"
      agent: claude
      findings:
        - id: BR-4
          severity: Important
          title: TestNarrowedStartupAnswersAsAFullProofWould ships 4 of the plan's 7 rows and 3 of its 4 readers
          detail: 'Missing rows: parked at cwd; cwd session gone; a cwd thread sharing a session name with an unasked thread at sandboxedChecker. Missing reader: PathHoldsUnreadableThread. The shared-name row is the only end-to-end composition of the narrowing with Task 1''s claim count.'
          family: plan-test-traceability
          round: 2
        - id: BR-5
          severity: Important
          title: Task 4's M1 close criterion, the operator smoke, has no Log entry
          detail: The plan replaced the pulled-forward trace with the operator restarting couch on a 10-plus-thread store and reporting. The Log records the suite and mutations only. Run and log it, or revise the plan to defer it to M2's smoke.
          family: operator-smoke-before-close
          round: 2
        - id: BR-6
          severity: Minor
          title: DetachedSessions derives each candidate's newest binding twice (lookupSessionName and effectiveBindings)
          detail: Derive bindings from effectiveBindings(reads)[address] so the candidate's name and its claim are one value by construction (artifactcollision.go:323-351).
          family: dry-duplicate-derivation
          round: 2
        - id: BR-7
          severity: Minor
          title: ProjectDetachedSessions' comment says "the scope's whole index"; the reach is legacy plus asked scopes
          detail: A claimant whose only row is in an unasked scope file is invisible, and a migrated thread's stale legacy row still counts under its old name. Theoretical for couch threads (random 8-byte tags), but the comment should say so.
          family: documented-rule-reach
          round: 2
        - id: BR-8
          severity: Minor
          title: The "conflicting layout elsewhere" row uses layout1, which ParseLayoutMode rejects, so it duplicates the unreadable row
          detail: 'Use layout3. Also in startup_proof_test.go: threadCounters is written and never read, cwd is discarded, containsAll reimplements strings.Contains, and the conflict comparison checks lengths rather than addresses.'
          family: test-fixture-honesty
          round: 2
        - id: BR-9
          severity: Minor
          title: 'Durable plan lags the code: Tasks 2-3 unticked, prose names sessionNameClaims, Task 2 Step 1 describes a zellij-seam count the test takes at the fake seam'
          detail: Tick the delivered steps, rename the helper in the Core concepts prose to effectiveBindings plus claimsFromBindings, and record the seam deviation in a Revisions entry.
          family: plan-drift-from-code
          round: 2
      boundary: M1
      blocked: true
    - "n": 3
      timestamp: "2026-09-11T15:55:21-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: Task 6 Step 0 adds the generated-sequence invariants in this window; the enumerated prose lists in Tasks 5.1, 8.1 and 9.1 remain, which is the stylistic half.
          round: 3
        - id: BR-2
          disposition: not-addressed
          note: Nothing in the window touched the envelope; M2 Task 12 scope, carry to the M2 close review and name the per-completion refresh and both 5 s bounds in its measurement bullet.
          round: 3
        - id: BR-3
          disposition: addressed
          note: Struct block, Task 6 Step 2, Task 8 Step 4 and the PathHoldsUnreadableThread prose are swept; Task 2 Step 4's sandboxedChecker clause is superseded by the M1-review Revisions entry. The family recurred, see the new finding.
          round: 3
        - id: BR-4
          disposition: addressed
          note: 'Seven rows and four readers; revert-verified: counting the fake''s claims over asked candidates only turns the shared-name row red.'
          round: 3
        - id: BR-5
          disposition: addressed
          note: Log entry "M1 operator smoke" records the 1.5x report, what it implies, and what remains unmeasured.
          round: 3
        - id: BR-6
          disposition: addressed
          note: DetachedSessions derives bindings from effectiveBindings(reads)[address]; lookupSessionName remains only for PairSession.
          round: 3
        - id: BR-7
          disposition: addressed
          note: The ProjectDetachedSessions comment now states legacy plus asked scopes and both theoretical gaps.
          round: 3
        - id: BR-8
          disposition: addressed
          note: layout3 row with the refusal test covering both kinds; threadCounter is used; cwd is asserted; strings.Contains; conflicts compared by address.
          round: 3
        - id: BR-9
          disposition: not-addressed
          note: Tasks 2-3 ticked and sessionNameClaims renamed, but Task 2 Step 1 still describes withOthers and a zellij-seam list-clients count no test takes, with no Revisions line; Task 4's delivered rows (smoke, atlas, mutation sweep, make test) are unticked while the Log records each.
          round: 3
      findings:
        - id: BR-10
          severity: Minor
          title: 'Round-2 decisions left six restatements unswept: the reader count, the M1 close criterion, and lookupSessionName''s orphaned doc comment'
          detail: 'This is the 2nd finding in family decision-restated-not-swept. Sites: startup.go:139-140 and atlas/couch.md:747 say three readers and a fourth would widen the predicate while the test asserts four (startup_proof_test.go:155; its header at line 5 still says three); plan lines 379-381 promise a measured first-frame time the Revisions entry replaced with the smoke; plan line 150 has a dangling "and"; artifactcollision.go:136-140 is lookupSessionName''s doc comment now heading scopedIndexRead and claiming DetachedSessions still calls it. BR-3''s rule (grep every restatement in the same edit) was stated in round 1 and the next two commits produced these six, so the rule as written is not holding. The rule that covers all of them: do not restate, point. A count or list lives in one place (the test''s row table, the reader list in startupAsks'' comment) and every other site names it rather than repeating a number; and each Revisions entry ends with a swept line listing every site changed, which a reviewer diffs against a grep of the old term. Measured prevalence this round: six sites across code, atlas and plan.'
          family: decision-restated-not-swept
          round: 3
        - id: BR-11
          severity: Minor
          title: DetachedSessions' comment says a scope whose index cannot be read contributes no bindings; after the union its legacy-bound threads are bound and counted from other reads
          detail: 'This is the 2nd finding in family documented-rule-reach. artifactcollision.go:275-278 states the old reach; the bindings loop at 329-336 iterates scopes rather than reads, so a failed scope''s threads take current[address] from legacy rows replayed in another scope''s successful read. A scope read fails only when its own file exists but will not read or decode (launcher/session_index.go:225-235), so this is rare and arguably the better answer, but it is unstated and no test pins either behaviour. The rule that covers BR-7 and this: a comment that names a reach (per scope, the whole index, fail closed) is a claim about which data the code consults, and it is pinned by a test the comment names. The class fix is a test per reach claim in DetachedSessions and ProjectDetachedSessions, including one for a scope whose file will not decode, rather than rewording this comment. Prevalence: two reach claims in the same two functions, neither pinned when raised.'
          family: documented-rule-reach
          round: 3
        - id: BR-12
          severity: Minor
          title: lookupSessionName and effectiveBindings are two derivations of a thread's newest binding in one read; PairSession uses one, DetachedSessions the other
          detail: 'This is the 2nd finding in family dry-duplicate-derivation. artifactcollision.go:196-204 scans one index backwards for one address; effectiveBindings'' per-read latest map at 165-168 computes the same fact for every address. If the single-read semantics ever diverge, PairSession and DetachedSessions judge the same thread by different names. The rule: a thread''s current session name has exactly one derivation and every reader calls it. The class fix is to delete lookupSessionName and have PairSession call effectiveBindings over its single read, so the rule is enforced by there being one function rather than by two agreeing. The test helper claimsOf (detachedsessions_test.go:253-261) is a third copy of the counting rule and can call claimsFromBindings. Prevalence: three derivations, two production readers.'
          family: dry-duplicate-derivation
          round: 3
      boundary: M1
      blocked: false
---

# Gate ledger — pair#206 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-11T15:16:10-07:00 (sdlc) — passed

### Raised

- **BR-1** [Minor] `test-plan-enumerates-cases` Compress the prose test-case lists into per-function strategy lines, and add generated event-sequence invariant tests for ReattachPass
  Tasks 2.4, 5.1, 7.1, 8.1 and 9.1 enumerate cases. Invariants to check: at most one attempt, root never queued, queue never grows, no emit while an operator op holds the slot, Queue, Attached and Failed disjoint. Hand-picked rows sample one interleaving each.
  (carried from plan-quality PQ-5, deferred to the boundary review)
- **BR-2** [Minor] `envelope-omits-cost-source` Worst-case attempt bound and pass duration omit repeated 5 s zellij queries and the O(N squared) per-completion inventory refresh
  DetachedSessions runs in ResumeContext and again in confirmStillDetached, each query bounded at 5 s (zellij.go:20). finishOperation's requestMenuRefresh (console.go:1884) re-asks every remaining candidate. Task 12 should measure it.
  (carried from plan-quality PQ-6, deferred to the boundary review)
- **BR-3** [Minor] `decision-restated-not-swept` Revisions changed the prose but not the code block, the task steps and the file lists that restate them
  The ReattachPass struct still has Attached map[...]bool, commented "awaiting an inventory that shows them Live", and Task 6 Step 2 names clearAttachedWhenLive; both contradict cell 13. Task 2 Step 4 says the fake does not model the duplicate-name rule, which is false since PQ-8. Task 8 omits declaring background on attach in ops.go and bumping the attach arity of 2 (run_test.go:586). Core concepts also says PathHoldsUnreadableThread reads only cwd rows; it checks the whole scope (startup.go:112-117). That is harmless, because unreadable records never reach the part of the scan the filter controls. Rule: when a decision changes, grep every restatement (code blocks, tables, steps, file lists, test names) and change them in the same edit. PQ-7's three sites belong to the same class.
  (carried from plan-quality PQ-10, deferred to the boundary review)

## Round 2 — 2026-09-11T15:16:10-07:00 (claude) — BLOCKED

### Raised

- **BR-4** [Important] `plan-test-traceability` TestNarrowedStartupAnswersAsAFullProofWould ships 4 of the plan's 7 rows and 3 of its 4 readers
  Missing rows: parked at cwd; cwd session gone; a cwd thread sharing a session name with an unasked thread at sandboxedChecker. Missing reader: PathHoldsUnreadableThread. The shared-name row is the only end-to-end composition of the narrowing with Task 1's claim count.
- **BR-5** [Important] `operator-smoke-before-close` Task 4's M1 close criterion, the operator smoke, has no Log entry
  The plan replaced the pulled-forward trace with the operator restarting couch on a 10-plus-thread store and reporting. The Log records the suite and mutations only. Run and log it, or revise the plan to defer it to M2's smoke.
- **BR-6** [Minor] `dry-duplicate-derivation` DetachedSessions derives each candidate's newest binding twice (lookupSessionName and effectiveBindings)
  Derive bindings from effectiveBindings(reads)[address] so the candidate's name and its claim are one value by construction (artifactcollision.go:323-351).
- **BR-7** [Minor] `documented-rule-reach` ProjectDetachedSessions' comment says "the scope's whole index"; the reach is legacy plus asked scopes
  A claimant whose only row is in an unasked scope file is invisible, and a migrated thread's stale legacy row still counts under its old name. Theoretical for couch threads (random 8-byte tags), but the comment should say so.
- **BR-8** [Minor] `test-fixture-honesty` The "conflicting layout elsewhere" row uses layout1, which ParseLayoutMode rejects, so it duplicates the unreadable row
  Use layout3. Also in startup_proof_test.go: threadCounters is written and never read, cwd is discarded, containsAll reimplements strings.Contains, and the conflict comparison checks lengths rather than addresses.
- **BR-9** [Minor] `plan-drift-from-code` Durable plan lags the code: Tasks 2-3 unticked, prose names sessionNameClaims, Task 2 Step 1 describes a zellij-seam count the test takes at the fake seam
  Tick the delivered steps, rename the helper in the Core concepts prose to effectiveBindings plus claimsFromBindings, and record the seam deviation in a Revisions entry.

## Round 3 — 2026-09-11T15:55:21-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — Task 6 Step 0 adds the generated-sequence invariants in this window; the enumerated prose lists in Tasks 5.1, 8.1 and 9.1 remain, which is the stylistic half.
- BR-2 — not-addressed — Nothing in the window touched the envelope; M2 Task 12 scope, carry to the M2 close review and name the per-completion refresh and both 5 s bounds in its measurement bullet.
- BR-3 — addressed — Struct block, Task 6 Step 2, Task 8 Step 4 and the PathHoldsUnreadableThread prose are swept; Task 2 Step 4's sandboxedChecker clause is superseded by the M1-review Revisions entry. The family recurred, see the new finding.
- BR-4 — addressed — Seven rows and four readers; revert-verified: counting the fake's claims over asked candidates only turns the shared-name row red.
- BR-5 — addressed — Log entry "M1 operator smoke" records the 1.5x report, what it implies, and what remains unmeasured.
- BR-6 — addressed — DetachedSessions derives bindings from effectiveBindings(reads)[address]; lookupSessionName remains only for PairSession.
- BR-7 — addressed — The ProjectDetachedSessions comment now states legacy plus asked scopes and both theoretical gaps.
- BR-8 — addressed — layout3 row with the refusal test covering both kinds; threadCounter is used; cwd is asserted; strings.Contains; conflicts compared by address.
- BR-9 — not-addressed — Tasks 2-3 ticked and sessionNameClaims renamed, but Task 2 Step 1 still describes withOthers and a zellij-seam list-clients count no test takes, with no Revisions line; Task 4's delivered rows (smoke, atlas, mutation sweep, make test) are unticked while the Log records each.

### Raised

- **BR-10** [Minor] `decision-restated-not-swept` Round-2 decisions left six restatements unswept: the reader count, the M1 close criterion, and lookupSessionName's orphaned doc comment
  This is the 2nd finding in family decision-restated-not-swept. Sites: startup.go:139-140 and atlas/couch.md:747 say three readers and a fourth would widen the predicate while the test asserts four (startup_proof_test.go:155; its header at line 5 still says three); plan lines 379-381 promise a measured first-frame time the Revisions entry replaced with the smoke; plan line 150 has a dangling "and"; artifactcollision.go:136-140 is lookupSessionName's doc comment now heading scopedIndexRead and claiming DetachedSessions still calls it. BR-3's rule (grep every restatement in the same edit) was stated in round 1 and the next two commits produced these six, so the rule as written is not holding. The rule that covers all of them: do not restate, point. A count or list lives in one place (the test's row table, the reader list in startupAsks' comment) and every other site names it rather than repeating a number; and each Revisions entry ends with a swept line listing every site changed, which a reviewer diffs against a grep of the old term. Measured prevalence this round: six sites across code, atlas and plan.
- **BR-11** [Minor] `documented-rule-reach` DetachedSessions' comment says a scope whose index cannot be read contributes no bindings; after the union its legacy-bound threads are bound and counted from other reads
  This is the 2nd finding in family documented-rule-reach. artifactcollision.go:275-278 states the old reach; the bindings loop at 329-336 iterates scopes rather than reads, so a failed scope's threads take current[address] from legacy rows replayed in another scope's successful read. A scope read fails only when its own file exists but will not read or decode (launcher/session_index.go:225-235), so this is rare and arguably the better answer, but it is unstated and no test pins either behaviour. The rule that covers BR-7 and this: a comment that names a reach (per scope, the whole index, fail closed) is a claim about which data the code consults, and it is pinned by a test the comment names. The class fix is a test per reach claim in DetachedSessions and ProjectDetachedSessions, including one for a scope whose file will not decode, rather than rewording this comment. Prevalence: two reach claims in the same two functions, neither pinned when raised.
- **BR-12** [Minor] `dry-duplicate-derivation` lookupSessionName and effectiveBindings are two derivations of a thread's newest binding in one read; PairSession uses one, DetachedSessions the other
  This is the 2nd finding in family dry-duplicate-derivation. artifactcollision.go:196-204 scans one index backwards for one address; effectiveBindings' per-read latest map at 165-168 computes the same fact for every address. If the single-read semantics ever diverge, PairSession and DetachedSessions judge the same thread by different names. The rule: a thread's current session name has exactly one derivation and every reader calls it. The class fix is to delete lookupSessionName and have PairSession call effectiveBindings over its single read, so the rule is enforced by there being one function rather than by two agreeing. The test helper claimsOf (detachedsessions_test.go:253-261) is a third copy of the counting rule and can call claimsFromBindings. Prevalence: three derivations, two production readers.

## Open findings

- **BR-2** [Minor] `envelope-omits-cost-source` Worst-case attempt bound and pass duration omit repeated 5 s zellij queries and the O(N squared) per-completion inventory refresh
- **BR-9** [Minor] `plan-drift-from-code` Durable plan lags the code: Tasks 2-3 unticked, prose names sessionNameClaims, Task 2 Step 1 describes a zellij-seam count the test takes at the fake seam
- **BR-10** [Minor] `decision-restated-not-swept` Round-2 decisions left six restatements unswept: the reader count, the M1 close criterion, and lookupSessionName's orphaned doc comment
- **BR-11** [Minor] `documented-rule-reach` DetachedSessions' comment says a scope whose index cannot be read contributes no bindings; after the union its legacy-bound threads are bound and counted from other reads
- **BR-12** [Minor] `dry-duplicate-derivation` lookupSessionName and effectiveBindings are two derivations of a thread's newest binding in one read; PairSession uses one, DetachedSessions the other
