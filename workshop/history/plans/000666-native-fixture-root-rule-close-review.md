# Boundary Review — pair#666 (whole-issue close)

| field | value |
|-------|-------|
| issue | 666 — Guard native fixture root handoffs |
| repo | pair |
| issue file | workshop/issues/000666-native-fixture-root-rule.md |
| boundary | whole-issue close |
| milestone | — |
| window | 9b841c43452eafa93b5613f0d93feb677e0b4a2f..7cb980f5ea0c51ebf5888864f8296c351b7bdcdb |
| command | sdlc close --issue 666 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-18T10:44:12-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: medium
```

The implementation matches the planned Pair-local ownership and preserves runtime protection. Focused tests pass, including all six original mutations. No code defect was found, but ordinary repository lint cannot be verified because this review checkout lacks main-ref authority and fails before reaching the new rule.

```findings
findings:
  - id: new
    severity: Important
    kind: unverifiable
    family: repository-lint-boundary-verification
    title: |
      Ordinary repository lint fails before reaching the new rule
    detail: |
      Running `go test ./internal/repolint -count=1` fails at internal/repolint/catalog_test.go:447 with “need exactly one remote main ref without main tracking config, found 0”. The sole ordinary-lint consumer added in this window, at catalog_test.go:460, is therefore not reached. Provide a successful package run at pinned head 7cb980f5ea0c51ebf5888864f8296c351b7bdcdb in a checkout with its real configured main authority available, or restore that authority in the review environment and rerun.
```

1. **Strengths**

   - Pair-specific names remain outside the shared catalog.
   - The pure checker and injected reader separate parsing policy from filesystem access.
   - All six original real-source mutations pass at the new boundary; the old duplicate matcher is removed.
   - The poisoned-root, direct/nested-child, cleanup/restoration, and Unix socket-budget regression remains unchanged and passes.

2. **Critical findings:** None.

3. **Important findings:** The verification gap above; no source fix is established as necessary.

4. **Minor findings:** None.

5. **Test coverage:** `TestNativeFixtureRoot*` passes, covering clean inputs, setup/handoff omissions, all seven root omissions, missing/malformed sources, bodyless declarations, and reader failures. `TestRecoveryLiveEnvironmentOwnsRoots` passes. Diff whitespace checks pass. Full repolint verification is blocked as described.

6. **Architecture:** **ARCH-DRY: pass**—one source matcher owns the contract. **ARCH-PURE: pass**—the checker has no IO and the reader accepts an injected dependency. **ARCH-PURPOSE: pass on inspection**—both setup entrypoints, the nested handoff, and all seven roots are protected. Atlas documents ownership and limitations; no new invocation, flag, or configuration surface requires a README change.

7. **Plan revisions:** None required. Supply successful ordinary-lint evidence before crossing the boundary.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"51ef3443-751e-41ac-abd1-688c6a78f5b9","repository":"github.com/xianxu/pair","issue_id":"000666","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/001ddd5a6ae3d287","reviewed_head":"7cb980f5ea0c51ebf5888864f8296c351b7bdcdb","slot":"pair:6","requested_at":"2026-10-18T17:43:01.162425Z"},"phase":"finished","pid":19235,"started_at":"2026-10-18T17:43:01.340479Z","finished_at":"2026-10-18T17:44:10.952034Z","session_id":"01a1501c-1bf0-7c41-bc71-2e2962c17550","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.359479458s
head: "7cb980f5ea0c51ebf5888864f8296c351b7bdcdb"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-18T10:48:21-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 666 — Guard native fixture root handoffs |
| repo | pair |
| issue file | workshop/issues/000666-native-fixture-root-rule.md |
| boundary | whole-issue close |
| milestone | — |
| window | 9b841c43452eafa93b5613f0d93feb677e0b4a2f..ed4453bdd51193696f695905c38a378ae956ca7a |
| command | sdlc close --issue 666 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-18T10:48:21-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The implementation delivers the bounded Pair-local lint contract, preserves runtime protection, and removes the duplicate matcher. Focused tests and races pass. BR-1 is withdrawn based on the measured host evidence and verification that executable code has not changed since that head. One minor documentation overstatement remains.

```findings
dispose:
  - id: BR-1
    disposition: withdrawn
    note: |
      This checkout reproduces the missing-main-authority failure. The supplied exact-head package PASS17.539s and named-consumer PASS4.217s answer the requested measurement; inspection confirms that only tracker/prose files changed between measured head 7cb980f5ea0c51ebf5888864f8296c351b7bdcdb and pinned head ed4453bdd51193696f695905c38a378ae956ca7a. No authority-guard change is warranted.
findings:
  - id: new
    severity: Minor
    family: coverage-claims-match-tests
    title: |
      Atlas overstates individual entrypoint-omission coverage
    detail: |
      atlas/go-lint-catalog.md:61 says synthetic tests omit each entrypoint, but internal/repolint/native_fixture_roots_test.go:39-44 individually removes only the warm entrypoint; park coverage removes its setup call. This is the sole universal entrypoint-omission coverage claim in the window. Say “each root and source, plus a missing entrypoint,” or add individual omission cases for the other protected functions.
```

Done-when: current — the Spec change relocates existing design text; the acceptance clauses still cover local ownership, ordinary lint consumption, mutation qualification, and retained runtime protection.

1. **Strengths**

   - The pure checker reports located findings and visibly rejects missing or malformed required sources.
   - The injected reader preserves read errors and verifies one read per protected file.
   - All six original real-source mutations are retained; poisoned direct/nested-child, cleanup/restoration, and socket-budget behavior remains unchanged.
   - Atlas documents ownership, the consumption interface, and the deliberately bounded syntax contract.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** The atlas coverage claim above.

5. **Test coverage notes**

   Passed the native-fixture rule, reader, and real-source mutation tests, plus `TestRecoveryLiveEnvironmentOwnsRoots`, both normally and under `-race`. `git diff --check` passed. The full repolint package remains locally blocked by missing main authority; the supplied successful host measurements cover that boundary. No behavior-changing prior fix required a revert experiment.

6. **Architectural notes**

   - **ARCH-DRY: pass.** The old matcher is removed; the local rule owns inspection.
   - **ARCH-PURE: pass.** AST inspection is deterministic; filesystem access stays in the injected reader.
   - **ARCH-PURPOSE: pass.** Both setup consumers, the nested handoff, and all seven root reads are covered. Runtime regression coverage remains intact.

   Atlas updates satisfy the architectural documentation gate. No new command, flag, configuration key, or usage step requires a README update. The quick plan has no Core concepts table.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"f88c376c-6242-41c4-8584-1e72379857ed","repository":"github.com/xianxu/pair","issue_id":"000666","recipe":"small-diff-review","round":2,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/6a72ff96f8bf5dc6","reviewed_head":"ed4453bdd51193696f695905c38a378ae956ca7a","slot":"pair:6","requested_at":"2026-10-18T17:47:11.239184Z"},"phase":"finished","pid":95368,"started_at":"2026-10-18T17:47:11.539449Z","finished_at":"2026-10-18T17:48:20.639282Z","session_id":"01a1501f-ed00-7d01-bd08-13fc2f97aa90","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.244966125s
head: "ed4453bdd51193696f695905c38a378ae956ca7a"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
