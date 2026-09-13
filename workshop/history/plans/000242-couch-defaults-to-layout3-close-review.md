# Boundary Review — 000242-couch-defaults-to-layout3#242 (whole-issue close)

| field | value |
|-------|-------|
| issue | 242 — couch defaults to --layout3: every thread gets pair's right-hand terminal unless --layout2 is asked for |
| repo | 000242-couch-defaults-to-layout3 |
| issue file | workshop/issues/000242-couch-defaults-to-layout3.md |
| boundary | whole-issue close |
| milestone | — |
| window | 72635c772fe38ef4e845c3b0ce87d703b0e598f0..b7dbd223bf0d663763980675f0f17539de228ea2 |
| command | sdlc close --issue 242 |
| reviewer | codex |
| timestamp | 2026-09-13T16:10:18-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned range satisfies the issue’s Spec: flagless Couch selects layout3, explicit layout2 remains supported, legacy records retain layout2 provenance, and refusal messages explain both remedies. Help, README, and atlas agree. No blocking findings.

1. **Strengths**
   - Both default consumers use `DefaultLayout`: `cmd/internal/couchcore/couch.go:112` and `cmd/internal/couchcmd/cli.go:71`.
   - `layout_launch_test.go:12` checks exact child arguments and the persisted witness together.
   - `layout_guard_test.go:30` verifies legacy-session refusal, actionable remedies, and no child launch.
   - Warm reattachment preserves its existing witness and omits the layout flag.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**
   - Focused layout/projection tests passed.
   - Complete `cmd/couch`, `couchcmd`, and `couchcore` suites passed.
   - Pinned-range `git diff --check` passed.
   - Full repository `make test`, live conformance, and operator installation were not rerun. No mutation testing was performed.
   - Reviewed pinned tracker versions; excluded uncommitted tracker edits.

6. **Architectural notes**
   - **ARCH-DRY — pass:** both process-default consumers share one constant.
   - **ARCH-PURE — pass:** parsing, normalization, and remedy logic remain pure; constructor integration stays separate.
   - **ARCH-PURPOSE — pass:** constructor, CLI, launch arguments, witnesses, compatibility, and documentation are covered.
   - **ARCH-MOCK — pass:** existing injected stateful fakes and portable temporary stores are reused; conformance checks remain available.
   - **ARCH-CONSTRAINTS — pass:** constant selection introduces no additional IO or concurrency.
   - **ARCH-SECURE — pass:** legacy and unreadable persisted values retain distinct handling.
   - **ARCH-ORDER — pass:** startup refusal remains before launch; cold/warm witness behavior is preserved.
   - **ARCH-FUNERAL — pass:** no new durable artifact family or background task.

7. **Plan revision recommendations:** None. Core-concept locations and classifications match the implementation; remaining release steps are explicitly unchecked.

```findings
{}
```
