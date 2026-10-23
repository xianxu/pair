# Boundary Review — pair#840 (whole-issue close)

| field | value |
|-------|-------|
| issue | 840 — Switcher: don't dim inactive hubs (dimming there means parked or asleep); tab bar keeps it |
| repo | pair |
| issue file | workshop/issues/000840-switcher-don-t-dim-inactive-hubs-dimming-there-means-parked-or-asleep-tab-bar.md |
| boundary | whole-issue close |
| milestone | — |
| window | a14c4a341a051b3819bef3ecfb95ec8ab5123007..3096420bbe9cb390c2903bd5053f006b973d4224 |
| command | sdlc close --issue 840 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-23T12:18:48-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned change fulfills #840: switcher rendering no longer depends on the active owner, while lifecycle styling and tab-bar dimming remain intact. The regression exercises both owners, hub and worker selection, parked/live rows, palette fallbacks, and click extents. No blocking findings.

Done-when: current — the revisions clarify acceptance and release sequencing without changing the two required rendering outcomes.

```findings
{}
```

1. **Strengths**
   - Removes the complete switcher-only mechanism without changing shared styling: `cmd/internal/couchtty/menu_render.go:528`.
   - Compares parsed terminal cells and click extents across owner focus changes: `cmd/internal/couchtty/hub_section_test.go:306`.
   - Corrects both atlas guidance and the emitted Couch skill while preserving the tab-bar contract.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**
   - Targeted hub, switcher, inactive-owner, idle, and age tests passed.
   - Couch skill tests and `make test-affected-inventory` passed.
   - Existing tests cover idle fading, blocked/attention precedence, hub underline/selection composition, and both tab styles.
   - `git diff --check` passed. The implementer’s recorded RED run was not independently reproduced.

6. **Architectural notes**
   - **ARCH-DRY: pass.** Deletes redundant presentation logic and reuses existing cell parsing.
   - **ARCH-PURE: pass.** Rendering remains deterministic with explicit state, palette, dimensions, and time inputs.
   - **ARCH-PURPOSE: pass.** Removes owner-relative dimming for both hub and worker rows; the separate tab-bar path remains unchanged.
   - **ARCH-EVOLVE: pass.** No persisted format or read-modify-write behavior changes.
   - Docs gate passes: existing atlas guidance is updated; no new command, flag, keybinding, or configuration surface requires a README addition.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"a4d5dbff-f3e0-4fa5-b194-0bd70d724a10","repository":"github.com/xianxu/pair","issue_id":"000840","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/d2f0a146b90d0a9d","reviewed_head":"3096420bbe9cb390c2903bd5053f006b973d4224","slot":"pair:9","requested_at":"2026-10-23T19:17:19.332984Z"},"phase":"finished","pid":98284,"started_at":"2026-10-23T19:17:19.584768Z","finished_at":"2026-10-23T19:18:46.911078Z","session_id":"01a16a32-3f74-70c1-8525-2f89ff274c08","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.883689958s
head: "3096420bbe9cb390c2903bd5053f006b973d4224"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
