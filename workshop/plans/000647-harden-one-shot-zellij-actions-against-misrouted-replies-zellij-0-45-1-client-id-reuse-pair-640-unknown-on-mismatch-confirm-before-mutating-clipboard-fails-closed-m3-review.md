# Boundary Review — pair#647 (milestone M3)

| field | value |
|-------|-------|
| issue | 647 — Harden one-shot zellij actions against misrouted replies (zellij 0.45.1 client-ID reuse, pair#640): unknown on mismatch, confirm before mutating, clipboard fails closed |
| repo | pair |
| issue file | workshop/issues/000647-harden-one-shot-zellij-actions-against-misrouted-replies-zellij-0-45-1-client-id-reuse-pair-640-unknown-on-mismatch-confirm-before-mutating-clipboard-fails-closed.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | d435ae7ac0ec395eab68e0cf1b2aeb3e7319f545..384231f85ac25f0660b311dfff7697b9bfb8e334 |
| command | sdlc milestone-close --issue 647 --milestone M3 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-18T01:12:59-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

The pinned M3 range matches the operative plan revisions. Editor admission rejects unknown or conflicting evidence, guards subsequent effects, and preserves completed DOM/review progress. No blocking findings. The retained native artifact confirms refusal after partial progress; it remains inconclusive for clean-path qualification. The recorded full suite and operator check remain outstanding under the documented sequencing exception.

```findings
{}
```

1. **Strengths**

   - Shared strict Lua decoding rejects duplicate keys, missing decision fields, and inexact identities ([zellij_observation.lua](nvim/zellij_observation.lua)).
   - Contextual selection requires the invoking terminal and a unique target role in its tab; routing and poke tests inspect actual CLI effects and bytes ([pane_identity.lua](nvim/pane_identity.lua)).
   - DOM guards preserve `written` versus `indeterminate` progress; review visibility refusal preserves acknowledged activation and the ready target.
   - README and atlas updates document refusals, partial progress, caller ownership, and protocol limits.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Passed:
   
   - `make test-lua`
   - Full `zellijreply`, `zellijtest`, `artifactpath`, and `layoutcmd` package tests
   - Focused race tests for native oracles and semantic-key fake behavior
   - Direct pane-identity tests and pinned-range whitespace checks

   Inspected the retained native log and JSON directly: 31 bad raw responses across 830 requests; decision two retained its return record and issued no mutation after the unknown observation. The missing final PTY witness remains explicit. No live native test was rerun.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared contracts, selectors, context guards, and effect checks.
   - **ARCH-PURE — pass:** direct parser/selector tests; injected IO and clock seams.
   - **ARCH-PURPOSE — pass:** all four editor consumers are wired; production query inventory matches the atlas classifications.
   - **ARCH-MOCK — pass:** production CLI seams exercise the stateful fake and inspect persisted effects.
   - **ARCH-CONSTRAINTS — pass:** shared five-second admission bounds, earlier-parent limits, and bounded native sampling.
   - **ARCH-SECURE — pass:** strict external parsing, context fencing, and explicit target arguments.
   - **ARCH-ORDER — pass:** terminal refusal and completed-prefix preservation are exercised across controlled event sequences.
   - **ARCH-FUNERAL — pass:** no new durable production state; test owners clean temporary roots and join native workers/connections. Native diagnostic evidence is intentionally retained.

7. **Plan revision recommendations:** None required. Existing revisions accurately distinguish pure functions from integration glue and preserve the native qualification limits.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"bf7c1016-7419-489a-adb3-d7255f6e0f71","repository":"github.com/xianxu/pair","issue_id":"000647","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/c05175aba9e10c14","reviewed_head":"384231f85ac25f0660b311dfff7697b9bfb8e334","slot":"pair:1","requested_at":"2026-10-18T08:10:00.024581Z"},"phase":"finished","pid":49984,"started_at":"2026-10-18T08:10:00.148266Z","finished_at":"2026-10-18T08:12:39.3548Z","session_id":"01a14e0f-7d59-77e2-bfee-e979521ccf08","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.734649083s
head: "384231f85ac25f0660b311dfff7697b9bfb8e334"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
