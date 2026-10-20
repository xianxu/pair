# Boundary Review — pair#720 (whole-issue close)

| field | value |
|-------|-------|
| issue | 720 — Slot scratch: export the RAM-disk temp variables only when the agent's sandbox can write there (the probe runs outside the sandbox) |
| repo | pair |
| issue file | workshop/issues/000720-slot-scratch-export-the-ram-disk-temp-variables-only-when-the-agent-s-sandbox-can-write-there-the-probe-runs-outside-the-sandbox.md |
| boundary | whole-issue close |
| milestone | — |
| window | 7d9ce7ca7988743e3ef201221a19c985c8cecec7..7d772f18a1e923e9ced69d630e8c142f52d43cc2 |
| command | sdlc close --issue 720 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-19T21:26:38-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The shared admission predicate, bounded settings reads, selected-profile plumbing, and documentation updates are sound foundations. Two gaps still permit RAM scratch exports when Claude’s effective sandbox denies access, so the issue’s central contract is not yet satisfied.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: effective-policy-denials
    title: |
      Claude admission ignores permission rules that deny sandbox writes
    detail: |
      cmd/internal/couchscratch/sandbox_settings.go:29-55 projects only sandbox.filesystem and ignores permissions.deny. With sandbox.enabled=true, allowWrite=["/Volumes/TestScratch"], and permissions.deny=["Edit(//Volumes/TestScratch/**)"], ScratchAllowed returns true although Claude applies Edit denies to sandbox access. Project these restrictions, or return unknown when unsupported write-denial rules occur; add a regression proving that this configuration produces no scratch exports.
  - id: new
    severity: Critical
    kind: defect
    family: effective-policy-source-discovery
    title: |
      Claude local settings discovery misses repository and main-checkout policy
    detail: |
      cmd/internal/couchscratch/sandbox_read.go:59-62 reads settings.local.json only beneath in.Project, supplied from record.StartingPath at cmd/internal/couchcmd/scratch.go:175. Claude can load local settings from the repository root when launched in a subdirectory, and from the main checkout for a worktree. A denying local policy there is skipped and absence is treated as sandbox disabled, allowing all three scratch exports. Resolve the native local-settings location or conservatively decline unresolved layouts; cover subdirectory and linked-worktree fixtures through the production reader.
```

1. **Strengths**

   - `ScratchAllowed` has direct, IO-free tests for ancestor grants, sibling prefixes, overlapping denies, and malformed paths.
   - `readSandboxFile` bounds reads and rejects special files; its tests include FIFOs and oversized settings.
   - Fresh and switched launches receive the selected profile without rewriting the durable profile.
   - README and atlas changes cover the new admission behavior and current-selection diagnostics.

2. **Critical findings**

   - **Ignored permission denies:** [sandbox_settings.go:29](/Volumes/TestScratch/review-checkouts/e0b2d34759bed559/cmd/internal/couchscratch/sandbox_settings.go:29). Claude’s documented sandbox policy includes `Edit` deny rules, so `allowWrite` alone cannot establish admission. Preserve the conservative projection by treating unsupported restrictions as unknown. See [Claude sandbox permission rules](https://code.claude.com/docs/en/sandboxing#permission-rules).
   - **Incorrect local-settings location:** [sandbox_read.go:59](/Volumes/TestScratch/review-checkouts/e0b2d34759bed559/cmd/internal/couchscratch/sandbox_read.go:59). The reader assumes both project files share one directory. Claude documents different local-file discovery for subdirectories and worktrees. See [Claude local settings locations](https://code.claude.com/docs/en/settings#keep-personal-settings-out-of-a-repository).

3. **Important findings**

   None separately raised.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Full `couchscratch`, `couchcmd`, and `artifactpath` packages passed. Focused fresh/warm launch, switch, show, and host-probe regression tests passed. The broader package run was stopped after roughly two minutes while `couchcore` remained running; its full result is unverified.

   Current fixtures omit both blocking cases. Corrections need tests that fail on this pinned head. There are no prior findings to dispose of.

6. **Architectural notes**

   - **ARCH-DRY — pass:** launch and diagnostics share admission logic.
   - **ARCH-PURE — pass:** predicate and projections remain IO-free; core-concept entities exist at their documented locations.
   - **ARCH-PURPOSE — flag:** the two findings leave the original host-success/consumer-denial failure class reachable.
   - **ARCH-MOCK — pass within the declared envelope:** portable settings fixtures exercise the production reader seam; native conformance remains deferred.
   - **ARCH-CONSTRAINTS — pass:** fixed-count, bounded reads; no background work added.
   - **ARCH-SECURE — flag:** incomplete external-policy interpretation turns missing evidence into permission.
   - **ARCH-ORDER — pass:** no new state retained between events.
   - **ARCH-FUNERAL — pass:** no new durable artifact family.
   - **ARCH-EVOLVE — pass:** settings are read without rewriting unknown fields.

7. **Plan revision recommendations**

   Add `## Revisions` entries specifying:

   - All write-restricting Claude policy inputs are projected or cause unknown admission, including permission denies.
   - Local-settings discovery accounts for launch directory, repository root, and linked-worktree main checkout, with explicit fallback and regression fixtures.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"8a945867-1589-45dc-b29c-29780582f2d2","repository":"github.com/xianxu/pair","issue_id":"000720","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/e0b2d34759bed559","reviewed_head":"7d772f18a1e923e9ced69d630e8c142f52d43cc2","slot":"pair:7","requested_at":"2026-10-20T04:23:39.668926Z"},"phase":"finished","pid":77636,"started_at":"2026-10-20T04:23:39.761125Z","finished_at":"2026-10-20T04:26:36.442624Z","session_id":"01a1578c-fcf1-7a12-af5d-793d2abf31bc","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.403145s
head: "7d772f18a1e923e9ced69d630e8c142f52d43cc2"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000409
