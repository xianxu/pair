# Boundary Review — pair#625 (whole-issue close)

| field | value |
|-------|-------|
| issue | 625 — Couch tab fades busy slots driven only by peer messages (activity ignores their agent work) |
| repo | pair |
| issue file | workshop/issues/000625-couch-tab-fades-busy-slots-driven-only-by-peer-messages-activity-ignores-their-agent-work.md |
| boundary | whole-issue close |
| milestone | — |
| window | 318510900c81a7f8cf6cf538b694d962f8aecb2a..fbf3dde7424a33ca92a0559319066aebb1c23836 |
| command | sdlc close --issue 625 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-16T20:37:22-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned diff matches the revised scope: admitted wrapper work keeps peer-driven turns bright, completion timestamps age normally, and admission behavior remains unchanged. No blocking defects found. The live restart check remains explicitly pending under the documented approval.

Done-when: current — it covers peer-only activity, idle fading, unchanged admission, and the approved background-only deferral to #632.

```findings
{}
```

1. **Strengths**
   - `couchmessage/registry.go:212–222` bounds reconnect evidence and prevents settled refreshes from renewing activity.
   - `couchcmd/message_service.go:1311` reuses admitted-session evidence with exact scope/tag and agent matching.
   - `couchtty/console_activity.go:108–116` preserves positive wrapper activity when the file probe fails.
   - README and atlas document the behavior and background-only limitation.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage:** All four affected packages passed: `couchmessage`, `couchcmd`, `couchtty`, and `threadactivity`. Focused race tests also passed. Coverage includes busy/completed peer turns, silent waits, reconnect seeds, replacement/disconnect, legacy sessions, identity mismatches, and eventual fading. The full repository suite was not independently rerun.

6. **Architecture:** **ARCH-DRY: pass** — reuses existing liveness, scheduling, and shared rendering state. **ARCH-PURE: pass** — timestamp transitions remain deterministic registry logic; clock access and UI wiring stay at the boundary. **ARCH-PURPOSE: pass** — both display consumers receive the merged signal, with the approved background-only scope split documented consistently.

7. **Plan revisions:** None required.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"cad7f8e2-41f3-43ee-b6f0-bdf822fc5d5a","repository":"github.com/xianxu/pair","issue_id":"000625","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/21d73ec09014564f","reviewed_head":"fbf3dde7424a33ca92a0559319066aebb1c23836","slot":"pair:6","requested_at":"2026-10-17T03:35:21.772675Z"},"phase":"finished","pid":97143,"started_at":"2026-10-17T03:35:22.077206Z","finished_at":"2026-10-17T03:37:21.285254Z","session_id":"01a147ed-b23c-75a2-941c-784c12e5eed3","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.085438041s
head: "fbf3dde7424a33ca92a0559319066aebb1c23836"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
