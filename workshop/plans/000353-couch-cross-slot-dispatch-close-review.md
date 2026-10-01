# Boundary Review — pair#353 (whole-issue close)

| field | value |
|-------|-------|
| issue | 353 — Live cross-slot dispatch between couch slots |
| repo | pair |
| issue file | workshop/issues/000353-couch-cross-slot-dispatch.md |
| boundary | whole-issue close |
| milestone | — |
| window | 91d843c373c5be00f838bec31c3c1cb2ed8b98f9..8ed8c245d51bce031446246c7caa3bf8a3ef483f |
| command | sdlc close --issue 353 |
| reviewer | codex |
| timestamp | 2026-09-30T21:27:35-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation delivers the main messaging path, and all five affected package suites passed. However, deterministic scratch tests reproduced mixed automatic input and failures within the declared message/inventory limits. Crashed wrapper sockets also lack cleanup.

1. **Strengths**

   - Exact incarnation checks and canonical receipts protect against slot replacement and altered message bodies.
   - Delivery reducers preserve uncertainty, prohibit automatic replay, and ignore late events after cancellation.
   - Stateful endpoint tests, captured harness fixtures, README, and atlas cover the new surface.

2. **Critical findings**

   - **Automatic input ownership ends too early.** [peer_delivery.go:199](/Users/xianxu/workspace/worktree/pair-slot2/pair/cmd/internal/wrapcmd/peer_delivery.go:199) and [orientation.go:111](/Users/xianxu/workspace/worktree/pair-slot2/pair/cmd/internal/wrapcmd/orientation.go:111) share a mutex around individual callbacks, but neither owns the composer across paste/render/submit. A deterministic test produced: orientation paste → peer paste before repaint → orientation submits both. Introduce shared transaction ownership and test both arrival orders. **ARCH-ORDER, ARCH-DRY.**
   - **Valid payloads exceed the transport envelope.** [transport.go:199](/Users/xianxu/workspace/worktree/pair-slot2/pair/cmd/internal/couchmessage/transport.go:199) marshals JSON before enforcing a 32 KiB limit. An accepted 8,192-byte `<` body becomes **49,285 bytes**; 128 ordinary actor records become **39,016 bytes**. Both failed scratch tests. Oversized responses also become EOF because the writer error is discarded. Align wire bounds with supported payloads, bindings, receipts, and inventories; paginate discovery if necessary. **ARCH-CONSTRAINTS.**

3. **Important findings**

   - **Crashed wrappers leave permanent socket residue.** [peer_runtime.go:128](/Users/xianxu/workspace/worktree/pair-slot2/pair/cmd/internal/wrapcmd/peer_runtime.go:128) creates an incarnation-specific socket. Only graceful teardown removes it; subsequent launches use different hashes, and supervisor cleanup removes only its broker socket. Add proven-dead-owner cleanup, with crash tests preserving live and replacement sockets. **ARCH-FUNERAL.**

4. **Minor findings**

   None.

5. **Test coverage notes**

   Uncached suites passed for `couchmessage`, `couchcmd`, `couchtty`, `wrapcmd`, and `launcher`; focused race tests passed. Temporary Go-overlay regressions failed for both critical findings without modifying the repository. Diff whitespace checks pass excluding raw terminal captures, whose preserved bytes trigger whitespace diagnostics. Live harness qualification was inspected, not rerun.

6. **Architectural notes**

   - **ARCH-DRY — flag:** automatic writers need one shared ownership authority.
   - **ARCH-PURE — pass:** routing, validation, and delivery transitions have separate deterministic cores.
   - **ARCH-PURPOSE — pass:** the amended live messaging scope is implemented; reduced operator smoke scope is explicitly recorded.
   - **ARCH-MOCK — pass:** stateful endpoints and real socket integration exercise production seams.
   - **ARCH-CONSTRAINTS — flag:** encoded frames cannot accommodate declared bounds.
   - **ARCH-SECURE — pass:** strict decoding, private sockets, incarnation checks, and canonical-content verification.
   - **ARCH-ORDER — flag:** callback serialization does not prevent overlapping delivery transactions.
   - **ARCH-FUNERAL — flag:** abnormal wrapper exit leaves uncollected sockets.

7. **Plan revision recommendations**

   Append revisions specifying shared automatic-input ownership across the complete transaction, wire-size budgets for every request/response family, and crash-safe socket cleanup ownership. Add corresponding regression cases to the execution ledger.

```findings
findings:
  - id: new
    severity: Critical
    family: automatic-input-transaction-ownership
    title: |
      Orientation and peer delivery can paste into the same composer and submit combined text
    detail: |
      cmd/internal/wrapcmd/peer_delivery.go:199 and orientation.go:111 serialize callbacks without retaining ownership across paste/render/submit. A deterministic scratch regression produced orientation paste, peer paste before repaint, then orientation submission of both. Add shared transaction ownership and controlled tests for both arrival orders. ARCH-ORDER, ARCH-DRY.
  - id: new
    severity: Critical
    family: wire-capacity-matches-domain-bounds
    title: |
      The frame limit rejects valid message bodies and supported actor inventories
    detail: |
      cmd/internal/couchmessage/transport.go:199-204 uses JSON encoding with a 32768-byte cap. Scratch regressions measured 49285 bytes for a valid 8192-byte less-than-character body and 39016 bytes for 128 ordinary actor records. Response write errors at line 180 become EOF. Align serialized bounds across sends, endpoint commits, receipts and discovery; test maximum payloads and inventories. ARCH-CONSTRAINTS.
  - id: new
    severity: Important
    family: runtime-artifact-crash-cleanup
    title: |
      Incarnation-specific wrapper sockets have no cleanup after process crashes
    detail: |
      cmd/internal/wrapcmd/peer_runtime.go:128-138 creates a new socket per incarnation; couchmessage/transport.go:152 removes it only during graceful teardown. Supervisor startup cleans only the broker socket, and reconciliation merely disconnects actors. Add proven-dead-owner cleanup and crash regressions preserving live and replacement handles. ARCH-FUNERAL.
```

---

## Re-review — 2026-09-30T21:36:44-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 353 — Live cross-slot dispatch between couch slots |
| repo | pair |
| issue file | workshop/issues/000353-couch-cross-slot-dispatch.md |
| boundary | whole-issue close |
| milestone | — |
| window | 91d843c373c5be00f838bec31c3c1cb2ed8b98f9..cbfaa080d71250a5c792cb9c0f824c3f5562fbf5 |
| command | sdlc close --issue 353 |
| reviewer | codex |
| timestamp | 2026-09-30T21:36:44-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

All three prior findings are addressed, and all five affected package suites passed. I found no additional runtime defect. One Core concepts mapping remains inaccurate; your explicit table-consistency rule makes this blocking, although the correction is documentation-only.

1. **Strengths**

   - Shared automatic-input ownership protects both orientation-first and peer-first delivery, including cancellation and repaint.
   - Socket tests exercise maximum escaped messages, commits, receipts, and 128-actor discovery.
   - Crash-cleanup tests kill a real listener process and verify preservation of live owners and replacement sockets.
   - README and atlas document the commands, qualification limits, uncertainty, and lifecycle.

2. **Critical findings**

   - `workshop/plans/000353-couch-cross-slot-dispatch-plan.md:123` names a new transport `Client`, but no such entity exists. The implemented integration boundary is `Call` at `cmd/internal/couchmessage/transport.go:212`. Append a `## Revisions` entry explicitly superseding this mapping with `Server`/`Call`; no runtime change is needed. Classified Critical solely under the requested Core concepts consistency rule.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Uncached tests passed for `couchmessage`, `couchcmd`, `couchtty`, `wrapcmd`, and `launcher`. Prior-finding regressions exercise the corrected production paths and contain assertions that the previous implementations violate. No rollback mutation or live-agent smoke was performed during this read-only review. Raw terminal captures produce expected whitespace-check diagnostics.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared automatic-input arbitration.
   - **ARCH-PURE — pass:** deterministic routing, validation, composer classification, and reducers.
   - **ARCH-PURPOSE — pass:** delivery implements the amended free-text coordination contract.
   - **ARCH-MOCK — pass:** stateful endpoints, controlled terminal fixtures, and isolated socket tests.
   - **ARCH-CONSTRAINTS — pass:** bounded admission, inventory, receipts, frames, and deadlines.
   - **ARCH-SECURE — pass:** strict framing, exact incarnation checks, and conservative ownership cleanup.
   - **ARCH-ORDER — pass:** transaction ownership and controlled interleaving regressions.
   - **ARCH-FUNERAL — pass:** bounded ephemeral state and proven-dead socket collection.

7. **Plan revision recommendation**

   Append: “Transport integration uses `Server` and the stateless `Call` function in `transport.go`; this supersedes the proposed `Client` entity in the Core concepts table.”

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Shared transaction ownership and both automatic-input regression tests cover competing arrival orders, cancellation, submission, and fresh-empty-repaint release.
  - id: BR-2
    disposition: addressed
    note: |
      Derived frame bounds and transport_bounds_test.go exercise maximum messages, commits, receipts, and actor inventories through sockets; overflow now reports uncertainty explicitly.
  - id: BR-3
    disposition: addressed
    note: |
      Startup collects proven-dead PID-owned sockets; real process-crash and controlled replacement tests verify cleanup while preserving live, uncertain, and replacement owners.
findings:
  - id: new
    severity: Critical
    family: core-concept-mappings-match-implementation
    title: |
      Core concepts names a transport Client that does not exist
    detail: |
      workshop/plans/000353-couch-cross-slot-dispatch-plan.md:123 lists Client, but transport.go:212 implements Call instead. Append a revision superseding the mapping with Server/Call. This documentation-only discrepancy is Critical under the explicitly requested Core concepts consistency rule.
```

---

## Re-review — 2026-09-30T21:59:35-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 353 — Live cross-slot dispatch between couch slots |
| repo | pair |
| issue file | workshop/issues/000353-couch-cross-slot-dispatch.md |
| boundary | whole-issue close |
| milestone | — |
| window | 91d843c373c5be00f838bec31c3c1cb2ed8b98f9..51fb8e66e2d207fff4678c7e5254e798e0e12421 |
| command | sdlc close --issue 353 |
| reviewer | codex |
| timestamp | 2026-09-30T21:59:35-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned range matches the approved, amended contract. BR-4 is addressed by an explicit plan revision backed by the implemented symbols. No new blocking findings emerged. All five affected package suites passed.

```findings
dispose:
  - id: BR-4
    disposition: addressed
    note: |
      Plan lines 499–506 explicitly supersede Server/Client with Server/Call and reconcile wrapper adapter mappings. transport.go:46 defines Server; transport.go:212 defines Call. This is a documentation-only correction.
  - id: BR-1
    disposition: addressed
    note: |
      Shared automatic-input ownership remains enforced; passing regressions cover both arrival orders, cancellation before repaint, and fresh empty repaint after submission.
  - id: BR-2
    disposition: addressed
    note: |
      Passing socket regressions exercise maximum escaped message bodies, endpoint commits, receipts, and complete 128-actor responses against derived frame bounds.
  - id: BR-3
    disposition: addressed
    note: |
      Startup collection removes proven-dead socket owners while preserving live or uncertain owners and replacement inodes; crash-cleanup regressions pass.
```

1. **Strengths**

   - Admission separates routing from synchronized reservation and checks observation generations before committing.
   - Delivery preserves uncertainty and never automatically retries potentially completed PTY writes.
   - Stateful endpoint tests, actual Unix sockets, and captured composer fixtures exercise meaningful production boundaries.
   - README and atlas document commands, qualification limits, receipt semantics, and runtime cleanup.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Passed `go test -count=1` for `couchmessage`, `couchcmd`, `couchtty`, `wrapcmd`, and `launcher`. Diff whitespace checks pass excluding raw terminal captures; those captures retain terminal whitespace. Live harness conformance and human acceptance were not rerun during this read-only review. The plan explicitly records the accepted arithmetic smoke scope without claiming an actual issue dispatch.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared identity parsing, composer recognition, and automatic-input arbitration.
   - **ARCH-PURE — pass:** routing, validation, and delivery transitions remain separate from socket and PTY effects.
   - **ARCH-PURPOSE — pass:** the CLI-to-broker-to-wrapper path delivers the amended live coordination contract.
   - **ARCH-MOCK — pass:** stateful doubles share production seams; socket tests and version-specific conformance cover external boundaries.
   - **ARCH-CONSTRAINTS — pass:** bounded actors, receipts, frames, handlers, deadlines, and pending-only polling.
   - **ARCH-SECURE — pass:** private sockets, strict decoding, exact incarnation checks, and canonical receipt verification.
   - **ARCH-ORDER — pass:** explicit delivery transitions, generation checks, and controlled interference tests.
   - **ARCH-FUNERAL — pass:** bounded ephemeral receipts, joined workers, graceful socket removal, and dead-owner collection.

7. **Plan revision recommendations:** None. The appended BR-4 revision supplies the requested final source mapping.

---

## Re-review — 2026-09-30T22:28:49-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 353 — Live cross-slot dispatch between couch slots |
| repo | pair |
| issue file | workshop/issues/000353-couch-cross-slot-dispatch.md |
| boundary | whole-issue close |
| milestone | — |
| window | 834078cedee432acecd4fac073e37768a171887d..db6a3ed04dfd04bca0131b2b8d84f65432197ad0 |
| command | sdlc close --issue 353 |
| reviewer | codex |
| timestamp | 2026-09-30T22:28:49-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation has strong delivery, lifecycle, and transport coverage, and the four prior findings remain addressed. One reproducible routing defect blocks shipping: excluding the sender before checking family ambiguity can dispatch work to a different repository sharing its name.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Shared automatic-input ownership spans paste/render/submit. Passing regressions exercise both writer arrival orders, cancellation, and fresh-empty-repaint release.
  - id: BR-2
    disposition: addressed
    note: |
      Derived frame bounds and passing real-socket tests cover escaped maximum bodies, bindings, receipts, and 128-actor discovery.
  - id: BR-3
    disposition: addressed
    note: |
      Startup cleanup has passing killed-process regressions and preserves live, uncertain, and replacement socket owners.
  - id: BR-4
    disposition: addressed
    note: |
      The final plan revision explicitly supersedes Server/Client with Server/Call and names the implemented automaticInputTransaction location; the referenced symbols exist.
findings:
  - id: new
    severity: Critical
    family: routing-ambiguity-before-eligibility-filtering
    title: |
      Sender exclusion hides repository-family ambiguity
    detail: |
      cmd/internal/couchmessage/broker.go:435 removes the sender before ResolveRecipient checks repository identities. With sender pair:0 in repository A and recipient pair:1 in repository B, sending to pair succeeds into B instead of returning ErrAmbiguous. An isolated regression reproduces this against the pinned HEAD. Preserve all connected actors for ambiguity detection, then exclude the sender only from recipient eligibility. Add a broker regression asserting refusal without reservation or delivery. ARCH-PURPOSE, ARCH-SECURE.
```

1. **Strengths**

   - Actor reducers centralize mailbox, allowance, and terminal-outcome transitions.
   - Monotonic human-submission observations prevent duplicate notifications from replenishing spent allowance.
   - Delivery commits once; subsequent polling verifies the exact envelope without replaying input.
   - README and atlas cover the new commands, qualification limits, receipt semantics, and cleanup.

2. **Critical findings**

   - [broker.go:435](/Users/xianxu/workspace/worktree/pair-slot2/pair/cmd/internal/couchmessage/broker.go:435): check ambiguity against the complete connected inventory before applying sender exclusion. The scratch regression returned `recipient=pair:1 repo=/other/.git err=<nil>` where `ErrAmbiguous` was required.

3. **Important findings**

   None.

4. **Minor findings**

   None.

5. **Test coverage notes**

   All five affected packages passed: `couchmessage`, `couchcmd`, `couchtty`, `wrapcmd`, and `launcher`. The full `couchmessage` race suite and focused automatic-input/peer race tests also passed.

   The added scratch-overlay routing regression fails against HEAD; repository files remain unchanged. Live harness conformance was not rerun. Whitespace checking passes excluding raw terminal captures, whose preserved bytes trigger trailing-whitespace diagnostics.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared identity parsing, composer recognition, and automatic-input ownership.
   - **ARCH-PURE — pass:** routing, validation, and delivery transitions have deterministic cores.
   - **ARCH-PURPOSE — flag:** ambiguous family addressing does not consistently refuse.
   - **ARCH-MOCK — pass:** stateful endpoints, real isolated sockets, and captured harness fixtures exercise production seams.
   - **ARCH-CONSTRAINTS — pass:** admission, delivery, handler, receipt, and wire bounds are explicit and tested.
   - **ARCH-SECURE — flag:** sender eligibility filtering can conceal conflicting repository identity.
   - **ARCH-ORDER — pass:** inspected paths preserve uncertain outcomes and reject late automatic submission.
   - **ARCH-FUNERAL — pass:** runtime sockets have crash cleanup; receipts and actor tombstones have bounded owner lifetimes.

7. **Plan revision recommendation**

   Append a dated `## Revisions` entry establishing the invariant: family ambiguity uses every connected actor; sender exclusion affects selection only. Record the broker-level regression proving refusal before any delivery effect.
