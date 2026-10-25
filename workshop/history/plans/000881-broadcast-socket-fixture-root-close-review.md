# Boundary Review — pair#881 (whole-issue close)

| field | value |
|-------|-------|
| issue | 881 — Broadcast socket fixture exceeds managed TMPDIR path budget |
| repo | pair |
| issue file | workshop/issues/000881-broadcast-socket-fixture-root.md |
| boundary | whole-issue close |
| milestone | — |
| window | 357670a37246e68471ec41a10cf32bd7f2916c5e..6d4f16661d169ac72fd6d5a3fddabca73583a37a |
| command | sdlc close --issue 881 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-25T01:05:25-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The fixture correction satisfies the functional purpose: it reuses the shared allocator, exercises a real Unix listener under long TMPDIR, and preserves production fallback behavior. All review checks passed. One atlas omission needs correction before closing.

```findings
findings:
  - id: new
    severity: Important
    kind: defect
    family: atlas-surface-traceability
    title: |
      Atlas update appears missing for broadcast socket fixture ownership
    detail: |
      internal/repolint/local_catalog_test.go:15 introduces broadcast-socket-fixture-root, but this range contains no atlas update. This is the sole new ownership convention in the window. Add its protected owner, required ShortRoot delegation, matcher limitations and regression-test location to atlas/go-lint-catalog.md; lines 139–144 explicitly require atlas mapping for Pair-local ownership policies.
```

1. **Strengths**
   - `cloudflared_test.go:25` delegates to the existing allocator, covering every `shortDir` caller without duplicating allocation logic.
   - `cloudflared_test.go:104` checks Unix availability before poisoning TMPDIR, then verifies the complete worst-case socket path, permissions, configuration and cleanup.
   - The catalog tests exercise valid aliases and five rejected allocator variants through the actual scanner.

2. **Critical findings:** None.

3. **Important findings:** The atlas mapping above is missing. No README change is required: no new command, flag or configuration key is introduced.

4. **Minor findings:** None.

5. **Test coverage notes**
   - PASS: named Unix tunnel, overflow TCP fallback and listener cleanup, each repeated three times.
   - PASS: complete broadcast and repolint packages.
   - PASS: tracked-source catalog and `make test-affected-inventory`.
   - No mutation run was performed; inspection confirms the old helper would inherit the deliberately oversized TMPDIR and fail the complete-path assertion.

6. **Architectural notes**
   - **ARCH-DRY — pass:** reuses `ShortRoot` and existing import-identity matching.
   - **ARCH-PURE — pass:** AST policy remains pure; fixture IO stays in integration tests.
   - **ARCH-PURPOSE — pass:** shared fixture callers inherit the correction; production fallback remains unchanged.
   - **ARCH-EVOLVE — pass:** no persisted-format semantics change.

7. **Plan revision recommendations:** No design revision needed. Add the atlas mapping to the completion checklist, and retain the recorded affected-check requirement before landing.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"6c08ba4d-c000-45b2-b69b-3b8b83eec0c1","repository":"github.com/xianxu/pair","issue_id":"000881","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/78dca7e74ff6300c","reviewed_head":"6d4f16661d169ac72fd6d5a3fddabca73583a37a","slot":"pair:5","requested_at":"2026-10-25T08:03:49.35308Z"},"phase":"finished","pid":24569,"started_at":"2026-10-25T08:03:49.450535Z","finished_at":"2026-10-25T08:05:24.200389Z","session_id":"01a17216-596b-7d32-8f52-3089a6415755","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.265343417s
head: "6d4f16661d169ac72fd6d5a3fddabca73583a37a"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-25T01:13:53-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 881 — Broadcast socket fixture exceeds managed TMPDIR path budget |
| repo | pair |
| issue file | workshop/issues/000881-broadcast-socket-fixture-root.md |
| boundary | whole-issue close |
| milestone | — |
| window | 357670a37246e68471ec41a10cf32bd7f2916c5e..619613bb434b245788ea417ec40a76859300757c |
| command | sdlc close --issue 881 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-25T01:13:53-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The change fulfills the fixture correction without changing production fallback behavior. Independent tests passed, and restoring the old allocator through a temporary Go overlay made the regression fail deterministically. BR-1 is addressed; no new findings.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      atlas/go-lint-catalog.md:126–138 now documents the protected broadcast.shortDir owner, required imported testpair.ShortRoot return, alias/shadow handling, matcher limitations and regression locations. The passage “checks top-level return expressions” matches internal/repolint/local_catalog_test.go:298–322; both named test referents exist and passed. This is a prose-only correction.
```

1. **Strengths**

   - `cloudflared_test.go:25` reuses `testpair.ShortRoot`, correcting all existing `shortDir` consumers through their shared owner.
   - `cloudflared_test.go:104` checks Unix availability before setting long TMPDIR, then verifies the complete worst-case derived path, real Unix listener, permissions, configuration and cleanup.
   - The catalog tests exercise valid aliases and five rejected allocator variants through the actual scanner.

2. **Critical findings:** None.

3. **Important findings:** None. The sole new ownership convention now has its atlas mapping. No new user-facing command, flag or configuration key requires a README update.

4. **Minor findings:** None.

5. **Test coverage notes**

   - PASS: named Unix tunnel, too-long TCP fallback and listener cleanup, each repeated three times without skips.
   - PASS: complete broadcast package and complete repolint package, including tracked-source checks and mutation controls.
   - PASS: `make test-affected-inventory` and pinned-range whitespace check.
   - Negative control: replacing only the allocator correction through a temporary overlay made the named regression fail with a **231-byte complete socket path against the 100-byte budget**. Temporary files were removed; the checkout remains clean.
   - The recorded affected-check workflow remains explicitly pending in the issue; this review does not establish that separate landing requirement.

6. **Architectural notes**

   - **ARCH-DRY — pass:** existing allocation and import-identity helpers are reused.
   - **ARCH-PURE — pass:** the AST matcher remains pure; fixture IO and injected repository scanning stay at integration boundaries.
   - **ARCH-PURPOSE — pass:** socket fixture consumers share the corrected owner, and production overflow fallback still passes.
   - **ARCH-EVOLVE — pass:** no persisted-format semantics or reader/writer contracts change.

7. **Plan revision recommendations:** None. Complete the existing verification checklist row when the recorded check finishes.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"8642f686-f483-443e-9fae-95e94a81964b","repository":"github.com/xianxu/pair","issue_id":"000881","recipe":"small-diff-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/7d25fe83b5367cfa","reviewed_head":"619613bb434b245788ea417ec40a76859300757c","slot":"pair:5","requested_at":"2026-10-25T08:11:38.117821Z"},"phase":"finished","pid":18954,"started_at":"2026-10-25T08:11:38.280034Z","finished_at":"2026-10-25T08:13:52.717108Z","session_id":"01a1721d-8169-7482-bfec-b9be2b048f75","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.589369083s
head: "619613bb434b245788ea417ec40a76859300757c"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
