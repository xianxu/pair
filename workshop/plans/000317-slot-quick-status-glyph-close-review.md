# Boundary Review — pair#317 (whole-issue close)

| field | value |
|-------|-------|
| issue | 317 — Slot quick-status glyph in Couch tab bar and switcher |
| repo | pair |
| issue file | workshop/issues/000317-slot-quick-status-glyph.md |
| boundary | whole-issue close |
| milestone | — |
| window | 1298e1b72fcfb6ff817ce330d08604b435ec34e7..1cd23da90c35e569a364959ef0c98ad70ffcbae8 |
| command | sdlc close --issue 317 |
| reviewer | codex |
| timestamp | 2026-09-24T13:02:38-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The implementation fulfills the issue’s purpose: shared glyph derivation, background refresh, stateful fakes, documentation, and broad test coverage are present. One Important parser-hardening issue remains before crossing the boundary.

Strengths:

1. `PresentThreads` derives glyphs once for both switcher and tab bar.
2. Refresh work stays outside `c.mu`, with timeout, cancellation, single-flight scheduling, and failure retention.
3. Stateful fake tests cover slow probes, failures, triggers, shutdown, and path selection.
4. Atlas, README, fixtures, and artifact manifests were updated.
5. `go test ./...`, focused tests, race tests, and `git diff --check` passed.

Critical findings:

None.

Important findings:

- `cmd/internal/couchcore/slotgit.go:70-84` — `ParseSlotGitStatus` does not enforce its documented closed grammar. `fmt.Sscanf` accepts trailing tokens in `branch.ab`, and an empty `# branch.head ` is accepted as a valid observation. This can turn malformed external output into display evidence. Parse the complete field strictly and reject empty/duplicate malformed headers. Add regression cases that fail without the fix. (`ARCH-SECURE`, family: `porcelain-grammar-validation`)

Minor findings:

None.

Test coverage notes:

The changed packages pass focused, race, and repository-wide Go tests. Add malformed trailing-token and empty-head cases for the parser regression.

Architectural notes for upcoming work:

- ARCH-DRY: pass — one glyph derivation feeds both views.
- ARCH-PURE: pass — parsing, precedence, and reduction are pure; Git access is injected.
- ARCH-PURPOSE: pass — covers `:0`, numbered slots, both views, refresh triggers, and docs.
- ARCH-MOCK: pass — stateful fake uses the production probe seam.
- ARCH-CONSTRAINTS: pass — render paths avoid Git I/O; probes are sequential and individually time-bounded.
- ARCH-SECURE: flag above — malformed porcelain output is not fully rejected.
- ARCH-ORDER: pass — refresh scheduling and state updates have one owner and reducer transitions.
- ARCH-FUNERAL: pass — observations are in-memory and rebuilt over the current inventory; workers and ticker are stopped.

Plan revision recommendations:

None; the design still matches the implementation. Mark the completed task checkboxes in the durable plan before close. 

```findings
findings:
  - id: new
    severity: Important
    family: porcelain-grammar-validation
    title: |
      ParseSlotGitStatus accepts malformed porcelain fields
    detail: |
      cmd/internal/couchcore/slotgit.go:70-84 accepts trailing branch.ab tokens and an empty branch.head despite documenting a closed grammar; reject malformed external output and add regression tests that fail without the fix. ARCH-SECURE.
```
