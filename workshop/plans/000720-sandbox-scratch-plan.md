---
collaboration-mode: discussion
---

# Sandbox-aware slot scratch

**Current contract: V2 persistent opt-in, in the final Revisions entry.** The settings-projection design below is retained as superseded history.

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
| ReadSandboxPolicy / sandboxInvocation | cmd/internal/couchscratch/sandbox_read.go | new | bounded read-only config files and invocation projection |
| scratchAdmission / configureScratch | cmd/internal/couchcmd/scratch.go | modified | policy before existing Manager preparation |
| SlotEnvironment callback input | cmd/internal/couchcore/couch.go, launch_existing.go | modified | actual selected LaunchProfile |
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

- [x] Write ScratchAllowed and projection unit tests first using the strategy table; run red before implementation. Keep the operator's enabled user-settings /Volumes/TestScratch allowance as the positive acceptance control.
- [x] Implement minimal projection/reader and actual-profile launch plumbing; reproduce the original host-probe-success/consumer-denied failure with a red integration test, then make it pass. Preserve warm/fresh routing and isolated runtime tests.
- [x] Add read-only show selection, tests, source inventory and atlas map. Check pure-vs-IO placement and bounded read/entry scaling. Add a local/shared lint proposal only if this resolves a mechanically detectable API misuse; this is consumer-policy semantics, not a general syntactic mistake.
- [ ] Run focused couchscratch/couchcmd/couchcore tests and artifactpath, then push and run the recorded full suite. Close through SDLC's fresh review, fix Important findings, land through sdlc merge, report Git-verified PR/merge to hub and stop.

## Acceptance and evidence

Hub approved the shape and native-relaunch follow-up (6cfb6a23-fb86-4013-93fd-1047d7a7b73c). No pair:0 staging. Headless config/export/show fixtures qualify this change; live follow-up rides the next natural slot relaunch. No native settings change is authorized. Current Codex tool environment has filesystem sandbox disabled/danger-full-access, so our own shell calls do not prove sandboxed access.

Codex source: [SandboxPolicy writable roots](https://github.com/openai/codex/blob/main/codex-rs/protocol/src/protocol.rs), new_workspace_write_policy and get_writable_roots_with_cwd; TMPDIR exclusion defaults false and TMPDIR is added unless excluded. [Config reference](https://developers.openai.com/codex/config-reference/) names sandbox_mode and exclude_tmpdir_env_var. Claude: [sandboxing](https://code.claude.com/docs/en/sandboxing), [settings](https://code.claude.com/docs/en/settings), [managed settings](https://code.claude.com/docs/en/managed-settings). Record these as source-derived semantics, not a full effective native resolver.

## Revisions

### 2026-10-19 — plan-quality advisory

Plan-quality round 1 INFO, no blocking findings. PQ-1 requests concrete projection function names and removal of a duplicated prose case inventory. Named ProjectClaudeSettings/ProjectCodexSettings and compressed the task to the existing adversarial strategy table, preserving the hub's positive acceptance control. No behavioral design change.

### 2026-10-19 — implementation map

Keep the existing callback signature: supply its selected profile through a call-local ThreadRecord copy at launch, leaving the durable last-registered profile unchanged. Warm attach already supplies the retained selected profile and needs no production change. The bounded IO/invocation reader lives separately in sandbox_read.go; scratchAdmission wraps the existing environment preparer so the manager and its generic fixture API stay unchanged. Update the map to actual ownership, not an anticipated signature expansion. Estimate-quality INFO's tight allocations are acknowledged: reader and launch rows include their tests, discovery includes completed research once (not additional research), and the review row includes foreground verification/landing attention; no numbers are backfitted.

### 2026-10-19 21:30 PDT — V2 approved: declare setup, do not emulate native policy

Hub approvals 0e40b3a8-694a-428b-8279-1e0dcc653e65, e84bb6d4-2146-47c0-beaf-48a95227d28c and 53c2a54b-e92a-4bd9-959d-0091c50793e1 replace the prior design and acceptance contract. Close round 1 REWORK identified unmodeled permission denies and native local-settings discovery (BR-1/BR-2); lexical alias handling was another uncertainty. Stop extending that resolver. The simpler alternative is a persistent operator declaration created together with the machine sandbox allowance. No agent resolver, native-config reads or TOML dependency remains.

#### V2 core concepts (authoritative)

| Entity | Location | Kind / status |
|--------|----------|---------------|
| ChooseSlotScratch(probe, optedIn) | cmd/internal/couchscratch/optin.go | pure, new; one admission predicate over the existing host probe and explicit opt-in |
| ScratchOptInPath(home, configHome) | cmd/internal/couchscratch/optin.go | pure, new; XDG path with HOME fallback; invalid path input cannot grant admission |
| scratchAdmission / configureScratch | cmd/internal/couchcmd/scratch.go | integration, modified; one read-only Lstat of declaration, existing probe/preparer and bounded note |
| ScratchDiagnostic / show projection | existing couchcore/couchcmd files named above | integration, new; current next-launch selection, no preparation or new stored state |

The declaration is a regular file at XDG_CONFIG_HOME/pair/agent-sandbox-allowed, or HOME/.config/pair/agent-sandbox-allowed when XDG_CONFIG_HOME is empty. Contents are ignored; no file is created, changed or removed by Couch. Missing, inaccessible or non-regular declarations decline optional scratch. HOME/XDG path routing is injected and isolated Couch fixtures cannot consult the operator's marker. The operator/TL owns creation after configuring every agent's sandbox allowance, and removal disables adoption on later launches. It persists on disk across reboot and is not stored on TestScratch.

ChooseSlotScratch requires both declaration and the existing ChooseScratch host checks. Existing slot identity, manager try-lock, owner declaration, write probe and cleanup remain unchanged. Denial returns no overlay (keeps normal inherited values) and one line naming the marker; no automatic retry or settings mutation. Current show reports the same selection without creating scratch. A marker is a declaration, not measured sandbox permission: creating it without the allowance can reproduce the original build refusal, explicitly accepted by the TL; ariadne#477 protects the SDLC temp consumer. This component never claims native-policy inference.

#### V2 visible states and recovery

| Condition | Current selection | One recovery action | Possible loss |
|-----------|-------------------|---------------------|---------------|
| Marker absent/unreadable/non-file | normal temp; note names marker | complete machine setup, then create marker and relaunch | optional RAM speed |
| Marker present, host available | RAM disk | retain setup | temp contents on RAM reboot, as today |
| Host unavailable despite marker | normal temp; existing fallback | use normal temp | optional RAM speed |
| Marker created without allowance | admission still says RAM, actual build can refuse | correct the machine setup and relaunch | failed temporary build work |

#### V2 function guards and cost

ChooseSlotScratch: pure cross-product table over declaration and host availability; no native-agent vocabulary. ScratchOptInPath: literal XDG/HOME table, malformed inputs cannot become paths. scratchAdmission: portable real files with marker mutation (missing → regular → removed), host probe success cannot substitute for opt-in, and non-file declaration refuses. Show shares admission and remains effect-free. Full artifactpath guards source inventory. Cost is one marker stat plus existing host probe per launch/show, independent of session count/settings-tree size; no scans, file-content reads, cache, worker, journal or new format. No lint catalog rule can syntactically infer consumer sandbox policy; deleting inference eliminates the semantic mistake class.

#### V2 implementation and acceptance (supersedes old task rows)

- [x] Delete the native policy/projection files and their tests, TOML dependency and selected-profile plumbing; restore unchanged legacy source expectations rather than leaving unused resolver pieces.
- [x] Add red-first pure opt-in tables and writable-host/no-marker composition regression, then the small read-only marker admission. Cover an explicit XDG path and isolated HOME fallback, marker lifecycle and current-selection diagnostics.
- [x] Update README/atlas around the declaration and its admitted limitation. Run focused tests/source guards, push the corrected code and queue a new recorded suite; cancel the superseded attempt only through SDLC.
- [ ] Close with BR-1/BR-2 disposed by eliminating native-policy interpretation, then land only after final-code recorded suite/CI pass. Report Git-verified merge to hub and stop. Hub creates the real marker and updates ops setup README before slot relaunches; worker never creates it on this machine.

No pair:0 staging or native config writes. Existing estimate remains the original forecast, including spent abandoned work; do not backfit it to the smaller remaining implementation. Re-run change-code on this revised design before code. The first suite attempt 8cffba3e5ad21df2e70eca2667f643f3 was cancelled while queued before execution; no result is accepted from it.

### 2026-10-19 — V2 marker basename and implemented evidence

Hub approved pair/sandbox-allowed (3827d91b-8eff-4aec-9ef2-9802731551e7), with the same XDG_CONFIG_HOME and HOME fallback. This supersedes the proposed agent-sandbox-allowed basename: the existing source guard correctly reserves the agent- path family for thread artifacts. No guard exception or new mechanism is needed.

Native settings readers, projections, tests, selected-profile plumbing and the TOML dependency are deleted. Pure opt-in red failed against a permissive stub; green passed 0.177s. The integration red then failed on the missing marker note (0.388s); implemented admission passed 0.362s. Focused pure/composition/show/launch controls passed couchscratch 0.155s, couchcmd 0.397s, couchcore 0.311s; all scratch-related couchcmd tests passed 0.462s and core controls 0.234s. Marker mutation, explicit XDG, isolated escape refusal and symlink refusal are covered. Source guard initially caught the basename collision; the renamed result must pass before publication. No native machine marker/settings were changed.

### 2026-10-19 — BR-3 diagnostic target coverage

Close round 2 disposed BR-1/BR-2 as addressed, but did not finalize because BR-3 identified missing scratch selection on a valid slot with no thread. Current-selection diagnostics apply to both thread targets and slot-only targets. Admission is machine-wide and does not require a ThreadRecord; retain the existing callback and invoke it with an empty record for the slot-only case. Extend the real slot-only show regression rather than inventing a thread or preparing scratch. Cancelled queued V2 suite through SDLC; final corrected code gets a new recorded run. No sandbox admission or setup contract changes.
