# Boundary Review — pair#769 (whole-issue close)

| field | value |
|-------|-------|
| issue | 769 — Preserve wrap Darwin PTY final output without changing termination |
| repo | pair |
| issue file | workshop/issues/000769-wrap-darwin-pty-output.md |
| boundary | whole-issue close |
| milestone | — |
| window | 7fabbeb93589b7c0c44a43fa699e83b09082bd5e..2119e58202baca40d25389862df0a3166e164bc4 |
| command | sdlc close --issue 769 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T07:28:57-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The Darwin lease preserves final output while retaining the approved process-exit lifetime and sole-reap ownership. Native and race-enabled qualification passed. One Important documentation gap remains: the new help entrypoint and changed Darwin behavior are absent from README.md.

```findings
findings:
  - id: new
    severity: Important
    kind: defect
    family: user-surface-readme-coverage
    title: |
      README update appears missing for wrap help and Darwin lifetime behavior.
    detail: |
      cmd/internal/wrapcmd/wrap.go:2776 introduces pair wrap --help and -h, and lines 2816–2819 describe the changed closed-stdio lifetime policy, but README.md is unchanged in the pinned range. Add a concise README usage entry linking the help command and explaining that Darwin targets closing stdio remain owned until exit or explicit stop; atlas documentation alone does not satisfy the README update gate.
```

1. **Strengths**

   - The descriptor acknowledgement forces child stdio closure before the first master read, directly exercising the output-loss ordering (`cmd/internal/wrapcmd/pty_darwin_test.go:126`).
   - Lease release is idempotent, concurrent-safe, and preserves the original close error (`cmd/internal/wrapcmd/pty.go:19`).
   - Initialization failure now uses the registered `ownedGroup` rather than a competing direct process reap (`cmd/internal/wrapcmd/wrap.go:3012`).
   - Caller tests exercise public names, fresh replacement arguments, and both actual KDL launch stanzas, including same-group and escaped descendants.

2. **Critical findings**

   None.

3. **Important findings**

   The README gap above. Document `pair wrap --help` and the approved Darwin lifetime change near the existing helper overview (`README.md:410`).

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed native owner/caller tests, managed readiness and exit-report scenarios, help tests, and the reap-rule test. A combined race-enabled run also passed, covering lease release, native lifetime scenarios, caller composition, managed lifecycle, transport failure, and terminal integration. `make test-affected-inventory` passed.

   No source mutation was performed. This is the first review round; there are no prior findings requiring fail-without-fix verification. The full recorded suite was not independently run during this review.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared caller fixture avoids duplicated lifecycle protocols.
   - **ARCH-PURE — pass:** PTY effects remain in a small adapter with an injected starter.
   - **ARCH-PURPOSE — pass:** implementation and native tests cover the approved lifetime change and enumerated launch boundaries.
   - **ARCH-MOCK — pass:** injected allocation/close failures complement real native qualification.
   - **ARCH-CONSTRAINTS — pass:** one retained descriptor per Darwin wrapper; no new polling or per-byte work.
   - **ARCH-SECURE — pass:** direct command arguments and original child identity remain intact; fixtures use disposable roots.
   - **ARCH-ORDER — pass:** `sync.Once` enforces release, and cleanup preserves sole-reap authority.
   - **ARCH-FUNERAL — pass:** normal and abort paths release the lease and join owned children.
   - **ARCH-EVOLVE — pass:** no persisted-format changes.

   The Core concepts integration rows match the implementation and stated file statuses. Atlas updates cover the new ownership terminology and flow.

7. **Plan revision recommendations**

   Add a `## Revisions` entry extending task 3 to require README documentation of `pair wrap --help` and the Darwin closed-stdio lifetime policy.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"d3461ec5-f3c6-4f19-ba7c-82b682a20008","repository":"github.com/xianxu/pair","issue_id":"000769","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/603593012770156a","reviewed_head":"2119e58202baca40d25389862df0a3166e164bc4","slot":"pair:4","requested_at":"2026-10-21T14:26:51.490456Z"},"phase":"finished","pid":71190,"started_at":"2026-10-21T14:26:51.642855Z","finished_at":"2026-10-21T14:28:55.795577Z","session_id":"01a15edb-97dd-7423-9aac-cc18c464843f","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.149278583s
head: "2119e58202baca40d25389862df0a3166e164bc4"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-21T07:33:37-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 769 — Preserve wrap Darwin PTY final output without changing termination |
| repo | pair |
| issue file | workshop/issues/000769-wrap-darwin-pty-output.md |
| boundary | whole-issue close |
| milestone | — |
| window | 7fabbeb93589b7c0c44a43fa699e83b09082bd5e..dfc195a1b30268c577cbe3e8d2e21123f60c19b3 |
| command | sdlc close --issue 769 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T07:33:37-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned implementation matches the issue’s approved lifetime contract and plan. Native Darwin tests passed, and removing slave retention through a temporary Go overlay made both final-output regressions fail with empty output. BR-1 is addressed; no new blocking findings were identified. This review does not establish the separately required full-suite or merge-gate results.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      README.md:417 now documents pair wrap --help/-h and ownership until target exit or explicit stop. This matches wrap.go:2776 and :2814, the retained-slave implementation in pty_darwin.go:50, and atlas/terminal.md:189–199. The correction is prose-only; native lifecycle tests independently passed.
```

1. **Strengths**

   - `pty_darwin.go:20` preserves direct `exec.Cmd` execution, synchronous startup errors, and supplied command fields while retaining the slave.
   - `pty.go:18` provides concurrent, idempotent release with a stable error. `wrap.go:2998` releases before deferred group reaping on abort.
   - Barrier-controlled tests establish child stdio closure before the first master read. The counterfactual test confirms they detect the original output-loss behavior.
   - Public, compatibility, fresh-restart, and both actual layout routes receive native normal/group/daemon qualification. README and atlas explain the changed policy.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Passed on native Darwin:

   - Targeted wrap/launcher lifecycle tests, including race-enabled runs.
   - Managed readiness, native-exit reporting, closed-stdio stop/restart, startup failures, large output, and resource cleanup.
   - Existing restart, output-failure, terminal integration, and selected Couch route tests.
   - `internal/repolint`, `artifactpath`, `make test-affected-inventory`, and diff whitespace checks.

   The temporary immediate-close mutation failed both status-0 and status-7 final-output cases as expected. Temporary files were removed; the checkout remains clean.

6. **Architectural notes**

   - **ARCH-DRY — pass:** Shared caller fixtures avoid duplicated protocols; existing lock and group owners remain authoritative.
   - **ARCH-PURE — pass:** PTY effects stay in the integration adapter behind an instance-local starter seam. Core-concept classifications match the code.
   - **ARCH-PURPOSE — pass:** Output preservation and the approved closed-stdio lifetime are delivered across the enumerated caller families.
   - **ARCH-MOCK — pass:** Injected allocation/close failures complement native kernel and process qualification.
   - **ARCH-CONSTRAINTS — pass:** One additional descriptor per live Darwin wrapper; no new polling or per-byte work. Resource and 128-KiB streaming tests passed.
   - **ARCH-SECURE — pass:** Original argv and directly created child identity are preserved; fixtures use disposable environments.
   - **ARCH-ORDER — pass:** `sync.Once` enforces lease release; cleanup retains sole group-reap authority. Barriers exercise consequential event ordering.
   - **ARCH-FUNERAL — pass:** Normal, abort, and startup-failure paths release descriptors and join owned children; no new durable artifact family.
   - **ARCH-EVOLVE — pass:** No persisted-format changes.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"ba70e633-df61-4c0c-8f26-093b0a9fabfe","repository":"github.com/xianxu/pair","issue_id":"000769","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/d80a1b641c93be44","reviewed_head":"dfc195a1b30268c577cbe3e8d2e21123f60c19b3","slot":"pair:4","requested_at":"2026-10-21T14:31:00.317841Z"},"phase":"finished","pid":46138,"started_at":"2026-10-21T14:31:00.424877Z","finished_at":"2026-10-21T14:33:29.853799Z","session_id":"01a15edf-6349-7123-bee1-9b52ddaa6ff2","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.670195041s
head: "dfc195a1b30268c577cbe3e8d2e21123f60c19b3"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
