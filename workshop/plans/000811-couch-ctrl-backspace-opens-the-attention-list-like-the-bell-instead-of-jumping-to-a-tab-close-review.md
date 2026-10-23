# Boundary Review — pair#811 (whole-issue close)

| field | value |
|-------|-------|
| issue | 811 — Couch: Ctrl+Return toggles the notification list (bell); Ctrl+Backspace stays previous |
| repo | pair |
| issue file | workshop/issues/000811-couch-ctrl-backspace-opens-the-attention-list-like-the-bell-instead-of-jumping-to-a-tab.md |
| boundary | whole-issue close |
| milestone | — |
| window | 454b965ade23bb9d2fae30a1c75693c0e7ff1a10..18da8382860fd1cb4e545f0873ad4e2fedea4fe7 |
| command | sdlc close --issue 811 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-23T09:41:29-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The implementation satisfies the requested toggle behavior and preserves previous-tab navigation. Both affected packages and focused race tests pass. One documentation correction remains: the recommended keyboard fallback also requires enhanced terminal encoding.

1. **Strengths**

   - Ctrl+Return reuses `ActionAttention` and `toggleAttention`; the obsolete jump path is removed.
   - Production-input tests cover all three encodings from actor and switcher focus, checking navigation, acknowledgements, selection and history.
   - Physical-key tests verify toggling after child terminal-mode changes and confirm ordinary Return still reaches the original child.
   - README and atlas explain the removed actor jump and switcher modified-Return behavior.

2. **Critical findings**

   None.

3. **Important findings**

   One incorrect fallback recommendation, enumerated below.

```findings
findings:
  - id: new
    severity: Important
    family: fallback-matches-terminal-capabilities
    title: |
      Unsupported-terminal guidance recommends another enhanced-only shortcut
    detail: |
      README.md:1427-1429 and atlas/couch.md:3203-3204 recommend Ctrl+Alt+a on terminals without enhanced keys. However, cmd/internal/couchkeys/couchkeys.go:98-101 explicitly declares that shortcut enhanced-only, with no legacy encoding; terminal/input.go:351-357 preserves its legacy bytes rather than converting them into the declared chord. Both documentation passages therefore recommend a fallback that cannot work in the stated environment. These are the two explicit instances in this window; also clarify the ambiguous “bell/attention shortcut” fallback comment at couchkeys.go:31. Recommend clicking the bell on unsupported terminals and qualify Ctrl+Alt+a as requiring enhanced encoding.
```

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed:

   - Complete `couchkeys` and `couchtty` packages.
   - Focused race tests covering Ctrl+Return, physical keys, attention and previous-thread navigation.
   - `make test-affected-inventory`.
   - Pinned-range `git diff --check`.

   No source files were changed during review.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared declaration and existing toggle handler remain authoritative.
   - **ARCH-PURE — pass:** recognition remains separate from the thin console routing layer; integration tests use injected terminal/child fakes.
   - **ARCH-PURPOSE — pass:** actor and switcher toggling, retained encodings, item selection and previous-tab behavior are delivered.
   - **ARCH-EVOLVE — pass:** no persisted formats change; removed enums are internal.

7. **Plan revision recommendations**

   None. Correcting the fallback passages brings documentation into line with the existing plan.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"8ff7d4a9-744e-4cbe-8515-ee33fb009d45","repository":"github.com/xianxu/pair","issue_id":"000811","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/d9306bb642ebaf1b","reviewed_head":"18da8382860fd1cb4e545f0873ad4e2fedea4fe7","slot":"pair:3","requested_at":"2026-10-23T16:39:15.079344Z"},"phase":"finished","pid":64877,"started_at":"2026-10-23T16:39:15.220533Z","finished_at":"2026-10-23T16:41:26.859414Z","session_id":"01a169a1-85e2-7830-9621-b5cc85869590","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.355836209s
head: "18da8382860fd1cb4e545f0873ad4e2fedea4fe7"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
