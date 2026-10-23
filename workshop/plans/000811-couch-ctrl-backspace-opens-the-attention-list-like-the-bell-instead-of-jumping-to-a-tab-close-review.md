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

---

## Re-review — 2026-10-23T09:46:10-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 811 — Couch: Ctrl+Return toggles the notification list (bell); Ctrl+Backspace stays previous |
| repo | pair |
| issue file | workshop/issues/000811-couch-ctrl-backspace-opens-the-attention-list-like-the-bell-instead-of-jumping-to-a-tab.md |
| boundary | whole-issue close |
| milestone | — |
| window | 454b965ade23bb9d2fae30a1c75693c0e7ff1a10..44fe1f589b0d2ced3bd9c07c9b5f20f9aea99616 |
| command | sdlc close --issue 811 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-23T09:46:10-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The implementation matches the Spec and Plan: Ctrl+Return toggles the existing notification list from actor and switcher focus, while ordinary Enter and Ctrl+Backspace retain their behavior. BR-1 is addressed across all three identified passages. No new findings.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      README.md:1427-1429 and atlas/couch.md:3202-3204 now recommend clicking the bell and explicitly qualify Ctrl+Alt+a as enhanced-only. couchkeys.go:29-31 likewise names the bell click. The pinned correction is prose-only and agrees with the enhanced-only binding at couchkeys.go:98-99 and terminal/input.go's preservation of legacy input.
```

1. **Strengths**

   - `couchkeys.go:80` reuses `ActionAttention` for all three Ctrl+Return encodings; the obsolete newest-page path is removed.
   - `terminal_input.go:111` correctly prioritizes the toggle before overlay decoding, preventing a second press from selecting an item.
   - `console_ctrl_return_test.go:11` exercises actor/switcher focus, navigation and acknowledgement preservation, repeated toggling, and Ctrl+Alt+a interoperability.
   - README and atlas document the changed behavior and supported fallback consistently.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Passed complete couchkeys and couchtty packages; focused race tests covering Ctrl+Return, physical keyboard generation, explicit events, and neighboring Return encodings; affected-inventory check; and diff whitespace check. Existing attention selection, mouse, expiration, and previous-thread tests remain covered by the complete package run.

6. **Architectural notes**

   - **ARCH-DRY — pass:** Both chords share the existing action and toggle handler.
   - **ARCH-PURE — pass:** Shortcut declarations remain pure; input routing stays a thin integration layer.
   - **ARCH-PURPOSE — pass:** Both focus contexts deliver the requested toggle, including closing the overlay, without preserving the superseded jump behavior.
   - **ARCH-EVOLVE — pass:** No persisted formats change; removed enums are internal.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"1f975bc7-1605-4def-a64c-72b75ee50f4c","repository":"github.com/xianxu/pair","issue_id":"000811","recipe":"small-diff-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/58811254edc82b3f","reviewed_head":"44fe1f589b0d2ced3bd9c07c9b5f20f9aea99616","slot":"pair:3","requested_at":"2026-10-23T16:44:07.326629Z"},"phase":"finished","pid":19412,"started_at":"2026-10-23T16:44:07.511548Z","finished_at":"2026-10-23T16:46:08.908733Z","session_id":"01a169a5-fb9a-7c00-8aab-678a2106170e","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.439227958s
head: "44fe1f589b0d2ced3bd9c07c9b5f20f9aea99616"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
