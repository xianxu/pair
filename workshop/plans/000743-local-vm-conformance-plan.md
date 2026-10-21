---
collaboration-mode: discussion
---

# Local VM conformance implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Provision one refreshed machine-wide macOS test image and a small command runner, then put Pair's existing native conformance through nightly burn-in without weakening required PR conformance.

**Architecture:** `pair vm run --checkout PATH -- COMMAND ARGS...` owns a private checkout snapshot, disposable Tart clone, guest HOME, artifacts and one machine-wide VM position for its whole operation. Pair supplies the first conformance command; SDLC remains the exact-head local-check/log/status owner. Initial delivery leaves landing activation off; ariadne#509 and promotion #745 own activation after seven clean nights.

**Tech stack:** Go CLI/OS adapters, Tart 2.32.1 (installed), APFS clonefile, Tart guest-agent command transport, shell provisioning, Python conformance evidence checker, launchd nightly trigger, existing SDLC local checks.

## Approved scope and sequencing

The issue's 14:00/14:15 revisions supersede its original per-repo `.tart` hook proposal. The authenticated 14:19 hub decision additionally approves initial infrastructure/nightly delivery, then a separate activation issue. `.tart/` remains interactive development infrastructure.

1. Refresh/provision and qualify the image first. Do not reuse four-month-old `pair-test` as the qualified base. Preserve that development VM; create a fresh purpose-built image from a newly resolved, digest-pinned base.
2. Supply the generic machine-wide runner and Pair's runnable local conformance, initially invoked manually/nightly only.
3. Retain required GitHub PR conformance throughout burn-in. Do not demote the workflow or declare landing-required conformance in this issue.
4. Promotion requires seven clean nights, ariadne#509's exact train hook, and operator acceptance/label. No auto-promotion or retry of a flake.
5. #744 is currently unowned. Existing native terminal and lifecycle targets can burn in now; genuine disposable-Couch/Claude/Codex cases adopt #744 when available. Installing CLIs is not evidence that those agents were tested. Qoder credit availability is reported explicitly.

## Measured starting point

- `tart list --format json`: `pair-test` stopped, last accessed 2026-06-09; 30 GB used / 50 GB virtual disk. Cached Tahoe base was last accessed in September; refresh from upstream before pinning it.
- `tart get pair-test --format json`: four CPUs, 8192 MiB RAM. Host has 12 logical CPUs and 96 GiB RAM; data volume had 375 GiB available at planning.
- Current workflow: Zellij 0.45.1 with verified binary checksum, Go from go.mod (1.26.3), Node 22, Python 3.12, Homebrew Neovim. `tic`, `infocmp`, compiler tools, Git/npm and xterm lockfile dependencies are also needed.
- Existing `test-native-terminal-ci` rejects missing/skipped required cases. `test-couch-zellij-live` is plain Go test and needs equivalent execution completeness checks.
- Existing Tart setup swallows final SSH status and treats hooks as best-effort; it cannot be the CI runner.
- SDLC's current schema has no train scope/path selection; required checks also run during car readiness. `weight` means suite slots, not VMs. ariadne#509 owns that extension.
- Tart's installed help supports `exec` via guest agent, headless operation, read-only shares, and disabling audio/clipboard. Official share semantics: [Tart quick start](https://tart.run/quick-start/#mounting-directories).

## Core concepts

### Pure entities

| Name | Lives in | Status |
|---|---|---|
| `RunRequest` / `ParseRun` | `cmd/internal/vmcmd/request.go` | new |
| `ImageManifest` | `cmd/internal/vmcmd/image.go` | new |
| `RunState` / `Advance` | `cmd/internal/vmcmd/state.go` | new |
| `LaunchClaim` | `cmd/internal/vmcmd/claim.go` | new |
| `GuestInvocation` | `cmd/internal/vmcmd/guest.go` | new |
| `ConformanceCases` / required-result validation | `tests/conformance-vm.py` | new |

`RunRequest` carries canonical checkout, lane (`train` or `adhoc`), timeout and literal command argv; no shell interpolation. Default lane is ad-hoc. `ImageManifest` binds an immutable qualified image name, base digest and exact tool versions/checksums. Each run pins one manifest; image refresh cannot change it in flight. `RunState` owns admitted resources, observed command result and pending cleanup, rather than independent success booleans. `GuestInvocation` produces an allowlisted environment and guest paths; no host session identity crosses it. `ConformanceCases` owns exact required Go case names and input paths; the hosted workflow and local command derive from this shared source or fail a drift assertion. The local case runner reuses the existing strict terminal evidence validation rather than inventing a second pass/skip parser.

`LaunchClaim` is the bounded durable startup-uncertainty record for one lane (nonce, managed VM name, stage, optional exact PID/start identity); only the transition owner updates it. Reuse `procutil` identity/positive-absence semantics; absent identity is unknown, not dead. Claim decoding/preservation follows the manifest's additive-format rule, and confirmed completed cleanup removes it.

Relationships: one request -> one image -> one run; one run -> one lane and one disposable VM. Lanes are shared across repositories. One conformance selection -> existing terminal and lifecycle command populations. Agent-case extension is at #744's helper, not in VM orchestration. ARCH-DRY/PURE: pure request, transition and evidence decisions are unit tested without subprocesses.

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| CLI composition | `cmd/pair-go/main.go`, `cmd/internal/dispatcher/dispatcher.go` | modified | streaming `vm run` route |
| VM runtime | `cmd/internal/vmcmd/runtime.go`, `runtime_darwin.go`, `runtime_other.go` | new | Tart, child handles, native user home, APFS, guest-agent |
| Machine positions | `cmd/internal/vmcmd/positions.go` | new | two stable kernel-lock files |
| Checkout snapshot | `cmd/internal/vmcmd/snapshot.go` | new | private COW copy and independent Git metadata |
| Stateful runtime fake | `cmd/internal/vmcmd/fake_test.go` | new | VM existence/running state, held steps, files and statuses |
| Image provisioning | `scripts/vm/provision.sh`, `scripts/vm/image.lock.json` | new | pinned base/tools and guest validation |
| Guest entry | `scripts/vm/guest-run.sh` | new | clean guest HOME, workspace copy, result collection |
| Pair conformance | `tests/conformance-vm.py`, `Makefile.local` | new / modified | existing native targets and strict case evidence |
| Nightly trigger | `scripts/vm/nightly.sh`, `scripts/vm/nightly.plist` | new | `sdlc local-check` against main |
| Runnable check declaration | `.sdlc/local-checks.json` | modified | named non-required `conformance` check, activation off |

All subprocess behavior is behind the runtime seam. Its fake retains image/clone/running state, command exit and interrupted effects across calls, enabling production orchestration tests. Real Tart conformance qualifies the modeled behaviors. No daemon, VM ticket queue, or per-repo pool is added.

## Command and resource contract

```text
pair vm run --checkout PATH [--lane adhoc|train] [--timeout 20m] [--artifacts DIR] -- COMMAND ARGS...
make conformance-vm
sdlc local-check run --name conformance
```

The machine root derives from the native account home, not inherited HOME/PAIR_DATA_DIR or a repository root. It contains two stable lock files and private per-run directories. Test-only runtime construction injects a portable root. A lock is never unlinked/replaced while clients may use it. Train and nightly use the train position; ad-hoc uses the other. Acquisition is a cancelable kernel-resource wait with progress, not a ticket scheduler; default total run budget includes acquisition. No acquisition of a full-suite slot while waiting for a VM. If current SDLC cannot avoid that, initial manual/nightly runner remains usable and named-check integration waits for ariadne#509; do not add an alternate gate or quietly occupy both pools.

Hold the lane lock through preparation, VM process lifetime, artifact collection, stop and Wait. Before spawning Tart, atomically mark the fixed lane's launch as unresolved. Clear that marker only after confirmed stop/Wait. If the runner dies before Tart publishes running state, the next acquisition sees unresolved startup and refuses even when Tart currently reports stopped; it must prove the exact prior child generation is dead before reuse. A crash before child identity publication remains unknown and needs explicit inspection; a stopped-list sample alone never authorizes cleanup. Fixed position-local VM names bound residue after abrupt runner death. A still-running prior VM is also occupied/unknown: do not clone/start another or stop an unproved VM. Report the exact managed name and explicit inspection/stop recovery; never infer absence from an unavailable Tart query. Unknown or foreign running VMs remain untouched. A named VM collision cannot be silently overwritten.

Provisioning and image qualification acquire the ad-hoc position before any boot and retain it through confirmed stop/Wait, using the same unresolved-start protection as ordinary runs. Provision into a fresh name, run version/isolation/native checks, stop and verify it, then atomically publish its manifest. Failed provision never replaces the active image. Keep one active and one previous image; remove superseded images only with both positions held (train then ad-hoc, fixed order) and no referencing run. Do not delete or mutate `pair-test` or the cached source image as cleanup.

Snapshot the selected checkout using APFS COW under Tart's permitted host storage, not a direct share of the live checkout. Enumerate tracked and non-ignored untracked files once with Git; do not export ignored auth/session/build artifacts or traverse `.git`/live `.couch` storage. Copy each admitted file with clonefile and preserve symlinks without following them into host trees. Build independent Git metadata for the exact HEAD with no host common-dir, alternates or credential-bearing config; overlay current file contents/deletions. The snapshot promises HEAD and worktree bytes, not the host's staging split. Tests cover linked worktrees, tracked dirt/deletions, non-ignored new code, excluded ignored secrets, source unchanged, and usable guest Git. Do not recursively export unrelated sibling repositories or arbitrary HOME files. Pair's only local replace is inside its checkout. Repositories needing outside local dependencies receive an explicit unsupported-input error until that use case is designed.

Expose only the owned read-only source snapshot and owned result directory to the guest. Copy the workspace into guest-local storage before building. `tart run --no-graphics --no-audio --no-clipboard` passes no host socket/HOME/store shares. Guest invocation uses `env -i` plus fixed tool PATH, unique guest HOME/TMPDIR/XDG, Zellij runtime, Couch/Pair and queue roots. Do not forward agent authentication or live caller identities. Keep argv as argv through `tart exec`; no host SSH keys or shell quoting transport. Missing guest-agent is a provisioning failure, not a fallback to permissive interactive setup.

Collect command status plus complete required-result streams, Tart boot log and bounded guest Zellij logs. The result exit reaches the host after cleanup: command failure stays nonzero; a successful command with failed collection/cleanup returns infrastructure failure, never success. Timeout/cancel joins the exact owned Tart child (bounded graceful stop, then exact-child termination/Wait); uncertain stop is visible and prevents lane reuse until inspected. No command or conformance retry occurs automatically.

## State and recovery

Actions: run, cancel, inspect retained diagnostics, stop an exact verified leftover, rerun after repair, refresh image.

| Internal state / failure | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| Position held by another run | waiting for VM position | cancel or wait within budget | none |
| Missing/unqualified image or unavailable dependency | unavailable | refresh image | no work started |
| Snapshot/clone failed before boot | failed before start | inspect and rerun after repair | partial owned scratch only |
| Booted, guest not ready at deadline | VM startup failed | inspect logs | command never started or readiness unknown |
| Command executing | running | cancel | incomplete command output |
| Command ended nonzero | failed | inspect diagnostics; fix before rerun | no passing evidence |
| Command result unavailable after transport loss | unconfirmed | inspect diagnostics and exact VM | no success claim; partial output |
| Command passed, collection/cleanup failed | cleanup failed | inspect/stop exact leftover | guest-only diagnostics may be incomplete |
| Runner killed before/after child identity or running publication | unresolved startup / occupied leftover | prove exact prior child dead, or inspect/stop verified leftover | partial output; no second launch |
| Artifacts complete, VM joined/stopped | passed or failed | inspect result | disposable guest changes removed |

Production effects pass through `Advance(state,event) -> (state,effects)`. Events include position acquired, snapshot/clone outcomes, child started, guest ready, command result, cancellation, collection outcome and stop/Wait outcome. Cancellation is valid at every wait. Late readiness/results cannot restore success after cancellation; a known command failure survives a later cleanup error. No asynchronous task outlives `Run` except an explicitly reported unconfirmed VM after abrupt process death; the next invocation observes/refuses it. ARCH-STATE/FUNERAL: a lock alone is not proof that a VM died.

## Operating envelope and residue

This is a batch CI path, not a keystroke/launch hot path. Initial choices: 4 guest CPUs and 8 GiB RAM (matching the prior development image), two VM positions, 50 GiB virtual disk per disposable clone, 20-minute total command-run budget and 2-minute boot-ready ceiling. Provisioning gets a separate explicit 60-minute budget; no installation during ordinary runs. Measure boot, snapshot, native test and cleanup times during M1/M2; revise these choices with measured evidence before declaring qualification.

Snapshot work is O(F+B) for F admitted files and B metadata bytes; one Git file enumeration/copy pass plus one independent Git setup. Count enumerations/visited entries/copy invocations at N and 2N in a controlled fixture, rejecting per-entry full rescans. Result validation is O(J) over Go JSON events, with one parse per event. Stream logs with a combined 256 MiB per-run cap; hitting it fails collection and retains an explicit truncated diagnostic. Do not record a pass with incomplete required JSON. Keep eight ordinary result bundles per lane (at most 4 GiB combined); prune oldest completed ordinary bundles under the lane lock. Failed/cancelled ordinary runs count toward that bound and print their retained path before return. An explicit fresh, disjoint `--artifacts DIR` is caller-owned and excluded from ordinary pruning; the nightly caller reserves one such directory per scheduled date, retaining up to sixteen complete nightly bundles (4 GiB). At that cap, nightly refuses further work and names review/archive as recovery; it never deletes burn-in evidence before the promotion decision. #745 records the selected seven nights, then authorizes archive/removal of the initial bounded qualification set. Manual runs cannot evict it.

Run directories/checkout snapshots/disposable VM disks are removed after the last consumer (artifact collector and stopped/joined VM). Startup under a lane lock collects stopped owned residue only; running/unknown residue refuses. Image manifest/log metadata are bounded (64 KiB each) and atomically written. Exhausted space refuses before boot; monitor the measured copy/provision peak rather than assuming sparse disks are free. Foreign VMs can consume Apple's capacity despite our locks: report that native refusal without stopping them.

## Trust and format boundaries

CLI paths, repository bytes, Tart JSON, guest results and manifests are untrusted at decode. Reject duplicate keys, missing required fields, wrong known types, unsupported versions and trailing values. Use `strictjson` object splitting/preservation for additive unknown manifest members and nested tool entries; publication is atomic, older-reader rewrites preserve unknown values. Required identity fields are never defaulted over malformed data. Fake tests include truncation, duplicates and unknown-member round trips (ARCH-EVOLVE).

Image sources and tool downloads use pinned digests/checksums in the lock file. Resolve current supported versions during M1, record them before qualifying, and never use floating latest during a run. Source includes no credentials. Claude/Codex/Qoder installation/version checks do not copy the host's auth, keychain, native sessions or Couch store. Genuine agent cases require #744's explicit private guest credentials/fixtures when available. Persist no command argv or arbitrary environment dump; result directories are private and logs are not uploaded automatically. Refuse result/source overlap and symlink escape before any cleanup.

## Milestones and exact verification

### Canonical risky-function test strategy

This table owns test strategy; milestone tasks reference it rather than restating case inventories. Pure rows use literal values/generated inputs, without runtime mocks; IO rows use the stateful runtime or real controlled resources.

| Function | Boundary | Adversarial input / mechanical guard |
|---|---|---|
| `ParseRun` | pure, `request.go` | Fuzz arbitrary argv; require a well-formed typed request or descriptive refusal before effects. |
| `DecodeImageManifest`, `DecodeLaunchClaim` | pure, `image.go`, `claim.go` | Fuzz JSON structure and additive nested members; require strict known fields and lossless unknown-member rewrites. |
| `Advance` | pure, `state.go` | Generate event sequences with interruptions/late completions; assert ownership, failure preservation and completion invariants independently of transitions. |
| `PlanSnapshot` | pure, `snapshot.go` | Fuzz admitted file descriptors and root relationships; forbid escape/overlap and count one traversal at N/2N. |
| `BuildGuestInvocation` | pure, `guest.go` | Generate poisoned ambient environment/path/argv values; compare output with the fixed allowlist and literal argv contract. |
| `Run` | IO orchestration, `runtime.go` | Stateful Tart fake pauses each effect and injects partial/unknown outcomes; trace that no launch/claim/resource is lost and no pass survives incomplete cleanup. |
| `AcquirePosition`, `RecoverLaunchClaim` | native resource seam, `positions.go` | Controlled real subprocess barriers, including parent death before running publication; require exclusion and exact positive-death evidence before reuse. |
| `SnapshotCheckout` | filesystem/Git seam, `snapshot.go` | Fixture independent/linked Git layouts with hostile content; assert isolated usable Git, admitted byte equality, source immutability and bounded copy counts. |
| `validate_results` | pure event reducer, `tests/conformance-vm.py` | Generate malformed/incomplete Go JSON streams against the required-case set; complete execution is the only accepting result, in one pass. |
| `nightly_main`, `reserve_nightly_artifacts` | command/filesystem seam, `scripts/vm/nightly.sh` | Stateful local-check fake and private date directories; require one exact-main run, no retries, isolated evidence retention and bounded saturation refusal. |

Real Tart qualification uses the same `Run` adapter with controlled commands and poisoned host sentinels, including deliberate command failure and interrupted startup/cleanup. It checks the fake's ownership/transport model, actual native-case completeness and immutable source, not elapsed timing alone.

### M1 — Refreshed image and generic isolated runner

**Files:** create `cmd/internal/vmcmd/{request,image,state,claim,guest,positions,snapshot,runtime,runtime_darwin,runtime_other}.go` with colocated tests and `fake_test.go`; `scripts/vm/{provision.sh,image.lock.json,guest-run.sh}`. Modify dispatcher registry, `cmd/internal/dispatcher/{dispatcher,route_coverage,readme}_test.go`, `cmd/pair-go/main.go`, `cmd/pair-go/{main,pty_proxy_route}_test.go`, source inventories when required, README and `atlas/vm.md`/index.

- [x] Write failing tests for M1's pure functions using the canonical strategy table. Run `go test ./cmd/internal/vmcmd -run 'Test(ParseRun|ImageManifest|LaunchClaim|Advance|GuestInvocation|PlanSnapshot)' -count=1` and observe red before implementation.
- [x] Add the table's stateful `Run` and native `AcquirePosition`/`RecoverLaunchClaim` tests. Seed a premature resource release and lost startup claim to prove the controlled ordering guard fails.
- [x] Implement the minimal runner/CLI composition behind those seams. Add Linux/unsupported-platform refusal without trying to boot Tart. Run `go test ./cmd/internal/vmcmd ./cmd/internal/dispatcher ./cmd/pair-go -count=1` plus relevant repository/source guards.
- [x] Refresh from current Tahoe base into a new qualified image candidate (outside sandbox, headless). Provision exact Go/Node/Python/Neovim/Zellij/agent CLI versions, compiler/terminal tools and guest-agent. Record source digest/checksums and versions; do not activate an unqualified candidate or change the old development VM.
- [x] Execute the table's real Tart qualification strategy and retain native ownership, transport and isolation evidence. Require a subsequent clean invocation after each confirmed cleanup; uncertain outcomes must remain refusals.
- [x] Run `SnapshotCheckout` and `BuildGuestInvocation` strategies through the actual VM composition, retaining immutable-source/sentinel evidence and N/2N counts alongside measured timing/resource evidence.
- [ ] Update issue Log/atlas, commit, then `sdlc milestone-close --issue 743 --milestone M1 --verified '<measured evidence>'`. Fix/dispute any blocking findings before M2.

### M2 — Strict Pair conformance and runnable named check

**Files:** create `tests/conformance-vm.py`, `tests/conformance-vm-test.py`, `tests/conformance-paths.json`; modify `tests/native-terminal-ci.py` only to share its existing strict evidence checker, `Makefile.local`, `.sdlc/local-checks.json`, workflow path drift/generation guard, `atlas/vm.md` and README.

- [ ] Write failing `validate_results` tests using the canonical strategy table. The required-case set covers both existing native populations and retains #432's explicit exclusions.
- [ ] Extract the workflow's existing path list to one source and enforce its workflow consumers. Keep all present PR triggers/required checks; no demotion. Test additions/deletions to the catalog fail the drift guard until generated/updated consumers match.
- [ ] Add `make conformance-vm` -> generic runner train lane -> strict guest conformance command -> existing terminal/lifecycle cases. Keep command construction in argv and avoid recursive VM invocation in the guest.
- [ ] Add non-required named `conformance` check only through the supported SDLC contract. ariadne#509 owns external-resource accounting; if not yet landed, obtain that prerequisite before claiming named-check acceptance. No hand-written commit-status publisher or wrapper around merge.
- [ ] Run `python3 -B tests/conformance-vm-test.py`, then `make conformance-vm` through the authorized outside-sandbox execution path. Run a deliberately failing fixture from a disposable checkout and retain its nonzero host result, logs and cleanup evidence. No live checkout mutation to create the fixture.
- [ ] Before landing, verify the declared command directly on a private checkout of the pushed head, including isolation sentinels and the known-bad fixture. SDLC reads check declarations from main, so a newly introduced name cannot be run by the production named-check route until #743 lands. Reserve its first exact-head status/log binding for the explicit post-landing bootstrap below; never inject candidate config into the gate or claim a pre-landing named-check pass. Record current coverage as legacy/native stand-ins; do not claim genuine agent cases before #744 exists.
- [ ] Update issue Log/atlas, commit, then `sdlc milestone-close --issue 743 --milestone M2 --verified '<measured evidence>'`.

### M3 — Nightly burn-in handoff and initial infrastructure landing

**Files:** create `scripts/vm/nightly.sh` and `scripts/vm/nightly.plist`; update README/atlas and issue evidence. `.sdlc/local-checks.json` stays non-blocking for landing; GitHub PR conformance remains required.

- [ ] Write failing `nightly_main`/`reserve_nightly_artifacts` tests from the canonical strategy table. Clock/due behavior is superseded by the scheduler revision below.
- [ ] Implement an off-hours launchd trigger (proposed 03:30 local). It invokes the installed named-check command on main and reserves a caller-owned nightly artifact directory under the separate sixteen-night bound. Pass its destination through documented host-only `PAIR_VM_ARTIFACTS_DIR` to `make conformance-vm`, which turns it into `--artifacts`; qualify that propagation through real SDLC/Make environment handling, never forwarding it into guest authority. Installation/bootstrap happens explicitly after landing, when main contains the declaration, after hub approval; no daemon/service side effect from build/help/run. Before landing, verify the schedule wrapper with its stateful fake and the actual underlying direct VM command; print installation and uninstall instructions.
- [ ] Preserve image/case identity with each nightly result so seven distinct clean nights can be audited. A failure remains a failure pending root-cause repair. Do not add an automatic streak counter/promotion state; existing retained exact-head results are the evidence.
- [ ] Review #744 availability at handoff; adopt its genuine agent cases only if independently landed, otherwise name that outstanding coverage explicitly in the promotion issue. No invented agent fixture inside VM code.
- [ ] Push final head and run the recorded full suite with `sdlc local-check run --name suite`, plus the declared VM command directly outside the sandbox. Stage the public CLI through `sdlc stage --build` only after the hub authorizes touching :0; otherwise retain own-slot real-VM evidence and ask the hub to decide staging. Obtain the operator's local-gates verification/label; never self-label.
- [ ] `sdlc close --issue 743 --verified '<suite, real VM, strict bad fixture, isolation, nightly handoff; PR conformance retained; named-check bootstrap follows landing>'`; resolve its mandatory fresh-context review, open PR, and wait for the operator's label before merging. On SHIP/label use `sdlc merge --yes`. Then bootstrap the now-main declared check once with `sdlc local-check run --name conformance`, verify exact-head retained status/log evidence, and install the approved nightly trigger. Report landing plus bootstrap outcome; a bootstrap failure is an explicit blocked follow-through, not a reason to waive GitHub PR conformance. Seven-night observation/promotion remains #745.

## Approval requested

Approve the full-flow design, especially fixed lane ownership/leftover refusal, the explicit image refresh and machine installation, proposed 03:30 nightly time, and the phased acceptance. Required pre-landing conformance remains GitHub-backed until promotion. The initial issue covers the image/runner/nightly shell; genuine per-agent fleet coverage is not claimed without #744. ariadne#509 must land before supported resource-aware named-check acceptance; planning and M1 can proceed independently.

## Revisions

### 2026-10-20 14:26 — Hub approval; one scheduling mechanism

Authenticated receipt 10faa55a-dfce-4057-8d28-9c2edf6912b7 approves the design with this governing change: **all launchd/cron trigger/installation references above are superseded**. The operator selected the shared Couch scheduler (#737), as tracked in ops's couch-scheduling project. This issue delivers the deterministic nightly command and a task-declaration handoff for `ops/scripts/scheduled/nightly-conformance/`; until #737 lands, the hub/operator triggers that same command by hand and records the nights. No private scheduling service or internal timer is installed. The machine/image refresh and installs are operator-approved, with a notice to the hub immediately before each such operation.

M3's authoritative files and tasks replace its earlier trigger-specific steps:

- Create `scripts/vm/nightly.sh` (deterministic run command) and `scripts/vm/scheduled-task.md` (ops task declaration/handoff: `30 3 * * *`, machine scope, timeout covering the bounded VM command, skip-if-missed; native Go task adapter calls this command). The scheduler owns executable declaration syntax; adapt the handoff to #737's landed schema before registration rather than inventing a second parser. Ops scheduled-task code follows ops's own worker/SDLC track; no direct code writes to its hub checkout.
- Test `nightly_main` and `reserve_nightly_artifacts` using the canonical strategy table. Scheduler clock/due/skip/catch-up policy belongs to #737, so these are declaration fields, not another implementation in this command.
- After landing, bootstrap the named check on main and hand the exact manual invocation, result path and declaration to the hub. The hub/operator runs it until #737 can register it. No scheduled invocation or machine scheduler installation is claimed as #743 acceptance.
- Keep M1/M2 and the remaining M3 verification, operator local-gates label and landing steps unchanged. Required hosted PR conformance stays active; #745 owns promotion. Genuine agent cases wait for #744; neither #737 nor #744 is assumed implemented here.

This revision also replaces the request for nightly-service installation approval: no service will be installed. Notify the hub before the already-approved image refresh/provisioning or host tool installation, then continue without asking again.

### 2026-10-20 14:30 — PQ-1, named test strategies

The first plan-quality review accepted the architecture but required compressing procedural case inventories into named risky-function strategies. Added the canonical table and changed milestone test tasks to reference it. This clarifies executable pure/IO surfaces without changing scope, mechanisms, state or approval boundaries. Existing lesson #721 PQ-1 already owns this rule; do not duplicate it.

### 2026-10-20 14:33 — Native image work held

The hub's authenticated hold supersedes the earlier operator-approved wording for image refresh/installs. Our already-started pull was interrupted; no VM was cloned/booted. Continue code and portable fake/fixture tests only until explicit operator approval. Before any authorized resume/provisioning, notify the hub as originally requested. M1 cannot close without its outstanding real image/VM qualification.

### 2026-10-20 14:53 — Tracked-only snapshot with bounded filesystem fallback

Authenticated hub receipt b59211cc-8cf9-485e-925b-1288d64cd3c2 approves the cross-volume fallback: one tracked-file enumeration, clone each regular file where supported, copy only on EXDEV or unsupported filesystem, and report cloned/copied bytes. This supersedes earlier non-ignored/untracked inclusion and clone-only assumptions: omit untracked files and caches; Pair regenerates runtime inputs in the guest from tracked sources. The snapshot retains independent Git metadata and current tracked worktree bytes/deletions. Its initial content budget is 100,000 paths / 2 GiB, inside the enclosing run timeout; native qualification must measure the real peak. The actual TestScratch and native Tart storage are on different volumes. No VM/image action is authorized by this correction; the 14:33 image hold remains.

The guest bootstrap is embedded at cmd/internal/vmcmd/guest-run.sh rather than scripts/vm/guest-run.sh so an installed generic runner can run any checkout without depending on a Pair source tree. Its paths/environment contract is unchanged.

Hub receipt 95d237ca-495c-4366-8822-ce855fc1433d additionally holds Pair issue filings/publishes to main until train 197 lands. Branch-local code/design checkpoints remain independent; do not publish or file while that hold is active.

### 2026-10-20 14:56 — Main publication hold lifted

Authenticated hub receipt 40831bf5-2c90-4de8-bd36-19f9ba4437ee confirms train 197 landed and Pair main filings/publishes are open again. The image hold is unchanged.

### 2026-10-20 16:51 — Execution timeout and bounded cleanup

Authenticated hub receipt d13df378-0a0a-4040-98a3-c20e29d43dbc approves the explicit envelope **timeout N + cleanup up to 10 s**. The requested timeout includes acquisition, preparation, startup/readiness and execution. Collection, stop/join and release share one separate cancellation-independent ten-second budget. This supersedes the earlier total-run wording; no cleanup reservation is subtracted from execution. Help text and retained results expose the envelope. Expired or unconfirmed cleanup retains the durable occupied launch claim and VM/snapshot inputs, reports the managed VM and inspection recovery, and makes the next run refuse reuse.

### 2026-10-20 17:03 — Proposed M1 lifetime/collection re-plan (awaiting hub approval)

Implementation is paused at this checkpoint. The new host/driver/CLI composition is incomplete and must not qualify M1 or land. Fresh component review found leader-only recovery, discarded secondary subprocess uncertainty, and artifact validation before the last writer. A subsequent ownership audit also found that signalling a numeric process group after its leader has been reaped can target a replacement generation. The same ownership mistake spans snapshot Git commands, synchronous Tart commands and the long-lived Tart run child; fix the class together rather than patching only Launch.

**Alternatives and recommendation.** Keep direct children unreaped until group termination has been requested (recommended); use the existing native process observations to detect terminal state without Wait, and only then join. This keeps the kernel PID generation pinned, adds no supervisor or public protocol, and preserves native exit status. A permanent self-owned guardian could pin each group instead, but adds another subprocess/protocol and parent-death behavior to qualify. Comparing a cached start identity immediately before signalling is insufficient by itself: reaping releases the stable resource and a later numeric group operation still races reuse. Reusing execalarm/broadcast guards does not solve this: execalarm execs the command and does not own descendant cleanup; broadcast's current guard also waits/reaps before a later group signal.

**Result storage choice.** Recommend removing the writable host result share and pulling bounded results over the existing guest-agent transport. The current post-hoc walk cannot enforce a storage ceiling while a guest can write its shared host directory. A quota-bearing mounted APFS image could enforce it but adds native mounts, cleanup and another resource population; a polling size watcher only detects overshoot. Bounded transport keeps the ordinary run's host artifact ceiling structural and needs no new quota/mount service. This changes the approved sharing detail and requires hub approval before implementation.

#### Revised concepts and integration

| Concept | Location | Change |
|---|---|---|
| Owned subprocess lifetime | vmcmd/process_unix.go plus platform observation helpers | one shared Start/observe/terminate-before-reap/Wait owner for Git and Tart; explicit pipes joined under context |
| Cleanup uncertainty | vmcmd/claim.go, host.go, driver.go | additive blocking claim stage, preserving exact Tart PID/start identity; older strict readers refuse the new stage |
| Bounded guest result transport | vmcmd/driver.go, driver_darwin.go, guest.go, guest-run.sh, host.go | guest-local result directory; only read-only input share; streamed byte/entry caps at host writes |
| Final host evidence | vmcmd/host.go | close all VM writers, then validate retained host artifacts and publish terminal result |

The process owner retains the direct child until no further numeric group signals can occur. Native terminal observation reads that one pinned child rather than rescanning the whole process table; unsupported/unavailable observation preserves uncertainty. Cancellation, stdout/stderr writer failure and ordinary parent exit all request group termination while the PID is pinned. Wait/reap follows the final signal, then observation-only group disappearance and pipe/goroutine joins. No signal follows reaping. Preserve the original command exit code when it was already terminal. A request deadline still permits only the approved shared ten-second cleanup envelope; nested helpers consume the same cleanup deadline rather than renewing private budgets.

Recovery needs positive absence of the entire recorded launch group as well as the exact prior leader generation and stopped VM. Readiness polling stops immediately on subprocess cleanup uncertainty. Preparation, readiness, execution and collection transport all propagate the same typed uncertainty. Before releasing the local lease, persist a blocking cleanup-unknown claim stage when any owned group cannot be confirmed gone; retain exact main-child identity and input files. That stage refuses automatic recovery until explicit inspection/repair. Startup uncertainty before identity publication stays launching and already refuses reuse. This is a failure/recovery state extension, not a claim that an unknown process is gone.

Guest result export uses a bounded manifest and per-member streams through Tart exec; paths are guest-local and host destinations are rooted, fresh and singly owned. No guest-writable host directory is exposed. Every retained byte is charged at its host writer against the combined per-run cap; entry count and per-diagnostic limits remain bounded. Collection may read changing guest files, so strict completeness validation still applies; transport overflow or incomplete required evidence fails the run. Stop/join precedes the final host-only artifact validation and terminal result publication. A command failure survives any collection/cleanup failure. M2 owns its strict required-case catalog and complete JSON streams; the generic runner must preserve stdout/stderr separately rather than merge diagnostic text into a JSON population.

#### Visible failure states

| Internal failure | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| Native terminal observation unavailable or any group unconfirmed | cleanup unconfirmed; exact lane/VM and inputs retained | inspect/repair exact managed residue, then rerun | partial output; no success evidence |
| Owner dies before pinned generation/group cleanup | occupied startup/launch claim | inspect exact claim, require leader + group absence and stopped VM | partial output; no second launch |
| Result transfer exceeds cap or manifest invalid | collection failed | inspect retained bounded diagnostics; repair command, rerun | guest-only excess/incomplete output |
| Host artifact validation fails after confirmed stop | infrastructure failure | inspect bounded retained bundle, rerun after repair | no passing evidence |

#### Named regression strategy and execution steps

| Function / seam | Regression that must fail first |
|---|---|
| Shared process owner | controlled direct-parent exit with surviving descendant and closed output; retain an unreaped-generation barrier; require signal-before-Wait, no signal-after-Wait, final EOF/absence and exact exit status |
| Native Recover | direct leader gone but controlled group descendant remains and VM stopped; refuse until entire recorded group absent |
| AwaitGuest + host uncertainty | injected unconfirmed readiness/exec/export subprocess; zero retry after uncertainty; main Tart Stop success cannot clear the blocking claim or delete inputs |
| Guest export / host byte writer | exact total cap and one byte over; oversized single member, duplicate/escape/symlink member, too many entries and truncated manifest; host disk writes never exceed admitted cap |
| Final retained evidence | Stop mutates a guest result after preliminary collection; no pass without final host-only validation; independently seeded stdout/stderr stay separate |
| CLI/result composition | exact requested timeout plus one shared cleanup cap; retained failure reports managed VM/claim and envelope; no native image effects on help/parse failure |

- [ ] After hub approval, re-run start-plan for the revised design and change-code's plan-quality check before further implementation; re-cost only if the gate requests it.
- [ ] Add the named shared-owner red regressions, implement the smallest native observation adapters and unified owner, then sweep Git and every Tart callsite together. Do not extend to unrelated repository process owners in this issue.
- [ ] Add the recovery/uncertainty red regressions, implement the blocking claim stage and propagation through every owned operation, update pure codec/state tests and recovery documentation.
- [ ] Add bounded export and final-evidence red regressions, replace the writable result share with guest-local results and capped transport, then qualify CLI composition through the stateful fake.
- [ ] Run targeted vmcmd race/dispatcher/main-route/source/reader guards; obtain one fresh component review, then resume the original native-image qualification only after explicit operator approval. Keep all M1/M2/M3 review boundaries and the hosted-conformance hold unchanged.

### 2026-10-20 17:04 — Lifetime/collection re-plan approved

Authenticated hub receipt 7869dd0e-f350-4711-86e7-ff320ce1f1c2 approves the recommended 17:03 re-plan, including bounded guest-agent result export in place of writable host sharing, with no quota mount. A third lifecycle mechanism change must stop for options review before coding. Keep the implementation focused on conformance. The revision's initial “awaiting hub approval” label records its proposal state; this entry governs execution now. Native image operations remain held.

The revised named seams are `startOwnedCommand`, `(*ownedCommand).Join`, platform `observeOwnedChild`, `(*nativeDriver).Recover`, `(*nativeDriver).AwaitGuest`, `(*hostRuntime).Execute`, `(*nativeDriver).ExportResults`, `decodeResultManifest`, `writeExportedResult`, and `(*hostRuntime).Release`/`WriteResult`. Their adversarial strategies are the corresponding 17:03 table rows; exported-result decoding/path/budget decisions are pure, with literal/fuzz unit fixtures, and transport/owned-process behavior uses controlled children and the persisted driver fake. Production subprocess callsites enumerate to snapshotGitInput, nativeDriver.run, and nativeDriver.Launch; all three must derive their lifetime from the shared owner.

### 2026-10-20 17:20 — Approved re-plan implementation mapping

The revised shared owner is named `ownedProcess`, returned by `configureSnapshotCommand`; its `AwaitTerminal`, guarded `Signal` and `Finish` implement the planned start/observe/terminate-before-reap/join seam. Platform observers are `nativeChildTerminal` and `nativeGroupPresent` in process_observe_darwin.go/process_observe_linux.go. This maps the 17:04 provisional `startOwnedCommand`/`ownedCommand.Join`/`observeOwnedChild` names to the actual shared source, not a mechanism change. `cleanupContext` derives nested and final cleanup from one lazy per-Run scope; the ordinary execution path does not start that clock.

`decodeResultManifest` and `writeExportedResult` admit the bootstrap's fixed catalog: required status (128 bytes) and up to eight optional diagnostic logs (64 KiB each), with a 4 KiB manifest. Command stdout/stderr are separately capped host streams, preserving M2's complete JSON populations without a generic arbitrary-file export feature. The pure decoder covers structural/type/path/duplicate/size inputs; export tests use a stateful file population and exact/one-over writes. The native transport supplies literal guest path argv and checks symlink ancestors before bounded reads. No writable host share, quota mount, additional scheduler or image action was added.

### 2026-10-20 21:18 — Native hold lifted after #748; official binary provenance

Authenticated submitted hub receipt3708133b-f037-4efe-bdb2-4bd40dd5e457 conveys explicit operator authorization to finish the clean Tahoe pull with digest verification and build the machine-wide image after #748 lands. PR500 has now landed. This supersedes the native hold; it does not authorize deleting any other ~/.tart content. Remove only positively identified partial state left by our interrupted pull and preserve pair-test. Read-only preflight found ~/.tart/tmp empty; recheck instead of assuming cleanup is necessary. Notify hub before machine effects as originally required.

Hub receipt0fe3af03-7588-42d1-9403-b7eaf6037fd8 requires the official checksum-verified Zellij0.45.1 release asset used in CI, not Homebrew's different same-version binary. Official aarch64 Darwin SHA256 is ca5f9333735bdbc59a621f1d8ed8e24798845302a28ff175d253d4793d5a4a2c; retain release-checksum provenance in image.lock.json and the qualified manifest. The two lanes, owned unreaped child lifetime, sticky cleanup uncertainty, bounded guest-local export, no host credential sharing, no quota mount and third-lifecycle-mechanism-change stop rule are unchanged. M1 still needs actual image/runner qualification before close.

### 2026-10-20 21:55 — M1 qualification checkpoint

The image builder is an explicit public `pair vm provision --checkout PATH` route plus the guest-only scripts/vm/provision.sh and fixed image.lock.json. It reuses the approved Run/position/claim/owned-child lifetime; activation occurs after final validation/log closes under the same lease and cleanup budget. Every unpublished managed candidate counts toward the two-image ceiling. Guest-local installation and read-only snapshot copy needed two measured Darwin compatibility corrections (symlink metadata copying and privileged link permissions); neither changes the lifecycle mechanism. Per-run result encoding now preserves image provenance via the shared codec and retains historical launch/command-request observations. Real native qualification and final cancellation/reuse evidence are retained under workshop/evidence/000743; exact snapshot and storage measurement limits are explicit. M1 boundary review is next. M2/M3 scope, #509 prerequisite and hosted PR-conformance hold are unchanged.
