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
