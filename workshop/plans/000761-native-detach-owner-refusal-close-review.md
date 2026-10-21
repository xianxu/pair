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

---

## Re-review — 2026-10-21T02:34:26-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 761 — Couch native detach intermittently loses post-client ownership proof |
| repo | pair |
| issue file | workshop/issues/000761-native-detach-owner-refusal.md |
| boundary | whole-issue close |
| milestone | — |
| window | bc79cbaeae75b518dcfa789fc2bd0e40ccf06f97..f7e4c03f134362d8173bb73725f014677aa464be |
| command | sdlc close --issue 761 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T02:34:26-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The retry and evidence plumbing are well bounded, and BR-2 is addressed. BR-1 remains open: explicit socket-root spelling is preserved, but the TMPDIR fallback still cleans symlink/parent segments before retry admission. A filesystem regression on the pinned head reproduced both false rejection and false admission.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      The explicit-root regressions pass and fail when its fix is removed, but family native-path-spelling-preservation remains incomplete. zellij_read_recovery.go:114 calls ResolveZellijSocketRoot, whose TMPDIR fallback at zellij_socket.go:28 still uses filepath.Join. With ZELLIJ_SOCKET_DIR unset and TMPDIR containing alias/../sockets, a disposable pinned-head regression observed native-present/admission-false and native-absent/admission-true. Preserve native spelling across all three root sources: explicit override, Linux runtime directory, and temporary-directory fallback; add positive and negative fallback regressions.
  - id: BR-2
    disposition: addressed
    note: |
      All three launcher Unix listener sites use the budgeted private allocator. The long-TMPDIR regression passes and fails with bind: invalid argument when the lookup fixture returns to inherited temporary allocation. The full launcher package also passed under a 261-byte inherited TMPDIR; temporary review checkouts were removed.
```

1. **Strengths**

   Retry admission validates the complete signature and retains the existing attempt/deadline bounds. Evidence capture preserves full-input parser decisions, independent byte ownership, and concise operator errors. Persistence tests exercise an actual owner refusal through recording, Save, and reopen.

2. **Critical findings**

   **BR-1 remains open — ARCH-SECURE, ARCH-PURPOSE.** At `cmd/internal/launcher/zellij_read_recovery.go:114`, the fallback root has already been cleaned by `cmd/internal/launcher/zellij_socket.go:28`. For `TMPDIR=<base>/alias/../sockets`, where `alias` points to `<base>/real/child`, the native path resolves beneath `real/sockets`; the check resolves beneath `<base>/sockets`.

   Apply the spelling-preservation rule throughout root construction, then test both socket arrangements through actual retry admission. The existing regressions cover only the explicit override.

3. **Important findings**

   None.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed: full launcher under long TMPDIR; focused launcher race tests; native-query persistence tests; full `zellijtest`; artifactpath; repolint; affected-reader inventory.

   Both prior fixes have meaningful mutation tests. The additional fallback regression fails on the pinned head. Whitespace checks pass outside captured evidence; native stdout captures intentionally retain trailing spaces. The required recorded suite remains a separate landing prerequisite.

6. **Architectural notes**

   - **ARCH-DRY — pass:** existing decoders, runner, and failure store are reused.
   - **ARCH-PURE — pass:** classification, bounding, and normalization remain directly testable.
   - **ARCH-PURPOSE — flag:** BR-1’s path-source enumeration missed the temporary-directory fallback.
   - **ARCH-MOCK — pass:** stateful fixtures share the production boundary; scheduled native conformance exists.
   - **ARCH-CONSTRAINTS — pass:** retry, deadline, evidence, history, and socket-path bounds are exercised.
   - **ARCH-SECURE — flag:** fallback spelling can select a different socket.
   - **ARCH-ORDER — pass:** ownership agreement, generation fences, and cancellation remain enforced.
   - **ARCH-FUNERAL — pass:** evidence uses existing bounded retention; fixture cleanup is explicit.
   - **ARCH-EVOLVE — pass:** additive fields retain strict decoding and unknown-field preservation.

   Atlas changes cover the new diagnostic surface. No new command, flag, or configuration key requires a README update.

7. **Plan revision recommendations**

   Add a `## Revisions` entry stating that native spelling must survive **explicit, runtime-directory, and temporary-directory root construction**, with positive and negative filesystem regressions for every applicable branch. Correct the current BR-1 completion claim accordingly.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"90800df8-3dc0-4e8e-b16c-44bc6b4e8105","repository":"github.com/xianxu/pair","issue_id":"000761","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/0cfbfc9bba29a20d","reviewed_head":"f7e4c03f134362d8173bb73725f014677aa464be","slot":"pair:5","requested_at":"2026-10-21T09:29:49.668376Z"},"phase":"finished","pid":16967,"started_at":"2026-10-21T09:29:49.854414Z","finished_at":"2026-10-21T09:34:25.288721Z","session_id":"01a15dcb-a7a7-7c11-8b4e-cc36cb9adf62","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.609373375s
head: "f7e4c03f134362d8173bb73725f014677aa464be"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-21T02:49:06-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 761 — Couch native detach intermittently loses post-client ownership proof |
| repo | pair |
| issue file | workshop/issues/000761-native-detach-owner-refusal.md |
| boundary | whole-issue close |
| milestone | — |
| window | bc79cbaeae75b518dcfa789fc2bd0e40ccf06f97..e26e24b0dc25a58f90a8f91beb9990b155631397 |
| command | sdlc close --issue 761 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T02:49:06-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned implementation matches the approved recovery and evidence contracts. BR-1 is verified addressed, with regression tests that fail when path cleaning is restored. No new blocking findings. The separately required recorded suite remains a landing prerequisite.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      zellij_socket.go:22,28 and zellij_read_recovery.go:120 share the spelling-preserving append helper. All six positive/negative filesystem cases pass; restoring filepath.Join through a disposable Go overlay makes all six fail, including actual retry-count assertions.
  - id: BR-2
    disposition: addressed
    note: |
      Remains addressed. The shared socket-fixture allocator and long-TMPDIR regression pass in the full launcher run.
```

1. **Strengths**

   - Recovery remains restricted to exact signatures, allowlisted reads, three attempts, and the original deadline (`zellij_read_recovery.go:85`, `zellij.go:199`).
   - Full parser results select diagnostic captures; cancellation and generation failures retain evidence without granting ownership (`session_owner_os.go:117`).
   - Persistence tests exercise refusal → operation recorder → save → reopen, checking exact raw bytes and concise operator text (`nativequery_failure_test.go:115`).

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Passed:

   - Full launcher, zellijtest, artifactpath, and repolint packages.
   - Focused couchcore native-evidence, recovery, and lifecycle-failure tests.
   - Focused launcher race tests.
   - `make test-affected-inventory`.
   - BR-1 mutation verification: restoring cleaning produces the expected failures.

   Native controls were inspected as retained evidence, not rerun. Full Couch supervisor qualification remains explicitly assigned to #744. Whitespace-check warnings are confined to exact captured native stdout.

6. **Architectural notes**

   - **ARCH-DRY — pass:** One helper preserves spelling across all retry-root sources and final socket assembly.
   - **ARCH-PURE — pass:** Classification and evidence bounding remain pure; subprocess and filesystem work stay in adapters. Core-concept classifications match the implementation.
   - **ARCH-PURPOSE — pass:** Both approved signatures and operation-owned evidence are delivered.
   - **ARCH-MOCK — pass:** Stateful fixtures exercise production boundaries; native conformance infrastructure remains present.
   - **ARCH-CONSTRAINTS — pass:** Retry, deadline, metadata, stream, and history bounds are enforced and tested.
   - **ARCH-SECURE — pass:** Exact socket identity gates only another read; unknown evidence retains refusal.
   - **ARCH-ORDER — pass:** Cancellation, generation changes, collector failures, and exhaustion preserve existing authority fences.
   - **ARCH-FUNERAL — pass:** Evidence uses existing bounded history; fixture resources have explicit cleanup.
   - **ARCH-EVOLVE — pass:** Additive fields use the preserving codec, with strict known-field validation.

   Atlas updates cover the new diagnostic surface. No new command, flag, keybinding, or configuration surface requires a README update.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"83b2a3fe-ae96-4ea3-8c89-4f93fbc0195e","repository":"github.com/xianxu/pair","issue_id":"000761","recipe":"milestone-review","round":3,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/64d818a8f6a0f26a","reviewed_head":"e26e24b0dc25a58f90a8f91beb9990b155631397","slot":"pair:5","requested_at":"2026-10-21T09:46:06.548921Z"},"phase":"finished","pid":90621,"started_at":"2026-10-21T09:46:06.713372Z","finished_at":"2026-10-21T09:49:05.243533Z","session_id":"01a15dda-8fc1-7931-acc6-c48beed95590","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.835234042s
head: "e26e24b0dc25a58f90a8f91beb9990b155631397"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-21T03:56:00-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 761 — Couch native detach intermittently loses post-client ownership proof |
| repo | pair |
| issue file | workshop/issues/000761-native-detach-owner-refusal.md |
| boundary | whole-issue close |
| milestone | — |
| window | d17806fe58b57b4a211968036cfa0a2a96f2a406..96e8bc0d3416e030877ffc1ac46f86ba675ee592 |
| command | sdlc close --issue 761 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T03:56:00-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The narrow interdiff preserves both sides of the inventory conflict without changing product code, approved scope, reader classifications, or hashes. No new defects found. Previously disposed findings remain closed.

```findings
{}
```

1. **Strengths**
   - Conflict resolution preserves the exact semantic union of both sides at `tests/affected-go-readers.json:155` and `:458`.
   - Strict parsing confirms no duplicate JSON keys across all 471 reader entries.
   - Spec, Done when, and implementation Plan remain unchanged.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes:** `make test-affected-inventory`, strict duplicate-key parsing, semantic union comparison, and `git diff --check` passed. Product tests were not rerun because product code is unchanged. This review does not certify the separately required recorded suite.

6. **Architectural notes**
   - **ARCH-DRY — pass:** No duplicated behavior introduced.
   - **ARCH-PURE — pass:** No logic or IO boundaries changed.
   - **ARCH-PURPOSE — pass:** Resolution retains both inventories’ intent.
   - **ARCH-EVOLVE — pass:** Existing schema, classifications, hashes, and all entries are preserved.

   No new architectural or user-facing surface requires atlas or README changes.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"3967cf31-35b0-4f15-b184-afe2066354e4","repository":"github.com/xianxu/pair","issue_id":"000761","recipe":"fix-review","round":4,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/520d7aef20807e20","reviewed_head":"96e8bc0d3416e030877ffc1ac46f86ba675ee592","slot":"pair:5","requested_at":"2026-10-21T10:55:04.179761Z"},"phase":"finished","pid":26146,"started_at":"2026-10-21T10:55:04.367772Z","finished_at":"2026-10-21T10:55:59.609263Z","session_id":"01a15e19-b21a-7ca1-b599-d553451bb6e8","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.465121375s
head: "96e8bc0d3416e030877ffc1ac46f86ba675ee592"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
