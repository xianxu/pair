# Boundary Review — pair#841 (whole-issue close)

| field | value |
|-------|-------|
| issue | 841 — Bell notifications: highlight slot (repo:N) and issue (repo#N) references in an accent color |
| repo | pair |
| issue file | workshop/issues/000841-bell-notifications-highlight-slot-repo-n-and-issue-repo-n-references-in-an.md |
| boundary | whole-issue close |
| milestone | — |
| window | 1ee96e9872ef195450d5c53f096f530ca7e95ee3..1f9fed7b419af84537105f82a99e9b764bf13776 |
| command | sdlc close --issue 841 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-23T17:07:12-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The change preserves plain-text copying and keeps reference detection separate from terminal rendering. Two matching defects block shipping: generated timestamps receive reference colors, and valid references beside paths or URLs lose their colors.

Done-when: current — it still covers the display-only behavior, punctuation/path boundaries, wrapping, and plain-text copy; the revision changes release sequencing only.

1. **Strengths**

   - Source detection happens before wrapping, preventing wrapped path fragments from becoming references.
   - Bell styling is opt-in; existing notes-menu rendering remains covered.
   - Clipboard coverage exercises Ctrl+Y and checks the actual plain-text payload.
   - `atlas/couch.md` documents the rendering change. No new command, keybinding, or configuration requires a README update.

2. **Critical findings**

```findings
findings:
  - id: new
    severity: Critical
    family: reference-highlighting-respects-source-role
    title: |
      Generated notification timestamps are highlighted as slot references
    detail: |
      cmd/internal/couchtty/console_attention.go:129 opts the entire composed row into matching, including the timestamp added at line 100. attention_references.go:12-20 accepts numeric repository names, so every HH:MM timestamp with minutes 10–59 receives the accent. This affects notify, prompt and question rows, with and without hover. Confirmed by running env TZ=Asia/Kolkata go test ./cmd/internal/couchtty -run '^TestAttentionReferenceCells/prose$' -count=1: all six assertions failed with an unexpected colored "05:36" prefix. Exclude generated timestamp metadata from reference spans, preserve legitimate numeric repository names, and use fixed timestamps covering minutes 09, 10 and 59 in regression tests. This violates the "and only those" contract (ARCH-PURPOSE).
  - id: new
    severity: Critical
    family: reference-exclusion-respects-token-boundaries
    title: |
      Path and URL exclusion suppresses separate punctuation-delimited references
    detail: |
      cmd/internal/couchtty/attention_references.go:32-36 rejects an entire whitespace chunk before separating punctuation-delimited tokens. Consequently [pair:3](https://example.test), [ariadne#575](https://example.test), and [#841](https://example.test) receive no reference accent even though their labels are outside the URL. The same guard suppresses separate references beside slash paths, backslash paths, or @-containing text; these are all instances of this single exclusion site. Apply exclusion to the actual path/URL token while retaining adjacent references, and add render regressions for all three reference forms alongside positive path/URL exclusions. Update the plan's whitespace-chunk rejection design accordingly (ARCH-PURPOSE).
```

3. **Important findings**

   None beyond the regression coverage required by the findings above.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Focused reference/notes tests passed, as did the attention/notes/toggle race subset and exhaustive artifact-source classification check. The timezone run exposed an existing wall-clock-dependent failure. Current punctuation fixtures do not combine references with neighboring paths or URLs.

6. **Architectural notes**

   - **ARCH-DRY: pass.** Reuses wrapping, palette, cells, and canonical workspace validation.
   - **ARCH-PURE: pass.** Detection and projection remain pure; clipboard IO uses the existing integration fixture.
   - **ARCH-PURPOSE: flag.** Both findings violate the promised matching boundaries.
   - **ARCH-EVOLVE: pass.** No persisted format changes.

7. **Plan revision recommendations**

   Add a `## Revisions` entry replacing whole-whitespace-chunk rejection with token-scoped exclusion and identifying generated timestamps as unaccented metadata. Include deterministic timestamp boundary tests and mixed reference/path/URL fixtures.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"9bdd736f-4f6c-4aa5-8b9e-03626fafa986","repository":"github.com/xianxu/pair","issue_id":"000841","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/66b08b03d8458ede","reviewed_head":"1f9fed7b419af84537105f82a99e9b764bf13776","slot":"pair:9","requested_at":"2026-10-24T00:04:35.505926Z"},"phase":"finished","pid":53162,"started_at":"2026-10-24T00:04:35.684297Z","finished_at":"2026-10-24T00:07:10.905189Z","session_id":"01a16b39-3f05-7ce1-ac89-41331f3c43f1","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.398375417s
head: "1f9fed7b419af84537105f82a99e9b764bf13776"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
