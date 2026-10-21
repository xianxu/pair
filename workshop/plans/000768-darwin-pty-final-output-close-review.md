# Boundary Review — pair#768 (whole-issue close)

| field | value |
|-------|-------|
| issue | 768 — Preserve short-lived command output across Darwin PTY slave close |
| repo | pair |
| issue file | workshop/issues/000768-darwin-pty-final-output.md |
| boundary | whole-issue close |
| milestone | — |
| window | 42bfa761430c9e27413135f2453997b1350b4907..250cea921d9d614678de214f25edf4926aa461de |
| command | sdlc close --issue 768 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T06:17:43-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned implementation matches the approved scribe-only Spec and Plan. Native Darwin checks confirm exact final output, descendant-held-slave completion, and abort cleanup. No blocking findings. The recorded full-suite pass remains a separate landing prerequisite; this review does not establish that result.

```findings
{}
```

1. **Strengths**

   - `pty_darwin.go:21` preserves upstream direct-exec setup while transferring slave ownership to the drain.
   - `pty.go:32` kills the owned child and closes both descriptors before waiting on initialization abort.
   - `pty_darwin_test.go:245` exercises production drain behavior through an injected starter and deterministic stdio-close barrier.
   - Atlas and scribe documentation accurately describe ownership and the remaining PTY owners.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage**

   Passed native, uncached scribe tests; scribe race and vet checks; artifact-path and repository-lint tests; reader inventory; and diff whitespace checks. Coverage includes exact stdout/log bytes, exit status, descendants, aborts, descriptor lifetimes, backpressure, signals, and terminal restoration. No repository changes were made.

6. **Architectural notes**

   - **ARCH-DRY — pass:** Reuses upstream allocation; local orchestration is small and justified.
   - **ARCH-PURE — pass:** New work is resource-management glue with injected boundaries.
   - **ARCH-PURPOSE — pass:** Fixes the approved scribe scope; other owners remain explicitly documented.
   - **ARCH-MOCK — pass:** Injected startup/allocation seams and close doubles complement native kernel qualification.
   - **ARCH-CONSTRAINTS — pass:** One retained descriptor per active Darwin child; no per-byte overhead or production timeout.
   - **ARCH-SECURE — pass:** Preserves argv execution and confines fixtures to temporary resources.
   - **ARCH-ORDER — pass:** Drain, release, abort, and sole-wait ordering are explicit and exercised.
   - **ARCH-FUNERAL — pass:** New handles have normal, startup-failure, and abort cleanup paths.
   - **ARCH-EVOLVE — pass:** No persisted-format changes.

   Core-concept entities exist at the documented paths and match their integration classification. No new command, flag, or configuration surface requires a README update.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"83bc5487-98e1-4bc3-abda-da9788a4dcdc","repository":"github.com/xianxu/pair","issue_id":"000768","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/00c32365c951e193","reviewed_head":"250cea921d9d614678de214f25edf4926aa461de","slot":"pair:2","requested_at":"2026-10-21T13:16:00.785905Z"},"phase":"finished","pid":28328,"started_at":"2026-10-21T13:16:00.938497Z","finished_at":"2026-10-21T13:17:42.374299Z","session_id":"01a15e9a-bbb3-76e0-82a6-011f4d14b7e8","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.11734675s
head: "250cea921d9d614678de214f25edf4926aa461de"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000411

---

## Re-review — 2026-10-21T08:10:27-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 768 — Preserve short-lived command output across Darwin PTY slave close |
| repo | pair |
| issue file | workshop/issues/000768-darwin-pty-final-output.md |
| boundary | whole-issue close |
| milestone | — |
| window | a39bdf2ecc0804c97648a68ca40db37e3a8bc445..b014d464b44e189062c9ca38e5764cc8feca048c |
| command | sdlc close --issue 768 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T08:10:27-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The interdiff correctly repairs the Darwin cwd test oracle without changing production behavior or the approved Spec. The explicit symlink fixture tests filesystem identity while preserving exact command-field, argv, and environment assertions. No prior findings require disposition, and no new defects were found.

```findings
{}
```

1. **Strengths**
   - `pty_darwin_test.go:474` explicitly exercises a symlinked working directory.
   - `pty_darwin_test.go:497` retains lexical command-field preservation checks.
   - `pty_darwin_test.go:511–526` verifies literal argument/environment output and cwd identity, handling physical path spelling correctly.
   - The updated reader certificate matches the test file’s SHA-256.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage:** Passed native Darwin `go test ./cmd/internal/scribecmd -count=1`, package vet, `make test-affected-inventory`, and diff whitespace checks. This review does not establish a full-suite or landing-gate pass.

6. **Architectural notes**
   - **ARCH-DRY — pass:** No duplicated behavior introduced.
   - **ARCH-PURE — pass:** Filesystem checks belong in this integration test.
   - **ARCH-PURPOSE — pass:** The correction preserves the cwd contract without weakening literal-value assertions.
   - **ARCH-EVOLVE — pass:** No persisted-format changes.
   
   No new architectural or user-facing surface requires atlas or README updates.

7. **Plan revision recommendations:** None. The checklist edit records review progress and retains final-code suite and CI landing requirements.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"3be23981-53f0-4f0d-a07a-201c418d6906","repository":"github.com/xianxu/pair","issue_id":"000768","recipe":"fix-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/0f204b51e793015c","reviewed_head":"b014d464b44e189062c9ca38e5764cc8feca048c","slot":"pair:2","requested_at":"2026-10-21T15:08:51.69767Z"},"phase":"finished","pid":25623,"started_at":"2026-10-21T15:08:51.872123Z","finished_at":"2026-10-21T15:09:50.146037Z","session_id":"01a15f02-0c91-79c2-92df-4cf95c978f2d","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.031916209s
head: "b014d464b44e189062c9ca38e5764cc8feca048c"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
