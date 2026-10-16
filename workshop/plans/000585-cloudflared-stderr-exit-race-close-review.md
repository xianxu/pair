# Boundary Review — pair#585 (whole-issue close)

| field | value |
|-------|-------|
| issue | 585 — Preserve cloudflared stderr on early process exit |
| repo | pair |
| issue file | workshop/issues/000585-cloudflared-stderr-exit-race.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4c6fcc8d437ce52c708b0a6bdef09ce42b86917b..c25736eab2950b3cae5906268db0550da7b02d94 |
| command | sdlc close --issue 585 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-15T18:54:10-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The stderr ownership and drain ordering fix is sound, and the regression tests exercise both failure modes deterministically. One cleanup regression blocks shipping: reaping a process-group leader does not mean its descendants have exited, but the new guards suppress their termination.

Done-when: current — the existing clauses cover diagnostic preservation, URL refusal, and process cleanup, including the revised lifecycle contract.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: process-group-lifetime
    title: |
      Reaped command leaders can leave live descendants behind after Close
    detail: |
      cmd/internal/broadcast/cloudflared.go:320-325 and :330-334 skip group termination once processExited closes. An unguarded fake that starts `sleep 300 &`, prints its failure, and exits leaves that descendant holding stderr; after the opening timeout and forced reader close, Close returns without killing the descendant because h.child is zero. The previous unconditional group SIGTERM reached it. Every instance in this window is the two production guards, the matching cleanup assumption in cloudflared_stderr_test.go:81-85, and the claim in atlas/broadcast.md:310-311. Track group lifetime independently of leader reaping, preserve safe group cleanup, and add a real descendant regression asserting that Close both releases stderr and terminates the descendant. ARCH-PURPOSE: reader release alone does not fulfill the promised process cleanup.
```

1. **Strengths**

   - Parent-owned stderr survives `cmd.Wait`, with explicit descriptor cleanup on startup failures.
   - `waitForExit` publishes completion after drain, preserving the final diagnostic.
   - Tests cover deterministic wait-before-read ordering and exact failures in guarded and direct modes.

2. **Critical findings**

   - **Process-group cleanup:** see the finding above. A local process probe confirmed that the owned group remains alive after its leader is reaped; the probe was cleaned up.

3. **Important findings**

   None.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Full broadcast package passed with `TMPDIR=/tmp` in 30.868s. The initial run failed because the review’s long temporary path caused the Unix-socket fixture to fall back to TCP.
   - Focused lifecycle and API-refusal tests passed ten repetitions under race detection, concurrently with package testing.
   - `git diff --check` passed.
   - The retained-writer fixture verifies reader interruption but cannot detect surviving descendants because its retained writer belongs to the parent.

6. **Architectural notes**

   - **ARCH-DRY: pass.** Shared pipe setup and completion ownership avoid duplicated production behavior.
   - **ARCH-PURE: pass.** Changes remain in the process/IO boundary; no business logic was moved into it.
   - **ARCH-PURPOSE: flag.** Forced cleanup must terminate owned descendants as well as unblock the scanner.
   - Atlas documentation was updated. No new user-facing command, flag, or configuration requires a README change.

7. **Plan revision recommendations**

   Add a `## Revisions` entry distinguishing leader reaping from process-group completion and requiring a real descendant cleanup regression. Correct the atlas’s corresponding assumption alongside the fix.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"c6f1c062-2b0f-44ea-951c-6476d1f67510","repository":"github.com/xianxu/pair","issue_id":"000585","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/b2a4a29dcb3059b2","reviewed_head":"c25736eab2950b3cae5906268db0550da7b02d94","slot":"pair:3","requested_at":"2026-10-16T01:52:06.423255Z"},"phase":"finished","pid":80736,"started_at":"2026-10-16T01:52:06.599342Z","finished_at":"2026-10-16T01:54:08.887581Z","session_id":"01a14268-cd49-7e42-bc6b-2be787f15725","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.49713775s
head: "c25736eab2950b3cae5906268db0550da7b02d94"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000411
