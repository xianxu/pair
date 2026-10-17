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
