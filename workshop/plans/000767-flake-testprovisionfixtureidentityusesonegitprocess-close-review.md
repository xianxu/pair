# Boundary Review — pair#767 (whole-issue close)

| field | value |
|-------|-------|
| issue | 767 — flake: TestProvisionFixtureIdentityUsesOneGitProcess |
| repo | pair |
| issue file | workshop/issues/000767-flake-testprovisionfixtureidentityusesonegitprocess.md |
| boundary | whole-issue close |
| milestone | — |
| window | 42bfa761430c9e27413135f2453997b1350b4907..07063d275d8b1f05d3093c2bdf937e958f419722 |
| command | sdlc close --issue 767 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T05:45:37-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The change replaces the unreliable pathname permission check with direct inherited-descriptor proof and preserves the one-Git-process assertion. Independent regression checks confirm the tests detect both the old shim and a disabled lint rule. No findings block this code review; the recorded full-suite result remains a separate close/landing gate.

```findings
{}
```

Done-when: current — the diagnostics, controlled regression, exact-lease proof and verification clauses still cover the revised implementation.

1. **Strengths**

   - `cmd/internal/couchcore/provision_identity_fixture_test.go:128` covers readable, mode000, absent and wrong-token descriptors.
   - The call-log assertion at line 155 verifies exactly one Git invocation per accepted case and none for rejected cases.
   - `internal/repolint/fd_path_permission_test.go:9` exercises the real scanner with accepted and rejected syntax.
   - `atlas/go-lint-catalog.md` documents the new rule, ownership and lexical limitations. No new user-facing command or configuration requires a README update.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Independently passed:

   - Identity family, 10 repetitions.
   - Full `internal/repolint` package.
   - `make test-affected-inventory`.
   - Retained native probe: **16,000 attempts, zero failures**.

   Temporary Go overlays established regression sensitivity without modifying tracked files: restoring the old shim fails mode000 and wrong-token cases; disabling the lint matcher fails all three positive cases.

6. **Architectural notes**

   - **ARCH-DRY: pass.** Reuses the composed catalog and scanner.
   - **ARCH-PURE: pass.** AST matching stays pure; native descriptor verification stays at the integration boundary.
   - **ARCH-PURPOSE: pass.** Corrects the evidenced oracle failure while retaining exact-lease and process-count checks; no retries or timeout increases.
   - **ARCH-EVOLVE: pass.** No persisted-format semantics change; the inventory certificate only refreshes its source hash.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"7788ec04-6868-4a03-bae2-c888025b02c8","repository":"github.com/xianxu/pair","issue_id":"000767","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/153a96fcbb492582","reviewed_head":"07063d275d8b1f05d3093c2bdf937e958f419722","slot":"pair:9","requested_at":"2026-10-21T12:43:40.08019Z"},"phase":"finished","pid":26024,"started_at":"2026-10-21T12:43:40.170982Z","finished_at":"2026-10-21T12:45:37.035228Z","session_id":"01a15e7d-1e14-7de2-bdb2-ffe35ba8efa9","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.827169792s
head: "07063d275d8b1f05d3093c2bdf937e958f419722"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-21T08:44:44-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 767 — flake: TestProvisionFixtureIdentityUsesOneGitProcess |
| repo | pair |
| issue file | workshop/issues/000767-flake-testprovisionfixtureidentityusesonegitprocess.md |
| boundary | whole-issue close |
| milestone | — |
| window | 46d40283cf1d09d26a25cf003714c9ef8c11a6ae..458897c5274b5313c32017e02f4effb6b73024cf |
| command | sdlc close --issue 767 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T08:44:44-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The interdiff correctly resolves both conflicts, preserving the independent lint rules and their atlas documentation. Tracker changes record verification and close evidence without expanding scope. No new defects found; no prior findings require disposition.

```findings
{}
```

Done-when: current — the existing diagnostics, regression, inherited-lease proof and verification requirements still cover this window.

1. **Strengths**
   - `internal/repolint/local_catalog_test.go:39` retains both catalog registrations with their correct matchers and incident references.
   - `atlas/go-lint-catalog.md:177` preserves both rules’ ownership, behavior and analysis limits.
   - Existing scanner regressions exercise accepted and rejected inputs through the composed catalog.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**
   - `go test ./internal/repolint -count=1` passed, including both rule regressions and repository scanning.
   - `make test-affected-inventory` passed.
   - `git diff --check` passed; checkout remains clean.
   - The earlier full-suite claims were not independently rerun in this narrow review.

6. **Architectural notes**
   - **ARCH-DRY: pass.** Both rules reuse the existing catalog and scanner.
   - **ARCH-PURE: pass.** Resolution adds no IO or business logic.
   - **ARCH-PURPOSE: pass.** Both sides’ enforcement intent survives; scope remains unchanged.
   - **ARCH-EVOLVE: pass.** No persisted-format semantics change.
   - Atlas coverage is preserved. No new user-facing surface requires a README update.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"934226aa-79a2-4ec3-9276-fa660e35a635","repository":"github.com/xianxu/pair","issue_id":"000767","recipe":"fix-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/f4c462fecab0a70b","reviewed_head":"458897c5274b5313c32017e02f4effb6b73024cf","slot":"pair:9","requested_at":"2026-10-21T15:43:47.815871Z"},"phase":"finished","pid":7042,"started_at":"2026-10-21T15:43:47.945064Z","finished_at":"2026-10-21T15:44:43.739995Z","session_id":"01a15f22-0814-7ab1-9c00-9d273235aee4","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.050620041s
head: "458897c5274b5313c32017e02f4effb6b73024cf"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
