---
id: 000891
status: working
deps: []
github_issue:
created: 2026-10-25
updated: 2026-10-25
estimate_hours:
card_mirror: 'cb11592656da35aba73a63fbbc30e4e8a75f0c5f' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-25T09:36:24-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:8
    worktree: /Users/xianxu/workspace/worktree/pair-slot8/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "35fb6ad0", done: "7aee7188"}
---

# Codex 0.164's shared background server prompt stalls every wrapped start; launch so it can't appear

## Problem

## Spec

**Codex 0.164 stops every Couch-launched start at a "Background server has incompatible feature settings" prompt
(10-25).** Codex 0.164 runs a shared background server for every session on the machine. Couch launches Codex with
`shell_snapshot` disabled; once a server is running with other settings (here, a Codex started outside Couch right
after the update), each Couch launch stops at a three-way prompt: run without daemon this time / restart the shared
server with these settings (which can interrupt other sessions) / cancel. Couch can't answer prompts (`couch --keys`
refuses), so the slot sits stalled until the operator presses a key; pair:11 did this twice on 10-25. Every Codex
slot will meet it on its next start.

`pair wrap` launches Codex so the prompt can't appear: either opt out of the shared server for wrapped sessions (the
equivalent of "run without daemon", via a supported flag or config if 0.164 has one), or launch with settings that
match the shared server's. Never restart the shared server from a slot. Each launch also records the Codex version
it ran (the hub notice on a version change is the follow-up #894).

## Done when

- A wrapped Codex ≥ 0.164 launch passes `--no-daemon` (a private app server), so a mismatched shared server cannot
  stop it at a prompt; below 0.164 or when the version probe fails, the launch argv is exactly unchanged.
- The probed version is recorded per launch: `PAIR_AGENT_VERSION` in the launch env, carried into the ready
  record as `agent_version`.
- Verified by launcher/readiness/wrap tests plus a documented manual PTY check: with this machine's mismatched
  shared server left running and untouched, the fixed argv reaches the composer with no "Background server has
  incompatible feature settings", and the control argv (without the flag) shows it.

## Plan

Part A of `workshop/plans/000891-codex-no-daemon-and-version-notice-plan.md` (Tasks 1–2); Part B is #894.

- [x] A1: launcher probes `codex --version` once per launch (3s bound, failure silent), sets
      `PAIR_AGENT_VERSION` on every launch (empty when unknown, clearing any inherited value), and prepends
      `--no-daemon` at ≥ 0.164.0 unless the caller already passed it. All Codex launches; the shared server is
      never contacted or restarted.
- [x] A2: launcher tests pin 0.164.0 (flag + env), 0.163.0 and a failed probe (argv unchanged; the version is
      still recorded when known), and an explicit `--no-daemon` (no duplicate). `FreshAgentArgs` walks past the
      new leading flag (it is in `codexBoolGlobalOption`), so a fresh start still drops `resume <id>`.
- [x] A3: ready record carries the optional validated `agent_version`; the wrapper copies
      `PAIR_AGENT_VERSION` into it (the transport #894 will read).
- [x] V1: focused + package tests green.
- [x] V2: manual PTY check as stated in Done when, control included.
- [x] V3: atlas `codex-resume.md`; `make test-affected-inventory` audited; suite recorded on the pushed head.

Landing after close: `sdlc pr`; `sdlc merge --yes`; report the outcome to ops:0.

## Log

### 2026-10-25

- Codex 0.164 evidence (read-only probes, no session started): `codex --version` → `codex-cli 0.164.0`;
  `codex --help` lists `--no-daemon` ("Run without the shared background server, even if it is already
  running"); `codex --no-daemon resume --help` exits 0, so the flag is top-level and precedes the subcommand.
  Design: private app server per wrapped launch, gated on a parsed version so older CLIs are untouched.
- Seams: the launcher composes agent argv in `createflow.go` before `EncodeAgentCommand`; the wrapper writes
  `PAIR_AGENT_VERSION` into the ready record; Couch's `c.FreshRegistration` polls every fresh launch
  (`launch_existing.go:awaitFreshRegistration`). Notices with an unspecial-cased class route to the owning
  hub by default (`RouteNotice`), and `publishSystemNotice` is the established publish seam; the message
  service is the first place console + delivery + reader all exist. `readiness.Decode` tolerates unknown
  members, so the new field is compatible across an upgrade in both directions.
- Design exception, deliberate: the first-ever `StartupFresh` launch runs before the message service exists;
  it is silent anyway (empty store = baseline).
- Size: ~230 added non-test lines across launcher/readiness/wrapcmd/couchcore/couchcmd — outside the
  quick-flow shell (100 lines), so this is a full-flow close. Estimate proposed in the durable plan.
- TL steer (same day): land Part A first — it removes the stall every Codex restart hits — and move the
  version-change notice plus its baseline store to a new unclaimed issue, #894 (filed, cites this plan). With
  Part B out, #891 is the launcher probe, `--no-daemon`, `PAIR_AGENT_VERSION` and the ready-record carrier:
  ~95 added non-test lines, inside the quick-flow shell.
- Implemented on the Part A plan (commits fcc2ca3ac, 13b06a465): `agent_version.go` (`parseAgentVersion`,
  `versionAtLeast`, `argumentPresent`), `AgentVersionOps` on `Runtime` + `OSRuntime` (codex only, 3s context),
  the `createflow.go` seam (probe → `SetEnv(PAIR_AGENT_VERSION)` → prepend `--no-daemon` at ≥ 0.164.0),
  `--no-daemon` in `codexBoolGlobalOption` (the plan-quality PQ-1 fix: without it the wrapper's fresh-start
  strip stops at the new leading flag), and the ready record's validated `agent_version`. Measured code growth
  at commit time: 107 added non-test lines, so the close will run the full boundary review rather than the
  small-diff one (the flow itself stays quick; no refusal).
- Probe cost on the launch path: `codex --version` measured 0.11–0.15 s real over 3 runs (3 s upper bound).
- Manual PTY check (real Codex 0.164, 120×40 PTY, 12 s each, driven through Python `pty.fork` so no keys were
  sent and nothing was answered; the machine's shared app-server daemon PID 6294 was left running, same
  start time before and after; no leftover session processes):
  - fixed argv `--no-daemon --disable shell_snapshot --sandbox read-only --no-alt-screen` → composer
    ("Ask Codex to do anything"), `incompatible feature settings` count **0**.
  - control argv `--disable shell_snapshot --sandbox read-only --no-alt-screen` → the three-way prompt
    "Background server has incompatible feature settings / This session requires shell_snapshot to be
    disabled / 1. Run without daemon this time 2. Restart with these settings 3. Cancel", count **1**. The
    control proves the mismatch is live; the fixed leg proves the flag avoids it.
  - A second control without `--disable shell_snapshot` showed no prompt (0): the mismatch is the
    shell_snapshot setting, matching the spec's diagnosis.
- Tests: `go test ./cmd/internal/launcher ./cmd/internal/readiness ./cmd/internal/wrapcmd -count=1` PASS
  (launcher 112.0 s, wrapcmd 143.1 s), `go build ./...` and `go vet` on the three packages clean.
- Self-review fix before the boundary review: the seam's new `argumentPresent` ignored the `--` boundary (a
  prompt word `--no-daemon` after `--` would have suppressed the opt-out) and duplicated `hasFlag`; the seam now
  calls `hasFlag`, and `TestCodexLaunchTreatsPromptTextAsNoFlag` pins the post-`--` case.
- Pre-close suite on the pushed head 0726d114 (after the artifact-inventory fix): `sdlc local-check run
  --pre-close` resolved to the affected check, which upgraded to the full suite (trains on, full selection);
  exit 0 in 17m11s, recorded as `sdlc-local/suite` on 0726d114. The only observation was a SysV segment
  warning with unattributed segments. The prior run failed exactly on
  `agent_version.go: production source is absent from the exhaustive inventory`; 0726d114 classifies it in
  `NonArtifactSources`.
- Integrate gate: the first close refused because origin/main had moved (11fece5cf, broadcast host) past the
  freshness threshold. Merged origin/main into the branch (32a7dd6ad; no conflicts, `go build ./...` clean; the
  review window is unchanged, #304) and re-ran the full suite on the merged head: exit 0 in 16m11s, recorded
  as `sdlc-local/suite` on 32a7dd6a. `make test-affected-inventory` passes on the merged tree.
- Boundary review round 1: FIX-THEN-SHIP with BR-1/BR-4 blocking; fixed BR-4 and both Minors, disputed BR-1.
  - BR-4 (Important, real class): the seam now probes for every agent and always `SetEnv(PAIR_AGENT_VERSION,
    version)` (empty when unknown), so a value inherited from an outer pane cannot be recorded for a failed
    probe or another agent. `TestAgentVersionEnvOverwritesAnInheritedValue` pre-seeds a stale value for a
    failed codex probe and for a claude launch; the failed-probe test now requires the cleared empty value.
  - Minor: the probe command sets `cmd.WaitDelay = 1s`, so a shim that forks the CLI and keeps stdout open
    cannot outlive the 3s context bound.
  - Minor: `TestStandaloneCodexLaunchOptsOutOfTheSharedServer` pins the gate on a non-Couch codex launch
    (`--no-daemon --no-alt-screen`).
  - BR-1 disputed as factually wrong: `--no-daemon` is in `codexBoolGlobalOption` since 13b06a465, and both
    `TestFreshAgentArgsWalksPastNoDaemon` and `TestCodexLaunchOptsOutOfTheSharedServer` pass on this head. The
    wrapper fresh-start path (`wrap.go` → `FreshAgentArgs` → `codexResumeCommandIndex` → the table) consumes
    the leading flag and drops `resume <id>`.
  - Context choice kept: the probe uses `context.Background()`; it is read-only and bounded (3s + 1s worst
    case), so threading the launch context through the Runtime seam buys at most 3s of cancellation.
- Main moved again during the fix cycle (#890 and #886 landed through 774eb26bf); merged origin/main a second
  time (01e0396cd, clean, build clean) so the freshness gate and the suite describe the shipping tree. The
  BR-1 dispute commit (be35a41f) rides below it. Full suite on 01e0396cd: exit 0 in 13m39s, recorded as
  `sdlc-local/suite`; `make test-affected-inventory` passes.

## Revisions

### 2026-10-25 — inherited env cleared (boundary review BR-4)

Reason: `PAIR_AGENT_VERSION` was written only after a successful probe and only for codex, so a value inherited
from an outer pane could reach the ready record for a failed probe or another agent.
Delta: the seam writes it on every launch (empty when unknown) and A1 says so; the probe gains `WaitDelay` and a
standalone gate test; BR-1 is disputed as factually wrong.

### 2026-10-25 — V3 split from the landing steps

Reason: the close gate refuses while any `## Plan` item is unchecked, and V3 bundled post-close steps (PR
merged, outcome reported) that cannot be true before the close.
Delta: V3 keeps only the pre-close evidence (atlas, affected inventory, suite); `sdlc pr`, `sdlc merge` and
the ops:0 report moved to a non-checkbox landing line.

### 2026-10-25 — scope to Part A (TL steer)

Reason: the stall fix should land now, in the cheap window; the version notice is independent follow-up work.
Delta: Done when, Plan and the manual check now cover only the launcher probe, `--no-daemon`,
`PAIR_AGENT_VERSION` and the ready-record carrier. Tasks 3–5 of the durable plan (ready-record reader,
store/observer, notice) moved to #894; the durable plan keeps the full design and gains a Revisions note.
