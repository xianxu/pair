# Boundary Review — pair#523 (whole-issue close)

| field | value |
|-------|-------|
| issue | 523 — broadcast: 👆 and 👽 are capability switches on both casts (👆 off: no one annotates; 👽 off: no one drives from outside) |
| repo | pair |
| issue file | workshop/issues/000523-broadcast-and-are-capability-switches-on-both-casts-off-no-one-annotates-off-no-one-drives-from-outside.md |
| boundary | whole-issue close |
| milestone | — |
| window | 844d29e698e481712f6a58f37f04e4d717a405f3..c1cb28e9d593293f8806a380881cf4069328cbed |
| command | sdlc close --issue 523 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-13T11:04:35-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The capability authorities, generation checks, recording clears, and documentation largely match the Spec/Plan. All six prior findings remain addressed. One new regression blocks shipping: ordinary drive actions now wait for recorder/CDP completion on Couch’s input loop, making the terminal unresponsive during those waits.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      replay.js:45 preserves annotation-clear during catch-up. Removing it made both shared-fixture prefix and Player.seek regressions fail; restored tests passed.
  - id: BR-2
    disposition: addressed
    note: |
      cast_drive.go routes control through recorder/control.go's acknowledged writer. Replacing acknowledgment with Emit made TestProductionControlRefusesClosedRecorder fail; the restored test passed.
  - id: BR-3
    disposition: addressed
    note: |
      The plan table at line 82 and revision at line 447 name modified board/board.go and new board/annotation.go, matching the pinned diff and AnnotationAdmission implementation.
  - id: BR-4
    disposition: addressed
    note: |
      README.md:1055 documents independent default-OFF capabilities and persistent links, matching Board/controller initialization and server admission. The README now also documents the completed controls.
  - id: BR-5
    disposition: addressed
    note: |
      Independently removing annotation and drive generation checks made the pre-ingress regressions fail across viewer and operator routes. Removing annotation-clear gesture cancellation also made capability-ui.test.mjs:70 fail.
  - id: BR-6
    disposition: addressed
    note: |
      The plan revision at lines 551–566 documents the unresolved-delivery retry exception, deadlines and shutdown, matching chrome_operator.go's single worker, conditional retry selection and deferred ticker stop.
findings:
  - id: new
    severity: Critical
    family: operating-envelope-traceability
    title: |
      Acknowledged drive actions block Couch's input loop
    detail: |
      ARCH-CONSTRAINTS: controller_mem.go:204–212 now synchronously awaits WriteControl and Release, but console_drive.go:180–199 still calls Grant, Deny and Revoke directly from mouse handlers. A gated-writer reproduction confirmed revokeDrive cannot return before acknowledgment. This is the 2nd finding in family operating-envelope-traceability. Enforce the rule that no terminal input handler awaits recorder/CDP completion: sweep grant, deny, popup revoke and Ctrl-click revoke, dispatch them through bounded owned work, and fence completion by cast identity. Preserve the acknowledged writer.
```

**1. Strengths**

- Generation-bound admission covers viewer annotation, operator mutations, and drive request/release/input; mutation tests confirm these guards matter.
- Cleanup preserves recording history while clearing current annotations, including replay seeking and pause/resume.
- Pure capability transitions model retry, terminal failure, and stale completion explicitly.
- README and atlas updates cover defaults, click semantics, recovery, and generation headers.

**2. Critical findings**

**Drive actions block terminal input.** The new synchronous waits in [controller_mem.go:204](/Users/xianxu/Library/Caches/sdlc/review-checkouts/ba808c009b825f3e/cmd/internal/webcast/drive/controller_mem.go:204) are reached directly through [console_drive.go:180](/Users/xianxu/Library/Caches/sdlc/review-checkouts/ba808c009b825f3e/cmd/internal/couchtty/console_drive.go:180), with menu actions executed inline at [console_notes.go:397](/Users/xianxu/Library/Caches/sdlc/review-checkouts/ba808c009b825f3e/cmd/internal/couchtty/console_notes.go:397).

A slow recorder or held-input release therefore stalls subsequent terminal input. This contradicts the plan’s explicit “without network/CDP on Couch’s input loop” requirement. A disposable regression held acknowledgment behind a barrier and failed with `input handler waits for recorder acknowledgment`.

Move **all four entry paths**—grant, deny, popup revoke, and Ctrl-click revoke—onto bounded, lifetime-owned work. Keep acknowledgment semantics and deliver results through an identity-checked terminal completion. Add a barrier test proving unrelated input remains serviceable while completion is withheld.

**3. Important findings**

None.

**4. Minor findings**

None.

**5. Test coverage notes**

- Passed: all 238 browser tests; webcast packages normally and under `-race`; Couch tests after generating the runtime bundle in the disposable checkout.
- Broadcast tests passed with `TMPDIR=/tmp`; the initial long temporary path caused a Unix-socket fixture failure.
- Prior-fix mutation checks failed as expected; restored regression tests passed.
- Headless conformance remained unverified: Chrome crashed during startup with “GPU process isn't usable”; another selected test encountered CDP EOF.
- Full `make test` was not rerun. The disposable checkout was removed; the reviewed tree remains unchanged.

**6. Architectural notes**

- **ARCH-DRY — Pass:** shared click policy and capability transitions.
- **ARCH-PURE — Pass:** transition, reducer, and presentation logic remain directly testable without IO.
- **ARCH-PURPOSE — Pass:** authorities cover both viewer roles and the operator overlay.
- **ARCH-MOCK — Pass structurally:** stateful production-seam fixtures exist; live conformance was environment-blocked.
- **ARCH-CONSTRAINTS — Flag:** synchronous drive actions violate the input-loop envelope.
- **ARCH-SECURE — Pass:** external generations are validated and intersected with existing permissions.
- **ARCH-ORDER — Pass:** cleanup attempts, admission generations, and uncertain effects retain explicit state and regression coverage.
- **ARCH-FUNERAL — Pass:** new state is cast-scoped; workers/subscriptions have cleanup paths.

**7. Plan revision recommendations**

Add a `## Revisions` entry stating that acknowledged ordinary control operations must execute outside terminal input handlers. Enumerate the four entry paths, their worker ownership and completion fencing, and the responsiveness regression test.
