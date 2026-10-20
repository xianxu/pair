# Boundary Review — pair#732 (whole-issue close)

| field | value |
|-------|-------|
| issue | 732 — couch: tab bar and switcher order: by hub, the hub's own slots right after it, then repos alphabetically |
| repo | pair |
| issue file | workshop/issues/000732-couch-tab-bar-and-switcher-order-by-hub-the-hub-s-own-slots-right-after-it-then-repos-alphabetically.md |
| boundary | whole-issue close |
| milestone | — |
| window | 856876acd65a089426e4d1f62ced00444db347e4..a93fe02d46530fe7178f5702a15de7d21d908de3 |
| command | sdlc close --issue 732 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-20T11:31:51-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned range delivers the specified ordering through the existing shared projection. Both UI consumers are covered, ownership boundaries remain intact, and the package tests pass. No blocking findings.

```findings
{}
```

1. **Strengths**
   - `thread_presentation.go:184` promotes the hub’s repository while preserving numeric slot order and the alphabetical remainder.
   - `hub_order_test.go:43` exercises eight shuffled inventories across shared projection, switcher inventory, switcher rendering, and tab rendering.
   - `reserve.go:592` groups primary hubs with their slots while preserving numbered-hub behavior.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**
   - Ran `go test ./cmd/internal/couchtty -count=1`: passed in 42.357s.
   - Existing tests cover absent, missing, primary, and numbered hubs, plus ownership and narrow rendering.
   - `git diff --check` passed. The repository-wide suite reported in the tracker was not independently rerun.

6. **Architectural notes**
   - **ARCH-DRY — pass:** switcher navigation, switcher rendering, and tabs derive ordering from `PresentThreads`.
   - **ARCH-PURE — pass:** ordering remains deterministic and free of IO; UI integration uses the fake host.
   - **ARCH-PURPOSE — pass:** both requested consumers implement the full ordering rule.
   - **ARCH-EVOLVE — pass:** no persisted formats change.
   - Atlas documents the change. No new command, flag, keybinding, or configuration surface requires a README update.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"69d344b7-f38e-4125-9adc-83a9bbb810a8","repository":"github.com/xianxu/pair","issue_id":"000732","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/a1504a8d258da3f3","reviewed_head":"a93fe02d46530fe7178f5702a15de7d21d908de3","slot":"pair:1","requested_at":"2026-10-20T18:30:27.07825Z"},"phase":"finished","pid":52507,"started_at":"2026-10-20T18:30:27.172712Z","finished_at":"2026-10-20T18:31:40.301668Z","session_id":"01a15a94-3f5b-76c3-a390-0d6ac45df125","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.4787255s
head: "a93fe02d46530fe7178f5702a15de7d21d908de3"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000410
