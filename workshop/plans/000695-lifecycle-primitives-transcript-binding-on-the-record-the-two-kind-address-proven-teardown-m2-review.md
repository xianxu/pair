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
