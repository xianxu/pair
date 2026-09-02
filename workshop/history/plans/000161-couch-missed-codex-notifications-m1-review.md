# Boundary Review — 000161-couch-missed-codex-notifications#161 (milestone M1)

| field | value |
|-------|-------|
| issue | 161 — Couch misses Codex completion notifications |
| repo | 000161-couch-missed-codex-notifications |
| issue file | workshop/issues/000161-couch-missed-codex-notifications.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 9a8e88528ab254e0c1d4765bf0a003b692506599..b18db3899df5ceea68efc561ee90a388f099d161 |
| command | sdlc milestone-close --issue 161 --milestone M1 |
| reviewer | codex |
| timestamp | 2026-09-01T14:52:38-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

M1 correctly broadens only Claude’s marker verb from ASCII letters to Unicode General Category `L`, while preserving the existing prefix, duration, colored-span ownership, duplicate suppression, and notification sink. Focused and package-wide tests pass. No blocking findings were found.

### 1. Strengths

- The implementation is narrowly scoped to `\p{L}+`; the surrounding grammar remains unchanged ([wrap.go](/Users/xianxu/workspace/worktree/pair/000161-couch-missed-codex-notifications/cmd/internal/wrapcmd/wrap.go:104)).
- Table tests cover ASCII, precomposed Unicode, another script, empty verbs, combining marks, digits, punctuation, and whitespace ([notification_marker_test.go](/Users/xianxu/workspace/worktree/pair/000161-couch-missed-codex-notifications/cmd/internal/wrapcmd/notification_marker_test.go:8)).
- The fuzz property independently compares matching behavior with `unicode.IsLetter`, directly pinning the required category boundary.
- The reported marker is tested through `updateAgentOutput → finalizeSpan → emitOuter`, including exact OSC output ([update_agent_output_test.go](/Users/xianxu/workspace/worktree/pair/000161-couch-missed-codex-notifications/cmd/internal/wrapcmd/update_agent_output_test.go:86)).
- Both acceptance tests necessarily fail with the base `[A-Za-z]+` expression because `Sautéed` cannot pass that class.

### 2. Critical findings

None.

### 3. Important findings

None.

### 4. Minor findings

None.

### 5. Test coverage notes

- `go test ./cmd/internal/wrapcmd -count=1`: passed.
- Focused grammar and production-path tests: passed.
- `git diff --check`: passed.
- The broader `go test ./...` progressed successfully through relevant packages but could not complete because generated `cmd/internal/runtimebundle/assets/runtime/files` content is absent from this checkout. This is unrelated to the reviewed range.
- The pure grammar tests perform no IO. The production integration regression uses injected output and temporary filesystem state appropriately.

### 6. Architectural notes for upcoming work

- `ARCH-DRY`: Pass. The existing matcher and shared `emitOuter` sink are reused; no parallel notification logic was introduced.
- `ARCH-PURE`: Pass. Grammar behavior remains declarative and directly unit-tested; IO stays at the existing delivery boundary.
- `ARCH-PURPOSE`: Pass for M1. All first-slice acceptance criteria are delivered, while Codex lifecycle work remains explicitly assigned to M2–M4.
- `ARCH-MOCK`: Pass/N/A for this slice. No new external dependency or stateful interaction was added.
- `ARCH-CONSTRAINTS`: Pass. The regex change adds no concurrency, fan-out, blocking work, or material resource expansion on the terminal path.
- Atlas and README updates are not required: M1 introduces no new architectural or user-operated surface, and the repository contains no stale documentation of the ASCII-only grammar.

### 7. Plan revision recommendations

None. The Core concepts row for the modified Claude grammar exists at the stated path and matches the implementation.

```findings
{}
```
