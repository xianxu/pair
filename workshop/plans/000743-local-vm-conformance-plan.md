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

- [ ] Write failing tests for M1's pure functions using the canonical strategy table. Run `go test ./cmd/internal/vmcmd -run 'Test(ParseRun|ImageManifest|LaunchClaim|Advance|GuestInvocation|PlanSnapshot)' -count=1` and observe red before implementation.
- [ ] Add the table's stateful `Run` and native `AcquirePosition`/`RecoverLaunchClaim` tests. Seed a premature resource release and lost startup claim to prove the controlled ordering guard fails.
- [ ] Implement the minimal runner/CLI composition behind those seams. Add Linux/unsupported-platform refusal without trying to boot Tart. Run `go test ./cmd/internal/vmcmd ./cmd/internal/dispatcher ./cmd/pair-go -count=1` plus relevant repository/source guards.
- [ ] Refresh from current Tahoe base into a new qualified image candidate (outside sandbox, headless). Provision exact Go/Node/Python/Neovim/Zellij/agent CLI versions, compiler/terminal tools and guest-agent. Record source digest/checksums and versions; do not activate an unqualified candidate or change the old development VM.
- [ ] Execute the table's real Tart qualification strategy and retain native ownership, transport and isolation evidence. Require a subsequent clean invocation after each confirmed cleanup; uncertain outcomes must remain refusals.
- [ ] Run `SnapshotCheckout` and `BuildGuestInvocation` strategies through the actual VM composition, retaining immutable-source/sentinel evidence and N/2N counts alongside measured timing/resource evidence.
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
