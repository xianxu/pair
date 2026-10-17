# Boundary Review — pair#619 (whole-issue close)

| field | value |
|-------|-------|
| issue | 619 — Couch key help: add Alt+Space, parley's attention finder (parley.nvim#354) |
| repo | pair |
| issue file | workshop/issues/000619-couch-key-help-add-alt-u-parley-s-attention-finder-parley-nvim-354.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4bac247c5738e44c0c5d6d4eb4a6511800a5bdee..b7311c1dedc544cd8261b1b2e1fbc034cbf0b7c9 |
| command | sdlc close --issue 619 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-16T18:36:08-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Both attention-finder keys are correctly documented under Parley editor without installing Pair handlers. All four relevant package suites passed. One Important documentation gap remains: README omits both keys and retains a claim contradicted by the new metadata source.

```findings
findings:
  - id: new
    severity: Important
    kind: defect
    family: user-facing-surface-documentation
    title: |
      README update appears missing for the Parley attention-finder bindings
    detail: |
      cmd/internal/keyhelp/catalog.go:95–96 adds Alt+Space and Ctrl+Y followed by Space, but README.md is unchanged. The complete family in this window is both missing keybinding entries and README.md:146–149's now-inaccurate claim that every help row derives from an implementing keymap or chord. Add both keys with Parley-local scope and qualify that claim to distinguish curated companion-editor metadata.
```

1. **Strengths**
   - Explicit `SourceParley` and `ContextParley` preserve source ownership without fabricating executable bindings.
   - The new regression checks both standalone and hosted help, exact descriptions, uniqueness, zero Pair chords, and absence of obsolete Alt+u.
   - Atlas documents the new metadata source and scope accurately at `atlas/architecture.md:312`.

2. **Critical findings:** None.

3. **Important findings:** README gap detailed above.

4. **Minor findings:** None.

5. **Test coverage:** Passed `go test ./cmd/internal/keyhelp ./cmd/internal/keyscmd ./cmd/internal/couchkeys ./cmd/internal/couchtty`. New pass-through tests cover legacy and Kitty encodings of both bindings. Zellij configuration inspection confirms neither key is bound. `git diff --check` passed.

6. **Architecture**
   - **ARCH-DRY — pass:** Both keys share one description constant and reuse existing rendering.
   - **ARCH-PURE — pass:** Metadata resolution adds no IO; source access remains behind `SourceReader`.
   - **ARCH-PURPOSE — pass:** Both bindings reach the shared help pipeline, including hosted rendering, with Parley-local scope.

7. **Plan revisions:** None required; complete the README correction before crossing the boundary.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"909b6f67-f6be-4e43-a489-9656f8aa3a67","repository":"github.com/xianxu/pair","issue_id":"000619","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/8a5d0659d75c2332","reviewed_head":"b7311c1dedc544cd8261b1b2e1fbc034cbf0b7c9","slot":"pair:7","requested_at":"2026-10-17T01:34:45.233845Z"},"phase":"finished","pid":93270,"started_at":"2026-10-17T01:34:45.530576Z","finished_at":"2026-10-17T01:36:07.76835Z","session_id":"01a1477f-46a8-7590-9045-b0ee74427437","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.48565525s
head: "b7311c1dedc544cd8261b1b2e1fbc034cbf0b7c9"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000411

---

## Re-review — 2026-10-16T18:39:40-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 619 — Couch key help: add Alt+Space, parley's attention finder (parley.nvim#354) |
| repo | pair |
| issue file | workshop/issues/000619-couch-key-help-add-alt-u-parley-s-attention-finder-parley-nvim-354.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4bac247c5738e44c0c5d6d4eb4a6511800a5bdee..3f5f7fed4f03aa2829be24ae339cb473260bf000 |
| command | sdlc close --issue 619 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-16T18:39:40-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The implementation fulfills the revised requirements: both attention-finder bindings appear uniquely under Parley editor, without adding Pair handlers. BR-1’s README omissions and inaccurate derivation claim are corrected. No new findings.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      README.md:147–150 now distinguishes curated companion-editor metadata from implementing keymaps; lines 155–156 document both Parley-local shortcuts. These passages match catalog.go:43–47 and 95–96, and sections.go's SourceParley resolution. This is a prose-only correction.
```

1. **Strengths**
   - Explicit `SourceParley` and `ContextParley` preserve ownership and scope.
   - The regression checks both standalone and hosted help, exact descriptions, uniqueness, zero chords, and absence of obsolete Alt+u.
   - README and atlas both document the new surface accurately.

2. **Critical findings:** None.

3. **Important findings:** None remaining.

4. **Minor findings:** None.

5. **Test coverage:** Keyhelp, keyscmd, and couchkeys suites passed. The couchtty suite initially failed in unchanged `TestWorkbenchPerformanceIdleUITick`; that test passed in isolation alongside both relevant pass-through/routing tests. Zellij configuration binds neither shortcut. `git diff --check` passed.

6. **Architecture**
   - **ARCH-DRY — pass:** Both keys share one description and existing rendering.
   - **ARCH-PURE — pass:** Metadata resolution introduces no IO; source access remains injected.
   - **ARCH-PURPOSE — pass:** Both bindings reach standalone and Couch-presented help with Parley-local scope.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"ecc5ffe0-efcc-4de4-9c71-32858dd4689c","repository":"github.com/xianxu/pair","issue_id":"000619","recipe":"small-diff-review","round":2,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/5fabcfbca16e215a","reviewed_head":"3f5f7fed4f03aa2829be24ae339cb473260bf000","slot":"pair:7","requested_at":"2026-10-17T01:37:17.755325Z"},"phase":"finished","pid":44559,"started_at":"2026-10-17T01:37:17.898522Z","finished_at":"2026-10-17T01:39:30.431374Z","session_id":"01a14781-9960-7871-8ade-430a4a8c28fa","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.642831792s
head: "3f5f7fed4f03aa2829be24ae339cb473260bf000"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
