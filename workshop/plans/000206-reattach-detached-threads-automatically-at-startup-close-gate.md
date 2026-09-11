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

## Open findings

- **BR-1** [Minor] `test-plan-enumerates-cases` Compress the prose test-case lists into per-function strategy lines, and add generated event-sequence invariant tests for ReattachPass
- **BR-2** [Minor] `envelope-omits-cost-source` Worst-case attempt bound and pass duration omit repeated 5 s zellij queries and the O(N squared) per-completion inventory refresh
- **BR-3** [Minor] `decision-restated-not-swept` Revisions changed the prose but not the code block, the task steps and the file lists that restate them
- **BR-4** [Important] `plan-test-traceability` TestNarrowedStartupAnswersAsAFullProofWould ships 4 of the plan's 7 rows and 3 of its 4 readers
- **BR-5** [Important] `operator-smoke-before-close` Task 4's M1 close criterion, the operator smoke, has no Log entry
- **BR-6** [Minor] `dry-duplicate-derivation` DetachedSessions derives each candidate's newest binding twice (lookupSessionName and effectiveBindings)
- **BR-7** [Minor] `documented-rule-reach` ProjectDetachedSessions' comment says "the scope's whole index"; the reach is legacy plus asked scopes
- **BR-8** [Minor] `test-fixture-honesty` The "conflicting layout elsewhere" row uses layout1, which ParseLayoutMode rejects, so it duplicates the unreadable row
- **BR-9** [Minor] `plan-drift-from-code` Durable plan lags the code: Tasks 2-3 unticked, prose names sessionNameClaims, Task 2 Step 1 describes a zellij-seam count the test takes at the fake seam
