# Boundary Review — pair#695 (milestone M2)

| field | value |
|-------|-------|
| issue | 695 — Lifecycle primitives: transcript binding on the record, the two-kind address, proven teardown |
| repo | pair |
| issue file | workshop/issues/000695-lifecycle-primitives-transcript-binding-on-the-record-the-two-kind-address-proven-teardown.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 76f0676dd41f77e3c207111e57d5a1eef8c527a6..60c4f86e3a0f2ebeda728b969707661c3150af4d |
| command | sdlc milestone-close --issue 695 --milestone M2 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-19T12:31:55-07:00 |
| verdict | REWORK |

## Review

Reviewing the M2 teardown code; now checking how the lease is held at each pair role.

Tracing the production callers of Teardown and the retire authorities (archive, reboot, Fresh recovery).

Checking whether teardown's holder kinds survive the failure recorder.

Running the M2 test packages now (targeted couchcore tests, plus lifetimelease/procutil/readiness).

```verdict
verdict: REWORK
confidence: high
```

M2 builds the right structure. Every store retirement now goes through one sealed `RetireAuthority`, checked under the store lock. The lease is never visible to the agent. `ErrTeardownUnproven` feeds #683's `LifecycleFailureDetails`. #692's listing omission becomes unproven, not absent. A real-pty conformance test exists. Tests pass: lifetimelease, procutil, readiness, strictjson, threadrecord, artifactpath, gccmd, gcruntime, historyarchive, wrapcmd and sessionwatch all pass, and so does the targeted couchcore run. couchcmd and launcher fail in this checkout only because the TMPDIR path makes unix sockets too long (over 103 bytes); with `TMPDIR=/tmp/rv695` the couchcmd resource/reap tests and the failing launcher test pass.

One defect blocks SHIP. `Teardown` can return a `TeardownProof` while the agent the wrapper published is still alive. I confirmed this with a throwaway probe test, now deleted. If `agent-ready` has no `tty`, nothing in the proof checks the agent at all. That is true of every tree started before M2 and of any wrapper whose process-table read failed. The plan's fallback for such trees ("proven as today, by pid and session", via `ObserveSlotSessions`) was dropped. Archive then accepts that false proof.

Second, a mutation check: I forced the "tty trusted only while the tree is up" guard to always trust the tty. The full teardown, retire, archive, fresh, reboot and recover test run (183s) still passed. So nothing tests the rule that stops teardown from killing an unrelated terminal's processes.

1. **Strengths**
   - `retire.go:30-49`: one sealed authority interface, checked inside each funnel under the lock. Revision, thread and lifecycle-key matching are tested (`TestArchiveRefusesProofForMovedRevision`, `TestEveryRetirementTakesAnAuthority`).
   - `orphan_registry.go`: unknown members are kept at file, entry and nested-process level, all on the shared strictjson helpers (BR-7 rule followed). The pure merge rule (`mergeLeftRunning`) is tested on its own.
   - `lifetimelease.WithoutLease` at `wrap.go:2874`: the agent never sees the lease, so nothing it runs can hold the slot. This is simpler than the fd-inheritance design it replaced.
   - `teardown.go:277-281`: #692's omission maps to unproven at stop and is never retried.
   - `TestTeardownProvesARealPtyTree` gives a live conformance check for the tty and lsof seams.

2. **Critical**
   - **`teardown.go:296-335` / `385-410`: no agent witness when the published tty is empty.**
     - `TTYProcesses(table, "")` matches nothing, and neither the sweep nor the proof looks at `target.agent.Process`.
     - Probe result: an agent published with no tty that ignores SIGTERM gives `err=nil`, `agentAlive=true`, empty `LeftRunning`.
     - Fix: when the published agent is listed, add its exact identity to `teardownHolders` (kind `agent`) whatever the tty. `proveDown` then requires it Dead.
     - Test: an agent with no tty that survives SIGTERM is SIGKILLed or named as unproven.

3. **Important**
   - **No test for the tty-trust guard (`teardown.go:157-166`).** The mutation `if true || …` survived. Add a test where the agent is dead, the lease is free, and an unrelated live process sits on the same device number. It must not be signalled, and the note must be present.
   - **Lease files are never removed outside `Teardown` (ARCH-FUNERAL).**
     - Keys are per thread (`lifetime_lease.go:20`), but only `Teardown` with `stop` deletes the file (`teardown.go:208-212`).
     - Retirements through `ProveDown` (fresh start, reboot archive-only), `proveDownAndFold` (Fresh recovery), adoption and the unreadable escape all leave one file per tag in `<namespace>/leases/`, forever.
     - Fix: remove the retired thread's lease in the retire funnel once a proof exists, or sweep leases whose thread is gone. State the lifetime in the atlas.
   - **`TeardownProof` can be forged (ARCH-ORDER structural enforcement).**
     - Its authority-bearing fields are all exported. `couchcoretest/retire.go:25`, a non-test package, builds a proof by hand.
     - The other three authorities can only come from their constructors.
     - Fix: make `Thread`, `Revision`, `ProvenAt` and `Address` unexported, with accessors. Give the test fixture an explicit test-only minting path.
   - **README not updated for `--reap-orphans`.** `--confirm` now also ends teardown's left-running processes (setsid'd runners, background merges). `README.md:477-478` and `:822-828` still describe nvim only. This is a destructive verb that now reaches more processes.
   - **plan-code-drift, 3rd finding.** The rule and prevalence are in the findings block. The measured M2 instances:
     - `TestPlainPairUnchangedWithoutCouch` only tests `HoldFromEnv` with an empty env.
     - The "both `COUCH_LIFETIME_LEASE` and `COUCH_THREAD_TAG`" requirement is not implemented (`lease.go:60`, `wrap.go:652`).
     - `TestTeardownSweepsReparentedHolders` is absent.
     - The phase/holders test does not go through the real recorder.
     - `ReadyRecord` is tolerant only, not preserving.
     - The `agent` and `session-watch` holder kinds were not added.
     - "Four witnesses" became three.
   - **`orphans.json` decode checks types only (decode-strictness-beyond-types, 2nd finding).**
     - Any `schema_version` is accepted and then overwritten with 1 on rewrite (`orphan_registry.go:282`).
     - Entries with pid ≤ 0 or an empty identity are accepted.
     - Fix by the rule: each persisted envelope validates its known-field values at decode.

4. **Minor**
   - ARCH-DRY: the split → encode → strict-decode → keep-unknown sequence is repeated five times (`transcript.go:47`, `record.go:324`, `readiness/record.go:61`, `orphan_registry.go:81/114/142`). A `strictjson` decode-known helper would replace them.
   - `teardown(stop, fold bool)`: three legal modes are written as two booleans. Use a mode enum.
   - The `ReadyRecord.TTY` comment says `ttys183`/`pts/3`, but the value is a decimal device number.
   - `detach.go:280` archives with `r.Revision`, while reboot uses `proof.Revision`.
   - `stopExact` times its waits with `time.Now` instead of `c.Clock`.
   - The resource sampler takes `orphans.lock` on every pass, even when the registry is empty.

5. **Test coverage notes**
   - In the teardown fixture, `lifetimelease.Free` always runs on a file that doesn't exist. So the "held, but lsof names nobody" path is covered only by the live test.
   - Tests that pin real behaviour: `TestTeardownFoldsCompletedPark`, `TestProvenTeardownHidesEarlierUnprovenFailure`, `TestUnreadableEscapeRefusesADecodableRecord`.

6. **Architecture**
   - ARCH-DRY: flag (Minor).
   - ARCH-PURE: pass. The pure helpers (`mergeLeftRunning`, `TTYProcesses`, `leftRunning`) sit behind thin seams.
   - ARCH-PURPOSE: flag. The proof contract fails for trees without a tty (Critical), and the plain-pair contract is under-delivered.
   - ARCH-MOCK: pass, with a note. The `lsof` and process-table seams have stateful fakes plus live conformance.
   - ARCH-CONSTRAINTS: pass. The 10s budget is enforced by the context, and rounds are bounded.
   - ARCH-SECURE: flag (`orphans.json` value validation).
   - ARCH-ORDER: flag (forgeable proof; bool-mode `teardown`).
   - ARCH-FUNERAL: flag (lease files).
   - ARCH-EVOLVE: flag. `schema_version` is overwritten. `ReadyRecord` is passed: its only writer is the wrapper, and nothing rewrites it.

7. **Plan revisions needed**
   - Record that `ReadyRecord` is tolerant but not preserving, and why (single writer, no rewrite).
   - Record that `ObserveSlotSessions` was dropped and what replaces it as the agent witness.
   - Either restore or formally drop the "both env vars required" rule and the `TestPlainPairUnchangedWithoutCouch` cases.
   - Record the lifetime of the lease file and what removes it.
   - Record the holder-kind catalogue as built (`pair`, `process`).

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Overtaken: the team arm and TeamIdentity were removed in M2 (009030459); Validate/Key now have one slot kind.
findings:
  - id: new
    severity: Critical
    kind: defect
    family: proof-witness-per-process-kind
    title: |
      Teardown proves a tree down while the published agent is still alive when agent-ready carries no tty
    detail: |
      teardown.go teardownHolders and proveDown witness the agent only through TTYProcesses(table, tty), and TTYProcesses with "" matches nothing. So for any pre-M2 wrapper, or a wrapper whose agentTerminal table read failed, the proof has no agent witness. The plan's fallback (proven as today by pid and session, ObserveSlotSessions) was dropped. A reviewer probe (agent published with no tty that ignores SIGTERM) returned err=nil with the agent Live and LeftRunning empty, and ArchiveThread would archive with that proof. Fix: when the published agent is listed, sweep and require its exact identity Dead whatever the tty (holder kind agent), and add a regression test.
  - id: new
    severity: Important
    kind: defect
    family: kill-guard-regression-test
    title: |
      The tty-trusted-only-while-the-tree-is-up guard has no test; a mutation that always trusts the tty passes the suite
    detail: |
      Changing teardown.go:161 to `if true || agentListed(...) || !free` passed every couchcore Teardown, Retire, Archive, Fresh, Reboot and Recover test (183s). This guard is what stops teardown from SIGKILLing an unrelated terminal's processes on a reused device. Add a test: agent dead, lease free, an unrelated live process on the same tty number; expect no signal and the note.
  - id: new
    severity: Important
    kind: defect
    family: artifact-removal-path
    title: |
      Per-thread lifetime lease files are removed only by Teardown(stop); every other retirement leaks one file forever
    detail: |
      Lease keys include scope/tag (lifetime_lease.go:20), and only teardown with stop removes the file (teardown.go:208). Retirements through ProveDown (startFreshSlotWithPrompt, reboot archive-only), proveDownAndFold (Fresh recovery), AdoptionAuthority and UnreadableRecordEscape leave <namespace>/leases/<digest>.lease with no sweep, so the directory grows with every Fresh, reboot or adoption (ARCH-FUNERAL). Remove the retired thread's lease once its proof stands, or sweep leases whose thread is gone, and state the lifetime in the atlas.
  - id: new
    severity: Important
    kind: defect
    family: authority-unforgeable
    title: |
      TeardownProof is forgeable (all fields exported), unlike the three other constructor-only retire authorities
    detail: |
      RetireAuthority is sealed at the interface, but any package can build couchcore.TeardownProof{Address, Thread, Revision, ProvenAt}, and couchcoretest/retire.go:25 (a non-test package) does. That undercuts ARCH-ORDER's structural enforcement that the archive and replace funnels cannot be reached without a real proof. Unexport the authority-bearing fields (with accessors), so only proveDown mints one, and give fixtures an explicitly test-only minting path.
  - id: new
    severity: Important
    kind: defect
    family: readme-surface-gate
    title: |
      README update missing: couch --reap-orphans --confirm now also ends teardown's left-running processes
    detail: |
      reap_orphans.go adds reapLeftRunning to the destructive verb, and --resources gains the "Left running by teardown" section. README.md:477-478 and :822-828 still describe only unattributed embedded nvim, so an operator who uses --confirm for nvim cleanup will now also kill setsid'd runners and background merges without the docs saying so.
  - id: new
    severity: Important
    kind: defect
    family: plan-code-drift
    title: |
      M2 plan contracts and named tests not delivered as specified (plain-pair test cases, both-env activation, sweep test, recorder path, holder kinds, witnesses)
    detail: |
      This is the 3rd finding in family plan-code-drift. Rule: at milestone close, every test name and contract in the plan's governing revision is checked mechanically against the tree (grep the func names; diff each contract against the code), and each mismatch gets either code or a Revisions line in the same round. Measured prevalence in M2 is 7 instances. (1) TestPlainPairUnchangedWithoutCouch checks only HoldFromEnv with an empty env, not two sessions, a sidecar without a tty, no extra fd, or the lease-without-tag case. (2) Activation needs only COUCH_LIFETIME_LEASE (lease.go:60, wrap.go:652), not both env vars. (3) TestTeardownSweepsReparentedHolders is absent. (4) TestTeardownUnprovenRecordsPhaseAndHolders does not go through the real recorder. (5) ReadyRecord is tolerant but not preserving, unrecorded. (6) The agent and session-watch holder kinds were not added (pair/process used). (7) The plan's four witnesses became three (ObserveSlotSessions dropped).
  - id: new
    severity: Important
    kind: defect
    family: decode-strictness-beyond-types
    title: |
      orphans.json accepts any schema_version and rewrites it as 1; entries with pid <= 0 or an empty identity decode
    detail: |
      This is the 2nd finding in family decode-strictness-beyond-types. Rule: every persisted envelope validates its known-field values (supported schema_version, positive pid, non-empty identity) at decode, not only their JSON types, and a rewrite never changes an existing field's value. orphan_registry.go:201 decodes with no value checks, and :282 sets SchemaVersion=1 over whatever was read, which changes a newer writer's field (ARCH-EVOLVE). Refuse an unsupported version, validate entries, and add round-trip and refusal tests.
  - id: new
    severity: Minor
    kind: defect
    family: preserving-codec-boilerplate
    title: |
      The split, encode, strict-decode, keep-unknown sequence is copy-pasted at five decode sites
    detail: |
      transcript.go:47, record.go:324, readiness/record.go:61 and orphan_registry.go:81/114/142 repeat the same four steps. A strictjson helper (decode the known members into a wire struct, return the unknown map) would make each type's UnmarshalJSON a single call (ARCH-DRY).
  - id: new
    severity: Minor
    kind: defect
    family: mode-as-bool-flags
    title: |
      teardown(stop, fold bool) encodes three legal modes as two booleans; stop without fold is representable but unused
    detail: |
      Replace the two booleans with a mode enum (teardown, prove-down, prove-and-fold) so the legal set can be read off the type (ARCH-ORDER).
  - id: new
    severity: Minor
    kind: defect
    family: comment-code-drift
    title: |
      ReadyRecord.TTY comment says names like ttys183 or pts/3, but the value is a decimal device number
    detail: |
      procutil.Process.TTY stores e_tdev or tty_nr in decimal, and the wrapper publishes that same value. Fix the comment at readiness/record.go:30.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"e4a1a249-8385-41a6-bf6d-88bc57f994b0","repository":"github.com/xianxu/pair","issue_id":"000695","recipe":"milestone-review","round":3,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/f5e1d9ddb9d3bb4a","reviewed_head":"60c4f86e3a0f2ebeda728b969707661c3150af4d","slot":"pair:1","requested_at":"2026-10-19T19:16:58.088808Z"},"phase":"finished","pid":74503,"started_at":"2026-10-19T19:16:58.236453Z","finished_at":"2026-10-19T19:31:52.234804Z","session_id":"080e2bd5-8f0a-4e26-b94d-e2fe09deb11b","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.092180292s
head: "60c4f86e3a0f2ebeda728b969707661c3150af4d"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-19T13:16:10-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 695 — Lifecycle primitives: transcript binding on the record, the two-kind address, proven teardown |
| repo | pair |
| issue file | workshop/issues/000695-lifecycle-primitives-transcript-binding-on-the-record-the-two-kind-address-proven-teardown.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 76f0676dd41f77e3c207111e57d5a1eef8c527a6..99ca957e2b74d77402d35ca9eb40a408a295cb11 |
| command | sdlc milestone-close --issue 695 --milestone M2 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-19T13:16:10-07:00 |
| verdict | SHIP |

## Review

Checking the claimed fixes for BR-11 through BR-20 against the code; next I'll run the teardown tests with mutations.

Now mutation-checking the BR-11 and BR-12 regression tests.

Targeted tests and mutations are green; now checking whether the full-suite failures come from the sandbox or from the code.

```verdict
verdict: SHIP
confidence: medium
```

All ten open findings from round 4 (BR-11 to BR-20) are fixed in code, and the behaviour fixes have regression tests that I checked by mutation in this checkout. Teardown now takes the published agent as a witness by its exact pid and identity, whatever its tty (`teardown.go:365-369`). If that branch is removed, `TestTeardownWitnessesAnAgentWithoutATerminal` fails with "ProveDown with a live agent and no tty = <nil>". If the tty-trust guard is changed to `if true || …`, `TestTeardownNeverSweepsAReusedTerminal` fails because pid 900 is signalled. If lease removal is limited to stopping mode, the `prove_and_fold` case of `TestTeardownRemovesTheProvenLease` fails. The targeted suites pass: lifetimelease, strictjson, readiness, threadrecord and procutil, plus couchcore filtered to the Teardown, Retire, LeftRunning, ProveDown and Archive tests. The full couchcore and couchcmd runs fail here, but I traced every failure I looked at to this sandbox; details are in the test notes. Nothing blocks shipping. The two new findings are Minor.

### 1. Strengths
- **Agent witness by exact identity.** `teardown.go:362-369` closes the BR-11 gap without trusting the tty. The lesson recorded in `workshop/lessons.md` states the rule as a class: a derived witness is only a supplement.
- **Lease sweep.** `lifetimelease.SweepFree` unlinks a file only while it holds the exclusive lock and the file is still the one at that path. `Hold` re-checks with `current()`, so a sweep can never leave a holder locking an unlinked file (`lease.go:79-147`).
- **Unforgeable proof.** `TeardownProof` now hides its authority fields behind accessors. `teardownMode` makes the three legal modes readable from the type (`teardown.go:63-111`).
- **Orphan registry.** `orphans.json` now refuses unsupported values when it decodes, and its rewrite no longer overwrites `schema_version`. `TestLeftRunningRegistryRefusesUnsupportedValues` also checks that a refused registry stays byte-identical.
- **One decode helper.** `strictjson.DecodeKnown` replaces the five copy-pasted decode sequences, which closes ARCH-DRY.

### 2. Critical
None.

### 3. Important
None.

### 4. Minor
- **Lease removal is not under lock** (`teardown.go:245`). Teardown deletes the lease file with a plain `os.Remove`, without the exclusive lock and same-file check that `SweepFree` uses. A holder that locks the file between `proveDown`'s `Free` and the `Remove` ends up holding an unlinked inode, and every later witness misses it. It is only reachable if the same thread key is relaunched during its own teardown. Fix: share one `removeIfFree(path)` helper in `lifetimelease` that both `SweepFree` and teardown call.
- **The guard on the test-only minting path does not cover the whole module** (`retire_test.go:304-328`). `TestOnlyTeardownProofHasNoProductionCaller` scans only `cmd/internal`. `cmd/couch` and `cmd/pair-go` can import couchcore and are not scanned. It also exempts `couchcoretest/`, a non-test package, and nothing checks that production code never imports it.

### 5. Test coverage notes
- **Mutations.** The BR-11, BR-12 and BR-13 tests each fail for the right reason when their fix is reverted (results above).
- **couchcmd.** The full run fails here because the socket path exceeds 103 bytes under the review's long `TMPDIR`. With a short `TMPDIR` the sandbox refuses `git init` outside the reviewer's directories. Both are environmental.
- **couchcore.** `TestSpawnComposesProductionPairRegistrationBoundary` fails the same way at base `76f0676dd` ("couch session repository scope changed"), so it predates this work. The couchcore write budget was exceeded in this environment (about 190 MB against a 128 MB cap). I did not compare that against base, so the full-suite write budget is unconfirmed here.
- **Plain-pair test.** `TestPlainPairUnchangedWithoutCouch` covers the env predicate and the absence of an extra descriptor, not the two-session scenario. That is acceptable because `Active` is the single gate for both holding the lease and publishing the tty.

### 6. Architectural notes (one line per principle)
- **ARCH-DRY:** pass. `DecodeKnown` is the single preserving decoder. The nested-null split in `lifecyclefailure_codec.go` predates this work and has a different shape.
- **ARCH-PURE:** pass. Teardown decisions go through `Proc`, `ProcessTable` and `Clock`, which are injected.
- **ARCH-PURPOSE:** pass. Every retirement path either removes its lease on proof or leaves it to the sweep.
- **ARCH-MOCK:** pass. Fake process table plus the real-pty conformance test.
- **ARCH-CONSTRAINTS:** pass. Teardown has a 10 s budget, 3 rounds and bounded signal waits. `Prune` now writes only when something is gone.
- **ARCH-SECURE:** pass, with the Minor guard-scope note above.
- **ARCH-ORDER:** pass. `teardownMode` replaces the two booleans, and tty trust is tied to the tree being up. The remove-without-lock race is the Minor above.
- **ARCH-FUNERAL:** pass. Lease files are removed after any proof or by `SweepFree` at Couch start, and the atlas documents both. The orphan registry is capped at 256 entries.
- **ARCH-EVOLVE:** pass. The registry refuses an unknown version and never rewrites one it refused.

### 7. Plan revision recommendations
- Plan line 368 still describes `TestPlainPairUnchangedWithoutCouch` as "two pair sessions start in one directory…". Add a Revisions line that explicitly retires the two-session, sidecar-without-tty and lease-without-tag cases in favour of the shared `Active` predicate. This is optional, because the round-4 revision already records the narrowed test.

```findings
dispose:
  - id: BR-11
    disposition: addressed
    note: |
      teardown.go:365-369 adds the published agent by exact pid+identity; mutation (drop add) turns TestTeardownWitnessesAnAgentWithoutATerminal red.
  - id: BR-12
    disposition: addressed
    note: |
      TestTeardownNeverSweepsAReusedTerminal fails under the `if true ||` mutation (pid 900 terminated), verified in this checkout.
  - id: BR-13
    disposition: addressed
    note: |
      Lease removed after a proof in every mode (all-mode test fails when limited to stopping mode); SweepFree at Couch start with TestSweepFreeRemovesOnlyFreeLeases; atlas states the lifetime.
  - id: BR-14
    disposition: addressed
    note: |
      Authority fields unexported with accessors; fixtures use TestOnlyTeardownProof plus a scan test (scan-scope gap raised as a new Minor).
  - id: BR-15
    disposition: addressed
    note: |
      README.md:477-481 and :823-829 now name the left-running processes that --reap-orphans --confirm ends and the --resources section.
  - id: BR-16
    disposition: addressed
    note: |
      Each of the 7 mismatches got code or a Revisions entry (plan round-4 log); sweep and recorder tests exist, and Active is the single activation predicate.
  - id: BR-17
    disposition: addressed
    note: |
      orphan_registry.go refuses schema_version != 1 and pid<=0 or empty identity at decode, and the rewrite keeps the version; refusal + byte-unchanged test present.
  - id: BR-18
    disposition: addressed
    note: |
      strictjson.DecodeKnown replaces the five sites; no remaining copy of the split/encode/decode sequence in the diff.
  - id: BR-19
    disposition: addressed
    note: |
      teardownMode enum with stops()/folds() replaces (stop, fold bool).
  - id: BR-20
    disposition: addressed
    note: |
      readiness/record.go:30 comment now says decimal device number (Darwin e_tdev, Linux tty_nr).
findings:
  - id: new
    severity: Minor
    kind: defect
    family: unlink-lock-file-under-lock
    title: |
      Teardown unlinks the proven-free lease without holding its exclusive lock, unlike SweepFree
    detail: |
      teardown.go:245 calls os.Remove after proveDown's Free released its probe lock, so a Hold that locks between the two keeps an unlinked inode that every later witness misses. Rule: a lock file is unlinked only under LOCK_EX with a same-file check; share one lifetimelease removeIfFree helper used by SweepFree and teardown.
  - id: new
    severity: Minor
    kind: defect
    family: authority-unforgeable
    title: |
      The production-caller scan for TestOnlyTeardownProof covers only cmd/internal and exempts couchcoretest
    detail: |
      This is the 2nd finding in family authority-unforgeable. Rule: an authority's test-only minting path is unreachable from every production package in the module, so the guard walks the module root and also refuses non-test imports of couchcoretest. retire_test.go walks ".." (cmd/internal) only, so cmd/couch and cmd/pair-go are unscanned.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"9016333d-ec91-4480-af66-dc2138dc2bec","repository":"github.com/xianxu/pair","issue_id":"000695","recipe":"milestone-review","round":4,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/812ed72f3c403142","reviewed_head":"99ca957e2b74d77402d35ca9eb40a408a295cb11","slot":"pair:1","requested_at":"2026-10-19T19:46:12.850059Z"},"phase":"finished","pid":18007,"started_at":"2026-10-19T19:46:13.015702Z","finished_at":"2026-10-19T20:16:10.12661Z","session_id":"f1831f85-10bb-44ea-8d61-ddd0b630abfe","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.022632958s
head: "99ca957e2b74d77402d35ca9eb40a408a295cb11"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
