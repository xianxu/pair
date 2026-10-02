---
gate: boundary-review
issue: 365
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-01T14:47:12-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Minor
          title: Task 2.1 enumerates 13 test cases in prose; compress to a strategy line
          detail: |-
            Drive permuted event sequences through Advance and assert invariants (no stale token/pane connects; newest token wins; displaced never resurrects) — enumeration misses orderings the generator would cover.
            (carried from plan-quality PQ-4, deferred to the boundary review)
          family: plan-enumerates-test-cases
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-10-01T14:47:12-07:00"
      agent: claude
      findings:
        - id: BR-2
          severity: Important
          title: probes/messageidle/bin/ is not gitignored; run.sh arm drops two Go binaries a git add -A would commit
          detail: zellijcalls ships probes/zellijcalls/.gitignore with bin/; messageidle has none (git check-ignore prints nothing). Add probes/messageidle/.gitignore with bin/; consider a test that every probe bin/ dir is ignored.
          family: probe-build-output-ignored
          round: 2
        - id: BR-3
          severity: Minor
          title: realBinary in probes/messageidle/main.go copies zellijcalls realZellij (ARCH-DRY); plan said extend zellijcalls
          detail: Record the deviation in Revisions or extract the PATH-skip-self lookup into a shared probe helper.
          family: shared-helper-not-reused
          round: 2
        - id: BR-4
          severity: Minor
          title: Idle test is single-wrapper with heartbeats, not the planned three-wrapper no-traffic test; Revisions does not record it
          detail: Done-when 1 says multi-slot; add a Revisions entry noting M2's rewrite covers multi-slot.
          family: plan-revision-drift
          round: 2
        - id: BR-5
          severity: Minor
          title: TestParentNameReadsThisTestsParent reads its own pid, not its parent
          family: test-name-matches-assertion
          round: 2
        - id: BR-6
          severity: Minor
          title: messageidle trace TSV grows unbounded while armed; only arm truncates it
          family: artifact-removal-path
          round: 2
      boundary: M1
      recipe: milestone-review
      blocked: true
    - "n": 3
      timestamp: "2026-10-01T14:48:44-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: Plan Task 2.1 still enumerates 13 named tests; compress to the invariant/permutation strategy line before M2 execution (Minor, non-blocking).
          round: 3
        - id: BR-2
          disposition: addressed
          note: Root .gitignore:163 adds /probes/*/bin/ for the whole class; git check-ignore -v confirms it matches probes/messageidle/bin/zellij.
          round: 3
        - id: BR-3
          disposition: addressed
          note: Plan Revisions (2026-10-01 M1 boundary review) records the separate shim and the realBinary/realZellij mirror, with the reason.
          round: 3
        - id: BR-4
          disposition: addressed
          note: Plan Revisions records the single-wrapper heartbeat form and assigns the multi-slot no-traffic rewrite to Task 2.4.
          round: 3
        - id: BR-5
          disposition: addressed
          note: Renamed TestParentNameReadsAKnownProcess with a comment; name now matches the own-pid assertion.
          round: 3
        - id: BR-6
          disposition: not-addressed
          note: run.sh unchanged; disarm could rm the trace since window writes its summary separately (Minor).
          round: 3
      boundary: M1
      recipe: milestone-review
      blocked: false
    - "n": 4
      timestamp: "2026-10-01T15:28:47-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: registry_test.go:151 TestRegistryInterleavingsKeepInvariants drives 400 random event orders through Advance and asserts the stated invariants (no Admit on a stale pane, effects-driven broker equals registry, one connected binding per slot, an older session is never connected once a newer one is admitted) plus coverage floors.
          round: 4
      findings:
        - id: BR-7
          severity: Important
          title: EffectConnect drops a broker.Register failure; registry snapshot says connected, broker has no actor, nothing retries
          detail: 'message_service.go:453 returns silently on Register error (also :459/:470 drop ReconcileObservation errors). Broker tombstones are never deleted, so after 128 distinct bindings per Couch lifetime Register fails "actor capacity reached" and the slot is silently unreachable with no retry (the old heartbeat retried). Rule (ARCH-ORDER): every IO effect''s outcome returns to the reducer as an event; feed Register failure back as a failed admission (backoff, then dormant), log it, and add a service test with a failing Register.'
          family: effect-outcome-not-returned
          round: 4
        - id: BR-8
          severity: Minor
          title: newerSessionForSlot lets a never-admitted newer session permanently displace a working older one
          detail: registry.go:191 counts AwaitingPane/Rejected/Dormant newer sessions; re-admitting an older working session after a pane flicker marks it Displaced (final) while the newer session never connects. The invariant test checks only safety, not liveness. Restrict to Admitting/Admitted newer sessions, or document the choice.
          family: registry-liveness-unpinned
          round: 4
        - id: BR-9
          severity: Minor
          title: M2 revision does not reconcile all Core-concepts rows and Task test bullets with the tree
          detail: '2nd instance of plan-revision-drift. Rule: at each milestone close the Revisions entry sweeps every Core-concepts row and every Task test bullet of that milestone against the tree. Instances: ReconnectBackoff lives in session_protocol.go, not backoff.go; Task 2.3''s helper-subprocess kill test was not written; Task 2.6''s wrapper-level real-broker idle/restart test was delivered at SessionClient level only, so the startPeerRuntime registry-socket wiring is untested.'
          family: plan-revision-drift
          round: 4
        - id: BR-10
          severity: Minor
          title: Exact-send SendTargeted posts from an untracked goroutine and matches the target by raw slot string
          detail: message_service.go:542; a send addressed by family alias would not wake a dormant session. Each send also spawns a goroutine that s.workers does not track; it ends at shutdown.
          family: send-target-slot-match
          round: 4
      boundary: M2
      recipe: milestone-review
      blocked: true
    - "n": 5
      timestamp: "2026-10-01T15:32:04-07:00"
      agent: claude
      dispose:
        - id: BR-7
          disposition: addressed
          note: ConnectFailed is fed back via execute's return and stepped before new input; registry and service tests go red without it. Still no log line for a refused Connect.
          round: 5
        - id: BR-8
          disposition: not-addressed
          note: registry.go:266 newerSessionForSlot still counts any non-Displaced newer session.
          round: 5
        - id: BR-9
          disposition: not-addressed
          note: No new plan Revisions entry this round; backoff.go still named, 2.3/2.6 gaps unrecorded.
          round: 5
        - id: BR-10
          disposition: not-addressed
          note: message_service.go:542 unchanged.
          round: 5
      findings:
        - id: BR-11
          severity: Minor
          title: Broker actor tombstones are never evicted, so after 128 bindings per Couch lifetime new sessions go dormant
          detail: '2nd finding in family artifact-removal-path. Rule: every bounded table a component writes must name its removal path at the writer, not just a cap. Here, a tombstone whose slot and repository have re-registered under a newer binding is superseded (the registry marks it Displaced, which is final), so evict it on that Register. Pre-existing at base broker.go:136, but M2 turns the failure into silent permanent dormancy where the old heartbeat kept retrying.'
          family: artifact-removal-path
          round: 5
      boundary: M2
      recipe: milestone-review
      blocked: false
    - "n": 6
      timestamp: "2026-10-01T17:20:04-07:00"
      agent: claude
      findings:
        - id: BR-12
          severity: Important
          title: Lost-receipt recovery test re-sends via the b.receipts short-circuit; the wire reserve→AlreadyCommitted adoption path is unpinned
          detail: 'TestPeerLostReceiptRecoveredAfterBrokerRestart calls StatusContext first, which records the receipt, so the re-send returns early at broker.go:427 and never reaches the wrapper''s reserve; peer_runtime.go:67-71, endpoint.go:456-457 and adoptRetained over RemoteEndpoint can be reverted with the test green. 2nd in family. Rule: each test added at this boundary needs an assertion that goes red when its named mechanism is removed. Mutation-check all new M3 tests, and add a fresh-broker reused-ID send plus an in-flight retained receipt case.'
          family: test-name-matches-assertion
          round: 6
        - id: BR-13
          severity: Important
          title: StatusContext records a wrapper-supplied receipt without checking the answering holder is its recipient
          detail: broker.go:239 accepts a receipt if From or To equals the caller; it never checks r.Message.To equals the answering actor's binding (or ID == id for non-Remote holders). Any connected wrapper can plant an outcome for another slot's message. adoptRetained already checks To == a.binding. Carry the binding with each holder, require To == binding and ID == id, and test a forged answer.
          family: untrusted-receipt-provenance
          round: 6
        - id: BR-14
          severity: Minor
          title: Chunk 3 plan lines are unticked; lost-receipt service test, redraw-bytes experiment and lesson are neither delivered nor revised
          detail: '3rd in family. Rule: before milestone-close, every Plan line in the closing milestone is ticked or has a Revisions disposition. Sweep all of Chunk 3, not only the named items.'
          family: plan-revision-drift
          round: 6
      boundary: M3
      recipe: milestone-review
      blocked: true
    - "n": 7
      timestamp: "2026-10-01T17:23:40-07:00"
      agent: claude
      dispose:
        - id: BR-12
          disposition: addressed
          note: Fresh-broker re-send reaches reserve then wire already-committed then adoptRetained; reverting the peer_runtime.go conversion or the endpoint.go rebuild turns the test red (verified in a scratch copy). In-flight is covered only through the shared recordRecoveredLocked via the StatusContext test.
          round: 7
        - id: BR-13
          disposition: addressed
          note: broker.go:658 requires ID == id and To == the answering binding; the forged-answer case in broker_recovery_test.go fails when that check is removed (verified).
          round: 7
        - id: BR-14
          disposition: not-addressed
          note: Plan file untouched in c0cf69af; all Chunk 3 lines still unticked; no Revisions disposition for TestLifecycleLostReceiptAfterBrokerRestartNoDuplicate or the redraw-bytes rerun. Only the lesson item is now delivered.
          round: 7
      findings:
        - id: BR-15
          severity: Minor
          title: TestPeerLostReceiptRecoveredAfterBrokerRestart final check (current ID == m.ID) can never fail
          detail: '3rd in family. A duplicate delivery with the same ID leaves current.Message.ID == m.ID, so the assertion cannot catch it; the test catches the bug only through Send''s error. Rule (already in lessons.md): every assertion must go red when the mechanism its message names is removed. Assert an enqueue or commit counter instead, or delete the line.'
          family: test-name-matches-assertion
          round: 7
      boundary: M3
      recipe: milestone-review
      blocked: false
---

# Gate ledger — pair#365 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-01T14:47:12-07:00 (sdlc) — passed

### Raised

- **BR-1** [Minor] `plan-enumerates-test-cases` Task 2.1 enumerates 13 test cases in prose; compress to a strategy line
  Drive permuted event sequences through Advance and assert invariants (no stale token/pane connects; newest token wins; displaced never resurrects) — enumeration misses orderings the generator would cover.
  (carried from plan-quality PQ-4, deferred to the boundary review)

## Round 2 — 2026-10-01T14:47:12-07:00 (claude) — BLOCKED

### Raised

- **BR-2** [Important] `probe-build-output-ignored` probes/messageidle/bin/ is not gitignored; run.sh arm drops two Go binaries a git add -A would commit
  zellijcalls ships probes/zellijcalls/.gitignore with bin/; messageidle has none (git check-ignore prints nothing). Add probes/messageidle/.gitignore with bin/; consider a test that every probe bin/ dir is ignored.
- **BR-3** [Minor] `shared-helper-not-reused` realBinary in probes/messageidle/main.go copies zellijcalls realZellij (ARCH-DRY); plan said extend zellijcalls
  Record the deviation in Revisions or extract the PATH-skip-self lookup into a shared probe helper.
- **BR-4** [Minor] `plan-revision-drift` Idle test is single-wrapper with heartbeats, not the planned three-wrapper no-traffic test; Revisions does not record it
  Done-when 1 says multi-slot; add a Revisions entry noting M2's rewrite covers multi-slot.
- **BR-5** [Minor] `test-name-matches-assertion` TestParentNameReadsThisTestsParent reads its own pid, not its parent
- **BR-6** [Minor] `artifact-removal-path` messageidle trace TSV grows unbounded while armed; only arm truncates it

## Round 3 — 2026-10-01T14:48:44-07:00 (claude) — passed

### Disposed

- BR-1 — not-addressed — Plan Task 2.1 still enumerates 13 named tests; compress to the invariant/permutation strategy line before M2 execution (Minor, non-blocking).
- BR-2 — addressed — Root .gitignore:163 adds /probes/*/bin/ for the whole class; git check-ignore -v confirms it matches probes/messageidle/bin/zellij.
- BR-3 — addressed — Plan Revisions (2026-10-01 M1 boundary review) records the separate shim and the realBinary/realZellij mirror, with the reason.
- BR-4 — addressed — Plan Revisions records the single-wrapper heartbeat form and assigns the multi-slot no-traffic rewrite to Task 2.4.
- BR-5 — addressed — Renamed TestParentNameReadsAKnownProcess with a comment; name now matches the own-pid assertion.
- BR-6 — not-addressed — run.sh unchanged; disarm could rm the trace since window writes its summary separately (Minor).

## Round 4 — 2026-10-01T15:28:47-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — registry_test.go:151 TestRegistryInterleavingsKeepInvariants drives 400 random event orders through Advance and asserts the stated invariants (no Admit on a stale pane, effects-driven broker equals registry, one connected binding per slot, an older session is never connected once a newer one is admitted) plus coverage floors.

### Raised

- **BR-7** [Important] `effect-outcome-not-returned` EffectConnect drops a broker.Register failure; registry snapshot says connected, broker has no actor, nothing retries
  message_service.go:453 returns silently on Register error (also :459/:470 drop ReconcileObservation errors). Broker tombstones are never deleted, so after 128 distinct bindings per Couch lifetime Register fails "actor capacity reached" and the slot is silently unreachable with no retry (the old heartbeat retried). Rule (ARCH-ORDER): every IO effect's outcome returns to the reducer as an event; feed Register failure back as a failed admission (backoff, then dormant), log it, and add a service test with a failing Register.
- **BR-8** [Minor] `registry-liveness-unpinned` newerSessionForSlot lets a never-admitted newer session permanently displace a working older one
  registry.go:191 counts AwaitingPane/Rejected/Dormant newer sessions; re-admitting an older working session after a pane flicker marks it Displaced (final) while the newer session never connects. The invariant test checks only safety, not liveness. Restrict to Admitting/Admitted newer sessions, or document the choice.
- **BR-9** [Minor] `plan-revision-drift` M2 revision does not reconcile all Core-concepts rows and Task test bullets with the tree
  2nd instance of plan-revision-drift. Rule: at each milestone close the Revisions entry sweeps every Core-concepts row and every Task test bullet of that milestone against the tree. Instances: ReconnectBackoff lives in session_protocol.go, not backoff.go; Task 2.3's helper-subprocess kill test was not written; Task 2.6's wrapper-level real-broker idle/restart test was delivered at SessionClient level only, so the startPeerRuntime registry-socket wiring is untested.
- **BR-10** [Minor] `send-target-slot-match` Exact-send SendTargeted posts from an untracked goroutine and matches the target by raw slot string
  message_service.go:542; a send addressed by family alias would not wake a dormant session. Each send also spawns a goroutine that s.workers does not track; it ends at shutdown.

## Round 5 — 2026-10-01T15:32:04-07:00 (claude) — passed

### Disposed

- BR-7 — addressed — ConnectFailed is fed back via execute's return and stepped before new input; registry and service tests go red without it. Still no log line for a refused Connect.
- BR-8 — not-addressed — registry.go:266 newerSessionForSlot still counts any non-Displaced newer session.
- BR-9 — not-addressed — No new plan Revisions entry this round; backoff.go still named, 2.3/2.6 gaps unrecorded.
- BR-10 — not-addressed — message_service.go:542 unchanged.

### Raised

- **BR-11** [Minor] `artifact-removal-path` Broker actor tombstones are never evicted, so after 128 bindings per Couch lifetime new sessions go dormant
  2nd finding in family artifact-removal-path. Rule: every bounded table a component writes must name its removal path at the writer, not just a cap. Here, a tombstone whose slot and repository have re-registered under a newer binding is superseded (the registry marks it Displaced, which is final), so evict it on that Register. Pre-existing at base broker.go:136, but M2 turns the failure into silent permanent dormancy where the old heartbeat kept retrying.

## Round 6 — 2026-10-01T17:20:04-07:00 (claude) — BLOCKED

### Raised

- **BR-12** [Important] `test-name-matches-assertion` Lost-receipt recovery test re-sends via the b.receipts short-circuit; the wire reserve→AlreadyCommitted adoption path is unpinned
  TestPeerLostReceiptRecoveredAfterBrokerRestart calls StatusContext first, which records the receipt, so the re-send returns early at broker.go:427 and never reaches the wrapper's reserve; peer_runtime.go:67-71, endpoint.go:456-457 and adoptRetained over RemoteEndpoint can be reverted with the test green. 2nd in family. Rule: each test added at this boundary needs an assertion that goes red when its named mechanism is removed. Mutation-check all new M3 tests, and add a fresh-broker reused-ID send plus an in-flight retained receipt case.
- **BR-13** [Important] `untrusted-receipt-provenance` StatusContext records a wrapper-supplied receipt without checking the answering holder is its recipient
  broker.go:239 accepts a receipt if From or To equals the caller; it never checks r.Message.To equals the answering actor's binding (or ID == id for non-Remote holders). Any connected wrapper can plant an outcome for another slot's message. adoptRetained already checks To == a.binding. Carry the binding with each holder, require To == binding and ID == id, and test a forged answer.
- **BR-14** [Minor] `plan-revision-drift` Chunk 3 plan lines are unticked; lost-receipt service test, redraw-bytes experiment and lesson are neither delivered nor revised
  3rd in family. Rule: before milestone-close, every Plan line in the closing milestone is ticked or has a Revisions disposition. Sweep all of Chunk 3, not only the named items.

## Round 7 — 2026-10-01T17:23:40-07:00 (claude) — passed

### Disposed

- BR-12 — addressed — Fresh-broker re-send reaches reserve then wire already-committed then adoptRetained; reverting the peer_runtime.go conversion or the endpoint.go rebuild turns the test red (verified in a scratch copy). In-flight is covered only through the shared recordRecoveredLocked via the StatusContext test.
- BR-13 — addressed — broker.go:658 requires ID == id and To == the answering binding; the forged-answer case in broker_recovery_test.go fails when that check is removed (verified).
- BR-14 — not-addressed — Plan file untouched in c0cf69af; all Chunk 3 lines still unticked; no Revisions disposition for TestLifecycleLostReceiptAfterBrokerRestartNoDuplicate or the redraw-bytes rerun. Only the lesson item is now delivered.

### Raised

- **BR-15** [Minor] `test-name-matches-assertion` TestPeerLostReceiptRecoveredAfterBrokerRestart final check (current ID == m.ID) can never fail
  3rd in family. A duplicate delivery with the same ID leaves current.Message.ID == m.ID, so the assertion cannot catch it; the test catches the bug only through Send's error. Rule (already in lessons.md): every assertion must go red when the mechanism its message names is removed. Assert an enqueue or commit counter instead, or delete the line.

## Open findings

- **BR-6** [Minor] `artifact-removal-path` messageidle trace TSV grows unbounded while armed; only arm truncates it
- **BR-8** [Minor] `registry-liveness-unpinned` newerSessionForSlot lets a never-admitted newer session permanently displace a working older one
- **BR-9** [Minor] `plan-revision-drift` M2 revision does not reconcile all Core-concepts rows and Task test bullets with the tree
- **BR-10** [Minor] `send-target-slot-match` Exact-send SendTargeted posts from an untracked goroutine and matches the target by raw slot string
- **BR-11** [Minor] `artifact-removal-path` Broker actor tombstones are never evicted, so after 128 bindings per Couch lifetime new sessions go dormant
- **BR-14** [Minor] `plan-revision-drift` Chunk 3 plan lines are unticked; lost-receipt service test, redraw-bytes experiment and lesson are neither delivered nor revised
- **BR-15** [Minor] `test-name-matches-assertion` TestPeerLostReceiptRecoveredAfterBrokerRestart final check (current ID == m.ID) can never fail
