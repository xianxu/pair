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
