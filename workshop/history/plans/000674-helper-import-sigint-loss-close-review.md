# Boundary Review — pair#674 (whole-issue close)

| field | value |
|-------|-------|
| issue | 674 — Startup SIGINT swallowed by importlib weakref callback before helper handler |
| repo | pair |
| issue file | workshop/issues/000674-helper-import-sigint-loss.md |
| boundary | whole-issue close |
| milestone | — |
| window | 8c302ba8d673a9fcffa0ffeadd433091e5ae5a26..902e3b20a057a77c6375d270a655cdb285d971bb |
| command | sdlc close --issue 674 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-18T13:44:32-07:00 |
| verdict | SHIP |

## Review

Tests pass (7 OK). I'm now reverting the fix in the disposable checkout to confirm the regression test goes red without it.

```verdict
verdict: SHIP
confidence: high
```

This change fixes the bug it targets and the regression test is real. The launcher in `tests/without-ancestors-test.py:49` now blocks SIGINT and SIGTERM before starting the helper and restores its own mask right afterwards (`:56`). The helper unblocks both only after its `cancel` handlers are installed (`tests/without-ancestors.py:38`). A cancellation sent during interpreter startup therefore stays pending in the kernel and reaches `cancel`. It never becomes a KeyboardInterrupt that importlib or zipimport could swallow. I ran the suite at the pinned head on macOS with Python 3.14.5: 7 tests OK in 2.8 s. I then reverted the fix three ways in the disposable checkout, and each time the regression went red:
- **Launcher block removed:** 3 failures on the `blocked=[2, 15]` precondition.
- **Launcher block and precondition both removed:** the gated tests still fail on behaviour, `-2 != 130` and `-15 != 143`. So the test checks what happens, not just how the fixture is set up.
- **Helper unblock removed:** every startup-cancellation subcase fails with the helper alive past the 3 s bound, which is the symptom seen in the field.

Nothing blocks SHIP. The findings below are Minor: one stale test assertion that could now be stricter, and some diagnostics and coverage claims that say slightly more than the code delivers.

1. **Strengths**
   - The fix makes the startup ordering structural instead of relying on timing. The kernel holds the signal pending, so no import order can matter, and no timeout was widened and no retry added. This matches the Spec.
   - The gate test controls the interleaving deterministically (`tests/without-ancestors-test.py:197-200`): the signal is sent while the child is held at the gate, and only then is the gate released.
   - The #598 SIG_IGN case and the new SIGINT/SIGTERM case share one helper, `check_cancellation_before_helper_entry`, instead of being copies (ARCH-DRY).
   - The test now requires exit status `128 + signal` (`:203`), which replaces the old weak `assertNotEqual(code, 0)`.
   - Unblocking before `os.fork()` means the worker and the command it runs inherit an unblocked mask, so the command's own signal behaviour is unchanged.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - `tests/without-ancestors-test.py:142-145`: the comment "Python startup can itself translate SIGINT into exit 1" is now stale, and the test still only checks for a nonzero exit. `Popen` returns only after the exec succeeds, so the child is always blocked by then and `test_cancellation_during_startup` can assert `128 + cancellation`. That is the test that failed in the field, so it should be the strictest one.
   - `tests/without-ancestors-test.py:78`: `caller_blocked` is read after the launcher has restored its mask, so in normal use it always shows `[]`. It cannot show whether the child was started blocked. Only the gate bootstrap reports the child's own mask; ordinary helper failures do not. In practice the two failure classes are told apart by `caller_SIGINT` and the captured "Exception ignored … KeyboardInterrupt" text, so the new field adds little.
   - `tests/without-ancestors-test.py:177`: when `ignore_in_caller` is false, `signal.signal(SIGINT, signal.getsignal(SIGINT))` does nothing. It would also raise TypeError if `getsignal` returned `None`.
   - `tests/without-ancestors.py:38`: the helper unblocks both signals unconditionally, even for a caller that blocked them on purpose. Harmless for the callers that exist today, but undocumented as a contract.
   - The other callers, `cmd/pair-go/main_test.go:359` and `tests/pair-embedded-runtime-test.sh:123`, do not block before launching the helper, so they keep the startup window. That is within the issue's scope, since the issue is about the fixture, but the helper comment could say that only the fixture launcher is covered.

5. **Test coverage notes**
   - Both halves of the fix fail tests when removed (results above).
   - The test does not reproduce the importlib weakref swallow itself. It shows that the whole window before the handler exists is closed for every swallow site at once, which is the right invariant.
   - I did not re-run the 25-repeat stability claim; the single full run passed.

6. **Architectural notes**
   - **ARCH-DRY:** pass. The gated scenario is shared.
   - **ARCH-PURE:** pass / N/A. This is a process-lifetime fixture whose IO is inherent.
   - **ARCH-PURPOSE:** pass. The fixture flake is fixed for both observed swallow sites (#650, #659/#671); the other callers are noted under Minor.
   - **ARCH-MOCK:** N/A. It drives real processes on purpose, with no external service.
   - **ARCH-CONSTRAINTS:** pass. The 3 s and 2 s bounds are unchanged.
   - **ARCH-SECURE:** N/A. No untrusted input or secrets.
   - **ARCH-ORDER:** pass. The ordering is now made explicit through the pending-signal mechanism, and the test has a seam that controls it.
   - **ARCH-FUNERAL:** pass. The new pipes are closed by registered cleanups and no durable artifacts are created.
   - **Docs:** `atlas/index.md` is updated. No README change is needed because there is no new user-facing surface.

7. **Plan revision recommendations**
   - The fourth Plan item says the child reports "its own disposition and mask". Only the gate bootstrap does that, so the item should be qualified to say so (see the `caller_blocked` note above).

```findings
findings:
  - id: new
    severity: Minor
    kind: defect
    family: stale-weak-assertion-after-fix
    title: |
      test_cancellation_during_startup keeps a stale "exit 1" comment and only asserts nonzero; it can now require 128+signal
    detail: |
      With launch-time blocking, Popen returns post-exec with the signals pending, so cancel always runs; tighten tests/without-ancestors-test.py:142-145 to assertEqual(code, 128 + cancellation) and drop the stale comment.
  - id: new
    severity: Minor
    kind: defect
    family: diagnostic-reports-wrong-moment
    title: |
      caller_blocked diagnostic is sampled after the mask is restored, so it never shows the launch-time child mask
    detail: |
      tests/without-ancestors-test.py:78 always reports the restored caller mask; only the gate bootstrap reports the child's mask, so the Plan's diagnostics claim is narrower than stated.
  - id: new
    severity: Minor
    kind: defect
    family: helper-contract-undocumented
    title: |
      Helper unconditionally unblocks SIGINT/SIGTERM, and non-fixture launchers (main_test.go, pair-embedded-runtime-test.sh) keep the startup window
    detail: |
      tests/without-ancestors.py:38 overrides any caller's deliberate block; other consumers do not block before launch, so the comment should say only the fixture launcher closes the window.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"dacf75bc-0f71-4c10-ae54-64900e1aa5a9","repository":"github.com/xianxu/pair","issue_id":"000674","recipe":"milestone-review","round":1,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/f77e15398001ad86","reviewed_head":"902e3b20a057a77c6375d270a655cdb285d971bb","slot":"pair:2","requested_at":"2026-10-18T20:41:48.34433Z"},"phase":"finished","pid":43638,"started_at":"2026-10-18T20:41:48.57428Z","finished_at":"2026-10-18T20:44:31.536232Z","session_id":"f0f5bd0f-ca12-4ffe-9fa7-c3aad7c60c68","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.956751333s
head: "902e3b20a057a77c6375d270a655cdb285d971bb"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
