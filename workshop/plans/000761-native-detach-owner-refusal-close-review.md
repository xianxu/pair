# Boundary Review — pair#761 (whole-issue close)

| field | value |
|-------|-------|
| issue | 761 — Couch native detach intermittently loses post-client ownership proof |
| repo | pair |
| issue file | workshop/issues/000761-native-detach-owner-refusal.md |
| boundary | whole-issue close |
| milestone | — |
| window | bc79cbaeae75b518dcfa789fc2bd0e40ccf06f97..e5bb4257eab53a300f3a52ed854f44b95778dd2d |
| command | sdlc close --issue 761 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T02:14:59-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The bounded retry policy, ownership fences, and typed failure-evidence flow are well covered. Two fixes remain: the socket admission check can inspect a different path from Zellij, and the new socket fixture fails under a long temporary-directory path.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: native-path-spelling-preservation
    title: |
      Preserve native socket-root spelling when checking retry admission
    detail: |
      cmd/internal/launcher/zellij_read_recovery.go:118 uses filepath.Join, which cleans symlink/.. segments that ResolveZellijSocketRoot deliberately preserves (zellij_socket.go:18–19). For root /tmp/example/alias/../sockets, where alias points to real/child, Zellij addresses real/sockets while this check addresses example/sockets. A filesystem reproduction confirmed the native socket existed while the cleaned path did not; the reverse arrangement can admit retries based on an unrelated socket. This violates the approved exact-target-socket condition (ARCH-SECURE). Append validated path components without cleaning the root, and add positive and negative symlink/parent-segment regressions.
  - id: new
    severity: Important
    kind: defect
    family: unix-socket-fixture-path-budget
    title: |
      Bound the socket fixture path independently of inherited TMPDIR
    detail: |
      cmd/internal/launcher/zellij_lookup_miss_test.go:46 creates its socket root beneath inherited TMPDIR without accounting for the Unix socket pathname limit (ARCH-CONSTRAINTS). The requested package run failed four new tests with bind: invalid argument under this review's TMPDIR; the generated paths were approximately 119 bytes. The focused tests passed with TMPDIR=/tmp. Use a short private socket root with cleanup, following existing repository socket fixtures, so ordinary test execution works under long temporary roots.
```

1. **Strengths**

   - Retry admission uses exact signatures, the existing read allowlist, and unchanged attempt/deadline bounds.
   - Evidence selection uses the complete parser input; tests cover large valid replies and observer mutation.
   - Typed evidence survives ownership refusal, operation recording, and reopening without exposing raw bytes in operator messages.
   - Persistence retains strict decoding, unknown fields, cloning, and bounded history.

2. **Critical findings**

   The socket-path defect above requires correction and regression coverage before shipping. Socket presence must describe the path the native process actually uses.

3. **Important findings**

   Repair the fixture’s pathname budget so the required package checks pass without an environment workaround.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Passed: `couchcore`, `artifactpath`, `repolint`, `zellijtest`, `zellijexec`, and `make test-affected-inventory`.
   - Full launcher run failed the four socket-fixture tests identified above.
   - Focused launcher recovery/evidence tests passed with `TMPDIR=/tmp`.
   - Code/docs/test whitespace checks passed. Whole-range whitespace diagnostics concern intentionally preserved native stdout.
   - Native evidence records retained-session reattach and exact-generation cleanup; it explicitly leaves supervisor restart qualification with #744.

6. **Architectural notes**

   | Marker | Result |
   |---|---|
   | ARCH-DRY | Pass: existing retry, runner, and codec mechanisms reused. |
   | ARCH-PURE | Pass: classifiers and bounds are pure; IO remains in adapters. |
   | ARCH-PURPOSE | Pass except the exact-socket defect above. |
   | ARCH-MOCK | Pass: stateful fixtures exercise the production query boundary. |
   | ARCH-CONSTRAINTS | Flag: socket fixture ignores the platform pathname budget. |
   | ARCH-SECURE | Flag: path cleaning changes the identity being observed. |
   | ARCH-ORDER | Pass: cancellation and generation-change paths preserve refusal and evidence. |
   | ARCH-FUNERAL | Pass: evidence uses existing bounded retention; no new durable store. |
   | ARCH-EVOLVE | Pass: additive fields preserve unknown data and strict validation. |

   Core-concept entities exist at the documented paths. Atlas updates cover the new evidence and recovery behavior. No new user command, flag, or config key requires a README update.

7. **Plan revision recommendations**

   Add a `## Revisions` entry requiring native path-spelling preservation, regressions for both false-negative and false-positive socket observations, and short socket fixtures independent of inherited `TMPDIR`.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"b3547ed7-dd71-4027-b427-696a0350f235","repository":"github.com/xianxu/pair","issue_id":"000761","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/86768c4d44ff1549","reviewed_head":"e5bb4257eab53a300f3a52ed854f44b95778dd2d","slot":"pair:5","requested_at":"2026-10-21T09:09:47.205831Z"},"phase":"finished","pid":84785,"started_at":"2026-10-21T09:09:47.367593Z","finished_at":"2026-10-21T09:14:58.735062Z","session_id":"01a15db9-4e4d-77d2-a865-c918e37d2c48","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.274185875s
head: "e5bb4257eab53a300f3a52ed854f44b95778dd2d"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
