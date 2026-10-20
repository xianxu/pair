---
collaboration-mode: discussion
---

# Sandbox-aware slot scratch

> Follow AGENTS.md §3 for delegation. One atomic close boundary; no milestone tags.

**Goal:** Never opt a slot into optional RAM scratch based solely on Couch's unsandboxed write probe.

**Architecture:** Read bounded non-credential settings at launch and project them into one normalized policy. A pure predicate owns admission; existing scratch preparation still owns mount/space/ownership checks. Diagnostics reuse that policy without preparing directories.

**Tech stack:** Go; JSON and a standard TOML decoder; existing Couch launch, scratch manager and show seams. No agent startup, settings edits, native sandbox probes or new runtime executable dependency.

## Core concepts

### Pure

| Name | Location | Status |
|------|----------|--------|
| SandboxPolicy / ScratchAllowed | cmd/internal/couchscratch/sandbox.go | new |
| ProjectClaudeSettings / ProjectCodexSettings | cmd/internal/couchscratch/sandbox_settings.go | new |

Policy is invocation-scoped: known, sandbox-enabled, allowed and denied concrete directories. Unknown refuses optimization; disabled permits it; otherwise deny overlaps win and a component-boundary ancestor allowance must cover the scratch root. Relative/home paths resolve against their settings source and injected home; unsupported globs remain conservative. Codex workspace-write maps to an allowance because TMPDIR is writable by default, unless excluded. Read-only refuses; danger-full-access permits. No second admission policy in diagnostics.

### Integration

| Name | Location | Status | Wraps |
|------|----------|--------|-------|
| ReadSandboxPolicy | cmd/internal/couchscratch/sandbox_settings.go | new | bounded read-only config files |
| scratchSlotEnvironment | cmd/internal/couchcmd/scratch.go | modified | policy before existing Manager preparation |
| SlotEnvironment callback | cmd/internal/couchcore/couch.go, launch_existing.go, warm_attach.go | modified | actual selected LaunchProfile |
| ScratchDiagnostic / show projection | cmd/internal/couchcore/couch.go, slotreport.go, operationdispatch.go; couchcmd/run.go | new | read-only current admission text |

Settings IO uses a portable fixture directory and injected getenv/read boundary. Tests model settings mutations between launches through the same reader; no live agent or credential access. The manager's existing note/cleanup/mount protections remain owners. Show reports current admission for the next launch, not observed child environment or a stored permission grant.

## Policy and supported envelope

- Claude: user settings under CLAUDE_CONFIG_DIR (otherwise home/.claude), checkout .claude/settings.json and settings.local.json. Last supplied enabled scalar wins; filesystem arrays combine; deny wins. Missing files are absent layers, default sandbox disabled as native documents. Malformed, oversized, unreadable, unsupported invocation settings/source overrides, or detected managed policy produce unknown. Managed files are never presumed equivalent to ordinary local settings; no credentials or remote managed-policy calls. This is a bounded standard-local-settings projection, not a replacement for native Claude's resolver.
- Codex: CODEX_HOME (otherwise home/.codex)/config.toml plus the same narrow checkout config projection. Only sandbox_mode and sandbox_workspace_write.exclude_tmpdir_env_var are consumed; profiles, CLI config/profile overrides and unsupported sandbox modes produce unknown. Known workspace-write with TMPDIR enabled permits export; read-only/exclusion refuse. No config/profile/Git/provider resolver. TOML parsing uses a maintained decoder, not regexes; unknown unrelated config fields are tolerated and nothing is rewritten.
- Unknown other agents or missing launch identity use normal temp with an explanatory note.
- The actual selected profile reaches cold/fresh/switch and warm attach callbacks. Never use the previous saved agent when a new profile is selected.
- Denied admission returns no scratch overlay, preserving ordinary inherited temp behavior. Existing unrelated environment/socket routing stays unchanged. A pre-existing external bad TMPDIR is outside this export fix (ariadne#477 addresses the SDLC consumer).
- Existing manager preparation remains authoritative for mount, available space, immutable slot owner and the host write probe. Policy approval alone never creates scratch on an absent volume.

## Operator projection

Actions: keep normal temp; edit the relevant sandbox allowance then relaunch.

| Internal condition | Visible selection / note | One recovery action | Possible loss |
|--------------------|--------------------------|---------------------|---------------|
| Known policy allows and volume ready | scratch: RAM disk | keep current setup | disposable temp on reboot, as today |
| Policy denies | scratch: normal temp (sandbox doesn't allow ROOT) | add allowance then relaunch | only optional speed benefit |
| Unknown settings/invocation | scratch: normal temp (sandbox policy unknown) | use normal temp | only optional speed benefit |
| Volume/space/owner probe unavailable | existing normal-temp fallback | use normal temp | only optional speed benefit |

No durable state, retry, background worker or permission escalation is added. Settings edits after launch can change next-launch diagnostics; they cannot repair an already-running child without relaunch. Config observation is evidence for optional optimization, not security authority or a promise against future policy changes.

## Bounds and architectural choices

ARCH-PURE: one pure policy predicate, literal table tests and pure decode/projection tests. ARCH-DRY: launch and show share the reader/predicate; existing manager retains preparation. ARCH-SECURE: settings paths/home/argv are untrusted inputs, bounded reads, strict known types; failures decline optimization and never print file contents. ARCH-EVOLVE: read-only tolerant parsing, no persisted format changes or rewriting unknown keys. ARCH-CONSTRAINTS: launch/show only, not UI ticks; fixed small config-file set, each capped at 1 MiB; O(bytes + path entries), no session inventory, process scans or recursive filesystem walks. Test 8/16 allowances and fixed read count. ARCH-MOCK: portable real-file settings fixtures and existing scratch/runner doubles. ARCH-FUNERAL/ARCH-STATE: values die with the call; no cache, record family, retained task or async effect. No lease can make an external sandbox allow a path; permissions must come from consumer policy rather than host-write ownership.

## Function strategy and mechanical guards

| Function / seam | Risk | Guard |
|-----------------|------|-------|
| ScratchAllowed | prefix grants, deny shadow, unknown treated as allowed | pure table with sibling prefix, ancestor, descendants and malformed/unknown policies |
| ProjectClaudeSettings / ProjectCodexSettings | ambiguous precedence and malformed boundary types | literal JSON/TOML tests plus fuzz seeds; portable scope fixtures verify precedence |
| ReadSandboxPolicy | native settings leaked or malformed accepted | bounded fixture reads, no config writes, unknown unrelated fields retained untouched |
| launch callback | stale saved agent on switch/fresh | real Couch fake-runner launch tests assert selected profile and no denied exports |
| scratch environment | host probe mistaken for agent permission | writable scratch fixture + denying Claude config yields nil overlay and one note |
| show rendering | projection differs from launch or implies observed child state | same-policy show fixture and explicit current-admission wording |
| source inventory | new Go files omitted | complete artifactpath package |

## Tasks

- [ ] Write ScratchAllowed and projection unit tests first using the strategy table; run red before implementation. Keep the operator's enabled user-settings /Volumes/TestScratch allowance as the positive acceptance control.
- [ ] Implement minimal projection/reader and actual-profile launch plumbing; reproduce the original host-probe-success/consumer-denied failure with a red integration test, then make it pass. Preserve warm/fresh routing and isolated runtime tests.
- [ ] Add read-only show selection, tests, source inventory and atlas map. Check pure-vs-IO placement and bounded read/entry scaling. Add a local/shared lint proposal only if this resolves a mechanically detectable API misuse; this is consumer-policy semantics, not a general syntactic mistake.
- [ ] Run focused couchscratch/couchcmd/couchcore tests and artifactpath, then push and run the recorded full suite. Close through SDLC's fresh review, fix Important findings, land through sdlc merge, report Git-verified PR/merge to hub and stop.

## Acceptance and evidence

Hub approved the shape and native-relaunch follow-up (6cfb6a23-fb86-4013-93fd-1047d7a7b73c). No pair:0 staging. Headless config/export/show fixtures qualify this change; live follow-up rides the next natural slot relaunch. No native settings change is authorized. Current Codex tool environment has filesystem sandbox disabled/danger-full-access, so our own shell calls do not prove sandboxed access.

Codex source: [SandboxPolicy writable roots](https://github.com/openai/codex/blob/main/codex-rs/protocol/src/protocol.rs), new_workspace_write_policy and get_writable_roots_with_cwd; TMPDIR exclusion defaults false and TMPDIR is added unless excluded. [Config reference](https://developers.openai.com/codex/config-reference/) names sandbox_mode and exclude_tmpdir_env_var. Claude: [sandboxing](https://code.claude.com/docs/en/sandboxing), [settings](https://code.claude.com/docs/en/settings), [managed settings](https://code.claude.com/docs/en/managed-settings). Record these as source-derived semantics, not a full effective native resolver.

## Revisions

### 2026-10-19 — plan-quality advisory

Plan-quality round 1 INFO, no blocking findings. PQ-1 requests concrete projection function names and removal of a duplicated prose case inventory. Named ProjectClaudeSettings/ProjectCodexSettings and compressed the task to the existing adversarial strategy table, preserving the hub's positive acceptance control. No behavioral design change.
