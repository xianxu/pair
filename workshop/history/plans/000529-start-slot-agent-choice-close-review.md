# Boundary Review — pair#529 (whole-issue close)

| field | value |
|-------|-------|
| issue | 529 — Couch start-slot form has no selectable agents |
| repo | pair |
| issue file | workshop/issues/000529-start-slot-agent-choice.md |
| boundary | whole-issue close |
| milestone | — |
| window | 20ffe5ddc15c646470a13aad902f885e35555c28..b4b87fd72559f931db7e04402458e1c57f585b74 |
| command | sdlc close --issue 529 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-13T19:06:33-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The change satisfies every Done-when clause and both Plan items. Production initialization now supplies canonical agent choices while preserving resolver defaults and explicit selection. No blocking findings.

1. **Strengths**
   - `cmd/internal/couchtty/menu.go:349` reuses the canonical inventory, which returns an owned copy.
   - `menu_agent_inventory_test.go:14` covers start and add-slot, Left/Right, stale/current previews, rendering, and committed agent arguments.
   - `menu_agent_inventory_test.go:80` verifies canonical contents and independent ownership.
   - `atlas/couch.md:135` accurately documents the initialization and preview contract.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage**
   - Both complete packages passed: `couchtty` and `launcher`.
   - Independently confirmed regression failures with the constructor fix removed, followed by success at the pinned head.
   - Initial execution required generated runtime assets; generation and verification occurred in a temporary copy, subsequently removed. Checkout remains clean.
   - Full repository suite was not run.

6. **Architecture**
   - **ARCH-DRY — pass:** Start, add-slot, and switch-agent derive choices from `launcher.AgentInventory()`.
   - **ARCH-PURE — pass:** Initialization and reducer tests require no IO or mocks.
   - **ARCH-PURPOSE — pass:** Both affected entry points are repaired and tested.
   - README already documents agent selection at `README.md:1317`; this restores existing behavior without introducing a new interface.

7. **Plan revisions:** None. No Core concepts table or prior findings require disposition.

```findings
{}
```
