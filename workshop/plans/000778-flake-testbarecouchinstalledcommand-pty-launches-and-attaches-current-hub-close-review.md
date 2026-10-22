# Boundary Review — pair#778 (whole-issue close)

| field | value |
|-------|-------|
| issue | 778 — flake: TestBareCouchInstalledCommand/pty_launches_and_attaches_current_hub |
| repo | pair |
| issue file | workshop/issues/000778-flake-testbarecouchinstalledcommand-pty-launches-and-attaches-current-hub.md |
| boundary | whole-issue close |
| milestone | — |
| window | d2a4bb5bbfc1a14c205c996af8b93cb5afacdbd7..d729188af8159560db1bfbf0d610fe0886eb5346 |
| command | sdlc close --issue 778 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T20:34:17-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The change fixes the fixture-name collision through both constructors while preserving production allocation, launch admission, and timeout behavior. Independent targeted checks pass, and the recorded full-suite receipt and matching raw-log hash confirm a pass on identical executable source.

Done-when: current — it covers both constructors, allocator continuity, PTY expectations, seed safety, lint enforcement, and recorded verification.

```findings
{}
```

1. **Strengths**
   - Both constructors reuse `testpair.SeedCouchIdentity`; no parallel seeding implementation.
   - Exclusive creation preserves existing authorities, with explicit seed bounds and permanent sequence exhaustion.
   - Constructor regressions exercise the real allocator; the installed PTY assertion derives its expected tag from allocated C.
   - Atlas documents fixture identity ownership and the syntax guard’s limitations.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None. Whitespace diagnostics are confined to retained evidence logs.

5. **Test coverage notes**
   - Passed affected packages: `testpair`, `testcouch`, `repolint`, and `artifactpath`.
   - Passed installed PTY, pipe refusal, constructor isolation, focused race checks, and `make test-affected-inventory`.
   - Verified recorded suite attempt `54cda992757ffb884f353a200db8f185`: exit 0, matching log SHA256. Only issue notes changed afterward.

6. **Architectural notes**
   - **ARCH-DRY: pass** — shared initializer serves both enumerated consumers.
   - **ARCH-PURE: pass** — seed calculation and AST matching are pure; filesystem work remains in fixture setup.
   - **ARCH-PURPOSE: pass** — both authority constructors are covered without weakening collision refusal.
   - **ARCH-EVOLVE: pass** — no production format change or existing-record rewrite is introduced. Atlas is updated; no new user-facing surface requires README changes.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"b39404fc-3a1a-43fd-b3ab-4a900c1fd7b8","repository":"github.com/xianxu/pair","issue_id":"000778","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/e35a90fffc6705a3","reviewed_head":"d729188af8159560db1bfbf0d610fe0886eb5346","slot":"pair:5","requested_at":"2026-10-22T03:32:29.228546Z"},"phase":"finished","pid":4418,"started_at":"2026-10-22T03:32:29.500704Z","finished_at":"2026-10-22T03:34:16.220547Z","session_id":"01a161aa-dcca-7703-aa3f-f9067a89a17b","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 4.731225333s
head: "d729188af8159560db1bfbf0d610fe0886eb5346"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
