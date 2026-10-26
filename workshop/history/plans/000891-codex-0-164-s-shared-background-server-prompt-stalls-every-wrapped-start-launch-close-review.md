# Boundary Review — pair#891 (whole-issue close)

| field | value |
|-------|-------|
| issue | 891 — Codex 0.164's shared background server prompt stalls every wrapped start; launch so it can't appear |
| repo | pair |
| issue file | workshop/issues/000891-codex-0-164-s-shared-background-server-prompt-stalls-every-wrapped-start-launch.md |
| boundary | whole-issue close |
| milestone | — |
| window | 802880a417ae347054f71959961ef7a860bb4e36..4a31184eff5b60725c7e3b46662ec39f0f3b989a |
| command | sdlc close --issue 891 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-25T13:36:11-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Diff and plan read; now checking `hasFlag` semantics, the fresh-start path, and the probe's launch-path behavior.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

**Done-when: current.** The diff builds what the three Done-when bullets ask for: `--no-daemon` gated at ≥ 0.164.0 with the argv left alone below the gate or when the probe fails, `PAIR_AGENT_VERSION` carried into the ready record's `agent_version`, and launcher/readiness/wrap tests plus the manual PTY check logged with its control leg. The Spec/Revisions edits only narrowed scope to Part A, and Done-when was rewritten in that same revision.

The core fix is correct and small. I read the code myself rather than relying on the commit messages:
- The argv gate is right, and it checks the caller's `--no-daemon` only before the `--` delimiter, through `hasFlag`/`FlagRegion`.
- The plan-quality finding is fixed: `--no-daemon` is now in `codexBoolGlobalOption`, so a fresh start still drops a stale `resume <id>`.
- The ready-record field is added in a backward-compatible way.
- The focused tests pass in this checkout: `go test ./cmd/internal/readiness` ok, and `go test ./cmd/internal/launcher -run 'ParseAgentVersion|VersionAtLeast|FreshAgentArgs|CodexLaunch'` ok.

One thing should be fixed before shipping. The version is only ever *set*, never cleared. A launch whose probe fails, or a non-Codex launch, inherits whatever `PAIR_AGENT_VERSION` was already in the environment, and the wrapper writes that into the ready record. That breaks the Spec's "records the Codex version it ran", and it is exactly the input #894 will read.

### 1. Strengths
- `createflow.go:830-841`: the gate fails closed. An unparseable or missing version is never "at least" anything (`agent_version.go:37-54`), so an unknown CLI never gets the flag. The shared server is never contacted.
- The PQ-1 fix landed with its test (`agentargs.go:109`, `agentargs_test.go:68`). Without the table entry, a fresh start would quietly keep the resume binding.
- The self-review caught that `argumentPresent` ignored `--` and duplicated an existing helper. It was replaced with `hasFlag`, and `TestCodexLaunchTreatsPromptTextAsNoFlag` pins the case where `--no-daemon` appears as prompt text after `--`.
- The ready record changes additively: the field is `omitempty`, the known-field list comes from the struct by reflection (`record.go:41`), and the new bound mirrors the `AgentIdentity` check.
- The manual PTY check includes a control leg that shows the prompt is live, and a second control that pins the cause on `shell_snapshot`.

### 2. Critical findings
None.

### 3. Important findings
- **A stale `PAIR_AGENT_VERSION` is inherited into the ready record** (`createflow.go:831-834`, `wrap.go:664`; ARCH-SECURE).
  - The launcher calls `rt.SetEnv(AgentVersionEnv, v)` only when `v != ""`, and only for codex. Nothing unsets it: `runcli.go:58-65` clears other launch-scoped variables, but not this one.
  - `OSRuntime.SetEnv` is a plain `os.Setenv`, and the wrapper reads `os.Getenv` for *every* agent.
  - So `pair` run from a tool shell inside a Codex pane (whose environment carries `PAIR_AGENT_VERSION=0.164.0`) gets that value recorded when:
    - it launches claude, qoder or grok, or
    - it launches codex and the probe fails.
  - The record then shows a version that never ran, and #894's version-change notice would treat it as evidence.
  - `TestCodexLaunchWithoutAVersionProbeIsUnchanged` can't catch this because the fake's `env` map always starts empty.
  - **Fix:** set the variable unconditionally for every agent, `rt.SetEnv(AgentVersionEnv, agentVersion)` with `""` when unknown, the same way `PAIR_AGENT_COMMAND` and `PAIR_LAUNCH_NONCE` are always overwritten. Add a test that pre-seeds `rt.env[AgentVersionEnv] = "0.150.0"` and expects `""` after a failed-probe launch and after a non-codex launch.

### 4. Minor findings
- `osruntime.go:1331-1334`: the 3 s probe bound isn't enforced if a grandchild holds stdout. The npm `codex` shim is node spawning the native binary, so killing node can leave `Output()` waiting on the pipe. Set `cmd.WaitDelay`. The probe also uses `context.Background()` instead of the launch's startup context (ARCH-CONSTRAINTS).
- Every launch test is a Couch-owned resume. No test pins the gate on a non-Couch fresh launch, which the plan explicitly includes ("all Codex launches").
- There is no recurring conformance check that `codex --help` still lists `--no-daemon` or that the version output format holds. The one-time manual PTY check is the only evidence (ARCH-MOCK).

### 5. Test coverage notes
- The pure helpers are tested without IO.
- The launch policy is tested through the injected `fakeRuntime.AgentVersion`.
- The wrapper test copies the variable into the record.
- Validation bounds are tested for too long, NUL and newline.
- Gap: inherited-environment staleness (the Important finding above).
- `OSRuntime.AgentVersion` has no test; it's a thin exec shell, which is acceptable.

### 6. Architectural notes
| Marker | Result | Notes |
|---|---|---|
| ARCH-DRY | pass | `hasFlag` reused; one flag table |
| ARCH-PURE | pass | pure parse/compare; the probe sits behind `AgentVersionOps` |
| ARCH-PURPOSE | pass | the stall fix is fully delivered; the notice is a separable follow-up (#894) |
| ARCH-MOCK | pass with Minor | seam plus fake; the read-only probe needs no persisted state; no live conformance check |
| ARCH-CONSTRAINTS | pass with Minor | latency measured at 0.11–0.15 s; the `WaitDelay` gap above |
| ARCH-SECURE | flag (Important) | stale inherited env reaches the record; the record field itself is validated on encode and decode |
| ARCH-ORDER | pass | no state carried between events; the version is a launch-time value, and the atlas documents the mid-session upgrade case |
| ARCH-FUNERAL | pass | no new durable artifact; the field rides the existing ready-record lifecycle |
| ARCH-EVOLVE | pass | additive `omitempty` field; old readers tolerate it; no rewrite path |

Before #894 consumes `agent_version`, the empty-vs-inherited semantics need to be airtight (see the Important finding).

The atlas was updated (`atlas/codex-resume.md`). README isn't needed because nothing user-typed is added: the environment variable is internal plumbing.

### 7. Plan revision recommendations
- Once the fix lands, add a Revisions entry: "`PAIR_AGENT_VERSION` is set on every launch (empty when unknown or non-codex) so an inherited value cannot reach the ready record."

```findings
findings:
  - id: new
    severity: Important
    kind: defect
    family: launch-env-must-overwrite-inherited
    title: |
      PAIR_AGENT_VERSION is only set on a successful codex probe and never cleared, so an inherited stale value reaches the ready record
    detail: |
      createflow.go:831-834 calls rt.SetEnv only when the probe returned a version, and only for codex; runcli.go does not unset it, and wrap.go:664 copies os.Getenv into agent_version for every agent. A pair launch from a shell inside a Codex pane (env already carries 0.164.0) records that version for a claude launch or a failed codex probe. Fix: SetEnv unconditionally for all agents (empty when unknown), and add a launcher test that pre-seeds rt.env with a stale value.
  - id: new
    severity: Minor
    kind: defect
    family: declared-bound-not-enforced
    title: |
      codex --version probe timeout is not enforced when a grandchild holds stdout (no cmd.WaitDelay)
    detail: |
      osruntime.go:1331-1334 uses exec.CommandContext(...).Output() with no WaitDelay; the npm codex shim forks the native binary, so killing the parent can leave Output blocked past the stated 3s bound. It also uses context.Background instead of the launch startup context.
  - id: new
    severity: Minor
    kind: defect
    family: policy-scope-untested
    title: |
      No launch test pins the --no-daemon gate on a non-Couch codex launch, though the plan applies it to all codex launches
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"42e51784-1983-4fc4-954d-3323952a4d14","repository":"github.com/xianxu/pair","issue_id":"000891","recipe":"milestone-review","round":1,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/5d945210df0140cf","reviewed_head":"4a31184eff5b60725c7e3b46662ec39f0f3b989a","slot":"pair:8","requested_at":"2026-10-25T20:34:25.005803Z"},"phase":"finished","pid":19397,"started_at":"2026-10-25T20:34:25.117794Z","finished_at":"2026-10-25T20:36:07.471694Z","session_id":"a0b79cf9-8828-46f1-9040-1339502c6eee","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.714691083s
head: "4a31184eff5b60725c7e3b46662ec39f0f3b989a"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-25T15:29:13-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 891 — Codex 0.164's shared background server prompt stalls every wrapped start; launch so it can't appear |
| repo | pair |
| issue file | workshop/issues/000891-codex-0-164-s-shared-background-server-prompt-stalls-every-wrapped-start-launch.md |
| boundary | whole-issue close |
| milestone | — |
| window | 774eb26bf56b367e0b54c212bc5bc2680179a3df..2fd584aa8b1dae3960d6b9fb135ca90940e5b75f |
| command | sdlc close --issue 891 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-25T15:29:13-07:00 |
| verdict | SHIP |

## Review

Prior findings mostly look addressed. Next I'm running the tests and reverting the `--no-daemon` table entry to see whether the BR-1 test goes red without it.

The launcher package failed. Checking which tests failed.

The one failure comes from the review sandbox refusing `git init`, not from the diff. One last check: what the existing `version` in peer_runtime refers to, in case the new field duplicates it.

```verdict
verdict: SHIP
confidence: high
```

All six prior findings are closed (five addressed, one dispute accepted) and I found nothing new worth raising. Codex launches at version 0.164.0 or later now start with a leading `--no-daemon`. The launcher sets `PAIR_AGENT_VERSION` on every launch for every agent, using an empty value when the version is unknown, so a value inherited from an outer pane is overwritten. The wrapper copies it into the ready record. I checked the BR-1 dispute by experiment: I removed `"--no-daemon"` from `codexBoolGlobalOption` and `TestFreshAgentArgsWalksPastNoDaemon` failed (`[--no-daemon --disable shell_snapshot resume id1]`), then I restored it. So the dispute holds. The `readiness` and `wrapcmd` packages pass. `launcher` had one failure, `TestSelectedRootNativeCreateListAndListedResume`, because the review sandbox refuses `git init` outside its directories ("outside the reviewer's directories (sdlc #339)"). That is the environment, not this diff.

Done-when: current — the three bullets still cover what was built: the ≥0.164 gate with argv unchanged otherwise, `PAIR_AGENT_VERSION` carried into `agent_version`, and the tests plus the manual PTY check. The Revisions change (set the env on every launch, empty when unknown) refines the "recorded per launch" bullet without falling outside it.

**1. Strengths**
- `createflow.go:833-840`: one small seam. It probes, always sets the env, then gates the flag through `hasFlag`. Since `hasFlag` respects `--`, prompt text after `--` can't suppress the flag (`TestCodexLaunchTreatsPromptTextAsNoFlag`).
- `versionAtLeast` is pure and refuses anything it can't parse. An unknown CLI never gains the flag, and table tests cover this without any IO (`agent_version_test.go`).
- `osruntime.go:1331-1336` adds `WaitDelay` on top of the context timeout, so the 3s bound holds even when the npm shim's child keeps stdout open.
- `TestAgentVersionEnvOverwritesAnInheritedValue` covers the BR-4 class for both codex and claude.
- The ready record decodes with `DecodeKnown` and known fields derived by reflection. The new field is additive and older readers ignore it safely.

**2. Critical findings:** none.

**3. Important findings:** none.

**4. Minor findings**
- `osruntime.go:1331`: the probe still uses `context.Background()` rather than the launch context, so a cancelled launch can wait up to about 4s (3s timeout plus 1s `WaitDelay`). This is noted under BR-5.
- `agent_version` has no reader in this range. That is by design: the notice that reads it is #894.

**5. Test coverage notes**
- The launcher tests cover 0.164.0, 0.163.0, a failed probe, an explicit `--no-daemon`, a standalone launch outside Couch (BR-6), and overwriting an inherited value (BR-4).
- The readiness tests cover round-trip and the size and control-character bounds on the version.
- The wrapper test covers copying the env into the record.
- `WaitDelay` has no automated test because it would need a real forking shim. That's acceptable at Minor.

**6. Architectural notes**
- **ARCH-DRY: pass.** The flag goes into the existing `codexBoolGlobalOption` table and the guard reuses `hasFlag`.
- **ARCH-PURE: pass.** Parsing and the version gate are pure; the IO is one injected `AgentVersionOps` method.
- **ARCH-PURPOSE: pass.** The gate applies to every codex launch, and the fresh-start strip handles the new flag.
- **ARCH-MOCK: pass.** The fake runtime models the probe behind the same seam. A live check against the real CLI is only the manual PTY check, which is enough for a read-only `--version`.
- **ARCH-CONSTRAINTS: pass.** The probe is measured at 0.11–0.15s and bounded at 3s, enforced by both the timeout and `WaitDelay`.
- **ARCH-SECURE: pass.** The version is checked against a strict regex at the probe, and `Validate` bounds the record field.
- **ARCH-ORDER: pass.** The launcher holds no state between events: the version is read once per launch and handed on through env. A Codex upgrade mid-session is documented as keeping the old version until the next launch.
- **ARCH-FUNERAL: pass.** Nothing new is durable. The env var lives as long as the process, and the record field rides on an existing artifact whose lifecycle is unchanged; it adds at most 64 bytes.
- **ARCH-EVOLVE: pass.** The new field is additive with `omitempty`, and readers tolerate unknown fields.

**7. Plan revision recommendations:** none. The plan and issue match the code.

```findings
dispose:
  - id: BR-1
    disposition: withdrawn
    note: |
      Evidence holds: --no-daemon is at agentargs.go:109; removing it makes TestFreshAgentArgsWalksPastNoDaemon fail with resume id1 retained (re-run in this checkout).
  - id: BR-2
    disposition: addressed
    note: |
      Issue Log records codex --version at 0.11-0.15 s real over 3 runs against the 3 s bound.
  - id: BR-3
    disposition: addressed
    note: |
      Plan snippet (plan line 201-206) now imports regexp, strconv, strings, time, matching agent_version.go.
  - id: BR-4
    disposition: addressed
    note: |
      createflow.go:833-834 calls SetEnv unconditionally for all agents; TestAgentVersionEnvOverwritesAnInheritedValue seeds a stale value for codex and claude and asserts it is cleared.
  - id: BR-5
    disposition: addressed
    note: |
      osruntime.go sets cmd.WaitDelay = 1s so the bound holds past a grandchild holding stdout; the context.Background vs launch-context point remains a minor nit.
  - id: BR-6
    disposition: addressed
    note: |
      TestStandaloneCodexLaunchOptsOutOfTheSharedServer pins --no-daemon on a non-Couch codex launch.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"c7b1684a-f277-48e8-9289-422b5c64b79d","repository":"github.com/xianxu/pair","issue_id":"000891","recipe":"milestone-review","round":3,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/14b0f5505c92bd72","reviewed_head":"2fd584aa8b1dae3960d6b9fb135ca90940e5b75f","slot":"pair:8","requested_at":"2026-10-25T22:22:12.067867Z"},"phase":"finished","pid":18292,"started_at":"2026-10-25T22:22:12.32679Z","finished_at":"2026-10-25T22:29:06.948842Z","session_id":"24b01096-64a7-4f3c-9545-869cca77d5a2","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.896153833s
head: "2fd584aa8b1dae3960d6b9fb135ca90940e5b75f"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
